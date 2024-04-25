package bb

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	// StatusQueued indicates the item has been created and is waiting to be processed.
	StatusQueued Status = "queued"
	// StatusSubmitted indicates the item has been handed to a worker.
	StatusSubmitted Status = "submitted"
	// StatusRunning indicates the item is being processed.
	StatusRunning Status = "running"
	// StatusFailed indicates the item has failed during processing.
	StatusFailed Status = "failed"
	// StatusSucceeded indicates the item has successfully finished being processed.
	StatusSucceeded Status = "succeeded"
	// StatusCanceled indicates the item was canceled before it was ever processed.
	StatusCanceled Status = "canceled"
	// StatusUnknown indicates the item is in an unknown state.
	StatusUnknown Status = "unknown"
)

// HasFinished returns true if the status indicates the workflow has finished, either succeeded, failed, or canceled.
func (s Status) HasFinished() bool {
	return s == StatusFailed || s == StatusSucceeded || s == StatusCanceled
}

// HasFailed returns true if the status indicates the workflow has either failed or been canceled.
func (s Status) HasFailed() bool {
	return s == StatusFailed || s == StatusCanceled
}
