import logging
import pytest
from lib import constants
from lib.runner import Runner

logger = logging.getLogger(__name__)


class TestRunnerRegistration:
    """
    For now this Test class is an example of using our test_controller for registering runners.

    Note: The autouse function provides a way to perform any "setup" of a test class as we cannot use __init__
    in pytest.

    If you want to assign variables for this class then you need to use __class__ as per https://stackoverflow.com/a/62289822
    """
    runner_one: Runner

    @pytest.fixture(autouse=True, scope="class")
    def _setup_initial_runners(self, test_controller):
        logger.info("Deploying runner '%s' during autouse...", constants.TEST_RUNNER_ONE_LINUX['name'])
        self.__class__.runner_one = test_controller.get_or_deploy_runner(constants.TEST_RUNNER_ONE_LINUX)
        logger.info('-- Finished deploying test runner during autouse.')

    def test_runner_multiple_registrations(self, test_controller):
        # This runner should have been created for us by the class autouse where we can just grab it
        re_retrieve_runner_one = test_controller.get_or_deploy_runner(constants.TEST_RUNNER_ONE_LINUX)
        logger.info('-- Finished getting existing runner public ip: %s', re_retrieve_runner_one.public_ip_address())
        # Test that it was retrieved from the existing runner
        assert self.runner_one.public_ip_address() == re_retrieve_runner_one.public_ip_address()

        # This runner will require deployment.
        runner_two = test_controller.get_or_deploy_runner(constants.TEST_RUNNER_TWO_LINUX)
        logger.info('-- Finished deploying runner two with public ip: %s', runner_two.public_ip_address())
        # Test that we did not get the same runner as before
        assert runner_two.public_ip_address() != re_retrieve_runner_one.public_ip_address()

    def test_runner_re_register_runner(self, test_controller):
        """
        Tests that we cannot register a runner that is already registered
        """

        # Get the runner that was created with our autouse
        existing_runner = self.runner_one
        logger.info("Attempting to re-register already registered runner '%s'...", existing_runner.name)
        with pytest.raises(Exception):
            test_controller.bb_api_client.register_runner(test_controller.person_legal_entity["id"],
                                                           existing_runner.name,
                                                           existing_runner.get_runner_cert())
        logger.info("-- Finished attempting to re-register already registered runner '%s'...", existing_runner.name)
