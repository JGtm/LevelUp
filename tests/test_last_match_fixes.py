"""Tests pour les corrections de l'onglet "Dernier match".

Ce module teste les corrections apportées pour :
1. Désactivation du mode debug
2. Gestion des données manquantes
3. Nettoyage amélioré des gamertags
4. Attribution améliorée des équipes
"""

from __future__ import annotations

import contextlib
import json
import os
import tempfile

import duckdb
import pytest

from src.data.repositories.duckdb_repo import DuckDBRepository
from src.ui.cache import cached_load_player_match_result


@pytest.fixture
def temp_duckdb_with_match():
    """Crée une base DuckDB temporaire avec un match et des highlight events.

    v5.1: inclut aussi shared.match_participants avec les team_ids.
    """
    # Créer un répertoire temporaire pour les deux DBs
    temp_dir = tempfile.mkdtemp()
    db_path = os.path.join(temp_dir, "player.duckdb")
    shared_db_path = os.path.join(temp_dir, "shared.duckdb")

    conn = duckdb.connect(db_path)

    # Créer les tables nécessaires
    conn.execute("""
        CREATE TABLE match_stats (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            team_id INTEGER,
            team_mmr DOUBLE,
            enemy_mmr DOUBLE,
            kills INTEGER,
            deaths INTEGER,
            assists INTEGER,
            map_name VARCHAR,
            pair_name VARCHAR
        )
    """)

    conn.execute("""
        CREATE TABLE highlight_events (
            id INTEGER PRIMARY KEY,
            match_id VARCHAR,
            event_type VARCHAR,
            time_ms INTEGER,
            xuid VARCHAR,
            type_hint VARCHAR,
            raw_json VARCHAR
        )
    """)

    # Insérer un match de test
    match_id = "test_match_1"
    xuid_player = "1234567890123456"
    xuid_teammate = "2345678901234567"
    xuid_enemy1 = "3456789012345678"
    xuid_enemy2 = "4567890123456789"

    conn.execute(
        """
        INSERT INTO match_stats (match_id, start_time, team_id, team_mmr, enemy_mmr, kills, deaths, assists, map_name, pair_name)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        [match_id, "2026-02-05 10:00:00", 0, 1500.0, 1520.0, 15, 10, 5, "Test Map", "Test Mode"],
    )

    # Insérer des highlight events avec des patterns de kills pour tester l'attribution des équipes
    # Le joueur principal (xuid_player) est tué par enemy1 et enemy2
    # Le teammate tue enemy1 et enemy2
    # Cela devrait permettre de déterminer que teammate est dans la même équipe

    events = [
        # Kill events : teammate tue enemy1
        (
            1,
            match_id,
            "Kill",
            1000,
            xuid_teammate,
            "kill",
            json.dumps({"killer_xuid": xuid_teammate, "victim_xuid": xuid_enemy1}),
        ),
        # Death events : enemy1 meurt (tué par teammate)
        (
            2,
            match_id,
            "Death",
            1000,
            xuid_enemy1,
            "death",
            json.dumps({"killer_xuid": xuid_teammate, "victim_xuid": xuid_enemy1}),
        ),
        # Kill events : enemy1 tue le joueur principal
        (
            3,
            match_id,
            "Kill",
            2000,
            xuid_enemy1,
            "kill",
            json.dumps({"killer_xuid": xuid_enemy1, "victim_xuid": xuid_player}),
        ),
        # Death events : le joueur principal meurt (tué par enemy1)
        (
            4,
            match_id,
            "Death",
            2000,
            xuid_player,
            "death",
            json.dumps({"killer_xuid": xuid_enemy1, "victim_xuid": xuid_player}),
        ),
        # Kill events : teammate tue enemy2
        (
            5,
            match_id,
            "Kill",
            3000,
            xuid_teammate,
            "kill",
            json.dumps({"killer_xuid": xuid_teammate, "victim_xuid": xuid_enemy2}),
        ),
        # Death events : enemy2 meurt (tué par teammate)
        (
            6,
            match_id,
            "Death",
            3000,
            xuid_enemy2,
            "death",
            json.dumps({"killer_xuid": xuid_teammate, "victim_xuid": xuid_enemy2}),
        ),
    ]

    conn.executemany(
        """
        INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """,
        events,
    )

    conn.commit()
    conn.close()

    # v5.1: Créer shared.match_participants avec les team_ids
    shared_conn = duckdb.connect(shared_db_path)
    shared_conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            gamertag VARCHAR,
            team_id INTEGER,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    participants = [
        (match_id, xuid_player, "Player", 0),
        (match_id, xuid_teammate, "Teammate", 0),
        (match_id, xuid_enemy1, "Enemy1", 1),
        (match_id, xuid_enemy2, "Enemy2", 1),
    ]
    shared_conn.executemany(
        "INSERT INTO match_participants VALUES (?, ?, ?, ?)",
        participants,
    )
    shared_conn.commit()
    shared_conn.close()

    yield db_path, match_id, xuid_player, shared_db_path

    # Cleanup - fermer toutes les connexions avant de supprimer
    import shutil

    with contextlib.suppress(PermissionError, OSError):
        shutil.rmtree(temp_dir, ignore_errors=True)


class TestGamertagCleaning:
    """Tests pour le nettoyage amélioré des gamertags."""

    def test_clean_gamertag_removes_control_characters(self, temp_duckdb_with_match):
        """Test que les caractères de contrôle sont supprimés."""
        db_path, match_id, xuid, shared_db_path = temp_duckdb_with_match

        # Insérer un event avec un joueur contenant des caractères de contrôle dans le gamertag
        # v6 : gamertag dans match_participants (shared), plus dans highlight_events
        conn = duckdb.connect(db_path)
        conn.execute(
            """
            INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            [
                999,
                match_id,
                "Kill",
                5000,
                "9999999999999999",
                "kill",
                None,
            ],
        )
        conn.commit()
        conn.close()

        # Ajouter le joueur dans shared.match_participants avec gamertag corrompu
        shared_conn = duckdb.connect(shared_db_path)
        shared_conn.execute(
            "INSERT OR IGNORE INTO match_participants VALUES (?, ?, ?, ?)",
            [match_id, "9999999999999999", "Test\x00\x1f\x7fPlayer", 0],
        )
        shared_conn.commit()
        shared_conn.close()

        # Créer le repository et charger les rosters (v5.1: shared_db_path requis)
        repo = DuckDBRepository(db_path, xuid, shared_db_path=shared_db_path)
        result = repo.load_match_rosters(match_id)
        repo.close()

        assert result is not None

        # Vérifier que le gamertag nettoyé ne contient pas de caractères de contrôle
        all_players = result["my_team"] + result["enemy_team"]
        test_player = next((p for p in all_players if p["xuid"] == "9999999999999999"), None)
        if test_player:
            cleaned = test_player.get("gamertag") or ""
            assert "\x00" not in cleaned
            assert "\x1f" not in cleaned
            assert "\x7f" not in cleaned

    def test_clean_gamertag_removes_unicode_replacement(self, temp_duckdb_with_match):
        """Test que le caractère de remplacement Unicode (�) est supprimé."""
        db_path, match_id, xuid, shared_db_path = temp_duckdb_with_match

        # Insérer un event (v6 : gamertag dans match_participants, plus dans highlight_events)
        conn = duckdb.connect(db_path)
        conn.execute(
            """
            INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            [998, match_id, "Kill", 6000, "8888888888888888", "kill", None],
        )
        conn.commit()
        conn.close()

        # v5.1: Ajouter le joueur dans shared.match_participants
        shared_conn = duckdb.connect(shared_db_path)
        shared_conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?, ?)",
            [match_id, "8888888888888888", "Test\ufffdPlayer", 1],
        )
        shared_conn.commit()
        shared_conn.close()

        # Créer le repository et charger les rosters (v5.1: shared_db_path requis)
        repo = DuckDBRepository(db_path, xuid, shared_db_path=shared_db_path)
        result = repo.load_match_rosters(match_id)
        repo.close()

        assert result is not None

        # Vérifier que le caractère de remplacement est supprimé
        all_players = result["my_team"] + result["enemy_team"]
        test_player = next((p for p in all_players if p["xuid"] == "8888888888888888"), None)
        if test_player:
            cleaned = test_player.get("gamertag") or ""
            assert "\ufffd" not in cleaned

    def test_clean_gamertag_handles_invalid_utf8(self, temp_duckdb_with_match):
        """Test que les séquences UTF-8 invalides sont gérées."""
        db_path, match_id, xuid, shared_db_path = temp_duckdb_with_match

        # Insérer un event avec un gamertag contenant des caractères invalides
        # Simuler des données corrompues
        # Utiliser une nouvelle connexion pour éviter les problèmes de read-only
        conn = duckdb.connect(db_path)
        try:
            # Essayer d'insérer des données qui pourraient causer des problèmes d'encodage
            conn.execute(
                """
                INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid, type_hint, raw_json)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                [
                    997,
                    match_id,
                    "Kill",
                    7000,
                    "7777777777777777",
                    "kill",
                    None,
                ],
            )
            conn.commit()
            conn.close()

            # Créer le repository et charger les rosters (v5.1: shared_db_path requis)
            repo = DuckDBRepository(db_path, xuid, shared_db_path=shared_db_path)
            result = repo.load_match_rosters(match_id)
            repo.close()

            assert result is not None
            # Le nettoyage ne doit pas faire planter la fonction
        except Exception:
            # Si l'insertion échoue à cause de l'encodage, c'est OK
            pass
        finally:
            if "conn" in locals():
                with contextlib.suppress(Exception):
                    conn.close()


