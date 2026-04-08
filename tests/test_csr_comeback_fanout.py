"""Tests unitaires — Fan-out CSR et comeback badges.

Couvre :
- write_csr_from_skill_update : écriture CSR dans player DB
- _collect_csr_for_other_players : filtrage et accumulation
- _run_other_dominance fusionnée : une seule connexion shared R/O
"""

from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

import duckdb
import pytest

from src.data.sync._skill_rating import write_csr_from_skill_update
from src.data.sync.models import SkillParticipantUpdate

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_player_db(path: Path) -> duckdb.DuckDBPyConnection:
    """Crée une player DB minimale avec match_skill_rank."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_skill_rank (
            match_id VARCHAR PRIMARY KEY,
            rating_type VARCHAR,
            rating_value FLOAT,
            rating_deviation FLOAT,
            tier VARCHAR,
            tier_fr VARCHAR,
            sub_tier INTEGER,
            tier_label VARCHAR,
            rating_delta FLOAT,
            playlist_group VARCHAR,
            start_time TIMESTAMP,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    return conn


def _skill_update(  # noqa: PLR0913
    match_id: str = "m-001",
    xuid: str = "2535400000002",
    post_csr: float | None = 1200.0,
    pre_csr: float | None = 1150.0,
    csr_tier: str | None = "Gold",
    csr_sub_tier: int | None = 3,
) -> SkillParticipantUpdate:
    return SkillParticipantUpdate(
        match_id=match_id,
        xuid=xuid,
        post_match_csr=post_csr,
        pre_match_csr=pre_csr,
        csr_tier=csr_tier,
        csr_sub_tier=csr_sub_tier,
    )


# =============================================================================
# write_csr_from_skill_update
# =============================================================================


class TestWriteCsrFromSkillUpdate:
    """Tests pour write_csr_from_skill_update (standalone, sans self)."""

    def test_writes_csr_and_returns_true(self, tmp_path: Path) -> None:
        """Écrit une ligne CSR et retourne True."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        update = _skill_update(post_csr=1200.0)
        result = write_csr_from_skill_update(conn, update)
        assert result is True
        row = conn.execute(
            "SELECT rating_type, rating_value, tier FROM match_skill_rank WHERE match_id = 'm-001'"
        ).fetchone()
        assert row is not None
        assert row[0] == "CSR"
        assert row[1] == pytest.approx(1200.0)
        assert row[2] == "Gold"
        conn.close()

    def test_returns_false_when_post_csr_is_none(self, tmp_path: Path) -> None:
        """Retourne False si post_match_csr est None — rien n'est écrit."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        update = _skill_update(post_csr=None)
        result = write_csr_from_skill_update(conn, update)
        assert result is False
        count = conn.execute("SELECT COUNT(*) FROM match_skill_rank").fetchone()[0]
        assert count == 0
        conn.close()

    def test_returns_false_when_match_id_missing(self, tmp_path: Path) -> None:
        """Retourne False si l'objet n'a pas de match_id."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        obj = SimpleNamespace(post_match_csr=1200.0, match_id=None)
        result = write_csr_from_skill_update(conn, obj)
        assert result is False
        conn.close()

    def test_computes_delta_correctly(self, tmp_path: Path) -> None:
        """Le delta CSR est bien calculé (post - pre)."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        update = _skill_update(post_csr=1250.0, pre_csr=1200.0)
        write_csr_from_skill_update(conn, update)
        row = conn.execute(
            "SELECT rating_delta FROM match_skill_rank WHERE match_id = 'm-001'"
        ).fetchone()
        assert row[0] == pytest.approx(50.0)
        conn.close()

    def test_delta_none_when_pre_csr_absent(self, tmp_path: Path) -> None:
        """Le delta est NULL si pre_match_csr est None."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        update = _skill_update(post_csr=1300.0, pre_csr=None)
        write_csr_from_skill_update(conn, update)
        row = conn.execute(
            "SELECT rating_delta FROM match_skill_rank WHERE match_id = 'm-001'"
        ).fetchone()
        assert row[0] is None
        conn.close()

    def test_idempotent_upsert(self, tmp_path: Path) -> None:
        """Deux appels successifs ne créent pas de doublon (ON CONFLICT DO UPDATE)."""
        conn = _make_player_db(tmp_path / "stats.duckdb")
        update_v1 = _skill_update(post_csr=1200.0)
        update_v2 = _skill_update(post_csr=1250.0)
        write_csr_from_skill_update(conn, update_v1)
        write_csr_from_skill_update(conn, update_v2)
        count = conn.execute("SELECT COUNT(*) FROM match_skill_rank").fetchone()[0]
        assert count == 1
        val = conn.execute(
            "SELECT rating_value FROM match_skill_rank WHERE match_id = 'm-001'"
        ).fetchone()[0]
        assert val == pytest.approx(1250.0)
        conn.close()


