package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type RepoService struct {
	db                *store.DB
	ownershipStore    store.OwnershipStore
	repoStore         store.RepoStore
	resourceLinkStore store.ResourceLinkStore
	scmRegistry       *scm.SCMRegistry
	keyPairService    services.KeyPairService
	secretService     services.SecretService
	logger.Log
}

func NewRepoService(
	db *store.DB,
	ownershipStore store.OwnershipStore,
	repoStore store.RepoStore,
	resourceLinkStore store.ResourceLinkStore,
	scmRegistry *scm.SCMRegistry,
	keyPairService services.KeyPairService,
	secretService services.SecretService,
	logFactory logger.LogFactory) *RepoService {

	return &RepoService{
		db:                db,
		ownershipStore:    ownershipStore,
		repoStore:         repoStore,
		resourceLinkStore: resourceLinkStore,
		scmRegistry:       scmRegistry,
		keyPairService:    keyPairService,
		secretService:     secretService,
		Log:               logFactory("RepoService"),
	}
}

// Read an existing repo, looking it up by ID.
// Returns models.ErrNotFound if the repo does not exist.
func (s *RepoService) Read(ctx context.Context, txOrNil *store.Tx, id models.RepoID) (*models.Repo, error) {
	return s.repoStore.Read(ctx, txOrNil, id)
}

// ReadByExternalID reads an existing repo, looking it up by its external id.
// Returns models.ErrNotFound if the repo does not exist.
func (s *RepoService) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.Repo, error) {
	return s.repoStore.ReadByExternalID(ctx, txOrNil, externalID)
}

// Upsert creates a repo if it does not exist, otherwise it updates its mutable properties
// if they differ from the in-memory instance. Returns true,false if the resource was created
// and false,true if the resource was updated. false,false if neither a create or update was necessary.
// Repo Metadata and selected fields will not be updated (including Enabled and SSHKeySecretID fields).
func (s *RepoService) Upsert(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) (created bool, updated bool, err error) {
	err = repo.Validate()
	if err != nil {
		return false, false, errors.Wrap(err, "error validating repo")
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		created, updated, err = s.repoStore.Upsert(ctx, tx, repo)
		if err != nil {
			return fmt.Errorf("error upserting repo: %w", err)
		}
		ownership := models.NewOwnership(models.NewTime(time.Now()), repo.LegalEntityID.ResourceID, repo.GetID())
		_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
		if err != nil {
			return fmt.Errorf("error upserting ownership: %w", err)
		}
		if created || updated {
			_, _, err = s.resourceLinkStore.Upsert(ctx, tx, repo)
			if err != nil {
				return fmt.Errorf("error upserting resource link: %w", err)
			}
		}
		if created {
			err := s.repoStore.InitializeBuildCounter(ctx, tx, repo.ID)
			if err != nil {
				return fmt.Errorf("error initializing repo build counter: %w", err)
			}
			s.Infof("Created repo %q", repo.ID)
		}
		return nil
	})
	return created, updated, err
}

// UpdateRepoEnabled enables or disables builds for a repo.
func (s *RepoService) UpdateRepoEnabled(ctx context.Context, repoID models.RepoID, update dto.UpdateRepoEnabled) (*models.Repo, error) {
	repo, err := s.repoStore.Read(ctx, nil, repoID)
	if err != nil {
		return nil, fmt.Errorf("error reading repo: %w", err)
	}
	repo.ETag = models.GetETag(repo, update.ETag)
	if update.Enabled {
		return s.enableRepo(ctx, repo)
	} else {
		return s.disableRepo(ctx, repo)
	}
}

