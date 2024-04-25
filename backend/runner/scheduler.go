package runner

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/version"
	runtime2 "github.com/buildbeaver/buildbeaver/runner/runtime"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

const (
	DefaultPollInterval   = time.Second * 5
	DefaultParallelBuilds = 0
	pollTimeout           = time.Second * 30
	buildTimeout          = time.Hour * 2
	// statusUpdateTimeout is the maximum time to spend trying to update job and step statuses. Keep trying for a while.
	statusUpdateTimeout = time.Minute * 5
	// cleanupTimeout is the maximum time to spend trying to clean up resources (docker containers etc)
	cleanupTimeout      = time.Minute * 15 // try pretty hard
	minimumParallelJobs = 2                // must be able to run a dynamic build job and at least one regular job
)

// getStatusUpdateContext returns a context with a timeout to use when updating job and step statuses.
func getStatusUpdateContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), statusUpdateTimeout)
}

// getCleanupContext returns a context with a timeout to use for cleanup operations.
func getCleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cleanupTimeout)
}

type SchedulerConfig struct {
	ParallelJobs int
	PollInterval time.Duration
}

type pollResult struct {
	err error
	job *documents.RunnableJob
}

type Scheduler struct {
	client              APIClient
	orchestratorFactory OrchestratorFactory
	pollResultChan      chan *pollResult
	jobCompleteC        chan bool
	mu                  sync.Mutex
	wg                  sync.WaitGroup
	state               struct {
		runtimeInfoSent    bool
		runningJobs        int
		lastBuildCompleted time.Time
		lastPollStarted    time.Time
		polling            bool
		exiting            bool
		exitChan           chan bool
		exitingWhenQuiet   bool
		exitWhenQuietChan  chan bool
	}
	config     SchedulerConfig
	stats      models.RunnerStats
	statsMutex sync.RWMutex
	log        logger.Log
}

func NewJobScheduler(
	client APIClient,
	orchestratorFactory OrchestratorFactory,
	logFactory logger.LogFactory,
	config SchedulerConfig,
) *Scheduler {

	log := logFactory("Scheduler")
	if config.ParallelJobs == 0 {
		config.ParallelJobs = runtime.NumCPU() / 2
		if config.ParallelJobs < minimumParallelJobs {
			config.ParallelJobs = minimumParallelJobs
		}
	}
	log.Infof("Using %d parallel jobs", config.ParallelJobs)

	return &Scheduler{
		client:              client,
		orchestratorFactory: orchestratorFactory,
		pollResultChan:      make(chan *pollResult),
		jobCompleteC:        make(chan bool),
		mu:                  sync.Mutex{},
		wg:                  sync.WaitGroup{},
		config:              config,
		log:                 log,
	}
}

// Start running queued builds.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.exitChan != nil {
		return
	}

	s.log.Info("Starting...")

	s.state.runningJobs = 0
	s.state.runtimeInfoSent = false
	s.state.polling = false
	s.state.exiting = false
	s.state.exitChan = make(chan bool)
	s.state.exitingWhenQuiet = false
	s.state.exitWhenQuietChan = make(chan bool)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.loop()
	}()
}

// Stop by immediately stopping to dequeue builds, and return after currently
// running builds have completed.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.exitChan == nil {
		return
	}

	s.log.Info("Exiting...")
	close(s.state.exitChan)
	s.wg.Wait()
	s.state.exitChan = nil
	s.state.exitWhenQuietChan = nil
}

// StopWhenQuiet stops when all builds are finished and there are no more to dequeue.
func (s *Scheduler) StopWhenQuiet() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.exitChan == nil {
		return
	}

	s.log.Info("Waiting for quiet period and then exiting...")
	close(s.state.exitWhenQuietChan)
	s.wg.Wait()
	s.state.exitChan = nil
	s.state.exitWhenQuietChan = nil
}

func (s *Scheduler) GetStats() *models.RunnerStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	statsCopy := s.stats
	return &statsCopy
}

