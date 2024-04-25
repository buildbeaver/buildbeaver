package bb

import (
	"fmt"
	"os"
	"sync"
)

// WorkflowStatsMap maps workflow name to statistics for the workflow
type WorkflowStatsMap map[ResourceName]*WorkflowStats

// WorkflowStats contains statistics about a workflow
type WorkflowStats struct {
	FinishedJobCount   int
	UnfinishedJobCount int
	FailedJobCount     int
}

// Workflows defines a set of workflows for the current build, and begins processing workflows to submit jobs
// to the build.
// Returns when the workflow handler functions for all required workflows have completed.
func Workflows(workflows ...*WorkflowDefinition) {
	// Add all workflows to global registry
	for _, workflow := range workflows {
		err := globalWorkflowManager.register(workflow)
		if err != nil {
			Log(LogLevelFatal, err.Error())
			os.Exit(1)
		}
	}

	// Create build from OS environment variables
	build := MustGetBuild()

	err := globalWorkflowManager.runWorkflows(build)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
}

// WorkflowsWithEnv defines a set of workflows for the current build for use in testing, and begins
// processing workflows to submit jobs to the build.
// The build environment variable data is sourced from the specified string map rather than from actual
// environment variables, to facilitate passing test values.
// If createNewWorkflowManager is true then the global workflow manager is replaced with a new workflow manager,
// effectively removing all existing registered workflows in order to start a new test.
// Returns when the workflow handler functions for all required workflows have completed.
// If an error occurs while registering or starting workflows then the error is returned rather than the process
// being terminated.
func WorkflowsWithEnv(
	envVars map[string]string,
	createNewWorkflowManager bool,
	workflows ...*WorkflowDefinition,
) (*Build, error) {
	if createNewWorkflowManager {
		globalWorkflowManager = newWorkflowManager()
	}

	// Add all workflows to global registry
	for _, workflow := range workflows {
		err := globalWorkflowManager.register(workflow)
		if err != nil {
			return nil, err
		}
	}

	// Create build from the supplied variables
	build, err := getBuildWithEnv(envVars)
	if err != nil {
		return nil, err
	}

	err = globalWorkflowManager.runWorkflows(build)
	if err != nil {
		return nil, err
	}

	return build, nil
}

// AddWorkflows registers extra workflows to the set of workflows for the current build. This can be called from
// an init function prior to calling Workflows() from the main function.
func AddWorkflows(workflows ...*WorkflowDefinition) {
	for _, workflow := range workflows {
		err := globalWorkflowManager.register(workflow)
		if err != nil {
			Log(LogLevelFatal, err.Error())
			os.Exit(1)
		}
	}
}

// workflowManager is a singleton that is responsible for registering and creating workflows for the build.
type workflowManager struct {
	build *Build

	// workflowsMutex covers workflows, workflowsStarted
	workflowsMutex sync.RWMutex
	// definitions is a map of registered workflow definitions, by workflow name
	definitions map[ResourceName]*WorkflowDefinition
	// workflows is a map of workflow objects to dynamically track each workflow
	workflows map[ResourceName]*Workflow
	// workflowsStarted is true if workflows have been started already
	workflowsStarted bool

	// wg is a WaitGroup that can be used to wait until all required workflow handlers have finished running
	wg sync.WaitGroup
}

var globalWorkflowManager = newWorkflowManager()

func newWorkflowManager() *workflowManager {
	return &workflowManager{
		definitions: make(map[ResourceName]*WorkflowDefinition),
		workflows:   make(map[ResourceName]*Workflow),
	}
}

func (m *workflowManager) register(workflow *WorkflowDefinition) error {
	m.workflowsMutex.Lock()
	defer m.workflowsMutex.Unlock()

	if m.workflowsStarted {
		return fmt.Errorf("error registering workflow '%s': workflows can only be registered prior to starting (i.e. inside Workflows() or init())",
			workflow.GetName())
	}

	err := workflow.validate()
	if err != nil {
		return err
	}
	if _, exists := m.definitions[workflow.GetName()]; exists {
		return fmt.Errorf("error: workflow with name '%s' already exists", workflow.GetName())
	}

	m.definitions[workflow.GetName()] = workflow
	return nil
}

func (m *workflowManager) getWorkflowOrNil(workflowName ResourceName) *Workflow {
	m.workflowsMutex.RLock()
	defer m.workflowsMutex.RUnlock()

	return m.workflows[workflowName]
}

func (m *workflowManager) mustGetWorkflow(workflowName ResourceName) *Workflow {
	workflow := m.getWorkflowOrNil(workflowName)
	if workflow == nil {
		Log(LogLevelFatal, fmt.Sprintf("mustGetWorkflow() could not find workflow '%s'", workflowName))
		os.Exit(1)
	}
	return workflow
}

