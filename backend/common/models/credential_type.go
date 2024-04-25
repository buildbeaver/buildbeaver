package models

import (
	"database/sql/driver"
	"strings"

	"github.com/pkg/errors"
)

const (
	CredentialTypeSharedSecret      CredentialType = "shared_secret"
	CredentialTypeGitHubOAuth       CredentialType = "github_oauth"
	CredentialTypeClientCertificate CredentialType = "client_certificate"
	CredentialTypeJWT               CredentialType = "jwt"
)

type CredentialType string

func (s CredentialType) Valid() bool {
	return s == CredentialTypeSharedSecret || s == CredentialTypeGitHubOAuth || s == CredentialTypeClientCertificate
}

func (s CredentialType) String() string {
	return string(s)
}

func (s *CredentialType) Scan(src interface{}) error {
	if src == nil {
		return errors.New("Cannot convert nil to credential type")
	}
	t := src.(string)
	switch strings.ToLower(t) {
	case string(CredentialTypeSharedSecret):
		*s = CredentialTypeSharedSecret
	case string(CredentialTypeGitHubOAuth):
		*s = CredentialTypeGitHubOAuth
	case string(CredentialTypeClientCertificate):
		*s = CredentialTypeClientCertificate
	case string(CredentialTypeJWT):
		*s = CredentialTypeJWT
	default:
		return errors.Errorf("Unsupported credential type: %s", t)
	}
	return nil
}

func (s CredentialType) Value() (driver.Value, error) {
	return string(s), nil
}
