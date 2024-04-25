package referencedata

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

var now = time.Now().UTC()

const (
	TestPersonEmail       = "alice123@not-a-real-domain.com"
	TestPersonLegalName   = "Alice Knoffler"
	TestPersonName        = "alice123"
	TestCompanyEmail      = "test@company.com"
	TestCompanyLegalName  = "Test Org"
	TestCompanyName       = "theorg"
	TestCompany2Email     = "2" + TestCompanyEmail
	TestCompany2LegalName = TestCompanyLegalName + "2"
	TestCompany2Name      = TestCompanyName + "2"
	TestCompany3Email     = "3" + TestCompanyEmail
	TestCompany3LegalName = TestCompanyLegalName + "3"
	TestCompany3Name      = TestCompanyName + "3"
	TestRepoName          = "buildbeaver"
	TestRef               = "refs/master/HEAD"
	TestRef2              = "refs/heads/main"
	TestRef3              = "refs/heads/anotherrefdown"
)

var ReferenceJob1 = &dto.JobGraph{
	Job: &models.Job{
		JobMetadata: models.JobMetadata{
			ID:        models.NewJobID(),
			CreatedAt: models.NewTime(now),
			UpdatedAt: models.NewTime(now),
		},
		JobData: models.JobData{
			Ref:    TestRef,
			Status: models.WorkflowStatusQueued,
			JobDefinitionData: models.JobDefinitionData{
				Name:                    "test_job_1",
				Description:             "this is the first test job",
				Type:                    "docker",
				DockerImage:             "golang",
				DockerImagePullStrategy: models.DockerPullStrategyDefault,
				StepExecution:           models.StepExecutionParallel,
				ArtifactDefinitions: []*models.ArtifactDefinition{
					&models.ArtifactDefinition{
						GroupName: "test_artifact_1",
						Paths: []string{
							".build/go/*",
						},
					},
				},
				Environment: []*models.EnvVar{
					&models.EnvVar{
						Name:         "foo",
						SecretString: models.SecretString{Value: "bar"},
					},
					&models.EnvVar{
						Name:         "foo2",
						SecretString: models.SecretString{Value: "bar2"},
					},
					&models.EnvVar{
						Name:         "foo3",
						SecretString: models.SecretString{ValueFromSecret: "name_of_secret"},
					},
				},
			},
		},
	},
	Steps: []*models.Step{
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step_1",
					Description: "builds the foo",
					Commands: []models.Command{
						"./build-foo.sh",
					},
				},
			},
		},
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step_2",
					Description: "builds the bar",
					Commands: []models.Command{
						"./build-bar.sh",
					},
					Depends: []*models.StepDependency{
						models.NewStepDependency("test_step_1"),
					},
				},
			},
		},
	},
}

var ReferenceJob2 = &dto.JobGraph{
	Job: &models.Job{
		JobMetadata: models.JobMetadata{
			ID:        models.NewJobID(),
			CreatedAt: models.NewTime(now),
			UpdatedAt: models.NewTime(now),
		},
		JobData: models.JobData{
			Ref:    TestRef,
			Status: models.WorkflowStatusQueued,
			JobDefinitionData: models.JobDefinitionData{
				Name:                    "test_job_2",
				Description:             "this is the second test job",
				Type:                    "docker",
				DockerImage:             "golang",
				DockerImagePullStrategy: models.DockerPullStrategyDefault,
				Depends: []*models.JobDependency{
					models.NewJobDependency(
						"",
						ReferenceJob1.Name,
						models.NewArtifactDependency(
							"",
							ReferenceJob1.Name,
							ReferenceJob1.ArtifactDefinitions[0].GroupName)),
				},
				StepExecution: models.StepExecutionSequential,
			},
		},
	},
	Steps: []*models.Step{
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step",
					Description: "does the things",
					Commands: []models.Command{
						"./do-all-the-things.sh",
					},
				},
			},
		},
	},
}

