package legal_entity

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/server/store"

	"github.com/buildbeaver/buildbeaver/server/services"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

type LegalEntityService struct {
	db                         *store.DB
	legalEntityStore           store.LegalEntityStore
	legalEntityMembershipStore store.LegalEntityMembershipStore
	ownershipStore             store.OwnershipStore
	resourceLinkStore          store.ResourceLinkStore
	identityStore              store.IdentityStore
	authorizationService       services.AuthorizationService
	groupService               services.GroupService
	logger.Log
}

func NewLegalEntityService(
	db *store.DB,
	legalEntityStore store.LegalEntityStore,
	legalEntityMembershipStore store.LegalEntityMembershipStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	identityStore store.IdentityStore,
	authorizationService services.AuthorizationService,
	groupService services.GroupService,
	logFactory logger.LogFactory,
) *LegalEntityService {
	return &LegalEntityService{
		db:                         db,
		legalEntityStore:           legalEntityStore,
		legalEntityMembershipStore: legalEntityMembershipStore,
		ownershipStore:             ownershipStore,
		resourceLinkStore:          resourceLinkStore,
		identityStore:              identityStore,
		authorizationService:       authorizationService,
		groupService:               groupService,
		Log:                        logFactory("LegalEntityService"),
	}
}

// Create creates a new legal entity and configures default access control rules.
func (s *LegalEntityService) Create(ctx context.Context, txOrNil *store.Tx, legalEntityData *models.LegalEntityData) (*models.LegalEntity, error) {
	var legalEntity *models.LegalEntity

	err := legalEntityData.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating legal entity")
	}

	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Create the Legal Entity
		legalEntity, err = s.legalEntityStore.Create(ctx, tx, legalEntityData)
		if err != nil {
			return errors.Wrap(err, "error creating legal entity")
		}

		// Ensure the legal entity has an up-to-date resource link matching its name
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, legalEntity)
		if err != nil {
			return fmt.Errorf("error maintaining legal entity name: %w", err)
		}

		// Set up other resources required for newly-created legal entity
		err = s.configureNewLegalEntity(ctx, tx, legalEntity)
		if err != nil {
			return fmt.Errorf("error configuring default legal entity grants: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return legalEntity, err
}

// Read an existing legal entity, looking it up by ID.
func (s *LegalEntityService) Read(ctx context.Context, txOrNil *store.Tx, id models.LegalEntityID) (*models.LegalEntity, error) {
	return s.legalEntityStore.Read(ctx, txOrNil, id)
}

// ReadByExternalID reads an existing legal entity, looking it up by its external id.
// Returns models.ErrNotFound if the legal entity does not exist.
func (s *LegalEntityService) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.LegalEntity, error) {
	return s.legalEntityStore.ReadByExternalID(ctx, txOrNil, externalID)
}

// ReadByIdentityID reads an existing legal entity, looking it up by the ID of its associated Identity.
func (s *LegalEntityService) ReadByIdentityID(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) (*models.LegalEntity, error) {
	// Read identity and check it is owned by a Legal Entity
	identity, err := s.identityStore.Read(ctx, txOrNil, identityID)
	if err != nil {
		return nil, fmt.Errorf("error reading Identity for Legal Entity: %w", err)
	}
	if identity.OwnerResourceID.Kind() != models.LegalEntityResourceKind {
		return nil, fmt.Errorf("error reading legal entity: Identity owner %s is not a legal entity", identity.OwnerResourceID)
	}
	legalEntityID := models.LegalEntityIDFromResourceID(identity.OwnerResourceID)

	return s.Read(ctx, txOrNil, legalEntityID)
}

// ReadIdentity reads and returns the Identity for the specified Legal Entity.
func (s *LegalEntityService) ReadIdentity(ctx context.Context, txOrNil *store.Tx, id models.LegalEntityID) (*models.Identity, error) {
	return s.identityStore.ReadByOwnerResource(ctx, txOrNil, id.ResourceID)
}

