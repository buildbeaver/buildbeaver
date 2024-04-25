package bb

import (
	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

type Artifact struct {
	definition client.ArtifactDefinition
}

func NewArtifact() *Artifact {
	return &Artifact{definition: client.ArtifactDefinition{}}
}

func (a *Artifact) GetData() client.ArtifactDefinition {
	return a.definition
}

func (a *Artifact) GetName() ResourceName {
	return ResourceName(a.definition.Name)
}

func (a *Artifact) Name(name string) *Artifact {
	a.definition.Name = name
	return a
}

func (a *Artifact) Paths(paths ...string) *Artifact {
	a.definition.Paths = paths
	return a
}
