package bb

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

const (
	defaultEventPollInterval = 2 * time.Second
	defaultEventLimit        = 100
)

type subscriberID int64

type ContextFactory func() context.Context

// jobIDReference contains a job reference (workflow and job name), together with the job's ID.
type jobIDReference struct {
	JobReference
	jobID ResourceID
}

func newJobIDReferenceForJobGraph(jGraph *client.JobGraph) jobIDReference {
	return jobIDReference{
		JobReference: JobReference{
			Workflow: ResourceName(jGraph.Job.GetWorkflow()),
			JobName:  ResourceName(jGraph.Job.GetName()),
		},
		jobID: ResourceID(jGraph.Job.GetId()),
	}
}

type EventManager struct {
	*StatefulService

	apiClient          *client.APIClient
	authContextFactory ContextFactory // Function to produce a context that includes authentication info
	accessToken        AccessToken    // JWT for accessing dynamic API
	buildID            BuildID        // ID of the build this event manager is tracking
	dynamicJobID       JobID          // ID of the dynamic job this event manager is running within
	eventPollInterval  time.Duration

	lastEventSequenceNumber int64 // should only be accessed by loop Goroutine

	eventsMutex      sync.RWMutex     // covers events, knownJobs and workflowJobStats
	events           []client.Event   // a list of all known events for the build
	knownJobs        []jobIDReference // list of jobs known to exist (especially those created by this SDK)
	workflowJobStats WorkflowStatsMap // maps workflow name to job stats, based on events received

	// A list of channels to notify when new events come in
	subscriberMutex  sync.Mutex
	nextSubscriberID subscriberID
	subscriberChans  map[subscriberID]chan int
}

// NewEventManager creates an event manager that will poll for events for the specified build.
// The returned event manager will automatically start polling immediately, and will continue until
// Stop() is called.
func NewEventManager(
	apiClient *client.APIClient,
	authContextFactory ContextFactory,
	buildID BuildID,
	dynamicJobID JobID,
	logger Logger,
) *EventManager {
	m := &EventManager{
		apiClient:          apiClient,
		authContextFactory: authContextFactory,
		buildID:            buildID,
		dynamicJobID:       dynamicJobID,
		eventPollInterval:  defaultEventPollInterval,
		workflowJobStats:   make(WorkflowStatsMap),
		nextSubscriberID:   1,
		subscriberChans:    make(map[subscriberID]chan int),
	}
	m.StatefulService = NewStatefulService(context.Background(), logger, "EventService", m.loop)
	m.Start()

	return m
}

// EventFilter is a function that returns true if the supplied event meets the criteria being evaluated
// when waiting for a job-related event. An EventFilter function should execute very quickly as it will be
// called frequently.
type EventFilter func(event *client.Event) bool

// WaitForJobStatusChangedEvent waits for a JobStatusChanged event for any of the specified jobs, that passes the supplied filter.
// Jobs with the specified workflows/names do not need to exist at the time of calling.
// If any suitable event already has been received then the function will immediately return the first suitable event.
// If no suitable event has already been received then the function will wait until a suitable event is received.
// If the caller wants to wait for events newer than some previously seen event, this should be done by supplying a
// suitable filter function that checks the SequenceNumber of candidate events.
func (m *EventManager) WaitForJobStatusChangedEvent(ctx context.Context, filter EventFilter, jobs ...JobReference) (*client.Event, error) {
	Log(LogLevelTrace, fmt.Sprintf("WaitForJob() waiting on JobStatusChanged events for any one of %d jobs", len(jobs)))
	var matchingEvent *client.Event = nil

	// Subscribe to be notified when new events arrive
	subscriberID, newEventsCh := m.subscribeForNewEvents()
	defer m.unsubscribeFromNewEvents(subscriberID)

	// Loop until we find a suitable event matching one of our target jobs
	// TODO: Implement a timeout based on ctx
	for matchingEvent == nil {
		matchingEvent = m.checkEventsForJobs(filter, jobs...)
		if matchingEvent == nil {
			// No matching events; wait for more events to arrive, or the build to finish
			nrNewEvents, ok := <-newEventsCh
			if ok {
				Log(LogLevelDebug, fmt.Sprintf("Received %d new events, checking for matching jobs", nrNewEvents))
			} else {
				// Channel closed because build is finished
				Log(LogLevelDebug, "Build is finished while still waiting for events; returning error")
				return nil, fmt.Errorf("error: waiting for events on one of the following jobs %v but all jobs in the build are already finished", jobs)
			}
		}
	}

	return matchingEvent, nil
}