var ReferenceJob3 = &dto.JobGraph{
	Job: &models.Job{
		JobMetadata: models.JobMetadata{
			ID:        models.NewJobID(),
			CreatedAt: models.NewTime(now),
			UpdatedAt: models.NewTime(now),
		},
		JobData: models.JobData{
			Ref:    TestRef,
			Status: models.WorkflowStatusQueued,
			JobDefinitionData: models.JobDefinitionData{
				Name:                    "test_job_3",
				Description:             "this is the third test job",
				Type:                    "docker",
				DockerImage:             "golang",
				DockerImagePullStrategy: models.DockerPullStrategyDefault,
				Depends: []*models.JobDependency{
					models.NewJobDependency("", ReferenceJob1.Name),
				},
				StepExecution: models.StepExecutionSequential,
			},
		},
	},
	Steps: []*models.Step{
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step",
					Description: "does the things",
					Commands: []models.Command{
						"./do-all-the-things.sh",
					},
				},
			},
		},
	},
}

var ReferenceJob4 = &dto.JobGraph{
	Job: &models.Job{
		JobMetadata: models.JobMetadata{
			ID:        models.NewJobID(),
			CreatedAt: models.NewTime(now),
			UpdatedAt: models.NewTime(now),
		},
		JobData: models.JobData{
			Ref:    TestRef,
			Status: models.WorkflowStatusQueued,
			JobDefinitionData: models.JobDefinitionData{
				Name:                    "test_job_4",
				Description:             "this is the fourth test job",
				Type:                    "docker",
				DockerImage:             "golang",
				DockerImagePullStrategy: models.DockerPullStrategyDefault,
				Depends: []*models.JobDependency{
					models.NewJobDependency("", ReferenceJob1.Name),
				},
				StepExecution: models.StepExecutionSequential,
			},
		},
	},
	Steps: []*models.Step{
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step",
					Description: "does the things",
					Commands: []models.Command{
						"./do-all-the-things.sh",
					},
				},
			},
		},
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step_2",
					Description: "does the things again",
					Commands: []models.Command{
						"./do-all-the-things.sh",
					},
					Depends: []*models.StepDependency{models.NewStepDependency("test_step")},
				},
			},
		},
		&models.Step{
			StepMetadata: models.StepMetadata{
				ID:        models.NewStepID(),
				CreatedAt: models.NewTime(now),
				UpdatedAt: models.NewTime(now),
			},
			StepData: models.StepData{
				Status: models.WorkflowStatusQueued,
				StepDefinitionData: models.StepDefinitionData{
					Name:        "test_step_3",
					Description: "does the things again, again!",
					Commands: []models.Command{
						"./do-all-the-things.sh",
					},
					Depends: []*models.StepDependency{models.NewStepDependency("test_step_2")},
				},
			},
		},
	},
}

var ReferenceBuild = &dto.BuildGraph{
	Build: &models.Build{
		ID:        models.NewBuildID(),
		CreatedAt: models.NewTime(now),
		UpdatedAt: models.NewTime(now),
		Status:    models.WorkflowStatusQueued,
		Opts: models.BuildOptions{
			NodesToRun: []models.NodeFQN{},
		},
		Name: "1",
		Ref:  TestRef,
	},
	Jobs: []*dto.JobGraph{
		ReferenceJob1,
		ReferenceJob2,
		ReferenceJob3,
		ReferenceJob4,
	},
}