// FindOrCreate creates a legal entity if no legal entity with the same External ID already exists,
// otherwise it reads and returns the existing legal entity.
// Returns the legal entity as it is in the database, and true iff a new legal entity was created.
func (s *LegalEntityService) FindOrCreate(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityData *models.LegalEntityData,
) (legalEntity *models.LegalEntity, created bool, err error) {
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		legalEntity, created, err = s.legalEntityStore.FindOrCreate(ctx, tx, legalEntityData)
		if err != nil {
			return fmt.Errorf("error finding or creating legal entity: %w", err)
		}
		if created {
			// Ensure the legal entity has an up-to-date resource link matching its name
			_, _, err = s.resourceLinkStore.Upsert(ctx, tx, legalEntity)
			if err != nil {
				return fmt.Errorf("error upserting resource link: %w", err)
			}
			// Set up other resources required for newly-created legal entity
			err = s.configureNewLegalEntity(ctx, tx, legalEntity)
			if err != nil {
				return fmt.Errorf("error configuring default legal entity grants: %w", err)
			}
		}
		return nil
	})
	return legalEntity, created, err
}

// Upsert creates a legal entity if no legal entity with the same External ID already exists, otherwise it updates
// the existing legal entity's data if it differs from the supplied data.
// Returns the LegalEntity as it exists in the database after the create or update, and
// true,false if the resource was created, false,true if the resource was updated, or false,false if
// neither create nor update was necessary.
func (s *LegalEntityService) Upsert(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityData *models.LegalEntityData,
) (*models.LegalEntity, bool, bool, error) {
	err := legalEntityData.Validate()
	if err != nil {
		return nil, false, false, errors.Wrap(err, "error validating legal entity")
	}
	var (
		legalEntity *models.LegalEntity
		created     bool
		updated     bool
	)
	err = s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		legalEntity, created, updated, err = s.legalEntityStore.Upsert(ctx, tx, legalEntityData)
		if err != nil {
			return fmt.Errorf("error upserting legal entity: %w", err)
		}

		if created || updated {
			// Ensure the legal entity has an up-to-date resource link matching its name
			_, _, err = s.resourceLinkStore.Upsert(ctx, tx, legalEntity)
			if err != nil {
				return fmt.Errorf("error upserting resource link: %w", err)
			}
		}

		if created {
			// Set up other resources required for newly-created legal entity
			err = s.configureNewLegalEntity(ctx, tx, legalEntity)
			if err != nil {
				return fmt.Errorf("error configuring default legal entity grants: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, false, err
	}
	return legalEntity, created, updated, err
}

// Update an existing legal entity with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *LegalEntityService) Update(ctx context.Context, txOrNil *store.Tx, legalEntity *models.LegalEntity) error {
	return s.legalEntityStore.Update(ctx, txOrNil, legalEntity)
}

// ListParentLegalEntities lists all legal entities a legal entity is a member of. Use cursor to page through results, if any.
func (s *LegalEntityService) ListParentLegalEntities(ctx context.Context, txOrNil *store.Tx, legalEntityID models.LegalEntityID, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error) {
	return s.legalEntityStore.ListParentLegalEntities(ctx, txOrNil, legalEntityID, pagination)
}

// ListMemberLegalEntities lists all legal entities that are members of a parent legal entity. Use cursor to page through results, if any.
func (s *LegalEntityService) ListMemberLegalEntities(ctx context.Context, txOrNil *store.Tx, parentLegalEntityID models.LegalEntityID, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error) {
	return s.legalEntityStore.ListMemberLegalEntities(ctx, txOrNil, parentLegalEntityID, pagination)
}

// ListAllLegalEntities lists all legal entities in the system. Use cursor to page through results, if any.
func (s *LegalEntityService) ListAllLegalEntities(ctx context.Context, txOrNil *store.Tx, pagination models.Pagination) ([]*models.LegalEntity, *models.Cursor, error) {
	return s.legalEntityStore.ListAllLegalEntities(ctx, txOrNil, pagination)
}

// AddCompanyMember adds a user as a member of a particular company. The legal entity for the user and the company
// must already exist in the database. This method is idempotent.
func (s *LegalEntityService) AddCompanyMember(
	ctx context.Context,
	txOrNil *store.Tx,
	companyID models.LegalEntityID,
	memberID models.LegalEntityID,
) error {
	// Add all required records for a company member inside a transaction
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		now := models.NewTime(time.Now())

		// Read company and member to check they exist and are of the correct types
		company, err := s.legalEntityStore.Read(ctx, tx, companyID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading company with ID %s: %w", companyID, err)
		}
		if company.Type != models.LegalEntityTypeCompany {
			return fmt.Errorf("error adding user to company: legal entity with ID %s name '%s' is not a company (type is '%s')",
				company.ID, company.Name, company.Type)
		}
		member, err := s.legalEntityStore.Read(ctx, tx, memberID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading user with ID %s: %w", memberID, err)
		}
		if member.Type != models.LegalEntityTypePerson {
			return fmt.Errorf("error adding user to company: legal entity with ID %s name '%s' is not a person (type is '%s')",
				member.ID, member.Name, member.Type)
		}
		memberIdentity, err := s.ReadIdentity(ctx, tx, member.ID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading identity for with ID %s, name %s: %w", member.ID, member.Name, err)
		}

		// Make the user a member of the company's legal entity
		membership := models.NewLegalEntityMembership(now, companyID, memberID)
		_, created, err := s.legalEntityMembershipStore.FindOrCreate(ctx, tx, membership)
		if err != nil {
			return fmt.Errorf("error creating legal entity membership: %w", err)
		}
		if created {
			s.Infof("Created new membership for member %s, name '%s' to company %s, name '%s'",
				member.ID, member.Name, company.ID, company.Name)
		}

		// Every company member should automatically be a member of the base group for the company
		baseGroup, err := s.groupService.ReadByName(ctx, tx, company.ID, models.BaseStandardGroup.Name)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading '%s' group for company legal entity with ID %s, name %s: %w",
				models.BaseStandardGroup.Name, company.ID, company.Name, err)
		}
		_, _, err = s.groupService.FindOrCreateMembership(ctx, tx, models.NewGroupMembershipData(
			baseGroup.ID,
			memberIdentity.ID,
			models.BuildBeaverSystem, // membership came from BuildBeaver
			company.ID,               // added by the company
		))
		if err != nil {
			return fmt.Errorf("error adding user to company: error adding member to '%s' group for company legal entity with ID %s, name %s: %w",
				models.BaseStandardGroup.Name, company.ID, company.Name, err)
		}

		return nil
	})
}

