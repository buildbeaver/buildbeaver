import logging
import shutil
import sys

import boto3
import pytest
from python_dynamodb_lock.python_dynamodb_lock import DynamoDBLockClient

from lib import util
from lib.bb_cli_test_controller import BBCLITestController
from lib.bb_test_controller import BBTestController
from lib.runner_manager import RunnerManager

logging.basicConfig(stream=sys.stdout, level=logging.INFO)
logger = logging.getLogger()


# region Command line arguments
def pytest_addoption(parser):
    """
    Command line argument parser
    """
    parser.addoption('--environment-id', action='store', default='1', help='The e2e environment id')

    parser.addoption('--skip-teardown',
                     action='store_true',
                     default=False,
                     help="Set this flag to not teardown the environment. YOU MUST MANUALLY TEARDOWN THE ENVIRONMENT IF YOU PROVIDE THIS ARGUMENT")


@pytest.fixture(scope="session")
def environment_name(request) -> str:
    """
    Returns the environment name that is passed into the tests.
    """
    environment_id = request.config.getoption('--environment-id')
    return "{0}-e2e".format(environment_id)


@pytest.fixture(scope="session")
def skip_teardown(request) -> bool:
    """
    Returns if teardown should be skipped.

    This is useful if you are tweaking your tests and do not want to run a full environment creation / deletion for a
    one-minute check.
    """
    return request.config.getoption('--skip-teardown')


# endregion

@pytest.fixture(scope="session")
def test_cli_controller() -> BBCLITestController:
    test_controller = BBCLITestController()
    try:
        yield test_controller
    finally:
        test_controller.teardown()


@pytest.fixture(scope="session")
def get_environment_name_with_lock(environment_name):
    """
    Handles getting the environment lock within our DynamoDB instance, returning the name of the current environment
    """
    logger.info("Getting our dynamodb instance...")
    logging.getLogger('python_dynamodb_lock').setLevel(logging.WARNING)
    # Create our dynamodb instance
    ddb_client = boto3.client('dynamodb')
    try:
        DynamoDBLockClient.create_dynamodb_table(ddb_client, table_name='e2e-test-environment-locks')
        logger.debug("Created lock table")
    except ddb_client.exceptions.ResourceInUseException:
        logger.debug("Lock table already exists")

    dynamodb_resource = boto3.resource('dynamodb')
    lock_client = DynamoDBLockClient(dynamodb_resource, table_name='e2e-test-environment-locks')

    logger.info("Taking environment lock...")
    lock = lock_client.acquire_lock(environment_name)
    yield environment_name
    logger.info("Releasing environment lock...")
    lock.release()
    lock_client.close()
    logger.info("Environment lock released.")


@pytest.fixture(scope="session")
def validate_pre_requisites():
    """
    Utility fixture to ensure we have all the required pre-requisites installed before running our E2E tests.

    This is useful to ensure we are not missing an application that we only call once much later in the process
    """
    required_execs = ['terraform', 'ansible-galaxy', 'ansible-playbook', 'docker', 'yarn']

    for exe in required_execs:
        if not cmd_exists(exe):
            raise Exception('{0} required but not found'.format(exe))


@pytest.fixture(scope="session")
def deploy_server_infra(environment_name, skip_teardown, validate_pre_requisites):
    """
    Deploys the full server infrastructure we need within AWS, yielding the url to the main app api.

    Note: Do not call this directly, instead use test_controller which provides a controller for our tests
    """
    try:
        logger.info("Deploying server infrastructure...")
        exit_code = util.run_command(["../build/scripts/deploy-infra.sh", environment_name])
        if exit_code != 0:
            raise Exception("Failed to deploy infrastructure: {:n}".format(exit_code))
        logger.info("Deployed server infrastructure.")
        logger.info("Deploying BB backend...")
        exit_code = util.run_command(["../build/scripts/deploy-backend.sh", environment_name])
        if exit_code != 0:
            raise Exception("Failed to deploy backend: {:n}".format(exit_code))
        logger.info("Deployed BB backend.")
        logger.info("Deploying BB frontend...")
        exit_code = util.run_command(["../build/scripts/deploy-frontend.sh", environment_name])
        if exit_code != 0:
            raise Exception("Failed to deploy frontend: {:n}".format(exit_code))
        logger.info("Deployed BB frontend.")
    except Exception as exception:
        logger.error("Exception hit trying to deploy server infrastructure, will attempt to destroy now")
        destroy_server_infra(environment_name, skip_teardown)
        # Still raise to ensure any wrapped fixtures run through their teardown here
        raise exception

    # TODO: At some point we should get this through terraform instead of manually constructing it for our single environment.
    api_endpoint = "https://app1.e2e.changeme.com/api/v1".format(environment_name)
    logger.info("Finished deploying BB to '%s'.", api_endpoint)
    yield api_endpoint
    # Note: We are calling this in multiple locations to ensure we are cleaning up the remote server infrastructure
    destroy_server_infra(environment_name, skip_teardown)


@pytest.fixture(scope="session")
def destroy_all_remote_infrastructure(environment_name):
    """
    NOTE: This is a highly destructive fixture that should only be called if you are sure you want to delete all:
    - infrastructure
    - runners
    """
    logger.info("Destroying all runners")
    runner_manager = RunnerManager(environment_name)
    runner_manager.destroy_all_runners()
    logger.info("Destroying server infra")
    destroy_server_infra(environment_name, False)


@pytest.fixture(scope="session")
def test_controller(get_environment_name_with_lock, skip_teardown, deploy_server_infra) -> BBTestController:
    """
    Deploys our full E2E infrastructure to AWS, returning a BBTestController that can be used within tests
    """
    test_controller = BBTestController(get_environment_name_with_lock, deploy_server_infra,
                                        remote_github_pat(get_environment_name_with_lock))

    yield test_controller

    if skip_teardown:
        logger.debug("SKIP TEARDOWN ENABLED - YOU MUST TEARDOWN THE ENVIRONMENT YOURSELF")
        return

    test_controller.teardown()


def destroy_server_infra(environment_name, skip_teardown):
    """
    Destroys the full server infrastructure within AWS
    """
    if skip_teardown:
        logger.debug("SKIP TEARDOWN ENABLED - YOU MUST TEARDOWN THE ENVIRONMENT YOURSELF")
        return

    logger.info("Destroying server infrastructure...")
    exit_code = util.run_command(["../build/scripts/destroy-infra.sh", environment_name])
    if exit_code != 0:
        raise Exception("Failed to destroy infrastructure: {:n}".format(exit_code))


def remote_github_pat(environment_name):
    """
    Returns the GitHub PAT for the given environment from SSM
    """
    logger.info("Loading GitHub PAT from SSM...")
    ssm = boto3.client('ssm')
    pat_name = "/buildbeaver-{0}/github_account_1_pat".format(environment_name)
    parameter = ssm.get_parameter(Name=pat_name, WithDecryption=True)
    if not parameter:
        raise Exception("Unable to load remote GitHub PAT")
    logger.info("GitHub PAT loaded from SSM.")
    return parameter['Parameter']['Value']


def cmd_exists(cmd) -> bool:
    return shutil.which(cmd) is not None
