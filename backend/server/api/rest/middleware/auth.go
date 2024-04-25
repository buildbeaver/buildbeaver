package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
)

const authenticationMetaContextKeyName = "authentication" // TODO should be a struct

type AuthenticationMeta struct {
	IdentityID     models.IdentityID
	CredentialType models.CredentialType
	OAuthToken     *oauth2.Token     // TODO Remove me (easy once we don't need this for sync)
	Claims         map[string]string // claim data from the authentication method (especially JWT)
}

// MakeMustAuthenticate makes a middleware that enforces that the request must be authenticated.
// If the request is not authenticated then a 401 error will be returned to the client.
func MakeMustAuthenticate(log logger.Log) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			meta := r.Context().Value(authenticationMetaContextKeyName)
			if meta == nil {
				log.Error(w, r, gerror.NewErrUnauthorized("Unauthorized"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// MakeSharedSecretAuthenticator makes a middleware that authenticates requests using
// a shared secret token from the request headers. If the request headers do not contain
// a token then this a no-op.
func MakeSharedSecretAuthenticator(log logger.Log, authenticationService services.AuthenticationService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("buildbeaver-token")
			if token != "" {
				identity, err := authenticationService.AuthenticateSharedSecret(r.Context(), token)
				if err != nil {
					log.Error(w, r, gerror.NewErrUnauthorized("Invalid shared secret"))
					return
				}
				meta := &AuthenticationMeta{
					IdentityID:     identity.ID,
					CredentialType: models.CredentialTypeSharedSecret,
				}
				ctx := context.WithValue(r.Context(), authenticationMetaContextKeyName, meta)
				r = r.WithContext(ctx)
				log.Infof("Authenticated identity %q using shared secret", identity.ID)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// MakeClientCertificateAuthenticator makes a middleware that authenticates requests using
// a client certificate, requiring it to be one of a registered set of client certificates.
// If no client certificate is provided then this is a no-op.
func MakeClientCertificateAuthenticator(log logger.Log, authenticationService services.AuthenticationService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Check that we are using TLS and the client provided a certificate
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {

				// The first certificate in the list is the 'leaf' certificate whose public key the connection was
				// verified against
				leafClientCert := r.TLS.PeerCertificates[0]

				// Normally another certificate's public key would be used to verify the signature on the client's
				// certificate, but if the certificate is self-signed then it's public key can be used to verify its
				// own signature
				// TODO: Allow a CA to sign the certificate as an alternative to a self-signed cert; a CA-signed cert
				// TODO: would fail the following check.
				err := leafClientCert.CheckSignatureFrom(leafClientCert)
				if err != nil {
					log.Error(w, r, gerror.NewErrUnauthorized("Invalid client certificate").Wrap(
						fmt.Errorf("error verifying that client certificate is self-signed: %w", err)))
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				log.Tracef("Self-signed client certificate found, raw bytes length %d, public key: %v",
					len(leafClientCert.Raw), leafClientCert.PublicKey)

				// Check the client certificate's public key is registered as a credential
				identity, err := authenticationService.AuthenticateClientCertificate(r.Context(), leafClientCert.Raw)
				if err != nil {
					log.Error(w, r, gerror.NewErrUnauthorized("Invalid client certificate").Wrap(
						fmt.Errorf("error authenticating client: %w", err)))
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				meta := &AuthenticationMeta{
					IdentityID:     identity.ID,
					CredentialType: models.CredentialTypeClientCertificate,
				}
				ctx := context.WithValue(r.Context(), authenticationMetaContextKeyName, meta)
				r = r.WithContext(ctx)
				log.Tracef("Authenticated identity %q using client certificate", identity.ID)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// MakeJWTAuthenticator makes a middleware that authenticates requests using a JWT (JSON Web Token) supplied
// by the client, requiring it to be valid and signed by the server.
// If no JWT was provided in the request then this is a no-op.
func MakeJWTAuthenticator(log logger.Log, authenticationService services.AuthenticationService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Check that the client provided a JWT in the Authorization header as a Bearer token
			authHeader := r.Header.Get("Authorization")
			const bearerPrefix = "Bearer "

			if len(authHeader) > len(bearerPrefix) && strings.HasPrefix(strings.ToLower(authHeader), strings.ToLower(bearerPrefix)) {
				token := strings.TrimSpace(authHeader[len(bearerPrefix):])

				// Check the signature on the JWT and read the identity it specifies (must exist in the database)
				identity, err := authenticationService.AuthenticateJWT(r.Context(), token)
				if err != nil {
					log.Error(w, r, gerror.NewErrUnauthorized("Invalid JWT").Wrap(
						fmt.Errorf("error authenticating client: %w", err)))
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				meta := &AuthenticationMeta{
					IdentityID:     identity.ID,
					CredentialType: models.CredentialTypeJWT,
				}
				ctx := context.WithValue(r.Context(), authenticationMetaContextKeyName, meta)
				r = r.WithContext(ctx)
				log.Tracef("Authenticated identity '%s' using JWT", identity.ID)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
