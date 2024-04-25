package app

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	ghub "golang.org/x/oauth2/github"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/dynamic_api"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/blob"
	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/buildbeaver/buildbeaver/server/services/encryption"
	"github.com/buildbeaver/buildbeaver/server/services/log"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/store"
)

const (
	DefaultServerCertFile       = "server-cert.pem"
	DefaultServerPrivateKeyFile = "server-private-key.pem"

	DefaultJWTCertFile       = "jwt-cert.pem"
	DefaultJWTPrivateKeyFile = "jwt-private-key.pem"
)

// LogSafeFlags is a list of flags by name whose values are safe to log.
var LogSafeFlags = []string{
	"key_manager_type",
	"key_manager_aws_kms_master_key_id",
	"key_manager_aws_kms_access_key_id",
	"blob_store_type",
	"blob_store_local_directory",
	"blob_store_aws_s3_bucket_name",
	"blob_store_aws_s3_region",
	"blob_store_aws_s3_access_key_id",
	"api_server_address",
	"api_server_github_auth_redirect_url",
	"dev_api_server_use_same_site_none_mode",
	"runner_api_server_address",
	"runner_api_certificate_directory",
	"runner_api_auto_create_certificate",
	"runner_api_auto_created_certificate_host",
	"runner_api_auto_created_certificate_organization",
	"dynamic_job_api_server_address",
	"dev_internal_runner_config_directory",
	"dev_start_internal_runners",
	"github_app_id",
	"github_app_private_key_file_path",
	"github_client_id",
	"github_app_commit_status_target_url",
	"github_app_deploy_key_name",
	"database_driver",
	"log_levels",
}

type BlobStoreConfig struct {
	// BlobStoreType specifies which blob store should be used.
	BlobStoreType string
	// LocalBlobStoreDir is the base directory on the local filesystem to store blobs to, if enabled.
	LocalBlobStoreDir string
	// S3BlobStoreConfig contains configuration for the S3 blob store, if enabled.
	S3BlobStoreConfig blob.S3BlobStoreConfig
}

func BlobStoreFactory(config BlobStoreConfig, logFactory logger.LogFactory) (services.BlobStore, error) {
	switch strings.ToLower(config.BlobStoreType) {
	case strings.ToLower(blob.AWSS3BlobStoreType.String()):
		return blob.NewS3BlobStore(config.S3BlobStoreConfig, logFactory)
	case strings.ToLower(blob.LocalBlobStoreType.String()):
		return blob.NewLocalBlobStore(blob.LocalBlobStoreDirectory(config.LocalBlobStoreDir)), nil
	default:
		return nil, fmt.Errorf("error unsupported blob store type: %v", config.BlobStoreType)
	}
}

type EncryptionConfig struct {
	// KeyManagerType specifies which key manager should be used.
	KeyManagerType string
	// LocalKeyManagerMasterKey is the static encryption key to use with the local key manager, if enabled.
	LocalKeyManagerMasterKey *[32]byte
	// AWSKeyManagerConfig contains configuration for the AWS Key Manager, if enabled.
	AWSKeyManagerConfig encryption.AWSKeyManagerConfig
}

func KeyManagerFactory(config EncryptionConfig, logFactory logger.LogFactory) (encryption.KeyManager, error) {
	switch strings.ToLower(config.KeyManagerType) {
	case strings.ToLower(encryption.AWSKeyManagerType.String()):
		return encryption.NewAWSKeyManager(config.AWSKeyManagerConfig, logFactory)
	case "", strings.ToLower(encryption.LocalKeyManagerType.String()):
		return encryption.NewLocalKeyManager(config.LocalKeyManagerMasterKey), nil
	default:
		return nil, fmt.Errorf("error unsupported key manager type: %v", config.KeyManagerType)
	}
}

type ServerConfig struct {
	CoreAPIConfig        server.AppAPIServerConfig
	RunnerAPIConfig      server.RunnerAPIServerConfig
	InternalRunnerConfig InternalRunnerConfig
	AuthenticationConfig server.AuthenticationConfig
	DatabaseConfig       store.DatabaseConfig
	GitHubAppConfig      github.AppConfig
	LogLevels            logger.LogLevelConfig
	LogServiceConfig     log.LogServiceConfig
	BlobStoreConfig      BlobStoreConfig
	EncryptionConfig     EncryptionConfig
	JWTConfig            credential.JWTConfig
	LimitsConfig         queue.LimitsConfig
}

