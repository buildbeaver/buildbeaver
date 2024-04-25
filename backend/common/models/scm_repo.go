package models

type SCMAuth interface {
	// Name returns the name of the SCM this authentication information is for.
	Name() SystemName
}

// SCMRepo is a basic representation of a repo that exists in an SCM (such as GitHub)
// and may or may not have been imported/configured within our server (where we would
// represent it as a models.Repo).
type SCMRepo struct {
	// ID uniquely identifies this repo within an SCM
	ID *ExternalResourceID `json:"id"`
	// Repo is the repo within the server that this SCM repo corresponds to,
	// or nil if this SCM repo is not enabled.
	Repo *Repo `json:"repo"`
	// Name is the human-friendly name of the repo e.g. joeblogs/big-repo
	Name          string `json:"name"`
	Description   string `json:"description"`
	SSHURL        string `json:"ssh_url"`
	HTTPURL       string `json:"http_url"`
	Link          string `json:"link"`
	DefaultBranch string `json:"default_branch"`
	Enabled       bool   `json:"enabled"`
}
