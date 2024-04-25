package builds_test

import (
	"context"
	"fmt"
	"log"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client/clienttest"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestBuild(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err, "Error initializing app")
	defer cleanup()
	ctx := context.Background()

	legalEntityA := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")
	legalEntityB := server_test.CreateCompanyLegalEntity(t, ctx, app, referencedata.TestCompany3Name, referencedata.TestCompany3LegalName, referencedata.TestCompany3Email)

	// Runners must exist that are capable of running the builds we enqueue or the builds will immediately fail
	_, clientCertA := clienttest.MakeClientCertificateAPIClient(t, app)
	server_test.CreateRunner(t, ctx, app, "test", legalEntityA.ID, clientCertA)
	_, clientCertB := clienttest.MakeClientCertificateAPIClient(t, app)
	server_test.CreateRunner(t, ctx, app, "test", legalEntityB.ID, clientCertB)

	// Create a new repo to use for this test. We will perform all searches within the context of this repo.
	repo := server_test.CreateRepo(t, ctx, app, legalEntityA.ID)

	// Commit used to hide away commits we do not want to show up in filtering.
	commit4 := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntityA.ID)

	buildId := dto.BuildGraph{Build: &models.Build{ID: models.NewBuildID()}}
	t.Run("CreateBuild", testBuildCreate(app, repo.ID, legalEntityA.ID, &buildId))

	// Change the first commit to be a status we do not check in later tests.
	// Note: If you filter tests then CreateRunner might not run so this can be skipped
	build, err := app.BuildStore.Read(context.Background(), nil, buildId.ID)
	if err == nil {
		require.Nil(t, err)
		build.Status = models.WorkflowStatusUnknown
		build.CommitID = commit4.ID
		err = app.BuildStore.Update(context.Background(), nil, build)
		require.NoError(t, err, "Error resetting build")
	}

	t.Run("BuildSearchFiltering", testBuildSearchFiltering(app, repo.ID, legalEntityB.ID))
}

func testBuildSearch(buildStore store.BuildStore, ref string, commitSha string, commitAuthorId *models.LegalEntityID, includeStatuses *[]models.WorkflowStatus, expectedBuildsFound int) func(t *testing.T) {
	return func(t *testing.T) {
		search := models.NewBuildSearch()
		search.Ref = ref
		search.CommitSHA = commitSha
		if includeStatuses != nil {
			search.IncludeStatuses = *includeStatuses
		}
		if commitAuthorId != nil {
			search.CommitAuthorID = commitAuthorId
		}
		search.Limit = 1
		buildResults := runSearch(t, buildStore, search)
		if expectedBuildsFound > 0 {
			// Validate the ordering of the returned search results
			lastBuildTime := models.NewTime(time.Now()).String()
			lastBuildId := "build:000"
			for _, buildResult := range buildResults {
				if buildResult.Build.CreatedAt.String() == lastBuildTime {
					assert.LessOrEqual(t, buildResult.Build.ID.String(), lastBuildId, "expected decreasing build ids")
				} else {
					assert.Less(t, buildResult.Build.CreatedAt.String(), lastBuildTime, "expected decreasing build created at times")
				}
				lastBuildTime = buildResult.Build.CreatedAt.String()
				lastBuildId = buildResult.Build.ID.String()
			}
		}
		assert.Equal(t, expectedBuildsFound, len(buildResults), "Build Search returned an unexpected number of builds")
	}
}

