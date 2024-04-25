package search

import (
	"github.com/buildbeaver/buildbeaver/common/models"
)

type BuildQuery struct {
	*Query
}

func NewBuildQuery(query *Query) BuildQuery {
	return BuildQuery{Query: query}
}

func (q BuildQuery) IsInCommitMessageSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("commit-message")
}

func (q BuildQuery) IsInHashSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("hash")
}

func (q BuildQuery) IsInRefSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("ref")
}

func (q BuildQuery) IsInAuthorSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("author")
}

func (q BuildQuery) IsInAuthorNameSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("author-name")
}

func (q BuildQuery) IsInAuthorEmailSet() bool {
	return !q.AnyInFieldsSet() || q.IsInFieldSet("author-email")
}

func (q BuildQuery) GetRepoIDFilter() *FieldFilter {
	return q.GetFilter("repo-id")
}

func (q BuildQuery) GetRepoFilter() *FieldFilter {
	return q.GetFilter("repo")
}

func (q BuildQuery) GetCommitIDFilter() *FieldFilter {
	return q.GetFilter("commit-id")
}

func (q BuildQuery) GetUserFilter() *FieldFilter {
	return q.GetFilter("user")
}

func (q BuildQuery) GetOrgFilter() *FieldFilter {
	return q.GetFilter("org")
}

func (q BuildQuery) GetStatusFilter() *FieldFilter {
	return q.GetFilter("status")
}

func (q BuildQuery) GetRefFilter() *FieldFilter {
	return q.GetFilter("ref")
}

func (q BuildQuery) GetHashFilter() *FieldFilter {
	return q.GetFilter("hash")
}

func (q BuildQuery) GetAuthorIDFilter() *FieldFilter {
	return q.GetFilter("author-id")
}

func (q BuildQuery) GetAuthorFilter() *FieldFilter {
	return q.GetFilter("author")
}

func (q BuildQuery) GetAuthorNameFilter() *FieldFilter {
	return q.GetFilter("author-name")
}

func (q BuildQuery) GetAuthorEmailFilter() *FieldFilter {
	return q.GetFilter("author-email")
}

func (q BuildQuery) GetCommitterIDFilter() *FieldFilter {
	return q.GetFilter("committer-id")
}

func (q BuildQuery) GetCommitterFilter() *FieldFilter {
	return q.GetFilter("committer")
}

func (q BuildQuery) GetCommitterNameFilter() *FieldFilter {
	return q.GetFilter("committer-name")
}

func (q BuildQuery) GetCommitterEmailFilter() *FieldFilter {
	return q.GetFilter("committer-email")
}

func (q BuildQuery) GetCreatedAtSortField() *SortField {
	if q.Sort != nil && q.Sort.Field == "created_at" {
		return q.Sort
	}
	return nil
}

type BuildQueryBuilder struct {
	builder *QueryBuilder
}

func NewBuildQueryBuilder(existing ...Query) *BuildQueryBuilder {
	return &BuildQueryBuilder{builder: NewQueryBuilder(existing...).Kind(models.BuildResourceKind)}
}

func (b *BuildQueryBuilder) Term(term Term) *BuildQueryBuilder {
	b.builder = b.builder.Term(term)
	return b
}

func (b *BuildQueryBuilder) InCommitMessage() *BuildQueryBuilder {
	b.builder = b.builder.In("commit-message")
	return b
}

func (b *BuildQueryBuilder) WhereRepo(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("repo", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereRepoID(operator Operator, value models.RepoID) *BuildQueryBuilder {
	b.builder = b.builder.Where("repo-id", operator, value.String())
	return b
}

func (b *BuildQueryBuilder) WhereCommitID(operator Operator, value models.CommitID) *BuildQueryBuilder {
	b.builder = b.builder.Where("commit-id", operator, value.String())
	return b
}

func (b *BuildQueryBuilder) WhereUser(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("user", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereOrg(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("org", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereStatus(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("status", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereRef(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("ref", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereHash(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("hash", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereAuthorID(operator Operator, value models.LegalEntityID) *BuildQueryBuilder {
	b.builder = b.builder.Where("author-id", operator, value.String())
	return b
}

func (b *BuildQueryBuilder) WhereAuthor(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("author", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereAuthorName(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("author-name", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereAuthorEmail(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("author-email", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereCommitterID(operator Operator, value models.LegalEntityID) *BuildQueryBuilder {
	b.builder = b.builder.Where("committer-id", operator, value.String())
	return b
}

func (b *BuildQueryBuilder) WhereCommitter(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("committer", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereCommitterName(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("committer-name", operator, value)
	return b
}

func (b *BuildQueryBuilder) WhereCommitterEmail(operator Operator, value string) *BuildQueryBuilder {
	b.builder = b.builder.Where("committer-email", operator, value)
	return b
}

func (b *BuildQueryBuilder) SortCreatedAt(direction ...SortDirection) *BuildQueryBuilder {
	b.builder = b.builder.Sort("created_at", direction...)
	return b
}

func (b *BuildQueryBuilder) Limit(limit int) *BuildQueryBuilder {
	b.builder = b.builder.Limit(limit)
	return b
}

func (b *BuildQueryBuilder) Compile() Query {
	return b.builder.Compile()
}
