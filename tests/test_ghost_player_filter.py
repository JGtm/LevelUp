"""Tests dédiés au filtrage des joueurs fantômes (ghost players).

Un joueur fantôme a kills=0, deaths=0, assists=0, score=0 (entiers 0 explicites).
Un joueur avec des NULL est partiellement chargé et doit être conservé.
Les bots (xuid commençant par 'bid(') sont aussi filtrés dans certaines requêtes.

Surfaces testées :
- _SQL_NOT_GHOST (constante SQL)
- load_match_scoreboard (scoreboard match)
- load_match_players_stats (stats joueurs match)
- list_top_teammates (coéquipiers fréquents)
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

# =============================================================================
# Helpers
# =============================================================================


def _create_shared_db(path: Path) -> None:
    """Crée un shared_matches.duckdb minimal pour les tests ghost."""
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
            team_id INTEGER, rank INTEGER,
            score INTEGER, kills INTEGER, deaths INTEGER, assists INTEGER,
            kda DOUBLE, max_killing_spree INTEGER, headshot_kills INTEGER,
            shots_fired INTEGER, shots_hit INTEGER, accuracy DOUBLE,
            melee_kills INTEGER, power_weapon_kills INTEGER,
            damage_dealt BIGINT, damage_taken BIGINT, avg_life_seconds DOUBLE,
            outcome INTEGER
        )
    """)
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
    conn.execute(
        "CREATE TABLE medals_earned "
        "(match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)"
    )
    conn.execute("""
        CREATE TABLE weapon_kills (
            match_id VARCHAR, xuid VARCHAR, time_ms INTEGER,
            weapon_id UBIGINT, delta_ms INTEGER, confidence VARCHAR,
            swap_detected BOOLEAN, delayed_damage BOOLEAN
        )
    """)
    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP,
            playlist_name VARCHAR, pair_name VARCHAR, map_name VARCHAR,
            game_variant_name VARCHAR, game_variant_id VARCHAR,
            duration_seconds INTEGER, is_firefight BOOLEAN DEFAULT FALSE,
            is_ranked BOOLEAN DEFAULT FALSE,
            team_0_score INTEGER, team_1_score INTEGER
        )
    """)
    conn.execute("""
        CREATE TABLE killer_victim_pairs (
            match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR,
            victim_xuid VARCHAR, victim_gamertag VARCHAR,
            kill_count INTEGER, time_ms INTEGER,
            is_validated BOOLEAN, created_at TIMESTAMP
        )
    """)
    # Vues v6
    conn.execute(
        "CREATE VIEW v_gamertag_lookup AS "
        "SELECT COALESCE(xa.xuid, mp.xuid) AS xuid, "
        "COALESCE(xa.gamertag, mp.gamertag) AS gamertag "
        "FROM xuid_aliases xa "
        "FULL OUTER JOIN ("
        "    SELECT xuid, MAX(gamertag) AS gamertag "
        "    FROM match_participants WHERE gamertag IS NOT NULL GROUP BY xuid"
        ") mp ON xa.xuid = mp.xuid "
        "WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL"
    )
    conn.execute(
        "CREATE VIEW v_weapon_kills AS "
        "SELECT *, COALESCE(weapon_id, weapon_id) AS effective_weapon_id "
        "FROM weapon_kills"
    )
    conn.execute(
        "CREATE VIEW v_killer_victim_full AS "
        "SELECT kv.*, "
        "COALESCE(vk.gamertag, kv.killer_gamertag) AS resolved_killer_gt, "
        "COALESCE(vv.gamertag, kv.victim_gamertag) AS resolved_victim_gt "
        "FROM killer_victim_pairs kv "
        "LEFT JOIN v_gamertag_lookup vk ON kv.killer_xuid = vk.xuid "
        "LEFT JOIN v_gamertag_lookup vv ON kv.victim_xuid = vv.xuid"
    )
    conn.close()


def _create_player_db(path: Path) -> None:
    """Crée un stats.duckdb vide."""
    duckdb.connect(str(path)).close()


def _insert_participant(  # noqa: PLR0913
    path: Path,
    match_id: str,
    xuid: str,
    *,
    gamertag: str | None = None,
    team_id: int = 0,
    rank: int = 1,
    score: int | None = 100,
    kills: int | None = 5,
    deaths: int | None = 3,
    assists: int | None = 2,
    outcome: int = 2,
) -> None:
    """Insère un participant avec valeurs explicites."""
    conn = duckdb.connect(str(path))
    conn.execute(
        """INSERT INTO match_participants (
            match_id, xuid, gamertag, team_id, rank,
            score, kills, deaths, assists,
            kda, max_killing_spree, headshot_kills,
            shots_fired, shots_hit, accuracy,
            melee_kills, power_weapon_kills,
            damage_dealt, damage_taken, avg_life_seconds, outcome
        ) VALUES (?,?,?,?,?, ?,?,?,?, 1.0,1,0, 10,5,0.5, 0,0, 500,500,30.0, ?)""",
        [match_id, xuid, gamertag, team_id, rank, score, kills, deaths, assists, outcome],
    )
    conn.close()


def _insert_ghost(path: Path, match_id: str, xuid: str, *, team_id: int = 1) -> None:
    """Insère un joueur fantôme (tous stats à 0 explicite)."""
    _insert_participant(
        path,
        match_id,
        xuid,
        gamertag=f"Ghost_{xuid}",
        team_id=team_id,
        score=0,
        kills=0,
        deaths=0,
        assists=0,
    )


def _insert_null_participant(path: Path, match_id: str, xuid: str) -> None:
    """Insère un joueur avec toutes les stats NULL (données partielles)."""
    _insert_participant(
        path,
        match_id,
        xuid,
        gamertag=f"Partial_{xuid}",
        score=None,
        kills=None,
        deaths=None,
        assists=None,
    )


def _insert_match_registry(path: Path, match_id: str) -> None:
    """Insère un match dans match_registry."""
    conn = duckdb.connect(str(path))
    conn.execute(
        "INSERT INTO match_registry VALUES (?, '2025-01-01 10:00:00', "
        "NULL, NULL, NULL, NULL, NULL, 600, FALSE, FALSE, 0, 0)",
        [match_id],
    )
    conn.close()


def _make_repo(tmp_path: Path, shared_path: Path, xuid: str = "ME"):
    """Construit un DuckDBRepository pour les tests."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    player_db = tmp_path / "stats.duckdb"
    _create_player_db(player_db)
    return DuckDBRepository(
        player_db_path=player_db,
        xuid=xuid,
        shared_db_path=shared_path,
        metadata_db_path=tmp_path / "meta.duckdb",
        read_only=False,
    )


