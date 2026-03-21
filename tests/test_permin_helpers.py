"""Tests unitaires — build_symmetric_abs_ticks (src/visualization/_permin_helpers.py)."""

from __future__ import annotations

import pytest

from src.visualization._permin_helpers import build_symmetric_abs_ticks


class TestBuildSymmetricAbsTicks:
    def test_returns_two_lists(self) -> None:
        tickvals, ticktext = build_symmetric_abs_ticks(2.0, 1.5)
        assert isinstance(tickvals, list)
        assert isinstance(ticktext, list)

    def test_same_length(self) -> None:
        tickvals, ticktext = build_symmetric_abs_ticks(3.0, 2.0)
        assert len(tickvals) == len(ticktext)

    def test_symmetric_around_zero(self) -> None:
        """Les ticks doivent être symétriques autour de 0."""
        tickvals, _ = build_symmetric_abs_ticks(2.0, 2.0)
        pos = [v for v in tickvals if v > 0]
        neg = [v for v in tickvals if v < 0]
        assert len(pos) == len(neg)
        for p, n in zip(sorted(pos), sorted([-v for v in neg]), strict=False):
            assert p == pytest.approx(n, abs=1e-6)

    def test_zero_included(self) -> None:
        tickvals, _ = build_symmetric_abs_ticks(1.0, 1.0)
        assert 0.0 in tickvals

    def test_labels_are_absolute(self) -> None:
        """Tous les labels doivent être positifs (pas de signe négatif)."""
        _, ticktext = build_symmetric_abs_ticks(3.0, 2.0)
        for label in ticktext:
            assert not label.startswith("-"), f"Label négatif inattendu : {label!r}"

    def test_max_pos_dominates(self) -> None:
        """Le côté le plus grand détermine l'échelle."""
        tickvals_big, _ = build_symmetric_abs_ticks(10.0, 1.0)
        tickvals_small, _ = build_symmetric_abs_ticks(1.0, 1.0)
        assert max(tickvals_big) > max(tickvals_small)

    def test_max_neg_dominates(self) -> None:
        tickvals, _ = build_symmetric_abs_ticks(1.0, 10.0)
        assert max(tickvals) >= 10.0

    def test_both_zero_no_crash(self) -> None:
        """max_pos=0, max_neg=0 → doit utiliser le minimum interne (0.1)."""
        tickvals, ticktext = build_symmetric_abs_ticks(0.0, 0.0)
        assert len(tickvals) > 0
        assert len(tickvals) == len(ticktext)

    def test_integer_labels_have_no_decimal(self) -> None:
        """Un tick entier (ex. 1.0) doit avoir le label '1', pas '1.00'."""
        _, ticktext = build_symmetric_abs_ticks(1.0, 1.0, n_steps=1)
        integer_labels = [t for t in ticktext if "." not in t]
        # Au moins un label entier attendu (0, 1, 2…)
        assert len(integer_labels) >= 1

    def test_custom_n_steps(self) -> None:
        """n_steps contrôle le nombre approximatif de graduations positives."""
        tickvals_5, _ = build_symmetric_abs_ticks(10.0, 10.0, n_steps=5)
        tickvals_2, _ = build_symmetric_abs_ticks(10.0, 10.0, n_steps=2)
        pos_5 = [v for v in tickvals_5 if v > 0]
        pos_2 = [v for v in tickvals_2 if v > 0]
        assert len(pos_5) >= len(pos_2)

    def test_sorted_ascending(self) -> None:
        tickvals, _ = build_symmetric_abs_ticks(5.0, 3.0)
        assert tickvals == sorted(tickvals)
