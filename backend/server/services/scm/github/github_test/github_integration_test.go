package github_test

import (
	"bytes"
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/server/services/sync"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client/clienttest"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	github_service "github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github/github_test_utils"
)

// eventTimeout is the maximum duration to wait for events to come in each time the test processes events.
// The test may end up taking many times this long since it waits for events in multiple places.
const eventTimeout = 20 * time.Second

// Pick an org to use for our tests
const testOrgName = github_test_utils.GitHubTestAccountOrg1Name

// TestGitHubIntegration runs a variety of integration tests with GitHub, including testing Webhook notifications
// for commits and Pull Requests, checking that suitable builds are queued.
func TestGitHubIntegration(t *testing.T) {
	t.Skip("Skipping GitHub integration test")

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Seed math/rand numbers
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Set up test server app
	app, cleanupServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanupServer()

	// Find the GitHub SCM service
	scm, err := app.SCMRegistry.Get(github_service.GitHubSCMName)
	githubService, ok := scm.(*github_service.GitHubService)
	require.True(t, ok, "GitHubService is of wrong type")

	// Set up GitHub client using our test account Personal Access Token, for direct interactions
	scmAuth := github_test_utils.MakeGitHubAuth(t)
	ghClient, err := github_test_utils.MakeGitHubTestClient(scmAuth)
	require.NoError(t, err, "Error creating GitHub client")

	// Sync to create our legal entity and identity in the database
	identity, err := app.SyncService.SyncAuthenticatedUser(ctx, scmAuth)
	assert.NoError(t, err)
	assert.NotNil(t, identity)

	// Set up a new test repo in GitHub under an organization, for use in this testing (and delete after)
	ghRepo, repoExternalID, cleanup, err := github_test_utils.SetupTestRepo(t, ghClient, testOrgName)
	defer cleanup()
	require.NoError(t, err)
	t.Logf("Test Repo set up, name %q, externalID %q", ghRepo.GetName(), repoExternalID)

	// Perform a baseline user-based sync to create the new repo in the database
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	assert.NoError(t, err)

	// Read Repo from the database so we have the ID
	repo, err := app.RepoStore.ReadByExternalID(ctx, nil, repoExternalID)
	require.NoError(t, err)
	require.Equal(t, *repo.ExternalID, repoExternalID, "Repo External ID doesn't match")
	repoID := repo.ID

	// Create a runner for the legal entity, so that builds won't immediately fail due to no compatible runner existing
	_, clientCertA := clienttest.MakeClientCertificateAPIClient(t, app)
	server_test.CreateRunner(t, ctx, app, "test", repo.LegalEntityID, clientCertA)

	// Enable the first repo so the GitHub SCM will process notifications for it.
	// Do not enable the second repo since we want to treat it as an external repo for PR testing.
	// The test account is assumed to have the 'BuildBeaver-Test' GitHub app installed for all repos
	// under the account, so we don't need to explicitly install the app when making a new repo.
	// The GitHub app has the smee.io Webhook URL set up, so notifications will automatically be
	// received for our new repo (and for every other repo in the test account).
	_, err = app.RepoService.UpdateRepoEnabled(ctx, repo.ID, dto.UpdateRepoEnabled{
		Enabled: true,
		ETag:    "",
	})
	require.NoError(t, err, "Error enabling repo")
	t.Logf("Enabled repo owned by org, ID %q", repo.ID)

	// Set up Smee to receive Webhook events from the GitHub test account.
	eventChan := make(chan *github_test_utils.SmeeNotification, 1000) // buffered chan; we won't always be reading
	smeeClient := github_test_utils.NewSmeeClientForGitHubTestAccount(app.LogFactory)
	err = smeeClient.SubscribeChan(eventChan)
	require.NoError(t, err, "Unable to subscribe to smee.io events")
	t.Log("Listening for smee client events...")

	// Perform basic tests
	testCommitConfigFile(t, app, githubService, ghClient, ghRepo, repoID, eventChan)
	testPullRequest(t, app, githubService, ghClient, ghRepo, repoID, eventChan)

	// Create a fork of the repo directly owned by the test account (no org), for testing cross-repo PRs.
	// Fork the repo immediately before testing cross-repo PRs, so we have a config file in the fork.
	ghForkedRepo, forkedRepoExternalID, cleanup, err := github_test_utils.SetupRepoFork(t, ghClient, "", ghRepo)
	defer cleanup()
	require.NoError(t, err)
	t.Logf("Test Forked Repo set up, externalID %q", forkedRepoExternalID)

	// Test cross-repo pull request before we sync, while the fork is an 'unknown' repo
	testCrossRepoPullRequest(t, app, githubService, ghClient, ghRepo, repoID, ghForkedRepo, eventChan)

	// Perform a sync again so the Repo is in our database, but do not enable the repo.
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	assert.NoError(t, err)
	forkedRepo, err := app.RepoStore.ReadByExternalID(ctx, nil, forkedRepoExternalID)
	require.NoError(t, err)
	require.Equal(t, *forkedRepo.ExternalID, forkedRepoExternalID, "Forked Repo External ID doesn't match")

	// Test cross-repo pull request again with the repo in our database
	testCrossRepoPullRequest(t, app, githubService, ghClient, ghRepo, repoID, ghForkedRepo, eventChan)

	// Test committing a bad config file at the end, after we've done the other tests with a valid config file
	testBadConfig(t, app, githubService, ghClient, ghRepo, repoID, eventChan)

	t.Log("Shutting down smee client...")
	smeeClient.UnsubscribeChan(eventChan)
}

