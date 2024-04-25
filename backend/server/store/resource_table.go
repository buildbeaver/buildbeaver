package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

const upsertUpdateRetries = 5

var (
	resourceInterface              = reflect.TypeOf((*models.Resource)(nil)).Elem()
	softDeletableResourceInterface = reflect.TypeOf((*models.SoftDeletableResource)(nil)).Elem()
)

type queryBuilder interface {
	ToSQL() (string, []interface{}, error)
}

type resourceTableMarker struct {
	Id        models.ResourceID `json:"id"`
	CreatedAt models.Time       `json:"created_at"`
}

type tableDescriptor struct {
	tableName         string
	idColName         string
	generationColName string
	createdAtColName  string
	deletedAtColName  string
	isSoftDeletable   bool
	isMutable         bool
}

type ResourceTable struct {
	logger.Log
	tableDescriptor
	db *DB
}

func NewResourceTable(db *DB, logFactory logger.LogFactory, resource models.Resource) *ResourceTable {
	return NewResourceTableWithTableName(db, logFactory, "", resource)
}

func NewResourceTableWithTableName(db *DB, logFactory logger.LogFactory, tableName string, resource models.Resource) *ResourceTable {
	desc := mustTableDescriptor(resource, tableName)
	return &ResourceTable{
		db:              db,
		tableDescriptor: desc,
		Log:             logFactory(fmt.Sprintf("%s_table", desc.tableName)),
	}
}

// MustDBModel verifies a resource model matches our conventions and contains suitable "db" tags.
//   - Model must contain one or more "db" tags
//   - All "db" tags must have a common field prefix e.g artifact_ or build_ etc.
//   - There must be a prefix_id field e.g. artifact_id or build_id etc.
//   - If the model is a models.MutableResource it must have a prefix_etag field e.g. artifact_etag
//   - If the model is a models.SoftDeletableResource it must have a prefix_delete_at field e.g artifact_deleted_at
func MustDBModel(resource models.Resource) {
	mustTableDescriptor(resource, "")
}

// Dialect returns the goqu dialect (aka SQL Driver e.g. sqlite3, postgres etc.) in use.
// You will typically want to use this when
func (d *ResourceTable) Dialect() goqu.DialectWrapper {
	return goqu.Dialect(d.db.DriverName())
}

// ReadByID reads an existing resource, looking it up by ResourceID.
// Returns models.ErrNotFound if the resource does not exist.
//
// For resources which implement the models.SoftDeletableResource interface, the behaviour is as follows:
//
//  1. If the resource is soft-deleted and the model's IsUnreachable() method returns true then it will be
//     treated as not existing and ErrNotFound will be returned.
//
//  2. If the resource is soft-deleted and the model's IsUnreachable() method returns false then the resource
//     will still be returned even though it has been soft-deleted, i.e. it is still 'reachable'.
//
// 3. If the resource is not soft-deleted then it will be returned, regardless of what IsUnreachable() returns.
func (d *ResourceTable) ReadByID(ctx context.Context, txOrNil *Tx, id models.ResourceID, resource models.Resource) error {
	// Read the resource, regardless of whether it is soft-deleted
	where := goqu.Ex{d.idColName: id}
	err := d.ReadIn(ctx, txOrNil, resource, d.Dialect().From(d.tableName).Select(resource).Where(where))
	if err != nil {
		return err
	}

	// If the resource is soft-deleted and unreachable then return a not found error
	if softDeletableResource, ok := resource.(models.SoftDeletableResource); ok {
		if softDeletableResource.GetDeletedAt() != nil && softDeletableResource.IsUnreachable() {
			return gerror.NewErrNotFound("Not Found").Wrap(err)
		}
	}

	return nil // success
}

