package ysmrr

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chelnak/ysmrr/pkg/colors"
	"github.com/chelnak/ysmrr/pkg/tput"
	"github.com/fatih/color"
	"github.com/nathan-fiscaletti/consolesize-go"
)

// Spinner manages a single spinner
type Spinner struct {
	mutex         sync.Mutex
	spinnerColor  *color.Color
	completeColor *color.Color
	errorColor    *color.Color
	messageColor  *color.Color
	message       string
	runtimeStart  time.Time
	runtimeEnd    time.Time
	runtimeColor  *color.Color
	complete      bool
	err           bool
	hasUpdate     chan bool
}

// GetMessage returns the current spinner message.
func (s *Spinner) GetMessage() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.message
}

// UpdateMessage updates the spinner message.
func (s *Spinner) UpdateMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.message = message
	s.notifyHasUpdate()
}

// UpdateMessagef updates the spinner message with a formatted string.
func (s *Spinner) UpdateMessagef(format string, a ...interface{}) {
	s.UpdateMessage(fmt.Sprintf(format, a...))
}

func (s *Spinner) IsStarted() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return !s.runtimeStart.IsZero()
}

// IsComplete returns true if the spinner is complete.
func (s *Spinner) IsComplete() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.complete
}

// IsError returns true if the spinner is in error state.
func (s *Spinner) IsError() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.err
}

// CompleteWithMessage marks the spinner as complete with a message.
func (s *Spinner) CompleteWithMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.message = message
	s.complete = true
	s.runtimeEnd = time.Now()
}

// CompleteWithMessagef marks the spinner as complete with a formatted string.
func (s *Spinner) CompleteWithMessagef(format string, a ...interface{}) {
	s.CompleteWithMessage(fmt.Sprintf(format, a...))
}

// Complete marks the spinner as complete.
func (s *Spinner) Complete() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.complete = true
	s.runtimeEnd = time.Now()
}

func (s *Spinner) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.runtimeStart = time.Now()
}

// ErrorWithMessage marks the spinner as error with a message.
func (s *Spinner) ErrorWithMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.message = message
	s.err = true
	s.runtimeEnd = time.Now()
}

// ErrorWithMessagef marks the spinner as error with a formatted string.
func (s *Spinner) ErrorWithMessagef(format string, a ...interface{}) {
	s.ErrorWithMessage(fmt.Sprintf(format, a...))
}

// Error marks the spinner as error.
func (s *Spinner) Error() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.err = true
	s.runtimeEnd = time.Now()
}

var divs = []time.Duration{
	time.Duration(1),
	time.Duration(10),
	time.Duration(100),
	time.Duration(1000),
}

// Print prints the spinner at a given position.
func (s *Spinner) Print(w io.Writer, char string) {
	cols, _ := consolesize.GetConsoleSize() // can return zero
	if cols < 40 {
		cols = 40
	} else if cols > 100 {
		cols = 100
	}
	remaining := cols - 1

	started := s.IsStarted()
	var runtimeColor *color.Color

	if s.IsComplete() {
		print(w, "✓", s.completeColor)
		runtimeColor = s.completeColor
	} else if s.IsError() {
		print(w, "✗", s.errorColor)
		runtimeColor = s.errorColor
	} else {
		print(w, char, s.spinnerColor)
		runtimeColor = s.runtimeColor
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var runtimeStr string
	if started {
		since := time.Now()
		if !s.runtimeEnd.IsZero() {
			since = s.runtimeEnd
		}
		runtimeStr = since.Sub(s.runtimeStart).Round(time.Second / 10).String()
	}

	// We don't have room to render the timer
	var maxRuntimeWidth = 12
	if remaining < maxRuntimeWidth {
		runtimeStr = ""
		maxRuntimeWidth = 0
	}

	message := fmt.Sprintf(" %s", s.message)

	if remaining < maxRuntimeWidth+len(message) {
		message = message[:remaining-maxRuntimeWidth-2]
		message += ".."
	}
	padding := remaining - len(runtimeStr) - len(message)

	print(w, message, s.messageColor)
	if padding > 0 {
		tput.Right(w, padding)
	}
	if len(runtimeStr) > 0 {
		print(w, runtimeStr, runtimeColor)
	}
	print(w, "\r\n", nil)
}

func print(w io.Writer, s string, c *color.Color) {
	if c != nil {
		_, _ = c.Fprintf(w, s)
	} else {
		fmt.Fprint(w, s)
	}
}

func (s *Spinner) notifyHasUpdate() {
	select {
	case s.hasUpdate <- true:
	default:
	}
}

type SpinnerOptions struct {
	SpinnerColor  colors.Color
	CompleteColor colors.Color
	ErrorColor    colors.Color
	MessageColor  colors.Color
	RuntimeColor  colors.Color
	Message       string
	Start         time.Time
	HasUpdate     chan bool
}

// NewSpinner creates a new spinner instance.
func NewSpinner(options SpinnerOptions) *Spinner {
	return &Spinner{
		spinnerColor:  colors.GetColor(options.SpinnerColor),
		completeColor: colors.GetColor(options.CompleteColor),
		errorColor:    colors.GetColor(options.ErrorColor),
		messageColor:  colors.GetColor(options.MessageColor),
		runtimeColor:  colors.GetColor(options.RuntimeColor),
		message:       options.Message,
		runtimeStart:  options.Start,
		hasUpdate:     options.HasUpdate,
	}
}
