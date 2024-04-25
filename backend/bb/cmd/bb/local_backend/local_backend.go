package local_backend

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	hash2 "hash"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alessio/shellescape"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type VerboseOutput bool

type JSONOutput bool

type LocalBackendConfig struct {
	JSON    JSONOutput
	Verbose VerboseOutput
}

// LocalBackendRequestContext provides a BaseURL() function returning a fake URL that can be used for document
// construction.
// Note that the resulting URLs in documents can not be followed since there's no server listening.
type LocalBackendRequestContext struct{}

// NewLocalBackendRequestContext creates a new request context that is suitable for returning from the LocalBackend
// object. The URLs produced using the BaseURL() method on this context will not actually work, since there is no
// real HTTP server in the local backend.
func NewLocalBackendRequestContext() routes.RequestContext {
	return &LocalBackendRequestContext{}
}

func (c *LocalBackendRequestContext) BaseURL() string {
	return "http://localhost/"
}

type LocalBackend struct {
	legalEntityService services.LegalEntityService
	queueService       services.QueueService
	stepService        services.StepService
	artifactService    services.ArtifactService
	logService         services.LogService
	runnerService      services.RunnerService
	repoService        services.RepoService
	jobStore           store.JobStore
	commitStore        store.CommitStore
	log                logger.Log
	config             *LocalBackendConfig
	// State
	buildID      models.BuildID
	build        *dto.BuildGraph
	buildMu      sync.RWMutex // protects build only
	failedJobs   []*models.Job
	failedJobsMu sync.Mutex // protects failedJobs only
	legalEntity  *models.LegalEntity
	runner       *models.Runner
	spinners     *BBSpinnerManager
}

func NewLocalBackend(
	legalEntityService services.LegalEntityService,
	queueService services.QueueService,
	stepService services.StepService,
	artifactService services.ArtifactService,
	logService services.LogService,
	runnerService services.RunnerService,
	repoService services.RepoService,
	jobStore store.JobStore,
	commitStore store.CommitStore,
	logFactory logger.LogFactory,
	config *LocalBackendConfig,
) *LocalBackend {
	return &LocalBackend{
		legalEntityService: legalEntityService,
		queueService:       queueService,
		stepService:        stepService,
		artifactService:    artifactService,
		logService:         logService,
		runnerService:      runnerService,
		repoService:        repoService,
		jobStore:           jobStore,
		commitStore:        commitStore,
		config:             config,
		log:                logFactory("LocalBackend"),
	}
}

// Start the local backend. Call Stop() to gracefully cleanup.
func (s *LocalBackend) Start() error {
	return nil
}

// Stop the local backend, freeing all resources and flushing state to disk.
func (s *LocalBackend) Stop() error {
	return nil
}

// Results returns nil if all jobs and steps completed successfully, or a list
// of accumulated errors. Call this after running all jobs.
func (s *LocalBackend) Results() []*models.Job {
	s.failedJobsMu.Lock()
	defer s.failedJobsMu.Unlock()
	if s.spinners != nil {
		s.spinners.Stop()
	}
	return s.failedJobs
}

