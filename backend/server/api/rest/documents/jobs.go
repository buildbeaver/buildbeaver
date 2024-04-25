package documents

import (
	"fmt"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

type Job struct {
	baseResourceDocument

	ID        models.JobID `json:"id"`
	CreatedAt models.Time  `json:"created_at"`
	UpdatedAt models.Time  `json:"updated_at"`
	DeletedAt *models.Time `json:"deleted_at,omitempty"`
	ETag      models.ETag  `json:"etag" hash:"ignore"`

	// Name of the job.
	Name models.ResourceName `json:"name"`
	// Workflow the job is a part of, or empty if the job is part of the default workflow
	Workflow models.ResourceName `json:"workflow"`
	// Description is an optional human-readable description of the job.
	Description string `json:"description" db:"job_description"`
	// Depends describes the dependencies this job has on other jobs.
	Depends []*JobDependency `json:"depends"`
	// Services is a list of services to run in the background for the duration of the job.
	// Services are started before the first step is run, and stopped after the last step completes.
	Services []*Service `json:"services"`
	// Type of the job (e.g. docker, exec etc.)
	Type models.JobType `json:"type"`
	// RunsOn contains a set of labels that this job requires runners to have.
	RunsOn []models.Label `json:"runs_on"`
	// DockerConfig provides information about how to configure Docker to run this job, if Type is 'docker'.
	DockerConfig *DockerConfig `json:"docker"`
	// StepExecution determines how the runner will execute steps within this job.
	StepExecution models.StepExecution `json:"step_execution"`
	// FingerprintCommands contains zero or more shell commands to execute to generate a unique fingerprint for the job.
	// Two jobs in the same repo with the same name and fingerprint are considered identical.
	FingerprintCommands []models.Command `json:"fingerprint_commands"`
	// ArtifactDefinitions contains a list of artifacts the job is expected to produce that
	// will be saved to the artifact store at the end of the job's execution.
	ArtifactDefinitions []*ArtifactDefinition `json:"artifact_definitions"`
	// Environment contains a list of environment variables to export prior to executing the job.
	Environment []*EnvVar `json:"environment"`

	// The ID of the build this job is a part of.
	BuildID models.BuildID `json:"build_id"`
	// RepoID that was committed to.
	RepoID models.RepoID `json:"repo_id"`
	// CommitID that the job was generated from.
	CommitID models.CommitID `json:"commit_id"`
	// LogDescriptorID points to the log for this job.
	LogDescriptorID models.LogDescriptorID `json:"log_descriptor_id"`
	// RunnerID is the id of the runner this job executed on, or empty if the job has not run yet (or did/will not run).
	RunnerID models.RunnerID `json:"runner_id"`
	// IndirectToJobID records the ID of a job that previously ran successfully as part of another build
	// and which is functionally identical to this job. If this is set it means this job did not actually
	// run to avoid redundantly running the same thing more than once.
	IndirectToJobID models.JobID `json:"indirect_to_job_id"`
	// Ref is the git ref from the build that the job was generated from (e.g. branch or tag)
	Ref string `json:"ref"`
	// Status reflects where the job is in the queue.
	Status models.WorkflowStatus `json:"status"`
	// Error is set if the job finished with an error (or nil if the job succeeded).
	Error *models.Error `json:"error"`
	// Timings records the times at which the job transitioned between statuses.
	Timings WorkflowTimings `json:"timings"`
	// Fingerprint contains the hashed output of FingerprintCommands, as well as any other inputs the agent added (such
	// as artifact hashes). This is only available after the job has run successfully.
	Fingerprint string `json:"fingerprint"`
	// FingerprintHashType is the type of hashing algorithm used to produce the fingerprint.
	FingerprintHashType *models.HashType `json:"fingerprint_hash_type"`
	// DefinitionDataHashType is the type of hashing algorithm used to produce DefinitionDataHash.
	DefinitionDataHashType models.HashType `json:"definition_data_hash_type" `
	// DefinitionDataHash is the hex-encoded hash of the job's definition data.
	// NOTE: This hash captures the hash of the job's step definition data too.
	DefinitionDataHash string `json:"definition_data_hash"`

	LogDescriptorURL string  `json:"log_descriptor_url"`
	IndirectJobURL   *string `json:"indirect_job_url"`
}

func MakeJob(rctx routes.RequestContext, job *models.Job) *Job {
	var indirectJobURL *string
	if !job.IndirectToJobID.IsZero() {
		link := routes.MakeJobLink(rctx, job.IndirectToJobID)
		indirectJobURL = &link
	}
	return &Job{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeJobLink(rctx, job.ID),
		},

		ID:        job.ID,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
		DeletedAt: job.DeletedAt,
		ETag:      job.ETag,

		Name:                job.Name,
		Workflow:            job.Workflow,
		Description:         job.Description,
		Depends:             MakeJobDependencies(job.Depends),
		Services:            MakeServices(job.Services),
		Type:                job.Type,
		RunsOn:              job.RunsOn,
		DockerConfig:        MakeDockerConfig(job.DockerImage, job.DockerImagePullStrategy, job.DockerAuth, job.DockerShell),
		StepExecution:       job.StepExecution,
		FingerprintCommands: job.FingerprintCommands,
		ArtifactDefinitions: MakeArtifactDefinitions(job.ArtifactDefinitions),
		Environment:         MakeEnvVars(job.Environment),

		BuildID:                job.BuildID,
		RepoID:                 job.RepoID,
		CommitID:               job.CommitID,
		LogDescriptorID:        job.LogDescriptorID,
		RunnerID:               job.RunnerID,
		IndirectToJobID:        job.IndirectToJobID,
		Ref:                    job.Ref,
		Status:                 job.Status,
		Error:                  job.Error,
		Timings:                *MakeWorkflowTimings(&job.Timings),
		Fingerprint:            job.Fingerprint,
		FingerprintHashType:    job.FingerprintHashType,
		DefinitionDataHashType: job.DefinitionDataHashType,
		DefinitionDataHash:     job.DefinitionDataHash,

		LogDescriptorURL: routes.MakeLogLink(rctx, job.LogDescriptorID),
		IndirectJobURL:   indirectJobURL,
	}
}

