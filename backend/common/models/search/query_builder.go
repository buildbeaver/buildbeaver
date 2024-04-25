package search

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/models"
)

// QueryBuilder makes it convenient to programmatically create search queries.
type QueryBuilder struct {
	query Query
}

func NewQueryBuilder(existing ...Query) *QueryBuilder {
	var query Query
	switch len(existing) {
	case 0:
		query = NewQuery()
	case 1:
		query = existing[0]
	default:
		panic("expected zero or one existing queries")
	}
	return &QueryBuilder{query: query}
}

// Kind sets the kind to filter search results to.
func (b *QueryBuilder) Kind(kind models.ResourceKind) *QueryBuilder {
	b.query.Kind = &kind
	return b
}

// Term adds a term to search for In() fields.
func (b *QueryBuilder) Term(term Term) *QueryBuilder {
	if b.query.Term == nil {
		b.query.Term = &term
	} else {
		term := Term(fmt.Sprintf("%s %s", *b.query.Term, term))
		b.query.Term = &term
	}
	return b
}

// In records a field to search for term in.
// In() fields are ORd together.
// If no In() fields are set, all fields are searched.
func (b *QueryBuilder) In(field FieldName) *QueryBuilder {
	if _, ok := b.query.fieldsByFieldName[field]; !ok {
		b.query.Fields = append(b.query.Fields, field)
		b.query.fieldsByFieldName[field] = field
	}
	return b
}

// Where records a field filter to constrain search results to.
// Where() fields are ANDd together.
func (b *QueryBuilder) Where(field FieldName, operator Operator, value string) *QueryBuilder {
	if _, ok := b.query.fieldFiltersByFieldName[field]; !ok {
		filter := NewFieldFilter(field, operator, value)
		b.query.Filters = append(b.query.Filters, filter)
		b.query.fieldFiltersByFieldName[field] = filter
	}
	return b
}

// Sort sets the field to sort search results on.
// If no Sort() is set, the default sort will be applied (varies per kind).
func (b *QueryBuilder) Sort(field string, direction ...SortDirection) *QueryBuilder {
	dir := Ascending
	switch len(direction) {
	case 0:
	case 1:
		dir = direction[0]
	default:
		panic("direction can be specified zero or one times")
	}
	b.query.Sort = NewSortField(field, dir)
	return b
}

// Limit sets the page size limit.
func (b *QueryBuilder) Limit(limit int) *QueryBuilder {
	b.query.Limit = limit
	return b
}

// Compile outputs the built query.
func (b *QueryBuilder) Compile() Query {
	return b.query
}
