package models

import (
	"database/sql/driver"
	"strings"

	"github.com/pkg/errors"
)

const (
	LegalEntityTypeCompany LegalEntityType = "company"
	LegalEntityTypePerson  LegalEntityType = "person"
)

type LegalEntityType string

func (s LegalEntityType) Valid() bool {
	return s == LegalEntityTypeCompany || s == LegalEntityTypePerson
}

func (s LegalEntityType) String() string {
	return string(s)
}

func (s *LegalEntityType) Scan(src interface{}) error {
	if src == nil {
		return errors.New("Cannot convert nil to legal entity type")
	}
	t := src.(string)
	switch strings.ToLower(t) {
	case string(LegalEntityTypeCompany):
		*s = LegalEntityTypeCompany
	case string(LegalEntityTypePerson):
		*s = LegalEntityTypePerson
	default:
		return errors.Errorf("Unsupported legal entity type: %s", t)
	}
	return nil
}

func (s LegalEntityType) Value() (driver.Value, error) {
	return string(s), nil
}