func MakeJobs(rctx routes.RequestContext, jobs []*models.Job) []*Job {
	var docs []*Job
	for _, job := range jobs {
		docs = append(docs, MakeJob(rctx, job))
	}
	return docs
}

func (d *Job) GetID() models.ResourceID {
	return d.ID.ResourceID
}

func (d *Job) GetKind() models.ResourceKind {
	return models.JobResourceKind
}

func (d *Job) GetCreatedAt() models.Time {
	return d.CreatedAt
}

// JobGraph is a document suitable for returning to API clients as part of a broader document.
// It provides a job and all the steps within that job.
type JobGraph struct {
	baseResourceDocument
	Job   *Job    `json:"job"`
	Steps []*Step `json:"steps"`
}

func (d *JobGraph) GetID() models.ResourceID {
	return d.Job.ID.ResourceID
}

func (d *JobGraph) GetKind() models.ResourceKind {
	return models.JobResourceKind
}

func (d *JobGraph) GetCreatedAt() models.Time {
	return d.Job.CreatedAt
}

func MakeJobGraph(rctx routes.RequestContext, job *dto.JobGraph) *JobGraph {
	return &JobGraph{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeJobLink(rctx, job.ID),
		},
		Job:   MakeJob(rctx, job.Job),
		Steps: MakeSteps(rctx, job.Steps),
	}
}

func MakeJobGraphs(rctx routes.RequestContext, jobs []*dto.JobGraph) []*JobGraph {
	var docs []*JobGraph
	for _, model := range jobs {
		docs = append(docs, MakeJobGraph(rctx, model))
	}
	return docs
}

// RunnableJob is a document suitable for returning to API clients when they dequeue a job to run.
// It contains the job with steps, as well as context information required to run the job including an
// authentication JWT (token).
type RunnableJob struct {
	baseResourceDocument
	// Job contains all information about the job resource
	Job *Job
	// Steps contains information for all the steps in the job
	Steps []*Step `json:"steps"`
	// Repo that was committed to.
	Repo *Repo `json:"repo"`
	// Commit that the job was generated from.
	Commit *Commit `json:"commit"`
	// Jobs is the set of jobs that this job depends on.
	Jobs []*Job `json:"jobs"`
	// JWT (JSON Web Token) that dynamic build jobs can use to access the dynamic API for this build
	JWT string `json:"jwt"`
	// WorkflowsToRun is a list of workflows that have been requested to run as part of the build options.
	// This does not include workflows that become required as new dependencies when new jobs are submitted.
	WorkflowsToRun []models.ResourceName `json:"workflows_to_run"`
	// Log descriptor for the log to write to for this job.
	LogDescriptorURL string `json:"log_descriptor_url"`
}

