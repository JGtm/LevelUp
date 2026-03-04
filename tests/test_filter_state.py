"""Tests unitaires pour le module de persistance des filtres."""

from __future__ import annotations

from datetime import date
from pathlib import Path

import pytest

from src.ui.filter_state import (
    FILTER_DATA_KEYS,
    FILTER_WIDGET_KEY_PREFIXES,
    FilterPreferences,
    _detect_filter_mode,
    apply_filter_preferences,
    clear_filter_preferences,
    get_all_filter_keys_to_clear,
    load_filter_preferences,
    save_filter_preferences,
)


class TestFilterPreferences:
    """Tests pour la classe FilterPreferences."""

    def test_to_dict(self):
        """Test de conversion en dictionnaire."""
        prefs = FilterPreferences(
            filter_mode="Période",
            start_date="2024-01-01",
            end_date="2024-01-31",
            playlists_selected=["Partie rapide", "Arène classée"],
        )
        result = prefs.to_dict()
        assert result["filter_mode"] == "Période"
        assert result["start_date"] == "2024-01-01"
        assert result["end_date"] == "2024-01-31"
        assert result["playlists_selected"] == ["Partie rapide", "Arène classée"]
        # Les valeurs None ne doivent pas être dans le dict
        assert "gap_minutes" not in result

    def test_to_dict_omits_none(self):
        """Test que les valeurs None sont omises."""
        prefs = FilterPreferences()
        result = prefs.to_dict()
        assert result == {}

    def test_from_dict(self):
        """Test de création depuis un dictionnaire."""
        data = {
            "filter_mode": "Sessions",
            "gap_minutes": 120,
            "playlists_selected": ["Partie rapide"],
        }
        prefs = FilterPreferences.from_dict(data)
        assert prefs.filter_mode == "Sessions"
        assert prefs.gap_minutes == 120
        assert prefs.playlists_selected == ["Partie rapide"]
        assert prefs.start_date is None

    def test_from_dict_ignores_unknown_keys(self):
        """Test que les clés inconnues sont ignorées."""
        data = {
            "filter_mode": "Période",
            "unknown_key": "value",
        }
        prefs = FilterPreferences.from_dict(data)
        assert prefs.filter_mode == "Période"
        # Vérifier que unknown_key n'a pas créé d'attribut
        assert not hasattr(prefs, "unknown_key")

    def test_v52_mode_fields_present(self):
        """Les nouveaux champs *_mode v5.2 sont présents dans FilterPreferences."""
        prefs = FilterPreferences(
            playlists_selected=["Firefight"],
            playlists_mode="exclude",
            modes_selected=["PVP"],
            modes_mode="include",
            maps_selected=[],
            maps_mode=None,
        )
        assert prefs.playlists_mode == "exclude"
        assert prefs.modes_mode == "include"
        assert prefs.maps_mode is None

    def test_experience_types_fields_present(self):
        """Les champs experience_types v5.2 sont présents."""
        prefs = FilterPreferences(
            experience_types=["PVE"],
            experience_types_mode="include",
        )
        assert prefs.experience_types == ["PVE"]
        assert prefs.experience_types_mode == "include"

    def test_to_dict_includes_mode_fields(self):
        """to_dict() inclut les champs *_mode si non-None."""
        prefs = FilterPreferences(
            playlists_selected=["Firefight"],
            playlists_mode="exclude",
        )
        d = prefs.to_dict()
        assert d["playlists_mode"] == "exclude"
        assert "modes_mode" not in d  # None → omis

    def test_backward_compat_no_mode_field(self):
        """JSON legacy sans *_mode → *_mode=None (backward compat)."""
        data = {
            "filter_mode": "Période",
            "playlists_selected": ["Partie rapide", "Arène classée"],
        }
        prefs = FilterPreferences.from_dict(data)
        assert prefs.playlists_mode is None  # Pas de mode dans le JSON legacy
        assert prefs.playlists_selected == ["Partie rapide", "Arène classée"]


