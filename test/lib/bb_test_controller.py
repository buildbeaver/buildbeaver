import tempfile
import time
from distutils.dir_util import copy_tree
from os.path import join
from typing import Dict

from git import Repo
from github import Github
from github.GithubException import UnknownObjectException
from github.Repository import Repository

from lib.bb_api_client import BBAPIClient
from lib.server_manager import ServerDefinition
from lib.runner import Runner
from lib.runner_manager import RunnerManager
import logging
import requests

logger = logging.getLogger(__name__)


class RepoHolder:
    """
    RepoHolder is a wrapper around storing all access points we need for a git repository:
    - GitHub Repository object for API access
    - Local Git checkout in a temporary directory
    - BB representation of a repo from the BB API
    """
    def __init__(self, github_repo: Repository):
        self.github_repo = github_repo
        self.git_checkout = None
        self.bb_entity = None

    def set_git_checkout(self, repo: Repo):
        self.git_checkout = repo

    def set_bb_entity(self, bb_entity):
        self.bb_entity = bb_entity


class BBTestController:

    def __init__(self, environment_name: str, api_endpoint: str, github_pat: str):
        self.api_endpoint = api_endpoint
        self.environment_name = environment_name
        self.github_pat = github_pat
        self.runner_api_endpoint = self.__get_runner_endpoint()

        self.runner_manager = self.__get_runner_manager()
        self.auth_token = self.__get_auth_token()
        self.bb_api_client = BBAPIClient(api_endpoint, self.auth_token)
        self.person_legal_entity = self.__get_current_person_legal_entity()
        self.github = Github(self.github_pat)

        self.temporary_test_directory = tempfile.TemporaryDirectory()

        self.repos: Dict[str, RepoHolder] = {}

    # region Initialization methods

    def __get_auth_token(self, timeout=240, retry_interval=10):
        """
        Retrieves an auth token, for our GitHub PAT, from the BB API for our use in authenticated calls.
        """
        retry_interval = float(retry_interval)
        timeout = int(timeout)
        timeout_start = time.time()

        token_url = "{0}/token-exchange".format(self.api_endpoint)
        logger.info("Attempting to perform token exchange with the BB API at %s", token_url)

        while time.time() < timeout_start + timeout:
            time.sleep(retry_interval)
            r = requests.post(token_url, timeout=10, json={"scm_name": "github", "token": self.github_pat})
            if r.status_code == 503:
                # We are probably still spooling up the load balances, try again after retry interval
                logger.info("-- Hit 503, sleeping for {interval} seconds before trying again".format(interval=retry_interval))
                continue

            assert r.status_code == 201, "Failed getting authentication token from token exchange with status code '{status_code}' and response '{message}'".format(
                status_code=r.status_code, message=r.text)
            j = r.json()
            logger.info("-- Finished performing token exchange with the BB API with a 201 response")
            return j["token"]

        assert False, "Failed getting authentication token from token exchange"

    def __get_current_person_legal_entity(self):
        """
        Returns the first 'person' legal entity from the BB API encountered for the user using the provided auth_token.
        """
        legal_entities_url = "{0}/legal-entities".format(self.api_endpoint)
        logger.info("Getting person legal entity for currently authed user")
        headers = {'buildbeaver-token': self.auth_token}
        r = requests.get(legal_entities_url, timeout=10, headers=headers)
        assert r.status_code == 200, "Failed to get information on the available legal entities for the current authenticated user with status_code '{status_code}".format(status_code=r.status_code)
        j = r.json()

        for entity in j["results"]:
            if entity["type"] == "person":
                return entity

        assert False, "Failed to find a person in list of legal entities available to the currently authenticated user"

    def __get_runner_endpoint(self):
        # TODO: This should definitely come from Terraform at some point

        # Environment name is <num>-e2e, we need to find <num>.e2e and replace it
        runner_api_endpoint = self.api_endpoint.replace("//app", "//runner")
        runner_api_endpoint = runner_api_endpoint.replace("/api/v1", "/")
        return runner_api_endpoint

    def _enable_repo_for_legal_entity(self, repo_name, timeout=240, retry_interval=10):
        """
        Calls out to the BB API to enable a repository for a given legal entity
        Note: This should be replaced by a Core SDK call instead of this.
        """
        legal_entity_repos_url = "{0}/legal-entities/{1}/repos".format(self.api_endpoint, self.current_legal_entity_id())
        logger.info("Getting repos for currently authed user to enable repo '%s'", repo_name)
        headers = {'buildbeaver-token': self.auth_token}

        repo_entity = None
        retry_interval = float(retry_interval)
        timeout = int(timeout)
        timeout_start = time.time()
        while time.time() < timeout_start + timeout:
            time.sleep(retry_interval)
            logger.info("-- Getting repos for currently authed user at '%s'...", legal_entity_repos_url)
            if repo_entity is not None:
                logger.info("-- Found requested repo '%s'", repo_name)
                break
            r = requests.get(legal_entity_repos_url, timeout=10, headers=headers)
            if r.status_code != 200:
                continue
            j = r.json()

            repo_results = j["results"]
            if repo_results is None:
                continue

            logger.info("-- Found '%s' repos, checking for matching repo", len(repo_results))

            for entity in repo_results:
                if entity["name"].lower() == repo_name.lower():
                    repo_entity = entity
                else:
                    logger.info("-- Found non-matching repo - %s", entity["name"])

        # Ensure we found a repo, otherwise fail whichever test has called here.
        assert repo_entity is not None, "Failed to find repo '{0}' available to the currently authenticated user".format(repo_name)

        logger.info("Found repo '%s', attempting to enable at '%s'...", repo_name, repo_entity["url"])
        self.repos.get(repo_name).set_bb_entity(repo_entity)
        if repo_entity["enabled"]:
            logger.info("Repo '%s' is already enabled, nothing to do.", repo_name)
            return
        r = requests.patch(repo_entity["url"], timeout=30, json={"enabled": True}, headers=headers)
        assert r.status_code == 200 or r.status_code == 422, "Failed to enable repo '{repo_name}' with status code '{status_code}' and message '{message}'".format(repo_name=repo_name, status_code=r.status_code, message=r.text)

    def __get_runner_manager(self):
        """
        Handles creating our Runner Manager for looking after our collection of Runners.
        """
        logger.info("Creating Runner Manager")
        return RunnerManager(self.environment_name)

    # endregion

    # region Runner methods

    # TODO: Parallelize runner deployments in the future as we don't need to wait for full EC2 creation in series.
    def __deploy_runner(self, runner_name, server_def: ServerDefinition) -> Runner:
        logger.info("Creating remote E2E Runner '%s'...", runner_name)
        runner = self.runner_manager.deploy_runner(runner_name, server_def, self.api_endpoint, self.runner_api_endpoint)
        runner.configure()

        logger.info("Registering runner against legal entity %s", self.current_legal_entity_id())
        self.bb_api_client.register_runner(self.current_legal_entity_id(), runner_name, runner.get_runner_cert())
        logger.info("Remote E2E Runner '%s' created with public ip '%s'", runner_name, runner.public_ip_address())
        return runner

    def get_or_deploy_runner(self, runner_details: Dict) -> Runner:
        """
        Returns an existing runner by name if it is already deployed, else deploys within our AWS infrastructure.
        """
        logger.info("Getting or deploying runner '%s'...", runner_details['name'])
        existing_runner = self.runner_manager.get_runner(runner_details['name'])
        if existing_runner is not None:
            logger.info("Existing runner found, returning")
            return existing_runner
        return self.__deploy_runner(runner_details['name'], ServerDefinition(runner_details['platform'], runner_details['variant'], runner_details['architecture']))

    # endregion

    # region GitHub methods

    def _get_repo_temporary_path(self, repo_name):
        return join(self.temporary_test_directory.name, 'repos', repo_name)

    def get_or_create_repo(self, repo_name, enable_repo=True) -> RepoHolder:
        """
        Creates (or gets if it already exists) a GitHub repository, populating a RepoHolder that is returned to the caller
        """
        logger.info("Creating repo '%s'...", repo_name)
        repo = self._get_or_create_github_repo(repo_name)
        logger.info("Finished creating repo '%s'.", repo_name)

        # Ensure we have the repo checked out locally, so we can do file operations on it
        repo_path = self._get_repo_temporary_path(repo_name)
        # TODO: username needs to come from secure storage
        username = 'buildbeaver-e2e1'
        remote = f"https://{username}:{self.github_pat}@github.com/{username}/{repo_name}.git"
        git_repo = Repo.clone_from(remote, repo_path)
        repo.set_git_checkout(git_repo)

        # Force a commit on a dummy branch to see if this gets the repo to show up
        self.create_commit_from_test_object_folder(repo_name, "dummy_branch_for_bb", "repo-init")

        if enable_repo:
            self._enable_repo_for_legal_entity(repo_name)
        return repo

    def _get_or_create_github_repo(self, repo_name) -> RepoHolder:
        """
        Creates a GitHub repository that is cloned locally, populating the locally stored RepoHolder
        :param repo_name: The name of the repository to create.
        """

        stored_repo = self.repos.get(repo_name)
        if stored_repo is not None:
            return stored_repo

        # Operate under the user context
        user = self.github.get_user()

        # check if the repo already exists
        try:
            existing_repo = user.get_repo(repo_name)
            logger.info("Repo '%s' already exists on GitHub, will use this", repo_name)
            self.repos[repo_name] = RepoHolder(github_repo=existing_repo)
            return self.repos[repo_name]
        except UnknownObjectException:
            # Thrown if the repo doesn't exist in GitHub
            pass

        # create the repo if not found
        repo = user.create_repo(repo_name, private=True, auto_init=True)
        self.repos[repo_name] = RepoHolder(github_repo=repo)

        return self.repos[repo_name]

    def create_branch(self, repo_name, branch_name) -> str:
        """
        Creates a branch of name branch_name in the repo repo_name returning the SHA of the HEAD of the branch
        """
        github_repo = self.repos.get(repo_name).github_repo
        head_sha = github_repo.get_branch(github_repo.default_branch).commit.sha
        _ = github_repo.create_git_ref(ref=f"refs/heads/" + branch_name, sha=head_sha)
        branch_sha = github_repo.get_branch(branch_name).commit.sha

        return branch_sha

    def create_commit_from_test_object_folder(self, repo_name, branch_name, folder_path) -> str:
        """
        Creates a commit using the contents of the folder_path in a branch (branch_name) within a repo given by repo_name
        :param repo_name: The name of the repository to create a commit under
        :param branch_name: The name of the branch to create the commit under
        :param folder_path: The name of the folder to add in the commit. Note this is a folder under test-data (do not include this in your path)
        :return: The SHA of the created commit
        """
        repo = self.repos.get(repo_name)

        assert repo is not None, "Trying to access local repo '{repo_name}' that hasn't been created yet".format(repo_name=repo_name)

        logger.info("Adding '%s' test folder to repo '%s' under branch '%s'", folder_path, repo_name, branch_name)
        # Git wrapper Repo
        git_repo = repo.git_checkout
        temp_repo_path = self._get_repo_temporary_path(repo_name)

        # Create the branch
        original_branch = git_repo.active_branch
        git_repo.git.checkout("-b", branch_name)

        # Copy test object folder contents
        logger.info("-- Copying files from '%s' folder to local temp checkout", folder_path)
        copy_tree(self.get_test_object_path(folder_path), temp_repo_path)

        # Add the files to git
        logger.info("-- Git adding all files from '%s' local temp checkout", folder_path)
        logger.info(git_repo.git_dir)
        logger.info(git_repo.working_dir)
        git_repo.git.add(git_repo.working_dir)
        logger.info("-- commit message set")
        commit = git_repo.index.commit("Test commit message")
        logger.info("-- Git pushing changes to remote")
        git_repo.git.push('--set-upstream', git_repo.remote().name, branch_name)
        logger.info("-- Finished adding '%s' test folder to repo '%s' under branch '%s'", folder_path, repo_name, branch_name)

        # If we need to check out the original branch
        original_branch.checkout()

        return commit.hexsha

    def create_pull_request(self, repo_name, branch_name):
        logger.info("Creating pull request from branch '%s'", branch_name)
        github_repo = self.repos.get(repo_name).github_repo
        return github_repo.create_pull(title="Automated PR title", body="Automated PR body", base=github_repo.default_branch, head=branch_name)

    # endregion

    # region Assertion Helpers

    def assert_build_exists(self, repo_name, commit_sha, timeout=60, retry_interval=5):
        """
        Calls out to the BB API to check and return a BB build entity by its commit SHA.
        Note: This should be replaced by a Core SDK call instead of this
        """
        repo = self.repos.get(repo_name)

        assert repo is not None, "Trying to access local repo '{repo_name}' that hasn't been created yet".format(repo_name=repo_name)

        # Grab the builds_url for the repo
        bb_entity = repo.bb_entity
        builds_url = bb_entity["builds_url"]
        logger.info("Checking for build for commit '%s' using builds endpoint '%s'", commit_sha, builds_url)
        headers = {'buildbeaver-token': self.auth_token}

        build_entity = None
        retry_interval = float(retry_interval)
        timeout = int(timeout)
        timeout_start = time.time()
        while time.time() < timeout_start + timeout:
            logger.info("Getting builds for currently authed user...")
            time.sleep(retry_interval)
            if build_entity is not None:
                logger.info("-- Found matching build for commit - %s", commit_sha)
                break
            r = requests.get(builds_url, timeout=10, headers=headers)
            if r.status_code != 200:
                continue
            j = r.json()

            # Skip for now if we haven't seen any builds for the repo
            build_results = j["results"]
            if build_results is None:
                continue

            # Otherwise see if we can find the build by its commit sha
            for build in build_results:
                if build["commit"]["sha"].lower() == commit_sha.lower():
                    build_entity = build
                else:
                    logger.info("-- Encountered build that doesn't match SHA - %s", build["commit"]["sha"])

        assert build_entity is not None, "Builds endpoint did not return a build for commit '{sha}' in time".format(sha=commit_sha)

        return build_entity

    # endregion

    # region General utility / shortcut methods

    def current_legal_entity_id(self):
        return self.person_legal_entity["id"]

    # endregion

    def get_test_object_path(self, path):
        return "test-data/{0}".format(path)

    def _teardown_repos(self):
        for repo_name, repo in self.repos.items():
            logger.info("Deleting repo '%s'", repo_name)
            repo.github_repo.delete()

    def teardown(self):
        logger.info("Tearing down BB Test Controller")
        self.runner_manager.destroy_all_runners()
        self.temporary_test_directory.cleanup()
        self._teardown_repos()