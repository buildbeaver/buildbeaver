package routes

import (
	"fmt"
)

func MakeSearchLink(rctx RequestContext) string {
	return fmt.Sprintf("%s/api/v1/search", rctx)
}
