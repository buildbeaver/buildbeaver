package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// BuildRepoLatestCommit will kick off a new build for the latest commit for a ref, if required.
// The ref can be a branch or a tag. The supplied ref is read from GitHub to determine the latest commit.
// If no ref is supplied then the head of the main/master branch for the repo will be used.
// If there is no build underway or complete for the latest commit then a new build will be queued.
// If all completed builds for this commit failed then a new build will be queued.
// Older builds for previous commits for this ref may be cancelled or elided from the queue, since they
// are out of date.
func (s *GitHubService) BuildRepoLatestCommit(
	ctx context.Context,
	repo *models.Repo,
	ref string,
) error {
	if !repo.Enabled {
		s.Infof("Ignoring request to Build the latest commit for for repo that is not enabled")
		return nil
	}
	if ref == "" {
		ref = repo.DefaultBranch
	}
	// If not already there then prepend "refs/heads/" to branch name to form a ref
	ref = fixGithubBranchRef(ref)

	s.Infof("Queuing a build for enabled repo %s", repo.GetName())

	// Fetch GitHub-specific values from repo metadata
	repoMetadata, err := GetRepoMetadata(repo)
	if err != nil {
		return err
	}
	installationID := repoMetadata.InstallationID
	ghOwner := repoMetadata.RepoOwner
	ghRepoName := repoMetadata.RepoName

	ghClient, err := s.makeGitHubAppInstallationClient(installationID)
	if err != nil {
		return fmt.Errorf("error making github client: %w", err)
	}

	// Find the commit at the head of this ref, and build it if necessary
	err = s.buildLatestCommit(ctx, ghClient, repo, ghRepoName, ghOwner, ref)
	if err != nil {
		return err
	}

	return nil
}

