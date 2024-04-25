package bb

import (
	"fmt"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

type Service struct {
	definition client.ServiceDefinition
}

func NewService() *Service {
	return &Service{
		definition: client.ServiceDefinition{
			Environment: make(map[string]client.SecretStringDefinition),
		},
	}
}

func (service *Service) GetData() client.ServiceDefinition {
	return service.definition
}

func (service *Service) GetName() ResourceName {
	return ResourceName(service.definition.Name)
}

func (service *Service) Name(name string) *Service {
	service.definition.Name = name
	return service
}

func (service *Service) Image(image string) *Service {
	service.definition.Image = image
	return service
}

func (service *Service) Env(env *Env) *Service {
	def := client.SecretStringDefinition{Value: &env.value}
	if env.secretName != "" {
		def = client.SecretStringDefinition{FromSecret: &env.secretName}
	}
	service.definition.Environment[env.name] = def
	Log(LogLevelInfo, fmt.Sprintf("Env var with name '%s' added for service '%s'", env.name, service.GetName()))
	return service
}

// BasicAuth configures basic auth credentials for the Docker registry.
func (service *Service) BasicAuth(auth *BasicAuth) *Service {
	username := client.SecretStringDefinition{Value: &auth.username}
	if auth.usernameFromSecret != "" {
		username = client.SecretStringDefinition{FromSecret: &auth.usernameFromSecret}
	}
	password := client.SecretStringDefinition{FromSecret: &auth.passwordFromSecret}
	service.definition.BasicAuth = &client.DockerBasicAuthDefinition{
		Username: username,
		Password: password,
	}
	return service
}

// AWSAuth configures AWS auth credentials for AWS ECR.
func (service *Service) AWSAuth(auth *AWSAuth) *Service {
	accessKeyID := client.SecretStringDefinition{Value: &auth.accessKeyID}
	if auth.accessKeyIDFromSecret != "" {
		accessKeyID = client.SecretStringDefinition{FromSecret: &auth.accessKeyIDFromSecret}
	}
	secretAccessKey := client.SecretStringDefinition{FromSecret: &auth.secretAccessKeyKeyFromSecret}
	service.definition.AwsAuth = &client.DockerAWSAuthDefinition{
		AwsAccessKeyId:     accessKeyID,
		AwsSecretAccessKey: secretAccessKey,
	}
	return service
}