# =============================================================================
# Tests _SQL_NOT_GHOST (constante SQL)
# =============================================================================


class TestSqlNotGhostConstant:
    """Vérifie le filtre SQL _SQL_NOT_GHOST sur des cas edge."""

    def _eval_filter(self, kills, deaths, assists, score) -> bool:
        """Évalue le filtre ghost sur une ligne, retourne True si le joueur est gardé."""
        from src.data.repositories._roster_loader import _SQL_NOT_GHOST

        sql = _SQL_NOT_GHOST.format(p="t")
        conn = duckdb.connect(":memory:")
        conn.execute("CREATE TABLE t (kills INT, deaths INT, assists INT, score INT)")
        conn.execute("INSERT INTO t VALUES (?,?,?,?)", [kills, deaths, assists, score])
        kept = conn.execute(f"SELECT COUNT(*) FROM t WHERE {sql}").fetchone()[0]  # noqa: S608
        conn.close()
        return kept == 1

    def test_normal_player_kept(self) -> None:
        """Joueur avec stats normales → gardé."""
        assert self._eval_filter(15, 5, 3, 2500) is True

    def test_ghost_all_zeros_excluded(self) -> None:
        """Fantôme (0,0,0,0 explicites) → exclu."""
        assert self._eval_filter(0, 0, 0, 0) is False

    def test_null_stats_kept(self) -> None:
        """Joueur avec stats NULL (données partielles) → gardé."""
        assert self._eval_filter(None, None, None, None) is True

    def test_partial_null_kept(self) -> None:
        """Joueur avec 1 stat non-nulle et le reste NULL → gardé."""
        assert self._eval_filter(5, None, None, None) is True

    def test_partial_zero_kept(self) -> None:
        """Joueur avec kills > 0 mais deaths/assists/score = 0 → gardé."""
        assert self._eval_filter(10, 0, 0, 0) is True

    def test_only_assists_kept(self) -> None:
        """Joueur avec seulement des assists → gardé."""
        assert self._eval_filter(0, 0, 5, 0) is True

    def test_only_score_kept(self) -> None:
        """Joueur avec seulement un score → gardé."""
        assert self._eval_filter(0, 0, 0, 100) is True

    def test_only_deaths_kept(self) -> None:
        """Joueur sans kills mais avec des deaths → gardé (pas un fantôme)."""
        assert self._eval_filter(0, 8, 0, 0) is True

    def test_mixed_null_and_zero_with_value(self) -> None:
        """kills=3, deaths=NULL, assists=0, score=0 → gardé."""
        assert self._eval_filter(3, None, 0, 0) is True

    def test_mixed_null_and_zero_no_value(self) -> None:
        """kills=NULL, deaths=0, assists=0, score=0 → exclu (données présentes, toutes 0)."""
        assert self._eval_filter(None, 0, 0, 0) is False


