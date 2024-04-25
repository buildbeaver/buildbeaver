package models

import (
	"errors"

	"github.com/hashicorp/go-multierror"
)

const GroupResourceKind ResourceKind = "group"

type GroupID struct {
	ResourceID
}

func NewGroupID() GroupID {
	return GroupID{ResourceID: NewResourceID(GroupResourceKind)}
}

func GroupIDFromResourceID(id ResourceID) GroupID {
	return GroupID{ResourceID: id}
}

type GroupMetadata struct {
	ID        GroupID `json:"id" goqu:"skipupdate" db:"access_control_group_id"`
	CreatedAt Time    `json:"created_at" goqu:"skipupdate" db:"access_control_group_created_at"`
	UpdatedAt Time    `json:"updated_at" db:"access_control_group_updated_at"`
	DeletedAt *Time   `json:"deleted_at,omitempty" db:"access_control_group_deleted_at"`
	ETag      ETag    `json:"etag" db:"access_control_group_etag" hash:"ignore"`
}

type Group struct {
	GroupMetadata
	// LegalEntityID is the id of the legal entity that owns the group (typically an organization)
	LegalEntityID LegalEntityID `json:"legal_entity_id" db:"access_control_group_legal_entity_id"`
	// Name of the group, unique within the groups owned by the owner legal entity
	Name ResourceName `json:"name" db:"access_control_group_name"`
	// Description is a human-readable description of the group
	Description string `json:"description" db:"access_control_group_description"`
	// IsInternal is true if this is a group created by the system that cannot be modified by users
	IsInternal bool                `json:"is_internal" db:"access_control_group_is_internal"`
	ExternalID *ExternalResourceID `json:"external_id" db:"access_control_group_external_id"`
}

func NewGroup(
	now Time,
	legalEntityID LegalEntityID,
	name ResourceName,
	description string,
	internal bool,
	externalID *ExternalResourceID,
) *Group {
	return &Group{
		GroupMetadata: GroupMetadata{
			ID:        NewGroupID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		LegalEntityID: legalEntityID,
		Name:          name,
		Description:   description,
		IsInternal:    internal,
		ExternalID:    externalID,
	}
}

// NewStandardGroup creates an access control Group object based on a standard group definition for a legal entity.
// The legal entity would normally represent a company or other organization.
func NewStandardGroup(now Time, legalEntityID LegalEntityID, groupDefinition *StandardGroupDefinition) *Group {
	return &Group{
		GroupMetadata: GroupMetadata{
			ID:        NewGroupID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		LegalEntityID: legalEntityID,
		Name:          groupDefinition.Name,
		Description:   groupDefinition.Description,
		IsInternal:    true, // users should not be able to edit standard groups
		ExternalID:    nil,  // no external ID for standard groups
	}
}

func (m *Group) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Group) GetKind() ResourceKind {
	return GroupResourceKind
}

func (m *Group) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Group) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Group) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Group) GetETag() ETag {
	return m.ETag
}

func (m *Group) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Group) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if m.DeletedAt != nil && m.DeletedAt.IsZero() {
		result = multierror.Append(result, errors.New("error deleted at must be non-zero when set"))
	}
	if !m.LegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error legal entity id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.ExternalID != nil && !m.ExternalID.Valid() {
		result = multierror.Append(result, errors.New("error external id is invalid"))
	}
	return result.ErrorOrNil()
}

// StandardGroupDefinition defines a standard access control group that can easily be created for a
// legal entity (normally for a company or other organization).
type StandardGroupDefinition struct {
	Name        ResourceName
	Description string
	Operations  []*Operation
}

var allStandardGroupDefinitions = map[ResourceName]*StandardGroupDefinition{
	BaseStandardGroup.Name:         BaseStandardGroup,
	ReadOnlyUserStandardGroup.Name: ReadOnlyUserStandardGroup,
	UserStandardGroup.Name:         UserStandardGroup,
	AdminStandardGroup.Name:        AdminStandardGroup,
	RunnerStandardGroup.Name:       RunnerStandardGroup,
}

