"""Tests d'intégration pour les 6 fonctions de citations custom.

Ces tests appellent les VRAIES fonctions Python au lieu de simuler la logique SQL.
"""

from __future__ import annotations

from src.analysis.citations.custom_rules import (
    compute_annexion_forcee,
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


class TestTechnicalIdSupport:
    """Tests pour les nouveaux IDs techniques (post-migration)."""

    def test_flag_em_down_with_runner_stopped(self):
        """Avec le nouvel ID technique runner_stopped."""
        awards = {"runner_stopped": 3}
        assert compute_flag_em_down(awards=awards) == 3

    def test_flag_em_down_mixed_legacy_and_technical(self):
        """Mélange legacy + technique doit sommer."""
        awards = {"runner_stopped": 2, "Flag Carrier Kill": 1}
        assert compute_flag_em_down(awards=awards) == 3

    def test_wraith_with_destroyed_wraith(self):
        """Avec le nouvel ID technique destroyed_wraith."""
        awards = {"destroyed_wraith": 2}
        assert compute_wraith_destroyer(awards=awards) == 2

    def test_wraith_mixed_legacy_and_technical(self):
        """Mélange legacy + technique doit sommer."""
        awards = {"destroyed_wraith": 1, "Wraith Destroyed": 2}
        assert compute_wraith_destroyer(awards=awards) == 3

    def test_mongoose_with_destroyed_mongoose(self):
        """Avec le nouvel ID technique destroyed_mongoose."""
        awards = {"destroyed_mongoose": 4}
        assert compute_mongoose_destroyer(awards=awards) == 4

    def test_warthog_with_destroyed_warthog(self):
        """Avec les nouveaux IDs techniques destroyed_warthog."""
        awards = {"destroyed_warthog": 3, "destroyed_rocket_warthog": 1}
        assert compute_warthog_destroyer(awards=awards) == 4

    def test_hijack_with_hijacked_prefix(self):
        """Avec les nouveaux IDs techniques hijacked_*."""
        awards = {"hijacked_banshee": 1, "hijacked_ghost": 2}
        assert compute_hijack(awards=awards) == 3

    def test_vandalism_with_destroyed_prefix(self):
        """Avec les nouveaux IDs techniques destroyed_*."""
        awards = {"destroyed_banshee": 1, "destroyed_ghost": 2, "destroyed_warthog": 3}
        assert compute_vandalism(awards=awards) == 6

    def test_vandalism_mixed_legacy_and_technical(self):
        """Mélange legacy + technique pour vandalism."""
        awards = {"destroyed_banshee": 1, "Warthog Destroyed": 2}
        assert compute_vandalism(awards=awards) == 3

    def test_zone_capture_with_zone_captured(self):
        """Avec le nouvel ID technique zone_captured."""
        awards = {"zone_captured": 9}
        assert compute_annexion_forcee(awards=awards) == 3

    def test_zone_capture_mixed_legacy(self):
        """zone_captured prioritaire via 'or' chain."""
        awards = {"zone_captured": 6}
        assert compute_annexion_forcee(awards=awards) == 2
