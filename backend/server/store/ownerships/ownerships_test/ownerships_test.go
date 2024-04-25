package artifacts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestUpsertOwnership(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	now := time.Now()

	originalOwnership := models.NewOwnership(
		models.NewTime(now),
		models.NewResourceID(models.LegalEntityResourceKind),
		models.NewResourceID(models.RunnerResourceKind))

	hash, err := hashstructure.Hash(originalOwnership, hashstructure.FormatV2, nil)
	require.Nil(t, err)
	originalEtag := models.ETag(fmt.Sprintf("\"%x\"", hash))

	created, updated, err := app.OwnershipStore.Upsert(context.Background(), nil, originalOwnership)
	require.Nil(t, err)
	require.True(t, created)
	require.False(t, updated)
	require.Equal(t, originalOwnership.ETag, originalEtag)

	ownership, err := app.OwnershipStore.Read(context.Background(), nil, originalOwnership.ID)
	require.Nil(t, err)
	require.Equal(t, originalOwnership, ownership)
	require.Equal(t, originalOwnership.ETag, originalEtag)

	created, updated, err = app.OwnershipStore.Upsert(context.Background(), nil, originalOwnership)
	require.Nil(t, err)
	require.False(t, created)
	require.False(t, updated)
	require.Equal(t, originalOwnership.ETag, originalEtag)

	ownership, err = app.OwnershipStore.Read(context.Background(), nil, originalOwnership.ID)
	require.Nil(t, err)
	require.Equal(t, originalOwnership, ownership)
	require.Equal(t, originalOwnership.ETag, originalEtag)

	originalOwnership.OwnerResourceID = models.NewResourceID(models.LegalEntityResourceKind)
	hash, err = hashstructure.Hash(originalOwnership, hashstructure.FormatV2, nil)
	require.Nil(t, err)
	newEtag := models.ETag(fmt.Sprintf("\"%x\"", hash))

	created, updated, err = app.OwnershipStore.Upsert(context.Background(), nil, originalOwnership)
	require.Nil(t, err)
	require.False(t, created)
	require.True(t, updated)
	require.Equal(t, originalOwnership.ETag, newEtag)

	ownership, err = app.OwnershipStore.Read(context.Background(), nil, originalOwnership.ID)
	require.Nil(t, err)
	require.Equal(t, originalOwnership, ownership)
	require.Equal(t, originalOwnership.ETag, newEtag)
}
