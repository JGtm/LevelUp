"""Tests pour les nouvelles features v5.2 sans couverture existante.

Couvre :
- _apply_experience_filter : pré-filtre PvP/PvE du sélecteur expérience
- load_match_scoreboard   : tableau de bord complet d'un match (DuckDB)
- load_match_pve_stats    : lecture stats PvE depuis shared_pve.duckdb
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import polars as pl
import pytest

# =============================================================================
# Helpers communs
# =============================================================================


def _create_shared_db(path: Path) -> None:
    """Crée un shared_matches.duckdb minimal avec les 3 tables du scoreboard."""
    conn = duckdb.connect(str(path))
    conn.execute(
        """
        CREATE TABLE match_participants (
            match_id            VARCHAR,
            xuid                VARCHAR,
            gamertag            VARCHAR,
            team_id             INTEGER,
            rank                INTEGER,
            score               INTEGER,
            kills               INTEGER,
            deaths              INTEGER,
            assists             INTEGER,
            kda                 DOUBLE,
            max_killing_spree   INTEGER,
            headshot_kills      INTEGER,
            shots_fired         INTEGER,
            shots_hit           INTEGER,
            accuracy            DOUBLE,
            melee_kills         INTEGER,
            power_weapon_kills  INTEGER,
            damage_dealt        BIGINT,
            damage_taken        BIGINT,
            avg_life_seconds    DOUBLE
        )
        """
    )
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
    conn.execute(
        "CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)"
    )
    conn.close()


def _create_empty_player_db(path: Path) -> None:
    """Crée un stats.duckdb vide (le fichier doit exister pour DuckDBRepository)."""
    duckdb.connect(str(path)).close()


def _insert_participant(
    shared_path: Path,
    match_id: str,
    xuid: str,
    **kwargs: object,
) -> None:
    """Insère un joueur dans match_participants directement dans le fichier shared."""
    defaults = {
        "gamertag": None,
        "team_id": 0,
        "rank": 1,
        "score": 0,
        "kills": 0,
        "deaths": 0,
        "assists": 0,
        "kda": 0.0,
        "max_killing_spree": 0,
        "headshot_kills": 0,
        "shots_fired": 0,
        "shots_hit": 0,
        "accuracy": 0.0,
        "melee_kills": 0,
        "power_weapon_kills": 0,
        "damage_dealt": 0,
        "damage_taken": 0,
        "avg_life_seconds": 0.0,
    }
    defaults.update(kwargs)
    conn = duckdb.connect(str(shared_path))
    conn.execute(
        """
        INSERT INTO match_participants VALUES (
            ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?
        )
        """,
        [
            match_id,
            xuid,
            defaults["gamertag"],
            defaults["team_id"],
            defaults["rank"],
            defaults["score"],
            defaults["kills"],
            defaults["deaths"],
            defaults["assists"],
            defaults["kda"],
            defaults["max_killing_spree"],
            defaults["headshot_kills"],
            defaults["shots_fired"],
            defaults["shots_hit"],
            defaults["accuracy"],
            defaults["melee_kills"],
            defaults["power_weapon_kills"],
            defaults["damage_dealt"],
            defaults["damage_taken"],
            defaults["avg_life_seconds"],
        ],
    )
    conn.close()


def _insert_medal(
    shared_path: Path, match_id: str, xuid: str, medal_name_id: int, count: int
) -> None:
    conn = duckdb.connect(str(shared_path))
    conn.execute(
        "INSERT INTO medals_earned VALUES (?, ?, ?, ?)",
        [match_id, xuid, medal_name_id, count],
    )
    conn.close()


def _insert_alias(shared_path: Path, xuid: str, gamertag: str) -> None:
    conn = duckdb.connect(str(shared_path))
    conn.execute("INSERT INTO xuid_aliases VALUES (?, ?)", [xuid, gamertag])
    conn.close()


# =============================================================================
# Tests _apply_experience_filter
# =============================================================================


class TestApplyExperienceFilter:
    """Tests du pré-filtre 'Type d'expérience' (v5.2).

    _apply_experience_filter() est une fonction pure Polars — aucune dépendance
    Streamlit ni DuckDB.
    """

    def _df(self, playlists: list[str]) -> pl.DataFrame:
        return pl.DataFrame({"playlist_ui": playlists})

    def test_empty_selection_no_filter(self) -> None:
        """Aucune option cochée → aucun filtre."""
        from src.app.filters_render import _apply_experience_filter

        df = self._df(["Partie rapide", "Firefight Standard"])
        result = _apply_experience_filter(df, [], ["Partie rapide", "Firefight Standard"])
        assert len(result) == 2

    def test_all_selected_no_filter(self) -> None:
        """Les 3 options sélectionnées → aucun filtre (identité)."""
        from src.app.filters_render import _EXPERIENCE_TYPES_OPTIONS, _apply_experience_filter

        df = self._df(["Partie rapide", "Arène classée", "Firefight Standard"])
        result = _apply_experience_filter(
            df, _EXPERIENCE_TYPES_OPTIONS, df["playlist_ui"].to_list()
        )
        assert len(result) == 3

    def test_pve_only_keeps_firefight_playlists(self) -> None:
        """PVE uniquement → only playlists qui contiennent 'firefight'."""
        from src.app.filters_render import _apply_experience_filter

        all_pls = ["Partie rapide", "Arène classée", "Firefight Standard", "Firefight KOTH"]
        df = self._df(all_pls)
        result = _apply_experience_filter(df, ["PVE"], all_pls)
        kept = result["playlist_ui"].to_list()
        assert "Firefight Standard" in kept
        assert "Firefight KOTH" in kept
        assert "Partie rapide" not in kept
        assert "Arène classée" not in kept

    def test_pvp_classe_only_keeps_ranked(self) -> None:
        """PVP classé → playlists contenant 'classé' (hors Firefight)."""
        from src.app.filters_render import _apply_experience_filter

        all_pls = ["Partie rapide", "Arène classée", "Firefight Standard"]
        df = self._df(all_pls)
        result = _apply_experience_filter(df, ["PVP classé"], all_pls)
        kept = result["playlist_ui"].to_list()
        assert "Arène classée" in kept
        assert "Partie rapide" not in kept
        assert "Firefight Standard" not in kept

    def test_pvp_non_classe_only(self) -> None:
        """PVP non classé → ni Firefight, ni ranked."""
        from src.app.filters_render import _apply_experience_filter

        all_pls = ["Partie rapide", "Arène classée", "Firefight Standard"]
        df = self._df(all_pls)
        result = _apply_experience_filter(df, ["PVP non classé"], all_pls)
        kept = result["playlist_ui"].to_list()
        assert "Partie rapide" in kept
        assert "Arène classée" not in kept
        assert "Firefight Standard" not in kept

    def test_classique_not_a_false_positive_for_ranked(self) -> None:
        """Régression : 'Fiesta Classique' ne doit PAS matcher 'PVP classé'.

        Avant la correction v5.2 (str.contains('class') → 'classé'),
        'Classique' levait un faux positif.
        """
        from src.app.filters_render import _apply_experience_filter

        all_pls = ["Fiesta Classique", "Arène classée"]
        df = self._df(all_pls)
        result = _apply_experience_filter(df, ["PVP classé"], all_pls)
        kept = result["playlist_ui"].to_list()
        assert "Arène classée" in kept
        assert "Fiesta Classique" not in kept  # ← c'est le bug corrigé

    def test_pve_and_ranked_union(self) -> None:
        """PVE + PVP classé → union des deux groupes, PVP non classé exclu."""
        from src.app.filters_render import _apply_experience_filter

        all_pls = ["Partie rapide", "Arène classée", "Firefight Standard"]
        df = self._df(all_pls)
        result = _apply_experience_filter(df, ["PVE", "PVP classé"], all_pls)
        kept = result["playlist_ui"].to_list()
        assert "Firefight Standard" in kept
        assert "Arène classée" in kept
        assert "Partie rapide" not in kept

    def test_empty_dataframe_returns_empty(self) -> None:
        from src.app.filters_render import _apply_experience_filter

        df = pl.DataFrame({"playlist_ui": pl.Series([], dtype=pl.Utf8)})
        result = _apply_experience_filter(df, ["PVE"], [])
        assert len(result) == 0


# =============================================================================
# Tests load_match_scoreboard
# =============================================================================


class TestLoadMatchScoreboard:
    """Tests de la requête DuckDB du scoreboard (RosterLoaderMixin)."""

    @pytest.fixture
    def shared_path(self, tmp_path: Path) -> Path:
        path = tmp_path / "shared_matches.duckdb"
        _create_shared_db(path)
        return path

    @pytest.fixture
    def repo(self, tmp_path: Path, shared_path: Path):
        from src.data.repositories import DuckDBRepository

        player_db = tmp_path / "stats.duckdb"
        _create_empty_player_db(player_db)

        return DuckDBRepository(
            player_db_path=player_db,
            xuid="100000000000001",
            shared_db_path=shared_path,
            # metadata absent → auto-détection échoue silencieusement
            metadata_db_path=tmp_path / "meta_inexistant.duckdb",
            read_only=False,
        )

    def test_empty_match_id_returns_empty(self, repo) -> None:
        assert repo.load_match_scoreboard("") == []

    def test_unknown_match_returns_empty(self, repo) -> None:
        assert repo.load_match_scoreboard("inexistant-uuid") == []

    def test_single_player_all_fields(self, repo, shared_path: Path) -> None:
        _insert_participant(
            shared_path,
            "m001",
            "111",
            gamertag="SpartanX",
            team_id=0,
            rank=1,
            score=2500,
            kills=15,
            deaths=5,
            assists=3,
            kda=3.2,
            headshot_kills=7,
        )
        rows = repo.load_match_scoreboard("m001")
        assert len(rows) == 1
        p = rows[0]
        assert p["xuid"] == "111"
        assert p["gamertag"] == "SpartanX"
        assert p["score"] == 2500
        assert p["kills"] == 15
        assert p["deaths"] == 5
        assert p["assists"] == 3
        assert p["perfect_kills"] == 0  # Aucune médaille

    def test_sorted_by_team_then_rank(self, repo, shared_path: Path) -> None:
        _insert_participant(shared_path, "m002", "A", team_id=1, rank=2)
        _insert_participant(shared_path, "m002", "B", team_id=0, rank=1)
        _insert_participant(shared_path, "m002", "C", team_id=1, rank=1)

        rows = repo.load_match_scoreboard("m002")
        assert len(rows) == 3
        ordering = [r["xuid"] for r in rows]
        assert ordering == ["B", "C", "A"]  # team 0 > team 1, rank 1 < rank 2

    def test_perfect_kills_medal_aggregated(self, repo, shared_path: Path) -> None:
        _insert_participant(shared_path, "m003", "999", team_id=0, rank=1)
        _insert_medal(shared_path, "m003", "999", 1512363953, 2)

        rows = repo.load_match_scoreboard("m003")
        assert rows[0]["perfect_kills"] == 2

    def test_gamertag_fallback_to_xuid_aliases(self, repo, shared_path: Path) -> None:
        """NULL gamertag dans match_participants → résolution via xuid_aliases."""
        _insert_participant(shared_path, "m004", "444", gamertag=None, team_id=0, rank=1)
        _insert_alias(shared_path, "444", "AliasGamer")

        rows = repo.load_match_scoreboard("m004")
        assert rows[0]["gamertag"] == "AliasGamer"

    def test_null_team_id_sorted_last(self, repo, shared_path: Path) -> None:
        """Joueurs sans team_id (NULL) placés en fin, triés après les équipes."""
        _insert_participant(shared_path, "m005", "SOLO", team_id=None, rank=1)
        _insert_participant(shared_path, "m005", "TEAM", team_id=0, rank=1)

        rows = repo.load_match_scoreboard("m005")
        assert len(rows) == 2
        assert rows[0]["xuid"] == "TEAM"  # team_id=0 en premier
        assert rows[1]["xuid"] == "SOLO"  # NULL via NULLS LAST

    def test_result_contains_all_expected_keys(self, repo, shared_path: Path) -> None:
        _insert_participant(shared_path, "m006", "555", team_id=0, rank=1)
        rows = repo.load_match_scoreboard("m006")
        expected_keys = {
            "xuid",
            "gamertag",
            "team_id",
            "rank",
            "score",
            "kills",
            "deaths",
            "assists",
            "kda",
            "max_killing_spree",
            "headshot_kills",
            "shots_fired",
            "shots_hit",
            "accuracy",
            "melee_kills",
            "power_weapon_kills",
            "damage_dealt",
            "damage_taken",
            "avg_life_seconds",
            "perfect_kills",
        }
        assert expected_keys == set(rows[0].keys())


# =============================================================================
# Tests load_match_pve_stats (CitationEngine)
# =============================================================================


class TestLoadMatchPveStats:
    """Tests du chargement des stats PvE depuis shared_pve.duckdb."""

    @pytest.fixture
    def pve_db_path(self, tmp_path: Path) -> Path:
        from src.data.sync.migrations import ensure_pve_schema

        path = tmp_path / "shared_pve.duckdb"
        conn = duckdb.connect(str(path))
        ensure_pve_schema(conn)
        conn.close()
        return path

    def _engine(self, tmp_path: Path, pve_db_path: Path | None):
        from src.analysis.citations.engine import CitationEngine

        player_db = tmp_path / "stats.duckdb"
        duckdb.connect(str(player_db)).close()

        engine = CitationEngine(
            db_path=player_db,
            xuid="100000000000001",
            shared_db_path=False,  # Désactive la recherche auto de shared
        )
        engine._pve_db_path = pve_db_path  # Injection directe
        return engine

    def test_returns_empty_when_pve_path_none(self, tmp_path: Path) -> None:
        engine = self._engine(tmp_path, None)
        assert engine.load_match_pve_stats("any") == {}

    def test_returns_empty_for_unknown_match(self, tmp_path: Path, pve_db_path: Path) -> None:
        engine = self._engine(tmp_path, pve_db_path)
        assert engine.load_match_pve_stats("inexistant") == {}

    def test_returns_row_as_dict(self, tmp_path: Path, pve_db_path: Path) -> None:
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.models import PveMatchStatsRow

        conn = duckdb.connect(str(pve_db_path))
        batch_insert_pve_stats(
            conn,
            [
                PveMatchStatsRow(
                    match_id="ff-001",
                    xuid="100000000000001",
                    grunt_kills=10,
                    elite_kills=5,
                    boss_kills=2,
                    total_enemy_kills=15,
                )
            ],
        )
        conn.close()

        engine = self._engine(tmp_path, pve_db_path)
        result = engine.load_match_pve_stats("ff-001")

        assert result["match_id"] == "ff-001"
        assert result["xuid"] == "100000000000001"
        assert result["grunt_kills"] == 10
        assert result["elite_kills"] == 5
        assert result["boss_kills"] == 2
        assert result["total_enemy_kills"] == 15

    def test_filters_by_xuid(self, tmp_path: Path, pve_db_path: Path) -> None:
        """Seule la ligne correspondant au xuid du joueur est retournée."""
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.models import PveMatchStatsRow

        conn = duckdb.connect(str(pve_db_path))
        batch_insert_pve_stats(
            conn,
            [
                PveMatchStatsRow(match_id="ff-multi", xuid="100000000000001", grunt_kills=8),
                PveMatchStatsRow(match_id="ff-multi", xuid="100000000000002", grunt_kills=3),
            ],
        )
        conn.close()

        engine = self._engine(tmp_path, pve_db_path)
        result = engine.load_match_pve_stats("ff-multi")

        assert result["xuid"] == "100000000000001"
        assert result["grunt_kills"] == 8

    def test_returns_empty_on_inaccessible_db(self, tmp_path: Path) -> None:
        """Si le fichier PvE est inaccessible, retourne {} sans lever d'exception."""
        engine = self._engine(tmp_path, Path("/chemin/inexistant/pve.duckdb"))
        result = engine.load_match_pve_stats("any-match")
        assert result == {}
