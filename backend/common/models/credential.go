package models

import (
	"github.com/hashicorp/go-multierror"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/gerror"
)

const CredentialResourceKind ResourceKind = "credential"

type CredentialID struct {
	ResourceID
}

func NewCredentialID() CredentialID {
	return CredentialID{ResourceID: NewResourceID(CredentialResourceKind)}
}

func CredentialIDFromResourceID(id ResourceID) CredentialID {
	return CredentialID{ResourceID: id}
}

type Credential struct {
	ID        CredentialID `json:"id" goqu:"skipupdate" db:"credential_id"`
	CreatedAt Time         `json:"created_at" goqu:"skipupdate" db:"credential_created_at"`
	UpdatedAt Time         `json:"updated_at" db:"credential_updated_at"`
	ETag      ETag         `json:"etag" db:"credential_etag" hash:"ignore"`
	// IdentityID is the id of the identity that this credential can be used to authenticate.
	IdentityID IdentityID `json:"identity_id" db:"credential_identity_id"`
	// Type of the credential e.g. shared secret or basic auth etc.
	Type CredentialType `json:"type" db:"credential_type"`
	// IsEnabled is true if this credential can be used or false if it is disabled
	IsEnabled bool `json:"is_enabled" db:"credential_is_enabled"`
	// SharedSecretID uniquely identifies the credential, if the credential
	// type is shared secret.
	SharedSecretID string `json:"-" db:"credential_shared_secret_id"`
	// SharedSecretSalt holds the unique salt used to hash the token, if
	// the credential type is shared secret.
	SharedSecretSalt BinaryBlob `json:"-" db:"credential_shared_secret_salt"`
	// SharedSecretDataHashed is a one-way hashed token that authenticates
	// a login account, if the credential type is shared secret.
	SharedSecretDataHashed BinaryBlob `json:"-" db:"credential_shared_secret_data_hashed"`
	// ClientPublicKeyASN1HashType is the type of hashing algorithm used to hash the client's public key,
	// if the credential type is 'client_certificate'.
	ClientPublicKeyASN1HashType HashType `json:"client_public_key_asn1_hash_type" db:"credential_client_public_key_asn1_hash_type"`
	// ClientPublicKeyASN1Hash contains the hex-encoded hash of the client's ASN.1 DER-encoded public key,
	// if the credential type is 'client_certificate'.
	ClientPublicKeyASN1Hash string `json:"client_public_key_asn1_hash" db:"credential_client_public_key_asn1_hash"`
	// ClientPublicKeyPEM is the client's public key in PEM format, if the credential type is 'client_certificate'.
	// Used for UI and display, but is not used during authentication.
	ClientPublicKeyPEM string `json:"client_public_key_pem" db:"credential_client_public_key_pem"`
	// ClientCertificateASN1 is the last certificate seen during TLS authentication that contains the client
	// public key, if the credential type is 'client_certificate'. The certificate is ASN.1 DER-encoded.
	// This certificate can change and does not have to match the certificate provided during registration of
	// the client's public key. Used for UI and display, but is not used during authentication.
	ClientCertificateASN1 BinaryBlob `json:"client_certificate_asn1" db:"credential_client_certificate_asn1"`
	// GitHubUserID is the unique id of the GitHub user this credential
	// belongs to, if the credential type is GitHub OAuth.
	GitHubUserID *int64 `json:"github_user_id" db:"credential_github_user_id"`
}

func NewSharedSecretCredential(now Time, identityID IdentityID, enabled bool, sharedSecret *SharedSecretToken) *Credential {
	salt, hash := sharedSecret.PrivateParts()
	return &Credential{
		ID:                     NewCredentialID(),
		CreatedAt:              now,
		UpdatedAt:              now,
		IdentityID:             identityID,
		Type:                   CredentialTypeSharedSecret,
		IsEnabled:              enabled,
		SharedSecretID:         sharedSecret.ID(),
		SharedSecretSalt:       salt,
		SharedSecretDataHashed: hash,
	}
}

func NewClientCertificateCredential(
	now Time,
	identityID IdentityID,
	enabled bool,
	clientCertificate certificates.CertificateData,
) (*Credential, error) {

	publicKey, err := certificates.GetPublicKeyFromCertificate(clientCertificate)
	if err != nil {
		return nil, err
	}

	return &Credential{
		ID:                          NewCredentialID(),
		CreatedAt:                   now,
		UpdatedAt:                   now,
		IdentityID:                  identityID,
		Type:                        CredentialTypeClientCertificate,
		IsEnabled:                   enabled,
		ClientPublicKeyASN1HashType: HashTypeSHA256,
		ClientPublicKeyASN1Hash:     publicKey.SHA256Hash(),
		ClientPublicKeyPEM:          publicKey.AsPEM(),
		ClientCertificateASN1:       []byte(clientCertificate),
	}, nil
}

func NewGitHubCredential(now Time, identityID IdentityID, enabled bool, gitHubUserID int64) *Credential {
	return &Credential{
		ID:           NewCredentialID(),
		CreatedAt:    now,
		UpdatedAt:    now,
		IdentityID:   identityID,
		Type:         CredentialTypeGitHubOAuth,
		IsEnabled:    enabled,
		GitHubUserID: &gitHubUserID,
	}
}

func (m *Credential) GetKind() ResourceKind {
	return CredentialResourceKind
}

func (m *Credential) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Credential) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Credential) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Credential) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Credential) GetETag() ETag {
	return m.ETag
}

func (m *Credential) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Credential) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, gerror.NewErrValidationFailed("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, gerror.NewErrValidationFailed("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, gerror.NewErrValidationFailed("error updated at must be set"))
	}
	if !m.IdentityID.Valid() {
		result = multierror.Append(result, gerror.NewErrValidationFailed("error login account id must be set"))
	}
	if !m.Type.Valid() {
		result = multierror.Append(result, gerror.NewErrValidationFailed("error credential type is invalid"))
	}
	switch m.Type {
	case CredentialTypeGitHubOAuth:
		if m.GitHubUserID == nil {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error github user id must be set"))
		}
	case CredentialTypeSharedSecret:
		if m.SharedSecretID == "" {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error shared secret id must be set"))
		}
		if m.SharedSecretSalt == nil {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error shared secret salt must be set"))
		}
		if m.SharedSecretDataHashed == nil {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error shared secret hash must be set"))
		}
	case CredentialTypeClientCertificate:
		if m.ClientPublicKeyASN1Hash == "" {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error client public key ASN1 hash must be set"))
		}
		if m.ClientPublicKeyASN1HashType == "" {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error client public key ASN1 hash type must be set"))
		}
		if m.ClientPublicKeyPEM == "" {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error client public key PEM must be set"))
		}
		if len(m.ClientCertificateASN1) == 0 {
			result = multierror.Append(result, gerror.NewErrValidationFailed("error client certificate ASN1 must be set"))
		}
	}
	return result.ErrorOrNil()
}
