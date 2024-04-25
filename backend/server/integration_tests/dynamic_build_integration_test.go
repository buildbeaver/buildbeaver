package integration_tests

import (
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

func TestDynamicBuildJob(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	checkJobCount(jobinatorEnv, 1)

	build := runDynamicBuildJob(jobinatorEnv)
	defer build.Shutdown() // stop event manager, or it may try to continue during other tests

	// Dynamic build job is now done; we should be able to updated it to 'succeeded'
	_, err = app.QueueService.UpdateJobStatus(ctx, nil, job.Job.ID, dto.UpdateJobStatus{
		Status: models.WorkflowStatusSucceeded,
		Error:  nil,
	})
	require.NoError(t, err)
}

func runDynamicBuildJob(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Test Job 1")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// Make a basic job
				baseJob := bb.NewJob().
					Name("base").
					Desc("Base job description").
					RunsOn("linux").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
					StepExecution(bb.StepExecutionSequential).
					Step(bb.NewStep().
						Name("go-builder").
						Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/"))
				workflow.Job(baseJob)

				checkJobCount(t, 1)

				returnedJobs, err := workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				checkJobCount(t, 2)
				require.Equal(t, 1, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")

				baseJobGraph := findJobByName(t.T, returnedJobs, baseJob.GetName())
				checkJobFromAPI(t.T, baseJob, baseJobGraph)
				require.Nil(t, baseJobGraph.Job.Error)

				// Make a second job that depends on the first one, with more complex data to test all features
				generateJob := bb.NewJob().
					Name("generate").
					Desc("Generates all code (wire files, protobufs etc.").
					Type(bb.JobTypeDocker).
					StepExecution(bb.StepExecutionParallel). // required in order to specify step dependencies
					DependsOnJobs(baseJob).
					Docker(bb.NewDocker().
						Image("buildbeaver/go-builder:latest").
						Pull(bb.DockerPullNever).
						Shell("/bin/bash").
						BasicAuth(bb.NewBasicAuth().
							Username("username1").
							PasswordFromSecret("password_secret"))).
					Service(bb.NewService().
						Name("postgres").
						Image("postgres:14").
						Env(bb.NewEnv().
							Name("POSTGRES_USER").
							Value("buildbeaver")).
						Env(bb.NewEnv().
							Name("POSTGRES_PASSWORD").
							ValueFromSecret("postgres_password_secret")).
						BasicAuth(bb.NewBasicAuth().
							UsernameFromSecret("username_secret").
							PasswordFromSecret("password_secret"))).
					Service(bb.NewService().
						Name("notifier").
						Image("notifier:2")). // test not having env and not having auth
					Step(bb.NewStep().
						Name("run-generate").
						Desc("This step does the actual generation of code").
						Commands(`. build/scripts/lib/go-env.sh`,
							`for wire_file in backend/*/app/wire.go backend/*/app/*/wire.go; do
                       pushd "$(dirname "${wire_file}")"
                       wire
                       popd
                     done`,
							"echo 'All done...'")).
					Step(bb.NewStep().
						Name("check-generated").
						Depends("run-generate").
						Commands(`echo "Checking generated files..."`)).
					Artifact(bb.NewArtifact().Name("wire").Paths("backend/*/app/wire_gen.go", "backend/*/app/*/wire_gen.go")).
					Artifact(bb.NewArtifact().Name("grpc").Paths("backend/api/grpc/*.pb.go"))
				workflow.Job(generateJob)

				checkJobCount(t, 2)
				_, err = workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")

				checkJobCount(t, 3)

				// Check the jobs went into the database correctly
				ctx := context.Background()
				buildFromServer, err := t.App.QueueService.ReadQueuedBuild(ctx, nil, t.BuildID)
				require.NoError(t, err)
				require.Equal(t, 3, len(buildFromServer.Jobs))
				jobMap := makeJobMap(buildFromServer.Jobs)
				baseJobFromServer := jobMap[jobReferenceToFQN(baseJob.GetReference())]
				require.NotNil(t, baseJobFromServer, "Could not find 'base' job on the server")
				generateJobFromServer := jobMap[jobReferenceToFQN(generateJob.GetReference())]
				require.NotNil(t, generateJobFromServer, "Could not find 'base' job on the server")

				checkJob(t.T, generateJob, generateJobFromServer)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func TestDynamicBuildSegregation(t *testing.T) {
	// This is a short test, no need to skip
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph1 := enqueueDynamicBuild(t, app, commit, nil)
	job1, env1 := dequeueJob(t, app, buildRunner.ID, buildGraph1.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job1, env1)

	// Enqueue a second build and dequeue a second dynamic build job; each job should not have access to the other
	// Don't enqueue job 2 until after we dequeued job 1, to ensure we get the correct job when we dequeue
	buildGraph2 := enqueueDynamicBuild(t, app, commit, nil)
	job2, _ := dequeueJob(t, app, buildRunner.ID, buildGraph2.Jobs[0].Name)

	require.NotEqual(t, job1.JWT, job2.JWT)
	require.NotEqual(t, job1.Job.BuildID, job2.Job.BuildID)

	// Create the dynamic build client for job 1
	build, err := bb.WorkflowsWithEnv(env1, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				checkJobCountForBuild(jobinatorEnv, job1.Job.BuildID, 1)
				checkJobCountForBuild(jobinatorEnv, job2.Job.BuildID, 1)

				// Make a basic new job
				baseJob := bb.NewJob().
					Name("base").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("go-builder").
						Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/"))
				workflow.Job(baseJob)

				// Try changing the build ID to the 'wrong' build (build 2); we should not be allowed to submit jobs
				// because the JWT is only valid for build 1
				build := workflow.GetBuild()
				build.ID, err = bb.ParseBuildID(buildGraph2.ID.String())
				require.NoError(t, err)
				_, err = workflow.Submit(false)
				require.Error(t, err, "Expected error submitting job to the wrong build")
				t.Logf("Returned error from wrong build: %v", err)

				checkJobCountForBuild(jobinatorEnv, job1.Job.BuildID, 1)
				checkJobCountForBuild(jobinatorEnv, job2.Job.BuildID, 1)

				// Now try setting the build ID back to the correct build (build 1); we should be granted access to submit jobs
				build.ID, err = bb.ParseBuildID(buildGraph1.ID.String())
				require.NoError(t, err)
				_, err = workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")

				checkJobCountForBuild(jobinatorEnv, job1.Job.BuildID, 2)
				checkJobCountForBuild(jobinatorEnv, job2.Job.BuildID, 1)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	defer build.Shutdown()
}

func TestDynamicBuildSubmitJobToFinishedBuild(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	checkJobCount(jobinatorEnv, 1)

	// Before we run the dynamic build job, set the main build status to failed
	buildModel, err := app.BuildService.Read(ctx, nil, buildGraph.ID)
	require.NoError(t, err)
	now := models.NewTime(time.Now())
	buildModel.Timings.FinishedAt = &now
	buildModel.Status = models.WorkflowStatusFailed
	err = app.BuildService.Update(ctx, nil, buildModel)
	require.NoError(t, err)

	// Build is finished, so submitting another job from a dynamic build should fail
	bbBuild := runDynamicBuildJobSubmitBasicJob(jobinatorEnv, true)
	bbBuild.Shutdown()
}

func runDynamicBuildJobSubmitBasicJob(t *JobinatorTestEnv, expectError bool) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Basic Job")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").SubmitFailureIsFatal(false).Handler(
			func(workflow *bb.Workflow) error {

				// Make a basic job
				workflow.Job(bb.NewJob().
					Name("base").
					Desc("Base job description").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					StepExecution(bb.StepExecutionSequential).
					Step(bb.NewStep().
						Name("test").
						Commands("echo Hello World")))

				returnedJobs, err := workflow.Submit(false)
				if expectError {
					require.Error(t, err, "Error expected when submitting new job(s) to build")
				} else {
					require.NoError(t, err, "Error submitting new job(s) to build")
					require.Len(t, returnedJobs, 1, "Expected 1 job back")
				}

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}
