import logging
import os
import socket

from lib.ansible import exec_playbook
from lib.server_manager import ServerManager

logger = logging.getLogger(__name__)


class BBCLITestController:

    def __init__(self):
        self.sm = ServerManager()
        build_name = os.environ.get('BB_BUILD_NAME', socket.gethostname())
        self.server_name_prefix = "bb-cli-e2e-test-{}".format(build_name)
        self.servers_by_name = {}

    def __find_or_create_server(self, server_def):
        server_name = "{}-{}-{}-{}".format(self.server_name_prefix, server_def.platform,
                                           server_def.variant, server_def.architecture)

        if server_name in self.servers_by_name:
            logger.info("Found existing server: name={}".format(server_name))
            return self.servers_by_name[server_name]

        logger.info("No existing server found; will deploy new server: name={}".format(server_name))
        server = self.sm.deploy(server_name, server_def)
        exec_playbook(server, "bb", "bb-servers")
        self.servers_by_name[server_name] = server
        return server

    def execute_test(self, server_def, cli_test_data_dirname, bb_command="bb run -v"):
        server = self.__find_or_create_server(server_def)
        logger.info("Copying test files to server: test_data={}".format(cli_test_data_dirname))
        local_script = "./test-data/bb-cli/bb-cli-test.sh"
        remote_script = "/tmp/bb-test.sh"
        local_test_dir = "./test-data/bb-cli/{}".format(cli_test_data_dirname)
        remote_test_dir = "/tmp/{}".format(cli_test_data_dirname)
        server.copy_file(from_local_path=local_test_dir, to_remote_path=remote_test_dir, recursive=True)
        server.copy_file(from_local_path=local_script, to_remote_path=remote_script)
        _, _, exit_code = server.exec("chmod +x {}".format(remote_script))
        assert exit_code == 0
        logger.info("Running bb: test_data={} bb_cmd=\"{}\"".format(cli_test_data_dirname, bb_command))
        return server.exec("{} {} \"{}\"".format(remote_script, remote_test_dir, bb_command))

    def teardown(self):
        logger.info("Tearing down BB CLI Test Controller")
        self.sm.destroy_all_servers()
