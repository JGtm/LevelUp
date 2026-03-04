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


# NOTE: ce mapping cible MatchStats.PlaylistMapModePairs (pair_name)
# Exemple: "Arena:CTF on Aquarius" -> "Arène : Capture du drapeau"
PAIR_FR: dict[str, str] = {
    # -------------------------------------------------------------------------
    # Variantes sans carte (fallback génériques)
    # -------------------------------------------------------------------------
    "Arena:CTF": "Arène : Capture du drapeau",
    "Arena:King of the Hill": "Arène : Roi de la colline",
    "Arena:Neutral Flag CTF": "Arène : Drapeau neutre",
    "Arena:Oddball": "Arène : Oddball",
    "Arena:Slayer": "Arène : Assassin",
    "Arena:Team Slayer": "Arène : Assassin en équipe",
    "Arena:Strongholds": "Arène : Bases",
    "Arena:Attrition": "Arène : Attrition",
    "Arena:One Flag CTF": "Arène : Drapeau neutre",
    "Arena:Escalation Slayer": "Arène : Escalade",
    "Arena:FFA Slayer": "Arène : Assassin FFA",
    "Arena:Shotty Snipes Slayer": "Fusils snipers à grenaille",
    "Arena:Team Snipers": "Arène : Snipers en équipe",
    "BTB:CTF": "BTB : Capture du drapeau",
    "BTB:Slayer": "BTB : Assassin",
    "BTB:Total Control": "BTB : Contrôle total",
    "BTB:Stockpile": "BTB : Stockage",
    "BTB:Fiesta CTF": "BTB : Fiesta CDD",
    "BTB:Fiesta Slayer": "BTB : Fiesta Assassin",
    "BTB:Fiesta Total Control": "BTB : Fiesta Contrôle total",
    "BTB:One Flag CTF": "BTB : Drapeau neutre",
    "BTB:Extraction": "BTB : Extraction",
    "BTB:Escalation Slayer": "BTB : Escalade",
    "BTB:Sentry Defense": "BTB : Défense sentinelle",
    "BTB:Team Snipers": "BTB : Snipers en équipe",
    "BTB Heavies:CTF": "BTB Heavies : Capture du drapeau",
    "BTB Heavies:Slayer": "BTB Heavies : Assassin",
    "BTB Heavies:Total Control": "BTB Heavies : Contrôle total",
    "Ranked:CTF": "Classé : Capture du drapeau",
    "Ranked:Slayer": "Classé : Assassin",
    "Ranked:Oddball": "Classé : Oddball",
    "Ranked:Strongholds": "Classé : Bases",
    "Ranked:King of the Hill": "Classé : Roi de la colline",
    "Tactical:Slayer": "Tactique : Assassin",
    "Community:Slayer": "Communauté : Assassin",
    "Community:Team Slayer": "Communauté : Assassin en équipe",
    "Event:Escalation Slayer": "Événement : Escalade",
    "Super Fiesta:Slayer": "Super Fiesta : Assassin",
    "Fiesta:FFA Slayer": "Fiesta : Assassin FFA",
    "Firefight:Heroic King of the Hill": "Baptême du feu : Roi de la colline héroïque",
    "Firefight:Legendary King of the Hill": "Baptême du feu : Roi de la colline légendaire",
    "Gruntpocalypse:Heroic KOTH": "Gruntpocalypse : Roi de la colline héroïque",
    "Husky Raid:CTF": "Husky Raid : CDD",
    "Super Husky Raid:CTF": "Super Husky Raid : CDD",
    "Assault:Neutral Bomb": "Arène : Bombe neutre",
    "Assault:Neutral Bomb Squad": "Arène : Escouade bombe neutre",
    # -------------------------------------------------------------------------
    # Arena
    # -------------------------------------------------------------------------
    "Arena:VIP on Catalyst": "Arène : VIP",
    # -------------------------------------------------------------------------
    # Assault
    # -------------------------------------------------------------------------
    "Assault:One Bomb on Curfew": "Arène : Bombe neutre",
    # -------------------------------------------------------------------------
    # BTB (Big Team Battle)
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # BTB Heavies
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # Community
    # -------------------------------------------------------------------------
    "Community:Fiesta Slayer on High Ground": "Fiesta",
    "Community:Fiesta Slayer on Snowbound": "Fiesta",
    "Community:Shotty Snipe Slayer FFA on Dynasty": "Fusils snipers à grenaille",
    # -------------------------------------------------------------------------
    # Event
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # Fiesta
    # -------------------------------------------------------------------------
    "Fiesta:Slayer on Behemoth - Forge": "Fiesta",
    "Fiesta:Slayer on Catalyst - Forge": "Fiesta",
    # -------------------------------------------------------------------------
    # Firefight / Gruntpocalypse
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # Husky Raid / Super Husky Raid
    # -------------------------------------------------------------------------
    "Husky Raid:Assault on Urban Raid": "Husky Raid",
    # -------------------------------------------------------------------------
    # Ranked
    # -------------------------------------------------------------------------
    "Ranked:CTF 3 Captures on Argyle": "Classé : CDD 3 captures",
    # -------------------------------------------------------------------------
    # Super Fiesta
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # Tactical
    # -------------------------------------------------------------------------
    # -------------------------------------------------------------------------
    # Autres / Events spéciaux
    # -------------------------------------------------------------------------
    "Survive The Undead 3.0 on TFF | Night Of The Undead": "Survivre aux morts-vivants 3.0",
}

