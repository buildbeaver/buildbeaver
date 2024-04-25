package app

import (
	"context"
	"time"

	"github.com/buildbeaver/buildbeaver/runner"
)

type Runner struct {
	config          *RunnerConfig
	registrar       *runner.Registrar
	jobScheduler    *runner.Scheduler
	executorFactory runner.ExecutorFactory
}

func NewRunner(
	config *RunnerConfig,
	registrar *runner.Registrar,
	jobScheduler *runner.Scheduler,
	executorFactory runner.ExecutorFactory,
) *Runner {
	return &Runner{
		config:          config,
		registrar:       registrar,
		jobScheduler:    jobScheduler,
		executorFactory: executorFactory,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	err := r.registrar.Register(ctx, r.config.RunnerCertificateFile, r.config.LogUnregisteredCert)
	if err != nil {
		return err
	}
	r.jobScheduler.Start()
	return nil
}

func (r *Runner) Stop() {
	r.jobScheduler.Stop()
}

// CleanUpOldResources cleans up containers and other resources left over from previous instances of the runner.
func (r *Runner) CleanUpOldResources() error {
	timeout := time.Second * 30

	// Make an executor that knows about supported runtimes to clean up
	executor := r.executorFactory(context.Background())

	return executor.CleanUp(timeout)
}