// testBuildSearchFiltering tests the various filtering options for a build. Note that
func testBuildSearchFiltering(app *server_test.TestServer, repoId models.RepoID, legalEntityID models.LegalEntityID) func(t *testing.T) {
	return func(t *testing.T) {
		buildStore := app.BuildStore

		t.Run("Search-Status-Queued-NoBuilds", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued},
			0,
		))

		buildsToCreate := 10
		refsToPick := []string{referencedata.TestRef2, referencedata.TestRef3}
		workflowsToUse := []models.WorkflowStatus{models.WorkflowStatusRunning, models.WorkflowStatusSucceeded, models.WorkflowStatusQueued}

		firstSha := models.CommitID{}

		// 10 queued / running / succeeded
		for i := 0; i < buildsToCreate; i++ {
			ref := refsToPick[i%2]
			for i, workflowStatus := range workflowsToUse {
				build := server_test.CreateAndQueueBuild(t, context.Background(), app, repoId, legalEntityID, ref)
				if i == 0 {
					firstSha = build.CommitID
				}
				if workflowStatus != models.WorkflowStatusQueued {
					build.Status = workflowStatus
					err := app.BuildService.Update(context.Background(), nil, build.Build)
					require.NoError(t, err)
				}
			}
		}

		firstCommit, _ := app.CommitStore.Read(context.Background(), nil, firstSha)

		// ---------- Build Ref Filtering ----------
		t.Run("Search-Ref", testBuildSearch(
			buildStore,
			referencedata.TestRef2,
			"",
			nil,
			nil,
			(buildsToCreate*3)/2,
		))

		t.Run("Search-Ref2", testBuildSearch(
			buildStore,
			referencedata.TestRef3,
			"",
			nil,
			nil,
			(buildsToCreate*3)/2,
		))

		t.Run("Search-Ref3", testBuildSearch(
			buildStore,
			"nothing/using/this/ref",
			"",
			nil,
			nil,
			0,
		))

		// ---------- Workflow Status Filtering ----------
		t.Run("Search-Status-Queued", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued},
			buildsToCreate,
		))

		t.Run("Search-Status-Running", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusRunning},
			buildsToCreate,
		))

		t.Run("Search-Status-Succeeded", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusSucceeded},
			buildsToCreate,
		))

		t.Run("Search-Status-Queued-Succeeded", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued, models.WorkflowStatusSucceeded},
			buildsToCreate*2,
		))

		t.Run("Search-Status-Queued-Running-Succeeded", testBuildSearch(
			buildStore,
			"",
			"",
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued, models.WorkflowStatusRunning, models.WorkflowStatusSucceeded},
			buildsToCreate*3,
		))

		// ---------- Commit SHA filtering ----------
		t.Run("Search-SHA-Invalid", testBuildSearch(
			buildStore,
			"",
			"an-invalid-commit-sha",
			nil,
			nil,
			0,
		))

		t.Run("Search-SHA", testBuildSearch(
			buildStore,
			"",
			firstCommit.SHA,
			nil,
			nil,
			1,
		))

		// ---------- Commit Author ID filtering ----------
		fakeLegalEntityId, _ := models.ParseResourceID("legal-entity:not-a-real-one")
		legalEntity := models.LegalEntityIDFromResourceID(fakeLegalEntityId)
		t.Run("Search-AuthorId-Invalid", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntity,
			nil,
			0,
		))

		t.Run("Search-AuthorId", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntityID,
			nil,
			buildsToCreate*3,
		))

		// ---------- Mixed filtering ----------
		t.Run("Search-AuthorId-Running", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntityID,
			&[]models.WorkflowStatus{models.WorkflowStatusRunning},
			buildsToCreate,
		))

		t.Run("Search-AuthorId-Running-Failed", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntityID,
			&[]models.WorkflowStatus{models.WorkflowStatusRunning, models.WorkflowStatusFailed},
			buildsToCreate,
		))

		t.Run("Search-AuthorId-Queued-Running-Succeeded", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntityID,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued, models.WorkflowStatusRunning, models.WorkflowStatusSucceeded},
			buildsToCreate*3,
		))

		t.Run("Search-AuthorId-Failed", testBuildSearch(
			buildStore,
			"",
			"",
			&legalEntityID,
			&[]models.WorkflowStatus{models.WorkflowStatusFailed},
			0,
		))

		t.Run("Search-SHA-Running", testBuildSearch(
			buildStore,
			"",
			firstCommit.SHA,
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusRunning},
			1,
		))

		t.Run("Search-SHA-Running-2", testBuildSearch(
			buildStore,
			"",
			firstCommit.SHA,
			nil,
			&[]models.WorkflowStatus{models.WorkflowStatusQueued},
			0,
		))
	}
}

func testBuildCreate(app *server_test.TestServer, repoId models.RepoID, legalEntityID models.LegalEntityID, createdBuild *dto.BuildGraph) func(t *testing.T) {

	return func(t *testing.T) {
		// Search for builds; should not find any within the new repo created by this test
		t.Run("Search-1", testBuildSearchByRepo(
			app.BuildStore, repoId,
			referencedata.TestRef,
			0,
			nil,
		))
		t.Run("Search-2", testBuildSearchByCommit(
			app.BuildStore,
			models.NewCommitID(),
			referencedata.TestRef,
			0,
			nil,
		))

		// Create new build
		build := server_test.CreateAndQueueBuild(t, context.Background(), app, repoId, legalEntityID, "")
		createdBuild.ID = build.ID

		t.Run("Read", testBuildRead(app.BuildStore, build.ID, build.Build))

		// Repeat searches, should now find 1 build (again the searches are only within the repo created by this test)
		t.Run("Search-3", testBuildSearchByRepo(
			app.BuildStore, repoId,
			build.Ref,
			1,
			&build.ID,
		))
		t.Run("Search-4", testBuildSearchByCommit(
			app.BuildStore,
			build.CommitID,
			build.Ref,
			1,
			&build.ID,
		))
	}
}

