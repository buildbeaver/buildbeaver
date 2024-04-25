package bb

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
	"golang.org/x/net/context"
)

// WorkflowHandler is a function that can create the jobs required for a particular workflow.
type WorkflowHandler func(workflow *Workflow) error

type Workflow struct {
	definition *WorkflowDefinition
	build      *Build

	// internal state
	startMutex         sync.Mutex // covers isStarted and the startup process
	isStarted          bool
	jobCallbackManager *JobCallbackManager

	isHandlerFinished  bool // no need to lock; this variable is monotonic and boolean
	isWorkflowFinished bool // no need to lock; this variable is monotonic and boolean
	hasWorkflowFailed  bool // no need to lock; this variable is monotonic and boolean

	jobMutex     sync.Mutex            // covers newJobs, newJobErrors
	newJobs      map[ResourceName]*Job // maps job name to job
	newJobErrors []string

	outputsMutex sync.RWMutex // covers outputs
	outputs      map[string]interface{}
}

func newWorkflowFromDefinition(definition *WorkflowDefinition, build *Build) *Workflow {
	return &Workflow{
		definition: definition,
		build:      build,
		outputs:    make(map[string]interface{}),
	}
}

func (w *Workflow) GetDefinition() WorkflowDefinition {
	return *w.definition
}

func (w *Workflow) GetName() ResourceName {
	return w.definition.GetName()
}

// start will start the workflow, and run its handler function (in a separate goroutine) to begin submitting jobs.
// The handler-running goroutine will be added to the specified WaitGroup, and will call wg.Done when finished.
// This method is idempotent, and is a no-op if the workflow is already started.
func (w *Workflow) start(wg *sync.WaitGroup) {
	w.startMutex.Lock()
	defer w.startMutex.Unlock()

	err := w.definition.validate()
	if err != nil {
		Log(LogLevelFatal, fmt.Sprintf("Attempt to start an invalid workflow: '%s'", err.Error()))
		os.Exit(1)
	}
	if w.build == nil {
		Log(LogLevelFatal, fmt.Sprintf("Attempt to start a workflow with no build (workflow name '%s')", w.GetName()))
		os.Exit(1)
	}
	if w.isStarted {
		return // no-op
	}

	Log(LogLevelInfo, fmt.Sprintf("Starting workflow '%s'", w.GetName()))

	// Set up internal state required to run a workflow
	w.jobCallbackManager = NewJobCallbackManager(w.build.eventManager)
	w.newJobs = make(map[ResourceName]*Job)
	w.newJobErrors = []string{}

	w.isStarted = true

	// Start a separate Goroutine to actually run the workflow. Be sure to add this goroutine to the WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for any dependency workflows to finish
		w.waitForDependencyWorkflows()

		// Run the workflow handler
		err := w.definition.handler(w)
		if err != nil {
			// TODO: Don't necessarily quit the process here; this is a bad thing to do in tests
			Log(LogLevelFatal, fmt.Sprintf("Error submitting new jobs to build for workflow '%s': %s", w.GetName(), err.Error()))
			os.Exit(1)
		}

		// Submit any jobs that haven't already been submitted, and wait for callbacks to be run.
		// This also updates the stats with the new jobs, via the event manager.
		_, err = w.Submit(true)
		if err != nil {
			msg := fmt.Sprintf("Error submitting new job to build at end of workflow: %s", err.Error())
			if w.definition.submitFailureIsFatal {
				Log(LogLevelFatal, msg)
				os.Exit(1)
			} else {
				Log(LogLevelError, msg)
			}
		}
		w.isHandlerFinished = true // no need for lock
		w.updateWorkflowStatus()
	}()
}

func (w *Workflow) statsUpdated() {
	Log(LogLevelDebug, fmt.Sprintf("Stats updated notification received for workflow '%s'", w.GetName()))

	// Only update workflow status if it has been started; otherwise w.build is still nil
	if w.isStarted {
		w.updateWorkflowStatus()
	}
}

