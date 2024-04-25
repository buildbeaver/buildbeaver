package bb

import (
	"errors"
	"fmt"
)

type AccessToken string

func ParseAccessToken(str string) (AccessToken, error) {
	token := AccessToken(str)
	err := token.Validate()
	if err != nil {
		return "", fmt.Errorf("error: invalid access token '%s': %w", str, err)
	}
	return token, nil
}

func (t AccessToken) Validate() error {
	if t == "" {
		return errors.New("error: access token must be set")
	}
	return nil
}

func (t AccessToken) String() string {
	return string(t)
}
