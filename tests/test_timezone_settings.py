"""Tests pour src/ui/tz.py et le champ user_timezone dans AppSettings.

Couvre :
- src/ui/tz.py : get_tz_name(), get_tz(), CURATED_TZ_LIST, logger
- src/ui/settings.py : champ user_timezone, validation IANA, fallback
- src/ui/_cache_loading.py : _convert_timezone respecte tz_name
- Intégration : get_tz_name() lit session_state correctement
"""

from __future__ import annotations

import logging
from datetime import datetime
from unittest.mock import MagicMock, patch
from zoneinfo import ZoneInfo

import polars as pl

from src.ui.settings import AppSettings

# =============================================================================
# Tests AppSettings.user_timezone
# =============================================================================


class TestAppSettingsTimezone:
    """Champ user_timezone dans AppSettings."""

    def test_default_is_europe_paris(self):
        """La valeur par défaut est 'Europe/Paris'."""
        s = AppSettings()
        assert s.user_timezone == "Europe/Paris"

    def test_valid_iana_accepted(self):
        """Une timezone IANA valide est acceptée telle quelle."""
        s = AppSettings(user_timezone="America/New_York")
        assert s.user_timezone == "America/New_York"

    def test_utc_accepted(self):
        """'UTC' est une timezone valide."""
        s = AppSettings(user_timezone="UTC")
        assert s.user_timezone == "UTC"

    def test_invalid_tz_falls_back_to_paris(self):
        """Une timezone invalide est remplacée par 'Europe/Paris'."""
        s = AppSettings(user_timezone="Not/ATimezone")
        assert s.user_timezone == "Europe/Paris"

    def test_empty_string_falls_back_to_paris(self):
        """Chaîne vide → fallback 'Europe/Paris'."""
        s = AppSettings(user_timezone="")
        assert s.user_timezone == "Europe/Paris"

    def test_none_falls_back_to_paris(self):
        """None → fallback 'Europe/Paris' (via strip_none_values)."""
        s = AppSettings(user_timezone=None)
        assert s.user_timezone == "Europe/Paris"

    def test_whitespace_falls_back_to_paris(self):
        """Espaces seuls → fallback 'Europe/Paris'."""
        s = AppSettings(user_timezone="   ")
        assert s.user_timezone == "Europe/Paris"

    def test_all_curated_tz_are_valid(self):
        """Toutes les TZ de CURATED_TZ_LIST sont acceptées par le validateur."""
        from src.ui.tz import CURATED_TZ_LIST

        for tz in CURATED_TZ_LIST:
            s = AppSettings(user_timezone=tz)
            assert s.user_timezone == tz, f"TZ '{tz}' rejetée par le validateur"

    def test_serialization_roundtrip(self, tmp_path):
        """user_timezone survit à une sérialisation JSON → désérialisation."""

        from src.ui.settings import load_settings, save_settings

        path = tmp_path / "test_settings.json"
        s = AppSettings(user_timezone="Asia/Tokyo")

        with patch("src.ui.settings.get_settings_path", return_value=str(path)):
            ok, _ = save_settings(s)
        assert ok

        with patch("src.ui.settings.get_settings_path", return_value=str(path)):
            loaded = load_settings()
        assert loaded.user_timezone == "Asia/Tokyo"


# =============================================================================
# Tests src/ui/tz.py
# =============================================================================


