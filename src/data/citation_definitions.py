"""Accès centralisé aux définitions de citations depuis metadata.duckdb.

Source unique : table ``citation_mappings`` dans ``metadata.duckdb``.
Aucun fallback JSON — les données doivent être en base.
"""

from __future__ import annotations

import logging
from typing import Any

from src.utils.db import duckdb_read_only
from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)

_COLUMNS = (
    "citation_name_norm",
    "citation_name_display",
    "mapping_type",
    "image_path",
    "category",
    "description",
    "tier_targets",
    "composite_children",
    "subcategory",
)

_SQL = (
    "SELECT " + ", ".join(_COLUMNS) + " "
    "FROM citation_mappings "
    "WHERE enabled IS NOT FALSE "
    "ORDER BY category, "
    "CASE WHEN mapping_type = 'composite' THEN 0 ELSE 1 END, "
    "citation_name_display"
)


def load_citation_definitions() -> list[dict[str, Any]]:
    """Charge toutes les citations activées depuis ``citation_mappings``.

    Returns:
        Liste de dicts avec les clés : citation_name_norm, citation_name_display,
        mapping_type, image_path, category, description, tier_targets,
        composite_children, subcategory.
    """
    db_path = get_metadata_db_path()
    if not db_path.exists():
        logger.warning("metadata.duckdb introuvable : %s", db_path)
        return []

    with duckdb_read_only(db_path) as conn:
        try:
            rows = conn.execute(_SQL).fetchall()
            return [dict(zip(_COLUMNS, row, strict=False)) for row in rows]
        except Exception:
            logger.exception("Erreur chargement citation_mappings")
            return []


def get_citation_display_name(norm: str) -> str:
    """Résout un nom normalisé de citation en nom d'affichage.

    Args:
        norm: Clé normalisée (ex: ``"defenseur_du_drapeau"``).

    Returns:
        Nom d'affichage ou la clé brute si non trouvée.
    """
    db_path = get_metadata_db_path()
    if not db_path.exists():
        return norm

    with duckdb_read_only(db_path) as conn:
        try:
            row = conn.execute(
                "SELECT citation_name_display FROM citation_mappings "
                "WHERE citation_name_norm = ?",
                [norm],
            ).fetchone()
            return row[0] if row and row[0] else norm
        except Exception:
            return norm


__all__ = ["get_citation_display_name", "load_citation_definitions"]
