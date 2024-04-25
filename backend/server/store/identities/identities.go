package identities

import (
	"context"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.Identity{})
}

type IdentityStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *IdentityStore {
	return &IdentityStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "identities", &models.Identity{}),
	}
}

// Create a new Identity.
// Returns store.ErrAlreadyExists if an Identity with matching ID already exists.
func (d *IdentityStore) Create(ctx context.Context, txOrNil *store.Tx, identity *models.Identity) error {
	return d.table.Create(ctx, txOrNil, identity)
}

// Read an existing Identity, looking it up by IdentityID.
// Returns models.ErrNotFound if the Identity does not exist.
func (d *IdentityStore) Read(ctx context.Context, txOrNil *store.Tx, id models.IdentityID) (*models.Identity, error) {
	Identity := &models.Identity{}
	return Identity, d.table.ReadByID(ctx, txOrNil, id.ResourceID, Identity)
}

// ReadByOwnerResource reads the Identity for an owner resource (e.g. a Legal Entity).
// Returns models.ErrNotFound if no Identity is associated with the specified resource.
func (d *IdentityStore) ReadByOwnerResource(ctx context.Context, txOrNil *store.Tx, ownerResourceID models.ResourceID) (*models.Identity, error) {
	identity := &models.Identity{}
	err := d.table.ReadWhere(ctx, txOrNil, identity, goqu.Ex{"identity_owner_resource_id": ownerResourceID})
	if err != nil {
		return nil, fmt.Errorf("error reading Identity for %s: %w", ownerResourceID, err)
	}

	return identity, nil
}

// FindOrCreateByOwnerResource creates an identity if no identity already exists for the specified owner resource,
// otherwise it reads and returns the existing identity.
// Returns the new or existing identity, and true iff a new identity was created.
func (d *IdentityStore) FindOrCreateByOwnerResource(ctx context.Context, txOrNil *store.Tx, ownerResourceID models.ResourceID) (result *models.Identity, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByOwnerResource(ctx, tx, ownerResourceID)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			newIdentity := models.NewIdentity(models.NewTime(time.Now()), ownerResourceID)
			err := d.Create(ctx, tx, newIdentity)
			if err != nil {
				return nil, err
			}
			return newIdentity, nil
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.Identity), created, nil
}

// Delete permanently and idempotently deletes an Identity.
func (d *IdentityStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.IdentityID) error {
	return d.table.DeleteWhere(ctx, txOrNil,
		goqu.Ex{"identity_id": id})
}

// DeleteByOwnerResource permanently and idempotently deletes the Identity with the specified owner resource (if any).
func (d *IdentityStore) DeleteByOwnerResource(ctx context.Context, txOrNil *store.Tx, ownerResourceID models.ResourceID) error {
	return d.table.DeleteWhere(ctx, txOrNil,
		goqu.Ex{"identity_owner_resource_id": ownerResourceID})
}
