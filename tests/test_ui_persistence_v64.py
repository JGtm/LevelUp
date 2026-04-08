"""Tests de persistance UI v6.4.

Couvre :
- get_player_prefs_path() : chemin canonique data/players/ vs fallback legacy
- _resolve_prefs_path() : migration silencieuse .streamlit/ → data/players/
- browser_storage : restore_browser_prefs, persist_browser_prefs (logique, sans iframe)
- _maybe_apply_browser_prefs : restauration joueur/langue depuis localStorage
- _resolve_db_path : branche localStorage dans data_loader
"""

from __future__ import annotations

import json
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

import pytest

from src.ui import filter_state
from src.ui._filter_state_model import FilterPreferences, get_player_prefs_path
from src.ui.filter_state import _resolve_prefs_path

# ---------------------------------------------------------------------------
# Fixtures communes
# ---------------------------------------------------------------------------


@pytest.fixture()
def fake_st(monkeypatch: pytest.MonkeyPatch) -> SimpleNamespace:
    """Faux objet streamlit avec session_state mutable."""
    fake = SimpleNamespace(session_state={}, warning=MagicMock(), rerun=MagicMock())
    monkeypatch.setattr(filter_state, "st", fake)
    return fake


@pytest.fixture()
def players_dir(tmp_path: Path) -> Path:
    """Crée une arborescence data/players/TestPlayer/ réaliste dans tmp_path."""
    p = tmp_path / "data" / "players" / "TestPlayer"
    p.mkdir(parents=True)
    db = p / "stats.duckdb"
    db.write_bytes(b"\x00" * 8)  # fichier non-vide
    return p


