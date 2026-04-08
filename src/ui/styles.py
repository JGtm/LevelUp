"""Gestion des styles CSS."""

from __future__ import annotations

import logging
import os

_log = logging.getLogger(__name__)


def get_css_path() -> str:
    """Retourne le chemin du fichier CSS."""
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    return os.path.join(
        repo_root,
        "static",
        "styles.css",
    )


def load_css() -> str:
    """Charge le contenu du fichier CSS (sans cache pour le dev).

    Returns:
        Contenu CSS avec balises <style>.
    """
    css_path = get_css_path()

    try:
        with open(css_path, encoding="utf-8") as f:
            css_content = f.read()
        return f"<style>\n{css_content}\n</style>"
    except FileNotFoundError:
        _log.warning("CSS introuvable: %s — fallback minimal", css_path)
        # Fallback: CSS minimal si le fichier n'existe pas
        return """
        <style>
            .hero { padding: 18px; margin-bottom: 14px; }
            .hero .title { font-size: 28px; font-weight: 700; }
            .hero .subtitle { color: #aaa; font-size: 14px; }
        </style>
        """


def get_hero_html(  # noqa: PLR0913
    *,
    player_name: str | None = None,
    service_tag: str | None = None,
    adornment_path: str | None = None,
    backdrop_path: str | None = None,
    nameplate_path: str | None = None,
    emblem_path: str | None = None,
    grid_mode: bool = False,
) -> str:
    """Génère le HTML du Spartan ID (façade vers render_spartan_id).

    Le Spartan ID est la carte visuelle standard Halo : backdrop, nameplate,
    emblème, adornment, gamertag et service tag.

    Pour construire le composant depuis des données typées, préférer directement
    ``SpartanIdCard`` + ``render_spartan_id()`` (src.ui.spartan_id).

    Args:
        player_name: Gamertag du joueur. Si vide, renvoie le hero par défaut LevelUp.
        service_tag: Tag de clan court (ex: «NS»).
        adornment_path: Chemin local vers le badge de rang.
        backdrop_path: Chemin local vers l'image de fond.
        nameplate_path: Chemin local vers la bannière principale.
        emblem_path: Chemin local vers le logo du joueur.
        grid_mode: Mode compact sans notches (grilles coéquipiers).
    """
    from src.ui.spartan_id import SpartanIdCard, render_spartan_id

    card = SpartanIdCard(
        gamertag=player_name or "",
        service_tag=service_tag,
        backdrop_path=backdrop_path,
        nameplate_path=nameplate_path,
        emblem_path=emblem_path,
        adornment_path=adornment_path,
    )
    return render_spartan_id(card, grid_mode=grid_mode)


def get_notches_html() -> str:
    """Retourne le HTML des découpes haut/bas."""
    return """
    <div class="wp-notch-top"></div>
    <div class="wp-notch-bottom"></div>
    """
