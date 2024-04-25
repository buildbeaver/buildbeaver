package models

import (
	"database/sql/driver"
	"strings"

	"github.com/pkg/errors"
)

const (
	HashTypeBlake2b HashType = "BLAKE2B"
	HashTypeSHA1    HashType = "SHA1"
	HashTypeSHA256  HashType = "SHA256"
	HashTypeMD5     HashType = "MD5"
	HashTypeFNV     HashType = "FNV"
)

type HashType string

func (s HashType) Valid() bool {
	return s == HashTypeBlake2b || s == HashTypeSHA1 || s == HashTypeSHA256 || s == HashTypeMD5 || s == HashTypeFNV
}

func (s HashType) String() string {
	return string(s)
}

func (s *HashType) Scan(src interface{}) error {
	if src == nil {
		return errors.New("error cannot convert nil to HashType")
	}
	t, ok := src.(string)
	if !ok {
		return errors.Errorf("error expected string but found: %T", src)
	}
	switch strings.ToUpper(t) {
	case "":
		return nil
	case string(HashTypeBlake2b):
		*s = HashTypeBlake2b
	case string(HashTypeSHA1):
		*s = HashTypeSHA1
	case string(HashTypeSHA256):
		*s = HashTypeSHA256
	case string(HashTypeMD5):
		*s = HashTypeMD5
	case string(HashTypeFNV):
		*s = HashTypeFNV
	default:
		return errors.Errorf("error unknown hash type: %s", t)
	}
	return nil
}

func (s HashType) Value() (driver.Value, error) {
	return string(s), nil
}
