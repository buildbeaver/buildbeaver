package logging

import (
	"fmt"
	"sync"

	"github.com/buildbeaver/buildbeaver/common/gerror"
)

// LogFileBufferIndex keeps index information about the location of log entries within a log buffer file.
// All methods are thread-safe.
type LogFileBufferIndex struct {
	mu                sync.Mutex    // protects all variables
	lastSeqNo         int           // last sequence number indexed
	seqNoToFileOffset map[int]int64 // maps sequence number to the file offset at the start of that log entry
}

func NewLogFileBufferIndex() *LogFileBufferIndex {
	return &LogFileBufferIndex{
		lastSeqNo:         0,
		seqNoToFileOffset: make(map[int]int64),
	}
}

// AddLogEntry adds information about a new log entry to the index.
func (i *LogFileBufferIndex) AddLogEntry(seqNo int, startFileOffset int64) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.lastSeqNo = seqNo
	i.seqNoToFileOffset[seqNo] = startFileOffset
}

// GetStartOffset returns the file offset at the start of the log entry with the given sequence number,
// or an error if the sequence number can't be found.
func (i *LogFileBufferIndex) GetStartOffset(seqNo int) (int64, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	fileOffset, ok := i.seqNoToFileOffset[seqNo]
	if !ok {
		return -1, gerror.NewErrNotFound(fmt.Sprintf("error: no offset found for sequence number %d", seqNo))
	}
	return fileOffset, nil
}

// GetLastSeqNo returns the sequence number most recently added to the index. If the log entries are added in
// order of sequence number this will be the highest sequence number in the index, although this is not
// enforced. If no log entries have been added then zero is returned.
func (i *LogFileBufferIndex) GetLastSeqNo() int {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.lastSeqNo
}
