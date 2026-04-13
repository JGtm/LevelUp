"""Service Coéquipiers — chargement des équipiers et calcul des KPIs comparatifs."""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime
from pathlib import Path

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.teammates import (
    TeammateKPIs,
    TeammateOption,
    TeammateRow,
    TeammatesPageResponse,
    TeammatesQueryRequest,
)

logger = logging.getLogger(__name__)

_OUTCOME_WIN = 2
_TOP_TEAMMATES_LIMIT = 50


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def get_teammates_page(
    player: PlayerContext, request: TeammatesQueryRequest
) -> TeammatesPageResponse:
    """Retourne la page coéquipiers avec options et statistiques comparatives."""
    xuid = player.xuid
    if not xuid:
        xuid = _resolve_xuid_from_db(player)

    if not xuid:
        return TeammatesPageResponse(options=[], teammates=[], solo_reference=None, total_matches=0)

    options = _load_teammate_options(player, xuid)
    total_all = _load_total_match_count(player, xuid)
    solo_ref = _load_solo_kpis(player)

    selected = request.selected_gamertags or []
    teammates: list[TeammateRow] = []
    for opt in options:
        selected_gt = [s.lower() for s in selected]
        if selected and opt.gamertag.lower() not in selected_gt:
            continue
        row = _build_teammate_row(player, xuid, opt)
        teammates.append(row)

    return TeammatesPageResponse(
        options=options,
        teammates=teammates,
        solo_reference=solo_ref,
        total_matches=total_all,
    )


# ---------------------------------------------------------------------------
# Chargement options coéquipiers
# ---------------------------------------------------------------------------


def _load_teammate_options(player: PlayerContext, xuid: str) -> list[TeammateOption]:
    """Charge la liste des coéquipiers les plus fréquents."""
    try:
        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)

        if not shared_path.exists():
            return []

        with duckdb_read_only(str(db_path)) as conn:
            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            result = conn.execute(
                """
                SELECT
                    COALESCE(gl.gamertag, t.gamertag, t.xuid)    AS gamertag,
                    t.xuid,
                    COUNT(*)                                      AS encounter_count,
                    MAX(r.start_time)                             AS last_seen_at
                FROM shared.match_participants p
                JOIN shared.match_participants t
                    ON t.match_id = p.match_id
                    AND t.xuid != ?
                    AND t.team_id = p.team_id
                LEFT JOIN shared.v_gamertag_lookup gl ON gl.xuid = t.xuid
                JOIN shared.match_registry r ON r.match_id = p.match_id
                WHERE p.xuid = ?
                GROUP BY gl.gamertag, t.gamertag, t.xuid
                ORDER BY encounter_count DESC
                LIMIT ?
                """,
                [xuid, xuid, _TOP_TEAMMATES_LIMIT],
            )
            rows = result.fetchall()
            columns = [d[0] for d in result.description]

        options = []
        for row in rows:
            data = dict(zip(columns, row, strict=False))
            last_seen = data.get("last_seen_at")
            if isinstance(last_seen, str):
                try:
                    last_seen = datetime.fromisoformat(last_seen)
                except Exception:
                    last_seen = None
            options.append(
                TeammateOption(
                    gamertag=str(data.get("gamertag") or data.get("xuid") or ""),
                    xuid=str(data.get("xuid") or ""),
                    encounter_count=int(data.get("encounter_count") or 0),
                    last_seen_at=last_seen,
                )
            )
        return options
    except Exception:
        logger.debug("_load_teammate_options(%s): erreur", player.player_slug, exc_info=True)
        return []


def _build_teammate_row(player: PlayerContext, xuid: str, opt: TeammateOption) -> TeammateRow:
    """Construit une ligne TeammateRow avec KPIs avec/sans l'équipier."""
    try:
        with_kpis = _load_kpis_with_teammate(player, xuid, opt.xuid)
        without_kpis = _load_kpis_without_teammate(player, xuid, opt.xuid)
        return TeammateRow(
            gamertag=opt.gamertag,
            xuid=opt.xuid,
            encounter_count=opt.encounter_count,
            last_seen_at=opt.last_seen_at,
            with_kpis=with_kpis,
            without_kpis=without_kpis,
        )
    except Exception:
        logger.debug("_build_teammate_row(%s, %s): erreur", opt.gamertag, exc_info=True)
        return TeammateRow(
            gamertag=opt.gamertag,
            xuid=opt.xuid,
            encounter_count=opt.encounter_count,
            last_seen_at=opt.last_seen_at,
        )


# ---------------------------------------------------------------------------
# KPIs par sous-ensemble
# ---------------------------------------------------------------------------


def _load_kpis_with_teammate(
    player: PlayerContext, player_xuid: str, teammate_xuid: str
) -> TeammateKPIs | None:
    """Calcule les KPIs du joueur sur les matchs joués avec cet équipier."""
    try:
        df = _load_matches_with_filter(player, player_xuid, teammate_xuid=teammate_xuid)
        if df is None or (hasattr(df, "is_empty") and df.is_empty()):
            return None
        return _compute_teammate_kpis(df)
    except Exception:
        logger.debug("_load_kpis_with_teammate: erreur", exc_info=True)
        return None


def _load_kpis_without_teammate(
    player: PlayerContext, player_xuid: str, teammate_xuid: str
) -> TeammateKPIs | None:
    """Calcule les KPIs du joueur sur les matchs joués sans cet équipier."""
    try:
        df = _load_matches_with_filter(player, player_xuid, exclude_teammate_xuid=teammate_xuid)
        if df is None or (hasattr(df, "is_empty") and df.is_empty()):
            return None
        return _compute_teammate_kpis(df)
    except Exception:
        logger.debug("_load_kpis_without_teammate: erreur", exc_info=True)
        return None


