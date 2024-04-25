package server

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type SecretAPI struct {
	secretService services.SecretService
	*APIBase
}

func NewSecretAPI(
	secretService services.SecretService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *SecretAPI {
	return &SecretAPI{
		secretService: secretService,
		APIBase:       NewAPIBase(authorizationService, resourceLinker, logFactory("SecretAPI")),
	}
}

func (a *SecretAPI) Get(w http.ResponseWriter, r *http.Request) {
	secretID, err := a.AuthorizedSecretID(r, models.SecretReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	secret, err := a.secretService.Read(r.Context(), nil, secretID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	plaintext, err := a.secretService.SecretToSecretPlaintext(r.Context(), secret)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeSecret(routes.RequestCtx(r), plaintext)
	a.CreatedResource(w, r, res, nil)
}

func (a *SecretAPI) Create(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.AuthorizedRepoID(r, models.SecretCreateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.CreateSecretRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	secret, err := a.secretService.Create(r.Context(), nil, repoID, req.Name, req.Value, false)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeSecret(routes.RequestCtx(r), secret)
	a.CreatedResource(w, r, res, nil)
}

func (a *SecretAPI) Patch(w http.ResponseWriter, r *http.Request) {
	secretID, err := a.AuthorizedSecretID(r, models.SecretUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.PatchSecretRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	secret, err := a.secretService.Read(r.Context(), nil, secretID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	if secret.IsInternal {
		a.Errorf("Refusing attempt to update an internal secret %s", secret.ID)
		w.WriteHeader(http.StatusNotFound)
	}
	plaintext, err := a.secretService.UpdatePlaintext(r.Context(), nil, secretID, dto.UpdateSecretPlaintext{
		KeyPlaintext:   req.Name,
		ValuePlaintext: req.Value,
		ETag:           a.GetIfMatch(r),
	})
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeSecret(routes.RequestCtx(r), plaintext)
	a.UpdatedResource(w, r, res, nil)
}

func (a *SecretAPI) Delete(w http.ResponseWriter, r *http.Request) {
	secretID, err := a.AuthorizedSecretID(r, models.SecretDeleteOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	secret, err := a.secretService.Read(r.Context(), nil, secretID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	if secret.IsInternal {
		a.Errorf("Refusing attempt to delete an internal secret %s", secret.ID)
		w.WriteHeader(http.StatusNotFound)
	}
	err = a.secretService.Delete(r.Context(), nil, secret.ID) // TODO ETag support
	if err != nil {
		a.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// List returns a list of secrets for a repo. The secrets returned do not
// contain any values, only the resource_links and associated data to be able to update them.
func (a *SecretAPI) List(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.AuthorizedRepoID(r, models.SecretReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// TODO support search/pagination
	pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
	secrets, cursor, err := a.secretService.ListPlaintextByRepoID(r.Context(), nil, repoID, pagination)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	docs := documents.MakeSecrets(routes.RequestCtx(r), secrets)
	res := documents.NewPaginatedResponse(models.SecretResourceKind, routes.MakeSecretsLink(routes.RequestCtx(r), repoID), nil, docs, cursor)
	a.JSON(w, r, res)
}

// ListPlainText returns a list of secrets in plaintext for a repo.
func (a *SecretAPI) ListPlainText(w http.ResponseWriter, r *http.Request) {
	meta := a.MustAuthenticationMeta(r)
	if meta.CredentialType != models.CredentialTypeClientCertificate {
		panic("Expected runner to authenticate with client certificate")
	}
	repoID, err := a.AuthorizedRepoID(r, models.SecretReadPlaintextOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// TODO support search/pagination
	pagination := models.NewPagination(models.DefaultPaginationLimit, nil)
	secrets, cursor, err := a.secretService.ListPlaintextByRepoID(r.Context(), nil, repoID, pagination)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// TODO convert to docs
	res := documents.NewPaginatedResponse(models.SecretResourceKind, routes.MakeSecretsLink(routes.RequestCtx(r), repoID), nil, secrets, cursor)
	a.JSON(w, r, res)
}
