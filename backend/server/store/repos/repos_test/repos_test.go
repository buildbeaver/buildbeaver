package repos_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/buildbeaver/server/store"
)

func TestRepo(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}
	defer cleanup()

	ctx := context.Background()

	legalEntityA, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "", "", "")

	// Create using our util
	repo := server_test.CreateRepo(t, ctx, app, legalEntityA.ID)
	t.Run("Read", testRepoRead(app.RepoStore, repo.ID, repo))
}

func testRepoRead(store store.RepoStore, testRepoID models.RepoID, referenceRepo *models.Repo) func(t *testing.T) {
	return func(t *testing.T) {
		repo, err := store.Read(context.Background(), nil, testRepoID)
		if err != nil {
			t.Fatalf("Error reading repo: %s", err)
		}
		if repo.ID != referenceRepo.ID {
			t.Error("Unexpected ResourceID")
		}
		if repo.CreatedAt != referenceRepo.CreatedAt {
			t.Error("Unexpected CreatedAt")
		}
		if repo.UpdatedAt != referenceRepo.UpdatedAt {
			t.Error("Unexpected UpdatedAt")
		}
		if repo.DeletedAt != referenceRepo.DeletedAt {
			t.Error("Unexpected DeletedAt")
		}
		if repo.LegalEntityID != referenceRepo.LegalEntityID {
			t.Error("Unexpected UserID")
		}
		if repo.ExternalID.ResourceID != referenceRepo.ExternalID.ResourceID {
			t.Error("Unexpected ResourceID")
		}
		if repo.ExternalID.ExternalSystem != referenceRepo.ExternalID.ExternalSystem {
			t.Error("Unexpected ExternalSystem")
		}
		if repo.Name != referenceRepo.Name {
			t.Error("Unexpected Key")
		}
		if repo.Description != referenceRepo.Description {
			t.Error("Unexpected Desc")
		}
		if repo.ExternalID == nil {
			t.Error("Unexpected ExternalResourceID is nil")
		}
		if repo.ExternalID.ExternalSystem != referenceRepo.ExternalID.ExternalSystem {
			t.Error("Unexpected ExternalResourceID.ExternalSystem")
		}
		if repo.ExternalID.ResourceID != referenceRepo.ExternalID.ResourceID {
			t.Error("Unexpected ExternalResourceID.ResourceID")
		}
		if repo.SSHURL != referenceRepo.SSHURL {
			t.Error("Unexpected SSHURL")
		}
		if repo.HTTPURL != referenceRepo.HTTPURL {
			t.Error("Unexpected HTTPURL")
		}
		if repo.Link != referenceRepo.Link {
			t.Error("Unexpected Link")
		}
		if repo.DefaultBranch != referenceRepo.DefaultBranch {
			t.Error("Unexpected DefaultBranch")
		}
	}
}

func TestSearchRepoAccessControl(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}
	defer cleanup()

	ctx := context.Background()
	now := models.NewTime(time.Now())

	legalEntityA, identityA := server_test.CreatePersonLegalEntity(t, ctx, app, "a", "Family Guy", "pg@fg.com")
	legalEntityB, identityB := server_test.CreatePersonLegalEntity(t, ctx, app, "b", "Stewie Griffin", "sg@fg.com")
	legalEntityC, identityC := server_test.CreatePersonLegalEntity(t, ctx, app, "c", "Brian Griffin", "bg@fg.com")

	// NOTE repo names give us a deterministic sort order leveraged later in the test
	repoA := server_test.CreateNamedRepo(t, ctx, app, "a", legalEntityA.ID)
	repoB := server_test.CreateNamedRepo(t, ctx, app, "b", legalEntityB.ID)
	repoC := server_test.CreateNamedRepo(t, ctx, app, "c", legalEntityC.ID)

	// A search without access control should find all three repos
	res, _, err := app.RepoStore.Search(ctx, nil, models.NoIdentity, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 3)

	// Each legal entity created through the legal entity service gets full permissions over things it owns
	// (default grant set) via a direct grant. Here we prove that each legal entity can only see the one repo
	// it owns
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoA.ID, res[0].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityB.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoB.ID, res[0].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityC.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoC.ID, res[0].ID)

	// A searcher shouldn't be able to subvert access control by filtering on a specific legal entity
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, search.NewRepoQueryBuilder().WhereLegalEntityID(search.Equal, legalEntityB.ID).Compile())
	require.Nil(t, err)
	require.Len(t, res, 0)

	// Create a "viewer" group with a single grant that enables members to read all repos owned by legal entity A
	group := models.NewGroup(now, legalEntityA.ID, "viewer", "", false, nil)
	err = app.GroupStore.Create(ctx, nil, group)
	require.Nil(t, err)
	err = app.GrantStore.Create(ctx, nil,
		models.NewGroupGrant(now, legalEntityA.ID, group.ID, *models.RepoReadOperation, legalEntityA.ID.ResourceID))
	require.Nil(t, err)

	// Add legal entity B to the group
	_, err = app.GroupMembershipStore.Create(ctx, nil,
		models.NewGroupMembershipData(group.ID, identityB.ID, models.TestsSystem, legalEntityA.ID))
	require.Nil(t, err)

	// We should find that legal entity A and C's access has not changed, but legal entity B should now have access
	// to A's repo (plus its own)
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoA.ID, res[0].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityB.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 2)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoA.ID, res[0].ID)
	require.Equal(t, repoB.ID, res[1].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityC.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoC.ID, res[0].ID)

	// Now give legal entity C a direct grant to legal entity B's repo
	err = app.GrantStore.Create(ctx, nil, models.NewIdentityGrant(now, legalEntityB.ID, identityC.ID, *models.RepoReadOperation, repoB.ID.ResourceID))
	require.Nil(t, err)

	// We should find that legal entity A and B's access has not changed, but legal entity C should now have access
	// to B's repo (plus its own)
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoA.ID, res[0].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityB.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 2)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoA.ID, res[0].ID)
	require.Equal(t, repoB.ID, res[1].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityC.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 2)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoB.ID, res[0].ID)
	require.Equal(t, repoC.ID, res[1].ID)

	// Now add legal entity C to the group created above (giving access to legal entity A's repos)
	_, err = app.GroupMembershipStore.Create(ctx, nil, models.NewGroupMembershipData(group.ID, identityC.ID, models.TestsSystem, legalEntityA.ID))
	require.Nil(t, err)

	// We should find that legal entity A and B's access has not changed, but legal entity C should now have access
	// to A's repo (via the group), (as well as B's repo via the direct grant, plus its own)
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoA.ID, res[0].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityB.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 2)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoA.ID, res[0].ID)
	require.Equal(t, repoB.ID, res[1].ID)

	res, _, err = app.RepoStore.Search(ctx, nil, identityC.ID, search.NewRepoQueryBuilder().Compile())
	require.Nil(t, err)
	require.Len(t, res, 3)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoA.ID, res[0].ID)
	require.Equal(t, repoB.ID, res[1].ID)
	require.Equal(t, repoC.ID, res[2].ID)

	// A legal entity that has access to repos belonging to other legal entities should
	// be able to filter to any one of those legal entities.
	res, _, err = app.RepoStore.Search(ctx, nil, identityC.ID, search.NewRepoQueryBuilder().WhereLegalEntityID(search.Equal, legalEntityB.ID).Compile())
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoB.ID, res[0].ID)
}

