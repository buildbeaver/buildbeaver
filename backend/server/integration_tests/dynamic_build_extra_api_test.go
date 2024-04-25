package integration_tests

import (
	"encoding/json"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestDynamicBuildMiscAPI(t *testing.T) {
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

	build := runDynamicBuildMiscAPI(jobinatorEnv)
	build.Shutdown()
}

func runDynamicBuildMiscAPI(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "Dynamic Build With Read API calls - Test Job")

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
						build := workflow.GetBuild()
						require.Equal(t, build.ID, event.BuildID)

						// Read the build graph; should include 3 jobs
						bGraph, err := build.GetBuildGraph()
						require.NoError(t, err)
						bGraphJSON, err := json.Marshal(bGraph)
						t.Logf("Build graph: %s", bGraphJSON)
						require.Equal(t, 3, len(bGraph.Jobs), "Wrong number of jobs in returned build graph")
						found := false
						for _, jGraph := range bGraph.Jobs {
							if jGraph.Job.Name == "job2" {
								found = true
							}
						}
						require.True(t, found, "Unable to find job2 in the returned build graph")

						// Read the job graph for the job directly
						jGraph, err := build.GetJobGraph(event.JobID)
						require.NoError(t, err)
						require.Equal(t, 1, len(jGraph.Steps), "Wrong number of steps returned in job graph")
						t.Logf("Job Graph returned with job status %s, log descriptor ID '%s'", jGraph.Job.Status, jGraph.Job.LogDescriptorId)

						// Locate the log descriptor ID for the step within job graph
						var stepLogID string
						for _, step := range jGraph.Steps {
							if step.Name == "job2-step" {
								stepLogID = step.LogDescriptorId
							}
						}
						require.NotEmptyf(t, stepLogID, "error finding the log descriptor ID for step within job")

						// Read log for the step as text, using its log descriptor ID
						logTextReader, err := build.ReadLogText(stepLogID, true)
						require.NoError(t, err)
						defer logTextReader.Close()
						logText, err := io.ReadAll(logTextReader)
						require.NoError(t, err)
						t.Logf("Got log text for step from server (%d bytes)", len(logText))

						// Read log data for the step, using its log descriptor ID
						logDataReader, err := build.ReadLogData(stepLogID, false)
						require.NoError(t, err)
						defer logDataReader.Close()
						logData, err := io.ReadAll(logDataReader)
						require.NoError(t, err)
						t.Logf("Got log data for step from server (%d bytes)", len(logData))

						// Read the log descriptor for step, containing extra information about the log
						stepLogDescriptor, err := build.GetLogDescriptor(stepLogID)
						require.NoError(t, err)
						t.Logf("Got log descriptor from server for step: log descriptor ID %s, log is for resource %s, log size %d bytes",
							stepLogDescriptor.Id, stepLogDescriptor.ResourceId, stepLogDescriptor.SizeBytes)
					})
				workflow.Job(job2)

				// Submit the first 2 jobs
				checkJobCount(t, 1)
				returnedJobs, err := workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				checkJobCount(t, 3)
				require.Equal(t, 2, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")

				job2ID := findJobIDByName(t.T, returnedJobs, job2.GetName())

				// Complete job 2 right away
				t.JobsToCompleteChan <- *NewJobCompletionRequest(job2ID, models.WorkflowStatusSucceeded)
				// No more jobs to set to complete
				close(t.JobsToCompleteChan)

				// Allow the callback a chance to run
				_, err = workflow.Submit()
				require.NoError(t, err)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func TestDynamicBuildArtifacts(t *testing.T) {
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

	build := runDynamicBuildArtifacts(jobinatorEnv)
	build.Shutdown()
}

func runDynamicBuildArtifacts(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "Dynamic Build with Artifact reading - Test Job")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// Make a job that simulates running tests and produces a 'test report' artifact
				workflow.Job(bb.NewJob().
					Name("run-tests").
					Desc("Run tests and produce performance report").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("produce-report").
						Commands("mkdir reports",
							"echo >>reports/test-report '----- PERFORMANCE TESTING REPORT -----'",
							"echo >>reports/test-report 'small-test-time: 12'",
							"echo >>reports/test-report 'large-test-time: 150'",
							"echo >>reports/test-report 'END OF REPORT'",
						)).
					Artifact(bb.NewArtifact().
						Name("report-artifacts").
						Paths("reports/test-report")).
					OnCompletion(func(event *bb.JobStatusChangedEvent) {
						// Read the report artifact
						page, err := workflow.GetBuild().ListArtifacts("", "run-tests", "report-artifacts")
						require.NoError(t, err, "Error returned from ListArtifacts API call")
						if err != nil {
							// TODO: Add some kind of Fail() method on the build to cause the build to fail?
							return
						}
						require.Equal(t, 1, len(page.Artifacts), "Wrong number of artifacts returned from ListArtifacts() call")
						require.Equal(t, "reports/test-report", page.Artifacts[0].Path, "Wrong path in returned artifact")

						// Read the artifact data for the report
						artifact := page.Artifacts[0]
						data, err := workflow.GetBuild().GetArtifactData(artifact.Id)
						require.NoError(t, err)
						t.Logf("Artifact data for path '%s:\n----------\n%s\n----------\n", artifact.Path, data)
					}))

				// Make a job that  produces 5 simple artifacts to test paging from the SDK
				workflow.Job(bb.NewJob().
					Name("make-artifacts").
					Desc("Run tests and produce performance report").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Step(bb.NewStep().
						Name("produce-files").
						Commands("mkdir test-art",
							"echo >>test-art/art1 'Artifact 1",
							"echo >>test-art/art2 'Artifact 2",
							"echo >>test-art/art3 'Artifact 3",
							"echo >>test-art/art4 'Artifact 4",
							"echo >>test-art/art5 'Artifact 5",
						)).
					Artifact(bb.NewArtifact().
						Name("test-art").
						Paths("test-art/*")).
					OnCompletion(func(event *bb.JobStatusChangedEvent) {
						checkArtifactPaging(t.T, workflow, "make-artifacts", "test-art", 5, 3)
					}))

				returnedJobs, err := workflow.Submit(false)
				require.NoError(t, err, "Error submitting new job(s) to build (1)")
				require.Equal(t, 2, len(returnedJobs), "Incorrect number of jobs returned from Submit() call")

				runTestsJobID := findJobIDByName(t.T, returnedJobs, "run-tests")
				makeArtifactsJobID := findJobIDByName(t.T, returnedJobs, "make-artifacts")

				// Complete the jobs with artifacts
				t.JobsToCompleteChan <- *NewJobCompletionRequestWithArtifact(
					runTestsJobID,
					"report-artifacts",
					"reports/test-report",
					[]byte("----- PERFORMANCE TESTING REPORT -----\n"+
						"small-test-time: 12\n"+
						"large-test-time: 150"+
						"echo >>test-report 'END OF REPORT'"),
				)
				t.JobsToCompleteChan <- *NewJobCompletionRequestWithArtifacts(
					makeArtifactsJobID,
					5,
					"test-art",
					"test-art/art",
					[]byte("Artifact "),
				)
				// No more jobs to set to complete
				close(t.JobsToCompleteChan)

				_, err = workflow.Submit() // Allow the callback a chance to run
				require.NoError(t, err)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func checkArtifactPaging(
	t *testing.T,
	workflow *bb.Workflow,
	jobName string,
	groupName string,
	expectedNrArtifacts int,
	pageSize int,
) {
	// Reading artifacts through the API with the requested page size
	resultsPage, err := workflow.GetBuild().ListArtifactsN("", jobName, groupName, pageSize)
	require.NoError(t, err, "Error returned from ListArtifacts API call")

	pageNr := 1
	for nrRemaining := expectedNrArtifacts; nrRemaining > 0; {
		// Check the correct number of results are in the page
		if nrRemaining >= pageSize {
			require.Equal(t, pageSize, len(resultsPage.Artifacts), "Expected page %d to be a full page of %d results", pageNr, pageSize)
		} else {
			require.Equal(t, nrRemaining, len(resultsPage.Artifacts), "Expected page %d to be a partial page of %d results", pageNr, nrRemaining)
		}
		// Check the next and previous cursors are correct
		if pageNr == 1 {
			require.False(t, resultsPage.HasPrev(), "Expected first page to have no previous page")
		} else {
			require.True(t, resultsPage.HasPrev(), "Expected page %d to have a previous page", pageNr)
		}

		for i := range resultsPage.Artifacts {
			t.Logf("Found page %d artifact %d with path '%s'", pageNr, i, resultsPage.Artifacts[i].Path)
		}
		nrRemaining -= len(resultsPage.Artifacts)

		if nrRemaining > 0 {
			require.True(t, resultsPage.HasNext(), "Expected page %d to have a next page; not all expected artifacts have been seen", pageNr)
			// Read the next page
			resultsPage, err = resultsPage.Next()
			require.NoError(t, err)
			pageNr++
		} else {
			require.False(t, resultsPage.HasNext(), "Expected page %d to be the last page and to have no next page", pageNr)
		}
	}
}
