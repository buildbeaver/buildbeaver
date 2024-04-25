package bb

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

const JobResourceKind = "job"

type JobID struct {
	ResourceID
}

func ParseJobID(str string) (JobID, error) {
	id, err := ParseResourceID(str)
	if err != nil {
		return JobID{}, err
	}
	if id.Kind() != JobResourceKind {
		return JobID{}, fmt.Errorf("error: Job ID expected to have kind '%s', found '%s'", JobResourceKind, id.Kind())
	}
	return JobID{ResourceID: id}, nil
}

// JobReference refers to a job within a specific workflow. An empty string for the workflow means the default workflow.
type JobReference struct {
	Workflow ResourceName
	JobName  ResourceName
}

func NewJobReference(workflow ResourceName, jobName ResourceName) JobReference {
	return JobReference{Workflow: workflow, JobName: jobName}
}

func StringToJobReference(ref string) JobReference {
	bits := strings.SplitN(ref, ".", 2)
	if len(bits) == 2 {
		// Two strings, workflow and job name
		return NewJobReference(ResourceName(bits[0]), ResourceName(bits[1]))
	}
	// Only one string which is assumed to be a job name
	return NewJobReference(ResourceName(""), ResourceName(ref))
}

func (j JobReference) String() string {
	if j.Workflow != "" {
		return j.Workflow.String() + "." + j.JobName.String()
	} else {
		return j.JobName.String()
	}
}

func (j JobReference) Equals(ref JobReference) bool {
	return j.Workflow.String() == ref.Workflow.String() && j.JobName.String() == ref.JobName.String()
}

func stringsToJobReferences(refs []string) []JobReference {
	res := make([]JobReference, len(refs))
	for i, ref := range refs {
		res[i] = StringToJobReference(ref)
	}
	return res
}

type Job struct {
	definition client.JobDefinition
	// Workflow this job was added to, if any
	workflow *Workflow
	// completionCallbacksToRegister is a list of callback functions to register once this job is part of a build
	completionCallbacksToRegister []JobCallback
	// successCallbacksToRegister is a list of callback functions to register once this job is part of a build
	successCallbacksToRegister []JobCallback
	// failureCallbacksToRegister is a list of callback functions to register once this job is part of a build
	failureCallbacksToRegister []JobCallback
	// cancelledCallbacksToRegister is a list of callback functions to register once this job is part of a build
	cancelledCallbacksToRegister []JobCallback
	// statusChangedCallbacksToRegister is a list of callback functions to register once this job is part of a build
	statusChangedCallbacksToRegister []JobCallback
}

type StepExecutionType string

func (t StepExecutionType) String() string {
	return string(t)
}

const (
	StepExecutionSequential StepExecutionType = "sequential"
	StepExecutionParallel   StepExecutionType = "parallel"
)

type JobType string

func (t JobType) String() string {
	return string(t)
}

func (t JobType) StringPtr() *string {
	str := string(t)
	return &str
}

const (
	JobTypeDocker JobType = "docker"
	JobTypeExec   JobType = "exec"
)

type DockerPullStrategy string

func (ps DockerPullStrategy) String() string {
	return string(ps)
}

const (
	DockerPullDefault     DockerPullStrategy = "default"
	DockerPullNever       DockerPullStrategy = "never"
	DockerPullAlways      DockerPullStrategy = "always"
	DockerPullIfNotExists DockerPullStrategy = "if-not-exists"
)

func NewJob() *Job {
	job := &Job{
		definition: client.JobDefinition{
			Environment: make(map[string]client.SecretStringDefinition),
		},
	}
	return job
}

func (job *Job) GetData() client.JobDefinition {
	return job.definition
}

// GetName returns the name of the job (not including any workflow name).
func (job *Job) GetName() ResourceName {
	return ResourceName(job.definition.Name)
}

