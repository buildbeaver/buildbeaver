package build

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type BuildService struct {
	db                   *store.DB
	authorizationService services.AuthorizationService
	buildStore           store.BuildStore
	repoStore            store.RepoStore
	ownershipStore       store.OwnershipStore
	resourceLinkStore    store.ResourceLinkStore
	identityStore        store.IdentityStore
	grantStore           store.GrantStore
	logger.Log
}

func NewBuildService(
	db *store.DB,
	authorizationService services.AuthorizationService,
	buildStore store.BuildStore,
	repoStore store.RepoStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	identityStore store.IdentityStore,
	grantStore store.GrantStore,
	logFactory logger.LogFactory,
) *BuildService {
	return &BuildService{
		db:                   db,
		authorizationService: authorizationService,
		buildStore:           buildStore,
		repoStore:            repoStore,
		ownershipStore:       ownershipStore,
		resourceLinkStore:    resourceLinkStore,
		identityStore:        identityStore,
		grantStore:           grantStore,
		Log:                  logFactory("BuildService"),
	}
}

// Create a new build.
// Returns store.ErrAlreadyExists if a build with matching unique properties already exists.
func (s *BuildService) Create(ctx context.Context, txOrNil *store.Tx, build *models.Build) error {
	err := build.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating build")
	}
	now := models.NewTime(time.Now())
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		buildNumber, err := s.repoStore.IncrementBuildCounter(ctx, tx, build.RepoID)
		if err != nil {
			return fmt.Errorf("error incrementing build counter: %w", err)
		}
		build.Name = models.ResourceName(buildNumber.String())
		err = s.buildStore.Create(ctx, tx, build)
		if err != nil {
			return fmt.Errorf("error creating build: %w", err)
		}
		ownership := models.NewOwnership(now, build.RepoID.ResourceID, build.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, build)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Created build %q", build.ID)
		return nil
	})
}

// Update an existing build with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *BuildService) Update(ctx context.Context, txOrNil *store.Tx, build *models.Build) error {
	err := build.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating build")
	}
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.buildStore.Update(ctx, tx, build)
		if err != nil {
			return fmt.Errorf("error updating build: %w", err)
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, build)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Updated build %q", build.ID)
		return nil
	})
}

// Read an existing build, looking it up by ResourceID.
// Returns models.ErrNotFound if the build does not exist.
func (s *BuildService) Read(ctx context.Context, txOrNil *store.Tx, id models.BuildID) (*models.Build, error) {
	return s.buildStore.Read(ctx, txOrNil, id)
}

// ReadByIdentityID looks up the build that corresponds to the specified identity, or returns a not found error
// if the identity doesn't correspond to a build.
func (s *BuildService) ReadByIdentityID(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) (*models.Build, error) {
	// Read identity and check it is owned by a Build
	identity, err := s.identityStore.Read(ctx, txOrNil, identityID)
	if err != nil {
		return nil, fmt.Errorf("error reading Identity for Build: %w", err)
	}
	if identity.OwnerResourceID.Kind() != models.BuildResourceKind {
		return nil, fmt.Errorf("error reading Build: Identity owner %s is not a build", identity.OwnerResourceID)
	}
	buildID := models.BuildIDFromResourceID(identity.OwnerResourceID)

	return s.Read(ctx, txOrNil, buildID)
}

// FindOrCreateIdentity returns an Identity that has permission to read and add jobs for a specific build only,
// for use by dynamic jobs running as part of that build.
// If no identity exists for the build then a new identity is created and returned.
func (s *BuildService) FindOrCreateIdentity(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) (*models.Identity, error) {
	// Read build to check it exists, and read its repo to determine the legal entity responsible
	build, err := s.Read(ctx, txOrNil, buildID)
	if err != nil {
		return nil, fmt.Errorf("error finding or creating identity for build: error reading build: %w", err)
	}
	repo, err := s.repoStore.Read(ctx, txOrNil, build.RepoID)
	if err != nil {
		return nil, fmt.Errorf("error finding or creating identity for build: error reading repo: %w", err)
	}

	identity, created, err := s.identityStore.FindOrCreateByOwnerResource(ctx, txOrNil, buildID.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("error finding or creating identity for build: %w", err)
	}

	if created {
		// Grant permissions for the identity, only for this build
		err = s.authorizationService.CreateGrantsForIdentity(
			ctx,
			txOrNil,
			repo.LegalEntityID, // granted by the legal entity that owns the repo
			identity.ID,
			[]*models.Operation{
				models.BuildReadOperation,
				models.JobReadOperation,
				models.ArtifactReadOperation,
				models.JobCreateOperation,
			},
			buildID.ResourceID,
		)
	}

	return identity, nil
}

