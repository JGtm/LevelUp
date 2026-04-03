"""Logique de backfill des sessions (sans Streamlit).

Utilisé par scripts/backfill_data.py --sessions pour charger les amis
et calculer/stocker session_id et session_label dans match_stats.
"""

from __future__ import annotations

import contextlib
import json
import logging
from contextlib import nullcontext
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import duckdb
import polars as pl

from src.data.sessions_backfill_shared import (
    _find_or_attach_shared,
    _load_matches_split,
    _resolve_shared_db_path,
)
from src.utils.db import duckdb_read_only, duckdb_read_write
from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)


def get_friends_xuids_for_backfill(  # noqa: C901, PLR0912
    db_path: str | Path,
    self_xuid: str,
    *,
    conn: duckdb.DuckDBPyConnection | None = None,
) -> frozenset[str]:
    """Charge les XUIDs des amis (friends_defaults.json) pour le backfill.

    Args:
        db_path: Chemin vers la DB.
        self_xuid: XUID du joueur.
        conn: Connexion existante (évite conflit DuckDB multi-connexions). Si None, ouvre une nouvelle.

    Returns:
        Set des XUIDs des amis. Vide si pas de config ou résolution échouée.
    """
    path = Path(db_path)
    if not path.exists():
        return frozenset()

    # Charger friends_defaults.json
    friends_json = Path(__file__).resolve().parents[2] / ".streamlit" / "friends_defaults.json"
    if not friends_json.exists():
        return frozenset()

    try:
        with open(friends_json, encoding="utf-8") as f:
            data = json.load(f)
    except Exception:
        return frozenset()

    if not isinstance(data, dict):
        return frozenset()

    friends_raw = data.get(str(self_xuid).strip())
    if not friends_raw or not isinstance(friends_raw, list):
        return frozenset()

    ctx = duckdb_read_only(str(path)) if conn is None else nullcontext(conn)
    with ctx as conn:
        # V5.1: Utiliser shared.xuid_aliases si disponible
        alias_table = "xuid_aliases"
        try:
            conn.execute("SELECT 1 FROM shared.xuid_aliases LIMIT 1")
            alias_table = "shared.xuid_aliases"
        except Exception:
            pass

        result = conn.execute(
            f"SELECT xuid, gamertag FROM {alias_table} WHERE xuid IS NOT NULL AND gamertag IS NOT NULL"
        ).fetchall()
        gamertag_to_xuid: dict[str, str] = {}
        for xuid, gamertag in result:
            if xuid and gamertag:
                gamertag_to_xuid[str(gamertag).strip().casefold()] = str(xuid).strip()

        xuids: set[str] = set()
        known_xuids = {
            str(r[0]).strip()
            for r in conn.execute(
                f"SELECT DISTINCT xuid FROM {alias_table} WHERE xuid IS NOT NULL"
            ).fetchall()
            if r[0]
        }

        for ident in friends_raw:
            s = str(ident or "").strip()
            if not s or s.startswith("~"):
                continue  # vide ou ami inactif (~prefix)
            if s.isdigit() and s in known_xuids:
                xuids.add(s)
                continue
            xu = gamertag_to_xuid.get(s.casefold())
            if xu:
                xuids.add(xu)
        return frozenset(xuids)


def get_top_two_teammate_xuids(
    db_path: str | Path,
    self_xuid: str,
    limit: int = 2,
    *,
    conn: duckdb.DuckDBPyConnection | None = None,
) -> frozenset[str]:
    """Top N coéquipiers depuis shared.match_participants."""
    path = Path(db_path)
    if not path.exists():
        return frozenset()

    # Trouver shared_matches
    shared_db = get_shared_matches_path_from_player(path)
    if shared_db is None:
        return frozenset()

    ctx = duckdb_read_only(str(path)) if conn is None else nullcontext(conn)
    with ctx as conn:
        # Attacher shared en read-only (DuckDB ne supporte pas ? pour ATTACH)
        with contextlib.suppress(Exception):
            conn.execute(f"ATTACH '{shared_db}' AS shared_tmp (READ_ONLY)")

        try:
            result = conn.execute(
                """
                SELECT mp2.xuid, COUNT(DISTINCT mp2.match_id) AS match_count
                FROM shared_tmp.match_participants mp1
                JOIN shared_tmp.match_participants mp2
                  ON mp1.match_id = mp2.match_id
                 AND mp1.xuid != mp2.xuid
                 AND mp1.team_id = mp2.team_id
                WHERE mp1.xuid = ?
                GROUP BY mp2.xuid
                ORDER BY match_count DESC
                LIMIT ?
                """,
                [str(self_xuid).strip(), limit],
            ).fetchall()
        except Exception:
            return frozenset()
        finally:
            with contextlib.suppress(Exception):
                conn.execute("DETACH shared_tmp")

        return frozenset(str(r[0]).strip() for r in result if r[0])


