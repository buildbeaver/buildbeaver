package logging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// TestLogFileBuffer runs basic tests for the log file buffer stage, testing the success code path.
func TestLogFileBuffer(t *testing.T) {

	logBufferDir := RunnerLogTempDirectory(t.TempDir())

	scenarios := []struct {
		inputs          []string
		expectedOutputs []string
	}{{
		inputs:          []string{"Hello world", "Hello World", "wor", "ld", "helloworld", "hello\nworld"},
		expectedOutputs: []string{"Hello world", "Hello World", "wor", "ld", "helloworld", "hello\nworld"},
	}}

	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)
	pipelineCloser := func() {} // no-op

	for _, scenario := range scenarios {
		writer := &fakeWriterConfirmer{}
		logDescriptorID := models.NewLogDescriptorID()
		fileBuffer := NewLogFileBuffer(logFactory, pipelineCloser, writer, logDescriptorID, logBufferDir, 20)

		// Start the fileBuffer stage after hooking up its confirmation channel
		writer.SetConfirmationChannel(fileBuffer.GetConfirmationChannel())
		err := fileBuffer.Start()
		require.NoError(t, err)

		for i, input := range scenario.inputs {
			fileBuffer.Write(models.NewLogEntryLine(i+1, models.NewTime(time.Now()), input, i+1, nil))
		}
		fileBuffer.Flush() // must call Flush before Close
		fileBuffer.Close()

		assert.Len(t, writer.entries, len(scenario.expectedOutputs))
		for i := 0; i < len(writer.entries); i++ {
			actual := writer.entries[i]
			expected := scenario.expectedOutputs[i]
			text := actual.Derived().(models.PlainTextLogEntry).GetText()
			assert.Equal(t, expected, text)
		}
	}
}

type fakeWriterConfirmer struct {
	entries          []*models.LogEntry
	confirmationChan chan LogConfirmation // only supports one confirmation channel
}

func (f *fakeWriterConfirmer) SetConfirmationChannel(ch chan LogConfirmation) {
	f.confirmationChan = ch
}

func (f *fakeWriterConfirmer) Write(entry *models.LogEntry) {
	f.entries = append(f.entries, entry)
	// Send success confirmation every time we see a sequence number; this is the minimum behaviour that will work
	persistent, ok := entry.Derived().(models.PersistentLogEntry)
	if ok && f.confirmationChan != nil {
		confirmation := NewSuccessConfirmation(persistent.GetSeqNo())
		f.confirmationChan <- *confirmation
	}
}

func (f *fakeWriterConfirmer) Flush() {}

func (f *fakeWriterConfirmer) Close() {}