// Enqueue queues all jobs/steps found in the build configuration file in the current working directory.
func (s *LocalBackend) Enqueue(ctx context.Context, opts *models.BuildOptions) (*dto.BuildGraph, error) {
	now := models.NewTime(time.Now())
	root, err := s.locateGitRoot()
	if err != nil {
		return nil, err
	}
	// A bunch of code below and in the local runner expects to be executing in the root of the repo
	if err := os.Chdir(root); err != nil {
		return nil, fmt.Errorf("error changing current working directory to %q: %w", root, err)
	}
	gRepo, err := git.PlainOpen(root)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening git repo")
	}
	gRef, err := gRepo.Head()
	if err != nil {
		return nil, errors.Wrap(err, "error reading HEAD ref")
	}
	gCommit, err := gRepo.CommitObject(gRef.Hash())
	if err != nil {
		return nil, errors.Wrap(err, "error reading HEAD commit")
	}

	legalEntityExternalID := models.NewExternalResourceID("local", "1")
	legalEntity, _, _, err := s.legalEntityService.Upsert(
		context.Background(),
		nil,
		models.NewPersonLegalEntityData(
			"todo",
			"TODO",
			"todo@todo.com",
			&legalEntityExternalID,
			"",
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error creating legal entity")
	}

	var runner *models.Runner
	runners, _, err := s.runnerService.Search(ctx, nil, models.NoIdentity, models.RunnerSearch{
		LegalEntityID: &legalEntity.ID, Pagination: models.Pagination{Limit: 1}})
	if err != nil {
		return nil, errors.Wrap(err, "error searching runners")
	}
	if len(runners) > 0 {
		runner = runners[0]
	} else {
		runner = models.NewRunner(
			now,
			"BB-runner",
			legalEntity.ID,
			"(bb internal)",
			runtime.GOOS,
			runtime.GOARCH,
			nil, // this field gets updated when runner updates its runtime info
			nil, // no labels need to be specified
			true,
		)
		err = s.runnerService.Create(context.Background(), nil, runner, nil)
		if err != nil {
			return nil, errors.Wrap(err, "error creating runner")
		}
	}

	repoExternalID := models.ExternalResourceID{
		ExternalSystem: "local",
		ResourceID:     "foo", // TODO this needs to work for multiple repos on the same machine, perhaps use git remote url?
	}
	repo := models.NewRepo(now, "fake", legalEntity.ID, "", "ssh://localhost", "", "", "master", true, true, nil, &repoExternalID, "")

	_, _, err = s.repoService.Upsert(ctx, nil, repo)
	if err != nil {
		return nil, errors.Wrap(err, "error upserting repo")
	}

	files, err := ioutil.ReadDir(".")
	if err != nil {
		return nil, errors.Wrap(err, "error listing files")
	}

	var (
		configType     models.ConfigType
		configFilePath string
	)

loop:
	for _, file := range files {

		if file.IsDir() {
			continue
		}

		path := file.Name()

		for _, p := range parser.YAMLBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeYAML
				configFilePath = path
				break loop
			}
		}

		for _, p := range parser.JSONBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeJSON
				configFilePath = path
				break loop
			}
		}

		for _, p := range parser.JSONNETBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeJSONNET
				configFilePath = path
				break loop
			}
		}
	}

	if configFilePath == "" {
		return nil, errors.New("Unable to locate buildbeaver config file in root of repo")
	}

	config, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading build configuration file %q: %w", configFilePath, err)
	}
	err = s.queueService.CheckBuildConfigLength(len(config))
	if err != nil {
		return nil, fmt.Errorf("error reading build configuration file %q: %w", configFilePath, err)
	}

	sha := gCommit.Hash.String()
	commit, err := s.commitStore.ReadBySHA(ctx, nil, repo.ID, sha)
	if err != nil {
		if gerror.IsNotFound(err) {
			commit = models.NewCommit(
				now,
				repo.ID,
				config,
				configType,
				sha,
				gCommit.Message,
				legalEntity.ID,
				"TODO",
				"todo@todo.com",
				legalEntity.ID,
				"TODO",
				"todo@todo.com",
				"")
			err = s.commitStore.Create(ctx, nil, commit)
			if err != nil {
				return nil, errors.Wrap(err, "error creating commit")
			}
		} else if err != nil {
			return nil, errors.Wrap(err, "error reading commit")
		}
	} else {
		commit.Config = config
		commit.ConfigType = configType
		err = s.commitStore.Update(ctx, nil, commit)
		if err != nil {
			return nil, errors.Wrap(err, "error updating commit")
		}
	}

	ref := gRef.Name().String()

	build, err := s.queueService.EnqueueBuildFromCommit(ctx, nil, commit, ref, opts)
	if err != nil {
		return nil, errors.Wrap(err, "error queuing build")
	}
	if build.Status == models.WorkflowStatusFailed {
		return nil, fmt.Errorf("error queueing build: %w", build.Error)
	}

	s.buildMu.Lock()
	s.build = build
	s.buildMu.Unlock()

	s.buildID = build.ID
	s.legalEntity = legalEntity
	s.runner = runner

	if !s.config.Verbose {
		// Set up spinners for the initial jobs
		s.spinners = NewBBSpinnerManager()
		build.Walk(false, func(jGraph *dto.JobGraph) error {
			s.spinners.FindOrCreateSpinner(jGraph.ID, jGraph.GetFQN(), jGraph.Status)
			return nil
		})
		s.spinners.Start()
	}

	return build, nil
}

