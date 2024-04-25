package github_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github/github_test_utils"
)

func TestSmeeEventHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err, "Error initializing app")
	defer cleanup()

	smeeClient := github_test_utils.NewSmeeClientForGitHubTestAccount(app.LogFactory)

	t.Log("Subscribing handler to smee notifications")
	shutdown := smeeClient.SubscribeHandler(func(event *github_test_utils.SmeeNotification) error {
		t.Logf("Received SMEE Notification: ID %d, Query %v, Headers:%v, Body length:%d",
			event.SmeeID, event.QueryParams, event.Headers, len(event.Body))
		return nil
	})

	t.Log("Shutting down...")
	shutdown()
}

func TestSmeeChannelIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err, "Error initializing app")
	defer cleanup()

	smeeClient := github_test_utils.NewSmeeClientForGitHubTestAccount(app.LogFactory)

	t.Log("Subscribing channel to smee notifications")
	eventChan := make(chan *github_test_utils.SmeeNotification)
	err = smeeClient.SubscribeChan(eventChan)
	assert.NoError(t, err, "Unable to subscribe to smee.io events")

	// TODO: Provide a way to create test events, so we can test reading them
	// for event := range eventChan {
	//	t.Logf("SMEE Notification: ID %d, Query %v, Headers:%v, Body length:%d",
	//		event.SmeeID, event.QueryParams, event.Headers, len(event.Body))
	//	if event.IsTheOneWeAreLookingFor() {
	//		break
	//	}
	// }

	t.Log("Shutting down...")
	smeeClient.UnsubscribeChan(eventChan)
}
