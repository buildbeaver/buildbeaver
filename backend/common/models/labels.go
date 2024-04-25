package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type Label string

func (s Label) String() string {
	return string(s)
}

func (s *Label) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return fmt.Errorf("error expected string: %#v", src)
	}
	*s = Label(t)
	return nil
}

func (s Label) Value() (driver.Value, error) {
	return string(s), nil
}

func (s Label) Valid() bool {
	return s.Validate() == nil
}

func (s Label) Validate() error {
	if s == "" {
		return errors.New("label value must be set")
	}
	if len(s) > resourceNameMaxLength {
		return fmt.Errorf("error label must not exceed %d characters", resourceNameMaxLength)
	}
	if !ResourceNameRegex.MatchString(s.String()) {
		return fmt.Errorf("error label must only contain alphanumeric, dash or underscore characters (matching `%s`): %s", ResourceNameRegexStr, s)
	}
	return nil
}

type Labels []Label

func (m *Labels) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m Labels) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
