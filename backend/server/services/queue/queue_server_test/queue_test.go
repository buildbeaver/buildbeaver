package queue_server_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
)

func TestQueue(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	ctx := context.Background()

	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")

	runner := server_test.CreateRunner(t, ctx, app, "", legalEntity.ID, nil)

	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	_ = server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)

	t.Run("Queue", testQueueBuild(app, repo.ID, legalEntity.ID, runner.ID))
	t.Run("BuildFailure", testBuildFailure(app, repo.ID, legalEntity.ID, runner.ID))
	t.Run("JobTimeout", testJobTimeout(app, repo.ID, legalEntity.ID, runner.ID))
}

func TestDequeueWithLabels(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()
	ctx := context.Background()

	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)

	labels := models.Labels{"supported"}

	runnerWithLabel := server_test.CreateRunner(t, ctx, app, "with-label", legalEntity.ID, nil)
	runnerWithLabel.Labels = labels
	_, err = app.RunnerService.Update(ctx, nil, runnerWithLabel)
	require.NoError(t, err)

	runnerWithoutLabel := server_test.CreateRunner(t, ctx, app, "without-label", legalEntity.ID, nil)

	buildDef := &models.BuildDefinition{
		Jobs: []models.JobDefinition{
			{
				JobDefinitionData: models.JobDefinitionData{
					Name:                    "one",
					Type:                    "docker",
					RunsOn:                  labels,
					DockerImage:             "golang:1.18",
					DockerImagePullStrategy: models.DockerPullStrategyDefault,
					StepExecution:           models.StepExecutionSequential,
				},
				Steps: []models.StepDefinition{{
					StepDefinitionData: models.StepDefinitionData{
						Name: "test",
						Commands: models.Commands{
							"echo 'hello world'",
						},
					},
				}},
			},
		}}

	// Should enqueue successfully
	build, err := app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.Nil(t, build.Error)
	require.Equal(t, models.WorkflowStatusQueued, build.Status)

	// The runner without the label should not be able to dequeue
	job, err := app.QueueService.Dequeue(ctx, runnerWithoutLabel.ID)
	require.Error(t, err)
	require.Nil(t, job)

	// The runner with the label should
	job, err = app.QueueService.Dequeue(ctx, runnerWithLabel.ID)
	require.NoError(t, err)
	require.Equal(t, buildDef.Jobs[0].Name, job.Name)

	buildDef.Jobs[0].RunsOn = nil

	// Queue it again, this timewithout any labels
	build, err = app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.Nil(t, build.Error)
	require.Equal(t, models.WorkflowStatusQueued, build.Status)

	// The runner without the label should be able to dequeue
	job, err = app.QueueService.Dequeue(ctx, runnerWithoutLabel.ID)
	require.NoError(t, err)
	require.Equal(t, buildDef.Jobs[0].Name, job.Name)
}

func TestNoCompatibleRunners(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()
	ctx := context.Background()

	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	runner := server_test.CreateRunner(t, ctx, app, "", legalEntity.ID, nil)
	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)

	// Add the supported label to the runner
	runner.Labels = models.Labels{"supported"}
	_, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)

	// A build with a job that doesn't depend on any labels
	buildDef := &models.BuildDefinition{
		Jobs: []models.JobDefinition{
			{
				JobDefinitionData: models.JobDefinitionData{
					Name:                    "one",
					Type:                    "docker",
					RunsOn:                  models.Labels{},
					DockerImage:             "golang:1.18",
					DockerImagePullStrategy: models.DockerPullStrategyDefault,
					StepExecution:           models.StepExecutionSequential,
				},
				Steps: []models.StepDefinition{{
					StepDefinitionData: models.StepDefinitionData{
						Name: "test",
						Commands: models.Commands{
							"echo 'hello world'",
						},
					},
				}},
			},
		}}

	// Should enqueue successfully
	build, err := app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.Nil(t, build.Error)
	require.Equal(t, models.WorkflowStatusQueued, build.Status)

	// A build with a job that specifies labels that a registered runner supports
	buildDef.Jobs[0].RunsOn = models.Labels{"supported"}

	// Should enqueue successfully
	build, err = app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.Nil(t, build.Error)
	require.Equal(t, models.WorkflowStatusQueued, build.Status)

	// A build with one job that specifies labels that no registered runner supports
	buildDef.Jobs[0].RunsOn = models.Labels{"supported", "not-supported"}

	// Should be enqueued in the failed state
	build, err = app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.NotNil(t, build.Error)
	require.Equal(t, models.WorkflowStatusFailed, build.Status)

	// A build with one job that specifies labels that no registered runner supports, and one job that does
	buildDef.Jobs = append(buildDef.Jobs, models.JobDefinition{
		JobDefinitionData: models.JobDefinitionData{
			Name:                    "two",
			Type:                    "docker",
			RunsOn:                  models.Labels{},
			DockerImage:             "golang:1.18",
			DockerImagePullStrategy: models.DockerPullStrategyDefault,
			StepExecution:           models.StepExecutionSequential,
		},
		Steps: []models.StepDefinition{{
			StepDefinitionData: models.StepDefinitionData{
				Name: "test",
				Commands: models.Commands{
					"echo 'hello world'",
				},
			},
		}},
	})

	// Should be enqueued in the queued state, but the job with unsupported labels should be in the failed state.
	build, err = app.QueueService.EnqueueBuildFromBuildDefinition(ctx, nil, repo.ID, commit.ID, buildDef, "refs/heads/master", nil)
	require.NoError(t, err)
	require.Nil(t, build.Error)
	require.Equal(t, models.WorkflowStatusRunning, build.Status)
	if build.Jobs[0].Status == models.WorkflowStatusQueued {
		require.Equal(t, build.Jobs[1].Status, models.WorkflowStatusFailed)
	} else {
		require.Equal(t, build.Jobs[1].Status, models.WorkflowStatusQueued)
	}
}

