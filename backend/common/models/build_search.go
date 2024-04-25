package models

import (
	"github.com/buildbeaver/buildbeaver/common/gerror"
)

type BuildSearch struct {
	Pagination
	// RepoID is the Repo associated with builds being searched for, or nil to include builds for any repo.
	RepoID *RepoID `json:"repo_id"`
	// CommitID is the Commit for builds being searched for, or nil to include builds for any commit.
	CommitID *CommitID `json:"commit_id"`
	// CommitSHA is a prefix match (minimum 4 characters) of a commit SHA, or empty string to include builds of any commit SHA.
	CommitSHA string `json:"commit_sha"`
	// CommitAuthorID is the legal entity id for commits to search by, or nil to includes builds by any author.
	CommitAuthorID *LegalEntityID `json:"commit_author_id"`
	// Ref is the ref associated with builds being searched for, or empty string to include builds of any ref.
	Ref string `json:"ref"`
	// ExcludeStatuses defines a list where builds will not be included in the results if their current status is in the list
	ExcludeStatuses []WorkflowStatus `json:"exclude_statuses"`
	// ExcludeFailed is true if the search results should not include builds that have finished with an error
	ExcludeFailed bool `json:"exclude_failed"`
	// IncludeStatuses defines a list of statuses where builds will be included if their status is in the list.
	IncludeStatuses []WorkflowStatus `json:"status"`
	// LegalEntityID defines the legal entity id that is filtered against a repo that a build belongs to.
	LegalEntityID *LegalEntityID `json:"legal_entity_id"`
}

func NewBuildSearch() *BuildSearch {
	return &BuildSearch{Pagination: Pagination{}}
}

// NewBuildSearchForRepo returns search criteria to search for builds for a particular repo.
// Other search criteria can be specified to narrow the search.
func NewBuildSearchForRepo(repoID RepoID, ref string, excludeFailed bool, excludeStatuses []WorkflowStatus, limit int) *BuildSearch {
	return &BuildSearch{
		Pagination: Pagination{
			Limit:  limit,
			Cursor: nil,
		},
		RepoID:          &repoID,
		CommitID:        nil, // any commit
		Ref:             ref,
		ExcludeFailed:   excludeFailed,
		ExcludeStatuses: excludeStatuses,
	}
}

// NewBuildSearchForCommit returns search criteria to search for builds for a particular commit.
// By specifying a commit, the repo for that commit is implicitly specified.
// Other search criteria can be specified to narrow the search.
func NewBuildSearchForCommit(commitID CommitID, ref string, excludeFailed bool, excludeStatuses []WorkflowStatus, limit int) *BuildSearch {
	return &BuildSearch{
		Pagination: Pagination{
			Limit:  limit,
			Cursor: nil,
		},
		RepoID:          nil, // repo is implied by the commit
		CommitID:        &commitID,
		Ref:             ref,
		ExcludeFailed:   excludeFailed,
		ExcludeStatuses: excludeStatuses,
	}
}

func (m *BuildSearch) Validate() error {
	if m.RepoID == nil && m.Ref != "" {
		return gerror.NewErrValidationFailed("RepoID must be specified when a ref is specified")
	}

	return nil
}
