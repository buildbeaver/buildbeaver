package bb

import (
	"fmt"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

// ArtifactPage is a 'page' of artifacts, and allows the next and previous pages (if any) to be fetched.
type ArtifactPage struct {
	Artifacts []client.Artifact
	// internal fields
	build      *Build
	request    *client.ApiListArtifactsRequest
	nextCursor string
	prevCursor string
}

func NewArtifactPage(
	build *Build,
	request *client.ApiListArtifactsRequest,
	artifactsResult *client.ArtifactsPaginatedResponse,
) *ArtifactPage {
	return &ArtifactPage{
		Artifacts:  artifactsResult.Results,
		build:      build,
		request:    request,
		nextCursor: artifactsResult.GetNextCursor(),
		prevCursor: artifactsResult.GetPrevCursor(),
	}
}

func (p *ArtifactPage) HasNext() bool {
	return p != nil && p.nextCursor != ""
}

func (p *ArtifactPage) Next() (*ArtifactPage, error) {
	if !p.HasNext() {
		return nil, fmt.Errorf("error: No next page exists")
	}
	// Create a new request same as old request but with Cursor set
	newRequest := p.request.Cursor(p.nextCursor)
	return ListArtifacts(p.build, &newRequest)
}

func (p *ArtifactPage) HasPrev() bool {
	return p != nil && p.prevCursor != ""
}

func (p *ArtifactPage) Prev() (*ArtifactPage, error) {
	if !p.HasPrev() {
		return nil, fmt.Errorf("error: No previous page exists")
	}
	// Create a new request same as old request but with Cursor set
	newRequest := p.request.Cursor(p.prevCursor)
	return ListArtifacts(p.build, &newRequest)
}

func NewBuildApiListArtifactsRequest(b *Build, workflow string, jobName string, groupName string, pageSize int) *client.ApiListArtifactsRequest {
	buildAPI := b.apiClient.BuildApi

	Log(LogLevelInfo, fmt.Sprintf("Fetching artifact information, job '%s', group name '%s' (page size %d)",
		jobName, groupName, pageSize))

	request := buildAPI.ListArtifacts(b.GetAuthorizedContext(), b.ID.String()).
		Workflow(workflow).
		JobName(jobName).
		GroupName(groupName).
		Limit(int32(pageSize))

	return &request
}

func ListArtifacts(b *Build, request *client.ApiListArtifactsRequest) (*ArtifactPage, error) {

	artifactsResult, response, err := request.Execute()
	var statusCode int
	if response != nil {
		statusCode = response.StatusCode
	}
	if err != nil {
		openAPIErr, ok := err.(*client.GenericOpenAPIError)
		if ok {
			return nil, fmt.Errorf("error listing artifacts from server (response status code %d): %s - %s", statusCode, openAPIErr.Error(), openAPIErr.Body())
		}
		return nil, fmt.Errorf("error listing artifacts from server (response status code %d): %w", statusCode, err)
	}
	Log(LogLevelInfo, fmt.Sprintf("Received information about %d artifacts from server", len(artifactsResult.Results)))

	return NewArtifactPage(b, request, artifactsResult), nil
}
