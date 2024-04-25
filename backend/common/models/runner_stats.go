package models

// RunnerStats stores statistics about the activities of a runner.
type RunnerStats struct {
	SuccessfulPollCount int64 `json:"successful_poll_count"`
	FailedPollCount     int64 `json:"failed_poll_count"`
}
