package jobs

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	_ = models.MutableResource(&models.Job{})
	_ = models.SoftDeletableResource(&models.Job{})
	store.MustDBModel(&models.Job{})
}

type JobStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *JobStore {
	return &JobStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.Job{}),
	}
}

// Create a new job.
// Returns store.ErrAlreadyExists if a job with matching unique properties already exists.
func (d *JobStore) Create(ctx context.Context, txOrNil *store.Tx, job *models.Job) error {
	return d.table.Create(ctx, txOrNil, job)
}

// Read an existing job, looking it up by ResourceID.
// Returns models.ErrNotFound if the job does not exist.
func (d *JobStore) Read(ctx context.Context, txOrNil *store.Tx, id models.JobID) (*models.Job, error) {
	job := &models.Job{}
	return job, d.table.ReadByID(ctx, txOrNil, id.ResourceID, job)
}

// ReadByName reads an existing job, looking it up by build, workflow and job name.
// Returns models.ErrNotFound if the job is not found.
func (d *JobStore) ReadByName(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	workflow models.ResourceName,
	jobName models.ResourceName,
) (*models.Job, error) {
	job := &models.Job{}
	return job, d.table.ReadWhere(ctx, txOrNil, job,
		goqu.Ex{"job_build_id": buildID},
		goqu.Ex{"job_workflow": workflow},
		goqu.Ex{"job_name": jobName},
	)
}

// ReadByFingerprint reads the most recent successful job inside a repo with a matching workflow, name and fingerprint.
// Returns models.ErrNotFound if the job does not exist.
func (d *JobStore) ReadByFingerprint(
	ctx context.Context,
	txOrNil *store.Tx,
	repoID models.RepoID,
	workflow models.ResourceName,
	jobName models.ResourceName,
	jobFingerprint string,
	jobFingerprintHashType *models.HashType) (*models.Job, error) {

	// TODO this is business logic - it should probably be a search
	job := &models.Job{}
	ds := goqu.
		Select(job).
		From(d.table.TableName()).
		Where(goqu.Ex{
			"job_repo_id":               repoID,
			"job_workflow":              workflow,
			"job_name":                  jobName,
			"job_fingerprint":           jobFingerprint,
			"job_fingerprint_hash_type": jobFingerprintHashType,
			"job_status":                models.WorkflowStatusSucceeded,
			"job_error":                 nil,
			"job_indirect_to_job_id":    nil,
		}).
		Order(goqu.I("job_created_at").Desc()).
		Limit(1)
	return job, d.table.ReadIn(ctx, txOrNil, job, ds)
}

// Update an existing job with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *JobStore) Update(ctx context.Context, txOrNil *store.Tx, job *models.Job) error {
	return d.table.UpdateByID(ctx, txOrNil, job)
}

// ListByBuildID gets all jobs that are associated with the specified build id.
func (d *JobStore) ListByBuildID(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) ([]*models.Job, error) {
	jobSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Job{}).
		Where(goqu.Ex{"job_build_id": buildID})
	pagination := models.NewPagination(10000, nil) // TODO this is a total hack
	var jobs []*models.Job
	_, err := d.table.ListIn(ctx, txOrNil, &jobs, pagination, jobSelect)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListByStatus returns all jobs that have the specified status, regardless of who owns the jobs or which build
// they are part of. Use cursor to page through results, if any.
func (d *JobStore) ListByStatus(ctx context.Context, txOrNil *store.Tx, status models.WorkflowStatus, pagination models.Pagination) ([]*models.Job, *models.Cursor, error) {
	jobSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Job{}).
		Where(goqu.Ex{"job_status": status})
	var jobs []*models.Job
	cursor, err := d.table.ListIn(ctx, txOrNil, &jobs, pagination, jobSelect)
	if err != nil {
		return nil, nil, err
	}
	return jobs, cursor, nil
}

