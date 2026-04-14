"""Pont pur vers les fonctions ``src.ui.*`` utilisées par l'API.

Chaque fonction ici est **sans dépendance Streamlit** et sert uniquement
à découpler la couche API de la couche UI.  L'import path devient :
``from apps.api.app._pure_bridge import ...`` au lieu de ``from src.ui.xxx import ...``

Quand une fonction sera migrée dans ``src/data/`` ou ``src/analysis/``,
il suffira de changer l'import *ici* — pas dans chaque service.
"""

from __future__ import annotations

import logging
import re
from datetime import datetime
from typing import Any

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Médailles — délègue directement à src.data.medal_definitions (source)
# ---------------------------------------------------------------------------


def load_medal_name_maps() -> tuple[dict[str, str], dict[str, str]]:
    """Charge les labels FR/EN des médailles depuis metadata.duckdb."""
    from src.data.medal_definitions import load_medal_name_maps as _load

    return _load()


def load_medal_description_map(lang: str = "fr") -> dict[str, str]:
    """Charge la map {medal_name_id: description} depuis metadata.duckdb."""
    from src.data.medal_definitions import load_medal_description_map as _load

    return _load(lang)


# ---------------------------------------------------------------------------
# Settings — fonctions pures (pas de st.session_state)
# ---------------------------------------------------------------------------


def load_settings() -> Any:
    """Charge AppSettings depuis app_settings.json (pur Python)."""
    from src.ui.settings import load_settings as _load

    return _load()


def save_settings(settings: Any) -> tuple[bool, str]:
    """Persiste AppSettings sur disque (pur Python, thread-safe)."""
    from src.ui.settings import save_settings as _save

    return _save(settings)


def get_app_settings_class() -> type:
    """Retourne la classe AppSettings (Pydantic v2, sans Streamlit)."""
    from src.ui.settings import AppSettings

    return AppSettings


# ---------------------------------------------------------------------------
# Career Ranks — accès metadata.duckdb (pur Python + DuckDB)
# ---------------------------------------------------------------------------


def get_rank_info(rank_number: int) -> Any:
    """Retourne CareerRankInfo pour un rang donné (1-272)."""
    from src.ui.career_ranks import get_rank_info as _get

    return _get(rank_number)


# ---------------------------------------------------------------------------
# Career Data — remplace get_cached_repository_st par DuckDBRepository
# ---------------------------------------------------------------------------


def load_career_data(db_path: str, xuid: str) -> dict | None:
    """Charge les données de rang depuis DuckDBRepository."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(db_path, xuid=xuid, read_only=True) as repo:
            return repo.load_latest_career_rank()
    except Exception:
        logger.debug("load_career_data: erreur", exc_info=True)
        return None


def load_career_history(db_path: str, xuid: str) -> list[dict]:
    """Charge l'historique de progression de rang."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(db_path, xuid=xuid, read_only=True) as repo:
            return repo.load_career_history()
    except Exception:
        logger.debug("load_career_history: erreur", exc_info=True)
        return []


def load_lusr_snapshot(db_path: str, xuid: str) -> list[dict]:
    """Charge le snapshot LUSR/CSR courant."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(db_path, xuid=xuid, read_only=True) as repo:
            return repo.load_lusr_snapshot()
    except Exception:
        logger.debug("load_lusr_snapshot: erreur", exc_info=True)
        return []


def load_lusr_history(db_path: str, xuid: str) -> list[dict]:
    """Charge l'historique LUSR/CSR."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(db_path, xuid=xuid, read_only=True) as repo:
            return repo.load_lusr_history()
    except Exception:
        logger.debug("load_lusr_history: erreur", exc_info=True)
        return []


def load_top_matches(
    db_path: str,
    xuid: str,
    *,
    best: bool = True,
    exclude_btb: bool = False,
    shared_db_path: str | None = None,
) -> list[dict]:
    """Charge les top meilleurs/pires matchs via DuckDBRepository."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(
            db_path,
            xuid=xuid,
            shared_db_path=shared_db_path,
            read_only=True,
        ) as repo:
            return repo.load_top_match_list(best=best, exclude_btb=exclude_btb)
    except Exception:
        logger.debug("load_top_matches(best=%s): erreur", best, exc_info=True)
        return []


def load_top_encountered(
    xuid: str,
    db_path: str,
    limit: int = 10,
    *,
    exclude_xuids: set[str] | None = None,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les joueurs les plus croisés via DuckDBRepository."""
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        with DuckDBRepository(db_path, xuid=xuid, read_only=True) as repo:
            return repo.load_top_encountered(limit=limit, exclude_xuids=exclude_xuids, since=since)
    except Exception:
        logger.debug("load_top_encountered: erreur", exc_info=True)
        return []


