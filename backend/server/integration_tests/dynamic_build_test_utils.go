package integration_tests

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"
	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/buildbeaver/sdk/dynamic/bb/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

// JobinatorTestEnv provides information about the context/environment in which a jobinator job should be run
// during testing.
type JobinatorTestEnv struct {
	*testing.T
	// App is the test server which is being used to track state during the test
	App *server_test.TestServer
	// Job is the Jobinator job being executed
	Job *documents.RunnableJob
	// BuildID is the server-side ID for the build, for passing to services
	BuildID models.BuildID
	// Env is the environment provided to the jobinator (equivalent to the env vars provided to a Jobinator process)
	Env map[string]string
	// JobsToCompleteChan is a channel that can be used by test Jobinators to request that jobs are updated to complete
	JobsToCompleteChan chan JobCompletionRequest
}

func NewJobinatorTestEnv(
	t *testing.T,
	app *server_test.TestServer,
	job *documents.RunnableJob,
	env map[string]string,
) *JobinatorTestEnv {
	return &JobinatorTestEnv{
		T:                  t,
		App:                app,
		Job:                job,
		BuildID:            job.Job.BuildID,
		Env:                env,
		JobsToCompleteChan: make(chan JobCompletionRequest, 10),
	}
}

var dynamicJobDef = models.JobDefinition{
	JobDefinitionData: models.JobDefinitionData{
		Name:                    "dynamic-job",
		Type:                    "docker",
		RunsOn:                  nil, // any runner
		DockerImage:             "golang:1.18",
		DockerImagePullStrategy: models.DockerPullStrategyDefault,
		StepExecution:           models.StepExecutionSequential,
	},
	Steps: []models.StepDefinition{{
		StepDefinitionData: models.StepDefinitionData{
			Name: "run-dynamic-build-job",
			Commands: models.Commands{
				"echo 'hello world'",
			},
		},
	}},
}

func createPrerequisiteObjects(t *testing.T, app *server_test.TestServer) (*models.Commit, *models.Runner) {
	ctx := context.Background()

	// Create the objects required in order to have context for a build
	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)

	// We need a runner or all jobs will be rejected because there is no runner capable of running the job
	buildRunner := server_test.CreateRunner(t, ctx, app, "basic-runner", legalEntity.ID, nil)
	buildRunner.Labels = models.Labels{"linux", "macos", "arm64", "amd64"}
	_, err := app.RunnerService.Update(ctx, nil, buildRunner)
	require.NoError(t, err)
	return commit, buildRunner
}

// enqueueDynamicBuild calls app.QueueService to enqueue a new build with a single dynamic build job.
// Returns the build graph returned by QueueService.EnqueueBuildFromBuildDefinition().
func enqueueDynamicBuild(t *testing.T, app *server_test.TestServer, commit *models.Commit, opts *models.BuildOptions) *dto.BuildGraph {
	ctx := context.Background()

	buildDef := &models.BuildDefinition{
		Jobs: []models.JobDefinition{
			dynamicJobDef,
		}}

	build, err := app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, commit.RepoID, commit.ID, buildDef, "refs/heads/master", opts)
	require.NoError(t, err)
	if len(build.Jobs) > 0 {
		for i, job := range build.Jobs {
			if job.Error != nil {
				assert.NoError(t, job.Error, "Job %d of %d failed with error: %v", i+1, len(build.Jobs), job.Error)
			}
		}
	}
	if build.Error != nil {
		require.NoError(t, build.Error, "Got build error: %s", build.Error.Error())
	}
	require.Equal(t, models.WorkflowStatusQueued, build.Status)
	require.False(t, build.ID.IsZero())

	return build
}

// dequeueJob dequeues a job for the specified build runner.
// If expectedJobName is not empty then the dequeued job must have the specified job name.
// Returns the job, plus a suitable environment to run it in (as would be provided by a runner).
func dequeueJob(
	t *testing.T,
	app *server_test.TestServer,
	buildRunnerID models.RunnerID,
	expectedJobName models.ResourceName,
) (*documents.RunnableJob, map[string]string) {
	ctx := context.Background()

	job, err := app.QueueService.Dequeue(ctx, buildRunnerID)
	require.NoError(t, err)
	if expectedJobName != "" {
		require.Equal(t, expectedJobName, job.GetName())
	}

	// Make up a request context that points back to the running test server, to use in URL construction during
	// document creation
	requestCtx := server_test.NewTestServerRequestContext(app)

	runnableJob := documents.MakeRunnableJob(requestCtx, job)

	// Make environment populated with standard env variables as would be provided by a runner
	env := make(map[string]string)
	runner.AddStandardGlobalEnvVars(
		runnableJob,
		dynamic_api.Endpoint(app.CoreAPIServer.GetServerURL()), // core API server provides the dynamic API
		func(name string, value string, isSecret bool) { env[name] = value },
	)

	return runnableJob, env
}

