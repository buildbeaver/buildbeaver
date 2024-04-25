package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

type OrchestratorFactory func() *Orchestrator

func MakeOrchestratorFactory(
	client APIClient,
	executorFactory ExecutorFactory,
	logFactory logger.LogFactory) OrchestratorFactory {
	return func() *Orchestrator {
		return NewOrchestrator(client, executorFactory, logFactory)
	}
}

// Orchestrator orchestrates the execution of a job by progressing it through
// a series of lifecycle phases.
type Orchestrator struct {
	logFactory      logger.LogFactory
	client          APIClient
	executorFactory ExecutorFactory
	// attemptedStepsByName is the list of steps within the job that the orchestrator has attempted to run
	attemptedStepsByName   map[models.ResourceName]*documents.Step
	attemptedStepsByNameMu sync.RWMutex // protects attemptedStepsByName
	executor               *Executor
	logger.Log
}

func NewOrchestrator(client APIClient, executorFactory ExecutorFactory, logFactory logger.LogFactory) *Orchestrator {
	return &Orchestrator{
		logFactory:           logFactory,
		client:               client,
		executorFactory:      executorFactory,
		attemptedStepsByName: make(map[models.ResourceName]*documents.Step),
		Log:                  logFactory("Orchestrator"),
	}
}

// Run all steps associated with job, respecting concurrency controls and
// the step dependency graph. The step's statuses will be reported to the
// server as the steps are executed (on both success and failure).
func (s *Orchestrator) Run(runnable *documents.RunnableJob) {

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	jobCtx := NewJobBuildContext(ctx, runnable)
	s.executor = s.executorFactory(ctx)

	jobDoc, err := s.client.UpdateJobStatus(
		ctx, // OK to use the job context for the initial status update since it won't have timed out yet
		runnable.Job.ID,
		models.WorkflowStatusRunning,
		nil,
		runnable.Job.ETag)
	if err != nil {
		s.Errorf("Error updating job status to running: %s", err)
		return
	}
	runnable.Job = jobDoc

	var (
		jobErr      error
		jobPrepared bool
	)

	for _, job := range runnable.Jobs {
		if job.Error.Valid() {
			jobErr = fmt.Errorf("Job dependency failed: %s", job.Name)
			break
		}
	}
	if jobErr == nil {
		jobErr = s.prepareJob(jobCtx)
		jobPrepared = true // we must tear down job if we called prepareJob(), even if it partly failed
	}

	// NOTE: We want to visit all steps (even if a dependency fails) to ensure that we
	// send an appropriate status back to the server. We intentionally do not bubble
	// errors up to the walk (by always returning nil) as this would cause it to abort.
	err = s.walkSteps(runnable.Job, runnable.Steps, true, func(step *documents.Step) error {
		// TODO reserve token and defer release

		// Use a new context for the step status update, so we can send an update even if the main context times out
		stepStatusContext, stepStatusCancel := getStatusUpdateContext()
		defer stepStatusCancel()
		stepDoc, err := s.client.UpdateStepStatus(
			stepStatusContext,
			step.ID,
			models.WorkflowStatusRunning,
			nil,
			step.ETag)
		if err != nil {
			s.Errorf("Error updating step status to running: %s", err)
			return nil
		}
		s.recordAttemptedStep(stepDoc)

		if jobErr == nil {
			stepCtx := NewStepBuildContext(jobCtx, stepDoc)
			err = s.executeStep(stepCtx)
			stepDoc = stepCtx.Step() // step may have been modified during execution
		} else {
			err = jobErr
		}
		if err != nil {
			// Record the error against the step;
			// error will have already been recorded in the step's log if there was a log for the step
			stepDoc.Error = models.NewError(err)
		}

		status := models.WorkflowStatusSucceeded
		if stepDoc.Error != nil {
			status = models.WorkflowStatusFailed
		}

		// Use another new context for the step status update, so we can send an update even if the main context
		// times out. The step log pipeline must have been flushed and closed before updating the status,
		// since completing the step will seal the log for the step.
		stepStatusContext2, stepStatusCancel2 := getStatusUpdateContext()
		defer stepStatusCancel2()
		stepDoc, err = s.client.UpdateStepStatus(
			stepStatusContext2,
			stepDoc.ID,
			status,
			stepDoc.Error,
			stepDoc.ETag)
		if err != nil {
			s.Errorf("Error updating step status to finished: %s", err)
			return nil
		}
		s.recordCompletedStep(stepDoc) // do this after step status has been updated

		return nil
	})
	if err != nil {
		panic(err)
	}

	if jobErr == nil {
		s.attemptedStepsByNameMu.RLock()
		for _, step := range s.attemptedStepsByName {
			if step.Error.Valid() {
				jobErr = fmt.Errorf("Step failed: %s", step.Name)
				break
			} else if step.Status != models.WorkflowStatusSucceeded {
				jobErr = fmt.Errorf("Step did not succeed (status is '%s'): %s", step.Status, step.Name)
				break
			}
		}
		s.attemptedStepsByNameMu.RUnlock()
	}

	if jobPrepared {
		// Write any job error to the job log pipeline before calling tearDownJob(), which closes the pipeline
		if jobErr != nil {
			s.executor.LogJobError(jobCtx, jobErr)
		}
		err := s.tearDownJob(jobCtx)
		// If we encounter an error we can continue unless it's an artifact upload error where we need to fail the build
		if err != nil {
			if gerror.IsArtifactUploadFailed(err) && jobErr == nil {
				jobErr = err
			} else {
				s.Errorf("Will ignore error tearing down job: %s", err)
			}
		}
		s.executor.Close()
	}

	// Calculate and set the job's final status; do this after tearing down the job.
	// The job log pipeline must have been flushed and closed before updating the status, since completing the job
	// will seal the log for the job and the server will reject any further log writes.
	if jobErr != nil {
		// Record the error against the job
		runnable.Job.Error = models.NewError(jobErr)
	}
	status := models.WorkflowStatusSucceeded
	if runnable.Job.Error != nil {
		status = models.WorkflowStatusFailed
	}
	// Use a new context for the job status update, so we can send an update even if the main job context timed out.
	jobStatusContext2, jobStatusCancel2 := getStatusUpdateContext()
	defer jobStatusCancel2()
	jobDoc, err = s.client.UpdateJobStatus(
		jobStatusContext2,
		runnable.Job.ID,
		status,
		runnable.Job.Error,
		runnable.Job.ETag)
	if err != nil {
		s.Errorf("Error updating job status to finished: %s", err)
		return
	}
	runnable.Job = jobDoc
}

