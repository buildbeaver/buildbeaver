package documents

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Runner struct {
	baseResourceDocument

	ID        models.RunnerID `json:"id"`
	CreatedAt models.Time     `json:"created_at"`
	UpdatedAt models.Time     `json:"updated_at"`
	DeletedAt *models.Time    `json:"deleted_at,omitempty"`
	ETag      models.ETag     `json:"etag" hash:"ignore"`

	// Name of the runner, supplied during registration. Must be unique within the owning legal entity.
	Name models.ResourceName `json:"name"`
	// LegalEntityID is the ID of the Legal Entity (user or organization) whose builds this runner will execute.
	LegalEntityID models.LegalEntityID `json:"legal_entity_id"`
	// SoftwareVersion is the software version of the runner process.
	SoftwareVersion string `json:"software_version"`
	// OperatingSystem is the operating system the runner process is currently running on.
	OperatingSystem string `json:"operating_system"`
	// Architecture is the processor architecture the runner process is currently running on.
	Architecture string `json:"architecture"`
	// SupportedJobTypes is the one or more job types this runner supports.
	SupportedJobTypes []models.JobType `json:"supported_job_types"`
	// Labels contains the set of labels this runner is configured with.
	Labels []models.Label `json:"labels"`
	// Enabled specifies if this runner is available to process jobs.
	Enabled bool `json:"enabled" db:"runner_enabled"`
}

func MakeRunner(rctx routes.RequestContext, runner *models.Runner) *Runner {
	return &Runner{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeRunnerLink(rctx, runner.ID),
		},

		ID:        runner.ID,
		CreatedAt: runner.CreatedAt,
		UpdatedAt: runner.UpdatedAt,
		DeletedAt: runner.DeletedAt,
		ETag:      runner.ETag,

		Name:              runner.Name,
		LegalEntityID:     runner.LegalEntityID,
		SoftwareVersion:   runner.SoftwareVersion,
		OperatingSystem:   runner.OperatingSystem,
		Architecture:      runner.Architecture,
		SupportedJobTypes: runner.SupportedJobTypes,
		Labels:            runner.Labels,
		Enabled:           runner.Enabled,
	}
}

func MakeRunners(rctx routes.RequestContext, runners []*models.Runner) []*Runner {
	var docs []*Runner
	for _, model := range runners {
		docs = append(docs, MakeRunner(rctx, model))
	}
	return docs
}

func (r *Runner) GetID() models.ResourceID {
	return r.ID.ResourceID
}

func (r *Runner) GetKind() models.ResourceKind {
	return models.RunnerResourceKind
}

func (r *Runner) GetCreatedAt() models.Time {
	return r.CreatedAt
}

type CreateRunnerRequest struct {
	Name                 models.ResourceName `json:"name"`
	ClientCertificatePEM string              `json:"client_certificate_pem"`
}

func (d *CreateRunnerRequest) Bind(r *http.Request) error {
	if d.Name == "" {
		return gerror.NewErrValidationFailed("Key must be specified")
	}
	if d.ClientCertificatePEM == "" {
		return gerror.NewErrValidationFailed("Client certificate must be specified")
	}
	err := certificates.ValidatePEMDataAsX509Certificate(d.ClientCertificatePEM)
	if err != nil {
		return err
	}

	return nil
}

type PatchRunnerRequest struct {
	Name    *models.ResourceName `json:"name"`
	Enabled *bool                `json:"enabled"`
}

func (d *PatchRunnerRequest) Bind(r *http.Request) error {
	if d.Enabled == nil && d.Name == nil {
		return gerror.NewErrValidationFailed("Enabled and/or Name must be specified")
	}
	return nil
}

type PatchRuntimeInfoRequest struct {
	SoftwareVersion   *string          `json:"software_version"`
	OperatingSystem   *string          `json:"operating_system"`
	Architecture      *string          `json:"architecture"`
	SupportedJobTypes *models.JobTypes `json:"supported_job_types"`
}

func (d *PatchRuntimeInfoRequest) Bind(r *http.Request) error {
	return nil
}

type RunnerSearchRequest struct {
	*models.RunnerSearch
}

func NewRunnerSearchRequest() *RunnerSearchRequest {
	return &RunnerSearchRequest{
		RunnerSearch: models.NewRunnerSearch(),
	}
}

func (d *RunnerSearchRequest) Bind(r *http.Request) error {
	return d.Validate()
}

func (d *RunnerSearchRequest) GetQuery() url.Values {
	values := makePaginationQueryParams(d.Pagination)
	return values
}

func (d *RunnerSearchRequest) FromQuery(values url.Values) error {
	pagination, err := getPaginationFromQueryParams(values)
	if err != nil {
		return fmt.Errorf("error parsing pagination: %w", err)
	}
	d.Pagination = pagination
	return d.Validate()
}

func (d *RunnerSearchRequest) Next(cursor *models.DirectionalCursor) PaginatedRequest {
	d.Cursor = cursor
	return d
}
