package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/work_queue"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// NOTE: Managing deploy key requires the "Repository Administration" permission to be set to read/write
// on the GitHub app Permissions & events page.

const (
	GitHubSCMName                = models.SystemName("github")
	groupNamePrefixForGitHubTeam = "github-team-"
	maxCharsInCommitStatus       = 140
	commitStatusUpdateTimeout    = 30 * time.Second
	DefaultCommitStatusTargetURL = "https://app.changeme.com"
)

var gitHubStatusContextText = "BuildBeaver" // Context text to appear in status updates from us on GitHub

type AppConfig struct {
	AppID              int64
	PrivateKeyProvider PrivateKeyProvider
	DeployKeyName      string
	// CommitStatusTargetURL is a string that can be passed to GitHub as the 'target URL' when updating
	// the status of a commit.
	CommitStatusTargetURL string
}

type GitHubSCMAuthentication struct {
	*oauth2.Token ``
}

func (m *GitHubSCMAuthentication) Name() models.SystemName {
	return GitHubSCMName
}

// PrivateKeyProvider is a function that can provide a private key for use by a GitHub client.
type PrivateKeyProvider func() ([]byte, error)

func MakeInMemoryPrivateKeyProvider(key []byte) func() ([]byte, error) {
	return func() ([]byte, error) { return key, nil }
}

// MakeFilePathPrivateKeyProvider returns a PrivateKeyProvider function that will load the
// private key from the specified file path.
func MakeFilePathPrivateKeyProvider(filePath string) func() ([]byte, error) {
	return func() ([]byte, error) { return ioutil.ReadFile(string(filePath)) }
}

// NoPrivateKey is a PrivateKeyProvider function that always fails with an error saying
// there is no private key available.
func NoPrivateKey() ([]byte, error) {
	return nil, errors.New("no private key available")
}

// WebhookEvent contains all the information provided by a Webhook event notification sent from GitHub.
type WebhookEvent struct {
	EventType    string
	Signature256 string
	Payload      io.Reader
}

const repoInstallationUnset int64 = 0

type RepoMetadata struct {
	InstallationID int64  `json:"github_installation_id"`
	RepoName       string `json:"github_repo_name"`
	RepoOwner      string `json:"github_repo_owner"`
}

func NewRepoMetadata(installationID int64, repoName string, repoOwner string) *RepoMetadata {
	return &RepoMetadata{
		InstallationID: installationID,
		RepoName:       repoName,
		RepoOwner:      repoOwner,
	}
}

// GetRepoMetadata extracts the RepoMetadata object stored in the ExternalMetadata field of a Repo from GitHub.
func GetRepoMetadata(repo *models.Repo) (*RepoMetadata, error) {
	repoMetadata := &RepoMetadata{}
	err := json.Unmarshal([]byte(repo.ExternalMetadata), repoMetadata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling GitHub repo metadata JSON: %w", err)
	}
	return repoMetadata, nil
}

const legalEntityInstallationUnset int64 = -1

type LegalEntityMetadata struct {
	// Login is the GitHub Login ID for the person or org
	Login string `json:"github_login"`
	// InstallationID is the installation ID for the BuildBeaver app for this account if installed, otherwise legalEntityInstallationUnset
	InstallationID int64 `json:"github_installation_id"`
	// OrgDefaultRepoPermission is the GitHub setting for an organization that specifies what permissions a
	// 'member' user (who is not an admin in the org) should have by default for all repos in the org.
	// In the GitHub Web UI this is also known as 'Base permissions' and 'Organization member permissions'
	// (see the 'Member Privileges' tab on the Organization 'Settings' Web UI page)
	// Possible values for Organization/Company legal entities are: "read", "write", "admin", or "none".
	// For User legal entities this should be an empty string.
	OrgDefaultRepoPermission string `json:"github_org_default_repo_permission"`
}

// GetLegalEntityMetadata extracts the GitHub LegalEntityMetadata object stored in the ExternalMetadata field of a
// Legal Entity created from GitHub. Also checks that the given legal entity has a GitHub external ID, i.e. that
// it corresponds to a GitHub login account.
func GetLegalEntityMetadata(legalEntity *models.LegalEntity) (*LegalEntityMetadata, error) {
	if legalEntity.ExternalID == nil {
		return nil, fmt.Errorf("error Legal Entity ID %s name '%s' has no external ID; should be from GitHub", legalEntity.ID, legalEntity.Name)
	}
	if legalEntity.ExternalID.ExternalSystem != GitHubSCMName {
		return nil, fmt.Errorf("error unexpected system name: %v", legalEntity.ExternalID.ExternalSystem)
	}
	if legalEntity.ExternalMetadata == "" {
		return nil, fmt.Errorf("error Legal Entity ID %s name '%s' has no external Metadata", legalEntity.ID, legalEntity.Name)
	}
	metadata := &LegalEntityMetadata{}
	err := json.Unmarshal([]byte(legalEntity.ExternalMetadata), metadata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling GitHub legal entity metadata JSON: %w", err)
	}
	return metadata, nil
}

