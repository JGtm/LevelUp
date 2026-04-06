"""Tests unitaires pour playable_duration_seconds et real_start_time (v6.3).

Couvre :
- Extraction + parsing de PlayableDuration depuis le JSON match
- Calcul de real_start_time (début effectif du gameplay)
- Guard contre des données incohérentes (playable > duration)
- Fallback end_time depuis l'API (EndTime)
- Migration idempotente via ensure_match_registry_playable_duration
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import Any

import duckdb
import pytest

from src.data.sync.migrations import ensure_match_registry_playable_duration
from src.data.sync.transformers import extract_match_registry_data


# =============================================================================
# Helpers
# =============================================================================

_BASE_JSON: dict[str, Any] = {
    "MatchId": "test-match-pd-001",
    "MatchInfo": {
        "StartTime": "2024-06-15T18:30:00Z",
        "EndTime": "2024-06-15T18:42:30Z",
        "Duration": "PT12M30S",  # 750 s
        "PlayableDuration": "PT8M0S",  # 480 s → countdown = 270 s
    },
}


def _make_json(**match_info_overrides: Any) -> dict[str, Any]:
    """Retourne un JSON de match avec les overrides appliqués à MatchInfo."""
    mi = dict(_BASE_JSON["MatchInfo"])
    mi.update(match_info_overrides)
    return {"MatchId": _BASE_JSON["MatchId"], "MatchInfo": mi}


# =============================================================================
# Tests parsing PlayableDuration
# =============================================================================


class TestExtractPlayableDuration:
    """Vérifie que PlayableDuration est parsé et retourné correctement."""

    def test_standard_playable_duration(self) -> None:
        """PT8M0S → 480 secondes."""
        result = extract_match_registry_data(_BASE_JSON)
        assert result is not None
        assert result["playable_duration_seconds"] == 480

    def test_absent_playable_duration(self) -> None:
        """PlayableDuration absent → None, pas d'exception."""
        data = _make_json()
        del data["MatchInfo"]["PlayableDuration"]
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["playable_duration_seconds"] is None

    def test_non_string_playable_duration(self) -> None:
        """PlayableDuration non-string (ex. None) → traité comme absent → None."""
        data = _make_json(PlayableDuration=None)
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["playable_duration_seconds"] is None

    def test_unparseable_playable_duration_logs_warning(
        self, caplog: pytest.LogCaptureFixture
    ) -> None:
        """PlayableDuration invalide → None + WARNING loggé."""
        data = _make_json(PlayableDuration="NOT_A_DURATION")
        with caplog.at_level(logging.WARNING):
            result = extract_match_registry_data(data)
        assert result is not None
        assert result["playable_duration_seconds"] is None
        assert any("PlayableDuration non parsé" in r.message for r in caplog.records)

    def test_playable_duration_variant_seconds(self) -> None:
        """PT510S → 510 secondes."""
        data = _make_json(PlayableDuration="PT510S", Duration="PT15M0S")
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["playable_duration_seconds"] == 510


# =============================================================================
# Tests calcul real_start_time
# =============================================================================


class TestRealStartTime:
    """Vérifie le calcul de real_start_time."""

    def test_real_start_time_calculated(self) -> None:
        """real_start_time = start_time + countdown (270 s)."""
        result = extract_match_registry_data(_BASE_JSON)
        assert result is not None
        assert result["real_start_time"] is not None
        start = result["start_time"]
        real = result["real_start_time"]
        # countdown = 750 − 480 = 270 s
        delta = (real - start).total_seconds()
        assert abs(delta - 270.0) < 1.0

    def test_real_start_time_none_when_playable_absent(self) -> None:
        """real_start_time est None si PlayableDuration absent."""
        data = _make_json()
        del data["MatchInfo"]["PlayableDuration"]
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["real_start_time"] is None

    def test_real_start_time_guard_playable_exceeds_duration(
        self, caplog: pytest.LogCaptureFixture
    ) -> None:
        """real_start_time = None si playable > duration (countdown négatif)."""
        # PlayableDuration > Duration — données incohérentes
        data = _make_json(Duration="PT5M0S", PlayableDuration="PT8M0S")
        with caplog.at_level(logging.WARNING):
            result = extract_match_registry_data(data)
        assert result is not None
        assert result["real_start_time"] is None
        assert any("real_start_time ignoré" in r.message for r in caplog.records)

    def test_real_start_time_zero_countdown(self) -> None:
        """Pas de countdown (playable == duration) → real_start_time == start_time."""
        data = _make_json(Duration="PT8M0S", PlayableDuration="PT8M0S")
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["real_start_time"] == result["start_time"]

    def test_real_start_time_debug_logged(self, caplog: pytest.LogCaptureFixture) -> None:
        """Un log DEBUG est émis quand real_start_time est calculé."""
        with caplog.at_level(logging.DEBUG, logger="src.data.sync.transformers._match"):
            result = extract_match_registry_data(_BASE_JSON)
        assert result is not None
        assert result["real_start_time"] is not None
        assert any("countdown" in r.message for r in caplog.records)


