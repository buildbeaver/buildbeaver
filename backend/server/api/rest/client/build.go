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

// GetBuildSummary retrieves the builds summary for a legal entity
func (a *APIClient) GetBuildSummary(ctx context.Context, legalEntityID models.LegalEntityID) (*documents.BuildSummary, error) {
	url := fmt.Sprintf("/api/v1/legal-entities/%s/builds/summary", legalEntityID)
	code, _, body, err := a.get(ctx, nil, url)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return nil, a.makeHTTPError(code, nil)
	}
	doc := &documents.BuildSummary{}
	err = json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return doc, nil
}
