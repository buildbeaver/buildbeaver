package gerror

import (
	"fmt"
)

const (
	AudienceInternal Audience = "internal"
	AudienceExternal Audience = "external"
)

type Audience string
type Code string
type DetailKey string
type Details map[DetailKey]Detail

type Error struct {
	innerErr error
	// errorText is the internal error chain suitable for logging and debugging
	errorText string
	// message is the human friendly error message suitable for display to an end user
	message        string
	details        Details
	audience       Audience
	code           Code
	httpStatusCode int
}

func NewError(message string, audience Audience, code Code, httpStatusCode int, inner error) Error {
	return NewErrorWithDetails(message, nil, audience, code, httpStatusCode, inner)
}

func NewErrorWithDetails(message string, details Details, audience Audience, code Code, httpStatusCode int, inner error) Error {
	return Error{
		message:        message,
		errorText:      makeErrorText(message, details, inner),
		details:        details,
		audience:       audience,
		code:           code,
		httpStatusCode: httpStatusCode,
	}
}

func (e Error) Error() string {
	if e.errorText != "" {
		return e.errorText
	} else {
		// If errorText not set, return the message
		return e.message
	}
}

func (e Error) Unwrap() error {
	return e.innerErr
}

func (e Error) Message() string {
	return e.message
}

func (e Error) Details() map[DetailKey]Detail {
	m := make(Details, len(e.details))
	for k, v := range e.details {
		m[k] = v
	}
	return m
}

func (e Error) Audience() Audience {
	return e.audience
}

func (e Error) Code() Code {
	return e.code
}

func (e Error) HTTPStatusCode() int {
	return e.httpStatusCode
}

// HasHTTPStatusCode returns true iff the supplied error is a gerror.Error object with the specified HTTP status code.
func HasHTTPStatusCode(err error, statusCode int) bool {
	errorDoc, ok := err.(Error)
	if !ok {
		return false
	}
	return errorDoc.HTTPStatusCode() == statusCode
}

// Wrap returns a copy of the error with the inner error set to the specified err.
func (e Error) Wrap(innerErr error) Error {
	return Error{
		innerErr:       innerErr,
		errorText:      makeErrorText(e.message, e.details, innerErr),
		message:        e.message,
		details:        e.Details(),
		audience:       e.audience,
		code:           e.code,
		httpStatusCode: e.httpStatusCode,
	}
}

// IDetail returns a copy of the error with a new internal detail appended to it.
func (e Error) IDetail(key DetailKey, value interface{}) Error {
	return e.withDetail(AudienceInternal, key, value)
}

// EDetail returns a copy of the error with a new external detail appended to it.
func (e Error) EDetail(key DetailKey, value interface{}) Error {
	return e.withDetail(AudienceExternal, key, value)
}

// withDetail returns a copy of the error with a new detail appended to it.
func (e *Error) withDetail(audience Audience, key DetailKey, value interface{}) Error {
	details := e.Details()
	details[key] = NewDetail(audience, key, value)
	return Error{
		details:        details,
		errorText:      makeErrorText(e.message, details, e.innerErr),
		innerErr:       e.innerErr,
		message:        e.message,
		audience:       e.audience,
		code:           e.code,
		httpStatusCode: e.httpStatusCode,
	}
}

func makeErrorText(message string, details Details, inner error) string {
	var detailsStr string
	if len(details) > 0 {
		detailsStr = " ["
		for k, v := range details {
			if detailsStr == " [" {
				detailsStr += fmt.Sprintf("%s=%s", k, v.value)
			} else {
				detailsStr += fmt.Sprintf(", %s=%s", k, v.value)
			}
		}
		detailsStr += "]"
	}
	var errStr string
	if inner != nil {
		errStr = fmt.Sprintf(": %v", inner)
	}
	return fmt.Sprintf("%s%s%s", message, detailsStr, errStr)
}

type Detail struct {
	audience Audience
	key      DetailKey
	value    interface{}
}

func NewDetail(audience Audience, key DetailKey, value interface{}) Detail {
	return Detail{
		audience: audience,
		key:      key,
		value:    value,
	}
}

func (a Detail) Audience() Audience {
	return a.audience
}

func (a Detail) Key() DetailKey {
	return a.key
}

func (a Detail) Value() interface{} {
	return a.value
}
