package work_queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

var (
	// NrWorkItemProcessors is the number of goroutines to start for processing work items.
	NrWorkItemProcessors = 10

	// PollInterval determines how often each processor will poll the database for new work items
	PollInterval = 2 * time.Second

	// workItemUpdateRetryAttempts is the number of times we will retry when attempting to update
	// a work item in the database
	workItemUpdateRetryAttempts = 60
	// workItemUpdateAttemptInterval is the delay between attempts to update a work item in the database
	workItemUpdateAttemptInterval = 2 * time.Second
)

type handlerRegistration struct {
	handler                 services.WorkItemHandler
	timeout                 time.Duration
	backoffAlgorithm        services.BackoffAlgorithm
	keepFailedWorkItems     bool
	keepSuccessfulWorkItems bool
}

type WorkQueueService struct {
	db                 *store.DB
	workItemStore      store.WorkItemStore
	workItemStateStore store.WorkItemStateStore

	// The ID that work queue processor Goroutines will use when allocating work items
	processorID models.WorkItemProcessorID

	handlerRegistrations      map[models.WorkItemType]*handlerRegistration
	handlerRegistrationsMutex sync.RWMutex

	started, shutdown bool
	startStopMutex    sync.Mutex

	// Channel closed when processor goroutines should shut down
	requestShutdownChan chan bool
	// Channel that individual processor Goroutines can use to report that they have shut down
	// by sending their processorNr.
	shutdownCompleteChan chan int

	logger.Log
}

func NewWorkQueueService(
	db *store.DB,
	workItemStore store.WorkItemStore,
	workItemStateStore store.WorkItemStateStore,
	logFactory logger.LogFactory,
) *WorkQueueService {
	s := &WorkQueueService{
		db:                   db,
		workItemStore:        workItemStore,
		workItemStateStore:   workItemStateStore,
		processorID:          models.NewWorkItemProcessorID(),
		handlerRegistrations: make(map[models.WorkItemType]*handlerRegistration),
		requestShutdownChan:  make(chan bool),
		shutdownCompleteChan: make(chan int),
		Log:                  logFactory("WorkQueueService"),
	}
	return s
}

// Start will start one or more goroutines to begin processing work items from the queue.
func (s *WorkQueueService) Start() {
	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()

	if s.shutdown {
		panic("Can not start WorkQueueService again once it has been shut down")
	}
	if s.started {
		s.Warn("WorkQueueService.Start() called but already started")
		return
	}

	s.Infof("Starting %d work item processor(s) for work queue", NrWorkItemProcessors)
	for processorNr := 1; processorNr <= NrWorkItemProcessors; processorNr++ {
		go s.processorLoop(processorNr)
	}
	s.started = true
}

// Shutdown will stop any running goroutines for processing work items.
func (s *WorkQueueService) Shutdown() {
	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()

	if s.shutdown {
		s.Warn("WorkQueueService.Stop() called but already shut down")
		return
	}

	s.Tracef("Requesting shutdown of work item processors")
	close(s.requestShutdownChan)

	s.Infof("Waiting for %d work item processor(s) to shut down", NrWorkItemProcessors)
	for processorNr := 1; processorNr <= NrWorkItemProcessors; processorNr++ {
		<-s.shutdownCompleteChan
	}

	s.Infof("All work item processors shut down successfully")
	s.shutdown = true
}

// AddWorkItem adds a new Work Item to the queue to be processed.
func (s *WorkQueueService) AddWorkItem(ctx context.Context, txOrNil *store.Tx, workItem *models.WorkItem) error {
	now := models.NewTime(time.Now())
	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Find or create a state record and lock the row.
		// If workItem.ConcurrencyKey is empty then a new state record will always be created, making
		// this work item independent of any other.
		state, err := s.workItemStateStore.FindOrCreateAndLockRow(ctx, tx,
			models.NewWorkItemState(now, workItem.ConcurrencyKey))
		if err != nil {
			return fmt.Errorf("error finding or creating state record for concurrency key %s: %w", workItem.ConcurrencyKey, err)
		}

		// Create the work item itself
		workItem.StateID = state.ID
		return s.workItemStore.Create(ctx, tx, workItem)
	})
	if err != nil {
		return err
	}

	s.Tracef("Queued work item %q of type %q", workItem.ID, workItem.Type)
	return nil
}

