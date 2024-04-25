package authorization_test

import (
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type UserInfo struct {
	LegalEntity *models.LegalEntity
	Identity    *models.Identity
}

func (u *UserInfo) LegalEntityID() models.LegalEntityID {
	return u.LegalEntity.ID
}

func (u *UserInfo) IdentityID() models.IdentityID {
	return u.Identity.ID
}

func (u *UserInfo) IdentityIDPtr() *models.IdentityID {
	return &u.Identity.ID
}

func (u *UserInfo) Name() models.ResourceName {
	return u.LegalEntity.Name
}

func TestAccessControl(t *testing.T) {
	ctx := context.Background()
	now := models.NewTime(time.Now())
	bbSystem := models.BuildBeaverSystem
	testsSystem := models.TestsSystem

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	// Create a company to perform a
	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "big-co", "BigCo Engineering Ltd", "spamscam@hotmail.com")
	testCompanyAdminGroup, err := app.GroupService.ReadByName(ctx, nil, testCompany.ID, models.AdminStandardGroup.Name)
	require.NoError(t, err)
	testCompanyReadOnlyUserGroup, err := app.GroupService.ReadByName(ctx, nil, testCompany.ID, models.ReadOnlyUserStandardGroup.Name)
	require.NoError(t, err)
	//testCompanyReadWriteUserGroup, err := app.GroupService.ReadByName(ctx, nil, testCompany.ID, models.UserStandardGroup.Name)
	//require.NoError(t, err)

	// Create some users
	var alice, bob, carol, dave UserInfo
	alice.LegalEntity, alice.Identity = server_test.CreatePersonLegalEntity(t, ctx, app, "alice", "Alice First", "alice@not-a-real-domain.com")
	bob.LegalEntity, bob.Identity = server_test.CreatePersonLegalEntity(t, ctx, app, "bob", "Bob Second", "bob@not-a-real-domain.com")
	carol.LegalEntity, carol.Identity = server_test.CreatePersonLegalEntity(t, ctx, app, "carol", "Carol Third", "carol@not-a-real-domain.com")
	dave.LegalEntity, dave.Identity = server_test.CreatePersonLegalEntity(t, ctx, app, "dave", "Dave Fourth", "dave@not-a-real-domain.com")

	// A couple of repos to test access control
	repo1 := server_test.CreateNamedRepo(t, ctx, app, "repo-1", testCompany.ID)
	//	repo2 := server_test.CreateNamedRepo(t, ctx, app, "repo-2", testCompany.ID)

	// Make Alice, Bob and Carol members of testCompany (but not poor old Dave)
	// TODO: Do this through LegalEntityService once we have suitable methods on that service
	err = app.LegalEntityMembershipStore.Create(ctx, nil, models.NewLegalEntityMembership(now, testCompany.ID, alice.LegalEntityID()))
	require.NoError(t, err)
	err = app.LegalEntityMembershipStore.Create(ctx, nil, models.NewLegalEntityMembership(now, testCompany.ID, bob.LegalEntityID()))
	require.NoError(t, err)
	err = app.LegalEntityMembershipStore.Create(ctx, nil, models.NewLegalEntityMembership(now, testCompany.ID, carol.LegalEntityID()))
	require.NoError(t, err)

	// No-one should have any access to the repo
	checkCanNotReadRepo(t, app, alice, repo1)
	checkCanNotReadRepo(t, app, bob, repo1)
	checkCanNotReadRepo(t, app, carol, repo1)
	checkCanNotReadRepo(t, app, dave, repo1)

	// TODO: Check they are members of the 'basic user' group once we have one

	// Alice the Admin
	app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
		testCompanyAdminGroup.ID, alice.IdentityID(), models.TestsSystem, testCompany.ID))
	checkCanReadRepo(t, app, alice, repo1)
	checkCanUpdateRepo(t, app, alice, repo1)
	checkCanNotReadRepo(t, app, bob, repo1)
	checkNrMemberships(t, app, alice, testCompanyAdminGroup, 1)

	// Bob the read-only user
	app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
		testCompanyReadOnlyUserGroup.ID, bob.IdentityID(), models.TestsSystem, testCompany.ID))
	checkCanReadRepo(t, app, bob, repo1)
	checkCanNotUpdateRepo(t, app, bob, repo1)

	// Make Alice also an admin from the BuildBeaver system, then remove again, then add back in
	app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
		testCompanyAdminGroup.ID, alice.IdentityID(), models.BuildBeaverSystem, testCompany.ID))
	checkNrMemberships(t, app, alice, testCompanyAdminGroup, 2)
	checkCanUpdateRepo(t, app, alice, repo1)
	app.GroupService.RemoveMembership(ctx, nil, testCompanyAdminGroup.ID, alice.IdentityID(), &bbSystem)
	checkNrMemberships(t, app, alice, testCompanyAdminGroup, 1)
	checkCanUpdateRepo(t, app, alice, repo1)
	app.GroupService.RemoveMembership(ctx, nil, testCompanyAdminGroup.ID, alice.IdentityID(), &testsSystem)
	checkNrMemberships(t, app, alice, testCompanyAdminGroup, 0)
	checkCanNotUpdateRepo(t, app, alice, repo1)
	app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
		testCompanyAdminGroup.ID, alice.IdentityID(), models.TestsSystem, testCompany.ID))
	checkCanReadRepo(t, app, alice, repo1)
	checkCanUpdateRepo(t, app, alice, repo1)
	checkNrMemberships(t, app, alice, testCompanyAdminGroup, 1)

	// Check Bob is still unchanged
	checkCanReadRepo(t, app, bob, repo1)
	checkCanNotUpdateRepo(t, app, bob, repo1)

	t.Run("CustomGroupsTest", testAccessControlCustomGroups(app, testCompany, alice, bob, carol, dave, repo1))
}

