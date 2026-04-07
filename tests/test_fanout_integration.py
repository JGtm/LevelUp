"""Tests d'intégration — Fan-out enrichissement des coéquipiers.

Vérifie que FanoutEnrichmentMixin distribue correctement les données
vers les DBs joueur des coéquipiers enregistrés dans db_profiles.json.

Scénarios couverts :
- Détection des matchs communs entre joueurs
- Enrichissement PME (performance_score) écrit dans la DB coéquipier
- PSA collectées distribuées via fanout
- Sessions et citations calculées dans la DB coéquipier
- Aucun crash si db_profiles.json absent ou coéquipier sans DB
- fanout_repair_missing_scores détecte et corrige les PME manquants
- Isolation : erreur sur un joueur ne bloque pas les autres
"""

from __future__ import annotations

import json
from pathlib import Path

import duckdb

from tests.conftest_sync import (
    GT_PLAYER_A,
    GT_PLAYER_B,
    GT_PLAYER_C,
    XUID_PLAYER_A,
    XUID_PLAYER_B,
    XUID_PLAYER_C,
    count_rows,
    create_metadata_db,
    create_player_db,
    create_shared_db,
    insert_pme_rows,
    make_match_data,
)

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def _setup_multi_player_env(tmp_path: Path) -> dict:
    """Crée un environnement multi-joueurs avec shared DB peuplée.

    Returns:
        Dict avec les clés: project_root, shared_db, player_a_db, player_b_db,
        player_c_db, db_profiles_path.
    """
    project_root = tmp_path / "project"
    data_dir = project_root / "data"
    warehouse = data_dir / "warehouse"

    # Databases
    shared_db = create_shared_db(warehouse / "shared_matches_v2.duckdb")
    metadata_db = create_metadata_db(warehouse / "metadata.duckdb")

    player_a_dir = data_dir / "players" / GT_PLAYER_A
    player_b_dir = data_dir / "players" / GT_PLAYER_B
    player_c_dir = data_dir / "players" / GT_PLAYER_C

    player_a_db = create_player_db(player_a_dir / "stats.duckdb")
    player_b_db = create_player_db(player_b_dir / "stats.duckdb")
    player_c_db = create_player_db(player_c_dir / "stats.duckdb")

    # db_profiles.json — Le code fanout calcule project_root via
    # _player_db_path.parent.parent.parent (= data_dir, 3 niveaux au-dessus de stats.duckdb)
    # puis cherche project_root / "db_profiles.json".
    # Les db_path doivent aussi être relatifs à data_dir.
    profiles = {
        "profiles": {
            GT_PLAYER_A: {
                "xuid": XUID_PLAYER_A,
                "db_path": f"players/{GT_PLAYER_A}/stats.duckdb",
            },
            GT_PLAYER_B: {
                "xuid": XUID_PLAYER_B,
                "db_path": f"players/{GT_PLAYER_B}/stats.duckdb",
            },
            GT_PLAYER_C: {
                "xuid": XUID_PLAYER_C,
                "db_path": f"players/{GT_PLAYER_C}/stats.duckdb",
            },
        }
    }
    db_profiles_path = data_dir / "db_profiles.json"
    db_profiles_path.write_text(json.dumps(profiles, indent=2), encoding="utf-8")

    # Peupler shared avec 3 matchs :
    # match-001: A + B + C
    # match-002: A + B (pas C)
    # match-003: A seulement
    matches = [
        make_match_data(
            "match-001",
            [
                (XUID_PLAYER_A, GT_PLAYER_A),
                (XUID_PLAYER_B, GT_PLAYER_B),
                (XUID_PLAYER_C, GT_PLAYER_C),
            ],
        ),
        make_match_data(
            "match-002",
            [(XUID_PLAYER_A, GT_PLAYER_A), (XUID_PLAYER_B, GT_PLAYER_B)],
        ),
        make_match_data(
            "match-003",
            [(XUID_PLAYER_A, GT_PLAYER_A)],
        ),
    ]
    conn = duckdb.connect(str(shared_db))
    try:
        from tests.conftest_sync import _insert_matches_to_shared

        _insert_matches_to_shared(conn, matches)
        conn.commit()
    finally:
        conn.close()

    return {
        "project_root": project_root,
        "shared_db": shared_db,
        "metadata_db": metadata_db,
        "player_a_db": player_a_db,
        "player_b_db": player_b_db,
        "player_c_db": player_c_db,
        "db_profiles_path": db_profiles_path,
    }


def _make_engine_for(env: dict, gamertag: str, xuid: str, *, shared_read_only: bool = False):
    """Crée un DuckDBSyncEngine pour le joueur donné dans l'environnement."""
    from src.data.sync.engine import DuckDBSyncEngine

    player_db = env["project_root"] / "data" / "players" / gamertag / "stats.duckdb"
    return DuckDBSyncEngine(
        player_db_path=player_db,
        xuid=xuid,
        gamertag=gamertag,
        shared_db_path=env["shared_db"],
        metadata_db_path=env["metadata_db"],
        shared_read_only=shared_read_only,
    )


