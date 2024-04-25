package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// DefaultFullSyncAfter is the default length of time after which a full sync of a legal entity will be performed.
const DefaultFullSyncAfter = 24 * time.Hour

// DefaultPerLegalEntityTimeout is the default timeout for syncing a Legal Entity (including all sub-objects like repos).
const DefaultPerLegalEntityTimeout = 1 * time.Hour

// DefaultGlobalSyncTimeout is the default timeout for performing a Global Sync (including all legal entities).
const DefaultGlobalSyncTimeout = 12 * time.Hour

type SyncService struct {
	db                   *store.DB
	legalEntityService   services.LegalEntityService
	repoService          services.RepoService
	scmRegistry          *scm.SCMRegistry
	credentialService    services.CredentialService
	groupService         services.GroupService
	authorizationService services.AuthorizationService
	syncTimer            *SyncTimer
	logger.Log
}

func NewSyncService(
	db *store.DB,
	legalEntityService services.LegalEntityService,
	repoService services.RepoService,
	scmRegistry *scm.SCMRegistry,
	credentialService services.CredentialService,
	groupService services.GroupService,
	authorizationService services.AuthorizationService,
	logFactory logger.LogFactory,
) *SyncService {
	s := &SyncService{
		db:                   db,
		legalEntityService:   legalEntityService,
		repoService:          repoService,
		scmRegistry:          scmRegistry,
		credentialService:    credentialService,
		groupService:         groupService,
		authorizationService: authorizationService,
		Log:                  logFactory("SyncService"),
	}

	s.syncTimer = NewSyncTimer(db, s, logFactory)
	s.syncTimer.Start()
	return s
}

func (s *SyncService) Stop() {
	s.syncTimer.Stop()
}

// SyncAuthenticatedUser reads the details for the currently authenticated user from their SCM, and ensures
// there is a LegalEntity and Identity for the user in the database. Returns the Identity for the user.
func (s *SyncService) SyncAuthenticatedUser(ctx context.Context, auth models.SCMAuth) (*models.Identity, error) {
	s.Info("Performing SCM Sync of Legal Entity for authenticated user")
	var (
		userLegalEntityData *models.LegalEntityData
		userIdentity        *models.Identity
		err                 error
	)

	scmService, err := s.scmRegistry.Get(auth.Name())
	if err != nil {
		return nil, fmt.Errorf("error getting SCM: %w", err)
	}

	// Read user details from the SCM (including the App installation ID)
	userLegalEntityData, err = scmService.GetUserLegalEntityData(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("error getting user legal entity from SCM: %w", err)
	}
	s.Tracef("SCM Sync: Got user legal entity from SCM: %s", userLegalEntityData)

	// Perform all operations for setting up a user's legal entity inside a transaction
	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		// Upsert a legal entity for the person
		userLegalEntity, _, _, err := s.legalEntityService.Upsert(ctx, tx, userLegalEntityData)
		if err != nil {
			return fmt.Errorf("error upserting user legal entity: %w", err)
		}

		// Look up the legal entity's Identity
		userIdentity, err = s.legalEntityService.ReadIdentity(ctx, tx, userLegalEntity.ID)
		if err != nil {
			return fmt.Errorf("error finding identity for legal entity: %w", err)
		}

		// TODO: Find or create a credential for the user's identity and the GitHub user ID
		//credential := models.NewGitHubCredential(models.NewTime(time.Now()), userIdentity.ID, true, githubUserID)
		//_, credential, err = s.credentialService.FindOrCreateByGitHubID(ctx, nil, userLegalEntity.ID, credential)
		//if err != nil {
		//	return nil, stats, fmt.Errorf("error reading or creating credential: %w", err)
		//}
		//if !credential.IsEnabled {
		//	return nil, stats, errors.New("Account disabled; Please contact your administrator")
		//}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return userIdentity, nil
}

// GlobalSync performs a one-way synchronization operation for all data in the specified SCM into the database.
// New organizations and repos found will be added to the database.
// The basic details for each Legal Entity will be synced, and for each Legal Entity if the time since it was last
// successfully synced is more than 'fullSyncAfter' then a full sync of that Legal Entity will be performed.
// If fullSyncAfter is zero then a full sync will always be performed.
// If perLegalEntityTimeout is not zero then each legal entity will have at most this much time to sync, after which
// global sync will move on to the next legal entity.
func (s *SyncService) GlobalSync(
	ctx context.Context,
	scmName models.SystemName,
	fullSyncAfter time.Duration,
	perLegalEntityTimeout time.Duration,
) error {
	var (
		fullSyncCount   int
		quickSyncCount  int
		syncedRepoCount int
	)

	s.Infof("Beginning SCM Global Sync operation for SCM '%s'", scmName)
	scmService, err := s.scmRegistry.Get(scmName)
	if err != nil {
		return fmt.Errorf("error getting SCM: %w", err)
	}

	entities, err := scmService.ListLegalEntitiesRegisteredAsUsers(ctx)
	if err != nil {
		return fmt.Errorf("error listing legal entities using BuildBeaver for SCM %q: %w", scmName, err)
	}
	s.Infof("Found %d legal entities on SCM", len(entities))
	for _, legalEntityData := range entities {
		// Check for global context timeout before starting to sync the next legal entity
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Optionally set an individual timeout for sync of this legal entity
		var (
			legalEntityCtx = ctx
			cancelFunc     context.CancelFunc
		)
		if perLegalEntityTimeout != 0 {
			legalEntityCtx, cancelFunc = context.WithTimeout(ctx, perLegalEntityTimeout)
		}
		entityFullSynced, entityRepoCount, err := s.SyncLegalEntity(legalEntityCtx, legalEntityData, fullSyncAfter)
		if cancelFunc != nil {
			cancelFunc()
		}
		if err != nil {
			s.Warnf("Will ignore error performing SCM sync for legal entity '%s' (external id '%s'): %v", legalEntityData.Name, legalEntityData.ExternalID, err)
			continue
		}
		if entityFullSynced {
			fullSyncCount++
			syncedRepoCount += entityRepoCount
		} else {
			quickSyncCount++
		}
	}

	// TODO: Look for installs that have been removed, do something relevant with them
	s.Infof("SCM Global Sync operation completed for SCM %s; full sync performed for %d Legal Entities and %d repos, quick sync for %d Legal Entities",
		scmService.Name(), fullSyncCount, syncedRepoCount, quickSyncCount)
	return nil
}

