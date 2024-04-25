package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeLogLink(rctx RequestContext, logDescriptorID models.LogDescriptorID) string {
	return fmt.Sprintf("%s/api/v1/logs/%s", rctx, logDescriptorID)
}

func MakeLogDataLink(rctx RequestContext, logID models.LogDescriptorID) string {
	return fmt.Sprintf("%s/data", MakeLogLink(rctx, logID))
}
