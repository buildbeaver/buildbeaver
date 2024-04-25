import logging
import os
import tempfile

from lib.server import Server

from . import util

logger = logging.getLogger(__name__)


def exec_playbook(server: Server, playbook_name: str, group_name: str, vars: [str] = None):
    logger.info("Executing Ansible playbook on server: server_name={}, playbook={}".format(server.name, playbook_name))
    if server.connection_type == 'ssh':
        server.wait_for_ssh_to_be_ready()

        temp_dir = tempfile.gettempdir()
        private_key_file_path = os.path.join(temp_dir, ".ssh", "buildbeaver-e2e.pem")
        os.makedirs(os.path.dirname(private_key_file_path), exist_ok=True)
        private_key_file = open(private_key_file_path, "w")
        private_key_file.write(server.connection_auth)
        private_key_file.close()
        os.chmod(private_key_file_path, 0o600)

        inventory_content = '''[{}]
        {} ansible_user={} ansible_ssh_private_key_file={} ansible_ssh_common_args='-o StrictHostKeyChecking=no'
        '''.format(group_name, server.public_ip_address(), server.username, private_key_file_path, )
        if vars:
            inventory_content = inventory_content + "\n\n[{}:vars]\n".format(group_name)
            for var in vars:
                inventory_content = inventory_content + var + "\n"

        inventory_file_path = os.path.join(temp_dir, "inventory.ini")
        inventory_file = open(inventory_file_path, "w")
        inventory_file.write(inventory_content)
        inventory_file.close()
        os.chmod(inventory_file_path, 0o744)

        util.run_command(["cp", "-R", "../build/ansible/inventory/group_vars", temp_dir])
        exit_code = util.run_command(["ansible-galaxy", "install", "-r", "../build/ansible/requirements.yml"])
        if exit_code != 0:
            raise Exception("Failed to run ansible-galaxy: {:n}".format(exit_code))
        exit_code = util.run_command(
            ["ansible-playbook", "-i", inventory_file_path, "../build/ansible/playbooks/{}.yml".format(playbook_name)])
        if exit_code != 0:
            raise Exception("Failed to run ansible-playbook: {:n}".format(exit_code))
    else:
        raise Exception("Unsupported connection type")
