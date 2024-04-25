package runtime

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strings"
)

type OS string

const (
	OSWindows OS = "windows"
	OSLinux   OS = "linux"
	OSMacOS   OS = "macos"
	OSUnknown OS = "unknown"
)

type Shell string

const (
	ShellCMD Shell = "cmd"
	ShellSH  Shell = "sh"
)

func ShellPath(shell Shell) (string, error) {
	switch shell {
	case ShellCMD:
		return "C:\\Windows\\System32\\cmd.exe", nil
	case ShellSH:
		return "/bin/sh", nil
	default:
		return "", fmt.Errorf("error unknown shell: %v", shell)
	}
}

func ShellOrDefault(platform OS, shell *string) string {
	if shell != nil {
		return *shell
	}
	switch platform {
	case OSWindows:
		return "C:\\Windows\\System32\\cmd.exe"
	case OSLinux, OSMacOS:
		return "/bin/sh"
	default:
		log.Panicf("Unsupported OS: %s", platform)
		return "" // Keep compiler happy
	}
}

func GetHostOS() OS {
	os := runtime.GOOS
	switch os {
	case "windows":
		return OSWindows
	case "darwin":
		return OSMacOS
	case "linux":
		return OSLinux
	default:
		return OSUnknown
	}
}

func WriteScript(dir string, name string, commands []string) (string, error) {
	path := filepath.Join(dir, name)
	commandStr := strings.Join(commands, "\n")
	err := ioutil.WriteFile(path, []byte(commandStr), 0755)
	if err != nil {
		return "", fmt.Errorf("error writing script: %w", err)
	}
	return path, nil
}
