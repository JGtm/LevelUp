"""Migration : référentiel des défis Halo dans metadata.duckdb."""

from src.data.migration.registry import Migration, register
from src.data.sync.challenge_migrations import (
    ensure_challenge_definitions_table,
    ensure_challenge_translations_table,
)


def _apply(conn):  # type: ignore[no-untyped-def]
    ensure_challenge_definitions_table(conn)
    ensure_challenge_translations_table(conn)


register(
    Migration(
        name="add_challenge_metadata",
        target_db="metadata",
        description="Tables challenge_definitions + challenge_translations dans metadata.duckdb",
        apply_schema=_apply,
    )
)