func testAccessControlCustomGroups(
	app *server_test.TestServer,
	testCompany *models.LegalEntity,
	alice, bob, carol, dave UserInfo,
	repo1 *models.Repo,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		now := models.NewTime(time.Now())
		specialSystem := models.SystemName("special-test-system")
		testsSystem := models.TestsSystem

		// Make custom access control groups for team-1 and team-2
		customGroup1, created, err := app.GroupService.FindOrCreateByName(ctx, nil, models.NewGroup(
			now, testCompany.ID, "team-1", "Custom group for team-1", false, nil))
		require.NoError(t, err)
		require.True(t, created, "New custom group should have been created")
		customGroup2, created, err := app.GroupService.FindOrCreateByName(ctx, nil, models.NewGroup(
			now, testCompany.ID, "team-2", "Custom group for team-2", false, nil))
		require.NoError(t, err)
		require.True(t, created, "New custom group should have been created")

		checkCanNotReadRepo(t, app, carol, repo1)
		checkNrMemberships(t, app, carol, customGroup1, 0)

		// Make carol a member of the custom group for team-1
		membership1, created, err := app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), models.TestsSystem, testCompany.ID))
		require.NoError(t, err)
		require.True(t, created)

		checkCanNotReadRepo(t, app, carol, repo1)
		checkNrMemberships(t, app, carol, customGroup1, 1)

		// Give read and update repo privileges to the group for team-1
		_, _, err = app.AuthorizationService.FindOrCreateGrant(ctx, nil, models.NewGroupGrant(
			now, testCompany.ID, customGroup1.ID, *models.RepoReadOperation, repo1.ID.ResourceID))
		_, _, err = app.AuthorizationService.FindOrCreateGrant(ctx, nil, models.NewGroupGrant(
			now, testCompany.ID, customGroup1.ID, *models.RepoUpdateOperation, repo1.ID.ResourceID))

		checkCanReadRepo(t, app, carol, repo1)
		checkCanUpdateRepo(t, app, carol, repo1)

		// Make Dave a member of custom group 2, and give group 2 read-only privileges to the repo
		_, created, err = app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup2.ID, dave.IdentityID(), models.TestsSystem, testCompany.ID))
		require.NoError(t, err)
		require.True(t, created)
		_, _, err = app.AuthorizationService.FindOrCreateGrant(ctx, nil, models.NewGroupGrant(
			now, testCompany.ID, customGroup2.ID, *models.RepoReadOperation, repo1.ID.ResourceID))
		checkCanReadRepo(t, app, dave, repo1)
		checkCanNotUpdateRepo(t, app, dave, repo1)

		// Repeat adding carol to group 1 for the same system name, should just find the existing membership
		membershipResult, created, err := app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), models.TestsSystem, testCompany.ID))
		require.NoError(t, err)
		require.False(t, created)
		require.Equal(t, membershipResult, membership1)

		checkCanReadRepo(t, app, carol, repo1)
		checkCanUpdateRepo(t, app, carol, repo1)
		checkNrMemberships(t, app, carol, customGroup1, 1)

		// Add a membership with a different system name, should create a new membership
		membership2, created, err := app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), specialSystem, testCompany.ID))
		require.NoError(t, err)
		require.True(t, created)

		checkCanReadRepo(t, app, carol, repo1)
		checkCanUpdateRepo(t, app, carol, repo1)
		checkNrMemberships(t, app, carol, customGroup1, 2)

		// Repeat adding a membership with a different system name, should add a second membership
		membershipResult, created, err = app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), specialSystem, testCompany.ID))
		require.NoError(t, err)
		require.False(t, created)
		require.Equal(t, membershipResult, membership2)
		checkNrMemberships(t, app, carol, customGroup1, 2)

		checkCanReadRepo(t, app, carol, repo1)
		checkCanUpdateRepo(t, app, carol, repo1)

		// Remove the 'special system' membership
		err = app.GroupService.RemoveMembership(ctx, nil, customGroup1.ID, carol.IdentityID(), &specialSystem)
		require.NoError(t, err)
		checkNrMemberships(t, app, carol, customGroup1, 1)
		// Repeating the remove should be a no-op
		err = app.GroupService.RemoveMembership(ctx, nil, customGroup1.ID, carol.IdentityID(), &specialSystem)
		checkNrMemberships(t, app, carol, customGroup1, 1)

		checkCanReadRepo(t, app, carol, repo1)
		checkCanUpdateRepo(t, app, carol, repo1)

		// Remove the other membership from source system 'tests'
		err = app.GroupService.RemoveMembership(ctx, nil, customGroup1.ID, carol.IdentityID(), &testsSystem)
		checkNrMemberships(t, app, carol, customGroup1, 0)
		checkCanNotReadRepo(t, app, carol, repo1)
		checkCanNotUpdateRepo(t, app, carol, repo1)
		// Repeating the remove should be a no-op
		err = app.GroupService.RemoveMembership(ctx, nil, customGroup1.ID, carol.IdentityID(), &testsSystem)
		checkNrMemberships(t, app, carol, customGroup1, 0)

		// Put Carol back in the groups, then remove Carol from the company and check she was removed from all groups
		_, _, err = app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), models.TestsSystem, testCompany.ID))
		require.NoError(t, err)
		_, _, err = app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			customGroup1.ID, carol.IdentityID(), specialSystem, testCompany.ID))
		require.NoError(t, err)
		checkNrMemberships(t, app, carol, customGroup1, 2)
		// TODO: Do this through LegalEntityService once we have suitable methods on that service
		err = app.LegalEntityMembershipStore.DeleteByMember(ctx, nil, testCompany.ID, carol.LegalEntityID())
		require.NoError(t, err)
		// TODO: Uncomment these checks once we implement removal of user from all groups when they are removed from a legal entity
		//checkNrMemberships(t, app, carol, customGroup, 0)
		//checkCanNotReadRepo(t, app, carol, repo1)
		//checkCanNotUpdateRepo(t, app, carol, repo1)
	}
}

