package bb_server

import (
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type BBAPIServerConfig struct {
	server.HTTPServerConfig
}

// BBAPIServer is an HTTP server running locally with bb, to service requests from dynamic API clients
// and other code executed during local builds.
type BBAPIServer struct {
	server.APIServer
}

func NewBBAPIServer(
	bbAPI *BBAPIRouter,
	config BBAPIServerConfig,
	httpServerFactory server.HTTPServerFactory,
	logFactory logger.LogFactory,
) (*BBAPIServer, error) {
	localHTTPServer, err := httpServerFactory(bbAPI, config.HTTPServerConfig, logFactory("BBAPIServer-Local"))
	if err != nil {
		return nil, fmt.Errorf("error creating local HTTP server: %w", err)
	}
	return &BBAPIServer{
		APIServer: localHTTPServer,
	}, nil
}

type BBAPIRouter struct {
	chi.Router
}

func NewBBAPIRouter(
	log *server.LogAPI,
	artifact server.ArtifactAPIDynamic,
	build *server.BuildAPI,
	job *server.JobAPI,
	dynamicJobAPI server.DynamicJobAPIDynamic,
	root *server.RootAPI,
	authenticationService services.AuthenticationService,
	logFactory logger.LogFactory,
) *BBAPIRouter {

	logger := logFactory("BBAPIRouter").
		WithField("version", "v1")

	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger, NoColor: true})
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(6, "gzip"))
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			// Public routes that can be accessed without auth
			r.Group(func(r chi.Router) {
				r.Get("/", root.GetRootDocument)
			})

			// Routes for Dynamic API clients to interact with; note this includes some read-only API functions
			r.Group(server.DynamicJobAPIRouterFactory(dynamicJobAPI, build, job, artifact, log, authenticationService, logFactory))
		})
	})
	return &BBAPIRouter{Router: r}
}
