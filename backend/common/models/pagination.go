package models

const DefaultPaginationLimit = 30

type Pagination struct {
	// Limit is the maximum number of results to return.
	Limit int `json:"limit"`
	// Cursor is an opaque value used to retrieve the next set of results.
	Cursor *DirectionalCursor `json:"cursor"`
}

func NewPagination(limit int, cursor *DirectionalCursor) Pagination {
	return Pagination{
		Limit:  limit,
		Cursor: cursor,
	}
}
