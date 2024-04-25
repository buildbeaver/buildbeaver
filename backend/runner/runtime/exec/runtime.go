package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/buildbeaver/buildbeaver/runner/runtime"
)

type Config struct {
	runtime.Config
	ShellOrNil *string
}

// Runtime executes jobs directly on the host machine.
type Runtime struct {
	config Config
}

func NewRuntime(config Config) *Runtime {
	return &Runtime{config: config}
}

// Start initializes the runtime and prepares it to have commands Exec'd inside it.
func (r *Runtime) Start(ctx context.Context) error {
	return nil
}

// Stop tears down the runtime.
func (r *Runtime) Stop(ctx context.Context) error {
	return nil
}

// Exec executes a command inside the runtime.
// Start must have been called before calling Exec.
func (r *Runtime) Exec(ctx context.Context, config runtime.ExecConfig) error {
	hostOS := runtime.GetHostOS()

	scriptName := config.Name
	if hostOS == runtime.OSWindows {
		// Windows cmd.exe requires scripts to end in ".bat", or they won't be executed
		scriptName += ".bat"
	}

	scriptPath, err := runtime.WriteScript(r.config.StagingDir, scriptName, config.Commands)
	if err != nil {
		return err
	}
	shell := runtime.ShellOrDefault(hostOS, r.config.ShellOrNil)

	var cmd *exec.Cmd
	if hostOS == runtime.OSWindows {
		// Windows cmd.exe requires the /C option to run commands, as well as some other recommended options.
		// NOTE that "/C" must be the last option, immediately before the actual command.
		cmd = exec.CommandContext(ctx, shell, "/D", "/E:ON", "/V:OFF", "/S", "/C", scriptPath)
	} else {
		cmd = exec.CommandContext(ctx, shell, scriptPath)
	}

	cmd.Dir = r.config.WorkspaceDir
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr

	// Keep the existing PATH env variable so that commands can still be found and run.
	// Do not keep all env variables since secrets are supplied in env vars when using the command-line tool.
	pathEnv := os.Getenv("PATH")
	cmd.Env = append(config.Env, "PATH="+pathEnv)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running command: %w", err)
	}
	return nil
}

// StartService starts a service inside the runtime.
// The service must be resolvable by name to commands run with Exec.
// Service names are unique within the runtime - it is an error to try start service with the same name twice.
func (r *Runtime) StartService(ctx context.Context, config runtime.ServiceConfig) error {
	return fmt.Errorf("services are not supported with exec jobs")
}

// CleanUp removes any resources left over from previous commands that may not have finished cleanly.
func (r *Runtime) CleanUp(ctx context.Context) error {
	// For Exec runtimes there are no services and commands run inline, so there's nothing to do.
	return nil
}
