package documents

import (
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Event struct {
	ID        models.EventID `json:"id"`
	CreatedAt models.Time    `json:"created_at"`
	// SequenceNumber is a monotonically increasing number to provide a well-defined order for events within a build.
	// Event sequence numbers will normally (but not always) be contiguous for a given build.
	SequenceNumber models.EventNumber `json:"sequence_number"`
	// BuildID is the build that generated this event
	BuildID models.BuildID `json:"build_id"`
	// Type identifies the type of event, determining what is expected in the event data
	Type models.EventType `json:"type"`
	// ResourceID is the ID of the resource this event is associated with
	ResourceID models.ResourceID `json:"resource_id"`
	// Workflow is the name of the workflow this event relates to, if applicable
	Workflow models.ResourceName `json:"workflow" db:"event_workflow"`
	// JobName is the name of the job this event relates to, if applicable
	JobName models.ResourceName `json:"job_name" db:"event_job_name"`
	// ResourceName is the name of the resource this event is associated with
	ResourceName models.ResourceName `json:"resource_name"`
	// Payload provides additional information for the event. The format of the payload data depends on the event Type
	Payload string `json:"payload"`
}

func MakeEvent(rctx routes.RequestContext, event *models.Event) *Event {
	return &Event{
		ID:             event.ID,
		CreatedAt:      event.CreatedAt,
		SequenceNumber: event.SequenceNumber,
		BuildID:        event.BuildID,
		Type:           event.Type,
		ResourceID:     event.ResourceID,
		Workflow:       event.Workflow,
		JobName:        event.JobName,
		ResourceName:   event.ResourceName,
		Payload:        event.Payload,
	}
}

func MakeEvents(rctx routes.RequestContext, events []*models.Event) []*Event {
	var docs []*Event
	for _, event := range events {
		docs = append(docs, MakeEvent(rctx, event))
	}
	return docs
}
