"""Helpers d'attachement de la base partagée pour le backfill des sessions."""

from __future__ import annotations

import logging
from pathlib import Path

import duckdb
import polars as pl

from src.utils.paths import get_shared_matches_path, get_shared_matches_path_from_player

logger = logging.getLogger(__name__)


def _find_or_attach_shared(  # noqa: C901, PLR0912
    conn: duckdb.DuckDBPyConnection,
    shared_db: Path,
) -> tuple[str | None, bool]:
    """Trouve un alias shared existant ou attache la DB.

    Retourne (alias, owns_attach). alias=None si impossible.
    """
    # Chercher via le path ou le nom
    try:
        dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
        shared_db_normalized = str(shared_db.resolve()).lower().replace("\\", "/")
        for db_name, db_path_val in dbs:
            if db_path_val:
                db_path_normalized = str(db_path_val).lower().replace("\\", "/")
                if (
                    shared_db_normalized in db_path_normalized
                    or db_path_normalized in shared_db_normalized
                ):
                    return db_name, False
            if db_name and "shared" in db_name.lower():
                try:
                    conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                    return db_name, False
                except Exception:
                    continue
    except Exception:
        pass

    # Chercher n'importe quelle DB avec match_participants
    try:
        dbs = conn.execute("SELECT database_name FROM duckdb_databases()").fetchall()
        for (db_name,) in dbs:
            if db_name not in ("memory", "temp", "main", "stats", "system"):
                try:
                    conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                    return db_name, False
                except Exception:
                    continue
    except Exception:
        pass

    # Dernier recours : attacher
    try:
        # DuckDB ne supporte pas les placeholders ? pour ATTACH
        conn.execute(f"ATTACH '{shared_db}' AS shared (READ_ONLY)")
        return "shared", True
    except Exception:
        return None, False


def _resolve_shared_db_path(player_db_path: Path) -> Path | None:
    """Trouve le chemin vers shared_matches (v6 : shared_matches_v2.duckdb)."""
    detected = get_shared_matches_path_from_player(player_db_path)
    if detected is not None:
        logger.debug("shared_matches détecté depuis player → %s", detected)
        return detected
    p = get_shared_matches_path()
    if p.exists():
        logger.debug("shared_matches fallback global → %s", p)
        return p
    logger.warning("shared_matches introuvable pour %s", player_db_path)
    return None


def _load_matches_from_shared(
    conn: duckdb.DuckDBPyConnection, shared_alias: str, xuid: str
) -> pl.DataFrame:
    """Charge depuis PME. Sig reconstruite : même équipe si joueur dans shared, sinon tous (best-effort)."""
    a = shared_alias
    s1 = (
        f"SELECT string_agg(o.xuid ORDER BY o.xuid) FROM {a}.match_participants me"
        f" JOIN {a}.match_participants o ON o.match_id=me.match_id"
        f" AND o.team_id=me.team_id AND o.xuid!=? WHERE me.match_id=pme.match_id AND me.xuid=?"
    )
    s2 = (
        f"SELECT string_agg(o.xuid ORDER BY o.xuid) FROM {a}.match_participants o"
        f" WHERE o.match_id=pme.match_id AND o.xuid!=?"
    )
    return conn.execute(
        f"SELECT pme.match_id, mr.start_time,"
        f" COALESCE(pme.teammates_signature, ({s1}), ({s2})) AS teammates_signature,"
        f" COALESCE(mr.is_ranked, FALSE) AS is_ranked,"
        f" pme.session_id FROM player_match_enrichment pme"
        f" LEFT JOIN {a}.match_registry mr ON mr.match_id=pme.match_id"
        f" WHERE mr.start_time IS NOT NULL ORDER BY mr.start_time ASC",
        [xuid, xuid, xuid],
    ).pl()
