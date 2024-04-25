package work_items

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.WorkItem{})
}

type WorkItemStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *WorkItemStore {
	return &WorkItemStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.WorkItem{}),
	}
}

// Create a new work item.
// Returns store.ErrAlreadyExists if a work item with this ID already exists.
func (d *WorkItemStore) Create(ctx context.Context, txOrNil *store.Tx, workItem *models.WorkItem) error {
	return d.table.Create(ctx, txOrNil, workItem)
}

// Read an existing work item, looking it up by ResourceID.
// Will return models.ErrNotFound if the work item does not exist.
func (d *WorkItemStore) Read(ctx context.Context, txOrNil *store.Tx, id models.WorkItemID) (*models.WorkItem, error) {
	workItem := &models.WorkItem{}
	return workItem, d.table.ReadByID(ctx, txOrNil, id.ResourceID, workItem)
}

// Update an existing work item with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *WorkItemStore) Update(ctx context.Context, txOrNil *store.Tx, workItem *models.WorkItem) error {
	return d.table.UpdateByID(ctx, txOrNil, workItem)
}

// Delete permanently and idempotently deletes a work item.
func (d *WorkItemStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.WorkItemID) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"work_item_id": id.ResourceID})
}