func (w *Workflow) updateWorkflowStatus() {
	// Read the latest stats from the event manager
	stats := w.build.eventManager.GetStatsForWorkflow(w.GetName())
	hasUnfinishedJobs := stats.UnfinishedJobCount > 0
	hasFailedJobs := stats.FailedJobCount > 0

	Log(LogLevelDebug, fmt.Sprintf("Checking finished/failed status for workflow '%s': %d unfinished and %d finished jobs",
		w.GetName(), stats.UnfinishedJobCount, stats.FinishedJobCount))

	if hasFailedJobs {
		Log(LogLevelInfo, fmt.Sprintf("Workflow '%s' set to failed", w.GetName()))
		w.hasWorkflowFailed = true // No need for lock. Never set this back to false.
	}

	if w.isHandlerFinished && !hasUnfinishedJobs {
		Log(LogLevelDebug, fmt.Sprintf("Workflow '%s' set to finished", w.GetName()))
		w.isWorkflowFinished = true // No need for lock. Never set this back to false.
	}
}

// IsFinished returns true if the workflow function has returned and there are no unfinished jobs remaining
// in the workflow, i.e. if the workflow is completely finished.
func (w *Workflow) IsFinished() bool {
	return w.isWorkflowFinished
}

// IsFailed returns true if any job or jobs in the workflow have failed, even if the workflow has not yet finished.
func (w *Workflow) IsFailed() bool {
	return w.hasWorkflowFailed
}

func (w *Workflow) GetBuild() *Build {
	return w.build
}

// MustSubmit submits all newly created jobs to the server by calling Submit(), and returns the details for
// the newly created jobs. If an error is returned then the error is logged and this program exits with error code 1.
// If waitForCallbacks is true, or not specified, MustSubmit then waits until all outstanding callbacks have been called.
// If waitForCallbacks is specified as false, or if after submitting new jobs there are no outstanding callbacks,
// then MustSubmit returns immediately.
func (w *Workflow) MustSubmit(waitForCallbacks ...bool) []client.JobGraph {
	jobGraph, err := w.Submit(waitForCallbacks...)
	if err != nil {
		Log(LogLevelFatal, fmt.Sprintf("Error submitting new jobs to build: %s", err.Error()))
		os.Exit(1)
	}
	return jobGraph
}

// Submit submits all newly created jobs to the server and returns the details for the newly created jobs.
// If waitForCallbacks is true, or not specified, Submit then waits until all outstanding callbacks have been called.
// If waitForCallbacks is specified as false, or if after submitting new jobs there are no outstanding callbacks,
// then Submit returns immediately.
func (w *Workflow) Submit(waitForCallbacks ...bool) ([]client.JobGraph, error) {
	if len(waitForCallbacks) > 1 {
		return nil, fmt.Errorf("Build.Submit() requires 0 or 1 arguments, but %d arguments were supplied", len(waitForCallbacks))
	}
	shouldWait := true // default is to wait
	if len(waitForCallbacks) > 0 {
		shouldWait = waitForCallbacks[0]
	}

	jGraph, err := w.sendNewJobsToServer()
	if err != nil {
		return nil, err
	}

	if shouldWait {
		w.jobCallbackManager.BlockAndProcessCallbacks()
	}

	return jGraph, nil
}

