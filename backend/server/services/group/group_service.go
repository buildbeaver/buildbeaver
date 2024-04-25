package group

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type GroupService struct {
	db                   *store.DB
	ownershipStore       store.OwnershipStore
	groupStore           store.GroupStore
	groupMembershipStore store.GroupMembershipStore
	grantStore           store.GrantStore
	authorizationService services.AuthorizationService
	logger.Log
}

func NewGroupService(
	db *store.DB,
	ownershipStore store.OwnershipStore,
	groupStore store.GroupStore,
	groupMembershipStore store.GroupMembershipStore,
	grantStore store.GrantStore,
	authorizationService services.AuthorizationService,
	logFactory logger.LogFactory,
) *GroupService {
	return &GroupService{
		db:                   db,
		ownershipStore:       ownershipStore,
		groupStore:           groupStore,
		groupMembershipStore: groupMembershipStore,
		grantStore:           grantStore,
		authorizationService: authorizationService,
		Log:                  logFactory("GroupService"),
	}
}

// ReadByName reads an existing access control Group, looking it up by group name and the ID of the
// legal entity that owns the group. Returns models.ErrNotFound if the group does not exist.
func (s *GroupService) ReadByName(
	ctx context.Context,
	txOrNil *store.Tx,
	ownerLegalEntityID models.LegalEntityID,
	groupName models.ResourceName,
) (*models.Group, error) {
	return s.groupStore.ReadByName(ctx, txOrNil, ownerLegalEntityID, groupName)
}

// ReadByExternalID reads an existing group, looking it up by its unique external id.
// Returns models.ErrNotFound if the group does not exist.
func (s *GroupService) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.Group, error) {
	return s.groupStore.ReadByExternalID(ctx, txOrNil, externalID)
}

// FindOrCreateStandardGroup finds or creates a new access control Group for a Legal Entity, and sets up
// permissions for any new group that was created, based on the supplied standard group definition.
func (s *GroupService) FindOrCreateStandardGroup(
	ctx context.Context,
	tx *store.Tx,
	legalEntity *models.LegalEntity,
	groupDefinition *models.StandardGroupDefinition,
) (*models.Group, error) {
	// Find or create the group
	groupData := models.NewStandardGroup(models.NewTime(time.Now()), legalEntity.ID, groupDefinition)
	group, created, err := s.FindOrCreateByName(ctx, tx, groupData)
	if err != nil {
		return nil, fmt.Errorf("error finding or creating %q group for %q: %w", groupDefinition.Name, legalEntity.ID, err)
	}

	// If we created a new group then grant it the permissions in the group definition
	if created {
		err = s.authorizationService.CreateGrantsForGroup(
			ctx,
			tx,
			legalEntity.ID,
			group.ID,
			groupDefinition.Operations,
			legalEntity.GetID())
		if err != nil {
			return nil, fmt.Errorf("error creating grants for %q group for %q, GroupID %q: %w",
				groupDefinition.Name, legalEntity.ID, group.ID, err)
		}
	}
	return group, nil
}

// FindOrCreateByName finds and returns the access control Group with the name and legal entity specified in
// the supplied group data.
// If no such group exists then a new group is created and returned, and true is returned for 'created'.
func (s *GroupService) FindOrCreateByName(ctx context.Context, txOrNil *store.Tx, groupData *models.Group) (*models.Group, bool, error) {
	err := groupData.Validate()
	if err != nil {
		return nil, false, fmt.Errorf("error validating group data: %w", err)
	}

	var (
		group   *models.Group
		created = false
	)
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		group, created, err = s.groupStore.FindOrCreateByName(ctx, tx, groupData)
		if err != nil {
			return fmt.Errorf("error finding or creating group: %w", err)
		}
		if created {
			ownership := models.NewOwnership(models.NewTime(time.Now()), group.LegalEntityID.ResourceID, group.GetID())
			_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
			if err != nil {
				return fmt.Errorf("error upserting ownership for group: %w", err)
			}
			s.Infof("Created group %q for %s", groupData.Name, groupData.LegalEntityID)
		}
		return nil
	})

	return group, created, err
}

// UpsertByExternalID creates a group if no group with the same External ID already exists, otherwise it updates
// the existing group's mutable properties if they differ from the in-memory instance.
// Returns true,false if the resource was created, false,true if the resource was updated, or false,false if
// neither create nor update was necessary.
// Returns an error if no External ID is filled out in the supplied Group.
// In all cases group.ID will be filled out in the supplied group object.
func (s *GroupService) UpsertByExternalID(ctx context.Context, txOrNil *store.Tx, group *models.Group) (created bool, updated bool, err error) {
	err = group.Validate()
	if err != nil {
		return false, false, errors.Wrap(err, "error validating group")
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Create or update the group if required
		created, updated, err := s.groupStore.UpsertByExternalID(ctx, tx, group)
		if err != nil {
			return fmt.Errorf("error upserting group: %w", err)
		}

		if created || updated {
			ownership := models.NewOwnership(models.NewTime(time.Now()), group.LegalEntityID.ResourceID, group.GetID())
			_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
			if err != nil {
				return fmt.Errorf("error upserting ownership for group: %w", err)
			}
		}
		if created {
			s.Infof("Created group %q for %s", group.Name, group.LegalEntityID)
		} else if updated {
			s.Infof("Updated group %q for %s", group.Name, group.LegalEntityID)
		}
		return nil
	})
	return created, updated, err
}

