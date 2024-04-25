package models

import "fmt"

// Operation represents an operation that can be performed on a resource
// that is subject to access control.
type Operation struct {
	// Name of the operation, unique within a resource type
	Name string `json:"name"`
	// ResourceKind is the unique type of resource that this operation applies to
	ResourceKind ResourceKind `json:"resource_kind"`
}

func (m Operation) String() string {
	return fmt.Sprintf("%s:%s", m.Name, m.ResourceKind)
}
