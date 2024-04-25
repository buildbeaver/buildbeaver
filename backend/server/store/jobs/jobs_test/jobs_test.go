package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestJob(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	ctx := context.Background()

	legalEntityA, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntityA.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntityA.ID)

	logDescriptor := models.NewLogDescriptor(models.NewTime(time.Now()), models.LogDescriptorID{}, referencedata.ReferenceBuild.ID.ResourceID)
	err = app.LogStore.Create(context.Background(), nil, logDescriptor)
	require.Nil(t, err)

	// Create a build graph
	build := referencedata.GenerateBuild(repo.ID, commit.ID, logDescriptor.ID, "refs/heads/master", 2)
	err = app.BuildService.Create(ctx, nil, build.Build)
	require.Nil(t, err)

	t.Run("CreateRunner", testJobCreate(app.JobStore, build))
}

func testJobCreate(store store.JobStore, build *dto.BuildGraph) func(t *testing.T) {
	return func(t *testing.T) {
		err := store.Create(context.Background(), nil, build.Jobs[0].Job)
		require.Nil(t, err)
		t.Run("Read", testJobRead(store, build.Jobs[0].ID, build.Jobs[0].Job))
		t.Run("CreateDependency", testCreateJobDependency(store, build))
	}
}

func testJobRead(store store.JobStore, testJobID models.JobID, referenceJob *models.Job) func(t *testing.T) {
	return func(t *testing.T) {
		job, err := store.Read(context.Background(), nil, testJobID)
		require.Nil(t, err)

		if job.ID != referenceJob.ID {
			t.Error("Unexpected ResourceID")
		}

		if job.CreatedAt != referenceJob.CreatedAt {
			t.Error("Unexpected CreatedAt")
		}

		if job.UpdatedAt != referenceJob.UpdatedAt {
			t.Error("Unexpected UpdatedAt")
		}

		if job.DeletedAt != referenceJob.DeletedAt {
			t.Error("Unexpected DeletedAt")
		}

		if job.BuildID != referenceJob.BuildID {
			t.Error("Unexpected BuildID")
		}

		if job.Ref != referenceJob.Ref {
			t.Error("Unexpected Ref")
		}

		if job.ETag != referenceJob.ETag {
			t.Error("Unexpected ETag")
		}

		if job.Status != referenceJob.Status {
			t.Error("Unexpected Status")
		}

		if job.Name != referenceJob.Name {
			t.Error("Unexpected Key")
		}

		if job.Description != referenceJob.Description {
			t.Error("Unexpected Desc")
		}

		if job.Type != referenceJob.Type {
			t.Error("Unexpected Type")
		}
		if len(job.Environment) != len(referenceJob.Environment) {
			t.Error("Mismatched Environment")
		} else {
			for i := 0; i < len(job.Environment); i++ {
				env := job.Environment[i]
				testEnv := referenceJob.Environment[i]
				if env.Name != testEnv.Name {
					t.Error("Mismatched Key")
				}
				if env.Value != testEnv.Value {
					t.Error("Mismatched Value")
				}
				if env.ValueFromSecret != testEnv.ValueFromSecret {
					t.Error("Mismatched ValueFromSecret")
				}
			}
		}
	}
}

func testCreateJobDependency(store store.JobStore, build *dto.BuildGraph) func(t *testing.T) {
	return func(t *testing.T) {
		err := store.Create(context.Background(), nil, build.Jobs[1].Job)
		require.Nil(t, err)
		err = store.CreateDependency(context.Background(), nil, build.Jobs[1].BuildID, build.Jobs[1].ID, build.Jobs[0].ID)
		require.Nil(t, err)
	}
}
