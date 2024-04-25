package event

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type EventService struct {
	db         *store.DB
	eventStore store.EventStore
	logger.Log
}

func NewEventService(
	db *store.DB,
	eventStore store.EventStore,
	logFactory logger.LogFactory,
) *EventService {
	return &EventService{
		db:         db,
		eventStore: eventStore,
		Log:        logFactory("EventService"),
	}
}

// PublishEvent publishes a new event. Subscribers matching the event type and resource will be notified.
func (s *EventService) PublishEvent(ctx context.Context, txOrNil *store.Tx, eventData *models.EventData) error {
	err := eventData.Validate()
	if err != nil {
		return errors.Wrap(err, "error validating event data")
	}
	return s.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Allocate a sequence number for the event
		sequenceNumber, err := s.eventStore.IncrementEventCounter(ctx, tx, eventData.BuildID)
		if err != nil {
			return fmt.Errorf("error incrementing event counter: %w", err)
		}

		event, err := s.eventStore.Create(ctx, tx, sequenceNumber, eventData)
		if err != nil {
			return fmt.Errorf("error creating build: %w", err)
		}

		// TODO: Change this to trace level logging
		s.Infof("Created event, ID=%q, SequenceNumber=%d", event.ID, event.SequenceNumber)
		return nil
	})
}

// FetchEvents fetches new events for a given build, i.e. those with event numbers greater than lastEventNumber.
// limit specifies the maximum number of events to return.
// Events will be returned in order of event number; event numbers provide a unique ordering within a build.
// If no new events are available then the function returns immediately.
func (s *EventService) FetchEvents(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	lastEventNumber models.EventNumber,
	limit int,
) ([]*models.Event, error) {
	return s.eventStore.FindEvents(ctx, txOrNil, buildID, lastEventNumber, limit)
}

// FetchFilteredEvents fetches new events for a given build, i.e. those with event numbers greater than lastEventNumber.
// limit specifies the maximum number of events to return.
// Events will be returned in order of event number; event numbers provide a unique ordering within a build.
// If no new events are available then the function returns immediately.
// Events are filtered; only events that match at least one of the 'include' filters wil be returned.
// If no includeFilters are specified then all events for the build will be returned.
//func (s *EventService) FetchFilteredEvents(
//	ctx context.Context,
//	buildID models.BuildID,
//	include []*models.EventFilter,
//	lastEventNumber int64,
//	limit int,
//) ([]*models.Event, error) {
//}