// ReadWhere reads an existing resource, looking it up using the supplied where clauses.
// If the resource is a models.SoftDeletableResource and it is deleted it will be treated as not existing, regardless
// of what the resource's IsUnreachable() method returns.
// Returns models.ErrNotFound if the resource does not exist.
func (d *ResourceTable) ReadWhere(ctx context.Context, txOrNil *Tx, resource models.Resource, where ...goqu.Expression) error {
	if _, ok := resource.(models.SoftDeletableResource); ok {
		where = append(where, goqu.Ex{d.deletedAtColName: nil})
	}
	return d.ReadIn(ctx, txOrNil, resource, d.Dialect().From(d.tableName).Select(resource).Where(where...))
}

// ReadAndLockRowForUpdateWhere reads an existing resource, looking it up using the supplied where clauses, and also
// locks the row using SELECT FOR UPDATE.
// This function must be called within a transaction, and will block other transactions from locking, updating
// or deleting the row until this transaction ends.
// If the resource is a models.SoftDeletableResource and it is deleted it will be treated as not existing, regardless
// of what the resource's IsUnreachable() method returns.
// Returns gerror.ErrNotFound if the resource does not exist.
func (d *ResourceTable) ReadAndLockRowForUpdateWhere(ctx context.Context, tx *Tx, resource models.Resource, where ...goqu.Expression) error {
	if tx == nil {
		return fmt.Errorf("error reading and locking database row for update: no transaction specified")
	}
	// If database doesn't support row locking then assume we have table locking by default and don't need row locking
	if !d.db.SupportsRowLevelLocking() {
		return d.ReadWhere(ctx, tx, resource, where...)
	}
	if _, ok := resource.(models.SoftDeletableResource); ok {
		where = append(where, goqu.Ex{d.deletedAtColName: nil})
	}
	ds := d.Dialect().From(d.tableName).Select(resource).Where(where...).ForUpdate(exp.Wait).Limit(1)
	return d.ReadIn(ctx, tx, resource, ds)
}

// ReadIn reads an existing resource from the supplied select dataset.
// The caller is responsible for filtering out soft-deleted resources if required, by adding suitable WHERE
// conditions to the query dataset (ds).
// Returns models.ErrNotFound if the resource does not exist.
func (d *ResourceTable) ReadIn(ctx context.Context, txOrNil *Tx, resource models.Resource, ds *goqu.SelectDataset) error {
	ds = ds.Limit(1)
	return d.db.Read2(txOrNil, func(db Reader) error {
		query, args, err := ds.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.LogQuery(query, args)
		found, err := db.ScanStructContext(ctx, resource, query, args...)
		if err != nil {
			return MakeStandardDBError(err)
		}
		if !found {
			return gerror.NewErrNotFound("Not Found")
		}
		return nil
	})
}

// LockRowForUpdate takes out an exclusive row lock on the row for the specified resource ID.
// This function must be called within a transaction, and will block other transactions from locking, updating
// or deleting the row until this transaction ends.
// Returns gerror.ErrNotFound if the resource does not exist.
func (d *ResourceTable) LockRowForUpdate(ctx context.Context, tx *Tx, id models.ResourceID) error {
	if tx == nil {
		return fmt.Errorf("error locking database row for resource %q: no transaction specified", id)
	}
	return d.LockRowForUpdateWhere(ctx, tx, goqu.Ex{d.idColName: id})
}