func testBuildRead(store store.BuildStore, testBuildID models.BuildID, referenceBuild *models.Build) func(t *testing.T) {

	return func(t *testing.T) {
		build, err := store.Read(context.Background(), nil, testBuildID)
		require.NoError(t, err, "Error reading build")

		assert.Equal(t, referenceBuild.ID, build.ID)
		assert.Equal(t, referenceBuild.CreatedAt, build.CreatedAt)
		assert.Equal(t, referenceBuild.UpdatedAt, build.UpdatedAt)
		assert.Equal(t, referenceBuild.DeletedAt, build.DeletedAt)
		assert.Equal(t, referenceBuild.CommitID, build.CommitID)
		assert.Equal(t, referenceBuild.Ref, build.Ref)
		assert.Equal(t, referenceBuild.ETag, build.ETag)
		assert.Equal(t, referenceBuild.Status, build.Status)
		assert.Equal(t, len(referenceBuild.Opts.NodesToRun), len(build.Opts.NodesToRun), "Unexpected opts.NodesToRun len")
		for i := 0; i < len(build.Opts.NodesToRun); i++ {
			assert.Equal(t, referenceBuild.Opts.NodesToRun[i], build.Opts.NodesToRun[i],
				fmt.Sprintf("Unexpected opts.NodesToRun at index %d", i))
		}
	}
}

func testBuildSearchByRepo(
	buildStore store.BuildStore,
	testRepoID models.RepoID,
	testRef string,
	expectedBuildsFound int,
	requiredBuildID *models.BuildID,
) func(t *testing.T) {
	return func(t *testing.T) {
		search := models.NewBuildSearchForRepo(
			testRepoID,
			testRef,
			true, // exclude failed builds
			[]models.WorkflowStatus{},
			1, // read a single build at once to test pagination
		)
		builds := runSearch(t, buildStore, search)
		assert.Equal(t, expectedBuildsFound, len(builds), "Build Search by Repo found wrong number of builds")
		if expectedBuildsFound > 0 && requiredBuildID != nil {
			checkBuildsContainID(t, builds, *requiredBuildID)
		}
	}
}

func testBuildSearchByCommit(
	buildStore store.BuildStore,
	testCommitID models.CommitID,
	testRef string,
	expectedBuildsFound int,
	requiredBuildID *models.BuildID,
) func(t *testing.T) {
	return func(t *testing.T) {
		search := models.NewBuildSearchForCommit(
			testCommitID,
			testRef,
			true, // exclude failed builds
			[]models.WorkflowStatus{},
			1, // read a single build at once to test pagination
		)
		builds := runSearch(t, buildStore, search)
		assert.Equal(t, expectedBuildsFound, len(builds), "Build Search by Commit found wrong number of builds")
		if expectedBuildsFound > 0 && requiredBuildID != nil {
			checkBuildsContainID(t, builds, *requiredBuildID)
		}
	}
}

// Run a search, page through and collect all the results in both directions, comparing the two result sets and
// returning the default direction.
func runSearch(t *testing.T, buildStore store.BuildStore, search *models.BuildSearch) []*models.BuildSearchResult {
	var allBuilds []*models.BuildSearchResult
	var reverseBuilds []*models.BuildSearchResult
	finishedSearch := false
	finishedReverseSearch := false
	for !finishedSearch || !finishedReverseSearch {
		moreBuilds, cursor, err := buildStore.Search(context.Background(), nil, models.NoIdentity, search)
		assert.NoError(t, err)
		if err != nil {
			break // stop on error
		}
		if !finishedSearch {
			allBuilds = append(allBuilds, moreBuilds...)
			if cursor != nil && cursor.Next != nil {
				search.Cursor = cursor.Next
			} else {
				finishedSearch = true
				reverseBuilds = append(reverseBuilds, moreBuilds...)
				if cursor != nil && cursor.Prev != nil {
					search.Cursor = cursor.Prev
				} else {
					finishedReverseSearch = true
				}
			}
		} else {
			reverseBuilds = append(reverseBuilds, moreBuilds...)
			if cursor != nil && cursor.Prev != nil {
				search.Cursor = cursor.Prev
			} else {
				finishedReverseSearch = true
			}
		}
	}

	compareSearchResults(t, allBuilds, reverseBuilds)

	return allBuilds
}

