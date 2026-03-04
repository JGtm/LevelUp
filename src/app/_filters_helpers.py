"""Utilitaires partagés entre les modules de rendu des filtres.

Ce module expose les constantes et fonctions pures utilisées par
_filters_period.py, _filters_session.py, _filters_cascade.py et
filters_render.py sans créer de dépendances circulaires.
"""

from __future__ import annotations

from datetime import date

import polars as pl
import streamlit as st

GAP_MINUTES_FIXED = 120  # Figé (sessions stockées en base, cf. SESSIONS_STOCKAGE_PLAN.md)

# Préfixes des clés widget individuelles créées par render_checkbox_filter /
# render_hierarchical_checkbox_filter pour les filtres cascade.
# Ces clés doivent être nettoyées lors d'un cascade reset pour éviter que
# Streamlit réutilise d'anciennes valeurs (ex: checkbox décochée) et écrase
# la réinitialisation programmée de la sélection.
_CASCADE_WIDGET_KEY_PREFIXES = (
    "filter_playlists_cb_",
    "filter_playlists_all",
    "filter_playlists_none",
    "filter_playlists_confirm",
    "filter_playlists_cancel",
    "filter_modes_cb_",
    "filter_modes_cat_",
    "filter_modes_mode_",
    "filter_modes_all",
    "filter_modes_none",
    "filter_modes_confirm",
    "filter_modes_cancel",
    "filter_maps_cb_",
    "filter_maps_all",
    "filter_maps_none",
    "filter_maps_confirm",
    "filter_maps_cancel",
)


def _cascade_reset_filters() -> None:
    """Réinitialise COMPLÈTEMENT les filtres cascade (playlists/modes/cartes).

    Supprime :
    - Les clés agrégées (filter_playlists, filter_modes, filter_maps)
    - Les clés de mode/exclusions intent-based
    - Les clés widget individuelles des checkboxes (filter_playlists_cb_*, etc.)

    Sans cette dernière étape, Streamlit réutiliserait les anciennes valeurs
    des checkboxes lors du prochain render (le paramètre ``value=`` est ignoré
    quand la clé widget existe déjà dans session_state).
    """
    for _k in (
        "filter_playlists",
        "filter_modes",
        "filter_maps",
        "_playlists_exclusions",
        "_modes_exclusions",
        "_maps_exclusions",
        "_playlists_filter_mode",
        "_modes_filter_mode",
        "_maps_filter_mode",
    ):
        st.session_state.pop(_k, None)
    # Nettoyage des clés widget individuelles (checkboxes, boutons associés)
    for wk in list(st.session_state.keys()):
        if any(wk.startswith(p) for p in _CASCADE_WIDGET_KEY_PREFIXES):
            del st.session_state[wk]


def _to_polars(df: object) -> pl.DataFrame:
    """Convertit un DataFrame Pandas en Polars si nécessaire (pont de sécurité)."""
    if isinstance(df, pl.DataFrame):
        return df
    try:
        return pl.from_pandas(df)  # type: ignore[arg-type]
    except Exception:
        return pl.DataFrame()


def _safe_to_date(val: object) -> date:
    """Convertit une valeur en date Python, date.today() si invalide."""
    if isinstance(val, date):
        return val
    try:
        from dateutil.parser import parse as _parse_dt

        return _parse_dt(str(val)).date()
    except (ValueError, TypeError, ImportError):
        return date.today()


def _session_labels_ordered_by_last_match(base_s: pl.DataFrame) -> list[str]:
    """Retourne les session_label ordonnées par date du dernier match (plus récent en premier).

    Robuste au type de session_id (stocké VARCHAR ou calculé) et à la logique 4h (Cas A/B).
    """
    base_s = _to_polars(base_s)
    if (
        base_s.is_empty()
        or "start_time" not in base_s.columns
        or "session_label" not in base_s.columns
    ):
        return []
    agg = (
        base_s.group_by(["session_id", "session_label"])
        .agg(pl.col("start_time").max())
        .sort("start_time", descending=True)
    )
    return agg["session_label"].to_list()
