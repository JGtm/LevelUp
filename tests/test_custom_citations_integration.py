"""Tests d'intégration pour les 6 fonctions de citations custom.

Ces tests appellent les VRAIES fonctions Python au lieu de simuler la logique SQL.
"""

from __future__ import annotations

from src.analysis.citations.custom_rules import (
    compute_flag_em_down,
    compute_hijack,
    compute_mongoose_destroyer,
    compute_vandalism,
    compute_warthog_destroyer,
    compute_wraith_destroyer,
)


class TestFlagEmDownIntegration:
    """Tests d'intégration pour compute_flag_em_down."""

    def test_with_flag_carrier_kill_award(self):
        """Avec award Flag Carrier Kill, retourne le count."""
        awards = {"Flag Carrier Kill": 3}
        result = compute_flag_em_down(awards=awards)
        assert result == 3

    def test_with_flag_carrier_killed_award(self):
        """Avec award Flag Carrier Killed, retourne le count."""
        awards = {"Flag Carrier Killed": 2}
        result = compute_flag_em_down(awards=awards)
        assert result == 2

    def test_with_both_awards_sums(self):
        """Avec les deux awards, somme les counts."""
        awards = {
            "Flag Carrier Kill": 3,
            "Flag Carrier Killed": 2,
        }
        result = compute_flag_em_down(awards=awards)
        assert result == 5

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_flag_em_down(awards=None)
        assert result == 0

    def test_with_empty_dict_returns_zero(self):
        """Avec dict vide, retourne 0."""
        result = compute_flag_em_down(awards={})
        assert result == 0

    def test_with_irrelevant_awards_returns_zero(self):
        """Avec awards non pertinents, retourne 0."""
        awards = {"Kill Assist": 10, "Melee Kill": 5}
        result = compute_flag_em_down(awards=awards)
        assert result == 0


class TestHijackIntegration:
    """Tests d'intégration pour compute_hijack."""

    def test_with_hijacked_award(self):
        """Avec award contenant 'Hijacked', retourne le count."""
        awards = {"Vehicle Hijacked": 2}
        result = compute_hijack(awards=awards)
        assert result == 2

    def test_with_hijack_award(self):
        """Avec award contenant 'Hijack', retourne le count."""
        awards = {"Vehicle Hijack": 3}
        result = compute_hijack(awards=awards)
        assert result == 3

    def test_with_skyjack_award(self):
        """Avec award contenant 'Skyjack', retourne le count."""
        awards = {"Aircraft Skyjack": 1}
        result = compute_hijack(awards=awards)
        assert result == 1

    def test_with_multiple_hijack_awards_sums(self):
        """Avec plusieurs awards hijack, somme tous les counts."""
        awards = {
            "Vehicle Hijacked": 2,
            "Warthog Hijack": 1,
            "Skyjack Master": 3,
        }
        result = compute_hijack(awards=awards)
        assert result == 6

    def test_case_insensitive_matching(self):
        """Le matching est case-insensitive."""
        awards = {
            "VEHICLE HIJACKED": 1,
            "vehicle hijack": 2,
            "SkYjAcK": 3,
        }
        result = compute_hijack(awards=awards)
        assert result == 6

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_hijack(awards=None)
        assert result == 0


class TestVandalismIntegration:
    """Tests d'intégration pour compute_vandalism."""

    def test_with_destroyed_award(self):
        """Avec award contenant 'Destroyed', retourne le count."""
        awards = {"Vehicle Destroyed": 5}
        result = compute_vandalism(awards=awards)
        assert result == 5

    def test_with_destruction_award(self):
        """Avec award contenant 'Destruction', retourne le count."""
        awards = {"Mass Destruction": 3}
        result = compute_vandalism(awards=awards)
        assert result == 3

    def test_with_multiple_destruction_awards_sums(self):
        """Avec plusieurs awards destruction, somme tous les counts."""
        awards = {
            "Warthog Destroyed": 2,
            "Mongoose Destroyed": 1,
            "Vehicle Destruction": 4,
        }
        result = compute_vandalism(awards=awards)
        assert result == 7

    def test_case_insensitive_matching(self):
        """Le matching est case-insensitive."""
        awards = {
            "DESTROYED": 2,
            "destruction": 3,
        }
        result = compute_vandalism(awards=awards)
        assert result == 5

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_vandalism(awards=None)
        assert result == 0