// RegisterHandler registers a handler function to process work items of the specified type.
// Only one handler function can be registered for each type; subsequent calls to RegisterHandler for that
// type will return an error.
//
// A timeout value MUST be supplied and must correspond to the longest time that any work item of this type should
// take to process. After the timeout period the context passed to the handler will expire, and the handler
// should cut short any work currently underway and return an error. After twice the timeout period the handler,
// or the server it is running on, will be assumed to have locked up or crashed, and the work item will become
// available for processing again by another server or handler.
//
// The specified backoff algorithm will be used to determine when and how often to retry if the handler
// returns an error that can be retried. If nil is supplied for the backoff algorithm then a default
// exponential backoff algorithm will be used.
//
// If keepFailedWorkItems is true then work items that have permanently failed will remain in the database,
// otherwise they will be deleted on failure.
//
// If keepSuccessfulWorkItems is true then work items that have completed successfully will remain in the
// database, otherwise they will be deleted on completion. Setting this to true may result in a large number
// of database records building up over time.
func (s *WorkQueueService) RegisterHandler(
	workItemType models.WorkItemType,
	handler services.WorkItemHandler,
	timeout time.Duration,
	backoffAlgorithm services.BackoffAlgorithm,
	keepFailedWorkItems bool,
	keepSuccessfulWorkItems bool,
) error {
	if timeout == time.Duration(0) {
		panic(fmt.Sprintf("Zero timeout supplied to WorkQueueService.RegisterHandler() for work item type %s", workItemType))
	}

	// Default backoff algorithm is max of 10 attempts, interval between 1 second and 1 minute
	if backoffAlgorithm == nil {
		backoffAlgorithm = ExponentialBackoff(10, 1*time.Second, 1*time.Minute)
	}

	s.Infof("Registering work item handler for type %s", workItemType)

	s.handlerRegistrationsMutex.Lock()
	defer s.handlerRegistrationsMutex.Unlock()

	_, handlerAlreadyExists := s.handlerRegistrations[workItemType]
	if handlerAlreadyExists {
		return fmt.Errorf("WorkItemHandler already registered for WorkItem Type %s", workItemType)
	}

	s.handlerRegistrations[workItemType] = &handlerRegistration{
		handler:                 handler,
		timeout:                 timeout,
		backoffAlgorithm:        backoffAlgorithm,
		keepFailedWorkItems:     keepFailedWorkItems,
		keepSuccessfulWorkItems: keepSuccessfulWorkItems,
	}
	return nil
}

// getHandler Finds and returns handler registration info for the given work item type, or nil if
// no handler is registered.
func (s *WorkQueueService) getHandler(workItemType models.WorkItemType) *handlerRegistration {
	s.handlerRegistrationsMutex.RLock()
	defer s.handlerRegistrationsMutex.RUnlock()

	handlerRegistration, exists := s.handlerRegistrations[workItemType]
	if !exists {
		return nil
	}
	return handlerRegistration
}

// listAllRegisteredTypes returns a list of all work item types with registered handlers.
func (s *WorkQueueService) listAllRegisteredTypes() []models.WorkItemType {
	s.handlerRegistrationsMutex.RLock()
	defer s.handlerRegistrationsMutex.RUnlock()

	results := make([]models.WorkItemType, len(s.handlerRegistrations))
	i := 0
	for workItemType := range s.handlerRegistrations {
		results[i] = workItemType
		i++
	}
	return results
}

// processorLoop sits in a loop looking for work items to process. Each work item is processed by calling the
// handler registered for its type.
// The loop terminates and the function returns when requestShutdownChan is closed.
func (s *WorkQueueService) processorLoop(processorNr int) {
	for {
		// Check for shutdown event - continue on immediately unless told to shut down
		select {
		case <-s.requestShutdownChan:
			s.Tracef("Work item processor %d shutting down", processorNr)
			s.shutdownCompleteChan <- processorNr
			return
		default:
		}

		workItem, err := s.allocateWorkItem()
		if err != nil {
			// Ignore errors, treat the same as not having a work item
			s.Errorf("Error attempting to allocate a work item: %s", err)
			workItem = nil
		}
		if workItem != nil {
			s.processWorkItem(workItem)
		} else {
			// No work item to process, so poll again after poll interval has elapsed
			time.Sleep(PollInterval)
		}
	}
}

