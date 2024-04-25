package app

import (
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
)

type Server struct {
	LegalEntityService    services.LegalEntityService
	RunnerService         services.RunnerService
	SyncService           services.SyncService
	CoreAPIServer         *server.AppAPIServer
	RunnerAPIServer       *server.RunnerAPIServer
	InternalRunnerManager *InternalRunnerManager
}

func NewServer(
	legalEntityService services.LegalEntityService,
	runnerService services.RunnerService,
	syncService services.SyncService,
	coreAPIServer *server.AppAPIServer,
	runnerAPIServer *server.RunnerAPIServer,
	internalRunnerManager *InternalRunnerManager,
	allSCMs []scm.SCM, // tell Wire the app has a dependency on the SCMs, to ensure they're created
) *Server {
	return &Server{
		LegalEntityService:    legalEntityService,
		RunnerService:         runnerService,
		SyncService:           syncService,
		CoreAPIServer:         coreAPIServer,
		RunnerAPIServer:       runnerAPIServer,
		InternalRunnerManager: internalRunnerManager,
	}
}
