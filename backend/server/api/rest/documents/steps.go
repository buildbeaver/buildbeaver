package documents

import (
	"fmt"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Step struct {
	baseResourceDocument

	ID        models.StepID `json:"id"`
	CreatedAt models.Time   `json:"created_at"`
	UpdatedAt models.Time   `json:"updated_at"`
	DeletedAt *models.Time  `json:"deleted_at,omitempty"`
	ETag      models.ETag   `json:"etag"`

	// Name of the step.
	Name models.ResourceName `json:"name"`
	// Description is an optional human-readable description of the step.
	Description string `json:"description"`
	// Commands is a list of at least one command to execute during the step.
	Commands []models.Command `json:"commands"`
	// Depends describes the dependencies this step has on other steps within the parent job.
	Depends []*StepDependency `json:"depends"`

	JobID models.JobID `json:"job_id"`
	// RepoID that the step is building from.
	RepoID models.RepoID `json:"repo_id"`
	// RunnerID that ran the step (or empty if the step has not run yet).
	RunnerID models.RunnerID `json:"runner_id"`
	// LogDescriptorID points to the log for this step.
	LogDescriptorID models.LogDescriptorID `json:"log_descriptor_id"`
	// Status reflects where the step is in processing.
	Status models.WorkflowStatus `json:"status"`
	// Error is set if the step finished with an error (or nil if the step succeeded).
	Error *models.Error `json:"error"`
	// Timings records the times at which the step transitioned between statuses.
	Timings *WorkflowTimings `json:"timings"`

	RunnerURL        *string `json:"runner_url"`
	LogDescriptorURL string  `json:"log_descriptor_url"`
}

func MakeStep(rctx routes.RequestContext, step *models.Step) *Step {
	var runnerLink *string
	if step.RunnerID.Valid() {
		link := routes.MakeRunnerLink(rctx, step.RunnerID)
		runnerLink = &link
	}
	return &Step{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeStepLink(rctx, step.ID),
		},

		ID:        step.ID,
		CreatedAt: step.CreatedAt,
		UpdatedAt: step.UpdatedAt,
		DeletedAt: step.DeletedAt,
		ETag:      step.ETag,

		Name:        step.Name,
		Description: step.Description,
		Commands:    step.Commands,
		Depends:     MakeStepDependencies(step.Depends),

		JobID:           step.JobID,
		RepoID:          step.RepoID,
		RunnerID:        step.RunnerID,
		LogDescriptorID: step.LogDescriptorID,
		Status:          step.Status,
		Error:           step.Error,
		Timings:         MakeWorkflowTimings(&step.Timings),

		RunnerURL:        runnerLink,
		LogDescriptorURL: routes.MakeLogLink(rctx, step.LogDescriptorID),
	}
}

func MakeSteps(rctx routes.RequestContext, steps []*models.Step) []*Step {
	var docs []*Step
	for _, model := range steps {
		docs = append(docs, MakeStep(rctx, model))
	}
	return docs
}

func (d *Step) GetID() models.ResourceID {
	return d.ID.ResourceID
}

func (d *Step) GetKind() models.ResourceKind {
	return models.StepResourceKind
}

func (d *Step) GetCreatedAt() models.Time {
	return d.CreatedAt
}

func (m *Step) GetName() models.ResourceName {
	return m.Name
}

// GetDependencies returns a list of names of steps that must execute before this step in the parent job by name.
func (m *Step) GetDependencies() []models.ResourceName {
	var depends []models.ResourceName
	for _, dependency := range m.Depends {
		depends = append(depends, dependency.StepName)
	}
	return depends
}

type PatchStepRequest struct {
	// Status reflects where the step is in processing.
	Status *models.WorkflowStatus `json:"status"`
	// Error signifies the step finished with an error, if status is failed.
	Error *models.Error `json:"error"`
}

func (d *PatchStepRequest) Bind(r *http.Request) error {
	if d.Status != nil && !d.Status.Valid() {
		return gerror.NewErrValidationFailed(fmt.Sprintf("Invalid status: %s", d.Status))
	}
	if d.Error.Valid() && (d.Status == nil || *d.Status != models.WorkflowStatusFailed) {
		return gerror.NewErrValidationFailed("Error can only be specified on failed steps")
	}
	if d.Status != nil && *d.Status == models.WorkflowStatusFailed && !d.Error.Valid() {
		return gerror.NewErrValidationFailed("Failed workflow statuses must be accompanied by an error")
	}
	return nil
}

type StepDependency struct {
	StepName models.ResourceName `json:"step_name"`
}

func MakeStepDependency(dependency *models.StepDependency) *StepDependency {
	return &StepDependency{
		StepName: dependency.StepName,
	}
}

func MakeStepDependencies(dependencies models.StepDependencies) []*StepDependency {
	var docs []*StepDependency
	for _, model := range dependencies {
		docs = append(docs, MakeStepDependency(model))
	}
	return docs
}
