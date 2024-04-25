package server

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/go-chi/chi/v5"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
)

type WebhookAPI struct {
	scmRegistry *scm.SCMRegistry
	*APIBase
}

func NewWebhooksAPI(
	scmRegistry *scm.SCMRegistry,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *WebhookAPI {

	return &WebhookAPI{
		scmRegistry: scmRegistry,
		APIBase:     NewAPIBase(authorizationService, resourceLinker, logFactory("WebhooksAPI")),
	}
}

func (a *WebhookAPI) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	scmName := models.SystemName(chi.URLParam(r, "scm"))
	scm, err := a.scmRegistry.Get(scmName)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	handler, err := scm.WebhookHandler()
	if err != nil {
		a.Error(w, r, err)
		return
	}
	handler(w, r)
}
