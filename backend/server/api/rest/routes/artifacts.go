package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeArtifactLink(rctx RequestContext, artifactID models.ArtifactID) string {
	return fmt.Sprintf("%s/api/v1/artifacts/%s", rctx, artifactID)
}

func MakeArtifactsDataLink(rctx RequestContext, artifactID models.ArtifactID) string {
	return fmt.Sprintf("%s/data", MakeArtifactLink(rctx, artifactID))
}

func MakeArtifactsLink(rctx RequestContext, buildID models.BuildID) string {
	return fmt.Sprintf("%s/api/v1/builds/%s/artifacts", rctx, buildID)
}

func MakeArtifactSearchLink(rctx RequestContext, buildID models.BuildID) string {
	return fmt.Sprintf("%s/search", MakeArtifactsLink(rctx, buildID))
}

// NOTE: The "ForRunner" links are a hack until build runners have their own identity and we can
// leverage access control to lock down which endpoints they can use (instead of firewalling endpoints
// under "../runner/.." and only applying ClientCertAuth to those endpoints.
func MakeArtifactsLinkForRunner(rctx RequestContext, buildID models.BuildID) string {
	return fmt.Sprintf("%s/api/v1/runner/builds/%s/artifacts", rctx, buildID)
}
