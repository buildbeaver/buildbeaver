package encryption

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/logger"
)

const (
	// the key spec used to generate keys for the AWSKeyManager
	// this will produce 32 byte keys
	awsKeySpec = "AES_256"
)

type AWSKeyManagerConfig struct {
	Region          string
	MasterKeyID     string
	AccessKeyID     string
	SecretAccessKey string
}

// AWSKeyManager provides an implementation of KeyManager based on Amazon KMS
type AWSKeyManager struct {
	kms    *kms.KMS
	config AWSKeyManagerConfig
	log    logger.Log
}

// NewAWSKeyManager creates an AWSKeyManager configured to use the specified Amazon KMS master key id.
func NewAWSKeyManager(config AWSKeyManagerConfig, logFactory logger.LogFactory) (*AWSKeyManager, error) {
	if config.MasterKeyID == "" {
		return nil, fmt.Errorf("MasterKeyID must be specified")
	}
	log := logFactory("AWSKMSKeyManager")
	cfg := &aws.Config{}
	log.Infof("Using MasterKeyID: %s", config.MasterKeyID)
	if config.Region != "" {
		log.Infof("Using region: %s", config.Region)
		cfg = cfg.WithRegion(config.Region)
	} else {
		log.Info("Using default region")
	}
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		log.Infof("Using static credentials: %s", config.AccessKeyID)
		cfg = cfg.WithCredentials(credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""))
	} else {
		log.Infof("Using default credentials")
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %w", err)
	}
	return &AWSKeyManager{
		kms:    kms.New(sess),
		config: config,
		log:    log,
	}, nil
}

// GenerateDataKey generates a unique data key that can be used encrypt/decrypt
// data. The data key is returned in both a plain text and encrypted format.
func (a *AWSKeyManager) GenerateDataKey(ctx context.Context) (dataKeyPlainText *[32]byte, dataKeyEncrypted []byte, err error) {
	input := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(a.config.MasterKeyID),
		KeySpec: aws.String(awsKeySpec),
	}
	result, err := a.kms.GenerateDataKeyWithContext(ctx, input)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating data key")
	}
	a.log.Infof("Generated data key")
	if len(result.Plaintext) != 32 {
		panic("Plain text data key is not 32 bytes")
	}
	var dataKey [32]byte
	copy(dataKey[:], result.Plaintext)
	return &dataKey, result.CiphertextBlob, nil
}

// DecryptDataKey decrypts a previously generated data key.
func (a *AWSKeyManager) DecryptDataKey(ctx context.Context, dataKeyEncrypted []byte) (dataKeyPlainText *[32]byte, err error) {
	input := &kms.DecryptInput{
		KeyId:          aws.String(a.config.MasterKeyID),
		CiphertextBlob: dataKeyEncrypted,
	}
	result, err := a.kms.DecryptWithContext(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "error decrypting data key")
	}
	a.log.Infof("Decrypted data key")
	if len(result.Plaintext) != 32 {
		panic("Plain text data key is not 32 bytes")
	}
	var dataKey [32]byte
	copy(dataKey[:], result.Plaintext)
	return &dataKey, nil
}
