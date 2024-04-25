package grants_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
)

func TestGrants(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	// Set up reference data
	// Don't use LegalEntityService since we don't want the standard grants to be set up for this test
	company, err := app.LegalEntityStore.Create(
		context.Background(),
		nil,
		models.NewCompanyLegalEntityData(
			"orgmang",
			"My org",
			"foso@bar.com",
			nil,
			"",
		),
	)
	require.NoError(t, err)
	ownership := models.NewOwnership(models.NewTime(now), company.GetID(), company.GetID())
	err = app.OwnershipStore.Create(context.Background(), nil, ownership)
	require.NoError(t, err)
	companyIdentity := models.NewIdentity(models.NewTime(now), company.ID.ResourceID)
	err = app.IdentityStore.Create(ctx, nil, companyIdentity)
	require.NoError(t, err)
	companyIdentityOwnership := models.NewOwnership(models.NewTime(now), company.ID.ResourceID, companyIdentity.ID.ResourceID)
	err = app.OwnershipStore.Create(ctx, nil, companyIdentityOwnership)
	require.NoError(t, err)

	person, err := app.LegalEntityStore.Create(ctx, nil,
		referencedata.GeneratePersonLegalEntity("frankieboi", "Frank Sinatra", "frank@bar.com"))
	require.NoError(t, err)

	// In this case the person is owned by the company; this ownership is required later in the tests
	err = app.OwnershipStore.Create(ctx, nil, models.NewOwnership(models.NewTime(now), company.GetID(), person.GetID()))
	require.NoError(t, err)
	personIdentity := models.NewIdentity(models.NewTime(now), person.ID.ResourceID)
	err = app.IdentityStore.Create(ctx, nil, personIdentity)
	require.NoError(t, err)
	err = app.OwnershipStore.Create(ctx, nil, models.NewOwnership(models.NewTime(now), person.ID.ResourceID, personIdentity.ID.ResourceID))
	require.NoError(t, err)

	// Create a second person who should not be put in the admin group
	irresponsiblePerson, err := app.LegalEntityStore.Create(ctx, nil,
		referencedata.GeneratePersonLegalEntity("mr-irresponsible", "I. M. Irresponsible", "ir@bar.com"))
	require.NoError(t, err)

	// The irresponsible person is owned by themselves; they are independent of the company
	err = app.OwnershipStore.Create(ctx, nil, models.NewOwnership(models.NewTime(now), irresponsiblePerson.GetID(), irresponsiblePerson.GetID()))
	require.NoError(t, err)
	irresponsibleIdentity := models.NewIdentity(models.NewTime(now), irresponsiblePerson.ID.ResourceID)
	err = app.IdentityStore.Create(ctx, nil, irresponsibleIdentity)
	require.NoError(t, err)
	err = app.OwnershipStore.Create(ctx, nil, models.NewOwnership(models.NewTime(now), irresponsiblePerson.ID.ResourceID, irresponsibleIdentity.ID.ResourceID))
	require.NoError(t, err)

	repo := referencedata.GenerateRepo("", company.ID)
	err = app.RepoStore.Create(ctx, nil, repo)
	require.NoError(t, err)

	ownership = models.NewOwnership(models.NewTime(now), person.GetID(), repo.GetID())
	err = app.OwnershipStore.Create(ctx, nil, ownership)
	require.NoError(t, err)

	// Create two groups for admins and general users
	adminGroup := models.NewGroup(
		models.NewTime(now),
		company.ID,
		"admin",
		"Can do all the things",
		true,
		nil)
	err = app.GroupStore.Create(ctx, nil, adminGroup)
	require.NoError(t, err)
	ownership = models.NewOwnership(models.NewTime(now), company.GetID(), adminGroup.GetID())
	err = app.OwnershipStore.Create(ctx, nil, ownership)
	require.NoError(t, err)
	userGroup := models.NewGroup(
		models.NewTime(now),
		company.ID,
		"user",
		"Can do some of the things",
		true,
		nil)
	err = app.GroupStore.Create(ctx, nil, userGroup)
	require.NoError(t, err)
	ownership = models.NewOwnership(models.NewTime(now), company.GetID(), userGroup.GetID())
	err = app.OwnershipStore.Create(ctx, nil, ownership)
	require.NoError(t, err)

	// Person is in adminGroup but not userGroup
	membership, err := app.GroupMembershipStore.Create(ctx, nil,
		models.NewGroupMembershipData(adminGroup.ID, personIdentity.ID, models.TestsSystem, company.ID))
	require.NoError(t, err)
	ownership = models.NewOwnership(models.NewTime(now), company.GetID(), membership.GetID())
	err = app.OwnershipStore.Create(ctx, nil, ownership)
	require.NoError(t, err)

	// irresponsiblePerson is in userGroup but not adminGroup
	membership, err = app.GroupMembershipStore.Create(ctx, nil,
		models.NewGroupMembershipData(userGroup.ID, irresponsibleIdentity.ID, models.TestsSystem, company.ID))
	require.NoError(t, err)
	ownership = models.NewOwnership(models.NewTime(now), company.GetID(), membership.GetID())
	err = app.OwnershipStore.Create(ctx, nil, ownership)
	require.NoError(t, err)

	// Check there are no grants initially
	count, err := app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		personIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Grant person read repo permissions explicitly
	grant := models.NewIdentityGrant(
		models.NewTime(now),
		company.ID,
		personIdentity.ID,
		*models.RepoReadOperation,
		repo.GetID())
	err = app.GrantStore.Create(ctx, nil, grant)
	require.NoError(t, err)

	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		personIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Grant read repo permissions to the company (and therefore indirectly to the person)
	grant = models.NewIdentityGrant(
		models.NewTime(now),
		company.ID,
		personIdentity.ID,
		*models.RepoReadOperation,
		company.GetID())

	err = app.GrantStore.Create(ctx, nil, grant)
	require.NoError(t, err)

	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		personIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Grant the adminGroup permissions to read repos owned by the company -
	// our person is a member of this adminGroup and so they should inherit this permission.
	adminGrant := models.NewGroupGrant(
		models.NewTime(now),
		company.ID,
		adminGroup.ID,
		*models.RepoReadOperation,
		company.GetID())
	err = app.GrantStore.Create(ctx, nil, adminGrant)
	require.NoError(t, err)

	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		personIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Company itself is not in the admin group, so it doesn't have a relevant grant
	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		companyIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Explicitly grant the company read rights to repos
	companyGrant := models.NewIdentityGrant(
		models.NewTime(now),
		company.ID,
		companyIdentity.ID,
		*models.RepoReadOperation,
		company.GetID())
	err = app.GrantStore.Create(ctx, nil, companyGrant)
	require.NoError(t, err)

	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		companyIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Check that irresponsiblePerson doesn't have access to the repos
	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		irresponsibleIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Grant the 'admin' group access to the repos through an upsert - should be a no-op since the grant already exists
	_, created, err := app.GrantStore.FindOrCreate(ctx, nil, adminGrant)
	require.NoError(t, err)
	require.False(t, created)

	// Person should still have the same number of grants
	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		personIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Grant the 'user' group access to the repos through an upsert - should succeed
	irresponsibleGrant := models.NewGroupGrant(
		models.NewTime(now),
		company.ID,
		userGroup.ID,
		*models.RepoReadOperation,
		company.GetID())
	_, created, err = app.GrantStore.FindOrCreate(ctx, nil, irresponsibleGrant)
	require.NoError(t, err)
	require.True(t, created)

	// irresponsiblePerson should now have a grant (the irresponsibleGrant)
	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		irresponsibleIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Perform the same upsert of irresponsibleGrant again - should be a no-op since the grant already exists
	_, created, err = app.GrantStore.FindOrCreate(ctx, nil, irresponsibleGrant)
	require.NoError(t, err)
	require.False(t, created)

	count, err = app.AuthorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		irresponsibleIdentity.ID,
		models.RepoReadOperation,
		repo.GetID())
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
