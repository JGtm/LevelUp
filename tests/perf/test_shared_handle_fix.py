"""Tests — Axe 2 : shared_matches en R/O direct (pas d'ATTACH sur player_conn).

Vérifie que :
- backfill_citations_for_player n'attache PAS shared à player_conn
- backfill_sessions_for_player n'attache PAS shared à player_conn
- _get_uncited_match_ids fonctionne avec shared_ro séparé
- _load_matches_split fonctionne sans ATTACH
- Les fonctions se dégradent proprement si shared est absent
"""

from __future__ import annotations

from datetime import datetime, timedelta
from pathlib import Path
from unittest.mock import patch

import duckdb
import pytest

from src.data.citations_backfill import (
    _get_uncited_match_ids,
    backfill_citations_for_player,
)
from src.data.sessions_backfill import backfill_sessions_for_player
from src.data.sessions_backfill_shared import _load_matches_split

_NOW = datetime(2025, 1, 1, 12, 0, 0)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def shared_and_player_dbs(tmp_path):
    """Crée une paire (player_db, shared_db) indépendantes sans ATTACH."""
    player_dir = tmp_path / "data" / "players" / "TestPlayer"
    player_dir.mkdir(parents=True, exist_ok=True)
    warehouse_dir = tmp_path / "data" / "warehouse"
    warehouse_dir.mkdir(parents=True, exist_ok=True)

    player_db = player_dir / "stats.duckdb"
    shared_db = warehouse_dir / "shared_matches_v2.duckdb"

    with duckdb.connect(str(player_db)) as conn:
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score FLOAT,
                session_id VARCHAR,
                session_label VARCHAR,
                is_with_friends BOOLEAN,
                teammates_signature VARCHAR,
                known_teammates_count SMALLINT,
                friends_xuids VARCHAR,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        conn.execute("""
            CREATE TABLE match_citations (
                match_id VARCHAR NOT NULL,
                citation_name_norm VARCHAR NOT NULL,
                value INTEGER DEFAULT 0,
                PRIMARY KEY (match_id, citation_name_norm)
            )
        """)

    with duckdb.connect(str(shared_db)) as conn:
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                is_ranked BOOLEAN DEFAULT FALSE
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                team_id INTEGER DEFAULT 1
            )
        """)
        conn.execute("""
            CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)
        """)

    return {"player_db": player_db, "shared_db": shared_db}


def _populate_shared(shared_db: Path, xuid: str, n: int = 3) -> list[str]:
    """Insère n matchs dans shared et retourne leurs IDs."""
    match_ids = [f"match-{i:03d}" for i in range(n)]
    with duckdb.connect(str(shared_db)) as conn:
        for i, mid in enumerate(match_ids):
            start = _NOW - timedelta(hours=1) + timedelta(minutes=i * 10)
            conn.execute("INSERT INTO match_registry VALUES (?, ?, FALSE)", [mid, start])
            conn.execute("INSERT INTO match_participants VALUES (?, ?, 1)", [mid, xuid])
        conn.execute("INSERT OR IGNORE INTO xuid_aliases VALUES (?, 'TestPlayer')", [xuid])
    return match_ids


def _populate_player(player_db: Path, match_ids: list[str]) -> None:
    """Insère les matchs dans player_match_enrichment."""
    with duckdb.connect(str(player_db)) as conn:
        for mid in match_ids:
            conn.execute("INSERT INTO player_match_enrichment (match_id) VALUES (?)", [mid])


# ---------------------------------------------------------------------------
# Tests _get_uncited_match_ids
# ---------------------------------------------------------------------------


class TestGetUncitedMatchIds:
    """_get_uncited_match_ids utilise shared_ro + player_conn séparément."""

    def test_returns_all_ids_when_no_citations(self, shared_and_player_dbs):
        xuid = "1111111111"
        match_ids = _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=3)
        shared_db = shared_and_player_dbs["shared_db"]
        player_db = shared_and_player_dbs["player_db"]

        with (
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
            duckdb.connect(str(player_db)) as player_conn,
        ):
            result = _get_uncited_match_ids(shared_ro, player_conn, xuid)

        assert set(result) == set(match_ids)

    def test_excludes_already_cited_matches(self, shared_and_player_dbs):
        xuid = "2222222222"
        match_ids = _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=3)
        player_db = shared_and_player_dbs["player_db"]
        shared_db = shared_and_player_dbs["shared_db"]

        # Marquer match-000 comme déjà traité
        with duckdb.connect(str(player_db)) as conn:
            conn.execute("INSERT INTO match_citations VALUES (?, '_processed', 1)", [match_ids[0]])

        with (
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
            duckdb.connect(str(player_db)) as player_conn,
        ):
            result = _get_uncited_match_ids(shared_ro, player_conn, xuid)

        assert match_ids[0] not in result
        assert set(result) == {match_ids[1], match_ids[2]}

    def test_returns_empty_when_all_cited(self, shared_and_player_dbs):
        xuid = "3333333333"
        match_ids = _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=2)
        player_db = shared_and_player_dbs["player_db"]
        shared_db = shared_and_player_dbs["shared_db"]

        with duckdb.connect(str(player_db)) as conn:
            for mid in match_ids:
                conn.execute("INSERT INTO match_citations VALUES (?, '_processed', 1)", [mid])

        with (
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
            duckdb.connect(str(player_db)) as player_conn,
        ):
            result = _get_uncited_match_ids(shared_ro, player_conn, xuid)

        assert result == []


# ---------------------------------------------------------------------------
# Tests _load_matches_split
# ---------------------------------------------------------------------------


class TestLoadMatchesSplit:
    """_load_matches_split utilise deux connexions séparées (pas d'ATTACH)."""

    def test_returns_inner_join_matches(self, shared_and_player_dbs):
        xuid = "4444444444"
        match_ids = _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=3)
        player_db = shared_and_player_dbs["player_db"]
        shared_db = shared_and_player_dbs["shared_db"]

        # Player a 2 des 3 matches
        _populate_player(player_db, match_ids[:2])

        with (
            duckdb.connect(str(player_db)) as player_conn,
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
        ):
            df = _load_matches_split(player_conn, shared_ro, xuid)

        assert len(df) == 2
        assert set(df["match_id"].to_list()) == set(match_ids[:2])
        assert "start_time" in df.columns
        assert "is_ranked" in df.columns

    def test_returns_empty_when_player_has_no_matches(self, shared_and_player_dbs):
        xuid = "5555555555"
        _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=3)
        player_db = shared_and_player_dbs["player_db"]
        shared_db = shared_and_player_dbs["shared_db"]

        with (
            duckdb.connect(str(player_db)) as player_conn,
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
        ):
            df = _load_matches_split(player_conn, shared_ro, xuid)

        assert df.is_empty()