// checkEventsForJobs searches all events in m.events for a JobStatusChanged event associated with one of
// the specified jobs, for which the specified filter function returns true.
// Returns the first matching event (containing the matching job name) or nil if no suitable event was found.
// jobNames is the set of job names to look for (provided as a map of jobName to 'true').
func (m *EventManager) checkEventsForJobs(filter EventFilter, jobs ...JobReference) *client.Event {
	// Build a map of jobs for faster lookup
	jobNameSet := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		jobNameSet[job.String()] = true
	}

	m.eventsMutex.RLock()
	defer m.eventsMutex.RUnlock()

	for _, event := range m.events {
		if event.BuildId == m.buildID.String() && event.Type == EventTypeJobStatusChanged.String() {
			eventJobRef := GetJobRefOrNilFromEvent(&event)
			if eventJobRef != nil {
				// Check if the job reference in the event matches a job on the list
				_, jobFound := jobNameSet[eventJobRef.String()]
				if jobFound {
					// Check if the event passes the filter
					if filter == nil || filter(&event) {
						return &event
					}
				}
			}
		}
	}
	return nil // no suitable event found
}

// GetStatsForWorkflow returns statistics derived from events seen for the specified workflow.
// If no jobs or events have been seen for the workflow yet then a zero WorkflowStats object will be returned.
func (m *EventManager) GetStatsForWorkflow(workflowName ResourceName) WorkflowStats {
	m.eventsMutex.RLock()
	defer m.eventsMutex.RUnlock()

	stats := m.workflowJobStats[workflowName]
	if stats != nil {
		return *stats
	} else {
		return WorkflowStats{}
	}
}

// GetJobRefOrNilFromEvent finds and returns a job reference from the specified event, or nil if the event has
// no job reference.
func GetJobRefOrNilFromEvent(event *client.Event) *JobReference {
	// Extract workflow from event, if specified. No workflow field or empty string means the default workflow.
	workflow := ResourceName("")
	if event.Workflow != nil {
		workflow = ResourceName(*event.Workflow)
	}

	// Some events will have an explicit job name (e.g. status changed event)
	if event.JobName != nil {
		ref := NewJobReference(workflow, ResourceName(*event.JobName))
		return &ref
	}

	// Older servers may not populate the workflow and job fields, but will send job status changed events with
	// the job name in the ResourceName field.
	if event.Type == EventTypeJobStatusChanged.String() && event.ResourceName != "" {
		ref := NewJobReference(workflow, ResourceName(event.ResourceName))
		return &ref
	}

	return nil // no job reference
}

// registerJobs records the existence of the specified jobs that have been submitted to the server, and
// updates the statistics about unfinished jobs.
// Call this method when the SDK submits new jobs to ensure they show up as 'unfinished' jobs, preventing
// their workflow is being marked as finished before the first events relating to these jobs come in.
func (m *EventManager) registerJobs(jobs []client.JobGraph) {
	m.eventsMutex.Lock()
	for _, jGraph := range jobs {
		m.knownJobs = append(m.knownJobs, newJobIDReferenceForJobGraph(&jGraph))
	}
	m.eventsMutex.Unlock()

	m.updateStats()
}

func (m *EventManager) loop() {
	m.log(LogLevelTrace, fmt.Sprintf("Starting event manager loop (build %s)...", m.buildID))
	for {
		select {
		case <-m.StatefulService.Ctx().Done():
			m.log(LogLevelTrace, fmt.Sprintf("Event manager service closed; exiting polling loop (build %s)...", m.buildID))
			m.closeAllSubscriberChannels()
			return

		case <-time.After(m.eventPollInterval):
			err := m.pollForEvents()
			if err != nil {
				// Log and ignore any errors; just try again next time around the loop
				m.log(LogLevelError, fmt.Errorf("error checking for events (build %s): %w", m.buildID, err).Error())
			}
			// TODO: Uncomment this code once we have reliable 'build finished' detection again, with workflows
			//if m.buildFinished {
			//	// Close and remove all subscriptions, including new subscriptions that were opened since last poll
			//	m.closeAllSubscriberChannels()
			//}
		}
	}
}

func (m *EventManager) pollForEvents() error {
	// TODO: Consider providing a context with a timeout for each poll;
	// TODO: wait until we have retries for API to see whether this is still necessary
	eventsAPI := m.apiClient.EventsApi
	authContext := m.authContextFactory()

	Log(LogLevelTrace, "EventManager: polling for events...")
	newEvents, response, err := eventsAPI.GetEvents(authContext, m.buildID.String()).
		Last(m.lastEventSequenceNumber).
		Limit(defaultEventLimit).
		Execute()
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error expected status 200 (OK) from API call but got status %d", response.StatusCode)
	}
	Log(LogLevelTrace, fmt.Sprintf("EventManager: got %d new events", len(newEvents)))
	if len(newEvents) > 0 {
		m.processNewEvents(newEvents)
		m.notifySubscribersOfEvents(len(newEvents))
	}
	return nil
}

func (m *EventManager) processNewEvents(newEvents []client.Event) {
	m.eventsMutex.Lock()
	m.events = append(m.events, newEvents...)
	if len(m.events) > 0 {
		m.lastEventSequenceNumber = m.events[len(m.events)-1].SequenceNumber
	}
	m.eventsMutex.Unlock()

	if DefaultLogLevel <= LogLevelDebug {
		for _, event := range newEvents {
			Log(LogLevelDebug, fmt.Sprintf("EVENT #%d: %s event, workflow %v, job %v, resource %s (name '%s'), payload '%s'",
				event.SequenceNumber, event.Type, event.Workflow, event.JobName, event.ResourceId, event.ResourceName, event.Payload))
		}
	}

	m.updateStats()
}

