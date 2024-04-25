package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client/clienttest"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

type expectedBuilds struct {
	Running       int
	Upcoming      int
	Completed     int
	LegalEntityId models.LegalEntityID
}

func TestBuildSummaryAPI(t *testing.T) {
	ctx := context.Background()
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.CoreAPIServer.Start()
	defer app.CoreAPIServer.Stop(ctx)

	// One legal entity we will have builds against, and create an APIClient for them
	legalEntityA, identityA := server_test.CreatePersonLegalEntity(t, ctx, app, "test", "Jim Bob", "jim@bob.com")
	tokenA, _, err := app.CredentialService.CreateSharedSecretCredential(ctx, nil, identityA.ID, true)
	require.Nil(t, err)
	apiClientA, err := client.NewAPIClient(
		[]string{app.CoreAPIServer.GetServerURL()},
		client.NewSharedSecretAuthenticator(client.SharedSecretToken(tokenA.String()), app.LogFactory),
		app.LogFactory)
	require.Nil(t, err)

	// And another without the builds
	legalEntityB, identityB := server_test.CreatePersonLegalEntity(t, ctx, app, "tester", "Jimmy Bibby", "jimmy@bibby.com")
	tokenB, _, err := app.CredentialService.CreateSharedSecretCredential(ctx, nil, identityB.ID, true)
	require.Nil(t, err)
	apiClientB, err := client.NewAPIClient(
		[]string{app.CoreAPIServer.GetServerURL()},
		client.NewSharedSecretAuthenticator(client.SharedSecretToken(tokenB.String()), app.LogFactory),
		app.LogFactory)
	require.Nil(t, err)

	// Runners must exist that are capable of running the builds we enqueue or the builds will immediately fail
	_, clientCertA := clienttest.MakeClientCertificateAPIClient(t, app)
	server_test.CreateRunner(t, ctx, app, "test", legalEntityA.ID, clientCertA)
	_, clientCertB := clienttest.MakeClientCertificateAPIClient(t, app)
	server_test.CreateRunner(t, ctx, app, "test", legalEntityB.ID, clientCertB)

	// Create three repos to ensure we have commits across them all
	repoA := server_test.CreateNamedRepo(t, ctx, app, "a", legalEntityA.ID)
	_ = server_test.CreateNamedRepo(t, ctx, app, "b", legalEntityA.ID)
	_ = server_test.CreateNamedRepo(t, ctx, app, "c", legalEntityA.ID)

	// Create our expected structure
	expectedBuilds := expectedBuilds{
		Running:       0,
		Upcoming:      0,
		Completed:     0,
		LegalEntityId: legalEntityA.ID,
	}

	// We shouldn't have any builds to start off with
	builds := getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	checkBuildSummaryResult(t, builds, expectedBuilds)

	var createdBuilds []*dto.BuildGraph

	// Create and queue a build to check that it comes back
	createdBuilds = append(createdBuilds, server_test.CreateAndQueueBuild(t, ctx, app, repoA.ID, legalEntityA.ID, ""))

	builds = getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	expectedBuilds.Upcoming = 1
	checkBuildSummaryResult(t, builds, expectedBuilds)

	// Create more builds than our default limit to ensure we still only get 10 back
	// 21 upcoming | 0 running | 0 completed
	for i := 0; i < 20; i++ {
		createdBuilds = append(createdBuilds, server_test.CreateAndQueueBuild(t, ctx, app, repoA.ID, legalEntityA.ID, ""))
	}

	builds = getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	expectedBuilds.Upcoming = 10
	checkBuildSummaryResult(t, builds, expectedBuilds)

	// Now move the first Upcoming build into Running state.
	// 20 upcoming | 1 running | 0 completed
	firstUpcomingBuild := createdBuilds[0]
	createdBuilds = createdBuilds[1:]

	firstUpcomingBuild.Status = models.WorkflowStatusRunning
	err = app.BuildService.Update(ctx, nil, firstUpcomingBuild.Build)
	require.NoError(t, err)

	// Check we still have 10 (limit) Upcoming
	builds = getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	expectedBuilds.Running = 1
	checkBuildSummaryResult(t, builds, expectedBuilds)
	require.Equal(t, builds.Running[0].Build.ID, firstUpcomingBuild.ID)

	// Now move 11 Upcoming builds into Running state
	// 9 upcoming | 12 running | 0 completed
	for i := 0; i < 11; i++ {
		nextBuild := createdBuilds[0]
		createdBuilds = createdBuilds[1:]
		nextBuild.Status = models.WorkflowStatusRunning
		err = app.BuildService.Update(ctx, nil, nextBuild.Build)
		require.NoError(t, err)
	}

	// Check we now have 9 upcoming | 12 running (10 limit)
	builds = getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	expectedBuilds.Upcoming = 9
	expectedBuilds.Running = 10
	checkBuildSummaryResult(t, builds, expectedBuilds)

	// Now move 3 Upcoming builds into Failed state and 3 into completed
	// 3 upcoming | 12 running | 6 completed
	for i := 0; i < 3; i++ {
		nextBuild := createdBuilds[0]
		createdBuilds = createdBuilds[1:]
		nextBuild.Status = models.WorkflowStatusFailed
		err = app.BuildService.Update(ctx, nil, nextBuild.Build)
		require.NoError(t, err)
		nextBuild = createdBuilds[0]
		createdBuilds = createdBuilds[1:]
		nextBuild.Status = models.WorkflowStatusSucceeded
		err = app.BuildService.Update(ctx, nil, nextBuild.Build)
		require.NoError(t, err)
	}

	// Check we now have 3 upcoming | 12 running | 6 completed
	builds = getBuildSummary(t, ctx, apiClientA, legalEntityA.ID)
	expectedBuilds.Upcoming = 3
	expectedBuilds.Running = 10
	expectedBuilds.Completed = 6
	checkBuildSummaryResult(t, builds, expectedBuilds)

	// Check that our other legal entity has no builds
	builds = getBuildSummary(t, ctx, apiClientB, legalEntityB.ID)
	expectedBuilds.Upcoming = 0
	expectedBuilds.Running = 0
	expectedBuilds.Completed = 0
	checkBuildSummaryResult(t, builds, expectedBuilds)

	// Check that we cannot access builds for a different legal entity
	testInvalidAccess(t, ctx, apiClientA, legalEntityB.ID)
	testInvalidAccess(t, ctx, apiClientB, legalEntityA.ID)
}

