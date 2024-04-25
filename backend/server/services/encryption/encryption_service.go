package encryption

import (
	"context"

	"github.com/pkg/errors"
)

// EncryptionService provides public functions for securely encrypting and decrypting data
// using a KeyManager.
type EncryptionService struct {
	keyManager KeyManager
}

// NewEncryptionService creates an EncryptionService configured to source keys from the provided KeyManager.
func NewEncryptionService(keyManager KeyManager) *EncryptionService {
	return &EncryptionService{
		keyManager: keyManager,
	}
}

// Encrypt plainTextData using the configured KeyManager.
func (e *EncryptionService) Encrypt(ctx context.Context, plainTextData []byte) (encryptedData []byte, encryptedDataKey []byte, err error) {
	dataKeyPlainText, dataKeyEncrypted, err := e.keyManager.GenerateDataKey(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating data key")
	}
	dataEncrypted, err := encrypt(plainTextData, dataKeyPlainText)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error encrypting data")
	}
	return dataEncrypted, dataKeyEncrypted, nil
}

// EncryptMulti encrypts each plainTextData using the same data key and returns an array
// of encrypted datas in the same order they were provided.
func (e *EncryptionService) EncryptMulti(ctx context.Context, plainTextData ...[]byte) (encryptedData [][]byte, encryptedDataKey []byte, err error) {
	dataKeyPlainText, dataKeyEncrypted, err := e.keyManager.GenerateDataKey(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating data key")
	}
	encrypted := make([][]byte, len(plainTextData))
	for i, data := range plainTextData {
		dataEncrypted, err := encrypt(data, dataKeyPlainText)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error encrypting data")
		}
		encrypted[i] = dataEncrypted
	}
	return encrypted, dataKeyEncrypted, nil
}

// Decrypt the encrypted data using the configured KeyManager.
func (e *EncryptionService) Decrypt(ctx context.Context, encryptedData []byte, encryptedDataKey []byte) (plainTextData []byte, err error) {
	dataKeyPlainText, err := e.keyManager.DecryptDataKey(ctx, encryptedDataKey)
	if err != nil {
		return nil, errors.Wrap(err, "error decrypting data key")
	}
	plainTextData, err = decrypt(encryptedData, dataKeyPlainText)
	if err != nil {
		return nil, errors.Wrap(err, "error decrypting data")
	}
	return plainTextData, nil
}
