package gerror

import (
	"errors"
	"net/http"
)

const (
	ErrCodeInternal              Code = "Internal"
	ErrCodeValidationFailed      Code = "ValidationFailed"
	ErrCodeInvalidQueryParameter Code = "InvalidQueryParameter"
	ErrCodeNotFound              Code = "NotFound"
	ErrCodeUnauthorized          Code = "Unauthorized"
	ErrCodeAlreadyExists         Code = "AlreadyExists"
	ErrCodeOptimisticLockFailed  Code = "OptimisticLockFailed"
	ErrCodeAccountDisabled       Code = "AccountDisabled"
	ErrCodeRunnerDisabled        Code = "RunnerDisabled"
	ErrCodeTimeout               Code = "Timeout"
	ErrCodeLogClosed             Code = "LogClosed"
	ErrHttpOperationFailed       Code = "HttpOperationFailed"
	ErrArtifactUploadFailed      Code = "ArtifactUploadFailed"
)

// ToError locates an Error in the provided error chain and returns it if it
// matches the provided code. Otherwise, returns nil.
func ToError(err error, code Code) *Error {
	if err == nil {
		return nil
	}
	var gErr Error
	if errors.As(err, &gErr) && gErr.Code() == code {
		return &gErr
	}
	return nil
}

func NewErrInternal() Error {
	return NewError(
		"An internal server error occurred",
		AudienceExternal,
		ErrCodeInternal,
		http.StatusInternalServerError,
		nil,
	)
}

func ToInternal(err error) *Error {
	return ToError(err, ErrCodeInternal)
}

func IsInternal(err error) bool {
	return ToInternal(err) != nil
}

func NewErrArtifactUploadFailed(message string, err error) Error {
	return NewError(message, AudienceInternal, ErrArtifactUploadFailed, http.StatusInternalServerError, err)
}

func ToArtifactUploadFailed(err error) *Error {
	return ToError(err, ErrArtifactUploadFailed)
}

func IsArtifactUploadFailed(err error) bool {
	return ToArtifactUploadFailed(err) != nil
}

func NewErrValidationFailed(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeValidationFailed, http.StatusBadRequest, nil)
}

func ToValidationFailed(err error) *Error {
	return ToError(err, ErrCodeValidationFailed)
}

func IsValidationFailed(err error) bool {
	return ToValidationFailed(err) != nil
}

func NewErrInvalidQueryParameter(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeInvalidQueryParameter, http.StatusBadRequest, nil)
}

func ToInvalidQueryParameter(err error) *Error {
	return ToError(err, ErrCodeInvalidQueryParameter)
}

func IsInvalidQueryParameter(err error) bool {
	return ToInvalidQueryParameter(err) != nil
}

func NewErrNotFound(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeNotFound, http.StatusNotFound, nil)
}

func ToNotFound(err error) *Error {
	return ToError(err, ErrCodeNotFound)
}

func IsNotFound(err error) bool {
	return ToNotFound(err) != nil
}

func NewErrCodeRunnerDisabled() Error {
	return NewError(
		"Runner disabled; Please enable if you would like this runner to run jobs",
		AudienceExternal,
		ErrCodeRunnerDisabled,
		http.StatusNotFound,
		nil,
	)
}

func ToRunnerDisabled(err error) *Error {
	return ToError(err, ErrCodeRunnerDisabled)
}

func IsRunnerDisabled(err error) bool {
	return ToRunnerDisabled(err) != nil
}

func NewErrUnauthorized(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeUnauthorized, http.StatusUnauthorized, nil)
}

func ToUnauthorized(err error) *Error {
	return ToError(err, ErrCodeUnauthorized)
}

func IsUnauthorized(err error) bool {
	return ToUnauthorized(err) != nil
}

func NewErrAlreadyExists(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeAlreadyExists, http.StatusBadRequest, nil)
}

func ToAlreadyExists(err error) *Error {
	return ToError(err, ErrCodeAlreadyExists)
}

func IsAlreadyExists(err error) bool {
	return ToAlreadyExists(err) != nil
}

func NewErrOptimisticLockFailed(message string) Error {
	return NewError(message, AudienceExternal, ErrCodeOptimisticLockFailed, http.StatusPreconditionFailed, nil)
}
func ToOptimisticLockFailed(err error) *Error {
	return ToError(err, ErrCodeOptimisticLockFailed)
}

func IsOptimisticLockFailed(err error) bool {
	return ToOptimisticLockFailed(err) != nil
}

func NewErrAccountDisabled() Error {
	return NewError(
		"Account disabled; Please contact your administrator",
		AudienceExternal,
		ErrCodeAccountDisabled,
		http.StatusForbidden,
		nil,
	)
}
func ToAccountDisabled(err error) *Error {
	return ToError(err, ErrCodeAccountDisabled)
}

func IsAccountDisabled(err error) bool {
	return ToAccountDisabled(err) != nil
}

func NewErrTimeout(description string) Error {
	return NewError("Timeout: "+description, AudienceInternal, ErrCodeTimeout, http.StatusInternalServerError, nil)
}
func ToTimeout(err error) *Error {
	return ToError(err, ErrCodeTimeout)
}

func IsTimeout(err error) bool {
	return ToTimeout(err) != nil
}

func NewErrLogClosed() Error {
	// http.StatusGone "Indicates that the resource requested was previously in use but is no longer available
	// and will not be available again". This seems appropriate when trying to write to a closed log.
	return NewError("Log is closed", AudienceExternal, ErrCodeLogClosed, http.StatusGone, nil)
}

func ToLogClosed(err error) *Error {
	return ToError(err, ErrCodeLogClosed)
}

func IsLogClosed(err error) bool {
	return ToLogClosed(err) != nil
}
