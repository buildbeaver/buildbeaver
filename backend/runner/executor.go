package runner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	hExec "os/exec"
	"path/filepath"
	hRuntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/runner/runtime"
	"github.com/buildbeaver/buildbeaver/runner/runtime/docker"
	"github.com/buildbeaver/buildbeaver/runner/runtime/exec"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type ExecutorFactory func(ctx context.Context) *Executor

func MakeExecutorFactory(
	config ExecutorConfig,
	client APIClient,
	gitRepoManager *GitCheckoutManager,
	logPipelineFactory logging.LogPipelineFactory,
	logFactory logger.LogFactory) ExecutorFactory {
	return func(ctx context.Context) *Executor {
		return NewExecutor(config, client, gitRepoManager, logPipelineFactory, logFactory)
	}
}

type ExecutorConfig struct {
	// IsLocal should be true if the currently executing process is running out
	// of a preconfigured workspace directory (e.g. when running under the command line tool)
	IsLocal bool
	// DynamicAPIEndpoint is the endpoint URL for build jobs to use when connecting to the Dynamic API.
	// Any 'localhost'-style endpoint will automatically be converted to an endpoint suitable for use
	// within docker containers as required.
	DynamicAPIEndpoint dynamic_api.Endpoint
}

// Executor executes the various lifecycle phases of a job and is driven by the orchestrator.
type Executor struct {
	config             ExecutorConfig
	apiClient          APIClient
	secretStore        *SecretStore
	checkoutManager    *GitCheckoutManager
	logPipelineFactory logging.LogPipelineFactory
	logFactory         logger.LogFactory
	log                logger.Log
	state              struct {
		runtime             runtime.Runtime
		workspaceDir        string
		stagingDir          string
		sshAgentPID         string
		globalEnvVars       []string
		globalEnvVarsByName map[string]string
	}
}

func NewExecutor(
	config ExecutorConfig,
	apiClient APIClient,
	gitRepoManager *GitCheckoutManager,
	logPipelineFactory logging.LogPipelineFactory,
	logFactory logger.LogFactory) *Executor {
	b := &Executor{
		config:             config,
		apiClient:          apiClient,
		checkoutManager:    gitRepoManager,
		logPipelineFactory: logPipelineFactory,
		logFactory:         logFactory,
		log:                logFactory("Executor"),
	}
	b.state.globalEnvVarsByName = map[string]string{}
	return b
}

func (b *Executor) Close() {}

// PreExecuteJob is called once per job, before the first step in the job is executed.
func (b *Executor) PreExecuteJob(ctx *JobBuildContext) error {
	log := b.withJobLogFields(b.log, ctx.job)
	log.Info("PreExecuteJob")
	b.secretStore = NewSecretStore(b.apiClient, ctx.Job().Job.RepoID)
	err := b.initFileSystem(ctx)
	if err != nil {
		return fmt.Errorf("error preparing job directories: %w", err)
	}
	err = b.secretStore.Init(ctx.Ctx())
	if err != nil {
		return fmt.Errorf("error loading secrets: %w", err)
	}
	// Prepare global environment before calling initJobLogPipeline, fingerprintJob job or prepareServices.
	// This function can add secrets to the secret store to be redacted from logs in the log pipeline
	err = b.prepareStandardGlobalEnv(ctx)
	if err != nil {
		return fmt.Errorf("error preparing dynamic build environment: %w", err)
	}
	err = b.initJobLogPipeline(ctx)
	if err != nil {
		return fmt.Errorf("error initializing log pipeline: %w", err)
	}
	err = b.initGitCheckout(ctx)
	if err != nil {
		return fmt.Errorf("error preparing checkout: %w", err)
	}
	err = b.prepareSSHAgent(ctx)
	if err != nil {
		return fmt.Errorf("error preparing SSH agent: %w", err)
	}
	err = b.prepareRuntime(ctx)
	if err != nil {
		return fmt.Errorf("error preparing runtime: %w", err)
	}
	err = b.fingerprintJob(ctx)
	if err != nil {
		return fmt.Errorf("error fingerprinting job: %w", err)
	}
	if ctx.IsJobIndirected() {
		return nil
	}
	err = NewArtifactManager(b.config.IsLocal, b.state.workspaceDir, b.apiClient).DownloadArtifacts(ctx)
	if err != nil {
		return fmt.Errorf("error downloading artifacts: %w", err)
	}
	err = b.prepareServices(ctx)
	if err != nil {
		return fmt.Errorf("error preparing services: %w", err)
	}
	return nil
}

