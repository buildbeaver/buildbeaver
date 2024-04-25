package documents

import "github.com/buildbeaver/buildbeaver/common/models"

// ArtifactDefinition is generated from steps in the build config.
// It declares that a step is expected to create one or more artifacts at the given paths, and
// that these artifact files should be saved and made available to other steps (see ArtifactDependency)
type ArtifactDefinition struct {
	// GroupName uniquely identifies the one or more artifacts specified in paths.
	GroupName models.ResourceName `json:"name"`
	// Paths contains one or more relative paths to artifacts that should be uploaded at the
	// end of the build. These paths will be globbed, so that each path may identify one or
	// more actual files.
	Paths []string `json:"paths"`
}

func MakeArtifactDefinition(definition *models.ArtifactDefinition) *ArtifactDefinition {
	return &ArtifactDefinition{
		GroupName: definition.GroupName,
		Paths:     definition.Paths,
	}
}

func MakeArtifactDefinitions(definitions models.ArtifactDefinitions) []*ArtifactDefinition {
	var docs []*ArtifactDefinition
	for _, definition := range definitions {
		docs = append(docs, MakeArtifactDefinition(definition))
	}
	return docs
}
