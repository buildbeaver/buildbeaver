package bb

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/net/context"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

const BuildResourceKind = "build"

// BuildDefinitionSyntaxVersion is the version of the syntax used when submitting build definitions.
// This is equivalent to the 'version' field in YAML for statically-defined builds
const BuildDefinitionSyntaxVersion = "0.3"

type BuildID struct {
	ResourceID
}

func ParseBuildID(str string) (BuildID, error) {
	id, err := ParseResourceID(str)
	if err != nil {
		return BuildID{}, err
	}
	if id.Kind() != BuildResourceKind {
		return BuildID{}, fmt.Errorf("error: Build ID expected to have kind '%s', found '%s'", BuildResourceKind, id.Kind())
	}
	return BuildID{ResourceID: id}, nil
}

type Build struct {
	ID             BuildID
	Name           ResourceName // name for this build (may actually be a number)
	OwnerName      string
	Ref            string
	DynamicJobID   JobID
	DynamicJobName ResourceName
	Repo           *Repo
	Commit         *Commit
	// workflowsToRun is a static list of workflows that were requested to be run when the build was queued;
	// an empty list means run all workflows
	WorkflowsToRun []ResourceName

	// internal fields
	eventManager  *EventManager
	dynamicAPIURL string
	accessToken   AccessToken // JWT for accessing dynamic API
	apiClient     *client.APIClient
}

func GetBuild() (*Build, error) {
	build, err := parseEnv(os.Getenv)
	if err != nil {
		return nil, err
	}

	return build, nil
}

func MustGetBuild() *Build {
	build, err := GetBuild()
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}

	return build
}

// getBuildWithEnv returns a Build object for the current build, sourcing the environment variable data
// from the specified string map rather than from actual environment variables.
func getBuildWithEnv(envVars map[string]string) (*Build, error) {
	getter := makeStringMapEnvGetter(envVars)
	build, err := parseEnv(getter)
	if err != nil {
		return nil, err
	}

	return build, nil
}

func newBuild(
	buildID BuildID,
	buildName ResourceName,
	buildOwnerName string,
	buildRefStr string,
	dynamicJobID JobID,
	dynamicJobName ResourceName,
	dynamicAPIURL string,
	accessToken AccessToken,
	workflowsToRun []ResourceName,
) *Build {
	openapiConfig := client.NewConfiguration()

	// Add the supplied endpoint to the start of the list of servers in the config, so it becomes the default
	endPoint := strings.Trim(dynamicAPIURL, "/") // remove trailing slash, if present
	serverURL := fmt.Sprintf("%s/api/v1/dynamic", endPoint)
	server := client.ServerConfiguration{
		URL:         serverURL,
		Description: "Default BB Dynamic API server URL",
	}
	openapiConfig.Servers = append(client.ServerConfigurations{server}, openapiConfig.Servers...)
	Log(LogLevelDebug, fmt.Sprintf("Dynamic API Server URL: %s", serverURL))

	// Create a separate HTTP client to configure; do not share HTTP clients between instances of Build
	httpClient := &http.Client{}
	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryWaitMin = time.Millisecond * 100
	retryableClient.RetryWaitMax = time.Second * 5
	retryableClient.RetryMax = 100
	retryableClient.Logger = NewLeveledLogger(Log) // use adaptor to get log level support
	retryableClient.HTTPClient = httpClient
	// Configure the generated openapi client to use the retryableClient for HTTP
	openapiConfig.HTTPClient = retryableClient.StandardClient()

	apiClient := client.NewAPIClient(openapiConfig)

	// Create an event manager, which will immediately start polling for events for this build
	authContextFactory := func() context.Context { return GetAuthorizedContext(accessToken) }
	eventManager := NewEventManager(apiClient, authContextFactory, buildID, dynamicJobID, DefaultLogger)

	build := &Build{
		ID:             buildID,
		Name:           buildName,
		OwnerName:      buildOwnerName,
		Ref:            buildRefStr,
		DynamicJobID:   dynamicJobID,
		DynamicJobName: dynamicJobName,
		dynamicAPIURL:  dynamicAPIURL,
		accessToken:    accessToken,
		apiClient:      apiClient,
		eventManager:   eventManager,
		WorkflowsToRun: workflowsToRun,
	}

	return build
}

// Shutdown performs a clean shutdown of all background Goroutines associated with this build.
// This is typically only required when running tests, where the process will be re-used.
func (b *Build) Shutdown() {
	b.eventManager.Stop()
}

// GetAPIClient returns the generated OpenAPI client, for directly calling API functions.
func (b *Build) GetAPIClient() *client.APIClient {
	return b.apiClient
}

// GetAuthorizedContext returns a context that  can be passed to generated OpenAPI functions to authenticate
// to the server. The context includes an "apikey" value containing a JWT access token for authentication.
func (b *Build) GetAuthorizedContext() context.Context {
	return GetAuthorizedContext(b.accessToken)
}

