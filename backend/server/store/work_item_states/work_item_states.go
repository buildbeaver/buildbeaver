package work_item_states

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.WorkItemState{})
}

type WorkItemStateStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *WorkItemStateStore {
	return &WorkItemStateStore{
		db:    db,
		table: store.NewResourceTableWithTableName(db, logFactory, "work_item_states", &models.WorkItemState{}),
	}
}

// FindOrCreateAndLockRow create a new work item state record if one does not already exist with the same ID
// as the supplied WorkItemState object, otherwise reads and returns the existing record.
//
// A row lock is taken out on the returned record for the duration of the supplied transaction.
func (d *WorkItemStateStore) FindOrCreateAndLockRow(
	ctx context.Context,
	tx *store.Tx,
	state *models.WorkItemState,
) (*models.WorkItemState, error) {
	if tx == nil {
		return nil, fmt.Errorf("error: Transaction must be supplied to FindOrCreateAndLockRow")
	}

	resource, _, err := d.table.FindOrCreate(ctx, tx,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			// Read function - try to read and lock an existing state object with the supplied ID
			existingState := &models.WorkItemState{}
			whereClause := goqu.Ex{"work_item_state_id": state.ID}
			err := d.table.ReadAndLockRowForUpdateWhere(ctx, tx, existingState, whereClause)
			if err != nil {
				return nil, err
			}
			return existingState, nil
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			// Create function - create and then explicitly lock a new object
			err := d.table.Create(ctx, tx, state)
			if err != nil {
				return nil, fmt.Errorf("error attempting to create work item state record during FindOrCreate: %w", err)
			}
			err = d.LockRowForUpdate(ctx, tx, state.ID)
			if err != nil {
				// Failing here will retry, and we'll pick up the object on the read path next time
				return nil, fmt.Errorf("error attempting to lock new work item state record during FindOrCreate: %w", err)
			}
			return state, err
		},
	)
	if err != nil {
		return nil, err
	}

	return resource.(*models.WorkItemState), nil
}

// Read an existing work item state record, looking it up by ResourceID.
// Will return models.ErrNotFound if the work item does not exist.
func (d *WorkItemStateStore) Read(ctx context.Context, txOrNil *store.Tx, id models.WorkItemStateID) (*models.WorkItemState, error) {
	item := &models.WorkItemState{}
	return item, d.table.ReadByID(ctx, txOrNil, id.ResourceID, item)
}

// Update an existing work item state record with optimistic locking. Overrides all previous values using
// the supplied model. Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *WorkItemStateStore) Update(ctx context.Context, txOrNil *store.Tx, state *models.WorkItemState) error {
	return d.table.UpdateByID(ctx, txOrNil, state)
}

// LockRowForUpdate takes out an exclusive row lock on the database row for the specified work item state.
// This must be done within a transaction, and will block other transactions from locking, reading or updating
// the row until this transaction ends.
func (d *WorkItemStateStore) LockRowForUpdate(ctx context.Context, tx *store.Tx, id models.WorkItemStateID) error {
	return d.table.LockRowForUpdate(ctx, tx, id.ResourceID)
}

// Delete permanently and idempotently deletes a work item state record.
func (d *WorkItemStateStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.WorkItemStateID) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"work_item_state_id": id.ResourceID})
}

// CountWorkItems returns the number of work items associated with the specified work item state record.
// This will include any completed or failed work items which have not been deleted.
func (d *WorkItemStateStore) CountWorkItems(ctx context.Context, txOrNil *store.Tx, workItemStateID models.WorkItemStateID) (int, error) {
	workItemSelect := goqu.From(goqu.T("work_items")).
		Join(goqu.T("work_item_states"), goqu.On(goqu.Ex{"work_items.work_item_state": goqu.I("work_item_states.work_item_state_id")})).
		Select(goqu.COUNT(goqu.C("work_item_id"))).
		Where(goqu.Ex{"work_item_state_id": workItemStateID.ResourceID})

	var count int
	err := d.db.Read2(txOrNil, func(db store.Reader) error {
		query, args, err := workItemSelect.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.table.LogQuery(query, args)
		found, err := db.ScanValContext(ctx, &count, query, args...)
		if err == nil && !found {
			return gerror.NewErrNotFound("Count result not found")
		}
		return store.MakeStandardDBError(err)
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

// FindQueuedWorkItem reads the next queued work item that is ready to be allocated to a work item processor.
// A row lock is taken out on the work item state row for the returned work item, for the duration of the
// supplied transaction.
//
// A work item is logically a combination of a WorkItemRecord and a WorkItemState object, and both records
// are returned. The WorkItemState row in the table is locked, preventing any other caller from allocating
// a work item with the same concurrency key (which would share the same WorkItemState row).
//
// The now parameter is the current time, for comparison with time values in the database like 'allocated until'.
// Only work items of the types in the supplied list will be returned.
//
// Will return gerror.ErrNotFound if no suitable work item can be found.
func (d *WorkItemStateStore) FindQueuedWorkItem(
	ctx context.Context,
	tx *store.Tx,
	now models.Time,
	types []models.WorkItemType,
) (*models.WorkItemRecords, error) {
	if tx == nil {
		return nil, fmt.Errorf("error: Transaction must be supplied to FindQueuedWorkItem")
	}

	// Format the 'now' time in a form usable in SQL queries
	nowValue, err := now.Value()
	if err != nil {
		return nil, fmt.Errorf("error converting time to database value: %w", err)
	}

	// No point in running the query if no types were supplied
	if len(types) == 0 {
		return nil, gerror.NewErrNotFound("No work items found because no types supplied")
	}

	workItemSelect := goqu.From(goqu.T("work_items")).
		Join(goqu.T("work_item_states"), goqu.On(goqu.Ex{"work_items.work_item_state": goqu.I("work_item_states.work_item_state_id")})).
		Select(&models.WorkItemRecords{}).
		Where(goqu.And(
			// work item must be one of the supplied types
			goqu.C("work_item_type").In(types),
			// work item must either not already be allocated, or allocation has expired
			goqu.Or(
				goqu.C("work_item_state_allocated_to").IsNull(), // if not allocated then include
				goqu.And(
					goqu.C("work_item_state_allocated_to").IsNotNull(), // if allocated but expired then include
					goqu.C("work_item_state_allocated_until").Lt(nowValue),
				),
			),
			// work item must not be completed
			goqu.C("work_item_completed_at").IsNull(),
			// if there is a 'not before' then that time must already have elapsed
			goqu.Or(
				goqu.C("work_item_state_not_before").IsNull(),     // if no 'not before' time then include
				goqu.C("work_item_state_not_before").Lt(nowValue), // if 'not before' time has elapsed then include
			),
		)).
		Order(goqu.I("work_item_created_at").Asc()).
		Limit(1)
	if d.db.SupportsRowLevelLocking() {
		workItemSelect = workItemSelect.ForUpdate(exp.SkipLocked)
	}

	records := &models.WorkItemRecords{
		Record: &models.WorkItem{},
		State:  &models.WorkItemState{},
	}

	err = d.db.Read2(tx, func(db store.Reader) error {
		query, args, err := workItemSelect.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.table.LogQuery(query, args)
		found, err := db.ScanStructContext(ctx, records, query, args...)
		if !found {
			return gerror.NewErrNotFound("Not Found").Wrap(err)
		}
		return store.MakeStandardDBError(err)
	})
	if err != nil {
		return nil, err
	}

	return records, nil
}
