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

type StepAPI struct {
	stepService  services.StepService
	queueService services.QueueService
	*APIBase
}

func NewStepAPI(
	stepService services.StepService,
	queueService services.QueueService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *StepAPI {
	return &StepAPI{
		stepService:  stepService,
		queueService: queueService,
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("StepAPI")),
	}
}

func (a *StepAPI) Patch(w http.ResponseWriter, r *http.Request) {
	stepID, err := a.AuthorizedStepID(r, models.BuildUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	req := &documents.PatchStepRequest{}
	err = render.Bind(r, req)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	var step *models.Step
	if req.Status != nil {
		step, err = a.queueService.UpdateStepStatus(r.Context(), nil, stepID, dto.UpdateStepStatus{
			Status: *req.Status,
			Error:  req.Error,
			ETag:   a.GetIfMatch(r),
		})
		if err != nil {
			a.Error(w, r, err)
			return
		}
	}
	res := documents.MakeStep(routes.RequestCtx(r), step)
	a.UpdatedResource(w, r, res, nil)
}
