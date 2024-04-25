# Building BuildBeaver from Source

The BuildBeaver server, Web UI and 'bb' command-line tool can be built on Linux, Mac or Windows.

# Architecture

BuildBeaver is made up of a server process (`backend/cmd/gserver`) and a build agent process (`backend/cmd/gagent`).
It is possible to run the server process with a built-in build agent in order
to simplify configuration when running on a development machine; this can be configured using command-line flags
passed to the server (e.g. in the Makefile).

A separate process provides a Web UI for the server (`frontend`), to be used by end-users of the CI system.
The Web UI interacts with the server using
a REST API. Any externally-running build agents also interact with the server using another separate REST API
which can be configured on a different port.

# Pre-requisites

## Pre-requisites for Building

If building on Windows, install the following (examples given using
[Chocolatey](https://docs.chocolatey.org/en-us/choco/setup) package manager):

1. **gcc**: Install the GCC compiler. This can be installed as part of mingw e.g. `choco install mingw`.
   Use the latest version.

2. **make**: A makefile is used for the initial build from source. Install using your package manager,
   e.g. `choco install make`

For all platforms (including Windows), install:

3. **Go 1.21** or later: 1.21 recommended but newer versions should work.
   See [Go Download and Install](https://go.dev/doc/install).

4. **Docker**: recommended for testing and required for using BuildBeaver to build itself.
   See [Get Docker](https://docs.docker.com/get-docker/).

5. **Wire**: required for code generation. Install using `go install github.com/google/wire/cmd/wire@latest`

6. **Node.js and Yarn**: required for building and running the Web UI. Install using your package manager,
   e.g. `choco install nodejs yarn`

7. **GitHub App private key**: If using the BuildBeaver server, create a GitHub app that can be installed
   by GitHub accounts or repos to provide access to the repos to build (see [GitHub App](#github-app) for details).
   Copy your GitHub app private key to`C:\github-private-key.pem` (Windows)
   or `/var/lib/buildbeaver/github-private-key.pem` (Mac or Linux).

## Local webhooks

To deliver Webhook notifications to a development machine running a BuildBeaver Server without requiring the
machine to listen on the Internet, a good option is to use [smee](https://smee.io/).

1. `npm install -g smee-client`
1. `export NODE_TLS_REJECT_UNAUTHORIZED=0; smee -u <your smee endpoint> -t http://127.0.0.1:3001/api/v1/webhooks/github`

Note that NODE_TLS_REJECT_UNAUTHORIZED=0 is only required if using a self-signed server certificate for your
development server (the default configuration for developers).
For windows replace `export` with `set`.

You will need to set up a [smee](https://smee.io/) endpoint for receiving notifications.


## GitHub App

In order for a BuildBeaver server to have access to GitHub Repos to build, it uses a GitHub App.
Each company or individual wanting to run a BuildBeaver server will need their own GitHub App.
The app must be installed by users for the repos they want to build.

See [Creating a BuildBeaver GitHub App](creating-github-app.md) for instructions on setting up your app.


# Building and Running

To build and run the various components of the BuildBeaver system on a development machine, type the
commands specified below.

Note that the command-line flags used to configure the server and runner can be changed by editing the Makefile.

## BuildBeaver Server (with internal runners)
1. `cd backend`
1. `make generate`
1. `make build`
1. `make run-server`

## Runner
1. `cd backend`
1. `make generate`
1. `make build`
1. `make run-runner`

## Frontend (Web UI)
1. `cd frontend`
1. `yarn install`
1. `yarn start`

To use the Web UI, browse to `http://localhost:3000/`. Log in using a GitHub account containing Repos you
would like to build.

## Command-line tool (bb)

1. `cd backend`
1. `make generate`
1. `make build`
1. `bb`

It is possible to use the `bb` command-line tool to build the entire BuildBeaver system;
the Makefile is for bootstrapping.

To do this, ensure Docker is running and then type `bb run` in the root directory of this repo.


# Local Postgres

The BuildBeaver server requires a database. The simplest option is to use a local file through the built-in
[SQLite](https://www.sqlite.org/) library.

Alternatively a postgres database server instance can be used:

```
docker run --name buildbeaver-postgres -e POSTGRES_USER=buildbeaver -e POSTGRES_PASSWORD=password -e POSTGRES_DB=buildbeaver -p 5432:5432 -d postgres:14
```

Run the server passing in the following flags (e.g. by editing the Makefile if you are running the server using make):

`--database_driver postgres`

`--database_connection_string "postgres://buildbeaver:password@localhost/buildbeaver?sslmode=disable"`

# Dynamic Build Configuration

When using dynamic builds, the runner must be told the dynamic API endpoint to provide to the user code that
creates new build jobs. This *Dynamic API* is used to submit new jobs to the BuildBeaver server during the build
process.

When using the `bb` command-line tool for local builds, or for developers running a local server configured
to use an internal runner, the Dynamic API endpoint configuration happens automatically.

When running an external runner the `--dynamic_api_endpoint` flag must be specified for the runner.
For production/staging this should be set to the endpoint URL of the server's Core API (which also provides
the dynamic API).

For developers running a local server and separate local runner, the following flag should be specified on
the runner (referring to the Core API on the local server):

    --dynamic_api_endpoint='http://localhost:3001'

# Running the Tests

## Go Tests

Run all tests by running `go test-mod=vendor ./...` in the `backend` directory.

### Unit Tests

Run unit tests only with `go test -mod=vendor -short ./...`

### Integration Tests

Integration tests have the word "Integration" in their name (case-sensitive), and a check for `Short()` (unit)
test mode where they skip themselves. This allows us to run unit only, integration only, or both in a
`go test` invocation.

Run integration tests only with `go test -mod=vendor -run Integration ./...`

```go
func TestRunnerAndServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	...
```

## Front-end (Web UI) tests

Run all Web UI tests by running `yarn test` in the `frontend` directory.


# OpenAPI Code Generation

The Go Dynamic SDK is in the [go-sdk repo](https://github.com/buildbeaver/go-sdk) and includes OpenAPI-generated code for the low-level client.
The source YAML for this API (dynamic-openapi.yaml) is in this repo.
The docker-based version of *openapi-generator* is used for generating the OpenAPI client code, so there's no need to
install the tool natively.

### Regenerating Go Dynamic SDK code

The Go Dynamic SDK is also in this repo as a separate Go module, under `sdk/dynamic/go`

To regenerate the Go SDK code based on the dynamic-openapi.yaml file:

```shell
bb run backend-openapi
```

This will generate code in the ` sdk/dynamic/go/client` directory.

### Dynamic SDK vendoring

The BuildBeaver Dynamic Go SDK is vendored in to the *backend* Go module for use in unit and integration tests.
It is vendored using a relative path within the file structure of the repo, so it doesn't require a separate
GitHub repo. (This is done by using a `replace` command in the go.mod file.)

If you change any part of the Go SDK, including regenerating code or changing the manually-written code, you should
re-vendor the library:

```shell
cd buildbeaver/backend
go mod vendor
```
