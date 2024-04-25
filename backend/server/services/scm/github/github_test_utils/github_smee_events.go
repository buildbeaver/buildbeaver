package github_test_utils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v28/github"
)

// EventMatcher is a function that determines whether an event matches its pre-determined criteria.
type EventMatcher func(*SmeeNotification) bool

// MatchNothing returns an event matcher that does not match any events, for testing.
func MatchNothing() EventMatcher {
	return func(event *SmeeNotification) bool {
		return false
	}
}

// MatchPushEvent is an EventMatcher function that matches a GitHub 'push' event for the specified repo.
func MatchPushEvent(repoID int64) EventMatcher {
	return func(event *SmeeNotification) bool {
		if event.Headers["x-github-event"] != "push" {
			return false // wrong type of event
		}
		pushEvent := &github.PushEvent{}
		err := json.Unmarshal(event.Body, pushEvent)
		if err != nil {
			return false // body isn't a PushEvent
		}
		// Check repo
		if pushEvent.GetRepo().GetID() != repoID {
			return false // wrong repo
		}
		return true
	}
}

// MatchPullRequestEvent is an EventMatcher function that matches a GitHub 'pull_request' event with
// the specified action string, for a pull request to merge content into the specified base repo.
// If 'action' is supplied as an empty string then any action is matched.
func MatchPullRequestEvent(baseRepoID int64, action string) EventMatcher {
	return func(event *SmeeNotification) bool {
		if event.Headers["x-github-event"] != "pull_request" {
			return false // wrong type of event
		}
		pullRequestEvent := &github.PullRequestEvent{}
		err := json.Unmarshal(event.Body, pullRequestEvent)
		if err != nil {
			return false // body isn't a PushEvent
		}
		// Check action and base repo
		if action != "" && pullRequestEvent.GetAction() != action {
			return false
		}
		if pullRequestEvent.GetPullRequest().GetBase().GetRepo().GetID() != baseRepoID {
			return false // wrong repo
		}
		return true
	}
}

// ProcessEventsUntilMatched processes Smee notifications from a channel, calling the supplied processor
// function for each event. No new Goroutine is started; everything is run on the calling Goroutine.
// Any errors returned from the processor function will cause this function
// to stop processing and return the error.
// The processor function will be called for every event, regardless of whether it matches any of the
// supplied EventMatchers.
// The function stops reading events and returns once an event has been matched to each the supplied
// eventMatchers. The same event matcher can be listed more than once; in that case a corresponding
// number of matching events must be seen before the function returns.
// EventMatchers can be supplied in any order.
// After the duration specified in 'timeout' has elapsed, if not all events have been seen then the
// function stops and returns a timeout error.
func ProcessEventsUntilMatched(
	eventChan chan *SmeeNotification,
	timeout time.Duration,
	processor SmeeEventHandler,
	eventMatchers ...EventMatcher,
) error {
	if len(eventMatchers) == 0 {
		return fmt.Errorf("error: no event matchers passed in to ProcessEventsUntilMatched()")
	}

	// Make a map to store EventMatchers that aren't matched yet. EventMatches will be removed
	// as they are matched to events.
	// We map an index to an EventMatcher since the same EventMatcher can be listed multiple times;
	// this is a map rather than a slice to allow for efficient removal from the map.
	unmatched := make(map[int]EventMatcher)
	for i, matcher := range eventMatchers {
		unmatched[i] = matcher
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Process events until all matches are found, or until we time out
	for {
		// Get the next event, or timeout and return
		var event *SmeeNotification
		select {
		case event = <-eventChan:
			// Got the next event; just move on
		case <-timer.C:
			return fmt.Errorf("timeout after %s waiting for events", timeout)
		}

		// Process the event
		err := processor(event)
		if err != nil {
			return err
		}

		// Give every unmatched EventMatcher a chance to match the event until we get a match
		for i, matcher := range unmatched {
			if matcher(event) {
				delete(unmatched, i)
				break // only match one event
			}
		}
		if len(unmatched) == 0 {
			break // all events matched so we're done
		}
	}

	return nil
}