// runWorkflows calls workflow functions for a subset of the currently registered set of workflows,
// as defined in the build. Returns an error if no workflows can be started, otherwise waits until
// all workflows have finished running before returning.
func (m *workflowManager) runWorkflows(build *Build) error {
	err := func() error {
		m.workflowsMutex.Lock()
		defer m.workflowsMutex.Unlock()

		if m.workflowsStarted {
			Log(LogLevelFatal, "runWorkflows() called but workflows are already started")
			os.Exit(1)
		}
		m.workflowsStarted = true
		m.build = build

		// Create a workflow object for each definition
		for _, definition := range m.definitions {
			workflow := newWorkflowFromDefinition(definition, m.build)
			m.workflows[workflow.GetName()] = workflow
		}

		// Start the workflows requested in the build
		workflowsToRun := m.build.WorkflowsToRun
		startedCount := 0
		if len(workflowsToRun) > 0 {
			// Initially run only the workflows explicitly specified, and their explicitly defined dependency workflows
			if len(workflowsToRun) > 1 {
				Log(LogLevelInfo, fmt.Sprintf("Starting %d requested workflows...", len(workflowsToRun)))
			} else if len(workflowsToRun) == 1 {
				Log(LogLevelInfo, fmt.Sprintf("Starting requested workflow '%s'", workflowsToRun[0]))
			}
			for _, workflowName := range workflowsToRun {
				workflow := m.workflows[workflowName]
				if workflow != nil {
					count, err := m.startWorkflowAndDependencies(workflow)
					if err != nil {
						return err
					}
					startedCount += count
				} else {
					return fmt.Errorf("error: workflow '%s' not found", workflowName)
				}
			}
		} else {
			// No set of workflows explicitly specified, so run all workflows
			Log(LogLevelDebug, fmt.Sprintf("Starting all registered workflows..."))
			for _, workflow := range m.workflows {
				count, err := m.startWorkflowAndDependencies(workflow)
				if err != nil {
					return err
				}
				startedCount += count
			}
		}
		if startedCount == 0 {
			return fmt.Errorf("error: no workflows were started")
		}
		Log(LogLevelDebug, fmt.Sprintf("Started %d workflow(s).", startedCount))
		return nil // success
	}()
	if err != nil {
		return err
	}

	// TODO: Consider handling newly registered workflows. Do we want to support this? If not, enforce this
	// TODO: in the AddWorkflows() so it can only be called before the workflows are started.

	m.wg.Wait() // wait for all workflow functions to finish
	return nil
}

// startWorkflowAndDependencies starts the given workflow, as well as any other workflows that are mentioned
// in the dependency list of the workflow, recursively. Returns the number of new workflows started.
// The caller must have already obtained the workflowMutex lock.
func (m *workflowManager) startWorkflowAndDependencies(workflow *Workflow) (startedCount int, err error) {
	if workflow == nil || workflow.isStarted {
		return 0, nil
	}

	// Start dependency workflows
	for _, dependency := range workflow.definition.dependencies {
		dependencyName := dependency.dependsOnWorkflow
		dependencyWorkflow := m.workflows[dependencyName]
		if dependencyWorkflow == nil {
			return startedCount, fmt.Errorf("error: workflow '%s' not found but is required by workflow '%s'",
				dependencyName, workflow.GetName())
		}
		Log(LogLevelInfo, fmt.Sprintf("Checking dependency workflow '%s' is started", dependencyName))
		depCount, err := m.startWorkflowAndDependencies(dependencyWorkflow)
		if err != nil {
			return startedCount, err
		}
		startedCount += depCount
	}

	// Start the specified workflow
	workflow.start(&m.wg) // start() is idempotent so no race condition
	startedCount++

	return startedCount, nil
}

// ensureWorkflowStarted is called when a new workflow dependency is discovered, from a newly created job.
// The specified workflow (and any dependent workflows) will be started if registered. This ensures the workflow
// has been started so the dependency can be satisfied.
func (m *workflowManager) ensureWorkflowStarted(workflowName ResourceName) error {
	m.workflowsMutex.Lock()
	defer m.workflowsMutex.Unlock()

	workflow := m.workflows[workflowName]
	if workflow == nil {
		return fmt.Errorf("error: dependency on workflow which is not registered: '%s'", workflowName)
	}

	if !workflow.isStarted {
		Log(LogLevelInfo, fmt.Sprintf("Starting depencency workflow '%s'...", workflow.GetName()))
		_, err := m.startWorkflowAndDependencies(workflow)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *workflowManager) statsUpdated() {
	// Get a list of workflows to notify, while holding the workflowsMutex lock for a minimal amount of time
	m.workflowsMutex.RLock()
	workflowsToNotify := make([]*Workflow, 0, len(m.workflows))
	for _, workflow := range m.workflows {
		workflowsToNotify = append(workflowsToNotify, workflow)
	}
	m.workflowsMutex.RUnlock()

	// Tell all the workflows in the list that new stats are available
	for _, workflow := range workflowsToNotify {
		workflow.statsUpdated()
	}
}
