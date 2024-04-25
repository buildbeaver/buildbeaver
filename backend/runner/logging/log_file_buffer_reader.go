package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
)

// idleInterval specifies the time period between writes that will cause the pipeline to be considered idle.
// After this time the LogFileBufferReader will check the local log file for new entries, and also flush the
// next pipeline stage to close any existing log writing HTTP connections, preventing them from timing out.
const idleInterval = 5 * time.Second

// defaultReadChunkSize is the default number of bytes to read from the temporary log file at once
const defaultReadChunkSize = 4096

type flushRequest struct {
	// Channel that will be closed once flush is complete
	completedChan chan bool
}

func newFlushRequest() *flushRequest {
	return &flushRequest{
		completedChan: make(chan bool),
	}
}

// LogFileBufferReader is a stateful service that reads data from a file buffer and sends it down the pipeline.
type LogFileBufferReader struct {
	service             *util.StatefulService
	log                 logger.Log
	nextStage           LogWriter
	readChunkSize       int
	file                *os.File             // file to read from
	fileIndex           *LogFileBufferIndex  // keeps information about the contents of the file
	confirmationChannel chan LogConfirmation // channel to receive confirmation that log entries are fully processed
	newLogEntryChan     chan bool            // channel to send a hint that one or more new log entries have been written
	flushRequestChan    chan *flushRequest   // channel to request flushing all log entries from the file down the pipeline
	state               struct {
		currentReadOffset   int64         // byte position within the file to read the next log entry from
		finishedReadingFile bool          // true if finished reading log entries from the file (i.e. have seen the closing ']')
		lastSeqNoSent       int           // last sequence number sent down the pipeline
		lastSeqNoConfirmed  int           // last sequence number confirmed as processed by the server
		flushing            bool          // true if we are currently processing a flush request (including waiting for confirmations)
		currentFlushRequest *flushRequest // if currently flushing then this is the flush request to complete at the end
		flushToSeqNo        int           // the sequence number we are seeking confirmation for when flushing
	}
}

// NewLogFileBufferReader creates a new LogFileBufferReader to read log entries from a buffer file and send them
// on down the pipeline to nextStage.
// If readChunkSize is zero then the default value will be used (recommended).
func NewLogFileBufferReader(
	logFactory logger.LogFactory,
	nextStage LogWriter,
	readChunkSize int,
) *LogFileBufferReader {
	if readChunkSize == 0 {
		readChunkSize = defaultReadChunkSize
	}
	l := &LogFileBufferReader{
		log:                 logFactory("LogFileBufferReader"),
		nextStage:           nextStage,
		readChunkSize:       readChunkSize,
		confirmationChannel: make(chan LogConfirmation, 1000), // don't block the sender of confirmations
		newLogEntryChan:     make(chan bool, 1000),            // don't block the sender of hints
		flushRequestChan:    make(chan *flushRequest),
	}

	// Create the service but don't start it the file to read is available
	l.service = util.NewStatefulService(context.Background(), l.log, l.readLoop)

	return l
}

func (l *LogFileBufferReader) Start(file *os.File, fileIndex *LogFileBufferIndex) {
	l.file = file
	l.fileIndex = fileIndex
	l.service.Start()
}

func (l *LogFileBufferReader) Stop() {
	l.service.Stop()
}

func (l *LogFileBufferReader) GetConfirmationChannel() chan LogConfirmation {
	return l.confirmationChannel
}

// SendNewLogEntryHint sends a hint to the reader that new log entries have been written, and that it should
// check the file for new entries. Returns immediately rather than waiting for the hint to be processed.
func (l *LogFileBufferReader) SendNewLogEntryHint() {
	// send the hint down a buffered channel
	l.newLogEntryChan <- true
}

// SendFlushRequest ensures that all log entries written to the local log file before this function is called
// have been read and sent down the pipeline to the server, and confirmed as being received.
// Returns immediately, with a channel that will be closed once the flush is completed.
// New log entries written to the buffer file after the main event loop processes the flush request will not
// be included in the flush, but all previously written log entries will be flushed.
// If this function returns before Stop() is called then the returned channel is guaranteed to be closed.
func (l *LogFileBufferReader) SendFlushRequest() chan bool {
	// Tell the reader to check for and process any remaining log entries
	req := newFlushRequest()
	l.flushRequestChan <- req

	return req.completedChan
}

