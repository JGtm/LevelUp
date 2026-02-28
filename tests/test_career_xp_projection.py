"""Tests des fonctions d'estimation et projection XP de la page Carrière.

Couvre :
- ``_compute_estimated_xp_curve`` : estimation de l'XP pré-sync
- ``_compute_active_xp_per_day``  : rythme XP/jour actif (hors inactivité)
- ``_compute_hero_projections``   : courbes de projection vers le rang Héros
"""

from __future__ import annotations

from datetime import datetime, timedelta

import pytest

from src.ui.components.career_progress_circle import XP_HERO_TOTAL
from src.ui.pages.career import (
    INACTIVITY_GAP_DAYS,
    WEEKLY_CHALLENGE_XP,
    XP_BOOST_MULTIPLIER,
    _compute_active_xp_per_day,
    _compute_estimated_xp_curve,
    _compute_hero_projections,
)

# ── Helpers ──────────────────────────────────────────────────────────────────


def _make_history(
    snapshots: list[tuple[datetime, int, int]],  # (date, rank, xp_total)
) -> list[dict]:
    """Construit une liste de snapshots career_progression factices."""
    return [
        {
            "rank": rank,
            "rank_name": "Private",
            "rank_tier": "Gold",
            "current_xp": 100,
            "xp_for_next_rank": 1000,
            "xp_total": xp,
            "is_max_rank": False,
            "recorded_at": dt,
        }
        for dt, rank, xp in snapshots
    ]


# ── Tests _compute_active_xp_per_day ─────────────────────────────────────────


class TestComputeActiveXpPerDay:
    def test_histoire_vide(self):
        assert _compute_active_xp_per_day([]) == 0.0

    def test_un_seul_snapshot(self):
        history = _make_history([(datetime(2024, 1, 1), 10, 5000)])
        assert _compute_active_xp_per_day(history) == 0.0

    def test_rythme_constant(self):
        """7000 XP gagnés en 7 jours continus → 1000 XP/jour."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 8), 20, 7000),
            ]
        )
        result = _compute_active_xp_per_day(history)
        assert result == pytest.approx(1000.0, rel=1e-3)

    def test_rythme_plusieurs_snapshots(self):
        """3 snapshots réguliers espacés de 7j : 10000 XP / 14 jours actifs = ~714 XP/j."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 8), 20, 5000),
                (datetime(2024, 1, 15), 30, 10000),
            ]
        )
        result = _compute_active_xp_per_day(history)
        assert result == pytest.approx(10000 / 14.0, rel=1e-3)

    def test_gap_inactivite_superieur_au_seuil(self):
        """Un gap de 30j est remplacé par INACTIVITY_GAP_DAYS/2 = 7j dans le calcul."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 0),
                (datetime(2024, 1, 31), 20, 7000),  # gap 30j > 14j
            ]
        )
        result = _compute_active_xp_per_day(history)
        # 7000 XP / (INACTIVITY_GAP_DAYS/2) jours = 7000 / 7 = 1000
        expected = 7000.0 / (INACTIVITY_GAP_DAYS / 2)
        assert result == pytest.approx(expected, rel=1e-3)

    def test_historique_xp_negatif_ou_nul(self):
        """Si l'XP n'a pas augmenté (ex: rechargement du même snapshot), retourner 0."""
        history = _make_history(
            [
                (datetime(2024, 1, 1), 10, 5000),
                (datetime(2024, 1, 8), 10, 5000),
            ]
        )
        assert _compute_active_xp_per_day(history) == 0.0


# ── Tests _compute_estimated_xp_curve ────────────────────────────────────────


class TestComputeEstimatedXpCurve:
    def _run(self, history, pre_sync_dates):
        """Appelle directement _compute_estimated_xp_curve (plus de mock DB)."""
        return _compute_estimated_xp_curve(history, pre_sync_dates)

    def test_pas_de_matchs_pre_sync(self):
        history = _make_history(
            [
                (datetime(2024, 2, 1), 10, 5000),
                (datetime(2024, 3, 1), 20, 10000),
            ]
        )
        result = self._run(history, [])
        assert result == []

    def test_history_vide(self):
        """Historique vide → pas d'estimation possible."""
        result = self._run([], [datetime(2024, 1, 15), datetime(2024, 1, 20)])
        assert result == []

    def test_courbe_relie_premier_snapshot(self):
        """Le dernier point de la courbe estimée doit être le 1er snapshot réel."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        history = _make_history([(d0, 10, 5000), (d1, 20, 15000)])
        pre_sync_dates = [datetime(2024, 1, 10), datetime(2024, 1, 20)]
        result = self._run(history, pre_sync_dates)
        assert len(result) >= 1
        # Le dernier point doit raccorder au 1er snapshot (date d0, xp 5000)
        last_date, last_xp = result[-1]
        assert last_date == d0
        assert last_xp == 5000

    def test_xp_ne_descend_pas_en_dessous_de_zero(self):
        """L'XP estimé ne doit jamais être négatif."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        # XP très faible au 1er snapshot : la remontée va saturer à 0
        history = _make_history([(d0, 2, 100), (d1, 10, 10100)])
        # Beaucoup de matchs pré-sync pour vite tomber à 0
        pre_sync_dates = [datetime(2024, 1, 1) + timedelta(days=i) for i in range(20)]
        result = self._run(history, pre_sync_dates)
        xp_values = [pt[1] for pt in result]
        assert all(v >= 0 for v in xp_values)

    def test_dates_chronologiques(self):
        """La courbe retournée doit être en ordre chronologique."""
        d0 = datetime(2024, 2, 1)
        d1 = datetime(2024, 3, 1)
        history = _make_history([(d0, 10, 5000), (d1, 20, 15000)])
        pre_sync_dates = [
            datetime(2024, 1, 10),
            datetime(2024, 1, 15),
            datetime(2024, 1, 25),
        ]
        result = self._run(history, pre_sync_dates)
        dates = [pt[0] for pt in result]
        assert dates == sorted(dates)


