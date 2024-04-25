package dto_test_test

import (
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

func TestJob(t *testing.T) {

	now := models.NewTime(time.Now())
	repoID := models.NewRepoID()
	commitID := models.NewCommitID()
	ref := "refs/master/HEAD"
	buildID := models.NewBuildID()
	jobID := models.NewJobID()

	job := &dto.JobGraph{
		Job: &models.Job{
			JobMetadata: models.JobMetadata{
				ID:        models.NewJobID(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			JobData: models.JobData{
				RepoID:   repoID,
				CommitID: commitID,
				Ref:      ref,
				BuildID:  buildID,
				Status:   models.WorkflowStatusQueued,
				JobDefinitionData: models.JobDefinitionData{
					Name:                    "test",
					Type:                    "docker",
					DockerImage:             "golang",
					DockerImagePullStrategy: models.DockerPullStrategyDefault,
				},
			},
		},
		Steps: []*models.Step{
			&models.Step{
				StepMetadata: models.StepMetadata{
					ID:        models.NewStepID(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				StepData: models.StepData{
					JobID:  jobID,
					RepoID: repoID,
					Status: models.WorkflowStatusQueued,
					StepDefinitionData: models.StepDefinitionData{
						Name:    "a",
						Depends: []*models.StepDependency{},
						Commands: []models.Command{
							"echo 'hello world'",
						},
					},
				},
			},
			&models.Step{
				StepMetadata: models.StepMetadata{
					ID:        models.NewStepID(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				StepData: models.StepData{
					JobID:  jobID,
					RepoID: repoID,
					Status: models.WorkflowStatusQueued,
					StepDefinitionData: models.StepDefinitionData{
						Name:    "b",
						Depends: []*models.StepDependency{},
						Commands: []models.Command{
							"echo 'hello world'",
						},
					},
				},
			},
			&models.Step{
				StepMetadata: models.StepMetadata{
					ID:        models.NewStepID(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				StepData: models.StepData{
					JobID:  jobID,
					RepoID: repoID,
					Status: models.WorkflowStatusQueued,
					StepDefinitionData: models.StepDefinitionData{
						Name: "c",
						Depends: []*models.StepDependency{
							models.NewStepDependency("a"),
						},
						Commands: []models.Command{
							"echo 'hello world'",
						},
					},
				},
			},
			&models.Step{
				StepMetadata: models.StepMetadata{
					ID:        models.NewStepID(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				StepData: models.StepData{
					JobID:  jobID,
					RepoID: repoID,
					Status: models.WorkflowStatusQueued,
					StepDefinitionData: models.StepDefinitionData{
						Name: "d",
						Depends: []*models.StepDependency{
							models.NewStepDependency("a"),
							models.NewStepDependency("b"),
						},
						Commands: []models.Command{
							"echo 'hello world'",
						},
					},
				},
			},
		},
	}

	err := job.Validate()
	if err != nil {
		t.Fatalf("Expected valid job: %s", err)
	}

	visited := make(map[models.ResourceName]*models.Step)
	err = job.Walk(false, func(step *models.Step) error {
		visited[step.Name] = step
		return nil
	})
	if err != nil {
		t.Fatalf("Expected successful walk: %s", err)
	}
}
