package server

import (
	"net/http"

	"github.com/go-chi/render"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type JobAPI struct {
	jobService   services.JobService
	queueService services.QueueService
	*APIBase
}

func NewJobAPI(
	jobService services.JobService,
	queueService services.QueueService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *JobAPI {
	return &JobAPI{
		jobService:   jobService,
		queueService: queueService,
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("JobAPI")),
	}
}

func (a *JobAPI) Get(w http.ResponseWriter, r *http.Request) {
	jobID, err := a.AuthorizedJobID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	job, err := a.jobService.Read(r.Context(), nil, jobID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeJob(routes.RequestCtx(r), job)
	a.GotResource(w, r, res)
}

func (a *JobAPI) GetGraph(w http.ResponseWriter, r *http.Request) {
	jobID, err := a.AuthorizedJobID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	job, err := a.queueService.ReadJobGraph(r.Context(), nil, jobID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeJobGraph(routes.RequestCtx(r), job)
	a.GotResource(w, r, res)
}

func (a *JobAPI) Patch(w http.ResponseWriter, r *http.Request) {
	jobID, err := a.AuthorizedJobID(r, models.BuildUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.PatchJobRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	var job *models.Job
	if req.Status != nil {
		job, err = a.queueService.UpdateJobStatus(r.Context(), nil, jobID, dto.UpdateJobStatus{
			Status: *req.Status,
			Error:  req.Error,
			ETag:   a.GetIfMatch(r),
		})
		if err != nil {
			a.Error(w, r, err)
			return
		}
	} else if req.Fingerprint != nil {
		job, err = a.queueService.UpdateJobFingerprint(r.Context(), jobID,
			dto.UpdateJobFingerprint{
				Fingerprint:         *req.Fingerprint,
				FingerprintHashType: *req.FingerprintHashType,
				ETag:                a.GetIfMatch(r),
			})
		if err != nil {
			a.Error(w, r, err)
			return
		}
	}
	res := documents.MakeJob(routes.RequestCtx(r), job)
	a.UpdatedResource(w, r, res, nil)
}
