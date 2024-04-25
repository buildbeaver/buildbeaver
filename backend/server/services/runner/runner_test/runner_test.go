package runner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner/app/runner_test"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

func TestRunnerRegistration(t *testing.T) {
	ctx := context.Background()
	pagination := models.NewPagination(30, nil) // read enough to cover our test data

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	// Create a company to register the runner against
	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")

	// Create a runner test app in order to get a client certificate
	runnerConfig := runner_test.TestConfig(t)
	_, err = runner_test.New(runnerConfig)
	require.NoError(t, err)
	clientCert, err := certificates.LoadCertificateFromPemFile(runnerConfig.RunnerCertificateFile)
	require.NoError(t, err)

	// Register the new runner using its client certificate
	runner := server_test.CreateRunner(t, ctx, app, "", testCompany.ID, clientCert)

	// Check that we have a registered runner, with an identity and a client certificate credential
	_, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	runnerIdentity, err := app.RunnerService.ReadIdentity(ctx, nil, runner.ID)
	require.NoError(t, err)
	credentials, _, err := app.CredentialService.ListCredentialsForIdentity(ctx, nil, runnerIdentity.ID, pagination)
	require.NoError(t, err)
	assert.Equal(t, 1, len(credentials), "Should have 1 credential for runner identity")

	// Update the runner
	runner.SoftwareVersion = "test-updated-software-version-1"
	_, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)

	// Remove the runner registration again
	err = app.RunnerService.SoftDelete(ctx, nil, runner.ID, dto.DeleteRunner{})
	require.NoError(t, err)

	// Check that the runner is soft-deleted but still reachable, i.e. it should not show up in a search
	// but should still be readable by ID
	runner, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	runnerIdentity, err = app.RunnerService.ReadIdentity(ctx, nil, runner.ID)
	require.NoError(t, err)

	// Check that the runner's credentials are gone
	credentials, _, err = app.CredentialService.ListCredentialsForIdentity(ctx, nil, runnerIdentity.ID, pagination)
	require.NoError(t, err)
	assert.Zero(t, len(credentials), "Should have no remaining credentials for identity")

	// Runner should not be able to be updated since it has been soft-deleted
	runner.SoftwareVersion = "test-updated-software-version-2"
	_, err = app.RunnerService.Update(ctx, nil, runner)
	require.Error(t, err)
}

func TestRunnerLabelUpdate(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.NoError(t, err)
	defer cleanup()

	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, app, "", "", "")
	runner := server_test.CreateRunner(t, ctx, app, "", testCompany.ID, nil)

	// Apply some labels to the runner
	labels := models.Labels{"one", "two", "three"}
	runner.Labels = labels
	runner, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// Should be able to read them back
	runner, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// Drop a label
	labels = models.Labels{"one", "two"}
	runner.Labels = labels
	runner, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// It should be gone
	runner, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// Add a label
	labels = models.Labels{"one", "two", "four"}
	runner.Labels = labels
	runner, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// It should stick
	runner, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// Add original label back
	labels = models.Labels{"one", "two", "three", "four"}
	runner.Labels = labels
	runner, err = app.RunnerService.Update(ctx, nil, runner)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)

	// It should stick
	runner, err = app.RunnerService.Read(ctx, nil, runner.ID)
	require.NoError(t, err)
	require.Equal(t, labels, runner.Labels)
}
