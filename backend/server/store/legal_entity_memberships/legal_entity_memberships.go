package legal_entity_memberships

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	store.MustDBModel(&models.LegalEntityMembership{})
}

type LegalEntityMembershipStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *LegalEntityMembershipStore {
	return &LegalEntityMembershipStore{
		table: store.NewResourceTableWithTableName(db, logFactory, "legal_entities_memberships", &models.LegalEntityMembership{}),
	}
}

// Create a new legal entity membership.
// Returns store.ErrAlreadyExists if a legal entity membership with matching unique properties already exists.
func (d *LegalEntityMembershipStore) Create(ctx context.Context, txOrNil *store.Tx, membership *models.LegalEntityMembership) error {
	return d.table.Create(ctx, txOrNil, membership)
}

// ReadByMember reads an existing legal entity membership, looking it up by (parent) legal entity
// and member legal entity.
// Returns models.ErrNotFound if the legal entity membership does not exist.
func (d *LegalEntityMembershipStore) ReadByMember(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityID models.LegalEntityID,
	memberLegalEntityID models.LegalEntityID,
) (*models.LegalEntityMembership, error) {
	membership := &models.LegalEntityMembership{}
	whereClause := goqu.Ex{
		"legal_entities_membership_legal_entity_id":        legalEntityID,
		"legal_entities_membership_member_legal_entity_id": memberLegalEntityID,
	}
	return membership, d.table.ReadWhere(ctx, txOrNil, membership, whereClause)
}

// FindOrCreate finds and returns the legal entity membership with the (parent) legal entity
// and member legal entity specified in the supplied membership data.
// If no such membership exists then a new one is created and returned, and true is returned for 'created'.
func (d *LegalEntityMembershipStore) FindOrCreate(
	ctx context.Context,
	txOrNil *store.Tx,
	membershipData *models.LegalEntityMembership,
) (membership *models.LegalEntityMembership, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.ReadByMember(ctx, tx, membershipData.LegalEntityID, membershipData.MemberLegalEntityID)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			err := d.table.Create(ctx, tx, membershipData)
			return membershipData, err
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.LegalEntityMembership), created, nil
}

// DeleteByMember removes a member legal entity from a (parent) legal entity by deleting the relevant
// membership record. This method is idempotent.
func (d *LegalEntityMembershipStore) DeleteByMember(
	ctx context.Context,
	txOrNil *store.Tx,
	legalEntityID models.LegalEntityID,
	memberLegalEntityID models.LegalEntityID,
) error {
	return d.table.DeleteWhere(ctx, txOrNil,
		goqu.Ex{
			"legal_entities_membership_legal_entity_id":        legalEntityID,
			"legal_entities_membership_member_legal_entity_id": memberLegalEntityID,
		})
}