func (s *LocalBackend) NewJobsCreated(ctx context.Context, newJobs []*documents.JobGraph) {
	// Re-read and store the entire build
	queuedBuild, err := s.queueService.ReadQueuedBuild(ctx, nil, s.buildID)
	if err != nil {
		s.log.Errorf("Error reading build after new jobs created: %s", err.Error())
		return
	}
	s.buildMu.Lock()
	s.build = queuedBuild.BuildGraph
	s.buildMu.Unlock()

	if s.spinners != nil {
		// Ensure we have spinners for all jobs in the build
		for _, jGraph := range s.build.Jobs {
			s.spinners.FindOrCreateSpinner(jGraph.Job.ID, jGraph.GetFQN(), jGraph.Job.Status)
		}
	}
}

// Dequeue returns the next build job that is ready to be executed, or nil if there are currently no queued builds.
func (s *LocalBackend) Dequeue(ctx context.Context) (*documents.RunnableJob, error) {
	var (
		dequeued *dto.RunnableJob
		err      error
	)
	for {
		dequeued, err = s.queueService.Dequeue(ctx, s.runner.ID)
		if err != nil {
			return nil, err
		}
		// HACK drain and skip jobs from previous builds
		if dequeued.BuildID == s.buildID {
			break
		}
	}
	if !s.config.Verbose && s.spinners != nil {
		s.spinners.UpdateSpinnerStatus(dequeued.ID, dequeued.Status)
	}
	return documents.MakeRunnableJob(NewLocalBackendRequestContext(), dequeued), nil
}

// Ping acts as a pre-flight check for a runner, contacting the server and checking that authentication
// and registration are in place ready to dequeue build jobs.
func (s *LocalBackend) Ping(ctx context.Context) error {
	// A local backend doesn't use network communications or authentication, so just say everything is OK
	return nil
}

// SendRuntimeInfo sends information about the runtime environment and version for this runner to the server.
func (s *LocalBackend) SendRuntimeInfo(ctx context.Context, info *documents.PatchRuntimeInfoRequest) error {
	runner, err := s.runnerService.Read(ctx, nil, s.runner.ID)
	if err != nil {
		return err
	}
	if info.SoftwareVersion != nil {
		runner.SoftwareVersion = *info.SoftwareVersion
	}
	if info.OperatingSystem != nil {
		runner.OperatingSystem = *info.OperatingSystem
	}
	if info.Architecture != nil {
		runner.Architecture = *info.Architecture
	}
	if info.SupportedJobTypes != nil {
		runner.SupportedJobTypes = *info.SupportedJobTypes
	}
	runner.ETag = models.ETagAny
	runner, err = s.runnerService.Update(ctx, nil, runner)
	if err != nil {
		return err
	}
	return nil
}

// UpdateJobStatus updates the status of the specified job.
// If the status is finished, err can be supplied to signal the job failed with an error
// or nil to signify the job succeeded.
func (s *LocalBackend) UpdateJobStatus(
	ctx context.Context,
	jobID models.JobID,
	status models.WorkflowStatus,
	jobError *models.Error,
	eTag models.ETag) (*documents.Job, error) {

	job, err := s.queueService.UpdateJobStatus(ctx, nil, jobID, dto.UpdateJobStatus{
		Status: status,
		Error:  jobError,
		ETag:   eTag,
	})
	if err != nil {
		return nil, err
	}

	if jobError != nil {
		s.failedJobsMu.Lock()
		defer s.failedJobsMu.Unlock()
		s.failedJobs = append(s.failedJobs, job)
	}

	if !s.config.Verbose && s.spinners != nil {
		s.spinners.UpdateSpinnerStatus(job.ID, job.Status)
	}

	return documents.MakeJob(NewLocalBackendRequestContext(), job), nil
}

// UpdateStepStatus updates the status of the specified step.
// If the status is finished, err can be supplied to signal the step failed with an error
// or nil to signify the step succeeded.
func (s *LocalBackend) UpdateStepStatus(
	ctx context.Context,
	stepID models.StepID,
	status models.WorkflowStatus,
	stepError *models.Error,
	eTag models.ETag) (*documents.Step, error) {

	step, err := s.queueService.UpdateStepStatus(ctx, nil, stepID, dto.UpdateStepStatus{
		Status: status,
		Error:  stepError,
		ETag:   eTag,
	})
	if err != nil {
		return nil, err
	}

	return documents.MakeStep(NewLocalBackendRequestContext(), step), nil
}

