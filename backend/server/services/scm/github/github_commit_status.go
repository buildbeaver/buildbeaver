package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v28/github"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// NotifyBuildUpdated is called when the status of a build is updated.
// Allows the SCM to notify users or take other actions when a build has progressed or finished.
func (s *GitHubService) NotifyBuildUpdated(ctx context.Context, txOrNil *store.Tx, build *models.Build, repo *models.Repo) error {
	s.Tracef("Received notification that build %q has been updated for repo %q", build.Name, repo.Name)
	return s.setGitHubCommitStatusForBuild(ctx, txOrNil, build, repo)
}

// setGitHubCommitStatusForBuild queues a work item to update GitHub with a status for the commit for the
// specified build, to reflect the build's status (including any errors).
// It's OK to call this function inside a DB transaction since GitHub will not actually be contacted directly.
func (s *GitHubService) setGitHubCommitStatusForBuild(ctx context.Context, txOrNil *store.Tx, build *models.Build, repo *models.Repo) error {

	repoMetadata, err := GetRepoMetadata(repo)
	if err != nil {
		return err
	}
	installationID := repoMetadata.InstallationID
	ghOwner := repoMetadata.RepoOwner
	ghRepoName := repoMetadata.RepoName

	repoOwner, err := s.legalEntityService.Read(ctx, txOrNil, repo.LegalEntityID)
	if err != nil {
		return fmt.Errorf("error repo owner legal entity for build: %w", err)
	}
	commit, err := s.commitStore.Read(ctx, txOrNil, build.CommitID)
	if err != nil {
		return fmt.Errorf("error reading commit for build: %w", err)
	}

	// Create suitable data for GitHub
	gitHubState := build.Status.ToGitHubState()
	var description string
	if build.Status == models.WorkflowStatusFailed {
		description = fmt.Sprintf("Build failed: %s", build.Error.Error())
	} else {
		description = fmt.Sprintf("Build status: %s", build.Status)
	}
	targetURL, err := s.makeWebUIBuildURL(repoOwner, repo, build)
	if err != nil {
		return err
	}

	err = s.setGitHubCommitStatus(ctx, txOrNil, installationID, ghOwner, ghRepoName, commit.SHA, gitHubState, targetURL, description)
	if err != nil {
		return err
	}

	return nil
}

func (s *GitHubService) makeWebUIBuildURL(repoOwner *models.LegalEntity, repo *models.Repo, build *models.Build) (string, error) {
	var orgsOrUsers string
	switch repoOwner.Type {
	case models.LegalEntityTypeCompany:
		orgsOrUsers = "orgs"
	case models.LegalEntityTypePerson:
		orgsOrUsers = "users"
	default:
		return "", fmt.Errorf("error unknown type of Legal Entity '%s' for repo owner, name '%s", repoOwner.Type, repoOwner.Name)
	}

	baseURL := strings.TrimSuffix(s.config.CommitStatusTargetURL, "/")

	// Example URL: https://app.staging.changeme.com/orgs/buildbeaver/repos/playground/builds/612
	targetURL := fmt.Sprintf("%s/%s/%s/repos/%s/builds/%s",
		baseURL,
		orgsOrUsers,
		url.QueryEscape(repoOwner.GetName().String()),
		url.QueryEscape(repo.GetName().String()),
		url.QueryEscape(build.GetName().String()),
	)

	return targetURL, nil
}

// setGitHubCommitStatus queues a Work Item to update the GitHub Status for a commit.
// installationID is the GitHub installation ID for the BuildBeaver GitHub app.
// owner, repo and sha are the GitHub repo owner name, GitHub repo name and GitHub SHA for the build.
func (s *GitHubService) setGitHubCommitStatus(
	ctx context.Context,
	txOrNil *store.Tx,
	installationID int64,
	owner, repo, sha string,
	gitHubState string,
	targetURL string,
	statusDescription string,
) error {
	// Ensure description is short enough, or it will be rejected by GitHub
	shortDescription := util.TruncateStringToMaxLength(statusDescription, maxCharsInCommitStatus)
	s.Tracef("Queuing work item to set GitHub Status for repo %s, commit %s to state %q, description %q",
		repo, sha, gitHubState, shortDescription)

	// Add a work item to the queue to send the status to GitHub
	workItem := NewCommitStatusWorkItem(
		installationID,
		owner, repo, sha,
		gitHubState, targetURL, shortDescription, gitHubStatusContextText,
	)
	err := s.workQueueService.AddWorkItem(ctx, txOrNil, workItem)
	if err != nil {
		return fmt.Errorf("error queueing work item to set Commit Status on GitHub: %w", err)
	}

	return nil
}

// ProcessCommitStatusWorkItem is a work item handler that contacts GitHub and sends a new commit status.
func (s *GitHubService) ProcessCommitStatusWorkItem(ctx context.Context, workItem *models.WorkItem) (canRetry bool, err error) {
	// Unmarshal CommitStatusWorkItemData
	workItemData := &CommitStatusWorkItemData{}
	err = json.Unmarshal([]byte(workItem.Data), workItemData)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling Commit Status work item data: %w", err)
	}

	// Make a GitHub client for the app installation specified in the work item
	ghClient, err := s.makeGitHubAppInstallationClient(workItemData.InstallationID)
	if err != nil {
		return false, fmt.Errorf("error making github client: %w", err)
	}

	status := &github.RepoStatus{
		State:       &workItemData.GitHubState,
		TargetURL:   &workItemData.TargetURL,
		Description: &workItemData.Description,
		Context:     &workItemData.ContextText,
	}

	_, response, err := ghClient.Repositories.CreateStatus(
		ctx, // allow the request to be cancelled by WorkQueueService
		workItemData.Owner,
		workItemData.Repo,
		workItemData.SHA,
		status,
	)
	if err != nil {
		canRetry := true
		if response.StatusCode == 404 {
			canRetry = false // no point trying again if the commit isn't there
		}
		if response.StatusCode == 403 {
			if response.Rate.Remaining == 0 {
				// we hit the rate limit, which returns a 403. Retry again later. See:
				// https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting
				s.Infof("GitHub API rate limit hit for app installation (ID %d) when setting a commit status", workItemData.InstallationID)
				canRetry = true
			} else {
				canRetry = false // for a more general access denied error there's no point retrying
			}
		}
		return canRetry, fmt.Errorf("error setting GitHub Commit Status: %w", err)
	}

	s.Tracef("GitHub Status set successfully by CommitStatusWorkItem")
	return false, nil
}
