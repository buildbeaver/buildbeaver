import logging

from lib.ansible import exec_playbook
from lib.server import Server

logger = logging.getLogger(__name__)


class Runner(Server):
    def __init__(self, server: Server, api_endpoint: str, runner_api_endpoint: str):
        super().__init__(server.name, server.platform, server.ec2_instance, server.username, server.connection_type,
                         server.connection_auth)
        self.client = server.client
        self.server_api_endpoint = api_endpoint
        self.runner_api_endpoint = runner_api_endpoint

    def configure(self):
        logger.info("Configuring runner...")
        vars = ["runner_env_runner_api_endpoints={}".format(self.runner_api_endpoint),
                "runner_env_dynamic_api_endpoint={}".format(self.server_api_endpoint)]
        exec_playbook(self, "buildbeaver-runner", "buildbeaver-runners", vars)

    def get_runner_cert(self):
        match self.platform:
            case 'linux':
                stdout, _, _ = self.exec('cat /var/lib/buildbeaver/runners/default/runner-client-cert.pem')
            case _:
                raise Exception("Unsupported platform")
        logger.info("runner (ip:%s) public key is: %s", self.public_ip_address(), stdout)

        if not stdout:
            raise Exception("Unable to load runner client cert")

        return stdout
