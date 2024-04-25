package certificates

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
)

const certificateExpiryDuration = 5 * 365 * 24 * time.Hour // 5 years

// CertificateFile is a filename for a pem file containing an X.509 certificate
type CertificateFile string

func (f CertificateFile) String() string {
	return string(f)
}

// PrivateKeyFile is a filename for a file containing a private key corresponding to the public key in a certificate.
type PrivateKeyFile string

func (f PrivateKeyFile) String() string {
	return string(f)
}

// CACertificateFile is a filename for a pem file containing an X.509 certificate for a CA
type CACertificateFile string

func (f CACertificateFile) String() string {
	return string(f)
}

// CertificateData contains the binary data for an ASN.1 DER-encoded X.509 certificate.
// This is the canonical format for a certificate.
type CertificateData []byte

func (c CertificateData) Equals(other CertificateData) bool {
	return bytes.Compare(c, other) == 0
}

// AsPEM converts the certificate data to a PEM-encoded certificate.
func (c CertificateData) AsPEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c}))
}

// PublicKeyData contains the binary data for an ASN.1 DER-encoded SubjectPublicKeyInfo format public key.
// This is the canonical format for a public key.
type PublicKeyData []byte

func (k PublicKeyData) Equals(other PublicKeyData) bool {
	return bytes.Compare(k, other) == 0
}

// AsPEM converts the public key data to a PEM-encoded public key.
func (k PublicKeyData) AsPEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: k}))
}

// SHA256Hash returns the SHA-256 hash of the public key as a hex-encoded string.
func (k PublicKeyData) SHA256Hash() string {
	hash := sha256.Sum256(k)
	return hex.EncodeToString(hash[:])
}

// GenerateServerSelfSignedCertificate checks whether a certificate and corresponding private key exist, and if not
// then a new public/private key pair and self-signed certificate are created, in two separate .pem files.
// certFilename and privateKeyFilename are the file names (with full path) of the files.
// The entire path to the directory the certificate file is in will be created if it doesn't exist.
// Any generated certificate will be suitable for use with Web servers serving (i.e. browser-compatible).
// host is a mandatory comma-separated list of hostnames and/or IP addresses to put in the certificate.
func GenerateServerSelfSignedCertificate(
	certFilename CertificateFile,
	privateKeyFilename PrivateKeyFile,
	host string,
	organization string,
) (bool, error) {
	if len(host) == 0 {
		return false, fmt.Errorf("error creating self-signed server certificate: host name required")
	}

	return GenerateCertificateIfNotExists(
		certFilename,
		privateKeyFilename,
		host,
		organization,
		false, // do not use ed25519; it's a great key type but not compatible with browsers
	)
}

// GenerateClientSelfSignedCertificate checks whether a certificate and corresponding private key exist, and if not
// then a new public/private key pair and self-signed certificate are created, in two separate .pem files.
// certFilename and privateKeyFilename are the file names (with full path) of the files.
// The entire path to the directory the certificate file is in will be created if it doesn't exist.
// Any generated certificate will be suitable for use for client-certificate authentication, and will use
// key types that are very secure but not compatible with Web browsers.
func GenerateClientSelfSignedCertificate(
	certFilename CertificateFile,
	privateKeyFilename PrivateKeyFile,
	organization string,
) (bool, error) {
	return GenerateCertificateIfNotExists(
		certFilename,
		privateKeyFilename,
		"", // no host required in certificate
		organization,
		true, // use ed25519 even though it's not compatible with browsers
	)
}

// GenerateEd25519SigningKeyAndCertificate checks whether a certificate and corresponding private key exist, and
// if not then a new public/private key pair and self-signed certificate are created, in two separate .pem files.
// certFilename and privateKeyFilename are the file names (with full path) of the files.
// The entire path to the directory the certificate file is in will be created if it doesn't exist.
// Any generated key pair will be compatible with the ed25519 signing algorithm and suitable for use when
// digitally signing documents including JWT tokens.
// This key type is very secure but not compatible with Web browsers.
func GenerateEd25519SigningKeyAndCertificate(
	certFilename CertificateFile,
	privateKeyFilename PrivateKeyFile,
	organization string,
) (bool, error) {
	return GenerateCertificateIfNotExists(
		certFilename,
		privateKeyFilename,
		"", // no host required in certificate
		organization,
		true,
	)
}

