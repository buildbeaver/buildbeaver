package github

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
)

// CommitStatusWorkItem is a work item that will send a commit status update via the GitHub API.
const CommitStatusWorkItem models.WorkItemType = "CommitStatus"

// CommitStatusWorkItemData is serialized to JSON and stored in the Data field of a SendCommitStatusWorkItem.
// This struct includes individual fields to go into github.RepoStatus, not the whole struct since it will
// be persisted in the database, and we want backwards compatibility if github.RepoStatus changes.
type CommitStatusWorkItemData struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	GitHubState    string
	TargetURL      string
	Description    string
	ContextText    string
}

func NewCommitStatusWorkItem(
	installationID int64,
	owner string,
	repo string,
	sha string,
	gitHubState string,
	targetURL string,
	description string,
	contextText string,
) *models.WorkItem {
	data := &CommitStatusWorkItemData{
		InstallationID: installationID,
		Owner:          owner,
		Repo:           repo,
		SHA:            sha,
		GitHubState:    gitHubState,
		TargetURL:      targetURL,
		Description:    description,
		ContextText:    contextText,
	}
	dataJson, err := json.Marshal(data)
	if err != nil {
		// If this happens we have a bug in SendCommitStatusWorkItemData definition
		panic("Unable to marshal SendCommitStatusWorkItemData object to JSON")
	}

	// Concurrency key is the combination of 'github-commit', repo and SHA
	concurrencyKey := models.NewWorkItemConcurrencyKey(fmt.Sprintf("github-commit/%s/%s", repo, sha))

	return models.NewWorkItem(CommitStatusWorkItem, string(dataJson), concurrencyKey, models.NewTime(time.Now()))
}