// LockRowForUpdateWhere takes out an exclusive row lock on the first row found in the resource table
// using the specified 'where' clause to locate the row.
// This function must be called within a transaction, and will block other transactions from locking, updating
// or deleting the row until this transaction ends.
// Returns gerror.ErrNotFound if the resource does not exist.
func (d *ResourceTable) LockRowForUpdateWhere(ctx context.Context, tx *Tx, where ...goqu.Expression) error {
	if tx == nil {
		return fmt.Errorf("error locking database row for update: no transaction specified")
	}
	// If database doesn't support row locking then assume we have table locking by default and don't need row locking
	if !d.db.SupportsRowLevelLocking() {
		return nil
	}
	if d.isSoftDeletable {
		where = append(where, goqu.Ex{d.deletedAtColName: nil})
	}

	return d.db.Read2(tx, func(db Reader) error {
		ds := d.Dialect().From(d.tableName).Select(goqu.C(d.idColName)).Where(where...).ForUpdate(exp.Wait).Limit(1)
		query, args, err := ds.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.LogQuery(query, args)

		var resultID string
		found, err := db.ScanValContext(ctx, &resultID, query, args...)
		if err != nil {
			return MakeStandardDBError(err)
		}
		if !found {
			return fmt.Errorf("error running SelectForUpdate query; no count returned")
		}
		if resultID == "" {
			return gerror.NewErrNotFound("Not Found").Wrap(err)
		}
		return nil
	})
}

// Create a new resource.
// Returns ErrAlreadyExists if a resource with matching unique properties already exists.
func (d *ResourceTable) Create(ctx context.Context, txOrNil *Tx, resource models.Resource) error {
	err := resource.Validate()
	if err != nil {
		return fmt.Errorf("error resource invalid: %w", err)
	}
	mutable, ok := resource.(models.MutableResource)
	if ok {
		hash, err := hashstructure.Hash(resource, hashstructure.FormatV2, nil)
		if err != nil {
			return fmt.Errorf("error calculating resource hash: %w", err)
		}
		mutable.SetETag(models.ETag(fmt.Sprintf("\"%x\"", hash)))
		defer func() {
			if err != nil {
				mutable.SetETag("")
			}
		}()
	}
	return d.db.Write2(txOrNil, func(db Writer) error {
		_, err := d.LogInsert(db.Insert(d.tableName).Rows(resource)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing create query: %w", MakeStandardDBError(err))
		}
		return nil
	})
}

// findOrCreateReadFn must return models.ErrNotFound if the resource does not exist
type findOrCreateReadFn func(ctx context.Context, txOrNil *Tx) (models.Resource, error)

// findOrCreateCreateFn must return ErrAlreadyExists if the resource already exists, and
// return the newly created resource on success
type findOrCreateCreateFn func(ctx context.Context, txOrNil *Tx) (models.Resource, error)

// FindOrCreate creates a resource if it does not exist, otherwise it reads and returns the existing resource.
// Returns the resource as it is in the database, and true iff the resource was created.
func (d *ResourceTable) FindOrCreate(
	ctx context.Context,
	txOrNil *Tx,
	readFn findOrCreateReadFn,
	createFn findOrCreateCreateFn,
) (resource models.Resource, created bool, err error) {
	resource, created, err = d.findOrCreateInner(ctx, txOrNil, readFn, createFn)
	if err != nil && gerror.ToAlreadyExists(err) != nil {
		// Try once to accommodate a racing create. We would expect the next time around we enter into
		// the 'find' path. We don't care to compensate for rapid creation/deletion of a resource.
		d.Infof("Conflicting create detected in findOrCreate - trying again once: %v", err)
		resource, created, err = d.findOrCreateInner(ctx, txOrNil, readFn, createFn)
	}
	return resource, created, err
}

// findOrCreateInner performs a find-or-create without any retries or compensating logic.
// It attempts to read a resource using readFn. If the resource exists then the data is returned,
// otherwise createFn is called and the newly created resource is returned.
func (d *ResourceTable) findOrCreateInner(
	ctx context.Context,
	txOrNil *Tx,
	readFn findOrCreateReadFn,
	createFn findOrCreateCreateFn,
) (resource models.Resource, created bool, err error) {
	// Try to read
	created = false
	resource, err = readFn(ctx, txOrNil)
	if err != nil {
		if gerror.ToNotFound(err) != nil {
			resource = nil // not found, so carry on to create
		} else {
			return nil, false, fmt.Errorf("error reading resource: %w", err)
		}
	}
	// If we didn't find a resource to read then create it
	if resource == nil {
		resource, err = createFn(ctx, txOrNil)
		if err != nil {
			return nil, false, fmt.Errorf("error creating resource: %w", err)
		}
		created = true
	}
	return resource, created, nil // either read or create succeeded
}

