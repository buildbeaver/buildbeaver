package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const (
	DefaultMaxBuildConfigLength int = 2 * 1024 * 1024 // 2 megabytes
	DefaultMaxJobsPerBuild      int = 256
	DefaultMaxStepsPerJob       int = 20
)

type LimitsConfig struct {
	// MaxBuildConfigLength is the maximum length a build configuration is allowed to be, in bytes.
	// This applies to static build config files obtained from version control, and also to sets of
	// jobs submitted during dynamic builds.
	MaxBuildConfigLength int
	// MaxJobsPerBuild is the maximum number of jobs allowed in a single build. Any build definition (static or
	// dynamic) that would cause the total number of jobs in the build to exceed this limit will be rejected.
	MaxJobsPerBuild int
	// MaxJobsPerBuild is the maximum number of steps allowed in any single job. Any build definition containing
	// a job with more than this number of steps will be rejected.
	MaxStepsPerJob int
}

type QueueService struct {
	db                *store.DB
	runnerService     services.RunnerService
	buildService      services.BuildService
	jobService        services.JobService
	stepService       services.StepService
	repoService       services.RepoService
	credentialService services.CredentialService
	logService        services.LogService
	eventService      services.EventService
	commitStore       store.CommitStore
	timeoutChecker    *TimeoutChecker
	scmRegistry       *scm.SCMRegistry
	limits            LimitsConfig
	logger.Log
}

func NewQueueService(
	db *store.DB,
	buildService services.BuildService,
	runnerService services.RunnerService,
	jobService services.JobService,
	stepService services.StepService,
	repoService services.RepoService,
	credentialService services.CredentialService,
	logService services.LogService,
	eventService services.EventService,
	commitStore store.CommitStore,
	scmRegistry *scm.SCMRegistry,
	logFactory logger.LogFactory,
	limits LimitsConfig,
) *QueueService {

	s := &QueueService{
		db:                db,
		buildService:      buildService,
		runnerService:     runnerService,
		jobService:        jobService,
		stepService:       stepService,
		repoService:       repoService,
		credentialService: credentialService,
		logService:        logService,
		eventService:      eventService,
		commitStore:       commitStore,
		scmRegistry:       scmRegistry,
		limits:            limits,
		Log:               logFactory("QueueService"),
	}

	s.timeoutChecker = NewTimeoutChecker(db, s, jobService, stepService, logFactory)
	s.timeoutChecker.Start()
	return s
}

func (s *QueueService) Stop() {
	s.timeoutChecker.Stop()
}

// EnqueueBuildFromCommit parses the build definition from the specified commit, and enqueues a new build from it.
// If there is a problem with the build definition then a skeleton build is enqueued that is immediately
// set to failed with an error describing the problem, and no error will be returned from this function.
// Returns an error only if there was a transient issue that could be retried.
func (s *QueueService) EnqueueBuildFromCommit(
	ctx context.Context,
	txOrNil *store.Tx,
	commit *models.Commit,
	ref string,
	opts *models.BuildOptions,
) (*dto.BuildGraph, error) {
	parser := parser.NewBuildDefinitionParser(s.getParserLimits())
	buildDef, err := parser.Parse(commit.Config, commit.ConfigType)
	if err != nil {
		return s.createFailedBuild(ctx, txOrNil, commit, ref, opts, err)
	}

	graph, err := s.makeNewBuildGraph(commit.RepoID, commit.ID, buildDef, ref, opts)
	if err != nil {
		err = fmt.Errorf("error parsing build configuration: %w", err)
		return s.createFailedBuild(ctx, txOrNil, commit, ref, opts, err)
	}

	return s.enqueueBuild(ctx, txOrNil, graph)
}

// EnqueueBuildFromBuildDefinition enqueues a new build based on the specified build definition, which is assumed
// to have come from the specified commit. Unlike EnqueueBuildFromCommit this function will return an error
// if there is a problem with the build definition (as well as any transient errors).
func (s *QueueService) EnqueueBuildFromBuildDefinition(
	ctx context.Context,
	txOrNil *store.Tx,
	repoID models.RepoID,
	commitID models.CommitID,
	buildDef *models.BuildDefinition,
	ref string,
	opts *models.BuildOptions,
) (*dto.BuildGraph, error) {
	graph, err := s.makeNewBuildGraph(repoID, commitID, buildDef, ref, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating build graph: %w", err)
	}

	return s.enqueueBuild(ctx, txOrNil, graph)
}

// AddConfigToBuild enqueues new jobs for an existing build, taken from the supplied build configuration.
// Returns the full build graph containing both existing and new jobs, as well as an array containing just the new jobs.
// This function will return an error if there is a problem with the jobs, as well as any transient errors.
func (s *QueueService) AddConfigToBuild(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	config []byte,
	configType models.ConfigType,
) (*dto.BuildGraph, []*dto.JobGraph, error) {
	// Check maximum length for build config
	err := s.CheckBuildConfigLength(len(config))
	if err != nil {
		return nil, nil, gerror.NewErrValidationFailed(fmt.Sprintf("Error dynamically creating jobs: %s", err.Error()))
	}

	// Parse the jobs into job definitions
	parser := parser.NewBuildDefinitionParser(s.getParserLimits())
	buildDef, err := parser.Parse(config, configType)
	if err != nil {
		return nil, nil, gerror.NewErrValidationFailed(err.Error())
	}

	return s.addJobsToBuild(ctx, txOrNil, buildID, buildDef.Jobs)
}

