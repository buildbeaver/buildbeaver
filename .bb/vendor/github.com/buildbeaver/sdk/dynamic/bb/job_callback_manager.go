package bb

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

// JobCallback is a function that can be called when a job has transitioned to a particular state.
type JobCallback func(event *JobStatusChangedEvent)

// jobSubscription records the requirement to call a JobCallback when a job has transitioned to a particular state.
type jobSubscription struct {
	// A reference to the job whose status is being waited on.
	jobRef JobReference
	// If true then the callback will only be called when the job has been completed. If onJobStatus is
	// also set then the callback will only ever be called if the job's final status matches.
	onCompleted bool
	// If set then the callback will only be called when the status has been set to this particular value.
	onJobStatus *Status
	// The function to call when an event matches the conditions of the subscription.
	callback JobCallback
	// The sequence number of the last event seen, or zero if no events have already been seen.
	// Only events with a higher sequence number will trigger the callback.
	// This field will be updated automatically if a callback is called more than once.
	lastEventSeenSequenceNumber int64
}

func newJobSubscription(
	jobRef JobReference,
	onCompleted bool,
	onJobStatus *Status,
	callback JobCallback,
) *jobSubscription {
	return &jobSubscription{
		jobRef:                      jobRef,
		onCompleted:                 onCompleted,
		onJobStatus:                 onJobStatus,
		callback:                    callback,
		lastEventSeenSequenceNumber: 0, // start with the first relevant event
	}
}

// JobCallbackManager tracks callbacks requested for when a job status changes.
type JobCallbackManager struct {
	eventManager *EventManager

	subscriptionsMutex sync.Mutex
	subscriptions      []*jobSubscription
}

func NewJobCallbackManager(eventManager *EventManager) *JobCallbackManager {
	return &JobCallbackManager{
		eventManager:  eventManager,
		subscriptions: []*jobSubscription{},
	}
}

func (b *JobCallbackManager) AddSubscription(subscription *jobSubscription) {
	b.subscriptionsMutex.Lock()
	defer b.subscriptionsMutex.Unlock()

	b.subscriptions = append(b.subscriptions, subscription)
}

// BlockAndProcessCallbacks will block and call callbacks until either all job callbacks have been run, or
// the build is finished.
// If there are no outstanding callbacks then this function will return immediately.
func (b *JobCallbackManager) BlockAndProcessCallbacks() {
	Log(LogLevelTrace, "BlockAndProcessCallbacks() called")
	for {
		// Make a map of job reference to list of outstanding subscriptions, and a list of all jobs to wait for.
		// If new subscriptions are added then we'll catch them next time around the loop.
		subscriptionsMap := make(map[string][]*jobSubscription)
		var jobsToWaitFor []JobReference
		b.subscriptionsMutex.Lock()
		for _, subscription := range b.subscriptions {
			jobRef := subscription.jobRef
			// Add subscription to the correct list (or a new list) within the map
			_, found := subscriptionsMap[jobRef.String()]
			if found {
				subscriptionsMap[jobRef.String()] = append(subscriptionsMap[jobRef.String()], subscription)
			} else {
				// Make a new list for this jobName, containing only the supplied callback
				subscriptionList := []*jobSubscription{subscription}
				subscriptionsMap[jobRef.String()] = subscriptionList
				// Add the jobName to the list of names to wait for
				jobsToWaitFor = append(jobsToWaitFor, jobRef)
			}
		}
		b.subscriptionsMutex.Unlock() // release the lock now we've copied the data from b.subscriptions
		Log(LogLevelTrace, fmt.Sprintf("BlockAndProcessCallbacks() has %d jobs to wait on", len(jobsToWaitFor)))

		// If there are no more outstanding subscriptions, this function is done
		if len(jobsToWaitFor) == 0 {
			return
		}

		// subscriptionFilter is a filter function that returns true for events that match one of our subscriptions.
		// Uses the subscriptionsMap above so does not require any lock to be held.
		subscriptionFilter := func(event *client.Event) bool {
			// Only match 'job status changed' events
			if event.Type != EventTypeJobStatusChanged.String() {
				return false
			}
			eventJobRef := GetJobRefOrNilFromEvent(event)
			if eventJobRef == nil {
				return false // events without a job ref don't match
			}
			// Find the subscriptions relating to this job ref and look for any that match the event
			subscriptions := subscriptionsMap[eventJobRef.String()]
			for _, subscription := range subscriptions {
				shouldRun, shouldDelete := b.evaluateJobSubscription(subscription, event)
				if shouldRun || shouldDelete {
					// We need to process the event if we want to run a callback and/or delete a subscription
					return true
				}
			}
			return false // No matching subscription found; not interested in this event
		}

		// Wait for an event matching one of our subscriptions (or that would cause a subscription to be deleted)
		event, err := b.eventManager.WaitForJobStatusChangedEvent(context.Background(), subscriptionFilter, jobsToWaitFor...)
		if err != nil {
			Log(LogLevelWarn, fmt.Sprintf("Unable to wait for jobs: %s", err.Error()))
			return
		}

		b.runCallbacksForEvent(event)
	}
}

