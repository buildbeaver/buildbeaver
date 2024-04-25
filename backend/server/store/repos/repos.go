package repos

import (
	"context"
	"fmt"
	"reflect"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
)

func init() {
	_ = models.MutableResource(&models.Repo{})
	_ = models.SoftDeletableResource(&models.Repo{})
	store.MustDBModel(&models.Repo{})
}

type RepoStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *RepoStore {
	return &RepoStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.Repo{}),
	}
}

// Create a new repo.
// Returns store.ErrAlreadyExists if a repo with matching unique properties already exists.
func (d *RepoStore) Create(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) error {
	return d.table.Create(ctx, txOrNil, repo)
}

// Read an existing repo, looking it up by ResourceID.
// Returns models.ErrNotFound if the repo does not exist.
func (d *RepoStore) Read(ctx context.Context, txOrNil *store.Tx, id models.RepoID) (*models.Repo, error) {
	repo := &models.Repo{}
	return repo, d.table.ReadByID(ctx, txOrNil, id.ResourceID, repo)
}

// ReadByExternalID reads an existing repo, looking it up by its external id.
// Returns models.ErrNotFound if the repo does not exist.
func (d *RepoStore) ReadByExternalID(ctx context.Context, txOrNil *store.Tx, externalID models.ExternalResourceID) (*models.Repo, error) {
	repo := &models.Repo{}
	return repo, d.table.ReadWhere(ctx, txOrNil, repo,
		goqu.Ex{"repo_external_id": externalID})
}

// Update an existing repo with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *RepoStore) Update(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) error {
	return d.table.UpdateByID(ctx, txOrNil, repo)
}

// Upsert creates a repo if it does not exist, otherwise it updates its mutable properties
// if they differ from the in-memory instance. Returns true,false if the resource was created
// and false,true if the resource was updated. false,false if neither a create or update was necessary.
// Repo Metadata and selected fields will not be updated (including Enabled and SSHKeySecretID fields).
func (d *RepoStore) Upsert(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) (bool, bool, error) {
	if repo.ExternalID == nil {
		return false, false, fmt.Errorf("error external id must be set to upsert")
	}
	return d.table.Upsert(ctx, txOrNil,
		func(tx *store.Tx) (models.Resource, error) {
			return d.ReadByExternalID(ctx, tx, *repo.ExternalID)
		}, func(tx *store.Tx) error {
			return d.Create(ctx, tx, repo)
		}, func(tx *store.Tx, obj models.Resource) (bool, error) {
			existing := obj.(*models.Repo)
			repo.RepoMetadata = existing.RepoMetadata
			repo.Enabled = existing.Enabled
			repo.SSHKeySecretID = existing.SSHKeySecretID
			if reflect.DeepEqual(existing, repo) {
				return false, nil
			}
			return true, d.Update(ctx, tx, repo)
		})
}

// SoftDelete soft deletes an existing repo.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *RepoStore) SoftDelete(ctx context.Context, txOrNil *store.Tx, repo *models.Repo) error {
	return d.table.SoftDelete(ctx, txOrNil, repo)
}

func (d *RepoStore) InitializeBuildCounter(ctx context.Context, txOrNil *store.Tx, repoID models.RepoID) error {
	return d.db.Write2(txOrNil, func(writer store.Writer) error {
		_, err := d.table.LogInsert(
			writer.Insert(goqu.T("repo_build_counters")).
				Rows(goqu.Record{
					"repo_build_counter_repo_id": repoID,
					"repo_build_counter_counter": 0,
				})).
			Executor().Exec()
		return err
	})
}

// IncrementBuildCounter increments and returns the build counter for the specified repo.
func (d *RepoStore) IncrementBuildCounter(ctx context.Context, txOrNil *store.Tx, id models.RepoID) (models.BuildNumber, error) {
	var buildCounter models.BuildNumber
	return buildCounter, d.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// TODO when we can upgrade to sqlite3 3.35.0+ we can use RETURNING and condense this into a single query
		// 	See https://www.sqlite.org/lang_returning.html and https://github.com/mattn/go-sqlite3/pull/926
		//  MySQL and Postgres and Goqu already support this.
		err := d.db.Write2(tx, func(writer store.Writer) error {
			_, err := d.table.LogUpdate(writer.Update(goqu.T("repo_build_counters")).
				Set(goqu.Record{"repo_build_counter_counter": goqu.L("repo_build_counter_counter+1")}).
				Where(goqu.Ex{"repo_build_counter_repo_id": id})).
				Executor().
				ScanVal(&buildCounter)
			return err
		})
		if err != nil {
			return err
		}
		return d.db.Read2(tx, func(reader store.Reader) error {
			_, err := d.table.LogSelect(reader.From("repo_build_counters").Select(goqu.C("repo_build_counter_counter")).
				Where(goqu.Ex{"repo_build_counter_repo_id": id})).
				Executor().
				ScanVal(&buildCounter)
			return err
		})
	})
}

