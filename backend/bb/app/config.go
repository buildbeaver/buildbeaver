package app

import (
	"fmt"
	"path/filepath"

	"github.com/buildbeaver/buildbeaver/bb/bb_server"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/local_backend"
	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services/blob"
	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/buildbeaver/buildbeaver/server/services/encryption"
	"github.com/buildbeaver/buildbeaver/server/services/log"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// DefaultLocalServerPort is the port that the local API server within bb should run on
// TODO: Consider picking a random high port number a fixed port is a problem
const DefaultLocalServerPort = 3003

type BBConfig struct {
	BBAPIConfig              bb_server.BBAPIServerConfig
	LocalKeyManagerMasterKey encryption.LocalKeyManagerMasterKey
	LogFilePath              logger.LogFilePath
	LogLevels                logger.LogLevelConfig
	DatabaseConfig           store.DatabaseConfig
	DatabaseFilePath         string
	LocalBlobStoreDir        blob.LocalBlobStoreDirectory
	RunnerLogTempDir         logging.RunnerLogTempDirectory
	LogServiceConfig         log.LogServiceConfig
	SchedulerConfig          runner.SchedulerConfig
	ExecutorConfig           runner.ExecutorConfig
	JWTConfig                credential.JWTConfig
	LimitsConfig             queue.LimitsConfig
	JSON                     local_backend.JSONOutput
	Verbose                  local_backend.VerboseOutput
}

func NewBBConfig(workDir string, verbose bool, jsonOutput bool) *BBConfig {
	// Add a hardcoded key for LocalKeyManagerMasterKey
	var key [32]byte
	copy(key[:], "ABCdefghijklmnopqrstuvwxyz123456")

	// Work out file paths
	databaseFilePath := filepath.Join(workDir, "sqlite.db")
	jwtCertDir := filepath.Join(workDir, "jwt-certs")
	jwtCertFilePath := filepath.Join(jwtCertDir, "jwt-cert.pem")
	jwtPrivateKeyFilePath := filepath.Join(jwtCertDir, "jwt-private-key.pem")

	// Listen for API calls on localhost
	localServerAddress := fmt.Sprintf("localhost:%d", DefaultLocalServerPort)
	dynamicAPIEndpoint := dynamic_api.Endpoint(fmt.Sprintf("http://%s", localServerAddress))

	return &BBConfig{
		BBAPIConfig: bb_server.BBAPIServerConfig{
			HTTPServerConfig: server.HTTPServerConfig{
				Address:      localServerAddress,
				TLSConfig:    nil,  // no need for TLS for localhost-based server
				DockerBridge: true, // make the dynamic API available to docker containers
			},
		},
		DatabaseConfig: store.DatabaseConfig{
			ConnectionString:   store.DatabaseConnectionString(fmt.Sprintf("file:%s?cache=shared", databaseFilePath)),
			Driver:             store.Sqlite,
			MaxIdleConnections: store.DefaultDatabaseMaxIdleConnections,
			MaxOpenConnections: store.DefaultDatabaseMaxOpenConnections,
		},
		DatabaseFilePath:         databaseFilePath,
		LocalBlobStoreDir:        blob.LocalBlobStoreDirectory(filepath.Join(workDir, "blob")),
		RunnerLogTempDir:         logging.RunnerLogTempDirectory(filepath.Join(workDir, "log-temp")),
		LogFilePath:              logger.LogFilePath(filepath.Join(workDir, "buildbeaver.log")),
		LocalKeyManagerMasterKey: &key,
		LogServiceConfig:         log.LogServiceConfig{WriterConfig: log.DefaultWriterConfig},
		JSON:                     local_backend.JSONOutput(jsonOutput),
		Verbose:                  local_backend.VerboseOutput(verbose),
		SchedulerConfig: runner.SchedulerConfig{
			PollInterval: runner.DefaultPollInterval,
			ParallelJobs: runner.DefaultParallelBuilds,
		},
		ExecutorConfig: runner.ExecutorConfig{
			IsLocal:            true,
			DynamicAPIEndpoint: dynamicAPIEndpoint,
		},
		JWTConfig: credential.JWTConfig{
			CertificateFile:   certificates.CertificateFile(jwtCertFilePath),
			PrivateKeyFile:    certificates.PrivateKeyFile(jwtPrivateKeyFilePath),
			AutoCreateKeyPair: true,
		},
		LimitsConfig: queue.LimitsConfig{
			MaxBuildConfigLength: queue.DefaultMaxBuildConfigLength,
			MaxJobsPerBuild:      queue.DefaultMaxJobsPerBuild,
			MaxStepsPerJob:       queue.DefaultMaxStepsPerJob,
		},
	}
}
