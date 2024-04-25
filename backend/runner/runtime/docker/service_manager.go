package docker

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/logging"
)

type ServiceNetworkType string

const (
	ServiceNetworkTypePrivate ServiceNetworkType = "private"
	ServiceNetworkTypeHost    ServiceNetworkType = "host"
)

type Network struct {
	NetworkType ServiceNetworkType
	NetworkID   string
}

type ServiceConfig struct {
	Name         string
	Aliases      []string
	ImageURI     string
	AuthOrNil    *Auth
	PullStrategy models.DockerPullStrategy
	Env          []string
}

type ServiceManagerConfig struct {
	// RuntimeID uniquely identifies an instance of a runtime.
	RuntimeID          string
	NetworkType        ServiceNetworkType
	PrivateNetworkName string
}

type ServiceManager struct {
	config           ServiceManagerConfig
	containerManager *ContainerManager
	log              logger.Log
	logPipeline      logging.LogPipeline
	state            struct {
		networkID    string
		containerIDs []string
	}
}

func NewServiceManager(config ServiceManagerConfig, containerManager *ContainerManager, logPipeline logging.LogPipeline, logFactory logger.LogFactory) *ServiceManager {
	return &ServiceManager{
		config:           config,
		containerManager: containerManager,
		logPipeline:      logPipeline,
		log:              logFactory("ServiceManager"),
	}
}

func (s *ServiceManager) Start(ctx context.Context) (*Network, error) {
	network := &Network{NetworkType: s.config.NetworkType}
	switch s.config.NetworkType {
	case ServiceNetworkTypeHost:
		return network, nil // no-op
	case ServiceNetworkTypePrivate:
		id, err := s.containerManager.CreateNetwork(ctx, s.config.PrivateNetworkName)
		if err != nil {
			return nil, err
		}
		s.state.networkID = id
		network.NetworkID = id
		return network, nil
	default:
		return nil, fmt.Errorf("error unknown network type: %v", s.config.NetworkType)
	}
}

func (s *ServiceManager) StartService(ctx context.Context, config ServiceConfig) error {
	pConfig := &ImagePullConfig{
		ImageURI:     config.ImageURI,
		Auth:         config.AuthOrNil,
		PullStrategy: config.PullStrategy,
	}
	pLog := s.logPipeline.StructuredLogger().Wrapf(
		fmt.Sprintf("job_service_%s", config.Name), "Configuring %s...", config.Name)
	err := s.containerManager.PullDockerImage(ctx, pLog, pConfig)
	if err != nil {
		return fmt.Errorf("error pulling Docker image: %w", err)
	}
	cConfig := ContainerConfig{
		Name:     makeContainerNameForService(&s.config, &config),
		ImageURI: config.ImageURI,
		Env:      config.Env,
		Aliases:  []string{config.Name},
		// TODO Web UI needs some work to support concurrent writing to multiple blocks.
		//  Right now everything shows up under the most recently declared block.
		Stdout: nil,
		Stderr: nil,
	}
	if s.state.networkID != "" {
		cConfig.Networks = []string{s.state.networkID}
	}
	containerID, err := s.containerManager.StartContainer(ctx, cConfig)
	if err != nil {
		return fmt.Errorf("error starting service container: %w", err)
	}
	s.state.containerIDs = append(s.state.containerIDs, containerID)
	return nil
}

func (s *ServiceManager) Stop(ctx context.Context) error {
	var results *multierror.Error
	for _, containerID := range s.state.containerIDs {
		err := s.containerManager.StopContainer(ctx, containerID)
		if err != nil {
			results = multierror.Append(results, fmt.Errorf("error stopping container %q: %w", containerID, err))
		}
	}
	if s.state.networkID != "" {
		err := s.containerManager.DeleteNetwork(ctx, s.state.networkID)
		if err != nil {
			results = multierror.Append(results, fmt.Errorf("error deleting network: %w", err))
		}
	}
	return results.ErrorOrNil()
}
