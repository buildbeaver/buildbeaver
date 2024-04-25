package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

type scenario struct {
	inputs          []string
	expectedOutputs []string
	readChunkSize   int
	streamSize      int
	errorRequests   []errorRequest
}

// errorRequest tells the log stream factory to cause a write error for the log streaming for a particular stream
// and log number. The streams for the log are numbered from 1, and the writes for a stream are numbered from 1.
// If permanent is false then the specified write number for the specified stream number will return a temporary error.
// If permanent is true then this and all subsequent writes to this log descriptor (including all writes to any
// subsequent stream number) will return a permanent error.
type errorRequest struct {
	streamNo  int  // stream number to close (starting from 1)
	writeNo   int  // write number to close (starting from 1)
	permanent bool // true if all subsequent writes to this log descriptor should be failed
}

func TestLogPipeline(t *testing.T) {

	logBufferDir := RunnerLogTempDirectory(t.TempDir())

	var longString string
	for i := 1; i <= 1000; i++ {
		longString += fmt.Sprintf("%d-1234567890---", i)
	}

	scenarios := []scenario{
		{
			// Test basic data with default read chunk size and stream size, with a retry
			inputs:          []string{"Hello world", "Hello World", "wor", "ld", "helloworld", "hello\nworld"},
			expectedOutputs: []string{"Hello world", "Hello World", "wor", "ld", "helloworld", "hello\nworld"},
			readChunkSize:   0,
			streamSize:      0,
			errorRequests:   []errorRequest{{streamNo: 2, writeNo: 2}},
		},
		{
			// Test small stream and read chunk sizes, with a retry
			inputs:          []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld"},
			expectedOutputs: []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld"},
			readChunkSize:   20,
			streamSize:      3,
			errorRequests:   []errorRequest{{streamNo: 2, writeNo: 2}},
		},
		{
			// Test small stream and read chunk sizes, with multiple retries
			inputs:          []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld", "Line 7", "Line 8", "Line 9"},
			expectedOutputs: []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld", "Line 7", "Line 8", "Line 9"},
			readChunkSize:   1000,
			streamSize:      3,
			errorRequests: []errorRequest{
				{streamNo: 1, writeNo: 1},
				{streamNo: 2, writeNo: 1},
				{streamNo: 3, writeNo: 1},
				{streamNo: 4, writeNo: 2},
				{streamNo: 5, writeNo: 4},
			},
		},
		{
			// Test specific error and test data combination that was causing a problem with the test infrastructure
			inputs:          []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld", "Line 7", "Line 8", "Line 9"},
			expectedOutputs: []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld", "Line 7", "Line 8", "Line 9"},
			readChunkSize:   1000,
			streamSize:      3,
			errorRequests:   []errorRequest{{streamNo: 1, writeNo: 4}},
		},
		{
			// Test for permanent errors, to ensure the pipeline gives up
			inputs:          []string{"Log line 1", "2", longString, "", "This is a bit of a longer string", "hello\nworld", "Line 7", "Line 8", "Line 9"},
			expectedOutputs: []string{"Log line 1", "2", longString},
			readChunkSize:   1000,
			streamSize:      3,
			errorRequests: []errorRequest{
				{streamNo: 2, writeNo: 1, permanent: true},
			},
		},
	}

	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	for i, scenario := range scenarios {
		t.Logf("Running Log Pipeline test scenario %d of %d", i+1, len(scenarios))

		logID := models.NewLogDescriptorID()
		logStore := NewFakeLogStore(t)
		errorRequests := NewErrorRequestList()
		logStreamFactory := NewFakeLogStreamFactory(t, logStore, errorRequests)

		// Make a full log pipeline
		pipeline, err := NewClientLogPipeline(
			context.Background(), // no need to cancel requests to the 'server'
			clock.New(),          // use a real-time clock for timestamps in the logs
			logFactory,           // this is for logging from the pipeline itself, not the entries flowing through it
			logStreamFactory,
			logID,
			[]*models.SecretPlaintext{}, // no secrets; we're not testing secrets in this test
			logBufferDir,
			scenario.readChunkSize,
			scenario.streamSize,
			0, // always use default max stream duration, should be long enough for tests
		)
		require.NoError(t, err)

		// Add in some errors
		for _, req := range scenario.errorRequests {
			errorRequests.AddErrorRequest(logID, &req)
		}

		for i, input := range scenario.inputs {
			t.Logf("Writing entry %d of %d", i+1, len(scenario.inputs))
			pipeline.writer.Write(models.NewLogEntryLine(i+1, models.NewTime(time.Now()), input, i+1, nil))
		}
		t.Logf("All entries written, flushing pipeline")
		pipeline.Flush() // must call Flush before Close
		t.Logf("All entries flushed, closing pipeline")
		pipeline.Close()
		t.Logf("Pipeline closed; checking output")

		assert.Equal(t, len(scenario.expectedOutputs), logStore.GetEntryCount())
		for i := 0; i < logStore.GetEntryCount(); i++ {
			expected := scenario.expectedOutputs[i]
			actual := logStore.GetEntry(i)
			require.NotNil(t, actual)
			text := actual.Derived().(models.PlainTextLogEntry).GetText()
			assert.Equal(t, expected, text)
		}
	}
}

