package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

// APIClient is an HTTP client used to interact with the BuildBeaver REST API.
type APIClient struct {
	endpoints       []string
	httpClient      *http.Client
	retryableClient *retryablehttp.Client
	authenticator   Authenticator
	log             logger.Log
}

func NewAPIClient(endpoints []string, authenticator Authenticator, logFactory logger.LogFactory) (*APIClient, error) {
	var err error
	log := logFactory("APIClient")

	// Create a separate HTTP client to configure; do not share HTTP clients between instances of APIClient
	// so that each APIClient can have separate authentication.
	httpClient := &http.Client{}
	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryWaitMin = time.Millisecond * 100
	retryableClient.RetryWaitMax = time.Second * 5
	retryableClient.RetryMax = 10
	retryableClient.Logger = NewLeveledLogger(log) // use adaptor to get log level support
	retryableClient.HTTPClient = httpClient

	if authenticator != nil {
		// Allow authenticator to add properties to the client (especially TLS certificates)
		retryableClient, err = authenticator.AuthenticateClient(retryableClient)
		if err != nil {
			return nil, fmt.Errorf("error setting up HTTP client for authentication: %w", err)
		}
	}
	return &APIClient{
		endpoints:       endpoints,
		authenticator:   authenticator,
		httpClient:      httpClient,
		retryableClient: retryableClient,
		log:             log,
	}, nil
}

// get performs a basic HTTP GET request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. Returns the HTTP status code,
// headers and full response body. Returns an error if there was a problem making the request. No status code
// inspection is made.
func (a *APIClient) get(ctx context.Context, headers http.Header, pathOrURL string) (int, http.Header, []byte, error) {
	return a.doRequest(ctx, headers, "GET", pathOrURL, nil)
}

// getStream performs a basic HTTP GET request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. Returns the HTTP status code,
// headers and response body. Returns an error if there was a problem making the request. No status code
// inspection is made.
func (a *APIClient) getStream(ctx context.Context, headers http.Header, pathOrURL string) (int, http.Header, io.ReadCloser, error) {
	return a.doRequestStream(ctx, headers, "GET", pathOrURL, nil)
}

// put performs a basic HTTP PUT request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. If data is not nil it will be
// serialized to JSON and sent in the request body. Returns the HTTP status code, headers and buffered response body.
// Returns an error if there was a problem making the request. No status code inspection is made.
func (a *APIClient) put(ctx context.Context, headers http.Header, pathOrURL string, data interface{}) (int, http.Header, []byte, error) {
	return a.doRequest(ctx, headers, "PUT", pathOrURL, data)
}

// patch performs a basic HTTP PATCH request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. If data is not nil it will be
// serialized to JSON and sent in the request body. Returns the HTTP status code, headers and buffered response body.
// Returns an error if there was a problem making the request. No status code inspection is made.
func (a *APIClient) patch(ctx context.Context, headers http.Header, pathOrURL string, data interface{}) (int, http.Header, []byte, error) {
	return a.doRequest(ctx, headers, "PATCH", pathOrURL, data)
}

// post performs a basic HTTP POST request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. If data is not nil it will be
// serialized to JSON and sent in the request body. Returns the HTTP status code, headers and buffered response body.
// Returns an error if there was a problem making the request. No status code inspection is made.
func (a *APIClient) post(ctx context.Context, headers http.Header, pathOrURL string, data interface{}) (int, http.Header, []byte, error) {
	return a.doRequest(ctx, headers, "POST", pathOrURL, data)
}

// postStream performs a basic HTTP POST request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. Returns the HTTP status code,
// headers and response body. Returns an error if there was a problem making the request. No status code
// inspection is made.
func (a *APIClient) postStream(ctx context.Context, headers http.Header, pathOrURL string, data io.ReadSeeker) (int, http.Header, io.ReadCloser, error) {
	return a.doRequestStream(ctx, headers, "POST", pathOrURL, data)
}

// delete performs a basic HTTP DELETE request. If a path is specified then a url will be made using the currently
// configured endpoints. If a full url is specified it will be used directly. Returns the HTTP status code,
// headers and buffered response body. Returns an error if there was a problem making the request. No status code
// inspection is made.
func (a *APIClient) delete(ctx context.Context, headers http.Header, pathOrURL string) (int, http.Header, []byte, error) {
	return a.doRequest(ctx, headers, "DELETE", pathOrURL, nil)
}

