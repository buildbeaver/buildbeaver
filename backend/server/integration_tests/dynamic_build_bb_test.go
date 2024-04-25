package integration_tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/server/app/server_test"
)

func TestBuildBeaverDynamicBuild(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)

	commit, buildRunner := createPrerequisiteObjects(t, app)
	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	build := runBuildBeaverDynamicBuildJob(jobinatorEnv)
	build.Shutdown()
}

func runBuildBeaverDynamicBuildJob(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Test Job 2")

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("test").Handler(
			func(workflow *bb.Workflow) error {

				// buildFinger is a list of fingerprint commands for all jobs, to detect build changes
				var buildFingerprint = []string{
					"find build/scripts -type f | sort | xargs sha1sum",
					"find build/docker -type f | sort | xargs sha1sum",
				}

				// goFinger is a list of fingerprint commands appropriate to Go-related jobs, including the jobs in buildFinger
				var goFingerprint = append(buildFingerprint,
					"find backend/ -name '*.go' -not -path \"*/vendor/*\" -type f | sort | xargs sha1sum",
				)

				checkJobCount(t, 1)

				baseJob := bb.NewJob().
					Name("base").
					Desc("Builds the base images needed for the build pipeline").
					Docker(bb.NewDocker().
						Image("docker:20.10").
						Pull(bb.DockerPullIfNotExists)).
					Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
					Step(bb.NewStep().
						Name("go-builder").
						Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/"))
				workflow.Job(baseJob)
				_, err := workflow.Submit(false) // Try submitting the job in multiple steps
				require.NoError(t, err)

				checkJobCount(t, 2)

				generateJob := bb.NewJob().
					Name("generate").
					Desc("Generates all code (wire files, protobufs etc.").
					DependsOnJobs(baseJob).
					RunsOn("linux").
					Docker(bb.NewDocker().
						Image("buildbeaver/go-builder:latest").
						Pull(bb.DockerPullNever).
						Shell("/bin/bash")).
					// Fingerprint(goFinger...).
					Step(bb.NewStep().
						Name("format").
						Commands(`cd backend
                     if [ "$(goimports -d $(find . -type f -name '*.go' -not -path '*/vendor/*' -not -path '*/wire_gen.go'))" != "" ]; then
                       echo "Looks like you forgot to run 'goimports' before committing"
                       exit 1
                     fi`)).
					Step(bb.NewStep().
						Name("generate").
						Commands(`. build/scripts/lib/go-env.sh
                     for wire_file in backend/*/app/wire.go backend/*/app/*/wire.go; do
                       pushd "$(dirname "${wire_file}")"
                       wire
                       popd
                     done`)).
					Artifact(bb.NewArtifact().
						Name("wire").
						Paths("backend/*/app/wire_gen.go", "backend/*/app/*/wire_gen.go")).
					Artifact(bb.NewArtifact().
						Name("grpc").
						Paths("backend/api/grpc/*.pb.go"))
				workflow.Job(generateJob)

				buildJob := bb.NewJob().
					Name("build").
					Desc("Build all binaries").
					DependsOnJobs(baseJob).
					DependsOnJobArtifacts(generateJob).
					Docker(bb.NewDocker().
						Image("buildbeaver/go-builder:latest").
						Pull(bb.DockerPullNever).
						Shell("/bin/bash")).
					Fingerprint(goFingerprint...).
					Step(bb.NewStep().
						Name("go").
						Commands(`. build/scripts/lib/go-env.sh
	         for cmd_dir in backend/*/cmd/*; do
               bin_name="$(basename "${cmd_dir}")"
               bin_out="${GOBIN}/${bin_name}"
               pushd "${cmd_dir}"
                 echo "Building: ${bin_name} > ${bin_out}"
                 go build -mod=vendor -o "${bin_out}" .
               popd
             done`)).
					Artifact(bb.NewArtifact().
						Name("default").
						Paths("build/output/go/bin/*"))
				workflow.Job(buildJob)

				testFrontendJob := bb.NewJob().
					Name("test-frontend").
					Desc("Run all frontend tests").
					Docker(bb.NewDocker().
						Image("node:16.16.0-buster").
						Pull(bb.DockerPullIfNotExists).
						Shell("/bin/bash")).
					Fingerprint(append(buildFingerprint,
						"find frontend/ -not -path \"*/node_modules/*\" -not -path \"frontend/public/*\" -type f | sort | xargs sha1sum")...).
					Step(bb.NewStep().
						Name("install").
						Commands(`. build/scripts/lib/node-env.sh
      	 cd frontend && yarn install --modules-folder "${NODE_PATH}"`)).
					Step(bb.NewStep().
						Name("format").
						Commands(`. build/scripts/lib/node-env.sh
                     cd frontend
                     if [ "$(prettier --list-different 'src/**/*.ts*')" != "" ]; then
                       echo "Looks like you forgot to run 'yarn format' before committing"
                       exit 1
                     fi `)).
					Step(bb.NewStep().
						Name("test").
						Commands(`. build/scripts/lib/node-env.sh
                     cd frontend && yarn test`))
				workflow.Job(testFrontendJob)

				goUnitSqliteJob := addBuildBeaverTestingJob(workflow, "go-unit-sqlite", "Run all Go unit tests using sqlite", false, "-short")
				goIntegrationSqliteJob := addBuildBeaverTestingJob(workflow, "go-integration-sqlite", "Run all Go integration tests using sqlite", false, "-run Integration")

				goUnitPostgresJob := addBuildBeaverTestingJob(workflow, "go-unit-postgres", "Run all Go unit tests using postgres", true, "-short")
				goIntegrationPostgresJob := addBuildBeaverTestingJob(workflow, "go-integration-postgres", "Run all Go integration tests using postgres", true, "-run Integration")

				checkJobCount(t, 2)

				_, err = workflow.Submit(false)
				require.NoError(t, err)
				checkJobCount(t, 9)

				// Check the jobs went into the database correctly
				ctx := context.Background()
				buildFromServer, err := t.App.QueueService.ReadQueuedBuild(ctx, nil, t.BuildID)
				require.NoError(t, err)
				require.Equal(t, 9, len(buildFromServer.Jobs))
				jobMap := makeJobMap(buildFromServer.Jobs)

				baseJobFromServer := jobMap[jobReferenceToFQN(baseJob.GetReference())]
				require.NotNil(t, baseJobFromServer, "Could not find 'base' job on the server")
				checkJob(t.T, baseJob, baseJobFromServer)

				generateJobFromServer := jobMap[jobReferenceToFQN(generateJob.GetReference())]
				require.NotNil(t, generateJobFromServer, "Could not find 'generate' job on the server")
				checkJob(t.T, generateJob, generateJobFromServer)

				buildJobFromServer := jobMap[jobReferenceToFQN(buildJob.GetReference())]
				require.NotNil(t, buildJobFromServer, "Could not find 'build' job on the server")
				checkJob(t.T, buildJob, buildJobFromServer)

				testFrontendJobFromServer := jobMap[jobReferenceToFQN(testFrontendJob.GetReference())]
				require.NotNil(t, testFrontendJobFromServer, "Could not find 'test frontend' job on the server")
				checkJob(t.T, testFrontendJob, testFrontendJobFromServer)

				goUnitSqliteJobFromServer := jobMap[jobReferenceToFQN(goUnitSqliteJob.GetReference())]
				require.NotNil(t, goUnitSqliteJobFromServer, "Could not find 'go unit sqlite' test job on the server")
				checkJob(t.T, goUnitSqliteJob, goUnitSqliteJobFromServer)

				goIntegrationSqliteJobFromServer := jobMap[jobReferenceToFQN(goIntegrationSqliteJob.GetReference())]
				require.NotNil(t, goIntegrationSqliteJobFromServer, "Could not find 'go integration sqlite' test job on the server")
				checkJob(t.T, goIntegrationSqliteJob, goIntegrationSqliteJobFromServer)

				goUnitPostgresJobFromServer := jobMap[jobReferenceToFQN(goUnitPostgresJob.GetReference())]
				require.NotNil(t, goUnitPostgresJobFromServer, "Could not find 'go unit postgres' test job on the server")
				checkJob(t.T, goUnitPostgresJob, goUnitPostgresJobFromServer)

				goIntegrationPostgresJobFromServer := jobMap[jobReferenceToFQN(goIntegrationPostgresJob.GetReference())]
				require.NotNil(t, goIntegrationPostgresJobFromServer, "Could not find 'go integration postgres' test job on the server")
				checkJob(t.T, goIntegrationPostgresJob, goIntegrationPostgresJobFromServer)

				return nil
			},
		))
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK")
	return build
}

