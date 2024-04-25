package models

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const CommitResourceKind ResourceKind = "commit"

type CommitID struct {
	ResourceID
}

func NewCommitID() CommitID {
	return CommitID{ResourceID: NewResourceID(CommitResourceKind)}
}

func CommitIDFromResourceID(id ResourceID) CommitID {
	return CommitID{ResourceID: id}
}

type Commit struct {
	// ID is the unique id of the commit
	ID        CommitID `json:"id" goqu:"skipupdate" db:"commit_id"`
	CreatedAt Time     `json:"created_at" goqu:"skipupdate" db:"commit_created_at"`
	// RepoID that the commit was made against
	RepoID RepoID `json:"repo_id" db:"commit_repo_id"`
	// Config is the raw bytes of the pipeline config file found in the commit
	Config BinaryBlob `json:"-" db:"commit_config"`
	// ConfigType is the type of the config file found in the commit
	ConfigType ConfigType `json:"config_type" db:"commit_config_type"`
	// SHA is the unique SHA hash of the commit
	SHA string `json:"sha" db:"commit_sha"`
	// Message is the commit message
	Message string `json:"message" db:"commit_message"`
	// AuthorID is the id of the legal entity that authored the commit, if known
	AuthorID LegalEntityID `json:"author_id" db:"commit_author_id"`
	// AuthorName is the author name recorded on the commit
	AuthorName string `json:"author_name" db:"commit_author_name"`
	// AuthorEmail is the author email address recorded on the commit
	AuthorEmail string `json:"author_email" db:"commit_author_email"`
	// CommitterID is the id of the legal entity that committed the commit, if known
	CommitterID LegalEntityID `json:"committer_id" db:"commit_committer_id"`
	// CommitterName is the committer name recorded on the commit, if any
	CommitterName string `json:"committer_name" db:"commit_committer_name"`
	// CommitterEmail is the committer email address recorded on the commit, if any
	CommitterEmail string `json:"committer_email" db:"commit_committer_email"`
	// Link is a url to the commit in the SCM.
	Link string `json:"link" db:"commit_link"`
}

func NewCommit(
	now Time,
	repoID RepoID,
	config []byte,
	configType ConfigType,
	sha string,
	message string,
	authorID LegalEntityID,
	authorName string,
	authorEmail string,
	committerID LegalEntityID,
	committerName string,
	committerEmail string,
	link string,
) *Commit {
	return &Commit{
		ID:             NewCommitID(),
		CreatedAt:      now,
		RepoID:         repoID,
		Config:         config,
		ConfigType:     configType,
		SHA:            sha,
		Message:        message,
		AuthorID:       authorID,
		AuthorName:     authorName,
		AuthorEmail:    authorEmail,
		CommitterID:    committerID,
		CommitterName:  committerName,
		CommitterEmail: committerEmail,
		Link:           link,
	}
}

func (m *Commit) GetKind() ResourceKind {
	return CommitResourceKind
}

func (m *Commit) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Commit) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Commit) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if m.ConfigType.Valid() && m.Config == nil {
		result = multierror.Append(result, errors.New("error config must be set when config type is set"))
	}
	if m.SHA == "" {
		result = multierror.Append(result, errors.New("error sha must be set"))
	}
	if m.Message == "" {
		result = multierror.Append(result, errors.New("error message must be set"))
	}
	return result.ErrorOrNil()
}
