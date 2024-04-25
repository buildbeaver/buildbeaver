package documents

import "github.com/buildbeaver/buildbeaver/common/models"

type WorkflowTimings struct {
	QueuedAt    *models.Time `json:"queued_at"`
	SubmittedAt *models.Time `json:"submitted_at"`
	RunningAt   *models.Time `json:"running_at"`
	FinishedAt  *models.Time `json:"finished_at"`
	CanceledAt  *models.Time `json:"canceled_at"`
}

func MakeWorkflowTimings(timings *models.WorkflowTimings) *WorkflowTimings {
	return &WorkflowTimings{
		QueuedAt:    timings.QueuedAt,
		SubmittedAt: timings.SubmittedAt,
		RunningAt:   timings.RunningAt,
		FinishedAt:  timings.FinishedAt,
		CanceledAt:  timings.CanceledAt,
	}
}