// doRequest performs an HTTP request and returns the status code, response headers and response body.
// Returns an error if there was a problem making the request but no HTTP status code inspection is made.
func (a *APIClient) doRequest(ctx context.Context, headers http.Header, verb string, pathOrURL string, data interface{}) (int, http.Header, []byte, error) {
	var (
		buf []byte
		err error
	)
	if data != nil {
		buf, err = json.Marshal(data)
		if err != nil {
			return -1, nil, nil, errors.Wrap(err, "error marshaling request data to JSON")
		}
	}
	statusCode, header, stream, err := a.doRequestStream(ctx, headers, verb, pathOrURL, buf)
	if err != nil {
		return 0, nil, nil, err
	}
	defer stream.Close()
	body, err := ioutil.ReadAll(stream)
	if err != nil {
		return -1, nil, nil, errors.Wrap(err, "error reading response body")
	}
	return statusCode, header, body, nil
}

// doRequestStream performs an HTTP request and returns the status code, response headers and response body.
// The caller is responsible for closing the response body.
// Returns an error if there was a problem making the request but no HTTP status code inspection is made.
func (a *APIClient) doRequestStream(ctx context.Context, headers http.Header, verb string, pathOrURL string, data interface{}) (int, http.Header, io.ReadCloser, error) {
	endpoint, err := a.getRequestEndpoint(pathOrURL)
	if err != nil {
		return -1, nil, nil, fmt.Errorf("error getting request endpoint: %w", err)
	}
	req, err := retryablehttp.NewRequest(verb, endpoint, data)
	if err != nil {
		return -1, nil, nil, errors.Wrap(err, "error making request")
	}
	req = req.WithContext(ctx)
	if a.authenticator != nil {
		req.Header, err = a.authenticator.AuthenticateRequest(req.Header)
		if err != nil {
			return -1, nil, nil, errors.Wrap(err, "error authenticating request")
		}
	}
	if headers != nil {
		for k, v := range headers {
			for _, vv := range v {
				req.Header.Set(k, vv)
			}
		}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := a.retryableClient.Do(req)
	if err != nil {
		return -1, nil, nil, errors.Wrap(err, "error during request")
	}
	return res.StatusCode, res.Header, res.Body, nil
}

func (a *APIClient) getRequestEndpoint(pathOrURL string) (string, error) {
	uri, err := url.ParseRequestURI(pathOrURL)
	if err != nil || uri.Host == "" {
		endpoint, err := a.getEndpoint()
		if err != nil {
			return "", errors.Wrap(err, "error getting endpoint")
		}
		// Ensure endpoint does not end in a slash; repeatedly trim any "/" suffix
		for len(endpoint) > 0 && strings.HasSuffix(endpoint, "/") {
			endpoint = strings.TrimSuffix(endpoint, "/")
		}
		// Ensure path begins with a slash
		if !strings.HasPrefix(pathOrURL, "/") {
			pathOrURL = fmt.Sprintf("/%s", pathOrURL)
		}
		uri, err = url.ParseRequestURI(fmt.Sprintf("%s%s", endpoint, pathOrURL))
		if err != nil {
			return "", errors.Wrap(err, "error forming url")
		}
	}
	return uri.String(), nil
}

// getEndpoint returns the base endpoint for the API or an error if no endpoint could be found.
func (a *APIClient) getEndpoint() (string, error) {
	if len(a.endpoints) == 0 {
		return "", errors.New("No endpoints")
	}
	return a.endpoints[0], nil
}

// isOneOf returns true iff an HTTP status code is one of the supplied set of valid codes.
func (a *APIClient) isOneOf(statusCode int, validCodes []int) bool {
	for _, code := range validCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// makeHTTPError attempts to parse an HTTP response body to a standard public error
// and return it. If the response body cannot be parsed, a generic error including
// the text of the response body will be returned instead.
func (a *APIClient) makeHTTPError(statusCode int, body []byte) error {
	doc := &documents.ErrorDocument{}
	err := json.Unmarshal(body, doc)
	if err != nil {
		// We don't have error info in the body so return a more generic HTTP error
		return gerror.NewError(
			fmt.Sprintf("error %d in HTTP response: %s", statusCode, string(body[:])),
			gerror.AudienceExternal,
			gerror.ErrHttpOperationFailed,
			statusCode,
			nil,
		)
	}
	var details map[gerror.DetailKey]gerror.Detail
	for k, v := range doc.Details {
		details[k] = gerror.NewDetail(gerror.AudienceExternal, k, v)
	}
	return gerror.NewErrorWithDetails(doc.Message, details, gerror.AudienceExternal, doc.Code, doc.HTTPStatusCode, nil)
}

func (a *APIClient) ifMatchHeader(eTag models.ETag) http.Header {
	h := http.Header{}
	h.Set("If-Match", eTag.String())
	return h
}
