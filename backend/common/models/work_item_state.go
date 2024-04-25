package models

import (
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const WorkItemStateResourceKind ResourceKind = "work_item_state"

type WorkItemStateID struct {
	ResourceID
}

// NewWorkItemStateID produces a unique ID for a work item state record from the supplied
// concurrency key. The same concurrencyKey will produce the same ID.
// If concurrencyKey is empty then a randomly generated ID is returned, suitable for a WorkItemState
// object that is only used for a single work item.
func NewWorkItemStateID(concurrencyKey WorkItemConcurrencyKey) WorkItemStateID {
	var resourceID ResourceID
	if concurrencyKey != "" {
		resourceID = NewResourceIDFromUniqueData(WorkItemStateResourceKind, concurrencyKey.String())
	} else {
		// No concurrency key available, so make up a random unique resource ID that will only
		// be used for a single work item
		resourceID = NewResourceID(WorkItemStateResourceKind)
	}
	return WorkItemStateID{ResourceID: resourceID}
}

// WorkItemProcessorID is a unique ID identifying a particular work item processor that work items can
// be allocated to for processing.
type WorkItemProcessorID string

func NewWorkItemProcessorID() WorkItemProcessorID {
	return WorkItemProcessorID(uuid.New().String())
}

func (t WorkItemProcessorID) String() string {
	return string(t)
}

// WorkItemState represents a database record containing locking and retry state for work items.
// A single state record is used for all work items with the same concurrency key.
type WorkItemState struct {
	// Unique ID for this state record, based off a ConcurrencyKey if available.
	// Work items with the same ConcurrencyKey share the same WorkItemState object.
	ID WorkItemStateID `json:"id" goqu:"skipupdate" db:"work_item_state_id"`
	// CreatedAt is the time at which point this work item state was created.
	CreatedAt Time `json:"created_at" goqu:"skipupdate" db:"work_item_state_created_at"`
	// AttemptsSoFar is the number of attempts that have been made to process this work item (including
	// any attempt currently in progress).
	AttemptsSoFar int `json:"attempts_so_far" db:"work_item_state_attempts_so_far"`
	// NotBefore is the earliest time at which this work item is eligible for processing, or nil if the
	// work item could be processed immediately. Used to implement back-off and retry algorithms.
	NotBefore *Time `json:"not_before,omitempty" db:"work_item_state_not_before"`
	// AllocatedAt is the time at which the work item was last allocated to a work item processor.
	// This field is not used to determine whether an item is currently allocated and is provided
	// for debugging purposes.
	AllocatedAt *Time `json:"allocated_at,omitempty" db:"work_item_state_allocated_at"`
	// AllocatedTo is the unique identifier of the work item processor that will process this item, or
	// nil if the work item is not currently allocated to a processor.
	// AllocatedTo, together with AllocatedUntil, NotBefore and CompletedAt, defines whether an item
	// is currently available to be allocated for processing.
	AllocatedTo *WorkItemProcessorID `json:"allocated_to,omitempty" db:"work_item_state_allocated_to"`
	// AllocatedUntil is a time at which this work item will be considered 'released' from a work item processor
	// and available to be allocated to another processor, in the event of a processor getting 'stuck' or going down
	AllocatedUntil *Time `json:"allocated_until,omitempty" db:"work_item_state_allocated_until"`
}

// NewWorkItemState creates a new state object with an ID based off the specified concurrency key.
// If the supplied concurrency key is empty then a new random ID will be used.
func NewWorkItemState(now Time, concurrencyKey WorkItemConcurrencyKey) *WorkItemState {
	return &WorkItemState{
		ID:        NewWorkItemStateID(concurrencyKey),
		CreatedAt: now,
	}
}

func (m *WorkItemState) GetKind() ResourceKind {
	return WorkItemStateResourceKind
}

func (m *WorkItemState) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *WorkItemState) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *WorkItemState) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error: id must be set"))
	}
	return result.ErrorOrNil()
}
