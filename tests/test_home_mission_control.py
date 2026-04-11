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


class TestNavigationState:
    """Valide la propagation du contexte V7 depuis l'accueil."""

    def test_builds_squad_session_scope(self) -> None:
        """Une navigation escouade doit préparer le scope session attendu."""
        from src.app.session_keys import SK
        from src.ui.pages.home_mission_control import _build_navigation_state

        state = _build_navigation_state(
            "squad",
            session_label="Carnage #7",
            squad_mode=True,
        )

        assert state[SK.V7_CURRENT_SECTION] == "squad"
        assert state[SK.FILTER_MODE] == "Sessions"
        assert state[SK.PICKED_SESSION_LABEL] == "Carnage #7"
        assert state[SK.PICKED_SESSIONS] == ["Carnage #7"]
        assert state[SK.PICKED_SQUAD_SESSION_LABEL] == "Carnage #7"
        assert state[SK.PICKED_SOLO_SESSION_LABEL] == "(toutes)"

    def test_builds_explorer_jump_with_pending_match(self) -> None:
        """Un raccourci Explorer doit conserver le match à ouvrir au rerun."""
        from src.app.session_keys import SK
        from src.ui.pages.home_mission_control import _build_navigation_state

        state = _build_navigation_state("explorer", pending_match_id="abc-123")

        assert state[SK.V7_CURRENT_SECTION] == "explorer"
        assert state[SK.PENDING_MATCH_ID] == "abc-123"


class TestRecentMatches:
    """Valide la timeline des matchs récents."""

    def test_selects_recent_matches_in_descending_order(self) -> None:
        """La timeline doit être triée du plus récent au plus ancien."""
        from src.ui.pages.home_mission_control import _select_recent_matches

        matches_df = pl.DataFrame(
            {
                "match_id": ["m1", "m2", "m3"],
                "start_time": [
                    datetime(2026, 4, 10, 9, 0),
                    datetime(2026, 4, 11, 12, 0),
                    datetime(2026, 4, 11, 8, 30),
                ],
                "map_name_fr": ["Bazaar", "Live Fire", "Recharge"],
                "pair_name": ["Slayer", "Oddball", "CTF"],
                "ratio": [1.1, 2.0, 0.8],
                "accuracy": [45.0, 58.0, 41.0],
                "outcome": [2, 3, 1],
            }
        )

        entries = _select_recent_matches(matches_df, limit=3)

        assert [entry.match_id for entry in entries] == ["m2", "m3", "m1"]
        assert entries[0].outcome_tone == "loss"
        assert "Live Fire" in entries[0].title


class TestTrendSnapshot:
    """Valide la fenêtre glissante de tendance récente."""

    def test_computes_delta_between_two_recent_windows(self) -> None:
        """La tendance doit comparer les 5 derniers matchs aux 5 précédents."""
        from src.ui.pages.home_mission_control import _compute_trend_snapshot

        matches_df = pl.DataFrame(
            {
                "match_id": [f"m{i}" for i in range(10)],
                "start_time": [datetime(2026, 4, 1, 12, i) for i in range(10)],
                "ratio": [1.0, 1.1, 1.2, 1.2, 1.3, 2.0, 2.1, 2.2, 2.3, 2.4],
                "accuracy": [40.0, 41.0, 42.0, 43.0, 44.0, 50.0, 51.0, 52.0, 53.0, 54.0],
                "kills": [10] * 10,
                "deaths": [8] * 10,
                "assists": [4] * 10,
                "outcome": [2, 2, 3, 2, 3, 2, 2, 2, 2, 2],
                "time_played_seconds": [600] * 10,
            }
        )

        snapshot = _compute_trend_snapshot(matches_df)

        assert snapshot is not None
        assert snapshot.ratio_delta is not None and snapshot.ratio_delta > 0
        assert snapshot.accuracy_delta is not None and snapshot.accuracy_delta > 0
        assert snapshot.win_rate_delta is not None and snapshot.win_rate_delta > 0
