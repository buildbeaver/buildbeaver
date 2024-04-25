package authorization

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type AuthorizationService struct {
	db                 *store.DB
	grantStore         store.GrantStore
	ownershipStore     store.OwnershipStore
	authorizationStore store.AuthorizationStore
	logger.Log
}

func NewAuthorizationService(
	db *store.DB,
	grantStore store.GrantStore,
	ownershipStore store.OwnershipStore,
	authorizationStore store.AuthorizationStore,
	logFactory logger.LogFactory,
) *AuthorizationService {
	return &AuthorizationService{
		db:                 db,
		grantStore:         grantStore,
		ownershipStore:     ownershipStore,
		authorizationStore: authorizationStore,
		Log:                logFactory("AuthorizationService"),
	}
}

func (s *AuthorizationService) IsAuthorized(
	ctx context.Context,
	identityID models.IdentityID,
	operation *models.Operation,
	resourceID models.ResourceID) (bool, error) {

	count, err := s.authorizationStore.CountGrantsForOperation(
		ctx,
		nil,
		identityID,
		operation,
		resourceID)
	if err != nil {
		return false, errors.Wrap(err, "error listing grants")
	}
	if count > 0 {
		s.Infof("ALLOWED '%s' to '%s:%s' on '%s'",
			identityID, operation.ResourceKind, operation.Name, resourceID)
		return true, nil
	}
	s.Warnf("DENIED '%s' to '%s:%s' on '%s'",
		identityID, operation.ResourceKind, operation.Name, resourceID)
	return false, nil
}

// CreateGrantsForIdentity grants the specified identity a set of permissions.
// For each operation in the supplied list, the identity will be allowed to perform the
// specified operation on the specified resource or on any resource it owns (directly or indirectly),
// as long as the resource kind matches the kind specified in the operation.
func (s *AuthorizationService) CreateGrantsForIdentity(
	ctx context.Context,
	txOrNil *store.Tx,
	grantedByLegalEntityID models.LegalEntityID,
	authorizedIdentityID models.IdentityID,
	operations []*models.Operation,
	resourceID models.ResourceID,
) error {
	return s.createGrants(ctx, txOrNil, authorizedIdentityID.ResourceID, operations, resourceID,
		func(now time.Time, operation *models.Operation, res models.ResourceID) *models.Grant {
			return models.NewIdentityGrant(
				models.NewTime(now),
				grantedByLegalEntityID,
				authorizedIdentityID,
				*operation,
				resourceID)
		})
}

// CreateGrantsForGroup grants the specified access control Group a set of permissions.
// For each operation in the supplied list, identities in the specified group will be allowed to perform the
// specified operation on the specified resource or on any resource it owns (directly or indirectly),
// as long as the resource kind matches the kind specified in the operation.
func (s *AuthorizationService) CreateGrantsForGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	grantedByLegalEntityID models.LegalEntityID,
	authorizedGroupID models.GroupID,
	operations []*models.Operation,
	resourceID models.ResourceID,
) error {
	return s.createGrants(ctx, txOrNil, authorizedGroupID.ResourceID, operations, resourceID,
		func(now time.Time, operation *models.Operation, res models.ResourceID) *models.Grant {
			return models.NewGroupGrant(
				models.NewTime(now),
				grantedByLegalEntityID,
				authorizedGroupID,
				*operation,
				resourceID)
		})
}

// createGrants grants the specified authorized ID (either a group or a legal entity) a set of permissions.
// A grantFactory function must be provided that creates grant data of the appropriate type for the authorized ID.
// Each grant added via a FindOrCreate, so grants that already exist are left in place.
func (s *AuthorizationService) createGrants(
	ctx context.Context,
	txOrNil *store.Tx,
	authorizedID models.ResourceID,
	operations []*models.Operation,
	resourceID models.ResourceID,
	grantFactory func(time.Time, *models.Operation, models.ResourceID) *models.Grant,
) error {
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		now := time.Now().UTC()
		for _, operation := range operations {
			grantData := grantFactory(now, operation, resourceID)
			grant, grantCreated, err := s.grantStore.FindOrCreate(ctx, tx, grantData)
			if err != nil {
				return errors.Wrap(err, "error creating grant")
			}
			if grantCreated {
				// Grant is owned by (i.e. a child of) the target resource, *not* GrantedByLegalEntity
				// This allows the owner of a resource to see all grants for that resource, no matter who made them
				ownership := models.NewOwnership(models.NewTime(now), grant.TargetResourceID, grant.GetID())
				_, _, err = s.ownershipStore.Upsert(ctx, tx, ownership)
				if err != nil {
					return errors.Wrap(err, "error creating ownership")
				}
				s.Infof("Created grant for %s to perform %s on %s",
					authorizedID, grant.GetOperation(), grant.TargetResourceID)
			}
		}
		return nil
	})
}

// DeleteGrant permanently and idempotently deletes a grant, identifying it by id.
func (s *AuthorizationService) DeleteGrant(ctx context.Context, txOrNil *store.Tx, id models.GrantID) error {
	return s.grantStore.Delete(ctx, txOrNil, id)
}

// DeleteAllGrantsForIdentity permanently and idempotently deletes all grants for the specified identity.
func (s *AuthorizationService) DeleteAllGrantsForIdentity(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) error {
	return s.grantStore.DeleteAllGrantsForIdentity(ctx, txOrNil, identityID)
}

// FindOrCreateGrant finds and returns a grant with the data specified in the supplied grant data.
// The GrantStore.ReadByAuthorizedOperation function is used to find matching grants.
// If no such grant exists then a new one is created and returned, and true is returned for 'created'.
func (s *AuthorizationService) FindOrCreateGrant(
	ctx context.Context,
	txOrNil *store.Tx,
	grantData *models.Grant,
) (grant *models.Grant, created bool, err error) {
	return s.grantStore.FindOrCreate(ctx, txOrNil, grantData)
}

// ListGrantsForGroup finds and returns all grants that give permissions to the specified group.
func (s *AuthorizationService) ListGrantsForGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	pagination models.Pagination,
) ([]*models.Grant, *models.Cursor, error) {
	return s.grantStore.ListGrantsForGroup(ctx, txOrNil, groupID, pagination)
}
