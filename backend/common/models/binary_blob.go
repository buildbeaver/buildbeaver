package models

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
)

// BinaryBlob exists to make up for shortcomings in the goqu library where it handles
// binary data completely wrong. Here we implement Valuer and Scanner to automagically
// convert to/from a hex-encoded blob which goqu can handle.
type BinaryBlob []byte

func (m *BinaryBlob) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	// Note: Based on the database used, we can receive different types for this model
	// Postgres returns []uint8 and sqlite string
	var str string
	switch t := src.(type) {
	case []uint8: // postgres
		str = string(t)
	case string: // sqlite
		str = t
	default:
		return fmt.Errorf("error unsupported type: %[1]T (%[1]v)", src)
	}
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return fmt.Errorf("error decoding hex: %w", err)
	}
	*m = decoded
	return nil
}

func (m BinaryBlob) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return hex.EncodeToString(m), nil
}
