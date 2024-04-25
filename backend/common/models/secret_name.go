package models

import (
	"fmt"
	"regexp"
)

const SecretNameRegexStr = "^[a-zA-Z0-9_]{1,100}$"

var SecretNameRegex = regexp.MustCompile(SecretNameRegexStr)

// ValidateSecretName checks the supplied resource name to ensure it is a valid secret name.
// Secret names must be valid resource names but are also subject to additional constraints so that they can
// be used as shell environment variable names.
func ValidateSecretName(s ResourceName) error {
	// Validate as a resource name first
	err := s.Validate()
	if err != nil {
		return err
	}

	// Additional constraints on secret names
	if !SecretNameRegex.MatchString(s.String()) {
		return fmt.Errorf("error secret name must only contain alphanumeric or underscore characters: '%s'", s)
	}

	return nil
}
