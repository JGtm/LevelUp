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
"""

from __future__ import annotations

import contextlib
import logging
from pathlib import Path
from typing import Any

import duckdb

logger = logging.getLogger(__name__)


def backfill_citations_for_player(
    db_path: Path | str,
    xuid: str,
    *,
    conn: duckdb.DuckDBPyConnection | None = None,
) -> dict[str, Any]:
    """Calcule les citations manquantes pour un joueur (mode incrémental).

    Traite uniquement les matchs sans entrée ``_processed`` dans
    ``match_citations``.  Ne génère aucun appel API.

    Args:
        db_path: Chemin vers la DB joueur (``stats.duckdb``).
        xuid: XUID du joueur.
        conn: Connexion DuckDB ouverte en écriture (réutilisée si fournie).
              La connexion shared doit être **fermée** avant l'appel, car
              cette fonction attache ``shared_matches.duckdb`` en READ_ONLY.

    Returns:
        ``{"matches_processed": int, "citations_computed": int}``
    """
    from src.analysis.citations.engine import CitationEngine

    db_path = Path(db_path) if not isinstance(db_path, Path) else db_path
    shared_path = db_path.parent.parent.parent / "warehouse" / "shared_matches.duckdb"

    if not shared_path.exists():
        logger.debug("citations_backfill: shared_matches.duckdb absent, skip")
        return {"matches_processed": 0, "citations_computed": 0}

    own_conn = conn is None
    if own_conn:
        conn = duckdb.connect(str(db_path))

    try:
        # ── 1. S'assurer que match_citations existe ──────────────────────────
        conn.execute("""
            CREATE TABLE IF NOT EXISTS match_citations (
                match_id VARCHAR NOT NULL,
                citation_name_norm VARCHAR NOT NULL,
                value INTEGER DEFAULT 0,
                PRIMARY KEY (match_id, citation_name_norm)
            )
        """)

        # ── 2. Attacher shared en READ_ONLY ──────────────────────────────────
        shared_alias = _ensure_shared_attached(conn, shared_path)
        if shared_alias is None:
            logger.warning("citations_backfill: impossible d'attacher shared_matches")
            return {"matches_processed": 0, "citations_computed": 0}

        # ── 3. Vérifier que les mappings de citations existent ───────────────
        engine = CitationEngine(str(db_path), xuid, conn=conn)
        if not engine.load_mappings():
            logger.debug("citations_backfill: aucun mapping de citations disponible")
            return {"matches_processed": 0, "citations_computed": 0}

        # ── 4. Identifier les matchs sans marqueur _processed ────────────────
        match_ids: list[str] = [
            r[0]
            for r in conn.execute(
                f"SELECT DISTINCT mp.match_id "
                f"FROM {shared_alias}.match_participants mp "
                f"JOIN {shared_alias}.match_registry mr ON mp.match_id = mr.match_id "
                f"WHERE mp.xuid = ? "
                f"  AND NOT EXISTS ("
                f"    SELECT 1 FROM match_citations mc WHERE mc.match_id = mp.match_id"
                f"  ) "
                f"ORDER BY mr.start_time",
                [xuid],
            ).fetchall()
        ]

        if not match_ids:
            logger.debug("citations_backfill: aucun match sans citations")
            return {"matches_processed": 0, "citations_computed": 0}

        logger.info(f"citations_backfill ({xuid[:8]}…): {len(match_ids)} match(s) à traiter")

        # ── 5. Calculer et insérer les citations ────────────────────────────
        citations_computed = 0
        for i, match_id in enumerate(match_ids):
            try:
                n = engine.compute_and_store_for_match(match_id, conn=conn)
                if n > 0:
                    citations_computed += 1
            except Exception as exc:
                logger.debug(f"citations_backfill: skip {match_id[:8]}… → {exc}")
            if i > 0 and i % 50 == 0:
                logger.info(f"  [{i}/{len(match_ids)}] {citations_computed} matchs traités")

        logger.info(
            f"citations_backfill: ✅ {citations_computed}/{len(match_ids)} matchs "
            f"avec citations"
        )
        return {
            "matches_processed": len(match_ids),
            "citations_computed": citations_computed,
        }

    finally:
        if own_conn and conn is not None:
            with contextlib.suppress(Exception):
                conn.close()


def _ensure_shared_attached(
    conn: duckdb.DuckDBPyConnection,
    shared_path: Path,
) -> str | None:
    """Vérifie si shared_matches est déjà attaché, sinon l'attache en READ_ONLY.

    Args:
        conn: Connexion DuckDB de la player DB.
        shared_path: Chemin absolu vers shared_matches.duckdb.

    Returns:
        Nom de l'alias sous lequel shared est accessible, ou ``None`` en cas d'échec.
    """
    # Chercher un alias existant pointant vers shared_matches.duckdb
    with contextlib.suppress(Exception):
        dbs = conn.execute("SELECT database_name, path FROM duckdb_databases()").fetchall()
        for db_name, db_path_val in dbs:
            if db_path_val and "shared_matches.duckdb" in str(db_path_val).lower():
                return db_name
            # Alias contenant "shared" avec match_participants accessible ?
            if db_name and "shared" in db_name.lower():
                with contextlib.suppress(Exception):
                    conn.execute(f"SELECT 1 FROM {db_name}.match_participants LIMIT 1")
                    return db_name

    # Tentative d'attachement READ_ONLY
    for alias in ("shared", "shared_ro", "shared_cit"):
        with contextlib.suppress(Exception):
            conn.execute(f"ATTACH '{shared_path}' AS {alias} (READ_ONLY)")
            return alias

    return None
