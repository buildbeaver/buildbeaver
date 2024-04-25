package routes

import (
	"fmt"
	"net/http"
	"strings"
)

type RequestContext interface {
	BaseURL() string
}

type HTTPRequestCtx struct {
	scheme string
	host   string
}

func RequestCtx(r *http.Request) *HTTPRequestCtx {
	return &HTTPRequestCtx{
		scheme: scheme(r),
		host:   host(r),
	}
}

func (r *HTTPRequestCtx) BaseURL() string {
	return fmt.Sprintf("%s://%s", r.scheme, r.host)
}

func (r *HTTPRequestCtx) String() string {
	return r.BaseURL()
}

// scheme returns the original scheme (http or https) the client specified when making the request.
func scheme(r *http.Request) string {
	// TODO support Forwarded header
	if r.URL.Scheme == "https" || r.TLS != nil {
		return "https"
	}
	if strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		return "https"
	}
	return "http"
}

// host returns the original host the client specified when making the request.
func host(r *http.Request) string {
	// TODO support Forwarded header
	if r.Header.Get("X-Forwarded-host") != "" {
		return r.Header.Get("X-Forwarded-host")
	}
	return r.Host
}