// SyncLegalEntity performs a sync for a legal entity (user or company) that is using BuildBeaver,
// against the external system referred to by the legal entity's ExternalID.
// The basic details for the Legal Entity will be synced, and if the time since it was last successfully synced is
// more than 'fullSyncAfter' then a full sync of the Legal Entity will be performed.
// If fullSyncAfter is zero then a full sync will always be performed.
// Returns the number of repos currently on the SCM if a full sync was returned, otherwise zero.
func (s *SyncService) SyncLegalEntity(
	ctx context.Context,
	legalEntityData *models.LegalEntityData,
	fullSyncAfter time.Duration,
) (fullSync bool, repoCount int, err error) {
	// Sync with the SCM that the legal entity came from, determined via its ExternalID
	if legalEntityData.ExternalID == nil {
		return false, 0, fmt.Errorf("error Legal Entity name '%s' has no external ID; can't identify external SCM to sync with", legalEntityData.Name)
	}
	scmName := legalEntityData.ExternalID.ExternalSystem
	s.Infof("Beginning SCM Sync operation for legal entity name %s, for SCM '%s'", legalEntityData.Name, scmName)
	scmService, err := s.scmRegistry.Get(scmName)
	if err != nil {
		return false, 0, fmt.Errorf("error looking up external SCM for legal entity name %s: %w", legalEntityData.Name, err)
	}

	// Create or update the legal entity. Do this first.
	// Note that legal entity might already exist because it was added for some other reason but may
	// have no groups, permissions etc., so do not return early if the legal entity already exists.
	legalEntity, err := s.UpsertLegalEntity(ctx, nil, legalEntityData)
	if err != nil {
		return false, 0, err
	}

	// Check whether legal entity has been synced recently; if so, skip the full sync
	if fullSyncAfter != 0 && legalEntity.SyncedAt != nil && !legalEntity.SyncedAt.Time.IsZero() {
		if time.Now().UTC().Sub(legalEntity.SyncedAt.Time) < fullSyncAfter {
			s.Infof("Legal entity '%s' was synced recently; full sync will not be run", legalEntity.Name)
			return false, 0, nil
		}
	}

	// Sync repos before syncing groups, so that groups can store permissions referring to the repos
	repoCount, err = s.SyncReposForLegalEntity(ctx, scmService, legalEntity)
	if err != nil {
		return false, 0, fmt.Errorf(" error syncing repos for legal entity ID %s, name %q: %v", legalEntity.ID, legalEntity.Name, err)
	}

	if legalEntity.Type == models.LegalEntityTypeCompany {
		// Add and remove custom access control groups for the company.
		// Do this after syncing repos for the company, so groups can set up permissions referring to the repos.
		err = s.syncCompanyCustomGroups(ctx, nil, scmService, legalEntity)
		if err != nil {
			s.Warnf("Will ignore error syncing access control groups for company legal entity ID %s, name %q: %v", legalEntity.ID, legalEntity.Name, err)
		}

		// Add and remove members of company.
		// Do this after syncing company custom groups so that removed company members can be removed from all groups.
		err = s.syncCompanyMembers(ctx, nil, scmService, legalEntity)
		if err != nil {
			s.Warnf("Will ignore error syncing members for company legal entity ID %s, name %q: %v", legalEntity.ID, legalEntity.Name, err)
		}

		// Sync members of standard groups.
		s.syncCompanyStandardGroups(ctx, nil, scmService, legalEntity)
	}

	// Sync completed successfully; update the SyncedAt time on the legal entity
	legalEntity.SyncedAt = models.NewTimePtr(time.Now().UTC())
	err = s.legalEntityService.Update(ctx, nil, legalEntity)
	if err != nil {
		return false, 0, fmt.Errorf("error updating Synced At time for Legal Entity %s, name '%s': %w", legalEntity.ID, legalEntityData.Name, err)
	}

	s.Infof("SCM Sync operation completed for legal entity '%s' on SCM %s; synched %d repos", legalEntity.GetName(), scmService.Name(), repoCount)
	return true, repoCount, nil
}

