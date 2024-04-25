package documents

import (
	"net/url"
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
)

const (
	defaultLimit = 30
)

type PaginatedRequest interface {
	GetQuery() url.Values
	FromQuery(url.Values) error
	Next(cursor *models.DirectionalCursor) PaginatedRequest
}

type PaginatedResponse struct {
	// Thw kind of objects contained in the results
	Kind models.ResourceKind `json:"kind,omitempty"`
	// A set of results, normally an array of objects
	Results interface{} `json:"results"`
	// A URL to fetch to obtain the previous page of results before this one
	PrevURL string `json:"prev_url"`
	// A cursor that can be used as a query parameter to obtain the previous page of results before this one
	PrevCursor string `json:"prev_cursor"`
	// A URL to fetch to obtain the next page of results after this one
	NextURL string `json:"next_url"`
	// A cursor that can be used as a query parameter to obtain the next page of results after this one
	NextCursor string `json:"next_cursor"`
}

func NewPaginatedResponse(
	kind models.ResourceKind,
	link string,
	req PaginatedRequest,
	results interface{},
	cursor *models.Cursor) *PaginatedResponse {
	res := &PaginatedResponse{Kind: kind, Results: results}
	if cursor != nil && req != nil {
		if cursor.Prev != nil {
			res.PrevURL = AddQueryParams(link, req.Next(cursor.Prev)).String()
			cursorStr, err := cursor.Prev.Encode()
			if err == nil && cursorStr != "" {
				res.PrevCursor = cursorStr
			}
		}
		if cursor.Next != nil {
			res.NextURL = AddQueryParams(link, req.Next(cursor.Next)).String()
			cursorStr, err := cursor.Next.Encode()
			if err == nil && cursorStr != "" {
				res.NextCursor = cursorStr
			}
		}
	}
	return res
}

// AddQueryParams returns a new url with pagination parameters added.
func AddQueryParams(searchURL string, req PaginatedRequest) *url.URL {
	u, err := url.Parse(searchURL)
	if err != nil {
		panic(err)
	}
	query := u.Query()
	for key, value := range req.GetQuery() {
		for _, v := range value {
			query.Set(key, v)
		}
	}
	u.RawQuery = query.Encode()
	return u
}

func makePaginationQueryParams(pagination models.Pagination) url.Values {
	values := make(url.Values)
	if pagination.Cursor != nil {
		cursorStr, err := pagination.Cursor.Encode()
		if err == nil && cursorStr != "" {
			values.Set("cursor", url.QueryEscape(cursorStr))
		}
	}
	limit := pagination.Limit
	if limit < 1 || limit > 50 {
		limit = defaultLimit
	}
	values.Set("limit", url.QueryEscape(strconv.Itoa(limit)))
	return values
}

func getPaginationFromQueryParams(values url.Values) (models.Pagination, error) {
	pagination := models.Pagination{}
	cursorStr, err := url.QueryUnescape(values.Get("cursor"))
	if err != nil {
		return models.Pagination{}, gerror.NewErrInvalidQueryParameter("error unescaping cursor").Wrap(err)
	}
	if cursorStr != "" {
		pagination.Cursor = &models.DirectionalCursor{}
		err := pagination.Cursor.Decode(cursorStr)
		if err != nil {
			return models.Pagination{}, gerror.NewErrInvalidQueryParameter("error decoding cursor").Wrap(err)
		}
	}
	limitStr := values.Get("limit")
	if limitStr == "" {
		pagination.Limit = defaultLimit
	} else {
		var err error
		pagination.Limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return models.Pagination{}, gerror.NewErrInvalidQueryParameter("error decoding limit").Wrap(err)
		}
	}
	return pagination, nil
}
