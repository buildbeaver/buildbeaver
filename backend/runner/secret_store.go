package runner

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
)

type SecretStore struct {
	repoID                models.RepoID
	apiClient             APIClient
	secretsPlaintext      []*models.SecretPlaintext
	secretsPlaintextByKey map[string]*models.SecretPlaintext
}

func NewSecretStore(apiClient APIClient, repoID models.RepoID) *SecretStore {
	return &SecretStore{
		repoID:                repoID,
		apiClient:             apiClient,
		secretsPlaintextByKey: map[string]*models.SecretPlaintext{},
	}
}

// Init loads all secrets for the configured repo into memory.
// Any existing secrets in the store are overwritten, so call this before calling AddSecret().
// TODO this should take a build ID, and we should get the restricted set of secrets for the build
//
//	encrypted with the runner's public key (which we will need to subsequently decrypt)
func (b *SecretStore) Init(ctx context.Context) error {
	secrets, err := b.apiClient.GetSecretsPlaintext(ctx, b.repoID)
	if err != nil {
		return errors.Wrap(err, "error getting secrets")
	}
	b.secretsPlaintextByKey = make(map[string]*models.SecretPlaintext)
	for _, secret := range secrets {
		b.secretsPlaintextByKey[b.makeSecretKey(secret.Key, secret.IsInternal)] = secret
	}
	b.secretsPlaintext = secrets
	return nil
}

// AddSecret adds a new secret to the store. The value for this secret will be redacted from logs
// and generally not disclosed.
func (b *SecretStore) AddSecret(secret *models.SecretPlaintext) {
	b.secretsPlaintext = append(b.secretsPlaintext, secret)
	b.secretsPlaintextByKey[b.makeSecretKey(secret.Key, secret.IsInternal)] = secret
}

// GetSecret looks up a secret for the configured repo. If the secret does not exist an error is returned.
// Init must be called prior to calling this function.
func (b *SecretStore) GetSecret(name string, isInternal bool) (*models.SecretPlaintext, error) {
	secret, ok := b.secretsPlaintextByKey[b.makeSecretKey(name, isInternal)]
	if !ok {
		return nil, errors.New("Secret does not exist")
	}
	return secret, nil
}

// GetAllSecrets returns all secrets for the configured repo.
// Init must be called prior to calling this function.
func (b *SecretStore) GetAllSecrets() []*models.SecretPlaintext {
	return b.secretsPlaintext
}

// makeSecretKey makes a key that can uniquely identify a secret in a map.
func (b *SecretStore) makeSecretKey(name string, isInternal bool) string {
	return fmt.Sprintf("%s:%v", name, isInternal)
}
