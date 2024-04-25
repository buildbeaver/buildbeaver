package bb

import (
	"fmt"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

type EventType string

func (s EventType) String() string {
	return string(s)
}

const (
	// EventTypeBuildStatusChanged is a type of event to notify subscribers that the status of a build has changed.
	// The event resource ID should be the ID of the build. The data should be a string of type WorkflowStatus.
	EventTypeBuildStatusChanged EventType = "BuildStatusChanged"

	// EventTypeJobStatusChanged is a type of event to notify subscribers that the status of a Job has changed.
	// The event resource ID should be the ID of the job. The data should be a string of type WorkflowStatus.
	EventTypeJobStatusChanged EventType = "JobStatusChanged"

	// EventTypeStepStatusChanged is a type of event to notify subscribers that the status of a Step has changed.
	// The event resource ID should be the ID of the step. The data should be a string of type WorkflowStatus.
	EventTypeStepStatusChanged EventType = "StepStatusChanged"
)

type JobStatusChangedEvent struct {
	// The ID of the Job whose status has changed.
	JobID JobID
	// The name of the workflow the job is part of, or empty string for the default workflow.
	Workflow ResourceName
	// The name of the Job whose status has changed.
	JobName ResourceName
	// The new status of the Job
	JobStatus Status
	// The ID of the build this Job is a part of
	BuildID BuildID
	// The event data as it was delivered to the SDK.
	RawEvent *client.Event
}

func NewJobStatusChangedEvent(rawEvent *client.Event) (*JobStatusChangedEvent, error) {
	if rawEvent == nil {
		return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event: event pointer is nil")
	}
	if rawEvent.Type != EventTypeJobStatusChanged.String() {
		return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event of type %s; should be %s", rawEvent.Type, EventTypeJobStatusChanged)
	}
	jobID, err := ParseJobID(rawEvent.ResourceId)
	if err != nil {
		return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event: Job ID not valid: %w", err)
	}
	var workflow ResourceName
	if rawEvent.Workflow != nil && *rawEvent.Workflow != "" {
		workflow, err = ParseResourceName(*rawEvent.Workflow)
		if err != nil {
			return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event: workflow not valid: %w", err)
		}
	}
	var rawJobName string
	if rawEvent.JobName != nil && *rawEvent.JobName != "" {
		rawJobName = *rawEvent.JobName
	} else {
		rawJobName = rawEvent.ResourceName
	}
	jobName, err := ParseResourceName(rawJobName)
	if err != nil {
		return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event: job and resouce name fields not valid: %w", err)
	}
	jobStatus := Status(rawEvent.Payload)
	buildID, err := ParseBuildID(rawEvent.BuildId)
	if err != nil {
		return nil, fmt.Errorf("error attempting to construct JobStatusChangedEvent object from event: Build ID not valid: %w", err)
	}

	return &JobStatusChangedEvent{
		JobID:     jobID,
		Workflow:  workflow,
		JobName:   jobName,
		JobStatus: jobStatus,
		BuildID:   buildID,
		RawEvent:  rawEvent,
	}, nil
}
