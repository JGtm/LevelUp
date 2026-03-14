"""Insertions batch DuckDB — Sprint 15.

Remplace les boucles `for row in rows: INSERT ...` par des insertions
groupées via `executemany` ou insertion depuis DataFrame Polars/Arrow.

Avantages :
- 10-50x plus rapide sur gros volumes (batches de N rows)
- Typage centralisé via CAST_PLAN
- Zéro dépendance Parquet (DuckDB-first)

Usage :
    from src.data.sync.batch_insert import batch_insert_rows

    batch_insert_rows(conn, "medals_earned", medal_rows, MEDALS_COLUMNS)
"""

from __future__ import annotations

import logging
from dataclasses import fields
from datetime import datetime
from typing import Any

from src.data.sync._batch_audit import audit_all_tables, audit_column_types  # noqa: F401
from src.data.sync._batch_columns import (  # noqa: F401
    ALIAS_COLUMNS,
    CAST_PLAN,
    CRITICAL_TABLES,
    HIGHLIGHT_EVENT_COLUMNS,
    MEDAL_COLUMNS,
    PARTICIPANT_ALL_COLUMNS,
    PARTICIPANT_COLUMNS,
    PARTICIPANT_MMR_COLUMNS,
    PERSONAL_SCORE_COLUMNS,
    SHARED_MEDAL_COLUMNS,
    SKILL_COLUMNS,
)

logger = logging.getLogger(__name__)


# =============================================================================
# Fonctions de conversion de type Python
# =============================================================================


def _coerce_value(value: Any, duckdb_type: str) -> Any:  # noqa: C901, PLR0912
    """Convertit une valeur Python pour correspondre au type DuckDB attendu.

    Args:
        value: Valeur à convertir.
        duckdb_type: Type DuckDB cible (VARCHAR, FLOAT, INTEGER, etc.).

    Returns:
        Valeur convertie, ou None si conversion impossible.
    """
    if value is None:
        return None

    duckdb_type = duckdb_type.upper()

    try:
        if duckdb_type in ("VARCHAR", "TEXT"):
            s = str(value)
            return None if s in ("nan", "None", "") else s

        if duckdb_type in ("FLOAT", "DOUBLE", "REAL"):
            import math

            f = float(value)
            return None if (math.isnan(f) or math.isinf(f)) else f

        if duckdb_type in ("INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT"):
            import math

            f = float(value)
            if math.isnan(f) or math.isinf(f):
                return None
            return int(f)

        if duckdb_type == "BOOLEAN":
            if isinstance(value, bool):
                return value
            if isinstance(value, int | float):
                return bool(value)
            if isinstance(value, str):
                return value.lower() in ("true", "1", "yes")
            return bool(value)

        if duckdb_type == "TIMESTAMP":
            if isinstance(value, datetime):
                return value
            if isinstance(value, str):
                s = value.replace("Z", "+00:00")
                return datetime.fromisoformat(s)
            return value

    except (TypeError, ValueError, OverflowError):
        return None

    return value


def coerce_row_types(
    row_dict: dict[str, Any],
    table_name: str,
) -> dict[str, Any]:
    """Applique le plan de cast à un dictionnaire de row.

    Args:
        row_dict: Dictionnaire colonne→valeur.
        table_name: Nom de la table cible.

    Returns:
        Dictionnaire avec les types corrigés.
    """
    plan = CAST_PLAN.get(table_name, {})
    if not plan:
        return row_dict

    result = {}
    for col, val in row_dict.items():
        if col in plan:
            result[col] = _coerce_value(val, plan[col])
        else:
            result[col] = val

    return result


# =============================================================================
# Helpers internes (DRY)
# =============================================================================


def _row_to_dict(row: Any, columns: list[str]) -> dict[str, Any]:
    """Convertit un row (dataclass, dict ou objet) en dict."""
    if hasattr(row, "__dataclass_fields__"):
        return {f.name: getattr(row, f.name, None) for f in fields(row)}
    if isinstance(row, dict):
        return row
    return {col: getattr(row, col, None) for col in columns}


def _rows_to_values(
    rows: list[Any],
    columns: list[str],
    table_name: str,
    *,
    apply_cast: bool = True,
) -> list[tuple]:
    """Convertit une liste de rows en liste de tuples prêts pour executemany."""
    values: list[tuple] = []
    for row in rows:
        row_dict = _row_to_dict(row, columns)
        if apply_cast:
            row_dict = coerce_row_types(row_dict, table_name)
        values.append(tuple(row_dict.get(col) for col in columns))
    return values


def _executemany_with_fallback(
    conn: Any,
    sql: str,
    values_list: list[tuple],
    table_name: str,
) -> int:
    """Exécute un batch, fallback row-by-row si échec."""
    try:
        conn.executemany(sql, values_list)
        return len(values_list)
    except Exception as e:
        logger.debug("Batch échoué pour %s, fallback row-by-row: %s", table_name, e)
        inserted = 0
        for values in values_list:
            try:
                conn.execute(sql, values)
                inserted += 1
            except Exception as row_err:
                logger.warning("Insert échoué %s: %s", table_name, row_err)
        return inserted


