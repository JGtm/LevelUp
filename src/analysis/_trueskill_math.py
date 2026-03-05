"""Fonctions mathématiques TrueSkill 2 et état du joueur.

Extraites de skill_rating.py — regroupe l'état PlayerState,
les fonctions gaussiennes, la mise à jour TrueSkill 2 et le decay inactivité.
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any

from src.analysis.skill_rating_config import (
    BETA,
    DRAW_PROBABILITY,
    INACTIVITY_SIGMA_PER_DAY,
    INACTIVITY_THRESHOLD_DAYS,
    INITIAL_MU,
    INITIAL_SIGMA,
    K_ELO,
    MAX_INACTIVITY_DAYS,
    MAX_SIGMA,
    MIN_RATING,
    MIN_SIGMA,
    TAU,
)
from src.utils.safe_types import clamp as _clamp

# =============================================================================
# État du joueur entre deux matchs
# =============================================================================


@dataclass
class PlayerState:
    """État TrueSkill 2 du joueur, mis à jour après chaque match."""

    mu: float = INITIAL_MU
    """Rating courant (centre de la distribution gaussienne)."""

    sigma: float = INITIAL_SIGMA
    """Déviation courante (incertitude sur le rating)."""

    match_count: int = 0
    """Nombre de matchs traités depuis l'initialisation."""

    last_match_time: Any = None
    """Timestamp du dernier match traité (datetime ou None)."""

    accuracy_history: list[float] = field(default_factory=list)
    """Historique des précisions des derniers matchs (pour accuracy_delta)."""

    damage_eff_history: list[float] = field(default_factory=list)
    """Historique damage_efficiency des derniers matchs (pour damage_eff_delta)."""

    @classmethod
    def from_csr(cls, csr: float) -> PlayerState:
        """Initialise un PlayerState à partir d'un CSR connu (seed LUSR).

        Mapping bijection tier-à-tier entre l'échelle CSR Halo et l'échelle LUSR :
            mu = 1000 + csr × (2/3)

        Le sigma est réduit à MIN_SIGMA (état stable) pour éviter la phase de
        haute volatilité qui cause des swings aléatoires dans les premiers matchs.
        """
        mu = 1000.0 + max(0.0, float(csr)) * (2.0 / 3.0)
        sigma = MIN_SIGMA
        return cls(mu=mu, sigma=sigma)


# =============================================================================
# Fonctions mathématiques gaussiennes — truncated Gaussian
# =============================================================================


def _standard_normal_pdf(x: float) -> float:
    """Densité de la loi normale standard."""
    return math.exp(-0.5 * x * x) / math.sqrt(2.0 * math.pi)


def _standard_normal_cdf(x: float) -> float:
    """Fonction de répartition de la loi normale standard."""
    return 0.5 * (1.0 + math.erf(x / math.sqrt(2.0)))


def _draw_margin(beta: float) -> float:
    """Calcule la marge d'égalité à partir de la probabilité de draw."""
    if DRAW_PROBABILITY <= 0:
        return 0.0
    p = (DRAW_PROBABILITY + 1.0) / 2.0
    t = math.sqrt(-2.0 * math.log(1.0 - p)) if p < 1.0 else 8.0
    return t * beta


def _v_win(t: float, eps: float) -> float:
    """Facteur v pour le cas victoire (score > 0.5)."""
    x = t - eps
    denom = _standard_normal_cdf(x)
    if denom < 1e-10:
        return -x
    return _standard_normal_pdf(x) / denom


def _w_win(t: float, eps: float) -> float:
    """Facteur w pour le cas victoire."""
    x = t - eps
    v = _v_win(t, eps)
    return v * (v + x)


def _v_draw(t: float, eps: float) -> float:  # pragma: no cover — non utilisé (v5.3)
    """Facteur v pour le cas égalité — conservé comme référence TrueSkill 2."""
    abs_t = abs(t)
    num = _standard_normal_pdf(eps - abs_t) - _standard_normal_pdf(-eps - abs_t)
    denom = _standard_normal_cdf(eps - abs_t) - _standard_normal_cdf(-eps - abs_t)
    if denom < 1e-10:
        return 0.0
    if t < 0:
        return -num / denom
    return num / denom


def _w_draw(t: float, eps: float) -> float:  # pragma: no cover — non utilisé (v5.3)
    """Facteur w pour le cas égalité — conservé comme référence TrueSkill 2."""
    abs_t = abs(t)
    eps_m = eps - abs_t
    eps_p = -eps - abs_t
    num = eps_m * _standard_normal_pdf(eps_m) - eps_p * _standard_normal_pdf(eps_p)
    denom = _standard_normal_cdf(eps_m) - _standard_normal_cdf(eps_p)
    if denom < 1e-10:
        return 1.0
    return (
        num / denom
        + ((eps_m * _standard_normal_pdf(eps_m) - eps_p * _standard_normal_pdf(eps_p)) / denom) ** 2
    )


# =============================================================================
# Mise à jour TrueSkill 2
# =============================================================================


def trueskill_update(  # noqa: PLR0913
    mu: float,
    sigma: float,
    mu_opp: float,
    sigma_opp: float,
    actual_score: float,
    weight_factor: float = 1.0,
) -> tuple[float, float]:
    """Met à jour (mu, sigma) après un match.

    Mu : formule Elo-style continue (K_ELO × (score - 0.5) × wf).
    Sigma : réduction TrueSkill à t=0 (symétrique, matchmaking ~équivalent).
    """
    delta_mu = K_ELO * (actual_score - 0.5) * weight_factor
    new_mu = max(MIN_RATING, mu + delta_mu)

    c2 = 2.0 * BETA**2 + sigma**2 + sigma_opp**2
    c = math.sqrt(c2)
    eps = _draw_margin(BETA)

    sigma2 = sigma**2
    w = _w_win(0.0, eps / c)
    delta_sigma2 = sigma2 * (sigma2 / c2) * w * weight_factor

    new_sigma2 = max(MIN_SIGMA**2, sigma2 - delta_sigma2)
    new_sigma = math.sqrt(new_sigma2)

    new_sigma = min(math.sqrt(new_sigma**2 + TAU**2), MAX_SIGMA)

    return new_mu, new_sigma


def apply_inactivity_decay(sigma: float, days_inactive: float) -> float:
    """Augmente sigma proportionnellement à la durée d'inactivité."""
    capped = min(days_inactive, MAX_INACTIVITY_DAYS)
    if capped <= INACTIVITY_THRESHOLD_DAYS:
        return sigma
    added = INACTIVITY_SIGMA_PER_DAY * (capped - INACTIVITY_THRESHOLD_DAYS)
    return _clamp(sigma + added, MIN_SIGMA, MAX_SIGMA)