// PreExecuteStep is called before executing each step. (and after PreExecuteJob).
func (b *Executor) PreExecuteStep(ctx *StepBuildContext) error {
	log := b.withStepLogFields(b.log, ctx.Job(), ctx.Step())
	log.Info("PreExecuteStep")
	err := b.initStepLogPipeline(ctx)
	if err != nil {
		return fmt.Errorf("error initializing log pipeline: %w", err)
	}
	return nil
}

// ExecuteStep executes the step defined in the build context.
// ExecuteStep is called after PreExecuteStep, and only if PreExecuteStep succeeded.
func (b *Executor) ExecuteStep(ctx *StepBuildContext) error {
	log := b.withStepLogFields(b.log, ctx.Job(), ctx.Step())
	log.Info("ExecuteStep")
	if ctx.IsJobIndirected() {
		return nil
	}
	env, err := b.makeEnvMappings(ctx.Job().Job.Environment)
	if err != nil {
		return fmt.Errorf("error making env vars for step: %w", err)
	}

	converter := ctx.LogPipeline().Converter()
	config := runtime.ExecConfig{
		Name:     ctx.Step().Name.String(),
		Commands: models.CommandsToStrings(ctx.Step().Commands),
		Env:      env,
		Stdout:   converter,
		Stderr:   converter,
	}
	return b.state.runtime.Exec(ctx.Ctx(), config)
}

// LogStepError writes an error to the step's log pipeline.
func (b *Executor) LogStepError(ctx *StepBuildContext, stepError error) {
	pipeline := ctx.LogPipeline() // this will always give us a valid pipeline
	// Write the step error at the top level of the step log, rather than inside a block
	pipeline.StructuredLogger().WriteError(stepError.Error())
}

// PostExecuteStep is called after executing each step (and before PostExecuteJob).
// PostExecuteStep is always called for each step, even if PreExecuteStep or ExecuteStep failed, and must
// clean up any allocated resources.
func (b *Executor) PostExecuteStep(ctx *StepBuildContext) error {
	log := b.withStepLogFields(b.log, ctx.Job(), ctx.Step())
	log.Info("PostExecuteStep")

	// Uncomment and use the following context for any cleanup that can time out
	//cleanupCtx, cleanupCancel := getCleanupContext()
	//defer cleanupCancel()

	var results *multierror.Error

	// Always flush and close any open log pipeline
	ctx.LogPipeline().Flush()
	ctx.LogPipeline().Close()
	ctx.ClearLogPipeline() // ensure no further entries are sent to the closed pipeline

	return results.ErrorOrNil()
}

// LogJobError writes an error to the job's log pipeline.
func (b *Executor) LogJobError(ctx *JobBuildContext, stepError error) {
	pipeline := ctx.LogPipeline() // this will always give us a valid pipeline
	// Write the job error at the top level of the job log, rather than inside a block
	pipeline.StructuredLogger().WriteError(stepError.Error())
}

