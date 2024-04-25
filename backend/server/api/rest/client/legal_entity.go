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

// GetLegalEntity gets a legal entity by ID.
func (a *APIClient) GetLegalEntity(ctx context.Context, legalEntityID models.LegalEntityID) (*documents.LegalEntity, error) {
	url := fmt.Sprintf("/api/v1/legal-entities/%s", legalEntityID)
	code, _, body, err := a.get(ctx, nil, url)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return nil, a.makeHTTPError(code, body)
	}
	doc := &documents.LegalEntity{}
	err = json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return doc, nil
}
