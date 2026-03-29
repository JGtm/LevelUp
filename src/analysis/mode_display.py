"""Résolution des labels de modes de jeu sans redondance contextuelle.

Supprime les préfixes redondants de ``game_variant_name`` lorsqu'ils sont
déjà implicites par ``mode_category`` (contexte playlist).

Exemples :
    BTB:Slayer        (BTB)       → "Assassin"
    BTB Heavies:CTF   (BTB)       → "Heavies : Capture du drapeau"
    Arena:CTF         (Assassin)  → "Capture du drapeau"
    CTF:Arena         (Assassin)  → "Capture du drapeau"  (format inversé)
    Tactical:Slayer   (Assassin)  → "Assassin Tactique"   (qualifier après mode)
    Ranked:Slayer     (Ranked)    → "Assassin"
    CASTLE WARS       (Fiesta)    → "CASTLE WARS"         (sans séparateur)

Ce module est **pur** : zéro accès DB, zéro Streamlit.
Les tables de traduction (issues de ``metadata.duckdb``) sont injectées
via le paramètre ``tables`` pour rester testable sans infrastructure.
"""

from __future__ import annotations

import logging
import re
from typing import Final, TypedDict

logger = logging.getLogger(__name__)

_MAP_SUFFIX_RE: Final = re.compile(r"^(.*?)(?:\s*[\-–—]\s*[0-9A-Za-z]{8,})$", re.IGNORECASE)

# ── Règles par préfixe anglais ────────────────────────────────────────────────
# category  : mode_category DB correspondante (valeur dans match_registry)
# qualifier : libellé à conserver quand le préfixe est redondant
#             None  → masquer le préfixe entièrement
#             str   → clé à résoudre dans mode_prefix_names (ex: "Tactical")
#                     ou valeur directe si absente de la table (ex: "Heavies")


class _PrefixRule(TypedDict):
    category: str
    qualifier: str | None


_PREFIX_RULES: Final[dict[str, _PrefixRule]] = {
    "Arena": {"category": "Assassin", "qualifier": None},
    "Tactical": {"category": "Assassin", "qualifier": "Tactical"},
    "Assault": {"category": "Assassin", "qualifier": None},
    "Community": {"category": "Other", "qualifier": None},
    "Event": {"category": "Other", "qualifier": None},
    "Fiesta": {"category": "Fiesta", "qualifier": None},
    "Super Fiesta": {"category": "Fiesta", "qualifier": None},
    "Husky Raid": {"category": "Fiesta", "qualifier": None},
    "Super Husky Raid": {"category": "Fiesta", "qualifier": None},
    "BTB": {"category": "BTB", "qualifier": None},
    "BTB Heavies": {"category": "BTB", "qualifier": "Heavies"},
    "Ranked": {"category": "Ranked", "qualifier": None},
    "Firefight": {"category": "Firefight", "qualifier": None},
    "Gruntpocalypse": {"category": "Firefight", "qualifier": None},
}

_KNOWN_PREFIXES: Final[frozenset[str]] = frozenset(_PREFIX_RULES)

# Préfixes dont la sémantique est appliquée même quand mode_category == "Other"
# (playlists non résolues / événementielles qui gardent la sémantique de leur préfixe)
_STRIP_IN_OTHER: Final[frozenset[str]] = frozenset(_PREFIX_RULES) - {"Community", "Event"}

_CASE_MAP: Final[dict[str, str]] = {
    "btb heavies": "BTB Heavies",
    "btb": "BTB",
    "super fiesta": "Super Fiesta",
    "super husky raid": "Super Husky Raid",
    "husky raid": "Husky Raid",
}


def _normalize_case(s: str) -> str:
    """Normalise la casse d'un pair_name (même logique que translations.py).

    Règles :
    - Préfixes spéciaux → CASE_MAP (BTB, BTB Heavies, etc.)
    - Tout-majuscules → conservé tel quel (acronymes : CTF, KOTH, BTB)
    - Sinon → chaque mot est capitalisé (title-case) : "team slayer" → "Team Slayer"
    """
    if ":" not in s:
        return s
    prefix, rest = s.split(":", 1)
    prefix = prefix.strip()
    rest = rest.strip()
    if prefix.lower() in _CASE_MAP:
        prefix = _CASE_MAP[prefix.lower()]
    elif not prefix.isupper():
        prefix = " ".join(w[:1].upper() + w[1:].lower() for w in prefix.split())
    if rest and rest == rest.lower():
        rest = " ".join(w[:1].upper() + w[1:] for w in rest.split())
    return f"{prefix}:{rest}" if prefix else rest


