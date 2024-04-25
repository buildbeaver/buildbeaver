package secrets_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestSecret(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	ctx := context.Background()

	company := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, company.ID)

	t.Run("CreateInvalid", testSecretCreateInvalid(app.SecretStore))
	t.Run("CreateSecret", testSecretCreate(app, repo.ID))
}

// testSecretCreateInvalid tests the invalid Secret creation cases throw errors.
func testSecretCreateInvalid(store store.SecretStore) func(t *testing.T) {
	return func(t *testing.T) {
		// Test passing in nil
		err := store.Create(context.Background(), nil, &models.Secret{
			ID:               models.NewSecretID(),
			CreatedAt:        models.Time{},
			UpdatedAt:        models.Time{},
			RepoID:           models.RepoID{},
			KeyEncrypted:     nil,
			ValueEncrypted:   nil,
			DataKeyEncrypted: nil,
			IsInternal:       false,
		})
		require.NotNil(t, err)
	}
}

// testSecretCreate tests creating a valid Secret
func testSecretCreate(app *server_test.TestServer, repoID models.RepoID) func(t *testing.T) {
	return func(t *testing.T) {
		secret := server_test.CreateSecret(t, context.Background(), app, repoID, "a")
		t.Run("Read", testSecretRead(app.SecretStore, secret.ID, secret))
	}
}

// testSecretRead tests reading back a Secret from the store matches the data passed into creation.
func testSecretRead(store store.SecretStore, testSecretID models.SecretID, referenceSecret *models.Secret) func(t *testing.T) {
	return func(t *testing.T) {
		secret, err := store.Read(context.Background(), nil, testSecretID)
		require.Nil(t, err)
		require.Equal(t, referenceSecret, secret)
		t.Run("Delete", testSecretDelete(store, secret.ID))
	}
}

// testSecretDelete tests that we are able to delete a stored Secret and that it no longer exists in the store.
func testSecretDelete(store store.SecretStore, testSecretID models.SecretID) func(t *testing.T) {
	return func(t *testing.T) {
		// Delete the known Secret
		err := store.Delete(context.Background(), nil, testSecretID)
		require.Nil(t, err)

		// Read the deleted Secret which should throw an error.
		_, err = store.Read(context.Background(), nil, testSecretID)
		require.NotNil(t, err)
	}
}
