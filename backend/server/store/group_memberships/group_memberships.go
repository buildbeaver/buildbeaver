package group_memberships

import (
	"context"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.GroupMembership{})
}

type GroupMembershipStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *GroupMembershipStore {
	return &GroupMembershipStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "access_control_group_memberships", &models.GroupMembership{}),
	}
}

// Create a new access control group membership.
// Returns store.ErrAlreadyExists if a group membership with matching unique properties already exists.
func (d *GroupMembershipStore) Create(ctx context.Context, txOrNil *store.Tx, groupMembershipData *models.GroupMembershipData) (*models.GroupMembership, error) {
	now := models.NewTime(time.Now())
	groupMembership := &models.GroupMembership{
		GroupMembershipData: *groupMembershipData,
		GroupMembershipMetadata: models.GroupMembershipMetadata{
			ID:        models.NewGroupMembershipID(),
			CreatedAt: now},
	}
	err := d.table.Create(ctx, txOrNil, groupMembership)
	if err != nil {
		return nil, err
	}
	return groupMembership, nil
}

// Read an existing access control group membership, looking it up by ResourceID.
// Returns models.ErrNotFound if the group membership does not exist.
func (d *GroupMembershipStore) Read(ctx context.Context, txOrNil *store.Tx, id models.GroupMembershipID) (*models.GroupMembership, error) {
	groupMembership := &models.GroupMembership{}
	return groupMembership, d.table.ReadByID(ctx, txOrNil, id.ResourceID, groupMembership)
}

// ReadByMember reads an existing access control group membership, looking it up by group, member identity and
// source system. Returns models.ErrNotFound if the group membership does not exist.
func (d *GroupMembershipStore) ReadByMember(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	memberID models.IdentityID,
	sourceSystem models.SystemName,
) (*models.GroupMembership, error) {
	membership := &models.GroupMembership{}
	whereClause := goqu.Ex{
		"access_control_group_membership_group_id":           groupID,
		"access_control_group_membership_member_identity_id": memberID,
		"access_control_group_membership_source_system":      sourceSystem,
	}
	return membership, d.table.ReadWhere(ctx, txOrNil, membership, whereClause)
}

// FindOrCreate finds and returns the access control group membership with the group, member identity and
// source system specified in the supplied group membership data.
// If no such group membership exists then a new one is created and returned, and true is returned for 'created'.
func (d *GroupMembershipStore) FindOrCreate(
	ctx context.Context,
	txOrNil *store.Tx,
	membershipData *models.GroupMembershipData,
) (membership *models.GroupMembership, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByMember(ctx, tx, membershipData.GroupID, membershipData.MemberIdentityID, membershipData.SourceSystem)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.Create(ctx, tx, membershipData)
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.GroupMembership), created, nil
}

// DeleteByMember removes a member identity from an access control group by deleting the relevant membership record(s).
// If sourceSystem is not nil then only the record matching the source system will be deleted; otherwise
// records from all source systems for the member will be deleted.
// This method is idempotent.
func (d *GroupMembershipStore) DeleteByMember(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	memberID models.IdentityID,
	sourceSystem *models.SystemName,
) error {
	whereClause := goqu.Ex{
		"access_control_group_membership_group_id":           groupID,
		"access_control_group_membership_member_identity_id": memberID,
	}
	if sourceSystem != nil {
		whereClause["access_control_group_membership_source_system"] = sourceSystem
	}

	return d.table.DeleteWhere(ctx, txOrNil, whereClause)
}

// DeleteAllMembersOfGroup removes all members from an access control group by deleting all membership records for
// that group. This method is idempotent.
func (d *GroupMembershipStore) DeleteAllMembersOfGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"access_control_group_membership_group_id": groupID})
}

// ListGroupMemberships returns a list of group memberships. Use cursor to page through results, if any.
// If groupID is provided then only memberships of the specified group will be returned.
// If memberID is provided then only groups that include the provided identity as a member will be returned.
// If sourceSystem is provided then only memberships with matching source system values will be returned.
func (d *GroupMembershipStore) ListGroupMemberships(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID *models.GroupID,
	memberID *models.IdentityID,
	sourceSystem *models.SystemName,
	pagination models.Pagination,
) ([]*models.GroupMembership, *models.Cursor, error) {
	membershipSelect := goqu.From(d.table.TableName()).Select(&models.GroupMembership{})
	if groupID != nil {
		membershipSelect = membershipSelect.Where(goqu.Ex{"access_control_group_membership_group_id": groupID})
	}
	if memberID != nil {
		membershipSelect = membershipSelect.Where(goqu.Ex{"access_control_group_membership_member_identity_id": memberID})
	}
	if sourceSystem != nil {
		membershipSelect = membershipSelect.Where(goqu.Ex{"access_control_group_membership_source_system": sourceSystem})
	}

	var memberships []*models.GroupMembership
	cursor, err := d.table.ListIn(ctx, txOrNil, &memberships, pagination, membershipSelect)
	if err != nil {
		return nil, nil, err
	}
	return memberships, cursor, nil
}