// RemoveInstallationForLegalEntity performs operations required when BuildBeaver is no longer being used for a
// particular Legal Entity.
func (s *SyncService) RemoveInstallationForLegalEntity(ctx context.Context, legalEntityData *models.LegalEntityData) error {
	// TODO: When the GitHub app is uninstalled for an account, this should:
	// 1. Delete all access control for the account: company members, groups, group members, group permissions
	// 2. Disable builds for all repos (but don't soft-delete the repos)
	// 3. Some kind of notification to BuildBeaver staff
	return nil
}

// UpsertLegalEntity will create or update a database record with the specified legal entity data.
// All data must be filled out, including ExternalID and ExternalMetadata.
// Metadata (especially ID) does not need to be filled out.
func (s *SyncService) UpsertLegalEntity(ctx context.Context, txOrNil *store.Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, error) {
	legalEntity, _, _, err := s.legalEntityService.Upsert(ctx, txOrNil, legalEntityData)
	if err != nil {
		return nil, fmt.Errorf("error upserting legal entity: %w", err)
	}
	return legalEntity, nil
}

// SyncReposForLegalEntity adds a record for each Repo in a legal entity (company or user), and removes records
// for repos which are no longer accessible on the SCM.
// Returns the number of repos currently on the SCM.
func (s *SyncService) SyncReposForLegalEntity(
	ctx context.Context,
	scmService scm.SCM,
	legalEntity *models.LegalEntity,
) (repoCount int, err error) {
	s.Tracef(" Beginning Sync repos operation for Legal entity ID %s, name %q, SCM '%s'",
		legalEntity.ID, legalEntity.Name, scmService.Name())

	scmRepos, err := scmService.ListReposRegisteredForLegalEntity(ctx, legalEntity)
	if err != nil {
		return 0, fmt.Errorf("error listing repos for SCM legal entity %q (external id %q): %w",
			legalEntity.ID, legalEntity.ExternalID, err)
	}
	s.Infof("SCM Legal Entity Sync operation: Found %d repos on SCM for legal entity %s (name %q)", len(scmRepos), legalEntity.ID, legalEntity.Name)

	// Remove repos that are no longer accessible on the SCM.
	// Do this before upserting repos that are on the SCM, in case an old repo has been replaced with a new repo
	// with the same name; in this case we must remove the old repo before we can create the new one, to avoid
	// violating DB constraints around the name.
	err = s.removeObsoleteRepos(ctx, scmService, legalEntity, scmRepos)
	if err != nil {
		s.Warnf("Will ignore error checking for obsolete repos for %q (name %s) memberships: %v", legalEntity.ID, legalEntity.Name, err)
		err = nil
	}

	// With the old ones cleared out of the way we can now upsert the new or existing repos
	for _, repo := range scmRepos {
		s.Infof("SCM Sync operation: Upsert for repo %s, legal entity %s", repo.Name, legalEntity.ID)

		// Upsert the repo; no need for a transaction since this is the only per-repo operation
		err = s.UpsertRepo(ctx, nil, repo)
		if err != nil {
			s.Warnf("will ignore error upserting repo: %s", err)
			continue
		}
	}

	return len(scmRepos), nil
}

// UpsertRepo creates a new repo or updates an existing repo.
func (s *SyncService) UpsertRepo(ctx context.Context, txOrNil *store.Tx, repoData *models.Repo) error {
	_, _, err := s.repoService.Upsert(ctx, txOrNil, repoData)
	return err
}

// removeObsoleteRepos finds and deletes repos in the database owned by the specified owner legal entity,
// that are no longer visible to BuildBeaver on the SCM.
// Only repos with external IDs matching the specified SCM system will be considered for deletion.
// scmRepos is the list of repos owned by ownerLegalEntity that are visible to BuildBeaver on the SCM.
// Repos are matched on their ExternalID field, so this field must be filled out.
func (s *SyncService) removeObsoleteRepos(
	ctx context.Context,
	scmService scm.SCM,
	ownerLegalEntity *models.LegalEntity,
	scmRepos []*models.Repo,
) error {
	// Make a map for the set of external IDs for repos we found on the SCM
	scmRepoMap := make(map[models.ExternalResourceID]bool, len(scmRepos))
	for _, repo := range scmRepos {
		if repo.ExternalID != nil {
			scmRepoMap[*repo.ExternalID] = true
		}
	}

	// Perform the search and all repo soft deletes inside a transaction for consistency
	return s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		query := search.NewRepoQueryBuilder().WhereLegalEntityID(search.Equal, ownerLegalEntity.ID).Compile()
		for moreResults := true; moreResults; {
			s.Tracef("removeObsoleteRepos: Searching database for repos for legal entity name %s", ownerLegalEntity.Name)
			reposInDatabase, cursor, err := s.repoService.Search(ctx, tx, models.NoIdentity, query)
			if err != nil {
				return err
			}
			s.Tracef("removeObsoleteRepos: Got a page of %d repos in search", len(reposInDatabase))
			for _, repo := range reposInDatabase {
				// Only consider repos for deletion if they have an external ID that matches the SCM
				if repo.ExternalID != nil && repo.ExternalID.ExternalSystem == scmService.Name() {
					s.Tracef("removeObsoleteRepos: Looking for repo ID %s (name %q) on scmRepoMap", repo.ID, repo.Name)
					if _, repoFoundOnSCM := scmRepoMap[*repo.ExternalID]; !repoFoundOnSCM {
						s.Infof("removeObsoleteRepos: DID NOT find repo ID %s (name %q) on SCM; soft-deleting repo", repo.ID, repo.Name)
						err = s.doRemoveRepo(ctx, tx, repo)
						if err != nil {
							s.Warnf("unable to remove repo ID %s (name %q) - continuing with Sync: %s", repo.ID, repo.Name, err)
							continue
						}
					} else {
						s.Tracef("removeObsoleteRepos: Found repo ID %s (name %q) on scmRepoMap", repo.ID, repo.Name)
					}
				}
			}
			if cursor != nil && cursor.Next != nil {
				query.Cursor = cursor.Next // move on to next page of results
			} else {
				moreResults = false
			}
		}
		return nil
	})
}

