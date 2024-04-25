package work_queue_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/work_queue"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const testWorkItem models.WorkItemType = "TestWorkItem"

// testWorkItemData is the data used for test work items. It provides fields used by the test work item handler
// to implement behaviour like failure testing and taking a long time to process.
type testWorkItemData struct {
	// All fields must be public to be serialized to JSON in the work queue
	// The zero setting for each field should be a suitable default so that any fields can be omitted
	ItemNr                   int // filled out by newTestWorkItem(), unique across all tests
	FailPermanently          bool
	NrTimesToFailTemporarily int
	TimeToProcess            time.Duration
	IgnoreCancelRequest      bool
}

var (
	nextItemNr      = 1
	nextItemNrMutex sync.Mutex
)

// Creates a new testWorkItem using the specified data.
// If ItemNr in the data is zero then it will be set to the next available item number.
func newTestWorkItem(data *testWorkItemData, concurrencyKey models.WorkItemConcurrencyKey) *models.WorkItem {
	if data.ItemNr == 0 {
		nextItemNrMutex.Lock()
		data.ItemNr = nextItemNr
		nextItemNr++
		nextItemNrMutex.Unlock()
	}
	dataJson, err := json.Marshal(data)
	if err != nil {
		panic("Unable to marshal TestWorkItemData object to JSON")
	}
	return models.NewWorkItem(testWorkItem, string(dataJson), concurrencyKey, models.NewTime(time.Now()))
}

// workItemTestState tracks information about an individual test work item, used by the test work item handler
// to implement behaviour like failure testing.
// Also used to identify work items that aren't done, so we know when to end the test.
type workItemTestState struct {
	workItemID             models.WorkItemID
	concurrencyKey         models.WorkItemConcurrencyKey
	workItemStateID        models.WorkItemStateID
	done                   bool // set to true when no more processing will occur for this work item
	currentlyProcessing    bool
	nrTimesProcessed       int
	nrTimesFailed          int
	expectedTimesProcessed int // use -1 to mean 'any number is acceptable'
	expectedTimesFailed    int // use -1 to mean 'any number is acceptable'
	// expectedStateStillInDB only needs to be specified when testing a mixture of deleting and not deleting
	// failed vs succeeded work items. Set to true if we expect this work item's state to remain in the database.
	expectedStateStillInDB *bool
}

// testEnvironment collects together objects required for testing work items, to be easily passed around.
type testEnvironment struct {
	t           *testing.T
	app         *server_test.TestServer
	testName    string
	states      map[int]*workItemTestState // maps itemNr to current state for that work item
	statesMutex sync.Mutex
}

func newTestEnvironment(t *testing.T, app *server_test.TestServer, testName string) *testEnvironment {
	return &testEnvironment{
		t:        t,
		app:      app,
		testName: testName,
		states:   make(map[int]*workItemTestState),
	}
}

func TestWorkQueueStartAndStop(t *testing.T) {
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()

	workQueue := app.WorkQueueService.(*work_queue.WorkQueueService)

	// Start work queue processing
	workQueue.Start()
	// Starting again should be a no-op
	workQueue.Start()

	// Shut down processing
	workQueue.Shutdown()
	// Shutting down again should be a no-op
	workQueue.Shutdown()
}

func TestWorkQueueStartupRaceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	const workItemTimeout = 10 * time.Second
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	env := newTestEnvironment(t, app, "WorkQueueStartupRace")

	// Start work queue processing, before registering the handler
	workQueue := app.WorkQueueService.(*work_queue.WorkQueueService)
	work_queue.NrWorkItemProcessors = 10 // test concurrency
	workQueue.Start()

	// Regular work item to be processed successfully, but submit before registering the handler
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-1", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Give it time to attempt to process work item
	time.Sleep(5 * time.Second)

	// Set up handler for test work item type
	err = workQueue.RegisterHandler(
		testWorkItem,
		makeTestWorkItemHandler(env),
		workItemTimeout,
		instrumentedBackoff(env, work_queue.LinearBackoff(5, 1*time.Second)),
		true,
		true,
	)
	require.NoError(t, err)

	err = waitUntilAllWorkItemsDone(env, 1*time.Minute)
	assert.NoError(t, err)
	workQueue.Shutdown()

	checkWorkItemResults(env, true, true)
}

func TestWorkQueueProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	const (
		workItemTimeout         = 10 * time.Second
		keepFailedWorkItems     = true
		keepSuccessfulWorkItems = true
	)
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	env := newTestEnvironment(t, app, "WorkQueueProcessing")

	// Set up work queue and handler for test work item type
	workQueue := app.WorkQueueService.(*work_queue.WorkQueueService)
	work_queue.NrWorkItemProcessors = 10 // test concurrency
	workQueue.Start()
	err = workQueue.RegisterHandler(
		testWorkItem,
		makeTestWorkItemHandler(env),
		workItemTimeout,
		instrumentedBackoff(env, work_queue.LinearBackoff(5, 1*time.Second)),
		keepFailedWorkItems,
		keepSuccessfulWorkItems,
	)
	require.NoError(t, err)

	// Regular work item to be processed successfully
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-2", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// A 'slow' work item that will be forced to time out
	_, err = submitNewWorkItem(env, &testWorkItemData{TimeToProcess: 2 * workItemTimeout}, "key-3", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// A work item which will fail permanently
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-4", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// A work item which will fail 3 times, then succeed
	_, err = submitNewWorkItem(env, &testWorkItemData{NrTimesToFailTemporarily: 3}, "key-5", 4, 3, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// A work item which will fail repeatedly until the backoff algorithm decides that's enough and fails it permanently
	_, err = submitNewWorkItem(env, &testWorkItemData{NrTimesToFailTemporarily: 1000}, "key-6", -1, -1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	err = waitUntilAllWorkItemsDone(env, 1*time.Minute)
	assert.NoError(t, err)
	workQueue.Shutdown()

	checkWorkItemResults(env, keepFailedWorkItems, keepSuccessfulWorkItems)
}

func TestWorkItemsDeleteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	const (
		workItemTimeout         = 10 * time.Second
		keepFailedWorkItems     = false // delete work items on failure
		keepSuccessfulWorkItems = false // delete work items on failure
	)
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	env := newTestEnvironment(t, app, "WorkItemsDelete")

	// Set up work queue and handler for test work item type
	workQueue := app.WorkQueueService.(*work_queue.WorkQueueService)
	work_queue.NrWorkItemProcessors = 10 // test concurrency
	workQueue.Start()
	err = workQueue.RegisterHandler(
		testWorkItem,
		makeTestWorkItemHandler(env),
		workItemTimeout,
		instrumentedBackoff(env, work_queue.LinearBackoff(5, 1*time.Second)),
		keepFailedWorkItems,
		keepSuccessfulWorkItems,
	)
	require.NoError(t, err)

	// Work item to be processed successfully, no concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Work item to be processed successfully, unique concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-2", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Multiple work items to be processed successfully with the same concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-3", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-3", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-3", 1, 0, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Work item to be failed, no concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Work item to be failed, unique concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-4", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	// Multiple work items to be failed with the same concurrency key
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-5", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-5", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-5", 1, 1, nil)
	assert.NoError(t, err, "error adding work item to queue")

	err = waitUntilAllWorkItemsDone(env, 1*time.Minute)
	assert.NoError(t, err)
	workQueue.Shutdown()

	checkWorkItemResults(env, keepFailedWorkItems, keepSuccessfulWorkItems)
}

func TestWorkItemDeleteMixedStateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	const (
		workItemTimeout         = 10 * time.Second
		keepFailedWorkItems     = false // delete work items on failure
		keepSuccessfulWorkItems = true  // allows us to keep some but not all work items
	)
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	env := newTestEnvironment(t, app, "WorkItemStateDelete")

	// Set up work queue and handler for test work item type
	workQueue := app.WorkQueueService.(*work_queue.WorkQueueService)
	work_queue.NrWorkItemProcessors = 10 // test concurrency
	workQueue.Start()
	err = workQueue.RegisterHandler(
		testWorkItem,
		makeTestWorkItemHandler(env),
		workItemTimeout,
		nil, // use default backoff algorithm
		keepFailedWorkItems,
		keepSuccessfulWorkItems,
	)
	require.NoError(t, err)

	// Multiple work items, one succeeds and one fails. The successful one should be deleted and the failed
	// one should be kept, so the work item state overall should be kept
	expectedStateStillInDB := true
	_, err = submitNewWorkItem(env, &testWorkItemData{}, "key-7", 1, 0, &expectedStateStillInDB)
	assert.NoError(t, err, "error adding work item to queue")
	_, err = submitNewWorkItem(env, &testWorkItemData{FailPermanently: true}, "key-7", 1, 1, &expectedStateStillInDB)
	assert.NoError(t, err, "error adding work item to queue")

	err = waitUntilAllWorkItemsDone(env, 1*time.Minute)
	assert.NoError(t, err)
	workQueue.Shutdown()

	checkWorkItemResults(env, keepFailedWorkItems, keepSuccessfulWorkItems)
}

