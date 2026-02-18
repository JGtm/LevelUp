"""Tests pour le service objective_events_service.

Valide l'extraction des événements d'objectifs (CTF, Strongholds, Oddball)
depuis personal_score_awards et match_participants.
"""

import pytest

try:
    import polars as pl

    POLARS_AVAILABLE = True
except ImportError:
    POLARS_AVAILABLE = False


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_flag_captures_by_team_returns_dataframe():
    """Vérifie que get_flag_captures_by_team retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_flag_captures_by_team

    # Mock du repository
    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "team_id": [0, 1],
            "total_captures": [3, 2],
            "total_score": [900, 600],
        }
    )

    # Appel
    result = get_flag_captures_by_team(mock_repo, "match_abc123")

    # Assertions
    assert isinstance(result, pl.DataFrame)
    assert result.height == 2
    assert "team_id" in result.columns
    assert "total_captures" in result.columns
    assert "total_score" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_flag_captures_by_player_returns_dataframe():
    """Vérifie que get_flag_captures_by_player retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_flag_captures_by_player

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "gamertag": ["JohnSpartan", "MasterChief"],
            "team_id": [0, 0],
            "flag_captured": [3, 1],
            "flag_stolen": [1, 2],
            "flag_returned": [0, 1],
            "flag_capture_assist": [1, 0],
            "total_score": [1025, 350],
        }
    )

    result = get_flag_captures_by_player(mock_repo, "match_abc123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 2
    assert "gamertag" in result.columns
    assert "flag_captured" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_base_captures_by_team_returns_dataframe():
    """Vérifie que get_base_captures_by_team retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_base_captures_by_team

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "team_id": [0, 1],
            "zone_50_count": [8, 7],
            "zone_75_count": [6, 5],
            "zone_100_count": [5, 4],
            "total_captures": [19, 16],
            "total_score": [1375, 1200],
        }
    )

    result = get_base_captures_by_team(mock_repo, "match_sh_123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 2
    assert "zone_50_count" in result.columns
    assert "zone_75_count" in result.columns
    assert "zone_100_count" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_base_captures_by_player_returns_dataframe():
    """Vérifie que get_base_captures_by_player retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_base_captures_by_player

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "gamertag": ["PlayerA", "PlayerB", "PlayerC"],
            "team_id": [0, 0, 1],
            "zone_50_count": [3, 2, 2],
            "zone_75_count": [2, 1, 2],
            "zone_100_count": [3, 2, 2],
            "zone_secured_count": [1, 1, 1],
            "total_score": [300, 200, 200],
        }
    )

    result = get_base_captures_by_player(mock_repo, "match_sh_123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 3
    assert "gamertag" in result.columns
    assert "zone_100_count" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_oddball_control_by_team_returns_dataframe():
    """Vérifie que get_oddball_control_by_team retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_oddball_control_by_team

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "team_id": [0, 1],
            "ball_control_count": [12, 8],
            "ball_taken_count": [3, 2],
            "carrier_stopped_count": [2, 3],
            "total_score": [600, 400],
        }
    )

    result = get_oddball_control_by_team(mock_repo, "match_ob_123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 2
    assert "ball_control_count" in result.columns
    assert "total_score" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_oddball_control_by_player_returns_dataframe():
    """Vérifie que get_oddball_control_by_player retourne un DataFrame Polars."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_oddball_control_by_player

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "gamertag": ["PlayerX", "PlayerY", "PlayerZ"],
            "team_id": [0, 0, 1],
            "ball_control_count": [8, 4, 6],
            "ball_taken_count": [2, 1, 1],
            "carrier_stopped_count": [1, 1, 2],
            "total_score": [400, 200, 300],
        }
    )

    result = get_oddball_control_by_player(mock_repo, "match_ob_123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 3
    assert "ball_control_count" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_objective_mvp_returns_single_row():
    """Vérifie que get_objective_mvp retourne une seule ligne (le MVP)."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_objective_mvp

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "gamertag": ["JohnSpartan"],
            "team_id": [0],
            "total_objective_score": [1025],
            "total_actions": [15],
        }
    )

    result = get_objective_mvp(mock_repo, "match_abc123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 1
    assert result["gamertag"][0] == "JohnSpartan"
    assert result["total_objective_score"][0] == 1025
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_objective_mvp_returns_none_when_empty():
    """Vérifie que get_objective_mvp retourne None si aucune donnée."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_objective_mvp

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame()  # DataFrame vide

    result = get_objective_mvp(mock_repo, "match_no_objectives")

    assert result is None
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_objective_events_by_team_returns_all_categories():
    """Vérifie que get_objective_events_by_team agrège tous les types d'objectifs."""
    from unittest.mock import MagicMock

    from src.data.services.objective_events_service import get_objective_events_by_team

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "team_id": [0, 0, 1, 1],
            "award_name": [
                "Flag Captured",
                "Zone Captured 100%",
                "Ball Control",
                "Flag Captured",
            ],
            "total_count": [3, 5, 12, 2],
            "total_score": [900, 500, 600, 600],
        }
    )

    result = get_objective_events_by_team(mock_repo, "match_mixed_123")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 4
    assert "Flag Captured" in result["award_name"].to_list()
    assert "Zone Captured 100%" in result["award_name"].to_list()
    assert "Ball Control" in result["award_name"].to_list()
    mock_repo.query_df.assert_called_once()