var PipelineJSON = `
{
	"version": "0.3",
    "jobs": [
		{
			"name": "` + ReferenceJob1.Name.String() + `",
			"description": "` + ReferenceJob1.Description + `",
			"type":"` + ReferenceJob1.Type.String() + `",
			"docker": {"image":"` + ReferenceJob1.DockerImage + `"},
			"step_execution":"` + ReferenceJob1.StepExecution.String() + `",
			"steps": [
				{
					"name": "` + ReferenceJob1.Steps[0].Name.String() + `",
					"description": "` + ReferenceJob1.Steps[0].Description + `",
					"commands": [
						"` + string(ReferenceJob1.Steps[0].Commands[0]) + `"
					]
				},
				{
					"name": "` + ReferenceJob1.Steps[1].Name.String() + `",
					"description":"` + ReferenceJob1.Steps[1].Description + `",
					"commands": [
						"` + string(ReferenceJob1.Steps[1].Commands[0]) + `"
					],
					"depends": [
						"` + ReferenceJob1.Steps[0].Name.String() + `"
					]
				}
			],
			"artifacts":[
				{
					"name": "` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `",
					"paths": [
						"` + ReferenceJob1.ArtifactDefinitions[0].Paths[0] + `"
					]
				}
			],
			"environment": {
				"foo": "bar",
				"foo2": {
					"value": "bar2"
				},
				"foo3": {
					"from_secret": "name_of_secret"
				}
			}
		},
		{
			"name": "` + ReferenceJob2.Name.String() + `",
			"description": "` + ReferenceJob2.Description + `",
			"type":"` + ReferenceJob2.Type.String() + `",
    	    "docker": {"image":"` + ReferenceJob2.DockerImage + `"},
			"step_execution":"` + ReferenceJob2.StepExecution.String() + `",
			"steps": [
				{
					"name":"` + ReferenceJob2.Steps[0].Name.String() + `",
					"description": "` + ReferenceJob2.Steps[0].Description + `",
					"commands": [
						"` + string(ReferenceJob2.Steps[0].Commands[0]) + `"
					]
				}
			],
	        "depends": [
			    "jobs.` + ReferenceJob1.Name.String() + `.artifacts.` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `"
			]
		},
		{
			"name": "` + ReferenceJob3.Name.String() + `",
			"description": "` + ReferenceJob3.Description + `",
			"type":"` + ReferenceJob3.Type.String() + `",
        	"docker": {"image":"` + ReferenceJob3.DockerImage + `"},
			"step_execution":"` + ReferenceJob3.StepExecution.String() + `",
			"steps": [
				{
					"name":"` + ReferenceJob3.Steps[0].Name.String() + `",
					"description": "` + ReferenceJob3.Steps[0].Description + `",
					"commands": [
						"` + string(ReferenceJob3.Steps[0].Commands[0]) + `"
					]
				}
			],
			"depends": [
				"` + ReferenceJob1.Name.String() + `"
			]
		},
		{
			"name": "` + ReferenceJob4.Name.String() + `",
			"description": "` + ReferenceJob4.Description + `",
			"type":"` + ReferenceJob4.Type.String() + `",
    	    "docker": {"image":"` + ReferenceJob4.DockerImage + `"},
			"step_execution":"` + ReferenceJob4.StepExecution.String() + `",
			"steps": [
				{
					"name":"` + ReferenceJob4.Steps[0].Name.String() + `",
					"description": "` + ReferenceJob4.Steps[0].Description + `",
					"commands": [
						"` + string(ReferenceJob4.Steps[0].Commands[0]) + `"
					]
				},
				{
					"name":"` + ReferenceJob4.Steps[1].Name.String() + `",
					"description": "` + ReferenceJob4.Steps[1].Description + `",
					"commands": [
						"` + string(ReferenceJob4.Steps[1].Commands[0]) + `"
					]
				},
				{
					"name":"` + ReferenceJob4.Steps[2].Name.String() + `",
					"description": "` + ReferenceJob4.Steps[2].Description + `",
					"commands": [
						"` + string(ReferenceJob4.Steps[2].Commands[0]) + `"
					]
				}
			],
			"depends": [
				"` + ReferenceJob1.Name.String() + `"
			]
		}
	]
}
`

