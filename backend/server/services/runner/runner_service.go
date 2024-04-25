package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type RunnerService struct {
	db                *store.DB
	credentialService services.CredentialService
	groupService      services.GroupService
	runnerStore       store.RunnerStore
	ownershipStore    store.OwnershipStore
	resourceLinkStore store.ResourceLinkStore
	identityStore     store.IdentityStore
	logger.Log
}

func NewRunnerService(
	db *store.DB,
	credentialService services.CredentialService,
	groupService services.GroupService,
	runnerStore store.RunnerStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	identityStore store.IdentityStore,
	logFactory logger.LogFactory) *RunnerService {

	return &RunnerService{
		db:                db,
		credentialService: credentialService,
		groupService:      groupService,
		runnerStore:       runnerStore,
		ownershipStore:    ownershipStore,
		resourceLinkStore: resourceLinkStore,
		identityStore:     identityStore,
		Log:               logFactory("RunnerService"),
	}
}

// Read an existing runner, looking it up by ID.
// Returns models.ErrNotFound if the runner does not exist.
func (s *RunnerService) Read(ctx context.Context, txOrNil *store.Tx, id models.RunnerID) (*models.Runner, error) {
	return s.runnerStore.Read(ctx, txOrNil, id)
}

// ReadByName reads an existing runner, looking it up by name and the ID of the legal entity that owns the runner.
// Returns models.ErrNotFound if the runner is not found.
func (s *RunnerService) ReadByName(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityID models.LegalEntityID,
	name models.ResourceName,
) (*models.Runner, error) {
	return s.runnerStore.ReadByName(ctx, txOrNil, legalEntityID, name)
}

// ReadByIdentityID reads an existing runner, looking it up by the ID of its associated Identity.
func (s *RunnerService) ReadByIdentityID(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) (*models.Runner, error) {
	// Read identity and check it is owned by a Runner
	identity, err := s.identityStore.Read(ctx, txOrNil, identityID)
	if err != nil {
		return nil, fmt.Errorf("error reading Identity for Runner: %w", err)
	}
	if identity.OwnerResourceID.Kind() != models.RunnerResourceKind {
		return nil, fmt.Errorf("error reading Runner: Identity owner %s is not a runner", identity.OwnerResourceID)
	}
	runnerID := models.RunnerIDFromResourceID(identity.OwnerResourceID)

	return s.Read(ctx, txOrNil, runnerID)
}

// ReadIdentity reads and returns the Identity for the specified runner.
func (s *RunnerService) ReadIdentity(ctx context.Context, txOrNil *store.Tx, id models.RunnerID) (*models.Identity, error) {
	return s.identityStore.ReadByOwnerResource(ctx, txOrNil, id.ResourceID)
}

