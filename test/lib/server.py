import io
import logging
import time
import sys

import paramiko
from scp import SCPClient

logger = logging.getLogger(__name__)


class Server:
    def __init__(self, name: str, platform: str, ec2_instance: any, username: str, connection_type: str,
                 connection_auth: any):
        self.name = name
        self.platform = platform
        self.ec2_instance = ec2_instance
        self.username = username
        self.connection_type = connection_type
        self.connection_auth = connection_auth
        self.client = None

    def id(self) -> str:
        return self.ec2_instance.id

    def public_ip_address(self) -> str:
        return self.ec2_instance.public_ip_address

    def private_ip_address(self) -> str:
        return self.ec2_instance.private_ip_address

    def copy_file(self, from_local_path: str, to_remote_path: str, recursive: bool = False,
                        preserve_times: bool = False):
        self.connect()
        if self.connection_type == 'ssh':
            with SCPClient(self.client.get_transport()) as scp:
                scp.put(from_local_path, to_remote_path, recursive, preserve_times)
        else:
            raise Exception("Unsupported connection type")

    def read_file(self, from_remote_path: str, to_local_path: str, recursive: bool = False,
                          preserve_times: bool = False):
        self.connect()
        if self.connection_type == 'ssh':
            with SCPClient(self.client.get_transport()) as scp:
                scp.get(from_remote_path, to_local_path, recursive, preserve_times)
        else:
            raise Exception("Unsupported connection type")

    def exec(self, command: str):
        self.connect()
        if self.connection_type == 'ssh':
            stdin, stdout, stderr = self.client.exec_command(command)
            # Important to drain stdout/err before reading exit status below
            # See https://stackoverflow.com/questions/31625788/paramiko-ssh-die-hang-with-big-output
            stdout_data = stdout.readlines()
            stdout_data = "".join(stdout_data)
            stderr_data = stderr.readlines()
            stderr_data = "".join(stderr_data)
            exit_code = stdout.channel.recv_exit_status()
            sys.stdout.write(stdout_data)
            sys.stderr.write(stderr_data)
            return stdout_data, stderr_data, exit_code
        else:
            raise Exception("Unsupported connection type")

    def connect(self):
        if self.client is not None:
            return
        if self.connection_type == 'ssh':
            self.wait_for_ssh_to_be_ready()
            key = paramiko.RSAKey.from_private_key(io.StringIO(self.connection_auth))
            client = paramiko.SSHClient()
            client.load_system_host_keys()
            client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
            client.connect(username=self.username, hostname=self.public_ip_address(), pkey=key)
            self.client = client
        else:
            raise Exception("Unsupported connection type")

    def wait_for_ssh_to_be_ready(self, timeout=60, retry_interval=5):
        """
        Blocks and waits until an SSH connection can be established against
        the runner's public IP address.

        :param int timeout: the total amount of time to wait before giving up
        :param int retry_interval:
          the amount of time between each connectivity check

        :raises: `.Exception` -- if SSH does not become ready in time
        """

        logger.info("Waiting for SSH to become available on %s...", self.public_ip_address())
        client = paramiko.client.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        retry_interval = float(retry_interval)
        timeout = int(timeout)
        timeout_start = time.time()
        while time.time() < timeout_start + timeout:
            time.sleep(retry_interval)
            try:
                client.connect(self.public_ip_address(), 22, allow_agent=False, look_for_keys=False)
            except paramiko.ssh_exception.SSHException as e:
                if str(e) == 'Error reading SSH protocol banner':
                    logger.warning(e)
                    continue
                logger.debug('SSH transport is available!')
                return
            except paramiko.ssh_exception.NoValidConnectionsError as e:
                logger.debug('SSH transport is not ready...')
                continue
        raise Exception("SSH did not become ready in time")