# =============================================================================
# Tests load_match_scoreboard avec ghost players
# =============================================================================


class TestScoreboardGhostFiltering:
    """Vérifie que load_match_scoreboard exclut les fantômes."""

    @pytest.fixture()
    def shared_path(self, tmp_path: Path) -> Path:
        path = tmp_path / "shared_matches.duckdb"
        _create_shared_db(path)
        return path

    def test_ghost_excluded_from_scoreboard(self, tmp_path: Path, shared_path: Path) -> None:
        """Un joueur fantôme (0,0,0,0) est exclu du scoreboard."""
        _insert_participant(shared_path, "m1", "ACTIVE", gamertag="Active")
        _insert_ghost(shared_path, "m1", "GHOST")

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_scoreboard("m1")
        xuids = [r["xuid"] for r in rows]

        assert "ACTIVE" in xuids
        assert "GHOST" not in xuids
        assert len(rows) == 1

    def test_null_stats_player_kept(self, tmp_path: Path, shared_path: Path) -> None:
        """Un joueur avec stats NULL (données partielles) reste dans le scoreboard."""
        _insert_participant(shared_path, "m2", "ACTIVE", gamertag="Active")
        _insert_null_participant(shared_path, "m2", "PARTIAL")

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_scoreboard("m2")
        xuids = [r["xuid"] for r in rows]

        assert "ACTIVE" in xuids
        assert "PARTIAL" in xuids
        assert len(rows) == 2

    def test_mix_ghost_null_normal(self, tmp_path: Path, shared_path: Path) -> None:
        """Mélange ghost + NULL + normal → seul le ghost est exclu."""
        _insert_participant(shared_path, "m3", "NORMAL", gamertag="Normal", kills=10, deaths=5)
        _insert_ghost(shared_path, "m3", "GHOST1")
        _insert_null_participant(shared_path, "m3", "PARTIAL")
        _insert_ghost(shared_path, "m3", "GHOST2")
        _insert_participant(
            shared_path, "m3", "SCORER", gamertag="Scorer", kills=0, deaths=0, assists=0, score=50
        )

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_scoreboard("m3")
        xuids = {r["xuid"] for r in rows}

        assert xuids == {"NORMAL", "PARTIAL", "SCORER"}

    def test_all_ghosts_returns_empty(self, tmp_path: Path, shared_path: Path) -> None:
        """Si tous les joueurs sont des fantômes, retourne une liste vide."""
        _insert_ghost(shared_path, "m4", "G1")
        _insert_ghost(shared_path, "m4", "G2")

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_scoreboard("m4")

        assert rows == []


# =============================================================================
# Tests load_match_players_stats avec ghost players
# =============================================================================


class TestPlayersStatsGhostFiltering:
    """Vérifie que load_match_players_stats exclut les fantômes."""

    @pytest.fixture()
    def shared_path(self, tmp_path: Path) -> Path:
        path = tmp_path / "shared_matches.duckdb"
        _create_shared_db(path)
        return path

    def test_ghost_excluded(self, tmp_path: Path, shared_path: Path) -> None:
        _insert_participant(shared_path, "m1", "P1", gamertag="Player1")
        _insert_ghost(shared_path, "m1", "GHOST")

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_players_stats("m1")
        xuids = [r["xuid"] for r in rows]

        assert "P1" in xuids
        assert "GHOST" not in xuids

    def test_null_stats_kept(self, tmp_path: Path, shared_path: Path) -> None:
        _insert_null_participant(shared_path, "m2", "PARTIAL")
        _insert_participant(shared_path, "m2", "ACTIVE", gamertag="Active")

        repo = _make_repo(tmp_path, shared_path)
        rows = repo.load_match_players_stats("m2")
        xuids = [r["xuid"] for r in rows]

        assert "PARTIAL" in xuids
        assert "ACTIVE" in xuids


# =============================================================================
# Tests list_top_teammates avec ghost et bots
# =============================================================================


