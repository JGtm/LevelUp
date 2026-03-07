"""Tests de l'abstraction API (Ports & Adapters).

Vérifie :
- Conformité Protocol de SPNKrAPIClient avec HaloAPIPort
- Factory create_api_client()
- Facade auth (_auth.py)
- Absence d'imports spnkr dans les modules UI migrés
"""

from __future__ import annotations

import ast
import importlib
from pathlib import Path

import pytest

from src.data.sync.api_factory import create_api_client
from src.data.sync.api_port import HaloAPIPort

REPO_ROOT = Path(__file__).resolve().parent.parent


class TestProtocolConformance:
    """SPNKrAPIClient doit implémenter HaloAPIPort."""

    def test_spnkr_is_subclass(self) -> None:
        from src.data.sync.api_client import SPNKrAPIClient

        assert issubclass(SPNKrAPIClient, HaloAPIPort)

    def test_spnkr_instance_check(self) -> None:
        from src.data.sync.api_client import SPNKrAPIClient

        client = SPNKrAPIClient(tokens=None)
        assert isinstance(client, HaloAPIPort)

    def test_protocol_is_runtime_checkable(self) -> None:
        assert hasattr(HaloAPIPort, "__protocol_attrs__") or hasattr(
            HaloAPIPort, "__abstractmethods__"
        )


class TestFactory:
    """create_api_client() retourne un HaloAPIPort."""

    def test_default_backend_spnkr(self) -> None:
        client = create_api_client()
        assert isinstance(client, HaloAPIPort)

    def test_explicit_spnkr_backend(self) -> None:
        client = create_api_client(backend="spnkr")
        assert isinstance(client, HaloAPIPort)

    def test_unknown_backend_raises(self) -> None:
        with pytest.raises(ValueError, match="Backend API inconnu"):
            create_api_client(backend="grunt")

    def test_tokens_forwarded(self) -> None:
        from src.data.sync._tokens import Tokens

        tokens = Tokens(spartan_token="st", clearance_token="ct")
        client = create_api_client(tokens=tokens)
        assert isinstance(client, HaloAPIPort)


class TestAuthFacade:
    """_auth.py encapsule l'auth sans exposer spnkr."""

    def test_module_has_no_top_level_spnkr_import(self) -> None:
        """_auth.py ne doit pas avoir d'import spnkr au top-level."""
        auth_path = REPO_ROOT / "src" / "data" / "sync" / "_auth.py"
        tree = ast.parse(auth_path.read_text(encoding="utf-8"))
        for node in ast.iter_child_nodes(tree):
            if isinstance(node, ast.Import):
                for alias in node.names:
                    assert not alias.name.startswith(
                        "spnkr"
                    ), f"Import spnkr au top-level dans _auth.py : {alias.name}"
            elif (
                isinstance(node, ast.ImportFrom) and node.module and node.module.startswith("spnkr")
            ):
                pytest.fail(f"Import spnkr au top-level dans _auth.py : {node.module}")

    def test_refresh_halo_tokens_importable(self) -> None:
        from src.data.sync._auth import refresh_halo_tokens

        assert callable(refresh_halo_tokens)

    def test_refresh_halo_tokens_from_env_importable(self) -> None:
        from src.data.sync._auth import refresh_halo_tokens_from_env

        assert callable(refresh_halo_tokens_from_env)


class TestExports:
    """Les exports publics sont disponibles depuis src.data.sync."""

    def test_haloapi_port_exported(self) -> None:
        mod = importlib.import_module("src.data.sync")
        assert hasattr(mod, "HaloAPIPort")

    def test_create_api_client_exported(self) -> None:
        mod = importlib.import_module("src.data.sync")
        assert hasattr(mod, "create_api_client")


class TestNoSpnkrInMigratedUI:
    """Les modules UI migrés ne doivent plus importer spnkr au top-level."""

    @pytest.mark.parametrize(
        "module_path",
        [
            "src/ui/profile_api_tokens.py",
            "src/ui/player_assets.py",
        ],
    )
    def test_no_top_level_spnkr_import(self, module_path: str) -> None:
        path = REPO_ROOT / module_path
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.iter_child_nodes(tree):
            if isinstance(node, ast.Import):
                for alias in node.names:
                    assert not alias.name.startswith(
                        "spnkr"
                    ), f"Import spnkr dans {module_path} : {alias.name}"
            elif (
                isinstance(node, ast.ImportFrom) and node.module and node.module.startswith("spnkr")
            ):
                pytest.fail(f"Import spnkr dans {module_path} : {node.module}")
