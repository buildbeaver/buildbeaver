package server_test

import (
	"path/filepath"
	"testing"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/app"
	"github.com/buildbeaver/buildbeaver/server/services/blob"
	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/buildbeaver/buildbeaver/server/services/encryption"
	"github.com/buildbeaver/buildbeaver/server/services/log"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github/github_test_utils"
)

func TestConfig(t *testing.T) *app.ServerConfig {
	// Create a temp directory for configuration, including certificates
	configDir := t.TempDir()
	// Store blobs in another temporary directory
	blobDir := t.TempDir()

	test256bitKeyStr := "abcdefghijklmnopqrstuvwxyz123456"
	var test256bitKey [32]byte
	copy(test256bitKey[:], test256bitKeyStr)

	return &app.ServerConfig{
		EncryptionConfig: app.EncryptionConfig{
			KeyManagerType:           encryption.LocalKeyManagerType.String(),
			LocalKeyManagerMasterKey: &test256bitKey,
		},
		BlobStoreConfig: app.BlobStoreConfig{
			BlobStoreType:     blob.LocalBlobStoreType.String(),
			LocalBlobStoreDir: blobDir,
		},
		CoreAPIConfig: server.AppAPIServerConfig{
			HTTPServerConfig: server.HTTPServerConfig{
				Address:      "",    // Test is expected to use httptest server which picks its own address
				DockerBridge: false, // do not listen on the docker bridge network when running tests
			},
		},
		RunnerAPIConfig: server.RunnerAPIServerConfig{
			HTTPServerConfig: server.HTTPServerConfig{
				Address: "", // Test is expected to use httptest server which picks its own address
				TLSConfig: &server.TLSConfig{
					CertificateFile:       certificates.CertificateFile(filepath.Join(configDir, app.DefaultServerCertFile)),
					PrivateKeyFile:        certificates.PrivateKeyFile(filepath.Join(configDir, app.DefaultServerPrivateKeyFile)),
					AutoCreateCertificate: true,
				},
				DockerBridge: false, // runner API does not need to be available to docker containers
			},
		},
		InternalRunnerConfig: app.InternalRunnerConfig{
			StartInternalRunners: false,
		},
		AuthenticationConfig: server.AuthenticationConfig{
			SessionAuthenticationKey: test256bitKey,
			SessionEncryptionKey:     test256bitKey,
			UseSameSiteNoneMode:      false, // no browsers should be accessing this test server
		},
		GitHubAppConfig: github.AppConfig{
			AppID:                 github_test_utils.GithubTestAppID,
			PrivateKeyProvider:    github_test_utils.TestAccountAppPrivateKey,
			CommitStatusTargetURL: github.DefaultCommitStatusTargetURL,
		},
		LogServiceConfig: log.LogServiceConfig{WriterConfig: log.DefaultWriterConfig},
		LogLevels:        "",
		JWTConfig: credential.JWTConfig{
			CertificateFile:   certificates.CertificateFile(filepath.Join(configDir, app.DefaultJWTCertFile)),
			PrivateKeyFile:    certificates.PrivateKeyFile(filepath.Join(configDir, app.DefaultJWTPrivateKeyFile)),
			AutoCreateKeyPair: true,
		},
		LimitsConfig: queue.LimitsConfig{
			MaxBuildConfigLength: queue.DefaultMaxBuildConfigLength,
			MaxJobsPerBuild:      queue.DefaultMaxJobsPerBuild,
			MaxStepsPerJob:       queue.DefaultMaxStepsPerJob,
		},
	}
}
