package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
)

// WebhookHandler returns the http handler func that should be invoked when
// the SCM service receives a webhook, or an error if the service does not
// support webhooks.
func (s *GitHubService) WebhookHandler() (http.HandlerFunc, error) {
	return func(w http.ResponseWriter, r *http.Request) {
		eventType := r.Header.Get("X-GitHub-Event")
		if eventType == "" {
			s.Error("No event type header present")
			w.WriteHeader(400)
			return
		}

		// Require a signature; verification will happen inside handleWebhookEvent()
		signature256 := r.Header.Get("X-Hub-Signature-256")
		if eventType == "" {
			s.Error("No SHA-256 signature header present")
			w.WriteHeader(400)
			return
		}

		event := &WebhookEvent{
			EventType:    eventType,
			Signature256: signature256,
			Payload:      r.Body,
		}
		err := s.HandleWebhookEvent(r.Context(), event)
		if err != nil {
			s.Errorf("Error processing %s event: %s", eventType, err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)

	}, nil
}

// HandleWebhookEvent process an incoming GitHub Webhook event.
// eventType is the GitHub event name, from the 'X-GitHub-Event' header
// hubSignature256 is the SHA-256 signature for the event, from the 'X-Hub-Signature-256' header
// payload is a reader for the payload data of the event, which is the body of the HTTP request
func (s *GitHubService) HandleWebhookEvent(ctx context.Context, event *WebhookEvent) error {
	// TODO validate the signature on the event by using the configured webhook secret
	s.Warnf("Received GitHub Webhook. WARNING: SIGNATURE WAS NOT VERIFIED: %s", event.EventType)

	// Read the event payload
	payload, err := ioutil.ReadAll(event.Payload)
	if err != nil {
		return errors.Wrap(err, "error reading webhook payload")
	}

	switch event.EventType {
	case "push":
		err = s.handlePushEvent(ctx, payload)
	case "pull_request":
		err = s.handlePullRequestEvent(ctx, payload)
	case "installation":
		err = s.handleInstallationEvent(ctx, payload)
	case "installation_target":
		err = s.handleInstallationTargetEvent(ctx, payload)
	case "installation_repositories":
		err = s.handleInstallationRepositoriesEvent(ctx, payload)
	case "organization":
		err = s.handleOrganizationEvent(ctx, payload)
	case "team":
		err = s.handleTeamEvent(ctx, payload)
	case "membership":
		err = s.handleMembershipEvent(ctx, payload)
	case "repository":
		err = s.handleRepositoryEvent(ctx, payload)
	case "team_add":
		// "team_add" events are effectively a duplicate of "team" events with action "added_to_repository";
		// ignore them and just process the corresponding "team" events instead
		s.Infof("Ignoring event of type '%s'", event.EventType)
	case "repository_import":
		// These events are not supported by the GitHub library and are probably not required for BuildBeaver
		s.Infof("Ignoring event of type '%s'", event.EventType)
	default:
		s.Infof("Ignoring event of type '%s'", event.EventType)
	}

	return err
}

func (s *GitHubService) handlePushEvent(ctx context.Context, payload []byte) error {
	event := &github.PushEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}

	// Find the repo that the commits were pushed to
	repo, repoEnabled, err := s.checkRepoEnabled(ctx, event.GetRepo().GetID())
	if err != nil {
		return err
	}
	if !repoEnabled {
		s.Infof("Ignoring Push notification for repo that is not enabled")
		return nil
	}

	ghClient, err := s.makeGitHubAppInstallationClient(event.GetInstallation().GetID())
	if err != nil {
		return fmt.Errorf("error making github client: %w", err)
	}

	repoName := event.GetRepo().GetName()
	repoOwner := event.GetRepo().GetOwner().GetLogin()
	ref := event.GetRef()

	// Find the commit at the head of this ref, and build it if necessary
	err = s.buildLatestCommit(ctx, ghClient, repo, repoName, repoOwner, ref)
	if err != nil {
		return err
	}

	return nil
}

