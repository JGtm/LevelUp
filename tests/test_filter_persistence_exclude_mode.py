"""Tests pour le mode exclude/include de la persistance des filtres."""

import pytest

from src.ui.filter_state import FilterPreferences, _detect_filter_mode


class TestDetectFilterMode:
    """Tests pour la détection automatique du mode de filtrage."""

    def test_detect_exclude_mode_high_ratio(self):
        """Mode exclude détecté quand >70% sélectionné."""
        selected = {"A", "B", "C", "D", "E", "F", "G", "H"}  # 8/10 = 80%
        all_options = {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "exclude"

    def test_detect_include_mode_low_ratio(self):
        """Mode include détecté quand <30% sélectionné."""
        selected = {"A", "B"}  # 2/10 = 20%
        all_options = {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_gray_zone_returns_include(self):
        """Zone grise (30-70%) retourne include par défaut."""
        selected = {"A", "B", "C", "D", "E"}  # 5/10 = 50%
        all_options = {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_mode_boundary_71_percent(self):
        """Frontière >70% → exclude."""
        selected = set(range(71))
        all_options = set(range(100))
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "exclude"

    def test_detect_mode_boundary_70_percent(self):
        """Frontière 70% → include (zone grise)."""
        selected = set(range(70))
        all_options = set(range(100))
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_mode_boundary_30_percent(self):
        """Frontière 30% → include (zone grise)."""
        selected = set(range(30))
        all_options = set(range(100))
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_mode_boundary_29_percent(self):
        """Frontière <30% → include."""
        selected = set(range(29))
        all_options = set(range(100))
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_mode_all_selected(self):
        """100% sélectionné → exclude."""
        selected = {"A", "B", "C"}
        all_options = {"A", "B", "C"}
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "exclude"

    def test_detect_mode_none_selected(self):
        """0% sélectionné → include."""
        selected = set()
        all_options = {"A", "B", "C"}
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"

    def test_detect_mode_with_list_input(self):
        """Fonctionne avec des listes."""
        selected = ["A", "B", "C", "D", "E", "F", "G", "H"]  # 8/10 = 80%
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "exclude"

    def test_detect_mode_empty_all_options(self):
        """Options vides → include par défaut."""
        selected = set()
        all_options = set()
        
        mode = _detect_filter_mode(selected, all_options)
        
        assert mode == "include"


class TestFilterPreferencesExcludeMode:
    """Tests pour FilterPreferences avec mode exclude/include."""

    def test_preferences_has_mode_fields(self):
        """FilterPreferences a les champs *_mode."""
        prefs = FilterPreferences()
        
        assert hasattr(prefs, "playlists_mode")
        assert hasattr(prefs, "modes_mode")
        assert hasattr(prefs, "maps_mode")

    def test_preferences_mode_fields_optional(self):
        """Les champs *_mode sont optionnels."""
        prefs = FilterPreferences(
            playlists_selected=["A", "B"],
            # pas de playlists_mode
        )
        
        assert prefs.playlists_mode is None
        assert prefs.playlists_selected == ["A", "B"]

    def test_preferences_to_dict_excludes_none(self):
        """to_dict() exclut les valeurs None."""
        prefs = FilterPreferences(
            playlists_selected=["A", "B"],
            playlists_mode="exclude",
            modes_mode=None,  # Pas défini
        )
        
        data = prefs.to_dict()
        
        assert "playlists_selected" in data
        assert "playlists_mode" in data
        assert "modes_mode" not in data

    def test_preferences_from_dict_backward_compatible(self):
        """from_dict() accepte les anciens JSON sans *_mode."""
        data = {
            "playlists_selected": ["A", "B"],
            # pas de playlists_mode
        }
        
        prefs = FilterPreferences.from_dict(data)
        
        assert prefs.playlists_selected == ["A", "B"]
        assert prefs.playlists_mode is None

    def test_preferences_from_dict_with_mode(self):
        """from_dict() charge correctement les *_mode."""
        data = {
            "playlists_selected": ["Firefight"],
            "playlists_mode": "exclude",
            "modes_selected": ["Slayer"],
            "modes_mode": "include",
        }
        
        prefs = FilterPreferences.from_dict(data)
        
        assert prefs.playlists_selected == ["Firefight"]
        assert prefs.playlists_mode == "exclude"
        assert prefs.modes_selected == ["Slayer"]
        assert prefs.modes_mode == "include"


class TestExcludeModeScenarios:
    """Tests de scénarios réels pour le mode exclude."""

    def test_scenario_everything_except_firefight(self):
        """Scénario: Tout sauf Firefight (90% des cas)."""
        all_playlists = ["Quick Play", "Ranked", "BTB", "Firefight", "Slayer"]
        selected = {"Quick Play", "Ranked", "BTB", "Slayer"}  # 4/5 = 80%
        
        mode = _detect_filter_mode(selected, all_playlists)
        
        assert mode == "exclude"
        # Dans le JSON, on sauvegarderait ["Firefight"]
        excluded = set(all_playlists) - selected
        assert excluded == {"Firefight"}

    def test_scenario_only_ranked(self):
        """Scénario: Uniquement Ranked (1% des cas)."""
        all_playlists = ["Quick Play", "Ranked", "BTB", "Firefight", "Slayer"]
        selected = {"Ranked"}  # 1/5 = 20%
        
        mode = _detect_filter_mode(selected, all_playlists)
        
        assert mode == "include"
        # Dans le JSON, on sauvegarderait ["Ranked"]

    def test_scenario_new_playlist_added_exclude_mode(self):
        """Nouvelle playlist ajoutée → auto-incluse en mode exclude."""
        # Sauvegarde initiale
        all_playlists_old = ["Quick Play", "Ranked", "BTB"]
        excluded_saved = ["BTB"]  # Mode exclude
        
        # Nouvelle situation
        all_playlists_new = ["Quick Play", "Ranked", "BTB", "New Mode"]
        
        # Application du mode exclude
        selected_result = set(all_playlists_new) - set(excluded_saved)
        
        assert "New Mode" in selected_result  # Auto-incluse !
        assert "BTB" not in selected_result  # Toujours exclu
        assert selected_result == {"Quick Play", "Ranked", "New Mode"}

    def test_scenario_new_playlist_added_include_mode(self):
        """Nouvelle playlist ajoutée → auto-exclue en mode include."""
        # Sauvegarde initiale
        included_saved = ["Ranked"]  # Mode include
        
        # Nouvelle situation
        all_playlists_new = ["Quick Play", "Ranked", "BTB", "New Mode"]
        
        # Application du mode include
        selected_result = set(included_saved) & set(all_playlists_new)
        
        assert "New Mode" not in selected_result  # Auto-exclue
        assert "Ranked" in selected_result  # Toujours inclus
        assert selected_result == {"Ranked"}


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