// readLoop will monitor the local log file and send any log entries down the pipeline.
// This function should be run in its own goroutine (as a service).
func (l *LogFileBufferReader) readLoop() {
	l.log.Tracef("Starting local log file reader loop...")
	for {
		// Only listen for flush requests when a flush isn't already underway
		var flushReqChan chan *flushRequest
		if !l.state.flushing {
			flushReqChan = l.flushRequestChan
		}

		select {
		case <-l.service.Ctx().Done():
			// The contract is to respond to all flush requests
			l.completeCurrentFlush()
			l.drainFlushRequestChan()
			l.drainNewLogEntryChan()
			l.log.Tracef("Service closed; exiting local log file reader loop...")
			return

		case <-l.newLogEntryChan: // this channel gives a hint that there are new log entries
			l.drainNewLogEntryChan() // elide multiple hints into one
			l.readAndSendLogEntries()

		case flushRequest := <-flushReqChan:
			l.log.Tracef("Flush request received; flushing log entries from local log file")
			// Send remaining log entries from the file
			l.readAndSendLogEntries()
			// Go into 'flushing' state while we wait for confirmations back from the streamer stage. Do this
			// AFTER we read everything from the file, so that flushToSeqNo includes everything in the file, and so
			// we don't return from the flush before getting confirmations from the log entries at end of the file.
			l.state.flushing = true
			l.state.currentFlushRequest = flushRequest
			l.state.flushToSeqNo = l.state.lastSeqNoSent
			// Tell the next stage to flush; this will close the connection and send a confirmation back
			l.nextStage.Flush()
			// Check to see if we can complete the flush immediately (before processing confirmations)
			l.checkAndCompleteFlush()

		case confirmation := <-l.confirmationChannel:
			l.log.Tracef("Received confirmation: %v", confirmation.String())
			if confirmation.Err != nil {
				l.rollBack(confirmation.RetryFromSeqNo)
				l.readAndSendLogEntries() // immediately send again from the rollback point
				if l.state.flushing {
					l.nextStage.Flush() // close connection and get confirmation back
				}
			} else {
				if confirmation.LastConfirmedSeqNo > l.state.lastSeqNoConfirmed {
					l.state.lastSeqNoConfirmed = confirmation.LastConfirmedSeqNo
				}
				l.checkAndCompleteFlush()
			}

		case <-time.After(idleInterval):
			// poll the file for new log entries, to recover from a restart or missed notification
			entriesSent := l.readAndSendLogEntries()
			if entriesSent > 0 {
				// Log that we had to rely on polling to process some log entries; this is not normal
				l.log.Tracef("Polling read of log file processed %d new entries", entriesSent)
			}
			// call flush on the next stage to close any existing connection to the server;
			// this avoids waiting until connections time out
			l.nextStage.Flush()
		}
	}
}

// drainNewLogEntryChan reads and discards all queued values from newLogEntryChan
func (l *LogFileBufferReader) drainNewLogEntryChan() {
	for {
		select {
		case <-l.newLogEntryChan:
			// do nothing; just discard the read value
		default:
			return // no more values in channel
		}
	}
}

// drainFlushRequestChan reads and responds to all queued values from flushRequestChan, although it does
// not actually perform a flush.
func (l *LogFileBufferReader) drainFlushRequestChan() {
	for {
		select {
		case flushRequest := <-l.flushRequestChan:
			close(flushRequest.completedChan) // say that flush has been completed immediately
		default:
			return // no more values in channel
		}
	}
}

// checkAndCompleteFlush checks to see if a flush is underway, and if all outstanding log entries being flushed
// have now been confirmed. If so, the flush is now complete and the flush completedChan will be closed.
func (l *LogFileBufferReader) checkAndCompleteFlush() {
	if !l.state.flushing {
		return // not flushing
	}
	if l.state.lastSeqNoConfirmed < l.state.flushToSeqNo {
		l.log.Tracef("Checking whether to end flush: not done yet; flushToSeqNo=%d, lastSeqNoConfirmed=%d",
			l.state.flushToSeqNo, l.state.lastSeqNoConfirmed)
		return // flush not finished yet
	}
	// all log entries now confirmed as being sent
	l.completeCurrentFlush()
}

// completeCurrentFlush will complete any currently running flush and close the completedChan, regardless of which
// confirmations have been received.
func (l *LogFileBufferReader) completeCurrentFlush() {
	if !l.state.flushing {
		return // not flushing
	}
	l.state.flushing = false
	l.state.flushToSeqNo = 0
	if l.state.currentFlushRequest != nil {
		l.log.Tracef("Sending 'flush request completed' notification")
		close(l.state.currentFlushRequest.completedChan)
		l.state.currentFlushRequest = nil
	} else {
		l.log.Errorf("Flushing=true and flush is done but no currentFlushRequest value set; unable to notify the caller that flush is complete")
	}
}

