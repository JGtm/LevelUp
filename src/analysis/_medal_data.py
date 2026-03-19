"""Résolution medal_name_id → nom / description FR/EN.

Source unique : table ``medal_definitions`` dans ``metadata.duckdb``.
Aucun fallback JSON — les données doivent être en base.
"""

from __future__ import annotations

import logging

from src.utils.db import duckdb_read_only
from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)


def _resolve_text_from_db(medal_name_id: int, lang: str, columns: list[str]) -> str | None:
    """Résolution d'un texte de médaille depuis metadata.duckdb."""
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return None

    with duckdb_read_only(db_path) as conn:
        try:
            row = conn.execute(
                f"SELECT {columns[0]} FROM medal_definitions WHERE medal_name_id = ?",
                [medal_name_id],
            ).fetchone()
            if row and row[0]:
                return str(row[0]).strip() or None
            # Tentative colonne alternative (autre langue)
            if len(columns) > 1:
                row = conn.execute(
                    f"SELECT {columns[1]} FROM medal_definitions WHERE medal_name_id = ?",
                    [medal_name_id],
                ).fetchone()
                if row and row[0]:
                    return str(row[0]).strip() or None
        except Exception:
            logger.debug("_resolve_text_from_db(%d, %s): erreur", medal_name_id, lang)
    return None


def resolve_medal_name(medal_name_id: int, lang: str = "fr") -> str:
    """Résout un medal_name_id en nom lisible.

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Langue cible (``"fr"`` ou ``"en"``).

    Returns:
        Label lisible ou ``str(medal_name_id)`` si non résolu.
    """
    columns = ["name_fr", "name_en"] if lang == "fr" else ["name_en", "name_fr"]
    name = _resolve_text_from_db(medal_name_id, lang, columns)
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
    return _resolve_text_from_db(medal_name_id, lang, columns)


__all__ = ["resolve_medal_description", "resolve_medal_name"]