func TestQueueInvalidYAML(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	ctx := context.Background()

	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")

	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)

	// Note: We are manually creating a few test items here as we do not need entity_utils in place for invalid YAML situations.
	var randomCommit = referencedata.GenerateInvalidCommit(repo.ID, legalEntity.ID)

	err = app.CommitStore.Create(ctx, nil, randomCommit)
	require.Nil(t, err)

	build, err := app.QueueService.EnqueueBuildFromCommit(ctx, nil, randomCommit, referencedata.TestRef, nil)
	require.NoError(t, err)
	require.NotNil(t, build.Error)
	require.Equal(t, build.Status, models.WorkflowStatusFailed)
}

func testQueueBuild(app *server_test.TestServer, repoId models.RepoID, legalEntityId models.LegalEntityID, runnerId models.RunnerID) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err), "Expected a not found error to be returned, but got '%v'", err)
		require.Nil(t, job, "No job should be returned when queue is empty, but found job '%v'", job)

		buildDTO := server_test.CreateAndQueueBuild(t, ctx, app, repoId, legalEntityId, "")
		buildID := buildDTO.ID

		checkBuildStatus(t, app, buildID, models.WorkflowStatusQueued)

		t.Run("Dequeue job 1", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 2", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 3", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 4", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))

		// Only after all jobs are finished should we see the build itself succeed
		checkBuildStatus(t, app, buildID, models.WorkflowStatusSucceeded)

		job, err = app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err))
		require.Nil(t, job, "Expected all jobs to have been processed")
	}
}

func testBuildFailure(app *server_test.TestServer, repoId models.RepoID, legalEntityId models.LegalEntityID, runnerId models.RunnerID) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err))
		require.Nil(t, job, "No job should be returned when queue is empty")

		buildDTO := server_test.CreateAndQueueBuild(t, ctx, app, repoId, legalEntityId, "")
		buildID := buildDTO.ID

		checkBuildStatus(t, app, buildID, models.WorkflowStatusQueued)

		t.Run("Dequeue job 1", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 2", testDequeueJob(app, runnerId, models.WorkflowStatusFailed))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 3", testDequeueJob(app, runnerId, models.WorkflowStatusFailed))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		t.Run("Dequeue job 4", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))

		// Only after all jobs are finished should we see the build itself fail
		checkBuildStatus(t, app, buildID, models.WorkflowStatusFailed)

		job, err = app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err))
		require.Nil(t, job, "Expected all jobs to have been processed")
	}
}

// testJobTimeout tests what happens if a timeout occurs in the middle of the build.
// Some jobs will be finished. Others will be running, submitted or queued and these should all time out.
// The build should be set to failed after all non-completed jobs timed out.
func testJobTimeout(app *server_test.TestServer, repoId models.RepoID, legalEntityId models.LegalEntityID, runnerId models.RunnerID) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err))
		require.Nil(t, job, "No job should be returned when queue is empty")

		buildDTO := server_test.CreateAndQueueBuild(t, ctx, app, repoId, legalEntityId, "")
		buildID := buildDTO.ID

		checkBuildStatus(t, app, buildID, models.WorkflowStatusQueued)

		t.Run("Dequeue job 1", testDequeueJob(app, runnerId, models.WorkflowStatusSucceeded))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		// Leave the second job running so it can time out
		t.Run("Dequeue job 2", testDequeueJob(app, runnerId, models.WorkflowStatusRunning))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		// Leave the third job in 'submitted' state so that it can time out
		t.Run("Dequeue job 2", testDequeueJob(app, runnerId, models.WorkflowStatusSubmitted))
		checkBuildStatus(t, app, buildID, models.WorkflowStatusRunning)

		// Leave the fourth job in 'queued' state, and this should also time out

		// Tell the queue to test for a very short timeout; this will cause all non-completed jobs to time out
		// (running, submitted and queued jobs)
		time.Sleep(2 * time.Millisecond) // long enough to fail a 1-millisecond timeout but not to slow down the tests
		nrJobsTimedOut := checkForTimeouts(t, app, 1*time.Millisecond)
		require.Equal(t, 3, nrJobsTimedOut, "All non-completed jobs should have timed out")

		// Build should now have failed since all remaining jobs timed out
		checkBuildStatus(t, app, buildID, models.WorkflowStatusFailed)

		job, err = app.QueueService.Dequeue(ctx, runnerId)
		require.NotNil(t, gerror.ToNotFound(err))
		require.Nil(t, job, "Expected all jobs to have been processed")
	}
}

