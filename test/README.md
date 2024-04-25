# E2E test framework

This folder contains our E2E test framework that allows us to run full E2E tests against a full setup environment 
across multiple operating systems.

IMPORTANT NOTE: AS PART OF THE E2E PROCESS WE RUN TERRAFORM TO CREATE / DESTROY THE E2E INFRASTRUCTURE. ENSURE YOU NEVER
CANCEL THE TESTS DURING THIS PROCESS AS IT MAY LEAVE TERRAFORM STATE IN AN INVALID STATE WHICH IS VERY HARD TO GET OUT OF.

## Requirements

To be able to run the E2E tests we have an assumption in place that you have built the following:
- BuildBeaver packages (`make build` -> `cp $GOPATH/bin/bb* build/output/go/bin`)
- BuildBeaver Server image (within the build/scripts directory -> `./build-docker.sh -p bb-server`)

## Running via Docker
It's recommended to build and use the e2e container within the [../build/docker/e2e](../build/docker/e2e) folder as it 
comes pre-installed with the required libraries. 

You will need to have set your AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables for AWS access, and these 
commands should be run from the build folder.

Build E2E Docker image:
```
docker build -t bb-e2e-runner:0.0.1 -f docker/e2e/Dockerfile .
```

Run tests:
```
docker run -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY -e AWS_DEFAULT_REGION=us-west-2 \
 -v ~/dev/go/buildbeaver:/development/buildbeaver:rw --rm -it **bb**-e2e-runner:0.0.1 \ 
 -v /var/run/docker.sock:/var/run/docker.sock \
 "test_static_builds.py::TestStaticBuilds::test_deploy_basic_yaml" 
```
Note: The final cmd passed into docker run is passed directly into pytest and should be the command line arguments for the tests you want to run.
See [Manually running pytest via CLI](#manually-running-pytest-via-cli) for more information on pytest CLI running.


If you want to see output from git commands that are run then add `-e GIT_PYTHON_TRACE=1` to your docker run command

## Test fixtures

Our test fixtures are set up as Test classes (see [test_static_builds.py](./test_static_builds.py) for an example) that 
use our [BBTestController object](./lib/bb_test_controller.py) as the mechanism for setting up / asserting that test 
states are as expected.  

Within our [BBTestController object](./lib/bb_test_controller.py) we hold a [BBAPIClient object](./lib/bb_test_controller.py) 
which provides a way of directly accessing the following for each GitHub Repository created:
- GitHub API object for performing direct API calls
- Git cli wrapper object checked out to a local temporary directory
- BB API response for a repository (note this isn't automatically updated, only retrieved on initial repo creation once it appears in the BB API)

It is recommended to look at the static test within [test_static_builds.py](./test_static_builds.py) to see how we approach 
creating repos / commits / verifying that a build is created in BB.

## PyTest configuration  

We have the following command line arguments that can be used to configure the E2E run:  
- *--environment-id* - The ID (int) of the test environment we are going to be running E2E tests against. Defaults to 1.
- *--skip-teardown* - Flag to skip tearing down any infrastructure / GitHub objects. Useful if you need to debug the state of the tests after they run.
  - Note: YOU MUST DELETE THE INFRASTRUCTURE MANUALLY AT THIS POINT VIA UNCOMMENTING THE *TestDestroyEverythingRemote* class and running it.

It is recommended that you provide the command line arguments in quotes with the (optional) name of the test to run, so they do not get picked up as docker arguments.

## Manually running pytest via CLI

See https://docs.pytest.org/en/7.1.x/how-to/usage.html for CLI options if you are not using an IDE.

An example of calling an individual test (*test_delete_all_infrastructure*) within an individual class (*TestRunnerRegistration*) 
within an individual file (*test_runner_registration.py*)

```
pytest test_runner_registration.py::TestRunnerRegistration::test_delete_all_infrastructure
```
