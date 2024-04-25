package bb

import (
	"fmt"
	"os"
)

type WorkflowDependsOption string

const (
	// WorkflowWait is an option to not start the dependent workflow until the other workflow
	// has completely finished (including all jobs). This is the default behaviour.
	WorkflowWait = WorkflowDependsOption("wait")
	// WorkflowConcurrent is an option to allow the dependent workflow to run concurrently with the
	// other workflow (as opposed to WaitForOtherWorkflowToFinish)
	WorkflowConcurrent = WorkflowDependsOption("concurrent")
	// WorkflowTerminateOnFailure will terminate the current process if the dependency workflow fails, instead
	// of starting the dependent workflow. This is the default behaviour.
	WorkflowTerminateOnFailure = WorkflowDependsOption("terminate-on-failure")
	// WorkflowStartOnFailure will start the dependent workflow even if the dependency workflow fails, instead
	// of terminating the process (as opposed to TerminateIfOtherWorkflowFails)
	WorkflowStartOnFailure = WorkflowDependsOption("start-on-failure")
)

type workflowDependency struct {
	dependsOnWorkflow ResourceName
	// wait is true to wait for dependency workflow to finish, false to run concurrently
	wait bool
	// terminateOnFailure is true to terminate the process if the dependency workflow fails, false to run
	// the dependent workflow anyway
	terminateOnFailure bool
}

func newWorkflowDependency(dependsOnWorkflow ResourceName, options []WorkflowDependsOption) *workflowDependency {
	dep := &workflowDependency{
		dependsOnWorkflow:  dependsOnWorkflow,
		wait:               true, // default
		terminateOnFailure: true, // default
	}

	// Process the supplied options; if they conflict then last one wins
	for _, option := range options {
		switch option {
		case WorkflowWait:
			dep.wait = true
		case WorkflowConcurrent:
			dep.wait = false
		case WorkflowTerminateOnFailure:
			dep.terminateOnFailure = true
		case WorkflowStartOnFailure:
			dep.terminateOnFailure = false
		default:
			Log(LogLevelFatal, fmt.Sprintf("Unknown workflow dependency option specified, terminating: '%s", option))
			os.Exit(1)
		}
	}

	return dep
}
