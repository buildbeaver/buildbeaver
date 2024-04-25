package search

import (
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/models"
)

type RepoQuery struct {
	*Query
}

func NewRepoQuery(query *Query) RepoQuery {
	return RepoQuery{Query: query}
}

func (q RepoQuery) IsInNameSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("name")
}

func (q RepoQuery) IsInDescriptionSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("description")
}

func (q RepoQuery) GetUserFilter() *FieldFilter {
	return q.GetFilter("user")
}

func (q RepoQuery) GetOrgFilter() *FieldFilter {
	return q.GetFilter("org")
}

func (q RepoQuery) GetLegalEntityIDFilter() *FieldFilter {
	return q.GetFilter("legal_entity_id")
}

func (q RepoQuery) GetEnabledFilter() *FieldFilter {
	return q.GetFilter("enabled")
}

func (q RepoQuery) GetSCMNameFilter() *FieldFilter {
	return q.GetFilter("scm_name")
}

func (q RepoQuery) GetCreatedAtSortField() *SortField {
	if q.Sort != nil && q.Sort.Field == "created_at" {
		return q.Sort
	}
	return nil
}

type RepoQueryBuilder struct {
	builder *QueryBuilder
}

func NewRepoQueryBuilder(existing ...Query) *RepoQueryBuilder {
	return &RepoQueryBuilder{builder: NewQueryBuilder(existing...).Kind(models.RepoResourceKind)}
}

func (b *RepoQueryBuilder) Term(term Term) *RepoQueryBuilder {
	b.builder = b.builder.Term(term)
	return b
}

func (b *RepoQueryBuilder) InName() *RepoQueryBuilder {
	b.builder = b.builder.In("name")
	return b
}

func (b *RepoQueryBuilder) InDescription() *RepoQueryBuilder {
	b.builder = b.builder.In("description")
	return b
}

func (b *RepoQueryBuilder) WhereUser(operator Operator, value string) *RepoQueryBuilder {
	b.builder = b.builder.Where("user", operator, value)
	return b
}

func (b *RepoQueryBuilder) WhereOrg(operator Operator, value string) *RepoQueryBuilder {
	b.builder = b.builder.Where("org", operator, value)
	return b
}

func (b *RepoQueryBuilder) WhereLegalEntityID(operator Operator, id models.LegalEntityID) *RepoQueryBuilder {
	b.builder = b.builder.Where("legal_entity_id", operator, id.String())
	return b
}

func (b *RepoQueryBuilder) WhereEnabled(operator Operator, value bool) *RepoQueryBuilder {
	b.builder = b.builder.Where("enabled", operator, strconv.FormatBool(value))
	return b
}

func (b *RepoQueryBuilder) WhereSCMName(operator Operator, value string) *RepoQueryBuilder {
	b.builder = b.builder.Where("scm_name", operator, value)
	return b
}

func (b *RepoQueryBuilder) SortCreatedAt(direction ...SortDirection) *RepoQueryBuilder {
	b.builder = b.builder.Sort("created_at", direction...)
	return b
}

func (b *RepoQueryBuilder) Limit(limit int) *RepoQueryBuilder {
	b.builder = b.builder.Limit(limit)
	return b
}

func (b *RepoQueryBuilder) Compile() Query {
	return b.builder.Compile()
}
