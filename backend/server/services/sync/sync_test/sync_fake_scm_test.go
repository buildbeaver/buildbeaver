package sync_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/server/services/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services/scm/fake_scm"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const (
	testUserName1    = "test-user-1"
	testUserName2    = "test-user-2"
	testCompanyName1 = "test-company-1"
	testRepoName1    = "test-repo-1"
	testRepoName2    = "test-repo-2"
	testRepoName3    = "test-repo-3"
	testRepoName4    = "test-repo-4"
	testRepoName5    = "test-repo-5"
	testTeamName1    = "test-team-1"
	testTeamName2    = "test-team-2"
)

// userDetails is a convenient way to group together all the information we know about a user
type userDetails struct {
	name        string
	scmID       fake_scm.UserID
	externalID  models.ExternalResourceID
	auth        models.SCMAuth
	identity    *models.Identity
	legalEntity *models.LegalEntity
}

// companyDetails is a convenient way to group together all the information we know about a company
type companyDetails struct {
	scmID       fake_scm.CompanyID
	externalID  models.ExternalResourceID
	legalEntity *models.LegalEntity
}

func TestGlobalSyncWithFakeSCM(t *testing.T) {
	// Setup
	rand.Seed(time.Now().UnixNano())
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	scmInterface, err := app.SCMRegistry.Get(fake_scm.FakeSCMName)
	require.NoError(t, err)
	fakeSCM := scmInterface.(*fake_scm.FakeSCMService)

	// Set up users and auth in the fake scm
	user1 := &userDetails{name: testUserName1}
	user1.scmID, user1.externalID = fakeSCM.CreateUser(user1.name, true)
	user1.auth, err = fakeSCM.CreateAuthForUser(user1.scmID)
	require.NoError(t, err)
	user2 := &userDetails{name: testUserName2}
	user2.scmID, user2.externalID = fakeSCM.CreateUser(user2.name, false) // not a BuildBeaver user

	// Sync authenticated user 1 to create Legal Entity and Identity, but don't sync user 2
	user1.identity = syncAuthenticatedUser(t, app, user1.auth)
	user1.legalEntity, _ = checkUserInDatabase(t, app, user1.externalID, testUserName1)
	checkUserNotInDatabase(t, app, user2.externalID)

	// Perform an initial global sync to establish baseline
	globalSyncWithFakeSCM(t, app)

	checkAccessibleRepoCount(t, app, user1.identity, user1.legalEntity, 0)
	// User 2 is not using BuildBeaver so should not have been synced to the database
	checkUserNotInDatabase(t, app, user2.externalID)

	t.Run("Sync user", testGlobalSyncUserWithFakeSCM(app, fakeSCM, user1))
	t.Run("Sync company", testGlobalSyncCompanyWithFakeSCM(app, fakeSCM, user1, user2))

	// TODO: Test multiple SCMs both with legal entities and repos, check they don't interfere with each other
}

