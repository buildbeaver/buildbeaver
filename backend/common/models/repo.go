package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	// RepoSSHKeySecretName is the name of the secret that will
	// be created against each repo that contains the repo's private
	// SSH key. This is an internal secret.
	// The repo private SSH key is known as a 'deploy key' on GitHub.
	RepoSSHKeySecretName              = "repo_ssh_key"
	RepoResourceKind     ResourceKind = "repo"
)

type RepoID struct {
	ResourceID
}

func NewRepoID() RepoID {
	return RepoID{ResourceID: NewResourceID(RepoResourceKind)}
}

func RepoIDFromResourceID(id ResourceID) RepoID {
	return RepoID{ResourceID: id}
}

type RepoMetadata struct {
	ID        RepoID `json:"id" goqu:"skipupdate" db:"repo_id"`
	CreatedAt Time   `json:"created_at" goqu:"skipupdate" db:"repo_created_at"`
	UpdatedAt Time   `json:"updated_at" db:"repo_updated_at"`
	DeletedAt *Time  `json:"deleted_at,omitempty" db:"repo_deleted_at"`
	ETag      ETag   `json:"etag" db:"repo_etag" hash:"ignore"`
}

type Repo struct {
	RepoMetadata
	LegalEntityID    LegalEntityID       `json:"legal_entity_id" db:"repo_legal_entity_id"`
	Name             ResourceName        `json:"name" db:"repo_name"`
	Description      string              `json:"description" db:"repo_description"`
	SSHURL           string              `json:"ssh_url" db:"repo_ssh_url"`
	HTTPURL          string              `json:"http_url" db:"repo_http_url"`
	Link             string              `json:"link" db:"repo_link"`
	DefaultBranch    string              `json:"default_branch" db:"repo_default_branch"`
	Private          bool                `json:"private" db:"repo_private"`
	Enabled          bool                `json:"enabled" db:"repo_enabled"`
	SSHKeySecretID   *SecretID           `json:"ssh_key_secret_id" db:"repo_ssh_key_secret_id"`
	ExternalID       *ExternalResourceID `json:"external_id" db:"repo_external_id"`
	ExternalMetadata string              `json:"external_metadata" db:"repo_external_metadata"`
}

func NewRepo(
	now Time,
	name ResourceName,
	legalEntityID LegalEntityID,
	description string,
	sshURL string,
	httpURL string,
	link string,
	defaultBranch string,
	private bool,
	enabled bool,
	sshKeySecretID *SecretID,
	externalID *ExternalResourceID,
	externalMetadata string,
) *Repo {
	return &Repo{
		RepoMetadata: RepoMetadata{
			ID:        NewRepoID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		LegalEntityID:    legalEntityID,
		Name:             name,
		Description:      description,
		SSHURL:           sshURL,
		HTTPURL:          httpURL,
		Link:             link,
		DefaultBranch:    defaultBranch,
		Private:          private,
		Enabled:          enabled,
		SSHKeySecretID:   sshKeySecretID,
		ExternalID:       externalID,
		ExternalMetadata: externalMetadata,
	}
}

func (m *Repo) GetKind() ResourceKind {
	return RepoResourceKind
}

func (m *Repo) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Repo) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Repo) GetParentID() ResourceID {
	return m.LegalEntityID.ResourceID
}

func (m *Repo) GetName() ResourceName {
	return m.Name
}

func (m *Repo) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Repo) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Repo) GetETag() ETag {
	return m.ETag
}

func (m *Repo) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Repo) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *Repo) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *Repo) IsUnreachable() bool {
	// Repos are unreachable after they are soft-deleted
	return m.DeletedAt != nil
}

func (m *Repo) Validate() error {
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
	if !m.LegalEntityID.Valid() {
		result = multierror.Append(result, errors.New("error legal entity id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.SSHURL == "" && m.HTTPURL == "" {
		result = multierror.Append(result, errors.New("error at least one of SSH url and HTTP url must be set"))
	}
	if m.DefaultBranch == "" {
		result = multierror.Append(result, errors.New("error default branch must be set"))
	}
	if m.SSHKeySecretID != nil && !m.SSHKeySecretID.Valid() {
		result = multierror.Append(result, errors.New("error SSH key secret ID must be valid if set"))
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
