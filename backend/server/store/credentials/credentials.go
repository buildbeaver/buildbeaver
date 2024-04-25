package credentials

import (
	"context"

	"github.com/doug-martin/goqu/v9"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func init() {
	_ = models.MutableResource(&models.Credential{})
	store.MustDBModel(&models.Credential{})
}

type CredentialStore struct {
	table *store.ResourceTable
}

func NewStore(db *store.DB, logFactory logger.LogFactory) *CredentialStore {
	return &CredentialStore{
		table: store.NewResourceTable(db, logFactory, &models.Credential{}),
	}
}

// Create a new credential.
// Returns store.ErrAlreadyExists if a credential with matching unique properties already exists.
func (d *CredentialStore) Create(ctx context.Context, txOrNil *store.Tx, credential *models.Credential) error {
	return d.table.Create(ctx, txOrNil, credential)
}

// Read an existing credential, looking it up by ResourceID.
// Returns models.ErrNotFound if the credential does not exist.
func (d *CredentialStore) Read(ctx context.Context, txOrNil *store.Tx, id models.CredentialID) (*models.Credential, error) {
	credential := &models.Credential{}
	return credential, d.table.ReadByID(ctx, txOrNil, id.ResourceID, credential)
}

// ReadBySharedSecretID reads an existing shared secret credential, looking it up by shared secret ResourceID.
// Returns models.ErrNotFound if the credential does not exist.
func (d *CredentialStore) ReadBySharedSecretID(ctx context.Context, txOrNil *store.Tx, sharedSecretID string) (*models.Credential, error) {
	credential := &models.Credential{}
	err := d.table.ReadWhere(ctx, txOrNil, credential,
		goqu.Ex{
			"credential_type":             models.CredentialTypeSharedSecret,
			"credential_shared_secret_id": sharedSecretID,
		})
	if err != nil {
		return nil, err
	}
	return credential, nil
}

// ReadByGitHubUserID reads an existing GitHub credential, looking it up by the GitHub user id ResourceID.
// Returns models.ErrNotFound if the credential does not exist.
func (d *CredentialStore) ReadByGitHubUserID(ctx context.Context, txOrNil *store.Tx, gitHubUserID int64) (*models.Credential, error) {
	credential := &models.Credential{}
	return credential, d.table.ReadWhere(ctx, txOrNil, credential,
		goqu.Ex{
			"credential_type":           models.CredentialTypeGitHubOAuth,
			"credential_github_user_id": gitHubUserID,
		})
}

// ReadByPublicKey reads an existing client certificate credential, looking it up
// by from the supplied public key. Returns models.ErrNotFound if the credential does not exist.
func (d *CredentialStore) ReadByPublicKey(ctx context.Context, txOrNil *store.Tx, publicKey certificates.PublicKeyData) (*models.Credential, error) {
	hash := publicKey.SHA256Hash()
	credential := &models.Credential{}

	err := d.table.ReadWhere(ctx, txOrNil, credential,
		goqu.Ex{
			"credential_type":                        models.CredentialTypeClientCertificate,
			"credential_client_public_key_asn1_hash": hash,
		})

	if err != nil {
		return nil, err
	}
	return credential, nil
}

// Delete permanently and idempotently deletes a credential.
func (d *CredentialStore) Delete(ctx context.Context, txOrNil *store.Tx, id models.CredentialID) error {
	return d.table.DeleteByID(ctx, txOrNil, id.ResourceID)
}

// ListCredentialsForIdentity returns a list of all credentials for the specified identity ID.
// Use cursor to page through results, if any.
func (d *CredentialStore) ListCredentialsForIdentity(
	ctx context.Context,
	txOrNil *store.Tx,
	identityID models.IdentityID,
	pagination models.Pagination,
) ([]*models.Credential, *models.Cursor, error) {
	credentialSelect := goqu.
		From(d.table.TableName()).
		Select(&models.Credential{}).
		Where(goqu.Ex{"credential_identity_id": identityID})

	var credentials []*models.Credential
	cursor, err := d.table.ListIn(ctx, txOrNil, &credentials, pagination, credentialSelect)
	if err != nil {
		return nil, nil, err
	}
	return credentials, cursor, nil
}
