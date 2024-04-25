package client

import (
	"fmt"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/buildbeaver/buildbeaver/common/logger"
)

type leveledLoggerWrapper struct {
	realLogger logger.Log
}

// NewLeveledLogger provides a LeveledLogger interface on top of the standard logging interface.
// This can be provided to retryableClient so that it can produce log messages at appropriate levels.
func NewLeveledLogger(realLogger logger.Log) retryablehttp.LeveledLogger {
	return &leveledLoggerWrapper{realLogger: realLogger}
}

func (l *leveledLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	l.realLogger.Error(l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	l.realLogger.Info(l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	l.realLogger.Debug(l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	l.realLogger.Warn(l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) convertMsg(msg string, keysAndValues ...interface{}) string {
	return fmt.Sprintf("%s: %v", msg, keysAndValues)
}
