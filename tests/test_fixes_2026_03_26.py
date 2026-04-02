"""Tests des fixes du 2026-03-26 :

- Fix 1 : KDA COALESCE dans mv_player_matches (colonne présente / NULL / absente)
- Fix 2 : Encounters filtrées par match_start_time (filter_past SQL + _fetch_match_start_time)
- Fix 3 : Media EXIF naïf ignoré (branche EXIF sans timezone)
- Fix 4 : UX squad score — bonus collectif affiché sur la carte équipe
"""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import patch

import duckdb

# =============================================================================
# Fix 1 — KDA COALESCE dans mv_player_matches
# =============================================================================

_SHARED_SCHEMA_WITH_KDA = """
CREATE TABLE match_registry (
    match_id VARCHAR PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    playlist_id VARCHAR,
    playlist_name VARCHAR,
    map_id VARCHAR,
    map_name VARCHAR,
    pair_id VARCHAR,
    pair_name VARCHAR,
    game_variant_id VARCHAR,
    game_variant_name VARCHAR,
    duration_seconds INTEGER,
    team_0_score INTEGER,
    team_1_score INTEGER,
    team_0_ps_score INTEGER,
    team_1_ps_score INTEGER,
    is_ranked BOOLEAN DEFAULT FALSE,
    is_firefight BOOLEAN DEFAULT FALSE
);
CREATE TABLE match_participants (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    gamertag VARCHAR,
    team_id INTEGER,
    outcome INTEGER,
    kills INTEGER DEFAULT 0,
    deaths INTEGER DEFAULT 0,
    assists INTEGER DEFAULT 0,
    kda FLOAT,
    rank INTEGER,
    score INTEGER,
    shots_fired INTEGER,
    shots_hit INTEGER,
    avg_life_seconds FLOAT,
    headshot_kills INTEGER,
    max_killing_spree INTEGER,
    team_mmr FLOAT,
    PRIMARY KEY (match_id, xuid)
);
CREATE TABLE medals_earned (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    medal_name_id BIGINT NOT NULL,
    count SMALLINT DEFAULT 1,
    PRIMARY KEY (match_id, xuid, medal_name_id)
);
CREATE TABLE highlight_events (
    id INTEGER PRIMARY KEY,
    match_id VARCHAR,
    event_type VARCHAR,
    time_ms INTEGER,
    killer_xuid VARCHAR,
    killer_gamertag VARCHAR,
    victim_xuid VARCHAR,
    victim_gamertag VARCHAR
);
CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR NOT NULL);
CREATE TABLE killer_victim_pairs (
    match_id VARCHAR NOT NULL,
    killer_xuid VARCHAR NOT NULL,
    victim_xuid VARCHAR NOT NULL,
    kill_count INTEGER DEFAULT 1,
    PRIMARY KEY (match_id, killer_xuid, victim_xuid)
);
"""

_XUID = "xuid-kda-test"
_MATCH = "match-kda-001"


def _make_shared_db(
    tmp_path: Path, *, with_kda_col: bool = True, kda_value: float | None = -1.5
) -> Path:
    """Crée un shared_matches DB de test avec ou sans colonne kda."""
    db_path = tmp_path / "warehouse" / "shared_matches_v2.duckdb"
    db_path.parent.mkdir(parents=True)
    with duckdb.connect(str(db_path)) as conn:
        schema = (
            _SHARED_SCHEMA_WITH_KDA
            if with_kda_col
            else _SHARED_SCHEMA_WITH_KDA.replace("    kda FLOAT,\n", "")
        )
        for stmt in schema.strip().split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        conn.execute(
            "INSERT INTO match_registry (match_id, start_time, playlist_name, map_name, duration_seconds, "
            "team_0_score, team_1_score, is_ranked) VALUES (?, '2025-01-15 20:00:00', 'Arena', 'Aquarius', 600, 50, 48, FALSE)",
            [_MATCH],
        )
        if with_kda_col:
            conn.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, "
                "kills, deaths, assists, kda, shots_fired, shots_hit) VALUES (?, ?, 'P', 0, 2, 10, 5, 3, ?, 100, 50)",
                [_MATCH, _XUID, kda_value],
            )
        else:
            conn.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, "
                "kills, deaths, assists, shots_fired, shots_hit) VALUES (?, ?, 'P', 0, 2, 10, 5, 3, 100, 50)",
                [_MATCH, _XUID],
            )
    return db_path