func addBuildBeaverTestingJob(workflow *bb.Workflow, jobName bb.ResourceName, description string, usePostgres bool, goTestArgs string) *bb.Job {
	testCommand := fmt.Sprintf(`. build/scripts/lib/go-env.sh
                                       cd backend && go test -v -count=1 -mod=vendor %s ./...`, goTestArgs)
	job := bb.NewJob().
		Name(jobName).
		Desc(description).
		Depends("test.base", "test.generate.artifacts").
		Type(bb.JobTypeDocker).
		Docker(bb.NewDocker().
			Image("buildbeaver/go-builder:latest").
			Pull(bb.DockerPullNever).
			Shell("/bin/bash")).
		Step(bb.NewStep().
			Name("run-tests").
			Commands(testCommand))

	if usePostgres {
		job.Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("postgres"))
		job.Env(bb.NewEnv().
			Name("TEST_CONNECTION_STRING").
			Value("postgresql://buildbeaver:password@postgres:5432/?sslmode=disable"))
		job.Service(bb.NewService().
			Name("postgres").
			Image("postgres:14").
			Env(bb.NewEnv().
				Name("POSTGRES_USER").
				Value("buildbeaver")).
			Env(bb.NewEnv().
				Name("POSTGRES_PASSWORD").
				Value("password")))
	} else {
		job.Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("sqlite3"))
	}

	workflow.Job(job)
	return job
}
