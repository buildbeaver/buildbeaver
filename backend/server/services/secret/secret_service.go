package secret

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type SecretService struct {
	db                *store.DB
	secretStore       store.SecretStore
	ownershipStore    store.OwnershipStore
	resourceLinkStore store.ResourceLinkStore
	encryptionService services.EncryptionService
	logger.Log
}

func NewSecretService(
	db *store.DB,
	secretStore store.SecretStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	encryptionService services.EncryptionService,
	logFactory logger.LogFactory) *SecretService {

	return &SecretService{
		db:                db,
		secretStore:       secretStore,
		ownershipStore:    ownershipStore,
		resourceLinkStore: resourceLinkStore,
		encryptionService: encryptionService,
		Log:               logFactory("SecretService"),
	}
}

// Create a new secret.
// Returns store.ErrAlreadyExists if a secret with matching unique properties already exists.
func (s *SecretService) Create(
	ctx context.Context,
	txOrNil *store.Tx,
	repoID models.RepoID,
	keyPlaintext string,
	valuePlaintext string,
	isInternal bool) (*models.SecretPlaintext, error) {

	partsEncrypted, dataKeyEncrypted, err := s.encryptionService.EncryptMulti(ctx, []byte(keyPlaintext), []byte(valuePlaintext))
	if err != nil {
		return nil, errors.Wrap(err, "error encrypting secret parts")
	}
	name, err := s.makeSecretName(keyPlaintext)
	if err != nil {
		return nil, fmt.Errorf("error making secret name: %w", err)
	}
	now := models.NewTime(time.Now())
	secret := models.NewSecret(now, name, repoID, partsEncrypted[0], partsEncrypted[1], dataKeyEncrypted, isInternal)
	err = secret.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating secret: %w", err)
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err = s.secretStore.Create(ctx, tx, secret)
		if err != nil {
			return errors.Wrap(err, "error creating secret")
		}
		ownership := models.NewOwnership(now, repoID.ResourceID, secret.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, secret)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Created secret %q", secret.ID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	plaintext := &models.SecretPlaintext{
		Key:    keyPlaintext,
		Value:  valuePlaintext,
		Secret: secret,
	}
	return plaintext, nil
}

// Read an existing secret, looking it up by ID.
// Returns models.ErrNotFound if the secret does not exist.
func (s *SecretService) Read(ctx context.Context, txOrNil *store.Tx, id models.SecretID) (*models.Secret, error) {
	return s.secretStore.Read(ctx, txOrNil, id)
}

// Update an existing secret with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *SecretService) Update(ctx context.Context, txOrNil *store.Tx, secret *models.Secret) (*models.Secret, error) {
	err := secret.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating secret: %w", err)
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.secretStore.Update(ctx, tx, secret)
		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, secret)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// UpdatePlaintext updates an existing secret's plaintext key and value with optimistic locking.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *SecretService) UpdatePlaintext(ctx context.Context, txOrNil *store.Tx, secretID models.SecretID, update dto.UpdateSecretPlaintext) (*models.SecretPlaintext, error) {
	secret, err := s.secretStore.Read(ctx, txOrNil, secretID)
	if err != nil {
		return nil, fmt.Errorf("error reading secret")
	}
	plaintext, err := s.SecretToSecretPlaintext(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("error decrypting secret")
	}
	if update.KeyPlaintext != nil {
		plaintext.Key = *update.KeyPlaintext
	}
	if update.ValuePlaintext != nil {
		plaintext.Value = *update.ValuePlaintext
	}
	partsEncrypted, dataKeyEncrypted, err := s.encryptionService.EncryptMulti(ctx, []byte(plaintext.Key), []byte(plaintext.Value))
	if err != nil {
		return nil, fmt.Errorf("error encrypting secret parts: %w", err)
	}
	name, err := s.makeSecretName(plaintext.Key)
	if err != nil {
		return nil, fmt.Errorf("error making secret name: %w", err)
	}
	secret.UpdatedAt = models.NewTime(time.Now())
	secret.Name = name
	secret.KeyEncrypted = partsEncrypted[0]
	secret.ValueEncrypted = partsEncrypted[1]
	secret.DataKeyEncrypted = dataKeyEncrypted
	secret.ETag = models.GetETag(secret, update.ETag)
	err = secret.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating secret: %w", err)
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.secretStore.Update(ctx, tx, secret)
		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, secret)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	plaintext, err = s.SecretToSecretPlaintext(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("error decrypting secret after update")
	}
	return plaintext, nil
}