class TestKdaCoalesce:
    """Fix 1 — COALESCE(p.kda, fallback) dans mv_player_matches."""

    def test_kda_uses_api_value_when_present(self, tmp_path: Path) -> None:
        """Si p.kda est peuplé, la vue utilise la valeur API (pas le recalcul)."""
        from src.data.sync.migrations import ensure_mv_player_matches_view

        db = _make_shared_db(tmp_path, with_kda_col=True, kda_value=-1.5)
        with duckdb.connect(str(db)) as conn:
            ensure_mv_player_matches_view(conn)
            row = conn.execute(
                "SELECT kda FROM mv_player_matches WHERE xuid = ?", [_XUID]
            ).fetchone()
        assert row is not None
        # kills=10, deaths=5, assists=3 → formule locale = (10 + 3/3)/5 = 2.2
        # Mais kda API = -1.5 → COALESCE doit retourner -1.5
        assert abs(row[0] - (-1.5)) < 0.01, f"Attendu -1.5 (API), obtenu {row[0]}"

    def test_kda_null_when_api_value_is_null(self, tmp_path: Path) -> None:
        """Si p.kda est NULL (ancien match sans valeur API), la vue retourne NULL — pas de recalcul."""
        from src.data.sync.migrations import ensure_mv_player_matches_view

        db = _make_shared_db(tmp_path, with_kda_col=True, kda_value=None)
        with duckdb.connect(str(db)) as conn:
            ensure_mv_player_matches_view(conn)
            row = conn.execute(
                "SELECT kda FROM mv_player_matches WHERE xuid = ?", [_XUID]
            ).fetchone()
        assert row is not None
        assert row[0] is None, f"Attendu NULL (pas de recalcul local), obtenu {row[0]}"

    def test_kda_null_when_column_absent(self, tmp_path: Path) -> None:
        """Sans colonne kda dans match_participants (schéma minimal), la vue retourne NULL."""
        from src.data.sync.migrations import ensure_mv_player_matches_view

        db = _make_shared_db(tmp_path, with_kda_col=False)
        with duckdb.connect(str(db)) as conn:
            ensure_mv_player_matches_view(conn)
            row = conn.execute(
                "SELECT kda FROM mv_player_matches WHERE xuid = ?", [_XUID]
            ).fetchone()
        assert row is not None
        assert row[0] is None, f"Attendu NULL (colonne absente), obtenu {row[0]}"


# =============================================================================
# Fix 2 — Encounters filtrées par match_start_time
# =============================================================================

_ENC_SCHEMA = """
CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL);
CREATE TABLE match_participants (
    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, gamertag VARCHAR,
    team_id INTEGER, outcome INTEGER, PRIMARY KEY (match_id, xuid)
);
CREATE TABLE killer_victim_pairs (
    match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
    victim_xuid VARCHAR NOT NULL, kill_count INTEGER DEFAULT 1,
    PRIMARY KEY (match_id, killer_xuid, victim_xuid)
);
CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR NOT NULL);
"""

_ME2 = "xuid-me-filter"
_OPP = "xuid-opp-filter"


def _make_encounter_db(tmp_path: Path) -> tuple[Path, str]:
    """Crée shared DB + chemin player DB pour les tests de filtre."""
    shared = tmp_path / "warehouse" / "shared_matches_v2.duckdb"
    shared.parent.mkdir(parents=True)
    player = str(tmp_path / "players" / "tester" / "stats.duckdb")
    with duckdb.connect(str(shared)) as conn:
        for stmt in _ENC_SCHEMA.strip().split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)
        # 3 matchs dans l'ordre chronologique
        conn.execute("INSERT INTO match_registry VALUES ('old-match', '2025-01-10 10:00:00')")
        conn.execute("INSERT INTO match_registry VALUES ('cur-match', '2025-01-15 20:00:00')")
        conn.execute("INSERT INTO match_registry VALUES ('fut-match', '2025-01-20 10:00:00')")
        for mid in ("old-match", "cur-match", "fut-match"):
            conn.execute("INSERT INTO match_participants VALUES (?, ?, 'Me', 0, 2)", [mid, _ME2])
            conn.execute("INSERT INTO match_participants VALUES (?, ?, 'Opp', 1, 3)", [mid, _OPP])
    return shared, player


