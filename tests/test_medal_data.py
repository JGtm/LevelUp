"""Tests pour src/analysis/_medal_data.py."""

from __future__ import annotations

from unittest.mock import patch

import duckdb


class TestResolveMedalNameJson:
    """Tests via la source JSON statique (sans metadata.duckdb)."""

    def test_returns_fr_name(self) -> None:
        from src.analysis._medal_data import resolve_medal_name

        # 17866865 = "Affection" en FR (vérifié dans medals_fr.json)
        result = resolve_medal_name(17866865, lang="fr")
        assert isinstance(result, str)
        assert len(result) > 0
        assert result != "17866865"

    def test_returns_en_name(self) -> None:
        from src.analysis._medal_data import resolve_medal_name

        # 17866865 doit exister en EN aussi
        result = resolve_medal_name(17866865, lang="en")
        assert isinstance(result, str)
        assert result != "17866865"

    def test_unknown_id_returns_str_id(self) -> None:
        from src.analysis._medal_data import resolve_medal_name

        result = resolve_medal_name(999999999, lang="fr")
        assert result == "999999999"

    def test_default_lang_is_fr(self) -> None:
        from src.analysis._medal_data import resolve_medal_name

        result_default = resolve_medal_name(17866865)
        result_fr = resolve_medal_name(17866865, lang="fr")
        assert result_default == result_fr


class TestResolveMedalNameDb:
    """Tests avec metadata.duckdb simulé."""

    def test_db_source_used_when_table_exists(self, tmp_path) -> None:
        from src.analysis._medal_data import _resolve_from_db

        # Créer une DB temporaire avec la table medals
        db_file = tmp_path / "metadata.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute(
                "CREATE TABLE medals (medal_name_id INTEGER, name_fr VARCHAR, name_en VARCHAR)"
            )
            conn.execute("INSERT INTO medals VALUES (42, 'TestFR', 'TestEN')")

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=db_file):
            result = _resolve_from_db(42, "fr")

        assert result == "TestFR"

    def test_db_returns_none_when_no_table(self, tmp_path) -> None:
        from src.analysis._medal_data import _resolve_from_db

        db_file = tmp_path / "metadata.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("CREATE TABLE career_ranks (id INTEGER)")

        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=db_file):
            result = _resolve_from_db(42, "fr")

        assert result is None

    def test_db_returns_none_when_file_missing(self, tmp_path) -> None:
        from src.analysis._medal_data import _resolve_from_db

        absent = tmp_path / "absent.duckdb"
        with patch("src.analysis._medal_data.get_metadata_db_path", return_value=absent):
            result = _resolve_from_db(42, "fr")

        assert result is None