// DeleteByID idempotently deletes one resource by id.
func (d *ResourceTable) DeleteByID(ctx context.Context, txOrNil *Tx, id models.ResourceID) error {
	return d.DeleteWhere(ctx, txOrNil, goqu.Ex{d.idColName: id})
}

// DeleteWhere idempotently deletes one or more resources that match the supplied where clauses.
func (d *ResourceTable) DeleteWhere(ctx context.Context, txOrNil *Tx, where ...goqu.Expression) error {
	return d.db.Write2(txOrNil, func(db Writer) error {
		_, err := d.logDelete(db.Delete(d.tableName).Where(where...)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing delete query: %w", MakeStandardDBError(err))
		}
		return nil
	})
}

// SoftDelete an existing resource. Identifies the resource by id.
// Applies optimistic locking if the resource supports models.MutableResource.
// Returns ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *ResourceTable) SoftDelete(ctx context.Context, txOrNil *Tx, resource models.SoftDeletableResource) (err error) {
	origDeletedAt := resource.GetDeletedAt()
	newDeletedAt := models.NewTime(time.Now())
	resource.SetDeletedAt(&newDeletedAt)
	defer func() {
		if err != nil {
			resource.SetDeletedAt(origDeletedAt)
		}
	}()
	return d.updateWhere(ctx, txOrNil, resource, goqu.Ex{d.idColName: resource.GetID()})
}

// UpdateByID updates an existing resource. Identifies the resource by id. Overrides all previous values using the supplied model.
// Applies optimistic locking if the resource supports models.MutableResource.
// Returns ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *ResourceTable) UpdateByID(ctx context.Context, txOrNil *Tx, resource models.Resource) error {
	return d.updateWhere(ctx, txOrNil, resource, goqu.Ex{d.idColName: resource.GetID()})
}

// updateWhere updates an existing resource. Identifies the resource via where clauses. Overrides all previous values using the supplied model.
// Applies optimistic locking if the resource supports models.MutableResource.
// Returns ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *ResourceTable) updateWhere(ctx context.Context, txOrNil *Tx, resource models.Resource, where ...goqu.Expression) (err error) {
	err = resource.Validate()
	if err != nil {
		return fmt.Errorf("error resource invalid: %w", err)
	}
	_, ok := resource.(models.SoftDeletableResource)
	if ok {
		where = append(where, goqu.Ex{d.deletedAtColName: nil})
	}
	mutable, ok := resource.(models.MutableResource)
	if ok {
		origETag := mutable.GetETag()
		hash, err := hashstructure.Hash(resource, hashstructure.FormatV2, nil)
		if err != nil {
			return fmt.Errorf("error calculating resource hash: %w", err)
		}
		mutable.SetETag(models.ETag(fmt.Sprintf("\"%x\"", hash)))
		if origETag != models.ETagAny {
			where = append(where, goqu.Ex{d.generationColName: origETag})
		}
		defer func() {
			if err != nil {
				mutable.SetETag(origETag)
			}
		}()
	}
	return d.db.Write2(txOrNil, func(db Writer) error {
		res, err := d.LogUpdate(db.Update(d.tableName).Set(resource).Where(where...)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing update query: %w", MakeStandardDBError(err))
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error reading rows affected: %w", MakeStandardDBError(err))
		}
		if rowsAffected == 0 {
			if mutable == nil {
				return gerror.NewErrNotFound(fmt.Sprintf("%s does not exist", resource.GetID()))
			}
			return gerror.NewErrOptimisticLockFailed("ETag does not match")
		}
		return nil
	})
}