class TestTzModule:
    """Tests pour src/ui/tz.py."""

    def test_has_logger(self):
        """Le module tz.py expose un logger."""
        import src.ui.tz as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)

    def test_has_default_tz_constant(self):
        """_DEFAULT_TZ vaut 'Europe/Paris'."""
        from src.ui.tz import _DEFAULT_TZ

        assert _DEFAULT_TZ == "Europe/Paris"

    def test_curated_tz_list_type(self):
        """CURATED_TZ_LIST est une liste de chaînes non vides."""
        from src.ui.tz import CURATED_TZ_LIST

        assert isinstance(CURATED_TZ_LIST, list)
        assert len(CURATED_TZ_LIST) >= 10
        for tz in CURATED_TZ_LIST:
            assert isinstance(tz, str)
            assert tz.strip(), "TZ vide dans CURATED_TZ_LIST"

    def test_curated_tz_list_contains_paris(self):
        """CURATED_TZ_LIST contient 'Europe/Paris'."""
        from src.ui.tz import CURATED_TZ_LIST

        assert "Europe/Paris" in CURATED_TZ_LIST

    def test_curated_tz_list_contains_utc(self):
        """CURATED_TZ_LIST contient 'UTC'."""
        from src.ui.tz import CURATED_TZ_LIST

        assert "UTC" in CURATED_TZ_LIST

    def test_curated_tz_list_all_valid_zoneinfo(self):
        """Toutes les TZ de CURATED_TZ_LIST sont des ZoneInfo valides."""
        from src.ui.tz import CURATED_TZ_LIST

        for tz in CURATED_TZ_LIST:
            ZoneInfo(tz)  # lève ZoneInfoNotFoundError si invalide

    def test_get_tz_name_fallback_without_streamlit(self):
        """get_tz_name() retourne le fallback quand streamlit n'est pas disponible."""
        from src.ui.tz import get_tz_name

        # Sans session_state streamlit actif, doit retourner le défaut
        result = get_tz_name()
        assert result == "Europe/Paris"

    def test_get_tz_name_reads_session_state(self):
        """get_tz_name() lit user_timezone depuis session_state."""
        from src.ui import tz as tz_mod

        mock_settings = MagicMock()
        mock_settings.user_timezone = "America/Los_Angeles"
        mock_st = MagicMock()
        mock_st.session_state.get.return_value = mock_settings

        with patch.dict("sys.modules", {"streamlit": mock_st}):
            # Recharge get_tz_name pour qu'il utilise le mock
            import importlib

            importlib.reload(tz_mod)
            result = tz_mod.get_tz_name()
            assert result == "America/Los_Angeles"

        # Recharger pour remettre le module dans l'état normal
        importlib.reload(tz_mod)

    def test_get_tz_name_fallback_when_settings_none(self):
        """get_tz_name() retourne le fallback quand session_state["app_settings"] est None."""
        from src.ui.tz import get_tz_name

        mock_st = MagicMock()
        mock_st.session_state.get.return_value = None

        with patch.dict("sys.modules", {"streamlit": mock_st}):
            result = get_tz_name()
        assert result == "Europe/Paris"

    def test_get_tz_returns_zoneinfo(self):
        """get_tz() retourne un ZoneInfo (avec fallback)."""
        from src.ui.tz import get_tz

        result = get_tz()
        assert isinstance(result, ZoneInfo)

    def test_get_tz_name_empty_user_timezone_returns_fallback(self):
        """get_tz_name() retourne fallback si user_timezone est vide."""
        from src.ui.tz import get_tz_name

        mock_settings = MagicMock()
        mock_settings.user_timezone = ""
        mock_st = MagicMock()
        mock_st.session_state.get.return_value = mock_settings

        with patch.dict("sys.modules", {"streamlit": mock_st}):
            result = get_tz_name()
        assert result == "Europe/Paris"


# =============================================================================
# Tests utc_to_local / local_to_utc
# =============================================================================


