package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
	"github.com/buildbeaver/buildbeaver/server/api/rest/middleware"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
)

const AuthenticationMetaContextKeyName = "authentication"

type APIBase struct {
	logger.Log
	resourceLinker       *routes.ResourceLinker
	authorizationService services.AuthorizationService
}

func NewAPIBase(authorizationService services.AuthorizationService, resourceLinker *routes.ResourceLinker, logger logger.Log) *APIBase {
	return &APIBase{
		resourceLinker:       resourceLinker,
		authorizationService: authorizationService,
		Log:                  logger,
	}
}

// JSON marshals 'v' to JSON, automatically escaping HTML and setting the
// Content-Type as application/json. Copied from chi/render.JSON and updated
// to log serialization errors.
func (a *APIBase) JSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		a.Error(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	a.Tracef("JSON Response: %s", buf.String())
	w.Write(buf.Bytes())
}

// Error writes the specified error to the http response as a standard
// API error document. Errors are sanitized for public display before
// being written. Status code is automatically inferred from the error.
// The error is logged to the server log at a Warning level.
func (a *APIBase) Error(w http.ResponseWriter, r *http.Request, err error) {
	a.Warnf("Error in API call: %v", err)
	a.ErrorNotLogged(w, r, err)
}

// ErrorNotLogged writes the specified error to the http response as a standard
// API error document. Errors are sanitized for public display before
// being written. Status code is automatically inferred from the error.
// The error is not logged to the server log.
func (a *APIBase) ErrorNotLogged(w http.ResponseWriter, r *http.Request, err error) {

	// START Legacy support
	// TODO Convert these at the store layer.
	cause := errors.Cause(err)
	if cause == sql.ErrNoRows {
		err = gerror.NewErrNotFound("Resource not found")
	}
	pqErr, ok := cause.(*pq.Error)
	if ok {
		// https://www.postgresql.org/docs/current/static/errcodes-appendix.html
		if pqErr.Code == "23505" {
			err = gerror.NewErrAlreadyExists("Resource already exists").Wrap(err)
		}
	}
	// END Legacy support

	// Look down through the chain of wrapped errors, including errors wrapped using fmt.Errorf(), and
	// and find the first error which is a gerror.Error
	var gErr gerror.Error
	if !errors.As(err, &gErr) || gErr.Audience() != gerror.AudienceExternal {
		gErr = gerror.NewErrInternal()
	}
	doc := &documents.ErrorDocument{
		Code:           gErr.Code(),
		HTTPStatusCode: gErr.HTTPStatusCode(),
		Message:        gErr.Message(),
		Details:        make(map[gerror.DetailKey]interface{}),
	}
	for _, detail := range gErr.Details() {
		if detail.Audience() == gerror.AudienceExternal {
			doc.Details[detail.Key()] = detail.Value()
		}
	}
	r = r.WithContext(context.WithValue(r.Context(), render.StatusCtxKey, gErr.HTTPStatusCode()))
	a.JSON(w, r, doc)
}

// Created writes a standardized created response to the http response object.
// The ID, Location and ETag headers will be set if corresponding arguments are specified,
// and data (if set) will optionally be serialized to JSON and written in the response body.
func (a *APIBase) Created(w http.ResponseWriter, r *http.Request, id string, location string, eTag models.ETag, data interface{}) {
	if eTag != "" {
		w.Header().Set("ETag", eTag.String())
	}
	if id != "" {
		w.Header().Set("Id", id)
	}
	if location != "" {
		w.Header().Set("Location", location)
	}
	r = r.WithContext(context.WithValue(r.Context(), render.StatusCtxKey, http.StatusCreated))
	if data != nil {
		a.JSON(w, r, data)
	}
}

// GotResource writes a standardized resource response to the http response object and is intended to be
// used in response to a GET request.
func (a *APIBase) GotResource(w http.ResponseWriter, r *http.Request, resource documents.ResourceDocument) {
	mutable, ok := resource.(models.MutableResource)
	if ok {
		w.Header().Set("ETag", mutable.GetETag().String())
	}
	r = r.WithContext(context.WithValue(r.Context(), render.StatusCtxKey, http.StatusOK))
	a.JSON(w, r, resource)
}

