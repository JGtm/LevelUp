"""Score de forme — historique rolling_mean(14) vs rolling_mean(90).

Calcul pur sans accès DB : entrée DataFrame, sortie DataFrame enrichi.

Fonctions disponibles :
- compute_form_score_history : rolling form_score par match (baseline)
- compute_bucket_form_score  : score par bucket intra-match (vue fine, ≤ DETAIL_THRESHOLD matchs)
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

logger = logging.getLogger(__name__)

_WINDOW_SHORT = 14
_WINDOW_LONG = 90

# Seuil de sélection en nombre de matchs pour activer le mode détail bucket
DETAIL_THRESHOLD = 30

# Durée d'un bucket en millisecondes (2 minutes)
BUCKET_MS = 2 * 60 * 1000


def compute_form_score_history(df: pl.DataFrame) -> pl.DataFrame:
    """Calcule le score de forme match par match sur l'historique complet.

    form_score = avg_perf(14 derniers matchs) - avg_perf(90 derniers matchs)

    Positif → en forme (récents au-dessus de l'habituel).
    Négatif → creux de forme.

    Args:
        df: DataFrame avec colonnes start_time et performance_score.

    Returns:
        DataFrame trié par start_time avec avg_14, avg_90, form_score ajoutés.
        Retourné tel quel si colonnes manquantes ou vide.
    """
    if df.is_empty():
        logger.debug("compute_form_score_history: DataFrame vide, retour immédiat.")
        return df
    missing = {"performance_score", "start_time"} - set(df.columns)
    if missing:
        logger.debug(
            "compute_form_score_history: colonnes manquantes %s, retour immédiat.", missing
        )
        return df

    logger.debug("compute_form_score_history: calcul sur %d matchs.", len(df))
    result = (
        df.sort("start_time")
        .with_columns(
            pl.col("performance_score")
            .rolling_mean(window_size=_WINDOW_SHORT, min_samples=1)
            .alias("avg_14"),
            pl.col("performance_score")
            .rolling_mean(window_size=_WINDOW_LONG, min_samples=1)
            .alias("avg_90"),
        )
        .with_columns((pl.col("avg_14") - pl.col("avg_90")).alias("form_score"))
    )
    logger.debug(
        "compute_form_score_history: form_score range [%.2f, %.2f].",
        result["form_score"].min() or 0.0,
        result["form_score"].max() or 0.0,
    )
    return result


_EMPTY_BUCKET_DF = pl.DataFrame(
    {
        "match_id": pl.Series([], dtype=pl.Utf8),
        "bucket_start": pl.Series([], dtype=pl.Datetime),
        "bucket_value": pl.Series([], dtype=pl.Float64),
        "bucket_label": pl.Series([], dtype=pl.Utf8),
    }
)


def _count_kd_per_bucket(
    events: list[dict[str, Any]], nb_buckets: int
) -> tuple[list[int], list[int]]:
    """Distribue kills et deaths des events dans les buckets de BUCKET_MS."""
    kills: list[int] = [0] * nb_buckets
    deaths: list[int] = [0] * nb_buckets
    for ev in events:
        t_ms = ev.get("time_ms") or 0
        idx = min(int(t_ms / BUCKET_MS), nb_buckets - 1)
        if ev.get("event_type") in ("kill", "Kill"):
            kills[idx] += 1
        elif ev.get("event_type") in ("death", "Death"):
            deaths[idx] += 1
    return kills, deaths


def _rows_for_match(
    match_id: str,
    anchor: dict[str, Any],
    meta: dict[str, Any],
    events: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    """Construit les lignes bucket pour un seul match."""
    form_score_match = anchor["form_score"]
    avg_14_match = anchor["avg_14"] or anchor["performance_score"] or 0.5
    start_time = anchor["start_time"]

    duration_ms = (meta.get("duration_seconds") or 0) * 1000
    if duration_ms <= 0:
        return []

    nb_buckets = max(1, int(duration_ms / BUCKET_MS))
    accuracy = meta.get("accuracy") or 0.0
    dmg_dealt = meta.get("damage_dealt") or 0.0
    dmg_taken = meta.get("damage_taken") or 0.0
    total_dmg = dmg_dealt + dmg_taken
    dmg_eff = (dmg_dealt / total_dmg) if total_dmg > 0 else 0.5

    kills_per_bucket, deaths_per_bucket = _count_kd_per_bucket(events, nb_buckets)
    total_kills = sum(kills_per_bucket)
    total_deaths = sum(deaths_per_bucket)

    rows = []
    for b_idx in range(nb_buckets):
        bk = kills_per_bucket[b_idx]
        bd = deaths_per_bucket[b_idx]
        total_kd = bk + bd
        kd_score = (bk / total_kd) if total_kd > 0 else 0.5
        bucket_composite = 0.6 * kd_score + 0.25 * dmg_eff + 0.15 * min(accuracy / 100, 1.0)
        bucket_display = form_score_match + (bucket_composite - avg_14_match)
        bucket_ms_offset = b_idx * BUCKET_MS
        minutes = bucket_ms_offset // 60000
        rows.append(
            {
                "match_id": str(match_id),
                "bucket_start": _offset_datetime(start_time, bucket_ms_offset),
                "bucket_value": float(bucket_display),
                "bucket_label": f"{match_id[:6]}… — {minutes}min",
                "kills": bk,
                "deaths": bd,
                "total_match_kills": total_kills,
                "total_match_deaths": total_deaths,
            }
        )
    return rows


def compute_bucket_form_score(
    history_with_form: pl.DataFrame,
    events_by_match: dict[str, list[dict[str, Any]]],
    match_meta: dict[str, dict[str, Any]],
) -> pl.DataFrame:
    """Calcule le score de forme par bucket de BUCKET_MS ms intra-match.

    Chaque bucket produit un point ancré sur le form_score du match parent :
        bucket_display = form_score_du_match + (bucket_composite - avg_14_du_match)

    Ainsi les buckets orbitent autour du point de forme, sans créer de ruptures
    entre matchs. La moyenne des buckets d'un match ≈ son form_score.

    Args:
        history_with_form: DataFrame issu de compute_form_score_history.
        events_by_match: {match_id: [{event_type, time_ms}, ...]} — kills/deaths horodatés.
        match_meta: {match_id: {accuracy, damage_dealt, damage_taken, duration_seconds}}
            — constantes par match issues de match_participants / match_registry.

    Returns:
        DataFrame colonnes : match_id, bucket_start, bucket_value, bucket_label.
        Vide si events_by_match vide ou aucune donnée.
    """
    if not events_by_match or history_with_form.is_empty():
        return _EMPTY_BUCKET_DF
    if "match_id" not in history_with_form.columns:
        return _EMPTY_BUCKET_DF

    form_idx: dict[str, dict[str, Any]] = {
        str(row["match_id"]): row
        for row in history_with_form.select(
            ["match_id", "start_time", "form_score", "avg_14", "performance_score"]
        ).to_dicts()
        if row.get("form_score") is not None
    }

    all_rows: list[dict[str, Any]] = []
    for match_id, events in events_by_match.items():
        anchor = form_idx.get(str(match_id))
        if anchor is None:
            continue
        meta = match_meta.get(str(match_id), {})
        all_rows.extend(_rows_for_match(str(match_id), anchor, meta, events))

    logger.debug("compute_bucket_form_score: %d buckets générés.", len(all_rows))
    return pl.DataFrame(all_rows) if all_rows else _EMPTY_BUCKET_DF


def _offset_datetime(base: Any, offset_ms: int) -> Any:
    """Ajoute offset_ms millisecondes à un datetime (Python datetime ou string)."""
    from datetime import datetime, timedelta

    if isinstance(base, datetime):
        return base + timedelta(milliseconds=offset_ms)
    try:
        dt = datetime.fromisoformat(str(base))
        return dt + timedelta(milliseconds=offset_ms)
    except (ValueError, TypeError):
        return base
