package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const WorkItemResourceKind ResourceKind = "work_item"

type WorkItemID struct {
	ResourceID
}

func NewWorkItemID() WorkItemID {
	return WorkItemID{ResourceID: NewResourceID(WorkItemResourceKind)}
}

type WorkItemType string

func (t WorkItemType) String() string {
	return string(t)
}

type WorkItemConcurrencyKey string

func NewWorkItemConcurrencyKey(key string) WorkItemConcurrencyKey {
	return WorkItemConcurrencyKey(key)
}

func (t WorkItemConcurrencyKey) String() string {
	return string(t)
}

// WorkItem represents a single piece of work to be queued and processed asynchronously.
type WorkItem struct {
	ID        WorkItemID `json:"id" goqu:"skipupdate" db:"work_item_id"`
	CreatedAt Time       `json:"created_at" goqu:"skipupdate" db:"work_item_created_at"`
	// ConcurrencyKey identifies a set of work items which can't be run concurrently and must be run in the
	// same order they were submitted, even when retries are required.
	// An empty string means this work item is independent and can be run concurrently with any other.
	ConcurrencyKey WorkItemConcurrencyKey `json:"concurrency_key"  db:"work_item_concurrency_key"`
	// StateID identifies the work item state record for this work item.
	StateID WorkItemStateID `json:"state_id" db:"work_item_state"`
	// Type identifies the type of work to be done, implying which code will be run to process the work item.
	Type WorkItemType `json:"type" db:"work_item_type"`
	// Data provides arbitrary details for use when processing the work item, in a format dependent on the Type.
	Data string `json:"data" db:"work_item_data"`
	// A message describing the current status of the work item, including any error info, for display purposes only.
	Status string `json:"status" db:"work_item_status"`
	// CompletedAt is the time at which work on this item was completed (either successful or failed), or nil if not yet completed.
	CompletedAt *Time `json:"completed_at,omitempty" db:"work_item_completed_at"`
}

func NewWorkItem(workItemType WorkItemType, data string, concurrencyKey WorkItemConcurrencyKey, now Time) *WorkItem {
	return &WorkItem{
		ID:             NewWorkItemID(),
		CreatedAt:      now,
		ConcurrencyKey: concurrencyKey,
		StateID:        WorkItemStateID{}, // to be filled out from the database
		Type:           workItemType,
		Data:           data,
		Status:         "new",
		CompletedAt:    nil,
	}
}

func (m *WorkItem) GetKind() ResourceKind {
	return WorkItemResourceKind
}

func (m *WorkItem) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *WorkItem) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *WorkItem) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error: id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error: created at must be set"))
	}
	if m.Type == "" {
		result = multierror.Append(result, errors.New("error: work item type must be set"))
	}
	return result.ErrorOrNil()
}

// WorkItemRecords is a convenience type that contains both the database records that make up the data for
// a work item.
type WorkItemRecords struct {
	Record *WorkItem      `db:"work_items"` // tag with table name
	State  *WorkItemState `db:"work_item_states"`
}
