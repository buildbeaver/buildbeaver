package steps

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	_ = models.MutableResource(&models.Step{})
	_ = models.SoftDeletableResource(&models.Step{})
	store.MustDBModel(&models.Step{})
}

type StepStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *StepStore {
	return &StepStore{
		table: store.NewResourceTable(db, logFactory, &models.Step{}),
	}
}

// Create a new step.
// Returns store.ErrAlreadyExists if a step with matching unique properties already exists.
func (d *StepStore) Create(ctx context.Context, txOrNil *store.Tx, step *models.Step) error {
	return d.table.Create(ctx, txOrNil, step)
}

// Read an existing step, looking it up by ResourceID.
// Returns models.ErrNotFound if the step does not exist.
func (d *StepStore) Read(ctx context.Context, txOrNil *store.Tx, id models.StepID) (*models.Step, error) {
	step := &models.Step{}
	return step, d.table.ReadByID(ctx, txOrNil, id.ResourceID, step)
}

// Update an existing step with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *StepStore) Update(ctx context.Context, txOrNil *store.Tx, step *models.Step) error {
	return d.table.UpdateByID(ctx, txOrNil, step)
}

// ListByJobID gets all steps that are associated with the specified job id.
func (d *StepStore) ListByJobID(ctx context.Context, txOrNil *store.Tx, jobID models.JobID) ([]*models.Step, error) {
	stepsSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Step{}).
		Where(goqu.Ex{"step_job_id": jobID})
	pagination := models.NewPagination(10000, nil) // TODO this is a total hack
	var steps []*models.Step
	_, err := d.table.ListIn(ctx, txOrNil, &steps, pagination, stepsSelect)
	if err != nil {
		return nil, err
	}
	return steps, nil
}