// sendNewJobsToServer sends all newly created jobs to the server and returns the details for the newly created jobs.
func (w *Workflow) sendNewJobsToServer() ([]client.JobGraph, error) {
	// Hold a lock on the job mutex for the entire time we are attempting to submit new jobs
	w.jobMutex.Lock()
	defer w.jobMutex.Unlock()

	if len(w.newJobs) == 0 && len(w.newJobErrors) == 0 {
		Log(LogLevelTrace, fmt.Sprintf("No new jobs or errors to submit to server for workflow %s in call to Submit()", w.GetName()))
		return nil, nil
	}

	if len(w.newJobErrors) > 0 {
		// Log all errors and return a suitable error
		for _, errorStr := range w.newJobErrors {
			Log(LogLevelError, errorStr)
		}
		return nil, fmt.Errorf("error: %d error(s) found during Job creation", len(w.newJobErrors))
	}

	// Ensure that all job dependencies specify a workflow, and that any dependent workflows are started.
	// Do this while holding the jobMutex.
	for _, job := range w.newJobs {
		err := validateJobDependencies(job.definition.Depends)
		if err != nil {
			return nil, fmt.Errorf("error: job '%s' has an indvalid job dependency: %w", job.GetReference(), err)
		}
		workflowsForJob := job.getWorkflowDependencies()
		for _, workflow := range workflowsForJob {
			err = globalWorkflowManager.ensureWorkflowStarted(workflow)
			if err != nil {
				return nil, fmt.Errorf("error validating new jobs to be submitted: %w", err)
			}
		}
	}

	jobsAPI := w.build.GetAPIClient().JobsApi

	// Send all new jobs to the server
	jobDefinitions := make([]client.JobDefinition, 0, len(w.newJobs))
	for _, job := range w.newJobs {
		jobDefinitions = append(jobDefinitions, job.definition)
	}
	buildDefinition := client.NewBuildDefinition(BuildDefinitionSyntaxVersion, jobDefinitions)

	Log(LogLevelInfo, fmt.Sprintf("Sending %d new jobs to server for workflow %s", len(w.newJobs), w.GetName()))
	jGraphs, response, err := jobsAPI.CreateJobs(w.build.GetAuthorizedContext(), w.build.ID.String()).
		BuildDefinition(*buildDefinition).
		Execute()
	statusCode := int(0)
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("Error sending jobs to server (response status code %d, %d jobs returned): %s - %s\n", statusCode, len(jGraphs), openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("Error sending jobs to server (response status code %d, %d jobs returned): %w\n", statusCode, len(jGraphs), err)
	}
	Log(LogLevelInfo, fmt.Sprintf("Sent %d new jobs, received back status code %d with %d jobs", len(buildDefinition.GetJobs()), statusCode, len(jGraphs)))
	err = w.processCreateJobResults(jGraphs)
	if err != nil {
		return nil, err
	}

	// clear new job data
	w.newJobs = make(map[ResourceName]*Job)
	w.newJobErrors = []string{}

	return jGraphs, nil
}

// processCreateJobResults processes the results from calling the CreateJobs() API function.
// The caller should be already holding a lock on jobMutex when this function is called so Job IDs can be stored.
func (w *Workflow) processCreateJobResults(results []client.JobGraph) error {
	Log(LogLevelDebug, fmt.Sprintf("API call returned jobs array: %v", results))

	// Register the existence of the new jobs with the event manager
	w.build.eventManager.registerJobs(results)

	return nil
}

func (w *Workflow) Job(job *Job) *Workflow {
	w.jobMutex.Lock()
	defer w.jobMutex.Unlock()

	// Ensure the job has a unique name within the workflow
	jobName := job.GetName()
	if _, found := w.newJobs[jobName]; found {
		w.newJobErrors = append(w.newJobErrors, fmt.Sprintf("ERROR: Job with name '%s' already exists in workflow '%s'; job will not be submitted", w.GetName(), jobName))
		return w
	}

	job.workflow = w
	workflowNameStr := w.GetName().String()
	job.definition.Workflow = &workflowNameStr

	w.newJobs[jobName] = job

	// Register all the callbacks that have already been declared, then clear the lists.
	// Subsequent callbacks will be registered by the job directly with the build, now that it's part of a build
	for _, callback := range job.completionCallbacksToRegister {
		w.OnJobCompletion(job.GetReference(), callback)
	}
	job.completionCallbacksToRegister = nil

	for _, callback := range job.successCallbacksToRegister {
		w.OnJobSuccess(job.GetReference(), callback)
	}
	job.successCallbacksToRegister = nil

	for _, callback := range job.failureCallbacksToRegister {
		w.OnJobFailure(job.GetReference(), callback)
	}
	job.failureCallbacksToRegister = nil

	for _, callback := range job.cancelledCallbacksToRegister {
		w.OnJobCancelled(job.GetReference(), callback)
	}
	job.cancelledCallbacksToRegister = nil

	for _, callback := range job.statusChangedCallbacksToRegister {
		// Job can't have seen any status changed events yet
		w.OnJobStatusChanged(job.GetReference(), callback)
	}
	job.statusChangedCallbacksToRegister = nil

	Log(LogLevelInfo, fmt.Sprintf("Job with name '%s' added to build", jobName))
	return w
}

func (w *Workflow) OnJobCompletion(jobRef JobReference, callback JobCallback) {
	w.jobCallbackManager.AddSubscription(newJobSubscription(
		jobRef,
		true, // only match events that signal a job is completed
		nil,  // match any (completed) status
		w.wrapCallback(callback),
	))
}

