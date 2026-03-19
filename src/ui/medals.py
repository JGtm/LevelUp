"""Gestion des médailles : labels, icônes et affichage en grille.

Ce module centralise les fonctions UI liées aux médailles Halo Infinite :
- Chargement des labels depuis ``medal_definitions`` (metadata.duckdb)
- Récupération des icônes embarquées (static/medals/icons/)
- Affichage d'une grille de médailles dans Streamlit
"""

from __future__ import annotations

import logging
import os

import streamlit as st

from src.utils.db import duckdb_read_only
from src.utils.paths import get_metadata_db_path

__all__ = [
    "load_medal_name_maps",
    "medal_has_known_label",
    "get_local_medals_icons_dir",
    "medal_label",
    "medal_icon_path",
    "render_medals_grid",
]

logger = logging.getLogger(__name__)


@st.cache_data(show_spinner=False)
def load_medal_name_maps() -> tuple[dict[str, str], dict[str, str]]:
    """Charge les labels de médailles depuis ``metadata.medal_definitions``.

    Returns:
        Tuple (fr_map, en_map) où chaque map est {str(NameId): "Label"}.
    """
    db_path = get_metadata_db_path()
    if not db_path.exists():
        logger.warning("metadata.duckdb introuvable : %s", db_path)
        return {}, {}

    with duckdb_read_only(db_path) as conn:
        try:
            rows = conn.execute(
                "SELECT medal_name_id, name_fr, name_en FROM medal_definitions"
            ).fetchall()
        except Exception:
            logger.debug("Erreur requête medal_definitions")
            return {}, {}

    fr_map = {str(r[0]): r[1] for r in rows if r[1]}
    en_map = {str(r[0]): r[2] for r in rows if r[2]}
    return fr_map, en_map


def get_local_medals_icons_dir() -> str:
    """Retourne le dossier d'icônes médailles embarquées dans le repo.

    Returns:
        Chemin absolu vers static/medals/icons
    """
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    return os.path.join(repo_root, "static", "medals", "icons")


def medal_has_known_label(nid: int) -> bool:
    """Vérifie si une médaille a un label connu (FR ou EN).

    Args:
        nid: NameId de la médaille.

    Returns:
        True si un label existe, False sinon.
    """
    fr_map, en_map = load_medal_name_maps()
    key = str(int(nid))
    return key in fr_map or key in en_map


def medal_label(nid: int, lang: str = "fr") -> str:
    """Retourne le label d'une médaille selon la langue.

    Args:
        nid: NameId de la médaille.
        lang: Langue cible ("fr" ou "en").

    Returns:
        Label de la médaille ou générique si inconnu.
    """
    fr_map, en_map = load_medal_name_maps()
    key = str(int(nid))
    if lang == "en":
        return en_map.get(key) or fr_map.get(key) or f"Medal #{nid}"
    return fr_map.get(key) or en_map.get(key) or f"Médaille #{nid}"


def medal_icon_path(nid: int) -> str | None:
    """Retourne le chemin de l'icône PNG d'une médaille si elle existe.

    Args:
        nid: NameId de la médaille.

    Returns:
        Chemin absolu vers l'icône ou None si introuvable.
    """
    local_p = os.path.join(get_local_medals_icons_dir(), f"{int(nid)}.png")
    return local_p if os.path.exists(local_p) else None


def render_medals_grid(
    medals: list[dict[str, int]],
    cols_per_row: int = 8,
    deltas: dict[int, int] | None = None,
    center: bool = False,
    lang: str = "fr",
) -> None:
    """Affiche une grille de médailles dans Streamlit.

    Args:
        medals: Liste de dicts avec 'name_id' et 'count'.
        cols_per_row: Nombre de colonnes (3-12, défaut 8).
        deltas: Dict {medal_id: delta_count} pour afficher +XXX à côté du compteur.
        center: Si True et que le nombre de médailles < cols_per_row, centre la grille.
    """
    if not medals:
        st.info("Aucune médaille.")
        return

    # Note: on n'affiche plus de warning pour les médailles inconnues
    # (certaines comme #590706932 sont des médailles internes/test à ignorer)

    local_dir = get_local_medals_icons_dir()
    if not os.path.isdir(local_dir):
        st.caption(
            "Icônes de médailles introuvables. "
            "Utilise scripts/sync_medal_icons.py pour copier les PNG en local."
        )

    cols_per_row = max(3, min(int(cols_per_row), 12))
    n = len(medals)
    actual_cols = cols_per_row
    if center and 0 < n < cols_per_row:
        _pad = max(1, (cols_per_row - n) // 2)
        _all_cols = st.columns([_pad] + [1] * n + [_pad])
        cols = _all_cols[1 : 1 + n]
        actual_cols = n
    else:
        cols = st.columns(cols_per_row)
    for i, m in enumerate(medals):
        col = cols[i % actual_cols]
        nid = int(m.get("name_id", 0))
        cnt = int(m.get("count", 0))
        name = medal_label(nid, lang=lang)
        icon = medal_icon_path(nid)

        if icon:
            col.image(icon, width="stretch")
        else:
            col.markdown(
                f"<div class='os-medal-missing'>#{nid}</div>",
                unsafe_allow_html=True,
            )

        # Afficher le delta si fourni
        delta_html = ""
        if deltas is not None and nid in deltas:
            delta_val = deltas[nid]
            if delta_val > 0:
                delta_html = (
                    f" <span style='color: #4CAF50; font-weight: bold;'>+{delta_val}</span>"
                )

        col.markdown(
            "<div class='os-medal-caption'>"
            + "<div class='os-medal-name'>"
            + name
            + "</div>"
            + "<div class='os-medal-count'>x"
            + str(cnt)
            + delta_html
            + "</div>"
            + "</div>",
            unsafe_allow_html=True,
        )
