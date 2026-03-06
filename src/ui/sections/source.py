"""Sections UI: configuration de la source (DB/XUID), profils et alias."""

from __future__ import annotations

import contextlib
import os
from collections.abc import Callable, Mapping

import streamlit as st

from src.app.data_loader import default_identity_from_secrets
from src.ui.aliases import display_name_from_xuid
from src.utils import (
    guess_xuid_from_db_path,
    load_profiles,
    resolve_xuid_from_db,
)


def _default_identity_from_secrets() -> tuple[str, str]:
    """Retourne (xuid_or_gamertag, waypoint_player) depuis secrets/env/constants."""
    xuid_or_gt, _xuid_fallback, wp = default_identity_from_secrets()
    return xuid_or_gt, wp


def render_source_section(
    default_db: str,
    *,
    get_local_dbs: Callable[[], list[str]],
    on_clear_caches: Callable[[], None],
) -> tuple[str, str, str]:
    """Rend la section "Source" dans la sidebar.

    Returns:
        (db_path, xuid, waypoint_player)
    """

    # --- Multi-DB / Profils ---
    _ = load_profiles()  # Précharge les profils

    if "db_path" not in st.session_state:
        st.session_state["db_path"] = default_db
    # IMPORTANT Streamlit: ne pas modifier une key de widget après instanciation.
    # On sépare donc l'entrée utilisateur (xuid_input) du XUID effectivement utilisé (résolu plus bas).
    if "xuid_input" not in st.session_state:
        # migration douce depuis l'ancien key "xuid" s'il existe encore
        legacy = str(st.session_state.get("xuid", "") or "").strip()
        guessed = guess_xuid_from_db_path(st.session_state.get("db_path", "")) or ""
        env_player = (os.environ.get("SPNKR_PLAYER") or "").strip()
        secret_player, _secret_wp = _default_identity_from_secrets()
        st.session_state["xuid_input"] = legacy or guessed or env_player or secret_player
    if "waypoint_player" not in st.session_state:
        _secret_player, secret_wp = _default_identity_from_secrets()
        st.session_state["waypoint_player"] = secret_wp

    # DuckDB v4 uniquement (SQLite legacy supprimé)

    c_top = st.columns(2)
    if c_top[0].button("Vider caches", width="stretch"):
        on_clear_caches()
        st.success("Caches vidés.")
        st.rerun()
    if c_top[1].button("Rafraîchir", width="stretch"):
        # Le get_local_dbs est censé être caché côté app (ttl) ; on le force via clear/closure.
        with contextlib.suppress(Exception):
            get_local_dbs.clear()  # st.cache_data wrapper
        st.rerun()

    # DuckDB v4 : la DB est sélectionnée via launcher ou session_state

    # Récupérer le db_path depuis session_state (pas d'UI manuelle)
    db_path = str(st.session_state.get("db_path", "") or "").strip()

    # Identité: résoudre le XUID depuis la DB
    raw_identity = str(st.session_state.get("xuid_input", "") or "").strip()
    xuid = resolve_xuid_from_db(str(db_path), raw_identity) or raw_identity
    if xuid and (not str(xuid).strip().isdigit()) and raw_identity and (not raw_identity.isdigit()):
        # Même fallback que le bouton: secrets → xuid
        try:
            player = st.secrets.get("player", {})
            if isinstance(player, Mapping):
                gt = str(player.get("gamertag") or "").strip()
                xu = str(player.get("xuid") or "").strip()
                if gt and xu and gt.casefold() == raw_identity.casefold():
                    xuid = xu
        except Exception:
            pass

    name_guess = display_name_from_xuid(str(xuid or "").strip())
    waypoint_player = str(st.session_state.get("waypoint_player", "") or "").strip()
    if name_guess and name_guess != "-":
        waypoint_player = str(name_guess).strip()
        st.session_state["waypoint_player"] = waypoint_player

    # On garde une valeur non vide (fallback secrets)
    if not waypoint_player:
        _secret_player, secret_wp = _default_identity_from_secrets()
        waypoint_player = secret_wp
        st.session_state["waypoint_player"] = waypoint_player

    # Alias (XUID → gamertag) : UI masquée.

    return str(db_path), str(xuid), str(waypoint_player)