func (w *Workflow) OnJobSuccess(jobRef JobReference, callback JobCallback) {
	status := StatusSucceeded
	w.jobCallbackManager.AddSubscription(newJobSubscription(
		jobRef,
		true,
		&status,
		w.wrapCallback(callback),
	))
}

func (w *Workflow) OnJobFailure(jobRef JobReference, callback JobCallback) {
	status := StatusFailed
	w.jobCallbackManager.AddSubscription(newJobSubscription(
		jobRef,
		true,
		&status,
		w.wrapCallback(callback),
	))
}

func (w *Workflow) OnJobCancelled(jobRef JobReference, callback JobCallback) {
	status := StatusCanceled
	w.jobCallbackManager.AddSubscription(newJobSubscription(
		jobRef,
		true,
		&status,
		w.wrapCallback(callback),
	))
}

// OnJobStatusChanged will call a callback function each time the status of a job changes.
func (w *Workflow) OnJobStatusChanged(jobRef JobReference, callback JobCallback) {
	w.jobCallbackManager.AddSubscription(newJobSubscription(
		jobRef,
		false, // call back even when job is not completed yet
		nil,   // call back for any status change
		w.wrapCallback(callback),
	))
}

// wrapCallback returns a new JobCallback function that calls the supplied function and then calls sendJobsToServer(),
// to ensure any new jobs created by the callback are submitted to the server.
// If the new jobs can't be submitted then the process wil lbe terminated with exit code 1.
func (w *Workflow) wrapCallback(callback JobCallback) JobCallback {
	return func(event *JobStatusChangedEvent) {
		callback(event)
		// don't wait for callbacks to be called here since we are already in a callback; just submit the jobs
		_, err := w.Submit(false)
		if err != nil {
			msg := fmt.Sprintf("Error submitting new job to build: %s", err.Error())
			if w.definition.submitFailureIsFatal {
				Log(LogLevelFatal, msg)
				os.Exit(1)
			} else {
				Log(LogLevelError, msg)
			}
		}
	}
}

// SetOutput sets an output value for the workflow, that can be used by other workflows.
// name is a name for the output, unique within this workflow. Any existing value with this name will be
// overwritten. value is the data for the output.
// Any object supplied as data should ideally be immutable, and at the very least be thread-safe so
// that it can be read by other workflows from other goroutines.
func (w *Workflow) SetOutput(outputName string, value interface{}) {
	w.outputsMutex.Lock()
	defer w.outputsMutex.Unlock()

	w.outputs[outputName] = value
}

// GetOutputOrNil gets a previously set output value for the workflow, by name,
// If no output value from this workflow exists with the specified name then nil is immediately returned.
func (w *Workflow) GetOutputOrNil(outputName string) interface{} {
	w.outputsMutex.Lock()
	defer w.outputsMutex.Unlock()

	return w.outputs[outputName]
}

// getOutputFromWorkflowOrNil gets the output value with the specified name from the specified workflow.
// If no such output value exists, or if no such workflow exists, then nil is immediately returned.
func (w *Workflow) getOutputFromWorkflowOrNil(workflowName ResourceName, outputName string) interface{} {
	workflow := globalWorkflowManager.getWorkflowOrNil(workflowName)
	if workflow == nil {
		return nil
	}

	return workflow.GetOutputOrNil(outputName)
}

// WaitForOutput waits until the workflow with the specified name has an output with the specified output name
// available, then returns the output value.
// Returns an error if the workflow has finished without providing the output.
func (w *Workflow) WaitForOutput(workflowName ResourceName, outputName string) (interface{}, error) {
	err := globalWorkflowManager.ensureWorkflowStarted(workflowName)
	if err != nil {
		// Log and ignore error starting the workflow
		Log(LogLevelWarn, fmt.Sprintf("unable to ensure workflow '%s' is started: %s", workflowName, err.Error()))
	}
	workflow := globalWorkflowManager.mustGetWorkflow(workflowName)

	for {
		// Record whether the workflow has finished, before checkout for output
		workflowFinished := workflow.IsFinished()

		output := w.getOutputFromWorkflowOrNil(workflowName, outputName)
		if output != nil {
			return output, nil
		}

		if workflowFinished {
			// Workflow has finished and output still isn't there - it's not coming
			return nil, fmt.Errorf("error waiting for output '%s' from workflow '%s': workflow has finished without setting the output",
				outputName, workflowName)
		}
		// TODO: Subscribe and be notified whenever a new value is available so we don't need to sleep
		time.Sleep(1 * time.Second)
	}
}

