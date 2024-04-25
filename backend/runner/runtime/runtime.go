package runtime

import (
	"context"
	"io"

	"github.com/buildbeaver/buildbeaver/runner/logging"
)

// Config is the base runtime configuration.
type Config struct {
	// RuntimeID uniquely identifies an instance of a runtime.
	RuntimeID string
	// StagingDir is a directory on the local filesystem where job related files
	// will be written by the runner.
	StagingDir string
	// WorkspaceDir is a directory on the local filesystem where a job's source
	// code will be checked out to and the job's steps will execute with as their
	// working directory.
	WorkspaceDir string
	// LogPipeline is the log pipeline the runtime should log to.
	LogPipeline logging.LogPipeline
}

// ServiceConfig describes a service that will execute inside a runtime.
type ServiceConfig struct {
	// Name is the DNS name the service will be resolvable via inside the runtime.
	Name string
	// Env is the environment in the form name=value to expose to the service.
	Env []string
}

// ExecConfig describes a command that will execute inside a runtime.
type ExecConfig struct {
	// Name is a human-readable name that uniquely identifies the command.
	Name string
	// Commands are the one or more shell commands to execute.
	Commands []string
	// Env is the environment in the form name=value to expose to the commands.
	Env []string
	// Stdout is optional. If supplied the command(s) stdout will be written to it.
	Stdout io.Writer
	// Stdout is optional. If supplied the command(s) stderr will be written to it.
	Stderr io.Writer
}

// Runtime is an execution environment for steps.
type Runtime interface {
	// Start initializes the runtime and prepares it to have commands Exec'd inside it.
	Start(ctx context.Context) error
	// StartService starts a service inside the runtime.
	// The service must be resolvable by name to commands run with Exec.
	// Service names are unique within the runtime - it is an error to try start service with the same name twice.
	StartService(ctx context.Context, config ServiceConfig) error
	// Exec executes a command inside the runtime.
	// Start must have been called before calling Exec.
	Exec(ctx context.Context, config ExecConfig) error
	// Stop tears down the runtime, freeing up any and all resources (including e.g. job container,
	// service containers, networks etc.)
	// ctx is a context with a timeout suitable for use in cleanup tasks. This context will *not* time out
	// when the job times out, to ensure we still clean up timed-out jobs.
	Stop(ctx context.Context) error
	// CleanUp removes any resources left over from previous commands that may not have finished cleanly
	// (e.g. old containers, networks). Assumes no commands are currently running.
	CleanUp(ctx context.Context) error
}