// PostExecuteJob is called once per job, after the last step in the job is executed.
// PostExecuteJob is always called, even if PreExecuteJob failed, and must clean up any allocated resources.
func (b *Executor) PostExecuteJob(ctx *JobBuildContext) error {
	log := b.withJobLogFields(b.log, ctx.job)
	log.Info("PostExecuteJob")

	cleanupCtx, cleanupCancel := getCleanupContext()
	defer cleanupCancel()

	// NOTE: We can't rely on ctx.LogPipeline() being non-nil in this method so be careful
	var results *multierror.Error

	// Upload all declared artifacts generated by the steps as they ran
	if len(ctx.Job().Job.ArtifactDefinitions) > 0 {
		log.Infof("Uploading %d artifacts...", len(ctx.Job().Job.ArtifactDefinitions))
	}
	err := NewArtifactManager(b.config.IsLocal, b.state.workspaceDir, b.apiClient).UploadArtifacts(ctx, b.state.globalEnvVarsByName)
	if err != nil {
		results = multierror.Append(results, fmt.Errorf("error uploading artifacts: %w", err))
	}

	if b.state.runtime != nil {
		// Use cleanup context, not job context, so we still clean up even if job has timed out
		err := b.state.runtime.Stop(cleanupCtx)
		if err != nil {
			results = multierror.Append(results, fmt.Errorf("error stopping runtime: %w", err))
		}
	}

	err = b.cleanupFileSystem(ctx)
	if err != nil {
		results = multierror.Append(results, fmt.Errorf("error tearing down job directories: %w", err))
	}

	// Always flush and close any open log pipeline
	ctx.LogPipeline().Flush()
	ctx.LogPipeline().Close()
	ctx.ClearLogPipeline() // ensure no further entries are sent to the closed pipeline

	return results.ErrorOrNil()
}

// CleanUp removes any resources left over by previous instances of each of the available runtimes,
// including Docker.
func (b *Executor) CleanUp(timeout time.Duration) error {
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancelFunc()
	baseConfig := &runtime.Config{
		RuntimeID:    "cleanup",
		StagingDir:   b.state.stagingDir,
		WorkspaceDir: b.state.workspaceDir,
		LogPipeline:  nil, // no log pipeline required for cleanup
	}
	var results *multierror.Error

	err := b.cleanUpDocker(ctx, baseConfig)
	if err != nil {
		results = multierror.Append(results, err)
	}

	err = b.cleanUpExec(ctx, baseConfig)
	if err != nil {
		results = multierror.Append(results, err)
	}

	return results.ErrorOrNil()
}

// cleanUpDocker cleans up any resources left over by the docker runtime.
func (b *Executor) cleanUpDocker(ctx context.Context, baseConfig *runtime.Config) error {
	// Create a docker client to use during the cleanup
	dClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return errors.Wrap(err, "error making Docker API client")
	}
	config := docker.Config{
		Config: *baseConfig,
	}
	dockerRuntime := docker.NewRuntime(config, dClient, b.logFactory)

	err = dockerRuntime.CleanUp(ctx)
	if err != nil {
		return err
	}

	return nil
}

// cleanUpExec cleans up any resources left over by the exec runtime.
func (b *Executor) cleanUpExec(ctx context.Context, baseConfig *runtime.Config) error {
	execConfig := exec.Config{
		Config: *baseConfig,
	}
	execRuntime := exec.NewRuntime(execConfig)

	err := execRuntime.CleanUp(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (b *Executor) initFileSystem(ctx *JobBuildContext) error {
	log := b.withJobLogFields(b.log, ctx.job)
	jobRootDir := filepath.Join(os.TempDir(), "buildbeaver", models.SanitizeFilePathID(ctx.Job().GetID()))
	// Local builds are expected to be operating out of a git checkout
	// directory, so we set the workspace to the current working directory.
	if b.config.IsLocal {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "error determining current working directory")
		}
		b.state.workspaceDir = cwd
	} else {
		b.state.workspaceDir = filepath.Join(jobRootDir, "workspace")
		err := os.MkdirAll(b.state.workspaceDir, 0777)
		if err != nil {
			return errors.Wrap(err, "error creating job workspace directory")
		}
	}
	b.state.stagingDir = filepath.Join(jobRootDir, "staging")
	err := os.MkdirAll(b.state.stagingDir, 0777)
	if err != nil {
		return errors.Wrap(err, "error creating job staging directory")
	}
	b.addGlobalEnvVar("CI_WORKSPACE", b.state.workspaceDir, false)
	log.WithFields(logger.Fields{"workspace": b.state.workspaceDir, "staging": b.state.stagingDir}).
		Info("Created filesystem directories")
	return nil
}

