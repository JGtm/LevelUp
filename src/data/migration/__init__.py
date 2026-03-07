"""Module de migration de schéma DuckDB versionnée.

Fournit un registre de migrations idempotentes (DDL + backfill optionnel)
et un runner qui applique automatiquement les migrations pendantes.

Usage typique::

    from src.data.migration.runner import apply_pending_migrations

    report = apply_pending_migrations(
        player_db_path=player_path,
        shared_db_path=shared_path,
        pve_db_path=pve_path,
    )
"""

from src.data.migration.registry import MIGRATIONS, Migration
from src.data.migration.runner import MigrationReport, apply_pending_migrations

__all__ = ["MIGRATIONS", "Migration", "MigrationReport", "apply_pending_migrations"]
