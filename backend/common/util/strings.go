package util

import "unicode/utf8"

// TruncateStringToMaxLength returns a copy of the string, truncated to be at most maxChars runes long.
// If the string is truncated, the last 3 characters are set to '...' if maxChars is greater than 3.
func TruncateStringToMaxLength(s string, maxChars int) string {
	if utf8.RuneCountInString(s) <= maxChars {
		return s // no need to truncate
	}
	runes := []rune(s)
	if maxChars > 3 {
		return string(runes[:maxChars-3]) + "..."
	} else {
		return string(runes[:maxChars]) // not enough room for "..."
	}
}
