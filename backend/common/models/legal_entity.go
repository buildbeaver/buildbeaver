package models

import (
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/pkg/errors"
)

const LegalEntityResourceKind ResourceKind = "legal-entity"

type LegalEntityID struct {
	ResourceID
}

func NewLegalEntityID() LegalEntityID {
	return LegalEntityID{ResourceID: NewResourceID(LegalEntityResourceKind)}
}

func LegalEntityIDFromResourceID(id ResourceID) LegalEntityID {
	return LegalEntityID{ResourceID: id}
}

type LegalEntityMetadata struct {
	ID        LegalEntityID `json:"id" goqu:"skipupdate" db:"legal_entity_id"`
	CreatedAt Time          `json:"created_at" goqu:"skipupdate" db:"legal_entity_created_at"`
	UpdatedAt Time          `json:"updated_at" db:"legal_entity_updated_at"`
	DeletedAt *Time         `json:"deleted_at,omitempty" db:"legal_entity_deleted_at"`
	SyncedAt  *Time         `json:"synced_at,omitempty" db:"legal_entity_synced_at"`
	ETag      ETag          `json:"etag" db:"legal_entity_etag" hash:"ignore"`
}

type LegalEntityData struct {
	Name ResourceName `json:"name" db:"legal_entity_name"`
	// Type of this legal entity e.g. "person" or "company"
	Type LegalEntityType `json:"type" db:"legal_entity_type"`
	// LegalName is the full legal name of the legal entity
	LegalName        string              `json:"legal_name" db:"legal_entity_legal_name"`
	EmailAddress     string              `json:"email_address" db:"legal_entity_email_address"`
	ExternalID       *ExternalResourceID `json:"external_id" db:"legal_entity_external_id"`
	ExternalMetadata string              `json:"external_metadata" db:"legal_entity_external_metadata"`
}

type LegalEntity struct {
	LegalEntityMetadata
	LegalEntityData
}

func NewCompanyLegalEntityData(name ResourceName, legalName string, emailAddress string, externalID *ExternalResourceID, externalMetadata string) *LegalEntityData {
	return &LegalEntityData{
		Name:             name,
		LegalName:        legalName,
		Type:             LegalEntityTypeCompany,
		EmailAddress:     emailAddress,
		ExternalID:       externalID,
		ExternalMetadata: externalMetadata,
	}
}

func NewPersonLegalEntityData(name ResourceName, legalName string, emailAddress string, externalID *ExternalResourceID, externalMetadata string) *LegalEntityData {
	return &LegalEntityData{
		Name:             name,
		Type:             LegalEntityTypePerson,
		LegalName:        legalName,
		EmailAddress:     emailAddress,
		ExternalID:       externalID,
		ExternalMetadata: externalMetadata,
	}
}

func (m *LegalEntity) GetKind() ResourceKind {
	return LegalEntityResourceKind
}

func (m *LegalEntity) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *LegalEntity) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *LegalEntity) GetParentID() ResourceID {
	return ResourceID{} // No parent
}

func (m *LegalEntity) GetName() ResourceName {
	return m.Name
}

func (m *LegalEntity) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *LegalEntity) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *LegalEntity) GetETag() ETag {
	return m.ETag
}

func (m *LegalEntity) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *LegalEntity) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *LegalEntity) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *LegalEntity) IsUnreachable() bool {
	// Legal entities should never be unreachable, even after being soft-deleted
	return false
}

func (m *LegalEntity) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if m.DeletedAt != nil && m.DeletedAt.IsZero() {
		result = multierror.Append(result, errors.New("error deleted at must be non-zero when set"))
	}
	err := m.LegalEntityData.Validate()
	if err != nil {
		result = multierror.Append(result, fmt.Errorf("data is invalid: %s", err))
	}
	return result.ErrorOrNil()
}

func (m *LegalEntityData) Validate() error {
	var result *multierror.Error
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if !m.Type.Valid() {
		result = multierror.Append(result, errors.New("error type is invalid"))
	}
	if m.ExternalID != nil {
		if !m.ExternalID.Valid() {
			result = multierror.Append(result, errors.New("error external id is invalid"))
		}
	} else {
		if m.ExternalMetadata != "" {
			result = multierror.Append(result, errors.New("error external metadata must be empty when external id is not set"))
		}
	}
	return result.ErrorOrNil()
}