class TestWraithDestroyerIntegration:
    """Tests d'intégration pour compute_wraith_destroyer."""

    def test_with_wraith_destroyed(self):
        """Avec Wraith Destroyed, retourne le count."""
        awards = {"Wraith Destroyed": 2}
        result = compute_wraith_destroyer(awards=awards)
        assert result == 2

    def test_with_apparition_destroyed(self):
        """Avec Apparition Destroyed, retourne le count."""
        awards = {"Apparition Destroyed": 1}
        result = compute_wraith_destroyer(awards=awards)
        assert result == 1

    def test_with_both_types_sums(self):
        """Avec les deux types, somme les counts."""
        awards = {
            "Wraith Destroyed": 2,
            "Apparition Destroyed": 1,
        }
        result = compute_wraith_destroyer(awards=awards)
        assert result == 3

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_wraith_destroyer(awards=None)
        assert result == 0


class TestMongooseDestroyerIntegration:
    """Tests d'intégration pour compute_mongoose_destroyer."""

    def test_with_mongoose_destroyed(self):
        """Avec Mongoose Destroyed, retourne le count."""
        awards = {"Mongoose Destroyed": 3}
        result = compute_mongoose_destroyer(awards=awards)
        assert result == 3

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_mongoose_destroyer(awards=None)
        assert result == 0

    def test_with_empty_dict_returns_zero(self):
        """Avec dict vide, retourne 0."""
        result = compute_mongoose_destroyer(awards={})
        assert result == 0


class TestWarthogDestroyerIntegration:
    """Tests d'intégration pour compute_warthog_destroyer."""

    def test_with_warthog_destroyed(self):
        """Avec Warthog Destroyed, retourne le count."""
        awards = {"Warthog Destroyed": 4}
        result = compute_warthog_destroyer(awards=awards)
        assert result == 4

    def test_with_rocket_warthog_destroyed(self):
        """Avec Rocket Warthog Destroyed, retourne le count."""
        awards = {"Rocket Warthog Destroyed": 2}
        result = compute_warthog_destroyer(awards=awards)
        assert result == 2

    def test_with_both_types_sums(self):
        """Avec les deux types, somme les counts."""
        awards = {
            "Warthog Destroyed": 4,
            "Rocket Warthog Destroyed": 2,
        }
        result = compute_warthog_destroyer(awards=awards)
        assert result == 6

    def test_with_no_awards_returns_zero(self):
        """Sans awards, retourne 0."""
        result = compute_warthog_destroyer(awards=None)
        assert result == 0


class TestCombinedCitationsIntegration:
    """Tests d'intégration avec plusieurs citations combinées."""

    def test_all_six_functions_with_realistic_awards(self):
        """Scénario réaliste : un match avec plusieurs types d'awards."""
        awards = {
            "Flag Carrier Kill": 2,
            "Vehicle Hijack": 1,
            "Warthog Destroyed": 3,
            "Wraith Destroyed": 1,
            "Mongoose Destroyed": 2,
            "Mass Destruction": 5,
        }

        assert compute_flag_em_down(awards=awards) == 2
        assert compute_hijack(awards=awards) == 1
        assert compute_warthog_destroyer(awards=awards) == 3
        assert compute_wraith_destroyer(awards=awards) == 1
        assert compute_mongoose_destroyer(awards=awards) == 2
        # vandalism compte TOUS les "destroyed" (7 total)
        assert compute_vandalism(awards=awards) == 11

    def test_overlapping_awards_counted_correctly(self):
        """vandalism compte TOUS les destroyed, y compris ceux des fonctions spécifiques."""
        awards = {
            "Warthog Destroyed": 2,
            "Ghost Destroyed": 1,
        }

        # Warthog spécifique
        assert compute_warthog_destroyer(awards=awards) == 2

        # Vandalism compte TOUS les "destroyed"
        assert compute_vandalism(awards=awards) == 3

    def test_kwargs_are_ignored(self):
        """Les kwargs supplémentaires sont ignorés (compatibilité)."""
        awards = {"Flag Carrier Kill": 3}

        result = compute_flag_em_down(
            df=None,  # Ignoré
            awards=awards,
            extra_param="ignored",  # Ignoré
            another_kwarg=123,  # Ignoré
        )

        assert result == 3
