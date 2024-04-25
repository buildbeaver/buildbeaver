package pull_requests

import (
	"context"
	"fmt"
	"reflect"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.PullRequest{})
}

type PullRequestStore struct {
	db    *store.DB
	table *store.ResourceTable
	logger.Log
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *PullRequestStore {
	return &PullRequestStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.PullRequest{}),
		Log:   logFactory("PullRequestStore"),
	}
}

// Create a new pull request.
// Returns store.ErrAlreadyExists if a pull request with matching unique properties already exists.
func (d *PullRequestStore) Create(ctx context.Context, txOrNil *store.Tx, pullRequest *models.PullRequest) error {
	return d.table.Create(ctx, txOrNil, pullRequest)
}

// Read an existing pull request, looking it up by PullRequestID.
// Returns models.ErrNotFound if the PR does not exist.
func (d *PullRequestStore) Read(ctx context.Context, txOrNil *store.Tx, id models.PullRequestID) (*models.PullRequest, error) {
	pullRequest := &models.PullRequest{}
	return pullRequest, d.table.ReadByID(ctx, txOrNil, id.ResourceID, pullRequest)
}

// ReadByExternalID reads an existing pull request, looking it up by its external id.
// Returns models.ErrNotFound if the pull request does not exist.
func (d *PullRequestStore) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.PullRequest, error) {
	pullRequest := &models.PullRequest{}
	return pullRequest, d.table.ReadWhere(ctx, txOrNil, pullRequest, goqu.Ex{"pull_request_external_id": externalID})
}

// Update an existing pull request with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *PullRequestStore) Update(ctx context.Context, txOrNil *store.Tx, pullRequest *models.PullRequest) error {
	return d.table.UpdateByID(ctx, txOrNil, pullRequest)
}

// Upsert creates a pull request for a given External ID if it does not exist, otherwise it updates its
// mutable properties if they differ from the in-memory instance. Returns true,false if the resource was
// created and false,true if the resource was updated. false,false if neither a create nor update was necessary.
func (d *PullRequestStore) Upsert(ctx context.Context, txOrNil *store.Tx, pullRequest *models.PullRequest) (bool, bool, error) {
	if pullRequest.ExternalID == nil {
		return false, false, fmt.Errorf("error external id must be set to upsert")
	}
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.ReadByExternalID(ctx, tx, *pullRequest.ExternalID)
		}, func(tx *store.Tx) error {
			return d.Create(ctx, tx, pullRequest)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.PullRequest)
			// ensure immutable fields don't change
			pullRequest.ID = existing.ID
			pullRequest.CreatedAt = existing.CreatedAt
			pullRequest.RepoID = existing.RepoID
			pullRequest.UserID = existing.UserID
			pullRequest.BaseRef = existing.BaseRef
			if reflect.DeepEqual(existing, pullRequest) {
				return false, nil
			}
			return true, d.Update(ctx, tx, pullRequest)
		})
}