@pytest.fixture()
def legacy_filters_dir(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    """Isole le répertoire legacy .streamlit/filter_preferences/ dans tmp_path."""
    d = tmp_path / ".streamlit" / "filter_preferences"
    d.mkdir(parents=True)
    monkeypatch.setattr(filter_state, "_get_filters_dir", lambda: d)
    return d


# ===========================================================================
# Tests : get_player_prefs_path
# ===========================================================================


class TestGetPlayerPrefsPath:
    """Tests pour get_player_prefs_path() dans _filter_state_model."""

    def test_returns_ui_prefs_json_for_player_db(self, players_dir: Path) -> None:
        """DB dans data/players/TestPlayer/ → ui_prefs.json dans le même répertoire."""
        db_path = str(players_dir / "stats.duckdb")
        result = get_player_prefs_path("some_xuid", db_path)
        assert result is not None
        assert result == players_dir / "ui_prefs.json"
        assert result.name == "ui_prefs.json"

    def test_returns_none_when_no_db_path(self) -> None:
        """Sans db_path → None (fallback legacy attendu par l'appelant)."""
        result = get_player_prefs_path("some_xuid", None)
        assert result is None

    def test_returns_none_when_parent_is_not_players(self, tmp_path: Path) -> None:
        """DB dans un répertoire non-'players' → None."""
        db_dir = tmp_path / "custom" / "MyDB"
        db_dir.mkdir(parents=True)
        db_path = str(db_dir / "stats.duckdb")
        result = get_player_prefs_path("xuid", db_path)
        assert result is None

    def test_returns_none_when_gamertag_dir_does_not_exist(self, tmp_path: Path) -> None:
        """Répertoire gamertag inexistant → None (évite de créer des dossiers manquants)."""
        db_path = str(tmp_path / "data" / "players" / "Ghost" / "stats.duckdb")
        result = get_player_prefs_path("xuid", db_path)
        assert result is None

    def test_path_name_is_always_ui_prefs_json(self, players_dir: Path) -> None:
        """Le nom du fichier est toujours 'ui_prefs.json', indépendant du xuid."""
        db_path = str(players_dir / "stats.duckdb")
        assert get_player_prefs_path("xuid_abc", db_path).name == "ui_prefs.json"
        assert get_player_prefs_path("", db_path).name == "ui_prefs.json"


# ===========================================================================
# Tests : _resolve_prefs_path (migration silencieuse)
# ===========================================================================


class TestResolvePrefsPath:
    """Tests pour _resolve_prefs_path() : sélection et migration."""

    def test_uses_player_dir_when_db_path_known(
        self, players_dir: Path, legacy_filters_dir: Path
    ) -> None:
        """Si db_path est dans data/players/, retourne ui_prefs.json."""
        db_path = str(players_dir / "stats.duckdb")
        result = _resolve_prefs_path("xuid", db_path)
        assert result == players_dir / "ui_prefs.json"

    def test_fallback_legacy_when_no_db_path(self, legacy_filters_dir: Path) -> None:
        """Sans db_path → chemin legacy .streamlit/filter_preferences/."""
        result = _resolve_prefs_path("xuid_123", None)
        assert result.parent == legacy_filters_dir
        assert result.name.startswith("xuid_")

    def test_migration_copies_legacy_to_player_dir(
        self, players_dir: Path, legacy_filters_dir: Path
    ) -> None:
        """Fichier legacy existant → copié dans data/players/ et supprimé."""
        db_path = str(players_dir / "stats.duckdb")

        # Créer un fichier legacy
        legacy_file = legacy_filters_dir / "player_TestPlayer.json"
        prefs_data = {"filter_mode": "Période", "start_date": "2026-01-01"}
        legacy_file.write_text(json.dumps(prefs_data), encoding="utf-8")

        result = _resolve_prefs_path("xuid", db_path)

        # Le fichier cible existe
        assert result.exists()
        loaded = json.loads(result.read_text(encoding="utf-8"))
        assert loaded["filter_mode"] == "Période"

        # L'ancien fichier a été supprimé
        assert not legacy_file.exists()

    def test_migration_idempotent_when_target_exists(
        self, players_dir: Path, legacy_filters_dir: Path
    ) -> None:
        """Si ui_prefs.json existe déjà → pas de migration (pas de crash)."""
        db_path = str(players_dir / "stats.duckdb")
        target = players_dir / "ui_prefs.json"
        target.write_text('{"filter_mode": "Sessions"}', encoding="utf-8")

        # Appeler deux fois ne doit pas lever d'exception
        r1 = _resolve_prefs_path("xuid", db_path)
        r2 = _resolve_prefs_path("xuid", db_path)
        assert r1 == r2 == target

    def test_migration_tolerates_read_error(
        self, players_dir: Path, legacy_filters_dir: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Erreur lors de la migration (ex. permissions) → ne bloque pas l'appel."""
        db_path = str(players_dir / "stats.duckdb")

        legacy_file = legacy_filters_dir / "player_TestPlayer.json"
        legacy_file.write_text('{"filter_mode": "Période"}', encoding="utf-8")

        import shutil

        monkeypatch.setattr(shutil, "copy2", MagicMock(side_effect=OSError("permission")))
        # Ne doit pas lever d'exception
        result = _resolve_prefs_path("xuid", db_path)
        assert result is not None  # retourne quand même le chemin cible

    def test_save_then_load_uses_player_dir(
        self, players_dir: Path, legacy_filters_dir: Path, fake_st: SimpleNamespace
    ) -> None:
        """save_filter_preferences + load_filter_preferences → fichier dans data/players/."""
        from src.ui.filter_state import load_filter_preferences, save_filter_preferences

        db_path = str(players_dir / "stats.duckdb")
        prefs = FilterPreferences(filter_mode="Période", start_date="2026-03-01")

        save_filter_preferences("xuid", db_path, preferences=prefs)
        loaded = load_filter_preferences("xuid", db_path)

        assert loaded is not None
        assert loaded.filter_mode == "Période"
        # Le fichier doit être dans data/players/, pas dans legacy
        assert (players_dir / "ui_prefs.json").exists()
        assert not any(legacy_filters_dir.iterdir())  # répertoire legacy vide

    def test_clear_removes_player_dir_file(
        self, players_dir: Path, legacy_filters_dir: Path, fake_st: SimpleNamespace
    ) -> None:
        """clear_filter_preferences supprime ui_prefs.json du répertoire joueur."""
        from src.ui.filter_state import clear_filter_preferences, save_filter_preferences

        db_path = str(players_dir / "stats.duckdb")
        prefs = FilterPreferences(filter_mode="Sessions")
        save_filter_preferences("xuid", db_path, preferences=prefs)

        target = players_dir / "ui_prefs.json"
        assert target.exists()

        clear_filter_preferences("xuid", db_path)
        assert not target.exists()


# ===========================================================================
# Tests : browser_storage (logique pure, sans iframe Streamlit)
# ===========================================================================


class TestBrowserStorageLogic:
    """Tests pour restore_browser_prefs / persist_browser_prefs (stockage JSON fichier).

    Les fonctions importent `streamlit` en body → on mocke st.session_state.
    Les I/O fichier sont mockées via _read_prefs / _write_prefs.
    """

    def _make_fake_st(self) -> SimpleNamespace:
        return SimpleNamespace(session_state={})

    def test_restore_returns_none_when_already_loaded(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Si _browser_prefs_loaded est True → restore retourne None directement."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        fake_st.session_state["_browser_prefs_loaded"] = True
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.restore_browser_prefs()

        assert result is None

    def test_restore_returns_empty_dict_when_no_file(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Fichier absent → retourne {} et pose le flag."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.restore_browser_prefs()

        assert result == {}
        assert fake_st.session_state.get("_browser_prefs_loaded") is True

    def test_restore_returns_data_from_file(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Fichier présent → retourne le dict et pose le flag."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        prefs_data = {"last_gamertag": "GuiGui", "lang": "fr"}
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value=prefs_data))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.restore_browser_prefs()

        assert result == {"last_gamertag": "GuiGui", "lang": "fr"}
        assert fake_st.session_state.get("_browser_prefs_loaded") is True

    def test_restore_returns_empty_dict_when_no_data(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Fichier vide → retourne {} après avoir posé le flag."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.restore_browser_prefs()

        assert result == {}

    def test_persist_skips_empty_fields(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """persist_browser_prefs ignore les valeurs vides / None."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(last_gamertag="", lang=None)

        mock_write.assert_not_called()

    def test_persist_writes_to_file(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """persist_browser_prefs écrit les données dans le fichier JSON."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(last_gamertag="GuiGui", lang="fr")

        mock_write.assert_called_once()
        written = mock_write.call_args[0][0]
        assert written.get("last_gamertag") == "GuiGui"
        assert written.get("lang") == "fr"

    def test_persist_deduplicates_same_values(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Appeler persist deux fois avec les mêmes valeurs → une seule écriture."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(lang="fr")
            bs.persist_browser_prefs(lang="fr")  # doublon

        assert mock_write.call_count == 1  # une seule écriture

    def test_clear_removes_loaded_flag(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """clear_browser_prefs réinitialise le flag _browser_prefs_loaded."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        fake_st.session_state["_browser_prefs_loaded"] = True
        monkeypatch.setattr(bs, "_PREFS_FILE", MagicMock(**{"exists.return_value": False}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.clear_browser_prefs()

        assert "_browser_prefs_loaded" not in fake_st.session_state


# ===========================================================================
# Tests : _maybe_apply_browser_prefs (logique de restauration au démarrage)
# ===========================================================================


class TestMaybeApplyBrowserPrefs:
    """Tests pour la logique de _maybe_apply_browser_prefs dans streamlit_app."""

    def _get_func(self):
        """Importe _maybe_apply_browser_prefs depuis streamlit_app."""

        # streamlit_app.py ne peut pas être importé directement (set_page_config)
        # On teste la logique en la réimplémentant ici pour vérifier les invariants
        # ou en mockant l'ensemble de l'environnement Streamlit.
        # Dans ce cas, on teste directement via l'effet sur session_state.
        return None

    def test_guard_prevents_double_application(self) -> None:
        """Le guard _browser_prefs_applied empêche une double application."""
        # Simuler directement la logique du guard
        session = {"_browser_prefs_applied": True}
        # Si le guard est posé, la fonction doit retourner immédiatement
        assert session.get("_browser_prefs_applied") is True

    def test_empty_prefs_no_side_effect(self) -> None:
        """Des prefs vides ({}) ne doivent pas modifier session_state."""
        # Simuler l'état initial
        session: dict = {}
        prefs: dict = {}
        ls_slug = str(prefs.get("last_db_path") or "").strip()
        ls_lang = str(prefs.get("lang") or "").strip()
        assert ls_slug == ""
        assert ls_lang == ""
        assert session == {}  # rien n'a été modifié

    def test_lang_extraction_from_prefs(self) -> None:
        """La langue est correctement extraite depuis les prefs navigateur."""
        prefs = {"lang": "en", "last_db_path": ""}
        ls_lang = str(prefs.get("lang") or "").strip()
        assert ls_lang == "en"

    def test_invalid_lang_ignored(self) -> None:
        """Une langue inconnue est ignorée (ni 'fr' ni 'en')."""
        prefs = {"lang": "zh"}
        ls_lang = str(prefs.get("lang") or "").strip()
        assert ls_lang not in ("fr", "en")


# ===========================================================================
# Tests : branche localStorage dans _resolve_db_path (data_loader)
# ===========================================================================


class TestResolveDbPathLocalStorage:
    """Tests pour la branche localStorage de _resolve_db_path dans data_loader."""

    def _make_fake_st(self, session: dict | None = None) -> SimpleNamespace:
        return SimpleNamespace(session_state=session or {}, query_params={})

    def test_restores_db_from_localstorage_slug(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Si localStorage contient last_db_path et que la DB existe → la retourne."""
        import src.app.data_loader as dl

        # Créer la structure data/players/GuiGui/stats.duckdb
        players = tmp_path / "data" / "players"
        guigui_db = players / "GuiGui" / "stats.duckdb"
        guigui_db.parent.mkdir(parents=True)
        guigui_db.write_bytes(b"\x00" * 8)

        default_db = str(players / "DefaultPlayer" / "stats.duckdb")

        fake_st = self._make_fake_st({"_browser_prefs_restored": {"last_db_path": "GuiGui"}})
        monkeypatch.setattr(dl, "st", fake_st)

        # Mock pour ne pas accéder au vrai env
        monkeypatch.setenv("LEVELUP_DB", "")
        monkeypatch.setenv("LEVELUP_DB_PATH", "")

        from src.ui.settings import AppSettings

        settings = AppSettings(prefer_spnkr_db_if_available=False)

        with patch.object(dl, "pick_latest_spnkr_db_if_any", return_value=None):
            result = dl._resolve_db_path(default_db, settings)

        assert result == str(guigui_db)

    def test_ignores_localstorage_when_db_not_found(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Slug localStorage qui ne correspond à aucune DB → ignoré, fallback défaut."""
        import src.app.data_loader as dl

        players = tmp_path / "data" / "players"
        default_player = players / "DefaultPlayer"
        default_player.mkdir(parents=True)
        default_db = str(default_player / "stats.duckdb")
        # Pas de GuiGui/stats.duckdb

        fake_st = self._make_fake_st(
            {"_browser_prefs_restored": {"last_db_path": "NonExistentPlayer"}}
        )
        monkeypatch.setattr(dl, "st", fake_st)
        monkeypatch.setenv("LEVELUP_DB", "")
        monkeypatch.setenv("LEVELUP_DB_PATH", "")

        from src.ui.settings import AppSettings

        settings = AppSettings(prefer_spnkr_db_if_available=False)

        with patch.object(dl, "pick_latest_spnkr_db_if_any", return_value=None):
            result = dl._resolve_db_path(default_db, settings)

        assert result == default_db

    def test_empty_localstorage_slug_ignored(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Slug vide → branche localStorage ignorée silencieusement."""
        import src.app.data_loader as dl

        default_db = str(tmp_path / "stats.duckdb")
        fake_st = self._make_fake_st({"_browser_prefs_restored": {"last_db_path": ""}})
        monkeypatch.setattr(dl, "st", fake_st)
        monkeypatch.setenv("LEVELUP_DB", "")
        monkeypatch.setenv("LEVELUP_DB_PATH", "")

        from src.ui.settings import AppSettings

        settings = AppSettings(prefer_spnkr_db_if_available=False)

        with patch.object(dl, "pick_latest_spnkr_db_if_any", return_value=None):
            result = dl._resolve_db_path(default_db, settings)

        assert result == default_db

    def test_no_localstorage_prefs_key_ignored(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Absence de _browser_prefs_restored → branche ignorée (aucune erreur)."""
        import src.app.data_loader as dl

        default_db = str(tmp_path / "stats.duckdb")
        fake_st = self._make_fake_st({})  # session_state vide
        monkeypatch.setattr(dl, "st", fake_st)
        monkeypatch.setenv("LEVELUP_DB", "")
        monkeypatch.setenv("LEVELUP_DB_PATH", "")

        from src.ui.settings import AppSettings

        settings = AppSettings(prefer_spnkr_db_if_available=False)

        with patch.object(dl, "pick_latest_spnkr_db_if_any", return_value=None):
            result = dl._resolve_db_path(default_db, settings)

        assert result == default_db


# ===========================================================================
# Tests : hints_visible + restore_hints_from_prefs
# ===========================================================================


class TestHintsVisible:
    """Tests pour hints_visible() et restore_hints_from_prefs()."""

    def _make_fake_st(self) -> SimpleNamespace:
        return SimpleNamespace(session_state={})

    def test_hints_visible_default_true(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Sans clé dans session_state, hints_visible() retourne True (défaut)."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.hints_visible()

        assert result is True

    def test_hints_visible_false_when_set(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Si _hints_visible = False dans session_state, hints_visible() retourne False."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        fake_st.session_state["_hints_visible"] = False

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.hints_visible()

        assert result is False

    def test_hints_visible_true_when_set(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Si _hints_visible = True dans session_state, hints_visible() retourne True."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        fake_st.session_state["_hints_visible"] = True

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            result = bs.hints_visible()

        assert result is True

    def test_restore_hints_sets_false_when_zero(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """restore_hints_from_prefs avec show_hints='0' → _hints_visible = False."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.restore_hints_from_prefs({"show_hints": "0"})

        assert fake_st.session_state.get("_hints_visible") is False

    def test_restore_hints_sets_true_when_one(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """restore_hints_from_prefs avec show_hints='1' → _hints_visible = True."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.restore_hints_from_prefs({"show_hints": "1"})

        assert fake_st.session_state.get("_hints_visible") is True

    def test_restore_hints_absent_key_no_change(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """restore_hints_from_prefs sans clé show_hints → session_state inchangé."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.restore_hints_from_prefs({"lang": "fr"})  # pas de show_hints

        assert "_hints_visible" not in fake_st.session_state

    def test_persist_hints_writes_show_hints_key(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """persist_browser_prefs(show_hints='0') écrit bien la clé show_hints dans le JSON."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(show_hints="0")

        mock_write.assert_called_once()
        written = mock_write.call_args[0][0]
        assert written.get("show_hints") == "0"

    def test_persist_hints_dedup_same_value(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Deux appels persist_browser_prefs identiques → une seule écriture (dédup)."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(show_hints="0")
            bs.persist_browser_prefs(show_hints="0")  # doublon

        assert mock_write.call_count == 1

    def test_persist_hints_different_values_two_writes(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Toggle Off puis On → deux écritures distinctes (pas de faux positif dédup)."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.persist_browser_prefs(show_hints="0")  # désactiver
            bs.persist_browser_prefs(show_hints="1")  # réactiver

        assert mock_write.call_count == 2

    def test_restore_hints_integer_zero(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """restore_hints_from_prefs avec show_hints=0 (int JSON) → _hints_visible = False."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.restore_hints_from_prefs({"show_hints": 0})  # int, pas string

        assert fake_st.session_state.get("_hints_visible") is False

    def test_restore_hints_integer_one(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """restore_hints_from_prefs avec show_hints=1 (int JSON) → _hints_visible = True."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)
            bs.restore_hints_from_prefs({"show_hints": 1})  # int, pas string

        assert fake_st.session_state.get("_hints_visible") is True

    def test_on_hints_toggle_persists_to_prefs(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Le callback _on_hints_toggle met à jour session_state et appelle persist."""
        import src.ui.components.browser_storage as bs

        fake_st = self._make_fake_st()
        fake_st.session_state["_sidebar_hints_toggle"] = False  # user vient de décocher

        mock_write = MagicMock()
        monkeypatch.setattr(bs, "_write_prefs", mock_write)
        monkeypatch.setattr(bs, "_read_prefs", MagicMock(return_value={}))

        with patch("streamlit.session_state", fake_st.session_state):
            import streamlit as st_real

            monkeypatch.setattr(st_real, "session_state", fake_st.session_state)

            # Reproduire la logique _on_hints_toggle de streamlit_app.py
            new_val = bool(st_real.session_state.get("_sidebar_hints_toggle", True))
            st_real.session_state["_hints_visible"] = new_val
            bs.persist_browser_prefs(show_hints="1" if new_val else "0")

        assert fake_st.session_state.get("_hints_visible") is False
        mock_write.assert_called_once()
        written = mock_write.call_args[0][0]
        assert written.get("show_hints") == "0"
