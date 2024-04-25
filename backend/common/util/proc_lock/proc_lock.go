package proc_lock

import (
	"io/ioutil"
	"strconv"
)

// GetLockFilePid returns the PID of the process currently holding the lock defined by filename, or zero if
// no process currently holds the lock.
func GetLockFilePid(filename string) (pid int, err error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	pid, err = strconv.Atoi(string(contents))
	return pid, err
}
