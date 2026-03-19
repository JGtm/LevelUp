"""Tests pour src/analysis/_medal_data.py — résolution DB-only."""

from __future__ import annotations

from unittest.mock import patch

import duckdb
import pytest


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
    """Tests de resolve_medal_name avec metadata.duckdb."""

    def test_returns_fr_name(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_name

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "Affection"

    def test_returns_en_name(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_name

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_name(17866865, lang="en")
        assert result == "Infected"

    def test_unknown_id_returns_str_id(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_name

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_name(999999999, lang="fr")
        assert result == "999999999"

    def test_default_lang_is_fr(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_name

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result_default = resolve_medal_name(42)
            result_fr = resolve_medal_name(42, lang="fr")
        assert result_default == result_fr == "TestFR"

    def test_returns_str_id_when_db_missing(self, tmp_path) -> None:
        from src.analysis._medal_data import resolve_medal_name

        absent = tmp_path / "absent.duckdb"
        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=absent):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "17866865"

    def test_returns_str_id_when_table_missing(self, empty_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_name

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=empty_metadata_db):
            result = resolve_medal_name(17866865, lang="fr")
        assert result == "17866865"


class TestResolveMedalDescription:
    """Tests pour resolve_medal_description — DB-only."""

    def test_returns_fr_description(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_description

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_description(9000000001, lang="fr")
        assert result == "Tuez votre tueur."

    def test_returns_en_description(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_description

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_description(9000000001, lang="en")
        assert result == "Kill your killer."

    def test_returns_none_for_unknown_id(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_description

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = resolve_medal_description(999999999, lang="fr")
        assert result is None

    def test_returns_none_when_db_missing(self, tmp_path) -> None:
        from src.analysis._medal_data import resolve_medal_description

        absent = tmp_path / "absent.duckdb"
        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=absent):
            result = resolve_medal_description(17866865, lang="fr")
        assert result is None

    def test_default_lang_is_fr(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import resolve_medal_description

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result_default = resolve_medal_description(42)
            result_fr = resolve_medal_description(42, lang="fr")
        assert result_default == result_fr == "Desc FR"

    def test_fallback_to_other_lang(self, tmp_path) -> None:
        """Si la colonne de la langue demandée est vide, retourne l'autre langue."""
        from src.analysis._medal_data import resolve_medal_description
        from src.data.sync.migrations import ensure_medal_definitions_table

        db_path = tmp_path / "metadata_partial.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            ensure_medal_definitions_table(conn)
            conn.execute(
                "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
                (100, "Nom FR", "Name EN", None, "Only EN desc", False),
            )

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=db_path):
            result = resolve_medal_description(100, lang="fr")
        assert result == "Only EN desc"


class TestResolveTextFromDb:
    """Tests pour _resolve_text_from_db (fonction interne)."""

    def test_selects_first_available_column(self, tmp_metadata_db) -> None:
        from src.analysis._medal_data import _resolve_text_from_db

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=tmp_metadata_db):
            result = _resolve_text_from_db(42, "fr", ["name_fr", "name_en"])
        assert result == "TestFR"

    def test_falls_back_to_second_column(self, tmp_path) -> None:
        from src.analysis._medal_data import _resolve_text_from_db
        from src.data.sync.migrations import ensure_medal_definitions_table

        db_path = tmp_path / "metadata_null.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            ensure_medal_definitions_table(conn)
            # name_fr est NOT NULL — on insère une chaîne vide pour forcer le fallback
            conn.execute(
                "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
                (200, "", "Fallback EN", "", "", False),
            )

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=db_path):
            result = _resolve_text_from_db(200, "fr", ["name_fr", "name_en"])
        assert result == "Fallback EN"


class TestNoJsonFallback:
    """Vérifie que le module n'utilise plus de JSON."""

    def test_no_json_import(self) -> None:
        import inspect

        import src.analysis._medal_data as mod

        source = inspect.getsource(mod)
        assert "import json" not in source
        assert "_load_json_map" not in source
        assert "_load_description_map" not in source
