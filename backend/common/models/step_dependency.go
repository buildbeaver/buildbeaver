package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// StepDependency declares that a step depends on the successful execution of another step within the same job.
type StepDependency struct {
	StepName ResourceName `json:"step_name"`
}

func NewStepDependency(stepName ResourceName) *StepDependency {
	return &StepDependency{StepName: stepName}
}

func (m *StepDependency) String() string {
	return m.StepName.String()
}

func (m *StepDependency) Equal(that *StepDependency) bool {
	return m.String() == that.String()
}

func (m *StepDependency) Validate() error {
	var result *multierror.Error
	if err := m.StepName.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	return result.ErrorOrNil()
}

// GetFQN returns a fully-qualified name for this step dependency, which includes workflow and job name.
// TODO: This should include workflow and job name, but these are left blank since they aren't available
func (m *StepDependency) GetFQN() NodeFQN {
	return NewNodeFQN("", "", m.StepName)
}

type StepDependencies []*StepDependency

func (m *StepDependencies) Scan(src interface{}) error {
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

func (m StepDependencies) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