func (s *Scheduler) loop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial startup poll
	s.poll(ctx)

	for {
		if s.state.exiting && !s.state.polling && s.state.runningJobs == 0 {
			s.log.Info("All jobs complete; Exiting")
			return
		}

		if s.state.exitingWhenQuiet && !s.state.polling && s.state.runningJobs == 0 && s.state.lastPollStarted.After(s.state.lastBuildCompleted) {
			s.log.Info("All jobs complete and queue is empty; Exiting")
			return
		}

		var exitChan <-chan bool
		if !s.state.exiting {
			exitChan = s.state.exitChan
		}

		var exitWhenQuietChan <-chan bool
		if !s.state.exiting && !s.state.exitingWhenQuiet {
			exitWhenQuietChan = s.state.exitWhenQuietChan
		}

		// TODO proper timer
		var pollTimer <-chan time.Time
		if !s.state.exiting && !s.state.polling && s.state.runningJobs < s.config.ParallelJobs {
			pollTimer = time.After(s.config.PollInterval)
		}

		select {
		case <-exitChan:
			cancel()
			s.state.exiting = true
			s.log.Infof("Exit signal received; Waiting for %d jobs(s) to complete before exiting", s.state.runningJobs)
		case <-exitWhenQuietChan:
			s.state.exitingWhenQuiet = true
			s.log.Info("Exit signal received; Will exit when jobs queue is empty")
		case <-pollTimer:
			if !s.state.polling {
				s.poll(ctx)
			}
		case res := <-s.pollResultChan:
			s.handlePollResult(ctx, res)
		case <-s.jobCompleteC:
			s.state.runningJobs--
			s.state.lastBuildCompleted = time.Now()
			if s.state.runningJobs < 0 {
				s.log.Panic("s.state.runningJobs < 0")
			}
			s.log.Infof("Job complete; %d jobs(s) now in progress", s.state.runningJobs)
			if !s.state.polling {
				s.log.Infof("More capacity available; Checking for more jobs to run")
				s.poll(ctx)
			}
		}
	}
}

func (s *Scheduler) poll(ctx context.Context) {
	if s.state.polling {
		s.log.Panic("Expected polling to be false")
	}
	s.state.lastPollStarted = time.Now()
	s.state.polling = true
	ctx, cancel := context.WithTimeout(ctx, pollTimeout)
	go func() {
		defer cancel()
		if s.state.runtimeInfoSent {
			job, err := s.client.Dequeue(ctx)
			s.pollResultChan <- &pollResult{
				err: err,
				job: job,
			}
		} else {
			err := s.sendRuntimeInfo(ctx)
			s.pollResultChan <- &pollResult{
				err: err,
			}
		}
	}()
}

func (s *Scheduler) handlePollResult(ctx context.Context, res *pollResult) {
	if !s.state.polling {
		s.log.Panic("Expected polling to be true")
	}
	s.state.polling = false
	if res.err != nil {
		s.recordFailedPoll()
		if !gerror.IsNotFound(res.err) && !gerror.IsRunnerDisabled(res.err) {
			s.log.Errorf("Will retry error during poll: %s", res.err)
		}
		return
	}
	s.recordSuccessfulPoll()
	if !s.state.runtimeInfoSent {
		s.state.runtimeInfoSent = true
		s.poll(ctx) // do the first poll straight away
	}
	if res.job != nil {
		s.state.runningJobs++
		if s.state.runningJobs > s.config.ParallelJobs {
			s.log.Panicf("s.state.runningJobs > %d", s.config.ParallelJobs)
		}
		s.log.Infof("Running job %s; %d jobs(s) now in progress", res.job.Job.ID, s.state.runningJobs)
		go func() {
			runner := s.orchestratorFactory()
			runner.Run(res.job)
			s.jobCompleteC <- true
		}()
		if s.state.runningJobs < s.config.ParallelJobs {
			s.log.Infof("More capacity available; Checking for more jobs to run")
			s.poll(ctx)
		}
	}
}

func (s *Scheduler) recordSuccessfulPoll() {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	s.stats.SuccessfulPollCount++
}

func (s *Scheduler) recordFailedPoll() {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	s.stats.FailedPollCount++
}

func (s *Scheduler) sendRuntimeInfo(ctx context.Context) error {
	var (
		os                = string(runtime2.GetHostOS())
		arch              = runtime.GOARCH
		softwareVersion   = version.VERSION
		supportedJobKinds = models.JobTypes{models.JobTypeDocker, models.JobTypeExec}
	)
	info := &documents.PatchRuntimeInfoRequest{
		SoftwareVersion:   &softwareVersion,
		OperatingSystem:   &os,
		Architecture:      &arch,
		SupportedJobTypes: &supportedJobKinds,
	}
	err := s.client.SendRuntimeInfo(ctx, info)
	if err != nil {
		return err
	}
	s.log.Infof("Sent runtime info to server: Software version: %s, Operating System: %s, Architecture: %s, Supported Job Types: %v\n",
		softwareVersion, os, arch, supportedJobKinds)
	return nil
}