// testGlobalSyncUserWithFakeSCM tests syncing repos for a user. This is run on against existing database and Fake SCM.
func testGlobalSyncUserWithFakeSCM(
	app *server_test.TestServer,
	fakeSCM *fake_scm.FakeSCMService,
	user1 *userDetails,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		// Add two repos on the fake SCM, owned directly by user 1
		repo1ID, repo1ExternalID, err := fakeSCM.CreateRepoForUser(user1.scmID, testRepoName1)
		require.NoError(t, err)
		repo2ID, repo2ExternalID, err := fakeSCM.CreateRepoForUser(user1.scmID, testRepoName2)
		require.NoError(t, err)

		// Run a sync, then check that the two repos are added to the database
		checkRepoNotInDatabase(t, app, repo1ExternalID)
		checkRepoNotInDatabase(t, app, repo2ExternalID)
		globalSyncWithFakeSCM(t, app)
		checkRepoInDatabase(t, app, repo1ExternalID, testRepoName1)
		checkRepoInDatabase(t, app, repo2ExternalID, testRepoName2)
		checkAccessibleRepoCount(t, app, user1.identity, user1.legalEntity, 2)

		// Enable and disable repos multiple times
		repo1 := checkRepoInDatabase(t, app, repo1ExternalID, testRepoName1)
		_, err = app.RepoService.UpdateRepoEnabled(ctx, repo1.ID, dto.UpdateRepoEnabled{Enabled: true})
		require.NoError(t, err)
		_, err = app.RepoService.UpdateRepoEnabled(ctx, repo1.ID, dto.UpdateRepoEnabled{Enabled: false})
		require.NoError(t, err)
		_, err = app.RepoService.UpdateRepoEnabled(ctx, repo1.ID, dto.UpdateRepoEnabled{Enabled: true})
		require.NoError(t, err)
		_, err = app.RepoService.UpdateRepoEnabled(ctx, repo1.ID, dto.UpdateRepoEnabled{Enabled: false})
		require.NoError(t, err)
		_, err = app.RepoService.UpdateRepoEnabled(ctx, repo1.ID, dto.UpdateRepoEnabled{Enabled: true})
		require.NoError(t, err)

		// Remove a repo, check it is gone from the database
		fakeSCM.DeleteRepo(repo2ID)
		globalSyncWithFakeSCM(t, app)
		checkRepoInDatabase(t, app, repo1ExternalID, testRepoName1)
		checkRepoNotInDatabase(t, app, repo2ExternalID)
		checkAccessibleRepoCount(t, app, user1.identity, user1.legalEntity, 1)

		// Make another repo 2 with the same name, check we can add it to the database
		repo2aID, repo2aExternalID, err := fakeSCM.CreateRepoForUser(user1.scmID, testRepoName2)
		t.Logf("Created another repo with name %q, ID %d, externalID %v", testRepoName2, repo2aID, repo2aExternalID)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		checkRepoInDatabase(t, app, repo1ExternalID, testRepoName1)
		checkRepoInDatabase(t, app, repo2aExternalID, testRepoName2)
		checkAccessibleRepoCount(t, app, user1.identity, user1.legalEntity, 2)

		// Replace repo 1 with another one with the same name, then do a sync and check that it copes
		fakeSCM.DeleteRepo(repo1ID)
		repo1aID, repo1aExternalID, err := fakeSCM.CreateRepoForUser(user1.scmID, testRepoName1)
		t.Logf("Created another repo with name %q, ID %d, externalID %v", testRepoName1, repo1aID, repo1aExternalID)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		checkRepoInDatabase(t, app, repo1aExternalID, testRepoName1)
		checkRepoNotInDatabase(t, app, repo1ExternalID)
		checkAccessibleRepoCount(t, app, user1.identity, user1.legalEntity, 2)
	}
}

