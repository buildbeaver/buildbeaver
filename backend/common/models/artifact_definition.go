package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// ArtifactDefinition is generated from jobs in the build config.
// It declares that a job is expected to create one or more artifacts at the given paths, and
// that these artifact files should be saved and made available to other jobs (see ArtifactDependency)
type ArtifactDefinition struct {
	// GroupName uniquely identifies the one or more artifacts specified in paths.
	GroupName ResourceName `json:"name"`
	// Paths contains one or more relative paths to artifacts that should be uploaded at the
	// end of the build. These paths will be globbed, so that each path may identify one or
	// more actual files.
	Paths []string `json:"paths"`
}

func (m *ArtifactDefinition) Validate() error {
	var result *multierror.Error
	if err := m.GroupName.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if len(m.Paths) == 0 {
		result = multierror.Append(result, errors.New("Artifact must specify at least one path"))
	}
	for _, path := range m.Paths {
		if filepath.IsAbs(path) {
			result = multierror.Append(result, fmt.Errorf("Artifact path %q must be relative to the checkout directory", path))
		}
	}
	return result.ErrorOrNil()
}

type ArtifactDefinitions []*ArtifactDefinition

func (m *ArtifactDefinitions) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m ArtifactDefinitions) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
