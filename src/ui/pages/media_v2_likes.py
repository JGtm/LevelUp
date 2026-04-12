"""Contrôles like pour la grille Media V2."""

from __future__ import annotations

import base64
import hashlib
from functools import lru_cache
from pathlib import Path

import streamlit as st

from src.ui.components.browser_storage import load_media_likes, toggle_media_like
from src.ui.streamlit_modern import fragment_if_available

_UI_ICONS_DIR: Path = Path(__file__).resolve().parents[3] / "static" / "ui-icons"
_HEART_ICON_PATH: Path = _UI_ICONS_DIR / "heart_16.png"
_HEART_ICON_NOT_LIKED_PATH: Path = _UI_ICONS_DIR / "heart_16_not_liked.png"

HEART_BUTTON_BASE_CSS = """<style>
div[data-testid="stBaseButton-tertiary"],
div[data-testid="stButton"] { margin: 0 !important; }
</style>"""


def _like_callback_flag_key(file_path: str) -> str:
    """Retourne la clé session_state utilisée pour tracer le callback like."""
    hk = hashlib.md5(file_path.encode()).hexdigest()[:12]
    return f"_mv2_like_callback_{hk}"


def _toggle_like_callback(file_path: str) -> None:
    """Bascule le like et marque le callback comme exécuté pour ce clic."""
    toggle_media_like(file_path)
    st.session_state[_like_callback_flag_key(file_path)] = True


def _sync_like_click(file_path: str, *, clicked: bool, force_full_rerun: bool) -> None:
    """Applique le fallback de clic pour les tests/mocks sans doubler le toggle."""
    if not clicked:
        return

    callback_ran = bool(st.session_state.pop(_like_callback_flag_key(file_path), False))
    if not callback_ran:
        _toggle_like_callback(file_path)
    if force_full_rerun:
        st.rerun()


@fragment_if_available
def render_media_like_button(file_path: str, *, force_full_rerun: bool = False) -> None:
    """Rend un like inline minimaliste sous forme d'un seul bouton."""
    liked = file_path in load_media_likes()
    label = "1" if liked else "0"
    hk = hashlib.md5(file_path.encode()).hexdigest()[:12]
    st.markdown(
        _heart_button_css(key=f"mv2_heart_{hk}", liked=liked),
        unsafe_allow_html=True,
    )
    clicked = st.button(
        label,
        key=f"mv2_heart_{hk}",
        type="tertiary",
        on_click=_toggle_like_callback,
        args=(file_path,),
    )
    _sync_like_click(file_path, clicked=clicked, force_full_rerun=force_full_rerun)


def _heart_button_css(*, key: str, liked: bool) -> str:
    """Retourne le CSS ciblé pour un bouton like précis."""
    icon_uri = _load_heart_icon_data_uri(liked)
    return f"""<style>
div.st-key-{key} button[data-testid="stBaseButton-tertiary"] {{
    all: unset !important;
    display: inline-flex !important;
    align-items: center !important;
    gap: 0.32rem !important;
    background: transparent !important;
    border: 0 !important;
    box-shadow: none !important;
    padding: 0 !important;
    margin: 0 !important;
    min-height: 0 !important;
    height: 24px !important;
    width: auto !important;
    line-height: 1 !important;
    font-size: 0.82rem !important;
    color: #c7ced8 !important;
    cursor: pointer !important;
    white-space: nowrap !important;
}}
div.st-key-{key} button[data-testid="stBaseButton-tertiary"]::before {{
    content: "";
    display: inline-block;
    width: 20px;
    height: 20px;
    background: url('{icon_uri}') center/contain no-repeat;
    image-rendering: pixelated;
    flex: 0 0 20px;
}}
div.st-key-{key} button[data-testid="stBaseButton-tertiary"] p {{
    margin: 0 !important;
    line-height: 1 !important;
}}
</style>"""


@lru_cache(maxsize=2)
def _load_heart_icon_data_uri(liked: bool) -> str:
    """Charge l'icône coeur locale et la convertit en data URI."""
    icon_path = _HEART_ICON_PATH if liked else _HEART_ICON_NOT_LIKED_PATH
    try:
        raw = icon_path.read_bytes()
    except OSError:
        return ""
    encoded = base64.b64encode(raw).decode("ascii")
    return f"data:image/png;base64,{encoded}"