// upsertReadFn must return models.ErrNotFound if the resource does not exist
type upsertReadFn func(txOrNil *Tx) (models.Resource, error)

// UpsertCreateFn must return ErrAlreadyExists if the resource already exists
type upsertCreateFn func(txOrNil *Tx) error

// upsertUpdateFn inspects the resource returned by the upsertReadFn and updates it
// in the database if necessary. Returns true if the update was performed or false if
// no update was required. Must return ErrOptimisticLockFailed if the resource was updated
// in between read and update.
type upsertUpdateFn func(txOrNil *Tx, resource models.Resource) (bool, error)

// Upsert creates a resource if it does not exist, otherwise it updates its mutable properties
// if they differ from the in-memory instance. Returns true,false if the resource was created,
// false,true if the resource was updated, and false,false if neither create nor update was necessary.
func (d *ResourceTable) Upsert(ctx context.Context, txOrNil *Tx, readFn upsertReadFn, createFn upsertCreateFn, updateFn upsertUpdateFn) (created bool, updated bool, err error) {
	created, updated, err = d.upsertInner(ctx, txOrNil, readFn, createFn, updateFn)
	if err != nil && gerror.ToAlreadyExists(err) != nil {
		// Try once to accommodate a racing create. We would expect the next time
		// around we enter into the update path. We don't care to compensate for
		// rapid creation/deletion of a resource.
		d.Infof("Conflicting create detected in upsert - trying again once: %v", err)
		created, updated, err = d.upsertInner(ctx, txOrNil, readFn, createFn, updateFn)
	}
	for i := 0; i < upsertUpdateRetries && err != nil; i++ {
		// Try a limited number of times to accommodate racing updates. We generally
		// would expect to win on the second time around, as we don't really have any
		// contentious update code paths.
		if gerror.ToOptimisticLockFailed(err) != nil {
			d.Infof("Conflicting update detected in upsert - trying again (%d/%d attempts): %v", i+1, upsertUpdateRetries, err)
			created, updated, err = d.upsertInner(ctx, txOrNil, readFn, createFn, updateFn)
		} else {
			return false, false, fmt.Errorf("error upserting resource: %w", err)
		}
	}
	return created, updated, err
}

// upsertInner performs an upsert without any retries or compensating logic.
// It attempts to read a resource using readFn. If the resource exists then updateFn is called, otherwise createFn is called.
func (d *ResourceTable) upsertInner(ctx context.Context, txOrNil *Tx, readFn upsertReadFn, createFn upsertCreateFn, updateFn upsertUpdateFn) (created bool, updated bool, err error) {
	resource, err := readFn(txOrNil)
	if err != nil {
		if gerror.ToNotFound(err) != nil {
			err := createFn(txOrNil)
			if err != nil {
				return false, false, fmt.Errorf("error creating resource: %w", err)
			}
			return true, false, nil
		}
		return false, false, fmt.Errorf("error reading resource: %w", err)
	}
	updated, err = updateFn(txOrNil, resource)
	if err != nil {
		return false, false, fmt.Errorf("error updating resource: %w", err)
	}
	return false, updated, nil
}

