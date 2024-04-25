package models

import (
	"errors"

	"github.com/hashicorp/go-multierror"
)

const SecretResourceKind ResourceKind = "secret"

type SecretID struct {
	ResourceID
}

func NewSecretID() SecretID {
	return SecretID{ResourceID: NewResourceID(SecretResourceKind)}
}

func SecretIDFromResourceID(id ResourceID) SecretID {
	return SecretID{ResourceID: id}
}

type SecretPlaintext struct {
	// Key of the secret, unique within a repo.
	Key string `json:"key"`
	// Value is the plaintext value of the secret.
	Value string `json:"value"`
	*Secret
}

type Secret struct {
	ID        SecretID     `json:"id" db:"secret_id"`
	Name      ResourceName `json:"name" db:"secret_name"`
	RepoID    RepoID       `json:"repo_id" db:"secret_repo_id"`
	CreatedAt Time         `json:"created_at" goqu:"skipupdate" db:"secret_created_at"`
	UpdatedAt Time         `json:"updated_at" db:"secret_updated_at"`
	ETag      ETag         `json:"etag" db:"secret_etag" hash:"ignore"`
	// KeyEncrypted is the name of the secret, unique within a repo, encrypted using DataKeyEncrypted.
	KeyEncrypted BinaryBlob `json:"-" db:"secret_key_encrypted"`
	// ValueEncrypted is the value of the secret, encrypted using DataKeyEncrypted.
	ValueEncrypted BinaryBlob `json:"-" db:"secret_value_encrypted"`
	// DataKeyEncrypted is the key that can be used to decrypt KeyEncrypted and ValueEncrypted.
	// This key is itself encrypted and must be decrypted before being used.
	DataKeyEncrypted BinaryBlob `json:"-" db:"secret_data_key_encrypted"`
	// IsInternal is true if this secret is an internal secret generated
	// by the system (as opposed to being supplied by a user).
	IsInternal bool `json:"is_internal" db:"secret_is_internal"`
}

func NewSecret(now Time, name ResourceName, repoID RepoID, keyEncrypted []byte, valueEncrypted []byte, dataKeyEncrypted []byte, internal bool) *Secret {
	return &Secret{
		ID:               NewSecretID(),
		Name:             name,
		RepoID:           repoID,
		CreatedAt:        now,
		UpdatedAt:        now,
		KeyEncrypted:     keyEncrypted,
		ValueEncrypted:   valueEncrypted,
		DataKeyEncrypted: dataKeyEncrypted,
		IsInternal:       internal,
	}
}

func (m *Secret) GetKind() ResourceKind {
	return SecretResourceKind
}

func (m *Secret) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Secret) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Secret) GetParentID() ResourceID {
	return m.RepoID.ResourceID
}

func (m *Secret) GetName() ResourceName {
	return m.Name
}

func (m *Secret) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Secret) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Secret) GetETag() ETag {
	return m.ETag
}

func (m *Secret) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Secret) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	// ValidateSecretName will call Name.Validate()
	if err := ValidateSecretName(m.Name); err != nil {
		result = multierror.Append(result, err)
	}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if m.KeyEncrypted == nil {
		result = multierror.Append(result, errors.New("error name must be set"))
	}
	if m.ValueEncrypted == nil {
		result = multierror.Append(result, errors.New("error value must be set"))
	}
	if m.DataKeyEncrypted == nil {
		result = multierror.Append(result, errors.New("error data key must be set"))
	}
	return result.ErrorOrNil()
}
