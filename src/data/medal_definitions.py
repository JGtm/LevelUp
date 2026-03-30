"""Accès centralisé aux définitions de médailles depuis metadata.duckdb.

Sources par priorité :
1. ``medal_translations`` (pivot multi-langue, BCP-47)
2. ``medal_definitions`` (legacy EN+FR, fallback)
"""

from __future__ import annotations

import logging

from src.data.sync._asset_langs import to_bcp47
from src.utils.db import duckdb_read_only
from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)


def load_medal_name_maps() -> tuple[dict[str, str], dict[str, str]]:
    """Charge les maps de labels de médailles depuis ``medal_definitions``.

    Returns:
        Tuple (fr_map, en_map) où chaque map est ``{str(medal_name_id): "Label"}``.
    """
    db_path = get_metadata_db_path()
    if not db_path.exists():
        logger.warning("metadata.duckdb introuvable : %s", db_path)
        return {}, {}

    with duckdb_read_only(db_path) as conn:
        try:
            rows = conn.execute(
                "SELECT medal_name_id, name_fr, name_en FROM medal_definitions"
            ).fetchall()
        except Exception:
            logger.debug("Erreur requête medal_definitions")
            return {}, {}

    fr_map = {str(r[0]): r[1] for r in rows if r[1]}
    en_map = {str(r[0]): r[2] for r in rows if r[2]}
    return fr_map, en_map


def resolve_medal_name(medal_name_id: int, lang: str = "fr") -> str:
    """Résout un medal_name_id en nom lisible.

    Cherche dans medal_translations (pivot multi-langue) en priorité,
    puis dans medal_definitions (legacy EN+FR) en fallback.

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Code court ('fr', 'en', 'de'…) ou BCP-47 ('fr-FR', 'en-US'…).

    Returns:
        Label lisible ou ``str(medal_name_id)`` si non résolu.
    """
    bcp = to_bcp47(lang) if len(lang) <= 3 else lang  # normalise 'fr' → 'fr-FR'
    name = _resolve_from_translations(medal_name_id, bcp)
    if not name and bcp != "en-US":
        name = _resolve_from_translations(medal_name_id, "en-US")
    if not name:
        name = _resolve_from_definitions(medal_name_id, lang)
    return name if name else str(medal_name_id)


def resolve_medal_description(medal_name_id: int, lang: str = "fr") -> str | None:
    """Résout une description de médaille si disponible.

    Cherche dans medal_translations (pivot multi-langue) en priorité.

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Code court ou BCP-47.

    Returns:
        Description ou None si non trouvée.
    """
    bcp = to_bcp47(lang) if len(lang) <= 3 else lang
    desc = _resolve_desc_from_translations(medal_name_id, bcp)
    if not desc and bcp != "en-US":
        desc = _resolve_desc_from_translations(medal_name_id, "en-US")
    if not desc:
        desc = _resolve_desc_from_definitions(medal_name_id, lang)
    return desc


# ─────────────────────────────────────────────────────────────────────────────
# Helpers internes
# ─────────────────────────────────────────────────────────────────────────────


def _resolve_from_translations(medal_name_id: int, bcp_lang: str) -> str | None:
    """Cherche le nom dans medal_translations (pivot multi-langue)."""
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return None
    with duckdb_read_only(db_path) as conn:
        try:
            row = conn.execute(
                "SELECT name FROM medal_translations WHERE medal_name_id = ? AND lang = ?",
                [medal_name_id, bcp_lang],
            ).fetchone()
            return str(row[0]).strip() if row and row[0] else None
        except Exception:
            return None


def _resolve_desc_from_translations(medal_name_id: int, bcp_lang: str) -> str | None:
    """Cherche la description dans medal_translations."""
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return None
    with duckdb_read_only(db_path) as conn:
        try:
            row = conn.execute(
                "SELECT description FROM medal_translations WHERE medal_name_id = ? AND lang = ?",
                [medal_name_id, bcp_lang],
            ).fetchone()
            return str(row[0]).strip() if row and row[0] else None
        except Exception:
            return None


def _resolve_from_definitions(medal_name_id: int, lang: str) -> str | None:
    """Fallback : cherche dans medal_definitions (legacy EN+FR)."""
    columns = ["name_fr", "name_en"] if lang in ("fr", "fr-FR") else ["name_en", "name_fr"]
    return _resolve_text_from_db(medal_name_id, columns)


def _resolve_desc_from_definitions(medal_name_id: int, lang: str) -> str | None:
    """Fallback description : cherche dans medal_definitions."""
    columns = (
        ["description_fr", "description_en"]
        if lang in ("fr", "fr-FR")
        else ["description_en", "description_fr"]
    )
    return _resolve_text_from_db(medal_name_id, columns)


def _resolve_text_from_db(medal_name_id: int, columns: list[str]) -> str | None:
    """Résolution d'un texte depuis medal_definitions (fallback générique)."""
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return None
    with duckdb_read_only(db_path) as conn:
        try:
            cols_sql = ", ".join(columns)
            row = conn.execute(
                f"SELECT {cols_sql} FROM medal_definitions WHERE medal_name_id = ?",
                [medal_name_id],
            ).fetchone()
            if not row:
                return None
            for val in row:
                text = str(val).strip() if val else ""
                if text:
                    return text
        except Exception:
            logger.debug("_resolve_text_from_db(%d): erreur", medal_name_id)
    return None


def load_medal_description_map(lang: str = "fr") -> dict[str, str]:
    """Charge une map {str(medal_name_id): description} en une seule requête.

    Priorité : medal_translations (BCP-47 demandé, puis en-US), puis
    medal_definitions (fallback legacy). Idéal pour les grilles de médailles
    qui ont besoin de toutes les descriptions d'un coup.

    Args:
        lang: Code court ('fr', 'en') ou BCP-47 ('fr-FR', 'en-US').

    Returns:
        Map ``{str(medal_name_id): description}``.
    """
    bcp = to_bcp47(lang) if len(lang) <= 3 else lang
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return {}

    result: dict[str, str] = {}
    with duckdb_read_only(db_path) as conn:
        # 1) medal_translations — langue demandée
        try:
            rows = conn.execute(
                "SELECT medal_name_id, description FROM medal_translations"
                " WHERE lang = ? AND description IS NOT NULL",
                [bcp],
            ).fetchall()
            result = {str(r[0]): str(r[1]).strip() for r in rows if r[1]}
        except Exception:
            pass

        # 2) Compléter avec en-US pour les entrées manquantes
        if bcp != "en-US":
            try:
                rows_en = conn.execute(
                    "SELECT medal_name_id, description FROM medal_translations"
                    " WHERE lang = 'en-US' AND description IS NOT NULL"
                ).fetchall()
                for row in rows_en:
                    key = str(row[0])
                    if key not in result and row[1]:
                        result[key] = str(row[1]).strip()
            except Exception:
                pass

        # 3) Fallback medal_definitions (legacy)
        if not result:
            col_primary = "description_fr" if lang in ("fr", "fr-FR") else "description_en"
            col_fallback = "description_en" if lang in ("fr", "fr-FR") else "description_fr"
            try:
                rows_def = conn.execute(
                    f"SELECT medal_name_id, {col_primary}, {col_fallback}"
                    " FROM medal_definitions"
                ).fetchall()
                for r in rows_def:
                    for val in r[1:]:
                        if val and str(val).strip():
                            result[str(r[0])] = str(val).strip()
                            break
            except Exception:
                pass

    return result


__all__ = [
    "load_medal_description_map",
    "load_medal_name_maps",
    "resolve_medal_description",
    "resolve_medal_name",
]