# =============================================================================
# Tests end_time depuis l'API
# =============================================================================


class TestEndTimeFromApi:
    """Vérifie que EndTime de l'API prend la priorité sur le calcul."""

    def test_end_time_uses_api_value(self) -> None:
        """EndTime de l'API est utilisé directement."""
        result = extract_match_registry_data(_BASE_JSON)
        assert result is not None
        # EndTime API = 2024-06-15T18:42:30Z
        expected = datetime(2024, 6, 15, 18, 42, 30, tzinfo=timezone.utc)
        assert result["end_time"] == expected

    def test_end_time_fallback_when_absent(self) -> None:
        """end_time calculé depuis start_time + duration si EndTime absent."""
        data = _make_json()
        del data["MatchInfo"]["EndTime"]
        result = extract_match_registry_data(data)
        assert result is not None
        assert result["end_time"] is not None
        # start = 18:30:00 + 750 s = 18:42:30
        expected = datetime(2024, 6, 15, 18, 42, 30, tzinfo=timezone.utc)
        assert result["end_time"] == expected

    def test_end_time_none_when_start_missing(self) -> None:
        """end_time est None si start_time absent et EndTime absent."""
        result = extract_match_registry_data(
            {
                "MatchId": "test-no-start",
                "MatchInfo": {"Duration": "PT10M0S"},
            }
        )
        # extract_match_registry_data retourne None si StartTime absent
        assert result is None


# =============================================================================
# Tests migration idempotente
# =============================================================================


class TestEnsurePlayableDurationMigration:
    """Vérifie l'idempotence et le comportement de la migration."""

    @pytest.fixture
    def mem_conn(self) -> duckdb.DuckDBPyConnection:
        """Connexion DuckDB en mémoire avec match_registry minimale."""
        conn = duckdb.connect(":memory:")
        conn.execute(
            """
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                duration_seconds INTEGER
            )
            """
        )
        return conn

    def test_columns_created(self, mem_conn: duckdb.DuckDBPyConnection) -> None:
        """Les deux colonnes sont absentes puis créées."""
        ensure_match_registry_playable_duration(mem_conn)
        cols = {
            row[0]
            for row in mem_conn.execute(
                "SELECT column_name FROM information_schema.columns WHERE table_name = 'match_registry'"
            ).fetchall()
        }
        assert "playable_duration_seconds" in cols
        assert "real_start_time" in cols

    def test_idempotent_double_call(self, mem_conn: duckdb.DuckDBPyConnection) -> None:
        """Appeler la migration deux fois ne lève pas d'erreur."""
        ensure_match_registry_playable_duration(mem_conn)
        ensure_match_registry_playable_duration(mem_conn)
        # Si on arrive ici sans exception, l'idempotence est confirmée

    def test_noop_when_table_absent(self) -> None:
        """Ne lève pas d'erreur si match_registry n'existe pas."""
        conn = duckdb.connect(":memory:")
        # Pas de table match_registry
        ensure_match_registry_playable_duration(conn)
        # Aucune exception levée

    def test_columns_type(self, mem_conn: duckdb.DuckDBPyConnection) -> None:
        """playable_duration_seconds est INTEGER, real_start_time est TIMESTAMP."""
        ensure_match_registry_playable_duration(mem_conn)
        rows = mem_conn.execute(
            """
            SELECT column_name, data_type
            FROM information_schema.columns
            WHERE table_name = 'match_registry'
              AND column_name IN ('playable_duration_seconds', 'real_start_time')
            ORDER BY column_name
            """
        ).fetchall()
        col_types = {row[0]: row[1] for row in rows}
        assert "INTEGER" in col_types.get("playable_duration_seconds", "")
        assert "TIMESTAMP" in col_types.get("real_start_time", "")

    def test_existing_data_preserved(self, mem_conn: duckdb.DuckDBPyConnection) -> None:
        """La migration ne supprime pas les données existantes."""
        mem_conn.execute(
            "INSERT INTO match_registry (match_id, duration_seconds) VALUES ('m1', 750)"
        )
        ensure_match_registry_playable_duration(mem_conn)
        count = mem_conn.execute("SELECT COUNT(*) FROM match_registry").fetchone()[0]
        assert count == 1
