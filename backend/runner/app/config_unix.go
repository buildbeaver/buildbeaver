//go:build !windows
// +build !windows

package app

const (
	defaultRunnerConfigDir  = "/var/lib/buildbeaver/runners/default"
	defaultRunnerLogTempDir = "/var/lib/buildbeaver/runners/default/" + DefaultRunnerLogTempDirName
)
