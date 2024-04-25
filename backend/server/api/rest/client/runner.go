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

// CreateRunner registers a new runner owned by the specified legal entity.
func (a *APIClient) CreateRunner(
	ctx context.Context,
	legalEntityID models.LegalEntityID,
	runnerName models.ResourceName,
	clientCertificatePEM string,
) (*documents.Runner, error) {
	url := fmt.Sprintf("/api/v1/legal-entities/%s/runners/", legalEntityID)
	doc := &documents.CreateRunnerRequest{
		Name:                 runnerName,
		ClientCertificatePEM: clientCertificatePEM,
	}
	code, _, body, err := a.post(ctx, nil, url, doc)
	if err != nil {
		return nil, fmt.Errorf("error in request: %w", err)
	}
	if !a.isOneOf(code, []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}) {
		return nil, a.makeHTTPError(code, body)
	}

	resDoc := &documents.Runner{}
	err = json.Unmarshal(body, resDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}

	return resDoc, nil
}

// GetRunner reads a runner registration.
func (a *APIClient) GetRunner(ctx context.Context, runnerID models.RunnerID) (*documents.Runner, error) {
	url := fmt.Sprintf("/api/v1/runners/%s", runnerID)
	code, _, body, err := a.get(ctx, nil, url)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return nil, a.makeHTTPError(code, nil)
	}
	doc := &documents.Runner{}
	err = json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return doc, nil
}
