package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/logger"
)

type TLSConfig struct {
	CertificateFile                    certificates.CertificateFile
	PrivateKeyFile                     certificates.PrivateKeyFile
	UseMTLS                            bool
	AutoCreateCertificate              AutoCreateServerCertificate
	AutoCreatedCertificateHost         string
	AutoCreatedCertificateOrganization string
}

type HTTPServerConfig struct {
	Address      string
	TLSConfig    *TLSConfig
	DockerBridge bool // true to listen on docker bridge network if necessary, false to never listen
}

func (c *HTTPServerConfig) GetAddressHost() string {
	if strings.Contains(c.Address, ":") {
		host, _, err := net.SplitHostPort(c.Address)
		if err != nil {
			return ""
		}
		return host
	} else {
		return c.Address // no port to the entire string is the host
	}
}

func (c *HTTPServerConfig) GetAddressPort() string {
	if strings.Contains(c.Address, ":") {
		_, port, err := net.SplitHostPort(c.Address)
		if err != nil {
			return ""
		}
		return port
	} else {
		return "" // no port
	}
}

// AutoCreateServerCertificate is a setting to specify whether to automatically create a key pair and certificate
// for the server if not currently configured.
type AutoCreateServerCertificate bool

func (b AutoCreateServerCertificate) Bool() bool {
	return bool(b)
}

// APIServer is implemented by HTTPServer and HttpAPITestServer
type APIServer interface {
	Start()
	Stop(ctx context.Context) error
	GetServerURL() string
	GetHTTPServer() *http.Server
	GetCertificateFile() certificates.CertificateFile
}

type HTTPServerFactory = func(handler http.Handler, config HTTPServerConfig, log logger.Log) (APIServer, error)

func RealHTTPServerFactory() HTTPServerFactory {
	return func(handler http.Handler, config HTTPServerConfig, log logger.Log) (APIServer, error) {
		return NewHTTPServer(handler, config, log)
	}
}

// HTTPServer is an HTTP(S) server that can serve BuildBeaver API requests.
type HTTPServer struct {
	httpServer         *http.Server
	dockerBridgeServer *http.Server // second server for docker connection to localhost
	config             HTTPServerConfig
	log                logger.Log
}

