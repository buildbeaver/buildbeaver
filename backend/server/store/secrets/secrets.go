package secrets

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/server/store"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

func init() {
	_ = models.MutableResource(&models.Secret{})
	store.MustDBModel(&models.Secret{})
}

type SecretStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *SecretStore {
	return &SecretStore{
		table: store.NewResourceTable(db, logFactory, &models.Secret{}),
	}
}

// Create a new secret.
// Returns store.ErrAlreadyExists if a secret with matching unique properties already exists.
func (d *SecretStore) Create(ctx context.Context, txOrNil *store.Tx, secret *models.Secret) error {
	return d.table.Create(ctx, txOrNil, secret)
}

// Read an existing secret, looking it up by ResourceID.
// Returns models.ErrNotFound if the secret does not exist.
func (d *SecretStore) Read(ctx context.Context, txOrNil *store.Tx, id models.SecretID) (*models.Secret, error) {
	secret := &models.Secret{}
	return secret, d.table.ReadByID(ctx, txOrNil, id.ResourceID, secret)
}

// Update an existing secret with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *SecretStore) Update(ctx context.Context, txOrNil *store.Tx, secret *models.Secret) error {
	return d.table.UpdateByID(ctx, txOrNil, secret)
}

// Delete permanently and idempotently deletes a secret, identifying it by id.
func (d *SecretStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.SecretID) error {
	return d.table.DeleteByID(ctx, txOrNil, id.ResourceID)
}

// ListByRepoID lists all secrets for a repo. Use cursor to page through results, if any.
func (d *SecretStore) ListByRepoID(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID, pagination models.Pagination) ([]*models.Secret, *models.Cursor, error) {
	secretsSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Secret{}).
		Where(goqu.Ex{"secret_repo_id": repoID})

	var secrets []*models.Secret
	cursor, err := d.table.ListIn(ctx, txOrNil, &secrets, pagination, secretsSelect)
	if err != nil {
		return nil, nil, err
	}
	return secrets, cursor, nil
}
