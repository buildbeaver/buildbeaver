//go:build windows
// +build windows

package proc_lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	BBLockFile     = "C:\\ProgramData\\buildbeaver\\bb-lock-file"
	RunnerLockFile = "C:\\ProgramData\\buildbeaver\\buildbeaver-runner-lock-file"
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

	if _, err := os.Stat(filename); err == nil {
		// If the files exists, we first try to remove it
		if err = os.Remove(filename); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	// Write PID to lock file
	_, err = file.WriteString(strconv.Itoa(os.Getpid()))
	if err != nil {
		return nil, err
	}

	return file, nil
}
