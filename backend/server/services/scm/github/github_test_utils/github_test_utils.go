package github_test_utils

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/buildbeaver/buildbeaver/common/models"
	github_service "github.com/buildbeaver/buildbeaver/server/services/scm/github"
)

// GitHubDefaultBranchName is the branch name set by default for new repos on the test GitHub account.
// This must match the setting on the test GitHub account and organizations.
const GitHubDefaultBranchName = "main"
const GitHubDefaultBranchRef = "refs/heads/" + GitHubDefaultBranchName

// GitHubTestAccountUserName is the GitHub name of the user for the integration test GitHub account.
// TODO: Add GitHub test user name below
const GitHubTestAccountUserName = "insert-github-test-user-here"

// GitHubTestAccountGitHubID is the GitHub API ID for the integration test GitHub account.
// TODO: Add GitHub API account ID below
const GitHubTestAccountGitHubID = 11111111

// GitHubTestAccountExternalID is the legal entity external ID (based on the GitHub API ID) for the
// integration test GitHub account.
var GitHubTestAccountExternalID = models.NewExternalResourceID(github_service.GitHubSCMName, strconv.FormatInt(GitHubTestAccountGitHubID, 10))

// GitHubTestAccountLegalEntityName is the name of user legal entity for the integration test GitHub account.
const GitHubTestAccountLegalEntityName = models.ResourceName("integration-test")

// GitHubTestAccountLegalName is the Legal Entity name for the integration test GitHub account.
const GitHubTestAccountLegalName = "Integration Test Account"

// GitHubTestAccountOrg1Name is the GitHub name of the first test GitHub organization created
// under the integration test GitHub account.
// TODO: Add GitHub test org name below (create org 1 manually on GitHub)
const GitHubTestAccountOrg1Name = "insert-github-test-org-1-name-here"

// GitHubTestAccountOrg1GitHubID is the GitHub API ID for the first test GitHub organization created
// under the integration test GitHub account.
// TODO: Add GitHub test org ID below (create org 1 manually on GitHub)
const GitHubTestAccountOrg1GitHubID = 11111111

// GitHubTestAccountOrg1ExternalID is the legal entity external ID (based on the GitHub API ID) for the first test
// GitHub organization created under the integration test GitHub account.
var GitHubTestAccountOrg1ExternalID = models.NewExternalResourceID(github_service.GitHubSCMName, strconv.FormatInt(GitHubTestAccountOrg1GitHubID, 10))

// GitHubTestAccountOrg1LegalEntityName is the name of the Legal Entity for the first test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg1LegalEntityName = models.ResourceName("test-org-1")

// GitHubTestAccountOrg1LegalName is the Legal Name (i.e. company name) for the first test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg1LegalName = "Test Organization 1"

// GitHubTestAccountOrg1EMail is the email address for the first test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg1EMail = "github-test-org-1@nodomain.com"

// GitHubTestAccountOrg2Name is the GitHub name of the second test GitHub organization created
// under the integration test GitHub account.
// TODO: Add GitHub test org name below (create org 2 manually on GitHub)
const GitHubTestAccountOrg2Name = "insert-github-test-org-2-name-here"

// GitHubTestAccountOrg2GitHubID is the GitHub API ID for the second test GitHub organization created
// under the integration test GitHub account.
// TODO: Add GitHub test org ID below (create org 2 manually on GitHub)
const GitHubTestAccountOrg2GitHubID = 11111111

// GitHubTestAccountOrg2ExternalID is the legal entity external ID (based on the GitHub API ID) for the second test
// GitHub organization created under the integration test GitHub account.
var GitHubTestAccountOrg2ExternalID = models.NewExternalResourceID(github_service.GitHubSCMName, strconv.FormatInt(GitHubTestAccountOrg2GitHubID, 10))

// GitHubTestAccountOrg2LegalEntityName is the name of the Legal Entity for the second test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg2LegalEntityName = models.ResourceName("test-org-2")

// GitHubTestAccountOrg2LegalName is the Legal Name (i.e. company name) for the second test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg2LegalName = "Test Organization 2"

// GitHubTestAccountOrg2EMail is the email address for the second test GitHub organization
// created under the integration test GitHub account.
const GitHubTestAccountOrg2EMail = "github-test-org-2@nodomain.com"

// SmeeTestAccountEndpoint is a smee.io endpoint for receiving notifications via Webhook from the test GitHub
// account during GitHub integration testing.
// TODO: Add smee channel URL for use in tests (set one up at https://smee.io )
const SmeeTestAccountEndpoint = "https://smee.io/INSERT-SMEE-CHANNEL-HERE"