type GitHubService struct {
	db                 *store.DB
	repoStore          store.RepoStore
	commitStore        store.CommitStore
	buildStore         store.BuildStore
	pullRequestService services.PullRequestService
	legalEntityService services.LegalEntityService
	queueService       services.QueueService
	workQueueService   services.WorkQueueService
	groupService       services.GroupService
	syncService        services.SyncService
	config             AppConfig
	logger.Log
}

func NewGitHubService(
	db *store.DB,
	repoStore store.RepoStore,
	commitStore store.CommitStore,
	buildStore store.BuildStore,
	pullRequestService services.PullRequestService,
	legalEntityService services.LegalEntityService,
	queueService services.QueueService,
	workQueueService services.WorkQueueService,
	groupService services.GroupService,
	syncService services.SyncService,
	config AppConfig,
	logFactory logger.LogFactory,
) *GitHubService {
	s := &GitHubService{
		db:                 db,
		repoStore:          repoStore,
		commitStore:        commitStore,
		buildStore:         buildStore,
		pullRequestService: pullRequestService,
		legalEntityService: legalEntityService,
		queueService:       queueService,
		workQueueService:   workQueueService,
		groupService:       groupService,
		syncService:        syncService,
		config:             config,
		Log:                logFactory("GitHubService"),
	}

	// Register the code to process work items for sending Commit Status updates to GitHub
	err := s.workQueueService.RegisterHandler(
		CommitStatusWorkItem,
		s.ProcessCommitStatusWorkItem,
		commitStatusUpdateTimeout,
		work_queue.ExponentialBackoff(20, 5*time.Second, 1*time.Hour),
		true, // keep failed work items; could change to delete later
		true, // keep successful work items; could change to delete later
	)
	if err != nil {
		panic(fmt.Sprintf("error registering event handler: %s", err.Error()))
	}

	return s
}

// Name returns the unique name of the SCM.
func (s *GitHubService) Name() models.SystemName {
	return GitHubSCMName
}

// EnableRepo is called when a repo is enabled within BuildBeaver - this is the SCM's opportunity
// to do any setup required to close the loop and make this work. Public key identifies the key that
// BuildBeaver will use when cloning the repo.
func (s *GitHubService) EnableRepo(ctx context.Context, repo *models.Repo, publicKey []byte) error {
	client, err := s.makeGitHubAppInstallationClientForRepo(repo)
	if err != nil {
		return fmt.Errorf("error creating app installation client for repo: %w", err)
	}

	repoID, err := githubIDFromExternalID(repo.ExternalID.ResourceID)
	if err != nil {
		return fmt.Errorf("error parsing external id to repo id: %w", err)
	}

	// NOTE we use the undocumented endpoint to get a repo by id here.
	// 	It's not an official endpoint but it's probably never going away and it's much easier to use.
	// 	Ideally GitHub would allow us to get all resources by their immutable ID but ¯\_(ツ)_/¯.
	// 	If it ever does go away the alternative here is to get the legal entity that the repo belongs
	//	to - by definition this must be a GitHub user or org. We can then use legal_entity.external_metadata.Login
	// 	as the owner and call client.Repositories.Get(ctx, owner, repoMetadata.RepoName). Make sure to check
	// 	the returned repo id matches our recorded external id. Similar story with DisableRepo.
	ghRepo, _, err := client.Repositories.GetByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("error getting repo: %w", err)
	}

	key := &github.Key{
		Key:      github.String(string(publicKey)),
		Title:    github.String(s.config.DeployKeyName),
		ReadOnly: github.Bool(true),
	}

	_, _, err = client.Repositories.CreateKey(ctx, ghRepo.Owner.GetLogin(), ghRepo.GetName(), key)
	if err != nil {
		return fmt.Errorf("error creating deploy key: %w", err)
	}

	return nil
}

// DisableRepo is called when a repo is disabled in BuildBeaver - this is the SCM's opportunity to do any required
// teardown such as deleting webhooks or deployment keys etc.
func (s *GitHubService) DisableRepo(ctx context.Context, repo *models.Repo) error {
	client, err := s.makeGitHubAppInstallationClientForRepo(repo)
	if err != nil {
		return fmt.Errorf("error creating app installation client for repo: %w", err)
	}

	repoID, err := githubIDFromExternalID(repo.ExternalID.ResourceID)
	if err != nil {
		return fmt.Errorf("error parsing external id to repo id: %w", err)
	}

	ghRepo, _, err := client.Repositories.GetByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("error getting repo: %w", err)
	}

	keys, err := ListAllDeployKeysForRepo(ctx, client, ghRepo)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if key.GetTitle() == s.config.DeployKeyName {
			_, err = client.Repositories.DeleteKey(ctx, ghRepo.Owner.GetLogin(), ghRepo.GetName(), key.GetID())
			if err != nil {
				return fmt.Errorf("error deleting deploy key: %w", err)
			}
		}
	}
	return nil
}

func githubIDToExternalID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func githubIDFromExternalID(externalID string) (int64, error) {
	return strconv.ParseInt(externalID, 10, 64)
}