# =============================================================================
# _collect_csr_for_other_players
# =============================================================================


def _make_engine_mock(
    my_xuid: str = "2535400000001",
    other_players: list[dict] | None = None,
) -> MagicMock:
    """Crée un mock de engine qui implémente _SyncProtocol."""
    if other_players is None:
        other_players = [
            {"xuid": "2535400000002", "gamertag": "PlayerB"},
            {"xuid": "2535400000003", "gamertag": "PlayerC"},
        ]
    engine = MagicMock()
    engine._xuid = my_xuid
    engine._pending_other_csr = {}
    engine._get_other_registered_players.return_value = other_players
    return engine


class TestCollectCsrForOtherPlayers:
    """Tests pour _collect_csr_for_other_players (mixin method)."""

    def _call(self, engine: MagicMock, updates: list[SkillParticipantUpdate]) -> None:
        from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin

        MatchProcessingHelpersMixin._collect_csr_for_other_players(engine, updates)

    def test_collects_csr_for_other_registered(self) -> None:
        """CSR d'un joueur enregistré différent de self est accumulé."""
        engine = _make_engine_mock(my_xuid="X_A")
        updates = [
            _skill_update(xuid="X_B", post_csr=1200.0),
        ]
        engine._get_other_registered_players.return_value = [{"xuid": "X_B"}]
        self._call(engine, updates)
        assert "X_B" in engine._pending_other_csr
        assert len(engine._pending_other_csr["X_B"]) == 1

    def test_skips_self(self) -> None:
        """Ne collecte pas le CSR du joueur courant."""
        engine = _make_engine_mock(my_xuid="X_A")
        updates = [_skill_update(xuid="X_A", post_csr=1200.0)]
        engine._get_other_registered_players.return_value = [{"xuid": "X_A"}]
        self._call(engine, updates)
        assert "X_A" not in engine._pending_other_csr

    def test_skips_unregistered_player(self) -> None:
        """Ne collecte pas le CSR d'un joueur non enregistré."""
        engine = _make_engine_mock(my_xuid="X_A")
        updates = [_skill_update(xuid="X_UNKNOWN", post_csr=1200.0)]
        engine._get_other_registered_players.return_value = [{"xuid": "X_B"}]
        self._call(engine, updates)
        assert "X_UNKNOWN" not in engine._pending_other_csr

    def test_skips_updates_without_csr(self) -> None:
        """Ignore les updates sans post_match_csr (matchs non classés)."""
        engine = _make_engine_mock(my_xuid="X_A")
        updates = [_skill_update(xuid="X_B", post_csr=None)]
        engine._get_other_registered_players.return_value = [{"xuid": "X_B"}]
        self._call(engine, updates)
        assert engine._pending_other_csr == {}

    def test_empty_list_does_nothing(self) -> None:
        """Liste vide → aucun changement."""
        engine = _make_engine_mock()
        self._call(engine, [])
        assert engine._pending_other_csr == {}

    def test_accumulates_multiple_matches(self) -> None:
        """Plusieurs matchs pour le même joueur s'accumulent correctement."""
        engine = _make_engine_mock(my_xuid="X_A")
        engine._get_other_registered_players.return_value = [{"xuid": "X_B"}]
        updates_m1 = [_skill_update(match_id="m-001", xuid="X_B", post_csr=1200.0)]
        updates_m2 = [_skill_update(match_id="m-002", xuid="X_B", post_csr=1250.0)]
        self._call(engine, updates_m1)
        self._call(engine, updates_m2)
        assert len(engine._pending_other_csr["X_B"]) == 2

    def test_skips_player_without_xuid_in_profiles(self) -> None:
        """Un profil sans clé 'xuid' ne génère pas d'erreur et est ignoré."""
        engine = _make_engine_mock(my_xuid="X_A")
        engine._get_other_registered_players.return_value = [{"gamertag": "PlayerB"}]  # sans xuid
        updates = [_skill_update(xuid="X_B", post_csr=1200.0)]
        self._call(engine, updates)
        assert engine._pending_other_csr == {}


