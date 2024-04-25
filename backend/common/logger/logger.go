package logger

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Log interface {
	WithField(name string, value interface{}) Log
	WithFields(fields Fields) Log
	Trace(args ...interface{})
	Tracef(msg string, args ...interface{})
	Debug(args ...interface{})
	Debugf(msg string, args ...interface{})
	Info(args ...interface{})
	Infof(msg string, args ...interface{})
	Warn(args ...interface{})
	Warnf(msg string, args ...interface{})
	Error(args ...interface{})
	Errorf(msg string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(msg string, args ...interface{})
	Panic(args ...interface{})
	Panicf(msg string, args ...interface{})
	Print(args ...interface{})
}

// Fields is a set of keys/values to include in a structured log message.
type Fields map[string]interface{}

type LogFilePath string

// LogFactory produces a logger that can be used to log messages for the
// specified subsystem.
type LogFactory func(subsystem string) Log

// LogrusLogger is a Log implementation that using the Logrus library.
type LogrusLogger struct {
	*logrus.Entry
}

func (l *LogrusLogger) WithField(name string, value interface{}) Log {
	fields := map[string]interface{}{name: value}
	return &LogrusLogger{Entry: l.Entry.WithFields(fields)}
}

func (l *LogrusLogger) WithFields(fields Fields) Log {
	return &LogrusLogger{Entry: l.Entry.WithFields(logrus.Fields(fields))}
}

func MakeLogrusLogFactoryStdOut(logRegistry *LogRegistry) LogFactory {
	return func(subsystem string) Log {
		log := logrus.New()
		log.SetLevel(logRegistry.GetLogLevel(subsystem))
		log.SetOutput(os.Stdout)

		if isatty.IsTerminal(os.Stdout.Fd()) {
			log.SetFormatter(&logrus.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
				FullTimestamp:   true,
				ForceColors:     false, // logrus does not support colors on Windows
				DisableQuote:    true,  // prevent logrus from quoting all values on Windows in terminals
			})
		} else {
			log.SetFormatter(&logrus.JSONFormatter{
				TimestampFormat:   "2006-01-02 15:04:05",
				DisableTimestamp:  false,
				DisableHTMLEscape: false,
				DataKey:           "",
				FieldMap:          nil,
				CallerPrettyfier:  nil,
				PrettyPrint:       false,
			})
		}
		entry := log.WithFields(logrus.Fields{
			"system": subsystem,
		})
		logRegistry.RegisterLogger(subsystem, log)
		return &LogrusLogger{Entry: entry}
	}
}

// MakeLogrusLogFactoryStdOutPlain creates a log factory that will output very plain-looking log lines,
// with no timestamp and no system field.
func MakeLogrusLogFactoryStdOutPlain(logRegistry *LogRegistry) LogFactory {
	return func(subsystem string) Log {
		log := logrus.New()
		log.SetLevel(logRegistry.GetLogLevel(subsystem))
		log.SetOutput(os.Stdout)
		log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
		entry := log.WithFields(logrus.Fields{})
		logRegistry.RegisterLogger(subsystem, log)
		return &LogrusLogger{Entry: entry}
	}
}

func MakeLogrusLogFactoryToFile(logRegistry *LogRegistry, logFile LogFilePath) (LogFactory, error) {
	file, err := os.OpenFile(string(logFile), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening log file: %s", logFile)
	}
	return func(subsystem string) Log {
		log := logrus.New()
		log.SetLevel(logRegistry.GetLogLevel(subsystem))
		log.SetOutput(file)
		log.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
		entry := log.WithFields(logrus.Fields{
			"system": subsystem,
		})
		logRegistry.RegisterLogger(subsystem, log)
		return &LogrusLogger{Entry: entry}
	}, nil
}

// NoOpLog implements the Log interface without actually performing any logging or other actions.
type NoOpLog struct {
}

func NewNoOpLog() *NoOpLog {
	return &NoOpLog{}
}

// NoOpLogFactory is a LogFactory function that always returns a NoOpLog, for when logging is not required.
func NoOpLogFactory(subsystem string) Log {
	return NewNoOpLog()
}

func (l *NoOpLog) WithField(name string, value interface{}) Log { return NewNoOpLog() }
func (l *NoOpLog) WithFields(fields Fields) Log                 { return NewNoOpLog() }
func (l *NoOpLog) Trace(args ...interface{})                    {}
func (l *NoOpLog) Tracef(msg string, args ...interface{})       {}
func (l *NoOpLog) Debug(args ...interface{})                    {}
func (l *NoOpLog) Debugf(msg string, args ...interface{})       {}
func (l *NoOpLog) Info(args ...interface{})                     {}
func (l *NoOpLog) Infof(msg string, args ...interface{})        {}
func (l *NoOpLog) Warn(args ...interface{})                     {}
func (l *NoOpLog) Warnf(msg string, args ...interface{})        {}
func (l *NoOpLog) Error(args ...interface{})                    {}
func (l *NoOpLog) Errorf(msg string, args ...interface{})       {}
func (l *NoOpLog) Fatal(args ...interface{})                    {}
func (l *NoOpLog) Fatalf(msg string, args ...interface{})       {}
func (l *NoOpLog) Panic(args ...interface{})                    {}
func (l *NoOpLog) Panicf(msg string, args ...interface{})       {}
func (l *NoOpLog) Print(args ...interface{})                    {}
