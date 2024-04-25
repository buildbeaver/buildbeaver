package work_items_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/store"
)

var (
	storeTestWorkItem       = models.WorkItemType("StoreTestWorkItem")
	allTestWorkItemTypes    = []models.WorkItemType{storeTestWorkItem}
	testWorkItemProcessorID = models.WorkItemProcessorID("TestWorkItemProcessorID")
	// testBaseTime is the time used as the start of the test when adding work items
	testBaseTime = time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC)
)

// timePlusXMinutes returns the time that is x minutes after the test base time.
func timePlusXMinutes(minutes int64) models.Time {
	theTime := testBaseTime.Add(time.Duration(minutes * int64(time.Minute)))
	return models.NewTime(theTime)
}

func timePlusXMinutesPtr(minutes int64) *models.Time {
	t := timePlusXMinutes(minutes)
	return &t
}

func TestWorkItem(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err, "Error initializing app")
	defer cleanup()

	t.Run("WorkItemLifeCycle", testWorkItemLifeCycle(app.WorkItemStore, app.WorkItemStateStore, app.DB))
	t.Run("WorkItemDeletion", testWorkItemDeletion(app.WorkItemStore, app.WorkItemStateStore, app.DB))
}

func testWorkItemLifeCycle(workItemStore store.WorkItemStore, stateStore store.WorkItemStateStore, db *store.DB) func(t *testing.T) {
	return func(t *testing.T) {
		const (
			concurrencyKey1 = "concurrency-key-1"
			concurrencyKey2 = "concurrency-key-2"
		)
		var err error

		// time t+0 (times are in minutes after base time, virtual time rather than real time)
		// Attempt to find queued work item, should not find one
		testFindQueuedWorkItem(t, db, timePlusXMinutes(0), stateStore, nil)

		// time t+10: Create new work item
		workItem := models.NewWorkItem(storeTestWorkItem, "Test Data", concurrencyKey1, timePlusXMinutes(10))
		records, err := insertNewWorkItem(workItemStore, stateStore, db, workItem)
		require.NoError(t, err, "error creating work item")

		// Read it back
		testWorkItemRead(t, workItemStore, stateStore, workItem.ID, workItem, false, records)

		// time t+10: Repeat attempt to find queued work item, should find it now
		testFindQueuedWorkItem(t, db, timePlusXMinutes(10), stateStore, &workItem.ID)

		// time t+15: Update work item to pretend it is allocated
		records.State.AllocatedTo = &testWorkItemProcessorID
		records.State.AllocatedAt = timePlusXMinutesPtr(15)
		records.State.AllocatedUntil = timePlusXMinutesPtr(20)
		err = stateStore.Update(context.Background(), nil, records.State)
		assert.NoError(t, err, "error updating work item (1)")

		// time t+15: Work item should now not be found since it's already allocated
		testFindQueuedWorkItem(t, db, timePlusXMinutes(15), stateStore, nil)

		// time t+19: Work item should now not be found since it's already allocated
		testFindQueuedWorkItem(t, db, timePlusXMinutes(19), stateStore, nil)

		// time t+21: Work item should now be found since it's allocation has expired
		testFindQueuedWorkItem(t, db, timePlusXMinutes(21), stateStore, &workItem.ID)

		// time t+25: Update work item to pretend it is completed
		records.State.AllocatedTo = nil
		records.State.AllocatedUntil = nil
		records.Record.CompletedAt = timePlusXMinutesPtr(25)
		err = stateStore.Update(context.Background(), nil, records.State)
		assert.NoError(t, err, "error updating work item state record (2)")
		err = workItemStore.Update(context.Background(), nil, records.Record)
		assert.NoError(t, err, "error updating work item (2)")

		// time t+30: Work item should not be found since it is completed
		testFindQueuedWorkItem(t, db, timePlusXMinutes(30), stateStore, nil)

		// time t+40: Create another new work item (independent of first one)
		workItem = models.NewWorkItem(storeTestWorkItem, "Test Data 2", concurrencyKey2, timePlusXMinutes(40))
		records, err = insertNewWorkItem(workItemStore, stateStore, db, workItem)
		require.NoError(t, err, "error creating work item 2")

		// time t+41: Work item should be found
		testFindQueuedWorkItem(t, db, timePlusXMinutes(41), stateStore, &workItem.ID)

		// Delete the second work item
		testWorkItemDelete(t, workItemStore, workItem.ID)

		// Try to read second work item back, should be missing
		testWorkItemRead(t, workItemStore, stateStore, workItem.ID, workItem, true, nil)

		// time t+41: No work items should not be found in the queue now
		testFindQueuedWorkItem(t, db, timePlusXMinutes(41), stateStore, nil)
	}
}

