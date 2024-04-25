package dto

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto/dag"
)

type UpdateBuildStatus struct {
	Status models.WorkflowStatus
	Error  *models.Error
	ETag   models.ETag
}

type BuildGraph struct {
	*models.Build
	// Jobs that make up the build.
	Jobs []*JobGraph `json:"jobs"`
}

// Validate the entire build pipeline including the job and step relationships/dependencies.
func (m *BuildGraph) Validate() error {
	var result *multierror.Error

	err := m.Build.Validate()
	if err != nil {
		result = multierror.Append(result, fmt.Errorf("Error validating build: %w", err))
	}

	if len(m.Jobs) == 0 {
		result = multierror.Append(result, errors.New("Build must have at least one job"))
	}

	jobsByFQN := make(map[models.NodeFQN]*models.Job, len(m.Jobs))
	stepsByFQN := make(map[models.NodeFQN]*models.Step)

	for i, job := range m.Jobs {

		err := job.Validate()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "Error validating job %q at index %d", job.Name, i))
		}

		// Check for duplicate jobs, and record the job by FQN
		_, ok := jobsByFQN[job.GetFQN()]
		if ok {
			result = multierror.Append(result, errors.Errorf("Found duplicate job %q in workflow %q; Jobs must have a unique name", job.Name, job.Workflow))
		} else {
			jobsByFQN[job.GetFQN()] = job.Job
		}

		for i, step := range job.Steps {

			err := step.Validate()
			if err != nil {
				result = multierror.Append(result, errors.Wrapf(err, "Error validating job %q step at index %d", job.Name, i))
			}

			// Check for duplicate steps, and record the job by FQN
			stepFQN := models.NewNodeFQN(job.Workflow, job.Name, step.Name)
			_, ok := stepsByFQN[stepFQN]
			if ok {
				result = multierror.Append(result, errors.Errorf("Found duplicate step %q in job %q, workflow %q; Steps must have a unique name", step.Name, job.Name, job.Workflow))
			} else {
				stepsByFQN[stepFQN] = step
			}
		}
	}

	// Validate job dependencies
	for _, job := range m.Jobs {
		for _, dependency := range job.Depends {
			dependencyFQN := models.NewNodeFQNForJob(dependency.Workflow, dependency.JobName)
			_, ok := jobsByFQN[dependency.GetFQN()]
			// Job dependencies can refer to jobs in other workflows that don't yet exist, so it's only an error
			// when the dependency job doesn't exist if it should be in the same workflow as the dependent job
			if !ok && (dependency.Workflow == job.Workflow) {
				result = multierror.Append(result, errors.Errorf("Job %q depends on job %q but it does not exist",
					job.Name, dependencyFQN))
			}
			// TODO validate artifact dependencies
		}
	}

	// Validate step dependencies
	for stepFQN, step := range stepsByFQN {
		for _, dependency := range step.Depends {
			// Step dependencies are within the same workflow and job
			_, ok := stepsByFQN[models.NewNodeFQN(stepFQN.WorkflowName, stepFQN.JobName, dependency.StepName)]
			if !ok {
				result = multierror.Append(result, errors.Errorf("Step %q depends on step %q in job %q, workflow %q, but it does not exist",
					stepFQN, dependency, stepFQN.JobName, stepFQN.WorkflowName))
			}
		}
	}

	// Form the graph into a DAG, which will verify there are no cycles
	_, err = m.dag()
	if err != nil {
		result = multierror.Append(result, err)
	}

	// Validate nodes to run
	for _, nodeFQN := range m.Opts.NodesToRun {
		if nodeFQN.JobName != "" {
			// Make a separate FQN referring to the job, without a step
			jobFQN := models.NewNodeFQNForJob(nodeFQN.WorkflowName, nodeFQN.JobName)
			_, ok := jobsByFQN[jobFQN]
			if !ok {
				result = multierror.Append(result, errors.Errorf("Build options specified job %q but it does not exist", jobFQN))
			}
		}
		if nodeFQN.StepName != "" {
			_, ok := stepsByFQN[nodeFQN]
			if !ok {
				result = multierror.Append(result, errors.Errorf("Build options specified step %q but it does not exist",
					nodeFQN))
			}
		}
	}

	// Return a validation error so an HTTP status of 400 (bad request) is returned
	err = result.ErrorOrNil()
	if err != nil {
		return gerror.NewErrValidationFailed(err.Error()).Wrap(err)
	}

	return nil
}

