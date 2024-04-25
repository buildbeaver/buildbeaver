package bb_server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/local_backend"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type ArtifactAPIProxy struct {
	realAPI         *server.ArtifactAPI
	artifactService services.ArtifactService
	localBackend    *local_backend.LocalBackend
	*server.APIBase
}

func NewArtifactAPIProxy(
	realAPI *server.ArtifactAPI,
	localBackend *local_backend.LocalBackend,
	artifactService services.ArtifactService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory,
) *ArtifactAPIProxy {
	return &ArtifactAPIProxy{
		realAPI:         realAPI,
		localBackend:    localBackend,
		artifactService: artifactService,
		APIBase:         server.NewAPIBase(authorizationService, resourceLinker, logFactory("ArtifactAPIProxy")),
	}
}

func (a *ArtifactAPIProxy) Get(w http.ResponseWriter, r *http.Request) {
	a.realAPI.Get(w, r)
}

// GetData processes an artifact GetData API request by reading the artifact data from the local filesystem,
// assuming the current local build has already created the artifact there.
func (a *ArtifactAPIProxy) GetData(w http.ResponseWriter, r *http.Request) {
	artifactID, err := a.AuthorizedArtifactID(r, models.ArtifactReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	artifact, err := a.artifactService.Read(r.Context(), nil, artifactID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	_, file := filepath.Split(artifact.Path)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Read data from the local file system
	a.Infof("Reading artifact data from local filesystem...")
	reader, err := a.localBackend.GetArtifactLocalData(artifact)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	defer reader.Close()

	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, reader)
	if err != nil {
		a.Errorf("error writing artifact data to response body: %w", err)
	}
}

func (a *ArtifactAPIProxy) List(w http.ResponseWriter, r *http.Request) {
	a.realAPI.List(w, r)
}