def _load_matches_with_filter(
    player: PlayerContext,
    player_xuid: str,
    teammate_xuid: str | None = None,
    exclude_teammate_xuid: str | None = None,
):
    """Charge les matchs du joueur, filtré par présence/absence d'un coéquipier."""
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)
        if not db_path.exists() or not shared_path.exists():
            return pl.DataFrame()

        with duckdb_read_only(str(db_path)) as conn:
            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            if teammate_xuid:
                # Matchs où player et l'équipier sont présents dans la même équipe
                sql = """
                    SELECT p.match_id, p.kills, p.deaths, p.assists, p.accuracy,
                           p.outcome
                    FROM shared.match_participants p
                    WHERE p.xuid = ?
                      AND EXISTS (
                          SELECT 1 FROM shared.match_participants t
                          WHERE t.match_id = p.match_id
                            AND t.xuid = ?
                            AND t.team_id = p.team_id
                      )
                """
                result = conn.execute(sql, [player_xuid, teammate_xuid])
            else:
                # Matchs où cet équipier N'est PAS dans l'équipe du joueur
                sql = """
                    SELECT p.match_id, p.kills, p.deaths, p.assists, p.accuracy,
                           p.outcome
                    FROM shared.match_participants p
                    WHERE p.xuid = ?
                      AND NOT EXISTS (
                          SELECT 1 FROM shared.match_participants t
                          WHERE t.match_id = p.match_id
                            AND t.xuid = ?
                            AND t.team_id = p.team_id
                      )
                """
                params = [player_xuid, exclude_teammate_xuid]
                result = conn.execute(sql, params)

            columns = [d[0] for d in result.description]
            rows = result.fetchall()
            if not rows:
                return pl.DataFrame()
            return pl.DataFrame(rows, schema=columns, orient="row")
    except Exception:
        logger.debug("_load_matches_with_filter: erreur", exc_info=True)
        try:
            import polars as pl

            return pl.DataFrame()
        except ImportError:
            return None


def _compute_teammate_kpis(df) -> TeammateKPIs:
    """Calcule les KPIs à partir d'un DataFrame de matchs filtrés."""
    try:
        import polars as pl

        total = len(df)
        wins = 0
        if "outcome" in df.columns:
            wins = int(df["outcome"].cast(pl.Int64, strict=False).eq(_OUTCOME_WIN).sum())

        kd_ratio = None
        if "kills" in df.columns and "deaths" in df.columns:
            total_k = int(df["kills"].cast(pl.Int64, strict=False).fill_null(0).sum())
            total_d = int(df["deaths"].cast(pl.Int64, strict=False).fill_null(0).sum())
            kd_ratio = round(total_k / total_d, 2) if total_d > 0 else float(total_k)

        win_rate = round(wins / total, 4) if total else 0.0

        accuracy = None
        if "accuracy" in df.columns:
            vals = df["accuracy"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                accuracy = round(float(vals.mean()), 1)

        kills_pg = None
        if "kills" in df.columns and total > 0:
            kills_pg = round(
                float(df["kills"].cast(pl.Float64, strict=False).fill_null(0).sum()) / total, 2
            )

        assists_pg = None
        if "assists" in df.columns and total > 0:
            assists_pg = round(
                float(df["assists"].cast(pl.Float64, strict=False).fill_null(0).sum()) / total, 2
            )

        return TeammateKPIs(
            match_count=total,
            wins=wins,
            kd_ratio=kd_ratio,
            win_rate=win_rate,
            accuracy=accuracy,
            kills_per_game=kills_pg,
            assists_per_game=assists_pg,
        )
    except Exception:
        return TeammateKPIs(match_count=len(df), wins=0, win_rate=0.0)


# ---------------------------------------------------------------------------
# Solo référence et totaux
# ---------------------------------------------------------------------------


def _load_solo_kpis(player: PlayerContext) -> TeammateKPIs | None:
    """Calcule les KPIs du joueur sur les matchs joués en solo."""
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)
        if not db_path.exists():
            return None

        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            result = conn.execute(
                """
                SELECT p.match_id, p.kills, p.deaths, p.assists, p.accuracy, p.outcome
                FROM player_match_enrichment pme
                JOIN shared.match_participants p ON p.match_id = pme.match_id
                WHERE pme.is_with_friends = FALSE AND p.xuid = (
                    SELECT value FROM sync_meta WHERE key = 'xuid'
                )
                """
            )
            columns = [d[0] for d in result.description]
            rows = result.fetchall()
            if not rows:
                return None
            df = pl.DataFrame(rows, schema=columns, orient="row")
        return _compute_teammate_kpis(df)
    except Exception:
        logger.debug("_load_solo_kpis(%s): erreur", player.player_slug, exc_info=True)
        return None


def _load_total_match_count(player: PlayerContext, xuid: str) -> int:
    """Retourne le nombre total de matchs indexés pour ce joueur."""
    try:
        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)
        if not shared_path.exists():
            return 0

        with duckdb_read_only(str(db_path)) as conn:
            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            row = conn.execute(
                "SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?", [xuid]
            ).fetchone()
            return int(row[0]) if row else 0
    except Exception:
        return 0


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _resolve_xuid_from_db(player: PlayerContext) -> str:
    """Récupère le xuid depuis sync_meta quand il n'est pas fourni dans le contexte."""
    try:
        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        if not db_path.exists():
            return ""
        with duckdb_read_only(str(db_path)) as conn:
            row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
            return str(row[0]).strip() if row else ""
    except Exception:
        return ""
