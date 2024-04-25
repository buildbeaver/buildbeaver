package models

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	JobResourceKind ResourceKind = "job"
)

type JobID struct {
	ResourceID
}

func NewJobID() JobID {
	return JobID{ResourceID: NewResourceID(JobResourceKind)}
}

func JobIDFromResourceID(id ResourceID) JobID {
	return JobID{ResourceID: id}
}

func ParseJobID(str string) (JobID, error) {
	resourceID, err := ParseResourceID(str)
	if err != nil {
		return JobID{}, fmt.Errorf("error parsing Job ID: %w", err)
	}
	return JobIDFromResourceID(resourceID), nil
}

// Job represents a single job in a multi-job pipeline. A job contains
// multiple steps that may be executed in a fan-out/fan-in workflow.
type Job struct {
	JobMetadata
	JobData
}

type JobMetadata struct {
	ID        JobID `json:"id" goqu:"skipupdate" db:"job_id"`
	CreatedAt Time  `json:"created_at" goqu:"skipupdate" db:"job_created_at"`
	UpdatedAt Time  `json:"updated_at" db:"job_updated_at"`
	DeletedAt *Time `json:"deleted_at,omitempty" db:"job_deleted_at"`
	ETag      ETag  `json:"etag" db:"job_etag" hash:"ignore"`
}

type JobData struct {
	JobDefinitionData
	BuildID BuildID `json:"build_id" db:"job_build_id"`
	// RepoID that was committed to.
	RepoID RepoID `json:"repo_id" db:"job_repo_id"`
	// CommitID that the job was generated from.
	CommitID CommitID `json:"commit_id" db:"job_commit_id"`
	// LogDescriptorID points to the log for this job.
	LogDescriptorID LogDescriptorID `json:"log_descriptor_id" db:"job_log_descriptor_id"`
	// RunnerID is the id of the runner this job executed on, or empty if the job has not run yet (or did/will not run).
	RunnerID RunnerID `json:"runner_id" db:"job_runner_id"`
	// IndirectToJobID records the ID of a job that previously ran successfully as part of another build
	// and which is functionally identical to this job. If this is set it means this job did not actually
	// run to avoid redundantly running the same thing more than once.
	IndirectToJobID JobID `json:"indirect_to_job_id" db:"job_indirect_to_job_id"`
	// Ref is the git ref from the build that the job was generated from (e.g. branch or tag)
	Ref string `json:"ref" db:"job_ref"`
	// Status reflects where the job is in the queue.
	Status WorkflowStatus `json:"status" db:"job_status"`
	// Error is set if the job finished with an error (or nil if the job succeeded).
	Error *Error `json:"error" db:"job_error"`
	// Timings records the times at which the job transitioned between statuses.
	Timings WorkflowTimings `json:"timings" db:"job_timings"`
	// Fingerprint contains the hashed output of FingerprintCommands, as well as any other inputs the agent added (such
	// as artifact hashes). This is only available after the job has run successfully.
	Fingerprint string `json:"fingerprint" db:"job_fingerprint"`
	// FingerprintHashType is the type of hashing algorithm used to produce the fingerprint.
	FingerprintHashType *HashType `json:"fingerprint_hash_type" db:"job_fingerprint_hash_type"`
	// DefinitionDataHashType is the type of hashing algorithm used to produce DefinitionDataHash.
	DefinitionDataHashType HashType `json:"definition_data_hash_type" db:"job_definition_data_hash_type"`
	// DefinitionDataHash is the hex-encoded hash of the job's definition data.
	// NOTE: This hash captures the hash of the job's step definition data too.
	DefinitionDataHash string `json:"definition_data_hash" db:"job_definition_data_hash"`
}

type JobDefinitionData struct {
	// Name of the job (excluding workflow)
	Name ResourceName `json:"name" db:"job_name"`
	// Workflow the job is a part of, or empty if the job is part of the default workflow
	Workflow ResourceName `json:"workflow" db:"job_workflow"`
	// Description is an optional human-readable description of the job.
	Description string `json:"description" db:"job_description"`
	// Depends describes the dependencies this job has on other jobs.
	Depends JobDependencies `json:"depends" db:"job_depends"`
	// Services are a list of services to run in the background for the duration of the job.
	// Services are started before the first step is run, and stopped after the last step completes.
	Services JobServices `json:"services" db:"job_services"`
	// Type of the job (e.g. docker, exec etc.)
	Type JobType `json:"type" db:"job_type"`
	// RunsOn contains a set of labels that this job requires runners to have.
	RunsOn Labels `json:"runs_on" db:"job_runs_on"`
	// DockerImage is the default Docker image to run the job's steps in, if the job is of type Docker.
	// In the future, steps may override this property by setting their own DockerImage.
	DockerImage string `json:"docker_image" db:"job_docker_image"`
	// DockerImagePullStrategy determines if/when the Docker image is pulled during job execution, if the job is of type Docker.
	DockerImagePullStrategy DockerPullStrategy `json:"docker_pull" db:"job_docker_image_pull_strategy"`
	// DockerAuth contains the optional authentication for pulling a docker image, if the job is of type Docker.
	DockerAuth *DockerAuth `json:"docker_auth" db:"job_docker_auth"`
	// DockerShell is the path to the shell to use to run build scripts with inside the container.
	DockerShell *string `json:"docker_shell" db:"job_docker_shell"`
	// StepExecution determines how the runner will execute steps within this job.
	StepExecution StepExecution `json:"step_execution" db:"job_step_execution"`
	// FingerprintCommands contains zero or more shell commands to execute to generate a unique fingerprint for the job.
	// Two jobs in the same repo with the same name and fingerprint are considered identical.
	FingerprintCommands Commands `json:"fingerprint_commands" db:"job_fingerprint_commands"`
	// ArtifactDefinitions contains a list of artifacts the job is expected to produce that
	// will be saved to the artifact store at the end of the job's execution.
	ArtifactDefinitions ArtifactDefinitions `json:"artifact_definitions" db:"job_artifact_definitions"`
	// Environment contains a list of environment variables to export prior to executing the job.
	Environment JobEnvVars `json:"environment" db:"job_environment"`
}

