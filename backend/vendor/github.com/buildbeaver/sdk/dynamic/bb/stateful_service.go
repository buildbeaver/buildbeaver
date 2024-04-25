package bb

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"
)

// StatefulService provides standard service lifecycle routines (start/stop) functionality for long-lived
// services that run background threads.
type StatefulService struct {
	serviceName string
	fn          func()
	log         Logger

	mu        sync.Mutex
	started   bool
	ctx       context.Context
	ctxCancel context.CancelFunc
	doneC     chan struct{}
}

func NewStatefulService(ctx context.Context, log Logger, serviceName string, fn func()) *StatefulService {
	ctx, cancel := context.WithCancel(ctx)
	s := &StatefulService{
		serviceName: serviceName,
		fn:          fn,
		log:         log,
		ctx:         ctx,
		ctxCancel:   cancel,
		doneC:       make(chan struct{}),
	}
	return s
}

// Ctx returns the service's context.
func (s *StatefulService) Ctx() context.Context {
	return s.ctx
}

// Done can be used to wait for the service to stop.
func (s *StatefulService) Done() <-chan struct{} {
	return s.doneC
}

// Start the service. Panics if called more than once.
func (s *StatefulService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.log(LogLevelFatal, "start can only be called once")
		panic("start ")
	}
	s.started = true
	s.log(LogLevelDebug, fmt.Sprintf("Starting '%s' service...", s.serviceName))
	go func() {
		defer close(s.doneC)
		s.log(LogLevelDebug, fmt.Sprintf("Started '%s' service", s.serviceName))
		s.fn()
	}()
}

// Stop the service. Blocks until the service has cleaned up all background threads and exited.
// This function is idempotent.
func (s *StatefulService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	s.log(LogLevelDebug, fmt.Sprintf("Stopping '%s' service...", s.serviceName))
	s.ctxCancel()
	<-s.doneC
	s.log(LogLevelDebug, fmt.Sprintf("Stopped '%s' service", s.serviceName))
}
