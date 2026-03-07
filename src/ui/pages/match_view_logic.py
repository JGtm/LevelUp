"""Logique métier pure pour match_view (sans Streamlit)."""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.analysis.performance_score import compute_relative_performance_score
from src.config import HALO_COLORS, OUTCOME_CODES
from src.ui.i18n import get_outcome_map
from src.utils.db import duckdb_read_only
from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)


def resolve_outcome(
    row: dict[str, Any],
) -> tuple[int | None, str, str]:
    """Résout le code outcome → (code, label, couleur)."""
    outcome_map = get_outcome_map()
    try:
        outcome_code = int(row.get("outcome")) if row.get("outcome") is not None else None
    except (TypeError, ValueError):
        outcome_code = None
    outcome_label = outcome_map.get(outcome_code, "?") if outcome_code is not None else "-"

    colors = HALO_COLORS.as_dict()
    if outcome_code == OUTCOME_CODES.WIN:
        outcome_color = colors["green"]
    elif outcome_code == OUTCOME_CODES.LOSS:
        outcome_color = colors["red"]
    elif outcome_code in (OUTCOME_CODES.TIE, OUTCOME_CODES.NO_FINISH):
        outcome_color = colors["violet"]
    else:
        outcome_color = colors["slate"]
    return outcome_code, outcome_label, outcome_color


def load_enrichment(db_path: str, match_id: str) -> tuple[bool, float | None]:
    """Charge had_bot_teammate et performance_score depuis player_match_enrichment."""
    had_bot = False
    stored_perf: float | None = None
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            pme_row = conn.execute(
                "SELECT had_bot_teammate, performance_score"
                " FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                [match_id],
            ).fetchone()
        if pme_row:
            had_bot = bool(pme_row[0])
            if pme_row[1] is not None:
                stored_perf = float(pme_row[1])
    except Exception:
        logger.debug("match_view: enrichment introuvable match=%s", match_id)
    return had_bot, stored_perf


def compute_perf_display(
    row: dict[str, Any],
    df_full: pl.DataFrame | None,
    stored_perf: float | None,
    had_bot: bool,
) -> tuple[float | None, str, str | None]:
    """Calcule le score de performance et sa représentation visuelle."""
    perf_score = stored_perf
    if perf_score is None and df_full is not None and len(df_full) >= 10:
        perf_score = compute_relative_performance_score(row, df_full, had_bot_teammate=had_bot)
    perf_display = f"{perf_score:.0f}" if perf_score is not None else "-"
    perf_color = None
    if perf_score is not None:
        colors = HALO_COLORS.as_dict()
        if perf_score >= SCORE_THRESHOLDS["excellent"]:
            perf_color = colors["green"]
        elif perf_score >= SCORE_THRESHOLDS["good"]:
            perf_color = colors["cyan"]
        elif perf_score >= SCORE_THRESHOLDS["average"]:
            perf_color = colors["amber"]
        elif perf_score >= SCORE_THRESHOLDS["below_average"]:
            perf_color = colors.get("orange", "#FF8C00")
        else:
            perf_color = colors["red"]
    return perf_score, perf_display, perf_color


def enrich_pm_from_row(pm: dict[str, Any], row: dict[str, Any]) -> None:
    """Enrichit pm avec les valeurs réelles si manquantes (fallback DuckDB v4)."""
    for stat_key in ("kills", "deaths", "assists"):
        if pm.get(stat_key, {}).get("count") is None:
            val = row.get(stat_key)
            if val is not None:
                pm.setdefault(stat_key, {})["count"] = float(val) if val == val else None


def detect_abandoned_match(match_id: str, db_path: str) -> bool:
    """Retourne True si tous les participants du match ont kills=deaths=score=0.

    Interroge shared_matches.duckdb pour vérifier si le match s'est terminé
    sans qu'aucun joueur n'ait enregistré de points (crash serveur, partie non disputée).
    """
    try:
        shared_path = get_shared_matches_path_from_player(db_path)
        if shared_path is None:
            return False
        with duckdb_read_only(str(shared_path)) as conn:
            result = conn.execute(
                "SELECT COUNT(*) AS n,"
                " COALESCE(SUM(kills), 0) AS k,"
                " COALESCE(SUM(deaths), 0) AS d,"
                " COALESCE(SUM(score), 0) AS s"
                " FROM match_participants WHERE match_id = ?",
                [match_id],
            ).fetchone()
        if result and result[0] > 0:
            return result[1] == 0 and result[2] == 0 and result[3] == 0
    except Exception:
        logger.debug("detect_abandoned_match: erreur pour match=%s", match_id)
    return False