// testGlobalSyncCompanyWithFakeSCM tests creating and syncing a company. This can be run on against existing database and
// Fake SCM which already has repos populated for the user, in order to test for when a user has both personal
// and company repos.
func testGlobalSyncCompanyWithFakeSCM(
	app *server_test.TestServer,
	fakeSCM *fake_scm.FakeSCMService,
	user1 *userDetails,
	user2 *userDetails,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		// Create a company, with its own repo
		company1 := &companyDetails{}
		company1.scmID, company1.externalID = fakeSCM.CreateCompany(testCompanyName1)
		_, repo3ExternalID, err := fakeSCM.CreateRepoForCompany(company1.scmID, testRepoName3)
		require.NoError(t, err)

		// Sync before the user joins the company. The company and its new repo should be in the database,
		// but the user should not have access rights.
		globalSyncWithFakeSCM(t, app)
		company1.legalEntity, err = app.LegalEntityStore.ReadByExternalID(ctx, nil, company1.externalID)
		require.NoError(t, err)
		checkRepoInDatabase(t, app, repo3ExternalID, testRepoName3)
		checkUserNotInCompany(t, app, user1.legalEntity, company1.legalEntity)
		checkUserNotInGroup(t, app, user1.identity, company1.legalEntity, models.AdminStandardGroup.Name)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 0)

		// Make the user a member of the company, with admin role
		err = fakeSCM.AddUserToCompany(company1.scmID, user1.scmID)
		require.NoError(t, err)
		err = fakeSCM.AddUserToGroup(company1.scmID, models.AdminStandardGroup.Name, user1.scmID)
		require.NoError(t, err)

		// Sync the user again and the user should now have access to the company repo.
		globalSyncWithFakeSCM(t, app)
		checkUserInCompany(t, app, user1.legalEntity, company1.legalEntity)
		checkUserInGroup(t, app, user1.identity, company1.legalEntity, models.AdminStandardGroup.Name)
		checkRepoInDatabase(t, app, repo3ExternalID, testRepoName3)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 1)

		// Remove user from the admin group for company 1; they should no longer have access to the company's repo,
		// but it should still be in the DB.
		err = fakeSCM.RemoveUserFromGroup(company1.scmID, models.AdminStandardGroup.Name, user1.scmID)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		checkUserInCompany(t, app, user1.legalEntity, company1.legalEntity)
		checkUserNotInGroup(t, app, user1.identity, company1.legalEntity, models.AdminStandardGroup.Name)
		checkRepoInDatabase(t, app, repo3ExternalID, testRepoName3)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 0)

		// Put user back in the company admin group
		err = fakeSCM.AddUserToGroup(company1.scmID, models.AdminStandardGroup.Name, user1.scmID)
		globalSyncWithFakeSCM(t, app)
		checkUserInCompany(t, app, user1.legalEntity, company1.legalEntity)
		checkUserInGroup(t, app, user1.identity, company1.legalEntity, models.AdminStandardGroup.Name)

		// Remove the user entirely as a member of the company
		err = fakeSCM.RemoveUserFromCompany(company1.scmID, user1.scmID)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		// The user should no longer be a member of the company, and should be removed from the admin group again
		checkUserNotInCompany(t, app, user1.legalEntity, company1.legalEntity)
		checkUserNotInGroup(t, app, user1.identity, company1.legalEntity, models.AdminStandardGroup.Name)
		checkRepoInDatabase(t, app, repo3ExternalID, testRepoName3)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 0)

		// Create a couple more repos for the company so we can test manipulating their permissions
		repo4ID, repo4ExternalID, err := fakeSCM.CreateRepoForCompany(company1.scmID, testRepoName4)
		require.NoError(t, err)
		repo5ID, repo5ExternalID, err := fakeSCM.CreateRepoForCompany(company1.scmID, testRepoName5)
		require.NoError(t, err)

		// Create custom groups within the company, with various members and permissions
		_, err = fakeSCM.CreateCustomGroupForCompany(company1.scmID, testTeamName1)
		require.NoError(t, err)
		_, err = fakeSCM.CreateCustomGroupForCompany(company1.scmID, testTeamName2)
		require.NoError(t, err)
		fakeSCM.AddUserToGroup(company1.scmID, testTeamName1, user1.scmID)
		require.NoError(t, err)
		// TODO: Uncomment the following once we have fixed the problem of resources showing up multiple times
		// TODO: in a list when there are multiple ways of being given access
		fakeSCM.AddUserToGroup(company1.scmID, testTeamName2, user1.scmID)
		require.NoError(t, err)

		// Add user 2 to a group; this should cause user 2 to be created in the database
		fakeSCM.AddUserToGroup(company1.scmID, testTeamName2, user2.scmID)
		require.NoError(t, err)
		fakeSCM.SetGroupPermissionForRepo(company1.scmID, testTeamName1, repo4ID, true, true, true)
		fakeSCM.SetGroupPermissionForRepo(company1.scmID, testTeamName1, repo5ID, true, false, false)
		fakeSCM.SetGroupPermissionForRepo(company1.scmID, testTeamName2, repo4ID, true, false, false)

		// Sync and test results
		checkUserNotInDatabase(t, app, user2.externalID)
		globalSyncWithFakeSCM(t, app)
		// User 2 should now be in the database even though they don't use BuildBeaver and have never authenticated,
		// because they are now a member of a group within company1
		user2.legalEntity, user2.identity = checkUserInDatabase(t, app, user2.externalID, testUserName2)

		testRepo4 := checkRepoInDatabase(t, app, repo4ExternalID, testRepoName4)
		testRepo5 := checkRepoInDatabase(t, app, repo5ExternalID, testRepoName5)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 2)
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 1)
		checkUserRepoAccess(t, app, user1, testRepo4.ID, true, true)
		checkUserRepoAccess(t, app, user1, testRepo5.ID, true, false)
		checkUserRepoAccess(t, app, user2, testRepo4.ID, true, false)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)

		// Add a new permission for repo5 for team 2, sync, check access was granted to repo5
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 1)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)
		fakeSCM.SetGroupPermissionForRepo(company1.scmID, testTeamName2, repo5ID, true, true, false)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 2)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, true, true)
		// Remove the extra permission for team 2 again, check access to repo5 has been revoked
		fakeSCM.SetGroupPermissionForRepo(company1.scmID, testTeamName2, repo5ID, false, false, false)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 1)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)

		// Add user 2 to another team 1, sync, remove user 2 again, sync checking results
		fakeSCM.AddUserToGroup(company1.scmID, testTeamName1, user2.scmID)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 2)
		checkUserRepoAccess(t, app, user2, testRepo4.ID, true, true)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, true, false)

		fakeSCM.RemoveUserFromGroup(company1.scmID, testTeamName1, user2.scmID)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user2.identity, company1.legalEntity, 1)
		checkUserRepoAccess(t, app, user2, testRepo4.ID, true, false)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)

		// Delete the group for team 1 altogether, check that user 1 no longer gets access to repos from team 1,
		// and that user 1 only has access to repos from from team 2
		team1Group := checkGroupInDatabase(t, app, company1.legalEntity, testTeamName1)
		require.NotZero(t, countGrantsForGroup(t, app, team1Group.ID), "expected to find some grants for Team1")
		err = fakeSCM.DeleteCustomGroupForCompany(company1.scmID, testTeamName1)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 1)
		checkUserRepoAccess(t, app, user2, testRepo4.ID, true, false)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)

		// We can still count grants for a group even if the group is deleted, but there should be no grants left
		require.Zero(t, countGrantsForGroup(t, app, team1Group.ID), "expected no remaining grants for Team1")

		// Delete the group for team 2 altogether; there should then be no teams and no access to repo5 and repo6
		team2Group := checkGroupInDatabase(t, app, company1.legalEntity, testTeamName2)
		require.NotZero(t, countGrantsForGroup(t, app, team2Group.ID), "expected to find some grants for Team2")
		err = fakeSCM.DeleteCustomGroupForCompany(company1.scmID, testTeamName2)
		require.NoError(t, err)
		globalSyncWithFakeSCM(t, app)
		checkAccessibleRepoCount(t, app, user1.identity, company1.legalEntity, 0)
		checkUserRepoAccess(t, app, user1, testRepo4.ID, false, false)
		checkUserRepoAccess(t, app, user1, testRepo5.ID, false, false)
		checkUserRepoAccess(t, app, user2, testRepo4.ID, false, false)
		checkUserRepoAccess(t, app, user2, testRepo5.ID, false, false)
		require.Zero(t, countGrantsForGroup(t, app, team2Group.ID), "expected no remaining grants for Team2")
	}
}