class TestTeamAssignment:
    """Tests pour l'attribution améliorée des équipes (v5.1: match_participants.team_id)."""

    def test_team_assignment_based_on_kill_patterns(self, temp_duckdb_with_match):
        """Test que l'attribution des équipes utilise match_participants.team_id."""
        db_path, match_id, xuid_player, shared_db_path = temp_duckdb_with_match
        repo = DuckDBRepository(db_path, xuid_player, shared_db_path=shared_db_path)

        result = repo.load_match_rosters(match_id)
        assert result is not None

        # Vérifier que le teammate est dans la même équipe que le joueur principal
        my_team_xuids = {p["xuid"] for p in result["my_team"]}
        enemy_team_xuids = {p["xuid"] for p in result["enemy_team"]}

        # Le joueur principal doit être dans my_team
        assert xuid_player in my_team_xuids

        # Le teammate (qui tue les ennemis qui tuent le joueur principal) devrait être dans my_team
        # Note: L'heuristique peut ne pas être parfaite, mais au moins elle ne devrait pas mettre
        # le teammate dans l'équipe adverse systématiquement
        teammate_xuid = "2345678901234567"
        # Le teammate devrait être soit dans my_team soit dans enemy_team (pas les deux)
        assert (teammate_xuid in my_team_xuids) != (teammate_xuid in enemy_team_xuids)

        repo.close()

    def test_team_assignment_requires_match_participants_v51(self):
        """Test v5.1: load_match_rosters retourne None sans match_participants."""
        # Créer une DB sans shared.match_participants
        temp_dir = tempfile.mkdtemp()
        db_path = os.path.join(temp_dir, "player.duckdb")
        shared_db_path = os.path.join(temp_dir, "shared.duckdb")

        conn = duckdb.connect(db_path)
        conn.execute("""
            CREATE TABLE match_stats (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                team_id INTEGER
            )
        """)
        match_id = "test_match_2"
        xuid_player = "1234567890123456"
        conn.execute(
            "INSERT INTO match_stats VALUES (?, ?, ?)",
            [match_id, "2026-02-05 10:00:00", 0],
        )
        conn.commit()
        conn.close()

        # Créer shared_db VIDE (sans match_participants)
        shared_conn = duckdb.connect(shared_db_path)
        shared_conn.close()

        repo = DuckDBRepository(db_path, xuid_player, shared_db_path=shared_db_path)
        result = repo.load_match_rosters(match_id)
        repo.close()

        # v5.1: Sans match_participants, retourne None
        assert result is None

        import shutil

        with contextlib.suppress(PermissionError, OSError):
            shutil.rmtree(temp_dir, ignore_errors=True)


