package authentication

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type AuthenticationService struct {
	db                *store.DB
	credentialStore   store.CredentialStore
	identityStore     store.IdentityStore
	credentialService services.CredentialService
	syncService       services.SyncService
	logger.Log
}

func NewAuthenticationService(
	db *store.DB,
	credentialStore store.CredentialStore,
	identityStore store.IdentityStore,
	credentialService services.CredentialService,
	syncService services.SyncService,
	logFactory logger.LogFactory,
) *AuthenticationService {
	return &AuthenticationService{
		db:                db,
		credentialStore:   credentialStore,
		identityStore:     identityStore,
		credentialService: credentialService,
		syncService:       syncService,
		Log:               logFactory("AuthenticationService"),
	}
}

// AuthenticateSharedSecret authenticates an identity using a shared secret token.
func (s *AuthenticationService) AuthenticateSharedSecret(ctx context.Context, tokenStr string) (*models.Identity, error) {

	token, err := models.NewPublicSharedSecretTokenFromString(tokenStr)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing token")
	}

	cred, err := s.credentialStore.ReadBySharedSecretID(ctx, nil, token.ID())
	if err != nil {
		return nil, errors.Wrap(err, "error locating credential")
	}

	valid, err := token.IsValid(cred.SharedSecretSalt, cred.SharedSecretDataHashed)
	if err != nil {
		return nil, errors.Wrap(err, "error validating token")
	}
	if !valid {
		return nil, gerror.NewErrUnauthorized("Unauthorized")
	}

	if !cred.IsEnabled {
		return nil, gerror.NewErrAccountDisabled()
	}

	identity, err := s.identityStore.Read(ctx, nil, cred.IdentityID)
	if err != nil {
		return nil, errors.Wrap(err, "error reading identity for credential")
	}

	return identity, nil
}

// AuthenticateSCMAuth authenticates an identity using an SCM as the authentication provider (typically OAuth).
// If the identity does not exist then a new legal entity and identity will automatically be created and authenticated.
func (s *AuthenticationService) AuthenticateSCMAuth(ctx context.Context, auth models.SCMAuth) (*models.Identity, error) {
	// Perform a sync operation against the current user now they are authenticated
	userIdentity, err := s.syncService.SyncAuthenticatedUser(ctx, auth)
	if err != nil {
		return nil, err
	}

	// Return just the bits we're interested in during authentication
	return userIdentity, nil
}

// AuthenticateClientCertificate authenticates an identity using a client certificate.
func (s *AuthenticationService) AuthenticateClientCertificate(
	ctx context.Context,
	certificateData certificates.CertificateData,
) (*models.Identity, error) {

	publicKey, err := certificates.GetPublicKeyFromCertificate(certificateData)
	if err != nil {
		return nil, errors.Wrap(err, "error locating client certificate credential for public key")
	}
	cred, err := s.credentialStore.ReadByPublicKey(ctx, nil, publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "error locating client certificate credential for public key")
	}
	// TODO: Do we need check validity of dates on the certificate?
	if !cred.IsEnabled {
		return nil, gerror.NewErrAccountDisabled()
	}

	identity, err := s.identityStore.Read(ctx, nil, cred.IdentityID)
	if err != nil {
		return nil, errors.Wrap(err, "error reading legal entity")
	}

	return identity, nil
}

// AuthenticateJWT authenticates an identity using a JWT.
func (s *AuthenticationService) AuthenticateJWT(ctx context.Context, jwt string) (*models.Identity, error) {
	// Verify the JWT and extract an Identity ID
	identityID, err := s.credentialService.VerifyIdentityJWT(jwt)
	if err != nil {
		return nil, err
	}

	// Check the identity is in the database
	identity, err := s.identityStore.Read(ctx, nil, identityID)
	if err != nil {
		return nil, fmt.Errorf("error reading legal entity for identity ID specified in JWT: %w", err)
	}

	return identity, nil
}
