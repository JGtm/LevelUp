"""Tests unitaires pour lookup_xuid_for_gamertag et GamertagResolverMixin.resolve_xuid_from_gamertag.

Couverture :
- lookup_xuid_for_gamertag : vue disponible, fallback xuid_aliases, vue absente,
  insensibilité casse, introuvable, view_prefix "shared."
- resolve_xuid_from_gamertag : via mixin (délègue au helper avec view_prefix="shared.")
"""

from __future__ import annotations

import duckdb
import pytest

from src.utils.xuid import lookup_xuid_for_gamertag

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def conn_with_view() -> duckdb.DuckDBPyConnection:
    """Connexion in-memory avec v_gamertag_lookup ET xuid_aliases."""
    conn = duckdb.connect(":memory:")
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
    conn.execute("INSERT INTO xuid_aliases VALUES ('111', 'Alpha'), ('222', 'Beta')")
    conn.execute("CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM xuid_aliases")
    return conn


@pytest.fixture()
def conn_only_aliases() -> duckdb.DuckDBPyConnection:
    """Connexion in-memory avec xuid_aliases seulement (vue absente)."""
    conn = duckdb.connect(":memory:")
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
    conn.execute("INSERT INTO xuid_aliases VALUES ('333', 'Gamma'), ('444', 'Delta')")
    return conn


@pytest.fixture()
def conn_empty() -> duckdb.DuckDBPyConnection:
    """Connexion in-memory sans aucune table."""
    return duckdb.connect(":memory:")


# ---------------------------------------------------------------------------
# Tests lookup_xuid_for_gamertag
# ---------------------------------------------------------------------------


class TestLookupXuidForGamertag:
    def test_trouve_via_vue(self, conn_with_view: duckdb.DuckDBPyConnection) -> None:
        """Résolution via v_gamertag_lookup quand la vue existe."""
        assert lookup_xuid_for_gamertag(conn_with_view, "Alpha") == "111"

    def test_trouve_via_vue_casse_differente(
        self, conn_with_view: duckdb.DuckDBPyConnection
    ) -> None:
        """Insensible à la casse (LOWER)."""
        assert lookup_xuid_for_gamertag(conn_with_view, "aLpHa") == "111"
        assert lookup_xuid_for_gamertag(conn_with_view, "BETA") == "222"

    def test_fallback_xuid_aliases_sans_vue(
        self, conn_only_aliases: duckdb.DuckDBPyConnection
    ) -> None:
        """Fallback sur xuid_aliases quand v_gamertag_lookup est absente."""
        assert lookup_xuid_for_gamertag(conn_only_aliases, "Gamma") == "333"

    def test_fallback_casse_differente(self, conn_only_aliases: duckdb.DuckDBPyConnection) -> None:
        """Fallback insensible à la casse."""
        assert lookup_xuid_for_gamertag(conn_only_aliases, "delta") == "444"

    def test_introuvable_retourne_none(self, conn_with_view: duckdb.DuckDBPyConnection) -> None:
        """Retourne None si gamertag inconnu."""
        assert lookup_xuid_for_gamertag(conn_with_view, "Inconnu") is None

    def test_aucune_table_retourne_none(self, conn_empty: duckdb.DuckDBPyConnection) -> None:
        """Retourne None silencieusement si ni vue ni table n'existent."""
        assert lookup_xuid_for_gamertag(conn_empty, "Quelconque") is None

    def test_view_prefix_shared(self) -> None:
        """view_prefix='shared.' utilise le schéma attaché."""
        conn = duckdb.connect(":memory:")
        conn.execute("CREATE SCHEMA shared")
        conn.execute("CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        conn.execute("INSERT INTO shared.xuid_aliases VALUES ('999', 'Zeta')")
        conn.execute(
            "CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases"
        )
        assert lookup_xuid_for_gamertag(conn, "Zeta", view_prefix="shared.") == "999"

    def test_view_prefix_shared_fallback(self) -> None:
        """Fallback xuid_aliases avec view_prefix quand la vue est absente."""
        conn = duckdb.connect(":memory:")
        conn.execute("CREATE SCHEMA shared")
        conn.execute("CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        conn.execute("INSERT INTO shared.xuid_aliases VALUES ('888', 'Eta')")
        assert lookup_xuid_for_gamertag(conn, "Eta", view_prefix="shared.") == "888"

    def test_gamertag_vide_retourne_none(self, conn_with_view: duckdb.DuckDBPyConnection) -> None:
        """Gamertag vide retourne None."""
        assert lookup_xuid_for_gamertag(conn_with_view, "") is None


# ---------------------------------------------------------------------------
# Tests GamertagResolverMixin.resolve_xuid_from_gamertag
# ---------------------------------------------------------------------------


def _make_mixin_stub(conn: duckdb.DuckDBPyConnection) -> object:
    """Crée un stub minimal du mixin qui retourne *conn* depuis _get_connection().

    Évite de créer de vrais fichiers DuckDB (problème de verrou Windows sur TCP).
    """
    from src.data.repositories._gamertag_resolver import GamertagResolverMixin

    class _Stub(GamertagResolverMixin):
        def _get_connection(self) -> duckdb.DuckDBPyConnection:
            return conn

    return _Stub()


class TestResolveXuidFromGamertagMixin:
    """Vérifie que le mixin délègue correctement au helper avec view_prefix='shared.'."""

    def _make_conn_shared(self) -> duckdb.DuckDBPyConnection:
        """Connexion in-memory avec schéma 'shared' contenant v_gamertag_lookup."""
        conn = duckdb.connect(":memory:")
        conn.execute("CREATE SCHEMA shared")
        conn.execute("CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        conn.execute("INSERT INTO shared.xuid_aliases VALUES ('xuid_abc', 'MyPlayer')")
        conn.execute(
            "CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases"
        )
        return conn

    def test_resolve_xuid_from_gamertag_trouve(self) -> None:
        """resolve_xuid_from_gamertag retourne le XUID via v_gamertag_lookup."""
        conn = self._make_conn_shared()
        stub = _make_mixin_stub(conn)
        assert stub.resolve_xuid_from_gamertag("MyPlayer") == "xuid_abc"

    def test_resolve_xuid_from_gamertag_insensible_casse(self) -> None:
        """Le mixin est insensible à la casse."""
        conn = self._make_conn_shared()
        stub = _make_mixin_stub(conn)
        assert stub.resolve_xuid_from_gamertag("myplayer") == "xuid_abc"
        assert stub.resolve_xuid_from_gamertag("MYPLAYER") == "xuid_abc"

    def test_resolve_xuid_from_gamertag_introuvable(self) -> None:
        """resolve_xuid_from_gamertag retourne None pour un gamertag inconnu."""
        conn = self._make_conn_shared()
        stub = _make_mixin_stub(conn)
        assert stub.resolve_xuid_from_gamertag("Inconnu") is None

    def test_resolve_xuid_from_gamertag_fallback_aliases(self) -> None:
        """Fallback sur shared.xuid_aliases quand shared.v_gamertag_lookup est absente."""
        conn = duckdb.connect(":memory:")
        conn.execute("CREATE SCHEMA shared")
        conn.execute("CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        conn.execute("INSERT INTO shared.xuid_aliases VALUES ('xuid_fb', 'FallbackPlayer')")
        # Pas de vue v_gamertag_lookup → fallback
        stub = _make_mixin_stub(conn)
        assert stub.resolve_xuid_from_gamertag("FallbackPlayer") == "xuid_fb"
