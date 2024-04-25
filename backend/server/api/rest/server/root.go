package server

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

var rootDocumentPaths = map[string]func(ctx routes.RequestContext) string{
	"current_legal_entity_url":  routes.MakeCurrentLegalEntityLink,
	"legal_entities_url":        routes.MakeLegalEntitiesLink,
	"github_authentication_url": routes.MakeGitHubAuthenticationURL,
}

type RootAPI struct {
	*APIBase
}

func NewRootAPI(
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *RootAPI {

	return &RootAPI{
		APIBase: NewAPIBase(authorizationService, resourceLinker, logFactory("RootAPI")),
	}
}

func (a *RootAPI) GetRootDocument(w http.ResponseWriter, r *http.Request) {
	res := make(documents.GetRootDocumentResponse)
	for name, fn := range rootDocumentPaths {
		res[name] = fn(routes.RequestCtx(r))
	}
	a.JSON(w, r, res)
}