// RemoveRepoByExternalID removes the repo with the specified External ID from the system.
func (s *SyncService) RemoveRepoByExternalID(ctx context.Context, txOrNil *store.Tx, repoExternalID models.ExternalResourceID) error {
	// Read the repo from the database in order to perform optimistic locking when soft deleting
	repo, err := s.repoService.ReadByExternalID(ctx, txOrNil, repoExternalID)
	if err != nil {
		if gerror.IsNotFound(err) {
			s.Infof("RemoveRepoByExternalID: repo with external ID '%s' not found; ignoring", repoExternalID)
			return nil
		} else {
			return err
		}
	}
	return s.doRemoveRepo(ctx, txOrNil, repo)
}

// doRemoveRepo removes a repo from BuildBeaver. The supplied repo must have been read from the database.
func (s *SyncService) doRemoveRepo(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) error {
	// Soft-delete the repo. If BuildBeaver con no longer see the repo using its current
	// installation ID because it was moved to another user or org then the repo will be
	// re-added under the new user or org with its new installation ID.
	// Do not delete grants for the soft-deleted repo since we still want to be able to display
	// the repo. (Defer deleting the grants until the repo is cleaned up by being hard-deleted).
	return s.repoService.SoftDelete(ctx, txOrNil, repo)
}

// syncCompanyStandardGroups adds and remove members of each standard group for a company legal entity.
// Errors are logged and ignored.
func (s *SyncService) syncCompanyStandardGroups(ctx context.Context, tx *store.Tx, scmService scm.SCM, company *models.LegalEntity) {
	err := s.syncMembersForCompanyGroupName(ctx, tx, scmService, company, models.AdminStandardGroup.Name)
	if err != nil {
		s.Warnf("Will ignore error syncing admin group members for company legal entity ID %s, name %q: %v", company.ID, company.Name, err)
	}
	err = s.syncMembersForCompanyGroupName(ctx, tx, scmService, company, models.ReadOnlyUserStandardGroup.Name)
	if err != nil {
		s.Warnf("Will ignore error syncing read-only user group members for company legal entity ID %s, name %q: %v", company.ID, company.Name, err)
	}
	err = s.syncMembersForCompanyGroupName(ctx, tx, scmService, company, models.UserStandardGroup.Name)
	if err != nil {
		s.Warnf("Will ignore error syncing read-write user group members for company legal entity ID %s, name %q: %v", company.ID, company.Name, err)
	}
}

// syncCompanyMembers adds a records for each user who is a member of a company, and removes records for users
// who are no longer members of the company.
// For each member user a legal entity will be created if this user doesn't already have one in the BuildBeaver database,
// and the user legal entity will be made a member of the company.
func (s *SyncService) syncCompanyMembers(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
) error {
	members, err := scmService.ListAllCompanyMembers(ctx, company)
	if err != nil {
		return err
	}
	s.Infof("Discovered %d members of company %s on SCM %s", len(members), company.Name, scmService.Name())
	for _, member := range members {
		err = s.AddCompanyMember(ctx, txOrNil, scmService, company, member)
		if err != nil {
			s.Errorf("ignoring error adding company member: %s", err.Error())
			continue
		}
	}

	err = s.removeObsoleteCompanyMemberships(ctx, txOrNil, scmService, company, members)
	if err != nil {
		s.Warnf("Ignoring error removing company '%s' (name '%s') memberships: %s", company.ID, company.Name, err.Error())
		err = nil
	}

	return nil
}

// AddCompanyMember adds records for a user who is a member of a particular company.
// A legal entity will be created if this user doesn't already have one in the database,
// and the user will be made a member of the company. This method is idempotent.
func (s *SyncService) AddCompanyMember(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	memberData *models.LegalEntityData,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Ensure we have a legal entity for the member
		memberLegalEntity, _, err := s.legalEntityService.FindOrCreate(ctx, tx, memberData)
		if err != nil {
			return fmt.Errorf("error attempting to find or create legal entity: %w", err)
		}

		s.Tracef("Find or create membership for user %s, name '%s' to company %s, name '%s'",
			memberLegalEntity.ID, memberLegalEntity.Name, company.ID, company.Name)

		err = s.legalEntityService.AddCompanyMember(ctx, tx, company.ID, memberLegalEntity.ID)
		if err != nil {
			return err
		}

		return nil
	})
}

