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
Axe 4 — Bulk SQL : 6 requêtes groupées pour N matchs au lieu de N×6 (executemany INSERT).
"""

from __future__ import annotations

import contextlib
import logging
from collections import defaultdict
from contextlib import nullcontext
from pathlib import Path
from typing import Any

import duckdb
import polars as pl

from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import get_pve_db_path_from_player, get_shared_matches_path_from_player

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


# ---------------------------------------------------------------------------
# Axe 4 — Bulk data loaders (N matchs en ~5-6 requêtes au lieu de N×6)
# ---------------------------------------------------------------------------


def _bulk_medals(
    shared_ro: duckdb.DuckDBPyConnection, xuid: str, match_ids: list[str]
) -> dict[str, dict[int, int]]:
    """Charge les médailles pour N matchs en 1 requête."""
    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, dict[int, int]] = defaultdict(dict)
    with contextlib.suppress(Exception):
        for mid, medal_id, count in shared_ro.execute(
            f"SELECT match_id, medal_name_id, count FROM medals_earned "
            f"WHERE match_id IN ({ph}) AND xuid = ?",
            [*match_ids, xuid],
        ).fetchall():
            result[mid][int(medal_id)] = int(count)
    return result


def _bulk_stats(
    shared_ro: duckdb.DuckDBPyConnection, xuid: str, match_ids: list[str]
) -> dict[str, dict[str, Any]]:
    """Charge match_participants + match_registry pour N matchs en 1 requête."""
    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, dict[str, Any]] = {}
    with contextlib.suppress(Exception):
        res = shared_ro.execute(
            f"SELECT p.*, r.map_name, r.playlist_name, r.game_variant_name, r.start_time "
            f"FROM match_participants p "
            f"LEFT JOIN match_registry r ON p.match_id = r.match_id "
            f"WHERE p.match_id IN ({ph}) AND p.xuid = ?",
            [*match_ids, xuid],
        )
        cols = [d[0] for d in res.description]
        for row in res.fetchall():
            d = dict(zip(cols, row, strict=False))
            result[d["match_id"]] = d
    return result


def _bulk_highlight_events(
    shared_ro: duckdb.DuckDBPyConnection, xuid: str, match_ids: list[str]
) -> dict[str, list[tuple[int, str]]]:
    """Charge les highlight_events (mode/death) pour N matchs en 1 requête."""
    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, list[tuple[int, str]]] = defaultdict(list)
    with contextlib.suppress(Exception):
        for mid, time_ms, event_type in shared_ro.execute(
            f"SELECT match_id, time_ms, event_type FROM highlight_events "
            f"WHERE match_id IN ({ph}) AND xuid = ? "
            f"AND event_type IN ('mode', 'death') ORDER BY match_id, time_ms",
            [*match_ids, xuid],
        ).fetchall():
            result[mid].append((int(time_ms), str(event_type)))
    return result


def _bulk_weapon_kills(
    shared_ro: duckdb.DuckDBPyConnection, xuid: str, match_ids: list[str]
) -> dict[str, dict[str, int]]:
    """Charge les weapon_kills pour N matchs en 1 requête via v_weapon_kills."""
    from src.analysis._weapon_data import (
        EXCLUDED_WEAPON_IDS,
        WEAPON_FUSION_MAP_ID,
        resolve_weapon_display,
    )

    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, dict[str, int]] = defaultdict(dict)
    with contextlib.suppress(Exception):
        for mid, weapon_id, kills in shared_ro.execute(
            f"SELECT match_id, effective_weapon_id, COUNT(*) FROM v_weapon_kills "
            f"WHERE match_id IN ({ph}) AND xuid = ? AND effective_weapon_id IS NOT NULL "
            f"GROUP BY match_id, effective_weapon_id",
            [*match_ids, xuid],
        ).fetchall():
            wid = int(weapon_id)
            if wid in EXCLUDED_WEAPON_IDS:
                continue
            canonical_id = WEAPON_FUSION_MAP_ID.get(wid, wid)
            name = resolve_weapon_display(canonical_id, lang="en") or "NON TROUVE"
            result[mid][name] = result[mid].get(name, 0) + int(kills)
    return result


def _bulk_awards(
    player_conn: duckdb.DuckDBPyConnection, match_ids: list[str]
) -> dict[str, dict[str, int]]:
    """Charge les personal_score_awards pour N matchs depuis la player DB."""
    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, dict[str, int]] = defaultdict(dict)
    with contextlib.suppress(Exception):
        exists = player_conn.execute(
            "SELECT 1 FROM information_schema.tables WHERE table_name='personal_score_awards'"
        ).fetchone()
        if exists:
            for mid, award_name, count in player_conn.execute(
                f"SELECT match_id, award_name, SUM(award_count) FROM personal_score_awards "
                f"WHERE match_id IN ({ph}) GROUP BY match_id, award_name",
                match_ids,
            ).fetchall():
                result[mid][str(award_name)] = int(count)
    return result


def _bulk_pve_stats(pve_path: Path, xuid: str, match_ids: list[str]) -> dict[str, dict[str, Any]]:
    """Charge les stats PVE pour N matchs depuis shared_pve.duckdb."""
    if not pve_path.exists():
        return {}
    ph = ",".join(["?"] * len(match_ids))
    result: dict[str, dict[str, Any]] = {}
    with contextlib.suppress(Exception), duckdb_read_only(pve_path) as pve_conn:
        res = pve_conn.execute(
            f"SELECT * FROM pve_match_stats WHERE match_id IN ({ph}) AND xuid = ?",
            [*match_ids, xuid],
        )
        cols = [d[0] for d in res.description]
        for row in res.fetchall():
            d = dict(zip(cols, row, strict=False))
            result[d["match_id"]] = d
    return result


def _build_match_data_map(
    shared_ro: duckdb.DuckDBPyConnection,
    player_conn: duckdb.DuckDBPyConnection,
    xuid: str,
    match_ids: list[str],
    pve_path: Path | None,
) -> dict[str, dict[str, Any]]:
    """Bulk-charge toutes les données pour N matchs (~6 requêtes SQL)."""
    all_medals = _bulk_medals(shared_ro, xuid, match_ids)
    all_stats = _bulk_stats(shared_ro, xuid, match_ids)
    all_events = _bulk_highlight_events(shared_ro, xuid, match_ids)
    all_wkills = _bulk_weapon_kills(shared_ro, xuid, match_ids)
    all_awards = _bulk_awards(player_conn, match_ids)
    all_pve = _bulk_pve_stats(pve_path, xuid, match_ids) if pve_path else {}

    out: dict[str, dict[str, Any]] = {}
    for match_id in match_ids:
        stats = all_stats.get(match_id, {})
        merged_stats = {
            **stats,
            **(all_pve.get(match_id) or {}),
            **{f"weapon_kills:{n}": k for n, k in all_wkills.get(match_id, {}).items()},
        }
        out[match_id] = {
            "medals": all_medals.get(match_id, {}),
            "stats": merged_stats,
            "awards": all_awards.get(match_id, {}),
            "df": pl.DataFrame([stats]) if stats else pl.DataFrame(),
            "events": all_events.get(match_id, []),
        }
    return out


def _process_citations_batch(
    db_path: Path,
    xuid: str,
    shared_path: Path,
    match_ids: list[str],
) -> dict[str, int]:
    """Traite les citations via bulk SQL + executemany INSERT (Axe 4).

    Bulk-charge toutes les données match en ~6 requêtes (au lieu de N×6),
    calcule en Python, insère en une seule passe.
    Plus d'ATTACH sur cit_conn depuis Axe 4 — shared_ro en R/O direct.
    """
    from src.analysis.citations.engine import CitationEngine

    pve_path = get_pve_db_path_from_player(db_path)
    citations_computed = 0

    with duckdb_read_write(str(db_path)) as cit_conn:
        _create_match_citations_if_needed(cit_conn)

        # Axe 4 : engine sans shared_conn — le bulk loader gère les lectures
        engine = CitationEngine(str(db_path), xuid, shared_db_path=False)
        if not engine.load_mappings():
            logger.debug("citations_backfill: aucun mapping disponible")
            return dict(_EMPTY_RESULT)

        # 6 requêtes groupées pour N matchs
        with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
            match_data = _build_match_data_map(shared_ro, cit_conn, xuid, match_ids, pve_path)

        # Calcul Python pur (0 SQL), collecte des lignes
        rows_to_insert: list[tuple[str, str, int]] = []
        for match_id in match_ids:
            data = match_data.get(match_id, {})
            try:
                citations = engine.compute_all_for_match(
                    match_id,
                    match_medals=data.get("medals", {}),
                    match_stats=data.get("stats", {}),
                    match_awards=data.get("awards", {}),
                    df_match=data.get("df"),
                    highlight_events=data.get("events"),
                )
            except Exception as exc:
                logger.debug("citations_backfill: skip %s → %s", match_id[:8], exc)
                citations = {}
            for norm_name, value in citations.items():
                rows_to_insert.append((match_id, norm_name, value))
            rows_to_insert.append((match_id, "_processed", 1))
            if citations:
                citations_computed += 1

        # 1 seul executemany INSERT au lieu de N individuels
        if rows_to_insert:
            cit_conn.executemany(
                "INSERT OR REPLACE INTO match_citations "
                "(match_id, citation_name_norm, value) VALUES (?, ?, ?)",
                rows_to_insert,
            )
            cit_conn.commit()

    logger.info(
        "citations_backfill: ✅ %d/%d matchs avec citations (batch+executemany)",
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
        logger.debug("citations_backfill: shared_matches_v2.duckdb absent, skip")
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
