package logs

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
)

func init() {
	_ = models.MutableResource(&models.LogDescriptor{})
	store.MustDBModel(&models.LogDescriptor{})
}

type LogStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *LogStore {
	return &LogStore{
		table: store.NewResourceTable(db, logFactory, &models.LogDescriptor{}),
	}
}

// Create a new logs.
// Returns store.ErrAlreadyExists if a logs with matching unique properties already exists.
func (d *LogStore) Create(ctx context.Context, txOrNil *store.Tx, log *models.LogDescriptor) error {
	return d.table.Create(ctx, txOrNil, log)
}

// Read an existing logs, looking it up by ID.
// Returns models.ErrNotFound if the logs does not exist.
func (d *LogStore) Read(ctx context.Context, txOrNil *store.Tx, id models.LogDescriptorID) (*models.LogDescriptor, error) {
	log := &models.LogDescriptor{}
	return log, d.table.ReadByID(ctx, txOrNil, id.ResourceID, log)
}

// Update an existing logs.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *LogStore) Update(ctx context.Context, txOrNil *store.Tx, log *models.LogDescriptor) error {
	return d.table.UpdateByID(ctx, txOrNil, log)
}

// Delete permanently and idempotently deletes a log.
func (d *LogStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.LogDescriptorID) error {
	return d.table.DeleteByID(ctx, txOrNil, id.ResourceID)
}

// Search all log descriptors. If searcher is set, the results will be limited to log descriptors the searcher
// is authorized to see (via the read:build permission). Use cursor to page through results, if any.
func (d *LogStore) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.LogDescriptorSearch) ([]*models.LogDescriptor, *models.Cursor, error) {
	logSelect := goqu.From(d.table.TableName()).Select(&models.LogDescriptor{})
	if !searcher.IsZero() {
		logSelect = authorizations.WithIsAuthorizedListFilter(logSelect, searcher, *models.BuildReadOperation, "log_descriptor_id")
	}
	if search.ParentLogID != nil {
		logSelect = d.table.Dialect().From("children")
		if !searcher.IsZero() {
			logSelect = authorizations.WithIsAuthorizedListFilter(logSelect, searcher, *models.BuildReadOperation, "log_descriptor_id")
		}
		logSelect = logSelect.WithRecursive("children",
			d.table.Dialect().From(d.table.TableName()).
				Select(&models.LogDescriptor{}).
				Where(goqu.Ex{"log_descriptor_id": *search.ParentLogID}).
				UnionAll(d.table.Dialect().From(goqu.T(d.table.TableName()).As("parent")).
					Select(
						goqu.C("log_descriptor_created_at").Table("parent"),
						goqu.C("log_descriptor_etag").Table("parent"),
						goqu.C("log_descriptor_id").Table("parent"),
						goqu.C("log_descriptor_parent_log_id").Table("parent"),
						goqu.C("log_descriptor_resource_id").Table("parent"),
						goqu.C("log_descriptor_sealed").Table("parent"),
						goqu.C("log_descriptor_size_bytes").Table("parent"),
						goqu.C("log_descriptor_updated_at").Table("parent")).
					Join(goqu.T("children").As("child"),
						goqu.On(goqu.Ex{"parent.log_descriptor_parent_log_id": goqu.I("child.log_descriptor_id")})))).
			Select(goqu.Star())
	}
	var logs []*models.LogDescriptor
	cursor, err := d.table.ListIn(ctx, txOrNil, &logs, search.Pagination, logSelect)
	if err != nil {
		return nil, nil, err
	}
	return logs, cursor, nil
}
