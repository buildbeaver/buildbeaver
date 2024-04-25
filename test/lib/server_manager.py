import logging

import boto3
from botocore.exceptions import ClientError
from lib.server import Server

server_configurations = {
    'linux': {
        'ubuntu-22.04': {
            'amd64': {
                'image_id': 'ami-095413544ce52437d',
                'instance_type': 't2.micro',
                'username': 'ubuntu',
                'connection_type': 'ssh'
            },
            'arm64': {
                'image_id': 'ami-072d13a1cd84b5f6b',
                'instance_type': 't4g.micro',
                'username': 'ubuntu',
                'connection_type': 'ssh'
            },
        }
    },
    'windows': {
        'server-2022-base': {
            'amd64': {
                'image_id': 'ami-0e2daa9ce776be2b0',
                'instance_type': 't2.micro',
                'username': 'administrator',
                'connection_type': 'winrm'
            }
        },
        'server-2019-base': {
            'amd64': {
                'image_id': 'ami-01aeb1044bb7cb673',
                'instance_type': 't2.micro',
                'username': 'administrator',
                'connection_type': 'winrm'
            }
        },
        'server-2016-base': {
            'amd64': {
                'image_id': 'ami-005b3f1e9bd7a715a',
                'instance_type': 't2.micro',
                'username': 'administrator',
                'connection_type': 'winrm'
            }
        }
    },
    'macos': {
        'ventura': {
            'amd64': {
                'image_id': 'ami-0dd2ded7568750663',
                'instance_type': 'mac1.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            },
            'arm64': {
                'image_id': 'ami-03dd0557beedd17d3',
                'instance_type': 'mac2.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            }
        },
        'monterey': {
            'amd64': {
                'image_id': 'ami-0d500eeebb40b2269',
                'instance_type': 'mac1.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            },
            'arm64': {
                'image_id': 'ami-084c6ab9d03ad4d46',
                'instance_type': 'mac2.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            }
        },
        'bigsur': {
            'amd64': {
                'image_id': 'ami-0c5e75b0a720163ac',
                'instance_type': 'mac1.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            },
            'arm64': {
                'image_id': 'ami-014950a66ddc4b722',
                'instance_type': 'mac2.metal',
                'username': 'ec2-user',
                'connection_type': 'ssh'
            }
        }
    }
}

logger = logging.getLogger(__name__)


class ServerDefinition:
    def __init__(self, platform: str, variant: str, architecture: str):
        self.platform = platform
        self.variant = variant
        self.architecture = architecture

    def string(self):
        return "platform={} variant={} arch={}".format(self.platform, self.variant, self.architecture)


class ServerManager:

    def __init__(self, subnet_name_filter="buildbeaver-public-us-west-2a", security_group_name_filter="buildbeaver-dmz"):
        self.ssh_private_key = None
        self.subnet_id = None
        self.security_group_id = None
        self.servers = {}
        self.subnet_name_filter = subnet_name_filter
        self.security_group_name_filter = security_group_name_filter

    def __load_ssh_key(self):
        logger.info("Loading private key from SSM...")
        ssm = boto3.client('ssm')
        parameter = ssm.get_parameter(Name='/buildbeaver-e2e/buildbeaver-e2e.pem', WithDecryption=True)
        self.ssh_private_key = parameter['Parameter']['Value']

    def __discover_subnet_id(self):
        ec2_client = boto3.client('ec2')
        sn_all = ec2_client.describe_subnets(Filters=[
            {'Name': 'tag:Name', 'Values': [self.subnet_name_filter]},
        ])
        if len(sn_all['Subnets']) == 0:
            raise Exception("Unable to find public subnet matching: {}".format(self.subnet_name_filter))
        self.subnet_id = sn_all['Subnets'][0]['SubnetId']

    def __discover_security_group_id(self):
        ec2_client = boto3.client('ec2')
        sg_all = ec2_client.describe_security_groups(Filters=[
            {'Name': 'tag:Name', 'Values': [self.security_group_name_filter]},
        ])
        if len(sg_all['SecurityGroups']) == 0:
            raise Exception("Unable to find security group matching: {}".format(self.security_group_name_filter))
        self.security_group_id = sg_all['SecurityGroups'][0]['GroupId']

    def deploy(self, server_name: str, server_def: ServerDefinition, tags=None):
        if tags is None:
            tags = []
        tags.append({'Key': 'Name', 'Value': server_name})

        if server_def.platform not in server_configurations:
            raise Exception("Unknown platform")
        if server_def.variant not in server_configurations[server_def.platform]:
            raise Exception("Unknown variant")
        if server_def.architecture not in server_configurations[server_def.platform][server_def.variant]:
            raise Exception("Unknown architecture")

        if self.ssh_private_key is None:
            self.__load_ssh_key()

        if self.subnet_id is None:
            self.__discover_subnet_id()

        if self.security_group_id is None:
            self.__discover_security_group_id()

        logger.info("Deploying server: name=%s platform=%s variant=%s architecture=%s",
                    server_name, server_def.platform, server_def.variant, server_def.architecture)

        config = server_configurations[server_def.platform][server_def.variant][server_def.architecture]
        try:
            instance_params = {
                'ImageId': config['image_id'],
                'InstanceType': config['instance_type'],
                'KeyName': 'buildbeaver-e2e',
                'SecurityGroupIds': [self.security_group_id],
                'SubnetId': self.subnet_id,
                'TagSpecifications': [
                    {
                        'ResourceType': 'instance',
                        'Tags': tags
                    },
                ],
            }

            ec2_resource = boto3.resource('ec2')
            instance = ec2_resource.create_instances(**instance_params, MinCount=1, MaxCount=1)[0]
            instance.wait_until_running()
            instance.reload()
            logger.info("Deployed server: name=%s id=%s public_ip_address=%s", server_name, instance.id,
                        instance.public_ip_address)
            deployed_server = Server(server_name, server_def.platform, instance, config['username'],
                                     config['connection_type'],
                                     self.ssh_private_key)
            self.servers[server_name] = deployed_server
            return deployed_server

        except ClientError as err:
            logger.error(
                "Couldn't create instance with image %s, instance type %s. "
                "Here's why: %s: %s", config['image_id'], config['instance_type'],
                err.response['Error']['Code'], err.response['Error']['Message'])
            raise

    def destroy_server(self, server):
        logger.info("Destroying server: name=%s id=%s...", server.name, server.id())
        ec2_client = boto3.client('ec2')
        ec2_client.terminate_instances(InstanceIds=[server.id()])

    def destroy_all_servers(self):
        logger.info("Destroying all servers...")
        for name, server in self.servers.items():
            self.destroy_server(server)

    def get_server(self, server_name) -> Server | None:
        """
        Returns a server by its name if it has been deployed, else None
        """
        return self.servers.get(server_name)
