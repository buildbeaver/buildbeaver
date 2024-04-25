package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

// Maximum number of log entries to send in each stream, before ending the stream HTTP operation and getting
// a status code back from the server to confirm that all entries in that stream have been stored.
// Log entries are typically 150-250 bytes each in JSON, so 10,000 entries is 1.5 to 2.5MB of logs
const defaultMaxStreamSize = 10000

// Maximum duration for each stream, before ending the stream HTTP operation and getting a status code back
// from the server to confirm that all entries in that stream have been stored.
// This value should be significantly less than the server's timeout duration for a WriteLog HTTP request,
// in order to prevent timeout errors when writing logs with regular writes of a small amount of data that
// hit the server timeout before hitting defaultMaxStreamSize.
const defaultMaxStreamDuration = 1 * time.Minute // assuming server's timeout is significantly more than 1 minute

type LogStreamFactory interface {
	// OpenLogWriteStream opens a writable stream to the specified log. Close the writer to finish writing.
	OpenLogWriteStream(ctx context.Context, logID models.LogDescriptorID) (io.WriteCloser, error)
}

// LogStreamer streams structured logs to the server.
// Logs are written in 'streams' where each stream is sent via an HTTP operation, and will send up to a
// fixed maximum number of log entries and stream for a fixed maximum time. A new stream will only be opened
// once the HTTP operation for the previous stream has returned a result, indicating that its log entries
// have been safely written to persistent storage by the server.
type LogStreamer struct {
	mu                        sync.Mutex
	ctx                       context.Context
	log                       logger.Log
	closePipeline             closeRequester
	streamFactory             LogStreamFactory
	logDescriptorID           models.LogDescriptorID
	maxStreamSize             int
	maxStreamDuration         time.Duration
	confirmationChannelsMutex sync.Mutex
	confirmationChannels      []chan LogConfirmation // each confirmation will be sent to each of these channels
	state                     struct {
		streamWriter              io.WriteCloser
		streamWriterOpenedAt      time.Time
		streamOpeningTokenWritten bool // true if opening token written to current stream
		streamEntriesWritten      int  // number of log entries written to current stream
		streamFirstSeqNo          int  // sequence number of the first log entry in the current stream
		streamLastSeqNoWritten    int  // sequence number of the most recent log entry written to the current stream
		lastSeqNoConfirmed        int  // sequence number of the most recent log entry confirmed by the server (in any stream)
		waitingForRetry           bool // true if waiting for log entries to be re-sent to this stage
		retryFromSeqNo            int  // if waitingForRetry, SeqNo waiting to be re-sent (other SeqNos will be discarded)
		logClosed                 bool
	}
}

// NewLogStreamer creates a new LogStreamer pipeline stage.
// If maxStreamSize is zero then the default value will be used (recommended).
func NewLogStreamer(
	ctx context.Context,
	logFactory logger.LogFactory,
	closePipeline closeRequester,
	streamFactory LogStreamFactory,
	logDescriptorID models.LogDescriptorID,
	maxStreamSize int,
	maxStreamDuration time.Duration,
) *LogStreamer {
	if maxStreamSize == 0 {
		maxStreamSize = defaultMaxStreamSize
	}
	if maxStreamDuration == 0 {
		maxStreamDuration = defaultMaxStreamDuration
	}
	w := &LogStreamer{
		ctx:               ctx,
		log:               logFactory("LogStreamer"),
		closePipeline:     closePipeline,
		streamFactory:     streamFactory,
		logDescriptorID:   logDescriptorID,
		maxStreamSize:     maxStreamSize,
		maxStreamDuration: maxStreamDuration,
	}
	return w
}

// SetStreamSize sets the number of log entries to send in each stream.
// This would normally only be set when running automated tests.
func (l *LogStreamer) SetStreamSize(size int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.maxStreamSize = size
}

// Write a new entry to the stream.
func (l *LogStreamer) Write(entry *models.LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state.logClosed {
		l.log.Warnf("Attempt to write to log stream after log is closed; discarding log entry")
		return
	}

	err := l.write(&documents.LogEntry{LogEntry: entry})
	if err != nil {
		l.log.Errorf("Error writing log entry: %v", err)
	}
}

func (l *LogStreamer) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state.logClosed || l.state.streamWriter == nil {
		return
	}
	l.finishStream()
}

// Close the log stream, sending a closing end array character ']'.
// If the server returns an error then any previously sent log entries which need to be re-sent will
// be discarded instead; this operation will complete relatively quickly and will not wait for retries.
func (l *LogStreamer) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state.logClosed {
		// Do not warn if log is already closed since this happens when a permanent error is received
		return
	}

	// Write closing token ']' and close the stream
	// If this fails then a retry will be requested back to upstream
	l.finishStream()
	// Close the log anyway; any retry from upstream from here on will fail but that's OK in Close()
	l.state.logClosed = true
}

