package legal_entities

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	_ = models.MutableResource(&models.LegalEntity{})
	_ = models.SoftDeletableResource(&models.LegalEntity{})
	store.MustDBModel(&models.LegalEntity{})
}

type LegalEntityStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *LegalEntityStore {
	return &LegalEntityStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "legal_entities", &models.LegalEntity{}),
	}
}

// Create a new legal entity.
// Returns store.ErrAlreadyExists if a legal entity with matching unique properties already exists.
func (d *LegalEntityStore) Create(ctx context.Context, txOrNil *store.Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, error) {
	now := models.NewTime(time.Now())
	legalEntity := &models.LegalEntity{
		LegalEntityData: *legalEntityData,
		LegalEntityMetadata: models.LegalEntityMetadata{
			ID:        models.NewLegalEntityID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := d.table.Create(ctx, txOrNil, legalEntity)
	if err != nil {
		return nil, err
	}

	return legalEntity, nil
}

// Read an existing legal entity, looking it up by ResourceID.
// Returns models.ErrNotFound if the legal entity does not exist.
func (d *LegalEntityStore) Read(ctx context.Context, txOrNil *store.Tx, id models.LegalEntityID) (*models.LegalEntity, error) {
	legalEntity := &models.LegalEntity{}
	return legalEntity, d.table.ReadByID(ctx, txOrNil, id.ResourceID, legalEntity)
}

// ReadByName reads an existing legal entity, looking it up by its name.
// Returns models.ErrNotFound if the legal entity does not exist.
func (d *LegalEntityStore) ReadByName(ctx context.Context, txOrNil *store.Tx, name models.ResourceName) (*models.LegalEntity, error) {
	legalEntity := &models.LegalEntity{}
	return legalEntity, d.table.ReadWhere(ctx, txOrNil, legalEntity, goqu.Ex{"legal_entity_name": name})
}

// ReadByExternalID reads an existing legal entity, looking it up by its external id.
// Returns models.ErrNotFound if the legal entity does not exist.
func (d *LegalEntityStore) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.LegalEntity, error) {
	legalEntity := &models.LegalEntity{}
	return legalEntity, d.table.ReadWhere(ctx, txOrNil, legalEntity, goqu.Ex{"legal_entity_external_id": externalID})
}

// FindOrCreate creates a legal entity if no legal entity with the same External ID already exists,
// otherwise it reads and returns the existing legal entity.
// Returns the legal entity as it is in the database, and true iff a new legal entity was created.
func (d *LegalEntityStore) FindOrCreate(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityData *models.LegalEntityData,
) (result *models.LegalEntity, created bool, err error) {
	if legalEntityData.ExternalID == nil {
		return nil, false, fmt.Errorf("error external id must be set to call FindOrCreate")
	}
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByExternalID(ctx, tx, *legalEntityData.ExternalID)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.Create(ctx, tx, legalEntityData)
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.LegalEntity), created, nil
}

// Update an existing legal entity with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *LegalEntityStore) Update(ctx context.Context, txOrNil *store.Tx, legalEntity *models.LegalEntity) error {
	return d.table.UpdateByID(ctx, txOrNil, legalEntity)
}

// Upsert creates a legal entity if no legal entity with the same External ID already exists, otherwise it updates
// the existing legal entity's data if it differs from the supplied data.
// Returns the LegalEntity as it exists in the database after the create or update, and
// true,false if the resource was created, false,true if the resource was updated, or false,false if
// neither create nor update was necessary.
func (d *LegalEntityStore) Upsert(ctx context.Context, txOrNil *store.Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, bool, bool, error) {
	if legalEntityData.ExternalID == nil {
		return nil, false, false, fmt.Errorf("error external id must be set to upsert")
	}
	var legalEntity *models.LegalEntity
	created, updated, err := d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			var err error
			legalEntity, err = d.ReadByExternalID(ctx, tx, *legalEntityData.ExternalID)
			return legalEntity, err
		}, func(tx *store.Tx) error {
			var err error
			legalEntity, err = d.Create(ctx, tx, legalEntityData)
			return err
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.LegalEntity)
			if reflect.DeepEqual(existing.LegalEntityData, legalEntityData) {
				return false, nil
			}
			// Update all data but none of the metadata
			existing.LegalEntityData = *legalEntityData
			return true, d.Update(ctx, tx, existing)
		})
	if err != nil {
		return nil, false, false, err
	}

	return legalEntity, created, updated, nil
}

// ListParentLegalEntities lists all legal entities a legal entity is a member of. Use cursor to page through results, if any.
func (d *LegalEntityStore) ListParentLegalEntities(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityID models.LegalEntityID,
	pagination models.Pagination,
) ([]*models.LegalEntity, *models.Cursor, error) {
	legalEntitiesSelect := goqu.
		From(d.table.TableName()).
		Select(&models.LegalEntity{}).
		Join(goqu.T("legal_entities_memberships"),
			goqu.On(goqu.Ex{"legal_entities.legal_entity_id": goqu.I("legal_entities_memberships.legal_entities_membership_legal_entity_id")})).
		Where(goqu.Ex{"legal_entities_membership_member_legal_entity_id": legalEntityID})

	var legalEntities []*models.LegalEntity
	cursor, err := d.table.ListIn(ctx, txOrNil, &legalEntities, pagination, legalEntitiesSelect)
	if err != nil {
		return nil, nil, err
	}
	return legalEntities, cursor, nil
}

// ListMemberLegalEntities lists all legal entities that are members of a parent legal entity.
// Use cursor to page through results, if any.
func (d *LegalEntityStore) ListMemberLegalEntities(
	ctx context.Context,
	txOrNil *store.Tx,
	parentLegalEntityID models.LegalEntityID,
	pagination models.Pagination,
) ([]*models.LegalEntity, *models.Cursor, error) {
	legalEntitiesSelect := goqu.
		From(d.table.TableName()).
		Select(&models.LegalEntity{}).
		Join(goqu.T("legal_entities_memberships"),
			goqu.On(goqu.Ex{"legal_entities.legal_entity_id": goqu.I("legal_entities_memberships.legal_entities_membership_member_legal_entity_id")})).
		Where(goqu.Ex{"legal_entities_membership_legal_entity_id": parentLegalEntityID})

	var legalEntities []*models.LegalEntity
	cursor, err := d.table.ListIn(ctx, txOrNil, &legalEntities, pagination, legalEntitiesSelect)
	if err != nil {
		return nil, nil, err
	}
	return legalEntities, cursor, nil
}

// ListAllLegalEntities lists all legal entities in the system. Use cursor to page through results, if any.
func (d *LegalEntityStore) ListAllLegalEntities(ctx context.Context, txOrNil *store.Tx, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error) {
	legalEntitiesSelect := goqu.From(d.table.TableName()).Select(&models.LegalEntity{})

	var legalEntities []*models.LegalEntity
	cursor, err := d.table.ListIn(ctx, txOrNil, &legalEntities, pagination, legalEntitiesSelect)
	if err != nil {
		return nil, nil, err
	}
	return legalEntities, cursor, nil
}
