package integration_tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/app/runner_test"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

func TestRunnerAndServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	gServer, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()

	gServer.RunnerAPIServer.Start() // Start the HTTPS server
	defer gServer.RunnerAPIServer.Stop(ctx)

	serverURL := gServer.RunnerAPIServer.GetServerURL()

	// Create a legal entity to use for runners
	testCompany := server_test.CreateCompanyLegalEntity(t, ctx, gServer, "", "", "")

	runnerConfig := runner_test.TestConfig(t)
	runnerConfig.RunnerAPIEndpoints = []string{serverURL}

	// Use the server certificate as a CA to test full TLS negotiation, including verifying the server certificate
	runnerConfig.InsecureSkipVerify = false
	copied, err := copyServerCertIfCACertMissing(runnerConfig.CACertFile, gServer.RunnerAPIServer.GetCertificateFile())
	require.NoError(t, err)
	if copied {
		t.Logf("No CA certificate found for test runner: copied server certificate to use as CA from %q to %q",
			gServer.RunnerAPIServer.GetCertificateFile(), runnerConfig.CACertFile)
	}

	// Create a runner test app; this will create a client certificate if required
	bbRunner, err := runner_test.New(runnerConfig)
	require.Nil(t, err)

	// Register the new runner using its client certificate
	clientCert, err := certificates.LoadCertificateFromPemFile(runnerConfig.RunnerCertificateFile)
	require.NoError(t, err)
	runner := server_test.CreateRunner(t, ctx, gServer, "", testCompany.ID, clientCert)

	// Start the runner
	bbRunner.Scheduler.Start()
	defer bbRunner.Scheduler.Stop()

	// Wait until we've seen at least one successful poll from the runner to the server
	waitForSuccessfulPolls(t, bbRunner.Scheduler, 1, 30*time.Second)

	// Remove the runner registration again
	err = gServer.RunnerService.SoftDelete(ctx, nil, runner.ID, dto.DeleteRunner{})
	require.NoError(t, err)
}

// waitForSuccessfulPolls blocks and waits until the runner has completed the specified number of successful polls.
// Any failed polls are flagged as test failures but this function will continue waiting until success or timeout.
func waitForSuccessfulPolls(t *testing.T, jobScheduler *runner.Scheduler, requiredPollCount int64, timeout time.Duration) {
	endTime := time.Now().Add(timeout)
	failedPollErrorReported := false

	t.Logf("Waiting for runner to poll successfully %d time(s)", requiredPollCount)
	for {
		stats := jobScheduler.GetStats()

		if stats.FailedPollCount > 0 && !failedPollErrorReported {
			t.Errorf("Runner failed at least once to poll the server")
			failedPollErrorReported = true
		}

		if stats.SuccessfulPollCount >= requiredPollCount {
			t.Logf("Runner has successfully polled %d time(s); finished waiting", stats.SuccessfulPollCount)
			return
		}

		if time.Now().After(endTime) {
			t.Errorf("Timed out after %v waiting for runner to successfully complete %d poll(s); only %d completed",
				timeout, requiredPollCount, stats.SuccessfulPollCount)
			return
		}

		//t.Logf("Runner has only successfully polled %d out of %d time(s); continue waiting",
		//	stats.SuccessfulPollCount, requiredPollCount)
		time.Sleep(100 * time.Millisecond)
	}
}

// copyServerCertIfCACertMissing checks whether a CA certificate file exists at the caCertFile path.
// If nothing exists at this path and a server certificate exists at the serverCertFile path then the server
// certificate is copied to the caCertFile path, so a client (e.g. a runner) will accept the server's
// certificate when connection via https.
// Returns true iff a certificate was copied over.
func copyServerCertIfCACertMissing(caCertFile certificates.CACertificateFile, serverCertFile certificates.CertificateFile) (copied bool, err error) {
	// Do we have anything at the CA certificate path already? Don't overwrite anything.
	caCertFound := true
	_, err = os.Stat(caCertFile.String())
	if err != nil {
		if os.IsNotExist(err) {
			caCertFound = false
		} else {
			return false, fmt.Errorf("error checking for existence of CA certificate: %w", err)
		}
	}
	if caCertFound {
		return false, nil // nothing to do
	}

	// Do we have a server certificate?
	// TODO: Parse the certificate and check that it as self-signed certificate
	serverCertFound := true
	_, err = os.Stat(serverCertFile.String())
	if err != nil {
		if os.IsNotExist(err) {
			serverCertFound = false
		} else {
			return false, fmt.Errorf("error checking for existence of server certificate: %w", err)
		}
	}
	if !serverCertFound {
		return false, nil // nothing to do
	}

	// Ensure destination certificate directory exists
	caCertDir := filepath.Dir(caCertFile.String())
	caCertDirInfo, err := os.Stat(caCertDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Directory doesn't exist, so create the entire path
			err = os.MkdirAll(caCertDir, 0755) // read and traverse permissions for everyone
			if err != nil {
				return false, fmt.Errorf("error making CA certificate directory %s: %w", caCertDir, err)
			}
		} else {
			return false, fmt.Errorf("error checking for existence of directory %s: %w", caCertDir, err)
		}
	} else if !caCertDirInfo.IsDir() {
		return false, fmt.Errorf("error making CA certificate directory %s: file is present with same name", caCertDir)
	}

	// Copy server cert over
	source, err := os.Open(serverCertFile.String())
	if err != nil {
		return false, fmt.Errorf("error opening server certificate file to copy: %w", err)
	}
	defer source.Close()
	destination, err := os.Create(caCertFile.String())
	if err != nil {
		return false, fmt.Errorf("error creating runner CA certificate file during copy: %w", err)
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return false, fmt.Errorf("error copying data from server certificate file to runner CA certificate file: %w", err)
	}

	return true, nil
}