// RemoveCompanyMember removes records for a user who is no longer a member of a particular company.
// The user will be removed from all groups owned by the company, and removed as a member of the company's legal entity.
// The legal entity for the user and the company must already exist in the database.
func (s *LegalEntityService) RemoveCompanyMember(
	ctx context.Context,
	txOrNil *store.Tx,
	companyID models.LegalEntityID,
	memberID models.LegalEntityID,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Read company and member to check they exist and are of the correct types
		company, err := s.legalEntityStore.Read(ctx, tx, companyID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading company with ID %s: %w", companyID, err)
		}
		if company.Type != models.LegalEntityTypeCompany {
			return fmt.Errorf("error adding user to company: legal entity with ID %s name '%s' is not a company (type is '%s')",
				company.ID, company.Name, company.Type)
		}
		member, err := s.legalEntityStore.Read(ctx, tx, memberID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading user with ID %s: %w", memberID, err)
		}
		if member.Type != models.LegalEntityTypePerson {
			return fmt.Errorf("error adding user to company: legal entity with ID %s name '%s' is not a person (type is '%s')",
				member.ID, member.Name, member.Type)
		}
		memberIdentity, err := s.ReadIdentity(ctx, tx, member.ID)
		if err != nil {
			return fmt.Errorf("error adding user to company: error reading identity for with ID %s, name %s: %w", member.ID, member.Name, err)
		}

		// Remove member from all access control groups within the parent company
		err = s.removeUserFromAllCompanyGroups(ctx, tx, member, memberIdentity, company)
		if err != nil {
			return fmt.Errorf("error removing user from access control group(s) of org %s: %w", company.ID, err)
		}

		// Remove the member from the parent company
		s.Infof("Removing user %s (name %q) as a member of organization %s (name %q)",
			member.ID, member.Name, company.ID, company.Name)
		err = s.legalEntityMembershipStore.DeleteByMember(ctx, tx, company.ID, member.ID)
		if err != nil {
			return fmt.Errorf("error removing user %s (name %q) as a member of organization %s (name %q): %w",
				member.ID, member.Name, company.ID, company.Name, err)
		}

		return nil
	})
}

