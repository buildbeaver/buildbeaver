package certificates_test_utils

import (
	"path/filepath"
	"testing"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/runner/app"
)

// CreateTestClientCertificate creates a public-private key pair and certificate for use in client certificate
// authentication in tests, in a temporary directory that will be removed when the tests finish.
// Returns the client certificate as a PEM string, as well as the paths to the certificate and the private key
// within the temporary directory.
func CreateTestClientCertificate(t *testing.T) (certificateAsPEM string, certificateFile certificates.CertificateFile, privateKeyFile certificates.PrivateKeyFile, err error) {
	// Create a temp directory for the certificate
	certDir := t.TempDir()
	certificateFile = certificates.CertificateFile(filepath.Join(certDir, app.DefaultRunnerCertFile))
	privateKeyFile = certificates.PrivateKeyFile(filepath.Join(certDir, app.DefaultRunnerPrivateKeyFile))

	// Make a certificate pair to use for testing
	_, err = certificates.GenerateClientSelfSignedCertificate(certificateFile, privateKeyFile, "Test Client")
	if err != nil {
		return "", "", "", err
	}

	certificateAsPEM, err = certificates.LoadCertificateFromPemFileAsString(certificateFile)
	if err != nil {
		return "", "", "", err
	}

	return certificateAsPEM, certificateFile, privateKeyFile, nil
}
