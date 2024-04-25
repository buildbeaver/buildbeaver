package logging

import (
	"sync"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// LogSequencer serializes and allocates sequence and line nos to log entries before writing them to an underlying stream.
type LogSequencer struct {
	mu            sync.Mutex
	log           logger.Log
	closePipeline closeRequester
	next          LogWriter
	state         struct {
		nextSeqNo  int
		nextLineNo int
	}
}

func NewLogSequencer(
	logFactory logger.LogFactory,
	closePipeline closeRequester,
	next LogWriter,
) *LogSequencer {
	w := &LogSequencer{
		log:           logFactory("LogSequencer"),
		closePipeline: closePipeline,
		next:          next,
	}
	w.state.nextSeqNo = 1
	w.state.nextLineNo = 1
	return w
}

// Write a new entry to the stream. The entry will be allocated a seq no and line no (if appropriate).
func (l *LogSequencer) Write(entry *models.LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	persistent, ok := entry.Derived().(models.PersistentLogEntry)
	if ok {
		persistent.SetSeqNo(l.state.nextSeqNo)
		l.state.nextSeqNo++
	}
	derived := entry.Derived()
	// TODO: Check if an error qualifies as a LogEntryLine since it includes one; may need to include error
	if _, ok := derived.(*models.LogEntryLine); ok {
		line := entry.Derived().(*models.LogEntryLine)
		line.LineNo = l.state.nextLineNo
		l.state.nextLineNo++
	}
	l.next.Write(entry)
}

func (l *LogSequencer) Flush() {
	l.next.Flush()
}

func (l *LogSequencer) Close() {
	l.next.Close()
}
