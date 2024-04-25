package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeRunnerLink(rctx RequestContext, runnerID models.RunnerID) string {
	return fmt.Sprintf("%s/api/v1/runners/%s", rctx, runnerID)
}

func MakeRunnersLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/runners", MakeLegalEntityLink(rctx, legalEntityID))
}

func MakeRunnerSearchLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/search", MakeRunnersLink(rctx, legalEntityID))
}
