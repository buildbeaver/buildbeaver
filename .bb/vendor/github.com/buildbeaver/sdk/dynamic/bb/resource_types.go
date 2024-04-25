package bb

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const kindIDSeparator = ":"

type ResourceID string

func (s ResourceID) String() string {
	return string(s)
}

func ParseResourceID(str string) (ResourceID, error) {
	id := ResourceID(str)
	err := id.Validate()
	if err != nil {
		return "", fmt.Errorf("error: invalid id '%s': %w", str, err)
	}
	return id, nil
}

func (s ResourceID) Validate() error {
	parts := strings.Split(s.String(), kindIDSeparator)
	if len(parts) != 2 {
		return fmt.Errorf("error validating resource ID: expected 2 parts in %q, found %d", s, len(parts))
	}
	kindPart := parts[0]
	idPart := parts[1]
	if kindPart == "" {
		return fmt.Errorf("error validating resource ID: kind part of ID is empty")
	}
	if idPart == "" {
		return fmt.Errorf("error validating resource ID: ID part is empty")
	}
	return nil
}

func (s ResourceID) Kind() string {
	parts := strings.Split(s.String(), kindIDSeparator)
	if len(parts) != 2 {
		return ""
	}
	return parts[0] // kind part

}

// ResourceName is a mutable, human-specified identifier of a resource.
// ResourceName must conform to length and character set requirements (see resourceNameMaxLength and resourceNameRegex).
// ResourceName is unique within a parent collection e.g. a repo's name must be unique within the
// legal entity it belongs to. Names should not be used as persistent references to a resource as
// they are mutable - use ResourceID instead.
type ResourceName string

func (s ResourceName) String() string {
	return string(s)
}

func ParseResourceName(str string) (ResourceName, error) {
	name := ResourceName(str)
	err := name.Validate()
	if err != nil {
		return "", fmt.Errorf("error invalid resource name '%s': %w", str, err)
	}
	return name, nil
}

// ParseResourceNames parses a comma-separated list of resource names, validating each one.
func ParseResourceNames(str string) ([]ResourceName, error) {
	strNames := strings.Split(str, ",")
	var names []ResourceName
	for i, nextStr := range strNames {
		nextName, err := ParseResourceName(nextStr)
		if err != nil {
			return names, fmt.Errorf("error parsing resource name list, item %d: %w", i, err)
		}
		names = append(names, nextName)
	}
	return names, nil
}

const resourceNameMaxLength = 100
const ResourceNameRegexStr = "^[a-zA-Z0-9\\._-]{1,100}$"

var resourceNameRegex = regexp.MustCompile(ResourceNameRegexStr)

func (s ResourceName) Validate() error {
	if s == "" {
		return errors.New("error: name must be set")
	}
	if len(s) > resourceNameMaxLength {
		return fmt.Errorf("error name must not exceed %d characters", resourceNameMaxLength)
	}
	if !resourceNameRegex.MatchString(s.String()) {
		return fmt.Errorf("error name must only contain alphanumeric, dash or underscore characters (matching `%s`): %s", ResourceNameRegexStr, s)
	}
	return nil
}
