package servertest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
)

func HTTPTestServerFactory() server.HTTPServerFactory {
	return func(handler http.Handler, config server.HTTPServerConfig, log logger.Log) (server.APIServer, error) {
		return NewHTTPTestServer(handler, config, log)
	}
}

// HTTPTestServer is an HTTP(S) test server that can serve BuildBeaver API requests.
// The HTTPTestServer is created using the Go httptest package, and will run on a random port.
type HTTPTestServer struct {
	testServer *httptest.Server
	config     server.HTTPServerConfig
	log        logger.Log
}

func NewHTTPTestServer(
	handler http.Handler,
	config server.HTTPServerConfig,
	log logger.Log,
) (*HTTPTestServer, error) {

	testServer := httptest.NewUnstartedServer(handler)

	if config.TLSConfig != nil {
		if config.TLSConfig.AutoCreateCertificate.Bool() {
			// Create a self-signed server certificate if we don't have a certificate already configured
			created, err := certificates.GenerateServerSelfSignedCertificate(
				config.TLSConfig.CertificateFile,
				config.TLSConfig.PrivateKeyFile,
				"localhost,127.0.0.1,[::1]",
				"BuildBeaver Limited",
			)
			if err != nil {
				return nil, fmt.Errorf("error ensuring server certificate exists: %w", err)
			}
			if created {
				log.Infof("Created private key file and certificate for server")
			} else {
				log.Infof("Found private key file and server certificate for server")
			}
		}
		// Load the key pair for use in the test server
		cert, err := tls.LoadX509KeyPair(config.TLSConfig.CertificateFile.String(), config.TLSConfig.PrivateKeyFile.String())
		if err != nil {
			return nil, err
		}
		// Create the TLS Config with the CA pool and enable Client certificate validation.
		// Allow an optional client cert to be used to establish a TLS connection, then later
		// ClientCertificateAuthenticator will check that the specific certificate is registered.
		// Specify the server certificate in the config to override the default test server behaviour.
		testServer.TLS = &tls.Config{
			//		ClientCAs: caCertPool,  // use a CA pool to verify CA-issued client certificates
			Certificates: []tls.Certificate{cert},
		}
		if config.TLSConfig.UseMTLS {
			testServer.TLS.ClientAuth = tls.RequestClientCert
		}
	}

	return &HTTPTestServer{
		testServer: testServer,
		config:     config,
		log:        log,
	}, nil
}

// Start starts listening on the API server HTTPS port.
// The server is started on a goroutine so this function returns immediately.
func (s *HTTPTestServer) Start() {
	if s.config.TLSConfig != nil {
		s.log.Infof("HTTPS listening on URL %s", s.GetServerURL())
		s.testServer.StartTLS()
	} else {
		s.log.Infof("HTTP listening on URL %s", s.GetServerURL())
		s.testServer.Start()
	}
}

// Stop shuts down the HTTP server that is listening on the API server HTTPS port.
// The server is shut down gracefully, allowing all existing HTTP requests to complete up until a
// timeout period expires.
// Shutdown should only be called once.
func (s *HTTPTestServer) Stop(ctx context.Context) error {
	s.testServer.Close()
	return nil
}

func (s *HTTPTestServer) GetServerURL() string {
	return s.testServer.URL
}

func (s *HTTPTestServer) GetHTTPServer() *http.Server {
	return s.testServer.Config
}

func (s *HTTPTestServer) GetCertificateFile() certificates.CertificateFile {
	if s.config.TLSConfig == nil {
		s.log.Panic("TLS is not enabled")
	}
	return s.config.TLSConfig.CertificateFile
}
