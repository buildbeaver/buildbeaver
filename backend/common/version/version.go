package version

import "fmt"

// VERSION indicates the major.minor.patch version the binary was built off of.
var VERSION string

// GITCOMMIT indicates which git hash (12char) the binary was built off of.
var GITCOMMIT string

func VersionToString() string {
	// Don't return a version if they haven't been injected
	if VERSION == "" && GITCOMMIT == "" {
		return ""
	}
	return fmt.Sprintf("%s - %s", VERSION, GITCOMMIT)
}
