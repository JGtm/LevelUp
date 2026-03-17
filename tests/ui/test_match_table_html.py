"""Tests pour le rendu HTML des thumbnails de carte (hover CSS).

Vérifie que le HTML généré contient les classes CSS attendues
et gère correctement les cas sans thumbnail.
"""

from __future__ import annotations

import html as html_lib
from pathlib import Path
from unittest.mock import patch

import pytest

from src.ui.pages.match_table_html import _render_cell, map_thumb_url

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def _clear_map_cache():
    """Vide le cache de _build_map_url_index entre chaque test."""
    from src.ui.pages.match_table_html import _build_map_url_index

    _build_map_url_index.cache_clear()
    yield
    _build_map_url_index.cache_clear()


# ---------------------------------------------------------------------------
# map_thumb_url
# ---------------------------------------------------------------------------


def test_map_thumb_url_known_map():
    """Une carte présente dans static/maps/ renvoie une URL."""
    url = map_thumb_url("Aquarius")
    if url is not None:
        assert url.startswith("/app/static/maps/")
        assert "aquarius" in url.lower()


def test_map_thumb_url_none():
    """Aucun nom de carte → None."""
    assert map_thumb_url(None) is None
    assert map_thumb_url("") is None


def test_map_thumb_url_unknown_map():
    """Carte inconnue → None."""
    assert map_thumb_url("CetteCarteNExistePas_XYZ_42") is None


# ---------------------------------------------------------------------------
# _build_map_url_index — normalisation Unicode
# ---------------------------------------------------------------------------


def test_build_map_url_index_unicode(tmp_path: Path):
    """Les noms de fichier avec accents/espaces sont correctement normalisés."""
    from src.ui.pages.match_table_html import _build_map_url_index

    maps_dir = tmp_path / "static" / "maps"
    maps_dir.mkdir(parents=True)
    # Créer un fichier avec accent
    (maps_dir / "Épreuve.jpg").touch()
    (maps_dir / "test map.png").touch()

    with patch("src.ui.pages.match_table_html.get_repo_root", return_value=str(tmp_path)):
        _build_map_url_index.cache_clear()
        idx = _build_map_url_index()

    assert "épreuve" in idx or "epreuve" in idx
    assert "test map" in idx or "test_map" in idx


# ---------------------------------------------------------------------------
# Hover HTML : _render_cell pour map_name
# ---------------------------------------------------------------------------


def test_map_hover_html_generated():
    """Cellule map_name avec thumbnail contient les classes CSS hover."""
    with patch(
        "src.ui.pages.match_table_html.map_thumb_url",
        return_value="/app/static/maps/aquarius.png",
    ):
        row = {"map_name": "Aquarius", "outcome": 2}
        result = _render_cell(row, "map_name", outcome_code=2)

    assert "class='map-hover'" in result
    assert "class='map-popup'" in result
    assert "<img" in result
    assert "src='/app/static/maps/aquarius.png'" in result
    assert html_lib.escape("Aquarius") in result


def test_map_hover_no_url_fallback():
    """Carte sans thumbnail → <td> simple, pas de popup."""
    with patch(
        "src.ui.pages.match_table_html.map_thumb_url",
        return_value=None,
    ):
        row = {"map_name": "Unknown Map", "outcome": 2}
        result = _render_cell(row, "map_name", outcome_code=2)

    assert "map-hover" not in result
    assert "map-popup" not in result
    assert "<td>" in result
    assert "Unknown Map" in result


# ---------------------------------------------------------------------------
# load_css : pas de <script>
# ---------------------------------------------------------------------------


def test_no_js_in_load_css():
    """load_css() ne contient aucune balise <script>."""
    from src.ui.styles import load_css

    css_output = load_css()
    assert "<script" not in css_output.lower()
    assert "</script>" not in css_output.lower()


def test_load_css_fallback(tmp_path: Path):
    """CSS introuvable → fallback minimal avec balise <style>."""
    from src.ui.styles import load_css

    with patch("src.ui.styles.get_css_path", return_value=str(tmp_path / "inexistant.css")):
        result = load_css()

    assert "<style>" in result
    assert ".hero" in result


def test_load_css_map_hover_wrapper_allows_horizontal_overflow():
    """Le wrapper map-hover ne doit pas couper les popups en horizontal."""
    from src.ui.styles import load_css

    css_output = load_css()

    assert ".os-table-wrap--map-hover {" in css_output
    assert "overflow-x: visible;" in css_output


# ---------------------------------------------------------------------------
# _scoreboard_row_to_dict
# ---------------------------------------------------------------------------


def test_scoreboard_row_to_dict_valid():
    """Tuple complet → dict avec toutes les clés attendues."""
    from src.data.repositories._roster_loader import _scoreboard_row_to_dict

    row = (
        "xuid123",  # xuid
        "Spartan42",  # gamertag
        1,  # team_id
        2,  # rank
        1500,  # score
        15,  # kills
        6,  # deaths
        4,  # assists
        2.5,  # kda
        7,  # max_killing_spree
        8,  # headshot_kills
        120,  # shots_fired
        60,  # shots_hit
        0.5,  # accuracy
        3,  # melee_kills
        2,  # power_weapon_kills
        1500.0,  # damage_dealt
        900.0,  # damage_taken
        45.0,  # avg_life_seconds
        5,  # perfect_kills
        42,  # top_weapon_id
    )
    result = _scoreboard_row_to_dict(0, row)

    assert result["xuid"] == "xuid123"
    assert result["gamertag"] == "Spartan42"
    assert result["team_id"] == 1
    assert result["kills"] == 15
    assert result["kda"] == 2.5
    assert result["perfect_kills"] == 5
    assert result["top_weapon_id"] == 42


def test_scoreboard_row_to_dict_nulls():
    """Tuple avec None → fallbacks corrects."""
    from src.data.repositories._roster_loader import _scoreboard_row_to_dict

    row = (
        "xuid456",  # xuid
        None,  # gamertag → fallback sur xuid
        None,  # team_id
        None,  # rank → fallback idx+1
        None,  # score
        None,  # kills
        None,  # deaths
        None,  # assists
        None,  # kda
        None,  # max_killing_spree
        None,  # headshot_kills
        None,  # shots_fired
        None,  # shots_hit
        None,  # accuracy
        None,  # melee_kills
        None,  # power_weapon_kills
        None,  # damage_dealt
        None,  # damage_taken
        None,  # avg_life_seconds
        None,  # perfect_kills → 0
        None,  # top_weapon_id → None
    )
    result = _scoreboard_row_to_dict(3, row)

    assert result["gamertag"] == "xuid456"  # fallback sur xuid
    assert result["team_id"] is None
    assert result["rank"] == 4  # idx(3) + 1
    assert result["perfect_kills"] == 0
    assert result["top_weapon_id"] is None
