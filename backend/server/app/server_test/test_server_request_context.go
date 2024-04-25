package server_test

import "github.com/buildbeaver/buildbeaver/server/api/rest/routes"

// TestServerRequestContext provides a BaseURL() function returning a URL that can be used for document
// construction during integration tests, pointing back to the local test server.
type TestServerRequestContext struct {
	baseURL string
}

// NewTestServerRequestContext creates a new request context that is suitable for URL construction during integration
// tests, pointing back to the local test server.
func NewTestServerRequestContext(app *TestServer) routes.RequestContext {
	return &TestServerRequestContext{
		baseURL: app.CoreAPIServer.GetServerURL(),
	}
}

func (c *TestServerRequestContext) BaseURL() string {
	return c.baseURL
}
