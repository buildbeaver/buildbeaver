package pull_request

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type PullRequestService struct {
	db               *store.DB
	pullRequestStore store.PullRequestStore
	ownershipStore   store.OwnershipStore
	logger.Log
}

func NewPullRequestService(
	db *store.DB,
	pullRequestStore store.PullRequestStore,
	ownershipStore store.OwnershipStore,
	logFactory logger.LogFactory,
) *PullRequestService {
	return &PullRequestService{
		db:               db,
		pullRequestStore: pullRequestStore,
		ownershipStore:   ownershipStore,
		Log:              logFactory("PullRequestService"),
	}
}

// Read an existing pull request, looking it up by ID.
// A models.ErrNotFound error is returned if the pull request does not exist.
func (s *PullRequestService) Read(ctx context.Context, txOrNil *store.Tx, id models.PullRequestID) (*models.PullRequest, error) {
	return s.pullRequestStore.Read(ctx, txOrNil, id)
}

// Upsert creates a pull request for a given ExternalID if it does not exist, otherwise it updates its
// mutable properties if they differ from the in-memory instance. Returns true,false if the resource was
// created and false,true if the resource was updated. false,false if neither a create nor update was necessary.
func (s *PullRequestService) Upsert(ctx context.Context, txOrNil *store.Tx, pullRequest *models.PullRequest) (created bool, updated bool, err error) {
	err = pullRequest.Validate()
	if err != nil {
		return false, false, errors.Wrap(err, "error validating pull request")
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		created, updated, err = s.pullRequestStore.Upsert(ctx, tx, pullRequest)
		if err != nil {
			return fmt.Errorf("error upserting pull request: %w", err)
		}
		// Pull request is owned by the Repo it is requesting a change to
		ownership := models.NewOwnership(models.NewTime(time.Now()), pullRequest.RepoID.ResourceID, pullRequest.GetID())
		_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
		if err != nil {
			return fmt.Errorf("error upserting ownership: %w", err)
		}
		if created {
			s.Infof("Created pull request %q", pullRequest.ID)
		}
		return nil
	})
	return created, updated, err
}