// stepDAGNode wraps a Step document, allowing it to be used as a node in a DAG by implementing
// the dto.GraphNode interface.
type stepDAGNode struct {
	job  *documents.Job
	step *documents.Step
}

func (s *stepDAGNode) GetFQN() models.NodeFQN {
	return models.NewNodeFQN(s.job.Workflow, s.job.Name, s.step.Name)
}

func (s *stepDAGNode) GetFQNDependencies() []models.NodeFQN {
	var depends []models.NodeFQN
	for _, dependency := range s.step.Depends {
		depends = append(depends, models.NewNodeFQN(s.job.Workflow, s.job.Name, dependency.StepName))
	}
	return depends
}

// WalkSteps walks the step dependency graph for the specified set of steps, visiting each step once, after that
// step's dependencies have been visited. If parallel is true, the walk will be performed
// in parallel, and errors (if any) will be accumulated and returned at the end.
// If parallel is false, the walk will be performed in series, and the first
// error (if any) will immediately cause the walk to fail and that error will be returned.
func (s *Orchestrator) walkSteps(
	job *documents.Job,
	steps []*documents.Step,
	parallel bool,
	callback func(*documents.Step) error,
) error {
	// Build a DAG from the steps in the supplied job
	nodes := make([]dto.GraphNode, len(steps))
	for i, step := range steps {
		nodes[i] = &stepDAGNode{job, step}
	}
	dag, err := dto.NewDAG(nodes)
	if err != nil {
		return errors.Wrap(err, "error building step dependency graph")
	}

	return dag.Walk(parallel, func(node interface{}) error {
		return callback(node.(*stepDAGNode).step)
	})
}