// GetBuildGraph reads the current build graph from the server. This will include all jobs and steps in the build,
// together with their current statuses.
func (b *Build) GetBuildGraph() (*client.BuildGraph, error) {
	// Call API function
	Log(LogLevelInfo, fmt.Sprintf("Fetching Build Graph from server for build %s", b.ID))
	buildAPI := b.apiClient.BuildApi

	bGraph, response, err := buildAPI.GetBuild(b.GetAuthorizedContext(), b.ID.String()).Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("Error fetching build graph from server (response status code %d): %s - %s\n", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("Error fetching build graph from server (response status code %d): %w\n", statusCode, err)
	}
	Log(LogLevelInfo, fmt.Sprintf("Received Build Graph back from server with %d jobs", len(bGraph.Jobs)))

	return bGraph, nil
}

// MustGetBuildGraph reads the current build graph from the server. This will include all jobs and steps in the build,
// together with their current statuses.
// Terminates this program if a persistent error occurs.
func (b *Build) MustGetBuildGraph() *client.BuildGraph {
	bGraph, err := b.GetBuildGraph()
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return bGraph
}

// GetJobGraph reads a job graph from the server for the specified job ID. This will include all steps in the job,
// together with the current job status.
func (b *Build) GetJobGraph(jobID JobID) (*client.JobGraph, error) {
	// Call API function
	Log(LogLevelInfo, fmt.Sprintf("Fetching Job Graph from server for job %s", jobID))
	buildAPI := b.apiClient.BuildApi

	jGraph, response, err := buildAPI.GetJobGraph(b.GetAuthorizedContext(), jobID.String()).Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("Error fetching job graph from server (response status code %d): %s - %s\n", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("Error fetching job graph from server (response status code %d): %w\n", statusCode, err)
	}

	return jGraph, nil
}

// MustGetJobGraph reads a job graph from the server for the specified job ID. This will include all steps in the job,
// together with the current job status.
// Terminates this program if a persistent error occurs.
func (b *Build) MustGetJobGraph(jobID JobID) *client.JobGraph {
	jGraph, err := b.GetJobGraph(jobID)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return jGraph
}

// GetJob reads information about the job with the specified Job ID from the server. This will include the current
// job status. but does not include information about the steps in the job; see GetJobGraph().
func (b *Build) GetJob(jobID JobID) (*client.Job, error) {
	// Call API function
	Log(LogLevelInfo, fmt.Sprintf("Fetching Job Graph from server for job %s", jobID))
	buildAPI := b.apiClient.BuildApi

	job, response, err := buildAPI.GetJob(b.GetAuthorizedContext(), jobID.String()).Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("Error fetching job from server (response status code %d): %s - %s\n", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("Error fetching job from server (response status code %d): %w\n", statusCode, err)
	}

	return job, nil
}

// MustGetJob reads information about the job with the specified Job ID from the server. This will include the current
// job status. but does not include information about the steps in the job; see GetJobGraph().
// Terminates this program if a persistent error occurs.
func (b *Build) MustGetJob(jobID JobID) *client.Job {
	job, err := b.GetJob(jobID)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return job
}

// ListArtifacts reads information about selected artifacts from the current build.
// The first page of results will be returned in an ArtifactPage object.
// Call Next() on the returned object to get the next page of results, or Prev() to get the previous page.
func (b *Build) ListArtifacts(workflow string, jobName string, groupName string) (*ArtifactPage, error) {
	request := NewBuildApiListArtifactsRequest(b, workflow, jobName, groupName, 30)
	return ListArtifacts(b, request)
}

// MustListArtifacts reads information about selected artifacts from the current build.
// The first page of results will be returned in an ArtifactPage object.
// Call Next() on the returned object to get the next page of results, or Prev() to get the previous page.
// Terminates this program if a persistent error occurs.
func (b *Build) MustListArtifacts(workflow string, jobName string, groupName string) *ArtifactPage {
	res, err := b.ListArtifacts(workflow, jobName, groupName)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return res
}

// ListArtifactsN reads information about selected artifacts from the current build.
// The first page of up to pageSize results will be returned in an ArtifactPage object.
// Call Next() on the returned object to get the next page of results, or Prev() to get the previous page.
func (b *Build) ListArtifactsN(workflow string, jobName string, groupName string, pageSize int) (*ArtifactPage, error) {
	request := NewBuildApiListArtifactsRequest(b, workflow, jobName, groupName, pageSize)
	return ListArtifacts(b, request)
}

// MustListArtifactsN reads information about selected artifacts from the current build.
// The first page of up to pageSize results will be returned in an ArtifactPage object.
// Call Next() on the returned object to get the next page of results, or Prev() to get the previous page.
// Terminates this program if a persistent error occurs.
func (b *Build) MustListArtifactsN(workflow string, jobName string, groupName string, pageSize int) *ArtifactPage {
	res, err := b.ListArtifactsN(workflow, jobName, groupName, pageSize)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return res
}