class TestTopTeammatesGhostFiltering:
    """Vérifie que list_top_teammates exclut fantômes et bots."""

    @pytest.fixture()
    def shared_path(self, tmp_path: Path) -> Path:
        path = tmp_path / "shared_matches.duckdb"
        _create_shared_db(path)
        return path

    def test_ghost_teammate_excluded(self, tmp_path: Path, shared_path: Path) -> None:
        """Un coéquipier fantôme ne doit pas apparaître."""
        # Match m1 : ME + ACTIVE (même équipe, team 0)
        _insert_participant(shared_path, "m1", "ME", gamertag="Me", team_id=0)
        _insert_participant(shared_path, "m1", "ACTIVE", gamertag="Friend", team_id=0)
        # Match m1 : GHOST (même équipe, team 0, mais stats=0)
        _insert_ghost(shared_path, "m1", "GHOST", team_id=0)

        repo = _make_repo(tmp_path, shared_path, xuid="ME")
        teammates = repo.list_top_teammates(limit=10)
        teammate_xuids = [t[0] for t in teammates]

        assert "ACTIVE" in teammate_xuids
        assert "GHOST" not in teammate_xuids

    def test_bot_teammate_excluded(self, tmp_path: Path, shared_path: Path) -> None:
        """Un bot (xuid bid(...)) ne doit pas apparaître."""
        _insert_participant(shared_path, "m1", "ME", gamertag="Me", team_id=0)
        _insert_participant(shared_path, "m1", "REAL", gamertag="RealPlayer", team_id=0)
        _insert_participant(
            shared_path,
            "m1",
            "bid(33.0)",
            gamertag="343 Bot",
            team_id=0,
            kills=8,
            deaths=2,
        )

        repo = _make_repo(tmp_path, shared_path, xuid="ME")
        teammates = repo.list_top_teammates(limit=10)
        teammate_xuids = [t[0] for t in teammates]

        assert "REAL" in teammate_xuids
        assert "bid(33.0)" not in teammate_xuids

    def test_null_stats_teammate_kept(self, tmp_path: Path, shared_path: Path) -> None:
        """Un coéquipier avec stats partielles (NULL) reste visible."""
        _insert_participant(shared_path, "m1", "ME", gamertag="Me", team_id=0)
        _insert_null_participant(shared_path, "m1", "PARTIAL")
        # Forcer le team_id du partial au team 0
        conn = duckdb.connect(str(shared_path))
        conn.execute("UPDATE match_participants SET team_id = 0 WHERE xuid = 'PARTIAL'")
        conn.close()

        repo = _make_repo(tmp_path, shared_path, xuid="ME")
        teammates = repo.list_top_teammates(limit=10)
        teammate_xuids = [t[0] for t in teammates]

        assert "PARTIAL" in teammate_xuids


# =============================================================================
# Tests top_encountered avec ghost et bots
# =============================================================================


class TestTopEncounteredGhostFiltering:
    """Vérifie que _load_top_encountered exclut fantômes et bots."""

    @pytest.fixture()
    def shared_path(self, tmp_path: Path) -> Path:
        path = tmp_path / "shared_matches.duckdb"
        _create_shared_db(path)
        return path

    def test_ghost_excluded_from_encounters(self, tmp_path: Path, shared_path: Path) -> None:
        """Un adversaire fantôme ne doit pas apparaître dans les rencontres."""
        _insert_match_registry(shared_path, "m1")
        _insert_participant(shared_path, "m1", "ME", gamertag="Me", team_id=0, outcome=2)
        _insert_participant(
            shared_path,
            "m1",
            "REAL_OPP",
            gamertag="RealOpp",
            team_id=1,
            kills=8,
            deaths=3,
            outcome=3,
        )
        _insert_ghost(shared_path, "m1", "GHOST_OPP", team_id=1)

        from unittest.mock import patch

        repo = _make_repo(tmp_path, shared_path, xuid="ME")
        with patch("src.ui._cache_core.get_cached_repository_st", return_value=repo):
            from src.ui.pages.career_encounters_data import _load_top_encountered

            result = _load_top_encountered("ME", "/fake", limit=10)

        gamertags = [r["gamertag"] for r in result]
        assert "RealOpp" in gamertags
        # Le ghost ne doit pas apparaître
        ghost_gts = [r for r in result if "Ghost" in str(r.get("gamertag", ""))]
        assert len(ghost_gts) == 0

    def test_bot_excluded_from_encounters(self, tmp_path: Path, shared_path: Path) -> None:
        """Un bot (bid(...)) ne doit pas apparaître dans les rencontres."""
        _insert_match_registry(shared_path, "m1")
        _insert_participant(shared_path, "m1", "ME", gamertag="Me", team_id=0, outcome=2)
        _insert_participant(
            shared_path,
            "m1",
            "bid(10.0)",
            gamertag="343 Bot",
            team_id=1,
            kills=5,
            deaths=1,
            outcome=3,
        )
        _insert_participant(
            shared_path,
            "m1",
            "HUMAN",
            gamertag="Human",
            team_id=1,
            kills=3,
            deaths=4,
            outcome=3,
        )

        from unittest.mock import patch

        repo = _make_repo(tmp_path, shared_path, xuid="ME")
        with patch("src.ui._cache_core.get_cached_repository_st", return_value=repo):
            from src.ui.pages.career_encounters_data import _load_top_encountered

            result = _load_top_encountered("ME", "/fake", limit=10)

        xuids = [r.get("xuid") for r in result]
        assert "HUMAN" in xuids
        assert "bid(10.0)" not in xuids
