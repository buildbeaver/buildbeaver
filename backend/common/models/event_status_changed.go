package models

const (
	// BuildStatusChangedEvent is an event to notify subscribers that the status of a build has changed.
	// The event resource ID should be the ID of the build. The data should be a string of type WorkflowStatus.
	BuildStatusChangedEvent EventType = "BuildStatusChanged"

	// JobStatusChangedEvent is an event to notify subscribers that the status of a Job has changed.
	// The event resource ID should be the ID of the job.
	// The event Name should be the fully-qualified name of the job (including workflow prefix if any).
	// The data should be a string of type WorkflowStatus.
	JobStatusChangedEvent EventType = "JobStatusChanged"

	// StepStatusChangedEvent is an event to notify subscribers that the status of a Step has changed.
	// The event resource ID should be the ID of the step. The data should be a string of type WorkflowStatus.
	StepStatusChangedEvent EventType = "StepStatusChanged"
)

func NewBuildStatusChangedEventData(build *Build) *EventData {
	return &EventData{
		BuildID:      build.ID,
		Type:         BuildStatusChangedEvent,
		ResourceID:   build.ID.ResourceID,
		Workflow:     "",
		JobName:      "",
		ResourceName: build.Name,
		Payload:      build.Status.String(),
	}
}

func NewJobStatusChangedEventData(job *Job) *EventData {
	return &EventData{
		BuildID:      job.BuildID,
		Type:         JobStatusChangedEvent,
		ResourceID:   job.ID.ResourceID,
		Workflow:     job.Workflow,
		JobName:      job.Name,
		ResourceName: job.Name,
		Payload:      job.Status.String(),
	}
}

func NewStepStatusChangedEventData(job *Job, step *Step) *EventData {
	return &EventData{
		BuildID:      job.BuildID,
		Type:         StepStatusChangedEvent,
		ResourceID:   step.ID.ResourceID,
		Workflow:     job.Workflow,
		JobName:      job.Name,
		ResourceName: step.Name,
		Payload:      step.Status.String(),
	}
}