func testWorkItemDeletion(workItemStore store.WorkItemStore, stateStore store.WorkItemStateStore, db *store.DB) func(t *testing.T) {
	return func(t *testing.T) {
		const concurrencyKey3 = "concurrency-key-3"
		var err error

		// time t+10 to t+12: Create 3 new work item
		workItem1 := models.NewWorkItem(storeTestWorkItem, "Test Data 1", concurrencyKey3, timePlusXMinutes(10))
		records1, err := insertNewWorkItem(workItemStore, stateStore, db, workItem1)
		require.NoError(t, err, "error creating work item")

		workItem2 := models.NewWorkItem(storeTestWorkItem, "Test Data 2", concurrencyKey3, timePlusXMinutes(11))
		records2, err := insertNewWorkItem(workItemStore, stateStore, db, workItem2)
		require.NoError(t, err, "error creating work item")

		workItem3 := models.NewWorkItem(storeTestWorkItem, "Test Data 3", concurrencyKey3, timePlusXMinutes(12))
		records3, err := insertNewWorkItem(workItemStore, stateStore, db, workItem3)
		require.NoError(t, err, "error creating work item")

		// time t+15: Find queued work item, should find the first one
		testFindQueuedWorkItem(t, db, timePlusXMinutes(15), stateStore, &workItem1.ID)

		// Delete the first work item
		err = workItemStore.Delete(context.Background(), nil, records1.Record.ID)
		assert.NoError(t, err, "error deleting work item (1)")

		// time t+15: We should now find the second work item
		testFindQueuedWorkItem(t, db, timePlusXMinutes(15), stateStore, &records2.Record.ID)

		// Delete the second work item
		err = workItemStore.Delete(context.Background(), nil, records2.Record.ID)
		assert.NoError(t, err, "error deleting work item (2)")

		// time t+15: We should now find the third work item
		testFindQueuedWorkItem(t, db, timePlusXMinutes(15), stateStore, &records3.Record.ID)

		// Delete the third work item
		err = workItemStore.Delete(context.Background(), nil, records3.Record.ID)
		assert.NoError(t, err, "error deleting work item (3)")

		// time t+15: We should now find no work items are left
		testFindQueuedWorkItem(t, db, timePlusXMinutes(15), stateStore, nil)
	}
}

func insertNewWorkItem(
	workItemStore store.WorkItemStore,
	stateStore store.WorkItemStateStore,
	db *store.DB,
	workItem *models.WorkItem,
) (*models.WorkItemRecords, error) {
	var err error
	ctx := context.Background()
	records := &models.WorkItemRecords{}
	now := models.NewTime(time.Now())
	err = db.WithTx(ctx, nil, func(tx *store.Tx) error {
		// Create a state record
		records.State, err = stateStore.FindOrCreateAndLockRow(ctx, tx, models.NewWorkItemState(now, workItem.ConcurrencyKey))
		if err != nil {
			return fmt.Errorf("error finding or creating state record for concurrency key %s: %w", workItem.ConcurrencyKey, err)
		}
		// Create a record for the work item itself
		workItem.StateID = records.State.ID
		records.Record = workItem
		err = workItemStore.Create(ctx, tx, records.Record)
		if err != nil {
			return fmt.Errorf("error creating work item record: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func testWorkItemRead(
	t *testing.T,
	workItemStore store.WorkItemStore,
	stateStore store.WorkItemStateStore,
	workItemID models.WorkItemID,
	originalWorkItem *models.WorkItem,
	expectMissing bool,
	expectedRecords *models.WorkItemRecords,
) {
	ctx := context.Background()

	// Read and check work item record
	workItemRecord, err := workItemStore.Read(ctx, nil, workItemID)
	if expectMissing {
		require.Error(t, err, "Expected error reading missing workItem, no error returned")
	} else {
		require.NoError(t, err, "Error reading record for work item")
	}
	if err != nil {
		return // nothing more we can do if there is no work item record
	}
	assert.Equal(t, originalWorkItem.ID, workItemRecord.ID)
	assert.Equal(t, originalWorkItem.CreatedAt, workItemRecord.CreatedAt)
	assert.Equal(t, originalWorkItem.ConcurrencyKey, workItemRecord.ConcurrencyKey)
	assert.Equal(t, originalWorkItem.Type, workItemRecord.Type)
	assert.Equal(t, originalWorkItem.Data, workItemRecord.Data)
	assert.Equal(t, originalWorkItem.Status, workItemRecord.Status)
	assert.Equal(t, expectedRecords.Record.CompletedAt, workItemRecord.CompletedAt)

	// Read and check state record
	state, err := stateStore.Read(ctx, nil, workItemRecord.StateID)
	require.NoError(t, err, "Error reading state record for work item")
	assert.Equal(t, workItemRecord.StateID, state.ID)
	assert.Equal(t, expectedRecords.State.AllocatedTo, state.AllocatedTo)
	assert.Equal(t, expectedRecords.State.AllocatedUntil, state.AllocatedUntil)
	assert.Equal(t, expectedRecords.State.AllocatedAt, state.AllocatedAt)
}

func testFindQueuedWorkItem(
	t *testing.T,
	db *store.DB,
	now models.Time,
	stateStore store.WorkItemStateStore,
	expectedWorkItemID *models.WorkItemID,
) {
	// Start a new transaction since this is what WorkQueueService does to atomically read and allocate
	// a work item using row locking
	err := db.WithTx(context.Background(), nil, func(tx *store.Tx) error {
		records, err := stateStore.FindQueuedWorkItem(context.Background(), tx, now, allTestWorkItemTypes)
		if expectedWorkItemID != nil {
			assert.NoError(t, err, "Unexpected error returned from FindQueuedWorkItem()")
			if err == nil {
				assert.NotNil(t, records, "Expected work item to be returned from FindQueuedWorkItem")
				if records != nil {
					assert.Equal(t, *expectedWorkItemID, records.Record.ID)
				}
			}
		} else {
			assert.Error(t, err, "Expected FindQueuedWorkItem() to not find any work items and return an error")
		}
		return nil
	})
	require.NoError(t, err)
}

func testWorkItemDelete(t *testing.T, store store.WorkItemStore, workItemID models.WorkItemID) {
	err := store.Delete(context.Background(), nil, workItemID)
	require.NoError(t, err, "Error deleting workItem")
}