// GitHubIDToExternalResourceID converts an ID from the GitHub API to an ExternalResourceID.
func GitHubIDToExternalResourceID(gitHubID int64) models.ExternalResourceID {
	return models.NewExternalResourceID(GitHubSCMName, githubIDToExternalID(gitHubID))
}

// makeGitHubAppClient returns a GitHub client configured to authenticate as the app with the
// GitHub App ID specified when the service was created.
func (s *GitHubService) makeGitHubAppClient() (*github.Client, error) {
	privateKey, err := s.config.PrivateKeyProvider()
	if err != nil {
		return nil, fmt.Errorf("error obtaining GitHub private key: %w", err)
	}
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, s.config.AppID, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "error loading auth")
	}
	client := github.NewClient(&http.Client{Transport: transport})
	return client, nil
}

// makeGitHubAppInstallationClient returns a GitHub client configured to authenticate as an app installation.
// The InstallationID recorded in the repo's ExternalMetadata field is used to construct the client.
func (s *GitHubService) makeGitHubAppInstallationClientForRepo(repo *models.Repo) (*github.Client, error) {
	installationID, err := s.getRepoInstallationID(repo)
	if err != nil {
		return nil, err
	}

	client, err := s.makeGitHubAppInstallationClient(installationID)
	if err != nil {
		return nil, fmt.Errorf("error making github client: %w", err)
	}

	return client, nil
}

// getRepoInstallationID returns the GitHub app installation ID recorded in the repo's ExternalMetadata field,
// or an error if no installation ID can be found.
func (s *GitHubService) getRepoInstallationID(repo *models.Repo) (int64, error) {
	if repo.ExternalID == nil {
		return 0, fmt.Errorf("error external id must be set")
	}
	if repo.ExternalID.ExternalSystem != GitHubSCMName {
		return 0, fmt.Errorf("error unexpected system name: %v", repo.ExternalID.ExternalSystem)
	}

	repoMetadata, err := GetRepoMetadata(repo)
	if err != nil {
		return 0, err
	}
	if repoMetadata.InstallationID == repoInstallationUnset {
		return 0, errors.New("error repo installation id is not set")
	}

	return repoMetadata.InstallationID, nil
}

// makeGitHubAppInstallationClientForLegalEntity returns a GitHub client configured to authenticate as an app installation.
// The InstallationID recorded in the legal entity's ExternalMetadata field is used to construct the client.
func (s *GitHubService) makeGitHubAppInstallationClientForLegalEntity(legalEntity *models.LegalEntity) (*github.Client, error) {
	installationID, err := s.getLegalEntityInstallationID(legalEntity)
	if err != nil {
		return nil, err
	}

	client, err := s.makeGitHubAppInstallationClient(installationID)
	if err != nil {
		return nil, fmt.Errorf("error making github client: %w", err)
	}

	return client, nil
}

// getLegalEntityInstallationID returns the GitHub app installation ID recorded in the legal entity's ExternalMetadata
// field, or an error if no installation ID can be found.
func (s *GitHubService) getLegalEntityInstallationID(legalEntity *models.LegalEntity) (int64, error) {
	// Check the legal entity has an External ID and metadata
	metadata, err := GetLegalEntityMetadata(legalEntity)
	if err != nil {
		return 0, err
	}
	if metadata.InstallationID == legalEntityInstallationUnset || metadata.InstallationID == 0 {
		return 0, errors.New("error Legal Entity installation id is not set")
	}

	return metadata.InstallationID, nil
}

// makeGitHubAppInstallationClient returns a GitHub client configured to authenticate as an app installation.
func (s *GitHubService) makeGitHubAppInstallationClient(installationID int64) (*github.Client, error) {
	if installationID <= 0 {
		return nil, fmt.Errorf("error making GitHub App Installation Client: invalid installationID %d", installationID)
	}
	privateKey, err := s.config.PrivateKeyProvider()
	if err != nil {
		return nil, fmt.Errorf("error obtaining GitHub private key: %w", err)
	}
	transport, err := ghinstallation.New(http.DefaultTransport, s.config.AppID, installationID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("error loading GitHub app auth: %w", err)
	}
	client := github.NewClient(&http.Client{Transport: transport})
	return client, nil
}

// makeGitHubOAuthClient returns a GitHub client configured to authenticate as a user.
// NOTE: The BuildBeaver server should not normally impersonate the user except during the OAuth login process.
// Consider using makeGitHubAppInstallationClient() instead.
func (s *GitHubService) makeGitHubOAuthClient(ctx context.Context, auth models.SCMAuth) (*github.Client, error) {
	ghAuth, ok := auth.(*GitHubSCMAuthentication)
	if !ok {
		return nil, fmt.Errorf("unrecognized auth type: %T", auth)
	}
	tokenSrc := oauth2.StaticTokenSource(ghAuth.Token)
	oauthClient := oauth2.NewClient(ctx, tokenSrc)
	return github.NewClient(oauthClient), nil
}
