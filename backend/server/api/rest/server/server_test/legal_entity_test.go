package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestLegalEntityReadSelf(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.CoreAPIServer.Start()
	defer app.CoreAPIServer.Stop(ctx)

	entity, identity := server_test.CreatePersonLegalEntity(t, ctx, app, "test", "Jim Bob", "jim@bob.com")
	token, _, err := app.CredentialService.CreateSharedSecretCredential(ctx, nil, identity.ID, true)
	require.NoError(t, err)
	client, err := client.NewAPIClient(
		[]string{app.CoreAPIServer.GetServerURL()},
		client.NewSharedSecretAuthenticator(client.SharedSecretToken(token.String()), app.LogFactory),
		app.LogFactory)
	require.Nil(t, err)

	// Legal entity should be able to read itself
	_, err = client.GetLegalEntity(ctx, entity.ID)
	require.Nil(t, err)
}
