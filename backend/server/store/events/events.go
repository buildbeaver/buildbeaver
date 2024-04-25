package events

import (
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/doug-martin/goqu/v9"
	"golang.org/x/net/context"
)

func init() {
	store.MustDBModel(&models.Event{})
}

type EventStore struct {
	db    *store.DB
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *EventStore {
	return &EventStore{
		db:    db,
		table: store.NewResourceTable(db, logFactory, &models.Event{}),
	}
}

// Create a new event with the specified sequence number and data.
// Returns store.ErrAlreadyExists if an event with this ID or build/sequence number already exists.
func (d *EventStore) Create(
	ctx context.Context,
	txOrNil *store.Tx,
	sequenceNumber models.EventNumber,
	eventData *models.EventData,
) (*models.Event, error) {
	now := models.NewTime(time.Now())
	event := &models.Event{
		EventData: *eventData,
		EventMetadata: models.EventMetadata{
			ID:             models.NewEventID(),
			SequenceNumber: sequenceNumber,
			CreatedAt:      now,
		},
	}

	err := d.table.Create(ctx, txOrNil, event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// Read an existing event, looking it up by ResourceID.
// Will return models.ErrNotFound if the event does not exist.
func (d *EventStore) Read(ctx context.Context, txOrNil *store.Tx, id models.EventID) (*models.Event, error) {
	event := &models.Event{}
	return event, d.table.ReadByID(ctx, txOrNil, id.ResourceID, event)
}

// DeleteEventsForBuild permanently and idempotently deletes all events for the specified build.
func (d *EventStore) DeleteEventsForBuild(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) error {
	return d.table.DeleteWhere(ctx, txOrNil, goqu.Ex{"event_build_id": buildID.ResourceID})
}

// FindEvents reads the next events for a build.
// If no matching events are present then an empty list is returned immediately.
func (d *EventStore) FindEvents(
	ctx context.Context,
	txOrNil *store.Tx,
	buildID models.BuildID,
	lastEventNumber models.EventNumber,
	limit int,
) ([]*models.Event, error) {
	var events []*models.Event

	eventSelect := goqu.From(d.table.TableName()).Select(&models.Event{}).
		Where(goqu.Ex{"event_build_id": buildID}).
		Where(goqu.C("event_sequence_number").Gt(lastEventNumber)).
		Order(goqu.C("event_sequence_number").Asc()).
		Limit(uint(limit))

	// Perform the read directly on the database; ResourceTable.ListIn() is not suitable because it forces
	// the wrong sort order, and it does pagination which is handled here through lastEventNumber and limit
	err := d.db.Read2(txOrNil, func(db store.Reader) error {
		query, args, err := eventSelect.ToSQL()
		if err != nil {
			return fmt.Errorf("error generating query: %w", err)
		}
		d.table.LogQuery(query, args)
		return db.ScanStructsContext(ctx, &events, query, args...)
	})
	if err != nil {
		return nil, store.MakeStandardDBError(err)
	}

	return events, nil
}

// IncrementEventCounter increments and returns the event counter for the specified build, to provide
// a sequence number for a new event.
func (d *EventStore) IncrementEventCounter(ctx context.Context, txOrNil *store.Tx, buildID models.BuildID) (models.EventNumber, error) {
	var counter models.EventNumber

	err := d.db.WithTx(ctx, txOrNil, func(tx *store.Tx) error {
		// Attempt to increment the counter; if no counter found for this build then 'found' will be set to false
		// TODO when we can upgrade to sqlite3 3.35.0+ we can use RETURNING and condense this into a single query
		var found bool
		err := d.db.Write2(tx, func(writer store.Writer) error {
			var err error
			updateResult, err := d.table.LogUpdate(writer.Update(goqu.T("build_event_counters")).
				Set(goqu.Record{"build_event_counter_counter": goqu.L("build_event_counter_counter+1")}).
				Where(goqu.Ex{"build_event_counter_build_id": buildID})).
				Executor().Exec()
			if err != nil {
				return fmt.Errorf("error updating event counter: %w", err)
			}
			nrRowsUpdated, err := updateResult.RowsAffected()
			if err != nil {
				return fmt.Errorf("error determining number of rows updated in IncrementEventCounter(): %w", err)
			}
			found = nrRowsUpdated == 1
			return nil
		})
		if err != nil {
			return err
		}
		if found {
			// Counter was found and incremented, so read the new value
			return d.db.Read2(tx, func(reader store.Reader) error {
				_, err := d.table.LogSelect(reader.From("build_event_counters").
					Select(goqu.C("build_event_counter_counter")).
					Where(goqu.Ex{"build_event_counter_build_id": buildID})).
					Executor().
					ScanVal(&counter)
				return err
			})
		} else {
			// Counter was not found, so initialize to value of 1
			counter = 1
			return d.initializeEventCounter(tx, buildID, counter)
		}
	})
	return counter, err
}

func (d *EventStore) initializeEventCounter(txOrNil *store.Tx, buildID models.BuildID, initialValue models.EventNumber) error {
	return d.db.Write2(txOrNil, func(writer store.Writer) error {
		result, err := d.table.LogInsert(
			writer.Insert(goqu.T("build_event_counters")).
				Rows(goqu.Record{
					"build_event_counter_build_id": buildID.String(),
					"build_event_counter_counter":  initialValue,
				})).
			Executor().Exec()
		if err != nil {
			return fmt.Errorf("error inserting new event counter: %w", err)
		}
		nrRowsInserted, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("error determining number of rows inserted in initializeEventCounter(): %w", err)
		}
		if nrRowsInserted != 1 {
			return fmt.Errorf("error inserting new event counter; expected 1 row to be inserted but %d rows inserted", nrRowsInserted)
		}
		return nil
	})
}
