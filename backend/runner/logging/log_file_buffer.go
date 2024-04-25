package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

type logFileBufferStatus int

const (
	bufferNotStarted logFileBufferStatus = iota
	bufferStarting
	bufferRunning
	bufferShuttingDown
	bufferShutDown
)

// LogFileBuffer adds a file-based buffer into the pipeline. Log entries are written to a local file on disk,
// in case the logs need to be re-sent to the server, then read back from the file again to send on to the next stage.
type LogFileBuffer struct {
	mu               sync.Mutex // mutex covers file write operations and all state
	log              logger.Log
	closePipeline    closeRequester
	next             LogWriter
	logID            models.LogDescriptorID
	logTempDir       RunnerLogTempDirectory
	fileBufferReader *LogFileBufferReader
	fileBufferIndex  *LogFileBufferIndex // keeps information about the contents of the file
	state            struct {
		status              logFileBufferStatus
		file                *os.File
		filename            string
		openingTokenWritten bool
		closingTokenWritten bool
		bytesWritten        int64 // the number of bytes written to the file (i.e. the current file length)
		isShuttingDown      bool  // true if we are currently shutting down the stage (including flushing)
		isShutDown          bool
	}
}

// NewLogFileBuffer creates a new LogFileBuffer stage.
// If readChunkSize is zero then the default value will be used (recommended).
func NewLogFileBuffer(
	logFactory logger.LogFactory,
	closePipeline closeRequester,
	next LogWriter,
	logID models.LogDescriptorID,
	logTempDir RunnerLogTempDirectory,
	readChunkSize int,
) *LogFileBuffer {
	l := &LogFileBuffer{
		log:              logFactory("LogFileBuffer"),
		closePipeline:    closePipeline,
		next:             next,
		logID:            logID,
		logTempDir:       logTempDir,
		fileBufferReader: NewLogFileBufferReader(logFactory, next, readChunkSize),
		fileBufferIndex:  NewLogFileBufferIndex(),
	}
	l.state.status = bufferNotStarted
	return l
}

func (l *LogFileBuffer) GetConfirmationChannel() chan LogConfirmation {
	return l.fileBufferReader.GetConfirmationChannel()
}

func (l *LogFileBuffer) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state.status != bufferNotStarted {
		return fmt.Errorf("error: LogFileBuffer pipeline stage already started")
	}
	l.state.status = bufferStarting

	// Open the log file - this must succeed in order for this pipeline stage to operate
	err := l.ensureFileIsOpen()
	if err != nil {
		return err
	}

	// Start reading from the log file
	l.fileBufferReader.Start(l.state.file, l.fileBufferIndex)

	l.state.status = bufferRunning
	return nil
}

// Write a new entry to the log file.
func (l *LogFileBuffer) Write(entry *models.LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state.status != bufferRunning {
		l.log.Errorf("Error writing log entry; log file buffer stage is not running (current status=%d). Log entry discarded", l.state.status)
		return
	}

	err := l.write(entry)
	if err != nil {
		l.log.Errorf("Error writing log entry to local file; closing log pipeline: %v", err)
		l.closePipeline()
		return
	}

	// Do not send the log entry directly on to the next stage; the file reading goroutine will do this
}

func (l *LogFileBuffer) Flush() {
	// Lock the mutex while we initiate the flush; manually unlock again part way through this function
	l.mu.Lock()

	if l.state.status != bufferRunning {
		l.log.Warnf("Ignoring call to Flush() while buffer stage is not running (current status=%d)", l.state.status)
		l.mu.Unlock()
		return
	}

	l.log.Tracef("Flush(): flushing log entries from log file down the pipeline")
	flushCompletedChan := l.fileBufferReader.SendFlushRequest()

	// Don't need to hold a lock on the mutex while flushing; once the flush request has been sent it is guaranteed
	// to be processed by fileBufferReader even if the state machine is stopped before the flush is actually started.
	// This means we can handle concurrent calls to Write() and even Close() while flushing
	l.mu.Unlock()

	<-flushCompletedChan // wait until flush is complete

	// Do not call Flush() directly on to the next stage; the file reading goroutine will do this
	l.log.Tracef("Flush(): flush operation completed for stage; calling Flush() on the next pipeline stage")
}

