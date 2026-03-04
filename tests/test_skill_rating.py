"""Tests unitaires pour le LUSR (LevelUp Skill Rank).

Couvre :
- Configuration (tiers, labels, images, playlist groups)
- Algorithme TrueSkill 2 (update, decay, composite score)
- Estimation μ individuel depuis kills_expected
- Calcul batch séquentiel
"""

from __future__ import annotations

import math
from datetime import datetime, timedelta, timezone
from pathlib import Path

import polars as pl
import pytest

from src.analysis.skill_rating import (
    PlayerState,
    apply_inactivity_decay,
    compute_composite_score,
    compute_enemy_strength,
    compute_skill_ratings_batch,
    estimate_individual_mu,
    trueskill_update,
)
from src.analysis.skill_rating_config import (
    INITIAL_MU,
    INITIAL_SIGMA,
    MIN_SIGMA,
    format_tier_label,
    get_playlist_group,
    get_rank_image_filename,
    get_rank_image_path,
    get_sub_tier_start,
    get_tier_color,
    get_tier_for_rating,
)

# =============================================================================
# Fixtures helpers
# =============================================================================

_NOW = datetime(2025, 1, 1, 12, 0, 0, tzinfo=timezone.utc)


def _make_row(  # noqa: PLR0913
    *,
    kills: float = 10.0,
    deaths: float = 5.0,
    kills_expected: float = 10.0,
    deaths_expected: float = 5.0,
    outcome: int = 2,  # WIN par défaut
    damage_dealt: float = 5000.0,
    damage_taken: float = 3000.0,
    accuracy: float = 30.0,
    team_id: int = 0,
    playlist_name: str = "Quick Play",
    pair_name: str = "Arena:Slayer on Aquarius",
    is_ranked: bool = False,
    match_id: str = "match_001",
    start_time: datetime | None = None,
) -> dict:
    return {
        "match_id": match_id,
        "start_time": start_time or _NOW,
        "kills": kills,
        "deaths": deaths,
        "kills_expected": kills_expected,
        "deaths_expected": deaths_expected,
        "outcome": outcome,
        "damage_dealt": damage_dealt,
        "damage_taken": damage_taken,
        "accuracy": accuracy,
        "team_id": team_id,
        "playlist_name": playlist_name,
        "pair_name": pair_name,
        "is_ranked": is_ranked,
    }


def _make_df_matches(*rows: dict) -> pl.DataFrame:
    if not rows:
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Datetime("us", "UTC"),
                "kills": pl.Float64,
                "deaths": pl.Float64,
                "kills_expected": pl.Float64,
                "deaths_expected": pl.Float64,
                "outcome": pl.Int32,
                "damage_dealt": pl.Float64,
                "damage_taken": pl.Float64,
                "accuracy": pl.Float64,
                "team_id": pl.Int32,
                "playlist_name": pl.Utf8,
                "pair_name": pl.Utf8,
                "is_ranked": pl.Boolean,
            }
        )
    return pl.DataFrame(list(rows))


def _make_df_participants(*rows: dict) -> pl.DataFrame:
    if not rows:
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "xuid": pl.Utf8,
                "team_id": pl.Int32,
                "kills_expected": pl.Float64,
                "deaths_expected": pl.Float64,
            }
        )
    return pl.DataFrame(list(rows))


# =============================================================================
# Tests config — tiers
# =============================================================================


class TestTierMapping:
    def test_bronze_start(self):
        tier, sub = get_tier_for_rating(1000.0)
        assert tier is not None
        assert tier.name == "Bronze"
        assert sub == 1

    def test_bronze_end(self):
        tier, sub = get_tier_for_rating(1199.9)
        assert tier is not None
        assert tier.name == "Bronze"
        assert sub == 6

    def test_gold_iii(self):
        # Or III : 1400 + 2 × (200/6) ≈ 1467
        tier, sub = get_tier_for_rating(1470.0)
        assert tier is not None
        assert tier.name == "Gold"
        assert sub == 3

    def test_onyx(self):
        tier, sub = get_tier_for_rating(2000.0)
        assert tier is not None
        assert tier.name == "Onyx"
        assert sub == 0

    def test_above_onyx(self):
        tier, sub = get_tier_for_rating(2500.0)
        assert tier is not None
        assert tier.name == "Onyx"

    def test_below_bronze(self):
        tier, sub = get_tier_for_rating(500.0)
        assert tier is None
        assert sub == 0


