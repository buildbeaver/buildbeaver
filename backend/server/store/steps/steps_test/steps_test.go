package steps_test

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

func TestStep(t *testing.T) {
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

	err = app.JobStore.Create(context.Background(), nil, build.Jobs[0].Job)
	require.Nil(t, err)

	t.Run("CreateRunner", testStepCreate(app.StepStore, build))
}

func testStepCreate(store store.StepStore, build *dto.BuildGraph) func(t *testing.T) {
	return func(t *testing.T) {
		err := store.Create(context.Background(), nil, build.Jobs[0].Steps[0])
		require.Nil(t, err)
		t.Run("Read", testStepRead(store, build.Jobs[0].Steps[0]))
	}
}

func testStepRead(store store.StepStore, referenceStep *models.Step) func(t *testing.T) {
	return func(t *testing.T) {
		step, err := store.Read(context.Background(), nil, referenceStep.ID)
		require.Nil(t, err)
		if step.ID != referenceStep.ID {
			t.Error("Unexpected ResourceID")
		}
		if step.CreatedAt != referenceStep.CreatedAt {
			t.Error("Unexpected CreatedAt")
		}
		if step.UpdatedAt != referenceStep.UpdatedAt {
			t.Error("Unexpected UpdatedAt")
		}
		if step.DeletedAt != referenceStep.DeletedAt {
			t.Error("Unexpected DeletedAt")
		}
		if step.ETag != referenceStep.ETag {
			t.Error("Unexpected ETag")
		}
		if step.Status != referenceStep.Status {
			t.Error("Unexpected Status")
		}
		if step.Name != referenceStep.Name {
			t.Error("Unexpected Key")
		}
		if step.Description != referenceStep.Description {
			t.Error("Unexpected Desc")
		}
		if len(step.Commands) != len(referenceStep.Commands) {
			t.Error("Mismatched Commands")
		} else {
			for i := 0; i < len(step.Commands); i++ {
				command := step.Commands[i]
				testCommand := referenceStep.Commands[i]

				if command != testCommand {
					t.Error("Mismatched Command")
				}
			}
		}
	}
}