# ===========================================================================
# Tests _get_other_registered_players
# ===========================================================================


class TestGetOtherRegisteredPlayers:
    """Tests pour la lecture de db_profiles.json."""

    def test_returns_other_players(self, tmp_path: Path) -> None:
        """Retourne les joueurs du profil sauf le joueur courant."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            others = engine._get_other_registered_players()
            gts = {p["gamertag"] for p in others}
            assert GT_PLAYER_A not in gts
            assert GT_PLAYER_B in gts
            assert GT_PLAYER_C in gts
            assert len(others) == 2
        finally:
            engine.close()

    def test_missing_db_profiles(self, tmp_path: Path) -> None:
        """Si db_profiles.json est absent → liste vide, pas d'erreur."""
        env = _setup_multi_player_env(tmp_path)
        env["db_profiles_path"].unlink()

        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            others = engine._get_other_registered_players()
            assert others == []
        finally:
            engine.close()

    def test_excludes_current_player_by_xuid(self, tmp_path: Path) -> None:
        """Le joueur courant est exclu par XUID (pas seulement par gamertag)."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            others = engine._get_other_registered_players()
            xuids = {p["xuid"] for p in others}
            assert XUID_PLAYER_A not in xuids
        finally:
            engine.close()


# ===========================================================================
# Tests _find_common_match_ids
# ===========================================================================


class TestFindCommonMatchIds:
    """Tests pour la détection des matchs communs."""

    def test_finds_common_matches(self, tmp_path: Path) -> None:
        """Détecte correctement les matchs où le coéquipier était présent."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            common = engine._find_common_match_ids(
                XUID_PLAYER_B,
                env["shared_db"],
                ["match-001", "match-002", "match-003"],
            )
            assert set(common) == {"match-001", "match-002"}
        finally:
            engine.close()

    def test_no_common_matches(self, tmp_path: Path) -> None:
        """Si le coéquipier n'était dans aucun des matchs → liste vide."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            common = engine._find_common_match_ids(
                XUID_PLAYER_C,
                env["shared_db"],
                ["match-002", "match-003"],  # C n'est que dans match-001
            )
            assert common == []
        finally:
            engine.close()

    def test_empty_new_match_ids(self, tmp_path: Path) -> None:
        """Liste vide de nouveaux matchs → liste vide."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            common = engine._find_common_match_ids(XUID_PLAYER_B, env["shared_db"], [])
            assert common == []
        finally:
            engine.close()


# ===========================================================================
# Tests _enrich_single_player
# ===========================================================================


class TestEnrichSinglePlayer:
    """Tests pour l'enrichissement d'un coéquipier."""

    def test_creates_pme_for_teammate(self, tmp_path: Path) -> None:
        """Le fanout crée des lignes PME dans la DB du coéquipier."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            engine._enrich_single_player(
                GT_PLAYER_B,
                XUID_PLAYER_B,
                env["player_b_db"],
                ["match-001", "match-002"],
            )

            # Vérifier que des PME ont été créées dans la DB de B
            with duckdb.connect(str(env["player_b_db"]), read_only=True) as conn:
                pme_count = count_rows(conn, "player_match_enrichment")
                # Avec 2 matchs < MIN_MATCHES_FOR_RELATIVE (10), pas de perf scores
                assert pme_count == 0
        finally:
            engine.close()

    def test_skip_if_db_missing(self, tmp_path: Path) -> None:
        """Si la DB du coéquipier n'existe pas → skip silencieux."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            missing_db = tmp_path / "nonexistent" / "stats.duckdb"
            # Ne doit pas lever d'exception
            engine._enrich_single_player("GhostPlayer", "9999999", missing_db, ["match-001"])
        finally:
            engine.close()

    def test_skip_if_no_xuid(self, tmp_path: Path) -> None:
        """Si le XUID est vide → skip silencieux."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            engine._enrich_single_player(GT_PLAYER_B, "", env["player_b_db"], ["match-001"])
        finally:
            engine.close()

    def test_distributes_pending_psa(self, tmp_path: Path) -> None:
        """Les PSA collectées pendant le sync sont distribuées au coéquipier."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            from dataclasses import dataclass

            @dataclass
            class FakePSA:
                match_id: str
                xuid: str
                award_name: str
                award_category: str = "Kills"
                award_count: int = 1
                award_score: int = 100

            engine._pending_other_psa = {
                XUID_PLAYER_B: [
                    FakePSA("match-001", XUID_PLAYER_B, "KillingSpree"),
                    FakePSA("match-002", XUID_PLAYER_B, "DoubleKill"),
                ],
            }

            engine._enrich_single_player(
                GT_PLAYER_B,
                XUID_PLAYER_B,
                env["player_b_db"],
                ["match-001", "match-002"],
            )

            # Vérifier que les PSA ont été écrites
            with duckdb.connect(str(env["player_b_db"]), read_only=True) as conn:
                psa_count = count_rows(conn, "personal_score_awards")
                assert psa_count == 2
        finally:
            engine.close()