// allocateWorkItem pulls a work item off the queue, and updates it to be allocated to this processor.
// Returns the database-layer records for the allocated work item.
// If no work items are ready then this function will return immediately with nil for the work item.
func (s *WorkQueueService) allocateWorkItem() (*models.WorkItemRecords, error) {
	var (
		err  error
		item *models.WorkItemRecords
	)

	// Start a new transaction to atomically read and allocate a work item using row locking
	err = s.db.WithTx(context.Background(), nil, func(tx *store.Tx) error {
		// Only find work items with registered types
		registeredTypes := s.listAllRegisteredTypes()

		// Find a work item and the corresponding state record, and lock the state record
		now := models.NewTime(time.Now())
		item, err = s.workItemStateStore.FindQueuedWorkItem(context.Background(), tx, now, registeredTypes)
		if err != nil {
			if gerror.IsNotFound(err) {
				// No work items are available
				item = nil
				return nil
			} else {
				return fmt.Errorf("error reading work item from database: %w", err)
			}
		}

		// Update 'now' since we could have taken a long time to get the lock on the work item
		now = models.NewTime(time.Now())

		// Find handler for this work item
		handlerRegistration := s.getHandler(item.Record.Type)
		if handlerRegistration == nil {
			// We only find items for registered types so this shouldn't happen unless the handler has
			// just been removed. Just say no work items are available and leave it until next poll.
			s.Warnf("No handler available for work item type %q, ignoring work item", item.Record.Type)
			item = nil
			return nil
		}

		// Update work item state record to be allocated to this work processor
		s.Tracef("Found work item of type %s, updating to allocate", item.Record.Type)
		allocationTTL := allocationTTLFromTimeout(handlerRegistration.timeout)
		item.State.AttemptsSoFar++
		item.State.AllocatedAt = &now
		item.State.AllocatedTo = &s.processorID
		item.State.AllocatedUntil = models.NewTimePtr(now.Add(allocationTTL))
		err = s.workItemStateStore.Update(context.Background(), tx, item.State)
		if err != nil {
			return fmt.Errorf("error updating work item state record to allocate: %w", err)
		}
		// Also update the work item record, just for visibility's sake
		item.Record.Status = "processing"
		err = s.workItemStore.Update(context.Background(), tx, item.Record)
		if err != nil {
			return fmt.Errorf("error updating work item record to record allocatation: %w", err)
		}

		return nil // success - we have a work item allocated
	})
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil // No work item was found,
	}
	return item, nil
}

// allocationTTLFromTimeout calculates the Time-To-Live value for allocating a work item, after which the
// work item will be considered 'released' from the work item processor and available to be allocated to another
// processor. This will be based on (and longer than) the timeout that applies to the event handler.
func allocationTTLFromTimeout(timeout time.Duration) time.Duration {
	// Double the timeout value should be safe...
	ttl := 2 * timeout
	// ...but regardless, make sure we allocate for a reasonable minimum amount of time
	const minTTL = 1 * time.Minute
	if ttl < minTTL {
		ttl = minTTL
	}
	return ttl
}

func (s *WorkQueueService) processWorkItem(item *models.WorkItemRecords) {
	// Find handler
	handlerRegistration := s.getHandler(item.Record.Type)
	if handlerRegistration == nil {
		// We only find items for registered types so this shouldn't happen unless the handler has
		// just been removed. We can't process the work item, so leave it until next poll.
		s.Warnf("No handler available for work item type %q, ignoring work item", item.Record.Type)
		return
	}

	// Make a copy of the work item to pass to the handler, so it can't change anything
	workItemCopy := &models.WorkItem{}
	*workItemCopy = *item.Record

	// Create a context with a timeout for the handler, in case it takes too long
	handlerContext, cancelFunc := context.WithTimeout(context.Background(), handlerRegistration.timeout)
	defer cancelFunc() // release resources

	// Call the handler
	s.Tracef("Calling handler for work item of type %s", workItemCopy.Type)
	canRetry, err := handlerRegistration.handler(handlerContext, workItemCopy)
	s.Tracef("Handler for work item of type %s returned, err = %v", workItemCopy.Type, err)
	if err != nil {
		if canRetry {
			s.workItemFailed(item, err, handlerRegistration.backoffAlgorithm, workItemCopy, handlerRegistration.keepFailedWorkItems)
		} else {
			s.workItemFailedPermanently(item, err, handlerRegistration.keepFailedWorkItems)
		}
	} else {
		s.workItemDone(item, handlerRegistration.keepSuccessfulWorkItems)
	}
}

