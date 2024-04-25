package models

import (
	"github.com/hashicorp/go-multierror"

	"github.com/pkg/errors"
)

const IdentityResourceKind ResourceKind = "identity"

// NoIdentity is a zero-value identity id, used as a shortcut for functions that support
// an optional identity (like our standard searches).
var NoIdentity = IdentityID{}

type IdentityID struct {
	ResourceID
}

func NewIdentityID() IdentityID {
	return IdentityID{ResourceID: NewResourceID(IdentityResourceKind)}
}

func IdentityIDFromResourceID(id ResourceID) IdentityID {
	return IdentityID{ResourceID: id}
}

type Identity struct {
	ID        IdentityID `json:"id" goqu:"skipupdate" db:"identity_id"`
	CreatedAt Time       `json:"created_at" goqu:"skipupdate" db:"identity_created_at"`
	// Resource that this identity represents, e.g. a Legal Entity (company or person) or a Runner
	// Each owner resource can only have at most one Identity.
	OwnerResourceID ResourceID `json:"owner_resource_id" goqu:"skipupdate" db:"identity_owner_resource_id"`
}

func NewIdentity(now Time, ownerResourceID ResourceID) *Identity {
	return &Identity{
		ID:              NewIdentityID(),
		CreatedAt:       now,
		OwnerResourceID: ownerResourceID,
	}
}

func (m *Identity) GetKind() ResourceKind {
	return IdentityResourceKind
}

func (m *Identity) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Identity) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Identity) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.OwnerResourceID.IsZero() {
		result = multierror.Append(result, errors.New("error owner resource ID must be set"))
	}
	return result.ErrorOrNil()
}
