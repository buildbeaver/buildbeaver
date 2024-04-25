package models

// ArtifactDependency declares that one job depends on artifact(s) produced by the successful
// execution of one or more other job's steps. The artifact(s) will be downloaded and made available
// when the dependent job executes.
// If group name is empty then it refers to all artifacts produced by the step.
// If step name is empty then it refers to all artifacts produced by the job.
type ArtifactDependency struct {
	Workflow  ResourceName `json:"workflow"`
	JobName   ResourceName `json:"job_name"`
	GroupName ResourceName `json:"group_name"`
}

func NewArtifactDependency(workflow ResourceName, jobName ResourceName, groupName ResourceName) *ArtifactDependency {
	return &ArtifactDependency{
		Workflow:  workflow,
		JobName:   jobName,
		GroupName: groupName,
	}
}

func (m *ArtifactDependency) Validate() error {
	return nil
}
