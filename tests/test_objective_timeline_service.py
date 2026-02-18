"""Tests pour le service d'estimation de timeline des objectifs.

Valide l'estimation de timeline approximative en croisant PersonalScores
avec highlight_events horodatés.
"""

import pytest

try:
    import polars as pl

    POLARS_AVAILABLE = True
except ImportError:
    POLARS_AVAILABLE = False


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_player_highlight_events_returns_dataframe():
    """Vérifie que get_player_highlight_events retourne un DataFrame."""
    from unittest.mock import MagicMock

    from src.data.services.objective_timeline_service import get_player_highlight_events

    mock_repo = MagicMock()
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "time_ms": [15000, 32000, 48000],
            "event_type": ["kill", "death", "kill"],
            "killer_xuid": ["xuid1", None, "xuid1"],
            "victim_xuid": [None, "xuid1", "xuid2"],
        }
    )

    result = get_player_highlight_events(mock_repo, "match123", "xuid1")

    assert isinstance(result, pl.DataFrame)
    assert result.height == 3
    assert "time_ms" in result.columns
    assert "event_type" in result.columns
    mock_repo.query_df.assert_called_once()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_distribute_captures_single_capture():
    """Vérifie la distribution d'une seule capture entre des événements."""
    from src.data.services.objective_timeline_service import (
        _distribute_captures_across_events,
    )

    events = pl.DataFrame(
        {
            "time_ms": [10000, 50000, 90000],
            "event_type": ["kill", "kill", "kill"],
        }
    )

    result = _distribute_captures_across_events(
        events=events,
        capture_count=1,
        team_id=0,
        xuid="xuid1",
        gamertag="TestPlayer",
        award_name="Flag Captured",
    )

    assert isinstance(result, pl.DataFrame)
    assert result.height == 1
    assert result["capture_index"][0] == 1
    assert 10000 <= result["estimated_time_ms"][0] <= 90000
    assert result["confidence"][0] in ["high", "medium", "low"]


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_distribute_captures_multiple_captures():
    """Vérifie la distribution de plusieurs captures."""
    from src.data.services.objective_timeline_service import (
        _distribute_captures_across_events,
    )

    events = pl.DataFrame(
        {
            "time_ms": [10000, 30000, 50000, 70000, 90000],
            "event_type": ["kill"] * 5,
        }
    )

    result = _distribute_captures_across_events(
        events=events,
        capture_count=3,
        team_id=0,
        xuid="xuid1",
        gamertag="TestPlayer",
        award_name="Flag Captured",
    )

    assert result.height == 3
    assert result["capture_index"].to_list() == [1, 2, 3]
    # Les timestamps doivent être croissants
    times = result["estimated_time_ms"].to_list()
    assert times[0] < times[1] < times[2]
    # Tous dans la plage
    assert all(10000 <= t <= 90000 for t in times)


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_distribute_captures_confidence_high():
    """Vérifie que beaucoup d'événements donnent une confiance "high"."""
    from src.data.services.objective_timeline_service import (
        _distribute_captures_across_events,
    )

    # Beaucoup d'événements rapprochés
    events = pl.DataFrame(
        {
            "time_ms": list(range(10000, 60000, 5000)),  # 10 événements
            "event_type": ["kill"] * 10,
        }
    )

    result = _distribute_captures_across_events(
        events=events,
        capture_count=1,
        team_id=0,
        xuid="xuid1",
        gamertag="TestPlayer",
        award_name="Flag Captured",
    )

    # Avec 10 événements sur 50s, la confiance devrait être "high"
    assert result["confidence"][0] == "high"
    assert result["nearby_events_count"][0] >= 5


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_distribute_captures_confidence_low():
    """Vérifie que peu d'événements donnent une confiance "low"."""
    from src.data.services.objective_timeline_service import (
        _distribute_captures_across_events,
    )

    # Peu d'événements éloignés
    events = pl.DataFrame(
        {
            "time_ms": [10000, 200000],  # 2 événements très espacés
            "event_type": ["kill", "kill"],
        }
    )

    result = _distribute_captures_across_events(
        events=events,
        capture_count=1,
        team_id=0,
        xuid="xuid1",
        gamertag="TestPlayer",
        award_name="Flag Captured",
    )

    # Avec seulement 2 événements sur 190s, la confiance devrait être "low"
    assert result["confidence"][0] == "low"
    assert result["nearby_events_count"][0] < 5


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_distribute_captures_empty_events():
    """Vérifie le comportement avec aucun événement."""
    from src.data.services.objective_timeline_service import (
        _distribute_captures_across_events,
    )

    events = pl.DataFrame()

    result = _distribute_captures_across_events(
        events=events,
        capture_count=3,
        team_id=0,
        xuid="xuid1",
        gamertag="TestPlayer",
        award_name="Flag Captured",
    )

    assert result.is_empty()


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_award_filters_for_ctf():
    """Vérifie les filtres d'awards pour le mode CTF."""
    from src.data.services.objective_timeline_service import _get_award_filters_for_mode

    filters = _get_award_filters_for_mode("CTF")

    assert isinstance(filters, list)
    assert "Flag Captured" in filters


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_award_filters_for_strongholds():
    """Vérifie les filtres d'awards pour le mode Strongholds."""
    from src.data.services.objective_timeline_service import _get_award_filters_for_mode

    filters = _get_award_filters_for_mode("Strongholds")

    assert isinstance(filters, list)
    assert "Zone Captured 100%" in filters
    assert "Zone Captured 75%" in filters
    assert "Zone Captured 50%" in filters


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_award_filters_for_oddball():
    """Vérifie les filtres d'awards pour le mode Oddball."""
    from src.data.services.objective_timeline_service import _get_award_filters_for_mode

    filters = _get_award_filters_for_mode("Oddball")

    assert isinstance(filters, list)
    assert "Ball Control" in filters


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_ms_to_mmss_formatting():
    """Vérifie la conversion millisecondes → MM:SS."""
    from src.data.services.objective_timeline_service import _ms_to_mmss

    assert _ms_to_mmss(90000) == "01:30"
    assert _ms_to_mmss(195000) == "03:15"
    assert _ms_to_mmss(0) == "00:00"
    assert _ms_to_mmss(599000) == "09:59"


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_format_timeline_for_display():
    """Vérifie le formatage de la timeline pour l'affichage."""
    from src.data.services.objective_timeline_service import format_timeline_for_display

    timeline = pl.DataFrame(
        {
            "team_id": [0, 0],
            "xuid": ["xuid1", "xuid1"],
            "gamertag": ["TestPlayer", "TestPlayer"],
            "award_name": ["Flag Captured", "Flag Captured"],
            "capture_index": [1, 2],
            "estimated_time_ms": [90000, 195000],
            "confidence": ["high", "medium"],
            "nearby_events_count": [5, 3],
        }
    )

    result = format_timeline_for_display(timeline, match_duration_ms=600000)

    assert "time_formatted" in result.columns
    assert result["time_formatted"][0] == "01:30"
    assert result["time_formatted"][1] == "03:15"
    assert "time_percent" in result.columns
    assert 0.0 <= result["time_percent"][0] <= 1.0


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_timeline_summary():
    """Vérifie le calcul des statistiques résumées."""
    from src.data.services.objective_timeline_service import get_timeline_summary

    timeline = pl.DataFrame(
        {
            "team_id": [0, 0, 1],
            "xuid": ["xuid1", "xuid1", "xuid2"],
            "gamertag": ["Player1", "Player1", "Player2"],
            "award_name": ["Flag Captured"] * 3,
            "capture_index": [1, 2, 1],
            "estimated_time_ms": [45000, 120000, 180000],
            "confidence": ["high", "medium", "low"],
            "nearby_events_count": [8, 3, 1],
        }
    )

    summary = get_timeline_summary(timeline)

    assert summary["total_captures"] == 3
    assert summary["high_confidence_count"] == 1
    assert summary["medium_confidence_count"] == 1
    assert summary["low_confidence_count"] == 1
    assert summary["average_nearby_events"] == (8 + 3 + 1) / 3
    assert summary["first_capture_time_ms"] == 45000
    assert summary["last_capture_time_ms"] == 180000


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_get_timeline_summary_empty():
    """Vérifie le comportement avec une timeline vide."""
    from src.data.services.objective_timeline_service import get_timeline_summary

    timeline = pl.DataFrame()

    summary = get_timeline_summary(timeline)

    assert summary["total_captures"] == 0
    assert summary["high_confidence_count"] == 0
    assert summary["first_capture_time_ms"] is None