class TestDetectFilterMode:
    """Tests pour _detect_filter_mode (heuristique intent-based)."""

    def test_majority_checked_returns_exclude(self):
        """Plus de 70% cochés → mode exclude."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B", "C", "D", "E", "F", "G", "H"]  # 80%
        assert _detect_filter_mode(selected, all_options) == "exclude"

    def test_minority_checked_returns_include(self):
        """Moins de 30% cochés → mode include."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B"]  # 20%
        assert _detect_filter_mode(selected, all_options) == "include"

    def test_grey_zone_keeps_current_mode_include(self):
        """Entre 30-70% cochés → garde le mode actuel (include)."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B", "C", "D", "E"]  # 50%
        assert _detect_filter_mode(selected, all_options, "include") == "include"

    def test_grey_zone_keeps_current_mode_exclude(self):
        """Entre 30-70% cochés → garde le mode actuel (exclude)."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B", "C", "D", "E"]  # 50%
        assert _detect_filter_mode(selected, all_options, "exclude") == "exclude"

    def test_empty_all_options_returns_include(self):
        """all_options vide → fallback include."""
        assert _detect_filter_mode(["A"], [], "exclude") == "include"

    def test_all_options_is_set(self):
        """Fonctionne avec des set()."""
        all_options = {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
        selected = {"A", "B", "C", "D", "E", "F", "G", "H"}  # 80%
        assert _detect_filter_mode(selected, all_options) == "exclude"

    def test_exact_70_percent_grey_zone(self):
        """Exactement 70% → zone grise → garde mode actuel."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B", "C", "D", "E", "F", "G"]  # 70% exactement (pas > 0.7)
        assert _detect_filter_mode(selected, all_options, "include") == "include"

    def test_exact_30_percent_grey_zone(self):
        """Exactement 30% → zone grise → garde mode actuel."""
        all_options = ["A", "B", "C", "D", "E", "F", "G", "H", "I", "J"]
        selected = ["A", "B", "C"]  # 30% exactement (pas < 0.3)
        assert _detect_filter_mode(selected, all_options, "exclude") == "exclude"


class TestFilterPersistence:
    """Tests pour la persistance des filtres."""

    @pytest.fixture
    def temp_filters_dir(self, monkeypatch, tmp_path):
        """Crée un répertoire temporaire pour les filtres."""
        filters_dir = tmp_path / ".streamlit" / "filter_preferences"
        filters_dir.mkdir(parents=True, exist_ok=True)

        # Mock _get_filters_dir pour utiliser le répertoire temporaire
        def mock_get_filters_dir():
            return filters_dir

        monkeypatch.setattr("src.ui.filter_state._get_filters_dir", mock_get_filters_dir)
        return filters_dir

    def test_save_and_load_preferences(self, temp_filters_dir):
        """Test sauvegarde et chargement des préférences."""
        prefs = FilterPreferences(
            filter_mode="Période",
            start_date="2024-01-01",
            end_date="2024-01-31",
            playlists_selected=["Partie rapide", "Arène classée"],
            modes_selected=["Assassin"],
            maps_selected=["Carte 1"],
        )

        # Sauvegarder
        save_filter_preferences("test_player", preferences=prefs)

        # Charger
        loaded = load_filter_preferences("test_player")

        assert loaded is not None
        assert loaded.filter_mode == "Période"
        assert loaded.start_date == "2024-01-01"
        assert loaded.end_date == "2024-01-31"
        assert loaded.playlists_selected == ["Partie rapide", "Arène classée"]
        assert loaded.modes_selected == ["Assassin"]
        assert loaded.maps_selected == ["Carte 1"]

    def test_save_include_mode(self, temp_filters_dir):
        """Save une sélection faible → stocké en mode include."""
        prefs = FilterPreferences(
            playlists_selected=["Arène classée"],  # 1 seul → include
            playlists_mode="include",
        )
        save_filter_preferences("test_player", preferences=prefs)
        loaded = load_filter_preferences("test_player")
        assert loaded is not None
        assert loaded.playlists_mode == "include"
        assert loaded.playlists_selected == ["Arène classée"]

    def test_save_exclude_mode(self, temp_filters_dir):
        """Save 'tout sauf FF' → JSON stocke les exclusions en mode exclude."""
        prefs = FilterPreferences(
            playlists_selected=["Firefight"],  # exclusions
            playlists_mode="exclude",
        )
        save_filter_preferences("test_player", preferences=prefs)
        loaded = load_filter_preferences("test_player")
        assert loaded is not None
        assert loaded.playlists_mode == "exclude"
        assert loaded.playlists_selected == ["Firefight"]  # Exclusions stockées

    def test_cycle_complet_save_json_load_apply(self, temp_filters_dir, monkeypatch):
        """Cycle complet : save → JSON sur disque → load → apply → session_state cohérent."""
        import streamlit as st

        session_state: dict = {}
        monkeypatch.setattr(st, "session_state", session_state)

        all_playlists = ["Partie rapide", "Arène classée", "Assassin classé", "Firefight", "Ranked"]

        # Simuler : l'utilisateur coche tout sauf Firefight
        session_state["filter_playlists"] = {
            "Partie rapide",
            "Arène classée",
            "Assassin classé",
            "Ranked",
        }
        session_state["_playlists_filter_mode"] = "include"

        # Save intent-based
        save_filter_preferences("p1", all_playlists=all_playlists)
        loaded = load_filter_preferences("p1")
        assert loaded is not None
        # 4/5 = 80% → exclude → exclusions = ["Firefight"]
        assert loaded.playlists_mode == "exclude"
        assert loaded.playlists_selected == ["Firefight"]

        # Apply avec all_playlists : résultat = all - {Firefight}
        session_state.clear()
        apply_filter_preferences("p1", preferences=loaded, all_playlists=all_playlists)
        assert session_state["filter_playlists"] == {
            "Partie rapide",
            "Arène classée",
            "Assassin classé",
            "Ranked",
        }

    def test_save_updates_exclusions_in_session_state(self, temp_filters_dir, monkeypatch):
        """Save met à jour _*_exclusions dans session_state pour la réconciliation mid-session."""
        import streamlit as st

        session_state: dict = {}
        monkeypatch.setattr(st, "session_state", session_state)

        all_maps = ["Map1", "Map2", "Map3", "Map4", "Map5"]

        # Utilisateur coche tout sauf Map2 (4/5 = 80% → exclude)
        session_state["filter_maps"] = {"Map1", "Map3", "Map4", "Map5"}
        session_state["_maps_filter_mode"] = "include"

        save_filter_preferences("p1", all_maps=all_maps)

        # Vérifier que les exclusions sont mises à jour dans session_state
        assert session_state["_maps_exclusions"] == {"Map2"}
        assert session_state["_maps_filter_mode"] == "exclude"

    def test_mid_session_uncheck_not_reverted_by_reconcile(self, temp_filters_dir, monkeypatch):
        """Décocher une carte mid-session ne doit PAS être annulé par _reconcile_filter_options."""
        import streamlit as st

        from src.app.filters_render import _reconcile_filter_options

        session_state: dict = {}
        monkeypatch.setattr(st, "session_state", session_state)

        all_maps = ["Map1", "Map2", "Map3", "Map4", "Map5"]

        # Simuler : utilisateur a tout coché sauf Map2 (exclude mode)
        session_state["filter_maps"] = {"Map1", "Map3", "Map4", "Map5"}
        session_state["_maps_filter_mode"] = "exclude"
        session_state["_maps_exclusions"] = {"Map2"}

        # Utilisateur décoche Map3 mid-session
        session_state["filter_maps"] = {"Map1", "Map4", "Map5"}

        # Save met à jour les exclusions (Map2 + Map3)
        save_filter_preferences("p1", all_maps=all_maps)
        assert session_state["_maps_exclusions"] == {"Map2", "Map3"}

        # Réconciliation NE DOIT PAS recocher Map3
        _reconcile_filter_options("filter_maps", all_maps, "_maps_filter_mode", "_maps_exclusions")

        # Map3 doit rester décochée
        assert "Map3" not in session_state["filter_maps"]
        assert session_state["filter_maps"] == {"Map1", "Map4", "Map5"}

    def test_load_nonexistent_preferences(self, temp_filters_dir):
        """Test chargement de préférences inexistantes."""
        loaded = load_filter_preferences("nonexistent_player")
        assert loaded is None

    def test_save_with_duckdb_v4_path(self, temp_filters_dir):
        """Test sauvegarde avec chemin DuckDB v4."""
        prefs = FilterPreferences(filter_mode="Sessions", gap_minutes=120)
        db_path = "data/players/MyGamertag/stats.duckdb"

        save_filter_preferences("dummy_xuid", db_path=db_path, preferences=prefs)

        # Vérifier que le fichier a été créé avec la bonne clé
        file_path = temp_filters_dir / "player_MyGamertag.json"
        assert file_path.exists()

        loaded = load_filter_preferences("dummy_xuid", db_path=db_path)
        assert loaded is not None
        assert loaded.filter_mode == "Sessions"
        assert loaded.gap_minutes == 120

    def test_clear_preferences(self, temp_filters_dir):
        """Test suppression des préférences."""
        prefs = FilterPreferences(filter_mode="Période")
        save_filter_preferences("test_player", preferences=prefs)

        # Vérifier que le fichier existe
        file_path = temp_filters_dir / "xuid_test_player.json"
        assert file_path.exists()

        # Supprimer
        clear_filter_preferences("test_player")

        # Vérifier que le fichier n'existe plus
        assert not file_path.exists()

    def test_save_handles_invalid_data(self, temp_filters_dir):
        """Test que la sauvegarde gère les données invalides."""
        # Sauvegarder avec des données valides
        prefs = FilterPreferences(filter_mode="Période")
        save_filter_preferences("test_player", preferences=prefs)

        # Corrompre le fichier manuellement
        file_path = temp_filters_dir / "xuid_test_player.json"
        file_path.write_text("invalid json{")

        # Le chargement doit retourner None sans lever d'exception
        loaded = load_filter_preferences("test_player")
        assert loaded is None


class TestApplyFilterPreferences:
    """Tests pour l'application des préférences dans session_state."""

    def test_apply_preferences(self, monkeypatch):
        """Test application des préférences dans session_state."""
        import streamlit as st

        session_state = {}
        original_session_state = getattr(st, "session_state", None)
        monkeypatch.setattr(st, "session_state", session_state)

        try:
            prefs = FilterPreferences(
                filter_mode="Période",
                start_date="2024-01-15",
                end_date="2024-01-20",
                gap_minutes=120,
                picked_session_label="Session 1",
                playlists_selected=["Partie rapide"],
                modes_selected=["Assassin"],
                maps_selected=["Carte 1"],
            )

            apply_filter_preferences("test_player", preferences=prefs)

            assert session_state["filter_mode"] == "Période"
            assert session_state["start_date_cal"] == date(2024, 1, 15)
            assert session_state["end_date_cal"] == date(2024, 1, 20)
            assert session_state["gap_minutes"] == 120
            assert session_state["picked_session_label"] == "Session 1"
            assert session_state["picked_sessions"] == ["Session 1"]
            assert session_state["filter_playlists"] == {"Partie rapide"}
            assert session_state["filter_modes"] == {"Assassin"}
            assert session_state["filter_maps"] == {"Carte 1"}
        finally:
            if original_session_state is not None:
                monkeypatch.setattr(st, "session_state", original_session_state)

    def test_apply_exclude_with_all_playlists(self, monkeypatch):
        """Apply exclude + all_playlists → session_state = all - exclusions."""
        import streamlit as st

        session_state = {}
        monkeypatch.setattr(st, "session_state", session_state)

        all_playlists = ["Partie rapide", "Arène classée", "Firefight", "Ranked"]
        prefs = FilterPreferences(
            playlists_selected=["Firefight"],  # exclusion
            playlists_mode="exclude",
        )

        apply_filter_preferences("p1", preferences=prefs, all_playlists=all_playlists)

        # session_state doit contenir all - {Firefight}
        assert session_state["filter_playlists"] == {"Partie rapide", "Arène classée", "Ranked"}
        assert session_state["_playlists_filter_mode"] == "exclude"
        assert session_state["_playlists_exclusions"] == {"Firefight"}

    def test_apply_exclude_without_all_playlists_fallback(self, monkeypatch):
        """Apply exclude sans all_playlists → fallback inclusions (les exclusions)."""
        import streamlit as st

        session_state = {}
        monkeypatch.setattr(st, "session_state", session_state)

        prefs = FilterPreferences(
            playlists_selected=["Firefight"],
            playlists_mode="exclude",
        )

        # Sans all_playlists, on ne peut pas calculer all - exclusions
        # → on prend playlists_selected comme inclusions directes
        apply_filter_preferences("p1", preferences=prefs, all_playlists=None)

        # Sans all_playlists, apply interprète les stored comme inclusions directes
        assert session_state["filter_playlists"] == {"Firefight"}

    def test_backward_compat_no_mode_in_json(self, monkeypatch):
        """JSON legacy sans *_mode → traité comme include (backward compat)."""
        import streamlit as st

        session_state = {}
        monkeypatch.setattr(st, "session_state", session_state)

        # Simuler un FilterPreferences chargé depuis JSON legacy (sans playlists_mode)
        prefs = FilterPreferences(
            playlists_selected=["Partie rapide", "Arène classée"],
            playlists_mode=None,  # Legacy : pas de mode → include
        )

        all_playlists = ["Partie rapide", "Arène classée", "Firefight"]
        apply_filter_preferences("p1", preferences=prefs, all_playlists=all_playlists)

        # Mode None → traité comme include → on restaure les inclusions telles quelles
        assert session_state["filter_playlists"] == {"Partie rapide", "Arène classée"}
        assert session_state["_playlists_filter_mode"] == "include"

    def test_new_playlist_auto_included_in_exclude_mode(self, monkeypatch):
        """Exclude "Firefight" + nouvelle playlist "Slayer" dans all → Slayer cochée."""
        import streamlit as st

        session_state = {}
        monkeypatch.setattr(st, "session_state", session_state)

        # JSON sauvegardé : mode exclude, Firefight exclu
        prefs = FilterPreferences(
            playlists_selected=["Firefight"],
            playlists_mode="exclude",
        )

        # Au chargement, all_playlists contient maintenant une nouvelle "Slayer"
        all_playlists = ["Partie rapide", "Arène classée", "Firefight", "Slayer"]
        apply_filter_preferences("p1", preferences=prefs, all_playlists=all_playlists)

        # Slayer non dans les exclusions → auto-cochée ✓
        assert "Slayer" in session_state["filter_playlists"]
        assert "Firefight" not in session_state["filter_playlists"]

    def test_new_playlist_not_included_in_include_mode(self, monkeypatch):
        """Include "Ranked" + nouvelle playlist "Slayer" → Slayer pas cochée."""
        import streamlit as st

        session_state = {}
        monkeypatch.setattr(st, "session_state", session_state)

        prefs = FilterPreferences(
            playlists_selected=["Arène classée"],
            playlists_mode="include",
        )

        all_playlists = ["Partie rapide", "Arène classée", "Firefight", "Slayer"]
        apply_filter_preferences("p1", preferences=prefs, all_playlists=all_playlists)

        # Mode include → seul ce qui est dans la liste est coché
        assert session_state["filter_playlists"] == {"Arène classée"}
        assert "Slayer" not in session_state["filter_playlists"]

    def test_apply_preferences_with_toutes_session(self, monkeypatch):
        """Test application avec session '(toutes)'."""
        import streamlit as st

        session_state = {}
        original_session_state = getattr(st, "session_state", None)
        monkeypatch.setattr(st, "session_state", session_state)

        try:
            prefs = FilterPreferences(
                picked_session_label="(toutes)",
            )

            apply_filter_preferences("test_player", preferences=prefs)

            assert session_state["picked_session_label"] == "(toutes)"
            assert session_state["picked_sessions"] == []
        finally:
            if original_session_state is not None:
                monkeypatch.setattr(st, "session_state", original_session_state)

    def test_apply_preferences_handles_invalid_dates(self, monkeypatch):
        """Test que les dates invalides sont ignorées."""
        import streamlit as st

        session_state = {}
        original_session_state = getattr(st, "session_state", None)
        monkeypatch.setattr(st, "session_state", session_state)

        try:
            prefs = FilterPreferences(
                start_date="invalid-date",
                end_date="2024-01-20",
            )

            apply_filter_preferences("test_player", preferences=prefs)

            # start_date_cal ne doit pas être défini (date invalide)
            assert "start_date_cal" not in session_state
            # end_date_cal doit être défini (date valide)
            assert session_state["end_date_cal"] == date(2024, 1, 20)
        finally:
            if original_session_state is not None:
                monkeypatch.setattr(st, "session_state", original_session_state)

    def test_apply_none_preferences(self, monkeypatch):
        """Test application de None (ne fait rien)."""
        import streamlit as st

        session_state = {"existing_key": "existing_value"}
        original_session_state = getattr(st, "session_state", None)
        monkeypatch.setattr(st, "session_state", session_state)

        try:
            apply_filter_preferences("test_player", preferences=None)

            # Rien ne doit être modifié
            assert len(session_state) == 1
            assert session_state["existing_key"] == "existing_value"
        finally:
            if original_session_state is not None:
                monkeypatch.setattr(st, "session_state", original_session_state)


class TestPlayerKeyGeneration:
    """Tests pour la génération de clés de joueur."""

    def test_get_player_key_with_xuid(self):
        """Test génération de clé avec XUID."""
        from src.ui.filter_state import _get_player_key

        key = _get_player_key("123456789", None)
        assert key == "xuid_123456789"

    def test_get_player_key_with_duckdb_v4_path(self):
        """Test génération de clé avec chemin DuckDB v4."""
        from src.ui.filter_state import _get_player_key

        db_path = "data/players/MyGamertag/stats.duckdb"
        key = _get_player_key("dummy_xuid", db_path)
        assert key == "player_MyGamertag"

    def test_get_player_key_with_relative_path(self):
        """Test génération de clé avec chemin relatif."""
        from src.ui.filter_state import _get_player_key

        db_path = Path("data/players/TestPlayer/stats.duckdb")
        key = _get_player_key("dummy_xuid", str(db_path))
        assert key == "player_TestPlayer"

    def test_get_player_key_fallback_to_xuid(self):
        """Test fallback vers XUID si le chemin n'est pas DuckDB v4."""
        from src.ui.filter_state import _get_player_key

        db_path = "some/other/path.db"
        key = _get_player_key("123456789", db_path)
        assert key == "xuid_123456789"


class TestGetAllFilterKeysToClear:
    """Tests pour get_all_filter_keys_to_clear (nettoyage exhaustif)."""

    def test_returns_matching_data_keys(self):
        """Les clés de données présentes dans session_state sont retournées."""
        session_state = {
            "filter_mode": "Sessions",
            "picked_session_label": "Session 1",
            "unrelated_key": "value",
        }
        result = get_all_filter_keys_to_clear(session_state)
        assert "filter_mode" in result
        assert "picked_session_label" in result
        assert "unrelated_key" not in result

    def test_returns_widget_prefix_keys(self):
        """Les clés de widgets checkbox (préfixes) sont retournées."""
        session_state = {
            "filter_playlists_cb_Partie rapide_v1": True,
            "filter_playlists_cb_Arène classée_v1": False,
            "filter_modes_cb_Assassin_v1": True,
            "filter_maps_cb_Carte 1_v1": True,
            "page": "Statistiques",
        }
        result = get_all_filter_keys_to_clear(session_state)
        assert len(result) == 4
        assert "page" not in result
        for key in result:
            assert any(key.startswith(p) for p in FILTER_WIDGET_KEY_PREFIXES)

    def test_returns_both_data_and_widget_keys(self):
        """Combinaison : clés de données + clés de widgets."""
        session_state = {
            "filter_mode": "Période",
            "start_date_cal": "2024-01-01",
            "filter_playlists_cb_test_v1": True,
            "filter_modes_cb_test_v1": False,
            "other_state": 42,
        }
        result = get_all_filter_keys_to_clear(session_state)
        assert "filter_mode" in result
        assert "start_date_cal" in result
        assert "filter_playlists_cb_test_v1" in result
        assert "filter_modes_cb_test_v1" in result
        assert "other_state" not in result

    def test_empty_session_state(self):
        """Aucune clé à retourner si session_state est vide."""
        assert get_all_filter_keys_to_clear({}) == []

    def test_all_data_keys_covered(self):
        """Toutes les clés de FILTER_DATA_KEYS sont reconnues."""
        session_state = dict.fromkeys(FILTER_DATA_KEYS, "dummy")
        result = get_all_filter_keys_to_clear(session_state)
        for k in FILTER_DATA_KEYS:
            assert k in result

    def test_intent_keys_in_filter_data_keys(self):
        """Les clés intent-based v5.2 sont dans FILTER_DATA_KEYS."""
        intent_keys = [
            "_playlists_filter_mode",
            "_modes_filter_mode",
            "_maps_filter_mode",
            "_playlists_exclusions",
            "_modes_exclusions",
            "_maps_exclusions",
            "_experience_types_filter_mode",
            "_experience_types_exclusions",
            "filter_experience_types",
        ]
        for k in intent_keys:
            assert k in FILTER_DATA_KEYS, f"Clé intent manquante dans FILTER_DATA_KEYS: {k}"

    def test_no_duplicate_keys(self):
        """Pas de doublons si une clé est à la fois dans DATA_KEYS et commence par un préfixe."""
        session_state = {"filter_playlists": {"a"}, "filter_playlists_cb_x_v1": True}
        result = get_all_filter_keys_to_clear(session_state)
        assert len(result) == len(set(result))  # Pas de doublons

    def test_simulates_player_switch_cleanup(self):
        """Scénario complet A -> B : après nettoyage, plus de clés de filtres."""
        session_state = {
            # Clés de données
            "filter_mode": "Sessions",
            "picked_session_label": "Session 5",
            "picked_sessions": ["Session 5"],
            "filter_playlists": {"Partie rapide"},
            "filter_modes": {"Assassin"},
            "filter_maps": {"Recharge"},
            "filter_experience_types": {"PVP non classé"},
            "_playlists_filter_mode": "exclude",
            "_playlists_exclusions": {"Firefight"},
            "min_matches_maps": 1,
            "_min_matches_maps_auto": True,
            "_latest_session_label": "Session 5",
            # Clés de widgets (générées par les checkboxes Streamlit)
            "filter_playlists_cb_Partie rapide_v2": True,
            "filter_playlists_cb_Arène classée_v2": False,
            "filter_modes_cb_Assassin_v2": True,
            "filter_maps_cb_Recharge_v2": True,
            "filter_maps_cb_Aquarius_v2": True,
            "filter_experience_types_cb_PVE_v1": False,
            # Clés non liées aux filtres (ne doivent pas être supprimées)
            "db_path": "/some/path.duckdb",
            "xuid_input": "player1",
            "page": "Statistiques",
        }

        keys_to_clear = get_all_filter_keys_to_clear(session_state)
        for key in keys_to_clear:
            del session_state[key]

        # Seules les clés non liées aux filtres restent
        assert "db_path" in session_state
        assert "xuid_input" in session_state
        assert "page" in session_state
        # Aucune clé de filtre ne subsiste
        assert "filter_mode" not in session_state
        assert "filter_playlists" not in session_state
        assert "filter_experience_types" not in session_state
        assert "_playlists_filter_mode" not in session_state
        assert not any(k.startswith(p) for k in session_state for p in FILTER_WIDGET_KEY_PREFIXES)
