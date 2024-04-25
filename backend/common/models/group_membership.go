package models

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

const GroupMembershipResourceKind ResourceKind = "group-membership"

type GroupMembershipID struct {
	ResourceID
}

func NewGroupMembershipID() GroupMembershipID {
	return GroupMembershipID{ResourceID: NewResourceID(GroupMembershipResourceKind)}
}

func GroupMembershipIDFromResourceID(id ResourceID) GroupMembershipID {
	return GroupMembershipID{ResourceID: id}
}

type GroupMembershipMetadata struct {
	ID        GroupMembershipID `json:"id" goqu:"skipupdate" db:"access_control_group_membership_id"`
	CreatedAt Time              `json:"created_at"  db:"access_control_group_membership_created_at"`
}

type GroupMembershipData struct {
	// GroupID is the id of the group that the legal entity is a member of
	GroupID GroupID `json:"group_id"  db:"access_control_group_membership_group_id"`
	// MemberLegalEntityID is the id of the legal entity that belongs to the group
	MemberIdentityID IdentityID `json:"member_identity_id"  db:"access_control_group_membership_member_identity_id"`
	// Name of the system (internal or external) that caused the membership to be added
	SourceSystem SystemName `json:"source_system_name" db:"access_control_group_membership_source_system"`
	// AddedByLegalEntityID is the id of the legal entity that created this group membership
	AddedByLegalEntityID LegalEntityID `json:"added_by_legal_entity_id"  db:"access_control_group_membership_added_by_legal_entity_id"`
}

type GroupMembership struct {
	GroupMembershipMetadata
	GroupMembershipData
}

func NewGroupMembershipData(
	groupID GroupID,
	memberIdentityID IdentityID,
	sourceSystem SystemName,
	addedByLegalEntityID LegalEntityID,
) *GroupMembershipData {
	return &GroupMembershipData{
		GroupID:              groupID,
		MemberIdentityID:     memberIdentityID,
		SourceSystem:         sourceSystem,
		AddedByLegalEntityID: addedByLegalEntityID,
	}
}

func (m *GroupMembership) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *GroupMembership) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *GroupMembership) GetKind() ResourceKind {
	return GroupMembershipResourceKind
}

func (m *GroupMembership) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	err := m.GroupMembershipData.Validate()
	if err != nil {
		result = multierror.Append(result, fmt.Errorf("data is invalid: %s", err))
	}
	return result.ErrorOrNil()
}

func (m *GroupMembershipData) Validate() error {
	var result *multierror.Error
	if !m.GroupID.Valid() {
		result = multierror.Append(result, errors.New("error group id must be set"))
	}
	if !m.MemberIdentityID.Valid() {
		result = multierror.Append(result, errors.New("error member identity id must be set"))
	}
	if !m.AddedByLegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error added by legal entity id must be set"))
	}
	return result.ErrorOrNil()
}
