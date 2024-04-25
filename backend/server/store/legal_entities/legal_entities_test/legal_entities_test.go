package legal_entities_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestLegalEntities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	company := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")

	personLegalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "billy", "Billy 2", "billy@billy2.billy")

	person2LegalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "nancy", "Nancy 1", "nancy@nancy.com")

	pagination := models.NewPagination(models.DefaultPaginationLimit, nil)

	entities, _, err := app.LegalEntityStore.ListParentLegalEntities(ctx, nil, personLegalEntity.ID, pagination)
	require.Nil(t, err)
	require.Len(t, entities, 0, "no membership created yet")

	memberEntities, _, err := app.LegalEntityStore.ListMemberLegalEntities(ctx, nil, personLegalEntity.ID, pagination)
	require.Nil(t, err)
	require.Len(t, memberEntities, 0, "no membership created yet 2")

	// Test Create method with personLegalEntity
	membership := models.NewLegalEntityMembership(
		models.NewTime(time.Now()),
		company.ID,
		personLegalEntity.ID)
	err = app.LegalEntityMembershipStore.Create(ctx, nil, membership)
	require.Nil(t, err)

	entities, _, err = app.LegalEntityStore.ListParentLegalEntities(ctx, nil, personLegalEntity.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 1, "there should be one membership for person 2")

	entities, _, err = app.LegalEntityStore.ListMemberLegalEntities(ctx, nil, company.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 1, "there should be one member for company")

	err = app.LegalEntityMembershipStore.Create(ctx, nil, membership)
	require.Error(t, err)
	require.NotNil(t, gerror.ToAlreadyExists(err))

	// Test FindOrCreate method with person2LegalEntity
	membership = models.NewLegalEntityMembership(
		models.NewTime(time.Now()),
		company.ID,
		person2LegalEntity.ID)
	membership, created, err := app.LegalEntityMembershipStore.FindOrCreate(ctx, nil, membership)
	require.Nil(t, err)
	require.True(t, created, "created should be true for FindOrCreate on new membership")
	require.NotNil(t, membership)

	entities, _, err = app.LegalEntityStore.ListParentLegalEntities(ctx, nil, person2LegalEntity.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 1, "there should be one membership for person 2")

	entities, _, err = app.LegalEntityStore.ListMemberLegalEntities(ctx, nil, company.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 2, "there should be 2 members for company")

	// Repeat the FindOrCreate, should find the membership this time
	existingMembership, created, err := app.LegalEntityMembershipStore.FindOrCreate(ctx, nil, membership)
	require.NoError(t, err)
	require.False(t, created, "created should be false for FindOrCreate on existing membership")
	require.NotNil(t, existingMembership)
	require.Equal(t, membership.ID, existingMembership.ID, "existing membership should be returned for FindOrCreate on existing membership")

	entities, _, err = app.LegalEntityStore.ListParentLegalEntities(ctx, nil, person2LegalEntity.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 1, "there should be one membership for person 2")

	// Remove one of the legal entity memberships
	err = app.LegalEntityMembershipStore.DeleteByMember(ctx, nil, company.ID, person2LegalEntity.ID)
	require.NoError(t, err)
	entities, _, err = app.LegalEntityStore.ListParentLegalEntities(ctx, nil, person2LegalEntity.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 0, "there should be no memberships left for person 2")
	entities, _, err = app.LegalEntityStore.ListMemberLegalEntities(ctx, nil, company.ID, pagination)
	require.NoError(t, err)
	require.Len(t, entities, 1, "there should be one member left for company")
}