func getBuildSummary(t *testing.T, ctx context.Context, client *client.APIClient, legalEntityId models.LegalEntityID) *documents.BuildSummary {
	summary, err := client.GetBuildSummary(ctx, legalEntityId)
	require.Nil(t, err)
	return summary
}

func testInvalidAccess(t *testing.T, ctx context.Context, client *client.APIClient, legalEntityId models.LegalEntityID) {
	_, err := client.GetBuildSummary(ctx, legalEntityId)
	require.Error(t, err)
}

func checkBuildSummaryResult(t *testing.T, builds *documents.BuildSummary, expectedBuilds expectedBuilds) {
	require.Len(t, builds.Upcoming, expectedBuilds.Upcoming)
	checkBuildOrder(t, builds.Upcoming, []models.WorkflowStatus{models.WorkflowStatusSubmitted, models.WorkflowStatusQueued})

	require.Len(t, builds.Running, expectedBuilds.Running)
	checkBuildOrder(t, builds.Running, []models.WorkflowStatus{models.WorkflowStatusRunning})

	require.Len(t, builds.Completed, expectedBuilds.Completed)
	checkBuildOrder(t, builds.Completed, []models.WorkflowStatus{models.WorkflowStatusSucceeded, models.WorkflowStatusFailed, models.WorkflowStatusCanceled})
}

func checkBuildOrder(t *testing.T, buildResults []*documents.BuildSearchResult, status []models.WorkflowStatus) {
	// Validate the ordering of the returned search results
	lastBuildTime := models.NewTime(time.Now()).String()
	lastBuildId := "build:000"
	for _, buildResult := range buildResults {
		if buildResult.Build.CreatedAt.String() == lastBuildTime {
			require.LessOrEqual(t, buildResult.Build.ID.String(), lastBuildId, "expected decreasing build ids")
		} else {
			require.Less(t, buildResult.Build.CreatedAt.String(), lastBuildTime, "expected decreasing build created at times")
		}
		lastBuildTime = buildResult.Build.CreatedAt.String()
		lastBuildId = buildResult.Build.ID.String()

		require.Contains(t, status, buildResult.Build.Status)
	}
}
