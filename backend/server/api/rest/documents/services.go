package documents

import "github.com/buildbeaver/buildbeaver/common/models"

type Service struct {
	*DockerConfig
	// Name is the unique name of the service, within the parent job.
	Name string `json:"name"`
	// Environment contains a list of environment variables to export prior
	// to starting the service.
	Environment []*EnvVar `json:"environment"`
}

func MakeService(service *models.Service) *Service {
	return &Service{
		Name:         service.Name,
		Environment:  MakeEnvVars(service.Environment),
		DockerConfig: MakeDockerConfig(service.DockerImage, models.DockerPullStrategyDefault, service.DockerRegistryAuthentication, nil),
	}
}
func MakeServices(services []*models.Service) []*Service {
	var docs []*Service
	for _, service := range services {
		docs = append(docs, MakeService(service))
	}
	return docs
}
