package logging

import (
	"fmt"

	"github.com/benbjohnson/clock"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// StructuredLogger writes structured entries to a structured log.
// Provides utility functions for managing the level of nested blocks.
type StructuredLogger struct {
	clk        clock.Clock
	log        logger.Log
	resourceID models.ResourceID
	block      *models.ResourceName
	outer      *StructuredLogger
	writer     LogWriter
}

func NewStructuredLogger(clk clock.Clock, factory logger.LogFactory, writer LogWriter) *StructuredLogger {
	return &StructuredLogger{
		clk:    clk,
		log:    factory("StructuredLogger"),
		writer: writer,
	}
}

// WriteLine writes a line to the log inside the current block (if any).
func (l *StructuredLogger) WriteLine(text string) {
	line := models.NewLogEntryLine(-1, models.NewTime(l.clk.Now()), text, -1, l.block)
	l.writer.Write(line)
}

// WriteLinef writes a line with formatting to the log inside the current block (if any).
func (l *StructuredLogger) WriteLinef(format string, args ...interface{}) {
	l.WriteLine(fmt.Sprintf(format, args...))
}

// WriteError writes an error message to the log inside the current block (if any).
func (l *StructuredLogger) WriteError(errorText string) {
	error := models.NewLogEntryError(-1, models.NewTime(l.clk.Now()), errorText, -1, l.block)
	l.writer.Write(error)
}

// WriteErrorf writes an error message with formatting to the log inside the current block (if any).
func (l *StructuredLogger) WriteErrorf(format string, args ...interface{}) {
	l.WriteError(fmt.Sprintf(format, args...))
}

// Wrap returns a new logger that will wrap lines inside the named block.
// Use Unwrap() to close the block and return to the current level.
func (l *StructuredLogger) Wrap(name string, text string) *StructuredLogger {
	resourceName := models.ResourceName(name)
	inner := &StructuredLogger{
		clk:        l.clk,
		log:        l.log,
		resourceID: l.resourceID,
		block:      &resourceName,
		outer:      l,
		writer:     l.writer,
	}
	block := models.NewLogEntryBlock(-1, models.NewTime(l.clk.Now()), text, resourceName, l.block)
	l.writer.Write(block)
	return inner
}

// Wrapf returns a new logger that will wrap lines inside the named block.
// Use Unwrap() to close the block and return to the current level.
func (l *StructuredLogger) Wrapf(name string, format string, args ...interface{}) *StructuredLogger {
	return l.Wrap(name, fmt.Sprintf(format, args...))
}

// Unwrap returns the logger above the current block.
func (l *StructuredLogger) Unwrap() *StructuredLogger {
	if l.outer == nil {
		l.log.Panicf("No more blocks to unwrap for %s", l.resourceID)
	}
	return l.outer
}