func testCommitConfigFile(
	t *testing.T,
	app *server_test.TestServer,
	githubService *github_service.GitHubService,
	ghClient *github.Client,
	ghRepo *github.Repository,
	repoID models.RepoID,
	eventChan chan *github_test_utils.SmeeNotification,
) {
	// Commit a new config file to the repo. This should cause a Webhook notification.
	commitSHA, err := github_test_utils.CommitTestConfigFile(ghClient, ghRepo, false)
	require.NoError(t, err, "Error committing new config file to GitHub")
	t.Logf("Committed new config file, commit SHA=%q", commitSHA)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		github_test_utils.MatchPushEvent(ghRepo.GetID()))
	assert.NoError(t, err, "Error processing incoming Webhook events")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, true)
}

func testPullRequest(
	t *testing.T,
	app *server_test.TestServer,
	githubService *github_service.GitHubService,
	ghClient *github.Client,
	ghRepo *github.Repository,
	repoID models.RepoID,
	eventChan chan *github_test_utils.SmeeNotification,
) {
	// Create a new branch (off master)
	branchName, err := github_test_utils.CreateRandomBranch(t, ghClient, ghRepo)
	require.NoError(t, err, "Error creating new branch on GitHub")

	// Commit a new file to the new branch
	commitSHA, err := github_test_utils.CommitRandomFile(ghClient, ghRepo, branchName)
	require.NoError(t, err, "Error committing new random file on branch on GitHub")
	t.Logf("Committed new random file, commit SHA=%q", commitSHA)

	// Create a Pull Request for the new branch. This will cause 'push' and 'pull_request' notifications
	pullRequestID, err := github_test_utils.CreatePullRequest(ghClient, ghRepo, branchName)
	require.NoError(t, err, "Error creating Pull Request on GitHub")
	t.Logf("Created new Pull Request with GitHub ID %d", pullRequestID)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		github_test_utils.MatchPushEvent(ghRepo.GetID()),
		github_test_utils.MatchPullRequestEvent(ghRepo.GetID(), "opened"),
	)
	assert.NoError(t, err, "Error processing incoming Webhook events")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, true)

	// Commit another new file to the branch. This should cause a 'pull_request' event with action "synchronize",
	// as well as a 'push' event.
	commitSHA, err = github_test_utils.CommitRandomFile(ghClient, ghRepo, branchName)
	require.NoError(t, err, "Error committing new random file 2 on branch on GitHub")
	t.Logf("Committed a second new random file, commit SHA=%q", commitSHA)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		github_test_utils.MatchPushEvent(ghRepo.GetID()),
		github_test_utils.MatchPullRequestEvent(ghRepo.GetID(), "synchronize"),
	)
	assert.NoError(t, err, "Error processing incoming Webhook events")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, true)
}

