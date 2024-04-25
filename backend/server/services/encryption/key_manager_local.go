package encryption

import (
	"context"

	"github.com/pkg/errors"
)

type LocalKeyManagerMasterKey *[32]byte

// LocalKeyManager provides an implementation of KeyManager based on an in-memory
// encryption key. This has significant limitations and an HSM-backed external
// provider is preferred.
type LocalKeyManager struct {
	encryptionKey *[32]byte
}

// NewLocalKeyManager creates a LocalKeyManager configured to use the specified
// key. Think very carefully about using this.
func NewLocalKeyManager(encryptionKey LocalKeyManagerMasterKey) *LocalKeyManager {
	return &LocalKeyManager{
		encryptionKey: encryptionKey,
	}
}

// GenerateDataKey generates a unique data key that can be used to encrypt/decrypt
// data. The data key is returned in both a plain text and encrypted format.
func (a *LocalKeyManager) GenerateDataKey(ctx context.Context) (dataKeyPlainText *[32]byte, dataKeyEncrypted []byte, err error) {
	plainTextDataKey := newEncryptionKey()
	encryptedDataKey, err := encrypt(plainTextDataKey[:], a.encryptionKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error encrypting data key")
	}
	return plainTextDataKey, encryptedDataKey, nil
}

// DecryptDataKey decrypts a previously generated data key.
func (a *LocalKeyManager) DecryptDataKey(ctx context.Context, dataKeyEncrypted []byte) (dataKeyPlainText *[32]byte, err error) {
	plainTextDataKey, err := decrypt(dataKeyEncrypted, a.encryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "error decrypting data key")
	}
	var dataKey [32]byte
	copy(dataKey[:], plainTextDataKey)
	return &dataKey, nil
}
