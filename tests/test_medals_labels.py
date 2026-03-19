"""Tests pour les labels de médailles — source DB (medal_definitions).

- Valide que ``load_medal_name_maps()`` charge depuis la DB.
- Valide que les médailles listées dans le fixture ont un label FR/EN.
- Les fichiers JSON ``static/medals/`` restent comme référence mais ne sont plus
  utilisés par le code applicatif.
"""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import patch

import duckdb
import pytest

FIXTURES_DIR = Path(__file__).resolve().parent / "fixtures"
MEDALS_NEW_IDS_PATH = FIXTURES_DIR / "medals_new_ids.json"


@pytest.fixture()
def tmp_medal_db(tmp_path):
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
                (17866865, "Affection", "Infected", "", "", False),
                (1427176344, "Double Kill", "Double Kill", "", "", False),
                (9000000001, "Vengeur", "Avenger", "", "", True),
            ],
        )
    return db_path


class TestLoadMedalNameMaps:
    """Tests pour load_medal_name_maps() — source DB."""

    def test_returns_fr_and_en_maps(self, tmp_medal_db) -> None:
        from src.ui.medals import load_medal_name_maps

        with patch("src.ui.medals.get_metadata_db_path", return_value=tmp_medal_db):
            load_medal_name_maps.clear()
            fr_map, en_map = load_medal_name_maps()

        assert isinstance(fr_map, dict)
        assert isinstance(en_map, dict)
        assert len(fr_map) >= 3
        assert "17866865" in fr_map
        assert fr_map["17866865"] == "Affection"

    def test_returns_empty_when_db_missing(self, tmp_path) -> None:
        from src.ui.medals import load_medal_name_maps

        absent = tmp_path / "absent.duckdb"
        with patch("src.ui.medals.get_metadata_db_path", return_value=absent):
            load_medal_name_maps.clear()
            fr_map, en_map = load_medal_name_maps()

        assert fr_map == {}
        assert en_map == {}


class TestMedalLabel:
    """Tests pour medal_label() et medal_has_known_label()."""

    def test_medal_label_fr(self, tmp_medal_db) -> None:
        from src.ui.medals import medal_label

        with patch("src.ui.medals.get_metadata_db_path", return_value=tmp_medal_db):
            from src.ui.medals import load_medal_name_maps

            load_medal_name_maps.clear()
            result = medal_label(17866865, lang="fr")

        assert result == "Affection"

    def test_medal_label_en(self, tmp_medal_db) -> None:
        from src.ui.medals import medal_label

        with patch("src.ui.medals.get_metadata_db_path", return_value=tmp_medal_db):
            from src.ui.medals import load_medal_name_maps

            load_medal_name_maps.clear()
            result = medal_label(17866865, lang="en")

        assert result == "Infected"

    def test_medal_label_unknown(self, tmp_medal_db) -> None:
        from src.ui.medals import medal_label

        with patch("src.ui.medals.get_metadata_db_path", return_value=tmp_medal_db):
            from src.ui.medals import load_medal_name_maps

            load_medal_name_maps.clear()
            result = medal_label(999999999, lang="fr")

        assert result == "Médaille #999999999"

    def test_medal_has_known_label(self, tmp_medal_db) -> None:
        from src.ui.medals import medal_has_known_label

        with patch("src.ui.medals.get_metadata_db_path", return_value=tmp_medal_db):
            from src.ui.medals import load_medal_name_maps

            load_medal_name_maps.clear()
            assert medal_has_known_label(17866865) is True
            assert medal_has_known_label(999999999) is False


class TestNoJsonFallback:
    """Vérifie que medals.py n'a plus de fallback JSON."""

    def test_no_json_import(self) -> None:
        import inspect

        import src.ui.medals as mod

        source = inspect.getsource(mod)
        assert "import json" not in source
        assert "_load_from_json" not in source
        assert "_medals_json_mtime" not in source

    def test_no_json_fallback_logic(self) -> None:
        import inspect

        import src.ui.medals as mod

        source = inspect.getsource(mod)
        assert "Fallback JSON" not in source


class TestNewMedalsLabels:
    """Vérifie que les médailles du fixture ont un label (tests fonctionnels avec DB réelle)."""

    @pytest.fixture(scope="class")
    def expected_new_medal_ids(self) -> list[int]:
        """Charge la liste des IDs attendus depuis le fixture JSON."""
        if not MEDALS_NEW_IDS_PATH.exists():
            pytest.skip(
                f"Fixture absent: {MEDALS_NEW_IDS_PATH}. "
                "Créer un JSON array de strings (IDs médailles) pour activer ces tests."
            )
        with open(MEDALS_NEW_IDS_PATH, encoding="utf-8") as f:
            raw = json.load(f)
        assert isinstance(raw, list)
        return [int(str(item).strip()) for item in raw if str(item).strip().isdigit()]

    def test_new_medals_have_known_label(self, expected_new_medal_ids: list[int]) -> None:
        """Chaque ID du fixture a un label connu via medal_has_known_label."""
        from src.ui.medals import load_medal_name_maps, medal_has_known_label

        load_medal_name_maps.clear()
        for nid in expected_new_medal_ids:
            assert medal_has_known_label(nid), f"Médaille {nid} doit avoir un label en DB"

    def test_new_medals_label_is_not_placeholder(self, expected_new_medal_ids: list[int]) -> None:
        """medal_label() ne renvoie pas le placeholder pour les IDs du fixture."""
        from src.ui.medals import load_medal_name_maps, medal_label

        load_medal_name_maps.clear()
        for nid in expected_new_medal_ids:
            label = medal_label(nid)
            assert label != f"Médaille #{nid}", f"Médaille {nid} ne doit pas être un placeholder"
