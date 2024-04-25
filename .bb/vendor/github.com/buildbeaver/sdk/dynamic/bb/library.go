package bb

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

// envGetter is a function that can be used to fetch environment variables.
// It returns the value for the specified key, which will be empty if the variable is not present.
type envGetter func(key string) string

// makeStringMapEnvGetter returns a function that can be used as an envGetter, that fetches environment
// variables by looking them up in the supplied string map.
func makeStringMapEnvGetter(envVars map[string]string) func(key string) string {
	return func(key string) string {
		return envVars[key]
	}
}

// parseEnv reads environment variables using the supplied envGetter function and populates the build, repo,
// commit and other information to be made available to the dynamic build code.
// The os.Getenv() function can be passed in directly as the env parameter.
// Returns a Build structure containing all the data.
func parseEnv(env envGetter) (*Build, error) {

	dynamicAPIURL := env("BB_DYNAMIC_BUILD_API")
	if dynamicAPIURL == "" {
		return nil, fmt.Errorf("error: Dynamic Build API endpoint must be provided")
	}

	buildIDStr := env("BB_BUILD_ID")
	buildID, err := ParseBuildID(buildIDStr)
	if err != nil {
		return nil, err
	}
	var buildName ResourceName
	buildNameStr := env("BB_BUILD_NAME")
	if buildNameStr != "" {
		buildName, err = ParseResourceName(buildNameStr)
		if err != nil {
			return nil, err
		}
	}
	buildOwnerName := env("BB_BUILD_OWNER_NAME")
	buildRefStr := env("BB_BUILD_REF")
	accessTokenStr := env("BB_BUILD_ACCESS_TOKEN")
	accessToken, err := ParseAccessToken(accessTokenStr)
	if err != nil {
		return nil, err
	}

	var workflowsToRun []ResourceName
	workflowsToRunStr := env("BB_WORKFLOWS_TO_RUN")
	if workflowsToRunStr != "" {
		workflowsToRun, err = ParseResourceNames(workflowsToRunStr)
		if err != nil {
			return nil, err
		}
	}

	dynamicJobIDStr := env("BB_CONTROLLER_JOB_ID")
	dynamicJobID, err := ParseJobID(dynamicJobIDStr)
	if err != nil {
		return nil, err
	}
	dynamicJobNameStr := env("BB_CONTROLLER_JOB_NAME")
	dynamicJobName, err := ParseResourceName(dynamicJobNameStr)
	if err != nil {
		return nil, err
	}

	build := newBuild(buildID, buildName, buildOwnerName, buildRefStr, dynamicJobID, dynamicJobName, dynamicAPIURL, accessToken, workflowsToRun)

	commitSHAStr := env("BB_COMMIT_SHA")
	if commitSHAStr == "" {
		return nil, fmt.Errorf("error: Commit SHA must be provided")
	}
	commitAuthorName := env("BB_COMMIT_AUTHOR_NAME")
	commitAuthorEmail := env("BB_COMMIT_AUTHOR_EMAIL")
	commitCommitterName := env("BB_COMMIT_COMMITTER_NAME")
	commitCommitterEmail := env("BB_COMMIT_COMMITTER_EMAIL")
	build.Commit = &Commit{
		SHA:            commitSHAStr,
		AuthorName:     commitAuthorName,
		AuthorEmail:    commitAuthorEmail,
		CommitterName:  commitCommitterName,
		CommitterEmail: commitCommitterEmail,
	}

	repoNameStr := env("BB_REPO_NAME")
	repoName, err := ParseResourceName(repoNameStr)
	if err != nil {
		return nil, err
	}
	repoSSHURL := env("BB_REPO_SSH_URL")
	repoLink := env("BB_REPO_LINK")
	build.Repo = &Repo{
		Name:   repoName,
		SSHURL: repoSSHURL,
		Link:   repoLink,
	}

	return build, nil
}

// GetAuthorizedContext returns a context that  can be passed to generated OpenAPI functions to authenticate
// to the server.
// The context includes an "apikey" value containing the supplied JWT access token for authentication.
func GetAuthorizedContext(accessToken AccessToken) context.Context {
	// Create a map of API keys for use by the generated code
	apiKeys := map[string]client.APIKey{
		"jwt_build_token": {
			Prefix: "Bearer",
			Key:    accessToken.String(),
		},
	}

	return context.WithValue(context.Background(), client.ContextAPIKeys, apiKeys)
}