// Search all repos. If searcher is set, the results will be limited to repos the searcher is authorized to
// see (via the read:repo permission). Use cursor to page through results, if any.
func (d *RepoStore) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, query search.Query) ([]*models.Repo, *models.Cursor, error) {
	repoQuery := search.NewRepoQuery(&query)
	reposSelect := d.table.Dialect().
		From(d.table.TableName()).
		Select(&models.Repo{})
	if !searcher.IsZero() {
		reposSelect = authorizations.WithIsAuthorizedListFilter(reposSelect, searcher, *models.RepoReadOperation, "repo_id")
	}
	if query.Term != nil {
		or := goqu.ExOr{}
		like := goqu.Op{"LIKE": fmt.Sprintf("%s%%", query.Term)}
		if repoQuery.IsInNameSet() {
			or["repo_name"] = like
		}
		if repoQuery.IsInDescriptionSet() {
			or["repo_description"] = like
		}
		reposSelect = reposSelect.Where(or)
	}
	if repoQuery.GetUserFilter() != nil || repoQuery.GetOrgFilter() != nil {
		reposSelect = reposSelect.InnerJoin(
			goqu.T("legal_entities"),
			goqu.On(goqu.Ex{"repos.repo_legal_entity_id": goqu.I("legal_entities.legal_entity_id")}))
		applyUserOrOrgFilter := func(filter *search.FieldFilter, legalEntityType models.LegalEntityType) {
			switch filter.Operator {
			case search.NotEqual:
				reposSelect = reposSelect.Where(goqu.Or(
					goqu.Ex{
						"legal_entity_type": goqu.Op{"neq": legalEntityType},
					},
					goqu.Ex{
						"legal_entity_type": goqu.Op{"eq": legalEntityType},
						"legal_entity_name": goqu.Op{"neq": filter.Value},
					},
				))
			default:
				reposSelect = reposSelect.Where(goqu.Ex{"legal_entity_name": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}}).
					Where(goqu.C("legal_entity_type").Eq(legalEntityType))
			}
		}
		if filter := repoQuery.GetUserFilter(); filter != nil {
			applyUserOrOrgFilter(filter, models.LegalEntityTypePerson)
		}
		if filter := repoQuery.GetOrgFilter(); filter != nil {
			applyUserOrOrgFilter(filter, models.LegalEntityTypeCompany)
		}
	}
	if filter := repoQuery.GetEnabledFilter(); filter != nil {
		reposSelect = reposSelect.
			Where(goqu.Ex{"repo_enabled": goqu.Op{filter.Operator.AsGoqu(): filter.ValueBool()}})
	}
	if filter := repoQuery.GetLegalEntityIDFilter(); filter != nil {
		reposSelect = reposSelect.
			Where(goqu.Ex{"repo_legal_entity_id": goqu.Op{filter.Operator.AsGoqu(): filter.ValueString()}})
	}
	if filter := repoQuery.GetSCMNameFilter(); filter != nil {
		// NOTE: This is a hack. This relies on knowledge of how models.ExternalResourceId is encoded in the DB.
		// It means we can't support the operator on this field for now.
		like := fmt.Sprintf("%s%%", filter.ValueString())
		reposSelect = reposSelect.
			Where(goqu.C("repo_external_id").Like(like))
	}

	if sort := repoQuery.GetCreatedAtSortField(); sort != nil {
		// TODO apply sort (will do in follow up PR)
	}

	var repos []*models.Repo
	cursor, err := d.table.ListIn(ctx, txOrNil, &repos, query.Pagination, reposSelect)
	if err != nil {
		return nil, nil, err
	}
	return repos, cursor, nil
}
