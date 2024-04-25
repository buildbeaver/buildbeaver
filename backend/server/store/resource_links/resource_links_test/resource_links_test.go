package resource_links_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
)

func TestResolve(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}
	defer cleanup()

	ctx := context.Background()

	legalEntity, err := app.LegalEntityStore.Create(context.Background(), nil,
		referencedata.GeneratePersonLegalEntity("", "", ""))
	require.Nil(t, err)

	repo := referencedata.GenerateRepo("", legalEntity.ID)
	err = app.RepoStore.Create(context.Background(), nil, repo)
	require.Nil(t, err)

	secret := server_test.CreateSecret(t, ctx, app, repo.ID, "my_secret")

	created, updated, err := app.ResourceLinkStore.Upsert(ctx, nil, legalEntity)
	assert.Nil(t, err)
	assert.True(t, created)
	assert.False(t, updated)

	created, updated, err = app.ResourceLinkStore.Upsert(ctx, nil, repo)
	assert.Nil(t, err)
	assert.True(t, created)
	assert.False(t, updated)

	created, updated, err = app.ResourceLinkStore.Upsert(ctx, nil, secret)
	assert.Nil(t, err)
	assert.True(t, created)
	assert.False(t, updated)

	resource, err := app.ResourceLinkStore.Resolve(ctx, nil, []models.ResourceLinkFragmentID{
		{
			Kind: models.LegalEntityResourceKind,
			Name: legalEntity.Name,
		},
		{
			Kind: models.RepoResourceKind,
			Name: repo.Name,
		},
		{
			Kind: models.SecretResourceKind,
			Name: secret.Name,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, secret.ID.ResourceID, resource.ID)
	assert.Equal(t, secret.Name, resource.Name)
	assert.Equal(t, secret.GetParentID(), resource.ParentID)
	assert.Equal(t, secret.GetKind(), resource.Kind)
}
