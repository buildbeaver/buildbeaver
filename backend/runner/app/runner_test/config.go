package runner_test

import (
	"path/filepath"
	"testing"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/app"
	"github.com/buildbeaver/buildbeaver/runner/logging"
)

func TestConfig(t *testing.T) *app.RunnerConfig {
	// Create a temp directory for runner configuration, including certificates
	configDir := t.TempDir()

	return &app.RunnerConfig{
		RunnerAPIEndpoints:    []string{"https://localhost:3002"}, // best guess at a default, local runner API server
		RunnerLogTempDir:      logging.RunnerLogTempDirectory(filepath.Join(configDir, app.DefaultRunnerLogTempDirName)),
		RunnerCertificateFile: certificates.CertificateFile(filepath.Join(configDir, app.DefaultRunnerCertFile)),
		RunnerPrivateKeyFile:  certificates.PrivateKeyFile(filepath.Join(configDir, app.DefaultRunnerPrivateKeyFile)),
		AutoCreateCertificate: true,
		CACertFile:            certificates.CACertificateFile(filepath.Join(configDir, app.DefaultRunnerCACertificateFile)),
		InsecureSkipVerify:    true,
		LogUnregisteredCert:   false,
		SchedulerConfig: runner.SchedulerConfig{
			PollInterval: runner.DefaultPollInterval,
			ParallelJobs: runner.DefaultParallelBuilds,
		},
		ExecutorConfig: runner.ExecutorConfig{
			IsLocal:            false,
			DynamicAPIEndpoint: "https://localhost:3001", // best guess at a default, local Core API server
		},
		LogLevels: "",
	}
}
