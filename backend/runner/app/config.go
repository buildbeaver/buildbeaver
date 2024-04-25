package app

import (
	"fmt"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
)

const (
	DefaultRunnerCertFile          = "runner-client-cert.pem"
	DefaultRunnerPrivateKeyFile    = "runner-private-key.pem"
	DefaultRunnerCACertificateFile = "ca-cert.pem"
	DefaultRunnerLogTempDirName    = "log-temp"
)

// LogSafeFlags is a list of flags by name whose values are safe to log.
var LogSafeFlags = []string{
	"runner_api_endpoints",
	"runner_config_directory",
	"runner_log_temp_directory",
	"dev_insecure_skip_verify",
	"log_levels",
}

type RunnerConfig struct {
	RunnerAPIEndpoints    []string
	RunnerLogTempDir      logging.RunnerLogTempDirectory
	RunnerCertificateFile certificates.CertificateFile
	RunnerPrivateKeyFile  certificates.PrivateKeyFile
	AutoCreateCertificate client.AutoCreateCertificate
	CACertFile            certificates.CACertificateFile
	InsecureSkipVerify    client.InsecureSkipVerify
	LogUnregisteredCert   bool
	LogLevels             logger.LogLevelConfig
	SchedulerConfig       runner.SchedulerConfig
	ExecutorConfig        runner.ExecutorConfig
}

func ConfigFromFlags() (*RunnerConfig, error) {
	var (
		runnerConfigDir     string
		runnerLogTempDirStr string
	)
	config := &RunnerConfig{
		ExecutorConfig: runner.ExecutorConfig{
			IsLocal: false, // this is a real runner, not part of bb
		},
	}

	flag.StringArrayVar(&config.RunnerAPIEndpoints, "runner_api_endpoints", []string{"https://runner.changeme.com"},
		"One or more endpoints to connect to the BuildBeaver server's Runner API")
	flag.StringVar((*string)(&config.ExecutorConfig.DynamicAPIEndpoint), "dynamic_api_endpoint", "https://app.changeme.com",
		"The endpoint for build jobs to connect to the Dynamic API.")
	flag.StringVar(&runnerConfigDir, "runner_config_directory",
		defaultRunnerConfigDir, "The path on the local host containing configuration and certificates for the runner.")
	flag.StringVar(&runnerLogTempDirStr, "runner_log_temp_directory",
		defaultRunnerLogTempDir, "The path on the local host where the runner can buffer build logs in temporary files.")
	flag.BoolVar((*bool)(&config.AutoCreateCertificate), "auto_create_certificate",
		true, "True to automatically create a key pair and client certificate for the runner to use when authenticating to the server, if not already configured.")
	flag.BoolVar((*bool)(&config.InsecureSkipVerify), "dev_insecure_skip_verify",
		false, "True to disable verification of the server's certificate when connecting via TLS. This option should not be used in production.")
	flag.StringVar((*string)(&config.LogLevels), "log_levels",
		"", fmt.Sprintf("A comma separated list of name=level pairs where name is the name of the logger and level is one of: %s", logger.ListLogLevels()))
	flag.DurationVar(&config.SchedulerConfig.PollInterval, "poll_interval",
		runner.DefaultPollInterval, "The interval to check for new jobs to run.")
	flag.IntVar(&config.SchedulerConfig.ParallelJobs, "parallel_jobs",
		runner.DefaultParallelBuilds, "The number of jobs to run in parallel.")
	flag.Parse()

	config.RunnerLogTempDir = logging.RunnerLogTempDirectory(runnerLogTempDirStr)
	config.RunnerCertificateFile = certificates.CertificateFile(filepath.Join(runnerConfigDir, DefaultRunnerCertFile))
	config.RunnerPrivateKeyFile = certificates.PrivateKeyFile(filepath.Join(runnerConfigDir, DefaultRunnerPrivateKeyFile))
	config.CACertFile = certificates.CACertificateFile(filepath.Join(runnerConfigDir, DefaultRunnerCACertificateFile))
	config.LogUnregisteredCert = true // log the certificate if not registered

	return config, nil
}
