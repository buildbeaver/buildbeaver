package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

type LogAPI struct {
	logService   services.LogService
	buildService services.BuildService
	*APIBase
}

func NewLogAPI(
	logService services.LogService,
	buildService services.BuildService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory) *LogAPI {
	return &LogAPI{
		logService:   logService,
		buildService: buildService,
		APIBase:      NewAPIBase(authorizationService, resourceLinker, logFactory("LogAPI")),
	}
}

func (a *LogAPI) Get(w http.ResponseWriter, r *http.Request) {
	logID, err := a.AuthorizedLogDescriptorID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	log, err := a.logService.Read(r.Context(), nil, logID)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	res := documents.MakeLog(routes.RequestCtx(r), log)
	a.GotResource(w, r, res)
}

func (a *LogAPI) WriteData(w http.ResponseWriter, r *http.Request) {
	meta := a.MustAuthenticationMeta(r)
	if meta.CredentialType != models.CredentialTypeClientCertificate {
		panic("Expected runner to authenticate with a client certificate")
	}
	logDescriptorID, err := a.AuthorizedLogDescriptorID(r, models.BuildUpdateOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	// HTTP 1.1 can stream log entries to us by using chunked transfer encoding. HTTP 2.0 natively supports this.
	if !r.ProtoAtLeast(2, 0) {
		if len(r.TransferEncoding) == 0 || r.TransferEncoding[0] != "chunked" {
			// Return a 400 error so the client doesn't retry
			a.Error(w, r, gerror.NewErrValidationFailed("HTTP 1.1 must use chunked encoding"))
			return
		}
	}
	err = a.logService.WriteData(r.Context(), logDescriptorID, r.Body)
	if err != nil {
		a.Error(w, r, fmt.Errorf("error writing log: %w", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *LogAPI) GetData(w http.ResponseWriter, r *http.Request) {
	logID, err := a.AuthorizedLogDescriptorID(r, models.BuildReadOperation)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	queryValues := r.URL.Query()
	shouldDownload := false
	vals, ok := queryValues["download"]
	if ok && len(vals) > 0 {
		shouldDownload, err = strconv.ParseBool(vals[0])
		if err != nil {
			a.Error(w, r, gerror.NewErrInvalidQueryParameter("invalid value for 'download' query parameter"))
			return
		}
	}

	search := documents.NewLogSearchRequest()
	err = search.FromQuery(queryValues)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	// Write and flush headers before we write the data
	flusher, ok := w.(http.Flusher)
	if !ok {
		a.Error(w, r, fmt.Errorf("error response body does not support http.Flusher"))
		return
	}
	if shouldDownload {
		// Set the headers required to download the log as a file
		fileName := logID.GetFileName() + ".txt"
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
	} else {
		// Stream the log rather than downloading as a file
		if search.Plaintext != nil && *search.Plaintext {
			w.Header().Set("Content-Type", "text/plain")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		// HTTP 1.1 supports streaming writes via chunked transfer encoding.
		// We simply omit the Content-Length header and this gets wired up for
		// us automatically in the Go stdlib. We must flush after every write
		// though to produce the next chunk.
		// HTTP 2.0 natively supports streaming. Similarly, we should flush to
		// ensure timely delivery.
		if !r.ProtoAtLeast(2, 0) {
			w.Header().Set("Transfer-Encoding", "chunked")
			w.WriteHeader(http.StatusOK)
		}
	}
	flusher.Flush() // Flush the headers before writing data

	stream, err := a.logService.ReadData(r.Context(), logID, search.LogSearch)
	if err != nil {
		a.Error(w, r, err)
		return
	}
	defer stream.Close()
	_, err = io.Copy(util.NewFlushingWriter(w, flusher), stream)
	if err != nil {
		a.Errorf("Ignoring error writing log stream: %v", err)
	}
}
