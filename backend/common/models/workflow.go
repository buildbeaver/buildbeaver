package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

const (
	// WorkflowStatusQueued indicates the item has been created and is waiting to be processed.
	WorkflowStatusQueued WorkflowStatus = "queued"
	// WorkflowStatusSubmitted indicates the item has been handed to a worker.
	WorkflowStatusSubmitted WorkflowStatus = "submitted"
	// WorkflowStatusRunning indicates the item is being processed.
	WorkflowStatusRunning WorkflowStatus = "running"
	// WorkflowStatusFailed indicates the item has failed during processing.
	WorkflowStatusFailed WorkflowStatus = "failed"
	// WorkflowStatusSucceeded indicates the item has successfully finished being processed.
	WorkflowStatusSucceeded WorkflowStatus = "succeeded"
	// WorkflowStatusCanceled indicates the item was canceled before it was ever processed.
	WorkflowStatusCanceled WorkflowStatus = "canceled"
	// WorkflowStatusUnknown indicates the item is in an unknown state.
	WorkflowStatusUnknown WorkflowStatus = "unknown"
)

var workflowStatuses = map[string]WorkflowStatus{
	string(WorkflowStatusQueued):    WorkflowStatusQueued,
	string(WorkflowStatusSubmitted): WorkflowStatusSubmitted,
	string(WorkflowStatusRunning):   WorkflowStatusRunning,
	string(WorkflowStatusFailed):    WorkflowStatusFailed,
	string(WorkflowStatusSucceeded): WorkflowStatusSucceeded,
	string(WorkflowStatusCanceled):  WorkflowStatusCanceled,
	string(WorkflowStatusUnknown):   WorkflowStatusUnknown,
}

type WorkflowStatus string

func (s WorkflowStatus) Valid() bool {
	_, ok := workflowStatuses[string(s)]
	return ok
}

// HasFinished returns true if the workflow has finished either in a
// successful, failure, or canceled state
func (s WorkflowStatus) HasFinished() bool {
	return s == WorkflowStatusFailed || s == WorkflowStatusSucceeded || s == WorkflowStatusCanceled
}

func (s WorkflowStatus) String() string {
	return string(s)
}

// ToGitHubState translates a Workflow (build) Status into one of GitHub's valid State strings.
// According to GitHub documentation: "Can be one of error, failure, pending, or success."
func (s WorkflowStatus) ToGitHubState() string {
	switch s {
	case WorkflowStatusQueued:
		return "pending"
	case WorkflowStatusSubmitted:
		return "pending"
	case WorkflowStatusRunning:
		return "pending"
	case WorkflowStatusFailed:
		return "failure"
	case WorkflowStatusSucceeded:
		return "success"
	case WorkflowStatusCanceled:
		return "failure"
	case WorkflowStatusUnknown:
		return "error"
	default:
		return "error"
	}
}

func (s *WorkflowStatus) Scan(src interface{}) error {
	if src == nil {
		*s = WorkflowStatusUnknown
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type for workflow status: %[1]T (%[1]v)", src)
	}
	status, ok := workflowStatuses[t]
	if !ok {
		*s = WorkflowStatusUnknown
		return nil
	}
	*s = status
	return nil
}

func (s WorkflowStatus) Value() (driver.Value, error) {
	return string(s), nil
}

type WorkflowTimings struct {
	QueuedAt    *Time `json:"queued_at"`
	SubmittedAt *Time `json:"submitted_at"`
	RunningAt   *Time `json:"running_at"`
	FinishedAt  *Time `json:"finished_at"`
	CanceledAt  *Time `json:"canceled_at"`
}

func (m *WorkflowTimings) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m WorkflowTimings) Value() (driver.Value, error) {
	buf, err := json.Marshal(&m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
