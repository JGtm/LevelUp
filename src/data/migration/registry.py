"""Registre des migrations de schéma DuckDB versionnées.

Chaque migration est une dataclass nommée par fonctionnalité.
Le runner les applique dans l'ordre de déclaration.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Literal

if TYPE_CHECKING:
    from collections.abc import Callable

    import duckdb


@dataclass(frozen=True)
class Migration:
    """Décrit une migration de schéma (DDL) et/ou un backfill de données."""

    name: str
    target_db: Literal["player", "shared", "shared_pve", "metadata"]
    description: str
    apply_schema: Callable[[duckdb.DuckDBPyConnection], None]
    apply_backfill: Callable[..., None] | None = field(default=None)
    requires_api: bool = field(default=False)


# Registre ordonné — les migrations sont appliquées dans cet ordre.
# NE PAS réorganiser les entrées existantes.
# Pour ajouter une migration : l'ajouter EN FIN de liste.
MIGRATIONS: list[Migration] = []


def register(migration: Migration) -> Migration:
    """Ajoute une migration au registre."""
    MIGRATIONS.append(migration)
    return migration
