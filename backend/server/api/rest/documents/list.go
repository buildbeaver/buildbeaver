package documents

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/buildbeaver/buildbeaver/common/models"
)

type ListRequest struct {
	models.Pagination
}

func NewListRequest() *ListRequest {
	return &ListRequest{Pagination: models.Pagination{}}
}

func (d *ListRequest) Bind(r *http.Request) error {
	return nil
}

func (d *ListRequest) GetQuery() url.Values {
	return makePaginationQueryParams(d.Pagination)
}

func (d *ListRequest) FromQuery(values url.Values) error {
	pagination, err := getPaginationFromQueryParams(values)
	if err != nil {
		return fmt.Errorf("error parsing pagination: %w", err)
	}
	d.Pagination = pagination
	return nil
}

func (d *ListRequest) Next(cursor *models.DirectionalCursor) PaginatedRequest {
	d.Cursor = cursor
	return d
}