// subscribeForNewEvents will create, register and return a subscriber channel to receive a notification
// each time there are new events available; the number of new events will be sent down the channel.
// To remove the subscription again please call unsubscribeFromNewEvents() with the returned subscriberID.
func (m *EventManager) subscribeForNewEvents() (subscriberID, chan int) {
	// Use a buffered channel to avoid holding up the polling Goroutine as it notifies subscribers
	ch := make(chan int, 10)

	m.subscriberMutex.Lock()
	defer m.subscriberMutex.Unlock()

	subscriberID := m.nextSubscriberID
	m.nextSubscriberID++
	m.subscriberChans[subscriberID] = ch
	return subscriberID, ch
}

func (m *EventManager) unsubscribeFromNewEvents(subscriberID subscriberID) {
	m.subscriberMutex.Lock()
	defer m.subscriberMutex.Unlock()

	delete(m.subscriberChans, subscriberID)
}

func (m *EventManager) notifySubscribersOfEvents(nrNewEvents int) {
	// Hold the subscriber mutex lock while we notify everyone, to guarantee that once unsubscribeFromNewEvents()
	// returns no more notifications will be delivered to a subscriber.
	m.subscriberMutex.Lock()
	defer m.subscriberMutex.Unlock()

	for _, ch := range m.subscriberChans {
		ch <- nrNewEvents
	}
}

func (m *EventManager) closeAllSubscriberChannels() {
	m.subscriberMutex.Lock()
	defer m.subscriberMutex.Unlock()

	for subscriberID, ch := range m.subscriberChans {
		close(ch)
		delete(m.subscriberChans, subscriberID)
	}
}

// updateStats updates m.workflowJobStats with the number of finished, unfinished and failed jobs in each
// workflow that has jobs, based on the known jobs submitted and the complete set of events delivered so far.
// The caller MUST NOT already hold a lock on the eventsMutex.
func (m *EventManager) updateStats() {
	m.eventsMutex.Lock()

	// Create a nested map of workflow name to a Job ID to the latest job status, based on the event stream
	jobStatusMap := make(map[ResourceName]map[ResourceID]Status)

	// Start by recording the existence of all known jobs
	for _, jobIDRef := range m.knownJobs {
		// Ensure there is a map for the workflow, mapping Job ID to the latest status
		_, workflowFound := jobStatusMap[jobIDRef.Workflow]
		if !workflowFound {
			jobStatusMap[jobIDRef.Workflow] = make(map[ResourceID]Status)
		}

		// Existing entries in the map can be updated but only as long as they aren't marked as finished.
		// After a job has finished subsequent status changed events will be ignored.
		_, jobFound := jobStatusMap[jobIDRef.Workflow][jobIDRef.jobID]
		if !jobFound {
			// Record that the job has been queued; this status will be updated/overwritten by status changed events
			jobStatusMap[jobIDRef.Workflow][jobIDRef.jobID] = StatusQueued
		}
	}

	// Process all known events to update job status
	for _, event := range m.events {
		if event.Type == EventTypeJobStatusChanged.String() {
			jobID := ResourceID(event.ResourceId)
			jobStatus := Status(event.Payload)
			jobRef := GetJobRefOrNilFromEvent(&event)
			if jobRef == nil {
				Log(LogLevelWarn, "Job status changed event does not contain a Job Reference; ignoring event")
				continue // this shouldn't really happen;
			}

			// Ensure there is a map for the workflow, mapping Job ID to the latest status
			_, workflowFound := jobStatusMap[jobRef.Workflow]
			if !workflowFound {
				jobStatusMap[jobRef.Workflow] = make(map[ResourceID]Status)
			}

			// Existing entries in the map can be updated but only as long as they aren't marked as finished.
			// After a job has finished subsequent status changed events will be ignored.
			existingStatus, jobFound := jobStatusMap[jobRef.Workflow][jobID]
			if !jobFound || !existingStatus.HasFinished() {
				jobStatusMap[jobRef.Workflow][jobID] = jobStatus
			}
		}
	}

	// Count the jobs in each state for each workflow
	stats := make(WorkflowStatsMap)
	for workflowName, jobMap := range jobStatusMap {
		stats[workflowName] = &WorkflowStats{}
		for _, status := range jobMap {
			if status.HasFailed() {
				stats[workflowName].FailedJobCount++
			}
			if status.HasFinished() {
				stats[workflowName].FinishedJobCount++
			} else {
				stats[workflowName].UnfinishedJobCount++
			}
		}
		Log(LogLevelDebug, fmt.Sprintf("eventManager: workflow '%s' has %d finished and %d unfinished jobs",
			workflowName, stats[workflowName].FinishedJobCount, stats[workflowName].UnfinishedJobCount))
	}

	// Replace the stored stats with the new stats
	m.workflowJobStats = stats

	m.eventsMutex.Unlock()

	// Tell the workflow manager (and hence the workflows) that we have new stats
	globalWorkflowManager.statsUpdated()
}