// GenerateCertificateIfNotExists checks whether a certificate and corresponding private key exist, and if not
// then a new public/private key pair and self-signed certificate are created, in two separate .pem files.
// certFilename and privateKeyFilename are the file names (with full path) of the files.
// The entire path to the directory the certificate file is in will be created if it doesn't exist.
// If useEd25519Key is true then an ed25519 key is generated (not browser compatible), otherwise an ecdsa key is
// generated using the "P256" ecdsaCurve (compatible with most browsers). This function will not generate an RSA key.
// Returns true if a new key pair and certificate were created.
func GenerateCertificateIfNotExists(
	certFilename CertificateFile,
	privateKeyFilename PrivateKeyFile,
	host string,
	organization string,
	useEd25519Key bool,
) (bool, error) {
	if certFilename == "" || privateKeyFilename == "" {
		return false, fmt.Errorf("error checking certificate: directory and filenames must not be empty")
	}

	// Ensure certificate directory exists, producing meaningful errors
	certDir := filepath.Dir(certFilename.String())
	certDirInfo, err := os.Stat(certDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Directory doesn't exist, so create the entire path
			err = os.MkdirAll(certDir, 0755) // read and traverse permissions for everyone
			if err != nil {
				return false, fmt.Errorf("error making runner certificate directory %s: %w", certDir, err)
			}
		} else {
			return false, fmt.Errorf("error checking for existence of directory %s: %w", certDir, err)
		}
	} else if !certDirInfo.IsDir() {
		return false, fmt.Errorf("error making runner certificate directory %s: file is present with same name", certDir)
	}

	certFileExists := false
	if _, err := os.Stat(certFilename.String()); err == nil {
		certFileExists = true
	} else if errors.Is(err, os.ErrNotExist) {
		certFileExists = false
	} else {
		return false, fmt.Errorf("error checking for certificate file at directory %s: %w", certDir, err)
	}
	privateKeyFileExists := false
	if _, err := os.Stat(privateKeyFilename.String()); err == nil {
		privateKeyFileExists = true
	} else if errors.Is(err, os.ErrNotExist) {
		privateKeyFileExists = false
	} else {
		return false, fmt.Errorf("error checking for private key file at directory %s: %w", certDir, err)
	}

	// Check we don't have one file without the other
	if certFileExists && !privateKeyFileExists {
		return false, fmt.Errorf("error: certificate file exists at %s but private key file is missing at %s",
			certFilename, privateKeyFilename)
	}
	if !certFileExists && privateKeyFileExists {
		return false, fmt.Errorf("error: private key file exists at %s but certificate file is missing at %s",
			privateKeyFilename, certFilename)
	}

	// Ensure we have a private/public key pair
	created := false
	if !certFileExists && !privateKeyFileExists {
		// Create private key file and certificate
		err = generateSelfSignedCertificate(
			certFilename,
			privateKeyFilename,
			host,
			organization,
			certificateExpiryDuration,
			useEd25519Key,
			"P256", // only used if useEd25519Key is false
		)
		if err != nil {
			return false, fmt.Errorf("error creating private key and certificate: %w", err)
		}
		created = true
	}
	return created, err
}

