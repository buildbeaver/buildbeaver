package routes

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func MakeJobLink(rctx RequestContext, jobID models.JobID) string {
	return fmt.Sprintf("%s/api/v1/jobs/%s", rctx, jobID)
}
