package documents

import (
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type Commit struct {
	ID        models.CommitID `json:"id"`
	CreatedAt models.Time     `json:"created_at"`

	// RepoID that the commit was made against
	RepoID models.RepoID `json:"repo_id"`
	// ConfigType is the type of the config file found in the commit (Config itself is not included in the document)
	ConfigType models.ConfigType `json:"config_type"`
	// SHA is the unique SHA hash of the commit
	SHA string `json:"sha"`
	// Message is the commit message
	Message string `json:"message"`
	// AuthorID is the id of the legal entity that authored the commit, if known
	AuthorID models.LegalEntityID `json:"author_id"`
	// AuthorName is the author name recorded on the commit
	AuthorName string `json:"author_name"`
	// AuthorEmail is the author email address recorded on the commit
	AuthorEmail string `json:"author_email"`
	// CommitterID is the id of the legal entity that committed the commit, if known
	CommitterID models.LegalEntityID `json:"committer_id"`
	// CommitterName is the committer name recorded on the commit, if any
	CommitterName string `json:"committer_name"`
	// CommitterEmail is the committer email address recorded on the commit, if any
	CommitterEmail string `json:"committer_email"`
	// Link is a url to the commit in the SCM.
	Link string `json:"link"`

	CommitterURL string `json:"committer_url"`
	AuthorURL    string `json:"author_url"`
}

func MakeCommit(rctx routes.RequestContext, commit *models.Commit) *Commit {
	return &Commit{
		ID:        commit.ID,
		CreatedAt: commit.CreatedAt,

		RepoID:         commit.RepoID,
		ConfigType:     commit.ConfigType,
		SHA:            commit.SHA,
		Message:        commit.Message,
		AuthorID:       commit.AuthorID,
		AuthorName:     commit.AuthorName,
		AuthorEmail:    commit.AuthorEmail,
		CommitterID:    commit.CommitterID,
		CommitterName:  commit.CommitterName,
		CommitterEmail: commit.CommitterEmail,
		Link:           commit.Link,

		CommitterURL: routes.MakeLegalEntityLink(rctx, commit.CommitterID),
		AuthorURL:    routes.MakeLegalEntityLink(rctx, commit.AuthorID),
	}
}
