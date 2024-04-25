package main

import (
	"fmt"

	"github.com/buildbeaver/sdk/dynamic/bb"
)

var useNodeJSDockerImage = bb.NewDocker().
	Image("node:16.16.0-buster").
	Pull(bb.DockerPullIfNotExists).
	Shell("/bin/bash")

var goJobFingerprint = []string{
	`find build/scripts -type f | sort | xargs sha1sum`,
	`find backend/ -name '*.go' -not -path "*/vendor/*" -type f | sort | xargs sha1sum`,
	`sha1sum backend/go.mod`,
	`sha1sum backend/go.sum`,
}

var frontendJobFingerprint = []string{
	`find build/scripts -type f | sort | xargs sha1sum`,
	`find frontend/ -not -path "*/node_modules/*" -not -path "frontend/public/*" -type f | sort | xargs sha1sum`,
}

func main() {
	bb.Workflows(
		bb.NewWorkflow().Name("base").Handler(submitBaseJobs),
		bb.NewWorkflow().Name("generate").Handler(submitGenerateJobs),
		bb.NewWorkflow().Name("unit-test").Handler(submitUnitTestJobs),
		bb.NewWorkflow().Name("integration-test").Handler(submitIntegrationTestJobs),
		bb.NewWorkflow().Name("build").Handler(submitBuildJobs),
		bb.NewWorkflow().Name("frontend").Handler(submitFrontEndJobs),
		bb.NewWorkflow().Name("openapi").Handler(submitOpenAPIJobs),
	)
}

func submitBaseJobs(w *bb.Workflow) error {
	w.Job(bb.NewJob().
		Name("base-images").
		Desc("Builds the base image needed for the build pipeline").
		Docker(bb.NewDocker().
			Image("docker:20.10").
			Pull("if-not-exists")).
		Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
		// Only require AWS credentials if we are pushing image to ECR registry
		//Env(bb.NewEnv().
		//	Name("AWS_ACCESS_KEY_ID").
		//	ValueFromSecret("AWS_ACCESS_KEY_ID")).
		//Env(bb.NewEnv().
		//	Name("AWS_SECRET_ACCESS_KEY").
		//	ValueFromSecret("AWS_SECRET_ACCESS_KEY")).
		Step(bb.NewStep().
			Name("go-builder").
			Commands(
				"apk add bash git aws-cli",
				"git config --global --add safe.directory $(pwd)",
				// Use -p option to push docker image to registry, when using multiple runners
				//"./build/scripts/build-docker.sh -t $BB_JOB_FINGERPRINT -p go-builder")).
				"./build/scripts/build-docker.sh -t $BB_JOB_FINGERPRINT go-builder")).
		OnSuccess(func(event *bb.JobStatusChangedEvent) {
			// Calculate the docker image name from this job's fingerprint
			jGraph := w.GetBuild().MustGetJobGraph(event.JobID)

			// Make a Docker Config for later steps to use to pull the docker image just built
			// This config assumes the build is running using a single runner for all jobs (e.g. when running
			// using the= bb command line tool), so the docker image can just be local
			goDockerConfig := bb.NewDocker().
				Image(fmt.Sprintf("go-builder:%s", *jGraph.Job.Fingerprint)).
				Pull(bb.DockerPullNever).
				Shell("/bin/bash")

			// If pushing image to ECR registry then make a docker config that authenticates to AWS and pulls from ECR
			//goDockerConfig := bb.NewDocker().
			//	Image(fmt.Sprintf("fill-this-out.dkr.ecr.us-west-2.amazonaws.com/go-builder:%s", *jGraph.Job.Fingerprint)).
			//	Pull(bb.DockerPullIfNotExists).
			//	Shell("/bin/bash").
			//	AWSAuth(bb.NewAWSAuth().
			//		Region("us-west-2").
			//		AccessKeyIDFromSecret("AWS_ACCESS_KEY_ID").
			//		SecretAccessKeyFromSecret("AWS_SECRET_ACCESS_KEY"))

			w.SetOutput("go-docker-config", goDockerConfig)
		}))
	w.MustSubmit()

	goDockerConfig := w.MustWaitForOutput("base", "go-docker-config").(*bb.DockerConfig)

	w.Job(bb.NewJob().
		Name("backend-preflight").
		Desc("Performs preflight checks on all backend code").
		Depends("base.base-images").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Step(bb.NewStep().
			Name("lint").
			Commands("./build/scripts/ci/backend-preflight-lint.sh")))

	return nil
}

