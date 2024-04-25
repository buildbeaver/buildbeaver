package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const LegalEntityMembershipResourceKind ResourceKind = "legal-entity-membership"

type LegalEntityMembershipID struct {
	ResourceID
}

func NewLegalEntityMembershipID() LegalEntityMembershipID {
	return LegalEntityMembershipID{ResourceID: NewResourceID(LegalEntityMembershipResourceKind)}
}

func LegalEntityMembershipIDFromResourceID(id ResourceID) LegalEntityMembershipID {
	return LegalEntityMembershipID{ResourceID: id}
}

type LegalEntityMembership struct {
	ID                  LegalEntityMembershipID `json:"id" goqu:"skipupdate" db:"legal_entities_membership_id"`
	CreatedAt           Time                    `json:"created_at" goqu:"skipupdate" db:"legal_entities_membership_created_at"`
	LegalEntityID       LegalEntityID           `json:"legal_entity_id" db:"legal_entities_membership_legal_entity_id"`
	MemberLegalEntityID LegalEntityID           `json:"legal_entity_member_id" db:"legal_entities_membership_member_legal_entity_id"`
}

func NewLegalEntityMembership(now Time, legalEntityID LegalEntityID, memberLegalEntityID LegalEntityID) *LegalEntityMembership {
	return &LegalEntityMembership{
		ID:                  NewLegalEntityMembershipID(),
		CreatedAt:           now,
		LegalEntityID:       legalEntityID,
		MemberLegalEntityID: memberLegalEntityID,
	}
}

func (m *LegalEntityMembership) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *LegalEntityMembership) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *LegalEntityMembership) GetKind() ResourceKind {
	return LegalEntityMembershipResourceKind
}

func (m *LegalEntityMembership) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if !m.LegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error legal entity id must be set"))
	}
	if !m.MemberLegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error member legal entity id must be set"))
	}
	return result.ErrorOrNil()
}
