package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	JobTypeDocker JobType = "docker"
	JobTypeExec   JobType = "exec"
)

type JobType string

func (m *JobType) Scan(src interface{}) error {
	if src == nil {
		*m = JobTypeDocker
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return errors.Errorf("error expected string but found: %T", src)
	}
	switch strings.ToLower(t) {
	case "", string(JobTypeDocker):
		*m = JobTypeDocker
	case string(JobTypeExec):
		*m = JobTypeExec
	default:
		return errors.Errorf("error unknown job type: %s", t)
	}
	return nil
}

func (m JobType) Valid() bool {
	return m == JobTypeDocker || m == JobTypeExec
}

func (m JobType) String() string {
	return string(m)
}

type JobTypes []JobType

func (m *JobTypes) Scan(src interface{}) error {
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

func (m JobTypes) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
