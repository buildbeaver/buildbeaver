package dto

import "github.com/buildbeaver/buildbeaver/common/models"

type QueuedBuild struct {
	*BuildGraph
	// Repo that was committed to.
	Repo *models.Repo `json:"repo"`
	// Commit that the build was generated from.
	Commit *models.Commit `json:"commit"`
}