@pytest.mark.skipif(not POLARS_AVAILABLE, reason="Polars non disponible")
def test_estimate_objective_captures_timeline_mocked():
    """Teste estimate_objective_captures_timeline avec des mocks complets."""
    from unittest.mock import MagicMock, patch

    from src.data.services.objective_timeline_service import (
        estimate_objective_captures_timeline,
    )

    mock_repo = MagicMock()
    
    # Mock de la requête capturers
    mock_repo.query_df.return_value = pl.DataFrame(
        {
            "team_id": [0],
            "xuid": ["xuid1"],
            "gamertag": ["TestPlayer"],
            "award_name": ["Flag Captured"],
            "award_count": [2],
        }
    )
    
    # Mock de get_player_highlight_events
    with patch(
        "src.data.services.objective_timeline_service.get_player_highlight_events"
    ) as mock_get_events:
        mock_get_events.return_value = pl.DataFrame(
            {
                "time_ms": [10000, 50000, 90000],
                "event_type": ["kill", "kill", "kill"],
                "killer_xuid": ["xuid1", "xuid1", "xuid1"],
                "victim_xuid": [None, None, None],
            }
        )
        
        result = estimate_objective_captures_timeline(mock_repo, "match123", "CTF")
        
        # Doit retourner 2 estimations (award_count=2)
        assert isinstance(result, pl.DataFrame)
        assert result.height == 2
        assert "estimated_time_ms" in result.columns
        assert "confidence" in result.columns
