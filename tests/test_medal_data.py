"""Tests pour src/data/medal_definitions.py — couche centralisée DB-only.

Teste les fonctions depuis ``src.data.medal_definitions`` (source canonique)
et vérifie que ``src.analysis._medal_data`` ré-exporte correctement.
"""

from __future__ import annotations

from unittest.mock import patch

import duckdb
import pytest

_PATCH_TARGET = "src.data.medal_definitions.get_metadata_db_path"


@pytest.fixture()
def tmp_metadata_db(tmp_path):
    """Crée un metadata.duckdb temporaire avec medal_definitions peuplée."""
    from src.data.sync.migrations import ensure_medal_definitions_table

    db_path = tmp_path / "metadata.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        ensure_medal_definitions_table(conn)
        conn.executemany(
            "INSERT INTO medal_definitions "
            "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
            "VALUES (?, ?, ?, ?, ?, ?)",
            [
                (
                    17866865,
                    "Affection",
                    "Infected",
                    "Infectez tout le monde.",
                    "Infect everyone.",
                    False,
                ),
                (9000000001, "Vengeur", "Avenger", "Tuez votre tueur.", "Kill your killer.", True),
                (42, "TestFR", "TestEN", "Desc FR", "Desc EN", False),
            ],
        )
    return db_path


@pytest.fixture()
def empty_metadata_db(tmp_path):
    """Crée un metadata.duckdb temporaire sans medal_definitions."""
    db_path = tmp_path / "metadata.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("CREATE TABLE career_ranks (id INTEGER)")
    return db_path


class TestResolveMedalNameDb:
    """Tests de resolve_medal_name depuis la couche centralisée."""

    def test_returns_fr_name(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_name

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "Affection"

    def test_returns_en_name(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_name

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_name(17866865, lang="en")
        assert result == "Infected"

    def test_unknown_id_returns_str_id(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_name

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_name(999999999, lang="fr")
        assert result == "999999999"

    def test_default_lang_is_fr(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_name

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result_default = resolve_medal_name(42)
            result_fr = resolve_medal_name(42, lang="fr")
        assert result_default == result_fr == "TestFR"

    def test_returns_str_id_when_db_missing(self, tmp_path) -> None:
        from src.data.medal_definitions import resolve_medal_name

        absent = tmp_path / "absent.duckdb"
        with patch(_PATCH_TARGET, return_value=absent):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "17866865"

    def test_returns_str_id_when_table_missing(self, empty_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_name

        with patch(_PATCH_TARGET, return_value=empty_metadata_db):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "17866865"


class TestResolveMedalDescription:
    """Tests pour resolve_medal_description — DB-only."""

    def test_returns_fr_description(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_description

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_description(9000000001, lang="fr")
        assert result == "Tuez votre tueur."

    def test_returns_en_description(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_description

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_description(9000000001, lang="en")
        assert result == "Kill your killer."

    def test_returns_none_for_unknown_id(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_description

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = resolve_medal_description(999999999, lang="fr")
        assert result is None

    def test_returns_none_when_db_missing(self, tmp_path) -> None:
        from src.data.medal_definitions import resolve_medal_description

        absent = tmp_path / "absent.duckdb"
        with patch(_PATCH_TARGET, return_value=absent):
            result = resolve_medal_description(17866865, lang="fr")
        assert result is None

    def test_default_lang_is_fr(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import resolve_medal_description

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result_default = resolve_medal_description(42)
            result_fr = resolve_medal_description(42, lang="fr")
        assert result_default == result_fr == "Desc FR"

    def test_fallback_to_other_lang(self, tmp_path) -> None:
        """Si la colonne de la langue demandée est vide, retourne l'autre langue."""
        from src.data.medal_definitions import resolve_medal_description
        from src.data.sync.migrations import ensure_medal_definitions_table

        db_path = tmp_path / "metadata_partial.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            ensure_medal_definitions_table(conn)
            conn.execute(
                "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
                (100, "Nom FR", "Name EN", None, "Only EN desc", False),
            )

        with patch(_PATCH_TARGET, return_value=db_path):
            result = resolve_medal_description(100, lang="fr")
        assert result == "Only EN desc"


class TestLoadMedalNameMaps:
    """Tests pour load_medal_name_maps — bulk load."""

    def test_returns_fr_and_en_maps(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import load_medal_name_maps

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            fr_map, en_map = load_medal_name_maps()
        assert len(fr_map) >= 3
        assert fr_map["42"] == "TestFR"
        assert en_map["42"] == "TestEN"

    def test_returns_empty_when_db_missing(self, tmp_path) -> None:
        from src.data.medal_definitions import load_medal_name_maps

        absent = tmp_path / "absent.duckdb"
        with patch(_PATCH_TARGET, return_value=absent):
            fr_map, en_map = load_medal_name_maps()
        assert fr_map == {}
        assert en_map == {}


class TestResolveTextFromDb:
    """Tests pour _resolve_text_from_db (fonction interne)."""

    def test_selects_first_available_column(self, tmp_metadata_db) -> None:
        from src.data.medal_definitions import _resolve_text_from_db

        with patch(_PATCH_TARGET, return_value=tmp_metadata_db):
            result = _resolve_text_from_db(42, ["name_fr", "name_en"])
        assert result == "TestFR"

    def test_falls_back_to_second_column(self, tmp_path) -> None:
        from src.data.medal_definitions import _resolve_text_from_db
        from src.data.sync.migrations import ensure_medal_definitions_table

        db_path = tmp_path / "metadata_null.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            ensure_medal_definitions_table(conn)
            conn.execute(
                "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
                (200, "", "Fallback EN", "", "", False),
            )

        with patch(_PATCH_TARGET, return_value=db_path):
            result = _resolve_text_from_db(200, ["name_fr", "name_en"])
        assert result == "Fallback EN"


class TestAnalysisReExport:
    """Vérifie que src.analysis._medal_data ré-exporte correctement."""

    def test_reexports_resolve_medal_name(self) -> None:
        import sys

        # S'assurer que les deux modules sont chargés dans le même processus
        import src.analysis._medal_data  # noqa: F401
        import src.data.medal_definitions  # noqa: F401

        mod_analysis = sys.modules["src.analysis._medal_data"]
        mod_data = sys.modules["src.data.medal_definitions"]

        # Les deux noms doivent pointer vers la même fonction (même module source)
        assert mod_analysis.resolve_medal_name is mod_data.resolve_medal_name

    def test_reexports_resolve_medal_description(self) -> None:
        import sys

        import src.analysis._medal_data  # noqa: F401
        import src.data.medal_definitions  # noqa: F401

        mod_analysis = sys.modules["src.analysis._medal_data"]
        mod_data = sys.modules["src.data.medal_definitions"]

        assert mod_analysis.resolve_medal_description is mod_data.resolve_medal_description


class TestNoJsonFallback:
    """Vérifie que les modules n'utilisent plus de JSON."""

    def test_no_json_import_in_medal_definitions(self) -> None:
        import inspect

        import src.data.medal_definitions as mod

        source = inspect.getsource(mod)
        assert "import json" not in source

    def test_no_json_import_in_medal_data(self) -> None:
        import inspect

        import src.analysis._medal_data as mod

        source = inspect.getsource(mod)
        assert "import json" not in source
