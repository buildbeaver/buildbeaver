package models

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	util2 "github.com/buildbeaver/buildbeaver/common/util"
)

const resourceNameMaxLength = 100
const ResourceNameRegexStr = "^[a-zA-Z0-9_-]{1,100}$"

var ResourceNameRegex = regexp.MustCompile(ResourceNameRegexStr)

// ResourceName is a mutable, human-specified identifier of a resource.
// ResourceName must conform to length and character set requirements (see resourceNameMaxLength and ResourceNameRegex).
// ResourceName is unique within a parent collection e.g. a repo's name must be unique within the
// legal entity it belongs to. Names should not be used as persistent references to a resource as
// they are mutable - use ResourceID instead.
type ResourceName string

func (s ResourceName) String() string {
	return string(s)
}

func (s *ResourceName) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return fmt.Errorf("error expected string: %#v", src)
	}
	*s = ResourceName(t)
	return nil
}

func (s ResourceName) Value() (driver.Value, error) {
	return string(s), nil
}

func (s ResourceName) Valid() bool {
	return s.Validate() == nil
}

func (s ResourceName) Validate() error {
	if s == "" {
		return errors.New("error name must be set")
	}
	if len(s) > resourceNameMaxLength {
		return fmt.Errorf("error name must not exceed %d characters", resourceNameMaxLength)
	}
	if !ResourceNameRegex.MatchString(s.String()) {
		return fmt.Errorf("error name must only contain alphanumeric, dash or underscore characters: '%s'", s)
	}
	return nil
}

const (
	replacementChar = "-"
	allowedChars    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUBWXYZ123456789.-_"
	prefixLen       = 10
)

func NormalizeResourceName(str string) string {
	if len(str) > resourceNameMaxLength {
		prefix := util2.RandAlphaString(prefixLen)
		str = prefix + str[:resourceNameMaxLength-prefixLen]
	}
	var out string
	for _, s := range str {
		if !strings.Contains(allowedChars, string(s)) {
			out += replacementChar
		} else {
			out += string(s)
		}
	}
	return out
}

func OptionalResourceName(name string) *ResourceName {
	n := ResourceName(name)
	return &n
}