// githubTestAccountAuthToken is a Personal Access Token for the test GitHub account.
// TODO: Add test account Auth token below (can use a manually created developer key from GitHub)
const githubTestAccountAuthToken = "ghp_INSERT-TEST-ACCOUNT-AUTH-TOKEN-HERE"

// GithubTestAppID is the ID of the GitHub Test app, on the GitHub test account.
// TODO: Add GitHub App ID for test app here (after manually setting up test app)
const GithubTestAppID = 111111

// githubTestAppPrivateKey is the private key for the GitHub Test app, private to the GitHub test account,
// to be used for integration testing.
// TODO: Add GitHub Test App private key here (after manually setting up GitHub test app)
const githubTestAppPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
*** INSERT GITHUB TEST APP PRIVATE KEY HERE ***
-----END RSA PRIVATE KEY-----
`

// TestAccountAppPrivateKey is a PrivateKeyProvider function that returns a hard-coded private
// key that can be used with the GitHub Test app, for integration testing.
func TestAccountAppPrivateKey() ([]byte, error) {
	return []byte(githubTestAppPrivateKey), nil
}

// GitHubTestAccount2UserName is the GitHub name of the user for the 2nd integration test GitHub account.
// TODO: Add GitHub second test user name below
const GitHubTestAccount2UserName = "insert-github-second-test-user-here"

// GitHubTestAccount2GitHubID is the GitHub API ID for the 2nd integration test GitHub account.
// TODO: Add GitHub API second account ID below
const GitHubTestAccount2GitHubID = 11111111

// GitHubTestAccount2ExternalID is the legal entity external ID (based on the GitHub API ID) for the 2nd
// integration test GitHub account.
var GitHubTestAccount2ExternalID = models.NewExternalResourceID(github_service.GitHubSCMName, strconv.FormatInt(GitHubTestAccount2GitHubID, 10))

// GitHubTestAccount2LegalEntityName is the name of user legal entity for the 2nd integration test GitHub account.
const GitHubTestAccount2LegalEntityName = models.ResourceName("integration-test-2")

// GitHubTestAccount2LegalName is the Legal Entity name for the 2nd integration test GitHub account.
const GitHubTestAccount2LegalName = "Integration Test Account 2"

// githubTestAccount2AuthToken is a Personal Access Token for the 2nd test GitHub account.
// TODO: Add second test account Auth token below (can use a manually created developer key from GitHub)
const githubTestAccount2AuthToken = "ghp_INSERT-SECOND-TEST-ACCOUNT-AUTH-TOKEN-HERE"

// MakeGitHubAuth creates an SCMAuth object for use when authenticating to GitHub as a user.
// The hard-coded githubTestAccountAuthToken personal access token will be used for authentication.
func MakeGitHubAuth(t *testing.T) models.SCMAuth {
	accessTokenStr := githubTestAccountAuthToken
	require.NotEmpty(t, accessTokenStr, "OAuth token for test account required")

	// Create a static token for use by the tests
	oAuthToken := &oauth2.Token{AccessToken: accessTokenStr}
	scmAuth := &github_service.GitHubSCMAuthentication{
		Token: oAuthToken,
	}

	return scmAuth
}

// MakeGitHubAuth2 creates an SCMAuth object for use when authenticating to GitHub as the 2nd integration test user.
// The hard-coded githubTestAccount2AuthToken personal access token will be used for authentication.
func MakeGitHubAuth2(t *testing.T) models.SCMAuth {
	accessTokenStr := githubTestAccount2AuthToken
	require.NotEmpty(t, accessTokenStr, "OAuth token for test account 2 required")

	// Create a static token for use by the tests
	oAuthToken := &oauth2.Token{AccessToken: accessTokenStr}
	scmAuth := &github_service.GitHubSCMAuthentication{
		Token: oAuthToken,
	}

	return scmAuth
}

// MakeGitHubTestClient returns a GitHub client configured to authenticate as a user, for use in testing.
func MakeGitHubTestClient(auth models.SCMAuth) (*github.Client, error) {
	ghAuth, ok := auth.(*github_service.GitHubSCMAuthentication)
	if !ok {
		return nil, fmt.Errorf("unrecognized auth type: %T", auth)
	}
	tokenSrc := oauth2.StaticTokenSource(ghAuth.Token)
	oauthClient := oauth2.NewClient(context.Background(), tokenSrc)
	return github.NewClient(oauthClient), nil
}

// GitHubIDToExternalResourceID converts an ID from the GitHub API to an ExternalResourceID.
func GitHubIDToExternalResourceID(gitHubID int64) models.ExternalResourceID {
	return github_service.GitHubIDToExternalResourceID(gitHubID)
}

// SetupTestRepo sets up a new private Repo in GitHub with a random name.
// If orgName is not empty then the repo will be created under that organization, which should
// be owned by the test GitHub account. Otherwise, the repo is created under the authenticated user.
// Returns the new Repo details, the ExternalResourceID of the repo as would be used in the Repo store,
// and a teardown function that will remove the repo again.
func SetupTestRepo(t *testing.T, ghClient *github.Client, orgName string) (*github.Repository, models.ExternalResourceID, func(), error) {
	if orgName != "" {
		t.Logf("Setup creating random test Repo owned by organization %s", orgName)
	} else {
		t.Logf("Setup creating random test Repo owned by test user")
	}

	repo, err := createRandomTestRepository(t, ghClient, orgName, true)
	require.NoError(t, err, "createRandomTestRepository returned error")
	repoExternalID := GitHubIDToExternalResourceID(repo.GetID())
	t.Logf("Setup successfully created repo %q with External Resource ID %q", *repo.Name, repoExternalID)

	teardown := func() {
		// delete the repository
		t.Logf("Teardown deleting repo %s", *repo.Name)
		_, err = ghClient.Repositories.Delete(context.Background(), *repo.Owner.Login, *repo.Name)
		require.NoError(t, err, "Repositories.Delete() returned error")
	}

	return repo, repoExternalID, teardown, nil
}

func createRandomTestRepository(t *testing.T, ghClient *github.Client, orgName string, autoinit bool) (*github.Repository, error) {
	// Determine owner (user name or organization name) of the new repo
	owner := orgName
	if orgName == "" {
		// get authenticated user
		me, _, err := ghClient.Users.Get(context.Background(), "")
		if err != nil {
			return nil, err
		}
		owner = *me.Login
	}

	// Create a random repo name that does not currently exist
	var repoName string
	for {
		repoName = fmt.Sprintf("test-%d", rand.Int63n(1000000000)) // use a 9-digit number
		_, resp, err := ghClient.Repositories.Get(context.Background(), owner, repoName)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				// no repo exists with this repoName
				break
			}
			return nil, err
		}
	}

	description := "Automatically-generated test repo, generated by integration tests"

	// Create the repository
	repo, _, err := ghClient.Repositories.Create(context.Background(), orgName, &github.Repository{
		Name:        github.String(repoName),
		Description: &description,
		AutoInit:    github.Bool(autoinit),
		Private:     github.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Check the repository is there by reading it back; we may need to try this several times
	err = waitForRepoCreate(t, ghClient, repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// waitForRepoCreate will attempt to read the specified repo back from GitHub, and retry until
// either the repo is present or the maximum number of retries have occurred.
func waitForRepoCreate(t *testing.T, ghClient *github.Client, repo *github.Repository) error {
	const (
		maxAttempts          = 5
		delayBetweenAttempts = 5 * time.Second
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, _, err := ghClient.Repositories.Get(context.Background(), repo.GetOwner().GetLogin(), repo.GetName())
		if err != nil {
			t.Logf("error reading back newly created repo; will retry up to %d times: %s", maxAttempts, err)
			time.Sleep(delayBetweenAttempts)
			continue
		}
		return nil // No error, so we're done
	}
	// We got through all attempts to read the repo without succeeding
	return fmt.Errorf("error: timed out waiting for new GitHub repo to be available")
}

// SetupRepoFork creates a fork of an existing Repo in GitHub.
// If orgName is not empty then the new forked repo will be created under that organization, which should
// be owned by the test GitHub account. Otherwise, the forked repo is created under the authenticated user.
// Returns the new fork repo details, the ExternalResourceID of the forked repo as would be used in the Repo store,
// and a teardown function that will remove the forked repo again.
func SetupRepoFork(t *testing.T, ghClient *github.Client, orgName string, repoToFork *github.Repository) (*github.Repository, models.ExternalResourceID, func(), error) {
	if orgName != "" {
		t.Logf("Setup forking Repo %q, fork will be owned by organization %s", repoToFork.GetName(), orgName)
	} else {
		t.Logf("Setup forking Repo %q, fork will be owned by test user", repoToFork.GetName())
	}

	repo, err := createRepoFork(ghClient, orgName, repoToFork)
	require.NoError(t, err, "createRepoFork returned error")
	repoExternalID := GitHubIDToExternalResourceID(repo.GetID())
	t.Logf("Setup successfully fork of repo %q with External Resource ID %q", *repo.Name, repoExternalID)

	teardown := func() {
		// delete the repository
		t.Logf("Teardown deleting forked repo %s", *repo.Name)
		_, err = ghClient.Repositories.Delete(context.Background(), *repo.Owner.Login, *repo.Name)
		require.NoError(t, err, "Repositories.Delete() returned error")
	}

	return repo, repoExternalID, teardown, nil
}

// createRepoFork creates a fork of an existing Repo in GitHub.
// This function will wait for the repo to be created, or return a TimeoutError if it takes too long.
// Note that GitHub will not fork an empty repo.
func createRepoFork(ghClient *github.Client, orgName string, repoToFork *github.Repository) (*github.Repository, error) {
	const (
		maxAttempts          = 5
		delayBetweenAttempts = 1 * time.Second
	)
	var (
		newRepo *github.Repository
		err     error
	)
	for attempt := 1; attempt <= maxAttempts && newRepo == nil; attempt++ {
		// We can repeatedly call this API function until it succeeds
		newRepo, _, err = ghClient.Repositories.CreateFork(
			context.Background(),
			repoToFork.GetOwner().GetLogin(),
			repoToFork.GetName(),
			&github.RepositoryCreateForkOptions{
				Organization: orgName,
			})
		if err != nil {
			_, ok := err.(*github.AcceptedError)
			if ok {
				// AcceptedError means GitHub is still processing the request, so wait and retry
				time.Sleep(delayBetweenAttempts)
				continue
			} else {
				// Not an 'accepted' error so return the real error
				return nil, err
			}
		}
	}
	if newRepo == nil {
		return nil, fmt.Errorf("error: timed out waiting for GitHub to fork repo")
	}

	return newRepo, nil
}

const sampleConfigFile = `# Example config file committed by Integration Tests
version: "0.2"
jobs:
  - name: test
    description: Run all tests
    docker:
      image: golang:1.14.7
    steps:
    - name: go
      commands:
        - echo "Hello world!!"
