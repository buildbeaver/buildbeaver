package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/runner/runtime"
)

// Auth defines what is sent to Docker when we authenticate for an image pull
type Auth struct {
	Basic *BasicAuth
	AWS   *AWSAuth
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AWSAuth struct {
	AWSRegion          string `json:"aws_region"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
}

type ImagePullConfig struct {
	ImageURI     string
	Auth         *Auth
	PullStrategy models.DockerPullStrategy
}

type ContainerConfig struct {
	Name       string
	ImageURI   string
	Entrypoint []string
	Command    []string
	WorkingDir string
	Env        []string
	Binds      []string
	// Networks is a list of network IDs the container should be connected to.
	// The container will always be connected to the host network by default.
	Networks []string
	// Aliases is a list of names this container will be resolvable on from
	// each of the configured networks (if any).
	Aliases []string
	Stdout  io.Writer
	Stderr  io.Writer
}

type ExecConfig struct {
	ContainerID string
	Command     []string
	WorkingDir  string
	Env         []string
	Stdout      io.Writer
	Stderr      io.Writer
}

type ContainerManager struct {
	client *client.Client
	log    logger.Log
}

func NewContainerManager(client *client.Client, logFactory logger.LogFactory) *ContainerManager {
	return &ContainerManager{client: client, log: logFactory("DockerContainerManager")}
}

// PullDockerImage pulls a Docker Image from a remote registry.
func (r *ContainerManager) PullDockerImage(ctx context.Context, log *logging.StructuredLogger, config *ImagePullConfig) error {
	image := parseDockerImageURI(config.ImageURI)
	fil := filters.NewArgs()
	fil.Add("reference", image.Reference())
	list, err := r.client.ImageList(ctx, types.ImageListOptions{
		All:     false,
		Filters: fil,
	})
	if err != nil {
		return fmt.Errorf("error listing images: %w", err)
	}

	alreadyExists := len(list) > 0
	if config.PullStrategy == models.DockerPullStrategyNever {
		log.WriteLinef("Docker pull strategy is %q; %q will not be pulled",
			models.DockerPullStrategyNever, image.FQN())
		return nil
	}
	if config.PullStrategy == models.DockerPullStrategyIfNotExists && alreadyExists {
		log.WriteLinef("Docker pull strategy is %q and image exists in cache; %q will not be pulled",
			models.DockerPullStrategyIfNotExists, image.FQN())
		return nil
	}
	if alreadyExists && !image.IsLatest() && config.PullStrategy == models.DockerPullStrategyDefault {
		log.WriteLinef("Docker pull strategy is %q, image exists in cache and is not latest; %q will not be pulled",
			models.DockerPullStrategyDefault, image.FQN())
		return nil
	}

	log.WriteLinef("Pulling image: %s", image)

	// If authentication has been provided then pass it into the image pull
	imagePullOptions := types.ImagePullOptions{}
	if config.Auth.Basic != nil {
		log.WriteLinef("Using Docker registry auth: Basic")
		jsonBytes, err := json.Marshal(config.Auth.Basic)
		if err != nil {
			return fmt.Errorf("error encoding docker auth: %w", err)
		}
		imagePullOptions.RegistryAuth = base64.StdEncoding.EncodeToString(jsonBytes)
	} else if config.Auth.AWS != nil {
		log.WriteLinef("Using Docker registry auth: AWS")
		cfg := &aws.Config{}
		if config.Auth.AWS.AWSRegion != "" {
			cfg = cfg.WithRegion(config.Auth.AWS.AWSRegion)
		}
		cfg = cfg.WithCredentials(credentials.NewStaticCredentials(config.Auth.AWS.AWSAccessKeyID, config.Auth.AWS.AWSSecretAccessKey, ""))
		sess, err := session.NewSession(cfg)
		if err != nil {
			return fmt.Errorf("error creating AWS session: %w", err)
		}
		svc := ecr.New(sess)
		token, err := svc.GetAuthorizationTokenWithContext(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return fmt.Errorf("error getting AWS ECR authorization token: %w", err)
		}
		if len(token.AuthorizationData) == 0 {
			return fmt.Errorf("error unexpected AWS ECR token format")
		}
		authData := token.AuthorizationData[0].AuthorizationToken
		data, err := base64.StdEncoding.DecodeString(*authData)
		if err != nil {
			return fmt.Errorf("error decoding AWS ECR token: %w", err)
		}
		parts := strings.SplitN(string(data), ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("error unexpected AWS ECR token data format")
		}
		basic := &BasicAuth{
			Username: "AWS",
			Password: parts[1],
		}
		jsonBytes, err := json.Marshal(basic)
		if err != nil {
			return fmt.Errorf("error encoding docker auth: %w", err)
		}
		imagePullOptions.RegistryAuth = base64.StdEncoding.EncodeToString(jsonBytes)
	} else {
		log.WriteLinef("Using Docker registry auth: None")
	}

	// TODO this error needs to go to the job log
	stream, err := r.client.ImagePull(ctx, image.FQN(), imagePullOptions)
	if err != nil {
		return errors.Wrap(err, "error pulling image")
	}
	defer stream.Close()

	// TODO can output image pull info to the build logs here:
	// 	Do a list on the image to discover its size
	//  Intercept stream and output some progress information as it's being read
	//  Make this a generic util and use it for other pulls
	res, err := r.client.ImageLoad(ctx, stream, false)
	if err != nil {
		return errors.Wrap(err, "error loading image")
	}
	defer res.Body.Close()
	return nil
}

// GetDockerImageOS returns the type of underlying guest OS the specified Docker image
// is made from. The docker image must have been pulled first.
func (r *ContainerManager) GetDockerImageOS(ctx context.Context, imageURI string) (runtime.OS, error) {
	image := parseDockerImageURI(imageURI)
	inspect, _, err := r.client.ImageInspectWithRaw(ctx, image.String())
	if err != nil {
		return "", fmt.Errorf("error inspecting image %q: %w", image, err)
	}
	return runtime.OS(inspect.Os), nil
}

// StartContainer starts a new container in the background and returns its unique ID.
// Call StopContainer to stop it and free up resources.
func (r *ContainerManager) StartContainer(ctx context.Context, config ContainerConfig) (string, error) {
	image := parseDockerImageURI(config.ImageURI)
	cConfig := &container.Config{
		Image:      image.FQN(),
		Entrypoint: config.Entrypoint,
		Cmd:        config.Command,
		WorkingDir: config.WorkingDir,
		Env:        config.Env,
		// User:       "1000:1000", // TODO https://medium.com/redbubble/running-a-docker-container-as-a-non-root-user-7d2e00f8ee15
	}
	hConfig := &container.HostConfig{
		AutoRemove: false,
		Binds:      config.Binds,
	}
	nConfig := &network.NetworkingConfig{}
	res, err := r.client.ContainerCreate(ctx, cConfig, hConfig, nConfig, nil, config.Name) // platform is optional
	if err != nil {
		return "", errors.Wrap(err, "error creating container")
	}
	for _, networkID := range config.Networks {
		nConfig := &network.EndpointSettings{Aliases: config.Aliases}
		err = r.client.NetworkConnect(ctx, networkID, res.ID, nConfig)
		if err != nil {
			return "", fmt.Errorf("error connecting container to network: %w", err)
		}
	}
	err = r.client.ContainerStart(ctx, res.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", errors.Wrap(err, "error starting container")
	}
	if config.Stdout != nil || config.Stderr != nil {
		opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Timestamps: false}
		reader, err := r.client.ContainerLogs(ctx, res.ID, opts)
		if err != nil {
			return "", fmt.Errorf("error connecting to container log stream: %w", err)
		}
		r.pipeContainerLogAsync(ctx, reader, config.Stdout, config.Stderr)
	}
	return res.ID, nil
}

// StopContainer stops and removes a previously started docker container.
func (r *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	var results *multierror.Error
	err := r.client.ContainerKill(ctx, containerID, "kill")
	if err != nil && !errdefs.IsNotFound(err) {
		results = multierror.Append(results, fmt.Errorf("error killing container %q: %w", containerID, err))
	}
	err = r.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		results = multierror.Append(results, fmt.Errorf("error removing container %q: %w", containerID, err))
	}
	return results.ErrorOrNil()
}

// CleanUpContainers looks for existing BuildBeaver containers and removes them.
func (r *ContainerManager) CleanUpContainers(ctx context.Context) error {
	r.log.Infof("Cleaning up docker containers...")

	containers, err := r.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true, // include containers that are not currently running
		Limit:   0,
		Filters: filters.Args{},
	})
	if err != nil {
		return fmt.Errorf("error listing docker containers: %w", err)
	}

	var results *multierror.Error
	for _, container := range containers {
		if containerIsBuildBeaver(container) {
			r.log.Infof("Deleting container '%s' with ID '%s' during cleanup", container.Names[0], container.ID)
			err := r.StopContainer(ctx, container.ID)
			if err != nil {
				results = multierror.Append(results, err)
			}
		}
	}
	return results.ErrorOrNil()
}

// Execute a command inside the container.
// StartContainer must have previously been called.
func (r *ContainerManager) Execute(ctx context.Context, config ExecConfig) error {
	eConfig := types.ExecConfig{
		Cmd:          config.Command,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
		Detach:       false,
		AttachStderr: true,
		AttachStdout: true,
	}
	createRes, err := r.client.ContainerExecCreate(ctx, config.ContainerID, eConfig)
	if err != nil {
		return fmt.Errorf("error creating script exec: %w", err)
	}
	resp, err := r.client.ContainerExecAttach(ctx, createRes.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("error attaching script exec: %w", err)
	}
	defer resp.Close()
	if config.Stdout != nil || config.Stderr != nil {
		err = r.pipeContainerLog(ctx, resp.Reader, config.Stdout, config.Stderr)
		if err != nil {
			return fmt.Errorf("error piping container log: %w", err)
		}
	}
	var exitCode int
	for {
		res, err := r.client.ContainerExecInspect(ctx, createRes.ID)
		if err != nil {
			return fmt.Errorf("error inspecting script exec: %w", err)
		}
		if res.Running {
			time.Sleep(time.Second)
			continue
		}
		exitCode = res.ExitCode
		break
	}
	if exitCode != 0 {
		return fmt.Errorf("error script exited with non-zero exit code: %d", exitCode)
	}
	return nil
}

// CreateNetwork creates a new private network and returns its ID.
func (r *ContainerManager) CreateNetwork(ctx context.Context, name string) (string, error) {
	res, err := r.client.NetworkCreate(ctx, name, types.NetworkCreate{})
	if err != nil {
		return "", fmt.Errorf("error creating network: %w", err)
	}
	return res.ID, nil
}

// DeleteNetwork deletes a previously created network.
func (r *ContainerManager) DeleteNetwork(ctx context.Context, networkID string) error {
	err := r.client.NetworkRemove(ctx, networkID)
	if err != nil {
		return errors.Wrap(err, "error removing network")
	}
	return nil
}

// CleanUpNetworks looks for existing BuildBeaver networks and removes them.
func (r *ContainerManager) CleanUpNetworks(ctx context.Context) error {
	r.log.Infof("Cleaning up docker networks...")

	networks, err := r.client.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.Args{},
	})
	if err != nil {
		return fmt.Errorf("error listing docker networks: %w", err)
	}

	var results *multierror.Error
	for _, network := range networks {
		if networkIsBuildBeaver(network) {
			r.log.Infof("Deleting network '%s' with ID '%s' during cleanup", network.Name, network.ID)
			err := r.DeleteNetwork(ctx, network.ID)
			if err != nil {
				results = multierror.Append(results, err)
			}
		}
	}
	return results.ErrorOrNil()
}

func (r *ContainerManager) pipeContainerLog(ctx context.Context, from io.Reader, stdout io.Writer, stderr io.Writer) error {
	// https://github.com/docker/cli/blob/master/cli/command/container/logs.go
	// https://github.com/docker/cli/blob/ebca1413117a3fcb81c89d6be226dcec74e5289f/vendor/github.com/docker/docker/pkg/stdcopy/stdcopy.go#L94
	_, err := stdcopy.StdCopy(stdout, stderr, from)
	if err != nil && err != io.EOF && err != io.ErrClosedPipe {
		return err
	}
	return nil
}

func (r *ContainerManager) pipeContainerLogAsync(ctx context.Context, from io.Reader, stdout io.Writer, stderr io.Writer) <-chan struct{} {
	doneC := make(chan struct{})
	go func() {
		defer close(doneC)
		err := r.pipeContainerLog(ctx, from, stdout, stderr)
		if err != nil {
			r.log.Warnf("Ignoring error piping container logs; Logs may be incomplete: %s", err)
		}
	}()
	return doneC
}

// containerIsBuildBeaver returns true if the specified container was created by BuildBeaver.
// Can be used to identify which containers to clean up and which to leave alone.
func containerIsBuildBeaver(container types.Container) bool {
	for _, name := range container.Names {
		// Docker container names come back with a slash on the front
		trimmedName := strings.TrimPrefix(name, "/")
		if isContainerNameForJob(trimmedName) || isContainerNameForService(trimmedName) {
			return true
		}
	}
	return false
}

// networkIsBuildBeaver returns true if the specified network was created by BuildBeaver.
// Can be used to identify which networks to clean up and which to leave alone.
func networkIsBuildBeaver(network types.NetworkResource) bool {
	return isNetworkName(network.Name)
}
