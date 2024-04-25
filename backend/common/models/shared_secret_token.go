package models

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// SharedSecretToken is a shared secret token that must never leave the server.
type SharedSecretToken struct {
	PublicSharedSecretToken
	salt []byte
}

// PublicSharedSecretToken is a shared secret token that is safe to share publicly with the token owner.
type PublicSharedSecretToken struct {
	id   string
	data []byte
}

// NewSharedSecretToken creates a new shared secret token that must never leave the server.
func NewSharedSecretToken() (*SharedSecretToken, error) {
	// 32 byte cryptographically generated random salt
	salt := [32]byte{}
	_, err := io.ReadFull(rand.Reader, salt[:])
	if err != nil {
		return nil, errors.Wrap(err, "error generating salt")
	}
	// 64 byte cryptographically generated random key
	tokenData := [64]byte{}
	_, err = io.ReadFull(rand.Reader, tokenData[:])
	if err != nil {
		return nil, errors.Wrap(err, "error generating token")
	}
	return &SharedSecretToken{
		PublicSharedSecretToken: PublicSharedSecretToken{
			id:   uuid.New().String(),
			data: tokenData[:],
		},
		salt: salt[:],
	}, nil
}

// NewPublicSharedSecretTokenFromString initializes a public shared secret token from the string
// previously returned by the token's String() function.
func NewPublicSharedSecretTokenFromString(str string) (PublicSharedSecretToken, error) {
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return PublicSharedSecretToken{}, errors.Wrap(err, "error decoding token wrapper")
	}
	parts := strings.SplitN(string(decoded[:]), ":", 2)
	if len(parts) != 2 {
		return PublicSharedSecretToken{}, errors.New("Invalid token format")
	}
	id := parts[0]
	data, err := hex.DecodeString(parts[1])
	if err != nil {
		return PublicSharedSecretToken{}, errors.Wrap(err, "error decoding token")
	}
	return PublicSharedSecretToken{
		id:   id,
		data: data[:],
	}, nil
}

func (m PublicSharedSecretToken) ID() string {
	return m.id
}

// String returns the string representation of the token that can be shared with the token owner.
func (m PublicSharedSecretToken) String() string {
	str := fmt.Sprintf("%s:%s", m.id, hex.EncodeToString(m.data))
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func (m PublicSharedSecretToken) IsValid(salt []byte, hash []byte) (bool, error) {
	computedHash := sha256.Sum256(append(m.data, salt...))
	return bytes.Equal(computedHash[:], hash), nil
}

// Public returns a token that contains parts that are safe to share publicly with the token owner.
func (m *SharedSecretToken) Public() PublicSharedSecretToken {
	return m.PublicSharedSecretToken
}

// PrivateParts returns the hex-encoded salt and hex-encoded hash of the token.
// These parts should be stored by the server and should never be exposed publicly.
// These parts can be provided to the IsValid function of a token created from PublicString to validate it.
func (m *SharedSecretToken) PrivateParts() ([]byte, []byte) {
	hash := sha256.Sum256(append(m.data, m.salt...))
	return m.salt, hash[:]
}
