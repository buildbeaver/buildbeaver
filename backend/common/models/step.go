package models

import (
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const StepResourceKind ResourceKind = "step"

type StepID struct {
	ResourceID
}

func NewStepID() StepID {
	return StepID{ResourceID: NewResourceID(StepResourceKind)}
}

func StepIDFromResourceID(id ResourceID) StepID {
	return StepID{ResourceID: id}
}

// Step represents a single build step executed as part of a pipeline job.
type Step struct {
	StepMetadata
	StepData
}

type StepMetadata struct {
	ID        StepID `json:"id" goqu:"skipupdate" db:"step_id"`
	CreatedAt Time   `json:"created_at" goqu:"skipupdate" db:"step_created_at"`
	UpdatedAt Time   `json:"updated_at" db:"step_updated_at"`
	DeletedAt *Time  `json:"deleted_at,omitempty" db:"step_deleted_at"`
	ETag      ETag   `json:"etag" db:"step_etag" hash:"ignore"`
}

type StepData struct {
	StepDefinitionData
	JobID JobID `json:"job_id" db:"step_job_id"`
	// RepoID that the step is building from.
	RepoID RepoID `json:"repo_id" db:"step_repo_id"`
	// RunnerID that ran the step (or empty if the step has not run yet).
	RunnerID RunnerID `json:"runner_id" db:"step_runner_id"`
	// LogDescriptorID points to the log for this step.
	LogDescriptorID LogDescriptorID `json:"log_descriptor_id" db:"step_log_descriptor_id"`
	// Status reflects where the step is in processing.
	Status WorkflowStatus `json:"status" db:"step_status"`
	// Error is set if the step finished with an error (or nil if the step succeeded).
	Error *Error `json:"error" db:"step_error"`
	// Timings records the times at which the step transitioned between statuses.
	Timings WorkflowTimings `json:"timings" db:"step_timings"`
}

type StepDefinitionData struct {
	// Name of the step.
	Name ResourceName `json:"name" db:"step_name"`
	// Description is an optional human-readable description of the step.
	Description string `json:"description" db:"step_description"`
	// Commands is a list of at least one command to execute during the step.
	Commands Commands `json:"commands" db:"step_commands"`
	// Depends describes the dependencies this step has on other steps within the parent job.
	Depends StepDependencies `json:"depends" db:"step_depends"`
}

func (m *Step) GetKind() ResourceKind {
	return StepResourceKind
}

func (m *Step) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Step) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Step) GetParentID() ResourceID {
	return m.JobID.ResourceID
}

func (m *Step) GetName() ResourceName {
	return m.Name
}

func (m *Step) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Step) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Step) GetETag() ETag {
	return m.ETag
}

func (m *Step) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Step) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *Step) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *Step) IsUnreachable() bool {
	// Steps are unreachable after they are soft-deleted
	return m.DeletedAt != nil
}

// GetFQN returns a fully-qualified name for this step.
// TODO: This should include workflow and job name, but these are left blank since they aren't available
func (m *Step) GetFQN() NodeFQN {
	return NewNodeFQN("", "", m.Name)
}

// GetFQNDependencies returns a list of the fully-qualified names of steps that must execute before this step
// in the parent job by name.
func (m *Step) GetFQNDependencies() []NodeFQN {
	var depends []NodeFQN
	for _, dependency := range m.Depends {
		depends = append(depends, dependency.GetFQN())
	}
	return depends
}

// Validate the step.
func (m *Step) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if m.DeletedAt != nil && m.DeletedAt.IsZero() {
		result = multierror.Append(result, errors.New("error deleted at must be non-zero when set"))
	}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if !m.JobID.Valid() {
		result = multierror.Append(result, errors.New("error job id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if !m.Status.Valid() {
		result = multierror.Append(result, errors.New("error status is invalid"))
	}
	if len(m.Commands) == 0 {
		result = multierror.Append(result, errors.New("error at least one command must be set"))
	}
	for i, command := range m.Commands {
		if strings.Trim(string(command), " \n") == "" {
			result = multierror.Append(result, errors.Errorf("error commands cannot be empty (index %d)", i))
		}
	}
	return result.ErrorOrNil()
}

// PopulateDefaults sets default values for all fields of all structs
// in the step that haven't been populated.
func (m *Step) PopulateDefaults(build *Build, job *Job) {
	if !m.ID.Valid() {
		m.ID = NewStepID()
	}
	m.JobID = job.ID
	if m.CreatedAt.IsZero() {
		m.CreatedAt = job.CreatedAt
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = job.CreatedAt
	}
	if m.Status == "" || m.Status == WorkflowStatusUnknown {
		m.Status = WorkflowStatusQueued
	}
	if m.Error == nil {
		m.Error = &Error{}
	}
}