// Create a new runner. clientCert is an optional ASN.1 DER-encoded X.509 client certificate; if provided then
// a client certificate credential will be created for authentication using TLS mutual authentication.
// Returns store.ErrAlreadyExists if a runner with matching unique properties already exists.
func (s *RunnerService) Create(ctx context.Context, txOrNil *store.Tx, runner *models.Runner, clientCert []byte) error {
	now := models.NewTime(time.Now())
	s.configureDefaultLabels(runner)
	err := runner.Validate()
	if err != nil {
		return fmt.Errorf("error validating runner: %w", err)
	}
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Create the runner
		err := s.runnerStore.Create(ctx, tx, runner)
		if err != nil {
			return fmt.Errorf("error creating runner: %w", err)
		}
		for _, label := range runner.Labels {
			err := s.runnerStore.CreateLabel(ctx, tx, runner.ID, label)
			if err != nil {
				return fmt.Errorf("error creating runner label: %w", err)
			}
		}
		for _, jobType := range runner.SupportedJobTypes {
			err := s.runnerStore.CreateSupportedJobType(ctx, tx, runner.ID, jobType)
			if err != nil {
				return fmt.Errorf("error creating supported job type: %w", err)
			}
		}
		// Create ownership record - runner is underneath a legal entity in the hierarchy
		ownership := models.NewOwnership(now, runner.LegalEntityID.ResourceID, runner.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return errors.Wrap(err, "error creating ownership")
		}
		// Ensure the runner has a resource link matching its name
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, runner)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		// Create an Identity for the Runner, owned by the Runner
		identity := models.NewIdentity(now, runner.ID.ResourceID)
		err = s.identityStore.Create(ctx, tx, identity)
		if err != nil {
			return fmt.Errorf("error creating an identity for new runner: %w", err)
		}
		identityOwnership := models.NewOwnership(now, runner.ID.ResourceID, identity.ID.ResourceID)
		err = s.ownershipStore.Create(ctx, tx, identityOwnership)
		if err != nil {
			return fmt.Errorf("error creating identity ownership for new runner: %w", err)
		}
		// Add the runner to the 'runner' standard group, so it picks up suitable permissions
		err = s.addRunnerToRunnerStandardGroup(ctx, tx, runner, identity)
		if err != nil {
			return err
		}
		// If a client certificate was provided then create a credential for the runner to authenticate
		if clientCert != nil {
			_, err = s.credentialService.CreateClientCertificateCredential(ctx, tx, identity.ID, true, clientCert)
			if err != nil {
				return fmt.Errorf("error creating client certificate credential: %w", err)
			}
		}
		s.Infof("Created runner %q", runner.ID)
		return nil
	})
}

// addRunnerToRunnerStandardGroup will add the specified runner to the runners access control group for its parent
// legal entity, to give it access rights required to dequeue and run builds for that legal entity.
func (s *RunnerService) addRunnerToRunnerStandardGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	runner *models.Runner,
	runnerIdentity *models.Identity,
) error {
	// Find the 'runner' standard group for the legal entity that owns this runner
	ownerLegalEntityID := runner.LegalEntityID
	group, err := s.groupService.ReadByName(ctx, txOrNil, ownerLegalEntityID, models.RunnerStandardGroup.Name)
	if err != nil {
		return fmt.Errorf("error reading standard group for runner: %w", err)
	}

	s.Infof("Attempting to add runner %s to group '%s' for legal entity %s (GroupID %s)",
		runner.ID, group.Name, ownerLegalEntityID, group.ID)
	addedBy := runner.LegalEntityID // record that the runner was added to this group by the owning legal entity
	_, _, err = s.groupService.FindOrCreateMembership(ctx, txOrNil, models.NewGroupMembershipData(
		group.ID, runnerIdentity.ID, models.BuildBeaverSystem, addedBy))
	if err != nil {
		return fmt.Errorf("error adding %s to group '%s' for legal entity %s: %w",
			runner.ID, group.Name, ownerLegalEntityID, err)
	}

	return nil
}