def _ensure_enrichment_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée/migre la table player_match_enrichment si nécessaire."""
    conn.execute("""
        CREATE TABLE IF NOT EXISTS player_match_enrichment (
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
    with contextlib.suppress(Exception):
        conn.execute(
            "ALTER TABLE player_match_enrichment ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
        )
    with contextlib.suppress(Exception):
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id)"
        )


def _upsert_session_rows(  # noqa: PLR0912
    conn: duckdb.DuckDBPyConnection,
    df_sessions: pl.DataFrame,
    friends_set: frozenset[str],
    *,
    include_recent: bool,
    threshold: float,
) -> tuple[int, int, list[str]]:
    """Upsert les lignes de sessions dans player_match_enrichment.

    Retourne (updated, skipped, errors).
    """
    from src.analysis.sessions import _parse_teammates_signature

    has_friends_config = len(friends_set) > 0
    updated = 0
    skipped = 0
    errors: list[str] = []

    for row in df_sessions.iter_rows(named=True):
        match_id = row["match_id"]
        session_id = row["session_id"]
        session_label = row["session_label"]
        st_ts = row.get("_ts")

        if st_ts is None:
            continue
        if not include_recent and st_ts > threshold:
            skipped += 1
            continue

        if has_friends_config:
            sig = row.get("teammates_signature")
            team_set = _parse_teammates_signature(sig)
            common = team_set & friends_set
            iwf: bool | None = bool(common)
            ktc: int | None = len(common)
            fx: str | None = ",".join(sorted(common)) if common else None
        else:
            iwf = None
            ktc = None
            fx = None

        try:
            conn.execute(
                "INSERT INTO player_match_enrichment "
                "(match_id, session_id, session_label, is_with_friends, known_teammates_count, friends_xuids, teammates_signature) "
                "VALUES (?, ?, ?, ?, ?, ?, ?) "
                "ON CONFLICT (match_id) DO UPDATE SET "
                "session_id = EXCLUDED.session_id, "
                "session_label = EXCLUDED.session_label, "
                "teammates_signature = COALESCE(EXCLUDED.teammates_signature, player_match_enrichment.teammates_signature), "
                "is_with_friends = COALESCE(EXCLUDED.is_with_friends, player_match_enrichment.is_with_friends), "
                "known_teammates_count = COALESCE(EXCLUDED.known_teammates_count, player_match_enrichment.known_teammates_count), "
                "friends_xuids = COALESCE(EXCLUDED.friends_xuids, player_match_enrichment.friends_xuids), "
                "updated_at = now()",
                [
                    match_id,
                    session_id,
                    str(session_label) if session_label else None,
                    iwf,
                    ktc,
                    fx,
                    sig if has_friends_config else row.get("teammates_signature"),
                ],
            )
            updated += 1
        except Exception as e:
            errors.append(f"{match_id}: {e}")

    return updated, skipped, errors


def _top_teammates_ro(
    shared_ro: duckdb.DuckDBPyConnection,
    xuid: str,
    limit: int = 2,
) -> frozenset[str]:
    """Top N coéquipiers depuis shared_ro direct (pas d'ATTACH)."""
    try:
        rows = shared_ro.execute(
            "SELECT mp2.xuid, COUNT(DISTINCT mp2.match_id) AS cnt"
            " FROM match_participants mp1"
            " JOIN match_participants mp2"
            "   ON mp1.match_id=mp2.match_id AND mp1.team_id=mp2.team_id AND mp1.xuid<>mp2.xuid"
            " WHERE mp1.xuid=? GROUP BY mp2.xuid ORDER BY cnt DESC LIMIT ?",
            [xuid, limit],
        ).fetchall()
        return frozenset(str(r[0]).strip() for r in rows if r[0])
    except Exception:
        return frozenset()