func submitGenerateJobs(w *bb.Workflow) error {
	goDockerConfig := w.MustWaitForOutput("base", "go-docker-config").(*bb.DockerConfig)

	w.Job(bb.NewJob().
		Name("backend-generate").
		Desc("Generates all backend code (wire files, protobufs etc.)").
		Depends("base.backend-preflight").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Step(bb.NewStep().
			Name("wire").
			Commands("./build/scripts/ci/backend-generate-wire.sh")).
		Artifact(bb.NewArtifact().
			Name("wire").
			Paths("backend/*/app/wire_gen.go", "backend/*/app/*/wire_gen.go")).
		Artifact(bb.NewArtifact().
			Name("grpc").
			Paths("backend/api/grpc/*.pb.go")))

	return nil
}

func submitUnitTestJobs(w *bb.Workflow) error {
	goDockerConfig := w.MustWaitForOutput("base", "go-docker-config").(*bb.DockerConfig)

	w.Job(bb.NewJob().
		Name("backend-sqlite").
		Desc("Runs all backend unit tests on top of SQLite").
		Depends("generate.backend-generate.artifacts").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("sqlite3")).
		Step(bb.NewStep().
			Name("test").
			Commands(
				". build/scripts/lib/go-env.sh",
				"cd backend && go test -v -count=1 -mod=vendor -short ./...")))

	w.Job(bb.NewJob().
		Name("backend-postgres").
		Desc("Runs all backend unit tests on top of Postgres").
		Depends("generate.backend-generate.artifacts").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("postgres")).
		Env(bb.NewEnv().
			Name("TEST_CONNECTION_STRING").
			Value("postgres://buildbeaver:password@postgres:5432/?sslmode=disable")).
		Step(bb.NewStep().
			Name("wait-for-postgres").
			Commands(
				`for i in {1..15}; do pg_isready -d "${TEST_CONNECTION_STRING}" && break || sleep 1; done`)).
		Step(bb.NewStep().
			Name("test").
			Commands(
				". build/scripts/lib/go-env.sh",
				"cd backend && go test -v -count=1 -mod=vendor -short ./...")).
		Service(bb.NewService().
			Name("postgres").
			Image("postgres:14").
			Env(bb.NewEnv().
				Name("POSTGRES_USER").
				Value("buildbeaver")).
			Env(bb.NewEnv().
				Name("POSTGRES_PASSWORD").
				Value("password"))))
	return nil
}

func submitIntegrationTestJobs(w *bb.Workflow) error {
	goDockerConfig := w.MustWaitForOutput("base", "go-docker-config").(*bb.DockerConfig)

	w.Job(bb.NewJob().
		Name("backend-sqlite").
		Desc("Runs all backend integration tests on top of SQLite").
		Depends("generate.backend-generate.artifacts").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("sqlite3")).
		Step(bb.NewStep().
			Name("test").
			Commands(
				". build/scripts/lib/go-env.sh",
				"cd backend && go test -v -count=1 -mod=vendor -run Integration ./...")))

	w.Job(bb.NewJob().
		Name("backend-postgres").
		Desc("Runs all backend integration tests on top of Postgres").
		Depends("generate.backend-generate.artifacts").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Env(bb.NewEnv().
			Name("TEST_DB_DRIVER").
			Value("postgres")).
		Env(bb.NewEnv().
			Name("TEST_CONNECTION_STRING").
			Value("postgres://buildbeaver:password@postgres:5432/?sslmode=disable")).
		Step(bb.NewStep().
			Name("wait-for-postgres").
			Commands(
				`for i in {1..15}; do pg_isready -d "${TEST_CONNECTION_STRING}" && break || sleep 1; done`)).
		Step(bb.NewStep().
			Name("test").
			Commands(
				". build/scripts/lib/go-env.sh",
				"cd backend && go test -v -count=1 -mod=vendor -run Integration ./...")).
		Service(bb.NewService().
			Name("postgres").
			Image("postgres:14").
			Env(bb.NewEnv().
				Name("POSTGRES_USER").
				Value("buildbeaver")).
			Env(bb.NewEnv().
				Name("POSTGRES_PASSWORD").
				Value("password"))))
	return nil
}

