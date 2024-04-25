package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/hashicorp/go-multierror"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/runner/runtime"
)

type Config struct {
	runtime.Config
	ImageURI     string
	AuthOrNil    *Auth
	PullStrategy models.DockerPullStrategy
	ShellOrNil   *string
	Services     []RuntimeServiceConfig
}

type RuntimeServiceConfig struct {
	Name         string
	Aliases      []string
	ImageURI     string
	AuthOrNil    *Auth
	PullStrategy models.DockerPullStrategy
}

type runtimeImageConfig struct {
	OS runtime.OS
}

type runtimeContainerConfig struct {
	Name                string
	GuestShellPath      []string
	GuestWorkspaceDir   string
	GuestStagingDir     string
	GuestPID0ScriptPath string
	Binds               []string
}

// Runtime executes jobs inside a Docker container.
type Runtime struct {
	config           Config
	containerManager *ContainerManager
	serviceManager   *ServiceManager
	log              logger.Log
	state            struct {
		started         bool
		containerID     string
		imageConfig     runtimeImageConfig
		containerConfig runtimeContainerConfig
		serviceNetwork  Network
	}
}

func NewRuntime(config Config, client *client.Client, logFactory logger.LogFactory) *Runtime {
	sConfig := ServiceManagerConfig{
		RuntimeID:          config.RuntimeID,
		NetworkType:        ServiceNetworkTypePrivate,
		PrivateNetworkName: makeNetworkName(&config),
	}
	cManager := NewContainerManager(client, logFactory)
	return &Runtime{
		config:           config,
		containerManager: cManager,
		serviceManager:   NewServiceManager(sConfig, cManager, config.LogPipeline, logFactory),
		log:              logFactory("DockerRuntime"),
	}
}

// Start initializes the runtime and prepares it to have commands Exec'd inside it.
func (r *Runtime) Start(ctx context.Context) error {
	if r.state.started {
		return fmt.Errorf("error starting docker runtime: already started")
	}
	r.state.started = true
	network, err := r.serviceManager.Start(ctx)
	if err != nil {
		return err
	}
	r.state.serviceNetwork = *network
	pLog := r.config.LogPipeline.StructuredLogger().Wrap("job_docker_image", "Pulling Docker image...")
	pConfig := &ImagePullConfig{
		ImageURI:     r.config.ImageURI,
		Auth:         r.config.AuthOrNil,
		PullStrategy: r.config.PullStrategy,
	}
	err = r.containerManager.PullDockerImage(ctx, pLog, pConfig)
	if err != nil {
		return fmt.Errorf("error pulling Docker image: %w", err)
	}
	imageOS, err := r.containerManager.GetDockerImageOS(ctx, r.config.ImageURI)
	if err != nil {
		return fmt.Errorf("error discovering image OS: %w", err)
	}
	r.state.imageConfig.OS = imageOS
	config, err := r.prepareJobContainerConfig(ctx)
	if err != nil {
		return err
	}
	r.state.containerConfig = *config
	r.log.Infof("Guest OS: %s", r.state.imageConfig.OS)
	r.log.Infof("Guest shell: %#v", config.GuestShellPath)
	r.log.Infof("Guest Working dir: %s", config.GuestWorkspaceDir)
	r.log.Infof("Guest Staging dir: %s", config.GuestStagingDir)
	r.log.Infof("Binds: %#v", config.Binds)
	converter := r.config.LogPipeline.Converter()
	cConfig := ContainerConfig{
		Name:       makeContainerNameForJob(&r.config),
		ImageURI:   r.config.ImageURI,
		Entrypoint: config.GuestShellPath,
		Command:    []string{config.GuestPID0ScriptPath},
		WorkingDir: config.GuestWorkspaceDir,
		Binds:      config.Binds,
		Networks:   []string{network.NetworkID},
		Stdout:     converter,
		Stderr:     converter,
	}
	containerID, err := r.containerManager.StartContainer(ctx, cConfig)
	if err != nil {
		return err
	}
	r.state.containerID = containerID
	return nil
}

