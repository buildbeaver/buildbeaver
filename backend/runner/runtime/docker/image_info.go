package docker

import (
	"fmt"
	"strings"
)

const defaultDockerRegistry = "docker.io"

type dockerImageInfo struct {
	Repository string
	Tag        string
	Registry   string
}

func parseDockerImageURI(imageURI string) *dockerImageInfo {
	var (
		nameAndTag string
		repository string
		tag        string
		registry   string
	)
	n := strings.LastIndex(imageURI, "/")
	if n == -1 {
		nameAndTag = imageURI
		registry = defaultDockerRegistry + "/library" // TODO this might need to change for self-hosted default registries
	} else {
		nameAndTag = imageURI
		registry = defaultDockerRegistry
	}
	parts := strings.Split(nameAndTag, ":")
	repository = parts[0]
	if len(parts) > 1 {
		tag = parts[1]
	}
	return &dockerImageInfo{
		Repository: repository,
		Tag:        tag,
		Registry:   registry,
	}
}

func (m *dockerImageInfo) IsLatest() bool {
	return m.Tag == "" || m.Tag == "latest"
}

func (m *dockerImageInfo) Reference() string {
	ref := m.Repository
	if m.Tag != "" {
		ref = fmt.Sprintf("%s:%s", ref, m.Tag)
	}
	return ref
}

func (m *dockerImageInfo) FQN() string {
	fqn := fmt.Sprintf("%s/%s", m.Registry, m.Repository)
	if m.Tag != "" {
		fqn = fmt.Sprintf("%s:%s", fqn, m.Tag)
	}
	return fqn
}

func (m *dockerImageInfo) String() string {
	return m.FQN()
}
