package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

const (
	timestampStorageFormat = "2006-01-02 15:04:05.999999-07:00"
)

type Time struct {
	time.Time
}

func NewTime(t time.Time) Time {
	// Note: Postgres only handles to Microsecond for times, so we round before we store the value to ensure we
	// do not retrieve a value with different precision (as golang can provide larger precision).
	return Time{Time: t.UTC().Round(time.Microsecond)}
}

func NewTimePtr(t time.Time) *Time {
	newTime := NewTime(t)
	return &newTime
}

func (s *Time) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	// Note: Based on the database used, we can receive different types for this model
	// Postgres returns time.Time and sqlite string
	switch t := src.(type) {
	case time.Time:
		*s = NewTime(t)
	case string:
		str, ok := src.(string)
		if !ok {
			return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
		}
		parsedTime, err := time.Parse(timestampStorageFormat, str)
		if err != nil {
			return errors.Wrap(err, "error parsing time")
		}
		*s = Time{Time: parsedTime.UTC()}
	default:
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}

	return nil
}

// Value converts a time into a format that can be passed to the database, for example in a WHERE clause
// of a query.
func (s Time) Value() (driver.Value, error) {
	return s.Format(timestampStorageFormat), nil
}
