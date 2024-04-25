package authorization

import (
	"context"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type NoOpAuthorizationService struct {
	logger.Log
}

func NewNoOpAuthorizationService(
	logFactory logger.LogFactory) *NoOpAuthorizationService {
	return &NoOpAuthorizationService{
		Log: logFactory("NoOpAuthorizationService"),
	}
}

func (s *NoOpAuthorizationService) IsAuthorized(
	ctx context.Context,
	identityID models.IdentityID,
	operation *models.Operation,
	resourceID models.ResourceID) (bool, error) {

	s.Warnf("Authorize identity %s to perform %s on %s:%s", identityID, operation, resourceID)
	return true, nil
}

func (s *NoOpAuthorizationService) CreateGrantsForIdentity(
	ctx context.Context,
	txOrNil *store.Tx,
	grantedByLegalEntityID models.LegalEntityID,
	authorizedIdentityID models.IdentityID,
	operations []*models.Operation,
	resourceID models.ResourceID,
) error {
	return nil
}

func (s *NoOpAuthorizationService) CreateGrantsForGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	grantedByLegalEntityID models.LegalEntityID,
	authorizedRoleID models.GroupID,
	operations []*models.Operation,
	resourceID models.ResourceID,
) error {
	return nil
}

// DeleteGrant permanently and idempotently deletes a grant, identifying it by id.
func (s *NoOpAuthorizationService) DeleteGrant(ctx context.Context, txOrNil *store.Tx, id models.GrantID) error {
	return nil
}

// DeleteAllGrantsForIdentity permanently and idempotently deletes all grants for the specified identity.
func (s *NoOpAuthorizationService) DeleteAllGrantsForIdentity(ctx context.Context, txOrNil *store.Tx, identityID models.IdentityID) error {
	return nil
}

// FindOrCreateGrant finds and returns a grant with the data specified in the supplied grant data.
// The GrantStore.ReadByAuthorizedOperation function is used to find matching grants.
// If no such grant exists then a new one is created and returned, and true is returned for 'created'.
func (s *NoOpAuthorizationService) FindOrCreateGrant(
	ctx context.Context,
	txOrNil *store.Tx,
	grantData *models.Grant,
) (grant *models.Grant, created bool, err error) {
	return grantData, false, nil
}

// ListGrantsForGroup finds and returns all grants that give permissions to the specified group.
func (s *NoOpAuthorizationService) ListGrantsForGroup(
	ctx context.Context,
	txOrNil *store.Tx,
	groupID models.GroupID,
	pagination models.Pagination,
) ([]*models.Grant, *models.Cursor, error) {
	return []*models.Grant{}, nil, nil
}
