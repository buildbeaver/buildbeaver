package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// JobDependency declares that one job depends on the successful execution of another, and optionally
// that the dependent job consumes one or more artifacts from the other.
type JobDependency struct {
	// Workflow is the workflow the job referenced in the dependency belongs to, or empty if the referenced
	// job belongs to the default workflow
	Workflow ResourceName `json:"workflow"`
	// JobName is the name of the job referenced in the dependency
	JobName ResourceName `json:"job_name"`
	// ArtifactDependencies lists artifacts produced by the referenced job that are required by the dependent job
	ArtifactDependencies []*ArtifactDependency `json:"artifact_dependencies"`
}

func NewJobDependency(workflow ResourceName, jobName ResourceName, artifactDependencies ...*ArtifactDependency) *JobDependency {
	return &JobDependency{
		Workflow:             workflow,
		JobName:              jobName,
		ArtifactDependencies: artifactDependencies,
	}
}

func (m *JobDependency) Equal(that *JobDependency) bool {
	return m.Workflow == that.Workflow && m.JobName == that.JobName
}

// GetFQN returns a fully-qualified name for this job dependency, which includes workflow and job name.
func (m *JobDependency) GetFQN() NodeFQN {
	return NewNodeFQNForJob(m.Workflow, m.JobName)
}

func (m *JobDependency) Validate() error {
	var result *multierror.Error
	if m.Workflow != "" {
		if err := m.Workflow.Validate(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	if err := m.JobName.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.Workflow != "" {
		if err := m.Workflow.Validate(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

type JobDependencies []*JobDependency

func (m *JobDependencies) Scan(src interface{}) error {
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

func (m JobDependencies) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