// makeTestWorkItemHandler returns a work item handler function that can exhibit various test behaviours
// when processing test work items.
func makeTestWorkItemHandler(env *testEnvironment) services.WorkItemHandler {
	t := env.t
	return func(ctx context.Context, workItem *models.WorkItem) (canRetry bool, err error) {
		workItemData := &testWorkItemData{}
		err = json.Unmarshal([]byte(workItem.Data), workItemData)
		if err != nil {
			env.t.Errorf("error unmarshaling test work item workItemData: %s", err)
			return false, err
		}

		t.Logf("Processing Work Item Nr %d for %s test", workItemData.ItemNr, env.testName)

		// Look up state for this work item
		env.statesMutex.Lock()
		defer env.statesMutex.Unlock()
		state, found := env.states[workItemData.ItemNr]
		if !found {
			// Can't use FailNow() or Fatalf() here since we're running in a different goroutine from the main test
			t.Errorf("error looking up test state for work item %d", workItemData.ItemNr)
			return false, fmt.Errorf("error looking up test state for work item; failing permanently")
		}

		// Work item should only be processed by a single processor at once
		assert.False(t, state.currentlyProcessing, "Work item processed by two processors at once!")
		state.currentlyProcessing = true
		defer func() { state.currentlyProcessing = false }()
		state.nrTimesProcessed++

		// Check if we should fail permanently
		if workItemData.FailPermanently {
			state.nrTimesFailed++
			state.done = true
			t.Logf("Finished processing Work Item %d, failing permanently (canRetry=false)", workItemData.ItemNr)
			return false, fmt.Errorf("error: test work item deliberately failed")
		}

		// Check if we should temporarily fail a certain number of times
		if workItemData.NrTimesToFailTemporarily > state.nrTimesFailed {
			state.nrTimesFailed++
			t.Logf("Finished processing Work Item %d, returning failure %d of %d",
				workItemData.ItemNr, state.nrTimesFailed, workItemData.NrTimesToFailTemporarily)
			return true, fmt.Errorf("error: test work item deliberately failed")
		}

		// Delay the processing if the work item requested a delay
		if workItemData.TimeToProcess > 0 {
			env.statesMutex.Unlock() // Unlock workItemStates while we delay...
			t.Logf("Delaying processing for Work Item %d for %s", workItemData.ItemNr, workItemData.TimeToProcess)
			err = sleepWithContext(ctx, workItemData.TimeToProcess)
			env.statesMutex.Lock() // ...but lock again right after; we deferred a call to Unlock() earlier
			if err != nil {
				state.nrTimesFailed++
				state.done = true
				t.Logf("Finished processing Work Item %d, timed out, failing permanently", workItemData.ItemNr)
				return false, err // processing was cancelled
			}
		}

		t.Logf("Finished processing Work Item Nr %d, returning success", workItemData.ItemNr)
		state.done = true
		return false, nil
	}
}