func ConfigFromFlags() (*ServerConfig, error) {
	var (
		localKeyManagerMasterKey           string
		databaseDriverStr                  string
		databaseConnectionString           string
		gitHubPrivateKeyFilePath           string
		gitHubPrivateKey                   string
		logLevels                          string
		coreAPISessionAuthenticationKeyStr string
		coreAPISessionEncryptionKeyStr     string
		runnerAPICertDir                   string
		jwtCertDir                         string
		alternateYAMLFilename              string
	)

	// Pre-configure values in the server config
	config := &ServerConfig{
		CoreAPIConfig: server.AppAPIServerConfig{
			HTTPServerConfig: server.HTTPServerConfig{
				DockerBridge: true, // make the Dynamic API (served by the Core API server) available to docker containers
			},
		},
		RunnerAPIConfig: server.RunnerAPIServerConfig{
			HTTPServerConfig: server.HTTPServerConfig{
				TLSConfig:    &server.TLSConfig{},
				DockerBridge: false, // runner API does not need to be available to docker containers
			},
		},
	}

	// Encryption
	flag.StringVar(&config.EncryptionConfig.KeyManagerType, "key_manager_type",
		encryption.LocalKeyManagerType.String(), fmt.Sprintf("The type of key manager to use. Options: %s", strings.Join(encryption.KeyManagerIDs(), ", ")))
	flag.StringVar(&localKeyManagerMasterKey, "key_manager_local_master_key",
		"", "A 256 Bit (32 Byte) key used to encrypt all sensitive data, if using the local key manager.")
	flag.StringVar(&config.EncryptionConfig.AWSKeyManagerConfig.MasterKeyID, "key_manager_aws_kms_master_key_id",
		"", "The KMS Master Key ID to encrypt data with, if using the AWS KMS key manager.")
	flag.StringVar(&config.EncryptionConfig.AWSKeyManagerConfig.AccessKeyID, "key_manager_aws_kms_access_key_id",
		"", "The AWS Access Key ID to use to authenticate to KMS, if using the AWS KMS key manager.")
	flag.StringVar(&config.EncryptionConfig.AWSKeyManagerConfig.SecretAccessKey, "key_manager_aws_kms_secret_key",
		"", "The AWS Secret Key to use to authenticate to KMS, if using the AWS KMS key manager.")

	// JWT Tokens
	flag.StringVar(&jwtCertDir, "jwt_certificate_directory",
		defaultJWTCertificateDir, "The path on the local host containing the private key and public key (certificate) used for signing and verifying JWT tokens.")
	flag.BoolVar((*bool)(&config.JWTConfig.AutoCreateKeyPair), "jwt_auto_create_key_pair",
		false, "True to automatically create a key pair and signing and verifying JWT tokens, if not already configured.")

	// Blob Storage
	flag.StringVar(&config.BlobStoreConfig.BlobStoreType, "blob_store_type",
		blob.LocalBlobStoreType.String(), fmt.Sprintf("The type of blob store to use. Options: %s", strings.Join(blob.BlobStoreTypes(), ", ")))
	flag.StringVar(&config.BlobStoreConfig.LocalBlobStoreDir, "blob_store_local_directory",
		defaultLocalBlobStoreDir, "The path on the local host to store blob files to, if using the local blob store.")
	flag.StringVar(&config.BlobStoreConfig.S3BlobStoreConfig.BucketName, "blob_store_aws_s3_bucket_name",
		"", "The name of the S3 bucket to store blobs to, if using the S3 blob store.")
	flag.StringVar(&config.BlobStoreConfig.S3BlobStoreConfig.Region, "blob_store_aws_s3_region",
		"", "The region of the S3 bucket to store blobs to, if using the S3 blob store.")
	flag.StringVar(&config.BlobStoreConfig.S3BlobStoreConfig.AccessKeyID, "blob_store_aws_s3_access_key_id",
		"", "The AWS Access Key ID to use to authenticate to the S3 bucket, if using the S3 blob store.")
	flag.StringVar(&config.BlobStoreConfig.S3BlobStoreConfig.SecretAccessKey, "blob_store_aws_s3_secret_key",
		"", "The AWS Secret Key to use to authenticate to the S3 bucket, if using the S3 blob store.")

	// App API
	flag.StringVar(&config.CoreAPIConfig.Address, "api_server_address",
		"0.0.0.0:80", "The interface and port to bind the Core API server to.")
	flag.StringVar(&coreAPISessionAuthenticationKeyStr, "api_server_session_authentication_key",
		"", "The 256 Bit key used to authenticate the validity of HTTP(S) session cookies on the Core API")
	flag.StringVar(&coreAPISessionEncryptionKeyStr, "api_server_session_encryption_key",
		"", "The 256 Bit key used to encrypt HTTP(S) session cookies on the Core API")
	flag.StringVar(&config.AuthenticationConfig.GitHub.RedirectURL, "api_server_github_auth_redirect_url",
		"", "The url GitHub will redirect to after authenticating users via OAuth2.")
	flag.BoolVar((*bool)(&config.AuthenticationConfig.UseSameSiteNoneMode), "dev_api_server_use_same_site_none_mode",
		false, "True to set SameSite=none mode when issuing session cookies, so the cookies will be sent along with cross-site requests. This option should not be used in production.")

	// Runner API
	flag.StringVar(&config.RunnerAPIConfig.Address, "runner_api_server_address",
		"0.0.0.0:443", "The interface and port to bind the Runner API server to.")
	flag.StringVar(&runnerAPICertDir, "runner_api_certificate_directory",
		defaultServerCertificateDir, "The path on the local host containing server certificates and private key for the Runner API server.")
	flag.BoolVar((*bool)(&config.RunnerAPIConfig.TLSConfig.AutoCreateCertificate), "runner_api_auto_create_certificate",
		true, "True to automatically create a key pair and server certificate for the Runner API server, if not already configured.")
	flag.StringVar(&config.RunnerAPIConfig.TLSConfig.AutoCreatedCertificateHost, "runner_api_auto_created_certificate_host",
		"runner.changeme.com", "The host to configure in the auto created Runner API server certificate.")
	flag.StringVar(&config.RunnerAPIConfig.TLSConfig.AutoCreatedCertificateOrganization, "runner_api_auto_created_certificate_organization",
		"BuildBeaver Limited", "The organization to configure in the auto created Runner API server certificate.")

	// Internal Runners
	flag.StringVar((*string)(&config.InternalRunnerConfig.ConfigDir), "dev_internal_runner_config_directory",
		defaultInternalRunnerConfigDir, "The path on the local host containing configuration and certificates for internal runners.")
	flag.BoolVar(&config.InternalRunnerConfig.StartInternalRunners, "dev_start_internal_runners",
		false, "True to start internal runners within the server")

	// GitHub
	flag.Int64Var(&config.GitHubAppConfig.AppID, "github_app_id",
		-1, "The GitHub App ID to connect to GitHub as.")
	flag.StringVar(&gitHubPrivateKeyFilePath, "github_app_private_key_file_path",
		defaultGitHubPrivateKeyFilePath, "The path on the local host to the GitHub app private key file")
	flag.StringVar(&config.AuthenticationConfig.GitHub.ClientID, "github_client_id",
		"", "The GitHub Client ID the server will present to GitHub when authenticating browser via OAuth2.")
	flag.StringVar(&config.AuthenticationConfig.GitHub.ClientSecret, "github_client_secret",
		"", "The GitHub Client Secret the server will present to GitHub when authenticating users via OAuth2.")
	flag.StringVar(&config.GitHubAppConfig.CommitStatusTargetURL, "github_app_commit_status_target_url",
		github.DefaultCommitStatusTargetURL, "The base URL to pass to the SCM in commit status updates as the target URL")
	flag.StringVar(&config.GitHubAppConfig.DeployKeyName, "github_app_deploy_key_name",
		"buildbeaver-autogenerated", "The name of the deploy key to install into GitHub repos that are enabled in BuildBeaver.")

	// Database
	flag.StringVar(&databaseConnectionString, "database_connection_string",
		defaultSQLiteConnectionString, "The connection string for the database")
	flag.StringVar(&databaseDriverStr, "database_driver",
		string(store.Sqlite), "The Database Driver to use (i.e sqlite3|postgres)")
	flag.IntVar(&config.DatabaseConfig.MaxIdleConnections, "database_max_idle_connections",
		store.DefaultDatabaseMaxIdleConnections, "The maximum number of idle database connections to use")
	flag.IntVar(&config.DatabaseConfig.MaxOpenConnections, "database_max_open_connections",
		store.DefaultDatabaseMaxOpenConnections, "The maximum number of open database connections to use")

	// Limits
	flag.IntVar(&config.LimitsConfig.MaxBuildConfigLength, "max_build_config_length",
		queue.DefaultMaxBuildConfigLength, "The maximum length of a build configuration, in bytes. This applies to static build definition files and to dynamic builds.")
	flag.IntVar(&config.LimitsConfig.MaxJobsPerBuild, "max_jobs_per_build",
		queue.DefaultMaxJobsPerBuild, "The maximum number of jobs allowed in a single build.")
	flag.IntVar(&config.LimitsConfig.MaxStepsPerJob, "max_steps_per_job",
		queue.DefaultMaxStepsPerJob, "The maximum number of steps allowed in any single job.")

	// Misc
	flag.StringVar(&logLevels, "log_levels",
		"", fmt.Sprintf("A comma separated list of name=level pairs where name is the name of the logger and level is one of: %s", logger.ListLogLevels()))
	flag.StringVar(&alternateYAMLFilename, "dev_alternate_yaml_filename",
		"", fmt.Sprintf("The name of a YAML file to use, if present, in preference to the normal YAML file names."))
	flag.Parse()

	// Encryption
	if config.EncryptionConfig.KeyManagerType == encryption.LocalKeyManagerType.String() {
		if len(localKeyManagerMasterKey) != 32 {
			return nil, errors.New("--key_manager_local_master_key must be set")
		}
		var key [32]byte
		copy(key[:], localKeyManagerMasterKey)
		config.EncryptionConfig.LocalKeyManagerMasterKey = &key
	}

	// Core API
	if len(coreAPISessionAuthenticationKeyStr) != 32 {
		return nil, errors.New("--api_server_session_authentication_key must be 256 Bit (32 Bytes)")
	}
	var sessionAuthenticationKey [32]byte
	copy(sessionAuthenticationKey[:], coreAPISessionAuthenticationKeyStr)
	config.AuthenticationConfig.SessionAuthenticationKey = sessionAuthenticationKey
	if len(coreAPISessionEncryptionKeyStr) != 32 {
		return nil, errors.New("--api_server_session_encryption_key must be 256 Bit (32 Bytes)")
	}
	var sessionEncryptionKey [32]byte
	copy(sessionEncryptionKey[:], coreAPISessionEncryptionKeyStr)
	config.AuthenticationConfig.SessionEncryptionKey = sessionEncryptionKey
	config.AuthenticationConfig.GitHub.Endpoint = ghub.Endpoint

	// Runner API
	config.RunnerAPIConfig.TLSConfig.CertificateFile = certificates.CertificateFile(filepath.Join(runnerAPICertDir, DefaultServerCertFile))
	config.RunnerAPIConfig.TLSConfig.PrivateKeyFile = certificates.PrivateKeyFile(filepath.Join(runnerAPICertDir, DefaultServerPrivateKeyFile))

	// JWT tokens
	config.JWTConfig.CertificateFile = certificates.CertificateFile(filepath.Join(jwtCertDir, DefaultJWTCertFile))
	config.JWTConfig.PrivateKeyFile = certificates.PrivateKeyFile(filepath.Join(jwtCertDir, DefaultJWTPrivateKeyFile))

	// GitHub App
	if gitHubPrivateKeyFilePath != "" {
		config.GitHubAppConfig.PrivateKeyProvider = github.MakeFilePathPrivateKeyProvider(gitHubPrivateKeyFilePath)
	} else if len(gitHubPrivateKey) != 0 {
		config.GitHubAppConfig.PrivateKeyProvider = github.MakeInMemoryPrivateKeyProvider([]byte(gitHubPrivateKey))
	} else {
		config.GitHubAppConfig.PrivateKeyProvider = github.NoPrivateKey
	}

	// Database
	config.DatabaseConfig.Driver = store.DBDriver(databaseDriverStr)
	config.DatabaseConfig.ConnectionString = store.DatabaseConnectionString(databaseConnectionString)

	// Misc
	config.LogLevels = logger.LogLevelConfig(logLevels)
	config.LogServiceConfig = log.LogServiceConfig{WriterConfig: log.DefaultWriterConfig}
	if alternateYAMLFilename != "" {
		// Add alternate to start of the YAMLBuildConfigFileNames list not the end, to make it highest priority
		parser.YAMLBuildConfigFileNames = append([]string{alternateYAMLFilename}, parser.YAMLBuildConfigFileNames...)
	}

	// Dynamic API endpoint for internal runners should refer to the server's core API on localhost
	// Do this after config.CoreAPIConfig.Address is configured via flags
	dynamicAPIEndpoint := "http://localhost"
	port := config.CoreAPIConfig.GetAddressPort()
	if port != "" {
		dynamicAPIEndpoint += ":" + port
	}
	config.InternalRunnerConfig.DynamicAPIEndpoint = dynamic_api.Endpoint(dynamicAPIEndpoint)

	return config, nil
}