// Update an existing runner
func (s *RunnerService) Update(ctx context.Context, txOrNil *store.Tx, runner *models.Runner) (*models.Runner, error) {
	s.configureDefaultLabels(runner)
	err := runner.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating runner")
	}
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err := s.runnerStore.LockRowForUpdate(ctx, tx, runner.ID)
		if err != nil {
			return fmt.Errorf("error locking runner: %w", err)
		}
		// Read the existing runner after locking it but before updating it, so
		// that we can work out which old labels to remove below
		existing, err := s.runnerStore.Read(ctx, tx, runner.ID)
		if err != nil {
			return fmt.Errorf("error reading runner: %w", err)
		}
		err = s.runnerStore.Update(ctx, tx, runner)
		if err != nil {
			return fmt.Errorf("error updating runner: %w", err)
		}
		toCreateLabels, toDeleteLabels := s.splitLabels(existing.Labels, runner.Labels)
		for _, label := range toDeleteLabels {
			err := s.runnerStore.DeleteLabel(ctx, tx, runner.ID, label)
			if err != nil {
				return fmt.Errorf("error deleting runner label: %w", err)
			}
		}
		for _, label := range toCreateLabels {
			err := s.runnerStore.CreateLabel(ctx, tx, runner.ID, label)
			if err != nil {
				return fmt.Errorf("error creating runner label: %w", err)
			}
		}
		toCreateTypes, toDeleteTypes := s.splitTypes(existing.SupportedJobTypes, runner.SupportedJobTypes)
		for _, jobType := range toDeleteTypes {
			err := s.runnerStore.DeleteSupportedJobType(ctx, tx, runner.ID, jobType)
			if err != nil {
				return fmt.Errorf("error deleting runner supported job type: %w", err)
			}
		}
		for _, kind := range toCreateTypes {
			err := s.runnerStore.CreateSupportedJobType(ctx, tx, runner.ID, kind)
			if err != nil {
				return fmt.Errorf("error creating runner supported job type: %w", err)
			}
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, runner)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Updated runner %q", runner.ID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return runner, nil
}

// RunnerCompatibleWithJob returns true if a runner exists that is capable of running job.
func (s *RunnerService) RunnerCompatibleWithJob(ctx context.Context, txOrNil *store.Tx, job *models.Job) (bool, error) {
	return s.runnerStore.RunnerCompatibleWithJob(ctx, txOrNil, job)
}

// SoftDelete an existing runner.
func (s *RunnerService) SoftDelete(ctx context.Context, txOrNil *store.Tx, runnerID models.RunnerID, delete dto.DeleteRunner) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		runner, err := s.runnerStore.Read(ctx, tx, runnerID)
		if err != nil {
			return fmt.Errorf("error reading runner: %w", err)
		}
		runner.ETag = models.GetETag(runner, delete.ETag)
		identity, err := s.ReadIdentity(ctx, tx, runner.ID)
		if err != nil {
			return err
		}
		err = s.runnerStore.SoftDelete(ctx, tx, runner)
		if err != nil {
			return fmt.Errorf("error soft deleting runner: %w", err)
		}
		err = s.resourceLinkStore.Delete(ctx, tx, runner.GetID())
		if err != nil {
			return fmt.Errorf("error deleting resource link: %w", err)
		}
		err = s.removeRunnerFromRunnerStandardGroup(ctx, tx, runner, identity)
		if err != nil {
			return err
		}
		err = s.deleteRunnerCredentials(ctx, tx, identity)
		if err != nil {
			return err
		}
		// NOTE: Ownership and identity are not deleted during soft deletes
		s.Infof("Deleted runner %q", runner.ID)
		return nil
	})
}

// removeRunnerFromRunnerStandardGroup will remove the specified runner from the runners access control group for
// its parent legal entity, to remove its access rights.
func (s *RunnerService) removeRunnerFromRunnerStandardGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	runner *models.Runner,
	runnerIdentity *models.Identity,
) error {
	// Find the 'runner' standard group for the legal entity that owns this runner
	ownerLegalEntityID := runner.LegalEntityID
	group, err := s.groupService.ReadByName(ctx, txOrNil, ownerLegalEntityID, models.RunnerStandardGroup.Name)
	if err != nil {
		return fmt.Errorf("error reading standard group for runner: %w", err)
	}

	s.Tracef("Removing runner %s from group '%s' for legal entity %s (GroupID %s)",
		runner.ID, group.Name, ownerLegalEntityID, group.ID)

	bbSystem := models.BuildBeaverSystem
	err = s.groupService.RemoveMembership(ctx, txOrNil, group.ID, runnerIdentity.ID, &bbSystem)
	if err != nil {
		return fmt.Errorf("error removing %s from group '%s' for legal entity %s: %w",
			runner.ID, group.Name, ownerLegalEntityID, err)
	}

	return nil
}