// FakeLogStore provides a thread-safe place to store and retrieve log entries during testing.
// Entries are append-only.
type FakeLogStore struct {
	t       *testing.T
	mu      sync.Mutex
	entries []*models.LogEntry
}

func NewFakeLogStore(t *testing.T) *FakeLogStore {
	return &FakeLogStore{t: t}
}

func (f *FakeLogStore) StoreEntry(entry *models.LogEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, entry)
}

func (f *FakeLogStore) GetEntryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

// GetEntry returns the log entry at the specified index within the list, or nil of the index is out of bounds.
// The store is append-only so it's safe to use any index between 0 and the results of a previous call to GetEntryCount().
func (f *FakeLogStore) GetEntry(index int) *models.LogEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index >= 0 && index < len(f.entries) {
		return f.entries[index]
	} else {
		return nil
	}
}

// ErrorRequestList maintains a list of errorRequest objects, to define a set of errors that should
// be caused during testing. Allows checking whether a given operation should return an error.
type ErrorRequestList struct {
	mu       sync.Mutex
	requests map[models.LogDescriptorID][]*errorRequest
}

func NewErrorRequestList() *ErrorRequestList {
	return &ErrorRequestList{
		requests: make(map[models.LogDescriptorID][]*errorRequest),
	}
}

func (l *ErrorRequestList) AddErrorRequest(logID models.LogDescriptorID, req *errorRequest) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests[logID] = append(l.requests[logID], req)
}

// CheckForError checks to see whether the specified write number for the specified stream and
// log should return an error. Returns the error to return, or nil if the write operation should succeed,
// as well as whether any returned error should be permanent.
func (l *ErrorRequestList) CheckForError(logID models.LogDescriptorID, streamNo int, writeNo int) (error, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Search through all error requests for this log descriptor
	requests := l.requests[logID]
	for _, req := range requests {
		if req.permanent {
			// Look for a permanent error at or before the current stream and write number
			if streamNo > req.streamNo || (streamNo == req.streamNo && writeNo >= req.writeNo) { // Record permanent error, so we can return it from Close()
				// Use LogClosed as a representative permanent error
				return gerror.NewErrLogClosed(), true
			}
		} else {
			// Look for an exact match for a temporary error
			if streamNo == req.streamNo && writeNo == req.writeNo {
				return fmt.Errorf("temporary error writing bytes to stream: test error"), req.permanent
			}
		}
	}
	return nil, false // no error found
}

// FakeLogStreamFactory will provide a function to create instances of FakeLogStreamWriter, allowing a stream of
// bytes containing JSON log entries to be written from a log pipeline, and causing errors as requested.
// The test data written is parsed into log entries and stored in a FakeLogStore.
type FakeLogStreamFactory struct {
	t             *testing.T
	mu            sync.Mutex
	store         *FakeLogStore
	errorRequests *ErrorRequestList
	streamCounts  map[models.LogDescriptorID]int // counts how many streams have been created for each LogID
}

func NewFakeLogStreamFactory(t *testing.T, store *FakeLogStore, errorRequests *ErrorRequestList) *FakeLogStreamFactory {
	return &FakeLogStreamFactory{
		t:             t,
		store:         store,
		errorRequests: errorRequests,
		streamCounts:  make(map[models.LogDescriptorID]int),
	}
}

// OpenLogWriteStream will open a FakeLogStreamWriter to accept the stream output from the test log pipeline.
func (f *FakeLogStreamFactory) OpenLogWriteStream(ctx context.Context, logID models.LogDescriptorID) (io.WriteCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	streamCount, found := f.streamCounts[logID]
	if !found {
		streamCount = 0
	}
	streamCount++
	f.streamCounts[logID] = streamCount

	writer := NewFakeLogStreamWriter(f.t, f.store, f.errorRequests, streamCount, logID)
	return writer, nil
}

