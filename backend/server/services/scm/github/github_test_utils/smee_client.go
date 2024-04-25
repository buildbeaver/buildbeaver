package github_test_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/r3labs/sse"

	"github.com/buildbeaver/buildbeaver/common/logger"
)

type SmeeNotification struct {
	SmeeID      int64
	Headers     map[string]string
	QueryParams interface{}
	Body        []byte
}

type SmeeEventHandler func(*SmeeNotification) error

type SmeeClient struct {
	sseClient *sse.Client

	// subscribers maps subscriber channels to state for each one
	subscribers      map[chan *SmeeNotification]subscriberState
	subscribersMutex sync.Mutex // just for subscribers

	logger.Log
}

// Information we keep about each channel subscription
type subscriberState struct {
	unsubscribeChan chan bool
	sseChan         chan *sse.Event
}

// NewSmeeClientForGitHubTestAccount returns a Smee Client that can be used to receive notifications from the
// GitHub test account.
func NewSmeeClientForGitHubTestAccount(logFactory logger.LogFactory) *SmeeClient {
	return NewSmeeClient(SmeeTestAccountEndpoint, logFactory)
}

func NewSmeeClient(smeeURL string, logFactory logger.LogFactory) *SmeeClient {
	return &SmeeClient{
		sseClient:   sse.NewClient(smeeURL),
		subscribers: make(map[chan *SmeeNotification]subscriberState),
		Log:         logFactory("SmeeClient"),
	}
}

// SubscribeHandler subscribes an event handler to Smee events. The eventHandler function will be
// called back on a different GoRoutine.
// Returns a function to call to shut down the subscription.
// TODO: Return error from this func if sseClient.SubscribeRawWithContext returns an error
func (s *SmeeClient) SubscribeHandler(eventHandler SmeeEventHandler) (shutdown func()) {
	// Create a context to cancel when we want to shut down
	ctx, ctxCancel := context.WithCancel(context.Background())
	startupCompletedChan := make(chan bool)
	shutdownCompletedChan := make(chan bool)
	// Cancel function to return to the caller
	shutdownFunc := func() {
		// We must wait for SSEClient to start up before cancelling the context;
		// if sseClient.SubscribeWithContext is called with an already-cancelled context
		// then it never cancels
		s.Trace("Waiting for startup")
		<-startupCompletedChan
		s.Trace("Calling cancel function on context")
		ctxCancel() // tell goroutine to shut down
		s.Trace("Waiting for shutdown")
		<-shutdownCompletedChan // wait for shutdown to complete
		s.Trace("Got shutdown complete signal")
	}

	go func() {
		started := false
		// smee.io ignores the stream used when subscribing, so use a raw subscribe
		err := s.sseClient.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			// Look for ready message so we know it's safe to return from SubscribeHandler()
			if string(msg.Event) == "ready" && !started {
				s.Trace("Received ready notification; signalling startup is completed")
				close(startupCompletedChan)
				started = true // only close startupCompletedChan once
			}

			if notification := s.sseEventToNotification(msg); notification != nil {
				_ = eventHandler(notification)
				// TODO: Shut down processing if an error returned from handler
			}
		})
		s.Infof("SMEE client unsubscribed - err returned: %v", err)
		close(shutdownCompletedChan)
	}()

	// Wait for startup so our caller doesn't miss any events by performing GitHub API calls too early
	s.Trace("Waiting for startup")
	<-startupCompletedChan

	return shutdownFunc
}

// SubscribeChan subscribes a channel to receive a stream of Smee events.
// Returns once the connection has been successfully established, or returns an error if
// the connection could not be established.
func (s *SmeeClient) SubscribeChan(notificationCh chan *SmeeNotification) error {
	// channel to use for underlying SSE library subscription
	sseChan := make(chan *sse.Event)
	// Send a value down this chan to unsubscribe
	unsubscribeChan := make(chan bool)

	s.subscribersMutex.Lock()
	s.subscribers[notificationCh] = subscriberState{
		sseChan:         sseChan,
		unsubscribeChan: unsubscribeChan,
	}
	s.subscribersMutex.Unlock()

	err := s.sseClient.SubscribeChanRaw(sseChan)
	if err != nil {
		s.cleanUpSubscriber(notificationCh)
		return err
	}

	// start receiving SSE library events, parsing and forwarding them to our client channel
	go func() {
		defer s.cleanUpSubscriber(notificationCh)
		for {
			// Wait for message to arrive, or exit
			var sseEvent *sse.Event
			select {
			case <-unsubscribeChan:
				return
			case sseEvent = <-sseChan:
			}

			if toSend := s.sseEventToNotification(sseEvent); toSend != nil {
				// Send outgoing notification, or exit
				select {
				case <-unsubscribeChan:
					return
				case notificationCh <- toSend:
				}
			}
		}
	}()

	return nil
}

