"""Traductions UI — bilingue FR/EN.

Centralise les mappings de libellés (playlist, mode/pair) afin de :
- réduire la liste de valeurs distinctes dans l'UI
- afficher des labels localisés (FR par défaut, EN optionnel)

Toutes les fonctions publiques acceptent un paramètre ``lang: str = "fr"``.
La valeur ``"fr"`` est le défaut pour assurer la rétrocompatibilité.

Source de vérité : fichiers JSON ``static/i18n/playlists_{lang}.json`` et
``static/i18n/modes_{lang}.json``.  Les dictionnaires hardcodés ci-dessous
servent de **fallback** historique.
"""

from __future__ import annotations

from src.ui.i18n.data_labels import label, load_domain

PLAYLIST_FR: dict[str, str] = {
    "Big Team Battle": "Grande bataille en équipe",
    "Big Team Battle: Refresh": "Grande bataille en équipe",
    "Big Team Social": "Grande bataille sociale",
    "Firefight": "Baptême du feu",
    "Firefight: Heroic King of the Hill": "Baptême du feu : Roi de la colline héroïque",
    "Firefight: Legendary King of the Hill": "Baptême du feu : Roi de la colline légendaire",
    "Quick Play": "Partie rapide",
    "Ranked Arena": "Arène classée",
    "Ranked Slayer": "Assassin classé",
    "Rumble Pit": "Mêlée générale",
    "SURVIVE THE UNDEAD": "Survivre aux morts-vivants",
    "Squad Battle": "Combat d'escouade",
    "Super Fiesta": "Super Fiesta",
    "Team Snipers": "Snipers en équipe",
    # IDs de playlists "Partie rapide" (fallback si nom non résolu)
    "a446725e-b281-414c-a21e": "Partie rapide",
    "bdceefb3-1c52-4848-a6b7": "Partie rapide",
}

# Mappings EN — nettoyage des noms de playlists (variants, UUIDs, etc.)
PLAYLIST_EN: dict[str, str] = {
    "Big Team Battle": "Big Team Battle",
    "Big Team Battle: Refresh": "Big Team Battle",
    "Big Team Social": "Big Team Social",
    "Firefight": "Firefight",
    "Firefight: Heroic King of the Hill": "Firefight",
    "Firefight: Legendary King of the Hill": "Firefight",
    "Quick Play": "Quick Play",
    "Ranked Arena": "Ranked Arena",
    "Ranked Slayer": "Ranked Slayer",
    "Rumble Pit": "Rumble Pit",
    "SURVIVE THE UNDEAD": "Survive the Undead",
    "Squad Battle": "Squad Battle",
    "Super Fiesta": "Super Fiesta",
    "Team Snipers": "Team Snipers",
    "a446725e-b281-414c-a21e": "Quick Play",
    "bdceefb3-1c52-4848-a6b7": "Quick Play",
}


# NOTE: PAIR_FR est vide — les traductions sont désormais dans
# static/i18n/modes_fr.json (_pairs pour les overrides, combinateur _prefixes+mode
# pour le reste). Conserver pour rétrocompatibilité des imports externes.
PAIR_FR: dict[str, str] = {}


def _is_uuid_like(s: str) -> bool:
    """Vérifie si une chaîne ressemble à un UUID (ex: a446725e-b281-414c-a21e)."""
    import re

    # UUID complet ou partiel (au moins 8 caractères hex avec tirets)
    return bool(re.match(r"^[a-f0-9]{8}(-[a-f0-9]{4}){0,3}(-[a-f0-9]{1,12})?$", s.lower()))


def translate_playlist_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un nom de playlist dans la langue demandée.

    Résolution : JSON i18n → dicts hardcodés → valeur brute.

    Args:
        name: Nom de playlist brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.

    Returns:
        Label localisé, ou ``None`` si ``name`` est vide/None.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None
    # Détection des UUIDs non résolus
    if _is_uuid_like(s):
        # Vérifier d'abord dans le JSON (les UUIDs peuvent y être mappés)
        json_val = label("playlists", s, lang=lang)
        if json_val != s:
            return json_val
        return label("playlists", "_unknown", lang=lang)
    # 1) JSON i18n
    json_val = label("playlists", s, lang=lang)
    if json_val != s:
        return json_val
    # 2) Fallback dicts hardcodés
    if lang == "en":
        return PLAYLIST_EN.get(s, s)
    return PLAYLIST_FR.get(s, s)


def _normalize_pair_case(s: str) -> str:
    """Normalise la casse d'un pair_name (préfixe canonique + mode titlecase).

    Exemples : ``"arena:slayer"`` → ``"Arena:Slayer"``
    """
    if ":" not in s:
        return s
    prefix, rest = s.split(":", 1)
    prefix = prefix.strip()
    rest = rest.strip()
    prefix_lower = prefix.lower()
    if prefix_lower == "btb heavies":
        prefix = "BTB Heavies"
    elif prefix_lower == "btb":
        prefix = "BTB"
    elif prefix_lower == "super fiesta":
        prefix = "Super Fiesta"
    elif prefix_lower == "super husky raid":
        prefix = "Super Husky Raid"
    elif prefix_lower == "husky raid":
        prefix = "Husky Raid"
    else:
        prefix = prefix[:1].upper() + prefix[1:].lower()
    if rest and rest == rest.lower():
        rest = " ".join(w[:1].upper() + w[1:] for w in rest.split())
    return f"{prefix}:{rest}" if prefix else rest


def translate_pair_name(name: str | None, lang: str = "fr") -> str | None:  # noqa: C901
    """Traduit un nom de mode/pair dans la langue demandée.

    Stratégie (source de vérité : static/i18n/modes_{lang}.json) :
    1. Override exact depuis ``_pairs`` (renommages spéciaux ou carte-spécifiques)
    2. Normalisation douce de la casse + retry ``_pairs``
    3. Strip " on <carte>" + retry ``_pairs``
    4. Combinateur ``_prefixes`` + clé mode (logique générique)
    5. Mode seul (pas de préfixe)
    6. UUID non résolu → "Mode inconnu" / "Unknown mode"

    Args:
        name: Nom de pair brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None

    # UUID non résolu
    if _is_uuid_like(s):
        return label("modes", "_unknown", lang=lang)

    # Charger le domaine une seule fois (mis en cache par load_domain)
    modes_data = load_domain("modes", lang)
    pairs = modes_data.get("_pairs", {})
    prefixes = modes_data.get("_prefixes", {})
    separator = modes_data.get("_separator", ": ")

    # 1) Override exact
    if s in pairs:
        return pairs[s]

    # 2) Normalisation douce de la casse
    candidate = _normalize_pair_case(s)
    if candidate in pairs:
        return pairs[candidate]

    # 3) Strip " on <carte>" + retry _pairs
    mode_without_map = candidate
    if " on " in candidate:
        mode_without_map = candidate.split(" on ", 1)[0].strip()
        if mode_without_map in pairs:
            return pairs[mode_without_map]

    # 4) Combinateur : _prefixes + clé mode
    if ":" in mode_without_map:
        prefix_raw, mode_part = mode_without_map.split(":", 1)
        prefix_loc = prefixes.get(prefix_raw.strip(), prefix_raw.strip())
        mode_loc = modes_data.get(mode_part.strip(), mode_part.strip())
        return f"{prefix_loc}{separator}{mode_loc}"

    # 5) Mode seul
    mode_loc = modes_data.get(mode_without_map)
    if mode_loc and not isinstance(mode_loc, dict):
        return str(mode_loc)

    return s