// Delete permanently and idempotently deletes a secret, identifying it by ID.
func (s *SecretService) Delete(ctx context.Context, txOrNil *store.Tx, secretID models.SecretID) error {
	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.secretStore.Delete(ctx, tx, secretID)
		if err != nil {
			return fmt.Errorf("error deleting secret: %w", err)
		}
		err = s.ownershipStore.Delete(ctx, tx, secretID.ResourceID)
		if err != nil {
			return fmt.Errorf("error deleting ownership: %w", err)
		}
		err = s.resourceLinkStore.Delete(ctx, tx, secretID.ResourceID)
		if err != nil {
			return fmt.Errorf("error deleting resource link: %w", err)
		}
		return nil
	})
	return err
}

// ListByRepoID gets all secrets that are associated with the specified repo id.
func (s *SecretService) ListByRepoID(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID, pagination models.Pagination) ([]*models.Secret, *models.Cursor, error) {
	return s.secretStore.ListByRepoID(ctx, txOrNil, repoID, pagination)
}

// ListPlaintextByRepoID gets all secrets in plaintext that are associated with the specified repo id.
func (s *SecretService) ListPlaintextByRepoID(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID, pagination models.Pagination) ([]*models.SecretPlaintext, *models.Cursor, error) {
	secrets, cursor, err := s.ListByRepoID(ctx, txOrNil, repoID, pagination)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error listing secrets")
	}
	plaintext, err := s.secretsToSecretsPlaintext(ctx, secrets)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error converting secrets to plaintext")
	}
	return plaintext, cursor, nil
}

// makeSecretName makes a name for a secret based on its key.
// This must be recalculated whenever the key changes.
func (s *SecretService) makeSecretName(keyPlaintext string) (models.ResourceName, error) {
	// We don't want to use the secret key as our name as that would store it in plaintext.
	// Instead we generate a hash based on the secret's name and use that. The hash is deterministic
	// which enforces uniqueness constraints on the secret key (via the database).
	hash := sha256.New()
	_, err := hash.Write([]byte(keyPlaintext))
	if err != nil {
		return "", fmt.Errorf("error hashing secret name: %w", err)
	}
	return models.ResourceName(hex.EncodeToString(hash.Sum(nil)[:18])), nil
}

// secretsToSecretsPlaintext converts a slice of secrets to a slice of plaintext secrets.
func (s *SecretService) secretsToSecretsPlaintext(ctx context.Context, secrets []*models.Secret) ([]*models.SecretPlaintext, error) {
	plaintext := make([]*models.SecretPlaintext, len(secrets))
	for i := 0; i < len(secrets); i++ {
		pt, err := s.SecretToSecretPlaintext(ctx, secrets[i])
		if err != nil {
			return nil, err
		}
		plaintext[i] = pt
	}
	return plaintext, nil
}

// SecretToSecretPlaintext converts a secret to a plaintext secret.
func (s *SecretService) SecretToSecretPlaintext(ctx context.Context, secret *models.Secret) (*models.SecretPlaintext, error) {
	name, value, err := s.getSecretValuePlaintext(ctx, secret)
	if err != nil {
		return nil, err
	}
	return &models.SecretPlaintext{
		Key:    name,
		Value:  value,
		Secret: secret,
	}, nil
}

// GetSecretValuePlaintext decrypts the encrypted secret value and returns it in plaintext.
func (s *SecretService) getSecretValuePlaintext(ctx context.Context, secret *models.Secret) (string, string, error) {
	namePlaintext, err := s.encryptionService.Decrypt(ctx, secret.KeyEncrypted, secret.DataKeyEncrypted)
	if err != nil {
		return "", "", errors.Wrap(err, "error decrypting name")
	}
	valuePlaintext, err := s.encryptionService.Decrypt(ctx, secret.ValueEncrypted, secret.DataKeyEncrypted)
	if err != nil {
		return "", "", errors.Wrap(err, "error decrypting secret")
	}
	return string(namePlaintext), string(valuePlaintext), nil
}
