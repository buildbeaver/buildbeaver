package groups

import (
	"context"
	"fmt"
	"reflect"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	_ = models.MutableResource(&models.Group{})
	store.MustDBModel(&models.Group{})
}

type GroupStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *GroupStore {
	return &GroupStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "access_control_groups", &models.Group{}),
	}
}

// Create a new access control Group.
// Returns store.ErrAlreadyExists if a group with matching unique properties already exists.
func (d *GroupStore) Create(ctx context.Context, txOrNil *store.Tx, group *models.Group) error {
	return d.table.Create(ctx, txOrNil, group)
}

// Read an existing access control Group, looking it up by ResourceID.
// Returns models.ErrNotFound if the Group does not exist.
func (d *GroupStore) Read(ctx context.Context, txOrNil *store.Tx, id models.GroupID) (*models.Group, error) {
	group := &models.Group{}
	return group, d.table.ReadByID(ctx, txOrNil, id.ResourceID, group)
}

// ReadByName reads an existing access control Group, looking it up by group name and the ID of the
// legal entity that owns the group. Returns models.ErrNotFound if the group does not exist.
func (d *GroupStore) ReadByName(
	ctx context.Context,
	txOrNil *store.Tx,
	ownerLegalEntityID models.LegalEntityID,
	groupName models.ResourceName,
) (*models.Group, error) {
	group := &models.Group{}
	whereClause := goqu.Ex{
		"access_control_group_name":            groupName,
		"access_control_group_legal_entity_id": ownerLegalEntityID,
	}
	return group, d.table.ReadWhere(ctx, txOrNil, group, whereClause)
}

// ReadByExternalID reads an existing group, looking it up by its unique external id.
// Returns models.ErrNotFound if the group does not exist.
func (d *GroupStore) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.Group, error) {
	group := &models.Group{}
	whereClause := goqu.Ex{"access_control_group_external_id": externalID}
	err := d.table.ReadWhere(ctx, txOrNil, group, whereClause)
	return group, err
}

// FindOrCreateByName finds and returns the access control Group with the name and legal entity specified in
// the supplied group data.
// If no such group exists then a new group is created and returned, and true is returned for 'created'.
func (d *GroupStore) FindOrCreateByName(ctx context.Context, txOrNil *store.Tx, groupData *models.Group) (group *models.Group, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByName(ctx, tx, groupData.LegalEntityID, groupData.Name)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			err := d.table.Create(ctx, tx, groupData)
			return groupData, err
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.Group), created, nil
}

// Update an existing group with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *GroupStore) Update(ctx context.Context, txOrNil *store.Tx, group *models.Group) error {
	return d.table.UpdateByID(ctx, txOrNil, group)
}

// UpsertByExternalID creates a group if no group with the same External ID already exists, otherwise it updates
// the existing group's mutable properties if they differ from the in-memory instance.
// Returns true,false if the resource was created, false,true if the resource was updated, or false,false if
// neither create nor update was necessary.
// Returns an error if no External ID is filled out in the supplied Group.
// In all cases group.ID will be filled out in the supplied group object.
func (d *GroupStore) UpsertByExternalID(ctx context.Context, txOrNil *store.Tx, group *models.Group) (bool, bool, error) {
	if group.ExternalID == nil {
		return false, false, fmt.Errorf("error external id must be set in group to upsert")
	}
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.ReadByExternalID(ctx, tx, *group.ExternalID)
		}, func(tx *store.Tx) error {
			return d.Create(ctx, tx, group)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.Group)
			group.GroupMetadata = existing.GroupMetadata
			if reflect.DeepEqual(existing, group) {
				return false, nil
			}
			return true, d.Update(ctx, tx, group)
		})
}

// Delete permanently and idempotently deletes an access control group.
// The caller is responsible for ensuring that all memberships and grants for the group have previously been deleted.
func (s *GroupStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.GroupID) error {
	return s.table.DeleteByID(ctx, txOrNil, id.ResourceID)
}

// ListGroups returns a list of groups. Use cursor to page through results, if any.
// If groupParent is provided then only groups owned by the supplied parent legal entity will be returned.
// If memberID is provided then only groups that include the provided identity as a member will be returned.
func (s *GroupStore) ListGroups(ctx context.Context, txOrNil *store.Tx, groupParent *models.LegalEntityID, memberID *models.IdentityID, pagination models.Pagination) ([]*models.Group, *models.Cursor, error) {
	groupsSelect := goqu.From(s.table.TableName()).Select(&models.Group{})
	if memberID != nil {
		groupsSelect = groupsSelect.Join(goqu.T("access_control_group_memberships"),
			goqu.On(goqu.Ex{"access_control_groups.access_control_group_id": goqu.I("access_control_group_memberships.access_control_group_membership_group_id")})).
			Where(goqu.Ex{"access_control_group_membership_member_identity_id": memberID})
	}
	if groupParent != nil {
		groupsSelect = groupsSelect.Where(goqu.Ex{"access_control_group_legal_entity_id": groupParent})
	}

	var groups []*models.Group
	cursor, err := s.table.ListIn(ctx, txOrNil, &groups, pagination, groupsSelect)
	if err != nil {
		return nil, nil, err
	}
	return groups, cursor, nil
}
