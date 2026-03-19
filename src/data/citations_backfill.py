"""Backfill des citations par match — appelé post-sync.

Module analogue à :mod:`src.data.sessions_backfill`.
Calcule les citations manquantes pour un joueur en mode incrémental (DB-only,
aucun appel API), à partir des données déjà présentes dans :

- ``shared.match_participants``  (kills, deaths, rank, …)
- ``shared.medals_earned``       (médailles)
- ``shared.highlight_events``    (events filmés)
- ``personal_score_awards``      (ObjectiveTypes — player DB)

Utilisé par :class:`src.data.sync.engine.DuckDBSyncEngine` après chaque sync
pour que les matchs nouvellement insérés aient toujours leurs citations.

Axe 2 — Option A : shared_matches ouvert en R/O direct (pas d'ATTACH sur player_conn).
Élimine le conflit de handle entre self._shared_connection (R/W) et un ATTACH R/O
depuis la connexion joueur.
"""

from __future__ import annotations

import contextlib
import logging
from contextlib import nullcontext
from pathlib import Path
from typing import Any

import duckdb

from src.utils.db import duckdb_read_write, ensure_shared_attached
from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)

_EMPTY_RESULT: dict[str, int] = {"matches_processed": 0, "citations_computed": 0}


def _create_match_citations_if_needed(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table match_citations si absente (idempotent)."""
    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_citations (
            match_id VARCHAR NOT NULL,
            citation_name_norm VARCHAR NOT NULL,
            value INTEGER DEFAULT 0,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)


def _get_uncited_match_ids(
    shared_ro: duckdb.DuckDBPyConnection,
    player_conn: duckdb.DuckDBPyConnection,
    xuid: str,
) -> list[str]:
    """Retourne les match_ids non encore cités pour ce joueur.

    Utilise shared_ro (R/O direct, pas d'ATTACH) pour lire les matchs shared,
    et player_conn pour filtrer ceux déjà traités dans match_citations.
    """
    rows = shared_ro.execute(
        "SELECT DISTINCT mp.match_id "
        "FROM match_participants mp "
        "JOIN match_registry mr ON mp.match_id = mr.match_id "
        "WHERE mp.xuid = ? "
        "ORDER BY mr.start_time",
        [xuid],
    ).fetchall()

    already_done: set[str] = set()
    with contextlib.suppress(Exception):
        already_done = {
            r[0] for r in player_conn.execute("SELECT match_id FROM match_citations").fetchall()
        }
    return [r[0] for r in rows if r[0] not in already_done]


def _process_citations_batch(
    db_path: Path,
    xuid: str,
    shared_path: Path,
    match_ids: list[str],
) -> dict[str, int]:
    """Traite les citations via une connexion dédiée + ATTACH shared R/O.

    Ouvre cit_conn (R/W sur player_db) séparé de la connexion engine.py,
    ATTACH shared en R/O → CitationEngine lit shared + écrit player via cit_conn.
    """
    from src.analysis.citations.engine import CitationEngine

    citations_computed = 0
    with duckdb_read_write(str(db_path)) as cit_conn:
        _create_match_citations_if_needed(cit_conn)
        shared_alias = ensure_shared_attached(
            cit_conn, shared_path, ("shared", "shared_ro", "shared_cit")
        )
        if shared_alias is None:
            logger.warning("citations_backfill: impossible d'attacher shared à cit_conn")
            return dict(_EMPTY_RESULT)

        # try/finally garantit le DETACH même sur retour anticipé.
        # DuckDB partage le catalogue entre toutes les connexions au même fichier :
        # un ATTACH sur cit_conn est visible depuis player_conn → on DETACH avant exit.
        try:
            engine = CitationEngine(str(db_path), xuid, conn=cit_conn)
            if not engine.load_mappings():
                logger.debug("citations_backfill: aucun mapping disponible")
                return dict(_EMPTY_RESULT)

            for i, match_id in enumerate(match_ids):
                try:
                    n = engine.compute_and_store_for_match(match_id, conn=cit_conn)
                    if n > 0:
                        citations_computed += 1
                except Exception as exc:
                    logger.debug("citations_backfill: skip %s… → %s", match_id[:8], exc)
                if i > 0 and i % 50 == 0:
                    logger.info(
                        "  [%d/%d] %d matchs traités", i, len(match_ids), citations_computed
                    )
                    cit_conn.commit()
        finally:
            with contextlib.suppress(Exception):
                cit_conn.execute(f"DETACH {shared_alias}")

    logger.info(
        "citations_backfill: ✅ %d/%d matchs avec citations",
        citations_computed,
        len(match_ids),
    )
    return {"matches_processed": len(match_ids), "citations_computed": citations_computed}


def backfill_citations_for_player(
    db_path: Path | str,
    xuid: str,
    *,
    conn: duckdb.DuckDBPyConnection | None = None,
) -> dict[str, Any]:
    """Calcule les citations manquantes pour un joueur (mode incrémental).

    Traite uniquement les matchs sans entrée dans ``match_citations``.
    Ne génère aucun appel API.

    Axe 2 — Option A : shared_matches ouvert en R/O direct (pas d'ATTACH sur conn).
    """
    db_path = Path(db_path) if not isinstance(db_path, Path) else db_path
    shared_path = get_shared_matches_path_from_player(db_path)

    if shared_path is None:
        logger.debug("citations_backfill: shared_matches.duckdb absent, skip")
        return dict(_EMPTY_RESULT)

    logger.debug("citations_backfill: shared_matches résolu → %s", shared_path)

    ctx = duckdb_read_write(str(db_path)) if conn is None else nullcontext(conn)
    with ctx as player_conn:
        _create_match_citations_if_needed(player_conn)

        # Option A : ouvrir shared_ro direct — pas d'ATTACH sur player_conn
        with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
            logger.debug("citations_backfill: shared ouvert en R/O direct (pas d'ATTACH)")
            match_ids = _get_uncited_match_ids(shared_ro, player_conn, xuid)

        if not match_ids:
            logger.debug("citations_backfill: aucun match sans citations")
            return dict(_EMPTY_RESULT)

        logger.info("citations_backfill (%s…): %d match(s) à traiter", xuid[:8], len(match_ids))

    return _process_citations_batch(db_path, xuid, shared_path, match_ids)
