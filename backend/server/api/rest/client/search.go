package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type searchResponse struct {
	Results json.RawMessage `json:"results"`
	NextURL string          `json:"next_url"`
}

type searchPaginator struct {
	apiClient *APIClient
	search    interface{}
	nextURL   string
}

func newSearchPaginator(apiClient *APIClient, url string, search interface{}) *searchPaginator {
	return &searchPaginator{
		apiClient: apiClient,
		search:    search,
		nextURL:   url,
	}
}

func (a *searchPaginator) HasNext() bool {
	return a.search != nil || a.nextURL != ""
}

func (a *searchPaginator) next(ctx context.Context) (json.RawMessage, error) {
	var body []byte
	if a.search != nil {
		code, _, buf, err := a.apiClient.post(ctx, nil, a.nextURL, a.search)
		if err != nil {
			return nil, fmt.Errorf("error in initial search: %w", err)
		}
		if !a.apiClient.isOneOf(code, []int{http.StatusOK, http.StatusSeeOther}) {
			return nil, a.apiClient.makeHTTPError(code, buf)
		}
		body = buf
	} else {
		code, _, buf, err := a.apiClient.get(ctx, nil, a.nextURL)
		if err != nil {
			return nil, fmt.Errorf("error in next: %w", err)
		}
		if !a.apiClient.isOneOf(code, []int{http.StatusOK}) {
			return nil, a.apiClient.makeHTTPError(code, buf)
		}
		body = buf
	}
	doc := &searchResponse{}
	err := json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	a.search = nil
	a.nextURL = doc.NextURL
	return doc.Results, nil
}
