package server

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type QueueAPI struct {
	queueService  services.QueueService
	runnerService services.RunnerService
	*APIBase
}

func NewQueueAPI(
	queueService services.QueueService,
	runnerService services.RunnerService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *QueueAPI {
	return &QueueAPI{
		queueService:  queueService,
		runnerService: runnerService,
		APIBase:       NewAPIBase(authorizationService, resourceLinker, logFactory("QueueAPI")),
	}
}

func (a *QueueAPI) Dequeue(w http.ResponseWriter, r *http.Request) {
	meta := a.MustAuthenticationMeta(r)
	// Read the currently authenticated runner
	runner, err := a.runnerService.ReadByIdentityID(r.Context(), nil, meta.IdentityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	job, err := a.queueService.Dequeue(r.Context(), runner.ID)
	if err != nil {
		if gerror.IsNotFound(err) || gerror.IsRunnerDisabled(err) {
			// Do not log 'not found' or 'Runner Disabled' errors as warnings - these are normal states when there's
			// either nothing in the queue or the runner has been disabled by the user.
			a.ErrorNotLogged(w, r, err)
		} else {
			a.Error(w, r, err)
		}
		return
	}
	res := documents.MakeRunnableJob(routes.RequestCtx(r), job)
	a.GotResource(w, r, res)
}

// Ping acts as a pre-flight check for a runner, checking that authentication and registration are
// in place ready to dequeue build jobs. The currently authenticated identity must be a runner.
func (a *QueueAPI) Ping(w http.ResponseWriter, r *http.Request) {
	meta := a.MustAuthenticationMeta(r)

	// Read the runner associated with the currently authenticated identity
	_, err := a.runnerService.ReadByIdentityID(r.Context(), nil, meta.IdentityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
