package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

// TestResourceAlreadyExistsThrown tests that MakeStandardDBError provides the correct error code when we attempt to
// create a unique resource that already exists
func TestResourceAlreadyExistsThrown(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	personData := models.NewPersonLegalEntityData(
		"frankieboi",
		"Frank Sinatra",
		"frank@bar.com",
		nil,
		"")

	// First legal entity creation will pass
	_, err = app.LegalEntityStore.Create(context.Background(), nil, personData)
	require.Nil(t, err)

	// Second legal entity creation should fail with ErrCodeAlreadyExists
	_, err = app.LegalEntityStore.Create(context.Background(), nil, personData)
	require.NotNil(t, err)
	require.NotNil(t, gerror.ToAlreadyExists(err))
}

// TestResourceNotFoundThrown tests that MakeStandardDBError provides the correct error code when we attempt to
// retrieve a resource that doesn't exist.
func TestResourceNotFoundThrown(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	_, err = app.LegalEntityStore.Read(context.Background(), nil, models.LegalEntityID{})
	require.NotNil(t, err)
	require.NotNil(t, gerror.ToNotFound(err))
}
