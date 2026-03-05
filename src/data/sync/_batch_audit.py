"""Audit de types DuckDB — Sprint 15.4.

Compare le schéma réel des tables avec le CAST_PLAN de référence
et identifie les incohérences.
"""

from __future__ import annotations

import logging
from typing import Any

from src.data.sync._batch_columns import CAST_PLAN

logger = logging.getLogger(__name__)


# =============================================================================
# Compatibilité de types
# =============================================================================

_TYPE_ALIASES: dict[str, set[str]] = {
    "TINYINT": {"TINYINT", "INT1"},
    "SMALLINT": {"SMALLINT", "INT2", "SHORT"},
    "INTEGER": {"INTEGER", "INT4", "INT", "SIGNED"},
    "BIGINT": {"BIGINT", "INT8", "LONG"},
    "FLOAT": {"FLOAT", "FLOAT4", "REAL"},
    "DOUBLE": {"DOUBLE", "FLOAT8"},
    "VARCHAR": {"VARCHAR", "TEXT", "STRING"},
    "BOOLEAN": {"BOOLEAN", "BOOL", "LOGICAL"},
    "TIMESTAMP": {"TIMESTAMP", "DATETIME", "TIMESTAMP WITH TIME ZONE"},
}

_INT_ORDER = ["TINYINT", "SMALLINT", "INTEGER", "BIGINT"]
_FLOAT_ORDER = ["FLOAT", "DOUBLE"]


def _types_compatible(expected: str, actual: str) -> bool:
    """Vérifie si deux types DuckDB sont compatibles."""
    eu = expected.upper()
    au = actual.upper()
    if eu == au:
        return True
    for group in _TYPE_ALIASES.values():
        if eu in group and au in group:
            return True
    # Les entiers plus larges sont acceptables
    if eu in _INT_ORDER and au in _INT_ORDER:
        return _INT_ORDER.index(au) >= _INT_ORDER.index(eu)
    if eu in _FLOAT_ORDER and au in _FLOAT_ORDER:
        return _FLOAT_ORDER.index(au) >= _FLOAT_ORDER.index(eu)
    return False


# =============================================================================
# Fonctions d'audit
# =============================================================================


def audit_column_types(
    conn: Any,
    table_name: str,
) -> list[dict[str, str]]:
    """Audite les incohérences de types entre le schéma réel et le CAST_PLAN.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.

    Returns:
        Liste de dicts {column, expected_type, actual_type, status}.
    """
    expected_types = CAST_PLAN.get(table_name, {})
    if not expected_types:
        return []

    try:
        actual_cols = conn.execute(
            """SELECT column_name, data_type
               FROM information_schema.columns
               WHERE table_name = ?
               ORDER BY ordinal_position""",
            (table_name,),
        ).fetchall()
    except Exception:
        return []

    actual_map = {row[0]: row[1] for row in actual_cols}
    issues: list[dict[str, str]] = []

    for col, expected_type in expected_types.items():
        actual_type = actual_map.get(col)
        if actual_type is None:
            issues.append(
                {
                    "column": col,
                    "expected_type": expected_type,
                    "actual_type": "MISSING",
                    "status": "MISSING_COLUMN",
                }
            )
        elif not _types_compatible(expected_type, actual_type):
            issues.append(
                {
                    "column": col,
                    "expected_type": expected_type,
                    "actual_type": actual_type,
                    "status": "TYPE_MISMATCH",
                }
            )

    # Colonnes inattendues (dans la DB mais pas dans le CAST_PLAN)
    for col, actual_type in actual_map.items():
        if col not in expected_types:
            issues.append(
                {
                    "column": col,
                    "expected_type": "N/A",
                    "actual_type": actual_type,
                    "status": "EXTRA_COLUMN",
                }
            )

    return issues


def audit_all_tables(conn: Any) -> dict[str, list[dict[str, str]]]:
    """Audite toutes les tables connues dans le CAST_PLAN.

    Args:
        conn: Connexion DuckDB.

    Returns:
        Dict table_name → liste d'issues.
    """
    results: dict[str, list[dict[str, str]]] = {}
    for table_name in CAST_PLAN:
        issues = audit_column_types(conn, table_name)
        if issues:
            results[table_name] = issues
    return results
