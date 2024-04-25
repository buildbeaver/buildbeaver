package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type nextEntry struct {
	entry *models.LogEntry
	raw   json.RawMessage
}

// windowReader provides a contiguous stream of deduplicated log entries over a set of chunks from the blob store.
type windowReader struct {
	ctx        context.Context
	log        logger.Log
	blobStore  services.BlobStore
	startSeqNo *int
	state      struct {
		lastSeqNo  int
		chunks     []*chunkDescriptor
		blobReader io.ReadCloser
		decoder    *json.Decoder
		err        error
	}
}

func newWindowReader(ctx context.Context, log logger.LogFactory, blobStore services.BlobStore, window *chunkWindow, startSeqNo *int) *windowReader {
	r := &windowReader{
		ctx:        ctx,
		log:        log("LogWindowReader"),
		blobStore:  blobStore,
		startSeqNo: startSeqNo,
	}
	r.state.chunks = window.chunks
	return r
}

// Next returns the next entry from the logs window, or nil if all entries have been read.
func (l *windowReader) Next() (*nextEntry, error) {
	if l.state.err != nil {
		return nil, l.state.err
	}
	for {
		if l.state.decoder == nil {
			if len(l.state.chunks) == 0 {
				return nil, nil // all done
			}
			nextChunk := l.state.chunks[0]
			l.state.chunks = l.state.chunks[1:]
			reader, err := l.blobStore.GetBlob(l.ctx, nextChunk.Key)
			if err != nil {
				l.state.err = fmt.Errorf("error reading data %q: %w", nextChunk.Key, err)
				return nil, l.state.err
			}
			l.state.decoder = json.NewDecoder(reader)
			_, err = l.state.decoder.Token()
			if err != nil {
				l.state.err = fmt.Errorf("error reading opening token: %w", err)
				return nil, l.state.err
			}
			l.state.blobReader = reader
			l.log.Debugf("Opened reader for chunk %s", nextChunk.Key)
		}
		if !l.state.decoder.More() {
			err := l.state.blobReader.Close()
			if err != nil {
				l.log.Errorf("Ignoring error closing reader: %v", err)
			}
			l.state.blobReader = nil
			l.state.decoder = nil
			continue
		}
		var (
			rawEntry json.RawMessage
			entry    = &models.LogEntry{}
		)
		err := l.state.decoder.Decode(&rawEntry)
		if err != nil {
			l.state.err = fmt.Errorf("error decoding JSON from blob stream: %w", err)
			return nil, l.state.err
		}
		l.log.Tracef("Read plaintext entry: %s", rawEntry[:])
		err = json.Unmarshal(rawEntry, entry)
		if err != nil {
			l.log.Errorf("Ignoring error unmarshalling entry; Entry will be skipped: %w", err)
			continue
		}
		persistent := entry.Derived().(models.PersistentLogEntry)
		if l.state.lastSeqNo >= persistent.GetSeqNo() {
			l.log.Debugf("Skipping seq no %d which comes before last seq no %d", persistent.GetSeqNo(), l.state.lastSeqNo)
		} else if l.startSeqNo != nil && *l.startSeqNo > persistent.GetSeqNo() {
			l.log.Debugf("Skipping seq no %d which comes before start seq no %d", persistent.GetSeqNo(), *l.startSeqNo)
		} else {
			l.state.lastSeqNo = persistent.GetSeqNo()
			next := &nextEntry{
				entry: entry,
				raw:   rawEntry,
			}
			return next, nil
		}
	}
}

// Close the reader and free up underlying resources.
func (l *windowReader) Close() error {
	if l.state.err != nil {
		l.state.err = fmt.Errorf("error reader is closed")
	}
	if l.state.blobReader != nil {
		return l.state.blobReader.Close()
	}
	return nil
}
