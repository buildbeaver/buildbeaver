package logger

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

const defaultLogLevel = logrus.InfoLevel

var levelMap = map[string]logrus.Level{
	"trace":   logrus.TraceLevel,
	"debug":   logrus.DebugLevel,
	"info":    logrus.InfoLevel,
	"warning": logrus.WarnLevel,
	"error":   logrus.ErrorLevel,
	"fatal":   logrus.FatalLevel,
	"panic":   logrus.PanicLevel,
}

type LogLevelConfig string

type LogRegistry struct {
	loggerBySubsystem map[string]*logrus.Logger
	levelBySubsystem  map[string]logrus.Level
	loggersMu         sync.Mutex
}

// ListLogLevels returns a comma seperated string listing valid log levels.
func ListLogLevels() string {
	str := ""
	for k, _ := range levelMap {
		if str != "" {
			str += ", "
		}
		str += fmt.Sprintf("%q", k)
	}
	return str
}

func NewLogRegistry(config LogLevelConfig) (*LogRegistry, error) {
	r := &LogRegistry{
		loggerBySubsystem: make(map[string]*logrus.Logger),
		levelBySubsystem:  make(map[string]logrus.Level),
		loggersMu:         sync.Mutex{},
	}
	if config != "" {
		pairs := strings.Split(string(config), ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("error invalid log level format: %v", pair)
			}
			level, ok := levelMap[parts[1]]
			if !ok {
				return nil, fmt.Errorf("error invalid log level for %q: %v", parts[0], parts[1])
			}
			r.levelBySubsystem[parts[0]] = level
		}
	}
	return r, nil
}

// GetLogLevel returns the configured log level for the specified subsystem.
func (r *LogRegistry) GetLogLevel(subsystem string) logrus.Level {
	r.loggersMu.Lock()
	defer r.loggersMu.Unlock()
	level, ok := r.levelBySubsystem[subsystem]
	if !ok {
		return defaultLogLevel
	}
	return level
}

// RegisterLogger registers a logger with the registry.
// Kind of useless right now, but the idea is that we will be able to dynamically change the log level
// of registered loggers in the future.
func (r *LogRegistry) RegisterLogger(subsystem string, logger *logrus.Logger) {
	r.loggersMu.Lock()
	defer r.loggersMu.Unlock()
	r.loggerBySubsystem[subsystem] = logger
}
