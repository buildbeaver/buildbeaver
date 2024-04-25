package runner_test

import (
	"github.com/buildbeaver/buildbeaver/runner"
)

type Runner struct {
	Scheduler *runner.Scheduler
}

func NewRunner(jobScheduler *runner.Scheduler) *Runner {
	return &Runner{
		Scheduler: jobScheduler,
	}
}