// GetReference returns a reference to the job, unique within the build. This will include
// any workflow the job is a part of.
func (job *Job) GetReference() JobReference {
	workflow := ResourceName("")
	if job.definition.Workflow != nil && *job.definition.Workflow != "" {
		workflow = ResourceName(*job.definition.Workflow)
	}
	// TODO: Do we need to parse the name in case a workflow was set? Currently we assume workflow will be set
	// TODO: in a separate field
	return NewJobReference(
		workflow,
		ResourceName(job.definition.Name),
	)
}

func (job *Job) Name(name ResourceName) *Job {
	job.definition.Name = name.String()
	return job
}

func (job *Job) Desc(description string) *Job {
	job.definition.Description = &description
	return job
}

func (job *Job) Type(jobType JobType) *Job {
	jobTypeStr := jobType.String()
	job.definition.Type = &jobTypeStr
	return job
}

func (job *Job) RunsOn(labels ...string) *Job {
	job.definition.RunsOn = append(job.definition.RunsOn, labels...)
	return job
}

func (job *Job) Docker(dockerConfig *DockerConfig) *Job {
	dockerConfigDefinition := dockerConfig.GetData()

	job.definition.Type = JobTypeDocker.StringPtr()
	job.definition.Docker = &dockerConfigDefinition
	return job
}

func (job *Job) StepExecution(executionType StepExecutionType) *Job {
	job.definition.StepExecution = executionType.String()
	return job
}

func (job *Job) Depends(dependencies ...string) *Job {
	job.definition.Depends = append(job.definition.Depends, dependencies...)
	return job
}

func (job *Job) DependsOnJobs(jobs ...*Job) *Job {
	for _, dependsOnJob := range jobs {
		job.definition.Depends = append(job.definition.Depends, dependsOnJob.GetReference().String())
	}
	return job
}

func (job *Job) DependsOnJobArtifacts(jobs ...*Job) *Job {
	for _, dependsOnJob := range jobs {
		dependency := fmt.Sprintf("%s.artifacts", dependsOnJob.GetReference())
		job.definition.Depends = append(job.definition.Depends, dependency)
	}
	return job
}

func (job *Job) Env(env *Env) *Job {
	def := client.SecretStringDefinition{Value: &env.value}
	if env.secretName != "" {
		def = client.SecretStringDefinition{FromSecret: &env.secretName}
	}
	job.definition.Environment[env.name] = def
	Log(LogLevelInfo, fmt.Sprintf("Env var with name '%s' added for job '%s'", env.name, job.GetReference()))
	return job
}

func (job *Job) Fingerprint(commands ...string) *Job {
	job.definition.Fingerprint = append(job.definition.Fingerprint, commands...)
	return job
}

func (job *Job) Step(step *Step) *Job {
	job.definition.Steps = append(job.definition.Steps, step.GetData())
	Log(LogLevelInfo, fmt.Sprintf("Step with name '%s' added to job '%s'", step.definition.Name, job.GetReference()))
	return job
}

func (job *Job) Service(service *Service) *Job {
	job.definition.Services = append(job.definition.Services, service.GetData())
	Log(LogLevelInfo, fmt.Sprintf("Service with name '%s' added", service.GetName()))
	return job
}

func (job *Job) Artifact(artifact *Artifact) *Job {
	job.definition.Artifacts = append(job.definition.Artifacts, artifact.GetData())
	Log(LogLevelInfo, fmt.Sprintf("Artifact with name '%s' added for job '%s'", artifact.GetName(), job.GetReference()))
	return job
}

func (job *Job) OnCompletion(fn JobCallback) *Job {
	if job.workflow != nil {
		job.workflow.OnJobCompletion(job.GetReference(), fn)
	} else {
		job.completionCallbacksToRegister = append(job.completionCallbacksToRegister, fn)
	}
	return job
}

func (job *Job) OnSuccess(fn JobCallback) *Job {
	if job.workflow != nil {
		job.workflow.OnJobSuccess(job.GetReference(), fn)
	} else {
		job.successCallbacksToRegister = append(job.successCallbacksToRegister, fn)
	}
	return job
}