// addJobsToBuild enqueues new jobs for an existing build.
// Returns the full build graph containing both existing and new jobs, as well as an array containing just the new jobs.
// This function will return an error if there is a problem with the jobs, as well as any transient errors.
func (s *QueueService) addJobsToBuild(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID, jobs []models.JobDefinition) (*dto.BuildGraph, []*dto.JobGraph, error) {
	var (
		bGraph     *dto.BuildGraph
		newJGraphs []*dto.JobGraph
		err        error
	)
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Lock the build, as we want to ensure:
		//  1) No other concurrent calls to AddJobsToBuild for this build are being made
		//  2) The build status cannot change underneath us
		err := s.buildService.LockRowForUpdate(ctx, tx, buildID)
		if err != nil {
			return fmt.Errorf("error locking build: %w", err)
		}
		// Read the existing build graph
		bGraph, err = s.ReadBuildGraph(ctx, tx, buildID)
		if err != nil {
			return fmt.Errorf("error reading build graph: %w", err)
		}
		if bGraph.Build.Status.HasFinished() {
			return gerror.NewErrValidationFailed(fmt.Sprintf("error build has already finished with status '%s'",
				bGraph.Build.Status))
		}
		// Append the new jobs to the existing graph
		err = s.makeJobGraphsAndAppendToBuildGraph(bGraph, jobs)
		if err != nil {
			return fmt.Errorf("error making new job graphs: %w", err)
		}
		bGraph.PopulateDefaults()
		// Validate the full graph containing all existing and new jobs; this will pick up any new cycles in the
		// graph resulting from previously-deferred dependencies on jobs that have now been added
		err = bGraph.Validate()
		if err != nil {
			return fmt.Errorf("error validating updated build graph: %w", err)
		}
		// Enqueue the new jobs
		newJGraphs, err = s.enqueueJobs(ctx, tx, bGraph)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return bGraph, newJGraphs, nil
}

func (s *QueueService) getParserLimits() parser.ParserLimits {
	return parser.ParserLimits{
		MaxStepsPerJob: s.limits.MaxStepsPerJob,
	}
}

// CheckBuildConfigLength returns an error if the supplied length (in bytes) is too long for a build configuration,
// or if the configuration is empty.
func (s *QueueService) CheckBuildConfigLength(length int) error {
	if length == 0 {
		return gerror.NewErrValidationFailed("build configuration is empty")
	}

	if length > s.limits.MaxBuildConfigLength {
		return gerror.NewErrValidationFailed(fmt.Sprintf(
			"build configuration is too long (length is %d bytes, maximum allowed is %d)",
			length, s.limits.MaxBuildConfigLength))
	}
	return nil
}

// Dequeue returns the next queued job that is ready for execution and that the specified
// runner is capable of running, or a ErrCodeNotFound if no jobs are ready for execution.
func (s *QueueService) Dequeue(ctx context.Context, runnerID models.RunnerID) (*dto.RunnableJob, error) {
	var dequeued *dto.RunnableJob
	err := s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		runner, err := s.runnerService.Read(ctx, tx, runnerID)
		if err != nil {
			return fmt.Errorf("error reading runner: %w", err)
		}
		// Don't return any jobs if we are not enabled
		if !runner.Enabled {
			return gerror.NewErrCodeRunnerDisabled()
		}
		stg, err := s.jobService.FindQueuedJob(ctx, tx, runner)
		if err != nil {
			return err
		}
		job := &dto.RunnableJob{
			JobGraph: &dto.JobGraph{
				Job: stg,
			},
		}
		build, err := s.buildService.Read(ctx, tx, job.BuildID)
		if err != nil {
			return fmt.Errorf("error reading build: %w", err)
		}
		repo, err := s.repoService.Read(ctx, tx, job.RepoID)
		if err != nil {
			return fmt.Errorf("error reading repo: %w", err)
		}
		commit, err := s.commitStore.Read(ctx, tx, build.CommitID)
		if err != nil {
			return fmt.Errorf("error reading commit: %w", err)
		}
		steps, err := s.stepService.ListByJobID(ctx, tx, job.ID)
		if err != nil {
			return fmt.Errorf("error listing job steps: %w", err)
		}
		// TODO: This is not transitive... it probably should be
		dependencyJobs, err := s.jobService.ListDependencies(ctx, tx, job.ID)
		if err != nil {
			return fmt.Errorf("error reading job dependencies: %w", err)
		}
		job.Jobs = dependencyJobs
		job.Steps = steps
		job.Repo = repo
		job.Commit = commit

		// Create an identity and a JWT token for use by dynamic build steps during the build
		identity, err := s.buildService.FindOrCreateIdentity(ctx, tx, build.ID)
		if err != nil {
			return err
		}
		jwtToken, err := s.credentialService.CreateIdentityJWT(identity.ID)
		job.JWT = jwtToken

		job.WorkflowsToRun = s.getInitialWorkflowsToRun(build)

		jobStatusChanged := job.Status != models.WorkflowStatusSubmitted
		job.Status = models.WorkflowStatusSubmitted
		job.RunnerID = runner.ID
		_, err = s.updateJob(ctx, tx, job.JobGraph.Job, jobStatusChanged)
		if err != nil {
			return fmt.Errorf("error updating job: %w", err)
		}
		for _, step := range steps {
			stepStatusChanged := step.Status != models.WorkflowStatusSubmitted
			step.Status = models.WorkflowStatusSubmitted
			step.RunnerID = runner.ID
			_, err = s.updateStep(ctx, tx, job.Job, step, stepStatusChanged)
			if err != nil {
				return fmt.Errorf("error updating step: %w", err)
			}
		}
		_, err = s.maintainBuildStatus(ctx, tx, job.BuildID)
		if err != nil {
			return fmt.Errorf("error maintaining build status: %w", err)
		}
		dequeued = job
		s.Infof("Dequeued job %s", dequeued.ID)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error dequeuing job: %w", err)
	}

	return dequeued, nil
}