func clientBuildIDToModelsBuildID(t *testing.T, clientBuildID bb.BuildID) models.BuildID {
	resourceID, err := models.ParseResourceID(clientBuildID.String())
	require.NoError(t, err, "Unable to parse Build ID '%s'", clientBuildID.String())
	return models.BuildIDFromResourceID(resourceID)
}

// checkJobCount checks the number of jobs in the database which are part of the build in the Jobinator environment.
func checkJobCount(t *JobinatorTestEnv, expectedJobCount int) {
	checkJobCountForBuild(t, t.BuildID, expectedJobCount)
}

// checkJobCount checks the number of jobs in the database for  a particular specified build.
func checkJobCountForBuild(t *JobinatorTestEnv, buildID models.BuildID, expectedJobCount int) {
	ctx := context.Background()
	jobs, err := t.App.JobService.ListByBuildID(ctx, nil, buildID)
	require.NoError(t, err, "Error listing jobs for build")
	require.Equal(t, expectedJobCount, len(jobs), "Unexpected number of jobs found for build")
}

// checkJob checks job data submitted from the client against a JobGraph read directly from the server's database.
func checkJob(t *testing.T, job *bb.Job, jobFromServer *dto.JobGraph) {
	jobData := job.GetData()

	// Check basic fields
	assert.Equal(t, job.GetName().String(), jobFromServer.Name.String())
	if jobData.Description != nil {
		assert.Equal(t, *jobData.Description, jobFromServer.Description)
	} else {
		assert.Empty(t, jobFromServer.Description)
	}
	if jobData.Type != nil {
		assert.Equal(t, *jobData.Type, jobFromServer.Type.String())
	}
	if jobData.StepExecution == "" {
		assert.Equal(t, bb.StepExecutionSequential.String(), jobFromServer.StepExecution.String())
	} else {
		assert.Equal(t, jobData.StepExecution, jobFromServer.StepExecution.String())
	}

	// Check labels
	assert.Equal(t, len(jobData.RunsOn), len(jobFromServer.RunsOn))
	for i := range jobData.RunsOn {
		assert.Equal(t, jobData.RunsOn[i], jobFromServer.RunsOn[i].String())
	}

	// Check docker configuration
	assert.Equal(t, jobData.Docker.Image, jobFromServer.DockerImage)
	assert.Equal(t, jobData.Docker.Pull, jobFromServer.DockerImagePullStrategy.String())
	assert.Equal(t, jobData.Docker.Shell, jobFromServer.DockerShell)
	if jobFromServer.DockerAuth != nil {
		require.NotNil(t, jobData.Docker)
		if jobFromServer.DockerAuth.Basic != nil {
			require.NotNil(t, jobData.Docker.BasicAuth)
			checkEnvValue(t, &jobData.Docker.BasicAuth.Username, jobFromServer.DockerAuth.Basic.Username)
			checkEnvValue(t, &jobData.Docker.BasicAuth.Password, jobFromServer.DockerAuth.Basic.Password)
		} else {
			require.Nil(t, jobData.Docker.BasicAuth)
		}
	} else if jobData.Docker != nil {
		require.Nil(t, jobData.Docker.BasicAuth)
		require.Nil(t, jobData.Docker.AwsAuth)
	}

	// Check all fingerprint commands match
	assert.Equal(t, len(jobData.Fingerprint), len(jobFromServer.FingerprintCommands))
	for i := range jobData.Fingerprint {
		assert.Equal(t, jobData.Fingerprint[i], jobFromServer.FingerprintCommands[i].String())
	}

	// Check all service definitions match
	assert.Equal(t, len(jobData.Services), len(jobFromServer.Services))
	serviceMap := makeServiceMap(jobFromServer.Services)
	for _, service := range jobData.Services {
		serviceFromServer := serviceMap[service.Name]
		require.NotNil(t, serviceFromServer, "Service with name '%s' not found on server under job '%s'", service.Name, job.GetName())
		checkService(t, &service, serviceFromServer, job)
	}

	// Check all dependencies match
	assert.Equal(t, len(jobData.Depends), len(jobFromServer.Depends))
	dependencyMap := makeDependencyMap(jobFromServer.Depends)
	for _, dependency := range jobData.Depends {
		dependencyFQN := parser.WorkflowDependencyFromString(dependency)
		require.NotEmptyf(t, dependencyFQN.JobName, "Unable to extract job name from dependency '%s'", dependency)
		dependencyFromServer := dependencyMap[dependencyFQN]
		require.NotNil(t, dependencyFromServer, "Dependency for job '%s' depending on job '%s' not found on server", job.GetName(), dependencyFQN)
		// TODO: Perform better checking of artifact dependencies once we support the full syntax
	}

	// Check all steps match
	assert.Equal(t, len(jobData.Steps), len(jobFromServer.Steps))
	stepMap := makeStepMap(jobFromServer.Steps)
	for _, step := range jobData.Steps {
		stepFromServer := stepMap[step.Name]
		require.NotNil(t, stepFromServer, "Step name '%s' not found on server under job '%s'", step.Name, job.GetName())
		checkStep(t, &step, stepFromServer, job)
	}

	// Check all artifact definitions
	assert.Equal(t, len(jobData.Artifacts), len(jobFromServer.ArtifactDefinitions))
	artifactMap := makeArtifactMap(jobFromServer.ArtifactDefinitions)
	for _, artifact := range jobData.Artifacts {
		artifactFromServer := artifactMap[artifact.Name]
		require.NotNil(t, artifactFromServer, "Artifact group name '%s' for job '%s' not found on server", artifact.Name, job.GetName())
		assert.Equal(t, len(artifact.Paths), len(artifactFromServer.Paths))
		for i := range artifact.Paths {
			assert.Equal(t, artifact.Paths[i], artifactFromServer.Paths[i], "Artifact path does not match artifact path on server for path %d of artifact group '%s', job '%s'", i, artifact.Name, job.GetName())
		}
	}

	// Check all env variables match
	checkEnvironment(t, jobData.Environment, jobFromServer.Environment, fmt.Sprintf("job '%s'", job.GetName()))
}

