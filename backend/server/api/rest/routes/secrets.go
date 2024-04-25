package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeSecretLink(rctx RequestContext, secretID models.SecretID) string {
	return fmt.Sprintf("%s/api/v1/secrets/%s", rctx, secretID)
}

func MakeSecretsLink(rctx RequestContext, repoID models.RepoID) string {
	return fmt.Sprintf("%s/secrets", MakeRepoLink(rctx, repoID))
}

func MakeSecretSearchLink(rctx RequestContext, repoID models.RepoID) string {
	return fmt.Sprintf("%s/search", MakeSecretsLink(rctx, repoID))
}