// prepareJob is called once per job, before the first step in the job is executed.
func (s *Orchestrator) prepareJob(ctx *JobBuildContext) error {
	err := templateJob(ctx.Job())
	if err != nil {
		return fmt.Errorf("error templating job: %w", err)
	}
	return s.executor.PreExecuteJob(ctx)
}

// executeStep executes the step defined in the build context.
// Calls PreExecuteStep, ExecuteStep and PostExecuteStep.
// By the time this function returns, the log pipeline for the step will have been flushed and closed.
func (s *Orchestrator) executeStep(ctx *StepBuildContext) (err error) {
	// All other steps that are dependencies should have successfully finished
	for _, dependency := range ctx.Step().Depends {
		dependency := s.getAttemptedStep(dependency.StepName)
		if dependency == nil {
			return fmt.Errorf("error locating result for step dependency: %s", dependency.Name)
		}
		if dependency.Error.Valid() {
			return fmt.Errorf("Step dependency failed: %s", dependency.Name)
		}
		// If we are walking the graph successfully then steps that are dependencies should be finished
		if dependency.Status != models.WorkflowStatusSucceeded {
			// At this stage there is no step log pipeline, so no need to log this error to the pipeline
			return fmt.Errorf("Step dependency did not succeed (status is '%s'): %s", dependency.Status, dependency.Name)
		}
	}

	// Ensure all returned errors are written to the step pipeline, and handle cleanup by always calling
	// PostExecuteStep(); if step setup or execution fails we still want to perform a best-efforts cleanup.
	// We expect the cleanup will probably return an error in this case as e.g. files or folders may
	// be missing, Docker containers were never started etc. In this specific case just log and ignore
	// the error, and allow the initial error from PreExecute or Execute to be returned.
	defer func() {
		// Write any error being returned to the step log pipeline before calling PostExecuteStep()
		if err != nil {
			s.executor.LogStepError(ctx, err)
		}
		// Always call PostExecuteStep; this closes the step log pipeline
		cleanupErr := s.executor.PostExecuteStep(ctx)
		if cleanupErr != nil {
			if err == nil {
				// Previously no error was being returned; now return the PostExecuteStep() error
				err = errors.Wrap(cleanupErr, "error in post execute")
			} else {
				// Just log and ignore the PostExecuteStep() error; keep the originally returned error
				s.Warnf("Will ignore error tearing down failed step: %s", cleanupErr)
			}
		}
	}()

	err = s.executor.PreExecuteStep(ctx)
	if err != nil {
		return errors.Wrap(err, "error in pre execute")
	}

	err = s.executor.ExecuteStep(ctx)
	if err != nil {
		return errors.Wrap(err, "error executing step")
	}

	return nil
}

// tearDownJob is called once per job, after the last step in the job is executed.
func (s *Orchestrator) tearDownJob(ctx *JobBuildContext) error {
	err := s.executor.PostExecuteJob(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Orchestrator) recordAttemptedStep(step *documents.Step) {
	s.attemptedStepsByNameMu.Lock()
	defer s.attemptedStepsByNameMu.Unlock()
	s.attemptedStepsByName[step.Name] = step
}

func (s *Orchestrator) recordCompletedStep(step *documents.Step) {
	s.attemptedStepsByNameMu.Lock()
	defer s.attemptedStepsByNameMu.Unlock()
	// Update attemptedStepsByName with the document containing new status, in case this is a different instance
	s.attemptedStepsByName[step.Name] = step
	s.Infof("Step %s completed: %s", step.ID, step.Error)
}

func (s *Orchestrator) getAttemptedStep(stepName models.ResourceName) *documents.Step {
	s.attemptedStepsByNameMu.RLock()
	defer s.attemptedStepsByNameMu.RUnlock()
	return s.attemptedStepsByName[stepName]
}