// checkJobFromAPI checks job data submitted from the client against a JobGraph read back through the server's API.
func checkJobFromAPI(t *testing.T, job *bb.Job, jGraph *client.JobGraph) {
	jobData := job.GetData()

	// Check basic fields
	assert.Equal(t, job.GetName().String(), jGraph.Job.Name)
	if jobData.Description != nil {
		assert.Equal(t, *jobData.Description, jGraph.Job.Description)
	}
	if jobData.Type != nil {
		assert.Equal(t, *jobData.Type, jGraph.Job.Type)
	}
	if jobData.StepExecution == "" {
		assert.Equal(t, bb.StepExecutionSequential.String(), jGraph.Job.StepExecution)
	} else {
		assert.Equal(t, jobData.StepExecution, jGraph.Job.StepExecution)
	}

	// Check labels
	assert.Equal(t, len(jobData.RunsOn), len(jGraph.Job.RunsOn))
	for i := range jobData.RunsOn {
		assert.Equal(t, jobData.RunsOn[i], jGraph.Job.RunsOn[i])
	}

	// Check docker configuration
	if jobData.Docker != nil {
		require.NotNil(t, jGraph.Job.Docker, "Docker config is nil in job from server")
		assert.Equal(t, jobData.Docker.Image, jGraph.Job.Docker.Image)
		assert.Equal(t, jobData.Docker.Pull, jGraph.Job.Docker.Pull)
		assert.Equal(t, jobData.Docker.Shell, jGraph.Job.Docker.Shell)
		if jobData.Docker.BasicAuth != nil {
			require.NotNil(t, jGraph.Job.Docker.BasicAuth, "Docker basic auth is nil in job from server")
			checkEnvValueFromAPI(t, &jobData.Docker.BasicAuth.Username, &jGraph.Job.Docker.BasicAuth.Username)
			checkEnvValueFromAPI(t, &jobData.Docker.BasicAuth.Password, &jGraph.Job.Docker.BasicAuth.Password)
		}
	} else {
		assert.Nil(t, jGraph.Job.Docker)
	}

	// Check all fingerprint commands match
	assert.Equal(t, len(jobData.Fingerprint), len(jGraph.Job.FingerprintCommands))
	for i := range jobData.Fingerprint {
		assert.Equal(t, jobData.Fingerprint[i], jGraph.Job.FingerprintCommands[i])
	}

	assert.Empty(t, jGraph.Job.Fingerprint)
	assert.Nil(t, jGraph.Job.Error)
	if jGraph.Job.Error != nil {
		t.Logf("Got error back from job graph: %s", *jGraph.Job.Error)
	}

	// Check all service definitions match
	assert.Equal(t, len(jobData.Services), len(jGraph.Job.Services))
	serviceMap := makeServiceMapFromAPI(jGraph.Job.Services)
	for _, service := range jobData.Services {
		serviceFromAPI := serviceMap[service.Name]
		require.NotNil(t, serviceFromAPI, "Service with name '%s' not found on server under job '%s'", service.Name, job.GetName())
		checkServiceFromAPI(t, &service, serviceFromAPI, job)
	}

	// Check all dependencies match
	assert.Equal(t, len(jobData.Depends), len(jGraph.Job.Depends))
	dependencyMap := makeDependencyMapFromAPI(jGraph.Job.Depends)
	for _, dependency := range jobData.Depends {
		dependencyFQN := parser.WorkflowDependencyFromString(dependency)
		require.NotEmptyf(t, dependencyFQN.JobName, "Unable to extract job name from dependency '%s'", dependency)
		dependencyFromServer := dependencyMap[dependencyFQN]
		require.NotNil(t, dependencyFromServer, "Dependency for job '%s' depending on job '%s' not found on server", job.GetName(), dependencyFQN)
		// TODO: Perform better checking of artifact dependencies once we support the full syntax
	}

	// Check all steps match
	assert.Equal(t, len(jobData.Steps), len(jGraph.Steps))
	stepMap := makeStepMapFromAPI(jGraph.Steps)
	for _, step := range jobData.Steps {
		stepFromServer := stepMap[step.Name]
		require.NotNil(t, stepFromServer, "Step name '%s' not found on server under job '%s'", step.Name, job.GetName())
		checkStepFromAPI(t, &step, stepFromServer, job)
	}

	// Check all artifact definitions
	assert.Equal(t, len(jobData.Artifacts), len(jGraph.Job.Artifacts))
	artifactMap := makeArtifactMapFromAPI(jGraph.Job.Artifacts)
	for _, artifact := range jobData.Artifacts {
		artifactFromAPI := artifactMap[artifact.Name]
		require.NotNil(t, artifactFromAPI, "Artifact group name '%s' for job '%s' not found on server", artifact.Name, job.GetName())
		assert.Equal(t, len(artifact.Paths), len(artifactFromAPI.Paths))
		for i := range artifact.Paths {
			assert.Equal(t, artifact.Paths[i], artifactFromAPI.Paths[i], "Artifact path does not match artifact path on server for path %d of artifact group '%s', job '%s'", i, artifact.Name, job.GetName())
		}
	}

	// Check all env variables match
	checkEnvironmentFromAPI(t, jobData.Environment, jGraph.Job.Environment, fmt.Sprintf("job '%s'", job.GetName()))
}

