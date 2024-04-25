package api_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client/clienttest"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
	"github.com/buildbeaver/buildbeaver/server/services"
)

func TestQueueAPI(t *testing.T) {
	ctx := context.Background()

	// Create a test server
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.RunnerAPIServer.Start()
	defer app.RunnerAPIServer.Stop(ctx)

	// Create a test API client to talk to the server via client certificate authentication
	apiClient, clientCert := clienttest.MakeClientCertificateAPIClient(t, app)

	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")

	// Create a runner in order to register the client certificate as a credential
	_ = server_test.CreateRunner(t, ctx, app, "", testCompany.ID, clientCert)

	// Create a repo and a commit to queue builds against
	repo := server_test.CreateRepo(t, ctx, app, testCompany.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, testCompany.ID)

	t.Run("Queue", testQueueBuild(app.QueueService, apiClient, commit))

	// Create a second parallel company, repo and commit
	testCompany2 := server_test.CreateCompanyLegalEntity(t, ctx, app, referencedata.TestCompany2Name, referencedata.TestCompany2LegalName, referencedata.TestCompany2Email)
	repo2 := server_test.CreateNamedRepo(t, ctx, app, "2", testCompany2.ID)
	commit2 := server_test.CreateCommit(t, ctx, app, repo2.ID, testCompany2.ID)

	// Enqueue a build for another company - this should be ignored when testing enqueue and dequeue for first company
	_, err = app.QueueService.EnqueueBuildFromCommit(context.Background(), nil, commit2, referencedata.TestRef, nil)

	// Repeat the queue (and dequeue) tests now we have a queued build for another company
	t.Run("Queue segregation", testQueueBuild(app.QueueService, apiClient, commit))
}

func testQueueBuild(service services.QueueService, client *client.APIClient, commit *models.Commit) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := client.Dequeue(ctx)
		require.Nil(t, err)

		if job != nil {
			t.Fatal("No job should be returned when queue is empty")
		}

		_, err = service.EnqueueBuildFromCommit(ctx, nil, commit, referencedata.TestRef, nil)
		require.Nil(t, err)

		t.Run("Dequeue job 1", testDequeueBuild(client))
		t.Run("Dequeue job 2", testDequeueBuild(client))
		t.Run("Dequeue job 3", testDequeueBuild(client))
		t.Run("Dequeue job 4", testDequeueBuild(client))

		job, err = client.Dequeue(ctx)
		require.Nil(t, err)

		if job != nil {
			t.Fatal("Expected all jobs to have been processed")
		}
	}
}

func testDequeueBuild(client *client.APIClient) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		job, err := client.Dequeue(ctx)
		require.Nil(t, err)

		if job == nil {
			t.Fatal("Expected to dequeue job")
		}

		switch job.Job.Name {
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

		for _, step := range job.Steps {
			_, err := client.UpdateStepStatus(ctx, step.ID, models.WorkflowStatusSucceeded, nil, step.ETag)
			require.Nil(t, err)
		}

		_, err = client.UpdateJobStatus(ctx, job.Job.ID, models.WorkflowStatusSucceeded, nil, job.Job.ETag)
		require.Nil(t, err)
	}
}

func testJobProperties(runnable *documents.RunnableJob, referenceJob *dto.JobGraph) func(t *testing.T) {
	return func(t *testing.T) {
		if runnable == nil {
			t.Fatal("Job should have been returned when queue is populated")
		}
		if runnable.Job.ETag == "" {
			t.Errorf("Expected job etag to be set: %s", runnable.Job.ETag)
		}
		if runnable.Job.Status != models.WorkflowStatusSubmitted {
			t.Errorf("Expected job to be in submitted state after dequeue: %s", runnable.Job.Status)
		}
		if len(runnable.Steps) != len(referenceJob.Steps) {
			t.Fatal("Expected job's steps to be populated")
		}
		sort.SliceStable(runnable.Steps, func(i, j int) bool {
			return runnable.Steps[i].Name < runnable.Steps[j].Name
		})
		sort.SliceStable(referenceJob.Steps, func(i, j int) bool {
			return referenceJob.Steps[i].Name < referenceJob.Steps[j].Name
		})
		for i := 0; i < len(runnable.Steps); i++ {
			if runnable.Steps[i].Name != referenceJob.Steps[i].Name {
				t.Fatal("Step name mismatch")
			}
		}
	}
}

func testQueueSegregation(service services.QueueService, client *client.APIClient) func(t *testing.T) {
	return func(t *testing.T) {
	}
}
