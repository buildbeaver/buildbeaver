package clienttest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/runner/app/runner_test"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

// MakeClientCertificateAPIClient creates an API client that can be used to communicate with the given test server.
// The client will use client certificate authentication, generating a client certificate if required.
// Returns the client, and the certificate to use.
func MakeClientCertificateAPIClient(t *testing.T, app *server_test.TestServer) (*client.APIClient, certificates.CertificateData) {
	// For configuring a client, use the same test configuration as is used by runner_test
	config := runner_test.TestConfig(t)
	config.RunnerAPIEndpoints = []string{app.RunnerAPIServer.GetServerURL()}

	// Create a client certificate authenticator; this will automatically generate certificates if required
	clientCertificateAuthenticator, err := client.NewClientCertificateAuthenticator(
		config.RunnerCertificateFile,
		config.RunnerPrivateKeyFile,
		true,
		config.CACertFile,
		true,
		app.LogFactory,
	)
	require.Nil(t, err)

	// Make client
	apiClient, err := client.NewAPIClient(config.RunnerAPIEndpoints, clientCertificateAuthenticator, app.LogFactory)
	require.Nil(t, err)

	// Load the certificate that was used by the client, so we can return it
	clientCert, err := certificates.LoadCertificateFromPemFile(config.RunnerCertificateFile)
	require.Nil(t, err)

	return apiClient, clientCert
}