// MustWaitForOutput waits until the workflow with the specified name has an output with the specified output name
// available, then returns the output value.
// Terminates this program if the workflow has finished without providing the output.
func (w *Workflow) MustWaitForOutput(workflowName ResourceName, outputName string) interface{} {
	result, err := w.WaitForOutput(workflowName, outputName)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return result
}

// WaitForJob waits until any of the specified jobs is finished, then returns the event that notified that the
// job is finished. This event includes the job's name, ID and final status.
// Jobs are specified by name, including the workflow, in the format 'workflow.jobname'.
// Any outstanding newly created jobs will be submitted to the server before waiting, via a call to MustSubmit().
func (w *Workflow) WaitForJob(jobs ...string) (*JobStatusChangedEvent, error) {
	w.MustSubmit()
	jobRefs := stringsToJobReferences(jobs)

	// Ensure all workflows that will be waited on are started
	for _, job := range jobRefs {
		if job.Workflow != "" {
			err := globalWorkflowManager.ensureWorkflowStarted(job.Workflow)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("error: no workflow specified in call to WaitForJob() for job '%s'", job.JobName)
		}
	}

	// a filter function that returns true for events where a job has finished
	jobCompletedFilter := func(event *client.Event) bool {
		return Status(event.Payload).HasFinished()
	}

	rawEvent, err := w.build.eventManager.WaitForJobStatusChangedEvent(context.Background(), jobCompletedFilter, jobRefs...)
	if err != nil {
		return nil, err
	}
	event, err := NewJobStatusChangedEvent(rawEvent)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// MustWaitForJob waits until any of the specified jobs is finished, then returns the event that notified that the
// job is finished. This event includes the job's name, ID and final status.
// Jobs are specified by name, including the workflow, in the format 'workflow.jobname'.
// Any outstanding newly created jobs will be submitted to the server before waiting, via a call to MustSubmit().
// Terminates this program if the build has finished without any event arriving that indicates one of the Jobs
// has finished.
func (w *Workflow) MustWaitForJob(jobs ...string) *JobStatusChangedEvent {
	result, err := w.WaitForJob(jobs...)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return result
}

// WaitForWorkflow waits until the workflow with the specified name has completely finished.
// Returns the workflow that has now finished; this can be used to check for failure by calling IsFailed()
func (w *Workflow) WaitForWorkflow(workflowName ResourceName) *Workflow {
	err := globalWorkflowManager.ensureWorkflowStarted(workflowName)
	if err != nil {
		// Log and ignore error starting the workflow
		Log(LogLevelWarn, fmt.Sprintf("WaitForWorkflow() is unable to ensure workflow '%s' is started: %s", workflowName, err.Error()))
	}
	workflow := globalWorkflowManager.mustGetWorkflow(workflowName)
	for {
		if workflow.IsFinished() {
			return workflow
		}
		// TODO: Subscribe and be notified whenever a new value is available so we don't need to sleep
		time.Sleep(1 * time.Second)
	}
}

// IsWorkflowFinished returns true iff the specified workflow has completely finished.
func (w *Workflow) IsWorkflowFinished(workflowName ResourceName) bool {
	workflow := globalWorkflowManager.getWorkflowOrNil(workflowName)
	if workflow == nil {
		return false
	}
	return workflow.IsFinished()
}

func (w *Workflow) waitForDependencyWorkflows() {
	for _, dep := range w.definition.dependencies {
		if dep.wait {
			Log(LogLevelDebug, fmt.Sprintf("Workflow '%s' is waiting for dependency workflow '%s' to finish", w.GetName(), dep.dependsOnWorkflow))
			finishedWorkflow := w.WaitForWorkflow(dep.dependsOnWorkflow)
			if dep.terminateOnFailure && finishedWorkflow.IsFailed() {
				Log(LogLevelFatal, fmt.Sprintf("Workflow '%s' failed, but is required for workflow '%s' to run; terminating", dep.dependsOnWorkflow, w.GetName()))
				os.Exit(1)
			}
			Log(LogLevelDebug, fmt.Sprintf("Workflow '%s' has finished waiting for dependency workflow '%s'", w.GetName(), dep.dependsOnWorkflow))
		}
	}
}
