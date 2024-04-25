package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/hashicorp/go-multierror"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

// searchHandler searches a particular resource kind.
// Must enforce access control by passing the authenticated legal entity through to the underlying search function.
type searchHandler func(r *http.Request, req *documents.SearchRequest) (*documents.PaginatedResponse, error)

type parallelSearchResult struct {
	err  error
	kind models.ResourceKind
	res  *documents.PaginatedResponse
}

type SearchAPI struct {
	*APIBase
	repoService  services.RepoService
	buildService services.BuildService
}

func NewSearchAPI(
	authorizationService services.AuthorizationService,
	repoService services.RepoService,
	buildService services.BuildService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *SearchAPI {
	return &SearchAPI{
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("SearchAPI")),
		repoService:  repoService,
		buildService: buildService,
	}
}

func (a *SearchAPI) List(w http.ResponseWriter, r *http.Request) {
	req := documents.NewSearchRequest()
	err := req.FromQuery(r.URL.Query())
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// Fixed limit for universal search
	req.Limit = 5
	// Add more handlers here to expand support for universal search
	searches := map[models.ResourceKind]searchHandler{
		models.RepoResourceKind:  a.searchRepos,
		models.BuildResourceKind: a.searchBuilds,
	}
	var (
		nSearches int
		resC      = make(chan parallelSearchResult)
	)
	for kind, searchFn := range searches {
		if req.Kind == nil || *req.Kind == kind {
			nSearches++
			go func(kind models.ResourceKind, searchFn searchHandler) {
				res, err := searchFn(r, req)
				resC <- parallelSearchResult{err: err, kind: kind, res: res}
			}(kind, searchFn)
		}
	}
	var (
		results    []*documents.PaginatedResponse
		overallErr *multierror.Error
	)
	for i := 0; i < nSearches; i++ {
		res := <-resC
		if res.err != nil {
			overallErr = multierror.Append(overallErr, fmt.Errorf("error searching %s: %w", res.kind, res.err))
			continue
		}
		results = append(results, res.res)
	}
	if overallErr.ErrorOrNil() != nil {
		a.Error(w, r, overallErr.ErrorOrNil())
		return
	}
	res := documents.NewPaginatedResponse("", routes.MakeSearchLink(routes.RequestCtx(r)), req, results, nil)
	a.JSON(w, r, res)
}

func (a *SearchAPI) Search(w http.ResponseWriter, r *http.Request) {
	req := documents.NewSearchRequest()
	err := render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	next := documents.AddQueryParams(routes.MakeSearchLink(routes.RequestCtx(r)), req)
	http.Redirect(w, r, next.String(), http.StatusSeeOther)
}

func (a *SearchAPI) searchRepos(r *http.Request, req *documents.SearchRequest) (*documents.PaginatedResponse, error) {
	repos, _, err := a.repoService.Search(r.Context(), nil, a.MustAuthenticatedIdentityID(r), req.Query)
	if err != nil {
		return nil, err
	}
	docs := documents.MakeRepos(routes.RequestCtx(r), repos)
	res := documents.NewPaginatedResponse(models.RepoResourceKind, "", req, docs, nil)
	return res, nil
}

func (a *SearchAPI) searchBuilds(r *http.Request, req *documents.SearchRequest) (*documents.PaginatedResponse, error) {
	builds, _, err := a.buildService.UniversalSearch(r.Context(), nil, a.MustAuthenticatedIdentityID(r), req.Query)
	if err != nil {
		return nil, err
	}
	docs := documents.MakeBuildSearchResultsDocument(routes.RequestCtx(r), builds)
	res := documents.NewPaginatedResponse(models.BuildResourceKind, "", req, docs, nil)
	return res, nil
}
