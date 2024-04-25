package sync_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/server/services/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	github_service "github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github/github_test_utils"
)

func TestSyncWithGitHubIntegration(t *testing.T) {
	t.Skip("Skipping sync with GitHub integration test")

	if testing.Short() {
		t.Skip("Skipping sync with GitHub integration test")
	}

	ctx := context.Background()

	// Seed math/rand numbers
	rand.Seed(time.Now().UnixNano())

	// Set up test server app
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	// Set up GitHub authentication and GitHub client for direct interactions
	scmAuth := github_test_utils.MakeGitHubAuth(t)
	githubClient, err := github_test_utils.MakeGitHubTestClient(scmAuth)
	require.NoError(t, err, "Error creating GitHub client")

	// Sync authenticated user to create Legal Entity and Identity
	_, err = app.SyncService.SyncAuthenticatedUser(context.Background(), scmAuth)
	require.NoError(t, err)

	// Check that the user is populated in the database after sync
	userLegalEntity, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, github_test_utils.GitHubTestAccountExternalID)
	require.NoError(t, err)
	assert.Equal(t, userLegalEntity.Name, github_test_utils.GitHubTestAccountLegalEntityName, "Test GitHub user name not synced correctly from GitHub")
	assert.Equal(t, userLegalEntity.LegalName, github_test_utils.GitHubTestAccountLegalName, "Test GitHub user name not synced correctly from GitHub")
	userIdentity, err := app.LegalEntityService.ReadIdentity(ctx, nil, userLegalEntity.ID)
	require.NoError(t, err)

	// Run a user-based sync to establish baseline set of repos
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	require.NoError(t, err)

	// Check that organization names and emails are correctly populated in the database after sync
	legalEntity1, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, github_test_utils.GitHubTestAccountOrg1ExternalID)
	require.NoError(t, err)
	assert.Equal(t, legalEntity1.LegalName, github_test_utils.GitHubTestAccountOrg1LegalName, "Company 1 name not synced correctly from GitHub")
	assert.Equal(t, legalEntity1.EmailAddress, github_test_utils.GitHubTestAccountOrg1EMail, "Company 1 email not synced correctly from GitHub")
	legalEntity2, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, github_test_utils.GitHubTestAccountOrg2ExternalID)
	require.NoError(t, err)
	assert.Equal(t, legalEntity2.LegalName, github_test_utils.GitHubTestAccountOrg2LegalName, "Company 2 name not synced correctly from GitHub")
	assert.Equal(t, legalEntity2.EmailAddress, github_test_utils.GitHubTestAccountOrg2EMail, "Company 2 email not synced correctly from GitHub")

	// Check that the GitHub integration test user has the correct roles in Org 1 (admin but not user)
	legalEntity1AdminGroup, err := app.GroupStore.ReadByName(ctx, nil, legalEntity1.ID, models.AdminStandardGroup.Name)
	require.NoError(t, err)
	adminMembership, err := app.GroupMembershipStore.ReadByMember(ctx, nil, legalEntity1AdminGroup.ID, userIdentity.ID, github_service.GitHubSCMName)
	require.NoError(t, err, "GitHub integration test user should be a member of 'admin' group")
	require.Equal(t, userIdentity.ID, adminMembership.MemberIdentityID, "GitHub integration test user should be a member of 'admin' group")

	legalEntity1UserGroup, err := app.GroupStore.ReadByName(ctx, nil, legalEntity1.ID, models.ReadOnlyUserStandardGroup.Name)
	require.NoError(t, err)
	_, err = app.GroupMembershipStore.ReadByMember(ctx, nil, legalEntity1UserGroup.ID, userIdentity.ID, github_service.GitHubSCMName)
	require.True(t, gerror.IsNotFound(err), "GitHub integration test user should not be a member of 'user' group")

	// Set up new test repo in GitHub for use in this test (and delete after)
	_, newRepo1ExternalID, teardown1, err := github_test_utils.SetupTestRepo(t, githubClient, "")
	defer teardown1()
	require.NoError(t, err)

	// Check that the new repo is NOT in our database
	_, err = app.RepoStore.ReadByExternalID(ctx, nil, newRepo1ExternalID)
	assert.Error(t, err)

	// Call Sync again, which should pick up the new repo
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	require.NoError(t, err)

	// See if the new repo is in our database
	repo1FromStore, err := app.RepoStore.ReadByExternalID(ctx, nil, newRepo1ExternalID)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, *repo1FromStore.ExternalID, newRepo1ExternalID, "Repo 1 External ID doesn't match")
	}

	// Set up two new test repo in GitHub in our first test organization
	_, newRepo2ExternalID, teardown2, err := github_test_utils.SetupTestRepo(t, githubClient, github_test_utils.GitHubTestAccountOrg1Name)
	defer teardown2()
	require.NoError(t, err)
	_, newRepo3ExternalID, teardown3, err := github_test_utils.SetupTestRepo(t, githubClient, github_test_utils.GitHubTestAccountOrg1Name)
	defer teardown3()
	require.NoError(t, err)

	// Call Sync again, which should pick up the new repos in the correct organization
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	assert.NoError(t, err)

	// Check the new repos are in our database and are under the correct legal entity
	repo2FromStore, err := app.RepoStore.ReadByExternalID(ctx, nil, newRepo2ExternalID)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, *repo2FromStore.ExternalID, newRepo2ExternalID, "Repo 2 External ID doesn't match")
		legalEntityForRepo2, err := app.LegalEntityStore.Read(ctx, nil, repo2FromStore.LegalEntityID)
		require.NoError(t, err)
		assert.Equal(t, legalEntityForRepo2.Name, github_test_utils.GitHubTestAccountOrg1LegalEntityName)
	}
	repo3FromStore, err := app.RepoStore.ReadByExternalID(ctx, nil, newRepo3ExternalID)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, *repo3FromStore.ExternalID, newRepo3ExternalID, "Repo 3 External ID doesn't match")
		legalEntityForRepo3, err := app.LegalEntityStore.Read(ctx, nil, repo3FromStore.LegalEntityID)
		require.NoError(t, err)
		assert.Equal(t, legalEntityForRepo3.Name, github_test_utils.GitHubTestAccountOrg1LegalEntityName)
	}
}