// write will write a log entry. An error is returned only if the log entry can't be written and might be lost;
// if a retry is requested then this will be logged but no error will be returned.
func (l *LogStreamer) write(entry *documents.LogEntry) error {
	// Find the sequence number of this log entry, or 0 if not a persistent log entry
	entrySeqNo := 0
	if persistentEntry, ok := entry.Derived().(models.PersistentLogEntry); ok {
		entrySeqNo = persistentEntry.GetSeqNo()
	}

	// If we are waiting for a retry, discard log entries until we see the one we want to retry from
	if l.state.waitingForRetry {
		if entrySeqNo == l.state.retryFromSeqNo {
			// This is the sequence number we've been waiting for; go back to normal processing
			l.state.waitingForRetry = false
			l.state.retryFromSeqNo = 0
		} else {
			// Discard this log entry; we are waiting for an entry with sequence number retryFromSeqNo to arrive
			l.log.Tracef("Discarding log entry with SeqNo %d while waiting for retry of SeqNo %d",
				entrySeqNo, l.state.retryFromSeqNo)
			return nil
		}
	}

	buf, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshalling entry to JSON: %w", err)
	}

	stream, err := l.findOrCreateStream()
	if err != nil {
		return fmt.Errorf("error getting stream handle: %w", err)
	}

	// Do this after calling findOrCreateStream(), which will reset streamFirstSeqNo and streamOpeningTokenWritten
	if entrySeqNo != 0 && l.state.streamFirstSeqNo == 0 {
		// This will be the first persistent log entry we've written to this stream
		// Update the variable before attempting to write to the stream, so it's available if there's an error
		l.state.streamFirstSeqNo = entrySeqNo
	}
	if !l.state.streamOpeningTokenWritten {
		buf = append([]byte{'['}, buf...)
		l.state.streamOpeningTokenWritten = true
	} else {
		buf = append([]byte{','}, buf...)
	}

	// From Golang docs for io package: "Write must return a non-nil error if it returns n < len(p)"
	// i.e. write will always either write *all* the data or return an error
	l.log.Tracef("log_streamer writing log entry with SeqNo %d (streamFirstSeqNo=%d)", entrySeqNo, l.state.streamFirstSeqNo)
	_, err = stream.Write(buf)
	if err != nil {
		// Write errors are handled via retries, so don't return an error
		l.finishStreamWithWriteError(err)
		return nil
	}
	if flusher, ok := stream.(http.Flusher); ok {
		flusher.Flush()
	}

	// Update count and sequence number state
	l.state.streamEntriesWritten++
	if entrySeqNo != 0 {
		l.state.streamLastSeqNoWritten = entrySeqNo
	}

	// If we have written enough log entries, or if the stream has been running long enough, then finish the stream;
	// the next write will start a new stream
	now := time.Now().UTC()
	openTooLong := now.After(l.state.streamWriterOpenedAt.Add(l.maxStreamDuration))
	if l.state.streamEntriesWritten >= l.maxStreamSize || openTooLong {
		l.finishStream()
	}

	return nil
}

func (l *LogStreamer) findOrCreateStream() (io.WriteCloser, error) {
	if l.state.logClosed {
		return nil, fmt.Errorf("error: log is closed")
	}
	if l.state.streamWriter != nil {
		return l.state.streamWriter, nil
	}
	writer, err := l.streamFactory.OpenLogWriteStream(l.ctx, l.logDescriptorID)
	if err != nil {
		return nil, fmt.Errorf("error opening stream: %w", err)
	}

	// Reset the per-stream state
	l.state.streamWriter = writer
	l.state.streamWriterOpenedAt = time.Now().UTC()
	l.state.streamOpeningTokenWritten = false
	l.state.streamEntriesWritten = 0
	l.state.streamFirstSeqNo = 0
	l.state.streamLastSeqNoWritten = 0

	return l.state.streamWriter, nil
}

// finishStream will finish writing to any currently open stream and close the stream.
// The closing ']' will be written to the stream before it is closed, to complete the JSON document being streamed.
// A confirmation (either success or error) will be sent to all registered confirmation channels specifying
// whether the stream succeeded or failed.
func (l *LogStreamer) finishStream() {
	if l.state.streamWriter != nil {
		// Write the closing ']' character to finish the stream
		if l.state.streamOpeningTokenWritten {
			// If we fail to write the ']' this may or may not cause a problem; only send an error confirmation
			// if there is an error in the call to Close() below
			l.state.streamWriter.Write([]byte{']'})
		}
		if flusher, ok := l.state.streamWriter.(http.Flusher); ok {
			flusher.Flush()
		}
		// The error returned from Close() determines whether the document was accepted and confirmed as stored
		writeErr := l.state.streamWriter.Close()
		l.state.streamWriter = nil

		// Send confirmations based on the result coming back from the server
		if writeErr != nil {
			l.log.Warnf("got error from server when closing log stream in finishStream: %s", writeErr.Error())
			l.processError(writeErr)
		} else {
			l.sendSuccessConfirmation()
		}
	}
}