func testCrossRepoPullRequest(
	t *testing.T,
	app *server_test.TestServer,
	githubService *github_service.GitHubService,
	ghClient *github.Client,
	ghRepo *github.Repository,
	repoID models.RepoID,
	ghForkedRepo *github.Repository,
	eventChan chan *github_test_utils.SmeeNotification,
) {
	// Create a new branch (off master) in the forked repo
	branchName, err := github_test_utils.CreateRandomBranch(t, ghClient, ghForkedRepo)
	require.NoError(t, err, "Error creating new branch on GitHub forked repo")

	// Commit a new file to the new branch in the forked repo
	commitSHA, err := github_test_utils.CommitRandomFile(ghClient, ghForkedRepo, branchName)
	require.NoError(t, err, "Error committing new random file on branch in forked repo on GitHub")
	t.Logf("Committed new random file for cross-repo PR, commit SHA=%q", commitSHA)

	// Use the login for the owner of the forked repo to specify the head repo for the GitHub API
	headRepoUserName := ghForkedRepo.GetOwner().GetLogin()

	// Create a Pull Request for the new branch. This will cause a 'pull_request' notification but no
	// 'Push' notification since the forked repo is not enabled.
	pullRequestID, err := github_test_utils.CreateCrossRepoPullRequest(ghClient, ghRepo, branchName, headRepoUserName)
	require.NoError(t, err, "Error creating cross-repo Pull Request on GitHub")
	t.Logf("Created new cross-repo Pull Request with GitHub ID %d", pullRequestID)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		github_test_utils.MatchPullRequestEvent(ghRepo.GetID(), "opened"),
	)
	assert.NoError(t, err, "Error processing incoming Webhook events for cross-repo PR")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, true)

	// Commit another new file to the forked repo's new branch. This should cause a 'pull_request' event with
	// action "synchronize", but still no 'Push' notification.
	commitSHA, err = github_test_utils.CommitRandomFile(ghClient, ghForkedRepo, branchName)
	require.NoError(t, err, "Error committing new random file 2 on cross-repo PR branch on GitHub")
	t.Logf("Committed a second new random file for cross-repo PR, commit SHA=%q", commitSHA)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		// TODO: This event doesn't always happen, is there something else we can wait on instead?
		github_test_utils.MatchPullRequestEvent(ghRepo.GetID(), "synchronize"),
	)
	assert.NoError(t, err, "Error processing incoming Webhook events for cross-repo PR")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, true)
}

func testBadConfig(
	t *testing.T,
	app *server_test.TestServer,
	githubService *github_service.GitHubService,
	ghClient *github.Client,
	ghRepo *github.Repository,
	repoID models.RepoID,
	eventChan chan *github_test_utils.SmeeNotification,
) {
	// Commit a bad config; we should not have a build queued but also not get an error back from the
	// Web service to prevent GitHub from retrying the notification.
	commitSHA, err := github_test_utils.CommitTestConfigFile(ghClient, ghRepo, true)
	require.NoError(t, err, "Error committing bad config file to GitHub")
	t.Logf("Committed bad config file, commit SHA=%q", commitSHA)
	err = github_test_utils.ProcessEventsUntilMatched(
		eventChan,
		eventTimeout,
		githubPushPullEventProcessor(t, githubService),
		github_test_utils.MatchPushEvent(ghRepo.GetID()))
	assert.NoError(t, err, "Error processing incoming Webhook events")
	checkCommitAndBuild(t, app, repoID, commitSHA, 1, false)
}