func checkStep(t *testing.T, step *client.StepDefinition, stepFromServer *models.Step, job *bb.Job) {
	jobData := job.GetData()

	// Check basic fields
	assert.Equal(t, step.Name, stepFromServer.Name.String())
	if step.Description != nil {
		assert.Equal(t, *step.Description, stepFromServer.Description)
	} else {
		assert.Empty(t, stepFromServer.Description)
	}

	// Check all step commands match
	assert.Equal(t, len(step.Commands), len(stepFromServer.Commands))
	for i := range step.Commands {
		assert.Equal(t, step.Commands[i], stepFromServer.Commands[i].String())
	}

	// Check all dependencies
	if jobData.StepExecution == bb.StepExecutionParallel.String() {
		// For parallel step execution the dependencies on the server should match those provided by the client
		assert.Equal(t, len(step.Depends), len(stepFromServer.Depends), "Wrong number of step dependencies for step '%s' job '%s'", step.Name, job.GetName())
		dependencyMap := makeStepDependencyMap(stepFromServer.Depends)
		for _, dependency := range step.Depends {
			dependencyFromServer := dependencyMap[dependency]
			require.NotNil(t, dependencyFromServer, "Dependency for step '%s' depending on step '%s' (within job '%s') not found on server",
				step.Name, dependency, job.GetName())
		}
	} else if jobData.StepExecution == bb.StepExecutionSequential.String() {
		// For sequential step execution there should be no explicit dependencies, and either zero or one implicit
		// dependencies added automatically on the server to put all steps in a sequential list
		assert.Zero(t, len(step.Depends), "Should be no step dependencies explicitly specified when using sequential step execution")
		assert.True(t, len(stepFromServer.Depends) == 0 || len(stepFromServer.Depends) == 1, "Should be either zero or one implicit step dependencies on the server when using sequential step execution")
	}
}

