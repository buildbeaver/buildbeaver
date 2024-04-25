package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/bb/app"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util/proc_lock"
	"github.com/buildbeaver/buildbeaver/runner/runtime/docker"
)

// ParseNodeFQNS parses each of the supplied arguments as a Fully Qualified Name identifying a node in the build graph.
// Only workflows can be specified, not "workflow.job.step".
func ParseNodeFQNS(args []string) ([]models.NodeFQN, error) {
	var fqns []models.NodeFQN
	for _, fqnStr := range args {
		fqn := models.NodeFQN{}
		err := fqn.ScanWorkflowOnly(fqnStr)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing %q to workflow FQN", fqnStr)
		}
		fqns = append(fqns, fqn)
	}
	return fqns, nil
}

func HomeifyPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "$HOME") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error locating user home directory: %w", err)
		}
		target := ""
		if path[:2] == "~/" {
			target = "~/"
		}
		if path[:5] == "$HOME" {
			target = "$HOME"
		}
		return filepath.Join(home, path[len(target):]), nil
	}
	return path, nil
}

// CleanUpOldResources cleans up containers and other resources left over from previous runs.
func CleanUpOldResources(bb *app.App, verbose bool) {
	timeout := time.Second * 30

	// Use a special non-standard prefix for Docker names for bb, so the CleanUp function doesn't try to clean up
	// resources belonging to other programs (especially runners)
	docker.SetDockerPrefix(docker.BBDockerPrefix)

	// Make an executor that knows about supported runtimes to clean up
	executor := bb.ExecutorFactory(context.Background())

	err := executor.CleanUp(timeout)
	if err != nil {
		// Log and ignore errors during cleanup
		if verbose {
			fmt.Fprintf(os.Stdout, "Warning: unable to clean up resources. Use --skip-cleanup flag to skip this check.\r\n%s\r\n", err.Error())
		} else {
			fmt.Fprintf(os.Stdout, "Warning: unable to clean up resources. Use --skip-cleanup flag to skip this check, or -v flag to see errors.\r\n")
		}
	}
}

// GetBBFileLock opens the lock file for BB exclusively for write, and returns a file handle.
// The caller should keep the file open for the duration of the command.
// Returns an error if any other instance of BB currently has the lock file open (i.e. is running a command).
func GetBBFileLock() (*os.File, error) {
	return proc_lock.CreateLockFile(proc_lock.BBLockFile)
}
