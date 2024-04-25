package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/certificates/certificates_test_utils"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestAccessControlReadSelf(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.CoreAPIServer.Start()
	defer app.CoreAPIServer.Stop(ctx)

	// Make a company,
	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "bigco", "BigCo Limited", "admin@bigco.not.real")

	// Make standard groups for the company that we can use for testing
	readOnlyUserGroup, err := app.GroupService.FindOrCreateStandardGroup(ctx, nil, testCompany, models.ReadOnlyUserStandardGroup)
	require.NoError(t, err)
	readWriteUserGroup, err := app.GroupService.FindOrCreateStandardGroup(ctx, nil, testCompany, models.UserStandardGroup)
	require.NoError(t, err)
	adminGroup, err := app.GroupService.FindOrCreateStandardGroup(ctx, nil, testCompany, models.AdminStandardGroup)
	require.NoError(t, err)

	// Make a repo
	repo := server_test.CreateRepo(t, ctx, app, testCompany.ID)

	// Create a client certificate to use when registering build runners
	certPEM, _, _, err := certificates_test_utils.CreateTestClientCertificate(t)
	require.NoError(t, err)

	// Test access control rights for a new user in the 'admin' group
	t.Run("Alice-Admin", testAccessControlForRole(app, app.CoreAPIServer.GetServerURL(), testCompany.ID, repo.ID, certPEM, "Alice", adminGroup, true))

	// Test access control rights for a new user in the 'read-only user' group
	t.Run("Bob-User", testAccessControlForRole(app, app.CoreAPIServer.GetServerURL(), testCompany.ID, repo.ID, certPEM, "Bob", readOnlyUserGroup, false))

	// Test access control rights for a new user in the 'read/write user' group
	t.Run("Carol-User", testAccessControlForRole(app, app.CoreAPIServer.GetServerURL(), testCompany.ID, repo.ID, certPEM, "Carol", readWriteUserGroup, false))
}

func testAccessControlForRole(
	app *server_test.TestServer,
	serverURL string,
	companyID models.LegalEntityID,
	repoID models.RepoID,
	certPEM string,
	usersName string,
	group *models.Group,
	isAdmin bool,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		// Make a person, with a login account, shared secret credential and client
		userData := models.NewPersonLegalEntityData(models.ResourceName(usersName), usersName+" Smith", usersName+"@not.real.domain", nil, "")
		user, err := app.LegalEntityService.Create(ctx, nil, userData)
		require.NoError(t, err)
		userIdentity, err := app.LegalEntityService.ReadIdentity(ctx, nil, user.ID)
		require.NoError(t, err)
		token, _, err := app.CredentialService.CreateSharedSecretCredential(ctx, nil, userIdentity.ID, true)
		require.NoError(t, err)
		client, err := client.NewAPIClient(
			[]string{serverURL},
			client.NewSharedSecretAuthenticator(client.SharedSecretToken(token.String()), app.LogFactory),
			app.LogFactory)
		require.NoError(t, err)

		// user initially should not be able to read the company legal user - they aren't in any access control group
		_, err = client.GetLegalEntity(ctx, companyID)
		require.Error(t, err, "User should not be able to read legal user until they have a role")

		// put user in the supplied access control group
		_, _, err = app.GroupService.FindOrCreateMembership(ctx, nil, models.NewGroupMembershipData(
			group.ID, userIdentity.ID, models.TestsSystem, user.ID))
		require.NoError(t, err)

		// user now *should* be able to read the company legal user based on being a member of any group
		_, err = client.GetLegalEntity(ctx, companyID)
		require.NoError(t, err, "User should be able to read legal user after they have a role")

		// Check that only admins can register runners
		t.Logf("Attempting to register runner for user %s, in group %s", usersName, group.Name)
		resp, err := client.CreateRunner(ctx, companyID, "test-runner-1", certPEM)
		if isAdmin {
			require.NoError(t, err, "Admin user should be able to register a runner")
			runnerID := resp.ID

			// Check the admin can read the registration back
			runnerDoc, err := client.GetRunner(ctx, runnerID)
			require.NoError(t, err, "Admin user should be able to read a runner registration")
			require.Equal(t, runnerID, runnerDoc.ID)
		} else {
			// Non-admins should not be able to register a runner
			require.Error(t, err, "Non-admin user should not be able to register a runner")
		}
	}
}
