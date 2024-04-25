package commits

import (
	"context"
	"reflect"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.Commit{})
}

type CommitStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *CommitStore {
	return &CommitStore{
		table: store.NewResourceTable(db, logFactory, &models.Commit{}),
	}
}

// Create a new commit.
// Returns store.ErrAlreadyExists if a commit with matching unique properties already exists.
func (d *CommitStore) Create(ctx context.Context, txOrNil *store.Tx, commit *models.Commit) error {
	return d.table.Create(ctx, txOrNil, commit)
}

// Read an existing commit, looking it up by ResourceID.
// Returns models.ErrNotFound if the commit does not exist.
func (d *CommitStore) Read(ctx context.Context, txOrNil *store.Tx, id models.CommitID) (*models.Commit, error) {
	commit := &models.Commit{}
	return commit, d.table.ReadByID(ctx, txOrNil, id.ResourceID, commit)
}

// ReadBySHA reads an existing commit, looking it up by its repo and SHA hash.
// Returns models.ErrNotFound if the commit does not exist.
func (d *CommitStore) ReadBySHA(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID, sha string) (*models.Commit, error) {
	commit := &models.Commit{}
	return commit, d.table.ReadWhere(ctx, txOrNil, commit,
		goqu.Ex{
			"commit_repo_id": repoID,
			"commit_sha":     sha,
		})
}

// Update an existing commit with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *CommitStore) Update(ctx context.Context, txOrNil *store.Tx, commit *models.Commit) error {
	return d.table.UpdateByID(ctx, txOrNil, commit)
}

// LockRowForUpdate takes out an exclusive row lock on the commit table row for the specified commit.
// This must be done within a transaction, and will block other transactions from locking, reading or updating
// the row until this transaction ends.
func (d *CommitStore) LockRowForUpdate(ctx context.Context, tx *store.Tx, id models.CommitID) error {
	return d.table.LockRowForUpdate(ctx, tx, id.ResourceID)
}

// Upsert creates a Commit for a given SHA if it does not exist, otherwise it updates any mutable properties,
// but only if that property has not previously been set. Commits are conceptually immutable, but it is valid
// to provide some fields initially, then add more fields later with an Upsert.
// Returns (created, updated), i.e. true,false if the resource was created, false,true if the resource was updated,
// or false,false if neither a create or update was necessary.
func (d *CommitStore) Upsert(ctx context.Context, txOrNil *store.Tx, commit *models.Commit) (bool, bool, error) {
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.ReadBySHA(ctx, tx, commit.RepoID, commit.SHA)
		}, func(tx *store.Tx) error {
			return d.Create(ctx, tx, commit)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.Commit)
			// Most fields in a commit are immutable; ensure they don't change
			commit.ID = existing.ID
			commit.CreatedAt = existing.CreatedAt
			commit.RepoID = existing.RepoID
			commit.SHA = existing.SHA
			commit.Message = existing.Message
			commit.AuthorName = existing.AuthorName
			commit.AuthorEmail = existing.AuthorEmail
			commit.CommitterName = existing.CommitterName
			commit.CommitterEmail = existing.CommitterEmail
			commit.Link = existing.Link
			// Allow previously missing mutable fields to have a value provided, but then they become immutable
			if existing.AuthorID.Valid() {
				commit.AuthorID = existing.AuthorID
			}
			if existing.CommitterID.Valid() {
				commit.CommitterID = existing.CommitterID
			}
			if existing.Config != nil {
				// If we have a config then both Config and ConfigType become immutable
				commit.Config = existing.Config
				commit.ConfigType = existing.ConfigType
			}
			if reflect.DeepEqual(existing, commit) {
				return false, nil
			}
			return true, d.Update(ctx, tx, commit)
		})
}
