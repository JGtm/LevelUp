"""Tests pour src/ui/pages/match_view_logic.py."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest


class TestDetectAbandonedMatch:
    """Tests pour detect_abandoned_match()."""

    def _patch_shared_db(self, shared_path: Path | None, db_result: tuple) -> tuple:
        """Retourne les patcheurs pour shared path + connexion DuckDB."""
        mock_conn = MagicMock()
        mock_conn.execute.return_value.fetchone.return_value = db_result
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(return_value=mock_conn)
        mock_ctx.__exit__ = MagicMock(return_value=False)
        return shared_path, mock_ctx

    def test_abandoned_when_all_zeros(self, tmp_path: Path) -> None:
        """Retourne True quand kills=deaths=score=0 pour tous les participants."""
        from src.ui.pages.match_view_logic import detect_abandoned_match

        shared_path = tmp_path / "shared_matches.duckdb"
        shared_path.touch()
        mock_conn = MagicMock()
        # COUNT=8, SUM(kills)=0, SUM(deaths)=0, SUM(score)=0
        mock_conn.execute.return_value.fetchone.return_value = (8, 0, 0, 0)
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(return_value=mock_conn)
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with (
            patch(
                "src.ui.pages.match_view_logic.get_shared_matches_path_from_player",
                return_value=shared_path,
            ),
            patch("src.ui.pages.match_view_logic.duckdb_read_only", return_value=mock_ctx),
        ):
            assert detect_abandoned_match("match-abc", "/some/stats.duckdb") is True

    def test_not_abandoned_when_kills_positive(self, tmp_path: Path) -> None:
        """Retourne False quand des kills ont été enregistrés."""
        from src.ui.pages.match_view_logic import detect_abandoned_match

        shared_path = tmp_path / "shared_matches.duckdb"
        shared_path.touch()
        mock_conn = MagicMock()
        # COUNT=8, SUM(kills)=45, SUM(deaths)=42, SUM(score)=5000
        mock_conn.execute.return_value.fetchone.return_value = (8, 45, 42, 5000)
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(return_value=mock_conn)
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with (
            patch(
                "src.ui.pages.match_view_logic.get_shared_matches_path_from_player",
                return_value=shared_path,
            ),
            patch("src.ui.pages.match_view_logic.duckdb_read_only", return_value=mock_ctx),
        ):
            assert detect_abandoned_match("match-xyz", "/some/stats.duckdb") is False

    def test_returns_false_when_no_participants(self, tmp_path: Path) -> None:
        """Retourne False si aucun participant trouvé en DB (match_id inconnu)."""
        from src.ui.pages.match_view_logic import detect_abandoned_match

        shared_path = tmp_path / "shared_matches.duckdb"
        shared_path.touch()
        mock_conn = MagicMock()
        # COUNT=0 — match_id inconnu
        mock_conn.execute.return_value.fetchone.return_value = (0, 0, 0, 0)
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(return_value=mock_conn)
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with (
            patch(
                "src.ui.pages.match_view_logic.get_shared_matches_path_from_player",
                return_value=shared_path,
            ),
            patch("src.ui.pages.match_view_logic.duckdb_read_only", return_value=mock_ctx),
        ):
            assert detect_abandoned_match("match-unknown", "/some/stats.duckdb") is False

    def test_returns_false_when_shared_db_not_found(self) -> None:
        """Retourne False si shared_matches.duckdb n'est pas accessible."""
        from src.ui.pages.match_view_logic import detect_abandoned_match

        with patch(
            "src.ui.pages.match_view_logic.get_shared_matches_path_from_player",
            return_value=None,
        ):
            assert detect_abandoned_match("match-abc", "/some/stats.duckdb") is False

    def test_returns_false_on_db_exception(self, tmp_path: Path) -> None:
        """Retourne False (sans lever d'exception) si la DB lève une erreur."""
        from src.ui.pages.match_view_logic import detect_abandoned_match

        shared_path = tmp_path / "shared_matches.duckdb"
        shared_path.touch()

        with (
            patch(
                "src.ui.pages.match_view_logic.get_shared_matches_path_from_player",
                return_value=shared_path,
            ),
            patch(
                "src.ui.pages.match_view_logic.duckdb_read_only",
                side_effect=RuntimeError("DB locked"),
            ),
        ):
            assert detect_abandoned_match("match-abc", "/some/stats.duckdb") is False