class TestFormatTierLabel:
    def test_bronze_i(self):
        label = format_tier_label(1000.0)
        assert label == "Bronze I"

    def test_or_iii(self):
        label = format_tier_label(1470.0)
        assert "Or" in label
        assert "III" in label

    def test_onyx(self):
        label = format_tier_label(2100.0)
        assert label == "Onyx"

    def test_unranked(self):
        label = format_tier_label(500.0)
        assert label == "Non classé"

    def test_silver_vi(self):
        label = format_tier_label(1399.0)
        assert "Argent" in label
        assert "VI" in label


class TestRankImagePath:
    def test_gold3_filename(self):
        filename = get_rank_image_filename("Gold", 3)
        assert filename == "120px-HINF-CSR_Gold3.png"

    def test_onyx_filename(self):
        filename = get_rank_image_filename("Onyx", 0)
        assert filename == "120px-HINF-CSR_Onyx.png"

    def test_path_format(self):
        path = get_rank_image_path(1470.0)
        assert path is not None
        assert path.startswith("static/ranks/")
        assert path.endswith(".png")

    def test_path_unranked_returns_none(self):
        path = get_rank_image_path(500.0)
        assert path is None

    def test_rank_images_exist(self):
        """Vérifie que les fichiers PNG référencés existent dans static/ranks/."""
        project_root = Path(__file__).parent.parent
        test_ratings = [1000.0, 1250.0, 1450.0, 1670.0, 1850.0, 2100.0]
        for rating in test_ratings:
            rel_path = get_rank_image_path(rating)
            if rel_path:
                full_path = project_root / rel_path
                assert full_path.exists(), f"Image manquante : {full_path}"


class TestTierColor:
    def test_bronze_color(self):
        color = get_tier_color(1050.0)
        assert color.startswith("#")

    def test_onyx_color(self):
        color = get_tier_color(2200.0)
        assert color.startswith("#")


class TestSubTierStart:
    def test_bronze_i_start(self):
        start = get_sub_tier_start(1000.0)
        assert start == pytest.approx(1000.0, abs=1)

    def test_gold_iii_start(self):
        # Or III démarre à 1400 + 2 × (200/6) ≈ 1466.7
        start = get_sub_tier_start(1470.0)
        assert 1460.0 <= start <= 1470.0


# =============================================================================
# Tests config — playlist groups
# =============================================================================


class TestPlaylistGroup:
    def test_ranked_by_pair_prefix(self):
        assert get_playlist_group("Ranked Arena", "Ranked:Slayer on Aquarius") == "ranked"

    def test_arena_by_pair_prefix(self):
        assert get_playlist_group("Quick Play", "Arena:Slayer on Aquarius") == "arena"

    def test_btb_by_pair_prefix(self):
        assert get_playlist_group("Big Team Battle", "BTB:Slayer on Deadlock") == "btb"

    def test_tactical_by_name(self):
        assert get_playlist_group("Team Snipers", None) == "tactical"

    def test_tactical_by_pair_prefix(self):
        assert get_playlist_group("Tactical Slayer", "Tactical:SWAT on Aquarius") == "tactical"

    def test_social_by_name(self):
        assert get_playlist_group("Super Fiesta", None) == "social"

    def test_fun_by_name(self):
        assert get_playlist_group("Infection", None) == "fun"

    def test_fun_by_pair_prefix(self):
        assert get_playlist_group("Grifball", "Grifball:Standard") == "fun"

    def test_unknown_defaults_to_social(self):
        result = get_playlist_group("Unknown Playlist", "Unknown:Mode")
        assert result == "social"  # DEFAULT_PLAYLIST_GROUP


# =============================================================================
# Tests algorithme — état initial
# =============================================================================


class TestInitialState:
    def test_initial_mu(self):
        state = PlayerState()
        assert state.mu == pytest.approx(INITIAL_MU)

    def test_initial_sigma(self):
        state = PlayerState()
        assert state.sigma == pytest.approx(INITIAL_SIGMA)

    def test_initial_match_count(self):
        state = PlayerState()
        assert state.match_count == 0


