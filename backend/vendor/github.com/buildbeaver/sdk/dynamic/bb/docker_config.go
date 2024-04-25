package bb

import (
	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

type DockerConfig struct {
	definition client.DockerConfigDefinition
}

func NewDocker() *DockerConfig {
	return &DockerConfig{definition: client.DockerConfigDefinition{}}
}

func (config *DockerConfig) GetData() client.DockerConfigDefinition {
	return config.definition
}

func (config *DockerConfig) Image(image string) *DockerConfig {
	config.definition.Image = image
	return config
}

func (config *DockerConfig) Pull(pullStrategy DockerPullStrategy) *DockerConfig {
	config.definition.Pull = pullStrategy.String()
	return config
}

func (config *DockerConfig) Shell(shell string) *DockerConfig {
	config.definition.Shell = &shell
	return config
}

type BasicAuth struct {
	username           string
	usernameFromSecret string
	passwordFromSecret string
}

func NewBasicAuth() *BasicAuth { return &BasicAuth{} }

func (m *BasicAuth) Username(username string) *BasicAuth {
	m.username = username
	return m
}

func (m *BasicAuth) UsernameFromSecret(secretName string) *BasicAuth {
	m.usernameFromSecret = secretName
	return m
}

func (m *BasicAuth) PasswordFromSecret(secretName string) *BasicAuth {
	m.passwordFromSecret = secretName
	return m
}

type AWSAuth struct {
	region                       string
	accessKeyID                  string
	accessKeyIDFromSecret        string
	secretAccessKeyKeyFromSecret string
}

func NewAWSAuth() *AWSAuth { return &AWSAuth{} }

func (m *AWSAuth) Region(region string) *AWSAuth {
	m.region = region
	return m
}

func (m *AWSAuth) AccessKeyID(accessKeyID string) *AWSAuth {
	m.accessKeyID = accessKeyID
	return m
}

func (m *AWSAuth) AccessKeyIDFromSecret(secretName string) *AWSAuth {
	m.accessKeyIDFromSecret = secretName
	return m
}

func (m *AWSAuth) SecretAccessKeyFromSecret(secretName string) *AWSAuth {
	m.secretAccessKeyKeyFromSecret = secretName
	return m
}

// BasicAuth configures basic auth credentials for the Docker registry.
func (config *DockerConfig) BasicAuth(auth *BasicAuth) *DockerConfig {
	username := client.SecretStringDefinition{Value: &auth.username}
	if auth.usernameFromSecret != "" {
		username = client.SecretStringDefinition{FromSecret: &auth.usernameFromSecret}
	}
	password := client.SecretStringDefinition{FromSecret: &auth.passwordFromSecret}
	config.definition.BasicAuth = &client.DockerBasicAuthDefinition{
		Username: username,
		Password: password,
	}
	return config
}

// AWSAuth configures AWS auth credentials for AWS ECR.
func (config *DockerConfig) AWSAuth(auth *AWSAuth) *DockerConfig {
	accessKeyID := client.SecretStringDefinition{Value: &auth.accessKeyID}
	if auth.accessKeyIDFromSecret != "" {
		accessKeyID = client.SecretStringDefinition{FromSecret: &auth.accessKeyIDFromSecret}
	}
	secretAccessKey := client.SecretStringDefinition{FromSecret: &auth.secretAccessKeyKeyFromSecret}
	config.definition.AwsAuth = &client.DockerAWSAuthDefinition{
		AwsAccessKeyId:     accessKeyID,
		AwsSecretAccessKey: secretAccessKey,
	}
	if auth.region != "" {
		config.definition.AwsAuth.AwsRegion = &auth.region
	}
	return config
}
