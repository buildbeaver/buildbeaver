package credential

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// AutoCreateJWTSigningKeyPair is a setting to specify whether to automatically create a key pair and certificate
// for signing and verifying JWT tokens, if not currently configured.
type AutoCreateJWTSigningKeyPair bool

func (b AutoCreateJWTSigningKeyPair) Bool() bool {
	return bool(b)
}

type JWTConfig struct {
	CertificateFile   certificates.CertificateFile
	PrivateKeyFile    certificates.PrivateKeyFile
	AutoCreateKeyPair AutoCreateJWTSigningKeyPair
}

type CredentialService struct {
	db                    *store.DB
	ownershipStore        store.OwnershipStore
	credentialStore       store.CredentialStore
	jwtSigningPrivateKey  crypto.PrivateKey
	jwtVerifyingPublicKey crypto.PublicKey
	logger.Log
}

func NewCredentialService(
	db *store.DB,
	jwtConfig JWTConfig,
	ownershipStore store.OwnershipStore,
	credentialStore store.CredentialStore,
	logFactory logger.LogFactory,
) (*CredentialService, error) {
	s := &CredentialService{
		db:              db,
		ownershipStore:  ownershipStore,
		credentialStore: credentialStore,
		Log:             logFactory("CredentialService"),
	}

	err := s.findOrCreateJWTKeyPair(jwtConfig)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// findOrCreateJWTKeyPair loads the public and private keys used for JWT verification and signing, storing
// them in the jwtSigningPrivateKey and jwtVerifyingPublicKey variables.
// If jwtConfig.AutoCreateKeyPair is true and no files exist at the locations specified in the config then
// a new key pair will be generated and stored.
func (s *CredentialService) findOrCreateJWTKeyPair(jwtConfig JWTConfig) error {
	if jwtConfig.AutoCreateKeyPair {
		created, err := certificates.GenerateEd25519SigningKeyAndCertificate(
			jwtConfig.CertificateFile,
			jwtConfig.PrivateKeyFile,
			"BuildBeaver Limited",
		)
		if err != nil {
			return err
		}
		if created {
			s.Infof("Created private/public key pair for JWT signing and verification")
		} else {
			s.Infof("Loading existing private key file and public key certificate for JWT signing and verification")
		}
	}

	privateKeyPEMBlock, err := os.ReadFile(jwtConfig.PrivateKeyFile.String())
	if err != nil {
		return fmt.Errorf("error loading JWT signing private key: %w", err)
	}
	privateKey, err := certificates.GetEd25519PrivateKeyFromPEM(string(privateKeyPEMBlock))
	if err != nil {
		return fmt.Errorf("error reading JWT signing private key from PEM file data: %w", err)
	}

	certPEMBlock, err := os.ReadFile(jwtConfig.CertificateFile.String())
	if err != nil {
		return fmt.Errorf("error loading JWT verification public key certificate: %w", err)
	}
	publicKey, err := certificates.GetEd25519PublicKeyFromCertificatePEM(string(certPEMBlock))
	if err != nil {
		return fmt.Errorf("error reading JWT verification public key from PEM file data: %w", err)
	}

	s.jwtSigningPrivateKey = privateKey
	s.jwtVerifyingPublicKey = publicKey
	return nil
}

// Create a new credential.
func (s *CredentialService) Create(ctx context.Context, txOrNil *store.Tx, credential *models.Credential) error {

	now := time.Now().UTC()

	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {

		err := s.credentialStore.Create(ctx, tx, credential)
		if err != nil {
			return errors.Wrap(err, "error creating credential")
		}

		ownership := models.NewOwnership(models.NewTime(now), credential.IdentityID.ResourceID, credential.GetID())

		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// CreateSharedSecretCredential creates a new shared secret credential for the specified identity.
// The plaintext shared secret value is returned.
// The plaintext can never be reconstructed so do something useful with it now.
func (s *CredentialService) CreateSharedSecretCredential(
	ctx context.Context,
	txOrNil *store.Tx,
	identityID models.IdentityID,
	enabled bool,
) (models.PublicSharedSecretToken, *models.Credential, error) {

	token, err := models.NewSharedSecretToken()
	if err != nil {
		return models.PublicSharedSecretToken{}, nil, fmt.Errorf("error making new shared secret token: %w", err)
	}

	now := models.NewTime(time.Now().UTC())
	credential := models.NewSharedSecretCredential(now, identityID, enabled, token)

	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.credentialStore.Create(ctx, tx, credential)
		if err != nil {
			return errors.Wrap(err, "error creating credential")
		}
		ownership := models.NewOwnership(now, identityID.ResourceID, credential.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		return nil
	})
	if err != nil {
		return models.PublicSharedSecretToken{}, nil, err
	}

	return token.Public(), credential, nil
}

// CreateClientCertificateCredential creates a new client certificate credential for the specified identity.
// clientCert is an X.509 certificate used to identify the client.
func (s *CredentialService) CreateClientCertificateCredential(
	ctx context.Context,
	txOrNil *store.Tx,
	identityID models.IdentityID,
	enabled bool,
	clientCert certificates.CertificateData,
) (*models.Credential, error) {

	now := models.NewTime(time.Now().UTC())
	credential, err := models.NewClientCertificateCredential(now, identityID, enabled, clientCert)
	if err != nil {
		return nil, fmt.Errorf("error creating credential from client certificate: %w", err)
	}

	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.credentialStore.Create(ctx, tx, credential)
		if err != nil {
			return errors.Wrap(err, "error creating credential")
		}
		ownership := models.NewOwnership(now, identityID.ResourceID, credential.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return credential, nil
}

// CreateIdentityJWT creates a new JWT (JSON Web Token) credential that can be used to authenticate as
// the specified identity.
// The JWT will not be stored but can be verified by calling VerifyIdentityJWT, or by any third
// party using the server's JWT public key.
func (s *CredentialService) CreateIdentityJWT(identityID models.IdentityID) (string, error) {
	jwt, claims, err := CreateIdentityJWT(identityID, DefaultJWTIssuer, DefaultJWTExpiryDuration, s.jwtSigningPrivateKey)
	if err != nil {
		return "", err
	}
	s.Infof("Created JWT token for subject '%s', expires at '%v'", claims.Subject, claims.ExpiresAt)
	return jwt, nil
}

// CreateIdentityJWTWithExpiry creates a new JWT (JSON Web Token) credential that can be used to authenticate as
// the specified identity, expiring after the specified duration.
// The JWT will not be stored but can be verified by calling VerifyIdentityJWT, or by any third
// party using the server's JWT public key.
func (s *CredentialService) CreateIdentityJWTWithExpiry(identityID models.IdentityID, expiry time.Duration) (string, error) {
	jwt, claims, err := CreateIdentityJWT(identityID, DefaultJWTIssuer, expiry, s.jwtSigningPrivateKey)
	if err != nil {
		return "", err
	}
	s.Infof("Created JWT token for subject '%s', custom expiry at '%v'", claims.Subject, claims.ExpiresAt)
	return jwt, nil
}

// VerifyIdentityJWT verifies the signature on the supplied JWT (JSON Web Token) and returns the identity ID
// specified in the subject field. The identity ID is NOT checked against the database.
func (s *CredentialService) VerifyIdentityJWT(tokenStr string) (models.IdentityID, error) {
	return VerifyIdentityJWT(tokenStr, s.jwtVerifyingPublicKey)
}

// Delete permanently and idempotently deletes a credential.
func (d *CredentialService) Delete(ctx context.Context, txOrNil *store.Tx, id models.CredentialID) error {
	return d.credentialStore.Delete(ctx, txOrNil, id)
}

// ListCredentialsForIdentity returns a list of all credentials for the specified identity ID.
// Use cursor to page through results, if any.
func (s *CredentialService) ListCredentialsForIdentity(
	ctx context.Context,
	txOrNil *store.Tx,
	identityID models.IdentityID,
	pagination models.Pagination,
) ([]*models.Credential, *models.Cursor, error) {
	return s.credentialStore.ListCredentialsForIdentity(ctx, txOrNil, identityID, pagination)
}
