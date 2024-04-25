package models

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const EventResourceKind ResourceKind = "event"

type EventID struct {
	ResourceID
}

func NewEventID() EventID {
	return EventID{ResourceID: NewResourceID(EventResourceKind)}
}

func EventIDFromResourceID(id ResourceID) EventID {
	return EventID{ResourceID: id}
}

type EventNumber uint64

func (n EventNumber) String() string {
	return strconv.FormatUint(uint64(n), 10)
}

type EventType string

func (t EventType) String() string {
	return string(t)
}

type EventMetadata struct {
	ID        EventID `json:"id" goqu:"skipupdate" db:"event_id"`
	CreatedAt Time    `json:"created_at" goqu:"skipupdate" db:"event_created_at"`
	// SequenceNumber is a monotonically increasing number to provide a well-defined order for events within a build.
	// Event sequence numbers will normally (but not always) be contiguous for a given build.
	SequenceNumber EventNumber `json:"sequence_number" db:"event_sequence_number"`
}

type EventData struct {
	// BuildID is the build that generated this event
	BuildID BuildID `json:"build_id" db:"event_build_id"`
	// Type identifies the type of event, determining what is expected in the event data
	Type EventType `json:"type" db:"event_type"`
	// ResourceID is the ID of the resource this event is associated with
	ResourceID ResourceID `json:"resource_id" db:"event_resource_id"`
	// Workflow is the name of the workflow this event relates to, if applicable
	Workflow ResourceName `json:"workflow" db:"event_workflow"`
	// JobName is the name of the job this event relates to, if applicable
	JobName ResourceName `json:"job_name" db:"event_job_name"`
	// ResourceName is the name of the resource this event is associated with
	ResourceName ResourceName `json:"resource_name" db:"event_resource_name"`
	// Payload provides additional information for the event. The format of the payload data depends on the event Type
	Payload string `json:"payload" db:"event_payload"`
}

type Event struct {
	EventMetadata
	EventData
}

func NewEventData(
	buildID BuildID,
	eventType EventType,
	resourceID ResourceID,
	workflow ResourceName,
	jobName ResourceName,
	resourceName ResourceName,
	payload string,
) *EventData {
	return &EventData{
		BuildID:      buildID,
		Type:         eventType,
		ResourceID:   resourceID,
		Workflow:     workflow,
		JobName:      jobName,
		ResourceName: resourceName,
		Payload:      payload,
	}
}

func NewEvent(now Time, eventData *EventData) *Event {
	return &Event{
		EventMetadata: EventMetadata{
			ID:        NewEventID(),
			CreatedAt: now,
		},
		EventData: *eventData,
	}
}

func (m *Event) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Event) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Event) GetKind() ResourceKind {
	return EventResourceKind
}

func (m *Event) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.SequenceNumber == 0 {
		result = multierror.Append(result, errors.Errorf("error SequenceNumber must be non-zero"))
	}
	err := m.EventData.Validate()
	if err != nil {
		result = multierror.Append(result, fmt.Errorf("data is invalid: %s", err))
	}
	return result.ErrorOrNil()
}

func (m *EventData) Validate() error {
	var result *multierror.Error
	if !m.BuildID.Valid() {
		result = multierror.Append(result, errors.New("error Build ID must be set"))
	} else if m.BuildID.ResourceID.Kind() != BuildResourceKind {
		result = multierror.Append(result, errors.Errorf("error Build ID must be a ResourceID of kind '%s'", BuildResourceKind))
	}
	if m.Type == "" {
		result = multierror.Append(result, errors.Errorf("error event Type must be specified"))
	}
	if !m.ResourceID.Valid() {
		result = multierror.Append(result, errors.New("error resource ID must be set"))
	}
	// Payload is optional
	return result.ErrorOrNil()
}