func checkStepFromAPI(t *testing.T, step *client.StepDefinition, stepFromAPI *client.Step, job *bb.Job) {
	jobData := job.GetData()

	// Check basic fields
	assert.Equal(t, step.Name, stepFromAPI.Name)
	if step.Description != nil {
		assert.Equal(t, *step.Description, stepFromAPI.Description)
	} else {
		assert.Empty(t, stepFromAPI.Description)
	}

	// Check all step commands match
	assert.Equal(t, len(step.Commands), len(stepFromAPI.Commands))
	for i := range step.Commands {
		assert.Equal(t, step.Commands[i], stepFromAPI.Commands[i])
	}

	// Check all dependencies
	if jobData.StepExecution == bb.StepExecutionParallel.String() {
		// For parallel step execution the dependencies on the server should match those provided by the client
		assert.Equal(t, len(step.Depends), len(stepFromAPI.Depends), "Wrong number of step dependencies for step '%s' job '%s'", step.Name, job.GetName())
		dependencyMap := makeStepDependencyMapFromAPI(stepFromAPI.Depends)
		for _, dependency := range step.Depends {
			dependencyFromServer := dependencyMap[dependency]
			require.NotNil(t, dependencyFromServer, "Dependency for step '%s' depending on step '%s' (within job '%s') not found on server",
				step.Name, dependency, job.GetName())
		}
	} else if jobData.StepExecution == bb.StepExecutionSequential.String() {
		// For sequential step execution there should be no explicit dependencies, and either zero or one implicit
		// dependencies added automatically on the server to put all steps in a sequential list
		assert.Zero(t, len(step.Depends), "Should be no step dependencies explicitly specified when using sequential step execution")
		assert.True(t, len(stepFromAPI.Depends) == 0 || len(stepFromAPI.Depends) == 1, "Should be either zero or one implicit step dependencies on the server when using sequential step execution")
	}
}

func checkService(t *testing.T, serviceDef *client.ServiceDefinition, serviceFromServer *models.Service, job *bb.Job) {
	// Check basic fields
	assert.Equal(t, serviceDef.Name, serviceFromServer.Name)
	assert.Equal(t, serviceDef.Image, serviceFromServer.DockerImage)

	// Check docker authentication
	if serviceFromServer.DockerRegistryAuthentication != nil {
		if serviceFromServer.DockerRegistryAuthentication.Basic != nil {
			require.NotNil(t, serviceDef.BasicAuth)
			checkEnvValue(t, &serviceDef.BasicAuth.Username, serviceFromServer.DockerRegistryAuthentication.Basic.Username)
			checkEnvValue(t, &serviceDef.BasicAuth.Password, serviceFromServer.DockerRegistryAuthentication.Basic.Password)
		} else {
			require.Nil(t, serviceDef.BasicAuth)
		}
	} else {
		require.Nil(t, serviceDef.BasicAuth)
		require.Nil(t, serviceDef.AwsAuth)
	}

	// Check all env variables match
	checkEnvironment(t, serviceDef.Environment, serviceFromServer.Environment, fmt.Sprintf("job '%s', service '%s'", job.GetName(), serviceDef.Name))
}

func checkServiceFromAPI(t *testing.T, serviceDef *client.ServiceDefinition, serviceFromAPI *client.Service, job *bb.Job) {
	// Check basic fields
	assert.Equal(t, serviceDef.Name, serviceFromAPI.Name)
	assert.Equal(t, serviceDef.Image, serviceFromAPI.Image)

	// Check docker authentication
	if serviceDef.BasicAuth != nil {
		require.NotNil(t, serviceFromAPI.BasicAuth)
		checkEnvValueFromAPI(t, &serviceDef.BasicAuth.Username, &serviceFromAPI.BasicAuth.Username)
		checkEnvValueFromAPI(t, &serviceDef.BasicAuth.Password, &serviceFromAPI.BasicAuth.Password)
	} else {
		require.Nil(t, serviceDef.BasicAuth)
	}

	// Check all env variables match
	checkEnvironmentFromAPI(t, serviceDef.Environment, serviceFromAPI.Environment, fmt.Sprintf("job '%s', service '%s'", job.GetName(), serviceDef.Name))
}

func checkEnvironment(t *testing.T, env map[string]client.SecretStringDefinition, envFromServer []*models.EnvVar, description string) {
	assert.Equal(t, len(env), len(envFromServer))
	// Iterate through server env since we already have a map in the client data to look things up
	for _, serverEnvVar := range envFromServer {
		clientEnvVar, found := env[serverEnvVar.Name]
		require.True(t, found, "Environment variable name '%s' found on server but not on client - %s",
			serverEnvVar.Name, description)

		if clientEnvVar.Value != nil {
			assert.Equal(t, *clientEnvVar.Value, serverEnvVar.Value, "Env variable explicit value on server doesn't match for variable name '%s' - %s",
				serverEnvVar.Name, description)
		} else {
			assert.Empty(t, serverEnvVar.Value, "Env variable explicit value on server should be empty for variable name '%s' - %s",
				serverEnvVar.Name, description)
		}

		if clientEnvVar.FromSecret != nil {
			assert.Equal(t, *clientEnvVar.FromSecret, serverEnvVar.ValueFromSecret, "Env variable secret name on server doesn't match, for variable name '%s' - %s",
				serverEnvVar.Name, description)
		} else {
			assert.Empty(t, serverEnvVar.ValueFromSecret, "Env variable secret name on server should be empty for variable name '%s' - %s",
				serverEnvVar.Name, description)
		}
	}
}

