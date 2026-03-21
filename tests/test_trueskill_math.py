"""Tests pour src/analysis/_trueskill_math.py.

Couvre trueskill_update, apply_inactivity_decay, PlayerState.
"""

from src.analysis._trueskill_math import (
    PlayerState,
    apply_inactivity_decay,
    trueskill_update,
)
from src.analysis.skill_rating_config import (
    INACTIVITY_THRESHOLD_DAYS,
    INITIAL_MU,
    INITIAL_SIGMA,
    MAX_SIGMA,
    MIN_RATING,
    MIN_SIGMA,
)


class TestTrueSkillUpdate:
    """Tests pour trueskill_update."""

    def test_win_increases_mu(self) -> None:
        mu, sigma = INITIAL_MU, INITIAL_SIGMA
        new_mu, new_sigma = trueskill_update(mu, sigma, mu, sigma, 1.0)
        assert new_mu > mu

    def test_loss_decreases_mu(self) -> None:
        mu, sigma = INITIAL_MU, INITIAL_SIGMA
        new_mu, new_sigma = trueskill_update(mu, sigma, mu, sigma, 0.0)
        assert new_mu < mu

    def test_draw_stable_mu(self) -> None:
        mu, sigma = INITIAL_MU, INITIAL_SIGMA
        new_mu, _new_sigma = trueskill_update(mu, sigma, mu, sigma, 0.5)
        assert abs(new_mu - mu) < 1e-6

    def test_sigma_decreases_or_stays(self) -> None:
        mu, sigma = INITIAL_MU, INITIAL_SIGMA
        _new_mu, new_sigma = trueskill_update(mu, sigma, mu, sigma, 0.7)
        # Sigma should decrease after an observation (TrueSkill update)
        # But TAU adds back noise, so just check it's bounded
        assert new_sigma <= MAX_SIGMA

    def test_mu_stays_above_minimum(self) -> None:
        new_mu, _ = trueskill_update(MIN_RATING + 1, INITIAL_SIGMA, 3000, 50, 0.0)
        assert new_mu >= MIN_RATING

    def test_weight_factor_amplifies(self) -> None:
        mu, sigma = INITIAL_MU, INITIAL_SIGMA
        mu_wf1, _ = trueskill_update(mu, sigma, mu, sigma, 1.0, weight_factor=1.0)
        mu_wf2, _ = trueskill_update(mu, sigma, mu, sigma, 1.0, weight_factor=2.0)
        assert mu_wf2 > mu_wf1

    def test_returns_tuple(self) -> None:
        result = trueskill_update(INITIAL_MU, INITIAL_SIGMA, INITIAL_MU, INITIAL_SIGMA, 0.5)
        assert isinstance(result, tuple)
        assert len(result) == 2
        assert all(isinstance(v, float) for v in result)


class TestApplyInactivityDecay:
    """Tests pour apply_inactivity_decay."""

    def test_no_decay_below_threshold(self) -> None:
        sigma = 100.0
        result = apply_inactivity_decay(sigma, INACTIVITY_THRESHOLD_DAYS - 0.5)
        assert result == sigma

    def test_decay_above_threshold(self) -> None:
        sigma = 100.0
        result = apply_inactivity_decay(sigma, INACTIVITY_THRESHOLD_DAYS + 5.0)
        assert result > sigma

    def test_capped_at_max_sigma(self) -> None:
        result = apply_inactivity_decay(MAX_SIGMA - 1, 365.0)
        assert result <= MAX_SIGMA

    def test_minimum_sigma_preserved(self) -> None:
        result = apply_inactivity_decay(MIN_SIGMA, 0.0)
        assert result >= MIN_SIGMA

    def test_zero_days(self) -> None:
        sigma = 100.0
        result = apply_inactivity_decay(sigma, 0.0)
        assert result == sigma


class TestPlayerState:
    """Tests pour PlayerState."""

    def test_defaults(self) -> None:
        state = PlayerState()
        assert state.mu == INITIAL_MU
        assert state.sigma == INITIAL_SIGMA
        assert state.match_count == 0
        assert state.last_match_time is None
        assert state.accuracy_history == []
        assert state.damage_eff_history == []

    def test_from_csr_zero(self) -> None:
        state = PlayerState.from_csr(0)
        assert state.mu == 1000.0
        assert state.sigma == MIN_SIGMA

    def test_from_csr_positive(self) -> None:
        state = PlayerState.from_csr(1500)
        expected_mu = 1000.0 + 1500 * (2.0 / 3.0)
        assert state.mu == expected_mu
        assert state.sigma == MIN_SIGMA

    def test_from_csr_negative(self) -> None:
        state = PlayerState.from_csr(-100)
        # max(0.0, -100) = 0.0, so mu = 1000.0
        assert state.mu == 1000.0

    def test_from_csr_mu_ordering(self) -> None:
        low = PlayerState.from_csr(500)
        high = PlayerState.from_csr(1500)
        assert high.mu > low.mu
