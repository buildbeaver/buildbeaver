package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeCredentialLink(rctx RequestContext, credentialID models.CredentialID) string {
	return fmt.Sprintf("%s/api/v1/credentials/%s", rctx, credentialID)
}