func checkBuildStatus(t *testing.T, app *server_test.TestServer, buildID models.BuildID, expectedStatus models.WorkflowStatus) {
	build, err := app.BuildService.Read(context.Background(), nil, buildID)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, build.Status)

	if expectedStatus == models.WorkflowStatusFailed {
		require.True(t, build.Error.Valid(), "Failed build with status WorkflowStatusFailed should have a valid error")
		t.Logf("Expected and found the following error: %v", build.Error.Error())
	}
}

// testDequeueJob will:
// 1. dequeue the next job
// 2. change the job status to WorkflowStatusRunning
// 3. change the status of all *steps* within the job to WorkflowStatusSucceeded
// 4. change the status of the job to jobFinalStatus
// If any of these operations fail then the test will be failed.
func testDequeueJob(app *server_test.TestServer, runnerId models.RunnerID, jobFinalStatus models.WorkflowStatus) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := app.QueueService.Dequeue(ctx, runnerId)
		require.Nil(t, err)
		require.NotNil(t, job)

		switch job.Name {
		case referencedata.ReferenceJob1.Name:
			t.Run("ValidateWithJob Reference Job 1", testJobProperties(job, referencedata.ReferenceJob1))
		case referencedata.ReferenceJob2.Name:
			t.Run("ValidateWithJob Reference Job 2", testJobProperties(job, referencedata.ReferenceJob2))
		case referencedata.ReferenceJob3.Name:
			t.Run("ValidateWithJob Reference Job 3", testJobProperties(job, referencedata.ReferenceJob3))
		case referencedata.ReferenceJob4.Name:
			t.Run("ValidateWithJob Reference Job 4", testJobProperties(job, referencedata.ReferenceJob4))
		default:
			t.Fatal("Unexpected job dequeued")
		}

		// Check that a JTW was created for the build
		require.NotEmpty(t, job.JWT, "JWT token should have been returned with dequeued job")

		// Mark job as running, so the entire build gets marked as running
		_, err = app.QueueService.UpdateJobStatus(ctx, nil, job.ID, dto.UpdateJobStatus{
			Status: models.WorkflowStatusRunning,
			Error:  nil,
			ETag:   "",
		})
		require.Nil(t, err)

		for _, step := range job.Steps {
			_, err = app.QueueService.UpdateStepStatus(ctx, nil, step.ID, dto.UpdateStepStatus{
				Status: models.WorkflowStatusSucceeded,
				Error:  nil,
				ETag:   step.ETag,
			})
			require.Nil(t, err)
		}

		// Set the desired final status for the job
		var jobErr *models.Error
		if jobFinalStatus == models.WorkflowStatusFailed {
			jobErr = models.NewError(fmt.Errorf("error introduced to test job failure"))
		}
		_, err = app.QueueService.UpdateJobStatus(ctx, nil, job.ID, dto.UpdateJobStatus{
			Status: jobFinalStatus,
			Error:  jobErr,
			ETag:   "",
		})
		require.Nil(t, err)
	}
}

func testJobProperties(job *dto.RunnableJob, referenceJob *dto.JobGraph) func(t *testing.T) {
	return func(t *testing.T) {
		if job == nil {
			t.Fatal("Job should have been returned when queue is populated")
		}
		if job.ETag == "" {
			t.Errorf("Expected job etag to be set: %s", job.ETag)
		}
		if job.Status != models.WorkflowStatusSubmitted {
			t.Errorf("Expected job to be in submitted state after dequeue: %s", job.Status)
		}
		if len(job.Steps) != len(referenceJob.Steps) {
			t.Fatal("Expected job's steps to be populated")
		}
		sort.SliceStable(job.Steps, func(i, j int) bool {
			return job.Steps[i].Name < job.Steps[j].Name
		})
		sort.SliceStable(referenceJob.Steps, func(i, j int) bool {
			return referenceJob.Steps[i].Name < referenceJob.Steps[j].Name
		})
		for i := 0; i < len(job.Steps); i++ {
			if job.Steps[i].Name != referenceJob.Steps[i].Name {
				t.Fatal("Step name mismatch")
			}
		}
	}
}

func checkForTimeouts(t *testing.T, app *server_test.TestServer, timeout time.Duration) int {
	realQueueService, ok := app.QueueService.(*queue.QueueService)
	require.True(t, ok, "Test app QueueService interface is not an instance of queue.QueueService")
	return realQueueService.CheckForTimeouts(timeout)
}
