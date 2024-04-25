package logging

import (
	"bytes"
	"sync"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

const filler = "*"

// LogScrubber scrubs plaintext user-supplied secrets from the log stream.
// NOTE: This introduces buffering into the stream - the buffer will be sized to match the largest secret.
type LogScrubber struct {
	mu               sync.Mutex
	log              logger.Log
	closePipeline    closeRequester
	entries          []*models.LogEntry
	buf              []byte
	secrets          []*models.SecretPlaintext
	fillerMap        map[models.SecretID][]byte
	longestSecretLen int
	logClosed        bool
	next             LogWriter
}

func NewLogScrubber(
	logFactory logger.LogFactory,
	closePipeline closeRequester,
	next LogWriter,
	secrets []*models.SecretPlaintext,
) *LogScrubber {
	var (
		longestSecretLen int
		fillerMap        = make(map[models.SecretID][]byte)
	)
	for _, secret := range secrets {
		if secret.IsInternal {
			continue
		}
		if len(secret.Value) > longestSecretLen {
			longestSecretLen = len(secret.Value)
		}
		fillerMap[secret.ID] = makeFiller(filler, len(secret.Value))
	}
	return &LogScrubber{
		log:              logFactory("LogScrubber"),
		closePipeline:    closePipeline,
		secrets:          secrets,
		fillerMap:        fillerMap,
		longestSecretLen: longestSecretLen,
		next:             next,
	}
}

// Write a new entry to the stream.
func (l *LogScrubber) Write(entry *models.LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logClosed {
		l.log.Errorf("Attempt to write to log stream after log is closed; discarding log entry")
		return
	}

	plaintext, ok := entry.Derived().(models.PlainTextLogEntry)
	if !ok || len(l.secrets) == 0 {
		l.next.Write(entry)
		return
	}
	l.entries = append(l.entries, entry)
	l.buf = append(l.buf, plaintext.GetText()...)
	for _, secret := range l.secrets {
		if secret.IsInternal {
			continue
		}
		l.buf = bytes.ReplaceAll(l.buf, []byte(secret.Value), l.fillerMap[secret.ID])
	}
	l.flush(max(len(l.buf)-l.longestSecretLen, 0))
}

func (l *LogScrubber) Flush() {
	// Hold a lock while we flush, no need to hold lock while the rest of the pipeline flushes
	l.mu.Lock()
	if l.logClosed {
		l.log.Errorf("Attempt to flush log stream after log is closed; ignoring Flush")
		l.mu.Unlock()
		return
	}
	l.flush(len(l.buf))
	l.mu.Unlock()

	l.next.Flush()
}

func (l *LogScrubber) Close() {
	// Hold a lock while we flush before closing, no need to hold lock while the rest of the pipeline closes
	l.mu.Lock()
	if l.logClosed {
		l.log.Warnf("Attempt to Close log stream that is already closed; ignoring Close")
		l.mu.Unlock()
		return
	}
	l.logClosed = true
	l.mu.Unlock()

	l.next.Close()
}

func (l *LogScrubber) flush(n int) {
	var (
		bufOffset   = 0
		entryOffset = 0
	)
	for _, entry := range l.entries {
		plaintext := entry.Derived().(models.PlainTextLogEntry)
		entryLen := len(plaintext.GetText())
		if entryLen > n {
			break
		}
		plaintext.SetText(string(l.buf[bufOffset : bufOffset+entryLen]))
		l.next.Write(entry)
		n = n - entryLen
		bufOffset += entryLen
		entryOffset++
	}
	l.buf = l.buf[bufOffset:]
	l.entries = l.entries[entryOffset:]
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func makeFiller(filer string, n int) []byte {
	buf := make([]byte, n)
	for i := 0; i < len(buf); i++ {
		buf[i] = filer[i%len(filer)]
	}
	return buf
}