// readAndSendLogEntries checks the local file for new/unprocessed log entries, reads and sends them down the
// pipeline until there are none left, or until l.StatefulService.Ctx is cancelled.
// Errors when reading from the log file are logged and ignored, and so will be retried in the next call.
// Returns the number of log entries sent down the pipeline.
func (l *LogFileBufferReader) readAndSendLogEntries() int {
	if l.state.finishedReadingFile {
		l.log.Tracef("readAndSendLogEntries() called but finishedReadingFile is true; skipping")
		return 0
	}
	l.log.Tracef("readAndSendLogEntries() called...")

	entriesSent := 0
	moreEntries := true
	for moreEntries && l.service.Ctx().Err() == nil {
		l.log.Tracef("Calling readNextLogEntriesFromFile()")
		logEntries, isEndOfLog, err := l.readNextLogEntriesFromFile()
		if err != nil {
			l.log.Errorf("Error reading log entry from local file: %v", err)
			return entriesSent
		}
		if len(logEntries) > 0 {
			for _, logEntry := range logEntries {
				l.log.Tracef("Writing log entry read from local file: %v", logEntry)
				l.nextStage.Write(logEntry)
				entriesSent++
				// Track the last sequence number sent down the pipeline, to match to confirmations
				persistent, ok := logEntry.Derived().(models.PersistentLogEntry)
				if ok {
					l.state.lastSeqNoSent = persistent.GetSeqNo()
				}
			}
		} else {
			moreEntries = false
		}
		if isEndOfLog {
			l.state.finishedReadingFile = true
			moreEntries = false
		}
	}

	l.log.Tracef("readAndSendLogEntries() processed %d entries", entriesSent)
	return entriesSent
}

// readNextLogEntriesFromFile reads and returns the next chunk of log entries from the local file, from the offset
// specified in l.state.currentReadOffset. The offset is updated to point to after the returned log entries,
// ready for the next read.
// If no new log entries are currently available in the file then nil is returned (not an error).
// If the end-of-log marker (the JSON closing ']' for the array of log entries) is next in the file then endOfLog
// will be returned as true.
func (l *LogFileBufferReader) readNextLogEntriesFromFile() (logEntries []*models.LogEntry, endOfLog bool, err error) {
	var (
		buffer         []byte
		results        []*models.LogEntry
		bytesProcessed int64
		isEndOfFile    bool
		isEndOfLog     bool // true if we have seen the JSON closing ']' so there will be no more log entries
	)

	l.log.Tracef("readNextLogEntriesFromFile() reading data from file")

	// Keep reading more chunks of data and adding to the buffer until we get at least one log entry, or we hit
	// the end of the file or end of the log data.
	// This ensures we can read log entries that are larger than the chunk size.
	for len(results) == 0 && !isEndOfFile && !isEndOfLog {
		nextChunk := make([]byte, l.readChunkSize)
		nextChunkOffset := l.state.currentReadOffset + int64(len(buffer))
		l.log.Tracef("Reading a chunk of up to %d bytes from file offset %d", l.readChunkSize, nextChunkOffset)
		numBytesRead, err := l.file.ReadAt(nextChunk, nextChunkOffset)
		l.log.Tracef("Got a chunk of %d bytes from file offset %d", numBytesRead, nextChunkOffset)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Don't treat EOF any differently, just process the data that was read. More data may be appended to the file later.
				err = nil
				isEndOfFile = true
			} else {
				return nil, false, err // this is a real error
			}
		}
		if numBytesRead > 0 {
			// Add chunk to buffer
			isFirstChunk := len(buffer) == 0
			buffer = append(buffer, nextChunk[0:numBytesRead]...)

			// Make the buffer into the beginning of valid JSON document containing a list of at least one log entry
			if isFirstChunk {
				l.log.Tracef("Checking first chunk character")
				switch buffer[0] {
				case '[':
					if l.state.currentReadOffset != 0 {
						return nil, false, fmt.Errorf("error: unexpected array separator read from local file at position %d", l.state.currentReadOffset)
					}
					// Buffer data is taken from the start of the file, so it's already a valid JSON document
				case ']':
					// Buffer is from the very end of the log data, so there are no more log entries.
					// Do not move the current offset past the end of log marker, so that we will see it
					// again if readNextLogEntriesFromFile() is called again.
					isEndOfLog = true
					l.log.Tracef("Seen end-of-log ']' character")
				case ',':
					// Buffer is from the middle of the file, so substitute '[' instead of ',' to form the
					// start of a new document starting part way through the file
					buffer[0] = '['
				case '{':
					if l.state.currentReadOffset == 0 {
						return nil, false, fmt.Errorf("error: expected first token in the local log file to begin an array of log entries")
					}
					// Buffer is at the start of a log entry rather than a separator, so prepend '[' to form the start
					// of a new document starting part way through the file, and move the file read offset back one
					// byte since readLogEntriesFromBuffer() will now report one extra byte processed
					buffer = append([]byte("["), buffer...)
					l.state.currentReadOffset-- // current offset is at least 1
				}
			}

			if !isEndOfLog {
				// Attempt to read some log entries from the buffer data we've accumulated
				l.log.Tracef("calling readLogEntriesFromBuffer for %d byte buffer", len(buffer))
				results, bytesProcessed, err = l.readLogEntriesFromBuffer(buffer)
				if err != nil {
					return nil, false, err
				}
			}
		}
	}
	l.state.currentReadOffset += bytesProcessed
	return results, isEndOfLog, nil
}