var globalSyncCount = 0

func syncAuthenticatedUser(t *testing.T, app *server_test.TestServer, userAuth models.SCMAuth) *models.Identity {
	identity, err := app.SyncService.SyncAuthenticatedUser(context.Background(), userAuth)
	require.NoError(t, err)
	require.NotNil(t, identity)
	return identity
}

func globalSyncWithFakeSCM(t *testing.T, app *server_test.TestServer) {
	err := app.SyncService.GlobalSync(context.Background(), fake_scm.FakeSCMName, 0, sync.DefaultPerLegalEntityTimeout)
	globalSyncCount++
	require.NoError(t, err)
}

func checkUserInDatabase(t *testing.T, app *server_test.TestServer, userExternalID models.ExternalResourceID, userName string) (*models.LegalEntity, *models.Identity) {
	ctx := context.Background()
	userLegalEntity, err := app.LegalEntityStore.ReadByExternalID(ctx, nil, userExternalID)
	require.NoError(t, err)
	assert.Equal(t, userLegalEntity.Name, models.ResourceName(userName), "User %q name not synced correctly from Fake SCM", userName)
	userIdentity, err := app.LegalEntityService.ReadIdentity(ctx, nil, userLegalEntity.ID)
	require.NoError(t, err)
	return userLegalEntity, userIdentity
}