var PipelineJSONNET = `
local build(version, jobs) = {
    version: version,
    jobs: jobs
};

local job(name, description, type, image, execution, steps, artifacts, depends, environment) = {
	name: name,
	description: description,
	type: type,
    docker: {
        image: image,
    },
	step_execution: execution,
	steps: steps,
	artifacts: artifacts,
	depends: depends,
	environment: environment
};

local step(name, description, commands, depends) = {
	name: name,
	description: description,
	commands: commands,
	depends: depends
};

build("0.3",
  [
	job(
		"` + ReferenceJob1.Name.String() + `", 
		"` + ReferenceJob1.Description + `",
		"` + ReferenceJob1.Type.String() + `",
        "` + ReferenceJob1.DockerImage + `",
		"` + ReferenceJob1.StepExecution.String() + `",
		[
			step(
				"` + ReferenceJob1.Steps[0].Name.String() + `",
				"` + ReferenceJob1.Steps[0].Description + `",
				[
					"` + string(ReferenceJob1.Steps[0].Commands[0]) + `"
				],
				[],
			),
			step(
				"` + ReferenceJob1.Steps[1].Name.String() + `", 
				"` + ReferenceJob1.Steps[1].Description + `",
				[
					"` + string(ReferenceJob1.Steps[1].Commands[0]) + `"
				],
				[
					"` + ReferenceJob1.Steps[0].Name.String() + `"
				],
			)
		],
		[
			{
				"name": "` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `",
				"paths": [
					"` + ReferenceJob1.ArtifactDefinitions[0].Paths[0] + `"
				]
			}
		],
		[],
		{
			"foo": "bar",
			"foo2": {
				"value": "bar2"
			},
			"foo3": {
				"from_secret": "name_of_secret"
			}
		},
	),
	job(
		"` + ReferenceJob2.Name.String() + `", 
		"` + ReferenceJob2.Description + `",
		"` + ReferenceJob2.Type.String() + `",
        "` + ReferenceJob2.DockerImage + `",
		"` + ReferenceJob2.StepExecution.String() + `",
		[
			step(
				"` + ReferenceJob2.Steps[0].Name.String() + `", 
				"` + ReferenceJob2.Steps[0].Description + `",
				[
					"` + string(ReferenceJob2.Steps[0].Commands[0]) + `",
				],
				[],
			)
		],
		[],
		[
			"jobs.` + ReferenceJob1.Name.String() + `.artifacts.` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `"
        ],
		{},
	),
	job(
		"` + ReferenceJob3.Name.String() + `", 
		"` + ReferenceJob3.Description + `",
		"` + ReferenceJob3.Type.String() + `",
        "` + ReferenceJob3.DockerImage + `",
		"` + ReferenceJob3.StepExecution.String() + `",
		[
			step(
				"` + ReferenceJob3.Steps[0].Name.String() + `", 
				"` + ReferenceJob3.Steps[0].Description + `",
				[
					"` + string(ReferenceJob3.Steps[0].Commands[0]) + `",
				],
				[],
			)
		],
		[],
		[
			"` + ReferenceJob1.Name.String() + `"
		],
		{},
	),
	job(
		"` + ReferenceJob4.Name.String() + `", 
		"` + ReferenceJob4.Description + `",
		"` + ReferenceJob4.Type.String() + `",
        "` + ReferenceJob4.DockerImage + `",
		"` + ReferenceJob4.StepExecution.String() + `",
		[
			step(
				"` + ReferenceJob4.Steps[0].Name.String() + `", 
				"` + ReferenceJob4.Steps[0].Description + `",
				[
					"` + string(ReferenceJob4.Steps[0].Commands[0]) + `",
				],
				[],
			),
			step(
				"` + ReferenceJob4.Steps[1].Name.String() + `", 
				"` + ReferenceJob4.Steps[1].Description + `",
				[
					"` + string(ReferenceJob4.Steps[1].Commands[0]) + `",
				],
				[],
			),
			step(
				"` + ReferenceJob4.Steps[2].Name.String() + `", 
				"` + ReferenceJob4.Steps[2].Description + `",
				[
					"` + string(ReferenceJob4.Steps[2].Commands[0]) + `",
				],
				[],
			)
		],
		[],
		[
			"` + ReferenceJob1.Name.String() + `"
		],
		{},
	)
  ]
)
`

