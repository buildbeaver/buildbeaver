package api_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/models/search"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestRepoSearch(t *testing.T) {
	ctx := context.Background()

	app, cleanup, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanup()
	app.CoreAPIServer.Start()
	defer app.CoreAPIServer.Stop(ctx)

	legalEntityA, identityA := server_test.CreatePersonLegalEntity(t, ctx, app, "test", "Jim Bob", "jim@bob.com")
	tokenA, _, err := app.CredentialService.CreateSharedSecretCredential(ctx, nil, identityA.ID, true)
	require.Nil(t, err)
	client, err := client.NewAPIClient(
		[]string{app.CoreAPIServer.GetServerURL()},
		client.NewSharedSecretAuthenticator(client.SharedSecretToken(tokenA.String()), app.LogFactory),
		app.LogFactory)
	require.Nil(t, err)

	legalEntityB, _ := server_test.CreatePersonLegalEntity(t, ctx, app, "b", "Stewie Griffin", "sg@fg.com")

	repoA := server_test.CreateNamedRepo(t, ctx, app, "a", legalEntityA.ID)
	repoB := server_test.CreateNamedRepo(t, ctx, app, "b", legalEntityA.ID)
	_ = server_test.CreateNamedRepo(t, ctx, app, "c", legalEntityB.ID)

	query := search.NewRepoQueryBuilder().
		WhereUser(search.Equal, legalEntityA.Name.String()).
		Compile()
	repos := readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 2)
	sort.SliceStable(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	require.Equal(t, repoA.ID, repos[0].ID)
	require.Equal(t, repoB.ID, repos[1].ID)

	query = search.NewRepoQueryBuilder().
		WhereUser(search.NotEqual, legalEntityA.Name.String()).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 0)

	query = search.NewRepoQueryBuilder().
		WhereEnabled(search.Equal, false).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 2)
	sort.SliceStable(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	require.Equal(t, repoA.ID, repos[0].ID)
	require.Equal(t, repoB.ID, repos[1].ID)

	repoA.Enabled = true
	err = app.RepoStore.Update(ctx, nil, repoA)
	require.Nil(t, err)

	query = search.NewRepoQueryBuilder().
		WhereEnabled(search.Equal, false).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 1)
	require.Equal(t, repoB.ID, repos[0].ID)

	query = search.NewRepoQueryBuilder().
		WhereEnabled(search.Equal, true).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 1)
	require.Equal(t, repoA.ID, repos[0].ID)

	query = search.NewRepoQueryBuilder().
		WhereEnabled(search.NotEqual, false).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 1)
	require.Equal(t, repoA.ID, repos[0].ID)

	query = search.NewRepoQueryBuilder().
		Term("idontexist").
		WhereEnabled(search.Equal, true).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 0)

	query = search.NewRepoQueryBuilder().
		Term("a").
		InName().
		WhereEnabled(search.Equal, true).
		Compile()
	repos = readAllRepoPages(t, ctx, client, legalEntityA.ID, query)
	require.Len(t, repos, 1)
	require.Equal(t, repoA.ID, repos[0].ID)
}

func readAllRepoPages(t *testing.T, ctx context.Context, client *client.APIClient, legalEntityID models.LegalEntityID, query search.Query) []*models.Repo {
	paginator, err := client.SearchRepos(ctx, legalEntityID, query)
	require.Nil(t, err)
	var repos []*models.Repo
	for paginator.HasNext() {
		page, err := paginator.Next(ctx)
		require.Nil(t, err)
		repos = append(repos, page...)
	}
	return repos
}