# =============================================================================
# Tests algorithme — score composite
# =============================================================================


class TestCompositeScore:
    def test_win_above_half(self):
        row = _make_row(outcome=2, kills=12, kills_expected=10, deaths=3, deaths_expected=5)
        score = compute_composite_score(row, None, None, None)
        assert score > 0.5

    def test_loss_below_half(self):
        row = _make_row(outcome=3, kills=5, kills_expected=10, deaths=10, deaths_expected=5)
        score = compute_composite_score(row, None, None, None)
        assert score < 0.5

    def test_expected_performance_near_half(self):
        row = _make_row(
            outcome=1,
            kills=10,
            kills_expected=10,
            deaths=5,
            deaths_expected=5,
            damage_dealt=5000,
            damage_taken=5000,
        )
        score = compute_composite_score(row, 30.0, None, None)
        assert 0.3 <= score <= 0.7

    def test_graceful_degradation_missing_kills_expected(self):
        row = _make_row()
        row["kills_expected"] = None
        row["deaths_expected"] = None
        # Doit quand même retourner un score valide (non NaN, entre 0 et 1)
        score = compute_composite_score(row, None, None, None)
        assert 0.0 <= score <= 1.0
        assert math.isfinite(score)

    def test_graceful_degradation_all_none(self):
        row = {
            "match_id": "x",
            "kills": None,
            "deaths": None,
            "kills_expected": None,
            "deaths_expected": None,
            "outcome": None,
            "damage_dealt": None,
            "damage_taken": None,
            "accuracy": None,
        }
        score = compute_composite_score(row, None, None, None)
        assert score == pytest.approx(0.5)

    def test_dnf_low_score(self):
        # Un DNF doit scorer moins bien que la même partie terminée en victoire.
        # On compare relatif (pas absolu) pour être indépendant des poids calibrés.
        row_win = _make_row(outcome=2)
        row_dnf = _make_row(outcome=4)
        score_win = compute_composite_score(row_win, None, None, None)
        score_dnf = compute_composite_score(row_dnf, None, None, None)
        assert score_dnf < score_win

    def test_carry_adjustment_reduces_win_bonus(self):
        """Si mes coéquipiers ont ke >> ma ke, une victoire vaut moins."""
        row_normal = _make_row(outcome=2, kills=10, kills_expected=10)
        row_carry = _make_row(outcome=2, kills=10, kills_expected=10)
        score_normal = compute_composite_score(row_normal, None, 10.0, None)
        score_carry = compute_composite_score(
            row_carry, None, 20.0, None
        )  # coéquipiers 2× plus forts
        # Avec carry, le score composite doit être plus faible (contribution moins certaine)
        assert score_carry <= score_normal + 0.05  # marge de tolérance


# =============================================================================
# Tests algorithme — TrueSkill update
# =============================================================================


