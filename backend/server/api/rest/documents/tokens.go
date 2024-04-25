package documents

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type ExchangeTokenRequest struct {
	// The name of the SCM for which we are exchanging tokens (e.g. 'github')
	SCMName string `json:"scm_name"`

	// A token from an external system that can be used to authenticate through that system
	// (e.g. a GitHub personal access token)
	Token string `json:"token"`
}

func (e *ExchangeTokenRequest) Bind(r *http.Request) error {
	if e.Token == "" {
		return gerror.NewErrValidationFailed("Token must be specified")
	}
	return nil
}

// SharedSecretToken is a document containing a shared-secret token that can be used to authenticate to BuildBeaver.
// The secret is included in this document.
type SharedSecretToken struct {
	baseResourceDocument

	ID        models.CredentialID `json:"id"`
	CreatedAt models.Time         `json:"created_at"`

	Token string `json:"token"`
}

func MakeSharedSecretToken(rctx routes.RequestContext, token *models.PublicSharedSecretToken, credential *models.Credential) *SharedSecretToken {
	return &SharedSecretToken{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeCredentialLink(rctx, credential.ID),
		},

		ID:        credential.ID,
		CreatedAt: credential.CreatedAt,

		Token: token.String(),
	}
}

func (t *SharedSecretToken) GetID() models.ResourceID {
	return t.ID.ResourceID
}

func (t *SharedSecretToken) GetKind() models.ResourceKind {
	return models.CredentialResourceKind
}

func (t *SharedSecretToken) GetCreatedAt() models.Time {
	return t.CreatedAt
}
