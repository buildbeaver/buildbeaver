package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/google/uuid"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/services"
)

const (
	logChunkKeyBaseFormat = "logs/%s/%s/"
	logChunkKeyFullFormat = logChunkKeyBaseFormat + "%d-%d-%s.json"
)

var DefaultWriterConfig = WriterConfig{
	ChunkSizeBytes:    1 * 1024 * 1024,
	ChunkTTL:          time.Second,
	ChunkWriteTimeout: 10 * time.Second,
}

type encodedEntry struct {
	seqNo int
	data  json.RawMessage
}

type WriterConfig struct {
	// hunkSizeBytes is the target size of a chunk of the logs that we will attempt to fill before persisting it.
	ChunkSizeBytes int64
	// ChunkTTL determines how long we attempt to fill a chunk for before giving up and flushing a smaller amount.
	ChunkTTL time.Duration
	// ChunkWriteTimeout is the maximum amount of time to wait to flush a chunk.
	ChunkWriteTimeout time.Duration
}

type writerFlushRequest struct {
	completedChan chan error // returns an error or nil once flush is completed
}

func newWriterFlushRequest() *writerFlushRequest {
	return &writerFlushRequest{
		completedChan: make(chan error),
	}
}

// writer buffers and writes chunks of log entries to blob storage for a single log.
// It is safe to operate multiple writers over the same log concurrently.
type writer struct {
	*util.StatefulService
	clk         clock.Clock
	log         logger.Log
	config      WriterConfig
	blobStore   services.BlobStore
	descriptor  *models.LogDescriptor
	sessionID   string
	entryInChan chan encodedEntry
	flushChan   chan *writerFlushRequest
	state       struct {
		entries              []json.RawMessage
		size                 int64
		startSeqNo, endSeqNo int
		writeErr             error // after a write error we stop writing and return errors from there on
	}
}

// newWriter creates a new writer service to buffer and write chunks of log entries to blob storage.
// Call Start() on the writer before using, and Stop() once finished.
func newWriter(logFactory logger.LogFactory, clk clock.Clock, config WriterConfig, blobStore services.BlobStore, descriptor *models.LogDescriptor) *writer {
	w := &writer{
		clk:         clk,
		log:         logFactory("LogWriter"),
		config:      config,
		descriptor:  descriptor,
		blobStore:   blobStore,
		sessionID:   uuid.New().String(),
		entryInChan: make(chan encodedEntry),
		flushChan:   make(chan *writerFlushRequest),
	}
	w.state.endSeqNo = 1
	w.state.entries = make([]json.RawMessage, 0, 300) // Random guess for cap
	w.StatefulService = util.NewStatefulService(context.Background(), w.log, w.loop)
	return w
}

func (l *writer) drain(ctx context.Context, reader io.Reader) error {
	dec := json.NewDecoder(reader)
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("error reading opening token: %w", err)
	}
	if token != json.Delim('[') {
		return fmt.Errorf("error expected first token to begin array (\"[\"), found: %s", token)
	}
	for dec.More() {
		l.log.Debug("Reading next entry")
		entry := &models.LogEntry{}
		err := dec.Decode(entry)
		if err != nil {
			return fmt.Errorf("error unmarshalling entry from JSON: %w", err)
		}
		persistent, ok := entry.Derived().(models.PersistentLogEntry)
		if !ok {
			// Return a 400-series error in this situation, since the client is sending entries this server doesn't understand
			return gerror.NewErrValidationFailed("error reading log entries: expected to see only persistent log entries")
		}
		l.log.Debugf("Read entry: %d", persistent.GetSeqNo())
		persistent.SetServerTimestamp(models.NewTime(l.clk.Now()))
		data, err := json.Marshal(persistent)
		if err != nil {
			return fmt.Errorf("error marshalling entry to JSON: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case l.entryInChan <- encodedEntry{data: data, seqNo: persistent.GetSeqNo()}:
		}
	}
	_, err = dec.Token()
	if err != nil {
		return fmt.Errorf("error reading closing token: %w", err)
	}

	// Flush all entries; this returns the first write error that happened during this or any previous flush
	err = l.requestFlushAndWait()
	if err != nil {
		return fmt.Errorf("error flushing/writing log entries: %w", err)
	}

	return nil
}

// requestFlushAndWait sends a flush request to the writer loop and waits for it to complete.
// The error value sent back from the flush will be returned.
func (l *writer) requestFlushAndWait() error {
	flushReq := newWriterFlushRequest()
	l.flushChan <- flushReq
	return <-flushReq.completedChan
}

func (l *writer) loop() {
	for {
		select {
		case <-l.Ctx().Done():
			if len(l.state.entries) > 0 && l.state.writeErr == nil {
				l.state.writeErr = l.flush()
				if l.state.writeErr != nil {
					l.log.Errorf("Ignoring error flushing logs on shutdown: %s", l.state.writeErr.Error())
				}
			}
			return

		case entry := <-l.entryInChan:
			if l.state.writeErr != nil {
				l.log.Debugf("Discarding log entry after write error")
				continue
			}
			l.state.writeErr = l.writeEntry(entry)
			if l.state.writeErr != nil {
				l.log.Errorf("Recording error writing to logs: %w", l.state.writeErr)
			}

		case flushReq := <-l.flushChan:
			if l.state.writeErr != nil {
				flushReq.completedChan <- l.state.writeErr // return any previous write error
				continue
			}
			if len(l.state.entries) > 0 {
				l.state.writeErr = l.flush()
			}
			if l.state.writeErr != nil {
				l.log.Errorf("Recording error flushing logs: %v", l.state.writeErr.Error())
			}
			flushReq.completedChan <- l.state.writeErr

		case <-l.clk.After(l.config.ChunkTTL):
			if len(l.state.entries) > 0 && l.state.writeErr == nil {
				l.state.writeErr = l.flush()
				if l.state.writeErr != nil {
					l.log.Errorf("Recording error flushing logs after TTL expiry: %w", l.state.writeErr)
				}
			}
		}
	}
}

func (l *writer) writeEntry(entry encodedEntry) error {
	l.state.entries = append(l.state.entries, entry.data)
	l.state.size += int64(len(entry.data))
	if l.state.startSeqNo <= 0 {
		l.state.startSeqNo = entry.seqNo
	}
	l.state.endSeqNo = entry.seqNo + 1
	if l.state.size >= l.config.ChunkSizeBytes {
		return l.flush()
	}
	return nil
}

func (l *writer) flush() error {
	defer func() {
		l.state.entries = l.state.entries[:0]
		l.state.size = 0
		l.state.startSeqNo = 0
		l.state.endSeqNo = 0
	}()
	ctx, cancel := context.WithTimeout(l.Ctx(), l.config.ChunkWriteTimeout)
	defer cancel()

	done := make(chan error)
	r, w := io.Pipe()
	go func() {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		err := enc.Encode(l.state.entries)
		w.Close()
		done <- err
	}()

	err := l.blobStore.PutBlob(ctx, l.makeChunkKey(l.state.startSeqNo, l.state.endSeqNo), r)
	encodeErr := <-done // always read from encoder done chan, even on error, to avoid blocking the goroutine forever
	if err != nil {
		return fmt.Errorf("error writing logs chunk: %w", err)
	}
	if encodeErr != nil {
		return fmt.Errorf("error encoding logs chunk: %w", encodeErr)
	}
	return nil
}

func (l *writer) makeChunkKey(startsAt int, endsAt int) string {
	return fmt.Sprintf(logChunkKeyFullFormat, l.descriptor.ResourceID, l.descriptor.ID, endsAt, startsAt, l.sessionID)
}