// Delete permanently and idempotently deletes an access control group.
// All memberships and grants for this group will also be permanently deleted.
func (s *GroupService) Delete(ctx context.Context, txOrNil *store.Tx, id models.GroupID) error {
	// Delete all group memberships for this group
	err := s.groupMembershipStore.DeleteAllMembersOfGroup(ctx, txOrNil, id)
	if err != nil {
		return fmt.Errorf("error attepting to delete all members of group before deleting group '%s'", id)
	}

	// Delete all grants for this group
	err = s.grantStore.DeleteAllGrantsForGroup(ctx, txOrNil, id)
	if err != nil {
		return fmt.Errorf("error attepting to delete all grants for group before deleting group '%s'", id)
	}

	err = s.groupStore.Delete(ctx, txOrNil, id)
	if err != nil {
		return fmt.Errorf("error attepting to delete group '%s'", id)
	}

	s.Infof("Deleted group with ID %s", id)
	return nil
}

// ListGroups returns a list of groups. Use cursor to page through results, if any.
// If groupParent is provided then only groups owned by the supplied parent legal entity will be returned.
// If memberID is provided then only groups that include the provided identity as a member will be returned.
func (s *GroupService) ListGroups(
	ctx context.Context,
	txOrNil *store.Tx,
	groupParent *models.LegalEntityID,
	memberID *models.IdentityID,
	pagination models.Pagination,
) ([]*models.Group, *models.Cursor, error) {
	return s.groupStore.ListGroups(ctx, txOrNil, groupParent, memberID, pagination)
}

// FindOrCreateMembership adds the specified identity to an access control Group by adding a group membership
// for a specific source system.
// This method is idempotent, and returns true if a new membership was created or false if there was already
// a membership for this identity for the group with the specified source system
func (s *GroupService) FindOrCreateMembership(ctx context.Context, txOrNil *store.Tx, membershipData *models.GroupMembershipData) (membership *models.GroupMembership, created bool, err error) {
	// Validate data for the membership
	err = membershipData.Validate()
	if err != nil {
		return nil, false, fmt.Errorf("error validating group membership data: %w", err)
	}

	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Read group to check it exists; this avoids hitting foreign key constraints
		group, err := s.groupStore.Read(ctx, tx, membershipData.GroupID)
		if err != nil {
			return fmt.Errorf("error adding user to group: %s not found: %w", membershipData.GroupID, err)
		}

		// Find or create a group membership matching the supplied membership data
		membership, created, err = s.groupMembershipStore.FindOrCreate(ctx, tx, membershipData)
		if err != nil {
			return fmt.Errorf("error creating group membership: %w", err)
		}
		if created {
			// The group membership is owned by (i.e. a child of) the group
			ownership := models.NewOwnership(models.NewTime(time.Now()), membership.GroupID.ResourceID, membership.GetID())
			_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
			if err != nil {
				return fmt.Errorf("error upserting ownership for group membership: %w", err)
			}

			s.Infof("Added %s to %s (group '%s' for %s) - added by %s",
				membershipData.MemberIdentityID, membershipData.GroupID, group.Name, group.LegalEntityID, membershipData.AddedByLegalEntityID)
		}
		return nil
	})

	return membership, created, err
}

// RemoveMembership removes a membership for the specified identity from an access control group.
// If sourceSystem is not nil then only the membership record matching the source system will be deleted;
// otherwise membership records from all source systems for the member will be deleted.
// This method is idempotent.
func (s *GroupService) RemoveMembership(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	memberID models.IdentityID,
	sourceSystem *models.SystemName,
) error {
	err := s.groupMembershipStore.DeleteByMember(ctx, txOrNil, groupID, memberID, sourceSystem)
	if err != nil {
		return err
	}
	if sourceSystem != nil {
		s.Infof("Removed membership record from source system %s for identity %s from group %s", sourceSystem, memberID, groupID)
	} else {
		s.Infof("Removed all membership records for identity %s from group %s", memberID, groupID)
	}
	return nil
}

// ReadMembership reads an existing access control group membership, looking it up by group member, identity and
// source system. Returns models.ErrNotFound if the group membership does not exist.
func (s *GroupService) ReadMembership(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	memberID models.IdentityID,
	sourceSystem models.SystemName,
) (*models.GroupMembership, error) {
	return s.groupMembershipStore.ReadByMember(ctx, txOrNil, groupID, memberID, sourceSystem)
}

// ListGroupMemberships returns a list of group memberships. Use cursor to page through results, if any.
// If groupID is provided then only memberships of the specified group will be returned.
// If memberID is provided then only groups that include the provided identity as a member will be returned.
// If sourceSystem is provided then only memberships with matching source system values will be returned.
func (s *GroupService) ListGroupMemberships(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID *models.GroupID,
	memberID *models.IdentityID,
	sourceSystem *models.SystemName,
	pagination models.Pagination,
) ([]*models.GroupMembership, *models.Cursor, error) {
	return s.groupMembershipStore.ListGroupMemberships(ctx, txOrNil, groupID, memberID, sourceSystem, pagination)
}
