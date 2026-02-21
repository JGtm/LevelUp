"""Tests d'intégration v5.2 — Pipeline PvE citations + Scoreboard.

Deux flux non couverts par les tests d'intégration existants :

1. Citations PvE (pve_stat mapping_type)
   - shared_pve.duckdb alimenté via batch_insert_pve_stats
   - CitationEngine.load_match_pve_stats + compute_and_store_for_match
   - Filtrage par xuid correct
   - Fallback gracieux si shared_pve.duckdb absent

2. Scoreboard end-to-end (load_match_scoreboard)
   - DuckDBRepository attaché à shared_matches.duckdb réel
   - Champs attendus, tri (team_id, rank), résolution gamertag, Perfect Kill
   - Cas limites : match inexistant, team_id NULL
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest


# =============================================================================
# Helpers de setup
# =============================================================================


def _create_pve_env(tmp_path: Path) -> tuple[Path, Path, Path]:
    """Crée la structure complète pour les tests PvE citations.

    Structure :
        {tmp_path}/data/players/TestPlayer/stats.duckdb  ← DB joueur
        {tmp_path}/data/warehouse/shared_pve.duckdb       ← auto-détectée
        {tmp_path}/data/warehouse/metadata.duckdb         ← citation_mappings
    """
    player_dir = tmp_path / "data" / "players" / "TestPlayer"
    player_dir.mkdir(parents=True)
    warehouse_dir = tmp_path / "data" / "warehouse"
    warehouse_dir.mkdir(parents=True)

    stats_path = player_dir / "stats.duckdb"
    pve_path = warehouse_dir / "shared_pve.duckdb"
    meta_path = warehouse_dir / "metadata.duckdb"

    # --- Player DB (tables minimales pour CitationEngine) ---
    conn = duckdb.connect(str(stats_path))
    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_citations (
            match_id TEXT NOT NULL,
            citation_name_norm TEXT NOT NULL,
            value INTEGER NOT NULL,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_stats (
            match_id TEXT PRIMARY KEY,
            kills INTEGER DEFAULT 0,
            deaths INTEGER DEFAULT 0,
            game_variant_category INTEGER DEFAULT 41  -- Firefight
        )
    """)
    conn.execute("INSERT INTO match_stats VALUES ('ff-001', 5, 2, 41)")
    conn.execute("INSERT INTO match_stats VALUES ('ff-002', 8, 1, 41)")
    conn.close()

    # --- Metadata DB : une citation pve_stat sur grunt_kills ---
    meta = duckdb.connect(str(meta_path))
    meta.execute("""
        CREATE TABLE IF NOT EXISTS citation_mappings (
            citation_name_norm TEXT PRIMARY KEY,
            citation_name_display TEXT NOT NULL,
            mapping_type TEXT NOT NULL,
            medal_id BIGINT,
            medal_ids TEXT,
            stat_name TEXT,
            award_name TEXT,
            award_category TEXT,
            custom_function TEXT,
            composite_children TEXT,
            confidence TEXT,
            notes TEXT,
            enabled BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            image_path TEXT,
            category TEXT,
            description TEXT,
            tier_targets TEXT
        )
    """)
    meta.execute("""
        INSERT INTO citation_mappings
            (citation_name_norm, citation_name_display, mapping_type, stat_name, confidence, notes, enabled)
        VALUES
            ('chasseur_grunt', 'Chasseur de Grunts', 'pve_stat', 'grunt_kills', 'high', 'test', TRUE),
            ('chasseur_elite', 'Chasseur d''Élites', 'pve_stat', 'elite_kills', 'high', 'test', TRUE)
    """)
    meta.close()

    # --- shared_pve.duckdb avec données ---
    from src.data.sync.migrations import ensure_pve_schema
    from src.data.sync.batch_insert import batch_insert_pve_stats
    from src.data.sync.models import PveMatchStatsRow

    pve_conn = duckdb.connect(str(pve_path))
    ensure_pve_schema(pve_conn)
    batch_insert_pve_stats(
        pve_conn,
        [
            PveMatchStatsRow(
                match_id="ff-001",
                xuid="xuid_me",
                total_enemy_kills=20,
                boss_kills=1,
                grunt_kills=8,
                elite_kills=3,
                jackal_kills=4,
                brute_kills=2,
                hunter_kills=1,
                skimmer_kills=1,
                sentinel_kills=0,
                marine_kills=0,
                pve_bits=0,
            ),
            # Deuxième joueur sur le même match (ne doit PAS être chargé pour xuid_me)
            PveMatchStatsRow(
                match_id="ff-001",
                xuid="xuid_friend",
                total_enemy_kills=15,
                boss_kills=0,
                grunt_kills=12,
                elite_kills=1,
                jackal_kills=0,
                brute_kills=2,
                hunter_kills=0,
                skimmer_kills=0,
                sentinel_kills=0,
                marine_kills=0,
                pve_bits=0,
            ),
            # ff-002 : pas d'entrée pour xuid_me (match non synchronisé)
        ],
    )
    pve_conn.close()

    return stats_path, pve_path, meta_path


