package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type ArtifactAPI struct {
	artifactService services.ArtifactService
	*APIBase
}

func NewArtifactAPI(
	artifactService services.ArtifactService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *ArtifactAPI {
	return &ArtifactAPI{
		artifactService: artifactService,
		APIBase:         NewAPIBase(authorizationService, resourceLinker, logFactory("ArtifactAPI")),
	}
}

func (a *ArtifactAPI) Create(w http.ResponseWriter, r *http.Request) {
	jobID, err := a.AuthorizedJobID(r, models.ArtifactCreateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	path := r.Header.Get("X-BuildBeaver-Artifact-Path")
	group := r.Header.Get("X-BuildBeaver-Artifact-Group")
	md5 := r.Header.Get("Content-MD5")
	artifact, err := a.artifactService.Create(r.Context(), jobID, models.ResourceName(group), path, md5, r.Body, true)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeArtifact(routes.RequestCtx(r), artifact)
	a.CreatedResource(w, r, res, nil)
}

func (a *ArtifactAPI) Get(w http.ResponseWriter, r *http.Request) {
	artifactID, err := a.AuthorizedArtifactID(r, models.ArtifactReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	artifact, err := a.artifactService.Read(r.Context(), nil, artifactID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeArtifact(routes.RequestCtx(r), artifact)
	a.GotResource(w, r, res)
}

func (a *ArtifactAPI) GetData(w http.ResponseWriter, r *http.Request) {
	artifactID, err := a.AuthorizedArtifactID(r, models.ArtifactReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	reader, err := a.artifactService.GetArtifactData(r.Context(), artifactID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	defer reader.Close()

	artifact, err := a.artifactService.Read(r.Context(), nil, artifactID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	_, file := filepath.Split(artifact.Path)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, reader)
	if err != nil {
		a.Errorf("error writing artifact data to response body: %w", err)
	}
}

func (a *ArtifactAPI) List(w http.ResponseWriter, r *http.Request) {
	buildID, err := a.BuildID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	search := documents.NewArtifactSearchRequest()
	err = search.FromQuery(r.URL.Query())
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// The artifact list is embedded under a build in the API, so this search
	// is always filtered to artifacts for that build.
	search.BuildID = buildID
	artifacts, cursor, err := a.artifactService.Search(r.Context(), nil, a.MustAuthenticatedIdentityID(r), *search.ArtifactSearch)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	link := routes.MakeArtifactsLink(routes.RequestCtx(r), buildID)
	// HACK: TODO: Remove when runners have their own identity
	// TODO: Come up with a better mechanism to provide appropriate links for runners and dynamic API
	if strings.Contains(r.URL.Path, "/api/v1/runner/") {
		link = routes.MakeArtifactsLinkForRunner(routes.RequestCtx(r), buildID)
	}

	docs := documents.MakeArtifacts(routes.RequestCtx(r), artifacts)
	res := documents.NewPaginatedResponse(models.ArtifactResourceKind, link, search, docs, cursor)
	a.JSON(w, r, res)
}

func (a *ArtifactAPI) Search(w http.ResponseWriter, r *http.Request) {
	buildID, err := a.BuildID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	search := documents.NewArtifactSearchRequest()
	err = render.Bind(r, search)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// The artifact list is embedded under a build in the API, so this search
	// is always filtered to artifacts for that build.
	search.BuildID = buildID
	link := routes.MakeArtifactsLink(routes.RequestCtx(r), buildID)

	// HACK: Remove when runners have their own identity
	if strings.Contains(r.URL.Path, "/api/v1/runner/") {
		link = routes.MakeArtifactsLinkForRunner(routes.RequestCtx(r), buildID)
	}

	next := documents.AddQueryParams(link, search)
	http.Redirect(w, r, next.String(), http.StatusSeeOther)
}