// TestSyncWithGitHubNotAdmin performs a sync test using the second GitHub integration test user, completely independently
// of the test using the first user. Checks group membership for non-admin users.
func TestSyncWithGitHubNotAdminIntegration(t *testing.T) {
	t.Skip("Skipping sync with GitHub (not admin) integration test")

	if testing.Short() {
		t.Skip("Skipping sync with GitHub (not admin) integration test")
	}

	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	// Set up test server app
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	// Set up GitHub authentication and GitHub client for direct interactions as test user 2
	scmAuth := github_test_utils.MakeGitHubAuth2(t)

	// Sync authenticated user to create Legal Entity and Identity
	_, err = app.SyncService.SyncAuthenticatedUser(context.Background(), scmAuth)
	require.NoError(t, err)

	// Call Sync to establish baseline set of legal entities and repos
	err = app.SyncService.GlobalSync(ctx, github_service.GitHubSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	require.NoError(t, err)

	// Check that the user is populated in the database after sync
	userLegalEntity, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, github_test_utils.GitHubTestAccount2ExternalID)
	require.NoError(t, err)
	assert.Equal(t, userLegalEntity.Name, github_test_utils.GitHubTestAccount2LegalEntityName, "Test GitHub user 2 name not synced correctly from GitHub")
	assert.Equal(t, userLegalEntity.LegalName, github_test_utils.GitHubTestAccount2LegalName, "Test GitHub user 2 name not synced correctly from GitHub")
	userIdentity, err := app.LegalEntityService.ReadIdentity(ctx, nil, userLegalEntity.ID)
	require.NoError(t, err)

	// Check that organization names and emails are correctly populated in the database after sync
	legalEntity1, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, github_test_utils.GitHubTestAccountOrg1ExternalID)
	require.NoError(t, err, "Company 1 not present in database after sync")
	assert.Equal(t, legalEntity1.LegalName, github_test_utils.GitHubTestAccountOrg1LegalName, "Company 1 name not synced correctly from GitHub")

	// Check that GitHub integration test user 2 has the correct roles in Org 1 (user but not admin)
	legalEntity1AdminGroup, err := app.GroupStore.ReadByName(ctx, nil, legalEntity1.ID, models.AdminStandardGroup.Name)
	require.NoError(t, err)
	_, err = app.GroupMembershipStore.ReadByMember(ctx, nil, legalEntity1AdminGroup.ID, userIdentity.ID, github_service.GitHubSCMName)
	require.True(t, gerror.IsNotFound(err), "GitHub integration test user 2 should not be a member of 'admin' group")

	legalEntity1UserGroup, err := app.GroupStore.ReadByName(ctx, nil, legalEntity1.ID, models.ReadOnlyUserStandardGroup.Name)
	require.NoError(t, err)
	userMembership, err := app.GroupMembershipStore.ReadByMember(ctx, nil, legalEntity1UserGroup.ID, userIdentity.ID, github_service.GitHubSCMName)
	require.NoError(t, err, "GitHub integration test user 2 should be a member of 'user' group")
	require.Equal(t, userIdentity.ID, userMembership.MemberIdentityID, "GitHub integration test user 2 should be a member of 'user' group")
}