// UpdateJobFingerprint sets the fingerprint that has been calculated for a job. If the build is not configured
// with the force option (e.g. force=false), the server will attempt to locate a previously successful job with a
// matching fingerprint and indirect this job to it. If an indirection has been set, the agent must skip the job.
func (s *LocalBackend) UpdateJobFingerprint(
	ctx context.Context,
	jobID models.JobID,
	jobFingerprint string,
	jobFingerprintHashType *models.HashType,
	eTag models.ETag) (*documents.Job, error) {

	job, err := s.queueService.UpdateJobFingerprint(
		ctx,
		jobID,
		dto.UpdateJobFingerprint{
			Fingerprint:         jobFingerprint,
			FingerprintHashType: *jobFingerprintHashType,
			ETag:                eTag,
		})
	if err != nil {
		return nil, err
	}
	if job.IndirectToJobID.Valid() {
		indirectedJob, err := s.jobStore.Read(ctx, nil, job.IndirectToJobID)
		if err != nil {
			return nil, err
		}
		search := models.NewArtifactSearch()
		search.Workflow = &indirectedJob.Workflow
		search.JobName = &indirectedJob.Name
		paginator, err := s.SearchArtifacts(ctx, indirectedJob.BuildID, search)
		if err != nil {
			return nil, errors.Wrap(err, "error searching artifacts")
		}
		for paginator.HasNext() {
			artifacts, err := paginator.Next(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "error getting next set of artifact search failedJobs")
			}
			for _, artifact := range artifacts {
				err := s.verifyArtifact(artifact)
				if err != nil {
					s.log.Warnf("error verifying artifact %q; fingerprint matching will be overridden: %v", artifact.Path, err)
					job.IndirectToJobID = models.JobID{}
					break
				}
			}
		}
	}
	return documents.MakeJob(NewLocalBackendRequestContext(), job), nil
}

// verifyArtifact verifies that the specified artifact exists on the local file system.
func (s *LocalBackend) verifyArtifact(artifact *models.Artifact) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error determining current working directory")
	}
	absolutePath := filepath.Join(cwd, artifact.Path)
	file, err := os.Open(absolutePath)
	if err != nil {
		return err
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if uint64(stat.Size()) != artifact.Size {
		return errors.New("error artifact size mismatch")
	}
	var hash hash2.Hash
	switch artifact.HashType {
	case models.HashTypeBlake2b:
		hash, err = blake2b.New256(nil)
		if err != nil {
			return fmt.Errorf("error initializing hash: %w", err)
		}
	case models.HashTypeMD5:
		hash = md5.New()
	default:
		return fmt.Errorf("error unsupported hash type: %s", artifact.HashType)
	}
	_, err = io.Copy(hash, file)
	if err != nil {
		return fmt.Errorf("error reading artifact file: %w", err)
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))
	if artifact.Hash != hashStr {
		return errors.New("error artifact hash mismatch")
	}
	return nil
}

// GetSecretsPlaintext gets all secrets for the specified repo in plaintext.
func (s *LocalBackend) GetSecretsPlaintext(ctx context.Context, repoID models.RepoID) ([]*models.SecretPlaintext, error) {
	// We don't have any secret storage when running local builds so instead source them from environment variables.
	// We need to know the resource_links of the secrets that steps are interested in first though.
	// We need to know the resource_links of the secrets that steps are interested in first though.
	secretNames := make(map[string]bool)

	s.buildMu.RLock()
	for _, job := range s.build.Jobs {
		for _, env := range job.Environment {
			if env.ValueFromSecret != "" {
				secretNames[strings.ToUpper(env.ValueFromSecret)] = true
			}
		}
	}
	s.buildMu.RUnlock()

	var (
		now     = time.Now().UTC()
		secrets []*models.SecretPlaintext
	)
	for _, pair := range os.Environ() {
		split := strings.SplitN(pair, "=", 2)
		if len(split) != 2 {
			s.log.Warnf("Ignoring malformed env var when generating secrets: %s", pair)
			continue
		}
		key := split[0]
		value := split[1]

		// Does this environment variable match a secret?
		_, ok := secretNames[strings.ToUpper(key)]
		if !ok {
			continue
		}
		secret := &models.SecretPlaintext{
			Secret: &models.Secret{
				ID:               models.NewSecretID(),
				Name:             models.ResourceName(key),
				RepoID:           repoID,
				CreatedAt:        models.NewTime(now),
				UpdatedAt:        models.NewTime(now),
				ETag:             "",
				KeyEncrypted:     nil,
				ValueEncrypted:   nil,
				DataKeyEncrypted: nil,
				IsInternal:       false,
			},
			Key:   key,
			Value: value,
		}
		secrets = append(secrets, secret)
		s.log.Infof("Generated secret from env var %s", key)
	}

	return secrets, nil
}

