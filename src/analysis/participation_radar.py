"""Calcul du profil de participation radar — 6 axes.

Module de calcul pur (sans Plotly/Streamlit) pour le radar de participation.
Calcule et normalise le profil de participation à partir des PersonalScores
et des match_stats.

Axes : Objectifs, Combat, Support, Score, Impact, Survie.

Les seuils peuvent être dérivés du "meilleur match" global (toutes DBs joueurs)
pour une normalisation plus réaliste.

Le rendu Plotly est dans src/visualization/participation_radar.py.
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING

from src.analysis._threshold_queries import (
    _CollectedStats,
    build_thresholds_result,
    query_player_db_stats,
)
from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    import polars as pl

# =============================================================================
# Seuils de référence par défaut (fallback si pas de données globales)
# Multipliés par 2 pour éviter que le radar soit "plein" sur tous les axes
# =============================================================================

RADAR_THRESHOLDS: dict[str, float] = {
    "objectifs": 1600.0,
    "combat": 3000.0,
    "support": 800.0,
    "score": 4000.0,
    "impact_pts_per_min": 250.0,
    "survie_deaths_per_min_ref": 2.0,  # 2 morts/min = 0%, 1 mort/min = 50%, 0 = 100%
    "survie_avg_life_ref_seconds": 90.0,  # 90 sec durée vie moy = 100%
}

# Seuils p90 de référence par mode individuel (fallback si données insuffisantes)
RADAR_THRESHOLDS_PER_MODE: dict[str, float] = {
    "ctf": 850.0, "oddball": 700.0, "strongholds": 1050.0,
    "koth": 850.0, "stockpile": 950.0, "extraction": 800.0,
    "land_grab": 850.0, "slayer": 3500.0, "fiesta": 3200.0,
    "other": 950.0,
}


def _get_mode_family(pair_name: str | None) -> str:
    """Retourne la famille canonique du mode (clé de RADAR_THRESHOLDS_PER_MODE)."""
    if not pair_name or not isinstance(pair_name, str):
        return "other"
    s = pair_name.strip().casefold()
    if re.search(r"\bctf\b|\bcapture\s*(?:the\s*)?flag\b|\bdrapeau\b", s):
        return "ctf"
    if re.search(r"\boddball\b|\bballe\b", s):
        return "oddball"
    if re.search(r"\bstrongholds?\b|\bzone[s]?\b|\btotal\s*control\b|\bcontrôle\s*total\b", s):
        return "strongholds"
    if re.search(r"\bking\s*of\s*the\s*hill\b|\bkoth\b|\bhill\b", s):
        return "koth"
    if re.search(r"\bstockpile\b", s):
        return "stockpile"
    if re.search(r"\bextraction\b", s):
        return "extraction"
    if re.search(r"\bland\s*grab\b", s):
        return "land_grab"
    if re.search(r"\bfiesta\b", s):
        return "fiesta"
    if re.search(r"\bslayer\b", s):
        return "slayer"
    return "other"


# Cache des seuils globaux (calcul unique par processus)
_global_thresholds_cache: dict[str, float] | None = None


def compute_global_radar_thresholds(
    players_base_path: str | Path | None = None,
) -> dict[str, float]:
    """Calcule les seuils de normalisation à partir du meilleur match de toutes les DBs.

    Scanne data/players/*/stats.duckdb et récupère les max par catégorie.
    Utilise RADAR_THRESHOLDS en fallback si aucune donnée n'est disponible.

    Args:
        players_base_path: Chemin vers data/players. Si None, déduit depuis config.

    Returns:
        Dict de seuils (objectifs, combat, support, score, impact_pts_per_min,
        survie_deaths_per_min_ref, per_mode).
    """
    global _global_thresholds_cache
    if _global_thresholds_cache is not None:
        return _global_thresholds_cache.copy()

    from src.utils.db import duckdb_read_only

    if players_base_path is None:
        from src.config import get_repo_root
        players_base_path = Path(get_repo_root()) / "data" / "players"

    base = Path(players_base_path)
    if not base.is_dir():
        return RADAR_THRESHOLDS.copy()

    shared_db = get_shared_matches_path()
    if not shared_db.exists():
        return RADAR_THRESHOLDS.copy()
    shared_path_sql = str(shared_db).replace("'", "''")

    agg = _CollectedStats()
    for player_dir in base.iterdir():
        if not player_dir.is_dir():
            continue
        db_path = player_dir / "stats.duckdb"
        if not db_path.exists():
            continue
        try:
            with duckdb_read_only(str(db_path)) as conn:
                try:
                    conn.execute(f"ATTACH '{shared_path_sql}' AS shared (READ_ONLY)")
                except Exception:
                    logger.debug("radar_thresholds: ATTACH shared échoué pour %s", db_path)
                    continue
                agg.merge(query_player_db_stats(conn, shared_path_sql, _get_mode_family))
        except Exception as e:
            logger.debug("radar_thresholds: player_dir %s ignoré: %s", db_path, e)

    if not agg.seen_any:
        return RADAR_THRESHOLDS.copy()

    result = build_thresholds_result(agg, RADAR_THRESHOLDS, RADAR_THRESHOLDS_PER_MODE)
    _global_thresholds_cache = result
    return result.copy()


def get_radar_thresholds(db_path: str | Path | None = None) -> dict[str, float]:
    """Retourne les seuils à utiliser pour le radar.

    Utilise les seuils globaux (meilleur match) si disponibles,
    sinon RADAR_THRESHOLDS.

    Args:
        db_path: Chemin vers une DB joueur (ex. data/players/X/stats.duckdb).
                 Utilisé pour déduire data/players.
    """
    players_path = None
    if db_path:
        p = Path(db_path)
        if p.name == "stats.duckdb" and p.parent.name:
            players_path = p.parent.parent
    return compute_global_radar_thresholds(players_path)


# Catégories PersonalScores utilisées
_CATEGORIES = ("kill", "assist", "objective", "vehicle", "penalty")


def _is_objective_mode_from_pair_name(pair_name: str | None) -> bool:
    """Détermine si le mode est à objectifs à partir du pair_name.

    En mode objectif (CTF, Oddball, Strongholds, etc.), l'axe Objectifs
    utilise objective_score. En mode Slayer, les frags = objectifs.

    Args:
        pair_name: Ex. "Arena:Slayer on Aquarius", "BTB:CTF on Fragmentation".

    Returns:
        True si mode objectif, False si Slayer/Fiesta/etc.
    """
    if not pair_name or not isinstance(pair_name, str):
        return True  # Fallback : considérer objectif (utilise objective_score)
    s = pair_name.strip().casefold()
    # Modes à objectifs (FR + EN)
    objective_patterns = [
        r"\bctf\b",
        r"\bcapture\s*(?:the\s*)?flag\b",
        r"\bdrapeau\b",
        r"\boddball\b",
        r"\bballe\b",
        r"\bstrongholds?\b",
        r"\bzone[s]?\b",
        r"\btotal\s*control\b",
        r"\bcontrôle\s*total\b",
        r"\bking\s*of\s*the\s*hill\b",
        r"\bkoth\b",
        r"\bhill\b",
        r"\bstockpile\b",
        r"\bextraction\b",
        r"\bland\s*grab\b",
    ]
    return any(re.search(pat, s) for pat in objective_patterns)


def _extract_scores_from_awards(awards_df: pl.DataFrame) -> dict[str, int]:
    """Extrait les scores par catégorie depuis le DataFrame PersonalScores."""
    import polars as pl

    if awards_df.is_empty():
        return dict.fromkeys(_CATEGORIES, 0)

    if "award_score" not in awards_df.columns:
        return dict.fromkeys(_CATEGORIES, 0)

    agg = awards_df.group_by("award_category").agg(pl.col("award_score").sum().alias("total"))
    scores = {row["award_category"]: int(row["total"]) for row in agg.iter_rows(named=True)}
    return {c: scores.get(c, 0) for c in _CATEGORIES}


def _get_match_stats_values(match_row: dict | None) -> tuple[int, float, float]:
    """Extrait deaths, duration_min et avg_life_seconds depuis match_row.

    Returns:
        (deaths, duration_minutes, avg_life_seconds). duration_minutes = 10.0 si absent.
        avg_life_seconds = 0.0 si absent (sera ignoré dans le calcul).
    """
    if not match_row:
        return 0, 10.0, 0.0

    def _safe_int(val: object) -> int:
        if val is None:
            return 0
        try:
            return int(float(val))
        except (ValueError, TypeError):
            return 0

    def _safe_float(val: object) -> float:
        if val is None:
            return 0.0
        try:
            return float(val)
        except (ValueError, TypeError):
            return 0.0

    deaths = _safe_int(match_row.get("deaths"))
    duration_sec = _safe_float(match_row.get("time_played_seconds"))
    if duration_sec <= 0:
        duration_sec = 600.0  # 10 min par défaut
    duration_min = duration_sec / 60.0
    avg_life = _safe_float(
        match_row.get("avg_life_seconds") or match_row.get("average_life_seconds")
    )
    return deaths, duration_min, avg_life


@dataclass
class ProfileOptions:
    """Options de construction d'un profil de participation radar."""

    name: str = "Profil"
    color: str | None = None
    mode_is_objective: bool | None = None
    pair_name: str | None = None
    thresholds: dict[str, float] | None = None


def _to_row_dict(match_row: object) -> dict | None:
    """Convertit match_row en dict, quel que soit son type."""
    if match_row is None:
        return None
    if isinstance(match_row, dict):
        return match_row
    if hasattr(match_row, "to_dict"):
        return match_row.to_dict()  # type: ignore[union-attr]
    try:
        return dict(match_row)  # type: ignore[call-overload]
    except Exception:
        return None


def _normalize_axis(val: float, key: str, thresholds: dict[str, float]) -> float:
    """Normalise une valeur brute (0-1) par rapport au seuil de référence."""
    ref = thresholds.get(key, 1.0)
    if ref <= 0:
        return 0.0
    return min(1.0, max(0.0, val / ref))


def compute_participation_profile(
    awards_df: pl.DataFrame,
    match_row: dict | pl.Series | None = None,
    options: ProfileOptions | None = None,
) -> dict:
    """Calcule le profil de participation (6 axes) pour un ou plusieurs matchs.

    Agrège les PersonalScores par catégorie et applique la normalisation
    via des seuils de référence. Compatible avec create_participation_profile_radar().

    Args:
        awards_df: DataFrame Polars avec colonnes award_category, award_score.
        match_row: Ligne match_stats avec deaths, time_played_seconds (optionnel).
        options: Paramètres de présentation et de normalisation (ProfileOptions).

    Returns:
        Dict avec clés : name, color, *_raw, *_norm.
    """
    opts = options or ProfileOptions()
    th = opts.thresholds or RADAR_THRESHOLDS
    row_dict = _to_row_dict(match_row)

    scores = _extract_scores_from_awards(awards_df)
    kill_score = scores.get("kill", 0)
    assist_score = scores.get("assist", 0)
    objective_score = scores.get("objective", 0)
    vehicle_score = scores.get("vehicle", 0)
    penalty_score = scores.get("penalty", 0)

    is_obj = opts.mode_is_objective
    if is_obj is None:
        is_obj = _is_objective_mode_from_pair_name(
            opts.pair_name or (row_dict.get("pair_name") if row_dict else None)
        )

    objectifs_raw = int(objective_score) if is_obj else int(kill_score)
    score_raw = int(
        kill_score + assist_score + objective_score + vehicle_score + penalty_score
    )

    deaths, duration_min, avg_life_sec = _get_match_stats_values(row_dict)
    impact_raw = max(0.0, float(score_raw)) / duration_min if duration_min > 0 else 0.0
    deaths_per_min = deaths / duration_min if duration_min > 0 else 0.0
    deaths_component = max(0.0, 1.0 - deaths_per_min / th.get("survie_deaths_per_min_ref", 2.0))
    avg_life_ref = th.get("survie_avg_life_ref_seconds", 90.0)
    avg_life_component = (
        min(1.0, avg_life_sec / avg_life_ref) if avg_life_ref > 0 and avg_life_sec > 0 else 0.0
    )
    survie_raw = (
        0.5 * deaths_component + 0.5 * avg_life_component
        if avg_life_component > 0
        else deaths_component
    )

    return {
        "name": opts.name,
        "color": opts.color,
        "objectifs_raw": objectifs_raw,
        "combat_raw": kill_score,
        "support_raw": assist_score,
        "score_raw": score_raw,
        "impact_raw": impact_raw,
        "survie_raw": survie_raw,
        "objectifs_norm": _normalize_axis(float(objectifs_raw), "objectifs", th),
        "combat_norm": _normalize_axis(float(kill_score), "combat", th),
        "support_norm": _normalize_axis(float(assist_score), "support", th),
        "score_norm": _normalize_axis(float(max(0, score_raw)), "score", th),
        "impact_norm": _normalize_axis(impact_raw, "impact_pts_per_min", th),
        "survie_norm": survie_raw,
    }


__all__ = [
    "RADAR_THRESHOLDS",
    "RADAR_THRESHOLDS_PER_MODE",
    "ProfileOptions",
    "compute_global_radar_thresholds",
    "compute_participation_profile",
    "get_mode_family",
    "get_radar_thresholds",
    "is_objective_mode_from_pair_name",
]


def get_mode_family(pair_name: str | None) -> str:
    """Alias public de _get_mode_family."""
    return _get_mode_family(pair_name)


def is_objective_mode_from_pair_name(pair_name: str | None) -> bool:
    """Alias public de _is_objective_mode_from_pair_name."""
    return _is_objective_mode_from_pair_name(pair_name)