func submitBuildJobs(w *bb.Workflow) error {
	goDockerConfig := w.MustWaitForOutput("base", "go-docker-config").(*bb.DockerConfig)

	w.Job(bb.NewJob().
		Name("backend-build").
		Desc("Builds all backend binaries").
		Depends(
			"generate.backend-generate.artifacts",
			"unit-test.backend-sqlite",
			"unit-test.backend-postgres",
			"integration-test.backend-sqlite",
			"integration-test.backend-postgres").
		Docker(goDockerConfig).
		Fingerprint(goJobFingerprint...).
		Step(bb.NewStep().
			Name("go").
			Commands("./build/scripts/ci/backend-build.sh")).
		Artifact(bb.NewArtifact().
			Name("go-binaries").
			Paths("build/output/go/bin/*")))
	return nil
}

func submitFrontEndJobs(w *bb.Workflow) error {
	w.Job(bb.NewJob().
		Name("preflight").
		Desc("Performs preflight checks on all frontend code").
		Docker(useNodeJSDockerImage).
		Fingerprint(frontendJobFingerprint...).
		Step(bb.NewStep().
			Name("lint").
			Commands(
				"npm install --location=global \"prettier@2.8.4\"",
				"./build/scripts/ci/frontend-preflight-lint.sh")))

	w.Job(bb.NewJob().
		Name("unit-test").
		Desc("Runs all frontend unit tests").
		Depends("frontend.preflight").
		Docker(useNodeJSDockerImage).
		Fingerprint(frontendJobFingerprint...).
		Step(bb.NewStep().
			Name("install").
			Commands(
				". build/scripts/lib/node-env.sh",
				"cd frontend && yarn install")).
		Step(bb.NewStep().
			Name("test").
			Commands(
				". build/scripts/lib/node-env.sh",
				"cd frontend && yarn test")))

	w.Job(bb.NewJob().
		Name("build").
		Desc("Builds all frontend code").
		Depends("frontend.unit-test").
		Docker(useNodeJSDockerImage).
		Fingerprint(frontendJobFingerprint...).
		Step(bb.NewStep().
			Name("install").
			Commands(
				". build/scripts/lib/node-env.sh",
				"cd frontend && yarn install")).
		Step(bb.NewStep().
			Name("build").
			Commands(
				". build/scripts/lib/node-env.sh",
				"cd frontend && yarn build")))
	return nil
}

func submitOpenAPIJobs(w *bb.Workflow) error {
	w.Job(bb.NewJob().
		Name("generate-code").
		Desc("Generates API Clients from our OpenAPI specs").
		Depends("base.backend-preflight").
		Docker(bb.NewDocker().
			Image("openapitools/openapi-generator-cli:v6.5.0").
			Pull(bb.DockerPullIfNotExists).
			Shell("/bin/bash")).
		Fingerprint(
			"sha1sum backend/server/api/rest/openapi/dynamic-openapi.yaml",
			"sha1sum backend/server/api/rest/openapi/core-openapi.yaml").
		Step(bb.NewStep().
			Name("core-go").
			Commands("./build/scripts/ci/backend-openapi.sh core-openapi.yaml go sdk/core/go/client")).
		Step(bb.NewStep().
			Name("core-python").
			Commands("./build/scripts/ci/backend-openapi.sh core-openapi.yaml python sdk/core/python/client")).
		Step(bb.NewStep().
			Name("dynamic-go").
			Commands("BB_DEBUG=1 ./build/scripts/ci/backend-openapi.sh dynamic-openapi.yaml go sdk/dynamic/go/client")))
	return nil
}