func (s *WorkQueueService) workItemFailedPermanently(item *models.WorkItemRecords, processingError error, keepRecord bool) {
	// Log the error
	s.Errorf("work item failed permanently; type %q, ID %q: %s", item.Record.Type, item.Record.ID, processingError.Error())

	if keepRecord {
		item.Record.Status = "failed permanently: " + processingError.Error()
		item.Record.CompletedAt = models.NewTimePtr(time.Now())
	}

	// Reset the state ready to process the next work item for this state (if any)
	item.State.AllocatedTo = nil
	item.State.AllocatedUntil = nil
	item.State.AttemptsSoFar = 0
	item.State.NotBefore = nil
	item.State.AllocatedAt = nil

	if keepRecord {
		s.updateWorkItem(item)
	} else {
		s.deleteWorkItemAndUpdateOrDeleteState(item)
	}
}

func (s *WorkQueueService) workItemFailed(
	item *models.WorkItemRecords,
	processingError error,
	backoffAlgorithm services.BackoffAlgorithm,
	workItemCopy *models.WorkItem,
	keepRecordIfFailedPermanently bool,
) {
	// Count backoff time from now, which is approximately when the current attempt *ended*.
	// This ensures a time gap between attempts, giving other work items a chance to be processed.
	lastAttemptAt := models.NewTime(time.Now())

	// Run backoff algorithm to decide whether and when to retry.
	// Use the deep copy of the work item in case the algorithm decides to change it.
	s.Tracef("Calling backoff algorithm, attemptsSoFar=%d", item.State.AttemptsSoFar)
	notBeforeTime := backoffAlgorithm(item.State.AttemptsSoFar, lastAttemptAt.Time, workItemCopy)
	if notBeforeTime == nil {
		// nil means don't retry because there have been too many errors
		wrappedErr := fmt.Errorf("giving up after %d errors; last error: %w", item.State.AttemptsSoFar, processingError)
		s.workItemFailedPermanently(item, wrappedErr, keepRecordIfFailedPermanently)
		return
	}
	notBefore := models.NewTimePtr(*notBeforeTime)
	s.Tracef("Retrying after %v, not before %v", notBefore.Time.Sub(lastAttemptAt.Time), notBefore)

	item.Record.Status = "awaiting retry, last error: " + processingError.Error()
	item.State.AllocatedTo = nil
	item.State.AllocatedUntil = nil
	item.State.NotBefore = notBefore

	s.updateWorkItem(item)
}

func (s *WorkQueueService) workItemDone(item *models.WorkItemRecords, keepRecord bool) {
	if keepRecord {
		item.Record.Status = "done"
		item.Record.CompletedAt = models.NewTimePtr(time.Now())
	}

	// Reset the state ready to process the next work item for this state (if any)
	item.State.AllocatedTo = nil
	item.State.AllocatedUntil = nil
	item.State.AttemptsSoFar = 0
	item.State.NotBefore = nil
	item.State.AllocatedAt = nil

	if keepRecord {
		s.updateWorkItem(item)
	} else {
		s.deleteWorkItemAndUpdateOrDeleteState(item)
	}
}

// retryTransactionUntilDone runs the supplied function inside a transaction until either it succeeds
// or it returns canRetry as false. Each time the function returns an error, the transaction will be
// rolled back ready to try again.
//
// After maxRetries attempts (with a gap of retryInterval between attempts) we give up and log an error,
// then return. For this reason maxRetries should normally be a large number.
func (s *WorkQueueService) retryTransactionUntilDone(
	taskName string,
	maxRetries int,
	retryInterval time.Duration,
	fn func(tx *store.Tx) (canRetry bool, err error),
) {
	var err error
	done := false
	for attempt := 1; attempt <= maxRetries && !done; attempt++ {
		var canRetry bool
		err = s.db.WithTx(context.Background(), nil, func(tx *store.Tx) error {
			canRetry, err = fn(tx)
			return err
		})
		if err != nil && canRetry {
			s.Warnf("error executing '%s', will retry after %v: %s", taskName, retryInterval, err.Error())
			time.Sleep(retryInterval)
			done = false
		} else {
			done = true
		}
	}
	if err != nil {
		// Log the error and give up
		s.Errorf("Failed %d times to execute '%s', giving up; last error: %s", maxRetries, taskName, err.Error())
	}
}