func (b *Executor) cleanupFileSystem(ctx *JobBuildContext) error {
	log := b.withJobLogFields(b.log, ctx.job)
	var results *multierror.Error
	if !b.config.IsLocal && b.state.workspaceDir != "" {
		err := os.RemoveAll(b.state.workspaceDir)
		if err != nil && !os.IsNotExist(err) {
			results = multierror.Append(results, errors.Wrap(err, "error destroying workspace directory"))
		}
	}
	if b.state.stagingDir != "" {
		err := os.RemoveAll(b.state.stagingDir)
		if err != nil && !os.IsNotExist(err) {
			results = multierror.Append(results, errors.Wrap(err, "error destroying staging directory"))
		}
	}
	log.Info("Cleaned filesystem")
	return results.ErrorOrNil()
}

func (b *Executor) initGitCheckout(ctx *JobBuildContext) error {
	if b.config.IsLocal {
		return nil
	}
	log := b.withJobLogFields(b.log, ctx.job)
	repoSSHKey, err := b.secretStore.GetSecret(models.RepoSSHKeySecretName, true)
	if err != nil {
		return fmt.Errorf("error finding repo SSH key: %w", err)
	}
	checkout := CheckoutInfo{
		Repo:        ctx.Job().Repo,
		Commit:      ctx.Job().Commit,
		Ref:         ctx.Job().Job.Ref,
		RepoSSHKey:  []byte(repoSSHKey.Value),
		CheckoutDir: b.state.workspaceDir,
	}
	err = b.checkoutManager.Checkout(ctx.Ctx(), checkout, ctx.LogPipeline())
	if err != nil {
		return fmt.Errorf("error checking out repo: %w", err)
	}
	log.WithFields(logger.Fields{"repo_id": ctx.Job().Repo.ID.String(), "checkout_dir": b.state.workspaceDir}).
		Info("Checked out git repo")
	return nil
}

func (b *Executor) prepareSSHAgent(ctx *JobBuildContext) error {
	if hRuntime.GOOS == "windows" {
		return nil
	}
	if b.config.IsLocal {
		return nil
	}

	// TODO can we avoid writing the key to the filesystem?
	log := b.withJobLogFields(b.log, ctx.job)
	repoSSHKeyPath := filepath.Join(b.state.stagingDir, "repo_ssh_key")
	repoSSHKey, err := b.secretStore.GetSecret(models.RepoSSHKeySecretName, true)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(repoSSHKeyPath, []byte(repoSSHKey.Value), 0600)
	if err != nil {
		return fmt.Errorf("error writing repo SSH key: %w", err)
	}

	var (
		sshAgentSocketPath = filepath.Join(b.state.stagingDir, "ssh_agent_sock")
		sshAgentPIDPath    = filepath.Join(b.state.stagingDir, "ssh_agent_pid")
	)

	shell := runtime.ShellOrDefault(runtime.OSLinux, ctx.Job().Job.DockerConfig.Shell)

	cmd := hExec.Command(shell, "-c", `
		eval $(ssh-agent -s -a `+sshAgentSocketPath+`)
		echo "${SSH_AGENT_PID}" > `+sshAgentPIDPath+`
		ssh-add "`+repoSSHKeyPath+`"
	`)
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "error creating SSH Agent")
	}

	// Stash the PID so we can kill it later
	buf, err := ioutil.ReadFile(sshAgentPIDPath)
	if err != nil {
		return errors.Wrap(err, "error reading SSH agent PID path")
	}
	b.state.sshAgentPID = strings.TrimSuffix(string(buf[:]), "\n")

	// We don't need this on disk anymore
	err = os.Remove(repoSSHKeyPath)
	if err != nil {
		b.log.Warnf("Ignoring error removing Repo SSH key from disk after loading into SSH agent: %v", err)
	}

	b.addGlobalEnvVar("SSH_AUTH_SOCK", sshAgentSocketPath, false)
	b.addGlobalEnvVar("SSH_AGENT_PID", b.state.sshAgentPID, false)

	log.WithFields(logger.Fields{"ssh_auth_sock": sshAgentSocketPath, "ssh_agent_pid": b.state.sshAgentPID}).
		Info("Started SSH Agent")
	return nil
}

func (b *Executor) teardownSSHAgent() error {
	if b.state.sshAgentPID == "" {
		return nil
	}
	cmd := hExec.Command("kill", "-9", b.state.sshAgentPID)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error stopping SSH agent: %w", err)
	}
	return nil
}

