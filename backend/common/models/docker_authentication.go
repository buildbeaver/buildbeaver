package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type DockerBasicAuth struct {
	Username SecretString `json:"username"`
	Password SecretString `json:"password"`
}

type DockerAWSAuth struct {
	AWSRegion          string       `json:"aws_region"`
	AWSAccessKeyID     SecretString `json:"aws_access_key_id"`
	AWSSecretAccessKey SecretString `json:"aws_secret_access_key"`
}

// DockerAuth provides the schema for an end user providing their Docker authentication details in job definition.
// Support auth types are:
//
//	Basic - Uses a username and password. This works for the official Docker Hub and most third party registries
//	        See: https://docs.docker.com/engine/api/v1.41/#section/Authentication
//	AWS   - Uses an AWS access key to generate a temporary password, for authenticating to AWS ECR
//	        See: https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html
type DockerAuth struct {
	Basic *DockerBasicAuth `json:"basic"`
	AWS   *DockerAWSAuth   `json:"aws"`
}

func (m *DockerAuth) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m DockerAuth) Value() (driver.Value, error) {
	if m == (DockerAuth{}) {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