// FakeLogStreamWriter allows a stream of bytes containing JSON log entries to be written from a log pipeline, and
// can introduce errors and failures. The test data written is parsed into log entries, and if the entire stream
// write succeeded and formed a valid JSON document then the parsed entries are stored in a FakeLogStore.
type FakeLogStreamWriter struct {
	t              *testing.T
	store          *FakeLogStore
	errorRequests  *ErrorRequestList
	streamNo       int
	writeNo        int
	logID          models.LogDescriptorID
	pipeReader     *io.PipeReader
	pipeWriter     *io.PipeWriter
	parserDone     chan error
	permanentError error // error to return from now on, including from Close(), or nil for no permanent error
}

func NewFakeLogStreamWriter(
	t *testing.T,
	store *FakeLogStore,
	errorRequests *ErrorRequestList,
	streamNo int,
	logID models.LogDescriptorID,
) *FakeLogStreamWriter {
	pipeReader, pipeWriter := io.Pipe()
	w := &FakeLogStreamWriter{
		t:             t,
		store:         store,
		errorRequests: errorRequests,
		streamNo:      streamNo,
		writeNo:       0,
		logID:         logID,
		pipeReader:    pipeReader,
		pipeWriter:    pipeWriter,
		parserDone:    make(chan error),
	}

	go func() {
		entries, err := parse(w.t, w.pipeReader)
		if err != nil {
			t.Logf("warning: parser returned the following error: %v", err.Error())
			w.parserDone <- err
			return
		}
		// Store entries only if the document was parsed successfully
		for _, entry := range entries {
			store.StoreEntry(entry)
		}
		w.parserDone <- nil // no error
	}()

	return w
}

func (w *FakeLogStreamWriter) Write(bytes []byte) (int, error) {
	w.writeNo++
	w.t.Logf("Write called with %d bytes, streamNo %d, writeNo %d", len(bytes), w.streamNo, w.writeNo)

	// If we've seen a permanent error then keep returning it
	if w.permanentError != nil {
		return 0, w.permanentError
	}

	err, permanent := w.errorRequests.CheckForError(w.logID, w.streamNo, w.writeNo)
	if err != nil {
		if permanent {
			w.permanentError = err
		}
		w.t.Logf("FakeLogStreamWriter: Failing Write() call for logID %s, stream %d, write %d with error: %v", w.logID, w.streamNo, w.writeNo, err)
		return 0, err
	}

	// Send the data down the pipeline to the log entry parser
	w.t.Logf("FakeLogStreamWriter: Writing %d bytes", len(bytes))
	_, err = w.pipeWriter.Write(bytes)
	require.NoError(w.t, err)

	return len(bytes), nil
}

func (w *FakeLogStreamWriter) Close() error {
	err := w.pipeWriter.Close()
	if err != nil {
		return err
	}

	// Wait for parser to finish, to confirm that the document is valid; any error must be returned
	parserErr := <-w.parserDone

	// If we have a permanent error then return it to the pipeline
	if w.permanentError != nil {
		return w.permanentError
	}
	// If no permanent error was thrown by the test then return any parser error
	if parserErr != nil {
		return parserErr
	}

	return nil
}

// parse will read and parse log entries from the supplied reader until the end of the document or the end of the
// stream is reached.
// Returns all the log entries if parsing was successful, or an error.
func parse(t *testing.T, reader io.Reader) ([]*models.LogEntry, error) {
	// Accumulate entries in this array until the end
	var allEntries []*models.LogEntry

	dec := json.NewDecoder(reader)
	token, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("error reading opening token: %w", err)
	}
	if token != json.Delim('[') {
		return nil, fmt.Errorf("error expected first token to begin array (\"[\"), found: %s", token)
	}
	for dec.More() {
		entry := &models.LogEntry{}
		err := dec.Decode(entry)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling entry from JSON: %w", err)
		}
		persistent, ok := entry.Derived().(models.PersistentLogEntry)
		if !ok {
			return nil, fmt.Errorf("error expected to read persistent log entries from reader")
		}
		t.Logf("Read log entry from stream, Seq No %d", persistent.GetSeqNo())
		// Save the entry for checking later
		allEntries = append(allEntries, entry)
	}
	_, err = dec.Token()
	if err != nil {
		return nil, fmt.Errorf("error reading closing token: %w", err)
	}
	t.Logf("Read closing token from log entry stream")
	return allEntries, nil
}