// ListIn lists resources in the specified select dataset with pagination.
// Resources are listed in order of the newest creation date first (with ID being the tie-breaker; any ordering
// specified in the supplied Dataset is ignored.
// Resources must be a pointer to a slice of the resource type e.g. &[]*models.Artifact
func (d *ResourceTable) ListIn(ctx context.Context, txOrNil *Tx, resources interface{}, pagination models.Pagination, ds *goqu.SelectDataset) (*models.Cursor, error) {
	slicePtr := reflect.TypeOf(resources)
	if slicePtr.Kind() != reflect.Ptr {
		d.Panicf("expected pointer to slice, found: %T", resources)
	}
	sliceT := slicePtr.Elem()
	sliceV := reflect.ValueOf(resources).Elem()
	if sliceT.Kind() != reflect.Slice {
		d.Panicf("expected slice, found: %T", resources)
	}
	if !sliceT.Elem().Implements(resourceInterface) {
		d.Panicf("expected slice of resource, found: %s", sliceT.Elem().Kind())
	}
	if sliceT.Elem().Implements(softDeletableResourceInterface) {
		ds = ds.Where(goqu.Ex{d.deletedAtColName: nil})
	}

	err := d.db.Read2(txOrNil, func(db Reader) error {
		ds = ds.Limit(uint(pagination.Limit + 1))
		if pagination.Cursor == nil {
			ds = ds.Order(goqu.I(d.createdAtColName).Desc()).OrderAppend(goqu.I(d.idColName).Desc())
		} else {
			var decodedMarker resourceTableMarker
			err := json.Unmarshal([]byte(pagination.Cursor.Marker), &decodedMarker)
			if err != nil {
				return fmt.Errorf("error JSON decoding cusor marker: %w", err)
			}
			if pagination.Cursor.Direction == models.CursorDirectionPrev {
				// Create a query in the opposite (i.e. oldest first) order
				ds = ds.
					Where(goqu.C(d.createdAtColName).Gte(decodedMarker.CreatedAt)).
					Where(
						goqu.Or(
							goqu.And(
								goqu.C(d.createdAtColName).Eq(decodedMarker.CreatedAt),
								goqu.C(d.idColName).Gt(decodedMarker.Id),
							),
							goqu.C(d.createdAtColName).Gt(decodedMarker.CreatedAt),
						)).
					Order(goqu.I(d.createdAtColName).Asc()).OrderAppend(goqu.I(d.idColName).Asc())

				// Nest the reversed query in a descending-order query to make it correctly ordered,
				// while forcing evaluation of the entire query.
				// Note that column names mentioned here must exactly match the column name aliases defined
				// in the inner query. (e.g. "build_created_at" for an embedded build vs "builds.build_created_at"
				// for a build included by composition). Here we assume the primary resource type is embedded so
				// no table name is included in the aliased column name.
				// TODO: Find a way to avoid doing this, since it stops us from being able to compose the primary
				// TODO: resource table rather than embedding it, since the resulting column names are different
				ds = d.Dialect().From(ds).
					Select(goqu.I("*")).
					Order(goqu.C(d.createdAtColName).Desc()).
					OrderAppend(goqu.C(d.idColName).Desc())
			} else {
				ds = ds.
					Where(goqu.C(d.createdAtColName).Lte(decodedMarker.CreatedAt)).
					Where(
						goqu.Or(
							goqu.And(
								goqu.C(d.createdAtColName).Eq(decodedMarker.CreatedAt),
								goqu.C(d.idColName).Lt(decodedMarker.Id),
							),
							goqu.C(d.createdAtColName).Lt(decodedMarker.CreatedAt),
						)).
					Order(goqu.I(d.createdAtColName).Desc()).OrderAppend(goqu.I(d.idColName).Desc())
			}
		}
		query, args, err := ds.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.LogQuery(query, args)
		return db.ScanStructsContext(ctx, resources, query, args...)
	})
	if err != nil {
		return nil, MakeStandardDBError(err)
	}

	var cursor *models.Cursor
	if sliceV.Len() > 0 {
		cursor = &models.Cursor{}
		if pagination.Cursor != nil {
			if pagination.Cursor.Direction == models.CursorDirectionNext {
				resource := sliceV.Index(0).Interface().(models.Resource)
				resourceMarker := &resourceTableMarker{
					CreatedAt: resource.GetCreatedAt(),
					Id:        resource.GetID(),
				}
				data, err := json.Marshal(resourceMarker)
				if err != nil {
					return nil, fmt.Errorf("error JSON encoding marker cusor: %w", err)
				}
				cursor.Prev = &models.DirectionalCursor{
					Direction: models.CursorDirectionPrev,
					Marker:    string(data),
				}
			} else {
				resource := sliceV.Index(sliceV.Len() - 1).Interface().(models.Resource)
				resourceMarker := &resourceTableMarker{
					CreatedAt: resource.GetCreatedAt(),
					Id:        resource.GetID(),
				}
				data, err := json.Marshal(resourceMarker)
				if err != nil {
					return nil, fmt.Errorf("error JSON encoding marker cusor: %w", err)
				}
				cursor.Next = &models.DirectionalCursor{
					Direction: models.CursorDirectionNext,
					Marker:    string(data),
				}
			}
		}

		// If we read one more record than needed we know there is a next page
		if sliceV.Len() > pagination.Limit {
			if pagination.Cursor == nil || pagination.Cursor.Direction == models.CursorDirectionNext {
				sliceV.Set(sliceV.Slice(0, pagination.Limit))
				resource := sliceV.Index(pagination.Limit - 1).Interface().(models.Resource)
				resourceMarker := &resourceTableMarker{
					CreatedAt: resource.GetCreatedAt(),
					Id:        resource.GetID(),
				}
				data, err := json.Marshal(resourceMarker)
				if err != nil {
					return nil, fmt.Errorf("error JSON encoding marker cusor: %w", err)
				}
				cursor.Next = &models.DirectionalCursor{
					Direction: models.CursorDirectionNext,
					Marker:    string(data),
				}
			} else {
				sliceV.Set(sliceV.Slice(1, pagination.Limit+1))
				resource := sliceV.Index(0).Interface().(models.Resource)
				resourceMarker := &resourceTableMarker{
					CreatedAt: resource.GetCreatedAt(),
					Id:        resource.GetID(),
				}
				data, err := json.Marshal(resourceMarker)
				if err != nil {
					return nil, fmt.Errorf("error JSON encoding marker cusor: %w", err)
				}
				cursor.Prev = &models.DirectionalCursor{
					Direction: models.CursorDirectionPrev,
					Marker:    string(data),
				}
			}
		}
	}

	return cursor, nil
}