func (b *Executor) prepareRuntime(ctx *JobBuildContext) error {
	job := ctx.Job().Job
	baseConfig := runtime.Config{
		// Short ID has plenty of uniqueness for a runtime ID for a job within a runner
		RuntimeID:    models.SanitizeFilePathShortID(job.GetID()),
		StagingDir:   b.state.stagingDir,
		WorkspaceDir: b.state.workspaceDir,
		LogPipeline:  ctx.LogPipeline(),
	}

	switch job.Type {
	case models.JobTypeDocker:
		if job.DockerConfig == nil {
			return fmt.Errorf("error no docker config provided for job of type '%s'", models.JobTypeDocker)
		}
		dClient, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return errors.Wrap(err, "error making Docker API client")
		}
		jobDockerAuth, err := b.getDockerAuth(job.DockerConfig)
		if err != nil {
			return fmt.Errorf("error making docker auth for job: %w", err)
		}
		config := docker.Config{
			Config:       baseConfig,
			ImageURI:     job.DockerConfig.Image,
			AuthOrNil:    jobDockerAuth,
			PullStrategy: job.DockerConfig.Pull,
			ShellOrNil:   job.DockerConfig.Shell,
		}
		for _, service := range job.Services {
			serviceDockerAuth, err := b.getDockerAuth(service.DockerConfig)
			if err != nil {
				return fmt.Errorf("error making docker auth for service: %w", err)
			}
			sConfig := docker.RuntimeServiceConfig{
				Name:         service.Name,
				ImageURI:     service.DockerConfig.Image,
				AuthOrNil:    serviceDockerAuth,
				PullStrategy: service.DockerConfig.Pull,
			}
			config.Services = append(config.Services, sConfig)
		}
		b.state.runtime = docker.NewRuntime(config, dClient, b.logFactory)
	case models.JobTypeExec:
		config := exec.Config{
			Config:     baseConfig,
			ShellOrNil: nil, // TODO
		}
		b.state.runtime = exec.NewRuntime(config)
	default:
		return fmt.Errorf("error unsupported job kind: %v", job.Type)
	}
	return b.state.runtime.Start(ctx.Ctx())
}

func (b *Executor) prepareServices(ctx *JobBuildContext) error {
	for _, service := range ctx.Job().Job.Services {
		env, err := b.makeEnvMappings(service.Environment)
		if err != nil {
			return fmt.Errorf("error making env for service %q: %w", service.Name, err)
		}
		sConfig := runtime.ServiceConfig{
			Name: service.Name,
			Env:  env,
		}
		err = b.state.runtime.StartService(ctx.Ctx(), sConfig)
		if err != nil {
			return fmt.Errorf("error starting service %q: %w", service.Name, err)
		}
	}
	return nil
}

