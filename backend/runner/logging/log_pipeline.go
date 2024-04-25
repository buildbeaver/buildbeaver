package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// LogPipelineFactory creates and starts a logs pipeline for a logs.
type LogPipelineFactory func(ctx context.Context, clk clock.Clock, secrets []*models.SecretPlaintext, logDescriptorID models.LogDescriptorID) (LogPipeline, error)

// RunnerLogTempDirectory is a value specifying the local directory in which logs are buffered by the runner
// in temporary files.
type RunnerLogTempDirectory string

type LogWriter interface {
	// Write a log entry to the pipeline.
	Write(entry *models.LogEntry)
	// Flush any previously written entries being buffered to the server. Temporary errors will be retried.
	Flush()
	// Close the pipeline, no longer accepting log entries for writing. Remaining entries being buffered
	// in the pipeline may be discarded.
	Close()
}

// NoOpLogWriter implements the LogWriter interface but takes no action.
type NoOpLogWriter struct {
}

func NewNoOpLogWriter() *NoOpLogWriter {
	return &NoOpLogWriter{}
}

func (w *NoOpLogWriter) Write(entry *models.LogEntry) {
}

func (w *NoOpLogWriter) Flush() {
}

func (w *NoOpLogWriter) Close() {
}

// closeRequester is a function that will request that the pipeline be closed.
// The function will return immediately without waiting for the Close to be complete, so it can be called
// without causing deadlocks.
type closeRequester func()

// LogConfirmation is a message indicating that logs have been confirmed as being fully processed/written
// up to and including the specified sequence number, or that there was a failure writing logs.
type LogConfirmation struct {
	// LastConfirmedSeqNo specifies the sequence number of the last log entry sent in the stream, if the
	// stream write was confirmed as succeeding. All log entries up to and including this sequence number are confirmed as
	// being written to persistent storage by the server. If err is not nil then LastConfirmedSeqNo will be zero.
	LastConfirmedSeqNo int
	// error if the stream write failed, or nil if it succeeded
	Err error
	// RetryFromSeqNo specifies the sequence number of the first log entry to re-sent, if the
	// stream write is returning an error. All log entries up to but not including this sequence number do not need
	// to be re-sent. If err is nil then RetryFromSeqNo will be zero.
	RetryFromSeqNo int
}

func (c *LogConfirmation) String() string {
	if c.Err != nil {
		return fmt.Sprintf("{Error confirmation: err: %v, RetryFromSeqNo: %d}", c.Err, c.RetryFromSeqNo)

	} else {
		return fmt.Sprintf("{Success confirmation: LastConfirmedSeqNo: %d}", c.LastConfirmedSeqNo)
	}
}

func NewSuccessConfirmation(lastConfirmedSeqNo int) *LogConfirmation {
	return &LogConfirmation{
		Err:                nil,
		LastConfirmedSeqNo: lastConfirmedSeqNo,
	}
}

func NewErrorConfirmation(err error, retryFromSeqNo int) *LogConfirmation {
	return &LogConfirmation{
		LastConfirmedSeqNo: 0,
		Err:                err,
		RetryFromSeqNo:     retryFromSeqNo,
	}
}

type LogPipeline interface {
	StructuredLogger() *StructuredLogger
	Converter() *LogConverter
	// Flush ensures all log entries have been written to the server. Temporary errors will be retried.
	// If a permanent error occurs then the remaining entries must be discarded.
	// Call Flush before calling Close.
	Flush()
	// Close closes the log pipeline, discarding any buffered log entries. To ensure no log entries are
	// discarded, call Flush() before calling Close().
	Close()
}

// NoOpLogPipeline implements the LogPipeline interface but takes no action.
type NoOpLogPipeline struct {
	clk        clock.Clock
	logFactory logger.LogFactory
	writer     LogWriter
}

func NewNoOpLogPipeline() *NoOpLogPipeline {
	return &NoOpLogPipeline{
		clk:        clock.New(), // use a normal basic clock
		logFactory: logger.NoOpLogFactory,
		writer:     NewNoOpLogWriter(),
	}
}

func (l *NoOpLogPipeline) StructuredLogger() *StructuredLogger {
	return NewStructuredLogger(l.clk, l.logFactory, l.writer)
}

func (l *NoOpLogPipeline) Converter() *LogConverter {
	converter := NewLogConverter(l.clk, l.logFactory, l.writer)
	converter.Start()
	return converter
}

func (l *NoOpLogPipeline) Flush() {
}

func (l *NoOpLogPipeline) Close() {
}

type ClientLogPipeline struct {
	clk        clock.Clock
	log        logger.Log
	logFactory logger.LogFactory
	writer     LogWriter
}

// NewClientLogPipeline creates a new ClientLogPipeline (implementing the LogPipeline interface) for processing
// log entries on a client and sending them to the server.
// If readChunkSize is zero then the default value will be used (recommended).
// If maxStreamSize is zero then the default value will be used (recommended).
// If maxStreamDuration is zero then the default value will be used (recommended).
func NewClientLogPipeline(
	ctx context.Context,
	clk clock.Clock,
	factory logger.LogFactory,
	client LogStreamFactory,
	id models.LogDescriptorID,
	secrets []*models.SecretPlaintext,
	logTempDir RunnerLogTempDirectory,
	readChunkSize int,
	maxStreamSize int,
	maxStreamDuration time.Duration,
) (*ClientLogPipeline, error) {
	l := &ClientLogPipeline{
		clk:        clk,
		log:        factory("LogPipeline"),
		logFactory: factory,
	}

	// Construct the pipeline stages in reverse order
	streamer := NewLogStreamer(ctx, factory, l.requestClose, client, id, maxStreamSize, maxStreamDuration)
	fileBuffer := NewLogFileBuffer(factory, l.requestClose, streamer, id, logTempDir, readChunkSize)
	sequencer := NewLogSequencer(factory, l.requestClose, fileBuffer)
	scrubber := NewLogScrubber(factory, l.requestClose, sequencer, secrets)

	l.writer = scrubber

	// Start the fileBuffer stage after hooking up its confirmation channel
	streamer.RegisterConfirmationChannel(fileBuffer.GetConfirmationChannel())
	err := fileBuffer.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting Log File Buffer pipeline stage: %w", err)
	}

	return l, nil
}

// Converter returns a LogConverter that is ready to be used (i.e. already started).
func (l *ClientLogPipeline) Converter() *LogConverter {
	converter := NewLogConverter(l.clk, l.logFactory, l.writer)
	converter.Start()
	return converter
}

func (l *ClientLogPipeline) StructuredLogger() *StructuredLogger {
	return NewStructuredLogger(l.clk, l.logFactory, l.writer)
}

// Flush ensures all log entries have been written to the server. Temporary errors will be retried.
// If a permanent error occurs then the remaining entries must be discarded.
// Call Flush before calling Close.
func (l *ClientLogPipeline) Flush() {
	l.writer.Flush()
}

// Close closes the log pipeline, discarding any buffered log entries. To ensure no log entries are
// discarded, call Flush() before calling Close().
func (l *ClientLogPipeline) Close() {
	l.writer.Close()
}

// requestClose closes the log pipeline on a goroutine, returning immediately.
func (l *ClientLogPipeline) requestClose() {
	go l.writer.Close()
}
