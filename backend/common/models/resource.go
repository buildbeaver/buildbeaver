package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Resource interface {
	// GetKind returns the unique name/type of the resource e.g. "build" or "repo".
	GetKind() ResourceKind
	// GetCreatedAt returns the Time at which this resource was created.
	GetCreatedAt() Time
	// GetID returns the globally unique ResourceID of the resource.
	GetID() ResourceID
	// Validate the model by checking for required fields, lengths and types etc.
	Validate() error
}

type NamedResource interface {
	Resource
	// GetParentID returns the globally unique ResourceID of this resource's parent. Or an empty
	// ID if this resource does not have a parent.
	GetParentID() ResourceID
	// GetName returns the name of the resource which, combined with the parent resource's ResourceID,
	// uniquely identifies the resource e.g. "my-company" inside "legal-entity:abcdedfg".
	GetName() ResourceName
}

type MutableResource interface {
	Resource
	GetETag() ETag
	SetETag(eTag ETag)
	GetUpdatedAt() Time
	SetUpdatedAt(t Time)
}

type SoftDeletableResource interface {
	Resource
	GetDeletedAt() *Time
	SetDeletedAt(deletedAt *Time)
	// IsUnreachable returns true if this resource can't be read by ID after being soft deleted
	IsUnreachable() bool
}

type BaseResource struct {
	kind ResourceKind
	id   ResourceID
}

func NewBaseResource(kind ResourceKind, id ResourceID) *BaseResource {
	return &BaseResource{
		kind: kind,
		id:   id,
	}
}

func (m *BaseResource) GetID() ResourceID {
	return m.id
}

func (m *BaseResource) SetID(id ResourceID) {
	m.id = id
}

func (m *BaseResource) GetKind() ResourceKind {
	return m.kind
}

func (m *BaseResource) Validate() error {
	var result *multierror.Error
	if m.kind == "" {
		result = multierror.Append(result, errors.New("error resource kind must be set"))
	}
	if !m.id.Valid() {
		result = multierror.Append(result, errors.New("error resource id must be set"))
	}
	return result.ErrorOrNil()
}