# ---------------------------------------------------------------------------
# Career Logic — fonctions de calcul pures (math, 0 DB, 0 Streamlit)
# ---------------------------------------------------------------------------


def compute_career_projections(
    history: list[dict], xp_total: int, last_date: datetime | None
) -> dict:
    """Calcule les projections XP vers le rang Héros.

    Returns:
        dict avec xp_per_day_active, xp_per_day_fallback, hero_date.
    """
    try:
        from src.ui.pages.career_logic import (
            CAREER_XP_LAUNCH_DATE,
            _compute_active_xp_per_day,
            _compute_fallback_xp_per_day,
            _compute_hero_projections,
        )

        xp_hero_total = 9_319_350
        xp_per_active = _compute_active_xp_per_day(history)
        ref_date = history[0]["recorded_at"] if history else CAREER_XP_LAUNCH_DATE
        xp_per_fallback = _compute_fallback_xp_per_day(xp_total, ref_date)

        hero_date = None
        if xp_total < xp_hero_total and last_date and xp_per_active > 0:
            normal_proj, _ = _compute_hero_projections(xp_total, last_date, xp_per_active)
            if normal_proj:
                hero_date = normal_proj[-1][0].date()

        return {
            "xp_per_day_active": round(xp_per_active, 2),
            "xp_per_day_fallback": round(xp_per_fallback, 2),
            "hero_date": hero_date,
        }
    except Exception:
        logger.warning("compute_career_projections: erreur", exc_info=True)
        return {"xp_per_day_active": 0.0, "xp_per_day_fallback": 0.0, "hero_date": None}


# ---------------------------------------------------------------------------
# Commendations — parse cibles de tiers (pur string)
# ---------------------------------------------------------------------------


def parse_tier_targets(csv_targets: str | None) -> list[dict[str, Any]]:
    """Convertit une chaîne CSV de cibles en liste de tiers dict.

    Input : ``"10,20,30,50,100"``
    Output : ``[{"tier": 1, "target_count": 10}, ...]``
    """
    if not csv_targets:
        return []
    result: list[dict[str, Any]] = []
    for i, part in enumerate(str(csv_targets).split(","), 1):
        part = part.strip()
        if not part:
            continue
        try:
            result.append({"tier": i, "target_count": int(part)})
        except ValueError:
            continue
    return result


# ---------------------------------------------------------------------------
# Session Compare — catégorie dominante (pur Polars)
# ---------------------------------------------------------------------------


def infer_session_dominant_category(df_session: Any) -> str:
    """Infère la catégorie dominante d'une session via pair_name."""
    try:
        from src.ui.pages.session_compare_logic import (
            infer_session_dominant_category as _infer,
        )

        return _infer(df_session)
    except Exception:
        return "Other"


# ---------------------------------------------------------------------------
# Setup Wizard — création profil joueur (pur Python, pas de Streamlit)
# ---------------------------------------------------------------------------


def validate_gamertag(gamertag: str) -> list[str]:
    """Valide le format d'un gamertag Xbox."""
    _PATTERN = re.compile(r"^[\w\s\-]{1,50}$")
    errors = []
    gamertag = gamertag.strip()
    if not gamertag:
        errors.append("Le gamertag est requis.")
    elif not _PATTERN.match(gamertag):
        errors.append(
            "Le gamertag contient des caractères invalides. "
            "Seuls les lettres, chiffres, espaces et tirets sont autorisés."
        )
    return errors


def create_player_profile(gamertag: str, xuid: str = "") -> str:
    """Crée un profil joueur dans db_profiles.json."""
    from src.ui.pages.setup_wizard_logic import create_player_profile as _create

    return _create(gamertag, xuid)


# ---------------------------------------------------------------------------
# Battlepass / Challenges — async Halo API calls (aucune dép. Streamlit)
# ---------------------------------------------------------------------------


async def fetch_battlepass_info(session: Any, *args: Any, **kwargs: Any) -> Any:
    """Proxy vers ``home_mission_control_battlepass.fetch_battlepass_info``."""
    from src.ui.pages.home_mission_control_battlepass import fetch_battlepass_info as _fetch

    return await _fetch(session, *args, **kwargs)


async def fetch_home_progressions(*args: Any, **kwargs: Any) -> Any:
    """Proxy vers ``home_mission_control_api.fetch_home_progressions``."""
    from src.ui.pages.home_mission_control_api import fetch_home_progressions as _fetch

    return _fetch(*args, **kwargs)