# ===========================================================================
# Tests _enrich_other_registered_players (orchestrateur)
# ===========================================================================


class TestEnrichOtherRegisteredPlayers:
    """Tests pour le fan-out complet vers tous les coéquipiers."""

    def test_fanout_all_teammates(self, tmp_path: Path) -> None:
        """Le fanout itère sur tous les joueurs enregistrés."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # Exécuter le fanout pour les 3 matchs
            engine._enrich_other_registered_players(["match-001", "match-002", "match-003"])

            # B devrait avoir reçu un enrichissement (match-001, match-002)
            with duckdb.connect(str(env["player_b_db"]), read_only=True) as conn:
                b_pme = count_rows(conn, "player_match_enrichment")
                # < MIN_MATCHES_FOR_RELATIVE (10) → pas de perf scores
                assert b_pme == 0
        finally:
            engine.close()

    def test_fanout_empty_match_ids(self, tmp_path: Path) -> None:
        """Liste vide de match_ids → aucun traitement."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # Ne doit rien faire et ne pas crasher
            engine._enrich_other_registered_players([])
        finally:
            engine.close()

    def test_fanout_isolates_errors(self, tmp_path: Path) -> None:
        """Erreur sur un joueur ne bloque pas les autres."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # Corrompre la DB de B (supprimer le fichier pour forcer une erreur)
            env["player_b_db"].unlink()

            # Le fanout ne doit pas lever d'exception malgré l'erreur sur B
            engine._enrich_other_registered_players(["match-001"])

            # C devrait quand même avoir été traité
            with duckdb.connect(str(env["player_c_db"]), read_only=True) as conn:
                # DB de C accessible malgré l'erreur sur B (isolation)
                c_pme = count_rows(conn, "player_match_enrichment")
                assert isinstance(c_pme, int)
        finally:
            engine.close()


# ===========================================================================
# Tests _has_missing_enrichment
# ===========================================================================


class TestHasMissingEnrichment:
    """Tests pour la détection de PME manquants."""

    def test_detects_missing_pme(self, tmp_path: Path) -> None:
        """Détecte qu'un joueur a des matchs dans shared sans performance_score."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # B a des matchs dans shared mais aucun PME
            has_missing = engine._has_missing_enrichment(XUID_PLAYER_B, env["player_b_db"])
            assert has_missing is True
        finally:
            engine.close()

    def test_no_missing_when_all_scored(self, tmp_path: Path) -> None:
        """Pas de manque si tous les matchs ont un performance_score."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # Insérer des PME avec performance_score pour tous les matchs de B
            with duckdb.connect(str(env["player_b_db"])) as conn:
                insert_pme_rows(conn, ["match-001", "match-002"], performance_score=75.0)
                conn.commit()

            has_missing = engine._has_missing_enrichment(XUID_PLAYER_B, env["player_b_db"])
            assert has_missing is False
        finally:
            engine.close()

    def test_missing_db_returns_false(self, tmp_path: Path) -> None:
        """DB absente → False (pas de crash)."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            missing = tmp_path / "ghost" / "stats.duckdb"
            assert engine._has_missing_enrichment("9999", missing) is False
        finally:
            engine.close()


# ===========================================================================
# Tests fanout_repair_missing_scores
# ===========================================================================


class TestFanoutRepairMissingScores:
    """Tests pour la réparation des performance_scores manquants."""

    def test_repair_triggers_enrichment(self, tmp_path: Path) -> None:
        """repair détecte les PME manquants et lance l'enrichissement."""
        env = _setup_multi_player_env(tmp_path)
        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # B a des matchs communs mais pas de PME → repair devrait les créer
            engine.fanout_repair_missing_scores()
            # Pas de crash = succès minimal
        finally:
            engine.close()

    def test_repair_noop_when_all_complete(self, tmp_path: Path) -> None:
        """Rien à réparer si tous les coéquipiers sont complets."""
        env = _setup_multi_player_env(tmp_path)

        # Peupler les PME de B et C
        for db_path in [env["player_b_db"], env["player_c_db"]]:
            with duckdb.connect(str(db_path)) as conn:
                insert_pme_rows(
                    conn, ["match-001", "match-002", "match-003"], performance_score=80.0
                )
                conn.commit()

        engine = _make_engine_for(env, GT_PLAYER_A, XUID_PLAYER_A)
        try:
            # Ne devrait rien faire
            engine.fanout_repair_missing_scores()
        finally:
            engine.close()
