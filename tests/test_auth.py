"""Tests pour src/utils/auth.py — Gestion de l'authentification OAuth."""

from __future__ import annotations

import os
from pathlib import Path
from unittest.mock import patch

from src.utils.auth import AuthStatus, get_auth_status, write_env_local

# =============================================================================
# AuthStatus
# =============================================================================


class TestAuthStatus:
    """Tests du dataclass AuthStatus."""

    def test_has_credentials_true(self) -> None:
        status = AuthStatus(has_client_id=True, has_client_secret=True)
        assert status.has_credentials is True

    def test_has_credentials_missing_id(self) -> None:
        status = AuthStatus(has_client_id=False, has_client_secret=True)
        assert status.has_credentials is False

    def test_has_credentials_missing_secret(self) -> None:
        status = AuthStatus(has_client_id=True, has_client_secret=False)
        assert status.has_credentials is False

    def test_is_fully_configured(self) -> None:
        status = AuthStatus(
            has_client_id=True,
            has_client_secret=True,
            has_refresh_token=True,
        )
        assert status.is_fully_configured is True

    def test_is_fully_configured_no_token(self) -> None:
        status = AuthStatus(
            has_client_id=True,
            has_client_secret=True,
            has_refresh_token=False,
        )
        assert status.is_fully_configured is False

    def test_missing_keys_tracked(self) -> None:
        status = AuthStatus(missing_keys=["KEY_A", "KEY_B"])
        assert len(status.missing_keys) == 2


# =============================================================================
# get_auth_status
# =============================================================================


class TestGetAuthStatus:
    """Tests de get_auth_status()."""

    @patch.dict(
        os.environ,
        {
            "SPNKR_AZURE_CLIENT_ID": "12345678-1234-1234-1234-123456789abc",
            "SPNKR_AZURE_CLIENT_SECRET": "my_secret_value",  # pragma: allowlist secret
            "SPNKR_OAUTH_REFRESH_TOKEN": "my_token",
        },
    )
    def test_fully_configured(self) -> None:
        status = get_auth_status()
        assert status.has_client_id is True
        assert status.has_client_secret is True
        assert status.has_refresh_token is True
        assert status.is_fully_configured is True
        assert status.missing_keys == []

    @patch.dict(
        os.environ,
        {
            "SPNKR_AZURE_CLIENT_ID": "",
            "SPNKR_AZURE_CLIENT_SECRET": "",
            "SPNKR_OAUTH_REFRESH_TOKEN": "",
        },
    )
    def test_empty_env_vars(self) -> None:
        status = get_auth_status()
        assert status.has_client_id is False
        assert status.has_client_secret is False
        assert status.has_refresh_token is False
        assert len(status.missing_keys) == 3

    @patch.dict(
        os.environ,
        {
            "SPNKR_AZURE_CLIENT_ID": "some-id",
            "SPNKR_AZURE_CLIENT_SECRET": "some-secret",  # pragma: allowlist secret
        },
        clear=False,
    )
    def test_no_refresh_token(self) -> None:
        # S'assurer que le token n'est pas dans l'env
        os.environ.pop("SPNKR_OAUTH_REFRESH_TOKEN", None)
        status = get_auth_status()
        assert status.has_credentials is True
        assert status.has_refresh_token is False
        assert "SPNKR_OAUTH_REFRESH_TOKEN" in status.missing_keys


# =============================================================================
# write_env_local
# =============================================================================


class TestWriteEnvLocal:
    """Tests de write_env_local()."""

    def test_create_new_file(self, tmp_path: Path) -> None:
        env_file = tmp_path / ".env.local"
        with patch("src.utils.auth._env_local_path", return_value=env_file):
            write_env_local({"KEY_A": "value_a", "KEY_B": "value_b"})

        content = env_file.read_text(encoding="utf-8")
        assert "KEY_A=value_a" in content
        assert "KEY_B=value_b" in content

    def test_update_existing_key(self, tmp_path: Path) -> None:
        env_file = tmp_path / ".env.local"
        env_file.write_text("KEY_A=old_value\nKEY_B=keep_me\n", encoding="utf-8")

        with patch("src.utils.auth._env_local_path", return_value=env_file):
            write_env_local({"KEY_A": "new_value"})

        content = env_file.read_text(encoding="utf-8")
        assert "KEY_A=new_value" in content
        assert "KEY_B=keep_me" in content
        assert "KEY_A=old_value" not in content

    def test_preserve_comments(self, tmp_path: Path) -> None:
        env_file = tmp_path / ".env.local"
        env_file.write_text(
            "# Azure config\nSPNKR_AZURE_CLIENT_ID=abc\n\n# Token\n",
            encoding="utf-8",
        )

        with patch("src.utils.auth._env_local_path", return_value=env_file):
            write_env_local({"SPNKR_AZURE_CLIENT_ID": "xyz"})

        content = env_file.read_text(encoding="utf-8")
        assert "# Azure config" in content
        assert "# Token" in content
        assert "SPNKR_AZURE_CLIENT_ID=xyz" in content

    def test_append_new_keys(self, tmp_path: Path) -> None:
        env_file = tmp_path / ".env.local"
        env_file.write_text("EXISTING=yes\n", encoding="utf-8")

        with patch("src.utils.auth._env_local_path", return_value=env_file):
            write_env_local({"NEW_KEY": "new_val"})

        content = env_file.read_text(encoding="utf-8")
        assert "EXISTING=yes" in content
        assert "NEW_KEY=new_val" in content