# ---------------------------------------------------------------------------
# Tests anti-ATTACH : player_conn ne doit jamais avoir shared attaché
# ---------------------------------------------------------------------------


class TestNoAttachOnPlayerConn:
    """Garantit que player_conn n'a jamais shared ATTACH après les fonctions."""

    def test_sessions_backfill_does_not_attach_shared_to_player_conn(self, shared_and_player_dbs):
        """backfill_sessions_for_player ne doit pas ATTACH shared sur player_conn."""
        xuid = "6666666666"
        match_ids = _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=2)
        _populate_player(shared_and_player_dbs["player_db"], match_ids)

        with duckdb.connect(str(shared_and_player_dbs["player_db"])) as player_conn:
            # Avant : player_conn has no shared attached
            dbs_before = player_conn.execute(
                "SELECT database_name FROM duckdb_databases()"
            ).fetchall()
            names_before = {r[0] for r in dbs_before}

            backfill_sessions_for_player(
                shared_and_player_dbs["player_db"],
                xuid=xuid,
                conn=player_conn,
            )

            # Après : player_conn ne doit toujours PAS avoir shared attaché
            dbs_after = player_conn.execute(
                "SELECT database_name FROM duckdb_databases()"
            ).fetchall()
            names_after = {r[0] for r in dbs_after}

        # Aucune nouvelle DB ne doit avoir été attachée à player_conn
        new_attachments = names_after - names_before
        shared_attached = any("shared" in n.lower() for n in new_attachments)
        assert (
            not shared_attached
        ), f"shared_matches a été attaché à player_conn : {new_attachments}"

    def test_citations_backfill_does_not_attach_shared_to_player_conn(self, shared_and_player_dbs):
        """backfill_citations_for_player ne doit pas ATTACH shared sur player_conn."""
        xuid = "7777777777"
        _populate_shared(shared_and_player_dbs["shared_db"], xuid, n=2)

        with duckdb.connect(str(shared_and_player_dbs["player_db"])) as player_conn:
            dbs_before = {
                r[0]
                for r in player_conn.execute(
                    "SELECT database_name FROM duckdb_databases()"
                ).fetchall()
            }

            # Appel sans vraies citations (pas de metadata.duckdb) → retourne tôt
            backfill_citations_for_player(
                shared_and_player_dbs["player_db"],
                xuid=xuid,
                conn=player_conn,
            )

            dbs_after = {
                r[0]
                for r in player_conn.execute(
                    "SELECT database_name FROM duckdb_databases()"
                ).fetchall()
            }

        new_attachments = dbs_after - dbs_before
        shared_attached = any("shared" in n.lower() for n in new_attachments)
        assert (
            not shared_attached
        ), f"shared_matches a été attaché à player_conn : {new_attachments}"


