package keypair

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

const (
	keySizeBits      = 4096
	privateKeyHeader = "RSA PRIVATE KEY"
)

type KeyPairService struct{}

func NewKeyPairService() *KeyPairService {
	return &KeyPairService{}
}

// ParsePrivateKey parses a PEM encoded RSA private key.
func (s *KeyPairService) ParsePrivateKey(privateKeyPlaintext []byte) (*rsa.PrivateKey, error) {

	block, next := pem.Decode(privateKeyPlaintext)
	if block == nil {
		return nil, errors.New("error decoding PEM-encoded key 1")
	}

	if len(next) != 0 {
		return nil, errors.New("error decoding PEM-encoded key 2")
	}

	switch block.Type {
	case privateKeyHeader:
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("Unsupported PEM block type: %s", block.Type)
	}
}

// MakeSSHKeyPair makes a 4096bit RSA key pair. The return values are plaintext
// public key bits and plaintext private key bits in a PEM encoded format.
func (s *KeyPairService) MakeSSHKeyPair() ([]byte, []byte, error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, keySizeBits)
	if err != nil {
		return nil, nil, fmt.Errorf("Error generating key pair: %s", err)
	}

	privateKeyBuf := new(bytes.Buffer)
	privateKeyPEM := &pem.Block{
		Type:  privateKeyHeader,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	err = pem.Encode(privateKeyBuf, privateKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("Error encoding private key as PEM: %s", err)
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Error generating public key: %s", err)
	}

	return ssh.MarshalAuthorizedKey(publicKey), privateKeyBuf.Bytes(), nil
}
