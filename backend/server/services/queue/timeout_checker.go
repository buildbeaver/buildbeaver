package queue

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const (
	defaultJobTimeout          = 2 * time.Hour
	defaultTimeoutPollInterval = 5 * time.Minute
)

// timeoutCheck is an object that can be sent to the timeout service to request that all currently running
// jobs are checked to see if they have timed out. The supplied timeout duration is used in the check.
type timeoutCheck struct {
	timeout       time.Duration
	completedChan chan int // returns the number of jobs timed out
}

func newTimeoutCheck(timeout time.Duration) *timeoutCheck {
	return &timeoutCheck{
		timeout:       timeout,
		completedChan: make(chan int),
	}
}

// TimeoutChecker implements a Service to periodically check for jobs that should be failed due inactivity
// for a timeout period.
type TimeoutChecker struct {
	*util.StatefulService
	db                  *store.DB
	queueService        services.QueueService
	jobService          services.JobService
	stepService         services.StepService
	timeoutPollInterval time.Duration
	timeoutCheckChan    chan *timeoutCheck
	logger.Log
}

func NewTimeoutChecker(
	db *store.DB,
	queueService services.QueueService,
	jobService services.JobService,
	stepService services.StepService,
	logFactory logger.LogFactory,
) *TimeoutChecker {
	s := &TimeoutChecker{
		db:                  db,
		queueService:        queueService,
		jobService:          jobService,
		stepService:         stepService,
		timeoutPollInterval: defaultTimeoutPollInterval,
		timeoutCheckChan:    make(chan *timeoutCheck),
		Log:                 logFactory("TimeoutChecker"),
	}
	s.StatefulService = util.NewStatefulService(context.Background(), s.Log, s.loop)
	return s
}

func (s *TimeoutChecker) loop() {
	s.Tracef("Starting job timeout polling loop...")
	for {
		select {
		case <-s.StatefulService.Ctx().Done():
			s.Tracef("Job timeout service closed; exiting...")
			return

		case timeoutReq := <-s.timeoutCheckChan:
			// This channel is designed for use in testing; check all running jobs against an arbitrary timeout value
			nrTimedOutJobs, err := s.checkForTimeouts(timeoutReq.timeout)
			if err != nil {
				s.Errorf("Error checking jobs for timeouts: %s", err.Error())
			}
			timeoutReq.completedChan <- nrTimedOutJobs

		case <-time.After(s.timeoutPollInterval):
			nrTimedOutJobs, err := s.checkForTimeouts(defaultJobTimeout)
			if err != nil {
				s.Errorf("Error checking jobs for timeouts: %s", err.Error())
			}
			if nrTimedOutJobs > 0 {
				s.Infof("Failed %d jobs due to timeouts", nrTimedOutJobs)
			}
		}
	}
}