// instrumentedBackoff returns a backoff algorithm function that behaves just like the supplied regular
// backoff algorithm, but will update the work item's state when permanently failed because of too many errors.
func instrumentedBackoff(env *testEnvironment, backoffAlgorithm services.BackoffAlgorithm) services.BackoffAlgorithm {
	return func(attemptsSoFar int, lastAttemptAt time.Time, workItem *models.WorkItem) *time.Time {
		// Run the original backoff algorithm
		result := backoffAlgorithm(attemptsSoFar, lastAttemptAt, workItem)
		if result == nil {
			env.t.Logf("Too many errors processing work item of type %q, failing permanently", workItem.Type)
			workItemIsDone(env, workItem)
		}
		return result
	}
}

// Sleeps for the specified duration, but will return immediately if the specified context is cancelled or expires.
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
		return nil // The duration has elapsed
	case <-ctx.Done():
		return fmt.Errorf("context was cancelled or expired")
	}
}

// submitNewWorkItem creates a new work item and submits it to the work queue.
//
// The work item details are added to workItemStates including the work item ID and the ID of the WorkItemState
// object created or used,
//
// For expectedTimesProcessed expectedTimesFailed use -1 to mean 'any number is acceptable'
func submitNewWorkItem(
	env *testEnvironment,
	data *testWorkItemData,
	concurrencyKey models.WorkItemConcurrencyKey,
	expectedTimesProcessed int,
	expectedTimesFailed int,
	expectedStateStillInDB *bool,
) (*models.WorkItem, error) {
	workItem := newTestWorkItem(data, concurrencyKey)

	// Hold the lock for work item states the entire time we submit the work item, so we can record
	// the workItem.StateID before the work item can be processed.
	env.statesMutex.Lock()
	defer env.statesMutex.Unlock()

	// Create a state record to track the work item
	state := &workItemTestState{
		workItemID:             workItem.ID,
		concurrencyKey:         workItem.ConcurrencyKey,
		expectedTimesProcessed: expectedTimesProcessed,
		expectedTimesFailed:    expectedTimesFailed,
		expectedStateStillInDB: expectedStateStillInDB,
	}
	env.states[data.ItemNr] = state

	// Submit the new item to the work queue.
	err := env.app.WorkQueueService.AddWorkItem(context.Background(), nil, workItem)
	if err != nil {
		delete(env.states, data.ItemNr) // failed to submit work item so remove its state
		return nil, err
	}
	state.workItemStateID = workItem.StateID // record ID of work item state record used

	return workItem, nil
}

// workItemIsDone updates the state for the given work item to mark it as 'done'
func workItemIsDone(env *testEnvironment, workItem *models.WorkItem) {
	// Deserialize the work item's data to get ItemNr
	workItemData := &testWorkItemData{}
	err := json.Unmarshal([]byte(workItem.Data), workItemData)
	if err != nil {
		env.t.Errorf("error unmarshaling test work item workItemData: %s", err)
		return
	}

	// Look up state for work item by ItemNr
	env.statesMutex.Lock()
	defer env.statesMutex.Unlock()
	state, found := env.states[workItemData.ItemNr]
	if !found {
		// Can't use FailNow() or Fatalf() here since we're (very) likely to be on a goroutine
		env.t.Errorf("error looking up test state for work item %d", workItemData.ItemNr)
		return
	}

	state.done = true
}

