package util

import (
	"context"
	"sync"

	"github.com/buildbeaver/buildbeaver/common/logger"
)

// StatefulService provides standard service lifecycle routines (start/stop) functionality for long-lived
// services that run background threads.
type StatefulService struct {
	mu        sync.Mutex
	started   bool
	ctx       context.Context
	ctxCancel context.CancelFunc
	doneC     chan struct{}
	fn        func()
	log       logger.Log
}

func NewStatefulService(ctx context.Context, log logger.Log, fn func()) *StatefulService {
	ctx, cancel := context.WithCancel(ctx)
	s := &StatefulService{
		ctx:       ctx,
		ctxCancel: cancel,
		doneC:     make(chan struct{}),
		fn:        fn,
		log:       log,
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
		s.log.Panic("start can only be called once")
	}
	s.started = true
	s.log.Info("Starting...")
	go func() {
		defer close(s.doneC)
		s.log.Info("Started")
		s.fn()
		// TODO if fn() exits for an out of band reason, we need to Stop()
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
	s.log.Info("Stopping...")
	s.ctxCancel()
	<-s.doneC
	s.log.Info("Stopped")
}
