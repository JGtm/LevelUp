"""Coordination cache + session_state après opérations modifiant la DB.

Responsabilité unique : invalider atomiquement le cache @st.cache_data
et les entrées session_state de filtres/buster après tout sync réussi.

Ce module appartient à la couche `app/` — seul niveau autorisé à écrire
dans `st.session_state`.
"""

from __future__ import annotations

import logging

import streamlit as st

from src.app.session_keys import SK
from src.ui.cache import clear_app_caches

logger = logging.getLogger(__name__)


def invalidate_after_sync() -> None:
    """Invalide cache + filtres session_state après tout sync réussi.

    Actions effectuées dans l'ordre :
    1. Vide tous les caches @st.cache_data (clear_app_caches).
    2. Incrémente SK.CACHE_BUSTER pour forcer le rechargement des données.
    3. Supprime toutes les clés ``_filters_loaded_*`` et ``_filters_db_key_*``
       afin que _init_filter_preferences() réinitialise les plages de dates
       et filtres au prochain rerun.

    L'appelant est responsable d'appeler st.rerun() après cette fonction.
    """
    clear_app_caches()

    buster = st.session_state.get(SK.CACHE_BUSTER, 0) + 1
    st.session_state[SK.CACHE_BUSTER] = buster

    removed = [
        k
        for k in list(st.session_state.keys())
        if k.startswith("_filters_loaded_") or k.startswith("_filters_db_key_")
    ]
    for k in removed:
        del st.session_state[k]

    logger.info(
        "invalidate_after_sync: cache_buster=%d, clés filtres supprimées=%s",
        buster,
        removed or "aucune",
    )