// getInitialWorkflowsToRun returns the set of workflows that are explicitly requested in the build options
// for the specified build.
func (s *QueueService) getInitialWorkflowsToRun(build *models.Build) []models.ResourceName {
	var workflows []models.ResourceName
	for _, fqn := range build.Opts.NodesToRun {
		// Include workflows that are mentioned individually, but also those that are part of a job or step FQN
		if fqn.WorkflowName != "" {
			workflows = append(workflows, fqn.WorkflowName)
		}
	}
	return workflows
}

// ReadQueuedBuild makes a queued build DTO including all child jobs and steps.
func (s *QueueService) ReadQueuedBuild(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) (*dto.QueuedBuild, error) {
	var queuedBuild *dto.QueuedBuild
	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		bGraph, err := s.ReadBuildGraph(ctx, tx, buildID)
		if err != nil {
			return fmt.Errorf("error build graph: %w", err)
		}
		repo, err := s.repoService.Read(ctx, tx, bGraph.Build.RepoID)
		if err != nil {
			return fmt.Errorf("error reading repo: %w", err)
		}
		commit, err := s.commitStore.Read(ctx, tx, bGraph.Build.CommitID)
		if err != nil {
			return fmt.Errorf("error reading commit: %w", err)
		}
		queuedBuild = &dto.QueuedBuild{Repo: repo, Commit: commit, BuildGraph: bGraph}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return queuedBuild, nil
}

// ReadBuildGraph returns the build graph for the specified build.
func (s *QueueService) ReadBuildGraph(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) (*dto.BuildGraph, error) {
	var bGraph *dto.BuildGraph
	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		build, err := s.buildService.Read(ctx, tx, buildID)
		if err != nil {
			return fmt.Errorf("error reading repo: %w", err)
		}
		jobs, err := s.jobService.ListByBuildID(ctx, tx, build.ID)
		if err != nil {
			return fmt.Errorf("error reading build jobs: %w", err)
		}
		var jGraphs []*dto.JobGraph
		for _, job := range jobs {
			steps, err := s.stepService.ListByJobID(ctx, tx, job.ID)
			if err != nil {
				return fmt.Errorf("error listing job steps: %w", err)
			}
			jGraphs = append(jGraphs, &dto.JobGraph{Job: job, Steps: steps})
		}
		bGraph = &dto.BuildGraph{Build: build, Jobs: jGraphs}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bGraph, nil
}

// ReadJobGraph makes and returns a JobGraph for the specified job.
func (s *QueueService) ReadJobGraph(ctx context.Context, txOrNil *store.Tx, jobID models.JobID) (*dto.JobGraph, error) {
	job, err := s.jobService.Read(ctx, txOrNil, jobID)
	if err != nil {
		return nil, fmt.Errorf("error reading job: %w", err)
	}
	steps, err := s.stepService.ListByJobID(ctx, txOrNil, job.ID)
	if err != nil {
		return nil, fmt.Errorf("error listing job steps: %w", err)
	}
	jGraph := &dto.JobGraph{
		Job:   job,
		Steps: steps,
	}
	return jGraph, nil
}

func (s *QueueService) CheckForTimeouts(timeout time.Duration) int {
	return s.timeoutChecker.CheckForTimeouts(timeout)
}

