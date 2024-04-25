package documents

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

// Secret is used to represent a plaintext secret without its value
type Secret struct {
	baseResourceDocument
	// TODO: Include all model fields directly in this object
	*models.Secret
	// Name of the secret, unique within a repo.
	// NOTE this is the plaintext name of the secret, and it overrides the hashed secret name that we
	// really store and address the secret by in the backend.
	Name string `json:"name"`
}

// MakeSecret converts a models.SecretPlaintext and associated data into a Secret
func MakeSecret(rctx routes.RequestContext, secret *models.SecretPlaintext) *Secret {
	return &Secret{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeSecretLink(rctx, secret.ID),
		},
		Secret: secret.Secret,
		Name:   secret.Key,
	}
}

// MakeSecrets converts an array models.SecretPlaintext and associated data into an array of Secrets
func MakeSecrets(rctx routes.RequestContext, secrets []*models.SecretPlaintext) []*Secret {
	var docs []*Secret
	for _, secretPlaintext := range secrets {
		if secretPlaintext.IsInternal {
			continue
		}
		docs = append(docs, MakeSecret(rctx, secretPlaintext))
	}
	return docs
}

// CreateSecretRequest is used when creating a secret
type CreateSecretRequest struct {
	// Name is the name of the secret
	Name string `json:"name"`
	// Value is the value of the secret
	Value string `json:"value"`
}

func (d *CreateSecretRequest) Bind(r *http.Request) error {
	if d.Name == "" {
		return gerror.NewErrValidationFailed("Name must not be empty")
	}
	if !regexp.MustCompile(models.SecretNameRegexStr).MatchString(d.Name) {
		return gerror.NewErrValidationFailed(fmt.Sprintf("Secret name can only contain alphanumeric or underscore characters: '%s'", d.Name))
	}
	if len(d.Value) == 0 {
		return gerror.NewErrValidationFailed("Value must not be empty")
	}
	return nil
}

// PatchSecretRequest is used when updating a secret
type PatchSecretRequest struct {
	// Name is the name of the secret
	Name *string `json:"name"`
	// Value is the value of the secret
	Value *string `json:"value"`
}

func (d *PatchSecretRequest) Bind(r *http.Request) error {
	if d.Name == nil && d.Value == nil {
		return gerror.NewErrValidationFailed("At least one of name and value must be specified")
	}
	if d.Name != nil && *d.Name == "" {
		return gerror.NewErrValidationFailed("Name must not be empty")
	}
	if d.Name != nil && !regexp.MustCompile(models.SecretNameRegexStr).MatchString(*d.Name) {
		return gerror.NewErrValidationFailed(fmt.Sprintf("Secret name can only contain alphanumeric or underscore characters: '%s'", *d.Name))
	}
	if d.Value != nil && len(*d.Value) == 0 {
		return gerror.NewErrValidationFailed("Value must not be empty")
	}
	return nil
}