// removeObsoleteCompanyMemberships finds members of the specified company in the BuildBeaver database that no longer
// show up as memberships on the SCM. Removes the corresponding legal entity memberships (and their access control
// group memberships) from the BuildBeaver database.
func (s *SyncService) removeObsoleteCompanyMemberships(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	scmMembers []*models.LegalEntityData,
) error {
	// Make a map of external IDs for the org members we found on the SCM
	scmMemberMap := make(map[models.ExternalResourceID]bool, len(scmMembers))
	for _, member := range scmMembers {
		if member.ExternalID != nil {
			scmMemberMap[*member.ExternalID] = true
		}
	}

	// Perform the search and all membership removals inside a transaction for consistency
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("removeObsoleteCompanyMemberships: Searching database for members of legal entity '%s'", company.Name)
			memberLegalEntities, cursor, err := s.legalEntityService.ListMemberLegalEntities(ctx, tx, company.ID, pagination)
			if err != nil {
				return err
			}
			s.Tracef("removeObsoleteCompanyMemberships: Got a page of %d legal entities in search", len(memberLegalEntities))
			for _, memberLegalEntity := range memberLegalEntities {
				// Only consider memberships for deletion if the member legal entity has an external ID that matches the SCM
				if memberLegalEntity.ExternalID != nil && memberLegalEntity.ExternalID.ExternalSystem == scmService.Name() {
					s.Tracef("removeObsoleteCompanyMemberships: Looking for legal entity ID %s (name %q) on scmMemberMap", memberLegalEntity.ID, memberLegalEntity.Name)
					if _, orgFoundOnSCM := scmMemberMap[*memberLegalEntity.ExternalID]; !orgFoundOnSCM {
						// Member that was read from the database is no longer a member on the SCM, so remove
						err = s.legalEntityService.RemoveCompanyMember(ctx, tx, company.ID, memberLegalEntity.ID)
						if err != nil {
							s.Warnf("error removing company member; continuing with Sync: %s", company.ID, err)
							continue
						}
					}
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

// RemoveCompanyMember removes records for a user who is no longer a member of a particular company.
// A legal entity will be created if this user doesn't already have one in the database,
// but the user will no longer be a member of the company.
func (s *SyncService) RemoveCompanyMember(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	memberData *models.LegalEntityData,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Ensure we have a legal entity for the member; this also ensures we know the ID of the member
		memberLegalEntity, _, err := s.legalEntityService.FindOrCreate(ctx, tx, memberData)
		if err != nil {
			return fmt.Errorf("error attempting to find or create legal entity to remove from company: %w", err)
		}

		err = s.legalEntityService.RemoveCompanyMember(ctx, tx, company.ID, memberLegalEntity.ID)
		if err != nil {
			return err
		}

		return nil
	})
}

// syncCompanyCustomGroups adds records for each custom access control group present on the SCM for a company, and
// removes records for access control groups for the company which are no longer present on the SCM.
// The meaning of a 'custom access control group' is dependent on the SCM; for example, GitHub teams are
// treated as SCM-specific custom access control groups.
// Standard groups will not be affected by this function; these access control groups are always present
// for any company or user.
func (s *SyncService) syncCompanyCustomGroups(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
) error {
	// Find the users who are members of this group
	groups, err := scmService.ListCompanyCustomGroups(ctx, company)
	if err != nil {
		return err
	}
	s.Infof("Discovered %d custom access control groups for company %s on SCM %s", len(groups), company.Name, scmService.Name())

	for _, groupData := range groups {
		group, err := s.UpsertCompanyCustomGroup(ctx, txOrNil, scmService, company, groupData)
		if err != nil {
			s.Errorf("ignoring error adding group: %s", err.Error())
			continue
		}
		err = s.syncCompanyGroupMembers(ctx, txOrNil, scmService, company, group)
		if err != nil {
			return fmt.Errorf("error syncing members of custom group '%s' (external ID '%s') for legal entity %s (external id %s): %s",
				group.Name, group.ExternalID, company.ID, company.ExternalID, err.Error())
		}
		err = s.SyncCompanyGroupPermissions(ctx, txOrNil, scmService, company, group)
		if err != nil {
			return fmt.Errorf("error syncing permissions for custom group '%s' (external ID '%s') for legal entity %s (external id %s): %s",
				group.Name, group.ExternalID, company.ID, company.ExternalID, err.Error())
		}
	}

	err = s.removeObsoleteCompanyCustomGroups(ctx, txOrNil, scmService, company, groups)
	if err != nil {
		s.Warnf("Ignoring error removing groups for company '%s' (name '%s'): %s", company.ID, company.Name, err.Error())
		err = nil
	}

	return nil
}

// UpsertCompanyCustomGroup adds a new custom group within a company, or updates an existing group.
// Returns the group as read from the database.
func (s *SyncService) UpsertCompanyCustomGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	groupData *models.Group,
) (*models.Group, error) {
	s.Infof("Find or create group '%s' (external ID '%s') for company %s, name '%s', SCM '%s'",
		groupData.Name, groupData.ExternalID, company.ID, company.Name, scmService.Name())
	now := models.NewTime(time.Now())

	// Make a new group data object to ensure metadata is correct; don't rely on the SCM code
	group := models.NewGroup(now, company.ID, groupData.Name, groupData.Description, false, groupData.ExternalID)
	_, _, err := s.groupService.UpsertByExternalID(ctx, txOrNil, group)
	if err != nil {
		return nil, fmt.Errorf("error adding group '%s' for legal entity %s (external id %s): %s",
			groupData.Name, company.ID, company.ExternalID, err.Error())
	}

	return group, nil
}

