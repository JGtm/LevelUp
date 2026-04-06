"""Traductions UI — multi-langue.

Centralise les fonctions de traduction de libellés (playlist, mode/pair, maps, assets).

Depuis v6 : la source de vérité est ``metadata.duckdb`` :
- Assets (maps, playlists, pairs, variants) : table ``asset_translations`` (pivot multi-langue).
- Modes/pairs : 4 tables ``mode_prefix_names``, ``mode_name_tr``,
  ``mode_pair_overrides``, ``mode_lang_settings``.

Toutes les fonctions publiques acceptent un paramètre ``lang: str = "fr"``.
La valeur ``"fr"`` est le défaut pour assurer la rétrocompatibilité.
"""

from __future__ import annotations

import logging
from functools import lru_cache

from src.utils.strings import is_uuid_like as _is_uuid_like

logger = logging.getLogger(__name__)

# Labels de fallback pour UUIDs non résolus (metadata.duckdb incomplet)
_UNKNOWN_PLAYLIST: dict[str, str] = {"fr": "Inconnue", "en": "Unknown"}
_UNKNOWN_MODE: dict[str, str] = {"fr": "Mode inconnu", "en": "Unknown mode"}


def translate_playlist_name(name: str | None, lang: str = "fr") -> str | None:
    """Traduit un nom de playlist.

    Résolution en 2 étapes :
    1. UUID brut → label "Inconnue"/"Unknown"
    2. Passthrough (les noms de playlists viennent de l'API ou asset_translations)

    Args:
        name: Nom de playlist brut (peut être ``None``).
        lang: ``"fr"`` (défaut) ou ``"en"``.

    Returns:
        Nom ou label "inconnue" pour les UUIDs bruts.
    """
    if name is None:
        return None
    s = str(name).strip()
    if not s:
        return None
    if _is_uuid_like(s):
        logger.warning("playlist_name non résolu (UUID brut) : %s — asset_translations vide ?", s)
        return _UNKNOWN_PLAYLIST.get(lang, s)
    return s


def translate_pair_name(
    name: str | None, lang: str = "fr", *, normalize: bool = True
) -> str | None:
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
    return resolve_display_mode(
        s, mode_category, lang, _load_mode_tables(lang), strip_redundant_prefix=normalize
    )


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


def resolve_map_display_names(
    map_id_to_fallback: dict[str, str],
    lang: str,
) -> dict[str, str]:
    """Résout les noms traduits de cartes pour un lot de map_id (requête unique).

    Consulte ``asset_translations`` (metadata.duckdb) en une seule requête SQL
    pour tous les IDs. Retourne ``{map_id: nom_traduit}``, avec fallback sur
    le map_name EN si aucune traduction n'est disponible.

    Args:
        map_id_to_fallback: Mapping ``{map_id: map_name_fallback}``.
        lang: Code de langue (``'fr'``, ``'en'``, …).

    Returns:
        Mapping ``{map_id: nom_localisé}``.
    """
    if not map_id_to_fallback:
        return {}

    from src.data.sync._asset_langs import to_bcp47
    from src.utils.db import duckdb_read_only
    from src.utils.paths import get_metadata_db_path

    result: dict[str, str] = dict(map_id_to_fallback)  # valeurs par défaut = EN
    bcp = to_bcp47(lang) if len(lang) <= 3 else lang
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return result

    try:
        ids = list(map_id_to_fallback.keys())
        placeholders = ", ".join("?" * len(ids))
        with duckdb_read_only(str(db_path)) as conn:
            tables = {
                r[0]
                for r in conn.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
                ).fetchall()
            }
            if "asset_translations" not in tables:
                return result
            for try_lang in (bcp, "en-US"):
                rows = conn.execute(
                    f"SELECT asset_id, name FROM asset_translations "
                    f"WHERE asset_id IN ({placeholders}) AND asset_type = 'map' AND lang = ?",
                    [*ids, try_lang],
                ).fetchall()
                for asset_id, name in rows:
                    if name and str(name).strip():
                        key = str(asset_id)
                        # Ne pas écraser une traduction cible par le fallback EN
                        if try_lang == "en-US" and result[key] != map_id_to_fallback[key]:
                            continue
                        result[key] = str(name).strip()
                if try_lang == bcp == "en-US":
                    break
    except Exception as exc:
        logger.debug(
            "resolve_map_display_names(%d ids, %s): %s", len(map_id_to_fallback), lang, exc
        )

    return result


def resolve_asset_name(
    asset_id: str | None,
    asset_type: str,
    lang: str = "fr",
    *,
    fallback: str | None = None,
) -> str | None:
    """Résout le nom localisé d'un asset depuis asset_translations.

    Ordre de priorité :
    1. asset_translations (asset_id + asset_type + BCP-47 de lang)
    2. asset_translations (asset_id + asset_type + 'en-US')
    3. fallback fourni (ex: map_name depuis match_registry)

    Args:
        asset_id: UUID de l'asset (map_id, playlist_id, etc.).
        asset_type: 'map' | 'playlist' | 'pair' | 'game_variant'.
        lang: Code court ('fr', 'en', 'de'…) ou BCP-47 ('fr-FR', 'en-US'…).
        fallback: Valeur à retourner si aucune traduction trouvée.

    Returns:
        Nom localisé, ou fallback, ou None.
    """
    if not asset_id:
        return fallback

    from src.data.sync._asset_langs import to_bcp47
    from src.utils.db import duckdb_read_only
    from src.utils.paths import get_metadata_db_path

    bcp = to_bcp47(lang) if len(lang) <= 3 else lang
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return fallback

    try:
        with duckdb_read_only(str(db_path)) as conn:
            # Vérifier que la table existe
            tables = {
                r[0]
                for r in conn.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
                ).fetchall()
            }
            if "asset_translations" not in tables:
                return fallback

            for try_lang in (bcp, "en-US"):
                row = conn.execute(
                    "SELECT name FROM asset_translations "
                    "WHERE asset_id = ? AND asset_type = ? AND lang = ?",
                    [asset_id, asset_type, try_lang],
                ).fetchone()
                if row and row[0]:
                    return str(row[0]).strip()
                if try_lang == bcp == "en-US":
                    break  # pas la peine de réessayer en-US deux fois
    except Exception as exc:
        logger.debug("resolve_asset_name(%s, %s, %s): %s", asset_id, asset_type, lang, exc)

    return fallback