func (s *GitHubService) handlePullRequestEvent(ctx context.Context, payload []byte) error {
	// Read the event
	event := &github.PullRequestEvent{}
	err := json.Unmarshal(payload, event)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling event")
	}
	s.Tracef("Received pull request event, sender %q", event.GetSender().GetLogin())

	prNumber := event.GetPullRequest().GetNumber()
	if prNumber == 0 {
		s.Infof("Ignoring Pull Request notification where Pull Request number is not specified")
		return nil
	}

	// Return early if the PR action is not one that we care about
	switch event.GetAction() {
	case "opened", "synchronize", "closed", "reopened", "edited":
		// Carry on to process these events
	default:
		s.Infof("Ignoring Pull Request notification with action=%q", event.GetAction())
		return nil
	}

	// Find the repo this PR wants to update with new code (the base repo). Must be enabled.
	ghBaseRepo := event.GetRepo()
	ghBaseRepoID := ghBaseRepo.GetID()
	baseRepoName := ghBaseRepo.GetName()
	baseRepoOwner := ghBaseRepo.GetOwner().GetLogin() // use GetLogin(), Name field is often empty
	baseRepo, baseRepoEnabled, err := s.checkRepoEnabled(ctx, ghBaseRepoID)
	if err != nil {
		return err
	}
	if !baseRepoEnabled {
		s.Infof("Ignoring Pull Request notification for repo that is not enabled")
		return nil
	}
	baseRef := fixGithubBranchRef(event.GetPullRequest().GetBase().GetRef())

	// Find the repo the head of this PR is from (often the same as the base repo)
	ghHeadRepo := event.GetPullRequest().GetHead().GetRepo()
	ghHeadRepoID := ghHeadRepo.GetID()

	var refToBuild string
	if ghHeadRepoID == ghBaseRepoID {
		// PR is from another branch within the same repo - do a build of the branch head.
		// This will often already be underway in response to the push Webhook notification.
		refToBuild = fixGithubBranchRef(event.GetPullRequest().GetHead().GetRef())
	} else {
		// PR is from a different repo. Both base and head repo MUST be private.
		// TODO: Re-evaluate this once we have security settings to allow public repo PRs in certain circumstances
		s.Tracef("Cross-repo PR detected - checking both repos are private")
		if !ghBaseRepo.GetPrivate() || !ghHeadRepo.GetPrivate() {
			s.Infof("Ignoring Pull Request notification for cross-repo PR with public repo: base repo '%s' private=%v, head repo '%s' private=%v",
				ghBaseRepo.GetFullName(), ghBaseRepo.GetPrivate(), ghHeadRepo.GetFullName(), ghHeadRepo.GetPrivate())
			return nil
		}
		// use the special GitHub PR ref for the build, in the context of the base repo
		refToBuild = makeGithubPullRequestRef(prNumber)
	}

	ghClient, err := s.makeGitHubAppInstallationClient(event.GetInstallation().GetID())
	if err != nil {
		return fmt.Errorf("error making github client: %w", err)
	}

	// Record legal entity for the user specified in the PR (the requester) if we don't already have one
	pullRequestUserLegalEntity, _, err := s.findOrCreateGithubUser(ctx, event.GetPullRequest().GetUser())
	if err != nil {
		return err
	}

	// Create or update the PR in the database; do this for all the actions we care about
	prExternalID := GitHubIDToExternalResourceID(event.GetPullRequest().GetID())
	var mergedAt, closedAt *models.Time
	if event.GetPullRequest().MergedAt != nil {
		mergedAtTime := models.NewTime(event.GetPullRequest().GetMergedAt())
		mergedAt = &mergedAtTime
	}
	if event.GetPullRequest().ClosedAt != nil {
		closedAtTime := models.NewTime(event.GetPullRequest().GetClosedAt())
		closedAt = &closedAtTime
	}
	pullRequest := models.NewPullRequest(
		models.NewTime(time.Now()),
		mergedAt,
		closedAt,
		event.GetPullRequest().GetTitle(),
		event.GetPullRequest().GetState(),
		baseRepo.ID,
		pullRequestUserLegalEntity.ID,
		baseRef,
		refToBuild,
		&prExternalID,
	)
	_, _, err = s.pullRequestService.Upsert(ctx, nil, pullRequest)
	if err != nil {
		return errors.Wrap(err, "error upserting pull request in database")
	}
	s.Tracef("Added or updated Pull Request in database successfully")

	// Only attempt a build if the action indicates there has been a new commit
	if event.GetAction() == "opened" || event.GetAction() == "synchronize" {
		err = s.buildLatestCommit(ctx, ghClient, baseRepo, baseRepoName, baseRepoOwner, refToBuild)
		if err != nil {
			return err
		}
	}

	return nil
}

// fixGithubBranchRef will attempt to fix an issue with Ref strings supplied by the GitHub API that
// refer to a branch.
// In particular, inside PR notifications a Ref is given as "branchname" rather than "refs/heads/branchname".
// This function will attempt to fix a Ref if it looks like it needs fixing.
func fixGithubBranchRef(ref string) string {
	// look for branch name without prefix
	if ref != "" && !strings.HasPrefix(ref, "refs/") {
		return "refs/heads/" + ref
	}
	// ...otherwise return the ref unchanged
	return ref
}

// makeGithubPullRequestRef will return a GitHub ref to a pull request within the base repo
// (the target repo for the pull request, containing the base branch). This acts like a GitHub-specific
// branch or tag, allowing the code for the pull request to be fetched in the context of the base repo.
func makeGithubPullRequestRef(pullRequestNumber int) string {
	return fmt.Sprintf("refs/pull/%d/head", pullRequestNumber)
}

// checkRepoEnabled looks up the Repo with the specified GitHub Repo ID in our database and checks that it is enabled.
// If the Repo is found and enabled, a Repo object is returned.
// If the Repo is disabled or not found in the database, no error is returned but 'enabled' is returned as false.
// An error is returned if some other problem occurred while trying to check in the database.
func (s *GitHubService) checkRepoEnabled(ctx context.Context, ghRepoID int64) (
	repo *models.Repo, enabled bool, err error,
) {
	repoExternalID := GitHubIDToExternalResourceID(ghRepoID)
	repo, err = s.repoStore.ReadByExternalID(ctx, nil, repoExternalID)
	if err != nil {
		if gerror.ToNotFound(err) != nil {
			// Repo not found; this is not an error, just return as not enabled
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "error getting repo")
	}
	if !repo.Enabled {
		return nil, false, nil
	}

	return repo, true, nil
}