// DeleteIdentity deletes any existing Identity associated with a build, and any associated access control grants.
func (s *BuildService) DeleteIdentity(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) error {
	// Look up the identity to allow deletion of all associated grants
	identity, err := s.identityStore.ReadByOwnerResource(ctx, txOrNil, buildID.ResourceID)
	if err != nil {
		if gerror.IsNotFound(err) {
			return nil // not an error; there is no identity
		} else {
			return fmt.Errorf("error trying to read identity for build: %w", err)
		}
	}
	err = s.authorizationService.DeleteAllGrantsForIdentity(ctx, txOrNil, identity.ID)
	if err != nil {
		return fmt.Errorf("error deleting grants for identity %s for build %s: %w", identity.ID, buildID, err)
	}

	// Delete the identity itself
	err = s.identityStore.DeleteByOwnerResource(ctx, txOrNil, buildID.ResourceID)
	if err != nil {
		return fmt.Errorf("error deleting identity for %s: %w", buildID, err)
	}
	return nil
}

// LockRowForUpdate takes out an exclusive row lock on the build table row for the specified build.
// This function must be called within a transaction, and will block other transactions from locking, updating
// or deleting the row until this transaction ends.
func (s *BuildService) LockRowForUpdate(ctx context.Context, tx *store.Tx, id models.BuildID) error {
	return s.buildStore.LockRowForUpdate(ctx, tx, id)
}

// Search all builds. If a searcher identity is provided then the search will be constrained to include only
// results that the identity has access to. Use cursor to page through results, if any.
func (s *BuildService) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search *models.BuildSearch) ([]*models.BuildSearchResult, *models.Cursor, error) {
	err := search.Validate()
	if err != nil {
		return nil, nil, fmt.Errorf("error validating search: %w", err)
	}
	return s.buildStore.Search(ctx, txOrNil, searcher, search)
}

// Summary returns a summary of builds for the given legalEntityId. If searcher is set, the results will be limited to build(s) the searcher is authorized to
// see (via the read:build permission).
func (s *BuildService) Summary(ctx context.Context, txOrNil *store.Tx, legalEntityID models.LegalEntityID, searcher models.IdentityID) (*models.BuildSummaryResult, error) {
	var buildSummaryResult = models.BuildSummaryResult{}

	summarySearch := models.NewBuildSearch()
	summarySearch.Limit = 10
	summarySearch.LegalEntityID = &legalEntityID

	err := s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Running builds
		summarySearch.IncludeStatuses = []models.WorkflowStatus{models.WorkflowStatusRunning}
		runningBuilds, _, err := s.Search(ctx, tx, searcher, summarySearch)
		if err != nil {
			return err
		}

		// Upcoming builds
		summarySearch.IncludeStatuses = []models.WorkflowStatus{models.WorkflowStatusSubmitted, models.WorkflowStatusQueued}
		upcomingBuilds, _, err := s.Search(ctx, tx, searcher, summarySearch)
		if err != nil {
			return err
		}

		// Completed builds
		summarySearch.IncludeStatuses = []models.WorkflowStatus{models.WorkflowStatusSucceeded, models.WorkflowStatusFailed, models.WorkflowStatusCanceled}
		completedBuilds, _, err := s.Search(ctx, tx, searcher, summarySearch)
		if err != nil {
			return err
		}

		buildSummaryResult.Running = runningBuilds
		buildSummaryResult.Upcoming = upcomingBuilds
		buildSummaryResult.Completed = completedBuilds

		return nil
	})

	return &buildSummaryResult, err
}

// UniversalSearch searches all builds. If searcher is set, the results will be limited to build(s) the searcher is authorized to
// see (via the read:build permission). Use cursor to page through results, if any.
func (s *BuildService) UniversalSearch(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search search.Query) ([]*models.BuildSearchResult, *models.Cursor, error) {
	return s.buildStore.UniversalSearch(ctx, txOrNil, searcher, search)
}
