package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type query struct {
	descriptors []*models.LogDescriptor
	// startSeqNo determines which entry the reader will start from (inclusive).
	// Only supported when reading from a single logs.
	startSeqNo *int
	// plaintext can be set to true to get a plaintext stream from the reader.
	plaintext bool
}

// reader is an io.Reader that multiplexes one or more log descriptors into a single contiguous stream.
// The stream can be configured to emit structured JSON log entries, or newline-delimited plaintext.
type reader struct {
	ctx   context.Context
	log   logger.Log
	query *query
	// assemblers we're multiplexing logs from (by timestamp, the oldest first)
	assemblers []*windowAssembler
	state      struct {
		// nextEntryByLog tracks the next available entry from each log
		nextEntryByLog map[models.LogDescriptorID]*nextEntry
		// currentEntry being drained by successive calls to Read()
		currentEntry []byte
		// arrayOpened is true if we've written the initial opening JSON array token
		arrayOpened bool
		// firstEntryWritten is true if we've written at least one entry
		firstEntryWritten bool
		// endEntryWritten is true when we've written the log_end stream terminator
		endEntryWritten bool
		// arrayOpened is true if we've written the final closing JSON array token
		arrayClosed bool
	}
}

func newReader(ctx context.Context, logFactory logger.LogFactory, blobStore services.BlobStore, query *query) *reader {
	if query.startSeqNo != nil && len(query.descriptors) > 1 {
		panic("startSeqNo is not supported when merging multiple logs")
	}
	var assemblers []*windowAssembler
	for _, descriptor := range query.descriptors {
		assemblers = append(assemblers, newWindowAssembler(ctx, logFactory, blobStore, descriptor, query.startSeqNo))
	}
	// We sort these to ensure we always merge log entries with identical server timestamps in a deterministic order.
	sort.Slice(assemblers, func(i, j int) bool {
		return assemblers[i].LogID().String() < assemblers[j].LogID().String()
	})
	r := &reader{
		ctx:        ctx,
		log:        logFactory("LogReader"),
		query:      query,
		assemblers: assemblers,
	}
	r.state.nextEntryByLog = make(map[models.LogDescriptorID]*nextEntry)
	return r
}

func (l *reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Write the opening JSON array token
	if !l.query.plaintext && !l.state.arrayOpened {
		p[n] = '['
		n++
		l.state.arrayOpened = true
	}
	// Locate the cursor to read the next entry from
	if l.state.currentEntry == nil {
		// Ensure we have the next entry from each assembler
		for _, assembler := range l.assemblers {
			_, ok := l.state.nextEntryByLog[assembler.LogID()]
			if !ok {
				next, err := assembler.Next()
				if err != nil {
					return n, err
				}
				// TODO we'd ideally block and wait here (or return n=0) until we've
				//  found an entry for the log or the log is sealed and no more entries
				//  are expected.
				l.state.nextEntryByLog[assembler.LogID()] = next
			}
		}
		// Locate the oldest entry available
		var (
			next    *nextEntry
			nextLog models.LogDescriptorID
		)
		for _, assembler := range l.assemblers {
			potential := l.state.nextEntryByLog[assembler.LogID()]
			if potential == nil {
				continue // no entries remaining for this log
			}
			// NOTE this assumes l.assemblers are sorted
			var (
				nPersistent models.PersistentLogEntry
				pPersistent = potential.entry.Derived().(models.PersistentLogEntry)
			)
			if next != nil {
				nPersistent = next.entry.Derived().(models.PersistentLogEntry)
			}
			if nPersistent == nil || pPersistent.GetServerTimestamp().Before(nPersistent.GetServerTimestamp().Time) {
				nextLog = assembler.LogID()
				next = potential
			}
		}
		// If we couldn't locate any entries then write the synthetic stream end entry
		// if all log streams are finished
		if next == nil && !l.state.endEntryWritten {
			end := true
			for _, desc := range l.query.descriptors {
				if !desc.Sealed {
					end = false
					break
				}
			}
			if end {
				next, err = l.getEndEntry()
				if err != nil {
					return n, err
				}
				l.state.endEntryWritten = true
			}
		}
		if next == nil {
			// Write the closing JSON array token
			if !l.query.plaintext && !l.state.arrayClosed {
				if len(p) < n {
					return n, nil
				}
				p[n] = ']'
				n++
				l.state.arrayClosed = true
			}
			// No entry could be located, there's nothing more to read
			return n, io.EOF
		} else {
			// Otherwise, we've located the next entry to write out
			if l.query.plaintext {
				text, ok := next.entry.Derived().(models.PlainTextLogEntry)
				if ok {
					// Skip this entry if it doesn't support plaintext
					l.state.currentEntry = []byte(text.GetText() + "\n")
				}
			} else {
				if l.state.firstEntryWritten {
					l.state.currentEntry = append([]byte{','}, next.raw...)
				} else {
					l.state.currentEntry = next.raw
				}
			}
			delete(l.state.nextEntryByLog, nextLog)
		}
	}
	// Drain the current entry (may not be able to do this in a single invocation)
	nn := copy(p[n:], l.state.currentEntry)
	l.state.currentEntry = l.state.currentEntry[nn:]
	n += nn
	if len(l.state.currentEntry) > 0 {
		l.log.Debugf("Read %d bytes from current entry; %d more to go", nn, len(l.state.currentEntry))
	} else {
		l.state.currentEntry = nil
		l.state.firstEntryWritten = true
		l.log.Debugf("Read %d bytes from current entry; All done", nn)
	}
	return n, nil
}

func (l *reader) Close() error {
	for _, assembler := range l.assemblers {
		err := assembler.Close()
		if err != nil {
			l.log.Errorf("Ignoring error closing windowAssembler: %v", err)
		}
	}
	return nil
}

func (l *reader) getEndEntry() (*nextEntry, error) {
	entry := models.NewLogEntryEnd()
	raw, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("error marshaling end entry: %w", err)
	}
	next := &nextEntry{
		entry: entry,
		raw:   raw,
	}
	return next, nil
}