// Checks all currently running jobs to see if they have timed out, using the specified default timeout duration.
// Returns the number of jobs that timed out.
func (s *TimeoutChecker) checkForTimeouts(defaultTimeout time.Duration) (nrTimedOutJobs int, err error) {
	var (
		ctx          = s.Ctx()
		timedOutJobs []*models.Job
	)

	// Find the list of all jobs which have timed out
	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		// A function to find jobs with the specified status that have timed out
		findTimedOutJobs := func(statusToCheck models.WorkflowStatus) ([]*models.Job, error) {
			var results []*models.Job
			pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
			for moreResults := true; moreResults; {
				s.Tracef("checkForTimeouts: Searching database for currently running jobs")
				// TODO: Add the ability to list only timed-out jobs in the database query instead of all jobs with a given status.
				// TODO: To do this, move each of the job.Timings fields into their own column in the jobs table and add
				// TODO: the ability to filter jobs in the query based on the Timings.QueuedAt field
				runningJobs, cursor, err := s.jobService.ListByStatus(context.Background(), tx, statusToCheck, pagination)
				if err != nil {
					return nil, err
				}
				s.Tracef("checkForTimeouts: Got a page of %d jobs in search", len(runningJobs))
				for _, job := range runningJobs {
					// TODO: Support non-default timeouts on a per-job basis
					timeout := defaultTimeout
					if s.hasJobTimedOut(job, timeout) {
						results = append(results, job)
					}
				}
				if cursor != nil && cursor.Next != nil {
					pagination.Cursor = cursor.Next // move on to next page of results
				} else {
					moreResults = false
				}
			}
			return results, nil
		}

		timedOutQueuedJobs, err := findTimedOutJobs(models.WorkflowStatusQueued)
		if err != nil {
			return err
		}
		timedOutSubmittedJobs, err := findTimedOutJobs(models.WorkflowStatusSubmitted)
		if err != nil {
			return err
		}
		timedOutRunningJobs, err := findTimedOutJobs(models.WorkflowStatusRunning)
		if err != nil {
			return err
		}
		timedOutJobs = append(timedOutJobs, timedOutQueuedJobs...)
		timedOutJobs = append(timedOutJobs, timedOutSubmittedJobs...)
		timedOutJobs = append(timedOutJobs, timedOutRunningJobs...)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error searching for timed-out jobs: %w", err)
	}

	// Cancel each timed out job in a separate transaction, so failure to cancel one does not impact the others
	errorCount := 0
	timedOutCount := 0
	for _, job := range timedOutJobs {
		err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
			return s.failTimedOutJob(ctx, tx, job)
		})
		if err != nil {
			// Log error and continue
			s.Errorf("error cancelling timed-out job with ID %s: %v", job.ID, err.Error())
			errorCount++
		} else {
			timedOutCount++
		}
	}
	if errorCount > 0 {
		return timedOutCount, fmt.Errorf("error cancelling jobs: failed to cancel %d out of %d timed-out jobs", errorCount, len(timedOutJobs))
	}

	return timedOutCount, nil
}

func (s *TimeoutChecker) hasJobTimedOut(job *models.Job, timeout time.Duration) bool {
	// Jobs can time out if they are in any non-finished state (including queued, submitted, running)
	if job.Status.HasFinished() {
		return false
	}

	// If there was no time recorded when the job was queued up then we can't tell if it has timed out
	if job.Timings.QueuedAt == nil {
		return false
	}

	// Timed out if the job was first queued up more than 'timeout' ago. Using this date allows queued
	// jobs to time out, as well as submitted and running jobs.
	return time.Now().After(job.Timings.QueuedAt.Time.Add(timeout))
}

func (s *TimeoutChecker) failTimedOutJob(ctx context.Context, tx *store.Tx, job *models.Job) error {
	// Fail the job itself
	_, err := s.queueService.UpdateJobStatus(ctx, tx, job.ID, dto.UpdateJobStatus{
		Status: models.WorkflowStatusFailed,
		Error:  models.NewError(fmt.Errorf("error: job timed out")),
		ETag:   "", // fail the job regardless of whether it has been updated in the meantime
	})
	if err != nil {
		return fmt.Errorf("error updating job status: %w", err)
	}

	// Fail every step in the job that hasn't already finished
	steps, err := s.stepService.ListByJobID(ctx, tx, job.ID)
	if err != nil {
		return fmt.Errorf("error listing job steps: %w", err)
	}
	for _, step := range steps {
		if !step.Status.HasFinished() {
			_, err = s.queueService.UpdateStepStatus(ctx, tx, step.ID, dto.UpdateStepStatus{
				Status: models.WorkflowStatusFailed,
				Error:  models.NewError(fmt.Errorf("error: step failed because parent job timed out")),
				ETag:   "", // fail the step regardless of whether it has been updated in the meantime					)
			})
			if err != nil {
				return fmt.Errorf("error updating step status: %w", err)
			}
		}
	}

	s.Infof("Job %s timed out and was failed with a timeout error", job.ID)
	return nil
}

// CheckForTimeouts will instruct the timeout service to check all currently running jobs against the specified
// timeout, and fail any jobs which have been running longer than the timeout duration with a timeout error.
// Returns the number of jobs that were failed.
func (s *TimeoutChecker) CheckForTimeouts(timeout time.Duration) int {
	timeoutCheck := newTimeoutCheck(timeout)
	s.timeoutCheckChan <- timeoutCheck
	return <-timeoutCheck.completedChan
}