func checkUserNotInDatabase(t *testing.T, app *server_test.TestServer, userExternalID models.ExternalResourceID) {
	_, err := app.LegalEntityStore.ReadByExternalID(context.Background(), nil, userExternalID)
	require.Error(t, err, "User with external ID '%s' should not be in database", userExternalID)
}

// checkAccessibleRepoCount performs a search on the repo store to check the number of repos that are accessible to
// the specified user identity.
// If repoOwner is not nil then only repos owned by the specified legal entity will be included.
func checkAccessibleRepoCount(
	t *testing.T,
	app *server_test.TestServer,
	user *models.Identity,
	repoOwner *models.LegalEntity,
	expectedNrRepos int,
) {
	queryBuilder := search.NewRepoQueryBuilder()
	if repoOwner != nil {
		queryBuilder = queryBuilder.WhereLegalEntityID(search.Equal, repoOwner.ID)
	}
	repoQuery := queryBuilder.Compile()
	repoQuery.Limit = 100 // enough to capture all our test data, so we don't need pagination

	reposFound, _, err := app.RepoService.Search(context.Background(), nil, user.ID, repoQuery)
	require.NoError(t, err)
	for i, repo := range reposFound {
		t.Logf("Repo %d of %d: Name %s, ID %s", i+1, len(reposFound), repo.Name, repo.ID)
	}
	require.Equal(t, expectedNrRepos, len(reposFound), "Did not find the expected number of accessible repos for identity %s", user.ID)
}

// checkUserRepoAccess checks that the specified user has the expected access to a particular repo.
// expectRead and expectWrite are true if the user should be able to read and write to the repo respectively.
func checkUserRepoAccess(
	t *testing.T,
	app *server_test.TestServer,
	user *userDetails,
	repoID models.RepoID,
	expectRead bool,
	expectWrite bool,
) {
	ctx := context.Background()
	canRead, err := app.AuthorizationService.IsAuthorized(ctx, user.identity.ID, models.RepoReadOperation, repoID.ResourceID)
	require.NoError(t, err)
	if expectRead {
		require.True(t, canRead, "User '%s' should be able to read repo %s (user %s)", user.name, repoID, user.identity.ID)
	} else {
		require.False(t, canRead, "User '%s' should NOT be able to read repo %s (user %s)", user.name, repoID, user.identity.ID)
	}
	require.Equal(t, expectRead, canRead)
	canWrite, err := app.AuthorizationService.IsAuthorized(ctx, user.identity.ID, models.RepoUpdateOperation, repoID.ResourceID)
	require.NoError(t, err)
	if expectWrite {
		require.True(t, canWrite, "User '%s' should be able to write to repo %s (user %s)", user.name, repoID, user.identity.ID)
	} else {
		require.False(t, canWrite, "User '%s' should NOT be able to write to repo %s (user %s)", user.name, repoID, user.identity.ID)
	}
}