// updateWorkItemWithRetry updates both records for the specified work item in the database.
// The new values for fields should already have been set prior to calling.
//
// If an update fails then it will be retried a number of times until it succeeds or
// a maximum number of retries is reached, at which point the error is logged and ignored.
// A NotFoundError will not be retried as the update will never succeed.
func (s *WorkQueueService) updateWorkItem(item *models.WorkItemRecords) {
	// Update both records in a single transaction
	s.retryTransactionUntilDone("update work item", workItemUpdateRetryAttempts, workItemUpdateAttemptInterval,
		func(tx *store.Tx) (canRetry bool, err error) {
			s.Trace("Updating work item...")
			err = s.workItemStore.Update(context.Background(), tx, item.Record)
			if err != nil {
				if gerror.IsNotFound(err) {
					return false, fmt.Errorf("error updating work item, not found: %w", err)
				}
				return true, fmt.Errorf("error updating work item: %w", err)
			}
			err = s.workItemStateStore.Update(context.Background(), tx, item.State)
			if err != nil {
				if gerror.IsNotFound(err) {
					return false, fmt.Errorf("error updating work item state record, not found: %w", err)
				}
				return true, fmt.Errorf("error updating work item state record: %w", err)
			}
			return false, nil // success
		},
	)
}

// deleteWorkItemAndUpdateOrDeleteState deletes the database record for a work item, and either updates or
// deletes the corresponding work item state record.
//
// The work item state record will be deleted if there are no other work items sharing the state, or
// updated ready for the next work item if there are other work items using the state.
// The new values for item.State fields should already have been set prior to calling, ready for an update.
//
// If a database operation fails then it will be retried a number of times until it succeeds or
// a maximum number of retries is reached, at which point the error is logged and ignored.
// A NotFoundError will not be retried as the update will never succeed.
func (s *WorkQueueService) deleteWorkItemAndUpdateOrDeleteState(item *models.WorkItemRecords) {
	ctx := context.Background()

	// Update/delete both records in a single transaction
	s.retryTransactionUntilDone("delete work item", workItemUpdateRetryAttempts, workItemUpdateAttemptInterval,
		func(tx *store.Tx) (canRetry bool, err error) {
			err = s.workItemStateStore.LockRowForUpdate(ctx, tx, item.State.ID)
			if err != nil {
				if gerror.IsNotFound(err) {
					return false, fmt.Errorf("error locking work item state ready to delete work item, state not found: %w", err)
				}
				return true, fmt.Errorf("error locking work item state ready to delete work item: %w", err)
			}

			// Always delete the work item.
			// Do this before deleting the work item state record to avoid violating referential integrity constraints
			s.Trace("Deleting work item...")
			err = s.workItemStore.Delete(ctx, tx, item.Record.ID)
			if err != nil {
				return true, fmt.Errorf("error deleting work item: %w", err)
			}

			// Check if this was the last work item for the state record
			nrWorkItems, err := s.workItemStateStore.CountWorkItems(ctx, tx, item.State.ID)
			if err != nil {
				return true, err
			}
			if nrWorkItems == 0 {
				// Delete the unused work item state record
				s.Trace("Deleting work item state record...")
				err = s.workItemStateStore.Delete(context.Background(), tx, item.State.ID)
				if err != nil {
					return true, fmt.Errorf("error deleting work item state record: %w", err)
				}
			} else {
				// State still used by at least one other work item, update it
				err = s.workItemStateStore.Update(context.Background(), tx, item.State)
				if err != nil {
					if gerror.IsNotFound(err) {
						return false, fmt.Errorf("error updating work item state record, not found: %w", err)
					}
					return true, fmt.Errorf("error updating work item state record: %w", err)
				}
			}
			return false, nil // success
		},
	)
}