# ── Tests _compute_hero_projections ──────────────────────────────────────────


class TestComputeHeroProjections:
    BASE_DATE = datetime(2024, 6, 1)
    # XP quelconque sous le seuil Hero
    BASE_XP = 5_000_000
    # Rythme suffisant pour atteindre Hero en < 10 ans
    # 4 319 350 XP restants / 2000 XP/j = ~2160 jours ≈ 5.9 ans
    FAST_XP_PER_DAY = 2000.0

    def test_rythme_nul_retourne_vide(self):
        normal, optimistic = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, 0.0)
        assert normal == []
        assert optimistic == []

    def test_deja_hero_retourne_vide(self):
        normal, optimistic = _compute_hero_projections(XP_HERO_TOTAL, self.BASE_DATE, 1000.0)
        assert normal == []
        assert optimistic == []

    def test_point_depart_est_xp_actuel(self):
        """Les deux courbes commencent à l'XP actuel du joueur."""
        normal, _ = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY)
        assert normal[0][1] == self.BASE_XP
        assert normal[0][0] == self.BASE_DATE

    def test_projection_atteint_hero(self):
        """Les deux courbes doivent se terminer à XP_HERO_TOTAL (rythme suffisant)."""
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        assert normal[-1][1] == XP_HERO_TOTAL
        assert optimistic[-1][1] == XP_HERO_TOTAL

    def test_optimiste_plus_rapide_que_normale(self):
        """La projection optimiste doit atteindre Hero avant la normale."""
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        assert optimistic[-1][0] < normal[-1][0]

    def test_courbes_chronologiques(self):
        """Les points de chaque courbe sont ordonnés dans le temps."""
        # Utiliser un rythme rapide pour rester dans le cap 10 ans
        normal, optimistic = _compute_hero_projections(
            self.BASE_XP, self.BASE_DATE, self.FAST_XP_PER_DAY
        )
        normal_dates = [pt[0] for pt in normal]
        opt_dates = [pt[0] for pt in optimistic]
        assert normal_dates == sorted(normal_dates)
        assert opt_dates == sorted(opt_dates)

    def test_courbes_chronologiques_avec_cap(self):
        """Même avec le cap 10 ans (Hero non atteint), l'ordre est préservé."""
        # 1 XP/jour → 9M+ jours, cap forcé
        normal, optimistic = _compute_hero_projections(100, self.BASE_DATE, 1.0)
        if normal:
            normal_dates = [pt[0] for pt in normal]
            assert normal_dates == sorted(normal_dates)
        if optimistic:
            opt_dates = [pt[0] for pt in optimistic]
            assert opt_dates == sorted(opt_dates)

    def test_optimiste_integre_challenges_et_boost(self):
        """Vérifie que le rythme optimiste = (xp_day + challenges/7) × boost."""
        xp_per_day = self.FAST_XP_PER_DAY
        challenge_per_day = WEEKLY_CHALLENGE_XP / 7.0
        expected_opt_rate = (xp_per_day + challenge_per_day) * XP_BOOST_MULTIPLIER

        xp_remaining = XP_HERO_TOTAL - self.BASE_XP
        expected_days = xp_remaining / expected_opt_rate

        normal, optimistic = _compute_hero_projections(self.BASE_XP, self.BASE_DATE, xp_per_day)
        actual_days = (optimistic[-1][0] - self.BASE_DATE).total_seconds() / 86400.0
        # ±7 jours de tolérance (granularité hebdomadaire)
        assert abs(actual_days - expected_days) <= 7

    def test_cap_10_ans(self):
        """Un rythme très faible ne doit pas générer une courbe > 10 ans."""
        normal, optimistic = _compute_hero_projections(
            100,
            self.BASE_DATE,
            1.0,  # 1 XP/jour → 9M+ jours sans le cap
        )
        # Avec le cap à 365*10 jours, la courbe créée doit rester bornée
        if normal:
            max_days = (normal[-1][0] - self.BASE_DATE).total_seconds() / 86400.0
            assert max_days <= 365 * 10 + 8  # +8j : arrondi hebdomadaire


# ── Constantes ───────────────────────────────────────────────────────────────


class TestConstants:
    def test_weekly_challenge_xp(self):
        """950 XP = 4×50 + 3×100 + 3×150 (Normal + Heroic + Legendary post-CU32)."""
        assert WEEKLY_CHALLENGE_XP == 4 * 50 + 3 * 100 + 3 * 150

    def test_xp_boost_multiplier(self):
        assert XP_BOOST_MULTIPLIER == 2.0

    def test_inactivity_gap_days(self):
        assert INACTIVITY_GAP_DAYS == 14