// UnsubscribeChan unsubscribes a previously subscribed channel from a stream of Smee events.
func (s *SmeeClient) UnsubscribeChan(ch chan *SmeeNotification) {
	s.subscribersMutex.Lock()
	defer s.subscribersMutex.Unlock()

	subscriber, ok := s.subscribers[ch]
	if ok {
		subscriber.unsubscribeChan <- true
	}
}

// sseEventToNotification examines an SSE event and converts to a notification if relevant.
// If the event is something that should be sent on to the subscriber then it's parsed into
// a SmeeNotification which is returned.
// Internal-only or unrecognised events are logged and ignored, and nil is returned.
// If the event passed in is nil then nil is returned.
func (s *SmeeClient) sseEventToNotification(event *sse.Event) *SmeeNotification {
	if event == nil {
		return nil
	}
	s.Tracef("Smee message received (handler): ID %q, Event %q, Retry %q, Data length %d, Data:%s",
		string(event.ID), string(event.Event), string(event.Retry), len(event.Data), event.Data)

	// Real events from smee.io have an empty event string; other events can be ignored
	switch string(event.Event) {
	case "": // a real event from smee.io
		smeeNotification, err := s.parseNotification(event.ID, event.Data)
		if err != nil {
			s.Warnf("Ignoring error parsing Smee notification: %s", err.Error())
			return nil
		}
		return smeeNotification
	default:
		s.Infof("Ignoring smee event of type %s", string(event.Event))
		return nil
	}
}

// cleanUpSubscriber closes channels and removes the subscriber's state from the map.
func (s *SmeeClient) cleanUpSubscriber(ch chan *SmeeNotification) {
	s.subscribersMutex.Lock()
	defer s.subscribersMutex.Unlock()

	subscriber, ok := s.subscribers[ch]
	if ok {
		close(subscriber.unsubscribeChan)
		delete(s.subscribers, ch)
	}
}

func (s *SmeeClient) parseNotification(id []byte, data []byte) (*SmeeNotification, error) {
	// a smee ID is just a number converted to a string
	idNum, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing smee Event ID: %w", err)
	}

	// Parse JSON data into array - what we want is item 'body' which is the HTTP body
	// There are also query parameters in 'query' and the rest are the names and values of the HTTP headers
	dataMap := make(map[string]interface{})
	err = json.Unmarshal(data, &dataMap)
	if err != nil {
		return nil, fmt.Errorf("error parsing smee data: %w", err)
	}

	// Extract query parameters (TODO: parse these into something more useful)
	query := dataMap["query"]

	// Put the body back into JSON
	bodyObj := dataMap["body"]
	body, err := json.Marshal(bodyObj)
	if err != nil {
		return nil, fmt.Errorf("error marshalling smee body data: %w", err)
	}

	// What's left are the header values
	headers := make(map[string]string)
	for key, value := range dataMap {
		if key != "body" && key != "query" && key != "timestamp" {
			valueStr, isStr := value.(string)
			if isStr {

				headers[key] = valueStr
			} else {
				valueByteArray, isByteArray := value.([]byte)
				if isByteArray {
					headers[key] = string(valueByteArray)
				} else {
					// Just log and ignore unknown types
					s.Warnf("Unknown type in data: %v (key %q, value %v)", reflect.TypeOf(value), key, value)
				}
			}
		}
	}

	return &SmeeNotification{
		SmeeID:      idNum,
		Headers:     headers,
		QueryParams: query,
		Body:        body,
	}, nil
}
