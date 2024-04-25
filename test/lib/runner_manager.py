import logging

import boto3
from lib.runner import Runner
from lib.server_manager import ServerDefinition
from lib.server_manager import ServerManager

logger = logging.getLogger(__name__)


class RunnerManager:

    def __init__(self, env_name):
        self.server_manager = ServerManager(subnet_name_filter='buildbeaver-{}-public-us-west-2a'.format(env_name),
                                            security_group_name_filter='buildbeaver-{}-dmz'.format(env_name))
        self.runners = {}
        self.env_name = env_name

    def deploy_runner(self, server_name, server_def: ServerDefinition, server_api_endpoint, runner_api_endpoint):
        deployed_server = self.server_manager.deploy(server_name, server_def,
                                                     tags=[{'Key': 'Env', 'Value': self.env_name}])
        deployed_runner = Runner(deployed_server, server_api_endpoint, runner_api_endpoint)
        self.runners[server_name] = deployed_runner
        return deployed_runner

    def destroy_all_runners(self):
        logger.info("Destroying all runners...")
        ec2 = boto3.resource('ec2')
        instances = ec2.instances.filter(
            Filters=[
                {'Name': 'instance-state-name', 'Values': ['running']},
                {'Name': 'tag:Env', 'Values': [self.env_name]},
            ],
        )
        if len([instance for instance in instances]) > 0:
            ec2_client = boto3.client('ec2')
            ec2_client.terminate_instances(InstanceIds=[instance.id for instance in instances])

    def get_runner(self, runner_name) -> Runner | None:
        """
        Returns a runner by its name if it has been deployed, else None
        """
        return self.runners.get(runner_name)
