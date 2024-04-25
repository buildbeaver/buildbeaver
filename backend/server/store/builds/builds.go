package builds

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
)

func init() {
	_ = models.MutableResource(&models.Build{})
	_ = models.SoftDeletableResource(&models.Build{})
	store.MustDBModel(&models.Build{})
}

type BuildStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *BuildStore {
	return &BuildStore{
		table: store.NewResourceTable(db, logFactory, &models.Build{}),
	}
}

// Create a new build.
// Returns store.ErrAlreadyExists if a build with matching unique properties already exists.
func (d *BuildStore) Create(ctx context.Context, txOrNil *store.Tx, build *models.Build) error {
	return d.table.Create(ctx, txOrNil, build)
}

// Read an existing build, looking it up by ResourceID.
// Returns models.ErrNotFound if the build does not exist.
func (d *BuildStore) Read(ctx context.Context, txOrNil *store.Tx, id models.BuildID) (*models.Build, error) {
	build := &models.Build{}
	return build, d.table.ReadByID(ctx, txOrNil, id.ResourceID, build)
}

// Update an existing build with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *BuildStore) Update(ctx context.Context, txOrNil *store.Tx, build *models.Build) error {
	return d.table.UpdateByID(ctx, txOrNil, build)
}

// LockRowForUpdate takes out an exclusive row lock on the build table row for the specified build.
// This function must be called within a transaction, and will block other transactions from locking, updating
// or deleting the row until this transaction ends.
func (s *BuildStore) LockRowForUpdate(ctx context.Context, tx *store.Tx, id models.BuildID) error {
	return s.table.LockRowForUpdate(ctx, tx, id.ResourceID)
}

// Search all builds. If searcher is set, the results will be limited to build(s) the searcher is authorized to
// see (via the read:build permission). Use cursor to page through results, if any.
func (d *BuildStore) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search *models.BuildSearch) ([]*models.BuildSearchResult, *models.Cursor, error) {
	buildsSelect := d.table.Dialect().From(d.table.TableName()).
		Select(&models.BuildSearchResult{})
	if !searcher.IsZero() {
		buildsSelect = authorizations.WithIsAuthorizedListFilter(buildsSelect, searcher, *models.BuildReadOperation, "build_id")
	}
	buildsSelect = buildsSelect.Join(goqu.T("repos"), goqu.On(goqu.Ex{"builds.build_repo_id": goqu.I("repos.repo_id")})).
		Join(goqu.T("commits"), goqu.On(goqu.Ex{"builds.build_commit_id": goqu.I("commits.commit_id")}))
	if search.RepoID != nil {
		buildsSelect = buildsSelect.Where(goqu.Ex{"build_repo_id": search.RepoID})
	}
	if search.CommitID != nil {
		buildsSelect = buildsSelect.Where(goqu.Ex{"build_commit_id": search.CommitID})
	}
	if search.CommitSHA != "" {
		buildsSelect = buildsSelect.Where(goqu.Ex{"commits.commit_sha": goqu.Op{"like": fmt.Sprintf("%s%%", search.CommitSHA)}})
	}
	if search.CommitAuthorID != nil {
		buildsSelect = buildsSelect.Where(goqu.Ex{"commits.commit_author_id": search.CommitAuthorID.String()})
	}
	if search.LegalEntityID != nil {
		buildsSelect = buildsSelect.Where(goqu.Ex{"repos.repo_legal_entity_id": search.LegalEntityID})
	}
	if search.Ref != "" {
		buildsSelect = buildsSelect.Where(goqu.Ex{"build_ref": search.Ref})
	}
	if search.ExcludeFailed {
		buildsSelect = buildsSelect.Where(goqu.C("build_error").IsNull())
	}
	for _, excludeStatus := range search.ExcludeStatuses {
		buildsSelect = buildsSelect.Where(goqu.Ex{"build_status": goqu.Op{"neq": excludeStatus}})
	}
	if search.IncludeStatuses != nil && len(search.IncludeStatuses) > 0 {
		buildsSelect = buildsSelect.Where(goqu.Ex{"build_status": goqu.Op{"in": search.IncludeStatuses}})
	}
	var builds []*models.BuildSearchResult
	cursor, err := d.table.ListIn(ctx, txOrNil, &builds, search.Pagination, buildsSelect)
	if err != nil {
		return nil, nil, err
	}
	return builds, cursor, nil
}

