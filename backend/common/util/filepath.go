package util

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// EscapeFileName escapes each part of the input path and makes it suitable to be used in a filename.
// The returned path is cleaned (which means it is separated using filepath.Separator, regardless of
// if the input path used slashes or filepath.Separator).
func EscapeFileName(path string) string {
	var (
		encoded string
		parts   = strings.Split(filepath.Clean(path), string(filepath.Separator))
	)
	for _, part := range parts {
		enc := url.QueryEscape(part)
		if encoded == "" {
			encoded = enc
		} else {
			encoded = filepath.Join(encoded, enc)
		}
	}
	return encoded
}

// UnescapeFileName unescapes each part of the input path and restores the original part values that were
// passed into EscapeFileName.
// The returned path is cleaned (which means it is separated using filepath.Separator, regardless of
// if the input path used slashes or filepath.Separator).
func UnescapeFileName(path string) (string, error) {
	var (
		decoded string
		parts   = strings.Split(filepath.Clean(path), string(filepath.Separator))
	)
	for _, part := range parts {
		dec, err := url.QueryUnescape(part)
		if err != nil {
			return "", fmt.Errorf("error decoding part %q: %w", part, err)
		}
		if decoded == "" {
			decoded = dec
		} else {
			decoded = filepath.Join(decoded, dec)
		}
	}
	return decoded, nil
}
