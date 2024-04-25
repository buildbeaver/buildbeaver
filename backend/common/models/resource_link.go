package models

import (
	"errors"

	"github.com/hashicorp/go-multierror"
)

type ResourceLink []ResourceLinkFragmentID

type ResourceLinkFragmentID struct {
	Name ResourceName
	Kind ResourceKind
}

type ResourceLinkFragment struct {
	ID        ResourceID   `json:"id" goqu:"skipupdate" db:"resource_link_fragment_id"`
	CreatedAt Time         `json:"created_at" goqu:"skipupdate" db:"resource_link_fragment_created_at"`
	Name      ResourceName `json:"name" db:"resource_link_fragment_name"`
	ParentID  ResourceID   `json:"parent_id" db:"resource_link_fragment_parent_id"`
	Kind      ResourceKind `json:"kind" db:"resource_link_fragment_kind"`
}

func (m *ResourceLinkFragment) GetKind() ResourceKind {
	return m.Kind
}

func (m *ResourceLinkFragment) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *ResourceLinkFragment) GetID() ResourceID {
	return m.ID
}

func (m *ResourceLinkFragment) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.Kind == "" {
		result = multierror.Append(result, errors.New("error kind must be set"))
	}
	return result.ErrorOrNil()
}
