package runners

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
)

func init() {
	_ = models.MutableResource(&models.Runner{})
	_ = models.SoftDeletableResource(&models.Runner{})
	store.MustDBModel(&models.Runner{})
}

type RunnerStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *RunnerStore {
	return &RunnerStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.Runner{}),
	}
}

// Create a new runner record in the database.
// Returns store.ErrAlreadyExists if a runner with matching unique properties already exists.
func (d *RunnerStore) Create(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) error {
	return d.table.Create(ctx, txOrNil, runner)
}

// Read an existing runner, looking it up by ResourceID.
// Returns models.ErrNotFound if the runner does not exist.
func (d *RunnerStore) Read(ctx context.Context, txOrNil *store.Tx, id models.RunnerID) (*models.Runner, error) {
	runner := &models.Runner{}
	return runner, d.table.ReadByID(ctx, txOrNil, id.ResourceID, runner)
}

// ReadByName reads an existing runner, looking it up by name and the ID of the legal entity that owns the runner.
// Returns models.ErrNotFound if the runner is not found.
func (d *RunnerStore) ReadByName(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityID models.LegalEntityID,
	name models.ResourceName,
) (*models.Runner, error) {
	runner := &models.Runner{}
	return runner, d.table.ReadWhere(ctx, txOrNil, runner,
		goqu.Ex{"runner_legal_entity_id": legalEntityID},
		goqu.Ex{"runner_name": name},
	)
}

// Update an existing runner.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *RunnerStore) Update(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) error {
	return d.table.UpdateByID(ctx, txOrNil, runner)
}

// LockRowForUpdate takes out an exclusive row lock on the runner table row for the specified runner.
// This must be done within a transaction, and will block other transactions from locking or updating
// the row until this transaction ends.
func (d *RunnerStore) LockRowForUpdate(ctx context.Context, tx *store.Tx, id models.RunnerID) error {
	return d.table.LockRowForUpdate(ctx, tx, id.ResourceID)
}

// SoftDelete soft deletes an existing runner.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *RunnerStore) SoftDelete(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) error {
	return d.table.SoftDelete(ctx, txOrNil, runner)
}

// CreateLabel records a label against a runner.
func (d *RunnerStore) CreateLabel(ctx context.Context, txOrNil *store.Tx, runnerID models.RunnerID, label models.Label) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Insert(
			goqu.T("runner_labels")).Rows(
			goqu.Record{
				"runner_label_runner_id": runnerID,
				"runner_label_label":     label},
		).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// DeleteLabel deletes an existing label from a runner.
func (d *RunnerStore) DeleteLabel(ctx context.Context, txOrNil *store.Tx, runnerID models.RunnerID, label models.Label) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Delete(goqu.T("runner_labels")).
			Where(goqu.I("runner_label_runner_id").Eq(runnerID)).
			Where(goqu.I("runner_label_label").Eq(label)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing delete query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// CreateSupportedJobType records a supported job type against a runner.
func (d *RunnerStore) CreateSupportedJobType(ctx context.Context, txOrNil *store.Tx, runnerID models.RunnerID, kind models.JobType) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Insert(
			goqu.T("runner_supported_job_types")).Rows(
			goqu.Record{
				"runner_supported_job_types_runner_id": runnerID,
				"runner_supported_job_types_job_type":  kind},
		).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// DeleteSupportedJobType deletes an existing supported job type from a runner.
func (d *RunnerStore) DeleteSupportedJobType(ctx context.Context, txOrNil *store.Tx, runnerID models.RunnerID, kind models.JobType) error {
	return d.db.Write2(txOrNil, func(db store.Writer) error {
		_, err := db.Delete(goqu.T("runner_supported_job_types")).
			Where(goqu.I("runner_supported_job_types_runner_id").Eq(runnerID)).
			Where(goqu.I("runner_supported_job_types_job_type").Eq(kind)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing delete query: %w", store.MakeStandardDBError(err))
		}
		return nil
	})
}

// RunnerCompatibleWithJob returns true if a runner exists that is capable of running job.
func (d *RunnerStore) RunnerCompatibleWithJob(ctx context.Context, txOrNil *store.Tx, job *models.Job) (bool, error) {
	query := d.table.Dialect().
		From(d.table.TableName()).
		Select(&models.Runner{}).
		Join(goqu.T("repos"), goqu.On(goqu.Ex{"runners.runner_legal_entity_id": goqu.I("repos.repo_legal_entity_id")})).
		Where(goqu.Ex{"repos.repo_id": job.RepoID}).
		Where(goqu.I("runners.runner_deleted_at").IsNull())

	if len(job.RunsOn) > 0 {
		var jobLabels []string
		for _, label := range job.RunsOn {
			jobLabels = append(jobLabels, label.String())
		}
		// Locate a runner that has all the labels the job needs.
		labelSubQuery := d.table.Dialect().From(goqu.T("runner_labels")).
			Select(goqu.COUNT("*")).
			Where(goqu.Ex{"runner_labels.runner_label_runner_id": goqu.I("runners.runner_id")}).
			Where(goqu.I("runner_labels.runner_label_label").In(jobLabels))
		query = query.Where(goqu.V(labelSubQuery).Eq(len(jobLabels)))
	}

	query = query.Limit(1)

	runner := &models.Runner{}
	err := d.table.ReadIn(ctx, txOrNil, runner, query)
	if err != nil {
		if gerror.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Search all runners. If searcher is set, the results will be limited to runners the searcher is authorized to
// see (via the read:runner permission). Use cursor to page through results, if any.
func (d *RunnerStore) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.RunnerSearch) ([]*models.Runner, *models.Cursor, error) {
	runnersSelect := d.table.Dialect().
		From(d.table.TableName()).
		Select(&models.Runner{})
	if !searcher.IsZero() {
		runnersSelect = authorizations.WithIsAuthorizedListFilter(runnersSelect, searcher, *models.RunnerReadOperation, "runner_id")
	}

	if search.LegalEntityID != nil {
		runnersSelect = runnersSelect.
			Where(goqu.Ex{"runner_legal_entity_id": search.LegalEntityID})
	}

	var runners []*models.Runner
	cursor, err := d.table.ListIn(ctx, txOrNil, &runners, search.Pagination, runnersSelect)
	if err != nil {
		return nil, nil, err
	}
	return runners, cursor, nil
}