func checkRepoInDatabase(
	t *testing.T,
	app *server_test.TestServer,
	repoExternalID models.ExternalResourceID,
	repoName string,
) *models.Repo {
	repo, err := app.RepoStore.ReadByExternalID(context.Background(), nil, repoExternalID)
	assert.NoError(t, err, "Could not find repo %q in database", repoName)
	if err == nil {
		require.Equal(t, repo.Name, models.ResourceName(repoName))
	}
	return repo
}

func checkRepoNotInDatabase(t *testing.T, app *server_test.TestServer, repoExternalID models.ExternalResourceID) {
	_, err := app.RepoStore.ReadByExternalID(context.Background(), nil, repoExternalID)
	assert.Error(t, err)
}

func checkUserInCompany(t *testing.T, app *server_test.TestServer, user *models.LegalEntity, company *models.LegalEntity) {
	require.True(t, isUserInCompany(t, app, user, company), "User %q should be in company %q", user.Name, company.Name)
}

func checkUserNotInCompany(t *testing.T, app *server_test.TestServer, user *models.LegalEntity, company *models.LegalEntity) {
	require.False(t, isUserInCompany(t, app, user, company), "User %q should NOT be in company %q", user.Name, company.Name)
}

func isUserInCompany(t *testing.T, app *server_test.TestServer, user *models.LegalEntity, company *models.LegalEntity) bool {
	pagination := models.Pagination{Limit: 100} // enough to cover all our test data without paging
	parentLegalEntities, _, err := app.LegalEntityService.ListParentLegalEntities(context.Background(), nil, user.ID, pagination)
	require.NoError(t, err)
	t.Logf("Found %d parent legal entities for user", len(parentLegalEntities))

	for _, parent := range parentLegalEntities {
		if parent.ID.Equal(company.ID.ResourceID) {
			return true
		}
	}
	return false
}

func checkUserInGroup(
	t *testing.T,
	app *server_test.TestServer,
	userIdentity *models.Identity,
	company *models.LegalEntity,
	groupName models.ResourceName,
) {
	require.True(t, isUserInGroup(t, app, userIdentity, company, groupName),
		"User identity %q should be in group %q for company %q", userIdentity.ID, groupName, company.Name)
}

func checkUserNotInGroup(
	t *testing.T,
	app *server_test.TestServer,
	userIdentity *models.Identity,
	company *models.LegalEntity,
	groupName models.ResourceName,
) {
	require.False(t, isUserInGroup(t, app, userIdentity, company, groupName),
		"User identity %q should NOT be in group %q for company %q", userIdentity.ID, groupName, company.Name)
}

func isUserInGroup(
	t *testing.T,
	app *server_test.TestServer,
	userIdentity *models.Identity,
	company *models.LegalEntity,
	groupName models.ResourceName,
) bool {
	pagination := models.Pagination{Limit: 100} // enough to cover all our test data without paging
	groupMemberships, _, err := app.GroupService.ListGroups(context.Background(), nil, &company.ID, &userIdentity.ID, pagination)
	require.NoError(t, err)
	t.Logf("Found %d group memberships for user in company %q", len(groupMemberships), company.Name)

	for _, nextGroup := range groupMemberships {
		if nextGroup.Name == groupName {
			return true
		}
	}
	return false
}

func checkGroupInDatabase(t *testing.T, app *server_test.TestServer, company *models.LegalEntity, groupName models.ResourceName) *models.Group {
	group, err := app.GroupService.ReadByName(context.Background(), nil, company.ID, groupName)
	require.NoError(t, err)
	return group
}

// countGrantsForGroup returns the number of grants in the database for a specific group, up to a maximum
// of 1 page of grants.
func countGrantsForGroup(t *testing.T, app *server_test.TestServer, groupID models.GroupID) int {
	ctx := context.Background()
	grantCount := 0
	err := app.DB.WithTx(ctx, nil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			grants, cursor, err := app.AuthorizationService.ListGrantsForGroup(ctx, tx, groupID, pagination)
			if err != nil {
				return err
			}
			grantCount += len(grants)
			if cursor != nil && cursor.Next != nil {
				pagination.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		return nil
	})
	require.NoError(t, err)
	return grantCount
}