func TestRepoSearch(t *testing.T) {
	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}
	defer cleanup()

	ctx := context.Background()
	now := models.NewTime(time.Now())

	legalEntityA, err := app.LegalEntityService.Create(ctx, nil, models.NewCompanyLegalEntityData("a", "Family Guy", "pg@fg.com", nil, ""))
	require.Nil(t, err)
	identityA, err := app.LegalEntityService.ReadIdentity(ctx, nil, legalEntityA.ID)
	require.NoError(t, err)

	legalEntityB, err := app.LegalEntityService.Create(ctx, nil, models.NewPersonLegalEntityData("b", "Stewie Griffin", "sg@fg.com", nil, ""))
	require.Nil(t, err)

	legalEntityC, err := app.LegalEntityService.Create(ctx, nil, models.NewPersonLegalEntityData("c", "Brian Griffin", "bg@fg.com", nil, ""))
	require.Nil(t, err)

	// NOTE repo names give us a deterministic sort order leveraged later in the test

	repoAExternalID := models.NewExternalResourceID("github", "a")
	repoA := models.NewRepo(now, "a", legalEntityA.ID, "", "", "https://github.com/a", "", "master", true, false, nil, &repoAExternalID, "")
	_, _, err = app.RepoService.Upsert(ctx, nil, repoA)
	require.Nil(t, err)

	repoBExternalID := models.NewExternalResourceID("github", "b")
	repoB := models.NewRepo(now, "b", legalEntityB.ID, "", "", "https://github.com/b", "", "master", true, false, nil, &repoBExternalID, "")
	_, _, err = app.RepoService.Upsert(ctx, nil, repoB)
	require.Nil(t, err)

	repoCExternalID := models.NewExternalResourceID("github", "c")
	repoC := models.NewRepo(now, "c", legalEntityC.ID, "", "", "https://github.com/c", "", "master", true, false, nil, &repoCExternalID, "")
	_, _, err = app.RepoService.Upsert(ctx, nil, repoC)
	require.Nil(t, err)

	repoXExternalID := models.NewExternalResourceID("github", "xenu")
	repoX := models.NewRepo(now, "github-xenu", legalEntityC.ID, "", "", "https://github.com/github-xenu", "", "master", true, true, nil, &repoXExternalID, "")
	_, _, err = app.RepoService.Upsert(ctx, nil, repoX)
	require.Nil(t, err)

	query := search.NewRepoQueryBuilder().
		WhereUser(search.Equal, "a").
		Compile()
	res, _, err := app.RepoStore.Search(ctx, nil, models.NoIdentity, query)
	require.Nil(t, err)
	require.Len(t, res, 0)

	query = search.NewRepoQueryBuilder().
		WhereOrg(search.Equal, "a").
		Compile()
	res, _, err = app.RepoStore.Search(ctx, nil, identityA.ID, query)
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoA.ID, res[0].ID)

	query = search.NewRepoQueryBuilder().
		WhereUser(search.NotEqual, "b").
		Compile()
	res, _, err = app.RepoStore.Search(ctx, nil, models.NoIdentity, query)
	require.Nil(t, err)
	require.Len(t, res, 3)
	sort.SliceStable(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	require.Equal(t, repoA.ID, res[0].ID)
	require.Equal(t, repoC.ID, res[1].ID)
	require.Equal(t, repoX.ID, res[2].ID)

	query = search.NewRepoQueryBuilder().
		Term("github-x").
		InName().
		Compile()
	res, _, err = app.RepoStore.Search(ctx, nil, models.NoIdentity, query)
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoX.ID, res[0].ID)

	query = search.NewRepoQueryBuilder().
		Term("github-x").
		InDescription().
		Compile()
	res, _, err = app.RepoStore.Search(ctx, nil, models.NoIdentity, query)
	require.Nil(t, err)
	require.Len(t, res, 0)

	query = search.NewRepoQueryBuilder().
		WhereEnabled(search.Equal, true).
		Compile()
	res, _, err = app.RepoStore.Search(ctx, nil, models.NoIdentity, query)
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, repoX.ID, res[0].ID)
}