# Traduction des modes génériques côté EN  (même logique d'agrégation qu'en FR)
_EN_GENERIC_MODES: dict[str, str] = {
    "Slayer": "Slayer",
    "Team Slayer": "Team Slayer",
    "FFA Slayer": "FFA Slayer",
    "Fiesta Slayer": "Fiesta Slayer",
    "Fiesta CTF": "Fiesta CTF",
    "Fiesta Total Control": "Fiesta Total Control",
    "Oddball": "Oddball",
    "CTF": "CTF",
    "CTF 3 Captures": "CTF (3 Captures)",
    "Neutral Flag CTF": "Neutral Flag CTF",
    "One Flag CTF": "One Flag CTF",
    "King of the Hill": "King of the Hill",
    "Heroic King of the Hill": "King of the Hill (Heroic)",
    "Legendary King of the Hill": "King of the Hill (Legendary)",
    "Heroic KOTH": "King of the Hill (Heroic)",
    "Strongholds": "Strongholds",
    "Attrition": "Attrition",
    "Escalation Slayer": "Escalation Slayer",
    "Team Snipers": "Team Snipers",
    "Shotty Snipes Slayer": "Shotty Snipers",
    "Shotty Snipe Slayer FFA": "Shotty Snipers FFA",
    "Total Control": "Total Control",
    "Stockpile": "Stockpile",
    "Extraction": "Extraction",
    "Land Grab": "Land Grab",
    "VIP": "VIP",
    "Sentry Defense": "Sentry Defense",
    "Assault": "Assault",
    "Neutral Bomb": "Neutral Bomb",
    "Neutral Bomb Squad": "Neutral Bomb Squad",
    "One Bomb": "One Bomb",
}

_EN_PREFIXES: dict[str, str] = {
    "Arena": "Arena",
    "BTB": "BTB",
    "BTB Heavies": "BTB Heavies",
    "Ranked": "Ranked",
    "Tactical": "Tactical",
    "Community": "Community",
    "Event": "Event",
    "Fiesta": "Fiesta",
    "Super Fiesta": "Super Fiesta",
    "Firefight": "Firefight",
    "Gruntpocalypse": "Gruntpocalypse",
    "Husky Raid": "Husky Raid",
    "Super Husky Raid": "Super Husky Raid",
    "Assault": "Assault",
}


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


def translate_pair_name(name: str | None, lang: str = "fr") -> str | None:  # noqa: C901, PLR0912
    """Traduit un nom de mode/pair dans la langue demandée.

    Stratégie de fallback:
    1. JSON i18n (modes)
    2. Match exact dans PAIR_FR (FR)
    3. Normalisation de la casse et retry
    4. Match par préfixe (mode sans carte, logique d'agrégation préservée)
    5. Fallback générique avec dictionnaires JSON localisés
    6. Si UUID non résolu -> "Mode inconnu" / "Unknown mode"

    Args:
        name: Nom de pair brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None

    # Détection précoce des UUIDs non résolus
    if _is_uuid_like(s):
        return label("modes", "_unknown", lang=lang)

    # EN : pas de PAIR_EN exhaustif — on passe directement à la logique de normalisation
    # FR : 1) Match exact hardcodé (legacy, PAIR_FR a ~300 entrées)
    if lang == "fr" and s in PAIR_FR:
        return PAIR_FR[s]

    # 2) Normalisation douce (casse) pour supporter des valeurs du type "arena:Team Slayer".
    candidate = s
    if ":" in s:
        prefix, rest = s.split(":", 1)
        prefix = prefix.strip()
        rest = rest.strip()
        if prefix:
            # Préfixes multi-mots à préserver
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
        # Si la partie mode est totalement en minuscules, on la TitleCase ("oddball" -> "Oddball").
        if rest and rest == rest.lower():
            rest = " ".join(w[:1].upper() + w[1:] for w in rest.split())
        candidate = f"{prefix}:{rest}" if prefix else rest

    if lang == "fr" and candidate in PAIR_FR:
        return PAIR_FR[candidate]

    # 3) Fallback: extraire le mode sans carte (logique d'agrégation)
    base = candidate
    mode_without_map = base
    if " on " in base:
        mode_without_map = base.split(" on ", 1)[0].strip()

    # Chercher le mode sans carte dans les fallbacks génériques (FR uniquement, legacy)
    if lang == "fr" and mode_without_map in PAIR_FR:
        return PAIR_FR[mode_without_map]

    # 4) Fallback générique avec JSON i18n (modes_{lang}.json)
    modes_data = load_domain("modes", lang)
    prefixes = modes_data.get("_prefixes", {})
    separator = modes_data.get("_separator", ": ")

    if ":" in mode_without_map:
        prefix, mode_part = mode_without_map.split(":", 1)
        prefix = prefix.strip()
        mode_part = mode_part.strip()

        prefix_loc = prefixes.get(prefix, prefix)
        mode_loc = modes_data.get(mode_part, mode_part)

        return f"{prefix_loc}{separator}{mode_loc}"

    # Fallback dernière chance : mode seul
    mode_loc = modes_data.get(mode_without_map)
    if mode_loc and not isinstance(mode_loc, dict):
        return str(mode_loc)

    return s
