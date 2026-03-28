"""Résolution des labels de modes de jeu sans redondance contextuelle.

Supprime les préfixes redondants de ``game_variant_name`` lorsqu'ils sont
déjà implicites par ``mode_category`` (contexte playlist).

Exemples :
    BTB:Slayer       (BTB)       → "Assassin"
    BTB Heavies:CTF  (BTB)       → "Heavies : Capture du drapeau"
    Arena:CTF        (Assassin)  → "Capture du drapeau"
    CTF:Arena        (Assassin)  → "Capture du drapeau"  (format inversé)
    Ranked:Slayer    (Ranked)    → "Assassin"
    CASTLE WARS      (Fiesta)    → "CASTLE WARS"          (sans séparateur)

Ce module est **pur** : zéro accès DB, zéro Streamlit.
Les tables de traduction (issues de ``metadata.duckdb``) sont injectées
via le paramètre ``tables`` pour rester testable sans infrastructure.
"""

from __future__ import annotations

import re
from typing import Final, TypedDict

_MAP_SUFFIX_RE: Final = re.compile(r"^(.*?)(?:\s*[\-–—]\s*[0-9A-Za-z]{8,})$", re.IGNORECASE)

# ── Règles par préfixe anglais ────────────────────────────────────────────────
# category  : mode_category DB correspondante (valeur dans match_registry)
# qualifier : libellé à conserver quand le préfixe est redondant
#             None  → masquer le préfixe entièrement
#             str   → afficher ce qualificatif comme nouveau préfixe


class _PrefixRule(TypedDict):
    category: str
    qualifier: str | None


_PREFIX_RULES: Final[dict[str, _PrefixRule]] = {
    "Arena": {"category": "Assassin", "qualifier": None},
    "Tactical": {"category": "Assassin", "qualifier": None},
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

_CASE_MAP: Final[dict[str, str]] = {
    "btb heavies": "BTB Heavies",
    "btb": "BTB",
    "super fiesta": "Super Fiesta",
    "super husky raid": "Super Husky Raid",
    "husky raid": "Husky Raid",
}


def _normalize_case(s: str) -> str:
    """Normalise la casse d'un pair_name (même logique que translations.py)."""
    if ":" not in s:
        return s
    prefix, rest = s.split(":", 1)
    prefix = prefix.strip()
    rest = rest.strip()
    prefix = _CASE_MAP.get(
        prefix.lower(), prefix if prefix.isupper() else prefix[:1].upper() + prefix[1:].lower()
    )
    if rest and rest == rest.lower():
        rest = " ".join(w[:1].upper() + w[1:] for w in rest.split())
    return f"{prefix}:{rest}" if prefix else rest


def _strip_map_suffix(s: str) -> str:
    """Retire le suffixe carte (' on MapName') et les suffixes ID techniques."""
    s = s.split(" on ", 1)[0].strip()
    m = _MAP_SUFFIX_RE.match(s)
    return (m.group(1) or "").strip() if m else s


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

    # 4) Préfixe redondant → simplifier
    if rule and rule["category"] == mode_category:
        qualifier = rule["qualifier"]
        if qualifier is None:
            return mode_label
        qual_label = prefix_tr.get(qualifier, qualifier)
        return f"{qual_label}{sep}{mode_label}"

    # 5) Préfixe non redondant → label complet
    prefix_label = prefix_tr.get(prefix_en, prefix_en)
    return f"{prefix_label}{sep}{mode_label}"


__all__ = ["resolve_display_mode"]