class TestEncounterFilterPast:
    """Fix 2 — filter_past dans _build_encounter_sql et load_encounter_stats."""

    def test_my_matches_cte_no_filter_has_no_r2_join(self) -> None:
        """Sans filter_past, my_matches ne fait pas de JOIN match_registry."""
        from src.data.repositories._encounter_loader import _my_matches_cte

        cte = _my_matches_cte(False)
        assert "match_registry" not in cte
        assert "start_time" not in cte

    def test_my_matches_cte_with_filter_has_start_time(self) -> None:
        """Avec filter_past, my_matches filtre par r2.start_time < ?."""
        from src.data.repositories._encounter_loader import _my_matches_cte

        cte = _my_matches_cte(True)
        assert "r2.start_time < ?" in cte
        assert "mp.match_id != ?" in cte
        assert "match_registry r2" in cte

    def test_filter_past_excludes_current_and_future_matches(self, tmp_path: Path) -> None:
        """Avec match_start_time du match courant, seuls les matchs antérieurs comptent."""
        from src.data.repositories._encounter_loader import load_encounter_stats

        shared, player = _make_encounter_db(tmp_path)
        match_start = datetime(2025, 1, 15, 20, 0, 0)

        result = load_encounter_stats(
            _ME2,
            [_OPP],
            player,
            match_start_time=match_start,
            current_match_id="cur-match",
        )
        assert not result.is_empty()
        row = result.row(0, named=True)
        # old-match (2025-01-10) < cur-match (2025-01-15) → 1 rencontre comptée
        # cur-match et fut-match exclus
        assert row["total_encounters"] == 1

    def test_filter_past_last_seen_is_previous_match(self, tmp_path: Path) -> None:
        """last_seen doit pointer vers le match précédent, pas le match courant."""
        from src.data.repositories._encounter_loader import load_encounter_stats

        shared, player = _make_encounter_db(tmp_path)
        match_start = datetime(2025, 1, 15, 20, 0, 0)

        result = load_encounter_stats(
            _ME2,
            [_OPP],
            player,
            match_start_time=match_start,
            current_match_id="cur-match",
        )
        row = result.row(0, named=True)
        last_seen = row["last_seen"]
        # last_seen = 2025-01-10 (old-match)
        assert last_seen is not None
        if isinstance(last_seen, datetime):
            assert last_seen.year == 2025 and last_seen.month == 1 and last_seen.day == 10
        else:
            assert "2025-01-10" in str(last_seen)

    def test_without_filter_counts_all_matches(self, tmp_path: Path) -> None:
        """Sans filtre, tous les matchs (y compris courant et futur) sont comptés."""
        from src.data.repositories._encounter_loader import load_encounter_stats

        shared, player = _make_encounter_db(tmp_path)

        result = load_encounter_stats(_ME2, [_OPP], player)
        row = result.row(0, named=True)
        assert row["total_encounters"] == 3  # old + cur + fut

    def test_fetch_match_start_time_returns_none_for_missing_match(self, tmp_path: Path) -> None:
        """_fetch_match_start_time retourne None si le match_id est inconnu."""
        from src.data.repositories._encounter_loader import _fetch_match_start_time

        shared, player = _make_encounter_db(tmp_path)
        result = _fetch_match_start_time("nonexistent-match", player)
        assert result is None

    def test_fetch_match_start_time_returns_datetime_for_known_match(self, tmp_path: Path) -> None:
        """_fetch_match_start_time retourne un datetime pour un match connu."""
        from src.data.repositories._encounter_loader import _fetch_match_start_time

        shared, player = _make_encounter_db(tmp_path)
        result = _fetch_match_start_time("cur-match", player)
        assert isinstance(result, datetime)
        assert result.year == 2025 and result.month == 1 and result.day == 15


