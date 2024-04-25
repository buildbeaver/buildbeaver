package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultPollTimeout  = time.Second * 30
)

type RegistrarConfig struct {
	pollTimeout  time.Duration
	pollInterval time.Duration
}

type Registrar struct {
	client APIClient
	config RegistrarConfig
	log    logger.Log
}

func NewRegistrar(log logger.LogFactory, client APIClient) *Registrar {
	return &Registrar{
		client: client,
		config: RegistrarConfig{pollTimeout: defaultPollTimeout, pollInterval: defaultPollInterval},
		log:    log("Registrar"),
	}
}

func (s *Registrar) Register(ctx context.Context, certificate certificates.CertificateFile, logUnregisteredCert bool) error {
	return s.waitForServerRegistration(ctx, certificate, logUnregisteredCert)
}

// waitForServerRegistration attempts to connect to the server and check whether the runner is correctly registered.
// If the server can't be contacted then this function will retry indefinitely until a connection can be established.
// If the server returns a 200 OK code this function returns with no error, indicating that this runner has been
// correctly registered with the server and is ready to go.
// If the server returns a 401 Unauthorized code and logUnregisteredCert is true then a message will be output asking
// for the runner to be registered with the server, and providing the client certificate to use in registration,
// and connection attempts will continue.
// If the server returns any other error code then the connection will be retried until either a 200 or 403 is returned.
func (s *Registrar) waitForServerRegistration(
	ctx context.Context,
	certificateFile certificates.CertificateFile,
	logUnregisteredCert bool,
) error {
	certStr, err := certificates.LoadCertificateFromPemFileAsString(certificateFile)
	if err != nil {
		return fmt.Errorf("error loading runner certificate from %q: %w", certificateFile, err)
	}
	var helpMessageShown bool
	for ctx.Err() == nil {
		err := s.checkServerRegistration(ctx)
		if err == nil {
			s.log.Infof("Runner is registered with server")
			return nil
		} else {
			if gerror.HasHTTPStatusCode(err, http.StatusUnauthorized) && logUnregisteredCert {
				if !helpMessageShown {
					helpMessageShown = true
					// Print registration message, but carry on so things will 'just work' once registration is complete
					s.log.Infof("\n\nRUNNER REGISTRATION\n"+
						"Runner must be registered with BuildBeaver.\n"+
						"Please browse to the registration screen and cut-and-paste the following Runner Certificate:\n"+
						"\n%s\n"+
						"Waiting for registration...\n",
						certStr,
					)
				}
				s.log.Tracef("Unauthorized error from server: runner is not correctly registered")
			} else {
				s.log.Infof("Retrying error checking connection with server in %s: %v", s.config.pollInterval, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.pollInterval):
			}
		}
	}
	return ctx.Err()
}

func (s *Registrar) checkServerRegistration(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.pollTimeout)
	defer cancel()
	return s.client.Ping(ctx)
}