def _strip_map_suffix(s: str) -> str:
    """Retire le suffixe carte (' on MapName') et les suffixes ID techniques."""
    s = s.split(" on ", 1)[0].strip()
    m = _MAP_SUFFIX_RE.match(s)
    return (m.group(1) or "").strip() if m else s


def _is_redundant(rule: _PrefixRule, mode_category: str, prefix_en: str) -> bool:
    """Indique si le préfixe est redondant dans le contexte ``mode_category``."""
    if rule["category"] == mode_category:
        return True
    return mode_category == "Other" and prefix_en in _STRIP_IN_OTHER


def resolve_display_mode(
    game_variant_name: str | None,
    mode_category: str,
    lang: str,
    tables: dict,
) -> str:
    """Résout le label d'affichage d'un mode sans redondance contextuelle.

    Args:
        game_variant_name: Valeur brute depuis ``match_registry.game_variant_name``.
        mode_category:     Catégorie du match (BTB, Assassin, Fiesta, Ranked,
                           Firefight, Other).
        lang:              ``"fr"`` ou ``"en"``.
        tables:            Dict issu de ``_load_mode_tables(lang)`` dans
                           ``src.ui.translations`` — clés attendues :
                           ``mode_pair_overrides``, ``mode_name_tr``,
                           ``mode_prefix_names``, ``separator``.

    Returns:
        Label d'affichage nettoyé (jamais vide — retourne le brut en dernier recours).
    """
    if not game_variant_name:
        return "Mode inconnu" if lang == "fr" else "Unknown mode"

    raw = _strip_map_suffix(str(game_variant_name).strip())
    candidate = _normalize_case(raw or str(game_variant_name))

    overrides: dict[str, str] = tables.get("mode_pair_overrides", {})
    mode_tr: dict[str, str] = tables.get("mode_name_tr", {})
    prefix_tr: dict[str, str] = tables.get("mode_prefix_names", {})
    sep: str = tables.get("separator", ": ")

    # 1) Override exact
    if candidate in overrides:
        return overrides[candidate]

    # 2) Pas de séparateur : CASTLE WARS, TFF | Survive…
    if ":" not in candidate:
        return mode_tr.get(candidate, candidate)

    left, right = candidate.split(":", 1)
    left, right = left.strip(), right.strip()

    # 3) Détection format inversé (CTF:Arena → prefix=Arena, mode=CTF)
    left_is_prefix = left in _KNOWN_PREFIXES
    right_is_prefix = right in _KNOWN_PREFIXES
    if right_is_prefix and not left_is_prefix:
        prefix_en, mode_en = right, left
    else:
        prefix_en, mode_en = left, right

    rule = _PREFIX_RULES.get(prefix_en)
    mode_label = mode_tr.get(mode_en, mode_en)
    if mode_en not in mode_tr:
        logger.debug(
            "mode_en absent de mode_name_tr : %r (pair_name=%r)", mode_en, game_variant_name
        )

    # 4) Préfixe redondant → simplifier
    if rule and _is_redundant(rule, mode_category, prefix_en):
        qualifier = rule["qualifier"]
        if qualifier is None:
            return mode_label
        qual_label = prefix_tr.get(qualifier, qualifier)
        return f"{qual_label}{sep}{mode_label}"

    # 5) Préfixe non redondant → label complet
    prefix_label = prefix_tr.get(prefix_en, prefix_en)
    return f"{prefix_label}{sep}{mode_label}"


def infer_mode_category_from_pair_name(pair_name: str) -> str:
    """Infère la mode_category depuis le pair_name sans accès DB.

    Extrait le préfixe (gauche du ":"), résout le format inversé si besoin,
    puis renvoie la catégorie canonique depuis ``_PREFIX_RULES``.

    Returns:
        Catégorie DB correspondante (BTB, Assassin, Fiesta, Ranked, Firefight)
        ou ``"Other"`` si le préfixe est inconnu.
    """
    if not pair_name or ":" not in pair_name:
        return "Other"
    raw = _strip_map_suffix(str(pair_name).strip())
    candidate = _normalize_case(raw or str(pair_name))
    if ":" not in candidate:
        return "Other"
    left, right = candidate.split(":", 1)
    left, right = left.strip(), right.strip()
    # Détection format inversé
    left_is_prefix = left in _KNOWN_PREFIXES
    right_is_prefix = right in _KNOWN_PREFIXES
    prefix_en = right if right_is_prefix and not left_is_prefix else left
    rule = _PREFIX_RULES.get(prefix_en)
    return rule["category"] if rule else "Other"


__all__ = ["resolve_display_mode", "infer_mode_category_from_pair_name"]