func publicKey(privateKey interface{}) interface{} {
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

// generateSelfSignedCertificate creates a self-signed certificate using an ed25519 or ecdsa key.
// If useEd25519Key is true then an ed25519 key is generated, otherwise an ecdsa key is generated using
// the specified ecdsaCurve. This function will not generate an RSA key.
func generateSelfSignedCertificate(
	certFilename CertificateFile,
	privateKeyFilename PrivateKeyFile,
	host string,
	organization string,
	validFor time.Duration,
	useEd25519Key bool,
	ecdsaCurve string,
) error {
	var privateKey interface{}
	var err error
	if useEd25519Key {
		_, privateKey, err = ed25519.GenerateKey(rand.Reader)
	} else {
		switch ecdsaCurve {
		case "P224":
			privateKey, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
		case "P256":
			privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		case "P384":
			privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		case "P521":
			privateKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		default:
			return fmt.Errorf("error: unrecognized elliptic curve %q", ecdsaCurve)
		}
	}
	if err != nil {
		return fmt.Errorf("error generating private key: %w", err)
	}

	// Set the DigitalSignature KeyUsage bits in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("error generating serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if host != "" {
		hosts := strings.Split(host, ",")
		for _, h := range hosts {
			if ip := net.ParseIP(h); ip != nil {
				template.IPAddresses = append(template.IPAddresses, ip)
			} else {
				template.DNSNames = append(template.DNSNames, h)
			}
		}
	}

	// Self-signed certs are their own CA
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	if err != nil {
		return fmt.Errorf("error creating certificate: %w", err)
	}

	certOut, err := os.Create(certFilename.String())
	if err != nil {
		return fmt.Errorf("error opening certificate file for writing: %w", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("error writing data to certificate file: %w", err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("error closing certificate file: %w", err)
	}

	keyOut, err := os.OpenFile(privateKeyFilename.String(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening private key file for writing: %w", err)
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("error marshalling private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("error writing data to private key file: %w", err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing private key file: %w", err)
	}

	return nil
}

// ValidatePEMDataAsX509Certificate checks that the first PEM-encoded block in the supplied data contains an
// X.509 certificate. Returns a Validation Failed error if no X.509 certificate is found.
func ValidatePEMDataAsX509Certificate(pemData string) error {
	_, err := GetEncodedCertificateFromPEMData(pemData)
	return err
}

// GetEncodedCertificateFromPEMData parses the supplied PEM data, extracting the first PEM-encoded block
// and checking that it contains an ASN.1 DER-encoded X.509 certificate. Returns the ASN.1 DER encoded certificate data.
// Returns a Validation Failed error if no X.509 certificate is found.
func GetEncodedCertificateFromPEMData(pemData string) (CertificateData, error) {
	// Find the first block of data in the PEM
	const certificatePEMBlockType = "CERTIFICATE"
	certBlock, _ := pem.Decode([]byte(pemData))
	if certBlock == nil {
		return nil, gerror.NewErrValidationFailed("client certificate PEM data does not contain a valid PEM block")
	}
	if certBlock.Type != certificatePEMBlockType {
		return nil, gerror.NewErrValidationFailed(fmt.Sprintf("Client certificate PEM data contains unknown block type: %s (should be %s)",
			certBlock.Type, certificatePEMBlockType))
	}

	// Check that the certificate data is actually an X.509 certificate
	_, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, gerror.NewErrValidationFailed("client certificate PEM data does not contain an X.509 certificate")
	}

	return certBlock.Bytes, nil
}

// LoadCertificateFromPemFile reads a certificate from a PEM file, and returns the certificate
// as ASN.1 DER encoded binary.
// The first data block in the PEM file must be an ASN.1 DER-encoded X.509 certificate.
func LoadCertificateFromPemFile(file CertificateFile) (CertificateData, error) {
	pemCert, err := ioutil.ReadFile(file.String())
	if err != nil {
		return nil, fmt.Errorf("error loading certificate from PEM file: %w", err)
	}
	return GetEncodedCertificateFromPEMData(string(pemCert))
}

// LoadCertificateFromPemFileAsString reads a certificate from a PEM file, and returns the certificate
// as a PEM-encoded string just as it is stored in the file.
// The data is checked to ensure that it does contain an X.509 certificate.
func LoadCertificateFromPemFileAsString(file CertificateFile) (string, error) {
	pemCert, err := ioutil.ReadFile(file.String())
	if err != nil {
		return "", fmt.Errorf("error loading certificate from PEM file: %w", err)
	}
	_, err = GetEncodedCertificateFromPEMData(string(pemCert))
	if err != nil {
		return "", fmt.Errorf("error decoding certificate from PEM file: %w", err)
	}
	return string(pemCert), nil
}

// GetPublicKeyFromCertificate parses the supplied certificate data to an X.509 certificate and
// extracts the public key, returning it as an ASN.1 DER-encoded public key.
func GetPublicKeyFromCertificate(certData CertificateData) (PublicKeyData, error) {
	// Parse the certificate data to extract public key
	x509Cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return nil, gerror.NewErrValidationFailed("client certificate data does not contain an X.509 certificate")
	}

	// Encode public key as ASN.1 DER
	publicKey, err := x509.MarshalPKIXPublicKey(x509Cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("error: unable to encode public key (algorithm %q) from X.509 certificate: %w",
			x509Cert.PublicKeyAlgorithm, err)
	}

	return publicKey, nil
}

// GetEd25519PublicKeyFromCertificatePEM extracts the public key from the provided PEM-encoded X.509 certificate,
// and checks that it is an ed25519 public key. The public key is returned as an object, ready to be used
// for signature verification.
func GetEd25519PublicKeyFromCertificatePEM(pemData string) (crypto.PublicKey, error) {
	certificateData, err := GetEncodedCertificateFromPEMData(pemData)
	if err != nil {
		return nil, fmt.Errorf("error decoding certificate from PEM file: %w", err)
	}

	x509Cert, err := x509.ParseCertificate(certificateData)
	if err != nil {
		return nil, gerror.NewErrValidationFailed("PEM data does not contain an X.509 certificate")
	}
	publicKey := x509Cert.PublicKey

	if _, ok := publicKey.(ed25519.PublicKey); !ok {
		return nil, errors.New("key is not a valid Ed25519 public key")
	}

	return publicKey, nil
}

// GetEd25519PrivateKeyFromPEM extracts the private key from the provided PEM-encoded data,
// and checks that it is an ed25519 private key.
// The key is returned as an object, ready to be used for signing.
func GetEd25519PrivateKeyFromPEM(pemData string) (crypto.PrivateKey, error) {
	// Find the first block of data in the PEM
	const privateKeyPEMBlockType = "PRIVATE KEY"
	privateKeyBlock, _ := pem.Decode([]byte(pemData))
	if privateKeyBlock == nil {
		return nil, gerror.NewErrValidationFailed("private key PEM data does not contain a valid PEM block")
	}
	if privateKeyBlock.Type != privateKeyPEMBlockType {
		return nil, gerror.NewErrValidationFailed(fmt.Sprintf("Private key PEM data contains unknown block type: %s (should be %s)",
			privateKeyBlock.Type, privateKeyPEMBlockType))
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, gerror.NewErrValidationFailed("error parsing private key").Wrap(err)
	}
	if _, ok := privateKey.(ed25519.PrivateKey); !ok {
		return nil, errors.New("key is not a valid Ed25519 private key")
	}

	return privateKey, nil
}