# ---------------------------------------------------------------------------
# Tests de comportement gracieux si shared absent
# ---------------------------------------------------------------------------


class TestGracefulDegradation:
    def test_sessions_returns_error_if_shared_absent(self, tmp_path):
        """backfill_sessions_for_player retourne une erreur si shared est introuvable."""

        player_dir = tmp_path / "players" / "X"
        player_dir.mkdir(parents=True)
        player_db = player_dir / "stats.duckdb"
        duckdb.connect(str(player_db)).close()

        # Forcer _resolve_shared_db_path à retourner None (isole du vrai shared)
        with patch("src.data.sessions_backfill._resolve_shared_db_path", return_value=None):
            result = backfill_sessions_for_player(player_db, xuid="9999999999")

        assert "errors" in result
        assert len(result["errors"]) > 0

    def test_citations_returns_empty_if_shared_absent(self, tmp_path):
        player_dir = tmp_path / "players" / "X"
        player_dir.mkdir(parents=True)
        player_db = player_dir / "stats.duckdb"
        duckdb.connect(str(player_db)).close()

        result = backfill_citations_for_player(player_db, xuid="9999999999")
        assert result["matches_processed"] == 0
        assert result["citations_computed"] == 0


# ---------------------------------------------------------------------------
# Test : shared_matches est ouvert en read_only=True — guard anti-régression
# ---------------------------------------------------------------------------


class TestSharedOpenedReadOnly:
    def test_citations_source_uses_read_only_true(self):
        """Guard : citations_backfill.py doit appeler duckdb.connect avec read_only=True."""
        import inspect

        import src.data.citations_backfill as _mod

        src_code = inspect.getsource(_mod)
        assert "read_only=True" in src_code, (
            "citations_backfill doit ouvrir shared_matches en read_only=True. "
            "Risque de régression Axe 2 : vérifier la présence de "
            "duckdb.connect(str(shared_path), read_only=True)"
        )

    def test_sessions_source_uses_read_only_true(self):
        """Guard : sessions_backfill.py doit appeler duckdb.connect avec read_only=True."""
        import inspect

        import src.data.sessions_backfill as _mod

        src_code = inspect.getsource(_mod)
        assert "read_only=True" in src_code, (
            "sessions_backfill doit ouvrir shared_matches en read_only=True. "
            "Risque de régression Axe 2 : vérifier la présence de "
            "duckdb.connect(str(shared_path), read_only=True)"
        )