// Because we perform all build searches here to a limit of 1, we can make use of this for testing the ordering of the
// returned build results
func compareSearchResults(t *testing.T, b1 []*models.BuildSearchResult, b2 []*models.BuildSearchResult) {
	b1Length := len(b1)

	require.Equal(t, b1Length, len(b2), "Expected Next direction search results to be same length as Prev direction")

	if b1Length <= 0 {
		return
	}

	buildLength := b1Length - 1

	for i, buildResult := range b1 {
		assert.Equal(t, buildResult.Build.ID, b2[buildLength-i].Build.ID)
	}
}

// Checks that the specified list of builds contains a build with an ID of requiredBuildID.
// If not then a test failure is raised.
func checkBuildsContainID(t *testing.T, buildResults []*models.BuildSearchResult, requiredBuildID models.BuildID) {
	found := false
	for _, buildResult := range buildResults {
		if buildResult.Build.ID.Equal(requiredBuildID.ResourceID) {
			found = true
			break
		}
	}
	require.True(t, found, "Build results must include one with ID %s", requiredBuildID)
}

func TestBuildSearch(t *testing.T) {
	ctx := context.Background()
	cfg := server_test.TestConfig(t)
	cfg.LogLevels = "builds_table=trace"

	app, cleanup, err := server_test.New(cfg)
	require.NoError(t, err, "error initializing app")
	defer cleanup()

	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")
	server_test.CreateCompanyLegalEntity(t, ctx, app, referencedata.TestCompany2Name, referencedata.TestCompany2LegalName, referencedata.TestCompany2Email)

	repo := server_test.CreateRepo(t, ctx, app, testCompany.ID)

	commit1 := server_test.CreateCommit(t, ctx, app, repo.ID, testCompany.ID)

	var created []*models.Build
	for i := 0; i < 10; i++ {
		now := models.NewTime(time.Now().Add(1 * time.Second))
		build := &models.Build{
			ID:        models.NewBuildID(),
			RepoID:    repo.ID,
			CreatedAt: now,
			UpdatedAt: now,
			CommitID:  commit1.ID,
			Ref:       "heads/master",
			Status:    models.WorkflowStatusSubmitted,
		}
		logDescriptor, err := app.LogService.Create(ctx, nil, models.NewLogDescriptor(now, models.LogDescriptorID{}, build.ID.ResourceID))
		require.NoError(t, err)
		build.LogDescriptorID = logDescriptor.ID
		created = append(created, build)
		err = app.BuildService.Create(ctx, nil, build)
		require.NoError(t, err)
	}
	sort.Slice(created, func(i, j int) bool {
		return created[i].CreatedAt.After(created[j].CreatedAt.Time)
	})
	for i, build := range created {
		log.Printf("%d: %s, %s", i, build.CreatedAt, build.ID)
	}

	// Page forward through the set
	builds, cursor, err := app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.Nil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[0].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[1].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Next}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[2].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[3].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Next}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[4].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[5].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Next}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[6].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[7].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Next}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.Nil(t, cursor.Next)
	require.Equal(t, created[8].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[9].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	// Page backwards through the set
	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Prev}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[6].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[7].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Prev}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[4].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[5].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Prev}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.NotNil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[2].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[3].CreatedAt.String(), builds[1].Build.CreatedAt.String())

	builds, cursor, err = app.BuildStore.Search(ctx, nil, models.NoIdentity, &models.BuildSearch{Pagination: models.Pagination{Limit: 2, Cursor: cursor.Prev}})
	require.NoError(t, err)
	require.Len(t, builds, 2)
	require.NotNil(t, cursor)
	require.Nil(t, cursor.Prev)
	require.NotNil(t, cursor.Next)
	require.Equal(t, created[0].CreatedAt.String(), builds[0].Build.CreatedAt.String())
	require.Equal(t, created[1].CreatedAt.String(), builds[1].Build.CreatedAt.String())
}
