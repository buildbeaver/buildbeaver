package bb

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

type Logger func(level LogLevel, message string)

type LogLevel int

const (
	LogLevelTrace = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

var (
	Log             Logger   = DefaultLogger
	DefaultLogLevel LogLevel = LogLevelInfo
)

// SetLogger sets the logger function that will be called by the client code to log messages.
func SetLogger(logger Logger) {
	Log = logger
}

func SetDefaultLogLevel(level LogLevel) {
	DefaultLogLevel = level
}

// DefaultLogger is used by default if SetLogger() is not called, and outputs the log messages to stdout.
func DefaultLogger(level LogLevel, message string) {
	if level >= DefaultLogLevel {
		fmt.Printf("%s: %s\n", strings.ToUpper(level.String()), message)
	}
}

type leveledLoggerWrapper struct {
	realLogger Logger
}

// NewLeveledLogger provides a retryablehttp.LeveledLogger interface on top of the basic Logger interface.
// This can be provided to retryableClient so that it can produce log messages at appropriate levels.
func NewLeveledLogger(realLogger Logger) retryablehttp.LeveledLogger {
	return &leveledLoggerWrapper{realLogger: realLogger}
}

func (l *leveledLoggerWrapper) Error(msg string, keysAndValues ...interface{}) {
	l.realLogger(LogLevelError, l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Info(msg string, keysAndValues ...interface{}) {
	l.realLogger(LogLevelInfo, l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Debug(msg string, keysAndValues ...interface{}) {
	l.realLogger(LogLevelDebug, l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) Warn(msg string, keysAndValues ...interface{}) {
	l.realLogger(LogLevelWarn, l.convertMsg(msg, keysAndValues))
}

func (l *leveledLoggerWrapper) convertMsg(msg string, keysAndValues ...interface{}) string {
	return fmt.Sprintf("%s: %v", msg, keysAndValues)
}
