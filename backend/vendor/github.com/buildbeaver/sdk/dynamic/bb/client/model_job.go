/*
BuildBeaver Dynamic Build API - OpenAPI 3.0

This is the BuildBeaver Dynamic Build API.

API version: 0.3.00
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
	"time"
)

// checks if the Job type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &Job{}

// Job struct for Job
type Job struct {
	// A link to the Job resource on the BuildBeaver server
	Url string `json:"url"`
	Id string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Etag string `json:"etag"`
	// Job name, in URL format
	Name string `json:"name"`
	// Workflow the job is a part of, or empty if the job is part of the default workflow.
	Workflow string `json:"workflow"`
	// Optional human-readable description of the job.
	Description string `json:"description"`
	// Type of the job (e.g. docker, exec etc.)
	Type string `json:"type"`
	// RunsOn contains a set of labels that this job requires runners to have.
	RunsOn []string `json:"runs_on"`
	Docker *DockerConfig `json:"docker,omitempty"`
	// Determines how the runner will execute steps within this job
	StepExecution string `json:"step_execution"`
	// Dependencies on other jobs and their artifacts. Each JobDependency declares that this job depends on the successful execution of another, and optionally that this job consumes one or more artifacts from the other.
	Depends []JobDependency `json:"depends"`
	// Services to run in the background for the duration of the job; services are started before the first step is run, and stopped after the last step completes
	Services []Service `json:"services"`
	// Shell commands to execute to generate a unique fingerprint for the jobs; two jobs in the same repo with the same name and fingerprint are considered identical
	FingerprintCommands []string `json:"fingerprint_commands"`
	// A list of all artifacts the job is expected to produce that will be saved to the artifact store at the end of the job's execution
	Artifacts []ArtifactDefinition `json:"artifacts,omitempty"`
	// A list of environment variables to export prior to executing the job
	Environment []EnvVar `json:"environment"`
	// ID of the build this job forms a part of.
	BuildId string `json:"build_id"`
	// RepoID that was committed to.
	RepoId string `json:"repo_id"`
	// CommitID that the job was generated from.
	CommitId string `json:"commit_id"`
	// LogDescriptorID points to the log for this job.
	LogDescriptorId string `json:"log_descriptor_id"`
	// RunnerID is the id of the runner this job executed on, or empty if the job has not run yet (or did/will not run).
	RunnerId string `json:"runner_id"`
	// IndirectToJobID records the ID of a job that previously ran successfully as part of another build and which is functionally identical to this job. If this is set it means this job did not actually run to avoid redundantly running the same thing more than once.
	IndirectToJobId string `json:"indirect_to_job_id"`
	// Ref is the git ref from the build that the job was generated from (e.g. branch or tag)
	Ref string `json:"ref"`
	// Status reflects where the job is in the queue.
	Status string `json:"status"`
	// Error is set if the job finished with an error (or empty if the job succeeded).
	Error *string `json:"error,omitempty"`
	Timings WorkflowTimings `json:"timings"`
	// Fingerprint contains the hashed output of FingerprintCommands, as well as any other inputs the agent added (such as artifact hashes). This is only available after the job has run successfully.
	Fingerprint *string `json:"fingerprint,omitempty"`
	// FingerprintHashType is the type of hashing algorithm used to produce the fingerprint.
	FingerprintHashType *string `json:"fingerprint_hash_type,omitempty"`
	// URL of the log for this job.
	LogDescriptorUrl string `json:"log_descriptor_url"`
	// URL to the job that this job indirects to, if any.
	IndirectJobUrl string `json:"indirect_job_url"`
	AdditionalProperties map[string]interface{}
}

type _Job Job

// NewJob instantiates a new Job object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewJob(url string, id string, createdAt time.Time, updatedAt time.Time, etag string, name string, workflow string, description string, type_ string, runsOn []string, stepExecution string, depends []JobDependency, services []Service, fingerprintCommands []string, environment []EnvVar, buildId string, repoId string, commitId string, logDescriptorId string, runnerId string, indirectToJobId string, ref string, status string, timings WorkflowTimings, logDescriptorUrl string, indirectJobUrl string) *Job {
	this := Job{}
	this.Url = url
	this.Id = id
	this.CreatedAt = createdAt
	this.UpdatedAt = updatedAt
	this.Etag = etag
	this.Name = name
	this.Workflow = workflow
	this.Description = description
	this.Type = type_
	this.RunsOn = runsOn
	this.StepExecution = stepExecution
	this.Depends = depends
	this.Services = services
	this.FingerprintCommands = fingerprintCommands
	this.Environment = environment
	this.BuildId = buildId
	this.RepoId = repoId
	this.CommitId = commitId
	this.LogDescriptorId = logDescriptorId
	this.RunnerId = runnerId
	this.IndirectToJobId = indirectToJobId
	this.Ref = ref
	this.Status = status
	this.Timings = timings
	this.LogDescriptorUrl = logDescriptorUrl
	this.IndirectJobUrl = indirectJobUrl
	return &this
}

// NewJobWithDefaults instantiates a new Job object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewJobWithDefaults() *Job {
	this := Job{}
	return &this
}

// GetUrl returns the Url field value
func (o *Job) GetUrl() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Url
}

// GetUrlOk returns a tuple with the Url field value
// and a boolean to check if the value has been set.
func (o *Job) GetUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Url, true
}

// SetUrl sets field value
func (o *Job) SetUrl(v string) {
	o.Url = v
}

// GetId returns the Id field value
func (o *Job) GetId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *Job) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value
func (o *Job) SetId(v string) {
	o.Id = v
}

// GetCreatedAt returns the CreatedAt field value
func (o *Job) GetCreatedAt() time.Time {
	if o == nil {
		var ret time.Time
		return ret
	}

	return o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value
// and a boolean to check if the value has been set.
func (o *Job) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return &o.CreatedAt, true
}

// SetCreatedAt sets field value
func (o *Job) SetCreatedAt(v time.Time) {
	o.CreatedAt = v
}

// GetUpdatedAt returns the UpdatedAt field value
func (o *Job) GetUpdatedAt() time.Time {
	if o == nil {
		var ret time.Time
		return ret
	}

	return o.UpdatedAt
}

// GetUpdatedAtOk returns a tuple with the UpdatedAt field value
// and a boolean to check if the value has been set.
func (o *Job) GetUpdatedAtOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return &o.UpdatedAt, true
}

// SetUpdatedAt sets field value
func (o *Job) SetUpdatedAt(v time.Time) {
	o.UpdatedAt = v
}

// GetDeletedAt returns the DeletedAt field value if set, zero value otherwise.
func (o *Job) GetDeletedAt() time.Time {
	if o == nil || IsNil(o.DeletedAt) {
		var ret time.Time
		return ret
	}
	return *o.DeletedAt
}

// GetDeletedAtOk returns a tuple with the DeletedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetDeletedAtOk() (*time.Time, bool) {
	if o == nil || IsNil(o.DeletedAt) {
		return nil, false
	}
	return o.DeletedAt, true
}

// HasDeletedAt returns a boolean if a field has been set.
func (o *Job) HasDeletedAt() bool {
	if o != nil && !IsNil(o.DeletedAt) {
		return true
	}

	return false
}

// SetDeletedAt gets a reference to the given time.Time and assigns it to the DeletedAt field.
func (o *Job) SetDeletedAt(v time.Time) {
	o.DeletedAt = &v
}

// GetEtag returns the Etag field value
func (o *Job) GetEtag() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Etag
}

// GetEtagOk returns a tuple with the Etag field value
// and a boolean to check if the value has been set.
func (o *Job) GetEtagOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Etag, true
}

// SetEtag sets field value
func (o *Job) SetEtag(v string) {
	o.Etag = v
}

// GetName returns the Name field value
func (o *Job) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *Job) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *Job) SetName(v string) {
	o.Name = v
}

// GetWorkflow returns the Workflow field value
func (o *Job) GetWorkflow() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Workflow
}

// GetWorkflowOk returns a tuple with the Workflow field value
// and a boolean to check if the value has been set.
func (o *Job) GetWorkflowOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Workflow, true
}

// SetWorkflow sets field value
func (o *Job) SetWorkflow(v string) {
	o.Workflow = v
}

// GetDescription returns the Description field value
func (o *Job) GetDescription() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Description
}

// GetDescriptionOk returns a tuple with the Description field value
// and a boolean to check if the value has been set.
func (o *Job) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Description, true
}

// SetDescription sets field value
func (o *Job) SetDescription(v string) {
	o.Description = v
}

// GetType returns the Type field value
func (o *Job) GetType() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *Job) GetTypeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value
func (o *Job) SetType(v string) {
	o.Type = v
}

// GetRunsOn returns the RunsOn field value
func (o *Job) GetRunsOn() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.RunsOn
}

// GetRunsOnOk returns a tuple with the RunsOn field value
// and a boolean to check if the value has been set.
func (o *Job) GetRunsOnOk() ([]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.RunsOn, true
}

// SetRunsOn sets field value
func (o *Job) SetRunsOn(v []string) {
	o.RunsOn = v
}

// GetDocker returns the Docker field value if set, zero value otherwise.
func (o *Job) GetDocker() DockerConfig {
	if o == nil || IsNil(o.Docker) {
		var ret DockerConfig
		return ret
	}
	return *o.Docker
}

// GetDockerOk returns a tuple with the Docker field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetDockerOk() (*DockerConfig, bool) {
	if o == nil || IsNil(o.Docker) {
		return nil, false
	}
	return o.Docker, true
}

// HasDocker returns a boolean if a field has been set.
func (o *Job) HasDocker() bool {
	if o != nil && !IsNil(o.Docker) {
		return true
	}

	return false
}

// SetDocker gets a reference to the given DockerConfig and assigns it to the Docker field.
func (o *Job) SetDocker(v DockerConfig) {
	o.Docker = &v
}

// GetStepExecution returns the StepExecution field value
func (o *Job) GetStepExecution() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.StepExecution
}

// GetStepExecutionOk returns a tuple with the StepExecution field value
// and a boolean to check if the value has been set.
func (o *Job) GetStepExecutionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.StepExecution, true
}

// SetStepExecution sets field value
func (o *Job) SetStepExecution(v string) {
	o.StepExecution = v
}

// GetDepends returns the Depends field value
func (o *Job) GetDepends() []JobDependency {
	if o == nil {
		var ret []JobDependency
		return ret
	}

	return o.Depends
}

// GetDependsOk returns a tuple with the Depends field value
// and a boolean to check if the value has been set.
func (o *Job) GetDependsOk() ([]JobDependency, bool) {
	if o == nil {
		return nil, false
	}
	return o.Depends, true
}

// SetDepends sets field value
func (o *Job) SetDepends(v []JobDependency) {
	o.Depends = v
}

// GetServices returns the Services field value
func (o *Job) GetServices() []Service {
	if o == nil {
		var ret []Service
		return ret
	}

	return o.Services
}

// GetServicesOk returns a tuple with the Services field value
// and a boolean to check if the value has been set.
func (o *Job) GetServicesOk() ([]Service, bool) {
	if o == nil {
		return nil, false
	}
	return o.Services, true
}

// SetServices sets field value
func (o *Job) SetServices(v []Service) {
	o.Services = v
}

// GetFingerprintCommands returns the FingerprintCommands field value
func (o *Job) GetFingerprintCommands() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.FingerprintCommands
}

// GetFingerprintCommandsOk returns a tuple with the FingerprintCommands field value
// and a boolean to check if the value has been set.
func (o *Job) GetFingerprintCommandsOk() ([]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.FingerprintCommands, true
}

// SetFingerprintCommands sets field value
func (o *Job) SetFingerprintCommands(v []string) {
	o.FingerprintCommands = v
}

// GetArtifacts returns the Artifacts field value if set, zero value otherwise.
func (o *Job) GetArtifacts() []ArtifactDefinition {
	if o == nil || IsNil(o.Artifacts) {
		var ret []ArtifactDefinition
		return ret
	}
	return o.Artifacts
}

// GetArtifactsOk returns a tuple with the Artifacts field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetArtifactsOk() ([]ArtifactDefinition, bool) {
	if o == nil || IsNil(o.Artifacts) {
		return nil, false
	}
	return o.Artifacts, true
}

// HasArtifacts returns a boolean if a field has been set.
func (o *Job) HasArtifacts() bool {
	if o != nil && !IsNil(o.Artifacts) {
		return true
	}

	return false
}

// SetArtifacts gets a reference to the given []ArtifactDefinition and assigns it to the Artifacts field.
func (o *Job) SetArtifacts(v []ArtifactDefinition) {
	o.Artifacts = v
}

// GetEnvironment returns the Environment field value
func (o *Job) GetEnvironment() []EnvVar {
	if o == nil {
		var ret []EnvVar
		return ret
	}

	return o.Environment
}

// GetEnvironmentOk returns a tuple with the Environment field value
// and a boolean to check if the value has been set.
func (o *Job) GetEnvironmentOk() ([]EnvVar, bool) {
	if o == nil {
		return nil, false
	}
	return o.Environment, true
}

// SetEnvironment sets field value
func (o *Job) SetEnvironment(v []EnvVar) {
	o.Environment = v
}

// GetBuildId returns the BuildId field value
func (o *Job) GetBuildId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.BuildId
}

// GetBuildIdOk returns a tuple with the BuildId field value
// and a boolean to check if the value has been set.
func (o *Job) GetBuildIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.BuildId, true
}

// SetBuildId sets field value
func (o *Job) SetBuildId(v string) {
	o.BuildId = v
}

// GetRepoId returns the RepoId field value
func (o *Job) GetRepoId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.RepoId
}

// GetRepoIdOk returns a tuple with the RepoId field value
// and a boolean to check if the value has been set.
func (o *Job) GetRepoIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RepoId, true
}

// SetRepoId sets field value
func (o *Job) SetRepoId(v string) {
	o.RepoId = v
}

// GetCommitId returns the CommitId field value
func (o *Job) GetCommitId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.CommitId
}

// GetCommitIdOk returns a tuple with the CommitId field value
// and a boolean to check if the value has been set.
func (o *Job) GetCommitIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.CommitId, true
}

// SetCommitId sets field value
func (o *Job) SetCommitId(v string) {
	o.CommitId = v
}

// GetLogDescriptorId returns the LogDescriptorId field value
func (o *Job) GetLogDescriptorId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.LogDescriptorId
}

// GetLogDescriptorIdOk returns a tuple with the LogDescriptorId field value
// and a boolean to check if the value has been set.
func (o *Job) GetLogDescriptorIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LogDescriptorId, true
}

// SetLogDescriptorId sets field value
func (o *Job) SetLogDescriptorId(v string) {
	o.LogDescriptorId = v
}

// GetRunnerId returns the RunnerId field value
func (o *Job) GetRunnerId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.RunnerId
}

// GetRunnerIdOk returns a tuple with the RunnerId field value
// and a boolean to check if the value has been set.
func (o *Job) GetRunnerIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RunnerId, true
}

// SetRunnerId sets field value
func (o *Job) SetRunnerId(v string) {
	o.RunnerId = v
}

// GetIndirectToJobId returns the IndirectToJobId field value
func (o *Job) GetIndirectToJobId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.IndirectToJobId
}

// GetIndirectToJobIdOk returns a tuple with the IndirectToJobId field value
// and a boolean to check if the value has been set.
func (o *Job) GetIndirectToJobIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IndirectToJobId, true
}

// SetIndirectToJobId sets field value
func (o *Job) SetIndirectToJobId(v string) {
	o.IndirectToJobId = v
}

// GetRef returns the Ref field value
func (o *Job) GetRef() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Ref
}

// GetRefOk returns a tuple with the Ref field value
// and a boolean to check if the value has been set.
func (o *Job) GetRefOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Ref, true
}

// SetRef sets field value
func (o *Job) SetRef(v string) {
	o.Ref = v
}

// GetStatus returns the Status field value
func (o *Job) GetStatus() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Status
}

// GetStatusOk returns a tuple with the Status field value
// and a boolean to check if the value has been set.
func (o *Job) GetStatusOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Status, true
}

// SetStatus sets field value
func (o *Job) SetStatus(v string) {
	o.Status = v
}

// GetError returns the Error field value if set, zero value otherwise.
func (o *Job) GetError() string {
	if o == nil || IsNil(o.Error) {
		var ret string
		return ret
	}
	return *o.Error
}

// GetErrorOk returns a tuple with the Error field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetErrorOk() (*string, bool) {
	if o == nil || IsNil(o.Error) {
		return nil, false
	}
	return o.Error, true
}

// HasError returns a boolean if a field has been set.
func (o *Job) HasError() bool {
	if o != nil && !IsNil(o.Error) {
		return true
	}

	return false
}

// SetError gets a reference to the given string and assigns it to the Error field.
func (o *Job) SetError(v string) {
	o.Error = &v
}

// GetTimings returns the Timings field value
func (o *Job) GetTimings() WorkflowTimings {
	if o == nil {
		var ret WorkflowTimings
		return ret
	}

	return o.Timings
}

// GetTimingsOk returns a tuple with the Timings field value
// and a boolean to check if the value has been set.
func (o *Job) GetTimingsOk() (*WorkflowTimings, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Timings, true
}

// SetTimings sets field value
func (o *Job) SetTimings(v WorkflowTimings) {
	o.Timings = v
}

// GetFingerprint returns the Fingerprint field value if set, zero value otherwise.
func (o *Job) GetFingerprint() string {
	if o == nil || IsNil(o.Fingerprint) {
		var ret string
		return ret
	}
	return *o.Fingerprint
}

// GetFingerprintOk returns a tuple with the Fingerprint field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetFingerprintOk() (*string, bool) {
	if o == nil || IsNil(o.Fingerprint) {
		return nil, false
	}
	return o.Fingerprint, true
}

// HasFingerprint returns a boolean if a field has been set.
func (o *Job) HasFingerprint() bool {
	if o != nil && !IsNil(o.Fingerprint) {
		return true
	}

	return false
}

// SetFingerprint gets a reference to the given string and assigns it to the Fingerprint field.
func (o *Job) SetFingerprint(v string) {
	o.Fingerprint = &v
}

// GetFingerprintHashType returns the FingerprintHashType field value if set, zero value otherwise.
func (o *Job) GetFingerprintHashType() string {
	if o == nil || IsNil(o.FingerprintHashType) {
		var ret string
		return ret
	}
	return *o.FingerprintHashType
}

// GetFingerprintHashTypeOk returns a tuple with the FingerprintHashType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *Job) GetFingerprintHashTypeOk() (*string, bool) {
	if o == nil || IsNil(o.FingerprintHashType) {
		return nil, false
	}
	return o.FingerprintHashType, true
}

// HasFingerprintHashType returns a boolean if a field has been set.
func (o *Job) HasFingerprintHashType() bool {
	if o != nil && !IsNil(o.FingerprintHashType) {
		return true
	}

	return false
}

// SetFingerprintHashType gets a reference to the given string and assigns it to the FingerprintHashType field.
func (o *Job) SetFingerprintHashType(v string) {
	o.FingerprintHashType = &v
}

// GetLogDescriptorUrl returns the LogDescriptorUrl field value
func (o *Job) GetLogDescriptorUrl() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.LogDescriptorUrl
}

// GetLogDescriptorUrlOk returns a tuple with the LogDescriptorUrl field value
// and a boolean to check if the value has been set.
func (o *Job) GetLogDescriptorUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LogDescriptorUrl, true
}

// SetLogDescriptorUrl sets field value
func (o *Job) SetLogDescriptorUrl(v string) {
	o.LogDescriptorUrl = v
}

// GetIndirectJobUrl returns the IndirectJobUrl field value
func (o *Job) GetIndirectJobUrl() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.IndirectJobUrl
}

// GetIndirectJobUrlOk returns a tuple with the IndirectJobUrl field value
// and a boolean to check if the value has been set.
func (o *Job) GetIndirectJobUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IndirectJobUrl, true
}

// SetIndirectJobUrl sets field value
func (o *Job) SetIndirectJobUrl(v string) {
	o.IndirectJobUrl = v
}

func (o Job) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o Job) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["url"] = o.Url
	toSerialize["id"] = o.Id
	toSerialize["created_at"] = o.CreatedAt
	toSerialize["updated_at"] = o.UpdatedAt
	if !IsNil(o.DeletedAt) {
		toSerialize["deleted_at"] = o.DeletedAt
	}
	toSerialize["etag"] = o.Etag
	toSerialize["name"] = o.Name
	toSerialize["workflow"] = o.Workflow
	toSerialize["description"] = o.Description
	toSerialize["type"] = o.Type
	toSerialize["runs_on"] = o.RunsOn
	if !IsNil(o.Docker) {
		toSerialize["docker"] = o.Docker
	}
	toSerialize["step_execution"] = o.StepExecution
	toSerialize["depends"] = o.Depends
	toSerialize["services"] = o.Services
	toSerialize["fingerprint_commands"] = o.FingerprintCommands
	if !IsNil(o.Artifacts) {
		toSerialize["artifacts"] = o.Artifacts
	}
	toSerialize["environment"] = o.Environment
	toSerialize["build_id"] = o.BuildId
	toSerialize["repo_id"] = o.RepoId
	toSerialize["commit_id"] = o.CommitId
	toSerialize["log_descriptor_id"] = o.LogDescriptorId
	toSerialize["runner_id"] = o.RunnerId
	toSerialize["indirect_to_job_id"] = o.IndirectToJobId
	toSerialize["ref"] = o.Ref
	toSerialize["status"] = o.Status
	if !IsNil(o.Error) {
		toSerialize["error"] = o.Error
	}
	toSerialize["timings"] = o.Timings
	if !IsNil(o.Fingerprint) {
		toSerialize["fingerprint"] = o.Fingerprint
	}
	if !IsNil(o.FingerprintHashType) {
		toSerialize["fingerprint_hash_type"] = o.FingerprintHashType
	}
	toSerialize["log_descriptor_url"] = o.LogDescriptorUrl
	toSerialize["indirect_job_url"] = o.IndirectJobUrl

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}

	return toSerialize, nil
}

func (o *Job) UnmarshalJSON(bytes []byte) (err error) {
	varJob := _Job{}

	if err = json.Unmarshal(bytes, &varJob); err == nil {
		*o = Job(varJob)
	}

	additionalProperties := make(map[string]interface{})

	if err = json.Unmarshal(bytes, &additionalProperties); err == nil {
		delete(additionalProperties, "url")
		delete(additionalProperties, "id")
		delete(additionalProperties, "created_at")
		delete(additionalProperties, "updated_at")
		delete(additionalProperties, "deleted_at")
		delete(additionalProperties, "etag")
		delete(additionalProperties, "name")
		delete(additionalProperties, "workflow")
		delete(additionalProperties, "description")
		delete(additionalProperties, "type")
		delete(additionalProperties, "runs_on")
		delete(additionalProperties, "docker")
		delete(additionalProperties, "step_execution")
		delete(additionalProperties, "depends")
		delete(additionalProperties, "services")
		delete(additionalProperties, "fingerprint_commands")
		delete(additionalProperties, "artifacts")
		delete(additionalProperties, "environment")
		delete(additionalProperties, "build_id")
		delete(additionalProperties, "repo_id")
		delete(additionalProperties, "commit_id")
		delete(additionalProperties, "log_descriptor_id")
		delete(additionalProperties, "runner_id")
		delete(additionalProperties, "indirect_to_job_id")
		delete(additionalProperties, "ref")
		delete(additionalProperties, "status")
		delete(additionalProperties, "error")
		delete(additionalProperties, "timings")
		delete(additionalProperties, "fingerprint")
		delete(additionalProperties, "fingerprint_hash_type")
		delete(additionalProperties, "log_descriptor_url")
		delete(additionalProperties, "indirect_job_url")
		o.AdditionalProperties = additionalProperties
	}

	return err
}

type NullableJob struct {
	value *Job
	isSet bool
}

func (v NullableJob) Get() *Job {
	return v.value
}

func (v *NullableJob) Set(val *Job) {
	v.value = val
	v.isSet = true
}

func (v NullableJob) IsSet() bool {
	return v.isSet
}

func (v *NullableJob) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableJob(val *Job) *NullableJob {
	return &NullableJob{value: val, isSet: true}
}

func (v NullableJob) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableJob) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

