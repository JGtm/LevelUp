"""Tests pour src/data/citation_definitions.py — couche centralisée citations."""

from __future__ import annotations

from unittest.mock import patch

import duckdb
import pytest


@pytest.fixture()
def tmp_metadata_with_citations(tmp_path):
    """Crée un metadata.duckdb temporaire avec citation_mappings peuplée."""
    db_path = tmp_path / "metadata.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE citation_mappings (
                citation_name_norm VARCHAR PRIMARY KEY,
                citation_name_display VARCHAR,
                mapping_type VARCHAR DEFAULT 'medal',
                medal_id BIGINT,
                medal_ids VARCHAR,
                stat_name VARCHAR,
                award_name VARCHAR,
                award_category VARCHAR,
                custom_function VARCHAR,
                composite_children VARCHAR,
                confidence VARCHAR DEFAULT 'high',
                notes VARCHAR,
                enabled BOOLEAN DEFAULT TRUE,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                image_path VARCHAR,
                category VARCHAR,
                description VARCHAR,
                tier_targets VARCHAR,
                subcategory VARCHAR
            )
        """)
        conn.executemany(
            "INSERT INTO citation_mappings "
            "(citation_name_norm, citation_name_display, mapping_type, category, "
            " description, tier_targets, enabled, image_path, composite_children, subcategory) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            [
                (
                    "defenseur_du_drapeau",
                    "Défenseur du drapeau",
                    "medal",
                    "Mode de jeu",
                    "Protégez le drapeau.",
                    "10,25,50",
                    True,
                    "static/comm/flag.png",
                    None,
                    None,
                ),
                (
                    "tueur_en_serie",
                    "Tueur en série",
                    "stat",
                    "Combat",
                    "Enchaînez les kills.",
                    "5,10,25,50",
                    True,
                    None,
                    None,
                    None,
                ),
                (
                    "desactive",
                    "Désactivé",
                    "medal",
                    "Combat",
                    "Ne devrait pas apparaître.",
                    "10",
                    False,
                    None,
                    None,
                    None,
                ),
            ],
        )
    return db_path


class TestLoadCitationDefinitions:
    """Tests pour load_citation_definitions()."""

    def test_loads_enabled_citations(self, tmp_metadata_with_citations) -> None:
        from src.data.citation_definitions import load_citation_definitions

        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=tmp_metadata_with_citations,
        ):
            result = load_citation_definitions()

        assert len(result) == 2  # 'desactive' filtré
        norms = [c["citation_name_norm"] for c in result]
        assert "defenseur_du_drapeau" in norms
        assert "tueur_en_serie" in norms
        assert "desactive" not in norms

    def test_returns_expected_keys(self, tmp_metadata_with_citations) -> None:
        from src.data.citation_definitions import load_citation_definitions

        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=tmp_metadata_with_citations,
        ):
            result = load_citation_definitions()

        assert len(result) > 0
        expected_keys = {
            "citation_name_norm",
            "citation_name_display",
            "mapping_type",
            "image_path",
            "category",
            "description",
            "tier_targets",
            "composite_children",
            "subcategory",
        }
        assert set(result[0].keys()) == expected_keys

    def test_returns_empty_when_db_missing(self, tmp_path) -> None:
        from src.data.citation_definitions import load_citation_definitions

        absent = tmp_path / "absent.duckdb"
        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=absent,
        ):
            result = load_citation_definitions()

        assert result == []

    def test_returns_empty_when_table_missing(self, tmp_path) -> None:
        from src.data.citation_definitions import load_citation_definitions

        db_path = tmp_path / "metadata.duckdb"
        with duckdb.connect(str(db_path)) as conn:
            conn.execute("CREATE TABLE career_ranks (id INTEGER)")

        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=db_path,
        ):
            result = load_citation_definitions()

        assert result == []


class TestGetCitationDisplayName:
    """Tests pour get_citation_display_name()."""

    def test_resolves_known_norm(self, tmp_metadata_with_citations) -> None:
        from src.data.citation_definitions import get_citation_display_name

        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=tmp_metadata_with_citations,
        ):
            result = get_citation_display_name("defenseur_du_drapeau")

        assert result == "Défenseur du drapeau"

    def test_returns_raw_key_for_unknown(self, tmp_metadata_with_citations) -> None:
        from src.data.citation_definitions import get_citation_display_name

        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=tmp_metadata_with_citations,
        ):
            result = get_citation_display_name("inconnu_xyz")

        assert result == "inconnu_xyz"

    def test_returns_raw_key_when_db_missing(self, tmp_path) -> None:
        from src.data.citation_definitions import get_citation_display_name

        absent = tmp_path / "absent.duckdb"
        with patch(
            "src.data.citation_definitions.get_metadata_db_path",
            return_value=absent,
        ):
            result = get_citation_display_name("defenseur_du_drapeau")

        assert result == "defenseur_du_drapeau"


class TestNoJsonDependency:
    """Vérifie que le module n'a aucune dépendance JSON."""

    def test_no_json_import(self) -> None:
        import inspect

        import src.data.citation_definitions as mod

        source = inspect.getsource(mod)
        assert "import json" not in source
        assert "label_obj" not in source
        assert "static/i18n" not in source
