"""Tests pour GamertagResolverMixin — vérifie suppression du fallback highlight_events."""

from __future__ import annotations

import duckdb

from src.data.migration.steps.drop_highlight_events_gamertag import _apply as drop_gamertag_column

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_conn_with_gamertag() -> duckdb.DuckDBPyConnection:
    """Connexion in-memory avec highlight_events AYANT encore la colonne gamertag."""
    conn = duckdb.connect(":memory:")
    conn.execute(
        "CREATE TABLE highlight_events ("
        "id INTEGER PRIMARY KEY, match_id VARCHAR, event_type VARCHAR, "
        "time_ms INTEGER, xuid VARCHAR, gamertag VARCHAR, type_hint INTEGER, raw_json VARCHAR)"
    )
    return conn


def _make_conn_without_gamertag() -> duckdb.DuckDBPyConnection:
    """Connexion in-memory avec highlight_events SANS la colonne gamertag (post-migration)."""
    conn = duckdb.connect(":memory:")
    conn.execute(
        "CREATE TABLE highlight_events ("
        "id INTEGER PRIMARY KEY, match_id VARCHAR, event_type VARCHAR, "
        "time_ms INTEGER, xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR)"
    )
    return conn


# ---------------------------------------------------------------------------
# Tests _resolve_from_highlight_events supprimée
# ---------------------------------------------------------------------------


class TestResolveFromHighlightEventsRemoved:
    """Vérifie que _resolve_from_highlight_events et _extract_ascii_token n'existent plus."""

    def test_resolve_from_highlight_events_absent(self) -> None:
        from src.data.repositories._gamertag_resolver import GamertagResolverMixin

        assert not hasattr(
            GamertagResolverMixin, "_resolve_from_highlight_events"
        ), "_resolve_from_highlight_events doit être supprimée (Commit 8)"

    def test_extract_ascii_token_absent(self) -> None:
        from src.data.repositories._gamertag_resolver import GamertagResolverMixin

        assert not hasattr(
            GamertagResolverMixin, "_extract_ascii_token"
        ), "_extract_ascii_token doit être supprimée (Commit 8)"


# ---------------------------------------------------------------------------
# Tests migration drop_highlight_events_gamertag
# ---------------------------------------------------------------------------


class TestDropHighlightEventsGamertag:
    """Vérifie la migration qui supprime la colonne gamertag."""

    def test_migration_drops_column(self) -> None:
        conn = _make_conn_with_gamertag()
        # Avant : la colonne existe
        col = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'highlight_events' AND column_name = 'gamertag'"
        ).fetchone()
        assert col is not None

        drop_gamertag_column(conn)

        # Après : la colonne est absente
        col_after = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'highlight_events' AND column_name = 'gamertag'"
        ).fetchone()
        assert col_after is None

    def test_migration_idempotent(self) -> None:
        """Appliquer la migration deux fois ne lève pas d'exception."""
        conn = _make_conn_with_gamertag()
        drop_gamertag_column(conn)
        drop_gamertag_column(conn)  # second appel : colonne déjà absente → skip silencieux


# _has_gamertag_column a été supprimée de WeaponKillsMixin lors du refactor v6
# (résolution gamertag déléguée à v_gamertag_lookup, pas de fallback colonne)
