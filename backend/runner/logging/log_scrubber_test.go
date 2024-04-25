package logging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

func TestMakeFiller(t *testing.T) {
	filler := makeFiller("secret", 0)
	assert.Equal(t, []byte(""), filler)

	filler = makeFiller("secret", 1)
	assert.Equal(t, []byte("s"), filler)

	filler = makeFiller("secret", 6)
	assert.Equal(t, []byte("secret"), filler)

	filler = makeFiller("secret", 9)
	assert.Equal(t, []byte("secretsec"), filler)

	filler = makeFiller("secret", 11)
	assert.Equal(t, []byte("secretsecre"), filler)

	filler = makeFiller("secret", 12)
	assert.Equal(t, []byte("secretsecret"), filler)
}

func TestLogScrubber_Write(t *testing.T) {

	scenarios := []struct {
		secretValues    []string
		inputs          []string
		expectedOutputs []string
	}{{
		secretValues:    []string{"world"},
		inputs:          []string{"Hello world", "Hello World", "wor", "ld", "helloworld", "hello\nworld"},
		expectedOutputs: []string{"Hello *****", "Hello World", "***", "**", "hello*****", "hello\n*****"},
	}}

	logRegistry, err := logger.NewLogRegistry("")
	assert.Nil(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)
	pipelineCloser := func() {} // no-op

	for _, scenario := range scenarios {
		var secrets []*models.SecretPlaintext
		for _, value := range scenario.secretValues {
			secrets = append(secrets, &models.SecretPlaintext{Value: value, Secret: &models.Secret{ID: models.NewSecretID()}})
		}

		writer := &logScrubberFakeWriter{}
		scrubber := NewLogScrubber(logFactory, pipelineCloser, writer, secrets)
		for i, input := range scenario.inputs {
			scrubber.Write(models.NewLogEntryLine(i, models.NewTime(time.Now()), input, i, nil))
		}
		scrubber.Flush() // ensure all buffered writes are sent
		scrubber.Close()

		assert.Len(t, writer.entries, len(scenario.expectedOutputs))
		for i := 0; i < len(writer.entries); i++ {
			actual := writer.entries[i]
			expected := scenario.expectedOutputs[i]
			text := actual.Derived().(models.PlainTextLogEntry).GetText()
			assert.Equal(t, expected, text)
		}
	}
}

type logScrubberFakeWriter struct {
	entries []*models.LogEntry
}

func (f *logScrubberFakeWriter) Write(entry *models.LogEntry) {
	f.entries = append(f.entries, entry)
}

func (f *logScrubberFakeWriter) Flush() {}

func (f *logScrubberFakeWriter) Close() {}
