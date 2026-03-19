"""Tests — LUSR UPSERT vectorisé (Axe 6).

Vérifie que :
- conn.executemany() est appelé une seule fois (batch vectorisé)
- conn.execute() N'EST PAS appelé (anti-pattern per-row supprimé)
- Le guard-rail ±100 pts reste séquentiel (delta capé)
- Les filtres existing_csr_ids / existing_lusr_ids sont respectés
- force=True ignore existing_lusr_ids
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import MagicMock

import polars as pl
import pytest

from src.data.sync._skill_rating import _LUSR_AVAILABLE, SkillRatingMixin

pytestmark = pytest.mark.skipif(
    not _LUSR_AVAILABLE,
    reason="Modules LUSR non disponibles",
)

_NOW = datetime(2025, 1, 1, 12, 0, 0, tzinfo=timezone.utc)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _ratings_df(*rows: dict) -> pl.DataFrame:
    """Construit un DataFrame minimal pour _upsert_lusr_ratings."""
    default = {
        "match_id": "m1",
        "rating_value": 1500.0,
        "rating_deviation": 50.0,
        "playlist_group": "social",
    }
    records = [{**default, **r} for r in rows]
    return pl.DataFrame(records)


def _matches_df(*match_ids: str) -> pl.DataFrame:
    return pl.DataFrame(
        {
            "match_id": list(match_ids),
            "start_time": [_NOW] * len(match_ids),
        }
    )


def _mock_conn() -> MagicMock:
    conn = MagicMock()
    conn.executemany = MagicMock()
    conn.execute = MagicMock()
    conn.commit = MagicMock()
    return conn


# ---------------------------------------------------------------------------
# Tests vectorisation
# ---------------------------------------------------------------------------


class TestLUSRVectorized:
    """executemany remplace les execute() par-ligne."""

    def test_executemany_called_once_for_batch(self):
        conn = _mock_conn()
        df = _ratings_df(
            {"match_id": "m1", "rating_value": 1500.0},
            {"match_id": "m2", "rating_value": 1510.0},
            {"match_id": "m3", "rating_value": 1520.0},
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,  # self non utilisé
            conn,
            df,
            _matches_df("m1", "m2", "m3"),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        conn.executemany.assert_called_once()
        conn.execute.assert_not_called()
        assert result == 3

    def test_execute_not_called_per_row(self):
        """Anti-pattern per-row : conn.execute() ne doit jamais être appelé."""
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": [f"m{i}" for i in range(10)],
                "rating_value": [float(1500 + i * 5) for i in range(10)],
                "rating_deviation": [50.0] * 10,
                "playlist_group": ["social"] * 10,
            }
        )
        SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df(*[f"m{i}" for i in range(10)]),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        conn.execute.assert_not_called()
        assert conn.executemany.call_count == 1

    def test_no_db_call_when_all_filtered(self):
        """Si toutes les lignes sont filtrées, ni executemany ni commit."""
        conn = _mock_conn()
        df = _ratings_df(
            {"match_id": "m1"},
            {"match_id": "m2"},
        )
        SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids={"m1", "m2"},
            existing_lusr_ids=set(),
            force=False,
        )
        conn.executemany.assert_not_called()
        conn.commit.assert_not_called()


# ---------------------------------------------------------------------------
# Tests guard-rail ±100
# ---------------------------------------------------------------------------


class TestLUSRGuardRail:
    """Le cap séquentiel ±100 pts doit être préservé après vectorisation."""

    def test_delta_within_limit_not_capped(self):
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m2"],
                "rating_value": [1500.0, 1580.0],  # delta = +80 < 100
                "rating_deviation": [50.0, 50.0],
                "playlist_group": ["social", "social"],
            }
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        assert result == 2
        rows = conn.executemany.call_args[0][1]
        # m1: rating=1500, delta=None (premier)
        assert rows[0][1] == pytest.approx(1500.0)
        assert rows[0][7] is None  # delta
        # m2: rating=1580, delta=80
        assert rows[1][1] == pytest.approx(1580.0)
        assert rows[1][7] == pytest.approx(80.0)

    def test_delta_over_limit_capped_positive(self):
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m2"],
                "rating_value": [1500.0, 1700.0],  # delta brut = +200 → capé à +100
                "rating_deviation": [50.0, 50.0],
                "playlist_group": ["social", "social"],
            }
        )
        SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        rows = conn.executemany.call_args[0][1]
        # m2 rating doit être 1500 + 100 = 1600, delta = +100
        assert rows[1][1] == pytest.approx(1600.0)
        assert rows[1][7] == pytest.approx(100.0)

    def test_delta_over_limit_capped_negative(self):
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m2"],
                "rating_value": [1500.0, 1200.0],  # delta brut = -300 → capé à -100
                "rating_deviation": [50.0, 50.0],
                "playlist_group": ["social", "social"],
            }
        )
        SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        rows = conn.executemany.call_args[0][1]
        assert rows[1][1] == pytest.approx(1400.0)
        assert rows[1][7] == pytest.approx(-100.0)

    def test_guardrail_independent_per_playlist_group(self):
        """Le guard-rail est indépendant par playlist_group."""
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": ["m1", "m2", "m3"],
                "rating_value": [1500.0, 1500.0, 1700.0],  # m3 dans social: delta=+200 → +100
                "rating_deviation": [50.0, 50.0, 50.0],
                "playlist_group": ["ranked", "social", "social"],
            }
        )
        SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2", "m3"),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        rows = conn.executemany.call_args[0][1]
        # m3 social doit être capé à 1600 (m2=1500 + 100)
        assert rows[2][1] == pytest.approx(1600.0)


# ---------------------------------------------------------------------------
# Tests filtres
# ---------------------------------------------------------------------------


class TestLUSRFilters:
    """existing_csr_ids et existing_lusr_ids filtrent correctement les lignes."""

    def test_existing_csr_ids_always_filtered(self):
        conn = _mock_conn()
        df = _ratings_df(
            {"match_id": "m1"},
            {"match_id": "m2"},
            {"match_id": "m3"},
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2", "m3"),
            existing_csr_ids={"m1", "m2"},
            existing_lusr_ids=set(),
            force=False,
        )
        assert result == 1
        rows = conn.executemany.call_args[0][1]
        inserted_ids = [r[0] for r in rows]
        assert "m3" in inserted_ids
        assert "m1" not in inserted_ids
        assert "m2" not in inserted_ids

    def test_existing_lusr_ids_filtered_without_force(self):
        conn = _mock_conn()
        df = _ratings_df(
            {"match_id": "m1"},
            {"match_id": "m2"},
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids=set(),
            existing_lusr_ids={"m1"},
            force=False,
        )
        assert result == 1
        rows = conn.executemany.call_args[0][1]
        assert rows[0][0] == "m2"

    def test_existing_lusr_ids_not_filtered_with_force(self):
        conn = _mock_conn()
        df = _ratings_df(
            {"match_id": "m1"},
            {"match_id": "m2"},
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            _matches_df("m1", "m2"),
            existing_csr_ids=set(),
            existing_lusr_ids={"m1"},
            force=True,
        )
        assert result == 2

    def test_empty_dataframe_returns_zero(self):
        conn = _mock_conn()
        df = pl.DataFrame(
            {
                "match_id": pl.Series([], dtype=pl.Utf8),
                "rating_value": pl.Series([], dtype=pl.Float64),
                "rating_deviation": pl.Series([], dtype=pl.Float64),
                "playlist_group": pl.Series([], dtype=pl.Utf8),
            }
        )
        result = SkillRatingMixin._upsert_lusr_ratings(
            None,
            conn,
            df,
            pl.DataFrame({"match_id": [], "start_time": []}),
            existing_csr_ids=set(),
            existing_lusr_ids=set(),
            force=False,
        )
        assert result == 0
        conn.executemany.assert_not_called()
