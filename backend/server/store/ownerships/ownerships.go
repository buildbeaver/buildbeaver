package ownerships

import (
	"context"
	"reflect"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.Ownership{})
}

type OwnershipStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *OwnershipStore {
	return &OwnershipStore{
		table: store.NewResourceTable(db, logFactory, &models.Ownership{}),
	}
}

// Create a new ownership.
// Returns store.ErrAlreadyExists if an ownership with matching unique properties already exists.
func (d *OwnershipStore) Create(ctx context.Context, txOrNil *store.Tx, ownership *models.Ownership) error {
	return d.table.Create(ctx, txOrNil, ownership)
}

// Read an existing ownership, looking it up by ResourceID.
// Returns models.ErrNotFound if the ownership does not exist.
func (d *OwnershipStore) Read(ctx context.Context, txOrNil *store.Tx, id models.OwnershipID) (*models.Ownership, error) {
	ownership := &models.Ownership{}
	return ownership, d.table.ReadByID(ctx, txOrNil, id.ResourceID, ownership)
}

// readByOwnedResource reads an existing ownership, looking it up by the owned resource.
// Returns models.ErrNotFound if the ownership does not exist.
func (d *OwnershipStore) readByOwnedResource(ctx context.Context, txOrNil *store.Tx, ownedResourceID models.ResourceID) (*models.Ownership, error) {
	ownership := &models.Ownership{}
	return ownership, d.table.ReadWhere(ctx, txOrNil, ownership, goqu.Ex{"access_control_ownership_owned_resource_id": ownedResourceID})
}

// Update an existing ownership with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *OwnershipStore) Update(ctx context.Context, txOrNil *store.Tx, ownership *models.Ownership) error {
	return d.table.UpdateByID(ctx, txOrNil, ownership)
}

// Upsert creates an ownership if it does not exist, otherwise it updates its mutable properties
// if they differ from the in-memory instance. Returns true,false if the resource was created
// and false,true if the resource was updated. false,false if neither a create or update was necessary.
func (d *OwnershipStore) Upsert(ctx context.Context, txOrNil *store.Tx, ownership *models.Ownership) (bool, bool, error) {
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.readByOwnedResource(ctx, tx, ownership.OwnedResourceID)
		}, func(tx *store.Tx) error {
			return d.Create(ctx, tx, ownership)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.Ownership)
			ownership.OwnershipMetadata = existing.OwnershipMetadata
			if reflect.DeepEqual(existing, ownership) {
				return false, nil
			}
			return true, d.Update(ctx, tx, ownership)
		})
}

// Delete permanently and idempotently deletes an ownership, identifying it by owned resource id.
func (d *OwnershipStore) Delete(ctx context.Context, txOrNil *store.Tx, ownedResourceID models.ResourceID) error {
	return d.table.DeleteWhere(ctx, txOrNil,
		goqu.Ex{"access_control_ownership_owned_resource_id": ownedResourceID})
}