// CreatedResource writes a standardized resource created response to the http response object and is
// intended to be used in response to a POST request. If data is nil the resource document will be directly
// serialized to JSON and sent in the response body, otherwise data will be used.
func (a *APIBase) CreatedResource(w http.ResponseWriter, r *http.Request, resource documents.ResourceDocument, data interface{}) {
	var (
		id             = resource.GetID().String()
		location       = resource.GetLink()
		eTag           models.ETag
		resourceOrData interface{} = resource
	)
	mutable, ok := resource.(models.MutableResource)
	if ok {
		eTag = mutable.GetETag()
	}
	if data != nil {
		resourceOrData = data
	}
	a.Created(w, r, id, location, eTag, resourceOrData)
}

// UpdatedResource writes a standardized resource updated response to the http response object and is
// intended to be used in response to a PUT or PATCH request.
func (a *APIBase) UpdatedResource(w http.ResponseWriter, r *http.Request, resource documents.ResourceDocument, data interface{}) {
	mutable, ok := resource.(models.MutableResource)
	if ok {
		w.Header().Set("ETag", mutable.GetETag().String())
	}
	r = r.WithContext(context.WithValue(r.Context(), render.StatusCtxKey, http.StatusOK))
	if data != nil {
		a.JSON(w, r, data)
	} else {
		a.JSON(w, r, resource)
	}
}

// Authorize is a convenience wrapper around authorizationService.IsAuthorized.
// It performs an access control check on the specified resource using the currently authenticated
// legal entity from the request as the principal. Returns an error if the user is not authorized.
func (a *APIBase) Authorize(r *http.Request, operation *models.Operation, resourceID models.ResourceID) error {
	meta := a.MustAuthenticationMeta(r)
	authorized, err := a.authorizationService.IsAuthorized(
		r.Context(),
		meta.IdentityID,
		operation,
		resourceID)
	if err != nil {
		return err
	}
	if !authorized {
		// Return 401 Unauthorized.
		// This does NOT give away information to unauthorized users about the existence or non-existence of
		// resources, since the authorization check is performed in Web service code before even attempting
		// to read the resource. This means a 404 Not Found error will only be returned for resources that
		// the user would have been allowed to see if they existed.
		return gerror.NewErrUnauthorized("Unauthorized")
	}
	return nil
}

// AuthorizedResourceID is a convenience wrapper around authorizationService.IsAuthorized.
// It extracts the leaf resource id from the url of a request and performs an access
// control check on it using the currently authenticated legal entity from the request as
// the principal. Returns the extracted id and nil error if the user is authorized.
func (a *APIBase) AuthorizedResourceID(r *http.Request, operation *models.Operation) (models.ResourceID, error) {
	id, err := a.resourceLinker.GetLeafResourceID(r)
	if err != nil {
		// Return a not found error to the user since the request URL was invalid
		return models.ResourceID{}, gerror.NewErrNotFound("Not Found").Wrap(err)
	}
	err = a.Authorize(r, operation, id)
	if err != nil {
		return models.ResourceID{}, err
	}
	return id, nil
}

// LegalEntityID returns the leaf resource id from the url of the request as a LegalEntityID.
func (a *APIBase) LegalEntityID(r *http.Request) (models.LegalEntityID, error) {
	id, err := a.resourceLinker.GetLeafResourceID(r)
	if err != nil {
		return models.LegalEntityID{}, err
	}
	return models.LegalEntityIDFromResourceID(id), nil
}

// RepoID returns the leaf resource id from the url of the request as a RepoID.
func (a *APIBase) RepoID(r *http.Request) (models.RepoID, error) {
	id, err := a.resourceLinker.GetLeafResourceID(r)
	if err != nil {
		return models.RepoID{}, err
	}
	return models.RepoIDFromResourceID(id), nil
}

