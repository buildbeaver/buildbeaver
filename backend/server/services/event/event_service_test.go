package event_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const testEvent models.EventType = "TestEvent"

func TestEventService(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err, "Error initializing app")
	defer cleanup()

	// Make a build to submit events against
	ctx := context.Background()
	legalEntity, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	server_test.CreateRunner(t, ctx, app, "", legalEntity.ID, nil) // there must be a runner to run the build
	repo := server_test.CreateRepo(t, ctx, app, legalEntity.ID)
	_ = server_test.CreateCommit(t, ctx, app, repo.ID, legalEntity.ID)
	bGraph1 := server_test.CreateAndQueueBuild(t, ctx, app, repo.ID, legalEntity.ID, "master")
	bGraph2 := server_test.CreateAndQueueBuild(t, ctx, app, repo.ID, legalEntity.ID, "a-branch")

	// Run tests sequentially since they use the same event store
	testBasicEvents(app, bGraph1.Build.ID)(t)
	testEventChunks(app, bGraph2.Build.ID)(t)
}

func testBasicEvents(app *server_test.TestServer, buildID models.BuildID) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		fakeJobID := models.NewResourceID(models.JobResourceKind)
		fakeJobName := models.ResourceName("test-job")
		fakeWorkflow := models.ResourceName("test-workflow")

		// Increment event counter to determine how many event sequence numbers to skip when reading events
		lastEventNumber, err := app.EventStore.IncrementEventCounter(ctx, nil, buildID)

		// Test incrementing counter again to ensure the number goes up
		newEventNumber, err := app.EventStore.IncrementEventCounter(ctx, nil, buildID)
		assert.Equal(t, lastEventNumber+1, newEventNumber)
		lastEventNumber = newEventNumber

		// There should be no more events outstanding at the start
		events, err := app.EventService.FetchEvents(ctx, nil, buildID, lastEventNumber, 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(events))

		// Submit two test events
		err = app.EventService.PublishEvent(ctx, nil, models.NewEventData(
			buildID,
			models.JobStatusChangedEvent,
			fakeJobID,
			fakeWorkflow,
			fakeJobName,
			fakeJobName,
			"test payload 1",
		))
		require.NoError(t, err, "error (1) returned from EventService.PublishEvent()")

		err = app.EventService.PublishEvent(ctx, nil, models.NewEventData(
			buildID,
			models.JobStatusChangedEvent,
			fakeJobID,
			fakeWorkflow,
			fakeJobName,
			fakeJobName,
			"test payload 2",
		))
		require.NoError(t, err, "error (2) returned from EventService.PublishEvent()")

		// Fetching events should now return the 2 events submitted
		events, err = app.EventService.FetchEvents(ctx, nil, buildID, lastEventNumber, 100)
		require.NoError(t, err)
		assert.Equal(t, 2, len(events))
		assert.Equal(t, "test payload 1", events[0].Payload)
		assert.Equal(t, "test payload 2", events[1].Payload)
		lastEventNumber = events[len(events)-1].SequenceNumber

		// There should be no more events
		events, err = app.EventService.FetchEvents(ctx, nil, buildID, lastEventNumber, 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(events))
	}
}

func testEventChunks(app *server_test.TestServer, buildID models.BuildID) func(t *testing.T) {
	return func(t *testing.T) {
		const (
			totalNrEvents = 1000 // submit a reasonable number of events
			readChunkSize = 90   // not an even divisor of totalNrEvents; should be fewer events in last fetch
		)
		ctx := context.Background()
		fakeJobID := models.NewResourceID(models.JobResourceKind)
		fakeJobName := models.ResourceName("test-job")
		fakeWorkflow := models.ResourceName("test-workflow")

		// Increment event counter to determine how many event sequence numbers to skip when reading events
		lastEventNumber, err := app.EventStore.IncrementEventCounter(ctx, nil, buildID)

		// There should be no more events outstanding at the start
		events, err := app.EventService.FetchEvents(ctx, nil, buildID, lastEventNumber, 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(events))

		// Submit a bunch of events
		for i := 0; i < totalNrEvents; i++ {
			err = app.EventService.PublishEvent(ctx, nil, models.NewEventData(
				buildID,
				models.BuildStatusChangedEvent,
				fakeJobID,
				fakeWorkflow,
				fakeJobName,
				fakeJobName,
				payloadForEvent(i+1), // number payloads from 1
			))
			require.NoError(t, err, "error returned from EventService.PublishEvent() for event %d", i)
		}

		// Fetch the events in chunks - start by fetching all full chunks, with a few events left over at the end
		nrFullChunks := totalNrEvents / readChunkSize
		for chunkNr := 0; chunkNr < nrFullChunks; chunkNr++ {
			lastEventNumber = readAndCheckChunk(
				t, app, buildID,
				readChunkSize,
				lastEventNumber,
				readChunkSize,             // expect a full chunk of events back
				(chunkNr*readChunkSize)+1, // payloads numbered from 1
			)
		}

		// There should be a partial chunk of events left over
		lastEventNumber = readAndCheckChunk(
			t, app, buildID,
			readChunkSize,
			lastEventNumber,
			totalNrEvents%readChunkSize,    // nr of leftovers after the last full chunk
			(nrFullChunks*readChunkSize)+1, // payloads numbered from 1
		)

		// There should be no more events
		events, err = app.EventService.FetchEvents(ctx, nil, buildID, lastEventNumber, readChunkSize)
		require.NoError(t, err)
		require.Equal(t, 0, len(events))
	}
}

func payloadForEvent(i int) string {
	return fmt.Sprintf("payload number %d", i)
}

// readAndCheckChunk attempts to read chunkSize events from the event service, starting from the event after
// lastEventNumber. Checks that expectedNrResults are returned and checks the returned event payloads to ensure
// the expected events are delivered and are in the correct order.
func readAndCheckChunk(
	t *testing.T,
	app *server_test.TestServer,
	buildID models.BuildID,
	chunkSize int,
	lastEventNumber models.EventNumber,
	expectedNrResults int,
	expectedFirstPayload int,
) (newLastEventNumber models.EventNumber) {
	t.Logf("Calling FetchEvents() with lastEventNumber=%d", lastEventNumber)
	events, err := app.EventService.FetchEvents(context.Background(), nil, buildID, lastEventNumber, chunkSize)
	require.NoError(t, err)
	require.Equal(t, expectedNrResults, len(events), "Expected a chunk of %d events", expectedNrResults)
	// Check events are in the correct order
	for i := 0; i < expectedNrResults; i++ {
		assert.Equal(t, payloadForEvent(expectedFirstPayload+i), events[i].Payload)
	}
	return events[len(events)-1].SequenceNumber
}

// TODO: Write test for concurrent event sequencing/sequence number allocation
