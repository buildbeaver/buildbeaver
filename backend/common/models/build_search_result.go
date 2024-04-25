package models

// BuildSearchResult represents the content for a search against the Builds API.
type BuildSearchResult struct {
	// Note that a BuildSearchResult instance is passed to ResourceTable.ListIn() and this function does
	// not support composing/nesting the primary resource table, so we must embed Build here.
	// (See comment inside ResourceTable.ListIn() for details).
	*Build
	Repo   *Repo   `db:"repos"`
	Commit *Commit `db:"commits"`
}
