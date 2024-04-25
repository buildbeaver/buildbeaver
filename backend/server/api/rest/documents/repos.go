package documents

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Repo struct {
	baseResourceDocument

	ID        models.RepoID `json:"id"`
	CreatedAt models.Time   `json:"created_at"`
	UpdatedAt models.Time   `json:"updated_at"`
	DeletedAt *models.Time  `json:"deleted_at,omitempty"`
	ETag      models.ETag   `json:"etag" hash:"ignore"`

	Name             models.ResourceName        `json:"name"`
	Description      string                     `json:"description"`
	LegalEntityID    models.LegalEntityID       `json:"legal_entity_id"`
	SSHURL           string                     `json:"ssh_url"`
	HTTPURL          string                     `json:"http_url"`
	Link             string                     `json:"link"`
	DefaultBranch    string                     `json:"default_branch"`
	Private          bool                       `json:"private"`
	Enabled          bool                       `json:"enabled"`
	SSHKeySecretID   *models.SecretID           `json:"ssh_key_secret_id"`
	ExternalID       *models.ExternalResourceID `json:"external_id"`
	ExternalMetadata string                     `json:"external_metadata"`

	BuildsURL      string `json:"builds_url"`
	BuildSearchURL string `json:"build_search_url"`
	SecretsURL     string `json:"secrets_url"`
}

func MakeRepo(rctx routes.RequestContext, repo *models.Repo) *Repo {
	return &Repo{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeRepoLink(rctx, repo.ID),
		},

		ID:        repo.ID,
		CreatedAt: repo.CreatedAt,
		UpdatedAt: repo.UpdatedAt,
		DeletedAt: repo.DeletedAt,
		ETag:      repo.ETag,

		Name:             repo.Name,
		Description:      repo.Description,
		LegalEntityID:    repo.LegalEntityID,
		SSHURL:           repo.SSHURL,
		HTTPURL:          repo.HTTPURL,
		Link:             repo.Link,
		DefaultBranch:    repo.DefaultBranch,
		Private:          repo.Private,
		Enabled:          repo.Enabled,
		SSHKeySecretID:   repo.SSHKeySecretID,
		ExternalID:       repo.ExternalID,
		ExternalMetadata: repo.ExternalMetadata,

		BuildsURL:      routes.MakeBuildsLink(rctx, repo.ID),
		BuildSearchURL: routes.MakeBuildSearchLink(rctx, repo.ID),
		SecretsURL:     routes.MakeSecretsLink(rctx, repo.ID),
	}
}

func MakeRepos(rctx routes.RequestContext, repos []*models.Repo) []*Repo {
	var docs []*Repo
	for _, model := range repos {
		docs = append(docs, MakeRepo(rctx, model))
	}
	return docs
}

func (d *Repo) GetID() models.ResourceID {
	return d.ID.ResourceID
}

func (d *Repo) GetKind() models.ResourceKind {
	return models.RepoResourceKind
}

func (d *Repo) GetCreatedAt() models.Time {
	return d.CreatedAt
}

type PatchRepoRequest struct {
	Enabled *bool `json:"enabled"`
}

func (d *PatchRepoRequest) Bind(r *http.Request) error {
	if d.Enabled == nil {
		return gerror.NewErrValidationFailed("Enabled must be specified")
	}
	return nil
}