// ListDependencies lists all jobs that the specified job depends on.
// Deferred dependencies (on jobs in other workflows that don't yet exist) will not be listed.
func (d *JobStore) ListDependencies(ctx context.Context, txOrNil *store.Tx, jobID models.JobID) ([]*models.Job, error) {
	jobSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Job{}).
		Join(goqu.T("jobs_depend_on_jobs"), goqu.On(goqu.Ex{"jobs.job_id": goqu.I("jobs_depend_on_jobs.jobs_depend_on_jobs_target_job_id")})).
		Where(goqu.Ex{"jobs_depend_on_jobs_source_job_id": jobID})
	pagination := models.NewPagination(10000, nil) // TODO this is a total hack
	var jobs []*models.Job
	_, err := d.table.ListIn(ctx, txOrNil, &jobs, pagination, jobSelect)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// CreateDependency records a dependency between jobs where source depends on target.
func (d *JobStore) CreateDependency(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	sourceJobID models.JobID,
	targetJobID models.JobID,
) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Insert(
			goqu.T("jobs_depend_on_jobs")).Rows(
			goqu.Record{
				"jobs_depend_on_jobs_build_id":        buildID,
				"jobs_depend_on_jobs_source_job_id":   sourceJobID,
				"jobs_depend_on_jobs_target_job_id":   targetJobID,
				"jobs_depend_on_jobs_target_workflow": nil,
				"jobs_depend_on_jobs_target_job_name": nil,
			},
		).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// CreateDeferredDependency records a dependency between a job and another job in another workflow
// which does not yet exist.
func (d *JobStore) CreateDeferredDependency(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	sourceJobID models.JobID,
	targetWorkflow models.ResourceName,
	targetJobName models.ResourceName,
) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Insert(
			goqu.T("jobs_depend_on_jobs")).Rows(
			goqu.Record{
				"jobs_depend_on_jobs_build_id":        buildID,
				"jobs_depend_on_jobs_source_job_id":   sourceJobID,
				"jobs_depend_on_jobs_target_job_id":   nil, // not resolved to a Job ID yet
				"jobs_depend_on_jobs_target_workflow": targetWorkflow,
				"jobs_depend_on_jobs_target_job_name": targetJobName,
			},
		).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create unresolved cross workflow dependency query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// UpdateDeferredDependencies updates any dependencies that refer to the target job's workflow and job name,
// clearing those fields and setting target job ID instead. This has the effect of converting all dependencies
// on the target job from deferred dependencies into 'real' dependencies.
func (d *JobStore) UpdateDeferredDependencies(ctx context.Context, txOrNil *store.Tx, targetJob *models.Job) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		res, err := d.table.LogUpdate(db.Update(goqu.T("jobs_depend_on_jobs")).
			Set(goqu.Record{
				"jobs_depend_on_jobs_target_job_id":   targetJob.ID,
				"jobs_depend_on_jobs_target_workflow": nil,
				"jobs_depend_on_jobs_target_job_name": nil,
			}).Where(goqu.Ex{
			"jobs_depend_on_jobs_build_id":        targetJob.BuildID,
			"jobs_depend_on_jobs_target_workflow": targetJob.Workflow,
			"jobs_depend_on_jobs_target_job_name": targetJob.Name,
			"jobs_depend_on_jobs_target_job_id":   nil,
		})).
			Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing update query to update job dependencies: %w", store.MakeStandardDBError(err))
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error reading rows affected when updating job dependencies: %w", store.MakeStandardDBError(err))
		}
		d.table.Tracef("Updated %d dependencies for job '%s.%s'", rowsAffected, targetJob.Workflow, targetJob.Name)

		return nil
	})
}

// CreateLabel records a label against a job.
func (d *JobStore) CreateLabel(ctx context.Context, txOrNil *store.Tx, jobID models.JobID, label models.Label) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Insert(
			goqu.T("job_labels")).Rows(
			goqu.Record{
				"job_label_job_id": jobID,
				"job_label_label":  label},
		).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// FindQueuedJob locates a queued job that the runner is capable of running, and which is ready for
