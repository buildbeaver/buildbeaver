package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const PullRequestResourceKind ResourceKind = "pull-request"

type PullRequestID struct {
	ResourceID
}

func NewPullRequestID() PullRequestID {
	return PullRequestID{ResourceID: NewResourceID(PullRequestResourceKind)}
}

func PullRequestIDFromResourceID(id ResourceID) PullRequestID {
	return PullRequestID{ResourceID: id}
}

type PullRequest struct {
	// ID is the unique id of the commit
	ID        PullRequestID `json:"id" goqu:"skipupdate" db:"pull_request_id"`
	CreatedAt Time          `json:"created_at" goqu:"skipupdate" db:"pull_request_created_at"`
	UpdatedAt Time          `json:"updated_at" db:"pull_request_updated_at"`
	// MergedAt is the date and time at which the PR was merged, if it has been
	MergedAt *Time `json:"merged_at,omitempty" db:"pull_request_merged_at"`
	// ClosedAt is the date and time at which the PR was closed, if it has been
	ClosedAt *Time `json:"closed_at,omitempty" db:"pull_request_closed_at"`
	// Title is the human-readable title of the PR
	Title string `json:"title" db:"pull_request_title"`
	// State is the current state of the PR in the SCM (e.g. GitHub)
	State string `json:"state" db:"pull_request_state"`
	// RepoID specifies the Repo that the PR is requesting changes to
	RepoID RepoID `json:"repo_id" db:"pull_request_repo_id"`
	// UserID is the id of the legal entity of the SCM user who submitted the PR
	UserID LegalEntityID `json:"user_id" db:"pull_request_user_id"`
	// BaseRef is the ref of the branch the PR is based off (before the changes requested in the PR)
	BaseRef string `json:"base_ref" db:"pull_request_base_ref"`
	// HeadRef is the ref containing the requested changes, i.e. the one to build
	// For single-repo PRs this will be the head of the branch containing the changes.
	// For cross-repo PRs this will be the GitHub PR ref, in the context of the base repo.
	HeadRef string `json:"head_ref" db:"pull_request_head_ref"`
	// ExternalID is the ID of this PR in the external SCM
	ExternalID *ExternalResourceID `json:"external_id" db:"pull_request_external_id"`
}

func NewPullRequest(
	now Time,
	mergedAt *Time,
	closedAt *Time,
	title string,
	state string,
	repoID RepoID,
	userID LegalEntityID,
	baseRef string,
	headRef string,
	externalID *ExternalResourceID,
) *PullRequest {
	return &PullRequest{
		ID:         NewPullRequestID(),
		CreatedAt:  now,
		UpdatedAt:  now,
		MergedAt:   mergedAt,
		ClosedAt:   closedAt,
		Title:      title,
		State:      state,
		RepoID:     repoID,
		UserID:     userID,
		BaseRef:    baseRef,
		HeadRef:    headRef,
		ExternalID: externalID,
	}
}

func (m *PullRequest) GetKind() ResourceKind {
	return PullRequestResourceKind
}

func (m *PullRequest) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *PullRequest) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *PullRequest) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if m.State == "" {
		result = multierror.Append(result, errors.New("error state must be set"))
	}
	if m.BaseRef == "" {
		result = multierror.Append(result, errors.New("error Base Ref must be set"))
	}
	if m.HeadRef == "" {
		result = multierror.Append(result, errors.New("error Head Ref must be set"))
	}
	if m.ExternalID != nil {
		if !m.ExternalID.Valid() {
			result = multierror.Append(result, errors.New("error external id is invalid"))
		}
	}
	return result.ErrorOrNil()
}