func (b *Executor) getDockerAuth(configOrNil *documents.DockerConfig) (*docker.Auth, error) {
	dockerAuth := &docker.Auth{}
	if configOrNil == nil {
		return dockerAuth, nil
	}
	if configOrNil.BasicAuth != nil {
		dockerAuth.Basic = &docker.BasicAuth{}
		// Check if the Username was provided by secret or direct
		if configOrNil.BasicAuth.Username.FromSecret != "" {
			secret, err := b.secretStore.GetSecret(configOrNil.BasicAuth.Username.FromSecret, false)
			if err != nil {
				return nil, errors.Wrapf(err, "Error sourcing value for Docker basic auth username from secret %q",
					configOrNil.BasicAuth.Username.FromSecret)
			}
			dockerAuth.Basic.Username = secret.Value[:]
		} else {
			dockerAuth.Basic.Username = configOrNil.BasicAuth.Username.Value
		}
		// Check if the Password has been provided by secret as this is the only option
		if configOrNil.BasicAuth.Password.FromSecret != "" {
			secret, err := b.secretStore.GetSecret(configOrNil.BasicAuth.Password.FromSecret, false)
			if err != nil {
				return nil, errors.Wrapf(err, "Error sourcing value for Docker basic auth password from secret %q",
					configOrNil.BasicAuth.Password.FromSecret)
			}
			dockerAuth.Basic.Password = secret.Value[:]
		} else {
			return nil, fmt.Errorf("error Docker basic auth password cannot be set in plaintext " +
				"and must be provided via secret")
		}
	}
	if configOrNil.AWSAuth != nil {
		dockerAuth.AWS = &docker.AWSAuth{}
		// TODO once env is specified at the job level we could check to see if env contains
		//  AWS_REGION and use that if it wasn't specified on the AWSAuth object
		if configOrNil.AWSAuth.AWSRegion != nil {
			dockerAuth.AWS.AWSRegion = *configOrNil.AWSAuth.AWSRegion
		}
		// Check if the id was provided by secret or direct
		if configOrNil.AWSAuth.AWSAccessKeyID.FromSecret != "" {
			secret, err := b.secretStore.GetSecret(configOrNil.AWSAuth.AWSAccessKeyID.FromSecret, false)
			if err != nil {
				return nil, errors.Wrapf(err, "Error sourcing value for Docker AWS auth Access Key ID from secret %q",
					configOrNil.AWSAuth.AWSAccessKeyID.FromSecret)
			}
			dockerAuth.AWS.AWSAccessKeyID = secret.Value[:]
		} else {
			dockerAuth.AWS.AWSAccessKeyID = configOrNil.AWSAuth.AWSAccessKeyID.Value
		}
		// Check if the key has been provided by secret as this is the only option
		if configOrNil.AWSAuth.AWSSecretAccessKey.FromSecret != "" {
			secret, err := b.secretStore.GetSecret(configOrNil.AWSAuth.AWSSecretAccessKey.FromSecret, false)
			if err != nil {
				return nil, errors.Wrapf(err, "Error sourcing value for Docker AWS auth Secret Access Key from secret %q",
					configOrNil.AWSAuth.AWSSecretAccessKey.FromSecret)
			}
			dockerAuth.AWS.AWSSecretAccessKey = secret.Value[:]
		} else {
			return nil, fmt.Errorf("error  Docker AWS auth Secret Access Key cannot be set in plaintext " +
				"and must be provided via secret")
		}
	}
	return dockerAuth, nil
}

// fingerprintJob calculates the job's fingerprint.
func (b *Executor) fingerprintJob(ctx *JobBuildContext) error {
	job := ctx.Job().Job

	if len(job.FingerprintCommands) == 0 {
		ctx.LogPipeline().StructuredLogger().WriteLine("Fingerprinting disabled as no fingerprint commands were defined. Consider using fingerprints to speed up this job.")
		return nil
	}

	hash := NewFingerprintHasher()
	hashType := models.HashTypeSHA1

	fingerPrintLogger := ctx.LogPipeline().StructuredLogger().Wrap("job_fingerprint", "Analyzing job fingerprint...")
	start := time.Now()

	// Include the job definition hash.
	// This ensures if someone changes the job definition (of this job, or any it depends
	// on due to fingerprint inclusion below) that we will rebuild it
	hash.Prepare("Job configuration")
	_, err := hash.Write([]byte(job.DefinitionDataHash))
	if err != nil {
		return fmt.Errorf("error writing job definition hash: %w", err)
	}

	// Include the fingerprint of all jobs this job depends on.
	// Sort as the server doesn't guarantee sort order across jobs.
	dependencies := make([]*documents.Job, len(ctx.Job().Jobs))
	copy(dependencies, ctx.Job().Jobs)
	sort.SliceStable(dependencies, func(i, j int) bool {
		return dependencies[i].Name < dependencies[j].Name
	})
	for _, job := range dependencies {
		hash.Append(fmt.Sprintf("%s fingerprint", job.Name), job.Fingerprint)
	}

	// TODO include hash of all artifact dependencies?
	// TODO if we ever support templating of fields of a job at runtime then we must include the template inputs in this hash
	fingerPrintLogger.WriteLine("Executing fingerprint command(s)")
	hash.Prepare("Command(s) stdout")

	converter := ctx.LogPipeline().Converter()

	env, err := b.makeEnvMappings(ctx.Job().Job.Environment)
	if err != nil {
		return fmt.Errorf("error making env vars for fingerprinting operation: %w", err)
	}

	config := runtime.ExecConfig{
		Name:     "fingerprint",
		Commands: models.CommandsToStrings(job.FingerprintCommands),
		Env:      env,
		Stdout:   hash,
		Stderr:   converter,
	}
	err = b.state.runtime.Exec(ctx.Ctx(), config)
	if err != nil {
		return err
	}

	fingerprint := hash.Finalize(fingerPrintLogger)

	updatedJobDoc, err := b.apiClient.UpdateJobFingerprint(
		ctx.Ctx(),
		job.ID,
		fingerprint,
		&hashType,
		job.ETag)
	if err != nil {
		return fmt.Errorf("error updating job fingerprint: %w", err)
	}
	ctx.SetJobDocument(updatedJobDoc)
	job = ctx.Job().Job

	// Set global env variable for fingerprint, for use in steps
	b.addGlobalEnvVar("BB_JOB_FINGERPRINT", ctx.Job().Job.Fingerprint, false)

	if job.IndirectToJobID.Valid() {
		fingerPrintLogger.WriteLinef("Fingerprint matched job %s; This job will be skipped", job.IndirectToJobID)
	} else {
		fingerPrintLogger.WriteLine("No fingerprint match found; This job will run")
	}
	fingerPrintLogger.WriteLinef("Fingerprinting completed in: %s", time.Now().Sub(start).Round(time.Millisecond))
	return nil
}

