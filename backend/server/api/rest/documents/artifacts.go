package documents

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Artifact struct {
	baseResourceDocument

	ID        models.ArtifactID `json:"id"`
	CreatedAt models.Time       `json:"created_at"`
	UpdatedAt models.Time       `json:"updated_at"`
	ETag      models.ETag       `json:"etag" hash:"ignore"`

	// Name of the artifact.
	Name models.ResourceName `json:"name"`
	// JobID is the ID of the job that created this artifact.
	JobID models.JobID `json:"job_id"`
	// GroupName is the name associated with the one or more artifacts identified by an ArtifactDefinition in the build config.
	GroupName models.ResourceName `json:"group_name"`
	// Path is the filesystem path that the artifact was found at, relative to the job workspace.
	Path string `json:"path"`
	// HashType is the type of hashing algorithm used to hash the data.
	HashType models.HashType `json:"hash_type"`
	// Hash is the hex-encoded hash of the artifact data. This This may be set later if the hash is not known yet.
	Hash string `json:"hash"`
	// Size of the artifact file in bytes.
	Size uint64 `json:"size"`
	// Mime type of the artifact, or empty if not known.
	Mime string `json:"mime"`
	// Sealed is true once the data for the artifact has successfully been uploaded and the file contents are now locked.
	// Until Sealed is true various pieces of metadata such as the file size and hash etc. will be unset.
	// NOTE: If sealed is false it doesn't necessarily mean no data has been uploaded to the blob store yet, and so
	// we must still verify that the backing data is deleted before garbage collecting unsealed artifact files.
	Sealed bool `json:"sealed"`

	DataURL string `json:"data_url"`
}

func MakeArtifact(rctx routes.RequestContext, artifact *models.Artifact) *Artifact {
	return &Artifact{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeArtifactLink(rctx, artifact.ID),
		},

		ID:        artifact.ID,
		CreatedAt: artifact.CreatedAt,
		UpdatedAt: artifact.UpdatedAt,
		ETag:      artifact.ETag,

		Name:      artifact.Name,
		JobID:     artifact.JobID,
		GroupName: artifact.GroupName,
		Path:      artifact.Path,
		HashType:  artifact.HashType,
		Hash:      artifact.Hash,
		Size:      artifact.Size,
		Mime:      artifact.Mime,
		Sealed:    artifact.Sealed,

		DataURL: routes.MakeArtifactsDataLink(rctx, artifact.ID),
	}
}

func MakeArtifacts(rctx routes.RequestContext, artifacts []*models.Artifact) []*Artifact {
	var docs []*Artifact
	for _, model := range artifacts {
		docs = append(docs, MakeArtifact(rctx, model))
	}
	return docs
}

func (m *Artifact) GetID() models.ResourceID {
	return m.ID.ResourceID
}

func (m *Artifact) GetKind() models.ResourceKind {
	return models.ArtifactResourceKind
}

func (m *Artifact) GetCreatedAt() models.Time {
	return m.CreatedAt
}

type ArtifactSearchRequest struct {
	*models.ArtifactSearch
}

func NewArtifactSearchRequest() *ArtifactSearchRequest {
	return &ArtifactSearchRequest{ArtifactSearch: models.NewArtifactSearch()}
}

func (d *ArtifactSearchRequest) Bind(r *http.Request) error {
	return d.Validate()
}

func (d *ArtifactSearchRequest) GetQuery() url.Values {
	values := makePaginationQueryParams(d.Pagination)
	if d.Workflow != nil && *d.Workflow != "" {
		values.Set("workflow", url.QueryEscape(d.Workflow.String()))
	}
	if d.JobName != nil && *d.JobName != "" {
		values.Set("job_name", url.QueryEscape(d.JobName.String()))
	}
	if d.GroupName != nil && *d.GroupName != "" {
		values.Set("group_name", url.QueryEscape((*d.GroupName).String()))
	}
	return values
}

func (d *ArtifactSearchRequest) FromQuery(values url.Values) error {
	pagination, err := getPaginationFromQueryParams(values)
	if err != nil {
		return fmt.Errorf("error parsing pagination: %w", err)
	}
	d.Pagination = pagination

	vals, ok := values["workflow"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping workflow: %w", err)
		}
		workflow := models.ResourceName(val)
		d.Workflow = &workflow
	}
	vals, ok = values["job_name"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping job name: %w", err)
		}
		name := models.ResourceName(val)
		d.JobName = &name
	}
	vals, ok = values["group_name"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping group name: %w", err)
		}
		name := models.ResourceName(val)
		d.GroupName = &name
	}
	return d.Validate()
}

func (d *ArtifactSearchRequest) Next(cursor *models.DirectionalCursor) PaginatedRequest {
	d.Cursor = cursor
	return d
}
