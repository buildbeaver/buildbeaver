package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

type Error struct {
	err error
}

func NewError(err error) *Error {
	return &Error{err: err}
}

func (e *Error) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *Error) MarshalJSON() ([]byte, error) {
	if !e.Valid() {
		return json.Marshal(nil)
	}
	return json.Marshal(e.Error())
}

func (e *Error) UnmarshalJSON(data []byte) error {
	var m string
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	if m != "" {
		e.err = errors.New(m)
	}
	return nil
}

func (e *Error) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return errors.Errorf("Expected string; found: %T", src)
	}
	e.err = errors.New(t)
	return nil
}

func (e *Error) Value() (driver.Value, error) {
	if !e.Valid() {
		return nil, nil
	}
	return e.Error(), nil
}

func (e *Error) Valid() bool {
	return e != nil && e.err != nil && e.err.Error() != ""
}
