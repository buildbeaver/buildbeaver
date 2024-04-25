import logging

import pytest

from lib.server_manager import ServerDefinition

logger = logging.getLogger(__name__)


def id_fn(server_def: ServerDefinition):
    return server_def.string()


class TestBB:

    @pytest.mark.parametrize("server_def", [ServerDefinition("linux", "ubuntu-22.04", "amd64")], ids=id_fn)
    def test_bb_cli_static_smoke(self, test_cli_controller, server_def):
        _, _, exit_code = test_cli_controller.execute_test(server_def, "static-smoke")
        assert exit_code == 0