func checkEnvironmentFromAPI(t *testing.T, env map[string]client.SecretStringDefinition, envFromAPI []client.EnvVar, description string) {
	assert.Equal(t, len(env), len(envFromAPI))
	// Iterate through server env since we already have a map in the client data to look things up
	for _, apiEnvVar := range envFromAPI {
		clientEnvVar, found := env[apiEnvVar.Name]
		require.True(t, found, "Environment variable name '%s' found on server but not on client - %s",
			apiEnvVar.Name, description)

		if clientEnvVar.Value != nil {
			assert.Equal(t, *clientEnvVar.Value, apiEnvVar.Value, "Env variable explicit value on server doesn't match for variable name '%s' - %s",
				apiEnvVar.Name, description)
		} else {
			assert.Empty(t, apiEnvVar.Value, "Env variable explicit value on server should be empty for variable name '%s' - %s",
				apiEnvVar.Name, description)
		}

		if clientEnvVar.FromSecret != nil {
			assert.Equal(t, *clientEnvVar.FromSecret, apiEnvVar.ValueFromSecret, "Env variable secret name on server doesn't match, for variable name '%s' - %s",
				apiEnvVar.Name, description)
		} else {
			assert.Empty(t, apiEnvVar.ValueFromSecret, "Env variable secret name on server should be empty for variable name '%s' - %s",
				apiEnvVar.Name, description)
		}
	}
}

func checkEnvValue(t *testing.T, envValue *client.SecretStringDefinition, valueFromServer models.SecretString) {
	if envValue == nil {
		assert.Empty(t, valueFromServer.Value)
		assert.Empty(t, valueFromServer.ValueFromSecret)
		return
	}

	if envValue.Value != nil {
		assert.Equal(t, *envValue.Value, valueFromServer.Value)
	} else {
		assert.Empty(t, valueFromServer.Value)
	}

	if envValue.FromSecret != nil {
		assert.Equal(t, *envValue.FromSecret, valueFromServer.ValueFromSecret)
	} else {
		assert.Empty(t, valueFromServer.ValueFromSecret)
	}
}

func checkEnvValueFromAPI(t *testing.T, envValue *client.SecretStringDefinition, valueFromServer *client.SecretString) {
	if envValue != nil {
		assert.Equal(t, envValue.Value, valueFromServer.Value)
		assert.Equal(t, envValue.FromSecret, valueFromServer.ValueFromSecret)
	} else {
		assert.Nil(t, valueFromServer)
	}
}

// makeJobMap populates and returns a map of job FQN to job.
func makeJobMap(jobs []*dto.JobGraph) map[models.NodeFQN]*dto.JobGraph {
	jobMap := make(map[models.NodeFQN]*dto.JobGraph)
	for _, job := range jobs {
		jobMap[job.GetFQN()] = job
	}
	return jobMap
}

// makeDependencyMap populates and returns a map of dependency target job FQN to JobDependency,
// from a list of dependencies from the server (models packages).
func makeDependencyMap(dependencies []*models.JobDependency) map[models.NodeFQN]*models.JobDependency {
	dependencyMap := make(map[models.NodeFQN]*models.JobDependency)
	for _, dependency := range dependencies {
		dependencyMap[dependency.GetFQN()] = dependency
	}
	return dependencyMap
}

// makeDependencyMapFromAPI populates and returns a map of dependency target job FQN to JobDependency,
// from a list of Dynamic SDK client dependencies.
func makeDependencyMapFromAPI(dependencies []client.JobDependency) map[models.NodeFQN]*client.JobDependency {
	dependencyMap := make(map[models.NodeFQN]*client.JobDependency)
	for _, dependency := range dependencies {
		fqn := models.NewNodeFQNForJob(models.ResourceName(dependency.Workflow), models.ResourceName(dependency.JobName))
		dependencyMap[fqn] = &dependency
	}
	return dependencyMap
}

// makeStepMap populates and returns a map of step name to step
func makeStepMap(steps []*models.Step) map[string]*models.Step {
	stepMap := make(map[string]*models.Step)
	for _, step := range steps {
		stepMap[step.Name.String()] = step
	}
	return stepMap
}

// makeStepMap populates and returns a map of step name to step
func makeStepMapFromAPI(steps []client.Step) map[string]*client.Step {
	stepMap := make(map[string]*client.Step)
	for _, step := range steps {
		stepMap[step.Name] = &step
	}
	return stepMap
}

