package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeStepLink(rctx RequestContext, stepID models.StepID) string {
	return fmt.Sprintf("%s/api/v1/steps/%s", rctx, stepID)
}
