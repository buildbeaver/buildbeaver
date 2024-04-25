package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Service struct {
	// Name is the unique name of the service, within the parent job.
	Name string `json:"name"`
	// DockerImage is the Docker image of the service to run.
	DockerImage string `json:"image"`
	// DockerRegistryAuthentication contains the optional authentication for pulling a docker image.
	DockerRegistryAuthentication *DockerAuth `json:"docker_authentication" db:"step_docker_authentication"`
	// Environment contains a list of environment variables to export prior
	// to starting the service.
	Environment []*EnvVar `json:"environment"`
}

func (m *Service) Validate() error {
	var result *multierror.Error
	if m.Name == "" {
		result = multierror.Append(result, errors.New("Service name must be specified"))
	} else if !ResourceNameRegex.MatchString(m.Name) {
		result = multierror.Append(result, errors.New("Service name must only contain alphanumeric, dash or underscore characters (matching ^[a-zA-Z0-9_-]+$)"))
	}
	if len(m.Name) > resourceNameMaxLength {
		result = multierror.Append(result, errors.Errorf("Service name must not be longer than %d characters", resourceNameMaxLength))
	}
	if m.DockerImage == "" {
		result = multierror.Append(result, errors.New("Service Docker image must be specified"))
	}

	return result.ErrorOrNil()
}

type JobServices []*Service

func (m *JobServices) Scan(src interface{}) error {
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

func (m JobServices) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
