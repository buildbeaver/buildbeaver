//go:build !windows
// +build !windows

package proc_lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	BBLockFile     = "/tmp/bb-lock-file"
	RunnerLockFile = "/tmp/buildbeaver-runner-lock-file"
)

// CreateLockFile tries to create a file with given name and acquire an exclusive lock on it.
// If the file already exists AND is still locked then an error is returned.
func CreateLockFile(filename string) (*os.File, error) {
	// Ensure the directory exists that will contain the lock file
	dirName := filepath.Dir(filename)
	err := os.MkdirAll(dirName, 0755) // read and traverse permissions for everyone
	if err != nil {
		return nil, fmt.Errorf("error ensuring directory '%s' exists: %w", dirName, err)
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Write PID to lock file
	contents := strconv.Itoa(os.Getpid())
	if err := file.Truncate(0); err != nil {
		file.Close()
		return nil, err
	}
	if _, err := file.WriteString(contents); err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}
