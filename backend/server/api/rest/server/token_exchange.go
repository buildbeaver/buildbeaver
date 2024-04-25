package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"golang.org/x/oauth2"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	github_service "github.com/buildbeaver/buildbeaver/server/services/scm/github"
)

type TokenExchangeAPI struct {
	credentialService services.CredentialService
	syncService       services.SyncService
	*APIBase
}

func NewTokenExchangeAPI(
	credentialService services.CredentialService,
	syncService services.SyncService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory,
) *TokenExchangeAPI {
	return &TokenExchangeAPI{
		credentialService: credentialService,
		syncService:       syncService,
		APIBase:           NewAPIBase(authorizationService, resourceLinker, logFactory("TokenExchangeAPI")),
	}
}

func (a *TokenExchangeAPI) Exchange(w http.ResponseWriter, r *http.Request) {
	req := &documents.ExchangeTokenRequest{}
	err := render.Bind(r, req)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error reading ExchangeTokenRequest from request: %w", err))
		return
	}

	// Create a GitHub SCM authentication object based on the supplied token (typically a GitHub Personal Access Token)
	// We only currently support exchanging GitHub tokens
	// TODO: Consider supporting any SCM here by adding and using a suitable 'SCM BasicAuth factory' method on the SCM interface
	if req.SCMName != "github" {
		a.Error(w, r, gerror.NewErrValidationFailed(fmt.Sprintf("Unknown SCM Name: '%s' - only 'github' currently supported", req.SCMName)))
		return
	}
	oAuthToken := &oauth2.Token{AccessToken: req.Token}
	scmAuth := &github_service.GitHubSCMAuthentication{
		Token: oAuthToken,
	}

	// Find or create user data (legal entity and identity) to match the SCM user.
	// Authenticate to the SCM (e.g. GitHub) using the token.
	identity, err := a.syncService.SyncAuthenticatedUser(r.Context(), scmAuth)
	if err != nil {
		a.Error(w, r, gerror.NewErrUnauthorized("error authenticating to GitHub").Wrap(err))
		return
	}

	// Create a new secret for the identity
	token, credential, err := a.credentialService.CreateSharedSecretCredential(r.Context(), nil, identity.ID, true)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error creating shared secret credential: %w", err))
		return
	}

	doc := documents.MakeSharedSecretToken(routes.RequestCtx(r), &token, credential)
	a.CreatedResource(w, r, doc, nil)
}
