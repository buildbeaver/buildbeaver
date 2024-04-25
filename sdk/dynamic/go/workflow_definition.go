package bb

import "fmt"

type WorkflowDefinition struct {
	// Name is the workflow name, in URL format
	name ResourceName `json:"name"`
	// Handler is a function that can submit jobs for the workflow
	handler WorkflowHandler
	// True if we should terminate the entire process if we can't submit a job
	submitFailureIsFatal bool
	// dependencies is a list of workflow dependencies for this workflow. The meaning of each dependency is
	// determined by the options specified in the dependency.
	dependencies []*workflowDependency
}

func NewWorkflow() *WorkflowDefinition {
	return &WorkflowDefinition{
		name:                 "",
		handler:              nil,
		submitFailureIsFatal: true, // default to true
	}
}

func (w *WorkflowDefinition) Name(name ResourceName) *WorkflowDefinition {
	w.name = name
	return w
}

func (w *WorkflowDefinition) GetName() ResourceName {
	return ResourceName(w.name)
}

func (w *WorkflowDefinition) Handler(handler WorkflowHandler) *WorkflowDefinition {
	w.handler = handler
	return w
}

func (w *WorkflowDefinition) SubmitFailureIsFatal(isFatal bool) *WorkflowDefinition {
	w.submitFailureIsFatal = isFatal
	return w
}

// Depends indicates that the specified workflow depends on another workflow. The specified options determine the
// exact behaviour; the default is to wait until the specified workflow is fully finished before running this workflow,
// and to terminate this process if the specified workflow fails.
func (w *WorkflowDefinition) Depends(workflowName ResourceName, options ...WorkflowDependsOption) *WorkflowDefinition {
	w.dependencies = append(w.dependencies, newWorkflowDependency(workflowName, options))
	return w
}

func (w *WorkflowDefinition) validate() error {
	if w.GetName() == "" {
		return fmt.Errorf("error validating workflow: name must not be an empty string")
	}
	if w.handler == nil {
		return fmt.Errorf("error validating workflow definition '%s': a handler function must be specified", w.GetName())
	}
	return nil
}
