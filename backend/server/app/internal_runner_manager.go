package app

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/app"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
)

// maxInternalRunners is the maximum number of legal entities to start internal runners for.
// This should be a reasonable number for developers but not so many as to cripple a development server.
const maxInternalRunners = 100

const pollForNewRunnersInterval = 5 * time.Second

// InternalRunnerConfigDirectory is a file path specifying the parent directory for configuration files
// for internal runners started within the server.
type InternalRunnerConfigDirectory string

func (d InternalRunnerConfigDirectory) String() string {
	return string(d)
}

type InternalRunnerConfig struct {
	ConfigDir            InternalRunnerConfigDirectory
	StartInternalRunners bool
	DynamicAPIEndpoint   dynamic_api.Endpoint
}

type InternalRunnerManager struct {
	legalEntityService services.LegalEntityService
	runnerService      services.RunnerService
	runnerAPIServer    *server.RunnerAPIServer
	config             InternalRunnerConfig

	startStopMutex sync.Mutex
	exitChan       chan bool
	wg             sync.WaitGroup // to wait until polling loop exits

	// allRunners maps legal entity ID to the runner for that legal entity,
	// This field is on accessed by a single goroutine at once (Start(), then pollLoop(), then Stop())
	allRunners map[models.LegalEntityID]*app.Runner

	logger.Log
}

func NewInternalRunnerManager(
	legalEntityService services.LegalEntityService,
	runnerService services.RunnerService,
	runnerAPIServer *server.RunnerAPIServer,
	config InternalRunnerConfig,
	logFactory logger.LogFactory,
) *InternalRunnerManager {
	return &InternalRunnerManager{
		legalEntityService: legalEntityService,
		runnerService:      runnerService,
		runnerAPIServer:    runnerAPIServer,
		config:             config,
		allRunners:         make(map[models.LegalEntityID]*app.Runner),
		Log:                logFactory("InternalRunnerManager"),
	}
}

func (m *InternalRunnerManager) Start() error {
	m.startStopMutex.Lock()
	defer m.startStopMutex.Unlock()

	if m.exitChan != nil {
		return nil
	}
	m.Info("Starting internal runners...")

	// Start initial set of internal runners in-line and don't start the polling goroutine if there's a problem
	// To keep things single-threaded, do not start pollLoop until after this call is complete
	err := m.startNewRunners()
	if err != nil {
		return err
	}

	m.Trace("Starting internal runner poll loop...")
	m.exitChan = make(chan bool)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.pollLoop()
	}()

	return nil
}

func (m *InternalRunnerManager) Stop() {
	m.startStopMutex.Lock()
	defer m.startStopMutex.Unlock()

	if m.exitChan == nil {
		return
	}

	m.Info("Stopping internal runners...")

	// Stop polling
	close(m.exitChan)
	m.wg.Wait()
	m.exitChan = nil

	// Once the polling goroutine has stopped then stopAllRunners() can safely access the allRunners member variable
	m.stopAllRunners()
}

func (m *InternalRunnerManager) pollLoop() {
	for {
		select {
		case <-m.exitChan:
			m.Trace("Exiting internal runner poll loop...")
			return

		case <-time.After(pollForNewRunnersInterval):
			m.Trace("Checking for new legal entities to start internal runners")
			err := m.startNewRunners()
			if err != nil {
				m.Errorf(err.Error()) // Log and ignore errors
			}
		}
	}
}