func (m *Job) GetKind() ResourceKind {
	return JobResourceKind
}

func (m *Job) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Job) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Job) GetParentID() ResourceID {
	return m.BuildID.ResourceID
}

func (m *Job) GetName() ResourceName {
	return m.Name
}

func (m *Job) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Job) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Job) GetETag() ETag {
	return m.ETag
}

func (m *Job) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Job) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *Job) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *Job) IsUnreachable() bool {
	// Jobs are unreachable after they are soft-deleted
	return m.DeletedAt != nil
}

// GetFQN returns a fully-qualified name for this job, which includes workflow and job name.
func (m *Job) GetFQN() NodeFQN {
	return NewNodeFQNForJob(m.Workflow, m.Name)
}

// GetFQNDependencies returns a list of the fully-qualified names of jobs that must execute before this job.
func (m *Job) GetFQNDependencies() []NodeFQN {
	var depends []NodeFQN
	for _, dependency := range m.Depends {
		depends = append(depends, dependency.GetFQN())
	}
	return depends
}

// Validate the job including the step relationships/dependencies.
func (m *Job) Validate() error {
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
	if m.DeletedAt != nil && m.DeletedAt.IsZero() {
		result = multierror.Append(result, errors.New("error deleted at must be non-zero when set"))
	}
	if !m.BuildID.Valid() {
		result = multierror.Append(result, errors.New("error build id must be set"))
	}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if !m.CommitID.Valid() {
		result = multierror.Append(result, errors.New("error commit id must be set"))
	}
	if m.Ref == "" {
		result = multierror.Append(result, errors.New("error ref must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.Workflow != "" { // it's valid for workflow to be an empty string
		if err := m.Workflow.Validate(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	if !m.Type.Valid() {
		result = multierror.Append(result, errors.New("error builder type is invalid"))
	} else if m.Type == JobTypeDocker {
		if m.DockerImage == "" {
			result = multierror.Append(result, errors.New("error docker image must be set"))
		}
		if !m.DockerImagePullStrategy.Valid() {
			result = multierror.Append(result, errors.New("error docker image pull strategy must be set"))
		}
	}
	if !m.Status.Valid() {
		result = multierror.Append(result, errors.New("error status is invalid"))
	}
	if m.Status == WorkflowStatusSubmitted && !m.RunnerID.Valid() {
		result = multierror.Append(result, errors.New("error runner id must be set when job is submitted"))
	}
	for _, label := range m.RunsOn {
		err := label.Validate()
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("error validating label %q: %w", label, err))
		}
	}
	dependenciesByName := make(map[ResourceName]*JobDependency, len(m.Depends))
	for i, dependency := range m.Depends {
		err := dependency.Validate()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "error validating depedency %q (index %d)", dependency.JobName, i))
		}
		_, ok := dependenciesByName[dependency.JobName]
		if ok {
			return errors.Errorf("Found duplicate dependency %q; Dependencies must have unique resource_links", dependency.JobName)
		}
		dependenciesByName[dependency.JobName] = dependency
	}
	servicesByName := make(map[string]*Service, len(m.Services))
	for i, service := range m.Services {
		err := service.Validate()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "error validating service %q (index %d)", service.Name, i))
		}
		_, ok := servicesByName[service.Name]
		if ok {
			return errors.Errorf("Found duplicate service %q; Services must have unique resource_links", service.Name)
		}
		servicesByName[service.Name] = service
	}
	artifactsByName := make(map[ResourceName]*ArtifactDefinition, len(m.ArtifactDefinitions))
	for i, artifact := range m.ArtifactDefinitions {
		err := artifact.Validate()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "error validating artifact %q (index %d)", artifact.GroupName, i))
		}
		_, ok := artifactsByName[artifact.GroupName]
		if ok {
			return errors.Errorf("error duplicate artifact definition %q; Artifacts must have unique names", artifact.GroupName)
		}
		artifactsByName[artifact.GroupName] = artifact
	}
	for i, env := range m.Environment {
		err := env.Validate()
		if err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "error validating environment variable %q (index %d)", env.Name, i))
		}
	}
	return result.ErrorOrNil()
}

// PopulateDefaults sets default values for all fields of all structs
// in the job that haven't been populated.
func (m *Job) PopulateDefaults(build *Build) {
	if !m.ID.Valid() {
		m.ID = NewJobID()
	}
	m.BuildID = build.ID
	if m.CreatedAt.IsZero() {
		m.CreatedAt = build.CreatedAt
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = build.CreatedAt
	}
	if m.Status == "" || m.Status == WorkflowStatusUnknown {
		m.Status = WorkflowStatusQueued
	}
	// Error value of nil should be used rather than an empty error struct
}
