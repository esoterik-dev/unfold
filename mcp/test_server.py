"""Regression tests for MCP command construction."""

import importlib.util
import os
import sys
import types
import unittest
from pathlib import Path
from unittest.mock import patch


class _FastMCP:
    def __init__(self, *_args, **_kwargs):
        pass

    def tool(self):
        return lambda function: function


def load_server_module():
    """Load the server without requiring the optional MCP dependency."""
    mcp_module = types.ModuleType("mcp")
    mcp_server_module = types.ModuleType("mcp.server")
    mcp_fastmcp_module = types.ModuleType("mcp.server.fastmcp")
    mcp_fastmcp_module.FastMCP = _FastMCP

    with patch.dict(
        sys.modules,
        {
            "mcp": mcp_module,
            "mcp.server": mcp_server_module,
            "mcp.server.fastmcp": mcp_fastmcp_module,
        },
    ), patch.dict(os.environ, {"UNFOLD_BIN": "/usr/bin/unfold"}, clear=False):
        spec = importlib.util.spec_from_file_location(
            "unfold_mcp_server", Path(__file__).with_name("server.py")
        )
        module = importlib.util.module_from_spec(spec)
        assert spec.loader is not None
        spec.loader.exec_module(module)
        return module


class SyncTransactionsTests(unittest.TestCase):
    def test_non_default_database_path_is_a_transactions_flag(self):
        server = load_server_module()

        with patch.object(server, "_run", return_value="ok") as run:
            result = server.sync_transactions(db_path="/tmp/custom.sqlite")

        self.assertEqual("ok", result)
        run.assert_called_once_with(
            ["transactions", "--db", "--db-path", "/tmp/custom.sqlite"]
        )


if __name__ == "__main__":
    unittest.main()
