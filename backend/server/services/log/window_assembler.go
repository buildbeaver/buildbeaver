package log

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
)

var logChunkKeyFormatRegex = regexp.MustCompile(`^logs/([a-z0-9:-]+)/([a-z0-9:-]+)/([0-9]+)-([0-9]+)-([a-z0-9:-]+)\.json$`)

type chunkDescriptor struct {
	*models.BlobDescriptor
	resourceID models.ResourceID
	logName    models.ResourceName
	// startSeqNo is the first seqNo number covered by the chunk (inclusive)
	startSeqNo int
	// endSeqNo is the last seqNo number covered by the chunk (exclusive)
	endSeqNo  int
	sessionID string
}

// chunkWindow wraps a set of logs chunks to provide a contiguous stream of log entries over a range of sequence numbers.
type chunkWindow struct {
	// desc is the logs the chunk window is for
	desc *models.LogDescriptor
	// chunks is the ordered set of chunks that covers range startSeqNo - endSeqNo
	chunks []*chunkDescriptor
	// startSeqNo is the first seqNo number covered by the window (inclusive)
	startSeqNo int
	// endSeqNo is the last seqNo number covered by the window (exclusive)
	endSeqNo int
}

// windowAssembler provides a contiguous stream of deduplicated log entries over a set of window readers.
type windowAssembler struct {
	ctx        context.Context
	log        logger.Log
	logFactory logger.LogFactory
	blobStore  services.BlobStore
	desc       *models.LogDescriptor
	startSeqNo *int
	state      struct {
		// pagination tracks our location in the chunk blob listing
		pagination *models.Pagination
		endOfStore bool
		// windows is the ordered set of windows to read from next
		windows []*chunkWindow
		// reader is a windowReader over the current window we're draining
		reader *windowReader
		err    error
	}
}

func newWindowAssembler(ctx context.Context, logFactory logger.LogFactory, blobStore services.BlobStore, desc *models.LogDescriptor, startSeqNo *int) *windowAssembler {
	a := &windowAssembler{
		ctx:        ctx,
		log:        logFactory("LogWindowAssembler"),
		logFactory: logFactory,
		blobStore:  blobStore,
		desc:       desc,
		startSeqNo: startSeqNo,
	}
	a.state.pagination = &models.Pagination{Limit: 1000}
	return a
}

// Next returns the next entry from the logs, or nil if all entries have been read.
func (l *windowAssembler) Next() (*nextEntry, error) {
	if l.state.err != nil {
		return nil, l.state.err
	}
	for {
		if l.state.reader == nil {
			if len(l.state.windows) > 0 {
				// Move onto the next window if we have any
				nextWindow := l.state.windows[0]
				l.state.windows = l.state.windows[1:]
				l.state.reader = newWindowReader(l.ctx, l.logFactory, l.blobStore, nextWindow, l.startSeqNo)
				l.log.Debugf("Moved to next chunk window (%d-%d)", nextWindow.startSeqNo, nextWindow.endSeqNo)
			} else {
				var (
					blobs  []*models.BlobDescriptor
					cursor *models.Cursor
					err    error
				)
				if !l.state.endOfStore {
					// Otherwise, produce more windows if we can
					prefix := l.makeChunkListPrefix()
					marker := l.makeChunkListMarker()
					l.log.Debugf("Reading next chunk list: prefix=%s marker=%s", prefix, marker)
					blobs, cursor, err = l.blobStore.ListBlobs(l.ctx, prefix, marker, *l.state.pagination)
					if err != nil {
						l.state.err = fmt.Errorf("error listing chunks: %w", err)
						return nil, err
					}
					l.log.Debugf("Read %d new chunk(s)", len(blobs))
				}
				if len(blobs) == 0 {
					// No windows left to drain, and no more chunks in the store. We're all done.
					l.log.Debug("No chunks left. All done.")
					return nil, nil
				}
				if cursor == nil {
					l.state.endOfStore = true
				} else {
					l.state.pagination.Cursor = cursor.Next
				}
				l.state.windows = l.mergeIntoWindows(blobs)
				l.log.Debugf("Merged %d chunk(s) into %d window(s)", len(blobs), len(l.state.windows))
				continue
			}
		}
		// If we have a current window then read from it
		l.log.Debug("Reading from window")
		next, err := l.state.reader.Next()
		if err != nil {
			l.state.err = err
			return nil, err
		}
		if next == nil {
			l.log.Debug("Exhausted current window")
			err := l.state.reader.Close()
			if err != nil {
				l.log.Errorf("Ignoring error closing reader: %v", err)
			}
			l.state.reader = nil
			continue
		}
		persistent := next.entry.Derived().(models.PersistentLogEntry)
		l.log.Debugf("Read next entry (seq no %d)", persistent.GetSeqNo())
		return next, nil
	}
}