class TestUtcToLocal:
    """Tests pour src.ui.tz.utc_to_local()."""

    def test_naive_treated_as_utc(self):
        """Un datetime naïf est traité comme UTC."""
        from src.ui.tz import utc_to_local

        naive_utc = datetime(2025, 7, 1, 12, 0, 0)
        result = utc_to_local(naive_utc)
        # Fallback Paris = UTC+2 en été
        assert result.hour == 14
        assert result.tzinfo is not None

    def test_aware_utc_converted(self):
        """Un datetime aware UTC est converti."""
        from datetime import timezone

        from src.ui.tz import utc_to_local

        aware_utc = datetime(2025, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        result = utc_to_local(aware_utc)
        # Paris = UTC+1 en hiver
        assert result.hour == 11

    def test_aware_other_tz(self):
        """Un datetime aware dans une autre TZ est converti en TZ utilisateur."""
        from src.ui.tz import utc_to_local

        # 10h Tokyo (UTC+9) = 01h UTC = 02h Paris (hiver)
        tokyo = datetime(2025, 1, 15, 10, 0, 0, tzinfo=ZoneInfo("Asia/Tokyo"))
        result = utc_to_local(tokyo)
        assert result.hour == 2


class TestLocalToUtc:
    """Tests pour src.ui.tz.local_to_utc()."""

    def test_naive_treated_as_user_tz(self):
        """Un datetime naïf est traité comme TZ utilisateur."""
        from src.ui.tz import local_to_utc

        naive_local = datetime(2025, 7, 1, 14, 0, 0)
        result = local_to_utc(naive_local)
        # Paris UTC+2 en été → 12h UTC
        assert result.hour == 12

    def test_aware_converted(self):
        """Un datetime aware est converti en UTC."""
        from src.ui.tz import local_to_utc

        aware_paris = datetime(2025, 1, 15, 11, 0, 0, tzinfo=ZoneInfo("Europe/Paris"))
        result = local_to_utc(aware_paris)
        assert result.hour == 10

    def test_roundtrip(self):
        """utc_to_local → local_to_utc est un round-trip."""
        from datetime import timezone

        from src.ui.tz import local_to_utc, utc_to_local

        original = datetime(2025, 3, 15, 8, 30, 0, tzinfo=timezone.utc)
        local = utc_to_local(original)
        back = local_to_utc(local)
        assert abs((back - original).total_seconds()) < 1


# =============================================================================
# Tests _convert_timezone respecte tz_name
# =============================================================================


class TestConvertTimezone:
    """Tests pour _convert_timezone avec tz_name dynamique."""

    def test_convert_utc_to_paris(self):
        """UTC 13:00 → Paris 14:00 (CET +1h)."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": [datetime(2024, 1, 15, 13, 0, 0)]}).with_columns(
            pl.col("start_time").dt.replace_time_zone("UTC")
        )

        result = _convert_timezone(df, "Europe/Paris")
        # 13:00 UTC → 14:00 CET
        assert result["start_time"][0].hour == 14

    def test_convert_utc_to_new_york(self):
        """UTC 13:00 → New York 08:00 (EST -5h)."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": [datetime(2024, 1, 15, 13, 0, 0)]}).with_columns(
            pl.col("start_time").dt.replace_time_zone("UTC")
        )

        result = _convert_timezone(df, "America/New_York")
        # 13:00 UTC → 08:00 EST
        assert result["start_time"][0].hour == 8

    def test_convert_adds_date_column(self):
        """_convert_timezone ajoute la colonne 'date'."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": [datetime(2024, 6, 1, 12, 0, 0)]}).with_columns(
            pl.col("start_time").dt.replace_time_zone("UTC")
        )

        result = _convert_timezone(df, "Europe/Paris")
        assert "date" in result.columns

    def test_convert_result_is_naive(self):
        """Après conversion, start_time est naïf (pas de timezone attachée)."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": [datetime(2024, 1, 15, 13, 0, 0)]}).with_columns(
            pl.col("start_time").dt.replace_time_zone("UTC")
        )

        result = _convert_timezone(df, "Europe/Paris")
        # Le dtype ne doit pas avoir un time_zone attaché
        assert result["start_time"].dtype.time_zone is None

    def test_convert_default_is_paris(self):
        """Le défaut de _convert_timezone est 'Europe/Paris'."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": [datetime(2024, 1, 15, 13, 0, 0)]}).with_columns(
            pl.col("start_time").dt.replace_time_zone("UTC")
        )

        result_explicit = _convert_timezone(df.clone(), "Europe/Paris")
        result_default = _convert_timezone(df.clone())
        assert result_explicit["start_time"][0] == result_default["start_time"][0]

    def test_convert_empty_df_no_crash(self):
        """_convert_timezone sur DataFrame vide ne lève pas d'exception."""
        from src.ui._cache_loading import _convert_timezone

        df = pl.DataFrame({"start_time": pl.Series([], dtype=pl.Datetime("us", "UTC"))})
        result = _convert_timezone(df, "America/Tokyo")
        assert isinstance(result, pl.DataFrame)


# =============================================================================
# Tests cached_friend_matches_df accepte tz_name
# =============================================================================


class TestCachedFriendMatchesDfSignature:
    """Vérifie la signature de cached_friend_matches_df."""

    def test_has_tz_name_parameter(self):
        """cached_friend_matches_df a un paramètre tz_name."""
        import inspect

        from src.ui.cache_filters import cached_friend_matches_df

        sig = inspect.signature(cached_friend_matches_df)
        assert "tz_name" in sig.parameters

    def test_tz_name_default_is_paris(self):
        """Le défaut de tz_name est 'Europe/Paris'."""
        import inspect

        from src.ui.cache_filters import cached_friend_matches_df

        param = inspect.signature(cached_friend_matches_df).parameters["tz_name"]
        assert param.default == "Europe/Paris"
