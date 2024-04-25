package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeRepoLink(rctx RequestContext, repoID models.RepoID) string {
	return fmt.Sprintf("%s/api/v1/repos/%s", rctx, repoID)
}

func MakeReposLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/repos", MakeLegalEntityLink(rctx, legalEntityID))
}

func MakeRepoSearchLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/search", MakeReposLink(rctx, legalEntityID))
}
