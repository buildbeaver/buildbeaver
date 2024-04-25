package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/go-chi/render"
)

type DynamicJobAPI struct {
	queueService services.QueueService
	*APIBase
}

type NewJobList struct {
	Jobs []*models.Job
}

func NewDynamicJobAPI(
	authorizationService services.AuthorizationService,
	queueService services.QueueService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory,
) *DynamicJobAPI {
	return &DynamicJobAPI{
		queueService: queueService,
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("RunnerAPI")),
	}
}

func (a *DynamicJobAPI) Ping(w http.ResponseWriter, r *http.Request) {
	a.Infof("Got call to Ping from dynamic build job")
	w.WriteHeader(http.StatusOK)
}

// CreateJobs creates a new set of jobs and adds them to the build dynamically.
func (a *DynamicJobAPI) CreateJobs(w http.ResponseWriter, r *http.Request) {
	a.CreateAndReturnJobs(w, r)
}

// CreateAndReturnJobs creates a new set of jobs and adds them to the build dynamically.
// It returns the job graphs for the newly created jobs both in the HTTP response and as an object.
func (a *DynamicJobAPI) CreateAndReturnJobs(w http.ResponseWriter, r *http.Request) []*documents.JobGraph {
	a.Tracef("CreateJobs called (dynamic build)")
	buildID, err := a.AuthorizedBuildID(r, models.JobCreateOperation)
	if err != nil {
		a.Error(w, r, err)
		return nil
	}

	// Determine the content type, which must be one of the types supported by our custom parser.
	// Note that render.ContentTypeForm (i.e. HTML form data) is not supported by our parser.
	var configType models.ConfigType
	switch render.GetRequestContentType(r) {
	case render.ContentTypeJSON:
		configType = models.ConfigTypeJSON
	default:
		a.Error(w, r, gerror.NewErrValidationFailed(fmt.Sprintf("error: unable to decode request with content type %s", r.Header.Get("Content-Type"))))
		return nil
	}

	configBytes, err := io.ReadAll(r.Body)
	if err != nil {
		a.Error(w, r, err)
		return nil
	}

	_, newJobs, err := a.queueService.AddConfigToBuild(r.Context(), nil, buildID, configBytes, configType)
	if err != nil {
		a.Error(w, r, err)
		return nil
	}
	a.Infof("Added %d dynamic jobs definitions from new configuration for build '%s'", len(newJobs), buildID)

	// Return a list of new jobs added to the build
	newJobList := documents.MakeJobGraphs(routes.RequestCtx(r), newJobs)
	a.JSON(w, r, newJobList)

	return newJobList
}
