package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeLegalEntitiesLink(rctx RequestContext) string {
	return fmt.Sprintf("%s/api/v1/legal-entities", rctx)
}

func MakeLegalEntityLink(rctx RequestContext, legalEntityID models.LegalEntityID) string {
	return fmt.Sprintf("%s/%s", MakeLegalEntitiesLink(rctx), legalEntityID)
}

func MakeCurrentLegalEntityLink(rctx RequestContext) string {
	return fmt.Sprintf("%s/api/v1/user", rctx)
}
