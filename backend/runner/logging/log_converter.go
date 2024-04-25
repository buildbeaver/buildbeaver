package logging

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/benbjohnson/clock"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
)

// LogConverter converts a plaintext log stream to a structured log stream.
type LogConverter struct {
	*util.StatefulService
	clk    clock.Clock
	log    logger.Log
	next   LogWriter
	reader io.ReadCloser
	writer io.WriteCloser
}

func NewLogConverter(clk clock.Clock, logFactory logger.LogFactory, next LogWriter) *LogConverter {
	reader, writer := io.Pipe()
	l := &LogConverter{clk: clk, log: logFactory("LogConverter"), next: next, reader: reader, writer: writer}
	l.StatefulService = util.NewStatefulService(context.Background(), l.log, l.loop)
	return l
}

func (l *LogConverter) Write(p []byte) (n int, err error) {
	return l.writer.Write(p)
}

func (l *LogConverter) Close() error {
	defer l.Stop()
	return l.reader.Close()
}

func (l *LogConverter) loop() {
	scanner := bufio.NewScanner(l.reader)
	for l.Ctx().Err() == nil && scanner.Scan() {
		// TODO We could look for special syntax in the plaintext stream and convert to other structures (like blocks) here
		// TODO: Consider a special syntax for marking error messages as well, to be translated to 'error' log entries
		l.log.Tracef("Writing line: %w", scanner.Text())
		l.next.Write(models.NewLogEntryLine(-1, models.NewTime(l.clk.Now()), scanner.Text(), -1, nil))
	}
	err := scanner.Err()
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe) {
		// Send an error down the log pipeline
		errorText := fmt.Sprintf("Error reading data to be logged; some log data may be missing: %s", err.Error())
		l.next.Write(models.NewLogEntryError(-1, models.NewTime(l.clk.Now()), errorText, -1, nil))
		// Log the error to the runner's log
		l.log.Errorf("Ignoring error reading from scanner: %v", err)
	}
}