var PipelineYAML = `
---
version: 0.3
jobs:
  - name: ` + ReferenceJob1.Name.String() + `
    description: ` + ReferenceJob1.Description + `
    type: ` + ReferenceJob1.Type.String() + `
    docker:
      image: ` + ReferenceJob1.DockerImage + `
    step_execution: ` + ReferenceJob1.StepExecution.String() + `
    steps:
      - name: ` + ReferenceJob1.Steps[0].Name.String() + `
        description: ` + ReferenceJob1.Steps[0].Description + `
        commands:
          - ` + string(ReferenceJob1.Steps[0].Commands[0]) + `
      - name: ` + ReferenceJob1.Steps[1].Name.String() + `
        description: ` + ReferenceJob1.Steps[1].Description + `
        commands:
          - ` + string(ReferenceJob1.Steps[1].Commands[0]) + `
        depends:
          - ` + ReferenceJob1.Steps[0].Name.String() + `
    artifacts:
      - name: ` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `
        paths:
          - ` + ReferenceJob1.ArtifactDefinitions[0].Paths[0] + `
    environment:
      ` + ReferenceJob1.Environment[0].Name + `: ` + ReferenceJob1.Environment[0].Value + `
      ` + ReferenceJob1.Environment[1].Name + `:
        value: ` + ReferenceJob1.Environment[1].Value + `
      ` + ReferenceJob1.Environment[2].Name + `:
        from_secret: ` + ReferenceJob1.Environment[2].ValueFromSecret + `

  - name: ` + ReferenceJob2.Name.String() + `
    description: ` + ReferenceJob2.Description + `
    type: ` + ReferenceJob2.Type.String() + `
    docker:
      image: ` + ReferenceJob2.DockerImage + `
    step_execution: ` + ReferenceJob2.StepExecution.String() + `
    depends:
      - jobs.` + ReferenceJob1.Name.String() + `.artifacts.` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `
    steps:
      - name: ` + ReferenceJob2.Steps[0].Name.String() + `
        description: ` + ReferenceJob2.Steps[0].Description + `
        commands:
          - ` + string(ReferenceJob2.Steps[0].Commands[0]) + `

  - name: ` + ReferenceJob3.Name.String() + `
    description: ` + ReferenceJob3.Description + `
    type: ` + ReferenceJob3.Type.String() + `
    docker:
      image: ` + ReferenceJob3.DockerImage + `
    step_execution: ` + ReferenceJob3.StepExecution.String() + `
    steps:
      - name: ` + ReferenceJob3.Steps[0].Name.String() + `
        description: ` + ReferenceJob3.Steps[0].Description + `
        commands:
          - ` + string(ReferenceJob3.Steps[0].Commands[0]) + `
    depends:
      - ` + ReferenceJob1.Name.String() + `

  - name: ` + ReferenceJob4.Name.String() + `
    description: ` + ReferenceJob4.Description + `
    type: ` + ReferenceJob4.Type.String() + `
    docker:
      image: ` + ReferenceJob4.DockerImage + `
    step_execution: ` + ReferenceJob4.StepExecution.String() + `
    steps:
      - name: ` + ReferenceJob4.Steps[0].Name.String() + `
        description: ` + ReferenceJob4.Steps[0].Description + `
        commands:
          - ` + string(ReferenceJob4.Steps[0].Commands[0]) + `
      - name: ` + ReferenceJob4.Steps[1].Name.String() + `
        description: ` + ReferenceJob4.Steps[1].Description + `
        commands:
          - ` + string(ReferenceJob4.Steps[1].Commands[0]) + `
      - name: ` + ReferenceJob4.Steps[2].Name.String() + `
        description: ` + ReferenceJob4.Steps[2].Description + `
        commands:
          - ` + string(ReferenceJob4.Steps[2].Commands[0]) + `
    depends:
      - ` + ReferenceJob1.Name.String() + `
`