// UpdateJobStatus updates the status of a job.
// If the new status is WorkflowStatusFailed then an error can be provided to indicate what happened.
// This function will maintain the status of the build containing this job, to reflect the overall
// status of the build each time the status of a job is changed, and publish build events for status changes.
func (s *QueueService) UpdateJobStatus(ctx context.Context, txOrNil *store.Tx, jobID models.JobID, update dto.UpdateJobStatus) (*models.Job, error) {
	var (
		err error
		job *models.Job
	)
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		job, err = s.jobService.Read(ctx, tx, jobID)
		if err != nil {
			return fmt.Errorf("error reading job: %w", err)
		}
		job.ETag = models.GetETag(job, update.ETag)
		job.Error = update.Error
		jobStatusChanged := job.Status != update.Status
		job.Status = update.Status
		_, err = s.updateJob(ctx, tx, job, jobStatusChanged)
		if err != nil {
			return fmt.Errorf("error maintaining job status: %w", err)
		}
		_, err = s.maintainBuildStatus(ctx, tx, job.BuildID)
		if err != nil {
			return fmt.Errorf("error maintaining build status: %w", err)
		}
		return nil
	})
	return job, err
}

// UpdateJobFingerprint sets the fingerprint that has been calculated for a job. If the build is not configured
// with the force option (e.g. force=false), the server will attempt to locate previously a successful job with a
// matching fingerprint and indirect this job to it. If an indirection has been set, the agent must skip the job.
func (s *QueueService) UpdateJobFingerprint(ctx context.Context, jobID models.JobID, update dto.UpdateJobFingerprint) (*models.Job, error) {
	var (
		job *models.Job
		err error
	)
	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		job, err = s.jobService.Read(ctx, tx, jobID)
		if err != nil {
			return fmt.Errorf("error reading job: %w", err)
		}
		build, err := s.buildService.Read(ctx, tx, job.BuildID)
		if err != nil {
			return fmt.Errorf("error reading build: %w", err)
		}
		var indirectToJobID models.JobID
		if !build.Opts.Force {
			matchingJob, err := s.jobService.ReadByFingerprint(
				ctx,
				tx,
				job.RepoID,
				job.Workflow,
				job.Name,
				update.Fingerprint,
				&update.FingerprintHashType)
			if err != nil && gerror.ToNotFound(err) == nil {
				return fmt.Errorf("error reading job by fingerprint: %w", err)
			}
			if matchingJob != nil {
				indirectToJobID = matchingJob.ID
			}
		}
		// NOTE: We don't set the job's status here as the runner is expected to note the job was
		// indirected and to immediately come back to us with a status update. We may want to rethink this.
		job.UpdatedAt = models.NewTime(time.Now())
		job.ETag = models.GetETag(job, update.ETag)
		job.Fingerprint = update.Fingerprint
		job.FingerprintHashType = &update.FingerprintHashType
		job.IndirectToJobID = indirectToJobID
		err = s.jobService.Update(ctx, tx, job)
		if err != nil {
			return fmt.Errorf("error updating job: %w", err)
		}
		if indirectToJobID.Valid() {
			s.Infof("Job %s fingerprint updated (indirected to %s): %s", job.ID, indirectToJobID, update.Fingerprint)
		} else {
			s.Infof("Job %s fingerprint updated (no indirect made): %s", job.ID, update.Fingerprint)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

// UpdateStepStatus updates the status of a step that is executing under a job that was previously dequeued.
// If the new status is WorkflowStatusFailed then an error can be provided to indicate what happened.
func (s *QueueService) UpdateStepStatus(ctx context.Context, txOrNil *store.Tx, stepID models.StepID, update dto.UpdateStepStatus) (*models.Step, error) {
	var (
		step *models.Step
		err  error
	)
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		step, err = s.stepService.Read(ctx, tx, stepID)
		if err != nil {
			return fmt.Errorf("error reading step: %w", err)
		}
		job, err := s.jobService.Read(ctx, tx, step.JobID)
		if err != nil {
			return fmt.Errorf("error reading job for step: %w", err)
		}
		step.ETag = models.GetETag(step, update.ETag)
		step.Error = update.Error
		stepStatusChanged := step.Status != update.Status
		step.Status = update.Status
		_, err = s.updateStep(ctx, tx, job, step, stepStatusChanged)
		if err != nil {
			return fmt.Errorf("error maintaining job status: %w", err)
		}
		s.Infof("Step %s transitioned to: %s", step.ID, step.Status)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (s *QueueService) updateBuild(ctx context.Context, tx *store.Tx, build *models.Build, statusChanged bool) (*models.Build, error) {
	now := models.NewTime(time.Now())
	build.UpdatedAt = now
	switch build.Status {
	case models.WorkflowStatusQueued:
		build.Timings.QueuedAt = &now
	case models.WorkflowStatusSubmitted:
		build.Timings.SubmittedAt = &now
	case models.WorkflowStatusRunning:
		build.Timings.RunningAt = &now
	case models.WorkflowStatusSucceeded, models.WorkflowStatusFailed:
		build.Timings.FinishedAt = &now
		err := s.logService.Seal(ctx, tx, build.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing build log: %w", err)
		}
	case models.WorkflowStatusCanceled:
		build.Timings.CanceledAt = &now
		err := s.logService.Seal(ctx, tx, build.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing build log: %w", err)
		}
	default:
		return nil, fmt.Errorf("error unsupported build status %s", build.Status)
	}
	err := s.buildService.Update(ctx, tx, build)
	if err != nil {
		return nil, fmt.Errorf("error updating build: %w", err)
	}
	err = s.notifySCMBuildUpdated(ctx, tx, build)
	if err != nil {
		// Log and ignore errors while notifying SCM of build status change
		s.Error(err)
	}

	if statusChanged {
		err = s.eventService.PublishEvent(ctx, tx, models.NewBuildStatusChangedEventData(build))
		if err != nil {
			return nil, fmt.Errorf("error publishing step status changed event: %w", err)
		}
		s.Infof("Build %s transitioned to: %s", build.ID, build.Status)
	} else {
		s.Infof("Build %s updated (no change to status)", build.ID)
	}

	return build, nil
}

func (s *QueueService) updateJob(ctx context.Context, tx *store.Tx, job *models.Job, statusChanged bool) (*models.Job, error) {
	now := models.NewTime(time.Now())
	job.UpdatedAt = now
	switch job.Status {
	case models.WorkflowStatusQueued:
		job.Timings.QueuedAt = &now
	case models.WorkflowStatusSubmitted:
		job.Timings.SubmittedAt = &now
	case models.WorkflowStatusRunning:
		job.Timings.RunningAt = &now
	case models.WorkflowStatusSucceeded, models.WorkflowStatusFailed:
		job.Timings.FinishedAt = &now
		err := s.logService.Seal(ctx, tx, job.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing job log: %w", err)
		}
	case models.WorkflowStatusCanceled:
		job.Timings.CanceledAt = &now
		err := s.logService.Seal(ctx, tx, job.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing job log: %w", err)
		}
	default:
		return nil, fmt.Errorf("error unsupported job status %s", job.Status)
	}
	err := s.jobService.Update(ctx, tx, job)
	if err != nil {
		return nil, fmt.Errorf("error updating job: %w", err)
	}

	if statusChanged {
		err = s.eventService.PublishEvent(ctx, tx, models.NewJobStatusChangedEventData(job))
		if err != nil {
			return nil, fmt.Errorf("error publishing step status changed event: %w", err)
		}
		s.Infof("Job %s transitioned to: %s", job.ID, job.Status)
	} else {
		s.Infof("Job %s updated (no change to status)", job.ID)
	}

	return job, nil
}

func (s *QueueService) updateStep(
	ctx context.Context,
	tx *store.Tx,
	job *models.Job,
	step *models.Step,
	statusChanged bool,
) (*models.Step, error) {
	now := models.NewTime(time.Now())
	step.UpdatedAt = now
	switch step.Status {
	case models.WorkflowStatusQueued:
		step.Timings.QueuedAt = &now
	case models.WorkflowStatusSubmitted:
		step.Timings.SubmittedAt = &now
	case models.WorkflowStatusRunning:
		step.Timings.RunningAt = &now
	case models.WorkflowStatusSucceeded, models.WorkflowStatusFailed:
		step.Timings.FinishedAt = &now
		err := s.logService.Seal(ctx, tx, step.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing step log: %w", err)
		}
	case models.WorkflowStatusCanceled:
		step.Timings.CanceledAt = &now
		err := s.logService.Seal(ctx, tx, step.LogDescriptorID)
		if err != nil {
			return nil, fmt.Errorf("error sealing step log: %w", err)
		}
	default:
		return nil, fmt.Errorf("error unsupported step status %s", step.Status)
	}
	err := s.stepService.Update(ctx, tx, step)
	if err != nil {
		return nil, fmt.Errorf("error updating step: %w", err)
	}

	if statusChanged {
		err = s.eventService.PublishEvent(ctx, tx, models.NewStepStatusChangedEventData(job, step))
		if err != nil {
			return nil, fmt.Errorf("error publishing step status changed event: %w", err)
		}
		s.Infof("Step %s transitioned to: %s", step.ID, step.Status)
	} else {
		s.Infof("Step %s updated (no change to status)", step.ID)
	}

	return step, nil
}

// maintainBuildStatus ensures that the status of the build reflects the status of jobs under the build.
// This should be called any time the status of the build's jobs change, including when jobs are newly created.
func (s *QueueService) maintainBuildStatus(ctx context.Context, tx *store.Tx, buildID models.BuildID) (*models.Build, error) {
	// Take out a row lock on the build row to prevent race conditions when the status of two or more jobs
	// are updated concurrently. Do this before updating the job status.
	// The last goroutine to run maintainBuildStatus() can be sure that all updates to job status have
	// completed before maintainBuildStatus() starts, and therefore it will calculate the correct status.
	// This works when we are using the default transaction isolation level for Postgres of "Read Committed"
	// (constant sql.LevelReadCommitted) which means that within a transaction we will see data committed by
	// other transactions, but each SELECT within the transaction effectively takes its own snapshot.
	// This means one thread will see the other thread's update once it that update is complete, if we delay the
	// call to maintainBuildStatus() until the other thread is done.
	// IMPORTANT NOTE: This solution will NOT work if we change the Postgres isolation level to "Repeatable Read"
	// (constant LevelRepeatableRead) since each thread will only see its own changes. For this isolation
	// level we need to split the transaction into two transactions (the second one being the call to
	// maintainBuildStatus()) and keep retrying the second transaction until it succeeds. Note that at this
	// transaction isolation level we need to handle retries anyway because Postgres can return concurrent update
	// errors.
	// For sqlite we do aggressive locking so there can only be one update transaction at once anyway.
	err := s.buildService.LockRowForUpdate(ctx, tx, buildID)
	if err != nil {
		return nil, fmt.Errorf("error locking build: %w", err)
	}
	build, err := s.buildService.Read(ctx, tx, buildID)
	if err != nil {
		return nil, fmt.Errorf("error reading build: %w", err)
	}
	if build.Status.HasFinished() {
		return build, nil
	}
	jobs, err := s.jobService.ListByBuildID(ctx, tx, buildID)
	if err != nil {
		return nil, fmt.Errorf("error listing jobs for build: %w", err)
	}
	var (
		nFailedJobs int
		allJobsDone = true
		nextStatus  models.WorkflowStatus
		nextErr     *models.Error
	)
	for _, job := range jobs {
		if !job.Status.HasFinished() {
			allJobsDone = false
		}
		if job.Status == models.WorkflowStatusFailed || job.Status == models.WorkflowStatusCanceled {
			nFailedJobs++
		}
		if job.Status != models.WorkflowStatusQueued && build.Status == models.WorkflowStatusQueued {
			nextStatus = models.WorkflowStatusRunning
		}
	}
	if allJobsDone {
		if nFailedJobs > 0 {
			nextErr = models.NewError(fmt.Errorf("%d job(s) failed", nFailedJobs))
			nextStatus = models.WorkflowStatusFailed
		} else {
			nextStatus = models.WorkflowStatusSucceeded
		}
	}
	if nextStatus != "" {
		build.Error = nextErr
		buildStatusChanged := build.Status != nextStatus
		build.Status = nextStatus
		build, err = s.updateBuild(ctx, tx, build, buildStatusChanged)
		if err != nil {
			return nil, fmt.Errorf("error updating build status: %w", err)
		}
		if allJobsDone {
			// The build no longer needs an identity, so clean it up
			err := s.buildService.DeleteIdentity(ctx, tx, build.ID)
			if err != nil {
				return nil, fmt.Errorf("error deleting build identity when updating build status: %w", err)
			}
		}
		// TODO: Check if this is a duplicate log entry and remove if it is
		s.Infof("Build %s transitioned to: %s", build.ID, build.Status)
	}
	return build, nil
}

// notifySCMBuildStatusChanged allows SCM-specific code to be run when the status for a build changes.
// The allows (for example) publishing of status info to an SCM such as GitHub.
func (s *QueueService) notifySCMBuildUpdated(ctx context.Context, txOrNil *store.Tx, build *models.Build) error {
	// Find the SCM for the repo for this build
	repo, err := s.repoService.Read(ctx, txOrNil, build.RepoID)
	if err != nil {
		return err
	}
	// Only notify if the repo is associated with an external SCM
	if repo.ExternalID != nil {
		scmName := repo.ExternalID.ExternalSystem
		externalSCM, err := s.scmRegistry.Get(scmName)
		if err != nil {
			return fmt.Errorf("error getting SCM from registry for %q: %w", scmName, err)
		}
		// Allow some SCM-specific code to run (e.g. to create status updates in GitHub)
		err = externalSCM.NotifyBuildUpdated(ctx, txOrNil, build, repo)
		if err != nil {
			return fmt.Errorf("error notifying SCM %s of build status change: %w", scmName, err)
		}
	}
	return nil
}

// Enqueue a new build based on the specified build graph.
// Returns a build graph containing the jobs, as well as a Build object with the latest build status.
// Returns an error if there is a problem with the build graph (as well as any transient errors).
func (s *QueueService) enqueueBuild(ctx context.Context, txOrNil *store.Tx, graph *dto.BuildGraph) (*dto.BuildGraph, error) {
	return graph, s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.createBuild(ctx, tx, graph.Build)
		if err != nil {
			return fmt.Errorf("error creating build: %w", err)
		}
		_, err = s.enqueueJobs(ctx, tx, graph)
		return err
	})
}

// EnqueueJobs enqueues jobs for an existing build idempotently. Assumes if a job by the same name already exists
// within the build then it must be identical to the job in the specified in the build graph (so make sure you've
// validated the graph before calling this function).
// Returns only the newly enqueued jobs.
// Returns an error if there is a problem with the build graph (as well as any transient errors).
func (s *QueueService) enqueueJobs(ctx context.Context, txOrNil *store.Tx, bGraph *dto.BuildGraph) ([]*dto.JobGraph, error) {
	var jGraphs []*dto.JobGraph
	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.failJobsWithNoCompatibleRunner(ctx, tx, bGraph.Jobs)
		if err != nil {
			return fmt.Errorf("error checking for compatible runners: %w", err)
		}
		err = bGraph.Walk(false, func(job *dto.JobGraph) error {
			_, err := s.jobService.Read(ctx, tx, job.ID)
			if err != nil && !gerror.IsNotFound(err) {
				return fmt.Errorf("error reading existing job: %w", err)
			}
			if err == nil { // job already exists, nothing to do
				return nil
			}
			err = s.createJob(ctx, tx, bGraph.Build, job.Job)
			if err != nil {
				return fmt.Errorf("error creating job: %w", err)
			}
			jGraphs = append(jGraphs, job) // we created it, track it
			return job.Walk(false, func(step *models.Step) error {
				err := s.createStep(ctx, tx, job.Job, step)
				if err != nil {
					return fmt.Errorf("error creating step: %w", err)
				}
				return nil
			})
		})
		if err != nil {
			return err
		}
		bGraph.Build, err = s.maintainBuildStatus(ctx, tx, bGraph.Build.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return jGraphs, nil
}

// makeNewBuildGraph creates and validates a build graph for a new build, for the specified commit.
func (s *QueueService) makeNewBuildGraph(
	repoID models.RepoID,
	commitID models.CommitID,
	buildDefinition *models.BuildDefinition,
	ref string,
	opts *models.BuildOptions) (*dto.BuildGraph, error) {

	now := models.NewTime(time.Now())

	bGraph := &dto.BuildGraph{Build: &models.Build{
		ID:        models.NewBuildID(),
		CreatedAt: now,
		RepoID:    repoID,
		CommitID:  commitID,
		Ref:       ref,
		Status:    models.WorkflowStatusQueued,
		Timings: models.WorkflowTimings{
			QueuedAt: &now,
		},
		Opts: models.BuildOptions{},
	}}
	err := s.makeJobGraphsAndAppendToBuildGraph(bGraph, buildDefinition.Jobs)
	if err != nil {
		return nil, fmt.Errorf("error making job graphs: %w", err)
	}

	bGraph.PopulateDefaults()
	if opts != nil {
		bGraph.Opts = *opts
		if len(opts.NodesToRun) > 0 {
			// Only trim the build graph if all nodes in the NodesToRun option specify jobs;
			// this is of limited usefulness because it doesn't work with dynamic builds
			allNodesAreJobs := true
			for _, node := range opts.NodesToRun {
				if node.JobName == "" {
					allNodesAreJobs = false
				}
			}
			if allNodesAreJobs {
				err := bGraph.Trim(opts.NodesToRun)
				if err != nil {
					return nil, errors.Wrap(err, "error trimming build")
				}
			}
		}
	}

	err = bGraph.Validate()
	if err != nil {
		return nil, err
	}

	return bGraph, nil
}

// makeJobGraphs creates (but does not persist) Job Graphs for a set of Job Definitions, in the context of a build.
// It does not validate the new job graphs.
func (s *QueueService) makeJobGraphs(build *models.Build, jobs []models.JobDefinition) ([]*dto.JobGraph, error) {
	var (
		jGraphs []*dto.JobGraph
		now     = models.NewTime(time.Now())
	)
	for _, job := range jobs {
		// NOTE: Very important that we use JobDefinition here as it includes the job's steps
		hash, err := hashstructure.Hash(job, hashstructure.FormatV2, &hashstructure.HashOptions{SlicesAsSets: true})
		if err != nil {
			return nil, fmt.Errorf("error hashing job definiton data: %w", err)
		}
		var steps []*models.Step
		for _, stepDef := range job.Steps {
			steps = append(steps, &models.Step{
				StepMetadata: models.StepMetadata{
					ID:        models.NewStepID(),
					CreatedAt: now,
				},
				StepData: models.StepData{
					StepDefinitionData: stepDef.StepDefinitionData,
					RepoID:             build.RepoID,
					Status:             models.WorkflowStatusQueued,
					Timings: models.WorkflowTimings{
						QueuedAt: &now,
					},
				},
			})
		}
		jGraphs = append(jGraphs, &dto.JobGraph{
			Job: &models.Job{
				JobMetadata: models.JobMetadata{
					ID:        models.NewJobID(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				JobData: models.JobData{
					JobDefinitionData: job.JobDefinitionData,
					BuildID:           build.ID,
					RepoID:            build.RepoID,
					CommitID:          build.CommitID,
					Ref:               build.Ref,
					Status:            models.WorkflowStatusQueued,
					Timings: models.WorkflowTimings{
						QueuedAt: &now,
					},
					DefinitionDataHashType: models.HashTypeFNV,
					DefinitionDataHash:     fmt.Sprintf("%x", hash),
				},
			},
			Steps: steps,
		})
	}
	return jGraphs, nil
}

// makeJobGraphsAndAppendToBuildGraph creates (but does not persist) Job Graphs for a set of Job Definitions,
// in the context of a build, and appends them to the build graph.
// It does not validate the new job graphs, or the updated build graph.
func (s *QueueService) makeJobGraphsAndAppendToBuildGraph(bGraph *dto.BuildGraph, jobs []models.JobDefinition) error {
	// Validate that we won't exceed the maximum number of jobs for the build
	if len(bGraph.Jobs)+len(jobs) > s.limits.MaxJobsPerBuild {
		return gerror.NewErrValidationFailed(
			fmt.Sprintf("Too many jobs in build; a maximum of %d jobs are allowed in a build", s.limits.MaxJobsPerBuild))
	}

	jGraphs, err := s.makeJobGraphs(bGraph.Build, jobs)
	if err != nil {
		return err
	}
	for _, job := range jGraphs {
		bGraph.Jobs = append(bGraph.Jobs, job)
	}
	return nil
}

// createFailedBuild creates a failed build with the minimal information available at the time of creation.
// We use this in case we are unable to create a build during the normal Enqueuing process where we need a build to
// represent a commit that is in a failed state.
func (s *QueueService) createFailedBuild(ctx context.Context, txOrNil *store.Tx, commit *models.Commit, ref string, opts *models.BuildOptions, err error) (*dto.BuildGraph, error) {
	now := models.NewTime(time.Now())
	graph := &dto.BuildGraph{
		Build: &models.Build{
			ID:        models.NewBuildID(),
			RepoID:    commit.RepoID,
			CreatedAt: now,
			CommitID:  commit.ID,
			Ref:       ref,
			Status:    models.WorkflowStatusFailed,
			Timings: models.WorkflowTimings{
				QueuedAt:    &now,
				SubmittedAt: &now,
				RunningAt:   &now,
				FinishedAt:  &now,
			},
			Error: models.NewError(err),
		},
	}
	graph.PopulateDefaults()
	if opts != nil {
		graph.Opts = *opts
	}
	return graph, s.createBuild(ctx, txOrNil, graph.Build)
}

// failJobsWithNoCompatibleRunner checks that a compatible runner exists that is capable
// of running each of the specified jobs. If no compatible runner is found the job is marked
// as failed *in-memory*. The caller is responsible for subsequently persisting the jobs.
func (s *QueueService) failJobsWithNoCompatibleRunner(ctx context.Context, tx *store.Tx, jobs []*dto.JobGraph) error {
	for _, job := range jobs {
		runnable, err := s.runnerService.RunnerCompatibleWithJob(ctx, tx, job.Job)
		if err != nil {
			return fmt.Errorf("error checking for compatible runner: %w", err)
		}
		if !runnable {
			err := models.NewError(fmt.Errorf("No runners are capable of running this job"))
			job.Status = models.WorkflowStatusFailed
			job.Error = err
			for _, step := range job.Steps {
				step.Status = models.WorkflowStatusFailed
				// NOTE intentionally do not set step error here, as it just duplicates the job error
			}
		}
	}
	return nil
}

func (s *QueueService) createBuild(ctx context.Context, tx *store.Tx, build *models.Build) error {
	logDescriptor, err := s.logService.Create(ctx, tx, models.NewLogDescriptor(models.NewTime(time.Now()), models.LogDescriptorID{}, build.ID.ResourceID))
	if err != nil {
		return fmt.Errorf("error creating log descriptor: %w", err)
	}
	build.LogDescriptorID = logDescriptor.ID
	err = s.buildService.Create(ctx, tx, build)
	if err != nil {
		return err
	}
	_, err = s.updateBuild(ctx, tx, build, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *QueueService) createJob(ctx context.Context, tx *store.Tx, build *models.Build, job *models.Job) error {
	logDescriptor, err := s.logService.Create(ctx, tx, models.NewLogDescriptor(models.NewTime(time.Now()), build.LogDescriptorID, job.ID.ResourceID))
	if err != nil {
		return fmt.Errorf("error creating log descriptor: %w", err)
	}
	create := &dto.CreateJob{
		Job:   job,
		Build: build,
	}
	create.LogDescriptorID = logDescriptor.ID
	err = s.jobService.Create(ctx, tx, create)
	if err != nil {
		return err
	}
	_, err = s.updateJob(ctx, tx, job, true)
	if err != nil {
		return err
	}
	return nil
}

func (s *QueueService) createStep(ctx context.Context, tx *store.Tx, job *models.Job, step *models.Step) error {
	logDescriptor, err := s.logService.Create(ctx, tx, models.NewLogDescriptor(models.NewTime(time.Now()), job.LogDescriptorID, step.ID.ResourceID))
	if err != nil {
		return fmt.Errorf("error creating log descriptor: %w", err)
	}
	create := &dto.CreateStep{
		Step: step,
		Job:  job,
	}
	create.LogDescriptorID = logDescriptor.ID
	err = s.stepService.Create(ctx, tx, create)
	if err != nil {
		return err
	}
	_, err = s.updateStep(ctx, tx, job, step, true)
	if err != nil {
		return err
	}
	return nil
}