func (job *Job) OnFailure(fn JobCallback) *Job {
	if job.workflow != nil {
		job.workflow.OnJobFailure(job.GetReference(), fn)
	} else {
		job.failureCallbacksToRegister = append(job.failureCallbacksToRegister, fn)
	}
	return job
}

func (job *Job) OnCancelled(fn JobCallback) *Job {
	if job.workflow != nil {
		job.workflow.OnJobCancelled(job.GetReference(), fn)
	} else {
		job.cancelledCallbacksToRegister = append(job.cancelledCallbacksToRegister, fn)
	}
	return job
}

// OnStatusChanged will call a callback function each time the status of the job changes.
func (job *Job) OnStatusChanged(fn JobCallback) *Job {
	if job.workflow != nil {
		job.workflow.OnJobStatusChanged(job.GetReference(), fn)
	} else {
		job.statusChangedCallbacksToRegister = append(job.statusChangedCallbacksToRegister, fn)
	}
	return job
}

// getWorkflowDependencies returns a list of names of workflows that this job depends on,
// based on the dependencies declared.
func (job *Job) getWorkflowDependencies() []ResourceName {
	var workflows []ResourceName
	for _, dependencyStr := range job.definition.Depends {
		workflow, _ := workflowDependencyFromString(dependencyStr)
		if workflow != "" {
			workflows = append(workflows, workflow)
		}
	}
	return workflows
}

// TODO: Find a way to not need to include this parsing detail in every dynamic SDK. Simplify it?
var (
	jobDependsOnOneArtifactFromJobRegex03           = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)\.artifacts\.([a-zA-Z0-9_-]+)$`)
	jobDependsOnAllArtifactsFromJobRegex03          = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnJobRegex03                          = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)$`)
	jobDependsOnAllArtifactsFromJobShorthandRegex03 = regexp.MustCompile(`(?im)^(?:([a-zA-Z0-9_-]+)\.)?([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnJobShorthandRegex03                 = regexp.MustCompile(`(?im)^(?:([a-zA-Z0-9_-]+)\.)?([a-zA-Z0-9_*-]+)$`)
)

// workflowDependencyFromString returns the names of the workflow and job in the given dependency string.
// An empty workflow name will be returned if no workflow was mentioned in the dependency.
func workflowDependencyFromString(dependency string) (workflow ResourceName, jobName ResourceName) {
	if match := jobDependsOnOneArtifactFromJobRegex03.FindStringSubmatch(dependency); match != nil {
		return ResourceName(match[1]), ResourceName(match[2])
	}
	if match := jobDependsOnAllArtifactsFromJobRegex03.FindStringSubmatch(dependency); match != nil {
		return ResourceName(match[1]), ResourceName(match[2])
	}
	if match := jobDependsOnAllArtifactsFromJobShorthandRegex03.FindStringSubmatch(dependency); match != nil {
		return ResourceName(match[1]), ResourceName(match[2])
	}
	if match := jobDependsOnJobRegex03.FindStringSubmatch(dependency); match != nil {
		return ResourceName(match[1]), ResourceName(match[2])
	}
	if match := jobDependsOnJobShorthandRegex03.FindStringSubmatch(dependency); match != nil {
		return ResourceName(match[1]), ResourceName(match[2])
	}
	return "", "" // no match found, so no workflow or job name to return
}

// validateJobDependency checks that the given job dependency is valid.
// Job dependencies must explicitly specify a workflow as well as a job name.
// Job dependencies on the default workflow (with empty string as name) are not allowed.
func validateJobDependency(dependency string) error {
	workflow, jobName := workflowDependencyFromString(dependency)
	if workflow == "" {
		return fmt.Errorf("workflow is not specified in dependency on job '%s'", jobName)
	}
	return nil
}

// validateJobDependencies checks that each of the given job dependencies are valid.
// Job dependencies must explicitly specify a workflow as well as a job name.
// Job dependencies on the default workflow (with empty string as name) are not allowed.
func validateJobDependencies(dependencies []string) error {
	for _, dependency := range dependencies {
		err := validateJobDependency(dependency)
		if err != nil {
			return err
		}
	}
	return nil
}
