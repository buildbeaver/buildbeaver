package commits_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestCommit(t *testing.T) {

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()

	ctx := context.Background()

	legalEntityA, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")
	repo := server_test.CreateRepo(t, ctx, app, legalEntityA.ID)

	commitA := server_test.CreateCommit(t, ctx, app, repo.ID, legalEntityA.ID)

	t.Run("CreateRunner", testCommitCreate(app.DB, app, commitA))
}

func testCommitCreate(db *store.DB, app *server_test.TestServer, commit *models.Commit) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("Read", testCommitRead(app.CommitStore, commit))
		t.Run("Update", testCommitUpdate(db, app.CommitStore, commit))
	}
}

func testCommitRead(store store.CommitStore, referenceCommit *models.Commit) func(t *testing.T) {
	return func(t *testing.T) {
		commit, err := store.Read(context.Background(), nil, referenceCommit.ID)
		if err != nil {
			t.Fatalf("Error reading commit: %s", err)
		}
		if !commit.ID.Equal(referenceCommit.ID.ResourceID) {
			t.Error("Unexpected ResourceID")
		}
		if commit.CreatedAt != referenceCommit.CreatedAt {
			t.Error("Unexpected CreatedAt")
		}
		if commit.RepoID != referenceCommit.RepoID {
			t.Error("Unexpected RepoID")
		}
		if !bytes.Equal(commit.Config, referenceCommit.Config) {
			t.Error("Unexpected Config")
		}
		if commit.ConfigType != referenceCommit.ConfigType {
			t.Error("Unexpected ConfigType")
		}
		if commit.SHA != referenceCommit.SHA {
			t.Error("Unexpected SHA")
		}
		if commit.Message != referenceCommit.Message {
			t.Error("Unexpected Message")
		}
		if commit.AuthorID != referenceCommit.AuthorID {
			t.Error("Unexpected AuthorID")
		}
		if commit.CommitterID != referenceCommit.CommitterID {
			t.Error("Unexpected CommitterID")
		}
		if commit.Link != referenceCommit.Link {
			t.Error("Unexpected Link")
		}
	}
}

func testCommitUpdate(db *store.DB, commitStore store.CommitStore, referenceCommit *models.Commit) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.WithTx(context.Background(), nil, func(tx *store.Tx) error {
			// Take out a row lock on the commit to test this function
			err := commitStore.LockRowForUpdate(context.Background(), tx, referenceCommit.ID)
			assert.NoError(t, err, "error returned from SelectForUpdate")

			// Change a bit of data and update the commit
			referenceCommit.Message = "A different message"
			err = commitStore.Update(context.Background(), tx, referenceCommit)
			assert.NoError(t, err, "error returned from Commit Store Update()")

			return nil
		})
		assert.NoError(t, err, "Error returned from WithTx()")
	}
}
