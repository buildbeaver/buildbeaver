package step

import (
	"context"
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type StepService struct {
	db                *store.DB
	stepStore         store.StepStore
	ownershipStore    store.OwnershipStore
	resourceLinkStore store.ResourceLinkStore
	logger.Log
}

func NewStepService(
	db *store.DB,
	stepStore store.StepStore,
	ownershipStore store.OwnershipStore,
	resourceLinkStore store.ResourceLinkStore,
	logFactory logger.LogFactory) *StepService {
	return &StepService{
		db:                db,
		stepStore:         stepStore,
		ownershipStore:    ownershipStore,
		resourceLinkStore: resourceLinkStore,
		Log:               logFactory("StepService"),
	}
}

// Read an existing step, looking it up by ID.
// Returns models.ErrNotFound if the step does not exist.
func (s *StepService) Read(ctx context.Context, txOrNil *store.Tx, id models.StepID) (*models.Step, error) {
	return s.stepStore.Read(ctx, txOrNil, id)
}

// ListByJobID gets all steps that are associated with the specified job id.
func (s *StepService) ListByJobID(ctx context.Context, txOrNil *store.Tx, id models.JobID) ([]*models.Step, error) {
	return s.stepStore.ListByJobID(ctx, txOrNil, id)
}

// Create a new step.
// Returns store.ErrAlreadyExists if a job with matching unique properties already exists.
func (s *StepService) Create(ctx context.Context, txOrNil *store.Tx, create *dto.CreateStep) error {
	err := create.Validate()
	if err != nil {
		return fmt.Errorf("error validating step: %w", err)
	}
	now := models.NewTime(time.Now())
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err = s.stepStore.Create(ctx, tx, create.Step)
		if err != nil {
			return fmt.Errorf("error creating step: %w", err)
		}
		ownership := models.NewOwnership(now, create.JobID.ResourceID, create.GetID())
		err = s.ownershipStore.Create(ctx, tx, ownership)
		if err != nil {
			return fmt.Errorf("error creating ownership: %w", err)
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, create)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Created step %q", create.ID)
		return nil
	})
}

// Update an existing step with optimistic locking. Overrides all previous values using the supplied model.
// Returns store.ErrOptimisticLockFailed if there is an optimistic lock mismatch.
func (s *StepService) Update(ctx context.Context, txOrNil *store.Tx, step *models.Step) error {
	err := step.Validate()
	if err != nil {
		return fmt.Errorf("error validating step: %w", err)
	}
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		err = s.stepStore.Update(ctx, tx, step)
		if err != nil {
			return fmt.Errorf("error updating step: %w", err)
		}
		_, _, err = s.resourceLinkStore.Upsert(ctx, tx, step)
		if err != nil {
			return fmt.Errorf("error upserting resource link: %w", err)
		}
		s.Infof("Updated step %q", step.ID)
		return nil
	})
}
