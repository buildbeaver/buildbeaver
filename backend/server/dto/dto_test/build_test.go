package dto_test_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

func TestBuild(t *testing.T) {

	now := models.NewTime(time.Now())
	repoID := models.NewRepoID()
	commitID := models.NewCommitID()
	ref := "refs/master/HEAD"
	buildID := models.NewBuildID()
	job1ID := models.NewJobID()
	job2ID := models.NewJobID()

	build := &dto.BuildGraph{
		Build: &models.Build{
			ID:        buildID,
			Name:      "1",
			CreatedAt: now,
			UpdatedAt: now,
			RepoID:    repoID,
			CommitID:  commitID,
			Ref:       ref,
			Status:    models.WorkflowStatusQueued,
		},
		Jobs: []*dto.JobGraph{
			&dto.JobGraph{
				Job: &models.Job{
					JobMetadata: models.JobMetadata{
						ID:        job1ID,
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
							Name:     "a",
							Workflow: "w",
							Depends: []*models.JobDependency{
								models.NewJobDependency("w", "b"),
							},
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
			},
			&dto.JobGraph{
				Job: &models.Job{
					JobMetadata: models.JobMetadata{
						ID:        job2ID,
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
							Name:                    "b",
							Workflow:                "w",
							Depends:                 []*models.JobDependency{},
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
							JobID:  job1ID,
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
			},
		},
	}

	err := build.Validate()
	if err != nil {
		t.Fatalf("Expected valid build pipeline: %s", err)
	}

	// Test ancestors works... b depends on a, a is therefore an ancestor of b
	a := build.Jobs[0]
	b := build.Jobs[1]
	ancestors, err := build.Ancestors(b)
	require.NoError(t, err)
	require.Len(t, ancestors, 1)
	require.Equal(t, ancestors[0], a)

	visited := make(map[models.ResourceName]*dto.JobGraph)
	err = build.Walk(false, func(job *dto.JobGraph) error {
		visited[job.Name] = job
		return nil
	})
	if err != nil {
		t.Fatalf("Expected successful walk: %s", err)
	}

	err = build.Trim([]models.NodeFQN{models.NewNodeFQN("w", "b", "c")})
	if err != nil {
		t.Fatalf("Expected successful trim: %s", err)
	}
}
