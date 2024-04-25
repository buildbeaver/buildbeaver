package server

import (
	"fmt"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
)

type LegalEntityAPI struct {
	legalEntityService services.LegalEntityService
	repoService        services.RepoService
	runnerService      services.RunnerService
	buildService       services.BuildService
	scmRegistry        *scm.SCMRegistry
	*APIBase
}

func NewLegalEntityAPI(
	legalEntityService services.LegalEntityService,
	runnerService services.RunnerService,
	repoService services.RepoService,
	buildService services.BuildService,
	scmRegistry *scm.SCMRegistry,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *LegalEntityAPI {
	return &LegalEntityAPI{
		legalEntityService: legalEntityService,
		runnerService:      runnerService,
		repoService:        repoService,
		buildService:       buildService,
		scmRegistry:        scmRegistry,
		APIBase:            NewAPIBase(authorizationService, resourceLinker, logFactory("LegalEntityAPI")),
	}
}

func (a *LegalEntityAPI) Get(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.AuthorizedLegalEntityID(r, models.LegalEntityReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	legalEntity, err := a.legalEntityService.Read(r.Context(), nil, legalEntityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeLegalEntity(routes.RequestCtx(r), legalEntity)
	a.GotResource(w, r, res)
}

func (a *LegalEntityAPI) GetCurrent(w http.ResponseWriter, r *http.Request) {
	meta := a.MustAuthenticationMeta(r)
	userLegalEntity, err := a.legalEntityService.ReadByIdentityID(r.Context(), nil, meta.IdentityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeLegalEntity(routes.RequestCtx(r), userLegalEntity)
	a.GotResource(w, r, res)
}

// GetSetupStatus reads and returns information about the extent to which the specified legal entity has been
// properly set up for use with BuildBeaver.
func (a *LegalEntityAPI) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	legalEntityID, err := a.AuthorizedLegalEntityID(r, models.LegalEntityReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	legalEntity, err := a.legalEntityService.Read(r.Context(), nil, legalEntityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	// Determine BuildBeaverInstalled value
	buildBeaverInstalled := false
	if legalEntity.ExternalID != nil {
		scmName := legalEntity.ExternalID.ExternalSystem
		externalSCM, err := a.scmRegistry.Get(scmName)
		if err != nil {
			a.Error(w, r, fmt.Errorf("error finding SCM %s: %w", scmName, err))
			return
		}
		if externalSCM != nil {
			buildBeaverInstalled, err = externalSCM.IsLegalEntityRegisteredAsUser(r.Context(), legalEntity)
			if err != nil {
				a.Error(w, r, fmt.Errorf("error asking SCM %s whether BuildBeaver is installed: %w", externalSCM.Name(), err))
				return
			}
		}
	}

	// Determine ReposEnabled value
	repoSearch := search.NewRepoQueryBuilder().
		WhereLegalEntityID(search.Equal, legalEntityID).
		WhereEnabled(search.Equal, true).
		Limit(1).
		Compile()
	repos, _, err := a.repoService.Search(r.Context(), nil, models.NoIdentity, repoSearch)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error determining whether any repos are enabled for legal entity '%s': %w", legalEntity.GetName(), err))
		return
	}
	reposEnabled := len(repos) > 0

	// Determine RunnersRegistered value
	runnerSearch := models.RunnerSearch{
		Pagination:    models.Pagination{Limit: 1},
		LegalEntityID: &legalEntity.ID,
	}
	runners, _, err := a.runnerService.Search(r.Context(), nil, models.NoIdentity, runnerSearch)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error determining whether any runners are registered for legal entity '%s': %w", legalEntity.GetName(), err))
		return
	}
	runnersRegistered := len(runners) > 0
	runnersSeen := anyRunnerSeen(runners)

	// Determine BuildsRun value
	buildSearch := models.BuildSearch{
		Pagination:    models.Pagination{Limit: 1},
		LegalEntityID: &legalEntity.ID,
	}
	builds, _, err := a.buildService.Search(r.Context(), nil, models.NoIdentity, &buildSearch)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error determining whether any builds have run for legal entity '%s': %w", legalEntity.GetName(), err))
		return
	}
	buildsRun := len(builds) > 0

	res := documents.MakeLegalEntitySetupStatus(
		routes.RequestCtx(r),
		legalEntity,
		buildBeaverInstalled,
		reposEnabled,
		runnersRegistered,
		runnersSeen,
		buildsRun,
	)
	a.GotResource(w, r, res)
}

// runnerSeen returns true if any runner in the set passed in has contacted the server successfully to update
// its details.
func anyRunnerSeen(allRunners []*models.Runner) bool {
	for _, runner := range allRunners {
		// The runner software version is only stored when the runner reports in to the server
		if runner.SoftwareVersion != "" {
			return true // runner has been seen / reported in to the server
		}
	}
	return false
}

// List returns all legal entities that the authenticated user belongs to (including the authenticated user's legal entity)
func (a *LegalEntityAPI) List(w http.ResponseWriter, r *http.Request) {
	search := documents.NewListRequest()
	err := search.FromQuery(r.URL.Query())
	if err != nil {
		a.Error(w, r, err)
		return
	}
	meta := a.MustAuthenticationMeta(r)
	userLegalEntity, err := a.legalEntityService.ReadByIdentityID(r.Context(), nil, meta.IdentityID)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	// Return the legal entities for which the current user is a member, plus the current user itself
	entities, cursor, err := a.legalEntityService.ListParentLegalEntities(r.Context(), nil, userLegalEntity.ID, search.Pagination)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	entities = append(entities, userLegalEntity)

	docs := documents.MakeLegalEntities(routes.RequestCtx(r), entities)
	res := documents.NewPaginatedResponse(models.LegalEntityResourceKind, routes.MakeLegalEntitiesLink(routes.RequestCtx(r)), search, docs, cursor)
	a.JSON(w, r, res)
}
