package dto

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto/dag"
)

type CreateJob struct {
	*models.Job
	Build *models.Build
}

// Validate the create, and the underlying job.
func (m *CreateJob) Validate() error {
	if m.BuildID != m.Build.ID {
		return fmt.Errorf("error mismatched build ids")
	}
	return m.Job.Validate()
}

type UpdateJobStatus struct {
	Status models.WorkflowStatus
	Error  *models.Error
	ETag   models.ETag
}

type UpdateJobFingerprint struct {
	Fingerprint         string
	FingerprintHashType models.HashType
	ETag                models.ETag
}

type UpdateJobFingerprintAndIndirect struct {
	UpdateJobFingerprint
	IndirectToJobID models.JobID
}

// JobGraph provides the details of a job, including the steps that the job contains.
type JobGraph struct {
	*models.Job
	// Steps is the set of steps within the job.
	Steps []*models.Step `json:"steps"`
}

// Validate the job including the step relationships/dependencies.
func (m *JobGraph) Validate() error {
	var result *multierror.Error
	if len(m.Steps) == 0 {
		result = multierror.Append(result, errors.New("Job must have at least one step"))
	}
	for i, step := range m.Steps {
		err := step.Validate()
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("error validating step %q (index %d): %w", step.Name, i, err))
		}
	}

	// Form the graph of steps into a DAG, which will verify there are no cycles
	_, err := m.dag()
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = m.Job.Validate()
	if err != nil {
		result = multierror.Append(result, err)
	}
	return result.ErrorOrNil()
}

// PopulateDefaults sets default values for all fields of all structs
// in the job that haven't been populated.
func (m *JobGraph) PopulateDefaults(build *BuildGraph) {
	m.Job.PopulateDefaults(build.Build)
	for _, step := range m.Steps {
		step.PopulateDefaults(build.Build, m.Job)
	}
}

// Walk the step dependency graph visiting each step once, after that step's
// dependencies have been visited. If parallel is true, the walk will be performed
// in parallel, and errors (if any) will be accumulated and returned at the end.
// If parallel is false, the walk will be performed in series, and the first
// error (if any) will immediately cause the walk to fail and that error will be returned.
func (m *JobGraph) Walk(parallel bool, callback func(*models.Step) error) error {
	dag, err := m.dag()
	if err != nil {
		return err
	}
	return dag.Walk(parallel, func(node interface{}) error {
		return callback(node.(*models.Step))
	})
}

// dag builds a dag containing the job's steps and validates it.
func (m *JobGraph) dag() (*DAG, error) {
	nodes := make([]GraphNode, len(m.Steps))
	for i, step := range m.Steps {
		nodes[i] = step
	}
	dag, err := NewDAG(nodes)
	if err != nil {
		return nil, errors.Wrap(err, "error building step dependency graph")
	}
	return dag, nil
}

func (m *JobGraph) Trim(keep []models.ResourceName) error {
	if len(keep) == 0 {
		return fmt.Errorf("error one or more steps must be kept")
	}
	currentDAG, err := m.dag()
	if err != nil {
		return fmt.Errorf("error making step dag: %w", err)
	}
	stepsToKeep := make(map[models.ResourceName]struct{})
	for _, stepName := range keep {
		stepsToKeep[stepName] = struct{}{}
	}
	keeping := new(dag.Set)
	for _, step := range m.Steps {
		if _, ok := stepsToKeep[step.Name]; !ok {
			continue
		}
		set, err := currentDAG.graph.Descendents(step)
		if err != nil {
			return errors.Wrap(err, "error computing descendents")
		}
		keeping.Add(step)
		toAdd := set.Difference(keeping)
		for _, v := range toAdd.List() {
			keeping.Add(v)
		}
	}
	m.Steps = nil
	for _, v := range keeping.List() {
		if v == RootNode {
			continue
		}
		m.Steps = append(m.Steps, v.(*models.Step))
	}
	return nil
}
