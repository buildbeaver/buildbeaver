import logging
import pytest
import datetime
import random
import string
from lib import constants
from lib.bb_test_controller import RepoHolder

logger = logging.getLogger(__name__)


class TestStaticBuilds:
    repo: RepoHolder

    @pytest.fixture(autouse=True, scope="class")
    def _setup_initial_test_state(self, test_controller):
        # Currently we only require the one test runner for this class,
        test_controller.get_or_deploy_runner(constants.TEST_RUNNER_ONE_LINUX)

        # And one repo in place
        date_name = datetime.datetime.today().strftime("%B-%d-%Y-%H")
        repo_name = "Automated-Repo-{date_string}".format(date_string=date_name)
        self.__class__.repo = test_controller.get_or_create_repo(repo_name)

    def test_deploy_basic_yaml(self, test_controller):
        """
        This test ensures that if we deploy a basic YAML file (see test-data/basic_yaml folder) we get a
        build running for the repo we have created
        """
        repo_name = self.repo.github_repo.name
        letters = string.ascii_lowercase
        # Create a branch in our repo that we will use for committing our test code
        branch_name = datetime.datetime.today().strftime("%B-%d-%Y-%H-%M-%S" + ''.join(random.choice(letters) for i in range(10)))
        test_controller.create_branch(repo_name, branch_name)

        # Commit the basic yaml code base
        commit_sha = test_controller.create_commit_from_test_object_folder(repo_name, branch_name, 'basic-yaml')
        _ = test_controller.create_pull_request(repo_name, branch_name)

        # Now query the BB Api to assert a build is created for our commit from above
        created_build = test_controller.assert_build_exists(repo_name, commit_sha, retry_interval=20)
        assert created_build is not None