var PipelineWithInvalidYAML = `
---
version: "0.3"
jobs:
  - name: ` + ReferenceJob1.Name.String() + `
    description: ` + ReferenceJob1.Description + `
    type: ` + ReferenceJob1.Type.String() + `
    docker:
      image_what: ` + ReferenceJob1.DockerImage + `
    step_execution: ` + ReferenceJob1.StepExecution.String() + `
    steps:
      - name: ` + ReferenceJob1.Steps[0].Name.String() + `
        description: ` + ReferenceJob1.Steps[0].Description + `
        commands:
          - ` + string(ReferenceJob1.Steps[0].Commands[0]) + `
      - name: ` + ReferenceJob1.Steps[1].Name.String() + `
        description: ` + ReferenceJob1.Steps[1].Description + `
        commands:
          - ` + string(ReferenceJob1.Steps[1].Commands[0]) + `
        depends:
          - ` + ReferenceJob1.Steps[0].Name.String() + `
	artifacts:
	  - name: ` + ReferenceJob1.ArtifactDefinitions[0].GroupName.String() + `
		paths:
		  - ` + ReferenceJob1.ArtifactDefinitions[0].Paths[0] + `
    environment:
      ` + ReferenceJob1.Environment[0].Name + `: ` + ReferenceJob1.Environment[0].Value + `
      ` + ReferenceJob1.Environment[1].Name + `:
        from_secret: ` + ReferenceJob1.Environment[1].ValueFromSecret + `
`

func GenerateCommit(repoID models.RepoID, legalEntityID models.LegalEntityID) *models.Commit {
	now := models.NewTime(time.Now())

	return &models.Commit{
		ID:          models.NewCommitID(),
		CreatedAt:   now,
		Config:      []byte(PipelineYAML),
		ConfigType:  models.ConfigTypeYAML,
		SHA:         util.RandAlphaString(8),
		Message:     util.RandAlphaString(24),
		RepoID:      repoID,
		CommitterID: legalEntityID,
		AuthorID:    legalEntityID,
	}
}

func GenerateInvalidCommit(repoID models.RepoID, legalEntityID models.LegalEntityID) *models.Commit {
	now := models.NewTime(time.Now())

	return &models.Commit{
		ID:          models.NewCommitID(),
		CreatedAt:   now,
		Config:      []byte(PipelineWithInvalidYAML),
		ConfigType:  models.ConfigTypeYAML,
		SHA:         util.RandAlphaString(8),
		Message:     util.RandAlphaString(24),
		RepoID:      repoID,
		CommitterID: legalEntityID,
		AuthorID:    legalEntityID,
	}
}

func GenerateName(prefix string) models.ResourceName {
	return models.ResourceName(fmt.Sprintf("%s%s", prefix, util.RandAlphaString(32)))
}

func GeneratePersonLegalEntity(name models.ResourceName, legalName string, emailAddress string) *models.LegalEntityData {
	if name == "" {
		name = TestPersonName
	}
	if legalName == "" {
		legalName = TestPersonLegalName
	}
	if emailAddress == "" {
		emailAddress = TestPersonEmail
	}

	return models.NewPersonLegalEntityData(name, legalName, emailAddress, nil, "")
}

func GenerateCompanyLegalEntity(name models.ResourceName, legalName string, emailAddress string) *models.LegalEntityData {
	if name == "" {
		name = TestCompanyName
	}
	if legalName == "" {
		legalName = TestCompanyLegalName
	}
	if emailAddress == "" {
		emailAddress = TestCompanyEmail
	}

	return models.NewCompanyLegalEntityData(name, legalName, emailAddress, nil, "")
}

func GenerateRepo(repoName string, legalEntityId models.LegalEntityID) *models.Repo {
	now := models.NewTime(time.Now())
	if repoName == "" {
		repoName = TestRepoName
	}
	repoExternalId := models.NewExternalResourceID("github", repoName)
	return models.NewRepo(now, models.ResourceName(repoName), legalEntityId, "", "", fmt.Sprintf("https://github.com/%s", repoName), "", "master", true, false, nil, &repoExternalId, "")
}

