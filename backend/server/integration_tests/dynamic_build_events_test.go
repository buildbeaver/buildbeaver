package integration_tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestDynamicBuildWaiting(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	startCompletionRequestProcessing(jobinatorEnv, buildRunner)

	build := runDynamicBuildWithWaits(jobinatorEnv)
	build.Shutdown()
}

func runDynamicBuildWithWaits(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelDebug)
	bb.Log(bb.LogLevelInfo, "Dynamic Build With Waits - Test Job")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// Make a couple of basic jobs
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

				job2 := bb.NewJob().
					Name("job2").
					Desc("Job 2 description").
					RunsOn("macos", "arm64").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("job2-step").
						Commands("echo 'Hello World!'"))
				workflow.Job(job2)

				// Submit the jobs
				checkJobCount(t, 1)
				returnedJobs, err := workflow.Submit()
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				checkJobCount(t, 3)
				require.Equal(t, 2, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")

				baseJobGraph := findJobByName(t.T, returnedJobs, baseJob.GetName())
				checkJobFromAPI(t.T, baseJob, baseJobGraph)

				job2Graph := findJobByName(t.T, returnedJobs, job2.GetName())
				job2ID := findJobIDByName(t.T, returnedJobs, job2.GetName())
				checkJobFromAPI(t.T, job2, job2Graph)

				// Notify our parent test that we are ready for a job to be completed
				t.JobsToCompleteChan <- *NewJobCompletionRequest(job2ID, models.WorkflowStatusSucceeded)
				close(t.JobsToCompleteChan) // No more jobs to set to complete

				// Wait for the job to complete (via events)
				_, err = workflow.WaitForJob(job2.GetReference().String())
				require.NoError(t, err, "error waiting for Job '%s' to finish", job2.GetReference())
				t.Logf("Received notification that job '%s' has completed", job2.GetName())

				// Submit a third job after job 2 completes
				job3 := bb.NewJob().
					Name("job3").
					Desc("Job 3 description").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("job3-step").
						Commands("echo 'Goodbye world!'"))
				workflow.Job(job3)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func TestDynamicBuildCallbacks(t *testing.T) {
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

	startCompletionRequestProcessing(jobinatorEnv, buildRunner)

	build := runDynamicBuildWithCallbacks(jobinatorEnv)
	build.Shutdown()
}