// CreateArtifact a new artifact with its contents provided by reader. It is the caller's responsibility to close reader.
// Returns store.ErrAlreadyExists if an artifact with matching unique properties already exists.
func (s *LocalBackend) CreateArtifact(
	ctx context.Context,
	jobID models.JobID,
	groupName models.ResourceName,
	relativePath string,
	reader io.ReadSeeker,
) (*documents.Artifact, error) {
	artifact, err := s.artifactService.Create(
		ctx,
		jobID,
		groupName,
		relativePath,
		"", // don't check the MD5 hash since there's no network hop
		reader,
		false, // don't store the data in a blob since it is already in the local filesystem
	)
	if err != nil {
		return nil, err
	}
	return documents.MakeArtifact(NewLocalBackendRequestContext(), artifact), nil
}

// GetArtifactData returns a reader to the data of an artifact.
// It is the caller's responsibility to close the reader.
func (s *LocalBackend) GetArtifactData(ctx context.Context, artifactID models.ArtifactID) (io.ReadCloser, error) {
	// We always expect the runner to find that artifacts already
	// exist at the desired path on the filesystem - they must do as all
	// parent jobs/steps must have executed to produce the artifacts
	// before a dependent step can run and request an artifact.
	return nil, errors.New("error not implemented")
}

// GetArtifactLocalData returns a reader to the data of an artifact, reading the file from the local filesystem.
// It is the caller's responsibility to close the reader.
func (s *LocalBackend) GetArtifactLocalData(artifact *models.Artifact) (io.ReadCloser, error) {
	// artifact file should already exist at the filesystem path specified in the artifact resource,
	// relative to the current working directory
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "error determining current working directory")
	}
	absolutePath := path.Join(currentWorkingDir, artifact.Path)

	file, err := os.Open(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("error opening artifact file for reading from local file system, file name '%s': %w", absolutePath, err)
	}
	return file, nil
}

// SearchArtifacts searches all artifacts for a build. Use cursor to page through failedJobs, if any.
func (s *LocalBackend) SearchArtifacts(ctx context.Context, buildID models.BuildID, search *models.ArtifactSearch) (models.ArtifactSearchPaginator, error) {
	return models.NewArtifactPager(search.Pagination, func(ctx context.Context, pagination models.Pagination) ([]*models.Artifact, *models.Cursor, error) {
		search.BuildID = buildID
		search.Pagination = pagination
		return s.artifactService.Search(ctx, nil, models.NoIdentity, *search)
	}), nil
}

