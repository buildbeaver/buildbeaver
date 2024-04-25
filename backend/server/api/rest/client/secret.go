package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type paginatedSecretResponse struct {
	*documents.PaginatedResponse
	Results []*models.SecretPlaintext `json:"results"` // TODO these should be documents
}

// GetSecretsPlaintext gets all secrets for the specified repo in plaintext.
func (a *APIClient) GetSecretsPlaintext(ctx context.Context, repoID models.RepoID) ([]*models.SecretPlaintext, error) {
	url := fmt.Sprintf("/api/v1/runner/repos/%s/secrets", repoID)
	code, _, body, err := a.get(ctx, nil, url)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return nil, a.makeHTTPError(code, body)
	}
	doc := &paginatedSecretResponse{}
	err = json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return doc.Results, nil
}
