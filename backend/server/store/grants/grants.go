package grants

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.Grant{})
}

type GrantStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *GrantStore {
	return &GrantStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "access_control_grants", &models.Grant{}),
	}
}

// Create a new grant.
// Returns store.ErrAlreadyExists if a grant with matching unique properties already exists.
func (d *GrantStore) Create(ctx context.Context, txOrNil *store.Tx, grant *models.Grant) error {
	d.table.Infof("Creating grant for %s to perform operation %s on resource %s",
		grant.GetAuthorizedResourceID(), grant.GetOperation(), grant.TargetResourceID)
	return d.table.Create(ctx, txOrNil, grant)
}

// Read an existing grant, looking it up by ResourceID.
// Returns models.ErrNotFound if the grant does not exist.
func (d *GrantStore) Read(ctx context.Context, txOrNil *store.Tx, id models.GrantID) (*models.Grant, error) {
	grant := &models.Grant{}
	return grant, d.table.ReadByID(ctx, txOrNil, id.ResourceID, grant)
}

// ReadByAuthorizedOperation reads an existing grant, looking it up by requiring the following fields to match:
// - OperationResourceType
// - OperationName
// - TargetResourceID
// - either AuthorizedIdentityID or AuthorizedGroupID must match, whichever one is not nil
// Returns models.ErrNotFound if the grant does not exist.
func (d *GrantStore) ReadByAuthorizedOperation(ctx context.Context, txOrNil *store.Tx, model *models.Grant) (*models.Grant, error) {
	grant := &models.Grant{}
	return grant, d.table.ReadWhere(ctx, txOrNil, grant,
		goqu.Or(
			goqu.Ex{
				// Won't match anything if AuthorizedIdentityID is nil
				"access_control_grant_authorized_identity_id": model.AuthorizedIdentityID,
			},
			goqu.Ex{
				// Won't match anything if AuthorizedGroupID is nil
				"access_control_grant_authorized_group_id": model.AuthorizedGroupID,
			},
		),
		goqu.Ex{
			"access_control_grant_operation_resource_kind": model.OperationResourceType,
			"access_control_grant_operation_name":          model.OperationName,
			"access_control_grant_target_resource_id":      model.TargetResourceID,
		})
}

func (d *GrantStore) Update(ctx context.Context, txOrNil *store.Tx, model *models.Grant) error {
	d.table.Infof("Updating grant with ID %s to grant %s permission to perform operation %s on resource %s",
		model.ID, model.GetAuthorizedResourceID(), model.GetOperation(), model.TargetResourceID)
	return d.table.UpdateByID(ctx, txOrNil, model)
}

// Delete permanently and idempotently deletes a grant, identifying it by id.
func (d *GrantStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.GrantID) error {
	d.table.Infof("Removing grant with ID %s", id)
	return d.table.DeleteByID(ctx, txOrNil, id.ResourceID)
}

// FindOrCreate finds and returns a grant with the data specified in the supplied grant data.
// The readByAuthorizedOperation function is used to find matching grants.
// If no such grant exists then a new one is created and returned, and true is returned for 'created'.
func (d *GrantStore) FindOrCreate(
	ctx context.Context,
	txOrNil *store.Tx,
	grantData *models.Grant,
) (grant *models.Grant, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByAuthorizedOperation(ctx, tx, grantData)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			err := d.Create(ctx, tx, grantData)
			return grantData, err
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.Grant), created, nil
}

// ListGrantsForGroup finds and returns all grants that give permissions to the specified group.
func (d *GrantStore) ListGrantsForGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	pagination models.Pagination,
) ([]*models.Grant, *models.Cursor, error) {
	grantSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Grant{}).
		Where(goqu.Ex{"access_control_grant_authorized_group_id": groupID})

	var grants []*models.Grant
	cursor, err := d.table.ListIn(ctx, txOrNil, &grants, pagination, grantSelect)
	if err != nil {
		return nil, nil, err
	}
	return grants, cursor, nil
}

// DeleteAllGrantsForGroup permanently and idempotently deletes all grants for the specified group.
func (d *GrantStore) DeleteAllGrantsForGroup(ctx context.Context, txOrNil *store.Tx, groupID models.GroupID) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"access_control_grant_authorized_group_id": groupID})
}

// DeleteAllGrantsForIdentity permanently and idempotently deletes all grants for the specified identity.
func (d *GrantStore) DeleteAllGrantsForIdentity(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"access_control_grant_authorized_identity_id": identityID})
}
