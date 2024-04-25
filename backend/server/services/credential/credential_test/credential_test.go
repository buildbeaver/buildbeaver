package credential_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/app/runner_test"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestCredentials(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	// Create a company to get an identity to use for testing credentials
	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")
	identity, err := app.LegalEntityService.ReadIdentity(ctx, nil, testCompany.ID)
	require.NoError(t, err)

	// Create a runner test app in order to get a client certificate, but don't start or register the runner
	runnerConfig := runner_test.TestConfig(t)
	_, err = runner_test.New(runnerConfig)
	require.NoError(t, err)
	certificateFile := runnerConfig.RunnerCertificateFile
	certificate, err := certificates.LoadCertificateFromPemFile(certificateFile)
	require.NoError(t, err)

	t.Run("Client-Certificate", testClientCertificateCredential(app, identity.ID, certificate))
	t.Run("JWT Credential", testJWTCredential(app, identity.ID))
	t.Run("JWT Credential Expiry", testJWTCredentialExpiry(app, identity.ID))
	t.Run("JWT Credential Wrong Key", testJWTCredentialWrongKey(app, identity.ID))
}

func testClientCertificateCredential(
	app *server_test.TestServer,
	identityID models.IdentityID,
	certificate certificates.CertificateData,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		pagination := models.NewPagination(30, nil) // read enough to cover our test data

		publicKey, err := certificates.GetPublicKeyFromCertificate(certificate)
		require.NoError(t, err, "Error extracting public key from supplied test certificate data")

		// Create a credential
		credential, err := app.CredentialService.CreateClientCertificateCredential(
			ctx,
			nil,
			identityID,
			true,
			certificate,
		)
		require.NoError(t, err, "Error creating client-certificate credential")

		// Read credential back by ID
		credReadBack, err := app.CredentialStore.Read(ctx, nil, credential.ID)
		assert.NoError(t, err, "Error reading back credential")
		assert.NotNil(t, credReadBack)
		certDataReadBack := certificates.CertificateData(credReadBack.ClientCertificateASN1)

		// Read credential back by public key; this checks the ClientPublicKeyASN1Hash field
		credReadBack2, err := app.CredentialStore.ReadByPublicKey(ctx, nil, publicKey)
		assert.NoError(t, err, "Error reading back credential")
		assert.NotNil(t, credReadBack2)
		certDataReadBack2 := certificates.CertificateData(credReadBack2.ClientCertificateASN1)

		// Check the public key and certificate data is correct
		assert.Equal(t, publicKey.AsPEM(), credReadBack.ClientPublicKeyPEM, "Unexpected PEM value in credential")
		assert.Equal(t, publicKey.AsPEM(), credReadBack2.ClientPublicKeyPEM, "Unexpected PEM value in credential (2)")
		assert.Equal(t, certificate, certDataReadBack, "Unexpected certificate data in credential")
		assert.Equal(t, certificate, certDataReadBack2, "Unexpected certificate data in credential (2)")

		// Log some data for a person to eyeball and sanity check
		t.Logf("Read back client certificate credential:\n"+
			"Public Key PEM:\n%s\n"+
			"Public Key Hash: %s (type %s)\n"+
			"Certificate data (binary length %d) PEM-encoded:\n%s\n",
			credReadBack.ClientPublicKeyPEM,
			credReadBack.ClientPublicKeyASN1Hash,
			credReadBack.ClientPublicKeyASN1HashType,
			len(certDataReadBack),
			certDataReadBack.AsPEM(),
		)

		// Try to create another credential with the same certificate; should fail
		_, err = app.CredentialService.CreateClientCertificateCredential(
			ctx,
			nil,
			identityID,
			true,
			certificate,
		)
		assert.Error(t, err, "Creating another client-certificate credential with the same certificate should fail")

		// List credentials for the identity; we should see the one we created
		credentials, _, err := app.CredentialService.ListCredentialsForIdentity(ctx, nil, identityID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 1, len(credentials), "Should have 1 credential for identity")

		// Delete credential and then check it is gone
		err = app.CredentialStore.Delete(ctx, nil, credential.ID)
		require.NoError(t, err)

		_, err = app.CredentialStore.Read(ctx, nil, credential.ID)
		assert.Error(t, err, "Should not be able to read deleted credential by ID")

		credentials, _, err = app.CredentialService.ListCredentialsForIdentity(ctx, nil, identityID, pagination)
		require.NoError(t, err)
		assert.Zero(t, len(credentials), "Should have no remaining credentials for identity")
	}
}

func testJWTCredential(app *server_test.TestServer, identityID models.IdentityID) func(t *testing.T) {
	return func(t *testing.T) {
		// Create a credential
		tokenStr, err := app.CredentialService.CreateIdentityJWT(identityID)
		require.NoError(t, err, "Error creating JTW credential")

		// Verify the credential
		identityReadBack, err := app.CredentialService.VerifyIdentityJWT(tokenStr)
		assert.NoError(t, err, "Error verifying JWT credential")
		assert.Equal(t, identityID, identityReadBack, "Identity verified by JTW doesn't match")
	}
}

func testJWTCredentialExpiry(app *server_test.TestServer, identityID models.IdentityID) func(t *testing.T) {
	return func(t *testing.T) {
		// Change to a negative expiry duration so all JWTs issued are expired, put back default before returning
		expiryDuration := -1 * time.Minute

		// Create a credential
		service := app.CredentialService.(*credential.CredentialService)
		tokenStr, err := service.CreateIdentityJWTWithExpiry(identityID, expiryDuration)
		require.NoError(t, err, "Error creating JTW credential")

		// Verify the credential - should fail
		_, err = app.CredentialService.VerifyIdentityJWT(tokenStr)
		assert.Error(t, err, "Expected an error verifying expired JWT credential")
	}
}

func testJWTCredentialWrongKey(app *server_test.TestServer, identityID models.IdentityID) func(t *testing.T) {
	return func(t *testing.T) {
		_, newPrivateKey, err := ed25519.GenerateKey(rand.Reader)

		// Create a credential directly using the util, signed with the 'wrong' custom private key
		tokenStr, _, err := credential.CreateIdentityJWT(identityID, credential.DefaultJWTIssuer, credential.DefaultJWTExpiryDuration, newPrivateKey)
		require.NoError(t, err, "Error creating JTW credential")

		// Verify the credential using the standard public key should fail
		_, err = app.CredentialService.VerifyIdentityJWT(tokenStr)
		assert.Error(t, err, "Expected an error verifying expired JWT credential")
	}
}
