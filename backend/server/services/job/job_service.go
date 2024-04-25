package job

import (
	"context"
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type JobService struct {
	db                *store.DB
	jobStore          store.JobStore
	ownershipStore    store.OwnershipStore
	resourceLinkStore store.ResourceLinkStore
	logger.Log
}

func NewJobService(
	db *store.DB,
	jobStore store.JobStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	logFactory logger.LogFactory) *JobService {
	return &JobService{
		db:                db,
		jobStore:          jobStore,
		ownershipStore:    ownershipStore,
		resourceLinkStore: resourceLinkStore,
		Log:               logFactory("JobService"),
	}
}

// Read an existing job, looking it up by ID.
// Returns models.ErrNotFound if the job does not exist.
func (s *JobService) Read(ctx context.Context, txOrNil *store.Tx, id models.JobID) (*models.Job, error) {
	return s.jobStore.Read(ctx, txOrNil, id)
}

// ReadByFingerprint reads the most recent successful job inside a repo with a matching workflow, name
// and fingerprint. Returns models.ErrNotFound if the job does not exist.
func (s *JobService) ReadByFingerprint(
	ctx context.Context,
	txOrNil *store.Tx,
	repoID models.RepoID,
	workflow models.ResourceName,
	jobName models.ResourceName,
	jobFingerprint string,
	jobFingerprintHashType *models.HashType) (*models.Job, error) {
	return s.jobStore.ReadByFingerprint(ctx, txOrNil, repoID, workflow, jobName, jobFingerprint, jobFingerprintHashType)
}

// ListDependencies lists all jobs that the specified job depends on.
func (s *JobService) ListDependencies(ctx context.Context, txOrNil *store.Tx, jobID models.JobID) ([]*models.Job, error) {
	return s.jobStore.ListDependencies(ctx, txOrNil, jobID)
}

// FindQueuedJob locates a queued job that the runner is capable of running, and which is ready for
// execution (e.g all dependencies are completed).
func (s *JobService) FindQueuedJob(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) (*models.Job, error) {
	return s.jobStore.FindQueuedJob(ctx, txOrNil, runner)
}

// Create a new job.
// Returns store.ErrAlreadyExists if a job with matching unique properties already exists.
func (s *JobService) Create(ctx context.Context, txOrNil *store.Tx, create *dto.CreateJob) error {
	err := create.Validate()
	if err != nil {
		return fmt.Errorf("error validating job: %w", err)
	}
	now := models.NewTime(time.Now())
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err = s.jobStore.Create(ctx, tx, create.Job)
		if err != nil {
			return fmt.Errorf("error creating job: %w", err)
		}
		for _, depends := range create.Depends {
			err = s.createDependency(ctx, tx, create.Job, depends)
			if err != nil {
				return err
			}
		}
		// Update any deferred dependencies that refer to the new job's workflow and name to instead refer to the
		// new job by ID, making them 'real' dependencies and potentially allowing the dependent jobs to be run.
		err = s.jobStore.UpdateDeferredDependencies(ctx, tx, create.Job)
		if err != nil {
			return fmt.Errorf("error updating deffered dependencies to real dependencies for job: %w", err)
		}
		for _, label := range create.RunsOn {
			err := s.jobStore.CreateLabel(ctx, tx, create.ID, label)
			if err != nil {
				return fmt.Errorf("error creating job label: %w", err)
			}
		}
		ownership := models.NewOwnership(now, create.BuildID.ResourceID, create.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return fmt.Errorf("error creating ownership: %w", err)
		}
		// Make resource links that include the workflow name, not just job name (which isn't unique)
		jobLinkWrapper := models.NewJobLinkWrapper(create.Job)
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, jobLinkWrapper)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Created job %q", create.ID)
		return nil
	})
}

// createDependency adds a new dependency for a job.
// If the (dependent) job and the dependency job are in the same workflow then the dependency job must already exist.
// If they are in different workflows and the dependency job doesn't exist yet then a deferred dependency
// will be created.
func (s *JobService) createDependency(
	ctx context.Context,
	txOrNil *store.Tx,
	job *models.Job,
	dependency *models.JobDependency,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		dependencyJob, err := s.jobStore.ReadByName(ctx, tx, job.BuildID, dependency.Workflow, dependency.JobName)
		if err != nil {
			if gerror.IsNotFound(err) {
				// It's OK if dependency job doesn't exist yet, as long as it will be in a different workflow
				if job.Workflow != dependencyJob.Workflow {
					dependencyJob = nil
					err = nil
				} else {
					return fmt.Errorf("error dependent job in same workflow was not found: %w", err)
				}
			} else {
				// all errors other than Not Found should be returned
				return fmt.Errorf("error reading dependent job: %w", err)
			}
		}
		if dependencyJob != nil {
			err = s.jobStore.CreateDependency(ctx, tx, job.BuildID, job.ID, dependencyJob.ID)
			if err != nil {
				return fmt.Errorf("error creating job dependency: %w", err)
			}
		} else {
			err = s.jobStore.CreateDeferredDependency(ctx, tx, job.BuildID, job.ID, dependency.Workflow, dependency.JobName)
			if err != nil {
				return fmt.Errorf("error creating job deferred dependency: %w", err)
			}
		}
		return nil
	})
}

// Update an existing job with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *JobService) Update(ctx context.Context, txOrNil *store.Tx, job *models.Job) error {
	err := job.Validate()
	if err != nil {
		return fmt.Errorf("error validating job: %w", err)
	}
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err = s.jobStore.Update(ctx, tx, job)
		if err != nil {
			return fmt.Errorf("error updating job: %w", err)
		}

		// Make resource links that include the workflow name, not just job name (which isn't unique)
		jobLinkWrapper := models.NewJobLinkWrapper(job)
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, jobLinkWrapper)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Updated job %q", job.ID)
		return nil
	})
}

// ListByStatus returns all jobs that have the specified status, regardless of who owns the jobs or which build
// they are part of. Use cursor to page through results, if any.
func (s *JobService) ListByStatus(
	ctx context.Context,
	txOrNil *store.Tx,
	status models.WorkflowStatus,
	pagination models.Pagination,
) ([]*models.Job, *models.Cursor, error) {
	return s.jobStore.ListByStatus(ctx, txOrNil, status, pagination)
}

// ListByBuildID gets all jobs that are associated with the specified build id.
func (s *JobService) ListByBuildID(ctx context.Context, txOrNil *store.Tx, id models.BuildID) ([]*models.Job, error) {
	return s.jobStore.ListByBuildID(ctx, txOrNil, id)
}