class TestTrueSkillUpdate:
    def test_win_increases_mu(self):
        new_mu, new_sigma = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU,
            INITIAL_SIGMA,  # adversaire de même niveau
            actual_score=0.8,  # sur-performance
        )
        assert new_mu > INITIAL_MU

    def test_loss_decreases_mu(self):
        new_mu, new_sigma = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU,
            INITIAL_SIGMA,
            actual_score=0.2,  # sous-performance
        )
        assert new_mu < INITIAL_MU

    def test_draw_neutral(self):
        new_mu, _ = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU,
            INITIAL_SIGMA,
            actual_score=0.5,  # performance attendue
        )
        # Un score neutre contre un adversaire de même niveau → mu ≈ stable
        assert abs(new_mu - INITIAL_MU) < 50.0

    def test_sigma_bounded(self):
        new_mu, new_sigma = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU,
            INITIAL_SIGMA,
            actual_score=0.7,
        )
        assert MIN_SIGMA <= new_sigma <= INITIAL_SIGMA + 50

    def test_weight_factor_ranked_bigger_than_fun(self):
        _, _, delta_ranked = _update_delta(actual_score=0.8, weight=1.0)
        _, _, delta_fun = _update_delta(actual_score=0.8, weight=0.2)
        assert abs(delta_ranked) > abs(delta_fun)

    def test_same_composite_same_delta_regardless_of_opponent(self):
        # Avec la mise à jour Elo-style, delta_mu = K_ELO × (composite - 0.5) × wf.
        # mu_opp n'affecte PAS mu (seulement sigma via c²).
        # → Même composite contre adversaire fort ou faible → même delta mu.
        mu_after_strong, _ = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU + 300,
            INITIAL_SIGMA,  # adversaire plus fort
            actual_score=0.75,
        )
        mu_after_weak, _ = trueskill_update(
            INITIAL_MU,
            INITIAL_SIGMA,
            INITIAL_MU - 300,
            INITIAL_SIGMA,  # adversaire plus faible
            actual_score=0.75,
        )
        assert mu_after_strong == pytest.approx(mu_after_weak, abs=0.01)

    def test_mu_never_below_min(self):
        # Pire cas possible
        mu = 300.0
        sigma = INITIAL_SIGMA
        for _ in range(20):
            mu, sigma = trueskill_update(mu, sigma, INITIAL_MU + 500, INITIAL_SIGMA, 0.0)
        assert mu >= 200.0  # MIN_RATING


def _update_delta(actual_score: float, weight: float) -> tuple[float, float, float]:
    new_mu, new_sigma = trueskill_update(
        INITIAL_MU,
        INITIAL_SIGMA,
        INITIAL_MU,
        INITIAL_SIGMA,
        actual_score=actual_score,
        weight_factor=weight,
    )
    return new_mu, new_sigma, new_mu - INITIAL_MU


# =============================================================================
# Tests algorithme — inactivité
# =============================================================================


class TestInactivityDecay:
    def test_no_decay_short_inactivity(self):
        sigma = apply_inactivity_decay(100.0, days_inactive=0.5)
        assert sigma == pytest.approx(100.0)

    def test_decay_increases_sigma(self):
        sigma_after = apply_inactivity_decay(100.0, days_inactive=10.0)
        assert sigma_after > 100.0

    def test_decay_capped_at_max(self):
        from src.analysis.skill_rating_config import MAX_SIGMA

        sigma_after = apply_inactivity_decay(100.0, days_inactive=365.0)
        assert sigma_after <= MAX_SIGMA

    def test_decay_not_below_min(self):
        sigma_after = apply_inactivity_decay(MIN_SIGMA, days_inactive=5.0)
        assert sigma_after >= MIN_SIGMA


# =============================================================================
# Tests algorithme — estimation μ individuel
# =============================================================================


class TestEstimateIndividualMu:
    def test_above_average_ke_gives_higher_mu(self):
        mu_high = estimate_individual_mu(
            12.0, match_avg_ke=10.0, match_std_ke=2.0, base_mu=INITIAL_MU
        )
        mu_avg = estimate_individual_mu(
            10.0, match_avg_ke=10.0, match_std_ke=2.0, base_mu=INITIAL_MU
        )
        assert mu_high > mu_avg

    def test_below_average_ke_gives_lower_mu(self):
        mu_low = estimate_individual_mu(
            8.0, match_avg_ke=10.0, match_std_ke=2.0, base_mu=INITIAL_MU
        )
        assert mu_low < INITIAL_MU

    def test_zero_std_returns_base(self):
        mu = estimate_individual_mu(10.0, match_avg_ke=10.0, match_std_ke=0.0, base_mu=INITIAL_MU)
        assert mu == pytest.approx(INITIAL_MU)


class TestEnemyStrength:
    def test_no_enemies_returns_defaults(self):
        mu_opp, sigma_opp = compute_enemy_strength([], 10.0, 2.0)
        assert mu_opp == pytest.approx(INITIAL_MU)

    def test_strong_enemies_higher_mu(self):
        enemies = [{"kills_expected": 15.0}, {"kills_expected": 14.0}]
        mu_opp, _ = compute_enemy_strength(enemies, 10.0, 2.0)
        assert mu_opp > INITIAL_MU

    def test_weak_enemies_lower_mu(self):
        enemies = [{"kills_expected": 5.0}, {"kills_expected": 6.0}]
        mu_opp, _ = compute_enemy_strength(enemies, 10.0, 2.0)
        assert mu_opp < INITIAL_MU

    def test_no_ke_data_returns_defaults(self):
        enemies = [{"kills_expected": None}, {"kills_expected": None}]
        mu_opp, sigma_opp = compute_enemy_strength(enemies, 10.0, 2.0)
        assert mu_opp == pytest.approx(INITIAL_MU)