// Stop tears down the runtime.
func (r *Runtime) Stop(ctx context.Context) error {
	if !r.state.started {
		return fmt.Errorf("error stopping docker runtime: not started")
	}
	var results *multierror.Error
	if r.state.containerID != "" {
		err := r.containerManager.StopContainer(ctx, r.state.containerID)
		if err != nil {
			results = multierror.Append(results, fmt.Errorf("error stopping job container: %w", err))
		}
	}
	err := r.serviceManager.Stop(ctx)
	if err != nil {
		results = multierror.Append(results, fmt.Errorf("error stopping services: %w", err))
	}
	r.state.started = false
	return results.ErrorOrNil()
}

// Exec executes a command inside the runtime.
// Start must have been called before calling Exec.
func (r *Runtime) Exec(ctx context.Context, config runtime.ExecConfig) error {
	_, err := runtime.WriteScript(r.config.StagingDir, config.Name, config.Commands)
	if err != nil {
		return err
	}
	shell := runtime.ShellOrDefault(r.state.imageConfig.OS, r.config.ShellOrNil)
	containerScriptPath, _, err := r.mapHostPath(runtime.GetHostOS(), filepath.Join(r.config.StagingDir, config.Name))
	if err != nil {
		return err
	}
	execConfig := ExecConfig{
		ContainerID: r.state.containerID,
		Command:     []string{shell, containerScriptPath},
		WorkingDir:  r.state.containerConfig.GuestWorkspaceDir,
		Env:         r.fixEnv(config.Env),
		Stdout:      config.Stdout,
		Stderr:      config.Stderr,
	}
	return r.containerManager.Execute(ctx, execConfig)
}

// StartService starts a service inside the runtime.
// The service must be resolvable by name to commands run with Exec.
// Service names are unique within the runtime - it is an error to try start service with the same name twice.
func (r *Runtime) StartService(ctx context.Context, config runtime.ServiceConfig) error {
	sConfig := ServiceConfig{
		Name: config.Name,
		Env:  config.Env,
	}
	var found bool
	for _, service := range r.config.Services {
		if service.Name == config.Name {
			sConfig.AuthOrNil = service.AuthOrNil
			sConfig.ImageURI = service.ImageURI
			sConfig.PullStrategy = service.PullStrategy
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("error unable to find service config")
	}
	return r.serviceManager.StartService(ctx, sConfig)
}

// CleanUp removes any resources left over from previous commands that may not have finished cleanly
// (e.g. old containers, networks). Assumes no commands are currently running.
func (r *Runtime) CleanUp(ctx context.Context) error {
	if r.state.started {
		return fmt.Errorf("error performing docker runtime cleanup: runtime is currently started, must be stopped in order to clean up")
	}
	var results *multierror.Error

	err := r.containerManager.CleanUpContainers(ctx)
	if err != nil {
		results = multierror.Append(results, err)
	}

	err = r.containerManager.CleanUpNetworks(ctx)
	if err != nil {
		results = multierror.Append(results, err)
	}

	return results.ErrorOrNil()
}

func (r *Runtime) prepareJobContainerConfig(ctx context.Context) (*runtimeContainerConfig, error) {
	switch r.state.imageConfig.OS {
	case runtime.OSLinux:
		return r.prepareLinuxContainerConfig(ctx)
	case runtime.OSWindows:
		return r.prepareWindowsContainerConfig(ctx)
	default:
		return nil, fmt.Errorf("error unsupported image OS: %v", r.state.imageConfig.OS)
	}
}

func (r *Runtime) prepareWindowsContainerConfig(ctx context.Context) (*runtimeContainerConfig, error) {
	// TODO work with the configured shell
	scriptName := fmt.Sprintf("pid0")
	_, err := runtime.WriteScript(r.config.StagingDir, scriptName, []string{"timeout /t -1"})
	if err != nil {
		return nil, err
	}
	shellPath, err := runtime.ShellPath(runtime.ShellCMD)
	if err != nil {
		return nil, err
	}
	guestWorkingDir := "C:\\buildbeaver\\workspace"
	guestStagingDir := "C:\\buildbeaver\\staging"
	guestKeepAliveScriptPath := fmt.Sprintf("C:\\buildbeaver\\staging\\%s", scriptName)
	binds := []string{
		fmt.Sprintf("%s:%s:rw", r.config.WorkspaceDir, guestWorkingDir),
		fmt.Sprintf("%s:%s:ro", r.config.StagingDir, guestStagingDir),
		// Windows containers only run on Windows, so use the Windows pipe syntax
		"\\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine",
	}
	return &runtimeContainerConfig{
		Name:                util.EscapeFileName(r.config.RuntimeID),
		Binds:               binds,
		GuestShellPath:      []string{shellPath},
		GuestWorkspaceDir:   guestWorkingDir,
		GuestStagingDir:     guestStagingDir,
		GuestPID0ScriptPath: guestKeepAliveScriptPath,
	}, nil
}

func (r *Runtime) prepareLinuxContainerConfig(ctx context.Context) (*runtimeContainerConfig, error) {
	// TODO work with the configured shell
	scriptName := fmt.Sprintf("pid0")
	_, err := runtime.WriteScript(r.config.StagingDir, scriptName, []string{"while :; do sleep 2073600; done"})
	if err != nil {
		return nil, err
	}
	shellPath, err := runtime.ShellPath(runtime.ShellSH)
	if err != nil {
		return nil, err
	}
	guestWorkingDir := "/tmp/buildbeaver/workspace"
	guestStagingDir := "/tmp/buildbeaver/staging"
	guestKeepAliveScriptPath := fmt.Sprintf("/tmp/buildbeaver/staging/%s", scriptName)
	binds := []string{
		fmt.Sprintf("%s:%s:rw", r.config.WorkspaceDir, guestWorkingDir),
		fmt.Sprintf("%s:%s:ro", r.config.StagingDir, guestStagingDir),
		// Linux containers run natively on Linux, and in a Linux VM on Windows and macOS,
		// so we can always refer to the Linux socket path here
		"/var/run/docker.sock:/var/run/docker.sock",
	}
	return &runtimeContainerConfig{
		Name:                r.config.RuntimeID,
		Binds:               binds,
		GuestShellPath:      []string{shellPath},
		GuestWorkspaceDir:   guestWorkingDir,
		GuestStagingDir:     guestStagingDir,
		GuestPID0ScriptPath: guestKeepAliveScriptPath,
	}, nil
}

func (r *Runtime) fixEnv(env []string) []string {
	for i, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			path, changed, err := r.mapHostPath(runtime.GetHostOS(), parts[1])
			if err != nil {
				r.log.Warnf("Ignoring error mapping host path for %q: %v", parts[1], err)
			} else if changed {
				env[i] = fmt.Sprintf("%s=%s", parts[0], path)
			}
		}
	}
	return env
}