// removeMemberFromAllCompanyGroups removes all access control group memberships for the specified member (user)
// for a particular company. Memberships for both standard and custom groups are removed.
func (s *LegalEntityService) removeUserFromAllCompanyGroups(
	ctx context.Context,
	txOrNil *store.Tx,
	member *models.LegalEntity,
	memberIdentity *models.Identity,
	company *models.LegalEntity,
) error {
	// Perform the search and all access control group membership removals inside a transaction for consistency
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
		for moreResults := true; moreResults; {
			s.Tracef("removeUserFromAllGroupForCompany: Searching database for groups within %s for identity %s", company.Name, memberIdentity.ID)
			groupsInDB, cursor, err := s.groupService.ListGroups(ctx, tx, &company.ID, &memberIdentity.ID, pagination)
			if err != nil {
				return err
			}
			s.Tracef("removeUserFromAllGroupForCompany: Got a page of %d groups in search", len(groupsInDB))
			for _, groupInDB := range groupsInDB {
				s.Infof("Removing user identity %s from access control group %s (name %q) for org %s (name %q)",
					memberIdentity.ID, groupInDB.ID, groupInDB.Name, member.ID, member.Name)
				err = s.groupService.RemoveMembership(ctx, tx, groupInDB.ID, memberIdentity.ID, nil)
				if err != nil {
					s.Warnf("error removing user identity from access control group %s (name %q) for org %s (name %q); continuing with Sync: %s",
						groupInDB.ID, groupInDB.Name, member.ID, member.Name, err)
					continue
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

// configureNewLegalEntity sets up various resources required when a new Legal Entity is created,
// including ownership, identity and permissions.
func (s *LegalEntityService) configureNewLegalEntity(ctx context.Context, tx *store.Tx, legalEntity *models.LegalEntity) error {
	if tx == nil {
		s.Panic("configureNewLegalEntity() expected to be inside transaction")
	}
	now := models.NewTime(time.Now())

	// Create ownership record - legal entity is top of the hierarchy, so it owns itself
	ownership := models.NewOwnership(now, legalEntity.ID.ResourceID, legalEntity.ID.ResourceID)
	err := s.ownershipStore.Create(ctx, tx, ownership)
	if err != nil {
		return fmt.Errorf("error creating ownership for new legal entity: %w", err)
	}

	// Create an Identity for the Legal Entity, owned by the Legal Entity
	identity := models.NewIdentity(now, legalEntity.ID.ResourceID)
	err = s.identityStore.Create(ctx, tx, identity)
	if err != nil {
		return fmt.Errorf("error creating an identity for new legal entity: %w", err)
	}
	identityOwnership := models.NewOwnership(now, legalEntity.ID.ResourceID, identity.ID.ResourceID)
	err = s.ownershipStore.Create(ctx, tx, identityOwnership)
	if err != nil {
		return fmt.Errorf("error creating identity ownership for new legal entity: %w", err)
	}

	// Create default permissions - the Legal Entity's identity has full access control privileges over
	// all resources that the Legal Entity owns.
	err = s.authorizationService.CreateGrantsForIdentity(
		ctx,
		tx,
		legalEntity.ID,
		identity.ID,
		models.LegalEntityAccessControlOperations,
		legalEntity.GetID())
	if err != nil {
		return fmt.Errorf("error creating grants for new legal entity: %w", err)
	}

	switch legalEntity.Type {
	case models.LegalEntityTypeCompany:
		// create standard company groups
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.BaseStandardGroup)
		if err != nil {
			return err
		}
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.ReadOnlyUserStandardGroup)
		if err != nil {
			return err
		}
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.UserStandardGroup)
		if err != nil {
			return err
		}
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.AdminStandardGroup)
		if err != nil {
			return err
		}
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.RunnerStandardGroup)
		if err != nil {
			return err
		}
	case models.LegalEntityTypePerson:
		// Ensure we have a standard group for runners for this person's repos
		_, err = s.groupService.FindOrCreateStandardGroup(ctx, tx, legalEntity, models.RunnerStandardGroup)
		if err != nil {
			return err
		}
	}

	s.Infof("Created legal entity %q", legalEntity.ID)
	return nil
}