// PopulateDefaults sets default values for all fields of all structs in the build pipeline that haven't been populated.
func (m *BuildGraph) PopulateDefaults() {
	if !m.ID.Valid() {
		m.ID = models.NewBuildID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = models.NewTime(time.Now())
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}
	if m.Status == "" || m.Status == models.WorkflowStatusUnknown {
		m.Status = models.WorkflowStatusQueued
	}
	for _, job := range m.Jobs {
		job.PopulateDefaults(m)
	}
}

// Walk the job dependency graph visiting each job once, after that job's
// dependencies have been visited. The Walk will be aborted if callback returns
// an error, and that error will be returned. If parallel is true, the Walk will
// be performed in parallel, and errors (if any) will be accumulated and returned
// at the end. If parallel is false, the Walk will be performed in series, and the first
// error (if any) will immediately cause the Walk to fail and that error will be returned.
func (m *BuildGraph) Walk(parallel bool, callback func(*JobGraph) error) error {
	dag, err := m.dag()
	if err != nil {
		return err
	}
	return dag.Walk(parallel, func(node interface{}) error {
		return callback(node.(*JobGraph))
	})
}

// dag builds a dag containing the pipeline's jobs and validates it.
func (m *BuildGraph) dag() (*DAG, error) {
	nodes := make([]GraphNode, len(m.Jobs))
	for i, job := range m.Jobs {
		nodes[i] = job
	}
	dag, err := NewDAG(nodes)
	if err != nil {
		return nil, errors.Wrap(err, "error building job dependency graph")
	}
	return dag, nil
}

// Ancestors returns all ancestors (dependencies) of the specified job. Includes transitive dependencies.
// Does not include dependencies on jobs in other workflows that don't exist yet.
func (m *BuildGraph) Ancestors(jGraph *JobGraph) ([]*JobGraph, error) {
	dag, err := m.dag()
	if err != nil {
		return nil, err
	}
	ancestors, err := dag.Ancestors(jGraph)
	if err != nil {
		return nil, err
	}
	out := make([]*JobGraph, len(ancestors))
	for i := 0; i < len(ancestors); i++ {
		out[i] = ancestors[i].(*JobGraph)
	}
	return out, nil
}

// Trim removes all nodes from the graph that are not either in the 'keep' set, or dependencies of a node
// in the 'keep' set. This function assumes that all jobs and steps in the graph have already been submitted,
// so they can be found in the graph; otherwise jobs may be trimmed that are transitive dependencies of jobs
// that don't yet exist.
func (m *BuildGraph) Trim(keep []models.NodeFQN) error {
	if len(keep) == 0 {
		return fmt.Errorf("error one or more steps must be kept")
	}

	currentDAG, err := m.dag()
	if err != nil {
		return fmt.Errorf("error making job dag: %w", err)
	}

	// Make a map of step names by job FQN
	stepsToKeepByJobFQN := make(map[models.NodeFQN][]models.ResourceName)
	for _, keepFQN := range keep {
		jobFQN := models.NewNodeFQNForJob(keepFQN.WorkflowName, keepFQN.JobName)
		stepsToKeepByJobFQN[jobFQN] = append(stepsToKeepByJobFQN[jobFQN], keepFQN.StepName)
	}

	// Make a set of jobs to keep
	keeping := new(dag.Set)
	for _, job := range m.Jobs {
		if _, ok := stepsToKeepByJobFQN[job.GetFQN()]; !ok {
			continue
		}

		set, err := currentDAG.graph.Descendents(job)
		if err != nil {
			return errors.Wrap(err, "error computing descendents")
		}

		keeping.Add(job)

		toAdd := set.Difference(keeping)
		for _, v := range toAdd.List() {
			keeping.Add(v)
		}
	}

	// Clear the set of jobs in this build graph and re-add only the jobs we want to keep
	m.Jobs = nil
	for _, v := range keeping.List() {
		if v == RootNode { // TODO get this shiz outa here
			continue
		}
		jobGraph := v.(*JobGraph)

		stepsToKeep, ok := stepsToKeepByJobFQN[jobGraph.GetFQN()]
		if ok && len(stepsToKeep) > 0 {
			// If any entry in 'keep' contained a node that just references the jobGraph, e.g `generate`
			// then we want to keep all steps for the jobGraph (even if `generate` and `generate.foo` were
			// specified - the all takes precedence)
			var skipTrim bool
			for _, step := range stepsToKeep {
				if step == "" {
					skipTrim = true
					break
				}
			}
			if !skipTrim {
				err := jobGraph.Trim(stepsToKeep)
				if err != nil {
					return errors.Wrap(err, "error trimming Job Graph")
				}
			}
		}
		m.Jobs = append(m.Jobs, jobGraph) // keep this job
	}

	return nil
}