// UniversalSearch searches all builds. If searcher is set, the results will be limited to build(s) the searcher is authorized to
// see (via the read:build permission). Use cursor to page through results, if any.
func (d *BuildStore) UniversalSearch(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, query search.Query) ([]*models.BuildSearchResult, *models.Cursor, error) {
	buildQuery := search.NewBuildQuery(&query)
	buildsSelect := d.table.Dialect().From(
		d.table.TableName()).
		Select(&models.BuildSearchResult{})
	if !searcher.IsZero() {
		buildsSelect = authorizations.WithIsAuthorizedListFilter(buildsSelect, searcher, *models.BuildReadOperation, "build_id")
	}
	buildsSelect = buildsSelect.Join(goqu.T("repos"), goqu.On(goqu.Ex{"builds.build_repo_id": goqu.I("repos.repo_id")})).
		Join(goqu.T("commits"), goqu.On(goqu.Ex{"builds.build_commit_id": goqu.I("commits.commit_id")}))
	if buildQuery.IsInAuthorSet() || buildQuery.GetAuthorFilter() != nil {
		buildsSelect = buildsSelect.LeftJoin(goqu.T("legal_entities").As("author_legal_entity"), goqu.On(goqu.Ex{"commits.commit_author_id": goqu.I("author_legal_entity.legal_entity_id")}))
	}
	if buildQuery.GetCommitterFilter() != nil {
		buildsSelect = buildsSelect.LeftJoin(goqu.T("legal_entities").As("committer_legal_entity"), goqu.On(goqu.Ex{"commits.commit_committer_id": goqu.I("committer_legal_entity.legal_entity_id")}))
	}
	if query.Term != nil {
		var (
			or          []goqu.Expression
			prefixMatch = fmt.Sprintf("%s%%", query.Term)
		)
		if buildQuery.IsInCommitMessageSet() {
			or = append(or, goqu.I("commits.commit_message").Like(prefixMatch))
		}
		if buildQuery.IsInHashSet() {
			or = append(or, goqu.I("commits.commit_sha").Like(prefixMatch))
		}
		if buildQuery.IsInAuthorSet() {
			or = append(or, goqu.I("author_legal_entity.legal_entity_name").Like(prefixMatch))
		}
		if buildQuery.IsInAuthorNameSet() {
			or = append(or, goqu.I("commits.commit_author_name").Like(prefixMatch))
		}
		if buildQuery.IsInAuthorEmailSet() {
			or = append(or, goqu.I("commits.commit_author_email").Like(prefixMatch))
		}
		if buildQuery.IsInRefSet() {
			or = append(or,
				goqu.I("builds.build_ref").Like(prefixMatch),
				goqu.I("builds.build_ref").Like(fmt.Sprintf("refs/heads/%s%%", query.Term)),
				goqu.I("builds.build_ref").Like(fmt.Sprintf("refs/tags/%s%%", query.Term)),
			)
		}
		buildsSelect = buildsSelect.Where(goqu.Or(or...))
	}
	if filter := buildQuery.GetRepoFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"repos.repo_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetRepoIDFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"repos.repo_id": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetCommitIDFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.repo_id": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if buildQuery.GetUserFilter() != nil || buildQuery.GetOrgFilter() != nil {
		buildsSelect = buildsSelect.InnerJoin(
			goqu.T("legal_entities"),
			goqu.On(goqu.Ex{"repos.repo_legal_entity_id": goqu.I("legal_entities.legal_entity_id")}))
		applyUserOrOrgFilter := func(filter *search.FieldFilter, legalEntityType models.LegalEntityType) {
			switch filter.Operator {
			case search.NotEqual:
				buildsSelect = buildsSelect.Where(goqu.Or(
					goqu.Ex{
						"legal_entity_type": goqu.Op{"neq": legalEntityType},
					},
					goqu.Ex{
						"legal_entity_type": goqu.Op{"eq": legalEntityType},
						"legal_entity_name": goqu.Op{"neq": filter.Value},
					},
				))
			default:
				buildsSelect = buildsSelect.Where(goqu.Ex{"legal_entity_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}}).
					Where(goqu.C("legal_entity_type").Eq(legalEntityType))
			}
		}
		if filter := buildQuery.GetUserFilter(); filter != nil {
			applyUserOrOrgFilter(filter, models.LegalEntityTypePerson)
		}
		if filter := buildQuery.GetOrgFilter(); filter != nil {
			applyUserOrOrgFilter(filter, models.LegalEntityTypeCompany)
		}
	}
	if filter := buildQuery.GetStatusFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"builds.build_status": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetRefFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"builds.build_ref": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetHashFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.commit_sha": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetAuthorIDFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"author_legal_entity.legal_entity_id": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetAuthorFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"author_legal_entity.legal_entity_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetAuthorNameFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.commit_author_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetAuthorEmailFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.commit_author_email": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetCommitterIDFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"committer_legal_entity.id": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetCommitterFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"committer_legal_entity.name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetCommitterNameFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.commit_committer_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := buildQuery.GetCommitterEmailFilter(); filter != nil {
		buildsSelect = buildsSelect.
			Where(goqu.Ex{"commits.commit_committer_email": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	var builds []*models.BuildSearchResult
	cursor, err := d.table.ListIn(ctx, txOrNil, &builds, query.Pagination, buildsSelect)
	if err != nil {
		return nil, nil, err
	}
	return builds, cursor, nil
}
