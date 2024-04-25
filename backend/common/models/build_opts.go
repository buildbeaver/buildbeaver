package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// BuildOptions contains options that affect how the build is scheduled or executed.
type BuildOptions struct {
	// Force all jobs in the build to run by ignoring fingerprints.
	Force bool `json:"force"`
	// NodesToRun contains zero or more jobs and steps to run. If no nodes are specified
	// then all jobs and steps will be run.
	NodesToRun []NodeFQN `json:"nodes_to_run"`
}

func (m *BuildOptions) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m BuildOptions) Value() (driver.Value, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
