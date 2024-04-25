package artifacts_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
)

func TestArtifact(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	commit := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)

	logDescriptor := models.NewLogDescriptor(models.NewTime(time.Now()), models.LogDescriptorID{}, referencedata.ReferenceBuild.ID.ResourceID)
	err = app.LogStore.Create(context.Background(), nil, logDescriptor)
	require.NoError(t, err)

	build := referencedata.GenerateBuild(repo.ID, commit.ID, logDescriptor.ID, "refs/heads/master", 2)

	// Create a build graph
	build1ID := build.ID
	build1Job1ID := build.Jobs[0].ID
	build1Job2ID := build.Jobs[1].ID
	err = app.BuildService.Create(ctx, nil, build.Build)
	require.NoError(t, err)
	for _, jGraph := range build.Jobs {
		err = app.JobService.Create(ctx, nil, &dto.CreateJob{
			Job:   jGraph.Job,
			Build: build.Build,
		})
		require.NoError(t, err)
		for _, step := range jGraph.Steps {
			err = app.StepService.Create(ctx, nil, &dto.CreateStep{
				Step: step,
				Job:  jGraph.Job,
			})
			require.NoError(t, err)
		}
	}

	// Create a second matching build graph to simulate a second run of the same build (swap out ids)
	build.ID = models.NewBuildID()
	build2ID := build.ID
	err = app.BuildService.Create(ctx, nil, build.Build)
	require.NoError(t, err)
	for i, jGraph := range build.Jobs {
		if i == 0 {
			jGraph.IndirectToJobID = build1Job1ID
		}
		jGraph.ID = models.NewJobID()
		jGraph.BuildID = build.ID
		err = app.JobService.Create(ctx, nil, &dto.CreateJob{
			Job:   jGraph.Job,
			Build: build.Build,
		})
		require.NoError(t, err)
		for _, step := range jGraph.Steps {
			step.ID = models.NewStepID()
			step.JobID = jGraph.ID
			err = app.StepService.Create(ctx, nil, &dto.CreateStep{
				Step: step,
				Job:  jGraph.Job,
			})
			require.NoError(t, err)
		}
	}

	search := models.ArtifactSearch{
		BuildID:   build1ID,
		JobName:   &build.Jobs[0].Name,
		GroupName: nil,
		Pagination: models.Pagination{
			Limit:  5,
			Cursor: nil,
		},
	}

	// No artifacts should exist
	artifacts, _, err := app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 0)

	// Create an artifact for build1 job1
	artifact1Data := models.NewArtifactData(
		models.NewTime(time.Now()),
		"path-foo-bar",
		build1Job1ID,
		"foobar",
		"path/foo/bar")
	artifact1, err := app.ArtifactStore.Create(context.Background(), nil, artifact1Data)
	require.NoError(t, err)
	require.True(t, artifact1.ID.Valid())

	// Ensure that we get a duplicate error if we try to create it again
	_, err = app.ArtifactStore.Create(context.Background(), nil, artifact1Data)
	require.Error(t, err)

	// Ensure we are able to retrieve the artifact that is a duplicate from its unique fields
	returnedArtifact, err := app.ArtifactStore.FindByUniqueFields(context.Background(), nil, artifact1Data)
	require.NoError(t, err)
	require.NotNil(t, returnedArtifact)
	require.Equal(t, artifact1Data.JobID, returnedArtifact.JobID)
	require.Equal(t, artifact1Data.Name, returnedArtifact.Name)
	require.Equal(t, artifact1Data.Path, returnedArtifact.Path)

	// And ensure that we do not get an artifact if we pass in only partially matching fields
	artifactUniqueData := models.NewArtifactData(
		models.NewTime(time.Now()),
		artifact1Data.Name,
		build1Job1ID,
		"foobar",
		"a-very-different-path-that-will-hopefully-not-match")
	artifactUnique, err := app.ArtifactStore.FindByUniqueFields(context.Background(), nil, artifactUniqueData)
	require.Error(t, err)
	require.False(t, artifactUnique.ID.Valid())

	artifact1Read, err := app.ArtifactStore.Read(context.Background(), nil, artifact1.ID)
	require.NoError(t, err)
	require.Equal(t, artifact1, artifact1Read)

	// 1 artifact should exist
	artifacts, _, err = app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	// Create another artifact for build1 job1
	artifact2Data := models.NewArtifactData(
		models.NewTime(time.Now()),
		"path-baz-giggy",
		build1Job1ID,
		"bazol",
		"path/baz/giggy")
	artifact2, err := app.ArtifactStore.Create(context.Background(), nil, artifact2Data)
	require.NoError(t, err)
	require.True(t, artifact2.ID.Valid())

	// 2 artifacts should exist
	artifacts, _, err = app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)

	// Build2 job1 indirected to build1 job1, so we should find two artifacts here
	search.BuildID = build2ID
	artifacts, _, err = app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)

	// Create an artifact for build1 job2
	artifact4Data := models.NewArtifactData(
		models.NewTime(time.Now()),
		"path-baz-giggy11",
		build1Job2ID,
		"bazol11",
		"path/baz/giggy11")
	artifact4, err := app.ArtifactStore.Create(context.Background(), nil, artifact4Data)
	require.NoError(t, err)
	require.True(t, artifact4.ID.Valid())

	// Build2 job2 is not indirected and no artifacts were created for it, shouldn't find anything here
	search.BuildID = build2ID
	search.JobName = &build.Jobs[1].Name
	artifacts, _, err = app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 0)

	// However we should find it if we go back to searching for build1
	search.BuildID = build1ID
	search.JobName = &build.Jobs[1].Name
	artifacts, _, err = app.ArtifactStore.Search(ctx, nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	// Now test pagination
	search = models.ArtifactSearch{
		BuildID:   build1ID,
		JobName:   &build.Jobs[0].Name,
		GroupName: nil,
		Pagination: models.Pagination{
			Limit:  1,
			Cursor: nil,
		},
	}

	artifacts, cursor, err := app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	search.Cursor = cursor.Next

	artifacts, cursor, err = app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	search.Cursor = cursor.Prev

	artifacts, cursor, err = app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	search.Cursor = nil
	search.Limit = 10

	artifacts, cursor, err = app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)

	search.GroupName = &artifact1Data.GroupName

	artifacts, cursor, err = app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	name := models.ResourceName("fake name")
	search.GroupName = &name

	artifacts, cursor, err = app.ArtifactStore.Search(context.Background(), nil, models.NoIdentity, search)
	require.NoError(t, err)
	require.Len(t, artifacts, 0)
}