func MakeRunnableJob(rctx routes.RequestContext, job *dto.RunnableJob) *RunnableJob {
	return &RunnableJob{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeJobLink(rctx, job.ID),
		},
		Job:              MakeJob(rctx, job.Job),
		Steps:            MakeSteps(rctx, job.Steps),
		Repo:             MakeRepo(rctx, job.Repo),
		Commit:           MakeCommit(rctx, job.Commit),
		Jobs:             MakeJobs(rctx, job.Jobs),
		JWT:              job.JWT,
		WorkflowsToRun:   job.WorkflowsToRun,
		LogDescriptorURL: routes.MakeLogLink(rctx, job.LogDescriptorID),
	}
}

func (d *RunnableJob) GetLink() string {
	return d.Job.GetLink()
}

func (d *RunnableJob) GetID() models.ResourceID {
	return d.Job.GetID()
}

func (d *RunnableJob) GetKind() models.ResourceKind {
	return d.Job.GetKind()
}

func (d *RunnableJob) GetCreatedAt() models.Time {
	return d.Job.GetCreatedAt()
}

type PatchJobRequest struct {
	// Fingerprint is the unique fingerprint of the job.
	Fingerprint *string `json:"fingerprint"`
	// FingerprintHashType is the type of hashing algorithm used to produce the fingerprint.
	FingerprintHashType *models.HashType `json:"fingerprint_hash_type"`
	// Status reflects where the job is in processing.
	Status *models.WorkflowStatus `json:"status"`
	// Error signifies the job finished with an error, if status is failed.
	Error *models.Error `json:"error"`
}

func (d *PatchJobRequest) Bind(r *http.Request) error {
	if (d.Status == nil) == (d.Fingerprint == nil) {
		return gerror.NewErrValidationFailed("Only one of status or fingerprint may be specified")
	}
	if d.Status != nil && !d.Status.Valid() {
		return gerror.NewErrValidationFailed(fmt.Sprintf("Invalid status: %s", d.Status))
	}
	if d.Error.Valid() && (d.Status == nil || *d.Status != models.WorkflowStatusFailed) {
		return gerror.NewErrValidationFailed("Error can only be specified on failed jobs")
	}
	if d.Status != nil && *d.Status == models.WorkflowStatusFailed && !d.Error.Valid() {
		return gerror.NewErrValidationFailed("Failed workflow statuses must be accompanied by an error")
	}
	if d.Fingerprint != nil && *d.Fingerprint == "" {
		return gerror.NewErrValidationFailed("Fingerprint cannot be empty")
	}
	if d.Fingerprint != nil && d.FingerprintHashType == nil {
		return gerror.NewErrValidationFailed("Fingerprint hash type must be specified")
	}
	return nil
}

// JobDependency declares that one job depends on the successful execution of another, and optionally
// that the dependent job consumes one or more artifacts from the other.
type JobDependency struct {
	Workflow             models.ResourceName   `json:"workflow"`
	JobName              models.ResourceName   `json:"job_name"`
	ArtifactDependencies []*ArtifactDependency `json:"artifact_dependencies"`
}

func MakeJobDependency(dependency *models.JobDependency) *JobDependency {
	return &JobDependency{
		Workflow:             dependency.Workflow,
		JobName:              dependency.JobName,
		ArtifactDependencies: MakeArtifactDependencies(dependency.ArtifactDependencies),
	}
}

func MakeJobDependencies(dependencies models.JobDependencies) []*JobDependency {
	var docs []*JobDependency
	for _, model := range dependencies {
		docs = append(docs, MakeJobDependency(model))
	}
	return docs
}

type ArtifactDependency struct {
	Workflow  models.ResourceName `json:"workflow"`
	JobName   models.ResourceName `json:"job_name"`
	GroupName models.ResourceName `json:"group_name"`
}

func MakeArtifactDependency(dependency *models.ArtifactDependency) *ArtifactDependency {
	return &ArtifactDependency{
		Workflow:  dependency.Workflow,
		JobName:   dependency.JobName,
		GroupName: dependency.GroupName,
	}
}
func MakeArtifactDependencies(dependencies []*models.ArtifactDependency) []*ArtifactDependency {
	var docs []*ArtifactDependency
	for _, dependency := range dependencies {
		docs = append(docs, MakeArtifactDependency(dependency))
	}
	return docs
}