// enableRepo enables builds for a repo.
func (s *RepoService) enableRepo(ctx context.Context, repo *models.Repo) (*models.Repo, error) {
	scm, err := s.scmRegistry.Get(repo.ExternalID.ExternalSystem)
	if err != nil {
		return nil, fmt.Errorf("error getting SCM from registry: %w", err)
	}
	publicKeyPlaintext, privateKeyPlaintext, err := s.keyPairService.MakeSSHKeyPair()
	if err != nil {
		return nil, fmt.Errorf("error generating repo private key: %w", err)
	}

	// Disable the repo first, to remove any existing SSH key from the SCM and the database
	err = scm.DisableRepo(ctx, repo)
	if err != nil {
		s.Warnf("Will ignore error disabling repo prior to enabling it: %v", err)
	}

	// Set up the SSH key on the SCM before updating the database, as this is most likely to fail
	err = scm.EnableRepo(ctx, repo, publicKeyPlaintext)
	if err != nil {
		return nil, fmt.Errorf("error enabling repo: %w", err)
	}

	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		// Record the repo's SSH key (aka the 'deploy key') as a secret in the database
		sshKeySecretPlaintext, err := s.secretService.Create(ctx, tx, repo.ID, models.RepoSSHKeySecretName, string(privateKeyPlaintext), true)
		if err != nil {
			return fmt.Errorf("error creating repo SSH key secret: %w", err)
		}
		repo.Enabled = true
		repo.SSHKeySecretID = &sshKeySecretPlaintext.ID
		repo.UpdatedAt = models.NewTime(time.Now())
		err = s.repoStore.Update(ctx, tx, repo)
		if err != nil {
			return fmt.Errorf("error updating repo: %w", err)
		}
		return nil
	})
	if err != nil {
		// After failing to record the SSH key in the database, disable the repo again on the SCM to get back
		// to a consistent state (i.e. disabled)
		s.Errorf("error updating database after enabling repo %s; disabling repo again (error: %s)", repo.ID, err)
		var disableErr error
		repo, disableErr = s.disableRepo(ctx, repo) // do not re-declare repo
		if disableErr != nil {
			s.Errorf("Error disabling repo again after failing to enable: %w", disableErr)
		}
		// Return the original error that happened during enabling
		return nil, err
	}

	// Attempt to kick off a build after enabling the repo
	if repo.ExternalID != nil {
		err = scm.BuildRepoLatestCommit(ctx, repo, "") // use default branch (main/master)
		if err != nil {
			// Log and ignore errors
			s.Errorf("error attempting to queue a build for newly enabled repo '%s' on SCM %s: %s",
				repo.GetName(), scm.Name(), err.Error())
			err = nil
		}
	}

	return repo, nil
}

// disableRepo disables all future builds for a repo.
func (s *RepoService) disableRepo(ctx context.Context, repo *models.Repo) (*models.Repo, error) {
	scm, err := s.scmRegistry.Get(repo.ExternalID.ExternalSystem)
	if err != nil {
		s.Warnf("Will ignore error getting SCM from registry: %v", err)
	} else {
		err = scm.DisableRepo(ctx, repo)
		if err != nil {
			s.Warnf("Will ignore error disabling repo on SCM: %v", err)
		}
	}

	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		oldSecretID := repo.SSHKeySecretID
		// Update the repo first; mark as disabled and remove the reference to the SSH key secret
		repo.Enabled = false
		repo.SSHKeySecretID = nil
		repo.UpdatedAt = models.NewTime(time.Now())
		err = s.repoStore.Update(ctx, tx, repo)
		if err != nil {
			return fmt.Errorf("error updating repo in database after disabling repo on SCM: %w", err)
		}
		if oldSecretID != nil {
			err = s.secretService.Delete(ctx, tx, *oldSecretID)
			if err != nil {
				return fmt.Errorf("error deleting repo SSH key secret: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// SoftDelete soft deletes an existing repo.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch, i.e. if the repo has changed in
// the database since the supplied object was read.
func (s *RepoService) SoftDelete(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Remove the secret from the repo before we soft-delete
		oldSecretID := repo.SSHKeySecretID
		repo.SSHKeySecretID = nil
		err := s.repoStore.Update(ctx, tx, repo)
		if err != nil {
			return fmt.Errorf("error updating repo in database to remove SSH key before soft deleting: %w", err)
		}
		if oldSecretID != nil {
			err = s.secretService.Delete(ctx, tx, *oldSecretID)
			if err != nil {
				return fmt.Errorf("error deleting repo SSH key secret: %w", err)
			}
		}

		err = s.repoStore.SoftDelete(ctx, tx, repo)
		if err != nil {
			return fmt.Errorf("error soft deleting repo: %w", err)
		}

		// NOTE: Ownership is not deleted during soft deletes

		err = s.resourceLinkStore.Delete(ctx, tx, repo.GetID())
		if err != nil {
			return fmt.Errorf("error deleting resource link: %w", err)
		}

		s.Infof("Soft deleted repo %q", repo.ID)
		return nil
	})
}

// Search all repos. If searcher is set, the results will be limited to repos the searcher is authorized to
// see (via the read:repo permission). Use cursor to page through results, if any.
func (s *RepoService) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, query search.Query) ([]*models.Repo, *models.Cursor, error) {
	err := query.Validate()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error validating query")
	}
	return s.repoStore.Search(ctx, txOrNil, searcher, query)
}

// AllocateBuildNumber increments and returns the build counter for the specified repo. The returned counter
// is safe to assign to a build and will never be returned again by this function.
func (s *RepoService) AllocateBuildNumber(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID) (models.BuildNumber, error) {
	return s.repoStore.IncrementBuildCounter(ctx, txOrNil, repoID)
}
