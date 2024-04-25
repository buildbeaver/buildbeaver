package commits_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestPullRequest(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	legalEntityA, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntityA.ID)

	// Create basic pull request
	pr := newTestPullRequest(legalEntityA.ID, repo.ID)
	err = app.PullRequestStore.Create(context.Background(), nil, pr)
	assert.NoError(t, err)

	// Read back the PR
	t.Run("Read", testPullRequestRead(app.PullRequestStore, pr))
}

func newTestPullRequest(userId models.LegalEntityID, repoId models.RepoID) *models.PullRequest {
	var now = time.Now().UTC()
	var githubPullRequestExternalID = models.NewExternalResourceID("github", "123456789")

	return &models.PullRequest{
		ID:         models.NewPullRequestID(),
		CreatedAt:  models.NewTime(now),
		UpdatedAt:  models.NewTime(now),
		Title:      "This is a test Pull Request",
		State:      "open",
		BaseRef:    "refs/heads/master",
		HeadRef:    "refs/heads/pr-branch",
		ExternalID: &githubPullRequestExternalID,
		UserID:     userId,
		RepoID:     repoId,
	}
}

func testPullRequestRead(store store.PullRequestStore, referencePullRequest *models.PullRequest) func(t *testing.T) {
	return func(t *testing.T) {
		pr, err := store.Read(context.Background(), nil, referencePullRequest.ID)
		assert.NoError(t, err, "Error reading Pull Request")

		assert.Equal(t, pr.ID, referencePullRequest.ID)
		assert.Equal(t, pr.CreatedAt, referencePullRequest.CreatedAt)
		assert.Equal(t, pr.Title, referencePullRequest.Title)
		assert.Equal(t, pr.State, referencePullRequest.State)
		assert.Equal(t, pr.ExternalID, referencePullRequest.ExternalID)
		assert.Equal(t, pr.RepoID, referencePullRequest.RepoID)
		assert.Equal(t, pr.UserID, referencePullRequest.UserID)
		assert.Equal(t, pr.BaseRef, referencePullRequest.BaseRef)
		assert.Equal(t, pr.HeadRef, referencePullRequest.HeadRef)
	}
}
