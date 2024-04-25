package server

import (
	"net/http"
	"time"

	"github.com/buildbeaver/buildbeaver/common/logger"
	bbmiddleware "github.com/buildbeaver/buildbeaver/server/api/rest/middleware"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// DynamicJobAPIDynamic is the subset of methods of DynamicJobAPI that are referenced by the dynamic job API router.
type DynamicJobAPIDynamic interface {
	Ping(w http.ResponseWriter, r *http.Request)
	CreateJobs(w http.ResponseWriter, r *http.Request)
}

// ArtifactAPIDynamic is the subset of methods of ArtifactAPI that are referenced by the dynamic job API router.
type ArtifactAPIDynamic interface {
	Get(w http.ResponseWriter, r *http.Request)
	GetData(w http.ResponseWriter, r *http.Request)
	List(w http.ResponseWriter, r *http.Request)
}

// DynamicJobAPIRouterFactory makes and returns a chi.Router function that can be slotted into an exiting
// chi router, to provide the routes for all Dynamic Job API functions.
// Routes are specified relative to "/api/v1/" and assume that the returned router will be slotted in
// under that route using r.Group().
func DynamicJobAPIRouterFactory(
	dynamicJobAPI DynamicJobAPIDynamic,
	build *BuildAPI,
	job *JobAPI,
	artifact ArtifactAPIDynamic,
	log *LogAPI,
	authenticationService services.AuthenticationService,
	logFactory logger.LogFactory,
) func(r chi.Router) {
	// Assume that middleware.DefaultLogger has already been set by the caller
	logger := logFactory("DynamicJobAPI").
		WithField("version", "v1")

	return func(r chi.Router) {
		// Routes for dynamic job clients to interact with are authenticated using JWT tokens.
		// No session authentication is allowed.
		r.Route("/dynamic", func(r chi.Router) {
			r.Use(middleware.Timeout(30 * time.Second))
			r.Use(bbmiddleware.MakeJWTAuthenticator(logger, authenticationService))
			r.Use(bbmiddleware.MakeMustAuthenticate(logger))

			r.Get("/ping", dynamicJobAPI.Ping)

			r.Route("/builds/{build_id}", func(r chi.Router) {
				r.Get("/", build.Get)
				r.Route("/artifacts", func(r chi.Router) {
					r.Get("/", artifact.List)
				})
				r.Post("/jobs", dynamicJobAPI.CreateJobs) // only available to dynamic builds
				r.Get("/events", build.GetEvents)
			})
			r.Route("/jobs/{job_id}", func(r chi.Router) {
				r.Get("/", job.Get)
				r.Get("/graph", job.GetGraph)
			})
			r.Route("/artifacts/{artifact_id}", func(r chi.Router) {
				r.Get("/", artifact.Get)
				r.Get("/data", artifact.GetData)
			})
			r.Route("/logs/{log_descriptor_id}", func(r chi.Router) {
				r.Get("/", log.Get)
				r.Get("/data", log.GetData)
			})
		})
	}
}