// readLogEntriesFromBuffer reads and returns as many log entries as it can from the supplied byte array buffer.
// The buffer data must form a valid JSON document (i.e. start with '[', forming an array of log entries) but
// can be truncated at any arbitrary point, including part way through a log entry.
// If no full log entries are currently available in the buffer then nil is returned (not an error).
func (l *LogFileBufferReader) readLogEntriesFromBuffer(buffer []byte) (logEntries []*models.LogEntry, bytesProcessed int64, err error) {
	logEntries = nil
	bytesProcessed = 0
	reader := bytes.NewReader(buffer)
	decoder := json.NewDecoder(reader)

	// Skip over the first opening '[' in the JSON
	token, err := decoder.Token()
	if err != nil {
		return nil, 0, fmt.Errorf("error reading opening token: %w", err)
	}
	if token != json.Delim('[') {
		return nil, 0, fmt.Errorf("error expected first token to begin array (\"[\"), found: %s", token)
	}

	for decoder.More() {
		l.log.Tracef("Attempting to read next entry")
		entry := &models.LogEntry{}
		err = decoder.Decode(entry)
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				l.log.Tracef("Partial log entry or empty log entry found at end of buffer; will be processed next time")
				break // at the end of the buffer
			} else {
				return nil, 0, fmt.Errorf("error unmarshalling entry from JSON: %w", err)
			}
		}
		// Change the number of bytes processed each time we read an entire log entry
		bytesProcessed = decoder.InputOffset()
		logEntries = append(logEntries, entry)

		// Return both persistent and non-persistent entries
		persistent, ok := entry.Derived().(models.PersistentLogEntry)
		if ok {
			l.log.Tracef("Read persistent log entry (seq no %d), processed %d bytes", persistent.GetSeqNo(), bytesProcessed)
		} else {
			l.log.Tracef("Read non-persistent log entry, processed %d bytes", bytesProcessed)
		}
	}

	l.log.Tracef("Read %d entries from buffer, %d bytes processed", len(logEntries), bytesProcessed)
	return logEntries, bytesProcessed, nil
}

// rollBack will roll back the currently sent item in the file to the requested sequence number, and re-send
// all log entries from that sequence number onwards. This can be used to re-send everything in the current stream
// of log entries if the stream has a problem, ensuring we don't miss log entries.
func (l *LogFileBufferReader) rollBack(retryFromSeqNo int) {
	var (
		retryFromFileOffset int64 = 0
		err                 error
	)

	if retryFromSeqNo != 0 {
		retryFromFileOffset, err = l.fileIndex.GetStartOffset(retryFromSeqNo)
		if err != nil {
			l.log.Errorf("error rolling back to sequence number %d: no file file offset found for sequence number in index: %v", retryFromSeqNo)
			return
		}
	}

	// Roll the read offset back so all following log entries will be re-sent down the pipeline
	l.log.Warnf("Error confirmation received: rolling log back to file offset %d", retryFromFileOffset)
	l.state.currentReadOffset = retryFromFileOffset
	l.state.finishedReadingFile = false
}
