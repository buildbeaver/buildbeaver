package routes

import "fmt"

func MakeGitHubAuthenticationURL(rctx RequestContext) string {
	return fmt.Sprintf("%s/api/v1/authentication/github", rctx)
}
