package bb_server

import (
	"net/http"

	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/local_backend"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
)

type DynamicJobAPIProxy struct {
	realAPI      *server.DynamicJobAPI
	localBackend *local_backend.LocalBackend
	logger.Log
}

func NewDynamicJobAPIProxy(
	realAPI *server.DynamicJobAPI,
	localBackend *local_backend.LocalBackend,
	logFactory logger.LogFactory,
) *DynamicJobAPIProxy {
	return &DynamicJobAPIProxy{
		realAPI:      realAPI,
		localBackend: localBackend,
		Log:          logFactory("DynamicJobAPIProxy"),
	}
}

func (a *DynamicJobAPIProxy) Ping(w http.ResponseWriter, r *http.Request) {
	a.realAPI.Ping(w, r)
}

// CreateJobs creates a new set of jobs and adds them to the build dynamically.
func (a *DynamicJobAPIProxy) CreateJobs(w http.ResponseWriter, r *http.Request) {
	newJobs := a.realAPI.CreateAndReturnJobs(w, r)

	// newJobs will be nil if an error occurred
	if len(newJobs) > 0 {
		a.localBackend.NewJobsCreated(r.Context(), newJobs)
	}
}
