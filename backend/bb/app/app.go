package app

import (
	"github.com/buildbeaver/buildbeaver/bb/bb_server"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/local_backend"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type App struct {
	Backend         *local_backend.LocalBackend
	JobScheduler    *runner.Scheduler
	ExecutorFactory runner.ExecutorFactory
	APIServer       *bb_server.BBAPIServer
	LogFactory      logger.LogFactory
	LogService      services.LogService
}
