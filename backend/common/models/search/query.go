package search

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buildbeaver/buildbeaver/common/models"
)

const (
	Equal                Operator = "="
	NotEqual             Operator = "!="
	GreaterThan          Operator = ">"
	GreaterThanOrEqualTo Operator = ">="
	LessThan             Operator = "<"
	LessThanOrEqualTo    Operator = "<="
)

const (
	Ascending  SortDirection = "asc"
	Descending SortDirection = "desc"
)

// operatorSet is the set of supported operators sorted by string length.
// Use to do prefix matching during query parsing, where we need to evaluate
// the longest prefix matches first. Sorted by the init() func below as a backup.
var operatorSet = []Operator{
	NotEqual,
	GreaterThanOrEqualTo,
	LessThanOrEqualTo,
	Equal,
	GreaterThan,
	LessThan,
}

func init() {
	sort.SliceStable(operatorSet, func(i, j int) bool {
		return len(operatorSet[i]) > len(operatorSet[j])
	})
}

type FieldName string

type SortDirection string

type Operator string

func (o Operator) String() string {
	return string(o)
}

// AsGoqu returns the operator formatted as a goqu-compatible string.
func (o Operator) AsGoqu() string {
	switch o {
	case Equal:
		return "eq"
	case NotEqual:
		return "neq"
	case GreaterThan:
		return "gt"
	case GreaterThanOrEqualTo:
		return "gte"
	case LessThan:
		return "lt"
	case LessThanOrEqualTo:
		return "lte"
	default:
		panic(fmt.Sprintf("unsupported op: %s", o))
	}
}

// Query describes a search against a resource kind.
type Query struct {
	models.Pagination
	// Kind is the kind to filter search results to.
	Kind *models.ResourceKind `json:"kind"`
	// Term will be searched for in Fields.
	Term *Term `json:"term"`
	// Fields nominates the set of fields to search for term in.
	// If no fields are set, all fields are searched.
	Fields []FieldName `json:"fields"`
	// Filters nominates zero or more fields to filter search results on.
	Filters []*FieldFilter `json:"filters"`
	// sort sets the field to sort search results on, or nil to use a default sort.
	Sort *SortField `json:"sort"`
	// fieldsByFieldName is Fields keyed by field name.
	fieldsByFieldName map[FieldName]FieldName
	// fieldFiltersByFieldName is Filters keyed by field name.
	fieldFiltersByFieldName map[FieldName]*FieldFilter
}

func NewQuery() Query {
	return Query{
		Pagination:              models.Pagination{Limit: models.DefaultPaginationLimit},
		fieldsByFieldName:       map[FieldName]FieldName{},
		fieldFiltersByFieldName: map[FieldName]*FieldFilter{},
	}
}

func (q *Query) AnyInFieldsSet() bool {
	return len(q.fieldsByFieldName) > 0
}

func (q *Query) IsInFieldSet(fieldName FieldName) bool {
	_, ok := q.fieldsByFieldName[fieldName]
	return ok
}

func (q *Query) GetFilter(fieldName FieldName) *FieldFilter {
	return q.fieldFiltersByFieldName[fieldName]
}

func (q *Query) Validate() error {
	return nil
}

// String returns query as a plaintext query string.
func (q *Query) String() string {
	str := ""
	if q.Term != nil {
		str += q.Term.String()
	}
	if q.Kind != nil {
		str += fmt.Sprintf(" kind:%s", q.Kind.String())
	}
	for _, field := range q.fieldsByFieldName {
		str += fmt.Sprintf(" in:%s", field)
	}
	for _, filter := range q.fieldFiltersByFieldName {
		if filter.Operator == Equal { // Default operator can be omitted
			str += fmt.Sprintf(" %s:%s", filter.Field, filter.Value)
		} else {
			str += fmt.Sprintf(" %s:%s%s", filter.Field, filter.Operator, filter.Value)
		}
	}
	if q.Sort != nil {
		str += fmt.Sprintf(" sort:%s-%s", q.Sort.Field, q.Sort.Direction)
	}
	return strings.Trim(str, " ")
}

func (q *Query) UnmarshalJSON(data []byte) error {
	x := struct {
		models.Pagination
		Kind    *models.ResourceKind `json:"kind"`
		Term    *Term                `json:"term"`
		Fields  []FieldName          `json:"fields"`
		Filters []*FieldFilter       `json:"filters"`
		Sort    *SortField           `json:"sort"`
	}{}
	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}
	q.Pagination = x.Pagination
	q.Kind = x.Kind
	q.Term = x.Term
	q.Fields = x.Fields
	q.Filters = x.Filters
	q.Sort = x.Sort
	for _, field := range q.Fields {
		q.fieldsByFieldName[field] = field
	}
	for _, filter := range q.Filters {
		q.fieldFiltersByFieldName[filter.Field] = filter
	}
	return nil
}

type Term string

func (t Term) String() string {
	return string(t)
}

type FieldFilter struct {
	Field    FieldName `json:"field"`
	Operator Operator  `json:"operator"`
	Value    string    `json:"value"`
}

func NewFieldFilter(field FieldName, operator Operator, value string) *FieldFilter {
	return &FieldFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

func (c FieldFilter) ValueString() string {
	return c.Value
}

func (c FieldFilter) ValueInt() int {
	i, _ := strconv.ParseInt(c.Value, 10, 64)
	return int(i)
}

func (c FieldFilter) ValueBool() bool {
	b, _ := strconv.ParseBool(c.Value)
	return b
}

type SortField struct {
	Field     string        `json:"field"`
	Direction SortDirection `json:"direction"`
}

func NewSortField(field string, direction SortDirection) *SortField {
	return &SortField{
		Field:     field,
		Direction: direction,
	}
}
