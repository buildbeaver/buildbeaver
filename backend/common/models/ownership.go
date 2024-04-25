package models

import (
	"errors"

	"github.com/hashicorp/go-multierror"
)

const OwnershipResourceKind ResourceKind = "ownership"

type OwnershipID struct {
	ResourceID
}

func NewOwnershipID() OwnershipID {
	return OwnershipID{ResourceID: NewResourceID(OwnershipResourceKind)}
}

func OwnershipIDFromResourceID(id ResourceID) OwnershipID {
	return OwnershipID{ResourceID: id}
}

type OwnershipMetadata struct {
	ID        OwnershipID `json:"id" goqu:"skipupdate" db:"access_control_ownership_id"`
	CreatedAt Time        `json:"created_at" goqu:"skipupdate" db:"access_control_ownership_created_at"`
	UpdatedAt Time        `json:"updated_at" db:"access_control_ownership_updated_at"`
	ETag      ETag        `json:"etag" db:"access_control_ownership_etag" hash:"ignore"`
}

// Ownership represents the ownership of one resource by another resource.
// The ownership hierarchy for resources is used for access control, and is not necessarily related to a
// resource's resource link parent relationships.
type Ownership struct {
	OwnershipMetadata
	// OwnerResourceType is the unique id of the resource that owns another resource
	OwnerResourceID ResourceID `json:"owner_resource_id" db:"access_control_ownership_owner_resource_id"`
	// OwnedResourceID is the unique id of the resource that is owned
	OwnedResourceID ResourceID `json:"owned_resource_id" db:"access_control_ownership_owned_resource_id"`
}

func NewOwnership(now Time, ownerResourceID ResourceID, ownedResourceID ResourceID) *Ownership {
	return &Ownership{
		OwnershipMetadata: OwnershipMetadata{
			ID:        NewOwnershipID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OwnerResourceID: ownerResourceID,
		OwnedResourceID: ownedResourceID,
	}
}

func (m *Ownership) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Ownership) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Ownership) GetKind() ResourceKind {
	return OwnershipResourceKind
}

func (m *Ownership) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Ownership) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Ownership) GetETag() ETag {
	return m.ETag
}

func (m *Ownership) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Ownership) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	return result.ErrorOrNil()
}
