package queue_test

import (
	"sort"
	"testing"

	"github.com/buildbeaver/buildbeaver/server/services/queue/parser"
	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto/dto_test/referencedata"
)

func TestPipelineGeneratorJSON(t *testing.T) {
	commit := &models.Commit{
		ID:         models.NewCommitID(),
		RepoID:     models.NewRepoID(),
		Config:     []byte(referencedata.PipelineJSON),
		ConfigType: models.ConfigTypeJSON,
	}
	t.Logf("JSON for build config: '%s'", referencedata.PipelineJSON)
	parser := parser.NewBuildDefinitionParser(parser.ParserLimits{})
	build, err := parser.Parse(commit.Config, commit.ConfigType)
	if err != nil {
		t.Fatalf("Error generating pipeline: %s", err)
	}
	t.Run("JSON", testPipelineAgainstReference(build))
}

func TestPipelineGeneratorJSONNET(t *testing.T) {
	commit := &models.Commit{
		ID:         models.NewCommitID(),
		RepoID:     models.NewRepoID(),
		Config:     []byte(referencedata.PipelineJSONNET),
		ConfigType: models.ConfigTypeJSONNET,
	}
	parser := parser.NewBuildDefinitionParser(parser.ParserLimits{})
	build, err := parser.Parse(commit.Config, commit.ConfigType)
	if err != nil {
		t.Fatalf("Error generating pipeline: %s", err)
	}
	t.Run("JSONNET", testPipelineAgainstReference(build))
}

func TestPipelineGeneratorYAML(t *testing.T) {
	commit := &models.Commit{
		ID:         models.NewCommitID(),
		RepoID:     models.NewRepoID(),
		Config:     []byte(referencedata.PipelineYAML),
		ConfigType: models.ConfigTypeYAML,
	}
	parser := parser.NewBuildDefinitionParser(parser.ParserLimits{})
	build, err := parser.Parse(commit.Config, commit.ConfigType)
	if err != nil {
		t.Fatalf("Error generating pipeline: %s", err)
	}
	t.Run("YAML", testPipelineAgainstReference(build))
}

func TestParseStepExecution(t *testing.T) {
	commit := &models.Commit{
		ID:         models.NewCommitID(),
		RepoID:     models.NewRepoID(),
		Config:     []byte(referencedata.PipelineJSON),
		ConfigType: models.ConfigTypeJSON,
	}
	parser := parser.NewBuildDefinitionParser(parser.ParserLimits{})
	build, err := parser.Parse(commit.Config, commit.ConfigType)
	if err != nil {
		t.Fatalf("Error generating pipeline: %s", err)
	}
	require.Len(t, build.Jobs, 4)

	// Job 4 has 3 steps, and step execution is sequential.
	// We expect the parser to automatically inject dependencies on each preceding step
	require.Equal(t, build.Jobs[3].StepExecution, models.StepExecutionSequential)
	require.Len(t, build.Jobs[3].Steps, 3)
	require.Len(t, build.Jobs[3].Steps[1].Depends, 1)
	require.Equal(t, build.Jobs[3].Steps[1].Depends[0].StepName, build.Jobs[3].Steps[0].Name)
	require.Len(t, build.Jobs[3].Steps[2].Depends, 1)
	require.Equal(t, build.Jobs[3].Steps[2].Depends[0].StepName, build.Jobs[3].Steps[1].Name)
}

func testPipelineAgainstReference(build *models.BuildDefinition) func(t *testing.T) {
	return func(t *testing.T) {
		if len(build.Jobs) != len(referencedata.ReferenceBuild.Jobs) {
			t.Fatal("Pipeline count mismatch")
		}
		for i := 0; i < len(build.Jobs); i++ {
			candidateJob := build.Jobs[i]
			referenceJob := referencedata.ReferenceBuild.Jobs[i]
			if candidateJob.Type != candidateJob.Type {
				t.Error("Job type mismatch")
			}
			if candidateJob.Name != referenceJob.Name {
				t.Error("Job name mismatch")
			}
			if candidateJob.Description != referenceJob.Description {
				t.Error("Job description mismatch")
			}
			if candidateJob.DockerImage != candidateJob.DockerImage {
				t.Error("Job docker image mismatch")
			}
			if candidateJob.StepExecution != referenceJob.StepExecution {
				t.Error("Job step execution mismatch")
			}
			if len(candidateJob.Depends) != len(referenceJob.Depends) {
				t.Fatal("Job dependency count mismatch")
			}
			for k := 0; k < len(candidateJob.Depends); k++ {
				candidateJobDependency := candidateJob.Depends[k]
				referenceJobDependency := referenceJob.Depends[k]
				if !candidateJobDependency.Equal(referenceJobDependency) {
					t.Errorf("Job dependency name mismatch: '%s.%s' should match reference '%s.%s'",
						candidateJobDependency.Workflow, candidateJobDependency.JobName, referenceJobDependency.Workflow, referenceJobDependency.JobName)
				}
			}
			if len(candidateJob.Environment) != len(candidateJob.Environment) {
				t.Fatal("Step environment count mismatch")
			}
			candidateJobEnv := candidateJob.Environment
			sort.SliceStable(candidateJobEnv, func(i, j int) bool {
				return candidateJobEnv[i].Name < candidateJobEnv[j].Name
			})
			referenceStepEnv := referenceJob.Environment
			sort.SliceStable(referenceStepEnv, func(i, j int) bool {
				return referenceStepEnv[i].Name < referenceStepEnv[j].Name
			})
			for k := 0; k < len(candidateJobEnv); k++ {
				candidateStepEnvironment := candidateJobEnv[k]
				referenceStepEnvironment := referenceStepEnv[k]
				if candidateStepEnvironment.Name != referenceStepEnvironment.Name {
					t.Error("Step environment name mismatch")
				}
				if candidateStepEnvironment.Value != referenceStepEnvironment.Value {
					t.Error("Step environment value mismatch")
				}
				if candidateStepEnvironment.ValueFromSecret != referenceStepEnvironment.ValueFromSecret {
					t.Error("Step environment from secret mismatch")
				}
			}
			for j := 0; j < len(candidateJob.Steps); j++ {
				candidateStep := candidateJob.Steps[j]
				referenceStep := referenceJob.Steps[j]
				if candidateStep.Name != referenceStep.Name {
					t.Error("Step name mismatch")
				}
				if candidateStep.Description != referenceStep.Description {
					t.Error("Step description mismatch")
				}
				if len(candidateStep.Depends) != len(referenceStep.Depends) {
					t.Fatal("Step dependency count mismatch")
				}
				for k := 0; k < len(candidateStep.Depends); k++ {
					candidateStepDependency := candidateStep.Depends[k]
					referenceStepDependency := referenceStep.Depends[k]
					if !candidateStepDependency.Equal(referenceStepDependency) {
						t.Error("Step dependency name mismatch")
					}
				}
				if len(candidateStep.Commands) != len(referenceStep.Commands) {
					t.Fatal("Step commands count mismatch")
				}
				for k := 0; k < len(candidateStep.Commands); k++ {
					candidateStepCommand := candidateStep.Commands[k]
					referenceStepCommand := referenceStep.Commands[k]
					if candidateStepCommand != referenceStepCommand {
						t.Error("Step command mismatch")
					}
				}
			}
		}
	}
}
