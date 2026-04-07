"""Chargement et initialisation des données pour l'application Streamlit.

Ce module gère :
- L'initialisation de l'état source (DB, XUID, joueur Waypoint)
- La résolution des identités joueur
- Le chargement des données avec cache
- La génération automatique des références (citations H5G)
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING

import polars as pl
import streamlit as st

from src.ui import AppSettings
from src.ui.cache import db_cache_key, load_df_optimized
from src.ui.sync import is_spnkr_db_path, pick_latest_spnkr_db_if_any
from src.utils import (
    guess_xuid_from_db_path,
    infer_spnkr_player_from_db_path,
)

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository

logger = logging.getLogger(__name__)

# =============================================================================
# Identité joueur depuis secrets/env
# =============================================================================


def default_identity_from_secrets() -> tuple[str, str, str]:
    """Retourne (xuid_or_gamertag, xuid_fallback, waypoint_player) depuis secrets/env/constants.

    Ordre de priorité :
    1. Secrets Streamlit (.streamlit/secrets.toml)
    2. Variables d'environnement
    3. Constantes du projet

    Returns:
        Tuple (xuid_or_gamertag, xuid_fallback, waypoint_player).
    """
    from src.app.profile import get_identity_from_secrets

    identity = get_identity_from_secrets()
    return identity.gamertag, identity.xuid, identity.waypoint_player


def propagate_identity_env(xuid_or_gt: str, xuid_fallback: str, wp: str) -> None:
    """Propage les defaults depuis secrets vers l'env.

    Utile notamment pour résoudre un XUID quand la DB SPNKr ne contient pas les gamertags.
    """
    try:
        if xuid_or_gt and not str(xuid_or_gt).strip().isdigit() and xuid_fallback:
            if not str(os.environ.get("LEVELUP_DEFAULT_GAMERTAG") or "").strip():
                os.environ["LEVELUP_DEFAULT_GAMERTAG"] = str(xuid_or_gt).strip()
            if not str(os.environ.get("LEVELUP_DEFAULT_XUID") or "").strip():
                os.environ["LEVELUP_DEFAULT_XUID"] = str(xuid_fallback).strip()
        if wp and not str(os.environ.get("LEVELUP_DEFAULT_WAYPOINT_PLAYER") or "").strip():
            os.environ["LEVELUP_DEFAULT_WAYPOINT_PLAYER"] = str(wp).strip()
    except Exception:
        pass


# =============================================================================
# Initialisation source state
# =============================================================================


def _resolve_db_path(default_db: str, settings: AppSettings) -> str:
    """Retourne le db_path à utiliser pour une nouvelle session.

    Priorité :
    1. ``LEVELUP_DB`` / ``LEVELUP_DB_PATH`` (env) — forcé, immuable
    2. Deep link match direct (``?gamertag=X&match_id=Y``) — restaure le bon joueur
       quand on navigue depuis l'historique ou la carrière vers un match spécifique.
       La présence conjointe de ``match_id`` distingue ce cas des liens d'encounter
       Explorer (``?gamertag=`` seul) qui ne doivent **pas** switcher de joueur.
    3. localStorage navigateur (``last_db_path``) — restaure le joueur du dernier run.
    4. SPNKr DB si ``prefer_spnkr_db_if_available`` est activé dans les settings
    5. ``default_db`` (premier joueur alphabétique fourni par ``get_default_db_path()``)
    """
    forced_env_db = str(
        os.environ.get("LEVELUP_DB") or os.environ.get("LEVELUP_DB_PATH") or ""
    ).strip()
    if forced_env_db:
        logger.debug("_resolve_db_path: DB forcée via env → %s", forced_env_db)
        return forced_env_db

    chosen = str(default_db or "")

    # Deep link "match direct" depuis historique / carrière.
    # ?gamertag=X&match_id=Y → restaurer la DB du joueur X si elle existe.
    # IMPORTANT : match_id obligatoire pour ne pas switcher sur les liens encounter.
    try:
        _url_gt = str((st.query_params or {}).get("gamertag") or "").strip()
        _url_mid = str((st.query_params or {}).get("match_id") or "").strip()
        if _url_gt and _url_mid and default_db:
            from pathlib import Path as _Path

            _candidate = _Path(default_db).parent.parent / _url_gt / "stats.duckdb"
            if _candidate.exists() and _candidate.stat().st_size > 0:
                logger.debug(
                    "_resolve_db_path: DB restaurée via deep link match → %s (gamertag=%s)",
                    _candidate,
                    _url_gt,
                )
                return str(_candidate)
    except Exception:
        pass

    # localStorage navigateur (v6.4) : restaure le dernier joueur du navigateur.
    try:
        ls_prefs = st.session_state.get("_browser_prefs_restored") or {}
        ls_slug = str(ls_prefs.get("last_db_path") or "").strip()
        if ls_slug and default_db:
            from pathlib import Path as _Path

            _candidate = _Path(default_db).parent.parent / ls_slug / "stats.duckdb"
            if _candidate.exists() and _candidate.stat().st_size > 0:
                logger.debug(
                    "_resolve_db_path: DB restaurée depuis localStorage → %s",
                    _candidate,
                )
                return str(_candidate)
    except Exception:
        pass

    if settings.prefer_spnkr_db_if_available:
        spnkr = pick_latest_spnkr_db_if_any()
        if spnkr and os.path.exists(spnkr) and os.path.getsize(spnkr) > 0:
            logger.debug("_resolve_db_path: SPNKr DB sélectionnée → %s", spnkr)
            return spnkr

    return chosen


def init_source_state(default_db: str, settings: AppSettings) -> None:
    """Initialise l'état source (DB path, xuid, waypoint) en session_state.

    Délègue la sélection du db_path à ``_resolve_db_path()``.

    Args:
        default_db: Chemin par défaut de la DB (résultat de ``get_default_db_path()``).
        settings: Paramètres de l'application.
    """
    if "db_path" not in st.session_state:
        chosen = _resolve_db_path(default_db, settings)
        logger.debug("init_source_state: db_path=%s", chosen or "(vide)")
        st.session_state["db_path"] = chosen

    if "xuid_input" not in st.session_state:
        legacy = str(st.session_state.get("xuid", "") or "").strip()
        guessed = guess_xuid_from_db_path(st.session_state.get("db_path", "") or "") or ""
        xuid_or_gt, _xuid_fallback, _wp = default_identity_from_secrets()

        # Pour les DB SPNKr, pré-remplir avec le joueur déduit du nom de DB.
        inferred = (
            infer_spnkr_player_from_db_path(str(st.session_state.get("db_path", "") or "")) or ""
        )
        xuid_input = legacy or inferred or guessed or xuid_or_gt

        # Si pas encore un XUID numérique valide, tenter sync_meta
        # (source canonique après le premier sync réussi).
        _current_db = str(st.session_state.get("db_path", "") or "").strip()
        if (
            (not xuid_input or not xuid_input.isdigit())
            and _current_db
            and os.path.exists(_current_db)
        ):
            try:
                from src.ui._cache_core import _resolve_player_xuid

                resolved = _resolve_player_xuid(_current_db)
                if resolved:
                    xuid_input = resolved
                    logger.debug("init_source_state: XUID résolu depuis sync_meta → %s", resolved)
            except Exception:
                pass

        logger.debug(
            "init_source_state: xuid_input=%s (legacy=%r inferred=%r guessed=%r)",
            xuid_input or "(vide)",
            bool(legacy),
            bool(inferred),
            bool(guessed),
        )
        st.session_state["xuid_input"] = xuid_input

    if "waypoint_player" not in st.session_state:
        _xuid_or_gt, _xuid_fallback, wp = default_identity_from_secrets()
        logger.debug("init_source_state: waypoint_player=%s", wp or "(vide)")
        st.session_state["waypoint_player"] = wp


def resolve_xuid_input(xuid_input: str, db_path: str) -> str:
    """Résout un XUID à partir de l'entrée utilisateur.

    Args:
        xuid_input: Entrée utilisateur (XUID ou gamertag).
        db_path: Chemin vers la base de données.

    Returns:
        XUID résolu ou chaîne vide.
    """
    from src.app.profile import get_identity_from_secrets, resolve_xuid

    return resolve_xuid(xuid_input, db_path, identity=get_identity_from_secrets())


def validate_db_path(db_path: str, default_db: str) -> str:
    """Valide et corrige le chemin de la DB si nécessaire.

    - Si le fichier n'existe pas, retourne chaîne vide
    - Si le fichier est vide (0 octet), tente un fallback

    Args:
        db_path: Chemin actuel de la DB.
        default_db: Chemin par défaut en fallback.

    Returns:
        Chemin validé ou chaîne vide.
    """
    if db_path and not os.path.exists(db_path):
        return ""

    # Si la DB existe mais est vide (0 octet), tenter un fallback automatique.
    if db_path and os.path.exists(db_path):
        try:
            if os.path.getsize(db_path) <= 0:
                fallback = ""
                if is_spnkr_db_path(db_path):
                    fallback = pick_latest_spnkr_db_if_any()
                    if fallback and os.path.exists(fallback) and os.path.getsize(fallback) <= 0:
                        fallback = ""
                if not fallback:
                    fallback = str(default_db or "").strip()
                    if not (fallback and os.path.exists(fallback)):
                        fallback = ""
                if fallback and fallback != db_path:
                    return fallback
                return ""
        except Exception:
            pass

    return db_path


# =============================================================================
# Cache keys
# =============================================================================


def get_db_cache_key(db_path: str) -> tuple[int, int, int, int] | None:
    """Retourne une clé de cache basée sur player DB + shared_matches_v2.duckdb.

    Args:
        db_path: Chemin vers la base de données joueur.

    Returns:
        Tuple (mtime_ns_player, size_player, mtime_ns_shared, size_shared) ou None.
    """
    return db_cache_key(db_path)


def get_aliases_cache_key() -> int | None:
    """Retourne toujours None depuis v5.2 (plus de fichier xuid_aliases.json).

    Les aliases sont désormais exclusivement dans shared_matches_v2.duckdb.
    L'invalidation de cache se fait via db_cache_key() sur la DB.
    """
    return None


# =============================================================================
# Chargement des données
# =============================================================================


def get_cached_repository(
    db_path: str,
    xuid: str,
    *,
    read_only: bool = True,
) -> DuckDBRepository:
    """Retourne un DuckDBRepository avec connexion pré-initialisée.

    Centralise la création de repository pour éviter les instanciations
    directes éparpillées dans les pages UI. La connexion est initialisée
    immédiatement (warm-up) pour éviter le coût du premier ATTACH.

    Note : Le cache Streamlit (@st.cache_resource) est géré dans
    ``src/ui/cache_loaders.py::get_cached_repository_st`` pour les pages UI.
    Cette fonction est le point d'entrée non-Streamlit.

    Args:
        db_path: Chemin vers stats.duckdb.
        xuid: XUID du joueur.
        read_only: Connexion en lecture seule.

    Returns:
        Instance DuckDBRepository avec connexion active.
    """
    from src.data.repositories.duckdb_repo import DuckDBRepository

    repo = DuckDBRepository(db_path, xuid, read_only=read_only)
    # Warm-up : forcer la connexion + ATTACH immédiatement
    repo._get_connection()
    return repo


def load_match_data(db_path: str, xuid: str, db_key: tuple[int, int] | None) -> pl.DataFrame:
    """Charge les données de matchs depuis la DB.

    Args:
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        db_key: Clé de cache.

    Returns:
        DataFrame Polars des matchs ou DataFrame vide.
    """
    if not db_path or not os.path.exists(db_path) or not str(xuid or "").strip():
        return pl.DataFrame()

    return load_df_optimized(db_path, xuid.strip(), db_key=db_key)


# =============================================================================

# =============================================================================
# Génération automatique des références (LEGACY — désormais via metadata.duckdb)
# =============================================================================


def ensure_h5g_commendations_repo() -> None:
    """No-op : les citations sont désormais stockées dans ``citation_mappings``
    de ``metadata.duckdb``, peuplé par ``scripts/populate_citation_mappings.py``.

    Conservé pour compatibilité d'appel (ne fait plus rien).
    """