// makeStepDependencyMap populates and returns a map of step name to step
func makeStepDependencyMap(dependencies []*models.StepDependency) map[string]*models.StepDependency {
	dependencyMap := make(map[string]*models.StepDependency)
	for _, dependency := range dependencies {
		dependencyMap[dependency.StepName.String()] = dependency
	}
	return dependencyMap
}

// makeStepDependencyMapFromAPI populates and returns a map of step name to step
func makeStepDependencyMapFromAPI(dependencies []client.StepDependency) map[string]*client.StepDependency {
	dependencyMap := make(map[string]*client.StepDependency)
	for _, dependency := range dependencies {
		dependencyMap[dependency.StepName] = &dependency
	}
	return dependencyMap
}

// makeArtifactMap populates and returns a map of artifact group name to artifact definition
func makeArtifactMap(artifacts []*models.ArtifactDefinition) map[string]*models.ArtifactDefinition {
	artifactMap := make(map[string]*models.ArtifactDefinition)
	for _, artifact := range artifacts {
		artifactMap[artifact.GroupName.String()] = artifact
	}
	return artifactMap
}

// makeArtifactMapFromAPI populates and returns a map of artifact group name to artifact definition
func makeArtifactMapFromAPI(artifacts []client.ArtifactDefinition) map[string]*client.ArtifactDefinition {
	artifactMap := make(map[string]*client.ArtifactDefinition)
	for _, artifact := range artifacts {
		artifactMap[artifact.Name] = &artifact
	}
	return artifactMap
}

// makeServiceMap populates and returns a map of service name to service
func makeServiceMap(services []*models.Service) map[string]*models.Service {
	serviceMap := make(map[string]*models.Service)
	for _, service := range services {
		serviceMap[service.Name] = service
	}
	return serviceMap
}

// makeServiceMap populates and returns a map of service name to service
func makeServiceMapFromAPI(services []client.Service) map[string]*client.Service {
	serviceMap := make(map[string]*client.Service)
	for _, service := range services {
		serviceMap[service.Name] = &service
	}
	return serviceMap
}

func jobReferenceToFQN(jobReference bb.JobReference) models.NodeFQN {
	return models.NewNodeFQNForJob(models.ResourceName(jobReference.Workflow), models.ResourceName(jobReference.JobName))
}

// findJobByName takes a list of JobGraph objects returned from the dynamic API, and looks up a particular
// job by name. Returns the job, or fails the test if the job name is not found.
func findJobByName(t *testing.T, jGraphs []client.JobGraph, jobName bb.ResourceName) *client.JobGraph {
	for i := range jGraphs {
		if jGraphs[i].Job.Name == jobName.String() {
			return &jGraphs[i]
		}
	}
	require.Fail(t, "Job name not found", "Could not find job name '%s' in JobGraph list", jobName)
	return nil
}

// findJobIDByName takes a list of JobGraph objects returned from the dynamic API, and looks up a particular
// job by name. Returns the ID of the job as a models.JobID, or fails the test if the job name is not found.
func findJobIDByName(t *testing.T, jGraphs []client.JobGraph, jobName bb.ResourceName) models.JobID {
	jGraph := findJobByName(t, jGraphs, jobName)
	id, err := models.ParseJobID(jGraph.Job.Id)
	require.NoError(t, err, "Error parsing returned Job ID for job '%s'", jobName)
	return id
}

// JobCompletionRequest is a request to complete a job by change the status to the specified final status.
type JobCompletionRequest struct {
	jobID       models.JobID
	finalStatus models.WorkflowStatus
	artifacts   []AddArtifactRequest // optional requests to add one or more artifacts before completing the job
}

type AddArtifactRequest struct {
	// groupName is the artifact group name to add an artifact to
	groupName models.ResourceName
	// path is the file system path to the artifact
	path string
	// content is the data for the artifact
	content []byte
}

func NewJobCompletionRequest(jobID models.JobID, finalStatus models.WorkflowStatus) *JobCompletionRequest {
	return &JobCompletionRequest{
		jobID:       jobID,
		finalStatus: finalStatus,
	}
}

// NewJobCompletionRequestWithArtifact creates a JobCompletionRequest to successfully complete the specified job
// and to add an artifact with the specified name and content.
func NewJobCompletionRequestWithArtifact(
	jobID models.JobID,
	artifactGroupName string,
	artifactPath string,
	artifactContent []byte,
) *JobCompletionRequest {
	return &JobCompletionRequest{
		jobID:       jobID,
		finalStatus: models.WorkflowStatusSucceeded,
		artifacts: []AddArtifactRequest{
			{
				groupName: models.ResourceName(artifactGroupName),
				path:      artifactPath,
				content:   artifactContent,
			},
		},
	}
}

