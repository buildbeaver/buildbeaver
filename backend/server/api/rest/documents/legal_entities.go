package documents

import (
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type LegalEntity struct {
	baseResourceDocument
	RepoSearchURL   string `json:"repo_search_url"`
	RunnerSearchURL string `json:"runner_search_url"`
	BuildSummaryURL string `json:"build_summary_url"`
	// TODO: Include all model fields directly in this object
	*models.LegalEntity
}

func MakeLegalEntity(rctx routes.RequestContext, legalEntity *models.LegalEntity) *LegalEntity {
	return &LegalEntity{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeLegalEntityLink(rctx, legalEntity.ID),
		},
		RepoSearchURL:   routes.MakeRepoSearchLink(rctx, legalEntity.ID),
		RunnerSearchURL: routes.MakeRunnerSearchLink(rctx, legalEntity.ID),
		BuildSummaryURL: routes.MakeBuildSummaryLink(rctx, legalEntity.ID),
		LegalEntity:     legalEntity,
	}
}

func MakeLegalEntities(rctx routes.RequestContext, list []*models.LegalEntity) []*LegalEntity {
	var docs []*LegalEntity
	for _, model := range list {
		docs = append(docs, MakeLegalEntity(rctx, model))
	}
	return docs
}

type LegalEntitySetupStatus struct {
	baseResourceDocument
	ID        models.LegalEntityID `json:"id"`
	CreatedAt models.Time          `json:"created_at"`
	// True if the BuildBeaver GitHub app has been installed for this Legal Entity (user or org)
	BuildBeaverInstalled bool `json:"buildbeaver_installed"`
	// True if one or more repos belonging to this Legal Entity have been enabled
	ReposEnabled bool `json:"repos_enabled"`
	// True if there are one or more runners registered for this Legal Entity.
	RunnersRegistered bool `json:"runners_registered"`
	// True if one or more of the currently registered runners have successfully reported in to the server.
	RunnersSeen bool `json:"runners_seen"`
	// True if one or more builds have been queued for this Legal Entity, implying there has been a
	// yaml config file set up in the repo.
	BuildsRun bool `json:"builds_run"`
}

func MakeLegalEntitySetupStatus(
	rctx routes.RequestContext,
	legalEntity *models.LegalEntity,
	buildBeaverInstalled bool,
	reposEnabled bool,
	runnersRegistered bool,
	runnersSeen bool,
	buildsRun bool,
) *LegalEntitySetupStatus {
	return &LegalEntitySetupStatus{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeLegalEntityLink(rctx, legalEntity.ID),
		},
		ID:        legalEntity.ID,
		CreatedAt: legalEntity.CreatedAt,

		BuildBeaverInstalled: buildBeaverInstalled,
		ReposEnabled:         reposEnabled,
		RunnersRegistered:    runnersRegistered,
		RunnersSeen:          runnersSeen,
		BuildsRun:            buildsRun,
	}
}

func (s *LegalEntitySetupStatus) GetID() models.ResourceID {
	return s.ID.ResourceID
}

func (s *LegalEntitySetupStatus) GetKind() models.ResourceKind {
	return models.LegalEntityResourceKind
}

func (s *LegalEntitySetupStatus) GetCreatedAt() models.Time {
	return s.CreatedAt
}
