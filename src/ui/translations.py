"""Traductions UI — bilingue FR/EN.

Centralise les fonctions de traduction de libellés (playlist, mode/pair).

Depuis v6 : la source de vérité est ``metadata.duckdb`` :
- Playlists  : colonnes ``name_fr`` / ``name_en`` exposées via ``v_match_full``.
- Modes/pairs : 4 tables ``mode_prefix_names``, ``mode_name_tr``,
  ``mode_pair_overrides``, ``mode_lang_settings``.

Toutes les fonctions publiques acceptent un paramètre ``lang: str = "fr"``.
La valeur ``"fr"`` est le défaut pour assurer la rétrocompatibilité.
"""

from __future__ import annotations

import logging
from functools import lru_cache

logger = logging.getLogger(__name__)

# Labels de fallback pour UUIDs non résolus (metadata.duckdb incomplet)
_UNKNOWN_PLAYLIST: dict[str, str] = {"fr": "Inconnue", "en": "Unknown"}
_UNKNOWN_MODE: dict[str, str] = {"fr": "Mode inconnu", "en": "Unknown mode"}


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


def translate_pair_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un pair_name depuis metadata.duckdb.

    Résolution en 3 étapes :
    1. Override exact (``mode_pair_overrides``) pour les cas non combinatoires
    2. Combinatoire générique : préfixe localisé + séparateur + mode localisé
    3. Mode seul (sans catégorie)

    Args:
        name: Nom de pair brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.
    """
    if not name:
        return None
    s = str(name).strip()
    if not s:
        return None
    if _is_uuid_like(s):
        logger.warning("pair_name UUID non résolu : %s", s)
        return _UNKNOWN_MODE.get(lang, "Unknown mode")

    no_map = s.split(" on ", 1)[0].strip()
    candidate = _normalize_pair_case(no_map)

    # 1) Override exact
    if result := _mode_db_lookup("mode_pair_overrides", candidate, lang):
        return result

    # 2) Combinatoire préfixe + mode
    if ":" in candidate:
        prefix_en, mode_en = candidate.split(":", 1)
        sep = _mode_sep(lang)
        prefix_loc = (
            _mode_db_lookup("mode_prefix_names", prefix_en.strip(), lang) or prefix_en.strip()
        )
        mode_loc = _mode_db_lookup("mode_name_tr", mode_en.strip(), lang) or mode_en.strip()
        return f"{prefix_loc}{sep}{mode_loc}"

    # 3) Mode seul
    return _mode_db_lookup("mode_name_tr", candidate, lang) or candidate


@lru_cache(maxsize=8)
def _load_mode_tables(lang: str) -> dict[str, object]:
    """Charge les tables modes depuis metadata.duckdb (cache process-level)."""
    import duckdb

    from src.utils.paths import get_metadata_db_path

    _empty: dict[str, object] = {
        "mode_prefix_names": {},
        "mode_name_tr": {},
        "mode_pair_overrides": {},
        "separator": " : ",
    }
    db_path = get_metadata_db_path()
    if not db_path.exists():
        logger.warning("metadata.duckdb introuvable — translate_pair_name en mode dégradé")
        return _empty
    try:
        with duckdb.connect(str(db_path), read_only=True) as conn:
            return {
                "mode_prefix_names": dict(
                    conn.execute(
                        "SELECT prefix_en, name FROM mode_prefix_names WHERE lang=?", [lang]
                    ).fetchall()
                ),
                "mode_name_tr": dict(
                    conn.execute(
                        "SELECT mode_en, name FROM mode_name_tr WHERE lang=?", [lang]
                    ).fetchall()
                ),
                "mode_pair_overrides": dict(
                    conn.execute(
                        "SELECT pattern, name FROM mode_pair_overrides WHERE lang=?", [lang]
                    ).fetchall()
                ),
                "separator": (
                    conn.execute(
                        "SELECT separator FROM mode_lang_settings WHERE lang=?", [lang]
                    ).fetchone()
                    or (" : ",)
                )[0],
            }
    except Exception as exc:
        logger.warning("Erreur chargement mode tables (%s): %s", lang, exc)
        return _empty


def _mode_db_lookup(table: str, key: str, lang: str) -> str | None:
    """Recherche une clé dans les tables modes cachées."""
    return _load_mode_tables(lang).get(table, {}).get(key)  # type: ignore[union-attr]


def _mode_sep(lang: str) -> str:
    """Retourne le séparateur préfixe+mode selon la langue."""
    return _load_mode_tables(lang).get("separator", ": ")  # type: ignore[return-value]
