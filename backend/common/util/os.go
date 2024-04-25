package util

import (
	"strings"
)

// FilterOSArgs returns args with masked values for all flags not on whitelist.
func FilterOSArgs(args []string, whitelist []string) []string {
	var (
		sanitized           = make([]string, len(args))
		sanitizeNext        = false
		whitelistByFlagName = make(map[string]struct{}, len(whitelist))
	)
	for _, name := range whitelist {
		whitelistByFlagName[name] = struct{}{}
	}
	for i, arg := range args {
		if strings.HasPrefix(arg, "--") {
			if _, ok := whitelistByFlagName[strings.TrimPrefix(strings.ToLower(arg), "--")]; !ok {
				sanitizeNext = true
			} else {
				sanitizeNext = false
			}
			sanitized[i] = arg
		} else {
			if sanitizeNext {
				sanitized[i] = strings.Repeat("*", len(arg))
				sanitizeNext = false
			} else {
				sanitized[i] = arg
			}
		}
	}
	return sanitized
}
