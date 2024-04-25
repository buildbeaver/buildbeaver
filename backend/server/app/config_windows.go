//go:build windows
// +build windows

package app

const (
	defaultLocalBlobStoreDir        = "C:\\ProgramData\\buildbeaver\\blob"
	defaultServerCertificateDir     = "C:\\ProgramData\\buildbeaver\\server-certs"
	defaultJWTCertificateDir        = "C:\\ProgramData\\buildbeaver\\jwt-certs"
	defaultInternalRunnerConfigDir  = "C:\\ProgramData\\buildbeaver\\runners"
	defaultSQLiteConnectionString   = "file:C:\\ProgramData\\buildbeaver\\db\\sqlite.db?cache=shared"
	defaultGitHubPrivateKeyFilePath = "C:\\github-private-key.pem"
)
