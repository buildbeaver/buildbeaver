package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// EnvVar represents a single key/value pair to export as an
// environment variable prior to executing all steps in a job.
type EnvVar struct {
	// Name of the environment variable
	Name string `json:"name"`
	SecretString
}

func (m *EnvVar) Validate() error {
	if m.Name == "" {
		return errors.New("error name must be set")
	}
	return nil
}

type JobEnvVars []*EnvVar

func (m *JobEnvVars) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("error unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m JobEnvVars) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}

// Merge combines the existing job environment vars with a new set from extraVars, and returns a new
// combined environment.
func (m JobEnvVars) Merge(extraVars JobEnvVars) JobEnvVars {
	var mergedEnv = make(JobEnvVars, 0, len(m)+len(extraVars))
	mergedEnv = append(mergedEnv, m...)
	mergedEnv = append(mergedEnv, extraVars...)
	// TODO: Remove vars with duplicate names if this is a problem
	return mergedEnv
}
