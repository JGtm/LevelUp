"""Composant Spartan ID — carte visuelle d'identité Halo Infinite.

Le Spartan ID est la carte d'identité visuelle standard du jeu, composée de :
- backdrop   : fond rectangulaire (≈2/3 de la largeur)
- nameplate  : bannière principale (plan intermédiaire)
- emblem     : logo du joueur (overlay gauche)
- adornment  : badge de rang (overlay droit)
- gamertag   : nom du joueur
- service_tag: tag de clan (ex : «NS»)

Ce module expose :
- ``SpartanIdCard``    — dataclass représentant les données visuelles
- ``render_spartan_id`` — rendu HTML du composant

Toute intégration du Spartan ID dans l'app doit passer par ces deux objets.
Ne jamais reconstruire le HTML manuellement.
"""

from __future__ import annotations

import html as _html
from dataclasses import dataclass

from src.ui.player_assets import file_to_data_url


@dataclass(frozen=True)
class SpartanIdCard:
    """Données visuelles du Spartan ID d'un joueur Halo Infinite.

    Contient exactement les informations nécessaires au rendu HTML.
    Utiliser ``render_spartan_id()`` pour produire le HTML correspondant.

    Attributes:
        gamertag: Nom du joueur (obligatoire).
        service_tag: Tag de clan court (ex: «NS»), optionnel.
        backdrop_path: Chemin local vers l'image de fond.
        nameplate_path: Chemin local vers la bannière principale.
        emblem_path: Chemin local vers le logo du joueur.
        adornment_path: Chemin local vers le badge de rang.
    """

    gamertag: str
    service_tag: str | None = None
    backdrop_path: str | None = None
    nameplate_path: str | None = None
    emblem_path: str | None = None
    adornment_path: str | None = None


def render_spartan_id(card: SpartanIdCard, *, grid_mode: bool = False) -> str:
    """Génère le HTML du Spartan ID.

    Structure visuelle (de l'arrière vers l'avant) :
    1. backdrop   — fond (≈2/3 de la largeur)
    2. nameplate  — bannière principale
    3. emblem     — logo du joueur (overlay gauche)
    4. gamertag + service_tag — texte centré
    5. adornment  — badge de rang (overlay droit)

    Args:
        card: Données visuelles du Spartan ID.
        grid_mode: Mode compact pour les grilles coéquipiers (sans notches).

    Returns:
        HTML complet du composant Spartan ID.
    """
    if not card.gamertag.strip():
        return _default_hero_html()

    backdrop_data = file_to_data_url(card.backdrop_path, max_bytes=8 * 1024 * 1024)
    nameplate_data = file_to_data_url(card.nameplate_path)
    emblem_data = file_to_data_url(card.emblem_path)
    adornment_data = file_to_data_url(card.adornment_path)

    safe_gamertag = _html.escape(card.gamertag.strip())
    safe_service_tag = _html.escape((card.service_tag or "").strip())

    backdrop_html = (
        f"<div class='spartan-id__backdrop'><img src='{backdrop_data}' alt='' /></div>"
        if backdrop_data
        else ""
    )
    nameplate_html = (
        f"<div class='spartan-id__nameplate'><img src='{nameplate_data}' alt='' /></div>"
        if nameplate_data
        else ""
    )
    emblem_html = (
        f"<div class='spartan-id__emblem'><img src='{emblem_data}' alt='emblem' /></div>"
        if emblem_data
        else ""
    )
    adornment_html = (
        f"<div class='spartan-id__adornment'><img src='{adornment_data}' alt='' /></div>"
        if adornment_data
        else ""
    )
    service_tag_html = (
        f"<div class='spartan-id__servicetag'>{safe_service_tag}</div>" if safe_service_tag else ""
    )

    wrapper_class = (
        "spartan-id-wrapper spartan-id-wrapper--grid" if grid_mode else "spartan-id-wrapper"
    )
    notches = (
        "" if grid_mode else "<div class='wp-notch-top'></div><div class='wp-notch-bottom'></div>"
    )

    return (
        f"{notches}"
        f"<div class='{wrapper_class}'>"
        "  <div class='spartan-id'>"
        f"    {backdrop_html}"
        f"    {nameplate_html}"
        f"    {emblem_html}"
        "    <div class='spartan-id__text'>"
        f"      <div class='spartan-id__gamertag'>{safe_gamertag}</div>"
        f"      {service_tag_html}"
        "    </div>"
        f"    {adornment_html}"
        "  </div>"
        "</div>"
    )


def _default_hero_html() -> str:
    """HTML de fallback quand aucun joueur n'est actif."""
    return """
    <div class="wp-notch-top"></div>
    <div class="wp-notch-bottom"></div>
    <div class="hero">
        <div class="title">LevelUp</div>
        <div class="subtitle">Analyse tes parties Halo Infinite depuis la DB SPNKr — filtres, séries temporelles, amis, maps.</div>
    </div>
    """
