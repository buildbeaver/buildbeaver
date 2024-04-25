package dto

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

type CreateStep struct {
	*models.Step
	Job *models.Job
}

// Validate the create, and the underlying step.
func (m *CreateStep) Validate() error {
	if m.JobID != m.Job.ID {
		return fmt.Errorf("error mismatched job ids")
	}
	return m.Step.Validate()
}

type UpdateStepStatus struct {
	Status models.WorkflowStatus
	Error  *models.Error
	ETag   models.ETag
}