func (b *Executor) initJobLogPipeline(ctx *JobBuildContext) error {
	jobLogPipeline, err := b.logPipelineFactory(ctx.Ctx(), clock.New(), b.secretStore.GetAllSecrets(), ctx.Job().Job.LogDescriptorID)
	if err != nil {
		return fmt.Errorf("error creating log pipeline for job: %w", err)
	}
	ctx.SetLogPipeline(jobLogPipeline)
	return nil
}

func (b *Executor) initStepLogPipeline(ctx *StepBuildContext) error {
	stepLogPipeline, err := b.logPipelineFactory(ctx.Ctx(), clock.New(), b.secretStore.GetAllSecrets(), ctx.Step().LogDescriptorID)
	if err != nil {
		return fmt.Errorf("error creating log pipeline for step: %w", err)
	}
	ctx.SetLogPipeline(stepLogPipeline)
	return nil
}

// prepareStandardGlobalEnv adds global env variables for use by fingerprint commands, build step commands and services.
// This includes all the variables required by dynamic builds.
// This can also include adding secrets to the secret store.
func (b *Executor) prepareStandardGlobalEnv(ctx *JobBuildContext) error { // Calculate a suitable dynamic API endpoint based on the type of job
	// Dynamic API endpoint is the one configured, unless we need it to work within a docker container
	var dynamicAPIEndpoint = b.config.DynamicAPIEndpoint
	if ctx.job.Job.Type == models.JobTypeDocker {
		var err error
		dynamicAPIEndpoint, err = dynamic_api.GetDockerDynamicEndpoint(b.config.DynamicAPIEndpoint)
		if err != nil {
			b.log.Warnf("unable to find suitable address to allow docker containers to connect to endpoint %s: %s",
				b.config.DynamicAPIEndpoint, err.Error())
			dynamicAPIEndpoint = ""
		}
	}

	AddStandardGlobalEnvVars(ctx.Job(), dynamicAPIEndpoint, b.addGlobalEnvVar)
	return nil
}

