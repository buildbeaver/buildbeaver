package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeBuildLink(rctx RequestContext, buildID models.BuildID) string {
	return fmt.Sprintf("%s/api/v1/builds/%s", rctx, buildID)
}

func MakeBuildsLink(rctx RequestContext, repoID models.RepoID) string {
	return fmt.Sprintf("%s/builds", MakeRepoLink(rctx, repoID))
}

func MakeBuildSearchLink(rctx RequestContext, repoID models.RepoID) string {
	return fmt.Sprintf("%s/search", MakeBuildsLink(rctx, repoID))
}

func MakeBuildSummaryLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/builds/summary", MakeLegalEntityLink(rctx, legalEntityID))
}