# =============================================================================
# Tests batch
# =============================================================================


class TestComputeSkillRatingsBatch:
    def test_empty_df_returns_empty(self):
        df_m = _make_df_matches()
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(df_m, df_p)
        assert result.is_empty()

    def test_single_match_returns_one_row(self):
        df_m = _make_df_matches(_make_row(match_id="m1"))
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(df_m, df_p)
        assert result.height == 1
        assert result["match_id"][0] == "m1"

    def test_sequential_order_matters(self):
        """L'ordre d'accumulation de l'historique affecte le rating via accuracy_delta.

        Avec 6 matchs : l'historique de précision (accuracy_history) est actif
        à partir du 6ème match (MIN_MATCHES_FOR_ACCURACY_DELTA=5). Si l'accuracy
        croît, le 6ème match est au-dessus de la moyenne → score bonus. Si elle
        décroît, le 6ème est en dessous → score malus. Les deux séquences donnent
        donc des ratings finals différents, vérifiant le traitement séquentiel.
        """
        # accuracy croissante : 10, 20, 30, 40, 50, 60
        # → historique avg au match 6 = mean([10..50]) = 30 → acc=60 > moy → bonus
        rows_rising = [
            _make_row(
                match_id=f"m{i}",
                outcome=2,
                accuracy=float(10 + i * 10),
                start_time=_NOW + timedelta(hours=i),
            )
            for i in range(6)
        ]
        # accuracy décroissante : 60, 50, 40, 30, 20, 10
        # → historique avg au match 6 = mean([60..20]) = 40 → acc=10 < moy → malus
        rows_falling = [
            _make_row(
                match_id=f"m{i}",
                outcome=2,
                accuracy=float(60 - i * 10),
                start_time=_NOW + timedelta(hours=i),
            )
            for i in range(6)
        ]

        df_p = _make_df_participants()
        res_rising = compute_skill_ratings_batch(_make_df_matches(*rows_rising), df_p)
        res_falling = compute_skill_ratings_batch(_make_df_matches(*rows_falling), df_p)

        final_rising = res_rising["rating_value"][-1]
        final_falling = res_falling["rating_value"][-1]
        # Accuracy croissante → dernier composite plus élevé → rating final plus haut
        assert final_rising != pytest.approx(final_falling, abs=1.0)

    def test_wins_increase_rating_over_time(self):
        """Enchaîner des victoires doit faire monter le rating."""
        rows = [
            _make_row(
                match_id=f"m{i}",
                outcome=2,
                kills=15,
                kills_expected=10,
                deaths=2,
                deaths_expected=5,
                start_time=_NOW + timedelta(hours=i),
            )
            for i in range(5)
        ]
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(_make_df_matches(*rows), df_p)
        final_rating = result["rating_value"][-1]
        assert final_rating > INITIAL_MU

    def test_losses_decrease_rating_over_time(self):
        """Enchaîner des défaites doit faire baisser le rating."""
        rows = [
            _make_row(
                match_id=f"m{i}",
                outcome=3,
                kills=3,
                kills_expected=10,
                deaths=10,
                deaths_expected=5,
                start_time=_NOW + timedelta(hours=i),
            )
            for i in range(5)
        ]
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(_make_df_matches(*rows), df_p)
        final_rating = result["rating_value"][-1]
        assert final_rating < INITIAL_MU

    def test_playlist_group_assigned_correctly(self):
        row = _make_row(pair_name="Ranked:Slayer on Aquarius", playlist_name="Ranked Arena")
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(_make_df_matches(row), df_p)
        assert result["playlist_group"][0] == "ranked"

    def test_output_columns(self):
        df_m = _make_df_matches(_make_row())
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(df_m, df_p)
        assert "match_id" in result.columns
        assert "rating_value" in result.columns
        assert "rating_deviation" in result.columns
        assert "playlist_group" in result.columns

    def test_rating_values_finite(self):
        rows = [
            _make_row(match_id=f"m{i}", start_time=_NOW + timedelta(hours=i)) for i in range(10)
        ]
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(_make_df_matches(*rows), df_p)
        for val in result["rating_value"].to_list():
            assert math.isfinite(val), f"rating_value non fini : {val}"

    def test_existing_states_is_used(self):
        """Avec des états initiaux différents par groupe, le rating de départ est différent."""
        # Le row par défaut utilise pair_name="Arena:Slayer on Aquarius" → groupe "arena"
        row = _make_row(outcome=2)
        df_p = _make_df_participants()

        state_high = PlayerState(mu=2000.0, sigma=100.0)
        state_low = PlayerState(mu=1000.0, sigma=100.0)

        res_high = compute_skill_ratings_batch(
            _make_df_matches(row),
            df_p,
            existing_states={"arena": state_high},
        )
        res_low = compute_skill_ratings_batch(
            _make_df_matches(row),
            df_p,
            existing_states={"arena": state_low},
        )

        assert res_high["rating_value"][0] != pytest.approx(res_low["rating_value"][0], abs=50)

    def test_independent_states_per_group(self):
        """Les groupes différents ont des ratings indépendants."""
        t1 = _NOW
        t2 = _NOW + timedelta(hours=1)
        # match en ranked
        row_ranked = _make_row(
            match_id="r1",
            outcome=2,
            start_time=t1,
            pair_name="Ranked:Slayer on Aquarius",
            playlist_name="Ranked Arena",
        )
        # match en arena
        row_arena = _make_row(
            match_id="a1",
            outcome=3,
            start_time=t2,
            pair_name="Arena:Slayer on Aquarius",
            playlist_name="Quick Play",
        )
        df_p = _make_df_participants()
        result = compute_skill_ratings_batch(_make_df_matches(row_ranked, row_arena), df_p)
        ranked_row = result.filter(pl.col("match_id") == "r1")
        arena_row = result.filter(pl.col("match_id") == "a1")
        # ranked: victoire → monte depuis INITIAL_MU
        assert ranked_row["rating_value"][0] > 1400.0
        # arena: défaite → descend depuis INITIAL_MU (état indépendant)
        assert arena_row["rating_value"][0] < 1600.0
        # Les ratings doivent être différents (états indépendants)
        assert ranked_row["rating_value"][0] != pytest.approx(arena_row["rating_value"][0], abs=1.0)

    def test_with_participants_data(self):
        """Un score composite plus élevé (surperformance kills) donne un delta mu plus grand.

        Avec la mise à jour Elo-style, le delta mu dépend uniquement du composite
        score, pas de mu_opp. On teste donc que surpasser kills_expected donne
        un gain supérieur à simplement les atteindre.
        """
        # Cas 1 : surperformance (kills >> kills_expected)
        row_good = _make_row(
            match_id="m1",
            outcome=2,
            team_id=0,
            kills=15,
            kills_expected=10,
            deaths=3,
            deaths_expected=5,
        )
        # Cas 2 : performance attendue (kills = kills_expected)
        row_avg = _make_row(
            match_id="m1",
            outcome=2,
            team_id=0,
            kills=10,
            kills_expected=10,
            deaths=5,
            deaths_expected=5,
        )

        df_p = _make_df_participants(
            {
                "match_id": "m1",
                "xuid": "e1",
                "team_id": 1,
                "kills_expected": 10.0,
                "deaths_expected": 5.0,
            },
            {
                "match_id": "m1",
                "xuid": "e2",
                "team_id": 1,
                "kills_expected": 10.0,
                "deaths_expected": 5.0,
            },
        )

        res_good = compute_skill_ratings_batch(_make_df_matches(row_good), df_p)
        res_avg = compute_skill_ratings_batch(_make_df_matches(row_avg), df_p)

        # Surperformance → gain mu plus important
        gain_good = res_good["rating_value"][0] - INITIAL_MU
        gain_avg = res_avg["rating_value"][0] - INITIAL_MU
        assert gain_good > gain_avg
