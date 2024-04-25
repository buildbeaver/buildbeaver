package models

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const GrantResourceKind ResourceKind = "grant"

type GrantID struct {
	ResourceID
}

func NewGrantID() GrantID {
	return GrantID{ResourceID: NewResourceID(GrantResourceKind)}
}

func GrantIDFromResourceID(id ResourceID) GrantID {
	return GrantID{ResourceID: id}
}

type GrantMetadata struct {
	// ID uniquely identifies the grant
	ID        GrantID `json:"id" goqu:"skipupdate" db:"access_control_grant_id"`
	CreatedAt Time    `json:"created_at" goqu:"skipupdate" db:"access_control_grant_created_at"`
	UpdatedAt Time    `json:"updated_at" db:"access_control_grant_updated_at"`
}

type Grant struct {
	GrantMetadata
	// GrantedByLegalEntityID is the legal entity that granted this permission
	GrantedByLegalEntityID LegalEntityID `json:"granted_by_legal_entity_id" db:"access_control_grant_granted_by_legal_entity_id"`
	// AuthorizedIdentityID is the id of the identity that is authorized to perform the granted operation,
	// if this grant is directly to a specific identity
	AuthorizedIdentityID IdentityID `json:"authorized_identity_id" db:"access_control_grant_authorized_identity_id"`
	// AuthorizedGroupID is the id of the access control Group that is authorized to perform the
	// granted operation, if this grant is for a group.
	AuthorizedGroupID GroupID `json:"authorized_group_id" db:"access_control_grant_authorized_group_id"`
	// OperationResourceType is the type of resource the grant gives permission to e.g. 'repo'
	OperationResourceType ResourceKind `json:"operation_resource_kind" db:"access_control_grant_operation_resource_kind"`
	// OperationName is the name of the operation the authorized legal entity or group has permission
	// to perform on the target resource e.g. 'read' 'repo'
	OperationName string `json:"operation_name" db:"access_control_grant_operation_name"`
	// TargetResourceID is the id of the of resource the grant applies to.
	TargetResourceID ResourceID `json:"target_resource_id" db:"access_control_grant_target_resource_id"`
}

func NewIdentityGrant(now Time, grantedByLegalEntityID LegalEntityID, authorizedIdentityID IdentityID, operation Operation, targetResourcedID ResourceID) *Grant {
	return &Grant{
		GrantMetadata: GrantMetadata{
			ID:        NewGrantID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		GrantedByLegalEntityID: grantedByLegalEntityID,
		AuthorizedIdentityID:   authorizedIdentityID,
		OperationResourceType:  operation.ResourceKind,
		OperationName:          operation.Name,
		TargetResourceID:       targetResourcedID,
	}
}

func NewGroupGrant(now Time, grantedByLegalEntityID LegalEntityID, authorizedGroupID GroupID, operation Operation, targetResourcedID ResourceID) *Grant {
	return &Grant{
		GrantMetadata: GrantMetadata{
			ID:        NewGrantID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		GrantedByLegalEntityID: grantedByLegalEntityID,
		AuthorizedGroupID:      authorizedGroupID,
		OperationResourceType:  operation.ResourceKind,
		OperationName:          operation.Name,
		TargetResourceID:       targetResourcedID,
	}
}

func (m *Grant) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Grant) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Grant) GetKind() ResourceKind {
	return GrantResourceKind
}

func (m *Grant) GetOperation() Operation {
	return Operation{
		Name:         m.OperationName,
		ResourceKind: m.OperationResourceType,
	}
}

func (m *Grant) SetOperation(operation Operation) {
	m.OperationName = operation.Name
	m.OperationResourceType = operation.ResourceKind
}

// GetAuthorizedResourceID returns the resource ID of the group or identity being authorized by this grant.
func (m *Grant) GetAuthorizedResourceID() ResourceID {
	if !m.AuthorizedIdentityID.IsZero() {
		return m.AuthorizedIdentityID.ResourceID
	} else {
		return m.AuthorizedGroupID.ResourceID
	}
}

// ToUniqueString returns a string that uniquely identifies a grant based on the data in the grant, without
// including the grant ID. This can be used to check whether two grants are functionally equivalent.
func (m *Grant) ToUniqueString() string {
	return fmt.Sprintf("%s-%s-%s",
		m.GetAuthorizedResourceID().String(),
		m.GetOperation().String(),
		m.TargetResourceID.String(),
	)
}

func (m *Grant) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if !m.GrantedByLegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error granted by must be set"))
	}
	if m.AuthorizedIdentityID.Valid() == m.AuthorizedGroupID.Valid() {
		result = multierror.Append(result, errors.New("error exactly one of authorized legal entity id or group id must be set"))
	}
	if m.OperationResourceType == "" {
		result = multierror.Append(result, errors.New("error operation resource type must be set"))
	}
	if m.OperationName == "" {
		result = multierror.Append(result, errors.New("error operation name must be set"))
	}
	if !m.TargetResourceID.Valid() {
		result = multierror.Append(result, errors.New("error target resource id must be set"))
	}
	return result.ErrorOrNil()
}
