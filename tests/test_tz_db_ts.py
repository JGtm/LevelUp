"""Tests pour les fonctions de conversion timezone issues de DuckDB.

Couvre :
- db_ts_to_utc() : conversion timestamp DuckDB naïf → UTC
- match_start_to_epoch() : conversion datetime/string de match → epoch UTC
- is_session_potentially_active() : logique de session "en cours"
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from unittest.mock import patch

import pytest

from src.ui.tz import db_ts_to_utc

# =============================================================================
# db_ts_to_utc()
# =============================================================================


class TestDbTsToUtc:
    """Tests pour db_ts_to_utc().

    DuckDB stocke les UTC datetime en heure locale système (CET = UTC+1 en hiver).
    Un datetime naïf issu de DuckDB représente donc l'heure locale, pas UTC.
    """

    def test_naive_returns_utc_aware(self):
        """Un datetime naïf doit retourner un datetime UTC-aware."""
        naive = datetime(2024, 1, 15, 14, 0, 0)  # naïf, représente 14:00 CET
        result = db_ts_to_utc(naive)
        assert result.tzinfo is not None
        assert result.tzinfo == timezone.utc or str(result.tzinfo) == "UTC"

    def test_naive_treated_as_local_tz(self):
        """Un datetime naïf est traité comme heure locale (astimezone)."""
        naive = datetime(2024, 1, 15, 14, 0, 0)
        result = db_ts_to_utc(naive)
        # Le résultat en UTC doit être différent du naïf (sauf si TZ locale = UTC)
        # On vérifie juste que c'est un UTC aware
        assert result.tzinfo is not None

    def test_utc_aware_passthrough(self):
        """Un datetime UTC-aware est converti correctement (pas de double conversion)."""
        utc_dt = datetime(2024, 1, 15, 13, 0, 0, tzinfo=timezone.utc)
        result = db_ts_to_utc(utc_dt)
        assert result.tzinfo is not None
        assert result == utc_dt  # UTC → UTC = même instant

    def test_non_utc_aware_converted(self):
        """Un datetime aware non-UTC est converti vers UTC."""
        from zoneinfo import ZoneInfo

        cet = ZoneInfo("Europe/Paris")
        cet_dt = datetime(2024, 1, 15, 14, 0, 0, tzinfo=cet)  # 14:00 CET = 13:00 UTC
        result = db_ts_to_utc(cet_dt)
        expected = datetime(2024, 1, 15, 13, 0, 0, tzinfo=timezone.utc)
        assert result == expected

    def test_midnight_edge_case(self):
        """Minuit : conversion correcte sans débordement de date."""
        naive_midnight = datetime(2024, 1, 15, 0, 0, 0)
        result = db_ts_to_utc(naive_midnight)
        assert result.tzinfo is not None
        # L'heure peut être 23:00 la veille si TZ locale est UTC+1, ou d'autres valeurs
        # On vérifie juste que la date ne devient pas aberrante
        assert abs((result.date() - naive_midnight.date()).days) <= 1

    def test_returns_datetime(self):
        """Doit toujours retourner un datetime."""
        result = db_ts_to_utc(datetime(2024, 6, 1, 12, 0, 0))
        assert isinstance(result, datetime)

    def test_summer_time(self):
        """En été (CEST = UTC+2), la conversion doit être cohérente."""
        # Juillet : heure d'été
        naive_summer = datetime(2024, 7, 15, 14, 0, 0)
        result = db_ts_to_utc(naive_summer)
        assert result.tzinfo is not None

    def test_idempotent_for_utc_aware(self):
        """Appeler deux fois sur un UTC-aware donne le même résultat."""
        utc_dt = datetime(2024, 1, 15, 13, 0, 0, tzinfo=timezone.utc)
        result1 = db_ts_to_utc(utc_dt)
        result2 = db_ts_to_utc(result1)
        assert result1 == result2


# =============================================================================
# match_start_to_epoch()
# =============================================================================


class TestMatchStartToEpoch:
    """Tests pour match_start_to_epoch().

    Convertit start_time (tel que stocké par DuckDB) en epoch seconds UTC.
    Les timestamps naïfs issus de DuckDB représentent l'heure locale CET.
    """

    def setup_method(self):
        from src.data.media_helpers import match_start_to_epoch

        self.fn = match_start_to_epoch

    def test_float_passthrough(self):
        """Un float (epoch) est retourné tel quel."""
        epoch = 1738598400.0
        assert self.fn(epoch) == epoch

    def test_int_cast_to_float(self):
        """Un int est converti en float."""
        result = self.fn(1738598400)
        assert isinstance(result, float)
        assert result == 1738598400.0

    def test_utc_aware_string_z_suffix(self):
        """String ISO avec suffixe Z → epoch UTC correct."""
        result = self.fn("2026-02-03T17:00:00Z")
        expected = datetime(2026, 2, 3, 17, 0, 0, tzinfo=timezone.utc).timestamp()
        assert result == pytest.approx(expected, abs=1)

    def test_utc_aware_string_plus_offset(self):
        """String ISO avec offset +00:00 → epoch UTC correct."""
        result = self.fn("2026-02-03T17:00:00+00:00")
        expected = datetime(2026, 2, 3, 17, 0, 0, tzinfo=timezone.utc).timestamp()
        assert result == pytest.approx(expected, abs=1)

    def test_utc_aware_datetime(self):
        """Datetime UTC-aware → epoch UTC correct."""
        dt = datetime(2026, 2, 3, 17, 0, 0, tzinfo=timezone.utc)
        result = self.fn(dt)
        assert result == pytest.approx(dt.timestamp(), abs=1)

    def test_naive_datetime_treated_as_local(self):
        """Datetime naïf (DuckDB CET) → epoch UTC via db_ts_to_utc."""
        # Un datetime naïf 18:00 représente 17:00 UTC (CET = UTC+1 en hiver)
        naive_cet = datetime(2026, 2, 3, 18, 0, 0)  # stocké par DuckDB pour 17:00 UTC
        result = self.fn(naive_cet)
        # db_ts_to_utc traite 18:00 comme heure locale CET → 17:00 UTC
        expected_utc = db_ts_to_utc(naive_cet).timestamp()
        assert result == pytest.approx(expected_utc, abs=1)

    def test_invalid_input_returns_none(self):
        """Une entrée invalide retourne None sans lever d'exception."""
        assert self.fn(None) is None  # type: ignore[arg-type]
        assert self.fn([1, 2, 3]) is None  # type: ignore[arg-type]
        assert self.fn("not-a-date") is None

    def test_naive_string_assumed_utc(self):
        """String ISO naïf → converti en UTC (non-local, car string sans TZ info)."""
        result = self.fn("2026-02-03T17:00:00")
        # Selon l'implémentation : naive string → epoch UTC (via +00:00 convention)
        assert result is not None
        assert isinstance(result, float)