// LogID returns the ID of the logs the assembler is reading.
func (l *windowAssembler) LogID() models.LogDescriptorID {
	return l.desc.ID
}

// Close the assembler, freeing up all underlying resources.
func (l *windowAssembler) Close() error {
	if l.state.reader != nil {
		return l.state.reader.Close()
	}
	return nil
}

func (l *windowAssembler) mergeIntoWindows(chunkBlobs []*models.BlobDescriptor) []*chunkWindow {
	var (
		windows []*chunkWindow
		chunks  = l.parseChunkBlobs(chunkBlobs)
	)
	for _, chunk := range chunks {
		merged := false
		for _, window := range windows {
			// Chunk boundaries align with existing window boundaries
			if chunk.startSeqNo == window.endSeqNo {
				window.endSeqNo = chunk.endSeqNo
				window.chunks = append(window.chunks, chunk)
				merged = true
			}
			if chunk.endSeqNo == window.startSeqNo {
				window.startSeqNo = chunk.startSeqNo
				window.chunks = append(window.chunks, chunk)
				merged = true
			}
			// Chunk boundaries overlap with existing window boundaries
			if chunk.startSeqNo <= window.startSeqNo && chunk.endSeqNo > window.startSeqNo && chunk.endSeqNo < window.endSeqNo {
				window.startSeqNo = chunk.startSeqNo
				window.chunks = append(window.chunks, chunk)
				merged = true
				break
			}
			if chunk.startSeqNo >= window.startSeqNo && chunk.startSeqNo < window.endSeqNo && chunk.endSeqNo > window.endSeqNo {
				window.endSeqNo = chunk.endSeqNo
				window.chunks = append(window.chunks, chunk)
				merged = true
				break
			}
			// Chunk boundaries subsume existing window boundaries
			if chunk.startSeqNo <= window.startSeqNo && chunk.endSeqNo >= window.endSeqNo {
				window.startSeqNo = chunk.startSeqNo
				window.endSeqNo = chunk.endSeqNo
				window.chunks = []*chunkDescriptor{chunk}
				merged = true
				break
			}
		}
		if !merged {
			windows = append(windows, &chunkWindow{
				desc:       l.desc,
				startSeqNo: chunk.startSeqNo,
				endSeqNo:   chunk.endSeqNo,
				chunks:     []*chunkDescriptor{chunk},
			})
		}
	}
	sort.SliceStable(windows, func(i, j int) bool {
		return windows[i].startSeqNo < windows[j].startSeqNo
	})
	for _, window := range windows {
		sort.SliceStable(window.chunks, func(i, j int) bool {
			return window.chunks[i].startSeqNo < window.chunks[j].startSeqNo
		})
	}
	return windows
}

func (l *windowAssembler) parseChunkBlobs(blobs []*models.BlobDescriptor) []*chunkDescriptor {
	var chunks []*chunkDescriptor
	for _, blob := range blobs {
		a, err := l.parseChunkBlob(blob)
		if err != nil {
			l.log.Errorf("Ignoring error parsing chunk blob: %v", err)
			continue
		}
		chunks = append(chunks, a)
	}
	return chunks
}

// parseChunkBlob parses a blob from the blob store into a chunk descriptor.
func (l *windowAssembler) parseChunkBlob(blob *models.BlobDescriptor) (*chunkDescriptor, error) {
	match := logChunkKeyFormatRegex.FindStringSubmatch(blob.Key)
	if match == nil {
		return nil, fmt.Errorf("error no match on logs chunk key regex")
	}
	var (
		resourceIDStr = match[1]
		logName       = match[2]
		endSeqNoStr   = match[3]
		startSeqNoStr = match[4]
		sessionID     = match[5]
	)
	resourceID, err := models.ParseResourceID(resourceIDStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing resource id: %w", err)
	}
	startSeqNo, err := strconv.ParseInt(startSeqNoStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing start seq no: %w", err)
	}
	endSeqNo, err := strconv.ParseInt(endSeqNoStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing end seq no: %w", err)
	}
	return &chunkDescriptor{
		BlobDescriptor: blob,
		resourceID:     resourceID,
		logName:        models.ResourceName(logName),
		startSeqNo:     int(startSeqNo),
		endSeqNo:       int(endSeqNo),
		sessionID:      sessionID,
	}, nil
}

// makeChunkListPrefix produces the most specific chunk blob key prefix possible given the current query.
func (l *windowAssembler) makeChunkListPrefix() string {
	return fmt.Sprintf(logChunkKeyBaseFormat, l.desc.ResourceID, l.desc.ID)
}

func (l *windowAssembler) makeChunkListMarker() string {
	if l.startSeqNo != nil {
		return fmt.Sprintf("%d-", *l.startSeqNo)
	}
	return ""
}