// finishStreamWithWriteError will terminate writing to any currently open stream after an error was returned
// from a call to stream.Write().
// The closing ']' will NOT be written to the stream before it is closed, so the stream will not form a valid JSON
// document. An error confirmation will be sent to all registered confirmation channels specifying
// that the stream failed, using the most relevant error obtained from the server.
func (l *LogStreamer) finishStreamWithWriteError(writeError error) {
	errorToProcess := writeError
	if l.state.streamWriter != nil {
		// Just close the stream; do not write the closing ']' character
		closeErr := l.state.streamWriter.Close()
		if closeErr != nil {
			// The error returned by Close() is the error from the HTTP connection; this must take precedence
			// over writeError which can be misleading (e.g. it can often be "io: read/write on closed pipe")
			l.log.Warnf("got error from server when closing log stream; replacing original error which was '%v'", writeError)
			errorToProcess = closeErr
		}
		l.state.streamWriter = nil
	}
	l.processError(errorToProcess)
}

func (l *LogStreamer) sendSuccessConfirmation() {
	// Stream succeeded; if any persistent entries were included in the stream then they are now confirmed
	if l.state.streamLastSeqNoWritten != 0 {
		l.state.lastSeqNoConfirmed = l.state.streamLastSeqNoWritten
		l.sendConfirmation(NewSuccessConfirmation(l.state.lastSeqNoConfirmed))
	}
}

func (l *LogStreamer) processError(streamError error) {
	if l.shouldRetry(streamError) {
		l.log.Warnf("Temporary error returned when writing to log stream, sending an error/retry confirmation; error: %s", streamError.Error())
		// If this stage has been asked to write one or more persistent log entries to this stream, send an
		// error confirmation to force a retry; otherwise nothing was missed so the error can be ignored
		if l.state.streamFirstSeqNo != 0 {
			// Change current state to wait for entries to be re-sent
			l.state.waitingForRetry = true
			l.state.retryFromSeqNo = l.state.streamFirstSeqNo
			// Send an error confirmation asking for a retry
			l.sendConfirmation(NewErrorConfirmation(streamError, l.state.retryFromSeqNo))
		}
	} else {
		// We can't retry this error so ignore any further writes and close the log.
		// Do not request a retry and don't send any more confirmations. Do not pass Go, do not collect $200.
		l.log.Errorf("Permanent error returned when writing to log stream, closing the log; error: %s", streamError.Error())
		l.state.logClosed = true
		// Tell the pipeline to close; this will cancel any outstanding flush requests, which would otherwise never
		// complete because no further confirmations will be sent back
		l.closePipeline()
	}
}

func (l *LogStreamer) shouldRetry(streamError error) bool {
	if streamError == nil {
		return false
	}
	// If the log streamer's overall context has been cancelled or timed out (e.g. DeadlineExceeded
	// because the maximum build duration has elapsed) then don't retry; any further attempts to contact the server
	// will immediately result in a context error again.
	if l.ctx.Err() != nil {
		return false
	}
	errorDoc, ok := streamError.(gerror.Error)
	if ok {
		// Do not retry 400 errors
		if errorDoc.HTTPStatusCode() >= 400 && errorDoc.HTTPStatusCode() < 500 {
			return false
		}
	}
	return true // default to retrying
}

// RegisterConfirmationChannel allows an interested party to register to receive a copy of any confirmations.
// The supplied channel should be buffered so as not to make the log streamer block.
func (l *LogStreamer) RegisterConfirmationChannel(ch chan LogConfirmation) {
	l.confirmationChannelsMutex.Lock()
	defer l.confirmationChannelsMutex.Unlock()

	l.confirmationChannels = append(l.confirmationChannels, ch)
}

// SendConfirmation sends the supplied log confirmation to all registered confirmation channels.
func (l *LogStreamer) sendConfirmation(confirmation *LogConfirmation) {
	// Make a copy of the list of confirmation channels so the lock can be released before confirmation delivery
	l.confirmationChannelsMutex.Lock()
	chanList := make([]chan LogConfirmation, len(l.confirmationChannels))
	for i, nextChan := range l.confirmationChannels {
		chanList[i] = nextChan
	}
	l.confirmationChannelsMutex.Unlock()

	// Ensure confirmations are delivered in order by not returning until the confirmation is queued.
	// These channels should be buffered so the sends should not block.
	l.log.Tracef("Sending confirmation to %d subscribed channels: %v", len(chanList), confirmation)
	for _, nextChan := range chanList {
		nextChan <- *confirmation
	}
}
