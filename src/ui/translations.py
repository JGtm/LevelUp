"""Traductions UI — bilingue FR/EN.

Centralise les fonctions de traduction de libellés (playlist, mode/pair).

Depuis v6 : la source de vérité pour les playlists est ``metadata.duckdb``
(colonnes ``name_fr`` / ``name_en`` exposées via ``v_match_full``).
``translate_playlist_name()`` est désormais un passthrough de sécurité,
utilisé uniquement pour les valeurs résiduelles hors-DB (UUIDs bruts).

Toutes les fonctions publiques acceptent un paramètre ``lang: str = "fr"``.
La valeur ``"fr"`` est le défaut pour assurer la rétrocompatibilité.
"""

from __future__ import annotations

import logging

from src.ui.i18n.data_labels import label, load_domain

logger = logging.getLogger(__name__)

# Labels de fallback pour UUIDs non résolus (metadata.duckdb incomplet)
_UNKNOWN_PLAYLIST: dict[str, str] = {"fr": "Inconnue", "en": "Unknown"}


def _is_uuid_like(s: str) -> bool:
    """Vérifie si une chaîne ressemble à un UUID (ex: a446725e-b281-414c-a21e)."""
    import re

    # UUID complet ou partiel (au moins 8 caractères hex avec tirets)
    return bool(re.match(r"^[a-f0-9]{8}(-[a-f0-9]{4}){0,3}(-[a-f0-9]{1,12})?$", s.lower()))


def translate_playlist_name(name: str | None, lang: str = "fr") -> str | None:
    """Passthrough de sécurité pour les noms de playlist hors-DB.

    Depuis v6 : les traductions sont dans ``metadata.duckdb`` (colonnes
    ``name_fr`` / ``name_en`` de ``v_match_full``). Cette fonction n'est
    appelée que pour les valeurs non résolues par la vue (ex : UUID brut).

    Args:
        name: Nom de playlist brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.

    Returns:
        ``name`` inchangé, ou label "inconnue" pour les UUIDs bruts.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None
    if _is_uuid_like(s):
        logger.warning("playlist_name non résolu (UUID brut) : %s — metadata.duckdb incomplet ?", s)
        return _UNKNOWN_PLAYLIST.get(lang, s)
    logger.debug("translate_playlist_name fallback pour '%s' (hors DB)", s)
    return s


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
