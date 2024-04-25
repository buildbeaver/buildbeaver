package dto

import "github.com/buildbeaver/buildbeaver/common/models"

// RunnableJob contains the job with steps, as well as all context information required to run the job including an
// authentication JWT (token).
type RunnableJob struct {
	// Repo that was committed to.
	Repo *models.Repo `json:"repo"`
	// Commit that the job was generated from.
	Commit *models.Commit `json:"commit"`
	// Jobs is the set of jobs that this job depends on.
	Jobs []*models.Job `json:"jobs"`
	// JWT (JSON Web Token) that dynamic build jobs can use to access the dynamic API for this build
	JWT string `json:"jwt"`
	// WorkflowsToRun is a list of workflows that have been requested to run as part of the build options.
	// This does not include workflows that become required as new dependencies when new jobs are submitted.
	WorkflowsToRun []models.ResourceName `json:"workflows_to_run"`
	*JobGraph
}
