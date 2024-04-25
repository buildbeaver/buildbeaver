package documents

import "github.com/buildbeaver/buildbeaver/common/models"

type DockerBasicAuth struct {
	Username *SecretString `json:"username"`
	Password *SecretString `json:"password"`
}

type DockerAWSAuth struct {
	AWSRegion          *string       `json:"aws_region"`
	AWSAccessKeyID     *SecretString `json:"aws_access_key_id"`
	AWSSecretAccessKey *SecretString `json:"aws_secret_access_key"`
}

type DockerConfig struct {
	// Image is the default Docker image to run the job's steps in.
	// In the future, steps may override this property by setting their own DockerImage.
	Image string `json:"image,omitempty"`
	// ImagePullStrategy determines if/when the Docker image is pulled during job execution.
	Pull models.DockerPullStrategy `json:"pull,omitempty"`
	// BasicAuth specifies the basic auth credentials to use when pulling the Docker image from the registry.
	BasicAuth *DockerBasicAuth `json:"basic_auth,omitempty"`
	// AWSAuth specifies the AWS auth credentials to use when pulling the Docker image from the AWS ECR-based registry.
	AWSAuth *DockerAWSAuth `json:"aws_auth,omitempty"`
	// Shell is the path to the shell to use to run build scripts with inside the container, or nil for the default.
	Shell *string `json:"shell,omitempty"`
}

func MakeDockerConfig(image string, pull models.DockerPullStrategy, auth *models.DockerAuth, shell *string) *DockerConfig {
	var basicAuth *DockerBasicAuth
	if auth != nil && auth.Basic != nil {
		basicAuth = &DockerBasicAuth{}
		basicAuth.Username = &SecretString{
			Value:      auth.Basic.Username.Value,
			FromSecret: auth.Basic.Username.ValueFromSecret,
		}
		basicAuth.Password = &SecretString{
			Value:      auth.Basic.Password.Value,
			FromSecret: auth.Basic.Password.ValueFromSecret,
		}
	}
	var awsAuth *DockerAWSAuth
	if auth != nil && auth.AWS != nil {
		awsAuth = &DockerAWSAuth{}
		if auth.AWS.AWSRegion != "" {
			awsAuth.AWSRegion = &auth.AWS.AWSRegion
		}
		awsAuth.AWSAccessKeyID = &SecretString{
			Value:      auth.AWS.AWSAccessKeyID.Value,
			FromSecret: auth.AWS.AWSAccessKeyID.ValueFromSecret,
		}
		awsAuth.AWSSecretAccessKey = &SecretString{
			Value:      auth.AWS.AWSSecretAccessKey.Value,
			FromSecret: auth.AWS.AWSSecretAccessKey.ValueFromSecret,
		}
	}
	return &DockerConfig{
		Image:     image,
		Pull:      pull,
		BasicAuth: basicAuth,
		AWSAuth:   awsAuth,
		Shell:     shell,
	}
}
