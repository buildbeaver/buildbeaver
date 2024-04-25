package server

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	bbmiddleware "github.com/buildbeaver/buildbeaver/server/api/rest/middleware"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type AppAPIServerConfig struct {
	HTTPServerConfig
}

type AppAPIServer struct {
	APIServer
}

func NewAppAPIServer(coreAPI *AppAPIRouter, config AppAPIServerConfig, httpServerFactory HTTPServerFactory, logFactory logger.LogFactory) (*AppAPIServer, error) {
	httpServer, err := httpServerFactory(coreAPI, config.HTTPServerConfig, logFactory("AppAPIServer"))
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP server: %w", err)
	}
	return &AppAPIServer{
		APIServer: httpServer,
	}, nil
}

type AppAPIRouter struct {
	chi.Router
}

func NewAppAPIRouter(
	log *LogAPI,
	authentication *CoreAuthenticationAPI,
	secret *SecretAPI,
	artifact *ArtifactAPI,
	webhook *WebhookAPI,
	legalEntity *LegalEntityAPI,
	repo *RepoAPI,
	build *BuildAPI,
	job *JobAPI,
	step *StepAPI,
	runner *RunnerAPI,
	search *SearchAPI,
	dynamicJobAPI *DynamicJobAPI,
	tokenExchange *TokenExchangeAPI,
	root *RootAPI,
	authenticationService services.AuthenticationService,
	logFactory logger.LogFactory) *AppAPIRouter {

	logger := logFactory("AppAPIRouter").
		WithField("version", "v1")

	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger, NoColor: true})
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(6))
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(r chi.Router) {

		// TODO should only be enabled on debug builds
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:3001", "http://127.0.0.1:3001"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link", "Id", "Location"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))

		r.Route("/v1", func(r chi.Router) {
			// Public routes that can be accessed without auth
			r.Group(func(r chi.Router) {
				r.Get("/", root.GetRootDocument)
				r.Get("/authentication/github", authentication.AuthenticateGitHub)
				r.Get("/authentication/github/callback", authentication.AuthenticateGitHubCallback)
				// Public routes for webhooks to go to - each SCM provides its own authentication
				r.Route("/webhooks", func(r chi.Router) {
					r.Post("/{scm}", webhook.HandleWebhook)
				})
				r.Post("/token-exchange", tokenExchange.Exchange)
			})

			// Routes for Dynamic API clients to interact with; note this includes some read-only API functions
			r.Group(DynamicJobAPIRouterFactory(dynamicJobAPI, build, job, artifact, log, authenticationService, logFactory))

			// Routes for API clients to interact with are authenticated using sessions
			r.Group(func(r chi.Router) {
				r.Use(authentication.SessionAuthenticator)
				r.Use(bbmiddleware.MakeSharedSecretAuthenticator(logger, authenticationService))
				r.Use(bbmiddleware.MakeMustAuthenticate(logger))

				r.Route("/legal-entities", func(r chi.Router) {
					r.Get("/", legalEntity.List)
					r.Route("/{legal_entity_id}", func(r chi.Router) {
						r.Get("/", legalEntity.Get)
						r.Get("/setup-status", legalEntity.GetSetupStatus)
						r.Route("/repos", func(r chi.Router) {
							r.Get("/", repo.List)
							r.Post("/search", repo.Search)
						})
						r.Route("/builds", func(r chi.Router) {
							r.Get("/summary", build.Summary)
						})
						r.Route("/runners", func(r chi.Router) {
							r.Get("/", runner.List)
							r.Post("/", runner.Create)
							r.Post("/search", runner.Search)
						})
					})
				})
				r.Route("/repos/{repo_id}", func(r chi.Router) {
					r.Get("/", repo.Get)
					r.Patch("/", repo.Patch)
					r.Route("/builds", func(r chi.Router) {
						r.Get("/", build.List)
						r.Post("/", build.Create)
						r.Post("/search", build.Search)
					})
					r.Route("/secrets", func(r chi.Router) {
						r.Get("/", secret.List)
						r.Post("/", secret.Create)
					})
				})
				r.Route("/runners/{runner_id}", func(r chi.Router) {
					r.Get("/", runner.Get)
					r.Patch("/", runner.Patch)
					r.Delete("/", runner.Delete)
				})
				r.Route("/builds/{build_id}", func(r chi.Router) {
					r.Get("/", build.Get)
					r.Route("/artifacts", func(r chi.Router) {
						r.Get("/", artifact.List)
						r.Post("/search", artifact.Search)
					})
					r.Get("/events", build.GetEvents)
				})
				r.Route("/artifacts/{artifact_id}", func(r chi.Router) {
					r.Get("/", artifact.Get)
					r.Get("/data", artifact.GetData)
				})
				r.Route("/logs/{log_descriptor_id}", func(r chi.Router) {
					r.Get("/", log.Get)
					r.Get("/data", log.GetData)
				})
				r.Route("/secrets/{secret_id}", func(r chi.Router) {
					r.Get("/", secret.Get)
					r.Patch("/", secret.Patch)
					r.Delete("/", secret.Delete)
				})
				r.Route("/jobs/{job_id}", func(r chi.Router) {
					r.Get("/", job.Get)
					r.Get("/graph", job.GetGraph)
					r.Patch("/", job.Patch)
				})
				r.Route("/steps/{step_id}", func(r chi.Router) {
					r.Patch("/", step.Patch)
				})
				r.Route("/user", func(r chi.Router) {
					r.Get("/", legalEntity.GetCurrent)
				})
				r.Route("/users", func(r chi.Router) {
					r.Route("/{legal_entity_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
						r.Get("/", legalEntity.Get)
						r.Route("/repos", func(r chi.Router) {
							r.Get("/", repo.List)
							r.Post("/search", repo.Search)
						})
						r.Route("/runners", func(r chi.Router) {
							r.Get("/", runner.List)
							r.Post("/", runner.Create)
							r.Post("/search", runner.Search)
							r.Route("/{runner_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
								r.Get("/", runner.Get)
								r.Patch("/", runner.Patch)
							})
						})
					})
				})
				r.Route("/orgs", func(r chi.Router) {
					r.Get("/", legalEntity.List)
					r.Route("/{legal_entity_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
						r.Get("/", legalEntity.Get)
						r.Route("/repos", func(r chi.Router) {
							r.Get("/", repo.List)
							r.Post("/search", repo.Search)
						})
						r.Route("/runners", func(r chi.Router) {
							r.Get("/", runner.List)
							r.Post("/", runner.Create)
							r.Post("/search", runner.Search)
							r.Route("/{runner_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
								r.Get("/", runner.Get)
								r.Patch("/", runner.Patch)
							})
						})
					})
				})
				r.Route("/repos/{legal_entity_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
					r.Route("/{repo_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
						r.Get("/", repo.Get)
						r.Patch("/", repo.Patch)
						r.Route("/builds", func(r chi.Router) {
							r.Route("/{build_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
								r.Get("/", build.Get)
							})
						})
						r.Route("/secrets", func(r chi.Router) {
							r.Get("/", secret.List)
							r.Post("/", secret.Create)
							r.Route("/{secret_name:"+models.ResourceNameRegexStr+"}", func(r chi.Router) {
								r.Get("/", secret.Get)
								r.Patch("/", secret.Patch)
								r.Delete("/", secret.Delete)
							})
						})
					})
				})
				r.Route("/search", func(r chi.Router) {
					r.Get("/", search.List)
					r.Post("/", search.Search)
				})
			})
		})
	})
	return &AppAPIRouter{Router: r}
}