def _fetch_shared_context_ro(
    player_conn: duckdb.DuckDBPyConnection,
    path: Path,
    xuid: str | None,
    shared_ro: duckdb.DuckDBPyConnection,
) -> tuple[str | None, pl.DataFrame, frozenset[str], list[str]]:
    """Résout xuid, charge amis et matchs via shared_ro direct (Option A — pas d'ATTACH)."""
    if not xuid:
        row = shared_ro.execute(
            "SELECT DISTINCT xuid FROM xuid_aliases WHERE xuid IS NOT NULL LIMIT 1"
        ).fetchone()
        xuid = str(row[0]).strip() if row and row[0] else None
    if not xuid:
        return None, pl.DataFrame(), frozenset(), ["XUID non trouvé"]

    # get_friends_xuids_for_backfill fonctionne avec shared_ro car il lit
    # xuid_aliases sans le préfixe 'shared.' (fallback alias_table = "xuid_aliases")
    friends = get_friends_xuids_for_backfill(path, xuid, conn=shared_ro)
    if not friends:
        friends = _top_teammates_ro(shared_ro, xuid)

    df = _load_matches_split(player_conn, shared_ro, xuid)
    return xuid, df, frozenset(friends) if friends else frozenset(), []


def _fetch_shared_context(
    conn: duckdb.DuckDBPyConnection,
    path: Path,
    xuid: str | None,
) -> tuple[str | None, pl.DataFrame, frozenset[str], list[str]]:
    """Attache shared, résout xuid, charge amis et matchs (legacy — ATTACH).

    Conservé pour backfill_teammates_signatures (pas dans le chemin critique sync).
    Pour backfill_sessions_for_player, utiliser _fetch_shared_context_ro.
    """
    errors: list[str] = []
    shared_db = _resolve_shared_db_path(path)
    if not shared_db:
        return None, pl.DataFrame(), frozenset(), ["shared_matches_v2.duckdb introuvable"]

    shared_alias, owns_shared_attach = _find_or_attach_shared(conn, shared_db)
    if not shared_alias:
        return None, pl.DataFrame(), frozenset(), ["Impossible d'attacher shared"]

    if not xuid:
        row = conn.execute(
            f"SELECT DISTINCT xuid FROM {shared_alias}.xuid_aliases WHERE xuid IS NOT NULL LIMIT 1"
        ).fetchone()
        xuid = str(row[0]).strip() if row and row[0] else None
    if not xuid:
        if owns_shared_attach:
            with contextlib.suppress(Exception):
                conn.execute(f"DETACH {shared_alias}")
        return None, pl.DataFrame(), frozenset(), ["XUID non trouvé"]

    friends = get_friends_xuids_for_backfill(path, xuid, conn=conn)
    if not friends:
        friends = get_top_two_teammate_xuids(path, xuid, limit=2, conn=conn)

    from src.data.sessions_backfill_shared import _load_matches_from_shared

    df = _load_matches_from_shared(conn, shared_alias, xuid)

    if owns_shared_attach:
        with contextlib.suppress(Exception):
            conn.execute(f"DETACH {shared_alias}")

    return xuid, df, frozenset(friends) if friends else frozenset(), errors


def _dry_run_count(df_sessions: pl.DataFrame, include_recent: bool, threshold: float) -> int:
    """Compte les matchs qui seraient mis à jour en mode dry-run."""
    to_update = df_sessions.filter(~pl.col("_ts").is_null())
    if not include_recent:
        to_update = to_update.filter(pl.col("_ts") <= threshold)
    return len(to_update)