// OpenLogWriteStream writes everything in reader to a log descriptor.
func (s *LocalBackend) OpenLogWriteStream(ctx context.Context, logDescriptorID models.LogDescriptorID) (io.WriteCloser, error) {

	paddingLen := 0
	s.buildMu.RLock()
	for _, job := range s.build.Jobs {
		for _, step := range job.Steps {
			prefixLen := len(s.makeLogLinePrefix(job.Workflow, job.Name, step.Name))
			if prefixLen > paddingLen {
				paddingLen = prefixLen
			}
		}
	}
	s.buildMu.RUnlock()

	desc, err := s.logService.Read(ctx, nil, logDescriptorID)
	if err != nil {
		return nil, fmt.Errorf("error reading log descriptor: %w", err)
	}
	var (
		job  *models.Job
		step *models.Step
	)
	prefix := ""
	switch desc.ResourceID.Kind() {
	case models.JobResourceKind:
		job, err = s.jobStore.Read(ctx, nil, models.JobIDFromResourceID(desc.ResourceID))
		if err != nil {
			return nil, fmt.Errorf("error reading ljob: %w", err)
		}
		prefix = s.makeLogLinePrefix(job.Workflow, job.Name, "")
	case models.StepResourceKind:
		step, err = s.stepService.Read(ctx, nil, models.StepIDFromResourceID(desc.ResourceID))
		if err != nil {
			return nil, fmt.Errorf("error reading step: %w", err)
		}
		job, err = s.jobStore.Read(ctx, nil, step.JobID)
		if err != nil {
			return nil, fmt.Errorf("error reading ljob: %w", err)
		}
		prefix = s.makeLogLinePrefix(job.Workflow, job.Name, step.Name)
	}
	if paddingLen > len(prefix) {
		prefix = fmt.Sprintf("%s%s", prefix, strings.Repeat(" ", paddingLen-len(prefix)))
	}
	logDescReader, logDescWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	in := &util.MultiReaderCloser{
		Writer:  io.MultiWriter(logDescWriter, stdoutWriter),
		Writers: []io.WriteCloser{logDescWriter, logDescWriter},
	}
	go func() {
		err := s.logService.WriteData(ctx, logDescriptorID, logDescReader)
		logDescReader.CloseWithError(err)
	}()
	go func() {
		err := s.streamLogToStdout(desc, prefix, job, step, stdoutReader)
		stdoutWriter.CloseWithError(err)
	}()

	return in, nil
}

func (s *LocalBackend) streamLogToStdout(desc *models.LogDescriptor, prefix string, job *models.Job, stepOrNil *models.Step, in io.Reader) error {
	if bool(s.config.Verbose) && bool(s.config.JSON) {
		_, err := io.Copy(os.Stdout, in)
		return err
	}
	dec := json.NewDecoder(in)
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("error reading opening token: %w", err)
	}
	if token != json.Delim('[') {
		return fmt.Errorf("error expected first token to begin array (\"[\"), found: %s", token)
	}
	for dec.More() {
		entry := &models.LogEntry{}
		err := dec.Decode(entry)
		if err != nil {
			return fmt.Errorf("error unmarshalling entry from JSON: %w", err)
		}
		if !s.config.Verbose && s.spinners != nil {
			plaintext, ok := entry.Derived().(models.PlainTextLogEntry)
			if !ok {
				continue
			}
			s.spinners.UpdateSpinnerText(job.ID, shellescape.StripUnsafe(strings.Trim(plaintext.GetText(), " \r\n\t")))
		} else {
			plaintext, ok := entry.Derived().(models.PlainTextLogEntry)
			if !ok {
				continue
			}
			_, err = os.Stdout.Write([]byte(prefix + plaintext.GetText() + "\n"))
			if err != nil {
				return fmt.Errorf("error writing to stdout: %w", err)
			}
		}
	}
	_, err = dec.Token()
	if err != nil {
		return fmt.Errorf("error reading closing token: %w", err)
	}
	return nil
}

func (s *LocalBackend) makeLogLinePrefix(workflowName models.ResourceName, jobName models.ResourceName, stepName models.ResourceName) string {
	workflowPrefix := ""
	if workflowName != "" {
		workflowPrefix = workflowName.String() + "."
	}
	stepNameSuffix := ""
	if stepName == "" {
		stepNameSuffix = "." + stepName.String()
	}
	return fmt.Sprintf("%s%s%s: ", workflowPrefix, jobName, stepNameSuffix)
}

// locateGitRoot walks up the directory tree starting at the current working directory looking for a .git
// directory representing a git repo. Returns the path to the first directory found that contains a .git subdir,
// or an error if we reached the root without finding one.
func (s *LocalBackend) locateGitRoot() (string, error) {
	path, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	for {
		gitDir := filepath.Join(path, ".git")
		info, err := os.Stat(gitDir)
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("error stating dir %v: %w", gitDir, err)
		}
		if info != nil && info.IsDir() && err == nil { // found it
			return path, nil
		}
		if filepath.Join(filepath.VolumeName(path), string(os.PathSeparator)) == path {
			return "", fmt.Errorf("error locating git repository (in current working directory or any of the parent directories)")
		}
		path = filepath.Clean(filepath.Join(path, ".."))
	}
}