// Close will shut down this and all later stages in the log pipeline, discarding any buffered log entries.
// To ensure no log entries are discarded, call Flush() before calling Close().
func (l *LogFileBuffer) Close() {
	// Lock the mutex while we initiate the shutdown process; manually unlock again part way through this function
	l.mu.Lock()

	if l.state.status != bufferRunning {
		l.log.Warnf("Ignoring call to Close() while buffer stage is not running (current status=%d)", l.state.status)
		l.mu.Unlock()
		return
	}
	// Set the current status to prevent further calls to Write(), Flush() or Close()
	l.state.status = bufferShuttingDown

	l.log.Tracef("Close(); writing closing entry to log file")
	err := l.finishWriting()
	if err != nil {
		l.log.Errorf("error finishing writing to local log file: %w", err)
	}

	// Temporarily unlock the mutex while we stop the reader.
	// This could take a while and releasing the lock allows this stage to deny further calls on other goroutines;
	// calls to Write(), Flush() and Close() will be rejected because status is bufferShuttingDown.
	l.mu.Unlock()
	l.log.Tracef("Close(): stopping the local file reader")
	l.fileBufferReader.Stop() // this waits synchronously
	l.mu.Lock()

	// After closing down the file reader it is now safe to close and delete the file
	l.log.Tracef("Close(): closing and deleting the buffer file")
	l.closeAndRemoveBufferFile()

	// This stage is now shut down, so release the lock before shutting down the following stage
	l.state.status = bufferShutDown
	l.mu.Unlock()

	l.log.Tracef("Close(): closing next pipeline stage")
	l.next.Close()
}

func (l *LogFileBuffer) closeAndRemoveBufferFile() {
	if l.state.file != nil {
		err := l.state.file.Close()
		if err != nil {
			l.log.Errorf("Ignoring error closing local log file: %v", err.Error())
		}
		err = os.Remove(l.state.filename)
		if err != nil {
			l.log.Errorf("Ignoring error deleting local log file: %v", err.Error())
		}
	}
	l.state.file = nil
	l.state.filename = ""
}

// ensureFileIsOpen will open a file to acts as a buffer for the log data if there isn't already a file open.
// If no error is returned then l.state.file will contain a file handle to the open file.
// The filename will be determined from the log descriptor ID supplied when this object was created.
// Any existing file with this filename will be opened, or otherwise a new file will be created.
// The file will be opened for reading and writing; writes will be append-only.
func (l *LogFileBuffer) ensureFileIsOpen() error {
	if l.state.file != nil {
		return nil // already open
	}

	// Create the directory if it doesn't exist
	err := os.MkdirAll(string(l.logTempDir), 0755) // read and traverse permissions for everyone
	if err != nil {
		return fmt.Errorf("error making runner log temp directory %s: %w", l.logTempDir, err)
	}

	// Open file for writing. Create the file if it doesn't exist. Always append only to the end of the file.
	filename := l.localLogFileName(l.logID)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error opening local log file: %w", err)
	}

	l.state.file = file
	l.state.filename = filename
	l.log.Tracef("Opened local log file '%s'", filename)
	return nil
}

// localLogFileName returns the file name (with full path) for the local log file for the specified
// log descriptor.
func (l *LogFileBuffer) localLogFileName(logID models.LogDescriptorID) string {
	return filepath.Join(string(l.logTempDir), models.SanitizeFilePathID(logID.ResourceID))
}

func (l *LogFileBuffer) write(entry *models.LogEntry) error {
	buf, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshalling entry to JSON: %w", err)
	}

	if !l.state.openingTokenWritten {
		buf = append([]byte{'['}, buf...)
		l.state.openingTokenWritten = true
	} else {
		buf = append([]byte{','}, buf...)
	}

	// Record the file offset for the start of this log entry against the sequence number (if the entry has one)
	startFileOffset := l.state.bytesWritten
	if persistentEntry, ok := entry.Derived().(models.PersistentLogEntry); ok {
		l.fileBufferIndex.AddLogEntry(persistentEntry.GetSeqNo(), startFileOffset)
	}

	l.log.Tracef("Writing log entry %v", *entry)
	n, err := l.state.file.Write(buf)
	l.state.bytesWritten += int64(n)
	if err != nil {
		return fmt.Errorf("error writing entry: %w", err)
	}

	// Tell the reader we have created a new log entry
	l.fileBufferReader.SendNewLogEntryHint()
	return nil
}

func (l *LogFileBuffer) finishWriting() error {
	if l.state.file != nil && l.state.openingTokenWritten && !l.state.closingTokenWritten {
		n, err := l.state.file.Write([]byte{']'})
		l.state.bytesWritten += int64(n)
		if err != nil {
			return fmt.Errorf("error writing closing ']' to local log file: %w", err)
		}
		l.state.closingTokenWritten = true
	}
	// Flush the file to disk
	err := l.state.file.Sync()
	if err != nil {
		return fmt.Errorf("error flushing local log file to disk: %w", err)
	}
	return nil
}
