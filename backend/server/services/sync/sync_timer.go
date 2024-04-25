package sync

import (
	"time"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/store"
	"golang.org/x/net/context"
)

// defaultSyncTimerInterval determines how often a global sync will be run. This should be a shorter time period than
// DefaultFullSyncAfter, so that partial sync is run more frequently and a full sync happens for each Legal Entity
// relatively soon after the minimum interval has elapsed.
const defaultSyncTimerInterval = 1 * time.Hour

// initialSyncDelay is the delay after startup before the sync timer runs an initial global sync.
// This delay gives time for SCM registration to occur so that the initial sync will succeed.
const initialSyncDelay = 1 * time.Minute

// SyncTimer implements a Service to periodically sync with external systems.
type SyncTimer struct {
	*util.StatefulService
	db                *store.DB
	syncService       services.SyncService
	syncTimerInterval time.Duration
	logger.Log
}

func NewSyncTimer(
	db *store.DB,
	syncService services.SyncService,
	logFactory logger.LogFactory,
) *SyncTimer {
	s := &SyncTimer{
		db:                db,
		syncService:       syncService,
		syncTimerInterval: defaultSyncTimerInterval,
		Log:               logFactory("SyncTimer"),
	}
	s.StatefulService = util.NewStatefulService(context.Background(), s.Log, s.loop)
	return s
}

func (s *SyncTimer) loop() {
	s.Tracef("Running sync timer loop function...")
	time.Sleep(initialSyncDelay)
	s.Infof("Performing initial sync...")
	s.doSync()
	s.Tracef("Starting sync timer loop...")
	for {
		select {
		case <-s.StatefulService.Ctx().Done():
			s.Infof("Sync timer service closed; exiting...")
			return

		case <-time.After(s.syncTimerInterval):
			s.doSync()
		}
	}
}

// Performs all periodic sync operations.
func (s *SyncTimer) doSync() {
	// Set an overall timeout on the global sync
	ctx, cancel := context.WithTimeout(context.Background(), DefaultGlobalSyncTimeout)
	defer cancel()

	// Sync with GitHub
	err := s.syncService.GlobalSync(ctx, github.GitHubSCMName, DefaultFullSyncAfter, DefaultPerLegalEntityTimeout)
	if err != nil {
		s.Errorf("Error performing global sync with SCM '%s': %s", github.GitHubSCMName, err.Error())
	}
}