func (r *Runtime) mapHostPath(fromOS runtime.OS, path string) (string, bool, error) {
	if r.state.imageConfig.OS == "" {
		return "", false, fmt.Errorf("error runtime is not prepared")
	}
	var changed bool
	if strings.HasPrefix(path, r.config.WorkspaceDir) {
		path = strings.Replace(path, r.config.WorkspaceDir, r.state.containerConfig.GuestWorkspaceDir, 1)
		changed = true
	} else if strings.HasPrefix(path, r.config.StagingDir) {
		path = strings.Replace(path, r.config.StagingDir, r.state.containerConfig.GuestStagingDir, 1)
		changed = true
	}
	if changed {
		switch fromOS {
		case runtime.OSMacOS:
			// macOS can only run Linux containers.
			// macOS's paths are compatible with Linux.
		case runtime.OSLinux:
			// Linux can only run Linux containers.
			// Linux to Linux does not need a conversion.
		case runtime.OSWindows:
			// Windows can run Windows or Linux containers.
			switch r.state.imageConfig.OS {
			case runtime.OSLinux:
				// Windows to Linux needs path separator tweaking after we've swapped out the path above.
				path = strings.Replace(path, "\\", "/", -1)
			case runtime.OSWindows:
				// Windows to Windows does not need a conversion.
			default:
				return "", false, fmt.Errorf("error unsupported container OS: %v", fromOS)
			}
		default:
			return "", false, fmt.Errorf("error unsupported OS: %v", fromOS)
		}
	}
	return path, changed, nil
}