def backfill_sessions_for_player(  # noqa: PLR0913
    db_path: Path | str,
    xuid: str | None = None,
    *,
    conn: duckdb.DuckDBPyConnection | None = None,
    gap_minutes: int = 120,
    include_recent: bool = True,
    force: bool = False,
    dry_run: bool = False,
) -> dict[str, Any]:
    """Backfill session_id et session_label dans player_match_enrichment (V5 final).

    Lit les matchs depuis shared.match_participants + match_registry.
    Écrit les sessions dans player_match_enrichment.
    """
    from src.analysis.sessions import compute_sessions_with_context_polars
    from src.config import SESSION_CONFIG

    path = Path(db_path)
    results: dict[str, Any] = {"updated": 0, "skipped_recent": 0, "errors": []}

    if not path.exists():
        results["errors"].append("DB non trouvée")
        return results

    shared_path = _resolve_shared_db_path(path)
    if shared_path is None:
        results["errors"].append("shared_matches_v2.duckdb introuvable")
        return results

    ctx = duckdb_read_write(str(path)) if conn is None else nullcontext(conn)
    with ctx as conn:
        _ensure_enrichment_table(conn)

        with duckdb.connect(str(shared_path), read_only=True) as shared_ro:
            xuid, df, friends_set, fetch_errors = _fetch_shared_context_ro(
                conn, path, xuid, shared_ro
            )
        results["errors"].extend(fetch_errors)
        if not xuid or df.is_empty():
            return results

        if not force and df.filter(pl.col("session_id").is_null()).is_empty():
            return results

        cols = ["match_id", "start_time", "teammates_signature"]
        if "is_ranked" in df.columns:
            cols.append("is_ranked")
        df_sessions = compute_sessions_with_context_polars(
            df.select(cols),
            gap_minutes=gap_minutes,
            friends_xuids=friends_set if friends_set else None,
            ranked_column="is_ranked" if "is_ranked" in df.columns else None,
        )

        threshold = datetime.now(timezone.utc).timestamp() - (
            SESSION_CONFIG.session_stability_hours * 3600
        )
        df_sessions = df_sessions.with_columns(
            pl.col("start_time").dt.epoch(time_unit="s").alias("_ts")
        )

        if dry_run:
            results["updated"] = _dry_run_count(df_sessions, include_recent, threshold)
            return results

        updated, skipped, errors = _upsert_session_rows(
            conn,
            df_sessions,
            friends_set,
            include_recent=include_recent,
            threshold=threshold,
        )
        results["updated"] = updated
        results["skipped_recent"] = skipped
        results["errors"].extend(errors)
        conn.commit()

    return results


def backfill_teammates_signatures(
    db_path: str | Path,
    xuid: str | None = None,
    *,
    force: bool = False,
) -> dict:
    """Backfille teammates_signature + is_with_friends depuis shared_matches."""
    results: dict = {"updated": 0, "errors": []}
    path = Path(db_path)
    if not path.exists():
        results["errors"].append(f"DB non trouvée : {path}")
        return results
    with duckdb_read_write(str(path)) as conn:
        _ensure_enrichment_table(conn)
        shared_db = _resolve_shared_db_path(path)
        if not shared_db:
            results["errors"].append("shared_matches_v2.duckdb introuvable")
            return results
        shared_alias, _ = _find_or_attach_shared(conn, shared_db)
        if not shared_alias:
            results["errors"].append("Impossible d'attacher shared")
            return results
        if not xuid:
            row = conn.execute(f"SELECT xuid FROM {shared_alias}.xuid_aliases LIMIT 1").fetchone()
            xuid = str(row[0]).strip() if row and row[0] else None
            if not xuid:
                results["errors"].append("XUID non trouvé")
                return results
        friends_set = get_friends_xuids_for_backfill(path, xuid, conn=conn)
        nf = "" if force else "WHERE pme.teammates_signature IS NULL"
        a = shared_alias
        sigs = conn.execute(
            f"SELECT pme.match_id, COALESCE("
            f" (SELECT string_agg(o.xuid ORDER BY o.xuid) FROM {a}.match_participants me"
            f"  JOIN {a}.match_participants o ON o.match_id=me.match_id"
            f"  AND o.team_id=me.team_id AND o.xuid!=? WHERE me.match_id=pme.match_id AND me.xuid=?),"
            f" (SELECT string_agg(o.xuid ORDER BY o.xuid) FROM {a}.match_participants o"
            f"  WHERE o.match_id=pme.match_id AND o.xuid!=?)) AS sig"
            f" FROM player_match_enrichment pme {nf}",
            [xuid, xuid, xuid],
        ).fetchall()
        for match_id, sig in sigs:
            iwf = ktc = fx = None
            if friends_set and sig:
                common = set(sig.split(",")) & friends_set
                iwf, ktc = bool(common), len(common)
                fx = ",".join(sorted(common)) if common else None
            conn.execute(
                "UPDATE player_match_enrichment SET "
                "teammates_signature=?, is_with_friends=?, "
                "known_teammates_count=?, friends_xuids=?, updated_at=now() "
                "WHERE match_id=?",
                [sig, iwf, ktc, fx, match_id],
            )
            results["updated"] += 1
        conn.commit()
    return results
