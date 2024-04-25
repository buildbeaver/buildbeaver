package documents

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
)

type SearchRequest struct {
	// TODO: Include all model fields directly in this object
	search.Query
}

func NewSearchRequest() *SearchRequest {
	return &SearchRequest{Query: search.NewQuery()}
}

func (d *SearchRequest) Bind(r *http.Request) error {
	return d.Validate()
}

func (d *SearchRequest) GetQuery() url.Values {
	values := makePaginationQueryParams(d.Pagination)
	values.Set("q", url.QueryEscape(d.String()))
	return values
}

func (d *SearchRequest) FromQuery(values url.Values) error {
	vals, ok := values["q"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping query: %w", err)
		}
		d.Query = search.ParseQuery(val)
	}
	// This has to come after parsing `q` above, or it will be clobbered
	pagination, err := getPaginationFromQueryParams(values)
	if err != nil {
		return fmt.Errorf("error parsing pagination: %w", err)
	}
	d.Pagination = pagination
	return d.Validate()
}

func (d *SearchRequest) Validate() error {
	return nil
}

func (d *SearchRequest) Next(cursor *models.DirectionalCursor) PaginatedRequest {
	d.Cursor = cursor
	return d
}
