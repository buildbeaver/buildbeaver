package models

import (
	"database/sql/driver"
	"fmt"
)

type ResourceKind string

func (s ResourceKind) String() string {
	return string(s)
}

func (s *ResourceKind) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return fmt.Errorf("error expected string: %#v", src)
	}
	*s = ResourceKind(t)
	return nil
}

func (s ResourceKind) Value() (driver.Value, error) {
	return string(s), nil
}
