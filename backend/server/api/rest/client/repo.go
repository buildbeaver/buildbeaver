package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type repoSearchPaginator struct {
	*searchPaginator
}

func newRepoSearchPaginator(
	apiClient *APIClient,
	url string,
	search *documents.SearchRequest) *repoSearchPaginator {
	return &repoSearchPaginator{
		searchPaginator: newSearchPaginator(apiClient, url, search),
	}
}

func (a *repoSearchPaginator) Next(ctx context.Context) ([]*models.Repo, error) {
	raw, err := a.next(ctx)
	if err != nil {
		return nil, err
	}
	var results []*models.Repo
	err = json.Unmarshal(raw, &results)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling artifacts: %w", err)
	}
	return results, nil
}

func (a *APIClient) SearchRepos(ctx context.Context, legalEntityID models.LegalEntityID, query search.Query) (models.RepoSearchPaginator, error) {
	doc := &documents.SearchRequest{
		Query: query,
	}
	url := fmt.Sprintf("/api/v1/legal-entities/%s/repos/search", legalEntityID)
	paginator := newRepoSearchPaginator(a, url, doc)
	return paginator, nil
}