// deleteRunnerCredentials will hard delete any credentials the runner was using to authenticate.
func (s *RunnerService) deleteRunnerCredentials(
	ctx context.Context,
	txOrNil *store.Tx,
	runnerIdentity *models.Identity,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("deleteRunnerCredentials: Searching database for credentials for runner identity %s", runnerIdentity.ID)
			credentials, cursor, err := s.credentialService.ListCredentialsForIdentity(ctx, tx, runnerIdentity.ID, pagination)
			if err != nil {
				return err
			}
			s.Tracef("deleteRunnerCredentials: Got a page of %d credentials in search", len(credentials))
			for _, credential := range credentials {
				s.Infof("Removing credential %s (type %s) for runner identity %s",
					credential.ID, credential.Type, runnerIdentity.ID)
				err = s.credentialService.Delete(ctx, tx, credential.ID)
				if err != nil {
					return fmt.Errorf("error removing credential %s (type %s) for runner identity %s: %w",
						credential.ID, credential.Type, runnerIdentity.ID, err)
				}
			}
			if cursor != nil && cursor.Next != nil {
				pagination.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		return nil
	})
}

// Search all runners. If searcher is set, the results will be limited to runners the searcher is authorized to
// see (via the read:runner permission). Use cursor to page through results, if any.
func (s *RunnerService) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.RunnerSearch) ([]*models.Runner, *models.Cursor, error) {
	err := search.Validate()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error validating search")
	}
	return s.runnerStore.Search(ctx, txOrNil, searcher, search)
}

// configureDefaultLabels ensures the runner's labels are populated with suitable defaults.
func (s *RunnerService) configureDefaultLabels(runner *models.Runner) {
	if runner.OperatingSystem != "" {
		runner.Labels = s.withLabel(runner.Labels, models.Label(runner.OperatingSystem))
	}
	if runner.Architecture != "" {
		runner.Labels = s.withLabel(runner.Labels, models.Label(runner.Architecture))
	}
}

// withLabel idempotently ensures labels contains label.
func (s *RunnerService) withLabel(labels models.Labels, label models.Label) models.Labels {
	for _, existing := range labels {
		if existing == label {
			return labels
		}
	}
	return append(labels, label)
}

// splitLabels looks at a runner's existing labels, and a new candidate set of labels, and works out which
// labels need to be created and which need to be deleted in order to apply the candidate set to the runner.
func (s *RunnerService) splitLabels(existing models.Labels, candidate models.Labels) (toCreate models.Labels, toDelete models.Labels) {
	var (
		candidateM = make(map[models.Label]struct{})
		existingM  = make(map[models.Label]struct{})
	)
	for _, label := range candidate {
		candidateM[label] = struct{}{}
	}
	for _, label := range existing {
		if _, ok := candidateM[label]; !ok {
			toDelete = append(toDelete, label)
		}
		existingM[label] = struct{}{}
	}
	for _, label := range candidate {
		if _, ok := existingM[label]; !ok {
			toCreate = append(toCreate, label)
		}
	}
	return toCreate, toDelete
}

// splitTypes looks at a runner's existing supported types, and a new candidate set of supported types, and works
// out which need to be created and which need to be deleted in order to apply the candidate set to the runner.
func (s *RunnerService) splitTypes(existing models.JobTypes, candidate models.JobTypes) (toCreate models.JobTypes, toDelete models.JobTypes) {
	var (
		candidateM = make(map[models.JobType]struct{})
		existingM  = make(map[models.JobType]struct{})
	)
	for _, jobType := range candidate {
		candidateM[jobType] = struct{}{}
	}
	for _, jobType := range existing {
		if _, ok := candidateM[jobType]; !ok {
			toDelete = append(toDelete, jobType)
		}
		existingM[jobType] = struct{}{}
	}
	for _, jobType := range candidate {
		if _, ok := existingM[jobType]; !ok {
			toCreate = append(toCreate, jobType)
		}
	}
	return toCreate, toDelete
}
