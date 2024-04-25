package server

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/buildbeaver/buildbeaver/common/logger"
	bbmiddleware "github.com/buildbeaver/buildbeaver/server/api/rest/middleware"
	"github.com/buildbeaver/buildbeaver/server/services"
)

const routerDefaultTimeout = 60 * time.Second

type RunnerAPIServerConfig struct {
	HTTPServerConfig
}

type RunnerAPIServer struct {
	APIServer
}

func NewRunnerAPIServer(runnerAPI *RunnerAPIRouter, config RunnerAPIServerConfig, httpServerFactory HTTPServerFactory, logFactory logger.LogFactory) (*RunnerAPIServer, error) {
	config.TLSConfig.UseMTLS = true
	httpServer, err := httpServerFactory(runnerAPI, config.HTTPServerConfig, logFactory("RunnerAPIServer"))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP server: %w", err)
	}
	return &RunnerAPIServer{
		APIServer: httpServer,
	}, nil
}

type RunnerAPIRouter struct {
	chi.Router
}

func NewRunnerAPIRouter(
	queue *QueueAPI,
	log *LogAPI,
	secret *SecretAPI,
	artifact *ArtifactAPI,
	job *JobAPI,
	step *StepAPI,
	runner *RunnerAPI,
	authenticationService services.AuthenticationService,
	logFactory logger.LogFactory) *RunnerAPIRouter {

	logger := logFactory("RunnerAPI").
		WithField("version", "v1")

	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger, NoColor: true})
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(6))

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			// Routes for runners to interact with are authenticated using client certificates via TLS mutual auth;
			// no session authentication is allowed.
			r.Route("/runner", func(r chi.Router) {
				r.Use(bbmiddleware.MakeClientCertificateAuthenticator(logger, authenticationService))
				r.Use(bbmiddleware.MakeMustAuthenticate(logger))

				// This group contains routes that all want the default timeout value
				r.Group(func(r chi.Router) {
					r.Use(middleware.Timeout(routerDefaultTimeout))

					r.Get("/ping", queue.Ping)
					r.Patch("/runtime", runner.PatchRuntimeInfo)
					r.Get("/queue", queue.Dequeue)
					r.Route("/repos/{repo_id}", func(r chi.Router) {
						r.Route("/secrets", func(r chi.Router) {
							r.Get("/", secret.ListPlainText)
						})
					})
					r.Route("/builds/{build_id}", func(r chi.Router) {
						r.Route("/artifacts", func(r chi.Router) {
							r.Get("/", artifact.List)
							r.Post("/search", artifact.Search)
						})
					})
					r.Route("/steps/{step_id}", func(r chi.Router) {
						r.Patch("/", step.Patch)
					})
					r.Route("/artifacts/{artifact_id}", func(r chi.Router) {
						r.Get("/data", artifact.GetData)
					})
				})

				r.Route("/jobs/{job_id}", func(r chi.Router) {
					// This group contains routes that all want the default timeout value
					r.Group(func(r chi.Router) {
						r.Use(middleware.Timeout(routerDefaultTimeout))
						r.Patch("/", job.Patch)
					})

					r.Route("/artifacts", func(r chi.Router) {
						r.Use(middleware.Timeout(5 * time.Minute)) // extra long timeout for posting artifacts
						r.Post("/", artifact.Create)
					})
				})

				r.Route("/logs/{log_descriptor_id}", func(r chi.Router) {
					// This group contains routes that all want the default timeout value
					r.Group(func(r chi.Router) {
						r.Use(middleware.Timeout(routerDefaultTimeout))
						r.Get("/data", log.GetData)
					})

					r.Group(func(r chi.Router) {
						// allow clients to stream log data for a longer-than-standard time to allow clients the
						// option to hold the connection open
						r.Use(middleware.Timeout(5 * time.Minute))
						r.Post("/data", log.WriteData)
					})
				})
			})
		})
	})
	return &RunnerAPIRouter{Router: r}
}
