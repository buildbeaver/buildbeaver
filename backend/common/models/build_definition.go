package models

type BuildDefinition struct {
	// Jobs is the set of jobs within the build.
	Jobs []JobDefinition
}

type JobDefinition struct {
	JobDefinitionData
	// Steps is the set of steps within the job.
	Steps []StepDefinition `json:"steps"`
}

type StepDefinition struct {
	StepDefinitionData
}
