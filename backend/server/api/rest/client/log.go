package client

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	url2 "net/url"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

// StreamingHTTPWriter is an io.WriteCloser that when closed, waits for a corresponding HTTP request to finish
// and returns the request error, if any.
type StreamingHTTPWriter struct {
	w     *io.PipeWriter
	doneC chan error
}

func (f *StreamingHTTPWriter) Write(p []byte) (int, error) {
	return f.w.Write(p)
}

// Close will close the writer and return the error (if any) received from the HTTP request, via the done channel.
// This is more accurate than the error returned from Write() since it reflects the overall status of the HTTP request.
func (f *StreamingHTTPWriter) Close() error {
	wErr := f.w.Close()
	reqErr := <-f.doneC
	if reqErr != nil {
		return reqErr
	}
	return wErr
}

func (a *APIClient) OpenLogWriteStream(ctx context.Context, logID models.LogDescriptorID) (io.WriteCloser, error) {
	endpoint, err := a.getRequestEndpoint(fmt.Sprintf("/api/v1/runner/logs/%s/data", logID))
	if err != nil {
		return nil, fmt.Errorf("error getting request endpoint: %w", err)
	}
	reader, writer := io.Pipe()
	req, err := http.NewRequest("POST", endpoint, reader)
	if err != nil {
		return nil, errors.Wrap(err, "error making request")
	}
	req = req.WithContext(ctx)
	if a.authenticator != nil {
		req.Header, err = a.authenticator.AuthenticateRequest(req.Header)
		if err != nil {
			return nil, errors.Wrap(err, "error authenticating request")
		}
	}
	req.Header.Set("Content-Type", "application/json")

	doneC := make(chan error)
	streamWriter := &StreamingHTTPWriter{w: writer, doneC: doneC}
	go func() {
		res, err := a.httpClient.Do(req)

		// Read and close the response body - this might contain an error document.
		var body []byte
		if res != nil {
			var bodyReadErr error // Do not overwrite err returned from httpClient.Do()
			body, bodyReadErr = ioutil.ReadAll(res.Body)
			if bodyReadErr != nil {
				a.log.Warnf("Warning: ignoring error reading HTTP response body when opening log write stream: %v", err)
				body = nil
			}
			res.Body.Close()
		}

		// If the HTTP request succeeded with no error but returned a non-success HTTP code, turn this into an
		// error based on the response body returned and return this error to the caller through the done channel
		if err == nil && !a.isOneOf(res.StatusCode, []int{http.StatusOK, http.StatusNoContent}) {
			err = a.makeHTTPError(res.StatusCode, body)
		}

		// Close the pipe with a reader error so that any subsequent calls to the writer will fail with the
		// correct error, rather than getting a generic error "io: read/write on closed pipe"
		// which is what gets returned if reader.CloseWithError() has not been called.
		// Do not call writer.CloseWithError() as well; if both errors are set this will also prevent the
		// correct error being returned to the writer.
		// NOTE: It's actually too late to get the correct error in, since the call to httpClient.Do() above
		// will call reader.Close() on the reader, setting the error to "io: read/write on closed pipe"
		// We must rely on the correct error being sent back down the doneC channel and read from Close()
		reader.CloseWithError(err)

		// Send the error back
		doneC <- err
	}()
	return streamWriter, nil
}

func (a *APIClient) OpenLogReadStream(ctx context.Context, logID models.LogDescriptorID, search *documents.LogSearchRequest) (io.ReadCloser, error) {
	endpoint, err := a.getRequestEndpoint(fmt.Sprintf("/api/v1/runner/logs/%s/data", logID))
	if err != nil {
		return nil, fmt.Errorf("error getting request endpoint: %w", err)
	}
	url, err := url2.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing request endpoint: %w", err)
	}
	url.RawQuery = search.GetQuery().Encode()
	code, _, body, err := a.getStream(ctx, nil, url.String())
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		body.Close()
		return nil, a.makeHTTPError(code, nil)
	}
	return body, nil
}