func runDynamicBuildWithCallbacks(t *JobinatorTestEnv) *bb.Build {
	var (
		statusChangedCount int
		job4Failed         bool
		job4Succeeded      bool
		job5Failed         bool
		job5Succeeded      bool
	)

	bb.SetDefaultLogLevel(bb.LogLevelDebug)
	bb.Log(bb.LogLevelInfo, "Dynamic Build With Waits - Test Job")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// Make a couple of basic jobs
				baseJob := bb.NewJob().
					Name("base").
					Desc("Base job description").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
					StepExecution(bb.StepExecutionSequential).
					Step(bb.NewStep().
						Name("go-builder").
						Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/"))
				workflow.Job(baseJob)

				job2 := bb.NewJob().
					Name("job2").
					Desc("Job 2 description").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("job2-step").
						Commands("echo 'Hello World!'")).
					OnCompletion(func(event *bb.JobStatusChangedEvent) {
						t.Logf("Got callback that job2 is done")
						require.Equal(t, workflow.GetBuild().ID, event.BuildID)
						require.Equal(t, bb.ResourceName("job2"), event.JobName)
						require.Equal(t, bb.StatusSucceeded, event.JobStatus)

						// Submit a third job after job 2 completes
						job3 := bb.NewJob().
							Name("job3").
							Desc("Job 3 description").
							Docker(bb.NewDocker().
								Image("docker:20.10").
								Pull(bb.DockerPullIfNotExists)).
							Step(bb.NewStep().
								Name("job3-step").
								Commands("echo 'Goodbye world!'"))
						workflow.Job(job3)
						// Call Submit() explicitly so that we don't terminate the test process on failure after
						// the implicit call to Submit() that happens after the callback returns
						returnedJobs, err := workflow.Submit(false)
						require.NoError(t, err)

						// Complete job 3
						job3ID := findJobIDByName(t.T, returnedJobs, job3.GetName())
						t.JobsToCompleteChan <- *NewJobCompletionRequest(job3ID, models.WorkflowStatusSucceeded)
					}).
					OnStatusChanged(func(event *bb.JobStatusChangedEvent) {
						bb.Log(bb.LogLevelInfo, fmt.Sprintf("Got event of type %s for job %s, status %s", event.RawEvent.Type, event.JobName, event.JobStatus))
						statusChangedCount++
					})
				workflow.Job(job2)

				// job3 is not submitted yet, not until job2 is done, but we can still wait on it by name
				workflow.OnJobSuccess(bb.NewJobReference("test", "job3"), func(event *bb.JobStatusChangedEvent) {
					// Submit another couple of jobs, with callbacks waiting for failure and success
					// TODO: Test OnCancelled once we can cancel jobs
					workflow.Job(bb.NewJob().
						Name("job4").
						Desc("Job 4 description").
						Docker(bb.NewDocker().
							Image("docker:20.10").
							Pull(bb.DockerPullIfNotExists)).
						Step(bb.NewStep().
							Name("job4-step").
							Commands("echo 'Job 4 says hi!'")).
						OnSuccess(func(event *bb.JobStatusChangedEvent) {
							bb.Log(bb.LogLevelInfo, fmt.Sprintf("Got OnSuccess callback for job %s, status %s", event.JobName, event.JobStatus))
							job4Succeeded = true
						}).
						OnFailure(func(event *bb.JobStatusChangedEvent) {
							bb.Log(bb.LogLevelInfo, fmt.Sprintf("Got OnFailure callback for job %s, status %s", event.JobName, event.JobStatus))
							job4Failed = true
						}))

					workflow.Job(bb.NewJob().
						Name("job5").
						Desc("Job 5 description").
						Docker(bb.NewDocker().
							Image("docker:20.10").
							Pull(bb.DockerPullIfNotExists)).
						Step(bb.NewStep().
							Name("job5-step").
							Commands("echo 'Job 5 says bye!'")).
						// Try adding the callback subscriptions the other way around
						OnFailure(func(event *bb.JobStatusChangedEvent) {
							bb.Log(bb.LogLevelInfo, fmt.Sprintf("Got OnFailure callback for job %s, status %s", event.JobName, event.JobStatus))
							job5Failed = true
						}).
						OnSuccess(func(event *bb.JobStatusChangedEvent) {
							bb.Log(bb.LogLevelInfo, fmt.Sprintf("Got OnSuccess callback for job %s, status %s", event.JobName, event.JobStatus))
							job5Succeeded = true
						}))

					returnedJobs, err := workflow.Submit(false)
					require.NoError(t, err)

					// Tell the main Goroutine to complete jobs 4 (failing) and 5 (succeeding)
					job4ID := findJobIDByName(t.T, returnedJobs, "job4")
					job5ID := findJobIDByName(t.T, returnedJobs, "job5")
					t.JobsToCompleteChan <- *NewJobCompletionRequest(job4ID, models.WorkflowStatusFailed)
					t.JobsToCompleteChan <- *NewJobCompletionRequest(job5ID, models.WorkflowStatusSucceeded)

					// No more jobs to set to complete
					close(t.JobsToCompleteChan)
				})

				// Submit the first 2 jobs
				checkJobCount(t, 1)
				returnedJobs, err := workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				checkJobCount(t, 3)
				require.Equal(t, 2, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")

				baseJobFromServer := findJobByName(t.T, returnedJobs, baseJob.GetName())
				checkJobFromAPI(t.T, baseJob, baseJobFromServer)

				job2FromServer := findJobByName(t.T, returnedJobs, job2.GetName())
				job2ID := findJobIDByName(t.T, returnedJobs, job2.GetName())
				checkJobFromAPI(t.T, job2, job2FromServer)

				// Notify our parent test that we are ready for a job to be completed
				t.Logf("Notifying main thread that is should now complete job %s", job2ID)
				t.JobsToCompleteChan <- *NewJobCompletionRequest(job2ID, models.WorkflowStatusSucceeded)

				// Allow the callback a chance to run
				t.Logf("Calling Submit())")
				_, err = workflow.Submit()
				require.NoError(t, err)
				t.Logf("Submit() returned")

				// All callbacks should now have been run
				expectedStatusChangedCount := 4
				require.Equal(t, expectedStatusChangedCount, statusChangedCount, "Unexpected number of job status changed events")
				require.False(t, job4Succeeded, "job4 succeeded callback should NOT have been called")
				require.True(t, job4Failed, "job4 failed callback should have been called")
				require.True(t, job5Succeeded, "job5 succeeded callback should have been called")
				require.False(t, job5Failed, "job5 failed callback should NOT have been called")

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func TestDynamicBuildFinishDetection(t *testing.T) {
	// TODO: Put this test back once we have finish detection again
	t.Skipf("Skipping TestDynamicBuildFinishDetection since finish detection was disabled when adding workflows")

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

	startCompletionRequestProcessing(jobinatorEnv, buildRunner)

	build := runDynamicBuildWithFinishDetection(jobinatorEnv)
	build.Shutdown()
}

func runDynamicBuildWithFinishDetection(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelDebug)
	bb.Log(bb.LogLevelInfo, "Dynamic Build With Waits - Test Job")

	var baseCallbackCalled bool

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// Make a basic job
				workflow.Job(bb.NewJob().
					Name("base").
					Desc("Base job description").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
					StepExecution(bb.StepExecutionSequential).
					Step(bb.NewStep().
						Name("go-builder").
						Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/")).
					OnCompletion(func(event *bb.JobStatusChangedEvent) {
						baseCallbackCalled = true
					}))

				// Submit the job
				returnedJobs, err := workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				require.Equal(t, 1, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")
				baseJobID := findJobIDByName(t.T, returnedJobs, "base")

				// Subscribe to a non-existent job, to check that we don't wait forever in Submit() below
				workflow.OnJobCompletion(bb.NewJobReference("", "non-existent-job"), func(event *bb.JobStatusChangedEvent) {
					t.Error("Received a callback for non-existent-job")
				})

				// Tell the main Goroutine to complete the base job. This means the build is done apart from this dynamic job.
				t.JobsToCompleteChan <- *NewJobCompletionRequest(baseJobID, models.WorkflowStatusSucceeded)
				close(t.JobsToCompleteChan) // No more jobs to set to complete

				// Submit jobs and process the callbacks. This should return once the build is 'done' even though the
				// callback for "non-existent-job" has not been run.
				_, err = workflow.Submit()
				require.NoError(t, err)
				require.True(t, baseCallbackCalled, "Base job callback should have been called")

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}