// startNewRunners checks for legal entities that do not currently have an internal runner, and starts
// a new internal runner for each one. No more than maxInternalRunners will be run.
func (m *InternalRunnerManager) startNewRunners() error {
	ctx := context.TODO()

	// Loop through up to the maximum number of legal entities, starting a runner for each one
	// TODO: Consider only starting runners for a nominated list of legal entities (from a command-line flag)
	legalEntities, cursor, err := m.legalEntityService.ListAllLegalEntities(ctx, nil, models.NewPagination(maxInternalRunners, nil))
	if err != nil {
		return fmt.Errorf("error checking for new legal entities to start runners: %w", err)
	}
	if cursor != nil && cursor.Next != nil {
		m.Infof("More than %d legal entities found; only starting runners for the first %d", maxInternalRunners, maxInternalRunners)
	}

	for _, legalEntity := range legalEntities {
		// We could hit the maximum number of runners from results from previous calls to ListAllLegalEntities
		if len(m.allRunners) >= maxInternalRunners {
			break
		}
		if _, runnerFound := m.allRunners[legalEntity.ID]; !runnerFound {
			runner, err := m.createRunner(legalEntity)
			if err != nil {
				return err
			}
			m.allRunners[legalEntity.ID] = runner

			m.Infof("Starting runner for %s", legalEntity.ID)
			err = runner.Start(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *InternalRunnerManager) stopAllRunners() {
	m.Tracef("Stopping %d internal runners...", len(m.allRunners))

	for legalEntityID, runner := range m.allRunners {
		m.Infof("Stopping runner for %s", legalEntityID)
		runner.Stop()
	}
	// Clear the list of runners after they have all been shut down
	m.allRunners = make(map[models.LegalEntityID]*app.Runner)
}

// Creates a new runner for the given legal entity. Does not start the runner.
func (m *InternalRunnerManager) createRunner(legalEntity *models.LegalEntity) (*app.Runner, error) {
	ctx := context.Background()
	now := models.NewTime(time.Now())

	// Create a suitable name based on the legal entity this runner is for
	runnerName := models.ResourceName(fmt.Sprintf("server-internal-runner-%s", models.SanitizeFilePathID(legalEntity.ID.ResourceID)))

	// Calculate file paths for a runner for this legal entity
	runnerConfigDir := filepath.Join(m.config.ConfigDir.String(), runnerName.String())
	runnerCertFile := certificates.CertificateFile(filepath.Join(runnerConfigDir, app.DefaultRunnerCertFile))
	runnerPrivateKeyFile := certificates.PrivateKeyFile(filepath.Join(runnerConfigDir, app.DefaultRunnerPrivateKeyFile))
	runnerLogTempDir := logging.RunnerLogTempDirectory(filepath.Join(runnerConfigDir, app.DefaultRunnerLogTempDirName))

	runnerConfig := &app.RunnerConfig{
		RunnerAPIEndpoints:    []string{m.runnerAPIServer.GetServerURL()},
		RunnerLogTempDir:      runnerLogTempDir,
		RunnerCertificateFile: runnerCertFile,
		RunnerPrivateKeyFile:  runnerPrivateKeyFile,
		AutoCreateCertificate: true,
		CACertFile:            "",
		InsecureSkipVerify:    true,
		LogUnregisteredCert:   false,
		SchedulerConfig: runner.SchedulerConfig{
			PollInterval: runner.DefaultPollInterval,
			ParallelJobs: runner.DefaultParallelBuilds,
		},
		ExecutorConfig: runner.ExecutorConfig{
			IsLocal:            false, // not running as BB; this is a genuine runner
			DynamicAPIEndpoint: m.config.DynamicAPIEndpoint,
		},
		LogLevels: "",
	}

	// Create an internal runner app; this will create a client certificate if required
	runnerApp, err := app.New(runnerConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating runner app: %w", err)
	}

	// Load client certificate for runner
	clientCert, err := certificates.LoadCertificateFromPemFile(runnerCertFile)
	if err != nil {
		return nil, fmt.Errorf("error loading client certificate for internal runner: %w", err)
	}

	// Look up the runner which may already have been registered, and register if required
	runner, err := m.runnerService.ReadByName(ctx, nil, legalEntity.ID, runnerName)
	if err != nil {
		if gerror.IsNotFound(err) {
			runner = nil
			err = nil
		} else {
			return nil, fmt.Errorf("error looking for internal runner registration: %w", err)
		}
	}
	if runner == nil {
		// Register the new runner with the server, using the client certificate for authentication
		runner = models.NewRunner(
			now,
			runnerName,
			legalEntity.ID,
			"(internal)",
			runtime.GOOS,
			runtime.GOARCH,
			nil, // this field gets updated when runner updates its runtime info
			nil, // no labels need to be specified
			true,
		)
		err = m.runnerService.Create(ctx, nil, runner, clientCert)
		if err != nil {
			return nil, fmt.Errorf("error creating runner: %w", err)
		}
		m.Infof("Internal runner registered for legal entity %q", legalEntity.ID)
	} else {
		m.Infof("Existing internal runner registration found for legal entity %q", legalEntity.ID)
	}

	return runnerApp, nil
}