// execution (e.g all dependencies are completed).
// Returns models.ErrNotFound if the job does not exist.
func (d *JobStore) FindQueuedJob(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) (*models.Job, error) {
	// Find other jobs that queued_jobs.job_id depends on that are not yet done, if any, which would stop it from
	// being eligible to run
	dependencySubQuery := goqu.From(goqu.T("jobs").As("candidate_jobs")).
		Select(goqu.I("job_dependency.job_id")).
		Join(goqu.T("jobs_depend_on_jobs"), goqu.On(goqu.Ex{"candidate_jobs.job_id": goqu.I("jobs_depend_on_jobs.jobs_depend_on_jobs_source_job_id")})).
		Join(goqu.T("jobs").As("job_dependency"), goqu.On(goqu.Ex{"job_dependency.job_id": goqu.I("jobs_depend_on_jobs.jobs_depend_on_jobs_target_job_id")})).
		Where(goqu.I("jobs_depend_on_jobs.jobs_depend_on_jobs_target_job_id").IsNotNull()).
		Where(goqu.Ex{
			"jobs_depend_on_jobs_source_job_id": goqu.I("queued_jobs.job_id"),
			"job_dependency.job_status":         goqu.Op{"notIn": []models.WorkflowStatus{models.WorkflowStatusCanceled, models.WorkflowStatusFailed, models.WorkflowStatusSucceeded}},
		}).
		Limit(1)

	// Find deferred cross-workflow dependencies for queued_jobs.job_id, if any, which would stop it from being
	// eligible to run
	deferredDependencySubQuery := goqu.From(goqu.T("jobs").As("candidate_deferred_jobs")).
		Select(goqu.I("jobs_depend_on_jobs_target_job_name")).
		Join(goqu.T("jobs_depend_on_jobs"), goqu.On(goqu.Ex{"candidate_deferred_jobs.job_id": goqu.I("jobs_depend_on_jobs.jobs_depend_on_jobs_source_job_id")})).
		Where(
			goqu.Ex{"jobs_depend_on_jobs_source_job_id": goqu.I("queued_jobs.job_id")},
			goqu.C("jobs_depend_on_jobs_target_workflow").IsNotNull(),
			goqu.C("jobs_depend_on_jobs_target_job_name").IsNotNull(),
		).
		Limit(1)

	var runnerSupportedJobTypes []string
	for _, kind := range runner.SupportedJobTypes {
		runnerSupportedJobTypes = append(runnerSupportedJobTypes, string(kind))
	}

	jobSelect := goqu.From(goqu.T("jobs").As("queued_jobs")).
		Select(&models.Job{}). // TODO: use SELECT FOR UPDATE SKIP LOCKED for Postgres/MySQL
		Join(goqu.T("repos"), goqu.On(goqu.Ex{"queued_jobs.job_repo_id": goqu.I("repos.repo_id")})).
		Where(goqu.Ex{"repos.repo_legal_entity_id": runner.LegalEntityID}). // only jobs under repos owned by correct legal entity
		Where(goqu.Ex{"job_status": models.WorkflowStatusQueued}).
		Where(goqu.V(dependencySubQuery).IsNull()).         // where all jobs this one depends on are done
		Where(goqu.V(deferredDependencySubQuery).IsNull()). // where this job has no deferred cross-workflow dependencies
		Where(goqu.Ex{"job_type": goqu.Op{"in": runnerSupportedJobTypes}})

	// All runners can run jobs that don't require any labels
	labelOrs := []goqu.Expression{goqu.I("job_runs_on").IsNull()}

	// Some runners may additionally be able to run jobs that require labels, if the runner also has labels
	// (and they match of course...)
	if len(runner.Labels) > 0 {
		var runnerLabels []string
		for _, label := range runner.Labels {
			runnerLabels = append(runnerLabels, string(label))
		}
		// Locate a job that the runner has all the required labels for.
		labelSubQuery := goqu.From(goqu.T("job_labels")).
			Select(goqu.I("job_labels.job_label_job_id")).
			Where(goqu.Ex{
				"job_labels.job_label_job_id": goqu.I("queued_jobs.job_id"),
				"job_labels.job_label_label":  goqu.Op{"notIn": runnerLabels},
			}).
			Limit(1)
		labelOrs = append(labelOrs, goqu.V(labelSubQuery).IsNull()) // where the runner has all labels this job needs
	}
	jobSelect = jobSelect.Where(goqu.Or(labelOrs...))

	jobSelect = jobSelect.
		Order(goqu.I("job_created_at").Asc()).
		Limit(1)

	job := &models.Job{}
	return job, d.table.ReadIn(ctx, txOrNil, job, jobSelect)
}