// AddStandardGlobalEnvVars adds a standard set of environment variables for passing to commands executed during the
// running of a job (including commands for steps, fingerprinting and services).
// The job parameter is the dequeued runnable job being executed.
// The dynamicAPIEndpoint is the endpoint after any translation of localhost has been done, ready to be set into
// an environment variable.
// The supplied setter function is called to set each variable name and value.
func AddStandardGlobalEnvVars(
	runnable *documents.RunnableJob,
	dynamicAPIEndpoint dynamic_api.Endpoint,
	setter func(name string, value string, isSecret bool),
) {
	// Server info
	setter("BB_DYNAMIC_BUILD_API", dynamicAPIEndpoint.String(), false)
	setter("BB_BUILD_ACCESS_TOKEN", runnable.JWT, true)
	// Build info
	setter("BB_BUILD_ID", runnable.Job.BuildID.String(), false)
	// TODO: Populate BB_BUILD_NAME and BB_BUILD_OWNER_NAME when this is available to the runner
	setter("BB_BUILD_NAME", "no-build-name-available", false)
	setter("BB_BUILD_OWNER_NAME", "", false)
	setter("BB_BUILD_REF", runnable.Job.Ref, false)
	setter("BB_WORKFLOWS_TO_RUN", makeWorkflowList(runnable.WorkflowsToRun), false)
	// Commit info
	setter("BB_COMMIT_SHA", runnable.Commit.SHA, false)
	setter("BB_COMMIT_AUTHOR_NAME", runnable.Commit.AuthorName, false)
	setter("BB_COMMIT_AUTHOR_EMAIL", runnable.Commit.AuthorName, false)
	setter("BB_COMMIT_COMMITTER_NAME", runnable.Commit.CommitterName, false)
	setter("BB_COMMIT_COMMITTER_EMAIL", runnable.Commit.CommitterEmail, false)
	// Repo info
	setter("BB_REPO_NAME", runnable.Repo.Name.String(), false)
	setter("BB_REPO_SSH_URL", runnable.Repo.SSHURL, false)
	setter("BB_REPO_LINK", runnable.Repo.HTTPURL, false)
	// Job info for the current dynamic build job
	setter("BB_CONTROLLER_JOB_ID", runnable.Job.ID.String(), false)
	setter("BB_CONTROLLER_JOB_NAME", runnable.Job.Name.String(), false)
	// Fingerprint will be empty if not yet calculated
	setter("BB_JOB_FINGERPRINT", runnable.Job.Fingerprint, false)
}

// makeWorkflowList converts an array of workflow names to a comma-separated list.
func makeWorkflowList(workflows []models.ResourceName) string {
	list := ""
	for _, workflow := range workflows {
		if len(list) > 0 {
			list += "," // this isn't the first item in the list
		}
		list += workflow.String()
	}
	return list
}

func (b *Executor) addGlobalEnvVar(name string, value string, isSecret bool) {
	b.state.globalEnvVarsByName[name] = value
	b.state.globalEnvVars = append(b.state.globalEnvVars, fmt.Sprintf("%s=%s", name, value))
	if isSecret {
		now := time.Now().UTC()
		// Add a secret to the secret store to ensure this variable value is redacted
		b.secretStore.AddSecret(
			&models.SecretPlaintext{
				Secret: &models.Secret{
					ID:               models.NewSecretID(),
					Name:             models.ResourceName(name),
					RepoID:           models.RepoID{},
					CreatedAt:        models.NewTime(now),
					UpdatedAt:        models.NewTime(now),
					ETag:             "",
					KeyEncrypted:     nil,
					ValueEncrypted:   nil,
					DataKeyEncrypted: nil,
					IsInternal:       false,
				},
				Key:   name,
				Value: value,
			})
	}
}

// makeEnvMappings converts the specified environment variables to a mapping of `key=value` strings
// ready to be exported.
func (b *Executor) makeEnvMappings(environment []*documents.EnvVar) ([]string, error) {
	mappings := append([]string{}, b.state.globalEnvVars...)
	for _, env := range environment {
		var (
			name  = env.Name
			value = env.Value
		)
		// TODO check if this is a build for a PR, and if so, only export these if enabled
		if env.ValueFromSecret != "" {
			secret, err := b.secretStore.GetSecret(env.ValueFromSecret, false)
			if err != nil {
				return nil, fmt.Errorf("error sourcing value for environment variable %q from secret: %w",
					env.ValueFromSecret, err)
			}
			value = secret.Value[:]
		}
		mappings = append(mappings, fmt.Sprintf("%s=%s", strings.ToUpper(name), value))
	}
	return mappings, nil
}

func (b *Executor) withJobLogFields(log logger.Log, job *documents.RunnableJob) logger.Log {
	return log.WithFields(logger.Fields{"job_id": job.Job.ID.String(), "job_name": job.Job.Name})
}

func (b *Executor) withStepLogFields(log logger.Log, job *documents.RunnableJob, step *documents.Step) logger.Log {
	return b.withJobLogFields(log, job).WithFields(logger.Fields{"step_id": step.ID.String(), "step_name": step.Name})
}