// Checks things that should have happened after a new commit is made to GitHub, in response to processing
// Webhook events:
// - a commit has been added to our database
// - the specified expected number of builds have been created for the commit
// - if any build was committed, and we expect there to be jobs, at least one job must have been queued for the build
func checkCommitAndBuild(
	t *testing.T,
	app *server_test.TestServer,
	repoID models.RepoID,
	commitSHA string,
	expectedNrBuilds int,
	expectedToHaveJobs bool,
) {
	ctx := context.Background()

	// Check that the commit was added to our database when the Webhook notification was processed
	commit, err := app.CommitStore.ReadBySHA(ctx, nil, repoID, commitSHA)
	assert.NoError(t, err, "Unable to find new commit in database after processing GitHub 'push' event")

	// Check that we have created the correct number of builds for the commit
	buildsForCommit, _, err := app.BuildStore.Search(ctx, nil, models.NoIdentity,
		models.NewBuildSearchForCommit(commit.ID, "", false, []models.WorkflowStatus{}, 2))
	assert.NoError(t, err, "error searching for builds for commit")
	t.Logf("Found %d build(s) for new commit", len(buildsForCommit))
	assert.Equal(t, expectedNrBuilds, len(buildsForCommit), "Found unexpected number of builds for new commit")

	// Check that there are jobs queued for each build
	if expectedToHaveJobs {
		for _, buildResult := range buildsForCommit {
			assert.Nil(t, buildResult.Build.Error)
			jobs, err := app.JobStore.ListByBuildID(ctx, nil, buildResult.Build.ID)
			assert.NoError(t, err, "error searching for jobs for build")
			t.Logf("Found %d job(s) queued for build for new commit", len(jobs))
			assert.GreaterOrEqual(t, len(jobs), 1, "No jobs found for queued build")
		}
	}
}

type eventAllowList map[string]bool

// githubPushPullEventProcessor returns a SmeeEventHandler function that checks if the event is a
// GitHub webhook event and calls the gitHubService to process the event.
// Only 'push' and 'pull_request' events will be sent to the gitHubService; all other event types will be ignored.
// Any errors returned from githubService will be logged as test failures and ignored.
func githubPushPullEventProcessor(
	t *testing.T, githubService *github_service.GitHubService,
) func(*github_test_utils.SmeeNotification) error {
	var pushPullAllowList = eventAllowList{
		"push":         true,
		"pull_request": true,
	}

	return githubEventProcessor(t, githubService, pushPullAllowList)
}

// githubEventProcessor returns a SmeeEventHandler function that checks if the event is a
// GitHub webhook event and calls the gitHubService to process the event.
// Only event types with the value 'true' in the allowList will be sent to the gitHubService.
// Any errors returned from githubService will be logged as test failures and ignored.
func githubEventProcessor(
	t *testing.T, githubService *github_service.GitHubService, allowList eventAllowList,
) func(*github_test_utils.SmeeNotification) error {
	return func(event *github_test_utils.SmeeNotification) error {
		eventType := event.Headers["x-github-event"]
		if eventType == "" {
			t.Log("ignoring Smee event that is not a GitHub event (no 'x-github-event' header')")
			return nil
		}
		signature256 := event.Headers["x-hub-signature-256"]
		if signature256 == "" {
			t.Log("event type header 'x-hub-signature-256' not found")
			return nil
		}
		t.Logf("Received GitHub Webhook notification of type %q", eventType)

		// Check the 'allow list' to see if the event should be sent to githubService
		allowed, found := allowList[eventType]
		if !allowed || !found {
			t.Logf("ignoring GitHub event of type '%s' that is not listed in the allowList", eventType)
			return nil
		}

		// This is a GitHub event - send it to our GitHub SCM service
		githubEvent := &github_service.WebhookEvent{
			EventType:    eventType,
			Signature256: signature256,
			Payload:      bytes.NewReader(event.Body),
		}
		err := githubService.HandleWebhookEvent(context.Background(), githubEvent)
		assert.NoError(t, err, "error while GitHub SCM processed Webhook event")

		return nil
	}
}
