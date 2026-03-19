"""Accès centralisé aux définitions de médailles depuis metadata.duckdb.

Source unique : table ``medal_definitions`` dans ``metadata.duckdb``.
Aucun fallback JSON — les données doivent être en base.
"""

from __future__ import annotations

import logging

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

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Langue cible (``"fr"`` ou ``"en"``).

    Returns:
        Label lisible ou ``str(medal_name_id)`` si non résolu.
    """
    columns = ["name_fr", "name_en"] if lang == "fr" else ["name_en", "name_fr"]
    name = _resolve_text_from_db(medal_name_id, columns)
    return name if name else str(medal_name_id)


def resolve_medal_description(medal_name_id: int, lang: str = "fr") -> str | None:
    """Résout une description de médaille si disponible.

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Langue cible (``"fr"`` ou ``"en"``).

    Returns:
        Description ou None si non trouvée.
    """
    columns = (
        ["description_fr", "description_en"]
        if lang == "fr"
        else ["description_en", "description_fr"]
    )
    return _resolve_text_from_db(medal_name_id, columns)


def _resolve_text_from_db(medal_name_id: int, columns: list[str]) -> str | None:
    """Résolution d'un texte de médaille depuis metadata.duckdb.

    Essaie la première colonne, puis la seconde en fallback (cross-lang).
    """
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


__all__ = [
    "load_medal_name_maps",
    "resolve_medal_description",
    "resolve_medal_name",
]