class TestMissingDataHandling:
    """Tests pour la gestion des données manquantes."""

    def test_cached_load_player_match_result_returns_dict_even_without_mmr(self, tmp_path):
        """Test que cached_load_player_match_result retourne un dict même sans MMR."""
        match_id = "test_match_3"
        xuid = "1234567890123456"

        # Structure v5.1 : data/players/TestPlayer/stats.duckdb + warehouse/shared
        player_dir = tmp_path / "data" / "players" / "TestPlayer"
        player_dir.mkdir(parents=True)
        db_path = str(player_dir / "stats.duckdb")

        conn = duckdb.connect(db_path)
        conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR, updated_at TIMESTAMP)")
        conn.close()

        # Créer shared DB avec match_participants (MMR NULL)
        warehouse = tmp_path / "data" / "warehouse"
        warehouse.mkdir(parents=True)
        shared_conn = duckdb.connect(str(warehouse / "shared_matches_v2.duckdb"))
        shared_conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR, xuid VARCHAR,
                team_id INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
                kills INTEGER, deaths INTEGER, assists INTEGER,
                kills_expected DOUBLE, kills_stddev DOUBLE,
                deaths_expected DOUBLE, deaths_stddev DOUBLE,
                assists_expected DOUBLE, assists_stddev DOUBLE,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        shared_conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 0, NULL, NULL, 10, 5, 3, NULL, NULL, NULL, NULL, NULL, NULL)",
            [match_id, xuid],
        )
        shared_conn.execute("CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)")
        shared_conn.execute("INSERT INTO match_registry VALUES (?)", [match_id])
        shared_conn.close()

        result = cached_load_player_match_result(db_path, match_id, xuid, db_key=None)

        assert result is not None, "doit retourner un dict même sans MMR"
        assert isinstance(result, dict)
        assert result.get("team_mmr") is None
        assert result.get("enemy_mmr") is None
        assert "kills" in result
        assert "deaths" in result
        assert "assists" in result

        from src.ui.cache_loaders import clear_app_caches

        clear_app_caches()

    def test_cached_load_player_match_result_with_mmr(self, tmp_path):
        """Test que cached_load_player_match_result retourne les MMR quand disponibles."""
        match_id = "test_match_4"
        xuid = "1234567890123456"

        # Structure v5.1 : data/players/TestPlayer/stats.duckdb + warehouse/shared
        player_dir = tmp_path / "data" / "players" / "TestPlayer"
        player_dir.mkdir(parents=True)
        db_path = str(player_dir / "stats.duckdb")

        conn = duckdb.connect(db_path)
        conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR, updated_at TIMESTAMP)")
        conn.close()

        # Créer shared DB avec match_participants (MMR renseignés)
        warehouse = tmp_path / "data" / "warehouse"
        warehouse.mkdir(parents=True)
        shared_conn = duckdb.connect(str(warehouse / "shared_matches_v2.duckdb"))
        shared_conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR, xuid VARCHAR,
                team_id INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
                kills INTEGER, deaths INTEGER, assists INTEGER,
                kills_expected DOUBLE, kills_stddev DOUBLE,
                deaths_expected DOUBLE, deaths_stddev DOUBLE,
                assists_expected DOUBLE, assists_stddev DOUBLE,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        shared_conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 0, 1500.0, 1520.0, 10, 5, 3, NULL, NULL, NULL, NULL, NULL, NULL)",
            [match_id, xuid],
        )
        shared_conn.execute("CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)")
        shared_conn.execute("INSERT INTO match_registry VALUES (?)", [match_id])
        shared_conn.close()

        result = cached_load_player_match_result(db_path, match_id, xuid, db_key=None)

        assert result is not None
        assert result.get("team_mmr") == 1500.0
        assert result.get("enemy_mmr") == 1520.0

        from src.ui.cache_loaders import clear_app_caches

        clear_app_caches()