func NewHTTPServer(
	handler http.Handler,
	config HTTPServerConfig,
	log logger.Log,
) (*HTTPServer, error) {
	httpServer := &http.Server{
		Addr:    config.Address,
		Handler: handler,
	}
	if config.TLSConfig != nil {
		if config.TLSConfig.AutoCreateCertificate.Bool() {
			// Create a self-signed server certificate if we don't have a certificate already configured
			created, err := certificates.GenerateServerSelfSignedCertificate(
				config.TLSConfig.CertificateFile,
				config.TLSConfig.PrivateKeyFile,
				config.TLSConfig.AutoCreatedCertificateHost,
				config.TLSConfig.AutoCreatedCertificateOrganization,
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
		// Create the TLS Config with the CA pool and enable Client certificate validation.
		// Allow an optional client cert to be used to establish a TLS connection, then later
		// ClientCertificateAuthenticator will check that the specific certificate is registered.
		httpServer.TLSConfig = &tls.Config{
			//		ClientCAs: caCertPool,  // use a CA pool to verify CA-issued client certificates
		}
		if config.TLSConfig.UseMTLS {
			httpServer.TLSConfig.ClientAuth = tls.RequestClientCert
		}
	}
	server := &HTTPServer{
		httpServer: httpServer,
		config:     config,
		log:        log,
	}
	server.configureDockerBridgeServer()
	return server, nil
}

// configureDockerBridgeServer configures a second server to listen on the docker bridge network interface
// if needed. The configuration will be set in HTTPServer.dockerBridgeServer.
// The second server is only needed for a local server on Linux, to enable clients running inside a docker
// container to connect to the local server.
func (s *HTTPServer) configureDockerBridgeServer() {
	if !s.config.DockerBridge {
		s.log.Tracef("Skipping configuration of docker bridge network server because config.DockerBridge is false")
		return // this server is configured to never listen on the docker bridge network
	}
	// Only Linux needs a second server on the docker bridge network; docker containers on Windows
	// and Mac can connect directly to localhost via the special 'host.docker.internal' address
	if runtime.GOOS != "linux" {
		s.log.Tracef("Skipping configuration of docker bridge network server because OS is %s, not linux", runtime.GOOS)
		return
	}
	// Only configure a server on the docker bridge network if listening on localhost (or equivalent)
	host := s.config.GetAddressHost()
	if host == "" {
		s.log.Warnf("Skipping configuration of docker bridge network server because server address host is not valid")
		return
	}
	if !dynamic_api.IsLocalhost(host) {
		s.log.Tracef("Skipping configuration of docker bridge network server because host '%s' is not local", host)
		return
	}

	// The docker bridge interface IP address must be determined dynamically
	dockerBridgeIP, err := dynamic_api.GetDockerBridgeInterfaceIPv4Address()
	if err != nil {
		s.log.Warnf("Skipping configuration of docker bridge network server because unable to determine docker bridge IP address to listen on: %s", err.Error())
		return
	}
	dockerBridgeAddr := dockerBridgeIP
	port := s.config.GetAddressPort()
	if port != "" {
		dockerBridgeAddr += ":" + port
	}

	// Configure the second HTTP server
	s.dockerBridgeServer = &http.Server{
		Addr:      dockerBridgeAddr,
		Handler:   s.httpServer.Handler,   // use same handler as the localhost http server
		TLSConfig: s.httpServer.TLSConfig, // use same TLS config as the localhost http server
	}
}

// Start starts listening on the API server HTTP port.
// ListenAndServeTLS is called on a goroutine so this function returns immediately.
func (s *HTTPServer) Start() {
	// Start the main server
	go func() {
		var err error
		if s.config.TLSConfig != nil {
			s.log.Infof("HTTPS listening on %s", s.httpServer.Addr)
			err = s.httpServer.ListenAndServeTLS(s.config.TLSConfig.CertificateFile.String(), s.config.TLSConfig.PrivateKeyFile.String())
		} else {
			s.log.Infof("HTTP listening on %s", s.httpServer.Addr)
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			// If we can't start the main HTTP server then log an error and terminate the process
			s.log.Fatalf("Error starting server: %s", err)
		}
	}()

	// Start an optional second server on the docker bridge network
	if s.dockerBridgeServer != nil {
		go func() {
			var err error
			if s.config.TLSConfig != nil {
				s.log.Infof("HTTPS listening on docker bridge network %s", s.dockerBridgeServer.Addr)
				err = s.dockerBridgeServer.ListenAndServeTLS(s.config.TLSConfig.CertificateFile.String(), s.config.TLSConfig.PrivateKeyFile.String())
			} else {
				s.log.Infof("HTTP listening on docker bridge network %s", s.dockerBridgeServer.Addr)
				err = s.dockerBridgeServer.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				// Log a warning but don't terminate the process if we can't start the docker bridge network server
				s.log.Warnf("Unable to start docker bridge network server: %s", err)
			}
		}()
	}
}

// Stop shuts down the HTTP server that is listening on the API server HTTPS port.
// The server is shut down gracefully, allowing all existing HTTP requests to complete up until a
// timeout period expires.
// Shutdown should only be called once.
func (s *HTTPServer) Stop(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("error shutting down HTTP server: %w", err)
	}
	return nil
}

func (s *HTTPServer) GetServerURL() string {
	if s.config.TLSConfig != nil {
		return fmt.Sprintf("https://%s", s.httpServer.Addr)
	}
	return fmt.Sprintf("http://%s", s.httpServer.Addr)
}

func (s *HTTPServer) GetHTTPServer() *http.Server {
	return s.httpServer
}

func (s *HTTPServer) GetCertificateFile() certificates.CertificateFile {
	if s.config.TLSConfig == nil {
		s.log.Panic("TLS is not enabled")
	}
	return s.config.TLSConfig.CertificateFile
}
