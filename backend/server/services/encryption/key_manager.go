package encryption

import (
	"context"
)

const (
	AWSKeyManagerType   KeyManagerID = "AWS_KMS"
	LocalKeyManagerType KeyManagerID = "LOCAL"
)

type KeyManagerID string

func (s KeyManagerID) String() string {
	return string(s)
}

func KeyManagerIDs() []string {
	return []string{AWSKeyManagerType.String(), LocalKeyManagerType.String()}
}

// KeyManager provides an interface for generating/managing encryption keys for
// wrapped encryption.
type KeyManager interface {
	// GenerateDataKey generates a unique data key that can be used encrypt/decrypt
	// data. The data key is returned in both a plain text and encrypted format.
	GenerateDataKey(ctx context.Context) (dataKeyPlainText *[32]byte, dataKeyEncrypted []byte, err error)
	// DecryptDataKey decrypts a previously generated data key.
	DecryptDataKey(ctx context.Context, dataKeyEncrypted []byte) (dataKeyPlainText *[32]byte, err error)
}
