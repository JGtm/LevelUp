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
    """Traduit un nom de playlist via les fichiers JSON i18n (``static/i18n/playlists_*.json``).

    Résolution en 3 étapes :
    1. UUID brut → label "Inconnue"/"Unknown"
    2. Lookup dans ``static/i18n/playlists_{lang}.json`` via ``label()``
    3. Passthrough (retourne le nom tel quel si aucune traduction trouvée)

    Args:
        name: Nom de playlist brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.

    Returns:
        Nom traduit, ou label "inconnue" pour les UUIDs bruts.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None
    if _is_uuid_like(s):
        logger.warning("playlist_name non résolu (UUID brut) : %s — metadata.duckdb incomplet ?", s)
        return _UNKNOWN_PLAYLIST.get(lang, s)
    # Lookup i18n (static/i18n/playlists_{lang}.json)
    from src.ui.i18n.data_labels import label as _label

    translated = _label("playlists", s, lang=lang)
    if translated != s:
        return translated
    return s


def translate_pair_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un pair_name depuis metadata.duckdb.

    Délègue à ``resolve_display_mode`` (``src.analysis.mode_display``) qui gère :
    1. Override exact (``mode_pair_overrides``)
    2. Format inversé (CTF:Arena → préfixe Arena, mode CTF)
    3. Suppression du préfixe redondant selon mode_category inféré
    4. Combinatoire générique préfixe localisé + séparateur + mode localisé

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

    from src.analysis.mode_display import (
        infer_mode_category_from_pair_name,
        resolve_display_mode,
    )

    mode_category = infer_mode_category_from_pair_name(s)
    return resolve_display_mode(s, mode_category, lang, _load_mode_tables(lang))


@lru_cache(maxsize=8)
def _load_mode_tables(lang: str) -> dict[str, object]:
    """Charge les tables modes depuis metadata.duckdb (cache process-level)."""
    from src.utils.db import duckdb_read_only
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
        with duckdb_read_only(str(db_path)) as conn:
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
