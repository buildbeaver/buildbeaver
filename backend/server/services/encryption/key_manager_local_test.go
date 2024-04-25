package encryption

import (
	"bytes"
	"context"
	"testing"
)

func TestLocalKeyManager(t *testing.T) {
	ctx := context.Background()
	var key [32]byte
	copy(key[:], []byte("12345678123456781234567812345678"))
	manager := NewLocalKeyManager(&key)
	for i := 0; i < 100; i++ {
		dataKeyPlainText, dataKeyEncrypted, err := manager.GenerateDataKey(ctx)
		if err != nil {
			t.Errorf("GenerateDataKey(): %s", err)
		}
		dataKeyPlainText2, err := manager.DecryptDataKey(ctx, dataKeyEncrypted)
		if err != nil {
			t.Errorf("DecryptDataKey(): %s", err)
		}
		if !bytes.Equal(dataKeyPlainText[:], dataKeyPlainText2[:]) {
			t.Errorf("plaintexts don't match")
		}
	}
}
