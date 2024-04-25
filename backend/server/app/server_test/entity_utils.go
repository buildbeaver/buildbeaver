package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/certificates"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
)

func CreateCommit(t *testing.T, ctx context.Context, app *TestServer, repoID models.RepoID, legalEntityID models.LegalEntityID) *models.Commit {
	var randomCommit = referencedata.GenerateCommit(repoID, legalEntityID)

	err := app.CommitStore.Create(ctx, nil, randomCommit)
	require.Nil(t, err)

	return randomCommit
}

func CreateAndQueueBuild(t *testing.T, ctx context.Context, app *TestServer, repoID models.RepoID, legalEntityID models.LegalEntityID, ref string) *dto.BuildGraph {
	var randomCommit = CreateCommit(t, ctx, app, repoID, legalEntityID)

	if ref == "" {
		ref = referencedata.TestRef
	}

	build, err := app.QueueService.EnqueueBuildFromCommit(ctx, nil, randomCommit, ref, nil)
	require.NoError(t, err)
	require.Nil(t, build.Error, build.Error.Error())
	require.Equal(t, models.WorkflowStatusQueued, build.Status)

	return build
}

// CreatePersonLegalEntity creates a new legal entity representing a person, for use during a test.
// Returns the new legal entity and its associated identity. Any errors will cause failure of the test.
// If name, legalName or emailAddress are left blank then default values suitable for a person will be used.
func CreatePersonLegalEntity(t *testing.T, ctx context.Context, app *TestServer, name models.ResourceName, legalName string, emailAddress string) (*models.LegalEntity, *models.Identity) {
	legalEntityData := referencedata.GeneratePersonLegalEntity(name, legalName, emailAddress)
	legalEntity, err := app.LegalEntityService.Create(ctx, nil, legalEntityData)
	require.Nil(t, err)
	identity, err := app.LegalEntityService.ReadIdentity(ctx, nil, legalEntity.ID)
	require.Nil(t, err)
	return legalEntity, identity
}

// CreateCompanyLegalEntity creates a new legal entity representing a company, for use during a test.
// Any errors will cause failure of the test.
// If name, legalName or emailAddress are left blank then default values suitable for a company will be used.
func CreateCompanyLegalEntity(t *testing.T, ctx context.Context, app *TestServer, name models.ResourceName, legalName string, emailAddress string) *models.LegalEntity {
	legalEntityData := referencedata.GenerateCompanyLegalEntity(name, legalName, emailAddress)
	legalEntity, err := app.LegalEntityService.Create(ctx, nil, legalEntityData)
	require.Nil(t, err)
	return legalEntity
}

func CreateRepo(t *testing.T, ctx context.Context, app *TestServer, legalEntityId models.LegalEntityID) *models.Repo {
	return CreateNamedRepo(t, ctx, app, "", legalEntityId)
}

func CreateNamedRepo(t *testing.T, ctx context.Context, app *TestServer, repoName string, legalEntityId models.LegalEntityID) *models.Repo {
	repo := referencedata.GenerateRepo(repoName, legalEntityId)
	_, _, err := app.RepoService.Upsert(ctx, nil, repo)
	require.Nil(t, err)

	return repo
}

func CreateRunner(t *testing.T, ctx context.Context, app *TestServer, name models.ResourceName, legalEntityId models.LegalEntityID, clientCert certificates.CertificateData) *models.Runner {
	now := time.Now()

	if name == "" {
		name = "test-runner"
	}

	runner := models.NewRunner(
		models.NewTime(now),
		name,
		legalEntityId,
		"1.2.3",
		"",
		"",
		models.JobTypes{models.JobTypeDocker, models.JobTypeExec},
		nil, // no labels need to be specified
		true,
	)

	err := app.RunnerService.Create(ctx, nil, runner, clientCert)
	require.Nil(t, err)

	return runner
}

func CreateSecret(t *testing.T, ctx context.Context, app *TestServer, repoID models.RepoID, name models.ResourceName) *models.Secret {
	now := time.Now()

	if name == "" {
		name = "test_secret"
	}

	var randomSecret = &models.Secret{
		ID:               models.NewSecretID(),
		Name:             name,
		CreatedAt:        models.NewTime(now),
		UpdatedAt:        models.NewTime(now),
		KeyEncrypted:     []byte{9, 154, 150, 92, 144, 151, 192, 2, 197, 172, 73, 133, 187, 56, 41, 197, 110, 78, 165, 100, 167, 120, 4, 185, 157, 145, 89, 112, 245, 22, 197, 74, 198, 219, 61, 36},
		ValueEncrypted:   []byte{136, 223, 103, 74, 76, 47, 73, 203, 194, 198, 113, 175, 184, 49, 119, 109, 227, 163, 247, 32, 15, 96, 174, 38, 212, 233, 154, 226, 28, 5, 49, 139, 32, 77, 136, 138, 68, 52, 209},
		DataKeyEncrypted: []byte{152, 78, 223, 173, 64, 147, 46, 56, 5, 28, 178, 80, 75, 38, 5, 192, 153, 240, 212, 64, 252, 206, 201, 52, 240, 83, 160, 79, 218, 40, 144, 9, 3, 65, 183, 76, 17, 44, 9, 21, 6, 71, 118, 16, 206, 112, 40, 46, 210, 44, 217, 87, 237, 182, 155, 111, 54, 170, 10, 205},
		IsInternal:       false,
		RepoID:           repoID,
	}

	err := app.SecretStore.Create(ctx, nil, randomSecret)
	require.Nil(t, err)

	return randomSecret
}