func MakeStandardDBError(err error) error {
	// TODO support other databases
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if sqliteErr.Code == sqlite3.ErrConstraint &&
			(sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey) {
			return gerror.NewErrAlreadyExists("Resource already exists").Wrap(sqliteErr)
		}
		if sqliteErr.Code == sqlite3.ErrNotFound {
			return gerror.NewErrNotFound("Resource not found").Wrap(sqliteErr)
		}
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		// 23505 -> unique_violation
		if pgErr.Code == "23505" {
			return gerror.NewErrAlreadyExists("Resource already exists").Wrap(pgErr)
		}
		// P0002 -> no_data_found
		// TODO: Check this matches above
		if pgErr.Code == "P0002" {
			return gerror.NewErrNotFound("Resource not found").Wrap(pgErr)
		}
	}
	return err
}

// LogSelect logs a select query via the configured logger.
func (d *ResourceTable) LogSelect(ds *goqu.SelectDataset) *goqu.SelectDataset {
	d.logQueryDS(ds)
	return ds
}

// LogInsert logs an insert query via the configured logger.
func (d *ResourceTable) LogInsert(ds *goqu.InsertDataset) *goqu.InsertDataset {
	d.logQueryDS(ds)
	return ds
}

// LogUpdate logs an update query via the configured logger.
func (d *ResourceTable) LogUpdate(ds *goqu.UpdateDataset) *goqu.UpdateDataset {
	d.logQueryDS(ds)
	return ds
}

// logDelete logs a delete query via the configured logger.
func (d *ResourceTable) logDelete(ds *goqu.DeleteDataset) *goqu.DeleteDataset {
	d.logQueryDS(ds)
	return ds
}

// logQuery generates and logs the raw SQL of a query to the configured logger.
func (d *ResourceTable) logQueryDS(ds queryBuilder) {
	query, args, err := ds.ToSQL()
	if err != nil {
		d.Errorf("Error generating query: %v", err)
		return
	}
	d.LogQuery(query, args)
}