# =============================================================================
# Fix 3 — Media EXIF naïf ignoré
# =============================================================================


class TestMediaExifNaif:
    """Fix 3 — branche EXIF sans timezone ignorée dans get_file_metadata."""

    def test_exif_naive_falls_back_to_mtime(self, tmp_path: Path) -> None:
        """Un EXIF sans tzinfo ne doit pas écraser capture_end_utc (utiliser mtime)."""
        from src.data.media_helpers import get_file_metadata

        img = tmp_path / "test.jpg"
        img.write_bytes(b"\xff\xd8\xff")  # JPEG minimal

        naive_exif = datetime(2020, 6, 15, 14, 0, 0)  # heure locale, sans tz
        with (
            patch("src.data.media_helpers.get_image_exif_datetime", return_value=naive_exif),
            patch("src.data.media_helpers.get_video_duration", return_value=None),
        ):
            meta = get_file_metadata(img)

        # capture_end_utc doit être basé sur mtime, PAS sur l'EXIF naïf
        capture = meta.get("capture_end_utc")
        assert capture is not None
        if isinstance(capture, datetime):
            # L'EXIF naïf était 2020-06-15 → ne doit PAS apparaître
            assert capture.year != 2020 or capture.month != 6 or capture.day != 15

    def test_exif_aware_is_used(self, tmp_path: Path) -> None:
        """Un EXIF avec tzinfo UTC est correctement utilisé."""
        from src.data.media_helpers import get_file_metadata

        img = tmp_path / "test2.jpg"
        img.write_bytes(b"\xff\xd8\xff")

        aware_exif = datetime(2024, 3, 10, 12, 0, 0, tzinfo=timezone.utc)
        with (
            patch("src.data.media_helpers.get_image_exif_datetime", return_value=aware_exif),
            patch("src.data.media_helpers.get_video_duration", return_value=None),
        ):
            meta = get_file_metadata(img)

        capture = meta.get("capture_end_utc")
        assert capture is not None
        assert isinstance(capture, datetime)
        assert capture.year == 2024 and capture.month == 3 and capture.day == 10


# =============================================================================
# Fix 4 — UX bonus collectif sur la carte équipe
# =============================================================================


class TestSquadScoreBonus:
    """Fix 4 — _render_compact_team_card affiche le bonus collectif."""

    def _render(self, team_perf: dict, grade: str = "SOLIDE", lang: str = "fr") -> str:
        """Capture le HTML généré par _render_compact_team_card."""
        from src.ui.components import performance as perf_mod

        captured: list[str] = []
        with patch.object(
            perf_mod.st, "markdown", side_effect=lambda html, **_: captured.append(html)
        ):
            perf_mod._render_compact_team_card(team_perf, grade, lang)
        return captured[0] if captured else ""

    def test_bonus_displayed_when_positive(self) -> None:
        """Le bonus collectif est affiché quand score > base_avg."""
        team_perf = {"score": 75.0, "components": {"base_avg": 60.0}}
        html = self._render(team_perf)
        assert "os-perf-card__detail" in html
        assert "+15" in html  # bonus = 75 - 60 = 15

    def test_base_displayed_when_bonus_zero(self) -> None:
        """La moyenne de base est affichée même quand bonus = 0."""
        team_perf = {"score": 60.0, "components": {"base_avg": 60.0}}
        html = self._render(team_perf)
        assert "os-perf-card__detail" in html
        assert "moy. 60" in html
        assert "+" not in html  # pas de bonus affiché

    def test_bonus_not_displayed_when_score_is_none(self) -> None:
        """Pas de bonus si le score est None."""
        team_perf = {"score": None, "components": {"base_avg": 60.0}}
        html = self._render(team_perf)
        assert "os-perf-card__detail" not in html

    def test_score_display_uses_api_value(self) -> None:
        """Le score affiché est bien la valeur totale (base + bonus)."""
        team_perf = {"score": 78.0, "components": {"base_avg": 65.0}}
        html = self._render(team_perf)
        assert "78" in html