func checkCanReadRepo(t *testing.T, app *server_test.TestServer, user UserInfo, repo *models.Repo) {
	checkRepoPermission(t, app, user, repo, models.RepoReadOperation, true)
}

func checkCanNotReadRepo(t *testing.T, app *server_test.TestServer, user UserInfo, repo *models.Repo) {
	checkRepoPermission(t, app, user, repo, models.RepoReadOperation, false)
}

func checkCanUpdateRepo(t *testing.T, app *server_test.TestServer, user UserInfo, repo *models.Repo) {
	checkRepoPermission(t, app, user, repo, models.RepoUpdateOperation, true)
}

func checkCanNotUpdateRepo(t *testing.T, app *server_test.TestServer, user UserInfo, repo *models.Repo) {
	checkRepoPermission(t, app, user, repo, models.RepoUpdateOperation, false)
}

func checkRepoPermission(
	t *testing.T,
	app *server_test.TestServer,
	user UserInfo,
	repo *models.Repo,
	operation *models.Operation,
	shouldAllow bool,
) {
	ctx := context.Background()

	hasAccess, err := app.AuthorizationService.IsAuthorized(ctx, user.IdentityID(), operation, repo.ID.ResourceID)
	require.NoError(t, err)
	if shouldAllow {
		require.True(t, hasAccess, "User %s should have %s access to %s", user.Name(), operation, repo.Name)
	} else {
		require.False(t, hasAccess, "User %s should NOT have %s access to %s", user.Name(), operation, repo.Name)
	}
}

// checkNrMemberships checks the number of memberships (for different source systems) a user has for a group
// matches the expected number.
func checkNrMemberships(t *testing.T, app *server_test.TestServer, user UserInfo, group *models.Group, expectedNrMemberships int) {
	ctx := context.Background()
	pagination := models.NewPagination(30, nil) // read enough to cover our test data

	memberships, _, err := app.GroupService.ListGroupMemberships(ctx, nil, &group.ID, user.IdentityIDPtr(), nil, pagination)
	require.NoError(t, err)
	require.Equal(t, expectedNrMemberships, len(memberships), "Expected %d membership records but found %d for user %s in group %s",
		expectedNrMemberships, len(memberships), user.Name(), group.Name)
}