# Some salt to make this file unique: `

const badConfigFile = `# Example INVALID config file committed by Integration Tests
version: "0.2"
jobs:
  - name: test
    description: Run all tests

    docker:
      image: golang:1.14.7
This is not {,,, a valid config file
{{{{
    - name: go
      commands:
        - echo "Hello world!!"
# Some salt to make this file unique: `

// CommitTestConfigFile commits a new YAML config file to the master branch of the specified repository.
// Unique data is added in a comment to ensure the commit has a unique SHA.
// If badConfig is true then an invalid config file is checked in, otherwise a valid config file.
// Any existing config file will be replaced.
func CommitTestConfigFile(ghClient *github.Client, repo *github.Repository, badConfig bool) (commitSHA string, err error) {
	path := "buildbeaver.yml"
	commitMessage := "This is a test commit of a config file from BuildBeaver integration tests"
	branch := GitHubDefaultBranchName
	authorName := "BuildBeaver Integration Test"
	authorEmail := "info@nodomain.com"
	authorDate := time.Now()
	configFileText := sampleConfigFile
	if badConfig {
		configFileText = badConfigFile
	}
	fileContent := []byte(configFileText + fmt.Sprintf("test-%d", rand.Int63n(1000000000000)))

	// Find the SHA of any existing config file - required to update the file
	var existingFileSHA string
	existingFileInfo, _, _, err := ghClient.Repositories.GetContents(
		context.Background(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		path,
		&github.RepositoryContentGetOptions{Ref: branch},
	)
	if err == nil && existingFileInfo != nil && existingFileInfo.GetSHA() != "" {
		existingFileSHA = existingFileInfo.GetSHA()
	}

	var contentResp *github.RepositoryContentResponse
	if existingFileSHA != "" {
		// Update existing file
		contentResp, _, err = ghClient.Repositories.UpdateFile(
			context.Background(),
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			path,
			&github.RepositoryContentFileOptions{
				Message: &commitMessage,
				Content: fileContent,
				SHA:     &existingFileSHA,
				Branch:  &branch,
				Author: &github.CommitAuthor{
					Date:  &authorDate,
					Name:  &authorName,
					Email: &authorEmail,
					Login: nil,
				},
			},
		)
		if err != nil {
			return "", fmt.Errorf("error commiting update to config file to GitHub: %w", err)
		}
	} else {
		// No existing file so create a new file
		contentResp, _, err = ghClient.Repositories.CreateFile(
			context.Background(),
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			path,
			&github.RepositoryContentFileOptions{
				Message: &commitMessage,
				Content: fileContent,
				Branch:  &branch,
				Author: &github.CommitAuthor{
					Date:  &authorDate,
					Name:  &authorName,
					Email: &authorEmail,
					Login: nil,
				},
			},
		)
		if err != nil {
			return "", fmt.Errorf("error commiting new config file to GitHub: %w", err)
		}
	}

	return contentResp.Commit.GetSHA(), nil
}

const sampleRandomFile = `# Example random file committed by Integration Tests
# Some salt to make this file unique: `

// CommitRandomFile commits a new random file to the specified branch of the specified repository.
// Unique data is added in a comment to ensure the commit has a unique SHA.
func CommitRandomFile(ghClient *github.Client, repo *github.Repository, branch string) (commitSHA string, err error) {
	path := fmt.Sprintf("test-file-%d.txt", rand.Int63n(1000000000))
	commitMessage := "This is a test commit of a random file from integration tests"
	fileContent := []byte(sampleRandomFile + fmt.Sprintf("test-%d", rand.Int63n(1000000000000)))
	authorName := "Integration Test"
	authorEmail := "info@nodomain.com"
	authorDate := time.Now()

	contentResp, _, err := ghClient.Repositories.CreateFile(
		context.Background(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		path,
		&github.RepositoryContentFileOptions{
			Message: &commitMessage,
			Content: fileContent,
			Branch:  &branch,
			Author: &github.CommitAuthor{
				Date:  &authorDate,
				Name:  &authorName,
				Email: &authorEmail,
				Login: nil,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("error commiting random new file to GitHub: %w", err)
	}

	return contentResp.Commit.GetSHA(), nil
}

// CreateRandomBranch creates a new randomly named branch off the head of the default in the specified repo.
// Unique data is added in a comment to ensure the branch has a unique name.
// The branch name is returned (without 'refs/heads/' on the front).
func CreateRandomBranch(t *testing.T, ghClient *github.Client, repo *github.Repository) (newBranchName string, err error) {
	// Call GitHub to find the SHA of the commit at the head of the default branch
	headOfDefaultBranch, err := getRefWithRetry(t, ghClient, repo, GitHubDefaultBranchRef)
	if err != nil {
		return "", err
	}

	sha := headOfDefaultBranch.GetObject().GetSHA()
	if sha == "" {
		return "", fmt.Errorf("no SHA returned from head of test repo")
	}

	branchName := fmt.Sprintf("test-branch-%d", rand.Int63n(1000000000))
	branchRef := "refs/heads/" + branchName

	// Create the new branch by creating its ref
	newRef := &github.Reference{
		Ref: &branchRef,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}
	err = createRefWithRetry(t, ghClient, repo, newRef)
	if err != nil {
		return "", err
	}

	return branchName, nil
}

// getRefWithRetry calls the GitHub client's GetRef function to fetch a single Reference object for a given Git ref.
// If GitHub returns certain errors then the request is retried a number of times before an error is returned.
// Note that when creating a fork of a repo, sometimes a 409 error can be temporarily returned from GetRef
// with the message "Git Repository is empty." until the forked repo is finished being copied.
func getRefWithRetry(t *testing.T, ghClient *github.Client, repo *github.Repository, ref string) (*github.Reference, error) {
	const (
		maxAttempts          = 5
		delayBetweenAttempts = 5 * time.Second
	)
	ctx := context.Background()
	var lastError error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		headOfBranch, response, err := ghClient.Git.GetRef(ctx, repo.GetOwner().GetLogin(), repo.GetName(), ref)
		if err != nil {
			if response.StatusCode == 409 || response.StatusCode == 422 || (response.StatusCode >= 500 && response.StatusCode <= 599) {
				lastError = err
				t.Logf("error reading ref from test repo; will retry up to %d times: %s", maxAttempts, err)
				time.Sleep(delayBetweenAttempts)
				continue
			} else {
				return nil, fmt.Errorf("error reading ref from test repo: %w", err)
			}
		}
		return headOfBranch, nil // No error, so we're done
	}
	// We got through all attempts to read the repo without succeeding
	return nil, fmt.Errorf("error too many errors reading ref from test repo; last error: %w", lastError)
}

// createRefWithRetry calls the GitHub client's CreateRef function to fetch a single Reference object for a given Git ref.
// If GitHub returns certain errors then the request is retried a number of times before an error is returned.
// Note that when creating a fork of a repo, sometimes a 422 error can be temporarily returned from CreateRef
// with the message "Reference update failed" until the forked repo is finished being copied.
func createRefWithRetry(t *testing.T, ghClient *github.Client, repo *github.Repository, newRef *github.Reference) error {
	const (
		maxAttempts          = 5
		delayBetweenAttempts = 5 * time.Second
	)
	ctx := context.Background()
	var lastError error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, response, err := ghClient.Git.CreateRef(ctx, repo.GetOwner().GetLogin(), repo.GetName(), newRef)
		if err != nil {
			if response.StatusCode == 409 || response.StatusCode == 422 || (response.StatusCode >= 500 && response.StatusCode <= 599) {
				lastError = err
				t.Logf("error creating ref in test repo; will retry up to %d times: %s", maxAttempts, err)
				time.Sleep(delayBetweenAttempts)
				continue
			} else {
				return fmt.Errorf("error creating ref in test repo: %w", err)
			}
		}
		return nil // No error, so we're done
	}
	// We got through all attempts to read the repo without succeeding
	return fmt.Errorf("error too many errors creating ref in test repo; last error: %w", lastError)
}

// CreatePullRequest creates a new Pull Request on GitHub for the specified branch to be merged back
// into the default branch. The branch name should not be prefixed with 'refs/heads'.
// This function assumes both branches are in the same repo.
// Returns the GitHub ID for the Pull Request.
func CreatePullRequest(ghClient *github.Client, repo *github.Repository, branchName string) (pullRequestID int64, err error) {
	return doCreatePullRequest(ghClient, repo, GitHubDefaultBranchRef, branchName, "")
}

// CreateCrossRepoPullRequest creates a new cross-repo Pull Request on GitHub. The specified 'head' branch
// in a forked repo owned by the specified user will be merged back into the default branch of the specified
// base repo. The branch name should not be prefixed with 'refs/heads'.
// Returns the GitHub ID for the Pull Request.
func CreateCrossRepoPullRequest(
	ghClient *github.Client,
	baseRepo *github.Repository,
	headBranchName string,
	headRepoUserName string,
) (pullRequestID int64, err error) {
	return doCreatePullRequest(ghClient, baseRepo, GitHubDefaultBranchRef, headBranchName, headRepoUserName)
}

// doCreatePullRequest creates a new Pull Request on GitHub for the specified head branch to be merged back
// into the specified base branch. The branch names should NOT be prefixed with 'refs/heads'.
// For regular pull requests within a single repo, crossRepoHeadUserName should be left blank.
// For cross-repo pull requests the user or org name which owns the head branch in the forked repo should
// be specified in crossRepoHeadUserName - for details see the GitHub docs here:
// https://docs.github.com/en/rest/reference/pulls#create-a-pull-request
// Returns the GitHub ID for the Pull Request.
func doCreatePullRequest(
	ghClient *github.Client,
	repo *github.Repository,
	baseBranchName string,
	headBranchName string,
	crossRepoHeadUserName string,
) (pullRequestID int64, err error) {
	var (
		title               = "Test Pull Request created by Integration Tests"
		body                = "This Test Pull Request is generated automatically by a Integration Test."
		draft               = false
		maintainerCanModify = false
	)
	// Create the 'head' string for the API, specifying branch and optional user for cross-repo PRs
	headStr := headBranchName
	if crossRepoHeadUserName != "" {
		headStr = fmt.Sprintf("%s:%s", crossRepoHeadUserName, headBranchName)
	}
	ghPullRequest, _, err := ghClient.PullRequests.Create(
		context.Background(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		&github.NewPullRequest{
			Title:               &title,
			Head:                &headStr,
			Base:                &baseBranchName,
			Body:                &body,
			Issue:               nil,
			MaintainerCanModify: &maintainerCanModify,
			Draft:               &draft,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("error creating pull request: %w", err)
	}
	if ghPullRequest.GetID() == 0 {
		return 0, fmt.Errorf("error creating pull request: ID returned as zero")
	}

	return ghPullRequest.GetID(), nil
}
