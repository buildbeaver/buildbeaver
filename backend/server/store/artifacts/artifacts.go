package artifacts

import (
	"context"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
	"github.com/doug-martin/goqu/v9"
)

func init() {
	_ = models.MutableResource(&models.Artifact{})
	store.MustDBModel(&models.Artifact{})
}

type ArtifactStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *ArtifactStore {
	return &ArtifactStore{
		table: store.NewResourceTable(db, logFactory, &models.Artifact{}),
	}
}

// Create a new artifact.
// Returns store.ErrAlreadyExists if an artifact with matching unique properties already exists.
func (d *ArtifactStore) Create(ctx context.Context, txOrNil *store.Tx, artifactData *models.ArtifactData) (*models.Artifact, error) {
	artifact := &models.Artifact{
		ArtifactData: *artifactData,
		ID:           models.NewArtifactID(),
	}

	err := d.table.Create(ctx, txOrNil, artifact)
	if err != nil {
		return nil, err
	}

	return artifact, nil
}

// FindOrCreate creates an artifact if no artifact with the same unique values exist,
// otherwise it reads and returns the existing artifact.
// Returns the artifact as it is in the database, and true iff a new artifact was created.
func (d *ArtifactStore) FindOrCreate(ctx context.Context, txOrNil *store.Tx, artifact *models.ArtifactData) (result *models.Artifact, created bool, err error) {
	resource, created, err := d.table.FindOrCreate(ctx, txOrNil,
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.FindByUniqueFields(ctx, tx, artifact)
		},
		func(ctx context.Context, tx *store.Tx) (models.Resource, error) {
			return d.Create(ctx, tx, artifact)
		},
	)
	if err != nil {
		return nil, false, err
	}
	return resource.(*models.Artifact), created, nil
}

// Read an existing artifact, looking it up by ResourceID.
// Returns models.ErrNotFound if the artifact does not exist.
func (d *ArtifactStore) Read(ctx context.Context, txOrNil *store.Tx, id models.ArtifactID) (*models.Artifact, error) {
	artifact := &models.Artifact{}
	return artifact, d.table.ReadByID(ctx, txOrNil, id.ResourceID, artifact)
}

// FindByUniqueFields returns a matching artifact from the fields that are unique within our store.
func (d *ArtifactStore) FindByUniqueFields(ctx context.Context, txOrNil *store.Tx, artifact *models.ArtifactData) (*models.Artifact, error) {
	foundArtifact := &models.Artifact{}
	return foundArtifact, d.table.ReadWhere(ctx, txOrNil, foundArtifact,
		goqu.Ex{
			"artifact_job_id": artifact.JobID,
			"artifact_name":   artifact.Name,
			"artifact_path":   artifact.Path,
		})
}

// Update an existing artifact with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (d *ArtifactStore) Update(ctx context.Context, txOrNil *store.Tx, artifact *models.Artifact) error {
	return d.table.UpdateByID(ctx, txOrNil, artifact)
}

// Search all artifacts. If searcher is set, the results will be limited to artifacts the searcher is authorized to
// see (via the read:artifact permission). Use cursor to page through results, if any.
func (d *ArtifactStore) Search(ctx context.Context, txOrNil *store.Tx, searcher models.IdentityID, search models.ArtifactSearch) ([]*models.Artifact, *models.Cursor, error) {
	jobsJoin := d.table.Dialect().From("jobs").
		Select(
			goqu.COALESCE(
				goqu.C("job_indirect_to_job_id"),
				goqu.C("job_id"),
			).As("selected_job_id"))
	if !search.BuildID.IsZero() {
		jobsJoin = jobsJoin.Where(goqu.Ex{"job_build_id": search.BuildID})
	}
	if search.JobName != nil {
		jobsJoin = jobsJoin.Where(goqu.Ex{"job_name": search.JobName})
	}

	artifactsSelect := d.table.Dialect().
		From(d.table.TableName()).
		Select(&models.Artifact{})
	if !searcher.IsZero() {
		artifactsSelect = authorizations.WithIsAuthorizedListFilter(artifactsSelect, searcher, *models.ArtifactReadOperation, "artifact_id")
	}
	if search.GroupName != nil {
		artifactsSelect = artifactsSelect.Where(goqu.Ex{"artifact_group_name": search.GroupName})
	}
	artifactsSelect = artifactsSelect.Join(jobsJoin.As("applicable_jobs"), goqu.On(goqu.Ex{"artifacts.artifact_job_id": goqu.I("applicable_jobs.selected_job_id")}))

	var artifacts []*models.Artifact
	cursor, err := d.table.ListIn(ctx, txOrNil, &artifacts, search.Pagination, artifactsSelect)
	if err != nil {
		return nil, nil, err
	}
	return artifacts, cursor, nil
}
