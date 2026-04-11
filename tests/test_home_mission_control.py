"""Tests ciblés pour l'accueil Mission Control V7."""

from __future__ import annotations

from datetime import datetime

import polars as pl


class TestBuildSessionSummary:
    """Valide la construction des résumés de sessions récentes."""

    def test_returns_latest_solo_session_summary(self) -> None:
        """La dernière session solo doit être retenue par date."""
        from src.ui.pages.home_mission_control import _build_session_summary

        matches_df = pl.DataFrame(
            {
                "match_id": ["m1", "m2", "m3"],
                "start_time": [
                    datetime(2026, 4, 10, 10, 0),
                    datetime(2026, 4, 10, 10, 20),
                    datetime(2026, 4, 11, 18, 0),
                ],
                "ratio": [1.1, 1.3, 1.8],
                "accuracy": [45.0, 50.0, 55.0],
                "kills": [10, 12, 16],
                "deaths": [9, 10, 8],
                "assists": [4, 5, 7],
                "outcome": [2, 2, 2],
                "time_played_seconds": [600, 620, 700],
            }
        )
        sessions_df = pl.DataFrame(
            {
                "match_id": ["m1", "m2", "m3"],
                "session_label": ["Session #1", "Session #1", "Session #2"],
                "start_time": [
                    datetime(2026, 4, 10, 10, 0),
                    datetime(2026, 4, 10, 10, 20),
                    datetime(2026, 4, 11, 18, 0),
                ],
                "is_with_friends": [False, False, False],
            }
        )

        summary = _build_session_summary(matches_df, sessions_df, squad_mode=False)

        assert summary is not None
        assert summary.session_label == "Session #2"
        assert summary.match_count == 1
        assert summary.kpis.total_matches == 1

    def test_returns_none_when_no_squad_session_exists(self) -> None:
        """Aucune session escouade ne doit produire de résumé."""
        from src.ui.pages.home_mission_control import _build_session_summary

        matches_df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "start_time": [datetime(2026, 4, 10, 10, 0)],
                "ratio": [1.2],
                "accuracy": [44.0],
                "kills": [10],
                "deaths": [8],
                "assists": [5],
                "outcome": [2],
                "time_played_seconds": [650],
            }
        )
        sessions_df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "session_label": ["Solo #1"],
                "start_time": [datetime(2026, 4, 10, 10, 0)],
                "is_with_friends": [False],
            }
        )

        assert _build_session_summary(matches_df, sessions_df, squad_mode=True) is None


class TestSelectRecentMedia:
    """Valide la sélection des médias récents affichés dans l'accueil."""

    def test_keeps_latest_unique_entries(self) -> None:
        """La liste doit être triée et dédupliquée par chemin."""
        from src.ui.pages.home_mission_control import _select_recent_media

        media_df = pl.DataFrame(
            {
                "path": ["a.png", "b.png", "a.png", "c.png"],
                "basename": ["a.png", "b.png", "a.png", "c.png"],
                "mtime_paris_epoch": [100.0, 300.0, 100.0, 200.0],
                "match_id": ["m1", "m2", "m1", None],
                "match_start_time": [
                    datetime(2026, 4, 10, 10, 0),
                    datetime(2026, 4, 11, 10, 0),
                    datetime(2026, 4, 10, 10, 0),
                    None,
                ],
            }
        )

        entries = _select_recent_media(media_df, limit=2)

        assert [entry.basename for entry in entries] == ["b.png", "c.png"]

    def test_returns_empty_list_when_no_media(self) -> None:
        """Un DataFrame vide ne doit rien rendre."""
        from src.ui.pages.home_mission_control import _select_recent_media

        assert _select_recent_media(pl.DataFrame()) == []