// waitUntilAllWorkItemsDone waits until all work items in the specified test environment are done.
func waitUntilAllWorkItemsDone(env *testEnvironment, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for !allWorkItemsDone(env) {
		if time.Now().After(deadline) {
			return gerror.NewErrTimeout("waiting for all work items to be done")
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// allWorkItemsDone checks whether all work items in the specified test environment are done.
func allWorkItemsDone(env *testEnvironment) bool {
	env.statesMutex.Lock()
	defer env.statesMutex.Unlock()

	for _, state := range env.states {
		if !state.done {
			return false // any work item which isn't done means the test isn't done
		}
	}
	return true // all work items done
}

// checkWorkItemResults checks that all work item processing proceeded as expected.
//
// Checks the number of times each work item was processed and number of times failed.
//
// Checks that database records were either kept or deleted after processing, depending on the supplied
// values for keepFailedWorkItems and keepSuccessfulWorkItems.
func checkWorkItemResults(env *testEnvironment, keepFailedWorkItems bool, keepSuccessfulWorkItems bool) {
	ctx := context.Background()
	t := env.t

	env.statesMutex.Lock()
	defer env.statesMutex.Unlock()

	t.Logf("Checking results for %d work item(s) for %s test", len(env.states), env.testName)

	for itemNr, state := range env.states {
		assert.True(t, state.done, "Work item processing not done at end of test for item %d", itemNr)

		if state.expectedTimesProcessed != -1 {
			assert.Equal(t, state.expectedTimesProcessed, state.nrTimesProcessed, "Work item %d processed an unexpected number of times", itemNr)
		}
		if state.expectedTimesFailed != -1 {
			assert.Equal(t, state.expectedTimesFailed, state.nrTimesFailed, "Work item %d failed an unexpected number of times", itemNr)
		}

		// Work out whether work item should have been kept in the database
		var workItemShouldBeInDatabase bool
		if state.nrTimesFailed > 0 {
			workItemShouldBeInDatabase = keepFailedWorkItems
		} else {
			workItemShouldBeInDatabase = keepSuccessfulWorkItems
		}

		// Read work item from DB to see if it was deleted
		_, err := env.app.WorkItemStore.Read(ctx, nil, state.workItemID)
		workItemInDB := true
		if err != nil {
			if gerror.IsNotFound(err) {
				workItemInDB = false
			} else {
				t.Errorf("error attempting to read work item %d from database: %s", itemNr, err)
			}
		}
		if workItemShouldBeInDatabase && !workItemInDB {
			t.Errorf("Work item %d should have been kept in database but was not found", itemNr)
		} else if !workItemShouldBeInDatabase && workItemInDB {
			t.Errorf("Work item %d should have been deleted but is still in database", itemNr)
		}

		// Work out whether the work item's state record should have been kept in the database
		canCheckState := true
		stateShouldBeInDatabase := workItemShouldBeInDatabase
		if keepFailedWorkItems != keepSuccessfulWorkItems {
			// Whether state is still in database depends on the mix of failed and succeeded work items, so
			// just require the test writer to explicitly say whether we expect the state to still be there
			if state.expectedStateStillInDB != nil {
				stateShouldBeInDatabase = *state.expectedStateStillInDB
			} else {
				t.Error("expectedStateStillInDB must be specified in tests with a mix of deleting and not deleting work items")
				canCheckState = false
			}
		}

		if canCheckState {
			// Read work item state record from DB to see if it was deleted
			_, err = env.app.WorkItemStateStore.Read(ctx, nil, state.workItemStateID)
			workItemStateInDB := true
			if err != nil {
				if gerror.IsNotFound(err) {
					workItemStateInDB = false
				} else {
					t.Errorf("error attempting to read work item state for work item %d (state ID %s) from database: %s",
						itemNr, state.workItemStateID, err)
				}
			}
			if stateShouldBeInDatabase && !workItemStateInDB {
				t.Errorf("Work item state for work item %d (state ID %s) should have been kept in database but was not found",
					itemNr, state.workItemStateID)
			} else if !stateShouldBeInDatabase && workItemStateInDB {
				t.Errorf("Work item state for work item %d (state ID %s) should have been deleted but is still in database",
					itemNr, state.workItemStateID)
			}
		}
	}
}