# =============================================================================
# Insertion batch — Sprint 15.1 + 15.2
# =============================================================================


def batch_insert_rows(  # noqa: PLR0913
    conn: Any,
    table_name: str,
    rows: list[Any],
    columns: list[str],
    *,
    on_conflict: str = "",
    apply_cast: bool = True,
) -> int:
    """Insère des rows en batch via executemany.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        rows: Liste de dataclass ou dicts.
        columns: Liste des colonnes à insérer.
        on_conflict: Clause ON CONFLICT optionnelle (ex: "DO NOTHING").
        apply_cast: Si True, applique le CAST_PLAN aux valeurs.

    Returns:
        Nombre de rows insérées.
    """
    if not rows:
        return 0

    values_list = _rows_to_values(rows, columns, table_name, apply_cast=apply_cast)

    placeholders = ", ".join(["?"] * len(columns))
    col_list = ", ".join(columns)
    sql = f"INSERT INTO {table_name} ({col_list}) VALUES ({placeholders})"
    if on_conflict:
        sql += f" {on_conflict}"

    return _executemany_with_fallback(conn, sql, values_list, table_name)


def batch_upsert_rows(
    conn: Any,
    table_name: str,
    rows: list[Any],
    columns: list[str],
    *,
    apply_cast: bool = True,
) -> int:
    """Upsert (INSERT OR REPLACE) des rows en batch.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        rows: Liste de dataclass ou dicts.
        columns: Liste des colonnes à insérer.
        apply_cast: Si True, applique le CAST_PLAN aux valeurs.

    Returns:
        Nombre de rows upsertées.
    """
    if not rows:
        return 0

    values_list = _rows_to_values(rows, columns, table_name, apply_cast=apply_cast)

    placeholders = ", ".join(["?"] * len(columns))
    col_list = ", ".join(columns)
    sql = f"INSERT OR REPLACE INTO {table_name} ({col_list}) VALUES ({placeholders})"

    return _executemany_with_fallback(conn, sql, values_list, table_name)


def batch_upsert_participants(
    conn: Any,
    rows: list[Any],
    columns: list[str] | None = None,
) -> int:
    """Upsert participants en préservant les colonnes MMR/skill existantes.

    Utilise INSERT ... ON CONFLICT (match_id, xuid) DO UPDATE SET
    pour ne mettre à jour que les colonnes non-MMR. Les colonnes MMR/skill
    ne sont jamais écrasées par cette fonction.

    Args:
        conn: Connexion DuckDB.
        rows: Liste de MatchParticipantRow dataclass ou dicts.
        columns: Colonnes à insérer (défaut: PARTICIPANT_COLUMNS).

    Returns:
        Nombre de rows upsertées.
    """
    if not rows:
        return 0

    if columns is None:
        columns = PARTICIPANT_COLUMNS

    values_list = _rows_to_values(
        rows,
        columns,
        "match_participants",
        apply_cast=True,
    )

    pk_cols = {"match_id", "xuid"}
    update_cols = [c for c in columns if c not in pk_cols]
    update_set = ", ".join(f"{c} = EXCLUDED.{c}" for c in update_cols)

    placeholders = ", ".join(["?"] * len(columns))
    col_list = ", ".join(columns)
    sql = (
        f"INSERT INTO match_participants ({col_list}) VALUES ({placeholders}) "
        f"ON CONFLICT (match_id, xuid) DO UPDATE SET {update_set}"
    )

    return _executemany_with_fallback(conn, sql, values_list, "match_participants")


# =============================================================================
# Insertion batch PvE — shared_pve.duckdb (v5.2)
# =============================================================================


def batch_insert_pve_stats(
    conn: Any,
    rows: list,
) -> int:
    """Insère les stats PvE en batch dans shared_pve.duckdb.

    Args:
        conn: Connexion DuckDB vers shared_pve.duckdb.
        rows: Liste de PveMatchStatsRow à insérer.

    Returns:
        Nombre de lignes insérées/remplacées.
    """
    if not rows:
        return 0

    values = [
        (
            r.match_id,
            r.xuid,
            r.total_enemy_kills,
            r.boss_kills,
            r.grunt_kills,
            r.elite_kills,
            r.jackal_kills,
            r.brute_kills,
            r.hunter_kills,
            r.skimmer_kills,
            r.sentinel_kills,
            r.marine_kills,
        )
        for r in rows
    ]

    try:
        conn.executemany(
            """
            INSERT OR REPLACE INTO pve_match_stats (
                match_id, xuid,
                total_enemy_kills, boss_kills,
                grunt_kills, elite_kills, jackal_kills, brute_kills,
                hunter_kills, skimmer_kills, sentinel_kills, marine_kills
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
            values,
        )
        return len(rows)
    except Exception as e:
        logger.warning("Erreur batch_insert_pve_stats: %s", e)
        return 0
