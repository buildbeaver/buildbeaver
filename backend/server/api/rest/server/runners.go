package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type RunnerAPI struct {
	runnerService services.RunnerService
	*APIBase
}

func NewRunnerAPI(
	runnerService services.RunnerService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *RunnerAPI {
	return &RunnerAPI{
		runnerService: runnerService,
		APIBase:       NewAPIBase(authorizationService, resourceLinker, logFactory("RunnerAPI")),
	}
}

func (a *RunnerAPI) Get(w http.ResponseWriter, r *http.Request) {
	runnerID, err := a.AuthorizedRunnerID(r, models.RunnerReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	runner, err := a.runnerService.Read(r.Context(), nil, runnerID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeRunner(routes.RequestCtx(r), runner)
	a.GotResource(w, r, res)
}

func (a *RunnerAPI) Create(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.AuthorizedLegalEntityID(r, models.RunnerCreateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.CreateRunnerRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error reading CreateRunnerRequest from request: %w", err))
		return
	}

	// Get certificate data from supplied PEM string
	// The data is an ASN.1 DER-encoded X.509 certificate.
	certData, err := certificates.GetEncodedCertificateFromPEMData(req.ClientCertificatePEM)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	runner := models.NewRunner(
		models.NewTime(time.Now()),
		req.Name,
		legalEntityID,
		"",   // this field gets updated when runner updates its runtime info
		"",   // this field gets updated when runner updates its runtime info
		"",   // this field gets updated when runner updates its runtime info
		nil,  // this field gets updated when runner updates its runtime info
		nil,  // no labels need to be specified
		true, // enable runner by default
	)
	err = a.runnerService.Create(r.Context(), nil, runner, certData)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	doc := documents.MakeRunner(routes.RequestCtx(r), runner)
	a.CreatedResource(w, r, doc, nil)
}

func (a *RunnerAPI) Patch(w http.ResponseWriter, r *http.Request) {
	runnerID, err := a.AuthorizedRunnerID(r, models.RunnerUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.PatchRunnerRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	runner, err := a.runnerService.Read(r.Context(), nil, runnerID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	if req.Name != nil {
		runner.Name = *req.Name
	}
	if req.Enabled != nil {
		runner.Enabled = *req.Enabled
	}
	etag := a.GetIfMatch(r)
	if etag != "" {
		runner.ETag = etag
	}
	runner, err = a.runnerService.Update(r.Context(), nil, runner)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeRunner(routes.RequestCtx(r), runner)
	a.UpdatedResource(w, r, res, nil)
}

func (a *RunnerAPI) PatchRuntimeInfo(w http.ResponseWriter, r *http.Request) {
	// This API function must be called by a runner. Read the runner associated with currently authenticated identity.
	meta := a.MustAuthenticationMeta(r)
	runner, err := a.runnerService.ReadByIdentityID(r.Context(), nil, meta.IdentityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	req := &documents.PatchRuntimeInfoRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	if req.SoftwareVersion != nil {
		runner.SoftwareVersion = *req.SoftwareVersion
	}
	if req.OperatingSystem != nil {
		runner.OperatingSystem = *req.OperatingSystem
	}
	if req.Architecture != nil {
		runner.Architecture = *req.Architecture
	}
	if req.SupportedJobTypes != nil {
		runner.SupportedJobTypes = *req.SupportedJobTypes
	}
	etag := a.GetIfMatch(r)
	if etag != "" {
		runner.ETag = etag
	}
	runner, err = a.runnerService.Update(r.Context(), nil, runner)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeRunner(routes.RequestCtx(r), runner)
	a.UpdatedResource(w, r, res, nil)
}

func (a *RunnerAPI) Delete(w http.ResponseWriter, r *http.Request) {
	runnerID, err := a.AuthorizedRunnerID(r, models.RunnerDeleteOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	err = a.runnerService.SoftDelete(r.Context(), nil, runnerID,
		dto.DeleteRunner{ETag: a.GetIfMatch(r)})
	if err != nil {
		a.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *RunnerAPI) List(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.LegalEntityID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	search := documents.NewRunnerSearchRequest()
	err = search.FromQuery(r.URL.Query())
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// The runners list is embedded under a legal entity in the API, so this search
	// is always filtered to runners for that legal entity.
	search.LegalEntityID = &legalEntityID
	runners, cursor, err := a.runnerService.Search(r.Context(), nil, a.MustAuthenticatedIdentityID(r), *search.RunnerSearch)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	docs := documents.MakeRunners(routes.RequestCtx(r), runners)
	res := documents.NewPaginatedResponse(models.RunnerResourceKind, routes.MakeRunnersLink(routes.RequestCtx(r), legalEntityID), search, docs, cursor)
	a.JSON(w, r, res)
}

func (a *RunnerAPI) Search(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.LegalEntityID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	search := documents.NewRunnerSearchRequest()
	err = render.Bind(r, search)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// The runners list is embedded under a legal entity in the API, so this search
	// is always filtered to runners for that legal entity.
	search.LegalEntityID = &legalEntityID
	next := documents.AddQueryParams(routes.MakeRunnersLink(routes.RequestCtx(r), legalEntityID), search)
	http.Redirect(w, r, next.String(), http.StatusSeeOther)
}