# =============================================================================
# _run_other_dominance (fusionnée avec comeback)
# =============================================================================


class TestRunOtherDominanceFused:
    """Tests pour _run_other_dominance fusionnée (dominance + comeback, 1 connexion)."""

    def _call(
        self,
        engine: MagicMock,
        gamertag: str,
        xuid: str,
        shared_path: Path,
        conn: MagicMock,
    ) -> None:
        from src.data.sync._engine_fanout import FanoutEnrichmentMixin

        FanoutEnrichmentMixin._run_other_dominance(engine, gamertag, xuid, shared_path, conn)

    @patch("src.data.sync._engine_fanout.duckdb")
    @patch("src.data.comeback_backfill.compute_comeback_badges_for_player")
    @patch("src.data.dominance_backfill.compute_dominance_for_player")
    def test_opens_single_connection(
        self,
        mock_dominance: MagicMock,
        mock_comeback: MagicMock,
        mock_duckdb: MagicMock,
        tmp_path: Path,
    ) -> None:
        """Une seule connexion duckdb est ouverte pour dominance + comeback."""
        mock_dominance.return_value = {"processed": 2}
        mock_comeback.return_value = {
            "processed": 1,
            "remontada": 1,
            "debandade": 0,
            "contre_remontada": 0,
        }
        shared_path = tmp_path / "shared.duckdb"
        shared_path.touch()
        player_conn = MagicMock()
        engine = MagicMock()

        self._call(engine, "PlayerB", "X_B", shared_path, player_conn)

        # duckdb.connect appelé une seule fois
        mock_duckdb.connect.assert_called_once_with(str(shared_path), read_only=True)

    @patch("src.data.sync._engine_fanout.duckdb")
    @patch("src.data.comeback_backfill.compute_comeback_badges_for_player")
    @patch("src.data.dominance_backfill.compute_dominance_for_player")
    def test_both_computations_called(
        self,
        mock_dominance: MagicMock,
        mock_comeback: MagicMock,
        mock_duckdb: MagicMock,
        tmp_path: Path,
    ) -> None:
        """compute_dominance et compute_comeback sont tous les deux appelés."""
        mock_dominance.return_value = {"processed": 3}
        mock_comeback.return_value = {
            "processed": 0,
            "remontada": 0,
            "debandade": 0,
            "contre_remontada": 0,
        }
        shared_path = tmp_path / "shared.duckdb"
        shared_path.touch()
        player_conn = MagicMock()

        self._call(MagicMock(), "PlayerB", "X_B", shared_path, player_conn)

        assert mock_dominance.called
        assert mock_comeback.called

    @patch("src.data.sync._engine_fanout.duckdb")
    def test_exception_is_caught_and_logged(
        self,
        mock_duckdb: MagicMock,
        tmp_path: Path,
    ) -> None:
        """Une exception dans dominance/comeback n'est pas propagée (non bloquant)."""
        mock_duckdb.connect.side_effect = RuntimeError("DB locked")
        shared_path = tmp_path / "shared.duckdb"
        player_conn = MagicMock()
        engine = MagicMock()

        # Ne doit pas lever d'exception
        self._call(engine, "PlayerB", "X_B", shared_path, player_conn)