func GenerateBuild(repoID models.RepoID, commitID models.CommitID, logDescriptorID models.LogDescriptorID, ref string, minimumCount int) *dto.BuildGraph {
	now := models.NewTime(time.Now())
	build := &dto.BuildGraph{
		Build: &models.Build{
			ID:              models.NewBuildID(),
			Name:            GenerateName("build-"),
			RepoID:          repoID,
			CreatedAt:       now,
			UpdatedAt:       now,
			DeletedAt:       nil,
			ETag:            "",
			CommitID:        commitID,
			LogDescriptorID: logDescriptorID,
			Ref:             ref,
			Status:          models.WorkflowStatusQueued,
			Timings: models.WorkflowTimings{
				QueuedAt: &now,
			},
			Error: nil,
			Opts:  models.BuildOptions{},
		},
		Jobs: nil,
	}
	nJobs := rand.Intn(10) + minimumCount
	for i := 0; i < nJobs; i++ {
		job := GenerateJob(repoID, commitID, build.ID, logDescriptorID, ref, minimumCount)
		build.Jobs = append(build.Jobs, job)
	}
	return build
}

func GenerateJob(repoID models.RepoID, commitID models.CommitID, buildID models.BuildID, logDescriptorID models.LogDescriptorID, ref string, minimumCount int) *dto.JobGraph {
	now := models.NewTime(time.Now())
	job := &dto.JobGraph{
		Job: &models.Job{
			JobMetadata: models.JobMetadata{
				ID:        models.NewJobID(),
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: nil,
				ETag:      "",
			},
			JobData: models.JobData{
				BuildID:         buildID,
				RepoID:          repoID,
				CommitID:        commitID,
				LogDescriptorID: logDescriptorID,
				RunnerID:        models.RunnerID{},
				Ref:             ref,
				Status:          models.WorkflowStatusQueued,
				Timings: models.WorkflowTimings{
					QueuedAt: &now,
				},
				Fingerprint:         "",
				FingerprintHashType: nil,
				IndirectToJobID:     models.JobID{},
				Error:               nil,
				JobDefinitionData: models.JobDefinitionData{
					Name:                    GenerateName("job-"),
					Description:             fmt.Sprintf("This is a random description %s", util.RandAlphaString(10)),
					Depends:                 nil,
					Services:                nil,
					Type:                    models.JobTypeDocker,
					DockerImage:             "golang:1.18",
					DockerImagePullStrategy: models.DockerPullStrategyDefault,
					DockerAuth:              nil,
					StepExecution:           models.StepExecutionSequential,
					FingerprintCommands:     nil,
					ArtifactDefinitions: models.ArtifactDefinitions{
						&models.ArtifactDefinition{
							GroupName: "default",
							Paths:     []string{fmt.Sprintf("random-file-%s", util.RandAlphaString(10))},
						},
					},
					Environment: models.JobEnvVars{&models.EnvVar{
						Name: "USEFUL_THING",
						SecretString: models.SecretString{
							Value: util.RandAlphaString(10),
						},
					}},
				},
			},
		},
		Steps: nil,
	}
	nSteps := rand.Intn(10) + minimumCount
	for j := 0; j < nSteps; j++ {
		step := GenerateStep(repoID, commitID, job.ID, logDescriptorID)
		job.Steps = append(job.Steps, step)
	}
	return job
}

func GenerateStep(repoID models.RepoID, commitID models.CommitID, jobID models.JobID, logDescriptorID models.LogDescriptorID) *models.Step {
	now := models.NewTime(time.Now())
	return &models.Step{
		StepMetadata: models.StepMetadata{
			ID:        models.NewStepID(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
			ETag:      "",
		},
		StepData: models.StepData{
			JobID:           jobID,
			RepoID:          repoID,
			RunnerID:        models.RunnerID{},
			LogDescriptorID: logDescriptorID,
			Status:          models.WorkflowStatusQueued,
			Timings: models.WorkflowTimings{
				QueuedAt: &now,
			},
			Error: nil,
			StepDefinitionData: models.StepDefinitionData{
				Name:        GenerateName("step-"),
				Description: fmt.Sprintf("This is a random description %s", util.RandAlphaString(10)),
				Commands:    models.Commands{"echo \"hello world\""},
				Depends:     nil,
			},
		},
	}
}