// GetArtifactData returns the binary data for an artifact.
func (b *Build) GetArtifactData(artifactID string) ([]byte, error) {
	Log(LogLevelInfo, fmt.Sprintf("Fetching artifact from server for ID %s", artifactID))
	buildAPI := b.apiClient.BuildApi

	artifactFile, response, err := buildAPI.GetArtifactData(b.GetAuthorizedContext(), artifactID).Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("error fetching artifact data from server (response status code %d): %s - %s", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("error fetching artifact data from server (response status code %d): %w", statusCode, err)
	}
	Log(LogLevelInfo, fmt.Sprintf("Received artifact data back from server"))

	data, err := io.ReadAll(artifactFile)
	if err != nil {
		return nil, fmt.Errorf("error reading file to fetch artifact data: %w", err)
	}

	return data, nil
}

// MustGetArtifactData returns the binary data for an artifact.
// Terminates this program if a persistent error occurs.
func (b *Build) MustGetArtifactData(artifactID string) []byte {
	data, err := b.GetArtifactData(artifactID)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return data
}

// GetLogDescriptor returns a log descriptor containing information/metadata about a log (e.g. the log for a job
// or for a step).
func (b *Build) GetLogDescriptor(logDescriptorID string) (*client.LogDescriptor, error) {
	Log(LogLevelInfo, fmt.Sprintf("Fetching log descriptor from server for ID %s", logDescriptorID))
	buildAPI := b.apiClient.BuildApi

	logDescriptor, response, err := buildAPI.GetLogDescriptor(b.GetAuthorizedContext(), logDescriptorID).Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("error fetching log descriptor from server (response status code %d): %s - %s", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("error fetching log descriptor from server (response status code %d): %w", statusCode, err)
	}

	return logDescriptor, nil
}

// MustGetLogDescriptor returns a log descriptor containing information/metadata about a log (e.g. the log for a job
// or for a step).
// Terminates this program if a persistent error occurs.
func (b *Build) MustGetLogDescriptor(logDescriptorID string) *client.LogDescriptor {
	res, err := b.GetLogDescriptor(logDescriptorID)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return res
}

// ReadLogText returns a reader for fetching the data for a log in plain text (e.g. the log for a job or for a step).
// If expand is true then nested logs will be expanded and returned.
func (b *Build) ReadLogText(logDescriptorID string, expand bool) (io.ReadCloser, error) {
	Log(LogLevelInfo, fmt.Sprintf("Fetching log data from server for log with log descriptor ID %s", logDescriptorID))
	buildAPI := b.apiClient.BuildApi

	logFile, response, err := buildAPI.GetLogData(b.GetAuthorizedContext(), logDescriptorID).
		Plaintext(true).
		Expand(expand).
		Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("error fetching log data from server (response status code %d): %s - %s", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("error fetching log data from server (response status code %d): %w", statusCode, err)
	}

	// An empty body means an empty log, but is returned from the generated client as nil
	if logFile == nil {
		// Return a ReadCloser that has no data, so the caller doesn't have to deal with nil
		return io.NopCloser(bytes.NewReader([]byte{})), nil
	}

	return logFile, nil
}

// MustReadLogText returns a reader for fetching the data for a log in plain text (e.g. the log for a job or for a step).
// If expand is true then nested logs will be expanded and returned.
// Terminates this program if a persistent error occurs.
func (b *Build) MustReadLogText(logDescriptorID string, expand bool) io.ReadCloser {
	reader, err := b.ReadLogText(logDescriptorID, expand)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return reader
}

// ReadLogData returns a reader for fetching data for a log (e.g. the log for a job or for a step) as a series of
// JSON log entries. If expand is true then nested logs will be expanded and returned.
func (b *Build) ReadLogData(logDescriptorID string, expand bool) (io.ReadCloser, error) {
	Log(LogLevelInfo, fmt.Sprintf("Fetching log data from server for log with log descriptor ID %s", logDescriptorID))
	buildAPI := b.apiClient.BuildApi

	logFile, response, err := buildAPI.GetLogData(b.GetAuthorizedContext(), logDescriptorID).
		Plaintext(false).
		Expand(expand).
		Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("error fetching log data from server (response status code %d): %s - %s", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("error fetching log data from server (response status code %d): %w", statusCode, err)
	}

	// An empty body means an empty log, but is returned from the generated client as nil
	if logFile == nil {
		// Return a ReadCloser that has no data, so the caller doesn't have to deal with nil
		return io.NopCloser(bytes.NewReader([]byte{})), nil
	}

	return logFile, nil
}

// MustReadLogData returns a reader for fetching data for a log (e.g. the log for a job or for a step) as a series of
// JSON log entries. If expand is true then nested logs will be expanded and returned.
// Terminates this program if a persistent error occurs.
func (b *Build) MustReadLogData(logDescriptorID string, expand bool) io.ReadCloser {
	reader, err := b.ReadLogData(logDescriptorID, expand)
	if err != nil {
		Log(LogLevelFatal, err.Error())
		os.Exit(1)
	}
	return reader
}