// AuthorizedLegalEntityID is like AuthorizedResourceID but returns a LegalEntityID.
func (a *APIBase) AuthorizedLegalEntityID(r *http.Request, operation *models.Operation) (models.LegalEntityID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.LegalEntityID{}, err
	}
	return models.LegalEntityIDFromResourceID(id), nil
}

// AuthorizedRepoID is like AuthorizedResourceID but returns a RepoID.
func (a *APIBase) AuthorizedRepoID(r *http.Request, operation *models.Operation) (models.RepoID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.RepoID{}, err
	}
	return models.RepoIDFromResourceID(id), nil
}

// AuthorizedSecretID is like AuthorizedResourceID but returns a SecretID.
func (a *APIBase) AuthorizedSecretID(r *http.Request, operation *models.Operation) (models.SecretID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.SecretID{}, err
	}
	return models.SecretIDFromResourceID(id), nil
}

// BuildID returns the leaf resource id from the url of the request as a BuildID.
func (a *APIBase) BuildID(r *http.Request) (models.BuildID, error) {
	id, err := a.resourceLinker.GetLeafResourceID(r)
	if err != nil {
		return models.BuildID{}, err
	}
	return models.BuildIDFromResourceID(id), nil
}

// AuthorizedBuildID is like AuthorizedResourceID but returns a BuildID.
func (a *APIBase) AuthorizedBuildID(r *http.Request, operation *models.Operation) (models.BuildID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.BuildID{}, err
	}
	return models.BuildIDFromResourceID(id), nil
}

// AuthorizedJobID is like AuthorizedResourceID but returns a JobID.
func (a *APIBase) AuthorizedJobID(r *http.Request, operation *models.Operation) (models.JobID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.JobID{}, err
	}
	return models.JobIDFromResourceID(id), nil
}

// AuthorizedStepID is like AuthorizedResourceID but returns a StepID.
func (a *APIBase) AuthorizedStepID(r *http.Request, operation *models.Operation) (models.StepID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.StepID{}, err
	}
	return models.StepIDFromResourceID(id), nil
}

// AuthorizedArtifactID is like AuthorizedResourceID but returns an ArtifactID.
func (a *APIBase) AuthorizedArtifactID(r *http.Request, operation *models.Operation) (models.ArtifactID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.ArtifactID{}, err
	}
	return models.ArtifactIDFromResourceID(id), nil
}

// AuthorizedRunnerID is like AuthorizedResourceID but returns a RunnerID.
func (a *APIBase) AuthorizedRunnerID(r *http.Request, operation *models.Operation) (models.RunnerID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.RunnerID{}, err
	}
	return models.RunnerIDFromResourceID(id), nil
}

// AuthorizedLogDescriptorID is like AuthorizedResourceID but returns a LogDescriptorID.
func (a *APIBase) AuthorizedLogDescriptorID(r *http.Request, operation *models.Operation) (models.LogDescriptorID, error) {
	id, err := a.AuthorizedResourceID(r, operation)
	if err != nil {
		return models.LogDescriptorID{}, err
	}
	return models.LogDescriptorIDFromResourceID(id), nil
}

// MustAuthenticationMeta returns information about the currently authenticated legal
// entity from the request. If the request is not authenticated then this panics.
func (a *APIBase) MustAuthenticationMeta(r *http.Request) *middleware.AuthenticationMeta {
	value := r.Context().Value(AuthenticationMetaContextKeyName)
	if value == nil {
		panic("Request is not authenticated")
	}
	meta, ok := value.(*middleware.AuthenticationMeta)
	if !ok {
		panic("Request is not authenticated")
	}
	return meta
}

// MustAuthenticatedIdentityID returns the id of the currently authenticated identity from the request.
// If the request is not authenticated then this panics.
func (a *APIBase) MustAuthenticatedIdentityID(r *http.Request) models.IdentityID {
	return a.MustAuthenticationMeta(r).IdentityID
}

func (a *APIBase) GetIfMatch(r *http.Request) models.ETag {
	return models.ETag(r.Header.Get("If-Match"))
}
