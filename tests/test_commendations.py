"""Tests pour src/ui/commendations.py — fonctions pures de traitement des citations."""

from __future__ import annotations

from src.ui.commendations import (
    _compute_mastery_display,
    _normalize_name,
    _parse_tier_targets,
)

# ============================================================================
# _normalize_name
# ============================================================================


class TestNormalizeName:
    def test_basic(self):
        assert _normalize_name("Hello World") == "hello world"

    def test_accents(self):
        result = _normalize_name("Récompensé")
        assert "e" in result
        assert "é" not in result

    def test_strip_whitespace(self):
        assert _normalize_name("  hello  ") == "hello"

    def test_multiple_spaces(self):
        assert _normalize_name("hello   world") == "hello world"

    def test_none(self):
        assert _normalize_name(None) == ""

    def test_empty(self):
        assert _normalize_name("") == ""

    def test_mixed_accents(self):
        result = _normalize_name("Tête-à-tête")
        assert result == "tete-a-tete"


# ============================================================================
# _parse_tier_targets
# ============================================================================


class TestParseTierTargets:
    def test_basic_csv(self):
        result = _parse_tier_targets("10,20,30,50,100")
        assert len(result) == 5
        assert result[0] == {"tier": 1, "target_count": 10}
        assert result[4] == {"tier": 5, "target_count": 100}

    def test_none(self):
        assert _parse_tier_targets(None) == []

    def test_empty_string(self):
        assert _parse_tier_targets("") == []

    def test_single_value(self):
        result = _parse_tier_targets("50")
        assert len(result) == 1
        assert result[0] == {"tier": 1, "target_count": 50}

    def test_spaces_around(self):
        result = _parse_tier_targets(" 10 , 20 , 30 ")
        assert len(result) == 3
        assert result[0]["target_count"] == 10
        assert result[2]["target_count"] == 30

    def test_invalid_values_skipped(self):
        result = _parse_tier_targets("10,abc,30")
        assert len(result) == 2
        assert result[0]["target_count"] == 10
        assert result[1]["target_count"] == 30


# ============================================================================
# _compute_mastery_display
# ============================================================================


class TestComputeMasteryDisplay:
    def test_master_achieved(self):
        tiers = [{"target_count": 10}, {"target_count": 50}, {"target_count": 100}]
        label, counter, is_master, ratio = _compute_mastery_display(150, tiers)
        assert label == "Maître"
        assert is_master is True
        assert ratio == 1.0
        assert "150" in counter

    def test_exactly_at_master(self):
        tiers = [{"target_count": 10}, {"target_count": 100}]
        label, counter, is_master, ratio = _compute_mastery_display(100, tiers)
        assert label == "Maître"
        assert is_master is True

    def test_level_1(self):
        tiers = [{"target_count": 10}, {"target_count": 50}]
        label, counter, is_master, ratio = _compute_mastery_display(5, tiers)
        assert label == "Niveau 1"
        assert is_master is False
        assert "5/10" in counter
        assert 0.0 <= ratio <= 1.0

    def test_level_2(self):
        tiers = [{"target_count": 10}, {"target_count": 50}]
        label, counter, is_master, ratio = _compute_mastery_display(30, tiers)
        assert label == "Niveau 2"
        assert is_master is False
        assert "30/50" in counter

    def test_no_tiers(self):
        label, counter, is_master, ratio = _compute_mastery_display(10, [])
        assert label == "—"
        assert is_master is False
        assert ratio == 0.0

    def test_zero_count(self):
        tiers = [{"target_count": 10}]
        label, counter, is_master, ratio = _compute_mastery_display(0, tiers)
        assert label == "Niveau 1"
        assert ratio == 0.0

    def test_negative_count(self):
        tiers = [{"target_count": 10}]
        label, counter, is_master, ratio = _compute_mastery_display(-5, tiers)
        assert ratio == 0.0

    def test_invalid_tier_values(self):
        tiers = [{"target_count": "invalid"}, {"target_count": None}]
        label, counter, is_master, ratio = _compute_mastery_display(10, tiers)
        assert label == "—"  # No valid tiers

    def test_duplicate_tiers(self):
        tiers = [{"target_count": 10}, {"target_count": 10}, {"target_count": 50}]
        label, counter, is_master, ratio = _compute_mastery_display(5, tiers)
        assert label == "Niveau 1"

    def test_progress_ratio_midway(self):
        tiers = [{"target_count": 100}]
        _, _, _, ratio = _compute_mastery_display(50, tiers)
        assert abs(ratio - 0.5) < 0.01

    def test_with_parsed_csv_tiers(self):
        """Vérifie la compatibilité avec le format issu de _parse_tier_targets."""
        tiers = _parse_tier_targets("10,20,30,50,100")
        label, counter, is_master, ratio = _compute_mastery_display(25, tiers)
        assert label == "Niveau 3"
        assert "25/30" in counter
        assert is_master is False