// NewJobCompletionRequestWithArtifacts creates a JobCompletionRequest to successfully complete the specified job
// and to add multiple artifacts with variations on the specified name and content.
func NewJobCompletionRequestWithArtifacts(
	jobID models.JobID,
	nrArtifacts int,
	artifactGroupName string,
	artifactBasePath string,
	artifactBaseContent []byte,
) *JobCompletionRequest {
	var artifacts []AddArtifactRequest
	for i := 1; i <= nrArtifacts; i++ {
		artifacts = append(artifacts, AddArtifactRequest{
			groupName: models.ResourceName(artifactGroupName),
			path:      artifactBasePath + "/" + strconv.Itoa(i),
			content:   append(artifactBaseContent, []byte(" "+strconv.Itoa(i))...),
		})
	}
	return &JobCompletionRequest{
		jobID:       jobID,
		finalStatus: models.WorkflowStatusSucceeded,
		artifacts:   artifacts,
	}
}

// startCompletionRequestProcessing starts a new Goroutine that will Process requests to complete jobs
// from t.JobsToCompleteChan, until the channel is closed (at which time the Goroutine will exit).
func startCompletionRequestProcessing(t *JobinatorTestEnv, buildRunner *models.Runner) {
	go func() {
		for {
			completionRequest, ok := <-t.JobsToCompleteChan
			if !ok {
				return
			}
			transitionJobThroughLifecycle(t.T, t.App, buildRunner, &completionRequest)
		}
	}()
}

// transitionJobThroughLifecycle takes the specified job through various realistic status values as if it
// is being run, ending in either 'succeeded' or 'failed' depending on the value of shouldSucceed.
// No delays are introduced, so the job cycles through its status in a very short space of time.
// Optionally an artifact an be submitted before the final status is set.
func transitionJobThroughLifecycle(
	t *testing.T,
	app *server_test.TestServer,
	buildRunner *models.Runner,
	completionRequest *JobCompletionRequest,
) {
	ctx := context.Background()

	// Job must be allocated to a runner in order to change its status to 'submitted'
	// This normally happens as the job is dequeued, but we want to manually 'dequeue' it here
	jobToComplete, err := app.JobService.Read(ctx, nil, completionRequest.jobID)
	require.NoError(t, err)
	jobToComplete.RunnerID = buildRunner.ID
	err = app.JobService.Update(ctx, nil, jobToComplete)
	require.NoError(t, err)

	t.Logf("Updating status of job '%s' to '%s'", completionRequest.jobID, models.WorkflowStatusSubmitted)
	_, err = app.QueueService.UpdateJobStatus(ctx, nil, completionRequest.jobID, dto.UpdateJobStatus{
		Status: models.WorkflowStatusSubmitted,
		Error:  nil,
	})
	require.NoError(t, err)

	t.Logf("Updating status of job '%s' to '%s'", completionRequest.jobID, models.WorkflowStatusRunning)
	_, err = app.QueueService.UpdateJobStatus(ctx, nil, completionRequest.jobID, dto.UpdateJobStatus{
		Status: models.WorkflowStatusRunning,
		Error:  nil,
	})
	require.NoError(t, err)

	// Submit any artifacts before completing the job
	for i, artifact := range completionRequest.artifacts {
		artifactID := createArtifact(t, app, completionRequest.jobID, artifact)
		t.Logf("Added new artifact %d from '%s', got artifact ID '%s'", i, completionRequest.jobID, artifactID)
	}

	t.Logf("Updating status of job '%s' to '%s'", completionRequest.jobID, completionRequest.finalStatus)
	_, err = app.QueueService.UpdateJobStatus(ctx, nil, completionRequest.jobID, dto.UpdateJobStatus{
		Status: completionRequest.finalStatus,
		Error:  nil,
	})
	require.NoError(t, err)
}

func createArtifact(
	t *testing.T,
	app *server_test.TestServer,
	jobID models.JobID,
	artifactRequest AddArtifactRequest,
) models.ArtifactID {
	ctx := context.Background()
	require.NotNil(t, artifactRequest)

	t.Logf("Creating artifact for '%s', group name '%s', path '%s'", jobID, artifactRequest.groupName, artifactRequest.path)
	artifactObj, err := app.ArtifactService.Create(
		ctx,
		jobID,
		artifactRequest.groupName,
		artifactRequest.path,
		"", // don't require any particular MD5 for the content
		bytes.NewReader(artifactRequest.content),
		true, // create a blob for the data
	)
	require.NoError(t, err, "error creating artifact for job %s", jobID)

	return artifactObj.ID
}
