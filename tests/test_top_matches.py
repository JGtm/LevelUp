"""Tests pour le Top 10 meilleurs / pires matchs (chantier H)."""

from __future__ import annotations

import html as html_mod
import os
import shutil
import tempfile
from datetime import datetime, timezone
from unittest.mock import patch

import duckdb
import pytest

from src.analysis._medal_verdicts import DominanceFlag
from src.data.domain.refdata import Outcome
from src.ui.pages.career_top_matches_data import (
    _BADGE_PRIORITY_EXPR,
    _BTB_FILTER_SQL,
    _TOP_MATCHES_SQL,
    MIN_MATCH_DURATION_SECONDS,
)
from src.ui.pages.career_top_matches_render import (
    _badge_html,
    _build_top_table_html,
    _format_date,
    _format_duration,
    _format_kda,
    _format_score,
    _kd_color,
    _kd_ratio,
)

# ── Helpers ──────────────────────────────────────────────────────────────────


def _make_row(**overrides: object) -> dict:
    """Construit un dict-row avec des valeurs par défaut."""
    base: dict = {
        "match_id": "m1",
        "start_time": datetime(2025, 1, 15, 18, 30, 0, tzinfo=timezone.utc),
        "map_name": "Recharge",
        "playlist_name": "Ranked Arena",
        "game_variant_name": "Slayer",
        "outcome": 2,
        "kills": 20,
        "deaths": 10,
        "assists": 5,
        "kda": 2.0,
        "time_played_seconds": 600,
        "my_team_score": 50,
        "enemy_team_score": 30,
        "dominance_flag": 0,
        "had_bot_teammate": False,
    }
    base.update(overrides)
    return base


# ── Tests format_duration ────────────────────────────────────────────────────


class TestFormatDuration:
    def test_normal(self) -> None:
        assert _format_duration(600) == "10:00"

    def test_with_seconds(self) -> None:
        assert _format_duration(125) == "2:05"

    def test_none(self) -> None:
        assert _format_duration(None) == "—"

    def test_zero(self) -> None:
        assert _format_duration(0) == "—"


# ── Tests format_score ───────────────────────────────────────────────────────


class TestFormatScore:
    def test_normal(self) -> None:
        assert _format_score(50, 30) == "50 — 30"

    def test_none_values(self) -> None:
        assert _format_score(None, None) == "0 — 0"


# ── Tests format_kda ──────────────────────────────────────────────────────────


class TestFormatKDA:
    def test_normal(self) -> None:
        row = _make_row(kills=20, deaths=10, assists=5)
        assert _format_kda(row) == "20/10/5"

    def test_zeros(self) -> None:
        row = _make_row(kills=0, deaths=0, assists=0)
        assert _format_kda(row) == "0/0/0"


# ── Tests kd_ratio ────────────────────────────────────────────────────────────


class TestKDRatio:
    def test_positive_ratio(self) -> None:
        row = _make_row(kills=20, deaths=10)
        assert _kd_ratio(row) == "2.00"

    def test_zero_deaths(self) -> None:
        row = _make_row(kills=5, deaths=0)
        assert _kd_ratio(row) == "5.0"

    def test_zero_zero(self) -> None:
        row = _make_row(kills=0, deaths=0)
        assert _kd_ratio(row) == "—"


# ── Tests kd_color ────────────────────────────────────────────────────────────


class TestKDColor:
    def test_high_ratio(self) -> None:
        row = _make_row(kills=30, deaths=10)
        assert _kd_color(row) == "#3DFFB5"

    def test_low_ratio(self) -> None:
        row = _make_row(kills=5, deaths=20)
        assert _kd_color(row) == "#FF4D6D"

    def test_neutral_ratio(self) -> None:
        row = _make_row(kills=10, deaths=10)
        assert _kd_color(row) == ""


# ── Tests badge_html ──────────────────────────────────────────────────────────


