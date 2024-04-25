package runner

import (
	"context"
	"io"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type APIClient interface {
	// Ping acts as a pre-flight check for a runner, contacting the server and checking that authentication
	// and registration are in place ready to dequeue build jobs.
	Ping(ctx context.Context) error
	// SendRuntimeInfo sends information about the runtime environment and version for this runner to the server.
	SendRuntimeInfo(ctx context.Context, info *documents.PatchRuntimeInfoRequest) error
	// Dequeue returns the next build job that is ready to be executed, or
	// nil if there are currently no queued builds.
	Dequeue(ctx context.Context) (*documents.RunnableJob, error)
	// UpdateJobStatus updates the status of the specified job.
	// If the status is finished, err can be supplied to signal the job failed with an error
	// or nil to signify the job succeeded.
	UpdateJobStatus(
		ctx context.Context,
		jobID models.JobID,
		status models.WorkflowStatus,
		jobError *models.Error,
		eTag models.ETag) (*documents.Job, error)
	// UpdateJobFingerprint sets the fingerprint that has been calculated for a job. If the build is not configured
	// with the force option (e.g. force=false), the server will attempt to locate a previously successful job with a
	// matching fingerprint and indirect this job to it. If an indirection has been set, the agent must skip the job.
	UpdateJobFingerprint(
		ctx context.Context,
		jobID models.JobID,
		jobFingerprint string,
		jobFingerprintHashType *models.HashType,
		eTag models.ETag) (*documents.Job, error)
	// UpdateStepStatus updates the status of the specified step.
	// If the status is finished, err can be supplied to signal the step failed with an error
	// or nil to signify the step succeeded.
	UpdateStepStatus(
		ctx context.Context,
		stepID models.StepID,
		status models.WorkflowStatus,
		stepError *models.Error,
		eTag models.ETag) (*documents.Step, error)
	// GetSecretsPlaintext gets all secrets for the specified repo in plaintext.
	GetSecretsPlaintext(ctx context.Context, repoID models.RepoID) ([]*models.SecretPlaintext, error)
	// CreateArtifact a new artifact with its contents provided by reader. It is the caller's responsibility to close reader.
	// Returns store.ErrAlreadyExists if an artifact with matching unique properties already exists.
	CreateArtifact(
		ctx context.Context,
		jobID models.JobID,
		groupName models.ResourceName,
		relativePath string,
		reader io.ReadSeeker) (*documents.Artifact, error)
	// GetArtifactData returns a reader to the data of an artifact.
	// It is the caller's responsibility to close the reader.
	GetArtifactData(ctx context.Context, artifactID models.ArtifactID) (io.ReadCloser, error)
	// SearchArtifacts searches all artifacts for a build. Use cursor to page through results, if any.
	SearchArtifacts(ctx context.Context, buildID models.BuildID, search *models.ArtifactSearch) (models.ArtifactSearchPaginator, error)
	// OpenLogWriteStream opens a writable stream to the specified log. Close the writer to finish writing.
	OpenLogWriteStream(ctx context.Context, logID models.LogDescriptorID) (io.WriteCloser, error)
}
