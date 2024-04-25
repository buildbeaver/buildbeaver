package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type BuildAPI struct {
	buildService services.BuildService
	queueService services.QueueService
	eventService services.EventService
	commitStore  store.CommitStore
	*APIBase
}

func NewBuildAPI(
	authorizationService services.AuthorizationService,
	buildService services.BuildService,
	queueService services.QueueService,
	eventService services.EventService,
	commitStore store.CommitStore,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *BuildAPI {
	return &BuildAPI{
		buildService: buildService,
		queueService: queueService,
		eventService: eventService,
		commitStore:  commitStore,
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("BuildAPI")),
	}
}

func (a *BuildAPI) Get(w http.ResponseWriter, r *http.Request) {
	buildID, err := a.AuthorizedBuildID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	queuedBuild, err := a.queueService.ReadQueuedBuild(r.Context(), nil, buildID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeBuildGraph(routes.RequestCtx(r), queuedBuild)
	a.GotResource(w, r, res)
}

func (a *BuildAPI) Create(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.AuthorizedRepoID(r, models.BuildCreateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.CreateBuildRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error parsing request: %w", err))
		return
	}
	// Make sure the user is actually allowed to read the build they nominated
	err = a.Authorize(r, models.BuildReadOperation, req.FromBuildID.ResourceID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	build, err := a.buildService.Read(r.Context(), nil, *req.FromBuildID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	if !repoID.Equal(build.RepoID.ResourceID) {
		a.Error(w, r, gerror.NewErrValidationFailed("Cannot create a build from a different repo"))
		return
	}
	commit, err := a.commitStore.Read(r.Context(), nil, build.CommitID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	newBuild, err := a.queueService.EnqueueBuildFromCommit(r.Context(), nil, commit, build.Ref, req.Opts)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	queuedBuild, err := a.queueService.ReadQueuedBuild(r.Context(), nil, newBuild.ID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeBuildGraph(routes.RequestCtx(r), queuedBuild)
	a.CreatedResource(w, r, res, nil)
}

func (a *BuildAPI) List(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.RepoID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := documents.NewSearchRequest()
	err = req.FromQuery(r.URL.Query())
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// The build list is embedded under a repo in the API, so this search
	// must always be limited to repos for that legal entity.
	req.Query = search.NewBuildQueryBuilder(req.Query).WhereRepoID(search.Equal, repoID).Compile()
	builds, cursor, err := a.buildService.UniversalSearch(r.Context(), nil, a.MustAuthenticatedIdentityID(r), req.Query)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	docs := documents.MakeBuildSearchResultsDocument(routes.RequestCtx(r), builds)
	res := documents.NewPaginatedResponse(models.BuildResourceKind, routes.MakeBuildsLink(routes.RequestCtx(r), repoID), req, docs, cursor)
	a.JSON(w, r, res)
}

func (a *BuildAPI) Search(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.RepoID(r)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := documents.NewSearchRequest()
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	next := documents.AddQueryParams(routes.MakeBuildsLink(routes.RequestCtx(r), repoID), req)
	http.Redirect(w, r, next.String(), http.StatusSeeOther)
}

func (a *BuildAPI) Summary(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.AuthorizedLegalEntityID(r, models.LegalEntityReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	builds, err := a.buildService.Summary(r.Context(), nil, legalEntityID, a.MustAuthenticatedIdentityID(r))
	if err != nil {
		a.Error(w, r, err)
		return
	}

	docs := documents.MakeBuildSummary(routes.RequestCtx(r), builds)
	a.JSON(w, r, docs)
}

func (a *BuildAPI) GetEvents(w http.ResponseWriter, r *http.Request) {
	buildID, err := a.AuthorizedBuildID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	// Parse query parameters, if present
	var (
		lastEventNumber = models.EventNumber(0)
		limit           = 1000
	)
	queryParams := r.URL.Query()
	lastStr := queryParams.Get("last")
	if lastStr != "" {
		lastInt, err := strconv.Atoi(lastStr)
		if err != nil {
			a.Error(w, r, fmt.Errorf("error parsing query parameter 'last': %w", err))
			return
		}
		lastEventNumber = models.EventNumber(lastInt)
	}
	limitStr := queryParams.Get("limit")
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil {
			a.Error(w, r, fmt.Errorf("error parsing query parameter 'limit': %w", err))
			return
		}
		limit = limitInt
	}

	events, err := a.eventService.FetchEvents(r.Context(), nil, buildID, lastEventNumber, limit)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	eventsDoc := documents.MakeEvents(routes.RequestCtx(r), events)
	a.JSON(w, r, eventsDoc)
}
