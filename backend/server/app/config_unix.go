//go:build !windows
// +build !windows

package app

const (
	defaultLocalBlobStoreDir        = "/var/lib/buildbeaver/blob"
	defaultServerCertificateDir     = "/var/lib/buildbeaver/server-certs"
	defaultJWTCertificateDir        = "/var/lib/buildbeaver/jwt-certs"
	defaultInternalRunnerConfigDir  = "/var/lib/buildbeaver/runners"
	defaultSQLiteConnectionString   = "file:/var/lib/buildbeaver/db/sqlite.db?cache=shared"
	defaultGitHubPrivateKeyFilePath = "/var/lib/buildbeaver/github-private-key.pem"
)
