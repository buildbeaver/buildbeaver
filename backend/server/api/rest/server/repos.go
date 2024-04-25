package server

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type RepoAPI struct {
	legalEntityService services.LegalEntityService
	repoService        services.RepoService
	*APIBase
}

func NewRepoAPI(
	repoService services.RepoService,
	legalEntityService services.LegalEntityService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *RepoAPI {
	return &RepoAPI{
		repoService:        repoService,
		legalEntityService: legalEntityService,
		APIBase:            NewAPIBase(authorizationService, resourceLinker, logFactory("RepoAPI")),
	}
}

func (a *RepoAPI) Get(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.AuthorizedRepoID(r, models.RepoReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	repo, err := a.repoService.Read(r.Context(), nil, repoID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeRepo(routes.RequestCtx(r), repo)
	a.GotResource(w, r, res)
}

func (a *RepoAPI) Patch(w http.ResponseWriter, r *http.Request) {
	repoID, err := a.AuthorizedRepoID(r, models.RepoUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.PatchRepoRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	var repo *models.Repo
	if req.Enabled != nil {
		repo, err = a.repoService.UpdateRepoEnabled(r.Context(), repoID, dto.UpdateRepoEnabled{
			Enabled: *req.Enabled,
			ETag:    a.GetIfMatch(r),
		})
		if err != nil {
			a.Error(w, r, err)
			return
		}
	}
	res := documents.MakeRepo(routes.RequestCtx(r), repo)
	a.UpdatedResource(w, r, res, nil)
}

func (a *RepoAPI) List(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.LegalEntityID(r)
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
	// The repo list is embedded under a legal entity in the API, so this search
	// must always be limited to repos for that legal entity.
	req.Query = search.NewRepoQueryBuilder(req.Query).WhereLegalEntityID(search.Equal, legalEntityID).Compile()
	repos, cursor, err := a.repoService.Search(r.Context(), nil, a.MustAuthenticatedIdentityID(r), req.Query)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	docs := documents.MakeRepos(routes.RequestCtx(r), repos)
	res := documents.NewPaginatedResponse(models.RepoResourceKind, routes.MakeReposLink(routes.RequestCtx(r), legalEntityID), req, docs, cursor)
	a.JSON(w, r, res)
}

func (a *RepoAPI) Search(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.LegalEntityID(r)
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
	next := documents.AddQueryParams(routes.MakeReposLink(routes.RequestCtx(r), legalEntityID), req)
	http.Redirect(w, r, next.String(), http.StatusSeeOther)
}
