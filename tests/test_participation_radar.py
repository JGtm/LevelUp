"""Tests pour src/visualization/participation_radar.py."""

from __future__ import annotations

import polars as pl
import pytest


# =============================================================================
# Tests get_mode_family
# =============================================================================


class TestGetModeFamily:
    """Tests exhaustifs de la détection de famille de mode."""

    def test_ctf_en(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:CTF on Aquarius") == "ctf"

    def test_ctf_btb(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("BTB:CTF on Fragmentation") == "ctf"

    def test_capture_flag_long(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Capture the Flag on Recharge") == "ctf"

    def test_drapeau_fr(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arène:Drapeau sur Aquarius") == "ctf"

    def test_oddball_en(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Oddball on Catalyst") == "oddball"

    def test_balle_fr(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arène:Balle sur Catalyst") == "oddball"

    def test_strongholds(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Strongholds on Streets") == "strongholds"

    def test_zones_fr(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arène:Zones sur Recharge") == "strongholds"

    def test_total_control(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Total Control on Aquarius") == "strongholds"

    def test_koth(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:King of the Hill on Streets") == "koth"

    def test_koth_abbrev(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:KOTH on Recharge") == "koth"

    def test_stockpile(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Stockpile on Scarr") == "stockpile"

    def test_extraction(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Extraction on Recharge") == "extraction"

    def test_land_grab(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Land Grab on Aquarius") == "land_grab"

    def test_fiesta(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Fiesta on Aquarius") == "fiesta"

    def test_slayer(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Slayer on Recharge") == "slayer"

    def test_ranked_slayer(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Ranked:Slayer on Streets") == "slayer"

    def test_none_returns_other(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family(None) == "other"

    def test_empty_string_returns_other(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("") == "other"

    def test_unknown_mode_returns_other(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("Arena:Infection on Aquarius") == "other"

    def test_case_insensitive(self) -> None:
        from src.analysis.participation_radar import get_mode_family
        assert get_mode_family("ARENA:CTF ON AQUARIUS") == "ctf"

    def test_global_thresholds_fallback_has_no_per_mode(self) -> None:
        """RADAR_THRESHOLDS (fallback sans DB) ne doit pas contenir 'per_mode'."""
        from src.analysis.participation_radar import RADAR_THRESHOLDS
        assert "per_mode" not in RADAR_THRESHOLDS

    def test_all_families_in_per_mode_constant(self) -> None:
        """Toutes les familles retournées par get_mode_family existent dans RADAR_THRESHOLDS_PER_MODE."""
        from src.analysis.participation_radar import RADAR_THRESHOLDS_PER_MODE, get_mode_family
        pair_names = [
            "Arena:CTF on A", "Arena:Oddball on A", "Arena:Strongholds on A",
            "Arena:King of the Hill on A", "Arena:Stockpile on A",
            "Arena:Extraction on A", "Arena:Land Grab on A",
            "Arena:Fiesta on A", "Arena:Slayer on A", None,
        ]
        for pn in pair_names:
            family = get_mode_family(pn)
            assert family in RADAR_THRESHOLDS_PER_MODE, f"{pn!r} -> {family!r} absent"

# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def empty_awards() -> pl.DataFrame:
    """DataFrame PersonalScores vide."""
    return pl.DataFrame(schema={"award_category": pl.Utf8, "award_score": pl.Int64})


@pytest.fixture
def awards_mode_objective() -> pl.DataFrame:
    """PersonalScores typique mode objectif (CTF, etc.)."""
    return pl.DataFrame(
        {
            "award_category": ["kill", "assist", "objective", "vehicle", "penalty"],
            "award_score": [700, 150, 400, 50, -100],
        }
    )


@pytest.fixture
def awards_mode_slayer() -> pl.DataFrame:
    """PersonalScores typique mode Slayer (frags = objectif)."""
    return pl.DataFrame(
        {
            "award_category": ["kill", "assist", "vehicle", "penalty"],
            "award_score": [1200, 200, 0, -50],
        }
    )


@pytest.fixture
def match_row_10min() -> dict:
    """Ligne match_stats : 10 min, 5 morts."""
    return {
        "deaths": 5,
        "time_played_seconds": 600,
        "pair_name": "Arena:Slayer on Aquarius",
    }


# =============================================================================
# Tests compute_participation_profile
# =============================================================================


class TestComputeParticipationProfile:
    """Tests pour compute_participation_profile."""

    def test_empty_awards_returns_default_profile(self, empty_awards: pl.DataFrame) -> None:
        from src.visualization.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
        )

        profile = compute_participation_profile(
            empty_awards, options=ProfileOptions(name="Vide", color="#000")
        )

        assert profile["name"] == "Vide"
        assert profile["color"] == "#000"
        assert profile["objectifs_raw"] == 0
        assert profile["combat_raw"] == 0
        assert profile["support_raw"] == 0
        assert profile["score_raw"] == 0

    def test_mode_objective_uses_objective_score(self, awards_mode_objective: pl.DataFrame) -> None:
        from src.visualization.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
        )

        profile = compute_participation_profile(
            awards_mode_objective,
            match_row=None,
            options=ProfileOptions(mode_is_objective=True),
        )

        assert profile["objectifs_raw"] == 400
        assert profile["combat_raw"] == 700
        assert profile["support_raw"] == 150
        assert profile["score_raw"] == 1200

    def test_mode_slayer_uses_kill_score_as_objectifs(
        self, awards_mode_slayer: pl.DataFrame
    ) -> None:
        from src.visualization.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
        )

        profile = compute_participation_profile(
            awards_mode_slayer,
            match_row=None,
            options=ProfileOptions(mode_is_objective=False),
        )

        assert profile["objectifs_raw"] == 1200
        assert profile["combat_raw"] == 1200

    def test_detect_mode_from_pair_name_slayer(self, awards_mode_slayer: pl.DataFrame) -> None:
        from src.visualization.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
        )

        profile = compute_participation_profile(
            awards_mode_slayer,
            options=ProfileOptions(pair_name="Arena:Slayer on Aquarius"),
        )

        assert profile["objectifs_raw"] == 1200

    def test_detect_mode_from_pair_name_objective(
        self, awards_mode_objective: pl.DataFrame
    ) -> None:
        from src.visualization.participation_radar import (
            ProfileOptions,
            compute_participation_profile,
        )

        profile = compute_participation_profile(
            awards_mode_objective,
            options=ProfileOptions(pair_name="BTB:CTF on Fragmentation"),
        )

        assert profile["objectifs_raw"] == 400

    def test_impact_and_survie_with_match_row(
        self, awards_mode_objective: pl.DataFrame, match_row_10min: dict
    ) -> None:
        from src.visualization.participation_radar import compute_participation_profile

        profile = compute_participation_profile(
            awards_mode_objective,
            match_row=match_row_10min,
        )

        # 1200 pts positifs / 10 min = 120 pts/min
        assert profile["impact_raw"] == pytest.approx(120.0, rel=0.01)
        # 5 morts / 10 min = 0.5 mort/min, ref 2.0 → deaths_component = 1 - 0.5/2 = 0.75
        # pas d'avg_life → survie = deaths_component = 0.75
        assert profile["survie_raw"] == pytest.approx(0.75, rel=0.01)

    def test_normalized_values_in_range(
        self, awards_mode_objective: pl.DataFrame, match_row_10min: dict
    ) -> None:
        from src.visualization.participation_radar import compute_participation_profile

        profile = compute_participation_profile(
            awards_mode_objective,
            match_row=match_row_10min,
        )

        for key in (
            "objectifs_norm",
            "combat_norm",
            "support_norm",
            "score_norm",
            "impact_norm",
            "survie_norm",
        ):
            assert 0 <= profile[key] <= 1.1, f"{key} hors plage: {profile[key]}"


# =============================================================================
# Tests _compute_shared_match_ids (régression : "Madina disparaît sur vieilles sessions")
# =============================================================================


def _make_df(match_ids: list[str]) -> "pl.DataFrame":
    """DataFrame minimal avec colonne match_id."""
    import polars as pl

    if not match_ids:
        return pl.DataFrame({"match_id": pl.Series([], dtype=pl.Utf8)})
    return pl.DataFrame({"match_id": match_ids})


class TestComputeSharedMatchIds:
    """Tests pour _compute_shared_match_ids — logique d'intersection robuste."""

    def test_two_players_with_data(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2", "m3"])
        f1 = _make_df(["m1", "m2"])
        result = _compute_shared_match_ids(me, f1, None, None)
        assert set(result) == {"m1", "m2"}

    def test_f3_empty_does_not_collapse_shared(self) -> None:
        """Bug : f3 DataFrame vide faisait shared=set() → radar disparaissait."""
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2", "m3"])
        f1 = _make_df(["m1", "m2", "m3"])
        f2 = _make_df(["m1", "m2", "m3"])
        f3_empty = _make_df([])  # Madina sans données pour cette session
        result = _compute_shared_match_ids(me, f1, f2, f3_empty)
        # L'intersection doit rester non vide — f3 vide est ignoré
        assert set(result) == {"m1", "m2", "m3"}

    def test_f2_empty_does_not_collapse_shared(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2"])
        f1 = _make_df(["m1", "m2"])
        f2_empty = _make_df([])
        result = _compute_shared_match_ids(me, f1, f2_empty, None)
        assert set(result) == {"m1", "m2"}

    def test_all_players_with_data_intersection(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2", "m3"])
        f1 = _make_df(["m1", "m2", "m4"])
        f2 = _make_df(["m1", "m3", "m4"])
        f3 = _make_df(["m1", "m2"])
        result = _compute_shared_match_ids(me, f1, f2, f3)
        # Seul m1 est dans les 4
        assert set(result) == {"m1"}

    def test_f3_none_ignored(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2"])
        f1 = _make_df(["m1", "m2"])
        result = _compute_shared_match_ids(me, f1, None, None)
        assert set(result) == {"m1", "m2"}

    def test_me_empty_returns_empty(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df([])
        f1 = _make_df(["m1", "m2"])
        result = _compute_shared_match_ids(me, f1, None, None)
        assert result == []

    def test_no_overlap_returns_empty(self) -> None:
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        me = _make_df(["m1", "m2"])
        f1 = _make_df(["m3", "m4"])
        result = _compute_shared_match_ids(me, f1, None, None)
        assert result == []

    def test_session_change_produces_different_result(self) -> None:
        """Le changement de session doit produire des match_ids différents."""
        from src.ui.pages.teammates_synergy import _compute_shared_match_ids

        # Session mars
        me_mars = _make_df(["m1", "m2", "m3"])
        f1_mars = _make_df(["m1", "m2", "m3"])
        result_mars = set(_compute_shared_match_ids(me_mars, f1_mars, None, None))

        # Session avril
        me_avril = _make_df(["m4", "m5"])
        f1_avril = _make_df(["m4", "m5"])
        result_avril = set(_compute_shared_match_ids(me_avril, f1_avril, None, None))

        assert result_mars != result_avril
        assert result_mars == {"m1", "m2", "m3"}
        assert result_avril == {"m4", "m5"}
