package models

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ExternalResourceID uniquely identifies a resource in an external system (like a repo in GitHub).
// This id must be immutable and cannot be reused after being deleted. Typically, we expect this to
// be a uuid or auto-incremented database id etc.
type ExternalResourceID struct {
	// ExternalSystem identifies the name of the external system e.g. GitHub
	ExternalSystem SystemName `json:"external_system"`
	// ResourceID identifies the resource within the external system e.g. github_repo.id
	ResourceID string `json:"resource_id"`
}

func NewExternalResourceID(systemName SystemName, resourceID string) ExternalResourceID {
	return ExternalResourceID{
		ExternalSystem: systemName,
		ResourceID:     resourceID,
	}
}

func (m ExternalResourceID) String() string {
	return fmt.Sprintf("%s:%s", m.ExternalSystem, m.ResourceID)
}

func (m *ExternalResourceID) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return errors.Errorf("error expected string but found: %T", src)
	}
	// Allow empty string as a valid ExternalResourceID, treated as a nil ID
	if str == "" {
		return nil
	}
	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return errors.New("error unexpected number of parts")
	}
	m.ExternalSystem = SystemName(parts[0])
	m.ResourceID = parts[1]
	return nil
}

func (m ExternalResourceID) Value() (driver.Value, error) {
	return m.String(), nil
}

func (m ExternalResourceID) Valid() bool {
	return m.ExternalSystem != "" && m.ResourceID != ""
}
