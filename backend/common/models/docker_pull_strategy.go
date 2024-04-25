package models

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	DockerPullStrategyDefault     DockerPullStrategy = "default"
	DockerPullStrategyNever       DockerPullStrategy = "never"
	DockerPullStrategyAlways      DockerPullStrategy = "always"
	DockerPullStrategyIfNotExists DockerPullStrategy = "if-not-exists"
)

type DockerPullStrategy string

func (m *DockerPullStrategy) Scan(src interface{}) error {
	if src == nil {
		*m = DockerPullStrategyDefault
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return errors.Errorf("error expected string but found: %T", src)
	}
	switch strings.ToLower(t) {
	case "", string(DockerPullStrategyDefault):
		*m = DockerPullStrategyDefault
	case string(DockerPullStrategyNever):
		*m = DockerPullStrategyNever
	case string(DockerPullStrategyAlways):
		*m = DockerPullStrategyAlways
	case string(DockerPullStrategyIfNotExists):
		*m = DockerPullStrategyIfNotExists
	default:
		return errors.Errorf("error unknown Docker pull strategy: %s", t)
	}
	return nil
}

func (m DockerPullStrategy) Valid() bool {
	return m == DockerPullStrategyDefault ||
		m == DockerPullStrategyNever ||
		m == DockerPullStrategyAlways ||
		m == DockerPullStrategyIfNotExists
}

func (m DockerPullStrategy) String() string {
	return string(m)
}
