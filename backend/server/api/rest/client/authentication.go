package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
)

type SharedSecretToken string

// AutoCreateCertificate is a setting to specify whether to automatically create a key pair and certificate
// for client certificate authentication, if not currently configured.
type AutoCreateCertificate bool

func (b AutoCreateCertificate) Bool() bool {
	return bool(b)
}

// InsecureSkipVerify is a debug setting to prevent the client from validating the server's certificate
// when connecting via HTTPS. Required when the server is using a self-signed certificate and the client is
// not configured to use this certificate as a CA.
type InsecureSkipVerify bool

func (b InsecureSkipVerify) Bool() bool {
	return bool(b)
}

// Authenticator enables the API client to make authenticated API requests
// using pluggable authentication methods.
type Authenticator interface {
	AuthenticateRequest(h http.Header) (http.Header, error)
	// AuthenticateClient is called after an HTTP client is set up for the API. Allows the authenticator to set
	// properties (e.g. certificates or CAs) for authenticating the TLS connection.
	AuthenticateClient(client *retryablehttp.Client) (*retryablehttp.Client, error)
}

// SharedSecretAuthenticator authenticates API client requests using a shared secret.
type SharedSecretAuthenticator struct {
	token string
	logger.Log
}

func NewSharedSecretAuthenticator(token SharedSecretToken, logFactory logger.LogFactory) *SharedSecretAuthenticator {
	return &SharedSecretAuthenticator{
		token: string(token),
		Log:   logFactory("ClientSharedSecretAuthenticator"),
	}
}

func (a *SharedSecretAuthenticator) AuthenticateClient(client *retryablehttp.Client) (*retryablehttp.Client, error) {
	return client, nil
}

func (a *SharedSecretAuthenticator) AuthenticateRequest(h http.Header) (http.Header, error) {
	h.Add("buildbeaver-token", a.token)
	return h, nil
}

// ClientCertificateAuthenticator authenticates API client communications using a client certificate and
// mutual TLS.
type ClientCertificateAuthenticator struct {
	ClientCert       *tls.Certificate // includes private key
	ServerCACertPool *x509.CertPool
	// InsecureSkipVerify is true to accept any server certificate without verifying
	InsecureSkipVerify InsecureSkipVerify
	logger.Log
}

// NewClientCertificateAuthenticator creates an Authenticator that supports communications using a client
// certificate and mutual TLS.
//
// clientCertificateFile is the full path to a PEM file containing the client certificate to present to the
// server for authentication. The public key in the certificate must match the supplied private key.
//
// privateKeyFile is the full path to a PEM file containing the private key to use during TLS negotiation.
//
// If autoCreateCertificate is true and the clientCertificateFile or privateKeyFile are missing then a new
// key pair and self-signed certificate will automatically be created and used.
//
// serverCACertPem (optional) is the full path to a PEM file containing one or more CA certificates to use
// for verifying the server's SSL certificate. If the server is using a self-signed certificate then this
// can be provided as the CA cert.
//
// insecureSkipVerify is true if the client should not verify the server's certificate during TLS negotiation.
// This allows connection to a server that is using a self-signed server certificate, without needing to provide
// the server's certificate as a CA certificate. This option is provided for development and testing only.
func NewClientCertificateAuthenticator(
	clientCertificateFile certificates.CertificateFile,
	privateKeyFile certificates.PrivateKeyFile,
	autoCreateCertificate AutoCreateCertificate,
	caCertFile certificates.CACertificateFile,
	insecureSkipVerify InsecureSkipVerify,
	logFactory logger.LogFactory,
) (*ClientCertificateAuthenticator, error) {
	logger := logFactory("ClientCertificateAuthenticator")

	if autoCreateCertificate {
		created, err := certificates.GenerateClientSelfSignedCertificate(
			clientCertificateFile,
			privateKeyFile,
			"BuildBeaver Limited",
		)
		if err != nil {
			return nil, err
		}
		if created {
			logger.Infof("Created private/public key pair and certificate for client certificate authentication")
		} else {
			logger.Infof("Loading existing private key file and certificate for client certificate authentication")
		}
	}

	// Load the client certificate key pair
	clientCert, err := tls.LoadX509KeyPair(clientCertificateFile.String(), privateKeyFile.String())
	if err != nil {
		return nil, fmt.Errorf("error loading x509 key pair for runner: %w", err)
	}

	// Load CA certificate(s) if provided
	caCertPool := x509.NewCertPool()
	if caCertFile != "" {
		caCertPem, err := ioutil.ReadFile(caCertFile.String())
		if err != nil {
			// If there is no CA certificate file just log a warning and continue
			logger.Warnf("Unable to load CA certificate: %v", err)
		}
		caCertPool.AppendCertsFromPEM(caCertPem)
	}

	if insecureSkipVerify {
		logger.Warnf("Warning: insecure_skip_verify set; API client will not verify server certificate")
	}

	return &ClientCertificateAuthenticator{
		ClientCert:         &clientCert,
		ServerCACertPool:   caCertPool,
		InsecureSkipVerify: insecureSkipVerify,
		Log:                logger,
	}, nil
}

func (a *ClientCertificateAuthenticator) AuthenticateClient(client *retryablehttp.Client) (*retryablehttp.Client, error) {
	// Make sure the HTTP client has an explicitly defined Transport object to set parameters against
	if client.HTTPClient.Transport == nil {
		client.HTTPClient.Transport = &http.Transport{}
	}
	transport := client.HTTPClient.Transport.(*http.Transport)

	// Set TLS config of transport to use our certificates when negotiating a TLS connection.
	// Leave any other HTTPClient properties unchanged.
	transport.TLSClientConfig = &tls.Config{
		Certificates:       []tls.Certificate{*a.ClientCert},
		RootCAs:            a.ServerCACertPool,
		InsecureSkipVerify: a.InsecureSkipVerify.Bool(),
	}
	return client, nil
}

func (a *ClientCertificateAuthenticator) AuthenticateRequest(h http.Header) (http.Header, error) {
	return h, nil
}