// removeObsoleteCompanyCustomGroups finds and deletes groups in the database owned by the specified legal entity,
// that are no longer visible to BuildBeaver on the SCM.
// Only repos with external IDs matching the specified SCM system will be considered for deletion.
// scmRepos is the list of repos owned by ownerLegalEntity that are visible to BuildBeaver on the SCM.
// Repos are matched on their ExternalID field, so this field must be filled out.
func (s *SyncService) removeObsoleteCompanyCustomGroups(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	ownerLegalEntity *models.LegalEntity,
	scmGroups []*models.Group,
) error {
	// Make a map for the set of external IDs for groups we found on the SCM
	scmGroupMap := make(map[models.ExternalResourceID]bool, len(scmGroups))
	for _, group := range scmGroups {
		if group.ExternalID != nil {
			scmGroupMap[*group.ExternalID] = true
		}
	}

	// Perform the search and all group deletes inside a transaction for consistency
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("removeObsoleteCompanyCustomGroups: Searching database for groups for legal entity name %s", ownerLegalEntity.Name)
			groupsInDB, cursor, err := s.groupService.ListGroups(ctx, tx, &ownerLegalEntity.ID, nil, pagination)
			if err != nil {
				return err
			}
			s.Tracef("removeObsoleteCompanyCustomGroups: Got a page of %d groups in search", len(groupsInDB))
			for _, group := range groupsInDB {
				// Only consider a group for deletion if it has an external ID that matches the SCM.
				// This prevents the deletion if standard groups, and custom groups from other SCMs.
				if group.ExternalID != nil && group.ExternalID.ExternalSystem == scmService.Name() {
					s.Tracef("removeObsoleteCompanyCustomGroups: Looking for group ID %s (name %q) on scmGroupMap", group.ID, group.Name)
					if _, groupFoundOnSCM := scmGroupMap[*group.ExternalID]; !groupFoundOnSCM {
						s.Infof("removeObsoleteCompanyCustomGroups: DID NOT find group ID %s (name %q) on SCM; deleting group", group.ID, group.Name)
						err = s.doRemoveGroup(ctx, tx, group)
						if err != nil {
							s.Warnf("error deleting group, continuing with Sync: %s", err)
							continue
						}
					} else {
						s.Tracef("removeObsoleteCompanyCustomGroups: Found group ID %s (name %q) on scmGroupMap", group.ID, group.Name)
					}
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

// RemoveGroupByExternalID removes the access control group with the specified External ID from the system.
// This method is idempotent; it is not an error if the group doesn't exist.
func (s *SyncService) RemoveGroupByExternalID(ctx context.Context, txOrNil *store.Tx, groupExternalID models.ExternalResourceID) error {
	group, err := s.groupService.ReadByExternalID(ctx, txOrNil, groupExternalID)
	if err != nil {
		if gerror.IsNotFound(err) {
			s.Infof("RemoveCustomGroupByExternalID: group with external ID '%s' not found; ignoring", groupExternalID)
			return nil
		} else {
			return err
		}
	}
	return s.doRemoveGroup(ctx, txOrNil, group)
}

// doRemoveGroup removes a custom group from BuildBeaver. The supplied group must have a valid ID.
// This method is idempotent; it is not an error if the group doesn't exist.
func (s *SyncService) doRemoveGroup(ctx context.Context, txOrNil *store.Tx, group *models.Group) error {
	// Hard-delete the group
	err := s.groupService.Delete(ctx, txOrNil, group.ID)
	if err != nil {
		return fmt.Errorf("error deleting group ID %s (name %q): %w", group.ID, group.Name, err)
	}
	return nil
}

// syncMembersForCompanyGroupName looks up an access control group by name within a company, then calls
// syncCompanyGroupMembers for the group.
func (s *SyncService) syncMembersForCompanyGroupName(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	standardGroupName models.ResourceName,
) error {
	// Look up the group name underneath the company's legal entity
	group, err := s.groupService.ReadByName(ctx, txOrNil, company.ID, standardGroupName)
	if err != nil {
		if gerror.IsNotFound(err) {
			return fmt.Errorf("error: Unable to find group '%s' for company ID %s, name '%s'", standardGroupName, company.ID, company.Name)
		} else {
			return fmt.Errorf("error attempting to find group '%s' for company ID %s, name '%s': %w", standardGroupName, company.ID, company.Name, err)
		}
	}

	return s.syncCompanyGroupMembers(ctx, txOrNil, scmService, company, group)
}

// syncCompanyGroupMembers adds records for each user who is a member of a particular access control group within a
// company, and removes records for users who are no longer part of the group.
// For each member user a legal entity will be created if this user doesn't already have one in the BuildBeaver database,
// and the user will be made a member of the access control group within the company.
func (s *SyncService) syncCompanyGroupMembers(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	group *models.Group,
) error {
	// Find the users who are members of this group
	members, err := scmService.ListCompanyGroupMembers(ctx, company, group)
	if err != nil {
		return err
	}

	s.Infof("Discovered %d members of company %s group %s on SCM %s", len(members), company.Name, group.Name, scmService.Name())
	for _, member := range members {
		err = s.AddCompanyGroupMember(ctx, txOrNil, scmService, company, group, member)
		if err != nil {
			s.Errorf("ignoring error adding group member: %s", err.Error())
			continue
		}
	}

	err = s.removeObsoleteGroupMembers(ctx, txOrNil, scmService, company, group, members)
	if err != nil {
		s.Warnf("Ignoring error removing group memberships for company '%s' (name '%s'): %s", company.ID, company.Name, err.Error())
		err = nil
	}

	return nil
}

// AddCompanyGroupMember adds records for a user who is a member of a particular access control group within a
// company. A legal entity will be created if this user doesn't already have one in the database,
// and the user will be made a member of the access control group within the company. The membership will be
// associated with the specified SCM service. This method is idempotent.
func (s *SyncService) AddCompanyGroupMember(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	group *models.Group,
	memberData *models.LegalEntityData,
) error {
	// Add all required records for a group member inside a transaction
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {

		// Ensure we have a legal entity for the member; this also ensures we know the ID of the member
		memberLegalEntity, _, err := s.legalEntityService.FindOrCreate(ctx, tx, memberData)
		if err != nil {
			return fmt.Errorf("error attempting to find or create legal entity to add to group: %w", err)
		}

		// Look up the member's Identity, required to add them to an access control group
		memberIdentity, err := s.legalEntityService.ReadIdentity(ctx, tx, memberLegalEntity.ID)
		if err != nil {
			return fmt.Errorf("error finding identity for legal entity to add to group: %w", err)
		}

		// Make the user a member of the company group
		s.Tracef("Find or create group membership for user %s, name '%s' to company %s, name '%s', group '%s'",
			memberLegalEntity.ID, memberLegalEntity.Name, company.ID, company.Name, group.Name)
		addedBy := company.ID                   // record that the user was added to this group by the company
		externalSystemName := scmService.Name() // the SCM is the external system for this membership record
		_, _, err = s.groupService.FindOrCreateMembership(ctx, tx, models.NewGroupMembershipData(
			group.ID, memberIdentity.ID, externalSystemName, addedBy))
		if err != nil {
			return fmt.Errorf("error adding user '%s' to group '%s' for SCM legal entity %s (external id %s): %w",
				memberLegalEntity.Name, group.Name, company.ID, company.ExternalID, err)
		}
		return nil
	})
}

// AddStandardGroupMember adds records for a user who is a member of a particular standard access control group
// within a company. A legal entity will be created if this user doesn't already have one in the database,
// and the user will be made a member of the standard group within the company. The membership will be associated
// with the specified SCM service. This method is idempotent.
func (s *SyncService) AddStandardGroupMember(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	standardGroupName models.ResourceName,
	memberData *models.LegalEntityData,
) error {
	// Look up the group name underneath the company's legal entity
	group, err := s.groupService.ReadByName(ctx, txOrNil, company.ID, standardGroupName)
	if err != nil {
		if gerror.IsNotFound(err) {
			return fmt.Errorf("error: Unable to find group '%s' for company ID %s, name '%s'", standardGroupName, company.ID, company.Name)
		} else {
			return fmt.Errorf("error attempting to find group '%s' for company ID %s, name '%s': %w", standardGroupName, company.ID, company.Name, err)
		}
	}

	return s.AddCompanyGroupMember(ctx, txOrNil, scmService, company, group, memberData)
}

// removeObsoleteGroupMembers finds and deletes membership records for members of the specified access control group
// in the BuildBeaver database that no longer show up as group members on the SCM.
func (s *SyncService) removeObsoleteGroupMembers(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	group *models.Group,
	scmGroupMembers []*models.LegalEntityData,
) error {
	// Make a map of external IDs for the group members we found on the SCM
	scmMemberMap := make(map[models.ExternalResourceID]bool, len(scmGroupMembers))
	for _, member := range scmGroupMembers {
		if member.ExternalID != nil {
			scmMemberMap[*member.ExternalID] = true
		}
	}

	// Perform the search and all access control group membership removals inside a transaction for consistency
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("removeObsoleteGroupMembers: Searching database for group memberships for org %s, group %s, system %s", company.Name, group.Name, scmService.Name())
			systemName := scmService.Name()
			memberships, cursor, err := s.groupService.ListGroupMemberships(ctx, tx, &group.ID, nil, &systemName, pagination)
			if err != nil {
				return err
			}
			s.Tracef("removeObsoleteGroupMembers: Got a page of %d group members in search", len(memberships))
			for _, membership := range memberships {
				// Find legal entity this identity is associated with (note it could be associated with a runner instead)
				memberLegalEntity, err := s.legalEntityService.ReadByIdentityID(ctx, tx, membership.MemberIdentityID)
				if err != nil {
					s.Infof("removeObsoleteGroupMembers ignoring Group member identity %s is not associated with a legal entity", membership.MemberIdentityID)
					continue
				}
				s.Tracef("removeObsoleteGroupMembers: Checking legal entity ID %s (name %q) on scmMemberMap", memberLegalEntity.ID, memberLegalEntity.Name)
				if _, memberFoundOnSCM := scmMemberMap[*memberLegalEntity.ExternalID]; !memberFoundOnSCM {
					// Remove the member identity from the group, but only if associated with the SCM
					s.Infof("Removing any group membership associated with SCM %s for user identity %s from access control group %s (name %q) for org %s (name %q)",
						scmService.Name(), membership.MemberIdentityID, group.ID, group.Name, company.ID, company.Name)
					systemName := scmService.Name()
					err = s.groupService.RemoveMembership(ctx, tx, membership.GroupID, membership.MemberIdentityID, &systemName)
					if err != nil {
						s.Warnf("error removing user identity %s from access control group %s (name %q) for org %s (name %q), system %q; continuing with Sync: %v",
							membership.MemberIdentityID, group.ID, group.Name, company.ID, company.Name, scmService.Name(), err)
						continue
					}
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

// RemoveCompanyGroupMember records that a user is no longer a member of a particular access control group
// within a company. A legal entity will be created if this user doesn't already have one in the database.
// Only memberships that were added in the context of the specified SCM service will be removed; memberships
// associated with other external systems and internal group memberships will not be touched.
func (s *SyncService) RemoveCompanyGroupMember(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	group *models.Group,
	memberData *models.LegalEntityData,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Ensure we have a legal entity for the member; this also ensures we know the ID of the member
		memberLegalEntity, _, err := s.legalEntityService.FindOrCreate(ctx, tx, memberData)
		if err != nil {
			return fmt.Errorf("error attempting to find or create legal entity to add to group: %w", err)
		}

		// Look up the member's Identity, required to add them to an access control group
		memberIdentity, err := s.legalEntityService.ReadIdentity(ctx, tx, memberLegalEntity.ID)
		if err != nil {
			return fmt.Errorf("error finding identity for legal entity to add to group: %w", err)
		}

		// Remove the member identity from the group
		s.Infof("Removing any group membership associated with SCM %s for user identity %s from access control group %s (name %q) for org %s (name %q)",
			scmService.Name(), memberIdentity.ID, group.ID, group.Name, company.ID, company.Name)
		systemName := scmService.Name()
		err = s.groupService.RemoveMembership(ctx, tx, group.ID, memberIdentity.ID, &systemName)
		if err != nil {
			return fmt.Errorf("error removing user identity %s from access control group %s (name %q) for org %s (name %q): %w",
				memberIdentity.ID, group.ID, group.Name, company.ID, company.Name, err)
		}

		return nil
	})
}

// SyncCompanyGroupPermissions adds and removes grant records to give a group appropriate permissions based
// on the corresponding permissions on the SCM.
func (s *SyncService) SyncCompanyGroupPermissions(
	ctx context.Context,
	txOrNil *store.Tx,
	scmService scm.SCM,
	company *models.LegalEntity,
	group *models.Group,
) error {
	// Find the permissions on the SCM for this group
	scmGrants, err := scmService.ListCompanyCustomGroupPermissions(ctx, company, group)
	if err != nil {
		return err
	}
	s.Infof("Sync group permissions: discovered %d grants for company %s group %s on SCM %s", len(scmGrants), company.Name, group.Name, scmService.Name())

	// Make all changes to the group's permissions in a single transaction
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		for _, grant := range scmGrants {
			s.Tracef("SyncCompanyGroupPermissions: Find or create grant for group '%s' (ID %s) for org '%s' (ID %s), operation %s, target %s",
				group.Name, group.ID, company.Name, company.ID, grant.GetOperation(), grant.TargetResourceID)
			_, _, err := s.authorizationService.FindOrCreateGrant(ctx, tx, grant)
			if err != nil {
				return fmt.Errorf("error attempting to find or create access control grant: %w", err)
			}
		}
		err = s.removeObsoleteGroupPermissions(ctx, tx, company, group, scmGrants)
		if err != nil {
			return fmt.Errorf("error removing group memberships for company '%s' (name '%s'): %v", company.ID, company.Name, err)
		}
		return nil
	})
}

// removeObsoleteGroupPermissions finds and deletes grants for the specified access control group in the
// BuildBeaver database that no longer show up as permissions on the SCM.
func (s *SyncService) removeObsoleteGroupPermissions(
	ctx context.Context,
	txOrNil *store.Tx,
	company *models.LegalEntity,
	group *models.Group,
	scmGrants []*models.Grant,
) error {
	// Make a map of unique strings for the grants found on the SCM. This allows us to compare grants in the
	// database with those returned by the SCM without having to match by ID.
	scmGrantMap := make(map[string]*models.Grant, len(scmGrants))
	for _, grant := range scmGrants {
		scmGrantMap[grant.ToUniqueString()] = grant
	}

	// Perform the search and all grant removals inside a transaction for consistency
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("removeObsoleteGroupPermissions: Searching database for group permissions for org %s, group %s", company.Name, group.Name)
			grants, cursor, err := s.authorizationService.ListGrantsForGroup(ctx, tx, group.ID, pagination)
			if err != nil {
				return err
			}
			s.Tracef("removeObsoleteGroupPermissions: Got a page of %d grants in search", len(grants))
			for _, grant := range grants {
				// Is the equivalent of this grant still active on the SCM?
				s.Tracef("removeObsoleteGroupPermissions: Checking grant ID %s on scmGrantMap", grant.ID)
				if _, grantFoundOnSCM := scmGrantMap[grant.ToUniqueString()]; !grantFoundOnSCM {
					s.Infof("Removing grant for group '%s' (ID %s) for org '%s' (ID %s), operation %s, target %s",
						group.Name, group.ID, company.Name, company.ID, grant.GetOperation(), grant.TargetResourceID)
					err = s.authorizationService.DeleteGrant(ctx, tx, grant.ID)
					if err != nil {
						s.Warnf("error deleting grant %s from access control group %s (name %q) for org %s (name %q); continuing with Sync: %s",
							grant.ID, group.ID, group.Name, company.ID, company.Name, err)
						continue
					}
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