class TestBadgeHTML:
    @patch("src.ui.pages.career_top_matches_render.t", return_value="Domination")
    def test_domination_badge_on_best(self, _mock_t: object) -> None:
        result = _badge_html(1, best=True)
        assert "Domination" in result
        assert "#2e7d32" in result

    @patch("src.ui.pages.career_top_matches_render.t", return_value="Humiliation")
    def test_humiliation_badge_on_worst(self, _mock_t: object) -> None:
        result = _badge_html(2, best=False)
        assert "Humiliation" in result
        assert "#6a1b9a" in result

    def test_no_badge_when_flag_zero(self) -> None:
        assert _badge_html(0, best=True) == ""

    def test_no_dom_badge_on_worst(self) -> None:
        assert _badge_html(1, best=False) == ""

    def test_no_hum_badge_on_best(self) -> None:
        assert _badge_html(2, best=True) == ""


# ── Tests build_top_table_html ────────────────────────────────────────────────


class TestBuildTopTableHTML:
    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_empty_rows(self, _mock_t: object) -> None:
        result = _build_top_table_html([], best=True)
        assert "<table" in result
        assert "<tbody></tbody>" in result

    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_single_row(self, _mock_t: object) -> None:
        rows = [_make_row()]
        result = _build_top_table_html(rows, best=True)
        assert "Recharge" in result
        assert "Slayer" in result
        assert "50 — 30" in result
        assert "20/10/5" in result

    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_all_columns_present(self, _mock_t: object) -> None:
        rows = [_make_row()]
        result = _build_top_table_html(rows, best=True)
        for key in [
            "career_top_col_date",
            "career_top_col_mode",
            "career_top_col_map",
            "career_top_col_score",
            "career_top_col_kda",
            "career_top_col_kd",
            "career_top_col_duration",
        ]:
            assert key in result

    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_html_escaping(self, _mock_t: object) -> None:
        """Les noms de carte avec caractères spéciaux sont échappés."""
        rows = [_make_row(map_name="<script>alert(1)</script>")]
        result = _build_top_table_html(rows, best=True)
        assert "<script>" not in result
        assert html_mod.escape("<script>alert(1)</script>") in result


# ── Tests format_date ──────────────────────────────────────────────────────────


class TestFormatDate:
    def test_datetime_object(self) -> None:
        dt = datetime(2025, 3, 15, 14, 30, 0, tzinfo=timezone.utc)
        assert _format_date(dt) == "15/03/2025 14:30"

    def test_iso_string(self) -> None:
        result = _format_date("2025-06-20T10:15:00")
        assert result == "20/06/2025 10:15"

    def test_none(self) -> None:
        assert _format_date(None) == "—"

    def test_invalid_string(self) -> None:
        assert _format_date("not-a-date") == "—"


# ── Tests fallback mode/map ──────────────────────────────────────────────────


class TestGameVariantFallback:
    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_fallback_to_playlist_name(self, _mock_t: object) -> None:
        """Si game_variant_name est None, on affiche playlist_name."""
        rows = [_make_row(game_variant_name=None, playlist_name="BTB")]
        result = _build_top_table_html(rows, best=True)
        assert "BTB" in result

    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_all_mode_fields_none(self, _mock_t: object) -> None:
        """Si les deux champs mode sont None, on affiche '—'."""
        rows = [_make_row(game_variant_name=None, playlist_name=None)]
        result = _build_top_table_html(rows, best=True)
        # Le tiret doit apparaître dans le HTML
        assert "—" in result

    @patch("src.ui.pages.career_top_matches_render.t", side_effect=lambda k: k)
    def test_none_kills_deaths(self, _mock_t: object) -> None:
        """Les None dans kills/deaths ne cassent pas le rendu."""
        rows = [_make_row(kills=None, deaths=None, assists=None)]
        result = _build_top_table_html(rows, best=True)
        assert "0/0/0" in result


# ── Tests intégration DuckDB ─────────────────────────────────────────────────


