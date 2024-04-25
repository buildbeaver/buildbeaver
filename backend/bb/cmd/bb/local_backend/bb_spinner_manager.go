package local_backend

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/chelnak/ysmrr"
)

type spinnerState struct {
	// spinner is the underlying spinner object (not nil)
	spinner *ysmrr.Spinner
	// jobName is the name to display in the spinner message for the job
	jobName string
	// jobNameDisplayLength is the desired length to pad or truncate the jobName to for display (in runes)
	jobNameDisplayLength int
	// jobFinished is true if this spinner's job is now finished, so no further text updates should be accepted
	jobFinished bool
	// text is the text to display in the spinner message after the job name
	text string
}

// newSpinnerState creates a spinner state object for the specified spinner, which must not be nil.
// jobName is the name to display for the job, and is immutable once set.
// The initial values supplied for jobNameDisplayLength and text will be recorded and used to
// update the spinner's message.
func newSpinnerState(spinner *ysmrr.Spinner, jobName string, jobNameDisplayLength int, text string) *spinnerState {
	state := &spinnerState{
		jobName:              jobName,
		jobNameDisplayLength: jobNameDisplayLength,
		jobFinished:          false,
		text:                 text,
		spinner:              spinner,
	}
	spinner.UpdateMessage(state.getDisplayMessage())
	return state
}

// setJobNameDisplayLength sets the length for the displayed job name (in runes), and updates the
// underlying spinner's message to reflect this change.
// Increasing or decreasing this number will display more or less of the originally supplied jobName.
func (s *spinnerState) setJobNameDisplayLength(length int) {
	s.jobNameDisplayLength = length
	s.spinner.UpdateMessage(s.getDisplayMessage())
}

// setText sets the text to display beside the job name in the spinner, and updates the underlying spinner's message.
// If finished is true then this will be the last text update for the spinner, and further updates will be ignored.
func (s *spinnerState) setText(text string, finished bool) {
	if s.jobFinished {
		return
	}
	s.text = text
	s.spinner.UpdateMessage(s.getDisplayMessage())
	s.jobFinished = finished
}

// getDisplayMessage returns the full message to display for the spinner, including the job name
// (padded or truncated to the correct length) followed by the text.
func (s *spinnerState) getDisplayMessage() string {
	// Create a name for the job which has the correct length (in runes, not in bytes)
	displayName := s.jobName
	nameLength := utf8.RuneCountInString(displayName)
	if s.jobNameDisplayLength > nameLength {
		// pad out with spaces
		displayName += strings.Repeat(" ", s.jobNameDisplayLength-nameLength)
	} else if s.jobNameDisplayLength < nameLength {
		displayName = truncateString(displayName, s.jobNameDisplayLength)
	}

	return fmt.Sprintf("%s %s", displayName, s.text)
}

// truncateString truncates the specified string to contain no more than maxLength runes.
// It works with multibyte runes (a basic string slice operation does not).
func truncateString(s string, maxLength int) string {
	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}
	return string(runes[0:maxLength])
}

// BBSpinnerManager maintains a set of spinners, one for each job being run by bb.
type BBSpinnerManager struct {
	manager ysmrr.SpinnerManager

	spinnersByID map[string]*spinnerState
	spinnersMu   sync.RWMutex // protects spinnersByID
}

func NewBBSpinnerManager() *BBSpinnerManager {
	return &BBSpinnerManager{
		manager:      ysmrr.NewSpinnerManager(),
		spinnersByID: map[string]*spinnerState{},
	}
}

func (s *BBSpinnerManager) Start() {
	s.manager.Start()
}

func (s *BBSpinnerManager) Stop() {
	s.manager.Stop()
}

// FindOrCreateSpinner checks if a spinner exists for the specified jobID, and creates a new spinner if required.
// The displayed job names for all existing spinners will be lengthened if necessary to match the new job name.
func (s *BBSpinnerManager) FindOrCreateSpinner(jobID models.JobID, jobName models.NodeFQN, jobStatus models.WorkflowStatus) {
	if s == nil {
		return // there is no spinner manager
	}
	s.spinnersMu.Lock()
	defer s.spinnersMu.Unlock()

	if _, exists := s.spinnersByID[jobID.String()]; exists {
		return
	}

	// Work out the maximum length (in runes) across all existing job names
	maxLen := 0
	for _, state := range s.spinnersByID {
		if state.jobNameDisplayLength > maxLen {
			maxLen = state.jobNameDisplayLength
		}
	}

	// If the new job name is longer than the existing ones then lengthen them all
	newJobNameLen := utf8.RuneCountInString(jobName.String())
	if newJobNameLen > maxLen {
		maxLen = newJobNameLen
		for _, state := range s.spinnersByID {
			state.setJobNameDisplayLength(maxLen)
		}
	}

	spinner := s.manager.AddSpinner("")
	state := newSpinnerState(spinner, jobName.String(), maxLen, jobStatus.String())
	s.spinnersByID[jobID.String()] = state
}

// UpdateSpinnerStatus updates the message and state being shown on the spinner for the specified jobID
// to match the specified job status.
// If there is no spinner for the job then this is a no-op.
func (s *BBSpinnerManager) UpdateSpinnerStatus(jobID models.JobID, jobStatus models.WorkflowStatus) {
	if s == nil {
		return // there is no spinner manager
	}
	s.spinnersMu.RLock()
	defer s.spinnersMu.RUnlock()

	state, found := s.spinnersByID[jobID.String()]
	if found {
		state.setText(jobStatus.String(), jobStatus.HasFinished())
		switch jobStatus {
		case models.WorkflowStatusSucceeded:
			state.spinner.Complete()
		case models.WorkflowStatusFailed, models.WorkflowStatusCanceled:
			state.spinner.Error()
		default:
			state.spinner.Start()
		}
	}
	// if not found then do nothing
}

// UpdateSpinnerText updates the text being shown on the spinner for the specified jobID.
// If there is no spinner for the job then this is a no-op.
func (s *BBSpinnerManager) UpdateSpinnerText(jobID models.JobID, newText string) {
	if s == nil {
		return // there is no spinner manager
	}
	s.spinnersMu.RLock()
	defer s.spinnersMu.RUnlock()

	state, found := s.spinnersByID[jobID.String()]
	if found {
		state.setText(newText, false)
	}
	// if not found then do nothing
}
