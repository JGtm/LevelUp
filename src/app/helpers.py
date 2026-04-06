"""Fonctions utilitaires pour l'application Streamlit.

Ce module contient des helpers génériques utilisés dans l'application principale.
"""

from __future__ import annotations

import re

import polars as pl
import streamlit as st

from src.config import OKABE_ITO_PALETTE
from src.ui.i18n import t
from src.utils.polars_compat import ensure_polars as _to_polars
from src.utils.strings import is_uuid_like

# =============================================================================
# Regex & constantes
# =============================================================================

_LABEL_SUFFIX_RE = re.compile(r"^(.*?)(?:\s*[\-–—]\s*[0-9A-Za-z]{8,})$", re.IGNORECASE)


# =============================================================================
# Nettoyage de labels
# =============================================================================


def clean_asset_label(s: str | None) -> str | None:
    """Nettoie un label d'asset en retirant les suffixes techniques (IDs).

    Args:
        s: Label brut.

    Returns:
        Label nettoyé ou None.
    """
    if s is None:
        return None
    v = str(s).strip()
    if not v:
        return None
    m = _LABEL_SUFFIX_RE.match(v)
    if m:
        v = (m.group(1) or "").strip()
    return v or None


def normalize_mode_label(
    pair_name: str | None,
    *,
    lang: str = "fr",
    normalize: bool = True,
) -> str | None:
    """Normalise le label d'un mode de jeu (délègue à mode_categories).

    - Traduit le mode
    - Retire le nom de carte si présent ("X on MapName")
    - Retire les suffixes Forge/Ranked

    Args:
        pair_name: Nom du pair (mode + carte).
        lang: Code de langue ("fr" ou "en").
        normalize: Si True, applique la normalisation des préfixes de modes.

    Returns:
        Label normalisé ou None.
    """
    from src.analysis.mode_categories import normalize_pair_name_to_mode_ui

    return normalize_pair_name_to_mode_ui(pair_name, lang=lang, normalize=normalize)


def normalize_map_label(map_name: str | None) -> str | None:
    """Normalise le nom d'une carte pour le filtre.

    - Supprime les suffixes après '-' (ex: 'Aquarius - Live Fire' → 'Aquarius')
    - Masque les UUIDs non résolus en 'Carte inconnue'

    Args:
        map_name: Nom brut de la carte.

    Returns:
        Nom normalisé ou None.
    """
    base = clean_asset_label(map_name)
    if base is None:
        return None
    if is_uuid_like(base):
        return t("unknown_map")
    if " - " in base:
        base = base.split(" - ")[0].strip()
    return base or None


# =============================================================================
# Helpers
# =============================================================================


# =============================================================================
# Calculs temporels
# =============================================================================


def compute_session_span_seconds(df_: pl.DataFrame) -> float | None:
    """Calcule la durée totale d'une session (premier match → fin dernier match).

    Args:
        df_: DataFrame (Pandas ou Polars) des matchs de la session.

    Returns:
        Durée en secondes ou None.
    """
    df_pl = _to_polars(df_)

    if df_pl.is_empty() or "start_time" not in df_pl.columns:
        return None

    # Cast start_time en datetime
    df_pl = df_pl.with_columns(pl.col("start_time").cast(pl.Datetime, strict=False))
    valid = df_pl.filter(pl.col("start_time").is_not_null())
    if valid.is_empty():
        return None

    t0 = valid.select(pl.col("start_time").min()).item()
    if "time_played_seconds" in valid.columns:
        # Calculer la fin de chaque match
        with_end = valid.with_columns(
            (
                pl.col("start_time")
                + pl.duration(
                    seconds=pl.col("time_played_seconds")
                    .cast(pl.Float64, strict=False)
                    .fill_null(0)
                )
            ).alias("end_time")
        )
        t1 = with_end.select(pl.col("end_time").max()).item()
    else:
        t1 = valid.select(pl.col("start_time").max()).item()

    if t0 is None or t1 is None:
        return None
    return float((t1 - t0).total_seconds())


def compute_total_play_seconds(df_: pl.DataFrame) -> float | None:
    """Calcule le temps de jeu total (somme des durées de matchs).

    Args:
        df_: DataFrame (Pandas ou Polars) des matchs.

    Returns:
        Durée totale en secondes ou None.
    """
    df_pl = _to_polars(df_)

    if df_pl.is_empty() or "time_played_seconds" not in df_pl.columns:
        return None
    val = df_pl.select(pl.col("time_played_seconds").cast(pl.Float64, strict=False).sum()).item()
    return float(val) if val is not None else None


def avg_match_duration_seconds(df_: pl.DataFrame) -> float | None:
    """Calcule la durée moyenne d'un match.

    Args:
        df_: DataFrame (Pandas ou Polars) des matchs.

    Returns:
        Durée moyenne en secondes ou None.
    """
    df_pl = _to_polars(df_)

    if df_pl.is_empty() or "time_played_seconds" not in df_pl.columns:
        return None
    val = df_pl.select(pl.col("time_played_seconds").cast(pl.Float64, strict=False).mean()).item()
    return float(val) if val is not None else None


# =============================================================================
# Plage de dates
# =============================================================================


def date_range(df: pl.DataFrame) -> tuple:
    """Retourne la plage de dates du DataFrame.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonne 'date'.

    Returns:
        Tuple (date_min, date_max).
    """
    df_pl = _to_polars(df)
    dmin = df_pl.select(pl.col("date").min()).item()
    dmax = df_pl.select(pl.col("date").max()).item()
    return dmin, dmax


# =============================================================================
# Couleurs joueurs
# =============================================================================


def assign_player_colors(names: list[str]) -> dict[str, str]:
    """Assigne des couleurs persistantes aux joueurs.

    Les couleurs sont stockées en session_state pour rester cohérentes
    tout au long de la session.

    Args:
        names: Liste des noms de joueurs.

    Returns:
        Mapping {nom: couleur_hex}.
    """
    cycle = OKABE_ITO_PALETTE
    state_key = "_os_player_colors"
    persisted = st.session_state.get(state_key)
    if not isinstance(persisted, dict):
        persisted = {}

    used = {str(v) for v in persisted.values() if v is not None}

    for n in names:
        key = str(n)
        if not key or key in persisted:
            continue

        chosen = None
        for c in cycle:
            if c not in used:
                chosen = c
                break
        if chosen is None:
            chosen = cycle[len(persisted) % len(cycle)]

        persisted[key] = chosen
        used.add(chosen)

    st.session_state[state_key] = persisted
    return {str(n): persisted[str(n)] for n in names if str(n) in persisted}


# =============================================================================
# Pandas Styler compat
# =============================================================================


def styler_map(styler, func, subset):
    """Applique une fonction de style (compat pandas anciennes versions).

    - pandas récents: .map(func, subset=...)
    - pandas anciens: .applymap(func, subset=...)

    Args:
        styler: Pandas Styler.
        func: Fonction de style.
        subset: Colonnes à cibler.

    Returns:
        Styler modifié.
    """
    try:
        if hasattr(styler, "map"):
            return styler.map(func, subset=subset)
    except Exception:
        pass
    # Fallback
    try:
        return styler.applymap(func, subset=subset)
    except Exception:
        return styler


# =============================================================================
# Session state helpers
# =============================================================================


def clear_min_matches_maps_auto() -> None:
    """Réinitialise le flag auto pour min_matches_maps."""
    st.session_state["_min_matches_maps_auto"] = False


def clear_min_matches_maps_friends_auto() -> None:
    """Réinitialise le flag auto pour min_matches_maps_friends."""
    st.session_state["_min_matches_maps_friends_auto"] = False