func IsStandardGroupName(name ResourceName) bool {
	_, found := allStandardGroupDefinitions[name]
	return found
}

// BaseStandardGroup is an access control Group for users that are members of an organization but have no specific
// permissions for repos. Every member of a company legal entity should be a member of this group.
// Provides read-only access to top-level organization resources, especially the organization's legal entity.
var BaseStandardGroup = &StandardGroupDefinition{
	Name:        ResourceName("base"),
	Description: "The base group for an organization gives members basic permissions needed to be part of this organization.",
	Operations: []*Operation{
		LegalEntityReadOperation,
	},
}

// ReadOnlyUserStandardGroup is an access control Group for users that are read-only members of an organization.
// Provides read-only access to the organization's resources, and no ability to read secrets.
var ReadOnlyUserStandardGroup = &StandardGroupDefinition{
	Name:        ResourceName("readonly-user"),
	Description: "Read-only users for an organization have read-only access for repos owned by the organization.",
	Operations: []*Operation{
		LegalEntityReadOperation,
		ArtifactReadOperation,
		BuildReadOperation,
		RepoReadOperation,
	},
}

// UserStandardGroup is an access control Group for basic/regular users that are members of an
// organization. Provides basic read-write access to the organization's resources.
var UserStandardGroup = &StandardGroupDefinition{
	Name:        ResourceName("user"),
	Description: "Users for an organization can perform basic read and write actions for repos owned by the organization.",
	Operations: append(ReadOnlyUserStandardGroup.Operations, []*Operation{
		BuildCreateOperation,
		RepoUpdateOperation,
		SecretCreateOperation,
		SecretDeleteOperation,
		SecretReadOperation,
		SecretUpdateOperation,
	}...),
}

// AdminStandardGroup is an access control Group for administrators of an organization.
// Provides full access to every operation any user can perform on resources owned by the organization.
var AdminStandardGroup = &StandardGroupDefinition{
	Name:        ResourceName("admin"),
	Description: "Administrators for an organization can perform all actions on behalf of the organization.",
	Operations: append(UserStandardGroup.Operations, []*Operation{
		// Access control permissions
		GroupCreateOperation,
		GroupReadOperation,
		GroupUpdateOperation,
		GroupDeleteOperation,
		GroupMembershipCreateOperation,
		GroupMembershipReadOperation,
		GroupMembershipUpdateOperation,
		GroupMembershipDeleteOperation,
		GrantCreateOperation,
		GrantReadOperation,
		GrantUpdateOperation,
		GrantDeleteOperation,
		// Authentication permissions
		CredentialCreateOperation,
		CredentialReadOperation,
		CredentialUpdateOperation,
		CredentialDeleteOperation,
		// Runner registration
		RunnerCreateOperation,
		RunnerReadOperation,
		RunnerUpdateOperation,
		RunnerDeleteOperation,
		// Other permissions
		ArtifactDeleteOperation,
		// Some operations can not be performed by admins for legal entities:
		// - create new legal entities is done only by the server
		// - create or update artifacts is done only by build agents
		// - build update is done only by build agents
		// - repo create and delete is done only by the server during a sync
		// - SecretReadPlaintextOperation must only be available to build agents, *not* admins
	}...),
}

// RunnerStandardGroup is an access control Group for build runners used in the context of an organization or user.
// Provides the basic access required by a runner while it runs builds.
var RunnerStandardGroup = &StandardGroupDefinition{
	Name:        ResourceName("runner"),
	Description: "Runners for an organization or user can perform build-related actions for repos owned by the organization or user.",
	Operations: []*Operation{
		BuildReadOperation,
		BuildUpdateOperation,
		ArtifactCreateOperation,
		ArtifactReadOperation,
		SecretReadPlaintextOperation,
	},
}
