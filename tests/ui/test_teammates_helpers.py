"""Tests pour teammates_helpers.py et teammates_synergy.py — Sprint 7ter (7t.4).

Teste les fonctions pures et les helpers d'affichage.
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import polars as pl
import pytest

# ============================================================================
# teammates_helpers.py — fonctions pures
# ============================================================================


class TestFormatDatetimeFrHm:
    def test_valid_datetime(self):
        from datetime import datetime

        from src.ui.pages.teammates_helpers import _format_datetime_fr_hm

        dt = datetime(2024, 3, 15, 14, 30)
        assert _format_datetime_fr_hm(dt) == "15/03/2024 14:30"

    def test_none(self):
        from src.ui.pages.teammates_helpers import _format_datetime_fr_hm

        assert _format_datetime_fr_hm(None) == "-"

    def test_non_datetime(self):
        from src.ui.pages.teammates_helpers import _format_datetime_fr_hm

        result = _format_datetime_fr_hm("not a date")
        assert isinstance(result, str)


class TestFormatScoreLabel:
    """Tests redirigés vers formatting.format_score_label (canonical)."""

    def test_normal(self):
        from src.ui.formatting import format_score_label

        assert format_score_label(50, 25) == "50 - 25"

    def test_floats(self):
        from src.ui.formatting import format_score_label

        assert format_score_label(50.7, 25.3) == "51 - 25"

    def test_none_values(self):
        from src.ui.formatting import format_score_label

        assert format_score_label(None, None) == "-"

    def test_nan_values(self):
        from src.ui.formatting import format_score_label

        assert format_score_label(float("nan"), 10) == "-"


class TestAppUrl:
    def test_basic(self):
        from src.ui.pages.teammates_helpers import _app_url

        url = _app_url("Match", match_id="abc-123")
        assert "page=Match" in url
        assert "match_id=abc-123" in url
        assert url.startswith("/")

    def test_no_params(self):
        from src.ui.pages.teammates_helpers import _app_url

        url = _app_url("Overview")
        assert "page=Overview" in url


class TestNormalizeModeLabel:
    def test_none(self):
        from src.app.helpers import normalize_mode_label

        assert normalize_mode_label(None) is None

    def test_empty(self):
        from src.app.helpers import normalize_mode_label

        assert normalize_mode_label("") is None


# ============================================================================
# teammates_synergy.py — _compute_player_profile
# ============================================================================


class TestComputePlayerProfile:
    """Test _compute_player_profile avec repo mocké."""

    def test_no_personal_scores(self):
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = False
        df = pl.DataFrame({"match_id": ["m1"], "deaths": [3], "time_played_seconds": [600]})
        result = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", None)
        assert result is None

    def test_empty_personal_scores(self):
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = True
        repo.load_personal_score_awards_as_polars.return_value = pl.DataFrame(
            schema={"award_category": pl.Utf8, "award_score": pl.Int64}
        )
        df = pl.DataFrame({"match_id": ["m1"], "deaths": [3], "time_played_seconds": [600]})
        result = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", None)
        assert result is None

    def test_valid_profile(self):
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = True
        repo.load_personal_score_awards_as_polars.return_value = pl.DataFrame(
            {
                "award_category": ["kill", "assist", "objective"],
                "award_score": [1000, 400, 600],
            }
        )
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "deaths": [5],
                "time_played_seconds": [600],
                "pair_name": ["Arena:Slayer on Streets"],
            }
        )
        result = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", None)
        assert result is not None
        assert result["name"] == "Test"
        assert result["color"] == "#FF0000"
        assert "objectifs_norm" in result
        assert "combat_norm" in result

    def test_missing_columns(self):
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = True
        repo.load_personal_score_awards_as_polars.return_value = pl.DataFrame(
            {"award_category": ["kill"], "award_score": [500]}
        )
        # DataFrame without deaths or time_played_seconds
        df = pl.DataFrame({"match_id": ["m1"]})
        result = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", None)
        assert result is not None

    def test_objective_mode_ctf_uses_per_mode_threshold(self):
        """En mode CTF, le seuil objectifs doit être basé sur p90 CTF, pas combat."""
        from src.analysis.participation_radar import RADAR_THRESHOLDS_PER_MODE
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = True
        repo.load_personal_score_awards_as_polars.return_value = pl.DataFrame(
            {
                "award_category": ["kill", "assist", "objective"],
                "award_score": [500, 200, 750],
            }
        )
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "deaths": [3],
                "time_played_seconds": [600],
                "pair_name": ["Arena:CTF on Aquarius"],
            }
        )
        # Thresholds sans per_mode → fallback statique
        result_ctf = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", None)
        assert result_ctf is not None
        # 750 pts objectifs / p90_ctf(850) = ~88% — bien supérieur à l'ancien 750/9000 ≈ 8%
        ctf_p90 = RADAR_THRESHOLDS_PER_MODE["ctf"]
        expected_norm = min(1.0, 750.0 / ctf_p90)
        assert result_ctf["objectifs_norm"] == pytest.approx(expected_norm, rel=0.05)

    def test_per_mode_passed_in_thresholds_is_consumed(self):
        """Si thresholds contient 'per_mode', il est utilisé et non propagé vers compute_participation_profile."""
        from src.ui.pages.teammates_synergy import _compute_player_profile

        repo = MagicMock()
        repo.has_personal_score_awards.return_value = True
        repo.load_personal_score_awards_as_polars.return_value = pl.DataFrame(
            {"award_category": ["objective"], "award_score": [600]}
        )
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "deaths": [2],
                "time_played_seconds": [600],
                "pair_name": ["Arena:CTF on Streets"],
            }
        )
        custom_thresholds = {
            "objectifs": 999.0, "combat": 3000.0, "support": 800.0,
            "score": 4000.0, "impact_pts_per_min": 250.0,
            "survie_deaths_per_min_ref": 2.0, "survie_avg_life_ref_seconds": 90.0,
            "per_mode": {"ctf": 600.0, "slayer": 3000.0, "other": 800.0},
        }
        result = _compute_player_profile(repo, df, ["m1"], "Test", "#FF0000", custom_thresholds)
        assert result is not None
        # 600 pts / per_mode["ctf"]=600 → norm doit être ~1.0
        assert result["objectifs_norm"] == pytest.approx(1.0, rel=0.05)


class TestRenderRadarDisplay:
    """Test _render_radar_display avec mock Streamlit."""

    def test_empty_profiles(self, mock_st):
        from src.ui.pages import teammates_synergy as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        mod._render_radar_display([])
        ms.calls["info"].assert_called()

    def test_with_profiles(self, mock_st):
        from src.ui.pages import teammates_synergy as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        profiles = [
            {
                "name": "Player",
                "color": "#FF0000",
                "objectifs_norm": 0.5,
                "combat_norm": 0.6,
                "support_norm": 0.3,
                "score_norm": 0.7,
                "impact_norm": 0.4,
                "survie_norm": 0.5,
                "objectifs_raw": 500,
                "combat_raw": 600,
                "support_raw": 300,
                "score_raw": 1400,
                "impact_raw": 140.0,
                "survie_raw": 0.5,
            }
        ]

        with patch.object(mod, "create_participation_profile_radar", return_value=MagicMock()):
            mod._render_radar_display(profiles)
        ms.calls["subheader"].assert_called()