// buildLatestCommit will kick off a new build for the latest commit for a ref, if required.
// The ref can be a branch or a tag. The supplied ref is read from GitHub to determine the latest commit.
// If there is no build underway or complete for the latest commit then a new build will be queued.
// If all completed builds for this commit failed then a new build will be queued.
// Older builds for previous commits for this ref may be cancelled or elided from the queue, since they
// are out of date.
// The caller should not already have a DB transaction open since this function makes calls to GitHub,
// and also uses transactions and row locking to ensure only one build will be queued for a commit.
func (s *GitHubService) buildLatestCommit(
	ctx context.Context,
	ghClient *github.Client,
	repo *models.Repo,
	ghRepoName string,
	ghOwner string,
	ref string,
) error {
	// Ask GitHub which commit is the head of the ref
	ghReference, _, err := ghClient.Git.GetRef(ctx, ghOwner, ghRepoName, ref)
	if err != nil {
		return fmt.Errorf("error reading ref '%s' (owner '%s', repo '%s') from GitHub: %w", ref, ghOwner, ghRepoName, err)
	}
	s.Tracef("Read reference for %q, got Ref %q, object type %q, sha %q, url %q",
		ref,
		ghReference.GetRef(),
		ghReference.GetObject().GetType(),
		ghReference.GetObject().GetSHA(),
		ghReference.GetObject().GetURL())
	if ghReference.GetRef() != ref {
		return fmt.Errorf("GitHub GetRef call returned the wrong reference: expected %q but got %q", ref, ghReference.GetRef())
	} else if ghReference.GetObject().GetType() != "commit" {
		return fmt.Errorf("GitHub GetRef call returned an Object of type %q rather than a commit", ghReference.GetObject().GetType())
	} else if ghReference.GetObject().GetSHA() == "" {
		return fmt.Errorf("GitHub GetRef call did not return a SHA for commit")
	}
	if ghReference.GetObject().GetURL() == "" {
		s.Warnf("GitHub GetRef call did not return a URL for commit")
	}
	headSHA := ghReference.GetObject().GetSHA()

	// Read the commit at the head of the ref from GitHub
	// TODO: Consider only reading the commit if we don't already have it in our database
	ghHeadCommit, _, err := ghClient.Repositories.GetCommit(ctx, ghOwner, ghRepoName, headSHA)
	if err != nil {
		return errors.Wrapf(err, "error getting commit from head of repo, SHA %q", headSHA)
	}
	s.Tracef("Read commit from GitHub, SHA %q", ghHeadCommit.GetSHA())

	// Record the commit in the database, if not already there. Do not read the config file yet in case we
	// don't want to build this commit.
	headCommit, err := s.upsertCommit(ctx, ghClient, ghHeadCommit, repo, ghRepoName, ghOwner, false)
	if err != nil {
		return err
	}

	// Work out whether to run a new build for head commit by searching for existing builds
	buildSearch := models.NewBuildSearchForCommit(
		headCommit.ID,
		ref,
		true, // exclude failed builds (but include successfully completed builds)
		[]models.WorkflowStatus{
			models.WorkflowStatusCanceled, // ignore cancelled builds
			models.WorkflowStatusUnknown,  // ignore builds in unknown status
		},
		1, // we only need to find one build matching the criteria
	)
	existingBuilds, _, err := s.buildStore.Search(context.Background(), nil, models.NoIdentity, buildSearch)
	if err != nil {
		return errors.Wrap(err, "error searching database for existing builds for a commit")
	}
	if len(existingBuilds) > 0 {
		s.Tracef("buildLatestCommit commit %q already has a build with status %q - will not queue another build",
			existingBuilds[0].Build.CommitID, existingBuilds[0].Build.Status)
		return nil
	}

	// Read the config file from GitHub if not already there, and record against the commit in the database.
	headCommit, err = s.upsertCommit(ctx, ghClient, ghHeadCommit, repo, ghRepoName, ghOwner, true)
	if err != nil {
		return err
	}

	// Start a new transaction and take out a row lock on the commit to create a critical section while we
	// enqueue a build. Do not contact GitHub inside this transaction.
	err = s.db.WithTx(ctx, nil, func(tx *store.Tx) error {
		s.Tracef("Locking row for update for commit %q", headCommit.ID)
		err = s.commitStore.LockRowForUpdate(ctx, tx, headCommit.ID)
		if err != nil {
			return fmt.Errorf("Error locking commit for update: %w", err)
		}
		s.Tracef("Got lock for row for update for commit %q", headCommit.ID)

		// Search again for existing builds after we have the lock on the commit, in case
		// a concurrent goroutine has just enqueued a build
		existingBuilds, _, err = s.buildStore.Search(context.Background(), tx, models.NoIdentity, buildSearch)
		if err != nil {
			return errors.Wrap(err, "error searching database for existing builds for a commit")
		}
		if len(existingBuilds) > 0 {
			s.Tracef("Found another build for commit %q before we could add it; not queuing a second build", headCommit.ID)
			return nil
		}

		// Queue the build inside the same transaction
		_, err := s.queueService.EnqueueBuildFromCommit(ctx, tx, headCommit, ref, nil)
		if err != nil {
			// Config is valid but some other error happened; return error so the caller can potentially retry
			return errors.Wrap(err, "error queueing build for PR commit")
		}

		s.Tracef("Enqueued build for commit %q", headCommit.SHA)

		// TODO: Queue a work item to search for queued/submitted/running builds for other (older) commits
		// TODO: for this ref, and cancel or elide those builds if configured to do so.
		// TODO: Be careful since we may not be the latest commit any more... (back to problem of how
		// TODO: we tell if we are the latest)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// upsertCommit ensures that a commit, as well as its author and committer, are present in the database.
// If shouldReadConfigFile is true then this function ensures that we have the config file for this
// Commit recorded in the database, reading it from GitHub only if needed.
func (s *GitHubService) upsertCommit(
	ctx context.Context,
	ghClient *github.Client,
	ghCommit *github.RepositoryCommit,
	repo *models.Repo,
	repoName string,
	repoOwner string,
	shouldReadConfigFile bool,
) (*models.Commit, error) {
	sha := ghCommit.GetSHA()
	if sha == "" {
		return nil, fmt.Errorf("no SHA provided in GitHub commit")
	}

	// Read any existing commit for this SHA from the database
	// Ignore the race condition since it's valid to upsert the config data twice, and we are only
	// ever adding data to the commit via an upsert.
	dbCommit, err := s.commitStore.ReadBySHA(ctx, nil, repo.ID, sha)
	if err != nil {
		if gerror.IsNotFound(err) {
			dbCommit = nil
		} else {
			return nil, fmt.Errorf("error reading commit from database: %w", err)
		}
	}
	hasConfig := (dbCommit != nil) && (dbCommit.Config != nil)
	hasAuthorLegalEntity := (dbCommit != nil) && (dbCommit.AuthorID.Valid())
	hasCommitterLegalEntity := (dbCommit != nil) && (dbCommit.CommitterID.Valid())

	// Record legal entity for commit author if we don't already have one
	var authorID models.LegalEntityID
	if !hasAuthorLegalEntity && ghCommit.GetAuthor() != nil {
		author, _, err := s.findOrCreateGithubUser(ctx, ghCommit.GetAuthor())
		if err != nil {
			return nil, err
		}
		authorID = author.ID
	}

	// Record legal entity for committer if we don't already have one
	var committerID models.LegalEntityID
	if !hasCommitterLegalEntity && ghCommit.GetCommitter() != nil {
		// If the committer has the same GitHub login as the author then use the author legal entity
		// Note that matching names or emails are not enough to ensure it's the same person/org
		if ghCommit.GetCommitter().GetLogin() == ghCommit.GetAuthor().GetLogin() {
			s.Tracef("Committer Login is same as Author login; using same legal entity: %q", ghCommit.GetAuthor().GetLogin())
			committerID = authorID
		} else {
			committer, _, err := s.findOrCreateGithubUser(ctx, ghCommit.GetCommitter())
			if err != nil {
				return nil, err
			}
			committerID = committer.ID
		}
	}

	// Read the config file from GitHub only if requested, and only if not already in the database
	var (
		config     []byte
		configType models.ConfigType
	)
	if shouldReadConfigFile && !hasConfig {
		// Read the config file for this SHA from GitHub
		s.Tracef("Attempting to read config file from Owner %q, repo %q, SHA %q", repoOwner, repoName, sha)
		config, configType, err = s.getConfigFileOrNil(ctx, ghClient, repoOwner, repoName, sha)
		if err != nil {
			return nil, errors.Wrap(err, "error getting config")
		}
		if config != nil {
			s.Tracef("Successfully read config of type %q", configType)
		}
	}

	// Create a new dbCommit to upsert if we didn't already have one
	upsertRequired := false
	if dbCommit == nil {
		dbCommit = models.NewCommit(
			models.NewTime(time.Now()),
			repo.ID,
			config,
			configType,
			sha,
			ghCommit.GetCommit().GetMessage(),
			authorID,
			ghCommit.GetCommit().GetAuthor().GetName(),  // use data from original Git Commit, not GitHub data
			ghCommit.GetCommit().GetAuthor().GetEmail(), // use data from original Git Commit, not GitHub data
			committerID,
			ghCommit.GetCommit().GetCommitter().GetName(),  // use data from original Git Commit, not GitHub data
			ghCommit.GetCommit().GetCommitter().GetEmail(), // use data from original Git Commit, not GitHub data
			ghCommit.GetHTMLURL())
		upsertRequired = true
	} else {
		// Set new fields in the existing commit ready to be updated
		if config != nil {
			dbCommit.Config = config
			dbCommit.ConfigType = configType
			upsertRequired = true
		}
		if authorID.Valid() {
			dbCommit.AuthorID = authorID
			upsertRequired = true
		}
		if committerID.Valid() {
			dbCommit.CommitterID = committerID
			upsertRequired = true
		}
	}

	// Upsert the commit, to insert the record or update with new data.
	if upsertRequired {
		// Upsert for a commit won't update immutable data, and won't clear or overwrite previously set mutable data.
		_, _, err = s.commitStore.Upsert(ctx, nil, dbCommit)
		if err != nil {
			return nil, errors.Wrap(err, "error upserting commit")
		}
		s.Tracef("Successfully upserted Commit in DB - CommitID %q, SHA %q, ConfigType=%q",
			dbCommit.ID, dbCommit.SHA, dbCommit.ConfigType)
	}

	return dbCommit, nil
}

// getConfigFileOrNil reads the config file from the commit with the specified SHA and returns its contents and type.
// Returns a nil byte array and empty string for the ConfigType if the commit does not contain a config file.
func (s *GitHubService) getConfigFileOrNil(
	ctx context.Context,
	client *github.Client,
	repoOwner string,
	repoName string,
	commitSHA string,
) ([]byte, models.ConfigType, error) {
	tree, _, err := client.Git.GetTree(ctx, repoOwner, repoName, commitSHA, false)
	if err != nil {
		return nil, "", errors.Wrap(err, "error getting repo tree")
	}

	var (
		configType  models.ConfigType
		configEntry *github.TreeEntry
	)

loop:
	for _, entry := range tree.Entries {

		path := entry.GetPath()

		for _, p := range parser.YAMLBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeYAML
				configEntry = &entry
				break loop
			}
		}

		for _, p := range parser.JSONBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeJSON
				configEntry = &entry
				break loop
			}
		}

		for _, p := range parser.JSONNETBuildConfigFileNames {
			if path == p {
				configType = models.ConfigTypeJSONNET
				configEntry = &entry
				break loop
			}
		}
	}

	// No-op if the commit does not contain a config file
	if configEntry == nil {
		return nil, models.ConfigTypeNoConfig, nil
	}

	config, _, err := client.Git.GetBlobRaw(ctx, repoOwner, repoName, configEntry.GetSHA())
	if err != nil {
		return nil, "", errors.Wrap(err, "error getting config file")
	}

	// If config is too long or is empty then return it as invalid config, with error message
	err = s.queueService.CheckBuildConfigLength(len(config))
	if err != nil {
		configType = models.ConfigTypeInvalid
		// Replace the too-long config with an error message
		config = []byte(err.Error())
	}

	return config, configType, nil
}

// findOrCreateGithubUser ensures that we have a Legal Entity in our database for the supplied GitHub user.
// If the user already exists, no action will be taken and no details will be updated.
// In particular any existing GitHub external metadata, including the installation ID, will not be overwritten.
func (s *GitHubService) findOrCreateGithubUser(
	ctx context.Context,
	ghUser *github.User,
) (legalEntity *models.LegalEntity, created bool, err error) {
	s.Tracef("findOrCreateGithubUser called with ghUser %v", ghUser)

	// Don't bother looking up any installation ID for this user; can be added later during sync or login
	legalEntityData, err := s.legalEntityDataFromGitHubUser(ghUser, legalEntityInstallationUnset)

	legalEntity, created, err = s.legalEntityService.FindOrCreate(ctx, nil, legalEntityData)
	if err != nil {
		return nil, false, err
	}
	if created {
		s.Tracef("findOrCreateGithubUser created new legal entity for user %s, ID %s", ghUser.GetLogin(), legalEntity.ExternalID)
	} else {
		s.Tracef("findOrCreateGithubUser found existing legal entity for user %s, ID %s", ghUser.GetLogin(), legalEntity.ExternalID)
	}
	return legalEntity, created, nil
}
