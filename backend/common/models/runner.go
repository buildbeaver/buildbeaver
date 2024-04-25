package models

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const RunnerResourceKind ResourceKind = "runner"

type RunnerID struct {
	ResourceID
}

func NewRunnerID() RunnerID {
	return RunnerID{ResourceID: NewResourceID(RunnerResourceKind)}
}

func RunnerIDFromResourceID(id ResourceID) RunnerID {
	return RunnerID{ResourceID: id}
}

type Runner struct {
	ID            RunnerID      `json:"id" goqu:"skipupdate" db:"runner_id"`
	Name          ResourceName  `json:"name" db:"runner_name"`
	LegalEntityID LegalEntityID `json:"legal_entity_id" db:"runner_legal_entity_id"`
	CreatedAt     Time          `json:"created_at" goqu:"skipupdate" db:"runner_created_at"`
	UpdatedAt     Time          `json:"updated_at" db:"runner_updated_at"`
	DeletedAt     *Time         `json:"deleted_at,omitempty" db:"runner_deleted_at"`
	ETag          ETag          `json:"etag" db:"runner_etag" hash:"ignore"`
	// SoftwareVersion is the software version of the runner process.
	SoftwareVersion string `json:"software_version" db:"runner_software_version"`
	// OperatingSystem is the operating system the runner process is currently running on.
	OperatingSystem string `json:"operating_system" db:"runner_operating_system"`
	// Architecture is the processor architecture the runner process is currently running on.
	Architecture string `json:"architecture" db:"runner_architecture"`
	// SupportedJobTypes is the one or more job types this runner supports.
	SupportedJobTypes JobTypes `json:"supported_job_types" db:"runner_supported_job_types"`
	// Labels contains the set of labels this runner is configured with.
	Labels Labels `json:"labels" db:"runner_labels"`
	// Enabled specifies if this runner is available to process jobs.
	Enabled bool `json:"enabled" db:"runner_enabled"`
}

func NewRunner(
	now Time,
	name ResourceName,
	legalEntityID LegalEntityID,
	softwareVersion string,
	operatingSystem string,
	architecture string,
	supportedJobTypes JobTypes,
	labels Labels,
	enabled bool,
) *Runner {
	return &Runner{
		ID:                NewRunnerID(),
		Name:              name,
		LegalEntityID:     legalEntityID,
		CreatedAt:         now,
		UpdatedAt:         now,
		SoftwareVersion:   softwareVersion,
		OperatingSystem:   operatingSystem,
		Architecture:      architecture,
		SupportedJobTypes: supportedJobTypes,
		Labels:            labels,
		Enabled:           enabled,
	}
}

func (m *Runner) GetKind() ResourceKind {
	return RunnerResourceKind
}

func (m *Runner) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Runner) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Runner) GetParentID() ResourceID {
	return m.LegalEntityID.ResourceID
}

func (m *Runner) GetName() ResourceName {
	return m.Name
}

func (m *Runner) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Runner) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Runner) GetETag() ETag {
	return m.ETag
}

func (m *Runner) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Runner) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *Runner) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *Runner) IsUnreachable() bool {
	// Runners should never be unreachable, even after being soft-deleted
	return false
}

func (m *Runner) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if !m.LegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error legal entity id must be set"))
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
	for _, label := range m.Labels {
		err := label.Validate()
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("error validating label %q: %w", label, err))
		}
	}
	return result.ErrorOrNil()
}