class TestLoadTopMatchesDuckDB:
    """Tests d'intégration avec DuckDB en mémoire."""

    @pytest.fixture()
    def shared_conn(self, tmp_path: object) -> duckdb.DuckDBPyConnection:
        """Crée une DB shared sur disque avec la vue mv_player_matches, attachée comme 'shared'."""
        import pathlib

        shared_path = str(pathlib.Path(str(tmp_path)) / "shared_matches_v2.duckdb")
        shared_db = duckdb.connect(shared_path)
        shared_db.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                map_name VARCHAR,
                playlist_name VARCHAR,
                game_variant_name VARCHAR,
                is_firefight BOOLEAN DEFAULT FALSE,
                is_ranked BOOLEAN DEFAULT FALSE,
                medals_loaded BOOLEAN DEFAULT TRUE,
                team_0_ps_score INTEGER,
                team_1_ps_score INTEGER,
                mode_category VARCHAR DEFAULT NULL
            )
        """)
        shared_db.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                team_id INTEGER,
                outcome INTEGER,
                kills INTEGER,
                deaths INTEGER,
                assists INTEGER,
                kda DOUBLE,
                score INTEGER,
                time_played_seconds INTEGER,
                my_team_score INTEGER,
                enemy_team_score INTEGER,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        shared_db.execute("""
            CREATE VIEW mv_player_matches AS
            SELECT
                mp.match_id,
                mr.start_time,
                mr.map_name,
                mr.playlist_name,
                mr.game_variant_name,
                mp.outcome,
                mp.kills,
                mp.deaths,
                mp.assists,
                mp.kda,
                mp.time_played_seconds,
                mp.my_team_score,
                mp.enemy_team_score,
                CASE WHEN mp.team_id = 0 THEN mr.team_0_ps_score ELSE mr.team_1_ps_score END AS my_team_ps_score,
                CASE WHEN mp.team_id = 0 THEN mr.team_1_ps_score ELSE mr.team_0_ps_score END AS enemy_team_ps_score,
                mr.is_firefight,
                mr.is_ranked,
                mp.xuid,
                mp.team_id
            FROM match_participants mp
            JOIN match_registry mr ON mr.match_id = mp.match_id
        """)
        shared_db.close()
        # Ouvrir une connexion en mémoire et attacher la shared DB comme 'shared'
        conn = duckdb.connect(":memory:")
        conn.execute(f"ATTACH '{shared_path}' AS shared")
        return conn

    @pytest.fixture()
    def tmp_dir(self) -> str:  # noqa: PT004
        """Crée un répertoire temporaire, nettoyé après le test."""
        d = tempfile.mkdtemp()
        yield d  # type: ignore[misc]
        shutil.rmtree(d, ignore_errors=True)

    def _insert_match(  # noqa: PLR0913
        self,
        conn: duckdb.DuckDBPyConnection,
        match_id: str,
        xuid: str,
        *,
        outcome: int = 2,
        kills: int = 15,
        deaths: int = 5,
        assists: int = 3,
        time_played: int = 600,
        my_score: int | None = 50,
        enemy_score: int | None = 30,
        is_firefight: bool = False,
        map_name: str = "Recharge",
        mode_category: str | None = None,
    ) -> None:
        conn.execute(
            "INSERT INTO shared.match_registry VALUES (?,CURRENT_TIMESTAMP,?,?,?,?,?,?,NULL,NULL,?)",
            [match_id, map_name, "Arena", "Slayer", is_firefight, False, True, mode_category],
        )
        kda = (kills + assists / 3) / max(deaths, 1)
        conn.execute(
            "INSERT INTO shared.match_participants VALUES (?,?,0,?,?,?,?,?,0,?,?,?)",
            [
                match_id,
                xuid,
                outcome,
                kills,
                deaths,
                assists,
                kda,
                time_played,
                my_score,
                enemy_score,
            ],
        )

    def _create_player_db(
        self,
        tmp_dir: str,
        match_ids: list[str],
        *,
        bot_ids: set[str] | None = None,
        badge_flags: dict[str, int] | None = None,
    ) -> str:
        """Crée une player DB sur disque avec les enrichissements."""
        player_path = os.path.join(tmp_dir, "stats.duckdb")
        player_db = duckdb.connect(player_path)
        player_db.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                dominance_flag TINYINT DEFAULT 0,
                had_bot_teammate BOOLEAN DEFAULT FALSE
            )
        """)
        bot_ids = bot_ids or set()
        for mid in match_ids:
            has_bot = mid in bot_ids
            flag = (badge_flags or {}).get(mid, 0)
            player_db.execute(
                "INSERT INTO player_match_enrichment VALUES (?, ?, ?)",
                [mid, flag, has_bot],
            )
        player_db.close()
        return player_path

    def _query_top(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_path: str,
        xuid: str,
        *,
        best: bool = True,
        exclude_btb: bool = False,
    ) -> list[dict]:
        """Exécute la requête top matches et retourne les résultats."""
        shared_conn.execute(f"ATTACH '{player_path}' AS player (READ_ONLY)")
        # Créer une vue proxy dans le schéma memory pour player_match_enrichment
        shared_conn.execute(
            "CREATE OR REPLACE VIEW player_match_enrichment AS "
            "SELECT * FROM player.player_match_enrichment"
        )
        target = int(Outcome.WIN) if best else int(Outcome.LOSS)
        sql = _TOP_MATCHES_SQL.format(
            badge_priority=_BADGE_PRIORITY_EXPR[best],
            btb_filter=_BTB_FILTER_SQL[exclude_btb],
        )
        result = shared_conn.execute(
            sql,
            [
                xuid,
                int(Outcome.WIN),
                int(Outcome.LOSS),
                MIN_MATCH_DURATION_SECONDS,
                target,
            ],
        )
        columns = [desc[0] for desc in result.description]
        return [dict(zip(columns, r, strict=True)) for r in result.fetchall()]

    def test_load_best_returns_wins(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """load_top_best ne retourne que les victoires."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "w1", xuid, outcome=2, kills=20)
        self._insert_match(shared_conn, "w2", xuid, outcome=2, kills=15)
        self._insert_match(shared_conn, "l1", xuid, outcome=3, kills=5, deaths=15)
        player_path = self._create_player_db(tmp_dir, ["w1", "w2", "l1"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 2
        assert all(r["outcome"] == 2 for r in rows)

    def test_excludes_short_matches(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Les matchs < 3 min sont exclus."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "short", xuid, outcome=2, time_played=60)
        self._insert_match(shared_conn, "long", xuid, outcome=2, time_played=600)
        player_path = self._create_player_db(tmp_dir, ["short", "long"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "long"

    def test_excludes_firefight(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Les matchs Firefight sont exclus."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "pve", xuid, outcome=2, is_firefight=True)
        self._insert_match(shared_conn, "pvp", xuid, outcome=2, is_firefight=False)
        player_path = self._create_player_db(tmp_dir, ["pve", "pvp"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "pvp"

    def test_excludes_bots(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Les matchs avec bots sont exclus."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "bot_match", xuid, outcome=2)
        self._insert_match(shared_conn, "real_match", xuid, outcome=2)
        player_path = self._create_player_db(
            tmp_dir,
            ["bot_match", "real_match"],
            bot_ids={"bot_match"},
        )

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "real_match"

    def test_load_worst_returns_losses(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """load_top_worst ne retourne que les défaites."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "w1", xuid, outcome=2, kills=20)
        self._insert_match(shared_conn, "l1", xuid, outcome=3, kills=3, deaths=20)
        self._insert_match(shared_conn, "l2", xuid, outcome=3, kills=5, deaths=15)
        player_path = self._create_player_db(tmp_dir, ["w1", "l1", "l2"])

        rows = self._query_top(shared_conn, player_path, xuid, best=False)
        assert len(rows) == 2
        assert all(r["outcome"] == 3 for r in rows)

    def test_excludes_tie_and_dnf(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Les matchs nuls (TIE=1) et DNF (4) sont exclus."""
        xuid = "xuid_test"
        self._insert_match(shared_conn, "tie", xuid, outcome=1)
        self._insert_match(shared_conn, "dnf", xuid, outcome=4)
        self._insert_match(shared_conn, "win", xuid, outcome=2)
        player_path = self._create_player_db(tmp_dir, ["tie", "dnf", "win"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "win"

    # ── Problème E : exclude_btb ──────────────────────────────────────────

    def test_btb_excluded_when_flag_true(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Avec exclude_btb=True, les matchs BTB sont absents des résultats."""
        xuid = "xuid_btb"
        self._insert_match(shared_conn, "btb_m", xuid, outcome=2, mode_category="BTB")
        self._insert_match(shared_conn, "arena_m", xuid, outcome=2)
        player_path = self._create_player_db(tmp_dir, ["btb_m", "arena_m"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True, exclude_btb=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "arena_m"

    def test_btb_included_when_flag_false(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Avec exclude_btb=False (défaut), les matchs BTB apparaissent."""
        xuid = "xuid_btb2"
        self._insert_match(shared_conn, "btb_m2", xuid, outcome=2, mode_category="BTB")
        self._insert_match(shared_conn, "arena_m2", xuid, outcome=2)
        player_path = self._create_player_db(tmp_dir, ["btb_m2", "arena_m2"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True, exclude_btb=False)
        assert len(rows) == 2

    def test_btb_worst_excluded_when_flag_true(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """exclude_btb=True fonctionne aussi pour les pires matchs."""
        xuid = "xuid_btb3"
        self._insert_match(shared_conn, "btb_loss", xuid, outcome=3, mode_category="BTB")
        self._insert_match(shared_conn, "arena_loss", xuid, outcome=3)
        player_path = self._create_player_db(tmp_dir, ["btb_loss", "arena_loss"])

        rows = self._query_top(shared_conn, player_path, xuid, best=False, exclude_btb=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "arena_loss"

    # ── Problème F : exclusion matchs sans team_score ─────────────────────

    def test_null_team_score_excluded(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Les matchs FFA (team_score NULL) sont exclus des top matches."""
        xuid = "xuid_ffa"
        self._insert_match(shared_conn, "ffa_m", xuid, outcome=2, my_score=None, enemy_score=None)
        self._insert_match(shared_conn, "team_m", xuid, outcome=2, my_score=50, enemy_score=30)
        player_path = self._create_player_db(tmp_dir, ["ffa_m", "team_m"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "team_m"

    def test_partial_null_team_score_excluded(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Un match avec un seul score NULL est aussi exclu."""
        xuid = "xuid_partial"
        self._insert_match(shared_conn, "partial_m", xuid, outcome=2, my_score=50, enemy_score=None)
        self._insert_match(shared_conn, "full_m", xuid, outcome=2, my_score=50, enemy_score=30)
        player_path = self._create_player_db(tmp_dir, ["partial_m", "full_m"])

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 1
        assert rows[0]["match_id"] == "full_m"

    # ── Problème G : tri par badge_priority ───────────────────────────────

    def test_badge_priority_best_order(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Pour best=True, l'ordre est : CONTRE_REMONTADA > REMONTADA > DOMINATION > sans badge."""
        xuid = "xuid_badge"
        for mid in ("b_none", "b_dom", "b_rem", "b_cr"):
            self._insert_match(shared_conn, mid, xuid, outcome=2)
        player_path = self._create_player_db(
            tmp_dir,
            ["b_none", "b_dom", "b_rem", "b_cr"],
            badge_flags={
                "b_dom": int(DominanceFlag.DOMINATION),
                "b_rem": int(DominanceFlag.REMONTADA),
                "b_cr": int(DominanceFlag.CONTRE_REMONTADA),
            },
        )

        rows = self._query_top(shared_conn, player_path, xuid, best=True)
        assert len(rows) == 4
        assert rows[0]["match_id"] == "b_cr"
        assert rows[1]["match_id"] == "b_rem"
        assert rows[2]["match_id"] == "b_dom"
        assert rows[3]["match_id"] == "b_none"

    def test_badge_priority_worst_order(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        tmp_dir: str,
    ) -> None:
        """Pour best=False, l'ordre est : DEBANDADE > HUMILIATION > sans badge."""
        xuid = "xuid_badge2"
        for mid in ("w_none", "w_hum", "w_deb"):
            self._insert_match(shared_conn, mid, xuid, outcome=3)
        player_path = self._create_player_db(
            tmp_dir,
            ["w_none", "w_hum", "w_deb"],
            badge_flags={
                "w_hum": int(DominanceFlag.HUMILIATION),
                "w_deb": int(DominanceFlag.DEBANDADE),
            },
        )

        rows = self._query_top(shared_conn, player_path, xuid, best=False)
        assert len(rows) == 3
        assert rows[0]["match_id"] == "w_deb"
        assert rows[1]["match_id"] == "w_hum"
        assert rows[2]["match_id"] == "w_none"