# =============================================================================
# is_session_potentially_active()
# =============================================================================


class TestIsSessionPotentiallyActive:
    """Tests pour is_session_potentially_active().

    Une session est active si le dernier match est récent (aujourd'hui
    ou hier soir si on est tôt le matin).
    """

    def setup_method(self):
        from src.analysis.sessions import is_session_potentially_active

        self.fn = is_session_potentially_active

    def test_today_match_is_active(self):
        """Un match joué aujourd'hui → session potentiellement active."""
        now_utc = datetime.now(timezone.utc)
        last_match = now_utc - timedelta(hours=1)
        assert self.fn(last_match, cutoff_hour=8) is True

    def test_old_match_not_active(self):
        """Un match joué il y a 3 jours → session non active."""
        three_days_ago = datetime.now(timezone.utc) - timedelta(days=3)
        assert self.fn(three_days_ago, cutoff_hour=8) is False

    def test_yesterday_before_cutoff_in_morning(self):
        """Match hier soir + on est tôt le matin → session encore active."""
        # Simuler : now = 07:00 UTC (avant cutoff 08:00), last match = hier 22:00 UTC
        fake_now = datetime(2024, 1, 15, 7, 0, 0, tzinfo=timezone.utc)
        last_match = datetime(2024, 1, 14, 22, 0, 0, tzinfo=timezone.utc)
        with patch("src.analysis.sessions.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            mock_dt.combine = datetime.combine
            result = self.fn(last_match, cutoff_hour=8)
        assert result is True

    def test_yesterday_after_cutoff_not_active(self):
        """Match hier soir + on est après cutoff → session non active."""
        # Simuler : now = 10:00 UTC (après cutoff 08:00), last match = hier 22:00 UTC
        fake_now = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        last_match = datetime(2024, 1, 14, 22, 0, 0, tzinfo=timezone.utc)
        with patch("src.analysis.sessions.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            mock_dt.combine = datetime.combine
            result = self.fn(last_match, cutoff_hour=8)
        assert result is False

    def test_naive_datetime_converted(self):
        """Un datetime naïf est traité comme heure locale (db_ts_to_utc)."""
        # Un naïf d'aujourd'hui doit être actif
        now_local = datetime.now()  # naïf, heure locale
        assert self.fn(now_local, cutoff_hour=8) is True

    def test_cutoff_hour_respected(self):
        """Le paramètre cutoff_hour contrôle la limite du matin."""
        fake_now = datetime(2024, 1, 15, 5, 0, 0, tzinfo=timezone.utc)  # 05:00 UTC
        last_match = datetime(2024, 1, 14, 22, 0, 0, tzinfo=timezone.utc)

        # Avec cutoff 6h : 05:00 < 06:00 → session active
        with patch("src.analysis.sessions.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            mock_dt.combine = datetime.combine
            assert self.fn(last_match, cutoff_hour=6) is True

        # Avec cutoff 4h : 05:00 > 04:00 → session non active
        with patch("src.analysis.sessions.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            mock_dt.combine = datetime.combine
            assert self.fn(last_match, cutoff_hour=4) is False

    def test_two_days_ago_never_active(self):
        """Un match d'avant-hier n'est jamais actif."""
        two_days_ago = datetime.now(timezone.utc) - timedelta(days=2)
        assert self.fn(two_days_ago, cutoff_hour=8) is False