def _create_shared_for_scoreboard(tmp_path: Path) -> tuple[Path, Path, Path]:
    """Crée la structure shared_matches.duckdb + stats.duckdb pour le scoreboard.

    Données :
        - match-sc01 : 2 équipes (team 0 et team 1), 4 joueurs
        - 1 joueur sans team_id (classé dernier)
        - 2 Perfect Kills pour xuid_a
    """
    player_dir = tmp_path / "data" / "players" / "Spartan"
    player_dir.mkdir(parents=True)
    warehouse_dir = tmp_path / "data" / "warehouse"
    warehouse_dir.mkdir(parents=True)

    stats_path = player_dir / "stats.duckdb"
    shared_path = warehouse_dir / "shared_matches.duckdb"
    meta_path = warehouse_dir / "metadata.duckdb"

    # --- Minimal player DB ---
    conn = duckdb.connect(str(stats_path))
    conn.close()

    # --- Metadata ---
    meta = duckdb.connect(str(meta_path))
    meta.execute("CREATE TABLE IF NOT EXISTS playlists (asset_id VARCHAR PRIMARY KEY)")
    meta.close()

    # --- shared_matches.duckdb ---
    sh = duckdb.connect(str(shared_path))
    sh.execute("""
        CREATE TABLE IF NOT EXISTS match_participants (
            match_id     VARCHAR NOT NULL,
            xuid         VARCHAR NOT NULL,
            gamertag     VARCHAR,
            team_id      INTEGER,
            rank         INTEGER,
            score        INTEGER,
            kills        INTEGER,
            deaths       INTEGER,
            assists      INTEGER,
            kda          FLOAT,
            max_killing_spree  INTEGER,
            headshot_kills     INTEGER,
            shots_fired        INTEGER,
            shots_hit          INTEGER,
            accuracy           FLOAT,
            melee_kills        INTEGER,
            power_weapon_kills INTEGER,
            damage_dealt       INTEGER,
            damage_taken       INTEGER,
            avg_life_seconds   FLOAT,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    sh.execute("""
        CREATE TABLE IF NOT EXISTS xuid_aliases (
            xuid     VARCHAR PRIMARY KEY,
            gamertag VARCHAR
        )
    """)
    sh.execute("""
        CREATE TABLE IF NOT EXISTS medals_earned (
            match_id       VARCHAR NOT NULL,
            xuid           VARCHAR NOT NULL,
            medal_name_id  BIGINT  NOT NULL,
            count          INTEGER NOT NULL DEFAULT 1,
            PRIMARY KEY (match_id, xuid, medal_name_id)
        )
    """)

    # Joueurs du match-sc01
    players = [
        # (match_id, xuid, gamertag,   team_id, rank, score, kills, deaths, assists, kda, spree, hs, sf, sh, acc, melee, pw, dd, dt, avg_life)
        ("match-sc01", "xuid_a", "Alpha",   0, 1, 1200, 12, 4, 3, 2.5, 3, 4, 200, 150, 75.0, 1, 2, 5000, 2000, 30.0),
        ("match-sc01", "xuid_b", "Bravo",   0, 2, 900,  8,  6, 2, 1.3, 2, 2, 180, 100, 55.5, 0, 1, 3500, 3000, 20.0),
        ("match-sc01", "xuid_c", "Charlie", 1, 1, 1000, 10, 5, 1, 2.0, 4, 5, 220, 180, 81.8, 2, 3, 4500, 2500, 25.0),
        ("match-sc01", "xuid_d", None,      1, 2, 700,  6,  8, 0, 0.75,1, 1, 160, 80,  50.0, 0, 0, 2000, 4000, 15.0),
        # Joueur sans team_id (classé dernier)
        ("match-sc01", "xuid_e", "Echo",    None, 5, 300, 2, 10, 0, 0.2, 0, 0, 100, 30, 30.0, 0, 0, 800, 5000, 10.0),
    ]
    sh.executemany(
        "INSERT INTO match_participants VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
        players,
    )

    # Alias pour xuid_d (gamertag NULL dans match_participants → résolu via xuid_aliases)
    sh.execute("INSERT INTO xuid_aliases VALUES ('xuid_d', 'Delta')")

    # Perfect Kills pour xuid_a : 2 médailles
    sh.execute(
        "INSERT INTO medals_earned VALUES ('match-sc01', 'xuid_a', 1512363953, 2)"
    )
    # Autre médaille (non Perfect Kill) pour xuid_a → ne doit PAS être comptée
    sh.execute(
        "INSERT INTO medals_earned VALUES ('match-sc01', 'xuid_a', 9999999999, 5)"
    )

    sh.close()
    return stats_path, shared_path, meta_path


# =============================================================================
# Tests Citations PvE
# =============================================================================


class TestCitationsPveIntegration:
    """Tests d'intégration pour les citations de type pve_stat (v5.2)."""

    def test_pve_db_auto_detecte(self, tmp_path: Path) -> None:
        """CitationEngine détecte automatiquement shared_pve.duckdb via le path joueur."""
        stats_path, pve_path, meta_path = _create_pve_env(tmp_path)

        from src.analysis.citations.engine import CitationEngine

        conn = duckdb.connect(str(stats_path))
        engine = CitationEngine(str(stats_path), "xuid_me", conn=conn)

        assert engine._pve_db_path is not None
        assert engine._pve_db_path == pve_path

        conn.close()

    def test_load_match_pve_stats_retourne_donnees_correctes(self, tmp_path: Path) -> None:
        """load_match_pve_stats retourne les stats du bon joueur pour le bon match."""
        stats_path, _pve_path, meta_path = _create_pve_env(tmp_path)

        from src.analysis.citations.engine import CitationEngine

        conn = duckdb.connect(str(stats_path))
        engine = CitationEngine(str(stats_path), "xuid_me", conn=conn)

        result = engine.load_match_pve_stats("ff-001")

        assert result != {}
        assert result["grunt_kills"] == 8
        assert result["elite_kills"] == 3
        assert result["total_enemy_kills"] == 20
        assert result["xuid"] == "xuid_me"

        conn.close()

    def test_load_match_pve_stats_filtre_par_xuid(self, tmp_path: Path) -> None:
        """load_match_pve_stats ne retourne PAS les stats d'un autre joueur du même match."""
        stats_path, _pve_path, meta_path = _create_pve_env(tmp_path)

        from src.analysis.citations.engine import CitationEngine

        conn = duckdb.connect(str(stats_path))
        engine = CitationEngine(str(stats_path), "xuid_me", conn=conn)
        result = engine.load_match_pve_stats("ff-001")

        # Les stats de xuid_friend ont grunt_kills=12, les nôtres = 8
        assert result.get("grunt_kills") == 8  # xuid_me, pas xuid_friend

        conn.close()

    def test_load_match_pve_stats_match_absent_retourne_vide(self, tmp_path: Path) -> None:
        """Un match sans données PvE retourne un dict vide."""
        stats_path, _pve_path, meta_path = _create_pve_env(tmp_path)

        from src.analysis.citations.engine import CitationEngine

        conn = duckdb.connect(str(stats_path))
        engine = CitationEngine(str(stats_path), "xuid_me", conn=conn)

        assert engine.load_match_pve_stats("ff-002") == {}
        assert engine.load_match_pve_stats("match-inexistant") == {}

        conn.close()

    def test_pve_stat_citation_computed_and_stored(self, tmp_path: Path) -> None:
        """compute_and_store_for_match calcule correctement les citations pve_stat."""
        stats_path, _pve_path, meta_path = _create_pve_env(tmp_path)

        from src.analysis.citations.engine import CitationEngine

        conn = duckdb.connect(str(stats_path))
        engine = CitationEngine(str(stats_path), "xuid_me", conn=conn)

        n = engine.compute_and_store_for_match("ff-001", conn=conn)
        assert n > 0  # Au moins une citation stockée

        # Vérifier la valeur de grunt_kills
        row = conn.execute(
            "SELECT value FROM match_citations "
            "WHERE match_id = 'ff-001' AND citation_name_norm = 'chasseur_grunt'"
        ).fetchone()
        assert row is not None
        assert row[0] == 8  # grunt_kills de xuid_me

        # Vérifier elite_kills
        row2 = conn.execute(
            "SELECT value FROM match_citations "
            "WHERE match_id = 'ff-001' AND citation_name_norm = 'chasseur_elite'"
        ).fetchone()
        assert row2 is not None
        assert row2[0] == 3

        conn.close()

    def test_pve_stat_absent_si_pve_db_inexistante(self, tmp_path: Path) -> None:
        """Sans shared_pve.duckdb, load_match_pve_stats retourne {} gracieusement."""
        # Setup sans créer shared_pve.duckdb
        player_dir = tmp_path / "data" / "players" / "Ghost"
        player_dir.mkdir(parents=True)
        stats_path = player_dir / "stats.duckdb"

        conn = duckdb.connect(str(stats_path))
        conn.execute("""
            CREATE TABLE IF NOT EXISTS match_citations (
                match_id TEXT NOT NULL, citation_name_norm TEXT NOT NULL, value INTEGER NOT NULL,
                PRIMARY KEY (match_id, citation_name_norm)
            )
        """)

        from src.analysis.citations.engine import CitationEngine

        engine = CitationEngine(str(stats_path), "xuid_ghost", conn=conn)

        assert engine._pve_db_path is None
        assert engine.load_match_pve_stats("any-match") == {}

        conn.close()


# =============================================================================
# Tests Scoreboard
# =============================================================================


class TestScoreboardIntegration:
    """Tests d'intégration pour load_match_scoreboard (v5.2)."""

    @pytest.fixture
    def scoreboard_env(self, tmp_path: Path) -> tuple[Path, Path, Path]:
        """Fixture retournant (stats_path, shared_path, meta_path)."""
        return _create_shared_for_scoreboard(tmp_path)

    def _make_repo(self, stats_path: Path, shared_path: Path, meta_path: Path):
        """Crée un DuckDBRepository attaché à shared_matches."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        return DuckDBRepository(
            str(stats_path),
            xuid="xuid_a",
            metadata_db_path=meta_path,
            shared_db_path=str(shared_path),
        )

    def test_champs_attendus_presents(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """Chaque entrée du scoreboard contient les 20 champs documentés."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")

        expected_keys = {
            "xuid", "gamertag", "team_id", "rank", "score",
            "kills", "deaths", "assists", "kda", "max_killing_spree",
            "headshot_kills", "shots_fired", "shots_hit", "accuracy",
            "melee_kills", "power_weapon_kills", "damage_dealt",
            "damage_taken", "avg_life_seconds", "perfect_kills",
        }
        assert len(result) == 5
        for entry in result:
            assert expected_keys == set(entry.keys()), f"Champs manquants pour {entry['xuid']}"

        repo.close()

    def test_tri_par_equipe_puis_rang(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """Les joueurs sont triés par (team_id ASC NULLS LAST, rank ASC NULLS LAST)."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")
        team_ids = [r["team_id"] for r in result]

        # team 0 en premier, team 1 ensuite, None en dernier
        assert team_ids[:2] == [0, 0]
        assert team_ids[2:4] == [1, 1]
        assert team_ids[4] is None

        # Dans chaque équipe, trié par rang
        assert result[0]["rank"] <= result[1]["rank"]  # équipe 0
        assert result[2]["rank"] <= result[3]["rank"]  # équipe 1

        repo.close()

    def test_gamertag_resolu_depuis_xuid_aliases(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """xuid_d n'a pas de gamertag dans match_participants → résolu via xuid_aliases."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")
        entry_d = next(r for r in result if r["xuid"] == "xuid_d")

        assert entry_d["gamertag"] == "Delta"

        repo.close()

    def test_perfect_kills_aggreages_correctement(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """xuid_a a 2 Perfect Kills. Les autres médailles ne doivent pas être comptées."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")
        entry_a = next(r for r in result if r["xuid"] == "xuid_a")
        entry_b = next(r for r in result if r["xuid"] == "xuid_b")

        assert entry_a["perfect_kills"] == 2
        assert entry_b["perfect_kills"] == 0

        repo.close()

    def test_match_inexistant_retourne_liste_vide(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """Un match_id inconnu retourne une liste vide."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-inexistant-xyz")
        assert result == []

        result_empty = repo.load_match_scoreboard("")
        assert result_empty == []

        repo.close()

    def test_team_id_null_classe_en_dernier(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """Un joueur sans team_id (NULL) est bien classé en fin de liste."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")
        last = result[-1]

        assert last["team_id"] is None
        assert last["xuid"] == "xuid_e"

        repo.close()

    def test_score_et_stats_numeriques_corrects(self, scoreboard_env: tuple[Path, Path, Path]) -> None:
        """Les champs numériques sont bien castés en int/float (pas de strings)."""
        stats_path, shared_path, meta_path = scoreboard_env
        repo = self._make_repo(stats_path, shared_path, meta_path)

        result = repo.load_match_scoreboard("match-sc01")
        entry_a = next(r for r in result if r["xuid"] == "xuid_a")

        assert entry_a["score"] == 1200
        assert entry_a["kills"] == 12
        assert entry_a["deaths"] == 4
        assert entry_a["kda"] == pytest.approx(2.5)
        assert entry_a["accuracy"] == pytest.approx(75.0)

        repo.close()
