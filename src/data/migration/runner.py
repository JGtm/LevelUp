"""Runner de migrations DuckDB — applique les migrations manquantes.

Exécuté dans launcher.py avant le lancement de Streamlit.
Chaque DB contient une table ``schema_migrations`` qui trace les
migrations déjà appliquées (idempotent).
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import TYPE_CHECKING

from src.data.migration.registry import MIGRATIONS, Migration
from src.utils.db import duckdb_read_write

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)

_SCHEMA_MIGRATIONS_DDL = """
CREATE TABLE IF NOT EXISTS schema_migrations (
    name         VARCHAR PRIMARY KEY,
    description  VARCHAR,
    applied_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    schema_done  BOOLEAN DEFAULT FALSE,
    backfill_done BOOLEAN DEFAULT FALSE
)
"""


@dataclass
class MigrationReport:
    """Rapport d'exécution des migrations."""

    schemas_applied: int = 0
    backfills_applied: int = 0
    errors: list[str] | None = None

    @property
    def had_work(self) -> bool:
        return self.schemas_applied > 0 or self.backfills_applied > 0


def _ensure_migration_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table schema_migrations si inexistante."""
    conn.execute(_SCHEMA_MIGRATIONS_DDL)


def _get_applied(conn: duckdb.DuckDBPyConnection) -> dict[str, tuple[bool, bool]]:
    """Retourne les migrations déjà appliquées {name: (schema_done, backfill_done)}."""
    rows = conn.execute("SELECT name, schema_done, backfill_done FROM schema_migrations").fetchall()
    return {r[0]: (r[1], r[2]) for r in rows}


def _apply_one_schema(conn: duckdb.DuckDBPyConnection, migration: Migration) -> None:
    """Applique le DDL d'une migration et l'enregistre."""
    migration.apply_schema(conn)
    now = datetime.now(tz=timezone.utc).isoformat()
    has_backfill = migration.apply_backfill is not None
    conn.execute(
        "INSERT INTO schema_migrations (name, description, applied_at, schema_done, backfill_done) "
        "VALUES (?, ?, ?, TRUE, ?)",
        [migration.name, migration.description, now, not has_backfill],
    )


def _apply_one_backfill(
    conn: duckdb.DuckDBPyConnection,
    migration: Migration,
    **kwargs: object,
) -> None:
    """Applique le backfill d'une migration et met à jour le flag."""
    if migration.apply_backfill is None:
        return
    migration.apply_backfill(conn, **kwargs)
    conn.execute(
        "UPDATE schema_migrations SET backfill_done = TRUE WHERE name = ?",
        [migration.name],
    )


def _run_for_db(
    db_path: Path,
    target_db: str,
    *,
    backfill_kwargs: dict[str, object] | None = None,
    on_progress: object | None = None,
    metadata_db_path: Path | None = None,
) -> MigrationReport:
    """Exécute les migrations pendantes sur une DB donnée."""
    report = MigrationReport(errors=[])
    relevant = [m for m in MIGRATIONS if m.target_db == target_db]
    if not relevant:
        return report

    if not db_path.exists():
        return report

    with duckdb_read_write(db_path) as conn:
        # Pour les migrations shared, attacher metadata.duckdb afin que
        # v_match_full (qui JOINt meta.asset_translations) soit accessible.
        if target_db == "shared" and metadata_db_path and metadata_db_path.exists():
            import contextlib

            with contextlib.suppress(Exception):
                conn.execute(f"ATTACH '{metadata_db_path}' AS meta (READ_ONLY)")
        _ensure_migration_table(conn)
        applied = _get_applied(conn)

        for mig in relevant:
            state = applied.get(mig.name)

            # Schéma pas encore appliqué
            if state is None:
                try:
                    _apply_one_schema(conn, mig)
                    report.schemas_applied += 1
                    logger.info("  ✓ [%s] %s", mig.name, mig.description)
                except Exception as e:
                    msg = f"Migration {mig.name} échouée: {e}"
                    logger.warning(msg)
                    report.errors.append(msg)
                    continue
                # Le schéma vient d'être appliqué, vérifier backfill
                state = (True, mig.apply_backfill is None)

            schema_done, backfill_done = state

            # Backfill restant
            if schema_done and not backfill_done and mig.apply_backfill is not None:
                if mig.requires_api and not (backfill_kwargs or {}).get("api"):
                    logger.info("  ⏭ [%s] backfill ignoré (API non disponible)", mig.name)
                    continue
                try:
                    _apply_one_backfill(conn, mig, **(backfill_kwargs or {}))
                    report.backfills_applied += 1
                    logger.info("  ✓ [%s] backfill terminé", mig.name)
                except Exception as e:
                    msg = f"Backfill {mig.name} échoué: {e}"
                    logger.warning(msg)
                    report.errors.append(msg)

    return report


def apply_pending_migrations(
    *,
    player_db_path: Path | None = None,
    shared_db_path: Path | None = None,
    pve_db_path: Path | None = None,
    metadata_db_path: Path | None = None,
    backfill_kwargs: dict[str, object] | None = None,
) -> MigrationReport:
    """Applique toutes les migrations pendantes sur les 3 types de DB.

    Args:
        player_db_path: Chemin vers stats.duckdb d'un joueur.
        shared_db_path: Chemin vers shared_matches_v2.duckdb.
        pve_db_path: Chemin vers shared_pve.duckdb.
        metadata_db_path: Chemin vers metadata.duckdb.
        backfill_kwargs: kwargs supplémentaires pour les fonctions de backfill.

    Returns:
        Rapport cumulé de toutes les migrations.
    """
    # Import les steps pour qu'ils s'enregistrent dans MIGRATIONS
    _load_migration_steps()

    total = MigrationReport(errors=[])
    db_map: list[tuple[Path | None, str]] = [
        (metadata_db_path, "metadata"),  # 1er — prérequis i18n pour les vues shared
        (shared_db_path, "shared"),
        (pve_db_path, "shared_pve"),
        (player_db_path, "player"),
    ]

    for path, target in db_map:
        if path is None:
            continue
        sub = _run_for_db(
            path,
            target,
            backfill_kwargs=backfill_kwargs,
            metadata_db_path=metadata_db_path,
        )
        total.schemas_applied += sub.schemas_applied
        total.backfills_applied += sub.backfills_applied
        if sub.errors:
            total.errors.extend(sub.errors)

    return total


def _load_migration_steps() -> None:
    """Charge les modules de steps pour qu'ils appellent register()."""
    if _load_migration_steps._loaded:
        return
    _load_migration_steps._loaded = True

    from src.data.migration import steps  # noqa: F401


_load_migration_steps._loaded = False  # type: ignore[attr-defined]
