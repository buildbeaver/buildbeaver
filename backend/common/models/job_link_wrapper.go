package models

import "fmt"

// JobLinkWrapper is a wrapper around a job that returns the job's name including workflow,
// for the creation of resource links where we don't have a parent resource for the workflow.
type JobLinkWrapper struct {
	job      *Job
	linkName ResourceName
}

func NewJobLinkWrapper(job *Job) *JobLinkWrapper {
	// Create a valid resource name that includes the workflow and the job name, for use in links
	// Resource names can't include a dot character so the fully-qualified name for the job can't be used;
	// instead separate the workflow name and the job name with a dash.
	var linkName ResourceName
	if job.Workflow != "" {
		linkName = ResourceName(fmt.Sprintf("%s-%s", job.Workflow, job.Name))
	} else {
		linkName = job.Name
	}

	return &JobLinkWrapper{
		job:      job,
		linkName: linkName,
	}
}

func (m *JobLinkWrapper) GetName() ResourceName {
	// Return a ResourceName for use in namespace links that includes both the workflow and the job name
	return m.linkName
}

func (m *JobLinkWrapper) GetParentID() ResourceID {
	// Parent ID for resource links is the build (not the workflow, which doesn't have an ID)
	return m.job.GetParentID()
}

func (m *JobLinkWrapper) GetKind() ResourceKind {
	return m.job.GetKind()
}

func (m *JobLinkWrapper) GetCreatedAt() Time {
	return m.job.GetCreatedAt()
}

func (m *JobLinkWrapper) GetID() ResourceID {
	return m.job.GetID()
}

func (m *JobLinkWrapper) Validate() error {
	return m.job.Validate()
}