// LogQuery logs a SQL query and args to the configured logger.
func (d *ResourceTable) LogQuery(query string, args []interface{}) {
	d.WithFields(logger.Fields{"query": query, "args": args}).Trace()
}

func (d *ResourceTable) TableName() string {
	return d.tableName
}

// mustTableDescriptor generates a table descriptor for a resource model. Panics if the model does not match our conventions.
// See MustDBModel for a description of the rules.
func mustTableDescriptor(resource models.Resource, tableNameOverride string) tableDescriptor {
	t := reflect.TypeOf(resource)
	fieldMap := make(map[string]struct{})
	collectDBTags(t, fieldMap)

	fieldPrefix := "" // e.g. artifact
	for val := range fieldMap {
		candidate := strings.TrimSuffix(val, idColSuffix) // in case there is only one field (assuming it's id, which is required)
		if fieldPrefix == "" {
			fieldPrefix = candidate
			continue
		}
		k := 0
		for ; k < min(len(candidate), len(fieldPrefix)); k++ {
			if candidate[k] != fieldPrefix[k] {
				k--
				break
			}
		}
		if k <= 0 {
			panic("All db fields must be prefixed with the table name")
		}
		fieldPrefix = candidate[:k]
	}

	if fieldPrefix == "" {
		panic("Unable to determine db field prefix")
	}

	expectedFieldExists := map[string]bool{
		makeIDColName(fieldPrefix): false, // e.g. artifact_id
	}
	_, isMutable := resource.(models.MutableResource)
	if isMutable {
		expectedFieldExists[makeETagColName(fieldPrefix)] = false // e.g. artifact_etag
	}
	_, isSoftDeletable := resource.(models.SoftDeletableResource)
	if isSoftDeletable {
		expectedFieldExists[makeDeletedAtFieldName(fieldPrefix)] = false // e.g. artifact_deleted_at
	}
	for val := range fieldMap {
		if _, ok := expectedFieldExists[val]; ok {
			expectedFieldExists[val] = true
		}
	}

	tableName := tableNameOverride
	if tableName == "" {
		tableName = fieldPrefix + "s" // e.g. artifacts
	}

	for field, exists := range expectedFieldExists {
		if !exists {
			panic(fmt.Sprintf("expected %q model to contain a field with a \"db\" tag matching %q", tableName, field))
		}
	}

	return tableDescriptor{
		tableName:         tableName,
		idColName:         makeIDColName(fieldPrefix),
		createdAtColName:  makeCreatedAtFieldName(fieldPrefix),
		deletedAtColName:  makeDeletedAtFieldName(fieldPrefix),
		generationColName: makeETagColName(fieldPrefix),
		isMutable:         isMutable,
		isSoftDeletable:   isSoftDeletable,
	}
}

// collectDBTags returns a map containing the db tag values of all fields in the flattened t.
func collectDBTags(t reflect.Type, fieldMap map[string]struct{}) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			collectDBTags(field.Type, fieldMap)
		} else {
			val, ok := field.Tag.Lookup(dbTagName)
			if ok {
				fieldMap[val] = struct{}{}
			}
		}
	}
}

const dbTagName = "db"

const idColSuffix = "_id"

func makeIDColName(fieldPrefix string) string {
	return fieldPrefix + idColSuffix
}

const eTagColSuffix = "_etag"

func makeETagColName(fieldPrefix string) string {
	return fieldPrefix + eTagColSuffix
}

const createdAtColSuffix = "_created_at"

func makeCreatedAtFieldName(fieldPrefix string) string {
	return fieldPrefix + createdAtColSuffix
}

const deletedAtColSuffix = "_deleted_at"

func makeDeletedAtFieldName(fieldPrefix string) string {
	return fieldPrefix + deletedAtColSuffix
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