// evaluateJobSubscription evaluates a subscription in the context of a specific event.
// Returns whether to run the subscription's callback, and whether to delete the subscription
// (either because its callback should be called and it's a one-off, or it can no longer ever be called).
func (b *JobCallbackManager) evaluateJobSubscription(
	subscription *jobSubscription,
	event *client.Event,
) (shouldRun bool, shouldDelete bool) {
	// First check if the event is for a different job; if so then ignore the event
	eventJobRef := GetJobRefOrNilFromEvent(event)
	if eventJobRef == nil {
		return false, false // not a job-related event
	}
	if !eventJobRef.Equals(subscription.jobRef) {
		return false, false
	}

	// Check if the event has already been seen/processed; if so then ignore the event
	if event.SequenceNumber <= subscription.lastEventSeenSequenceNumber {
		return false, false
	}

	eventJobStatus := Status(event.Payload)

	if eventJobStatus.HasFinished() {
		// Job has finished, so one way or another this subscription is done
		shouldDelete = true
		shouldRun = subscription.onJobStatus == nil || *subscription.onJobStatus == eventJobStatus
		return shouldRun, shouldDelete
	}

	// Job not finished yet; if subscription requires it to be finished then ignore the event
	if subscription.onCompleted {
		return false, false
	}

	// Check if subscription is looking for a specific status
	if subscription.onJobStatus != nil {
		if *subscription.onJobStatus == eventJobStatus {
			// Found the status the subscription is looking for; call the callback and we're done
			return true, true
		} else {
			// Status is not the one the subscription is looking for; ignore the event
			return false, false
		}
	}

	// Subscription is not for any specific status and the job isn't finished; run the callback but don't
	// delete it so that it can be called again for the next status change
	return true, false
}

// runCallbacksForEvent runs all registered callbacks that match the event.
func (b *JobCallbackManager) runCallbacksForEvent(event *client.Event) {
	Log(LogLevelTrace, fmt.Sprintf("runCallbacksForEvent() called, event %v", event))

	// Work out which subscription callbacks to run, and which subscriptions to keep vs delete
	var subscriptionsToRun []*jobSubscription
	subscriptionsToKeep := make([]*jobSubscription, 0, len(b.subscriptions))
	b.subscriptionsMutex.Lock()
	for _, subscription := range b.subscriptions {
		shouldRun, shouldDelete := b.evaluateJobSubscription(subscription, event)
		if shouldRun {
			subscriptionsToRun = append(subscriptionsToRun, subscription)
		}
		if !shouldDelete {
			if shouldRun {
				// If we're running the callback and still keeping the subscription then we must update the last
				// event seen, so we don't deliver the same event again next time (i.e. an infinite loop)
				subscription.lastEventSeenSequenceNumber = event.SequenceNumber
			}
			subscriptionsToKeep = append(subscriptionsToKeep, subscription)
		}
	}
	// Update the subscription list to remove those we want to delete.
	// Once this change is made we are committed to calling all callbacks in the subscriptionsToRun list.
	Log(LogLevelTrace, fmt.Sprintf("runCallbacksForEvent() running %d callbacks, keeping %d subscriptions out of %d", len(subscriptionsToRun), len(subscriptionsToKeep), len(b.subscriptions)))
	if len(subscriptionsToKeep) < len(b.subscriptions) {
		b.subscriptions = subscriptionsToKeep
	}
	b.subscriptionsMutex.Unlock()

	// Convert event to a type suitable for passing to callbacks
	jobStatusChangedEvent, err := NewJobStatusChangedEvent(event)
	if err != nil {
		Log(LogLevelError, err.Error())
		return
	}

	// Run callbacks after releasing the lock
	for _, subscription := range subscriptionsToRun {
		Log(LogLevelTrace, fmt.Sprintf("runCallbacksForEvent() calling callback for job '%s'", subscription.jobRef.String()))
		subscription.callback(jobStatusChangedEvent)
	}
}
