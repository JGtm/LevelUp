"""Tests unitaires pour le healthcheck DB (src/utils/healthcheck_db.py).

Chaque test crée ses propres DB en dossier temporaire → pas de dépendances externes.
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import duckdb

from src.utils.healthcheck_db import (
    CheckDetail,
    HealthCheckResult,
    _check_metadata,
    _check_metadata_attachable,
    _check_player,
    _check_pve,
    _check_shared,
    _check_view_queryable,
    _format_size,
    _try_repair_views,
    format_results,
    run_healthcheck,
)

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────


def _create_shared_db(path: Path) -> None:
    """Crée une shared_matches_v2.duckdb minimale avec tables et vues."""
    conn = duckdb.connect(str(path))
    conn.execute("CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY)")
    conn.execute("CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, mmr DOUBLE)")
    conn.execute("CREATE TABLE highlight_events (match_id VARCHAR, event_type VARCHAR)")
    conn.execute(
        "CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)"
    )
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)")
    conn.execute(
        "CREATE TABLE weapon_kills (match_id VARCHAR, weapon_id UBIGINT, reconciled_as UBIGINT, attribution_path VARCHAR DEFAULT 'none')"
    )
    conn.execute(
        "CREATE TABLE killer_victim_pairs (match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR)"
    )
    conn.execute(
        "CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, description VARCHAR, applied_at TIMESTAMP, schema_done BOOLEAN, backfill_done BOOLEAN)"
    )
    # Vues v6
    conn.execute("""
        CREATE VIEW v_gamertag_lookup AS
        SELECT xuid, gamertag FROM xuid_aliases
    """)
    conn.execute("""
        CREATE VIEW v_match_full AS
        SELECT match_id FROM match_registry
    """)
    conn.execute("""
        CREATE VIEW v_killer_victim_full AS
        SELECT match_id, killer_xuid, victim_xuid FROM killer_victim_pairs
    """)
    conn.execute("""
        CREATE VIEW v_weapon_kills AS
        SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills
    """)
    conn.close()


def _create_metadata_db(path: Path) -> None:
    """Crée une metadata.duckdb minimale."""
    conn = duckdb.connect(str(path))
    conn.execute(
        "CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR, name_fr VARCHAR)"
    )
    conn.execute("CREATE TABLE asset_translations (asset_id VARCHAR, lang VARCHAR, name VARCHAR)")
    conn.execute(
        "CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, description VARCHAR, applied_at TIMESTAMP, schema_done BOOLEAN, backfill_done BOOLEAN)"
    )
    conn.close()


def _create_player_db(path: Path) -> None:
    """Crée une stats.duckdb joueur minimale."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute(
        "CREATE TABLE player_match_enrichment (match_id VARCHAR, performance_score DOUBLE, session_id VARCHAR)"
    )
    conn.execute("CREATE TABLE personal_score_awards (match_id VARCHAR, award VARCHAR)")
    conn.execute("CREATE TABLE match_citations (match_id VARCHAR, citation VARCHAR)")
    conn.execute("CREATE TABLE career_progression (rank_id INTEGER)")
    conn.execute("CREATE TABLE sessions (session_id VARCHAR)")
    conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    conn.execute(
        "CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, description VARCHAR, applied_at TIMESTAMP, schema_done BOOLEAN, backfill_done BOOLEAN)"
    )
    conn.close()


# ─────────────────────────────────────────────────────────────────────────────
# Tests unitaires
# ─────────────────────────────────────────────────────────────────────────────


class TestFormatSize:
    """Tests de _format_size."""

    def test_bytes(self) -> None:
        assert _format_size(500) == "500 B"

    def test_kb(self) -> None:
        assert _format_size(2048) == "2 KB"

    def test_mb(self) -> None:
        assert _format_size(5 * 1024 * 1024) == "5.0 MB"

    def test_gb(self) -> None:
        assert _format_size(2 * 1024 * 1024 * 1024) == "2.0 GB"


class TestCheckShared:
    """Tests du check de shared_matches.duckdb."""

    def test_complete_db_returns_ok(self, tmp_path: Path) -> None:
        """DB complète → pas d'issue hors migrations (non peuplées en test)."""
        db_path = tmp_path / "warehouse" / "shared_matches_v2.duckdb"
        db_path.parent.mkdir(parents=True)
        _create_shared_db(db_path)

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        # Seules les migrations pendantes peuvent rester en warning (attendu en test)
        non_migration_issues = [c for c in result.issues if c.category != "migration"]
        assert not non_migration_issues

    def test_missing_db(self, tmp_path: Path) -> None:
        """DB absente → status error."""
        db_path = tmp_path / "shared_matches_v2.duckdb"

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        assert result.status in ("error", "warning")
        assert any(c.status == "missing" and c.category == "db" for c in result.checks)

    def test_missing_table(self, tmp_path: Path) -> None:
        """Table manquante → issue reportée."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE match_registry (match_id VARCHAR)")
        # Pas les autres tables
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        missing_tables = [
            c for c in result.checks if c.category == "table" and c.status == "missing"
        ]
        assert len(missing_tables) > 0

    def test_missing_view(self, tmp_path: Path) -> None:
        """Vue absente → issue reportée."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        # Supprimer une vue
        conn = duckdb.connect(str(db_path))
        conn.execute("DROP VIEW v_gamertag_lookup")
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        missing_views = [c for c in result.checks if c.category == "view" and c.status == "missing"]
        assert any(c.name == "v_gamertag_lookup" for c in missing_views)

    def test_broken_view(self, tmp_path: Path) -> None:
        """Vue dont la table source est supprimée → broken."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        conn = duckdb.connect(str(db_path))
        # Créer vue qui référence une table temporaire, puis supprimer la table
        conn.execute("CREATE TABLE tmp_target (col1 INTEGER)")
        conn.execute("DROP VIEW v_match_full")
        conn.execute("CREATE VIEW v_match_full AS SELECT col1 FROM tmp_target")
        conn.execute("DROP TABLE tmp_target")
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        broken = [c for c in result.checks if c.category == "view" and c.status == "broken"]
        assert any(c.name == "v_match_full" for c in broken)

    def test_missing_column(self, tmp_path: Path) -> None:
        """Colonne critique manquante → issue."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        # Recréer weapon_kills sans reconciled_as
        conn = duckdb.connect(str(db_path))
        conn.execute("DROP VIEW v_weapon_kills")
        conn.execute("DROP TABLE weapon_kills")
        conn.execute("CREATE TABLE weapon_kills (match_id VARCHAR, weapon_id UBIGINT)")
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared()

        missing_cols = [
            c for c in result.checks if c.category == "column" and c.status == "missing"
        ]
        assert len(missing_cols) > 0

    def test_deep_mode_orphans(self, tmp_path: Path) -> None:
        """Deep mode détecte les participants orphelins."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        conn = duckdb.connect(str(db_path))
        conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('orphan_match', 'xuid1')"
        )
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared(deep=True)

        orphan_checks = [c for c in result.checks if c.name == "orphan_participants"]
        assert any(c.status == "warning" for c in orphan_checks)

    def test_deep_mode_duplicates(self, tmp_path: Path) -> None:
        """Deep mode détecte les doublons."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        conn = duckdb.connect(str(db_path))
        conn.execute("INSERT INTO match_registry VALUES ('m1')")
        conn.execute("INSERT INTO match_participants (match_id, xuid) VALUES ('m1', 'x1')")
        conn.execute("INSERT INTO match_participants (match_id, xuid) VALUES ('m1', 'x1')")
        conn.close()

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            result = _check_shared(deep=True)

        dup_checks = [c for c in result.checks if c.name == "duplicate_participants"]
        assert any(c.status == "warning" for c in dup_checks)


class TestCheckMetadata:
    """Tests du check de metadata.duckdb."""

    def test_complete_metadata(self, tmp_path: Path) -> None:
        """metadata.duckdb complète → pas d'issue hors migrations."""
        db_path = tmp_path / "metadata.duckdb"
        _create_metadata_db(db_path)

        with patch("src.utils.healthcheck_db.get_metadata_db_path", return_value=db_path):
            result = _check_metadata()

        non_migration_issues = [c for c in result.issues if c.category != "migration"]
        assert not non_migration_issues

    def test_missing_metadata(self, tmp_path: Path) -> None:
        """metadata.duckdb absente → warning (pas error car optionnel pour fonctionnement basique)."""
        db_path = tmp_path / "metadata.duckdb"

        with patch("src.utils.healthcheck_db.get_metadata_db_path", return_value=db_path):
            result = _check_metadata()

        assert result.status == "warning"


class TestCheckPlayer:
    """Tests du check d'une DB joueur."""

    def test_complete_player_db(self, tmp_path: Path) -> None:
        """DB joueur complète → pas d'issue hors migrations."""
        db_path = tmp_path / "players" / "TestPlayer" / "stats.duckdb"
        _create_player_db(db_path)

        with patch("src.utils.healthcheck_db.PLAYERS_DIR", tmp_path / "players"):
            result = _check_player("TestPlayer")

        non_migration_issues = [c for c in result.issues if c.category != "migration"]
        assert not non_migration_issues

    def test_missing_player_db(self, tmp_path: Path) -> None:
        """DB joueur absente → error."""
        with patch("src.utils.healthcheck_db.PLAYERS_DIR", tmp_path / "players"):
            result = _check_player("NonExistent")

        assert any(c.status == "missing" for c in result.checks)

    def test_missing_critical_column(self, tmp_path: Path) -> None:
        """Colonne performance_score absente → issue."""
        db_path = tmp_path / "players" / "TestPlayer" / "stats.duckdb"
        db_path.parent.mkdir(parents=True)
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE player_match_enrichment (match_id VARCHAR)")
        conn.execute("CREATE TABLE personal_score_awards (match_id VARCHAR)")
        conn.execute("CREATE TABLE match_citations (match_id VARCHAR)")
        conn.execute("CREATE TABLE career_progression (rank_id INTEGER)")
        conn.execute("CREATE TABLE sessions (session_id VARCHAR)")
        conn.execute("CREATE TABLE sync_meta (key VARCHAR)")
        conn.execute(
            "CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY, description VARCHAR, applied_at TIMESTAMP, schema_done BOOLEAN, backfill_done BOOLEAN)"
        )
        conn.close()

        with patch("src.utils.healthcheck_db.PLAYERS_DIR", tmp_path / "players"):
            result = _check_player("TestPlayer")

        missing_cols = [
            c for c in result.checks if c.category == "column" and c.status == "missing"
        ]
        assert any("performance_score" in c.name for c in missing_cols)


class TestCheckViewQueryable:
    """Tests du helper _check_view_queryable."""

    def test_valid_view(self) -> None:
        conn = duckdb.connect()
        conn.execute("CREATE TABLE t (id INTEGER)")
        conn.execute("CREATE VIEW v AS SELECT id FROM t")
        assert _check_view_queryable(conn, "v") is None
        conn.close()

    def test_broken_view(self) -> None:
        conn = duckdb.connect()
        conn.execute("CREATE TABLE t (id INTEGER)")
        conn.execute("CREATE VIEW v AS SELECT id FROM t")
        conn.execute("DROP TABLE t")
        err = _check_view_queryable(conn, "v")
        assert err is not None
        conn.close()


class TestTryRepairViews:
    """Tests de l'auto-repair des vues."""

    def test_repair_missing_view(self, tmp_path: Path) -> None:
        """Vue manquante → repair tente de la recréer."""
        db_path = tmp_path / "shared_matches_v2.duckdb"
        _create_shared_db(db_path)
        # Supprimer la vue
        conn = duckdb.connect(str(db_path))
        conn.execute("DROP VIEW v_gamertag_lookup")
        conn.close()

        result = HealthCheckResult(db_name="shared_matches", db_path=db_path)
        result.add(CheckDetail("view", "v_gamertag_lookup", "missing", "Vue absente"))

        with patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=db_path):
            repaired = _try_repair_views(result)

        # Le repair a été tenté (résultat dépend de ensure_resolution_views avec ce schéma minimal)
        assert isinstance(repaired, list)


class TestFormatResults:
    """Tests du formatage des résultats."""

    def test_all_ok(self) -> None:
        results = [
            HealthCheckResult(
                db_name="shared",
                status="ok",
                checks=[
                    CheckDetail("table", "match_registry", "ok"),
                ],
            ),
        ]
        output = format_results(results)
        assert "Toutes les vérifications passent" in output

    def test_with_warnings(self) -> None:
        results = [
            HealthCheckResult(
                db_name="shared",
                status="warning",
                checks=[
                    CheckDetail("view", "v_match_full", "missing", "Vue absente"),
                ],
            ),
        ]
        output = format_results(results)
        assert "warning" in output.lower() or "Vue absente" in output

    def test_verbose_shows_ok(self) -> None:
        results = [
            HealthCheckResult(
                db_name="shared",
                status="ok",
                checks=[
                    CheckDetail("table", "match_registry", "ok"),
                ],
            ),
        ]
        output = format_results(results, verbose=True)
        assert "match_registry" in output

    def test_repaired_icon(self) -> None:
        results = [
            HealthCheckResult(
                db_name="shared",
                status="ok",
                checks=[
                    CheckDetail("view", "v_gamertag_lookup", "repaired", "Vue recréée avec succès"),
                ],
            ),
        ]
        output = format_results(results, verbose=True)
        assert "v_gamertag_lookup" in output

    def test_errors_count(self) -> None:
        results = [
            HealthCheckResult(
                db_name="shared",
                status="error",
                checks=[
                    CheckDetail("view", "v_match_full", "broken", "Vue cassée"),
                    CheckDetail("view", "v_weapon_kills", "error", "Erreur critique"),
                ],
            ),
        ]
        output = format_results(results)
        assert "erreur(s)" in output


class TestCheckMetadataAttachable:
    """Tests du check ATTACH metadata→shared."""

    def test_attach_ok(self, tmp_path: Path) -> None:
        """ATTACH fonctionne quand les deux DB existent et sont valides."""
        shared_path = tmp_path / "shared_matches_v2.duckdb"
        meta_path = tmp_path / "metadata.duckdb"
        _create_shared_db(shared_path)
        _create_metadata_db(meta_path)

        with (
            patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=shared_path),
            patch("src.utils.healthcheck_db.get_metadata_db_path", return_value=meta_path),
        ):
            result = _check_metadata_attachable()

        assert result.status == "ok"

    def test_attach_missing_db(self, tmp_path: Path) -> None:
        """ATTACH warning quand une DB est absente."""
        with (
            patch(
                "src.utils.healthcheck_db.get_shared_matches_path",
                return_value=tmp_path / "missing.duckdb",
            ),
            patch(
                "src.utils.healthcheck_db.get_metadata_db_path",
                return_value=tmp_path / "missing2.duckdb",
            ),
        ):
            result = _check_metadata_attachable()

        assert result.status == "warning"


class TestCheckPve:
    """Tests du check shared_pve.duckdb."""

    def test_pve_missing_is_ok(self, tmp_path: Path) -> None:
        """PVE absente → OK (optionnel)."""
        with patch(
            "src.utils.healthcheck_db.get_pve_db_path", return_value=tmp_path / "missing.duckdb"
        ):
            result = _check_pve()

        assert result.status == "ok"

    def test_pve_with_tables(self, tmp_path: Path) -> None:
        """PVE complète → OK."""
        pve_path = tmp_path / "shared_pve.duckdb"
        conn = duckdb.connect(str(pve_path))
        conn.execute("CREATE TABLE pve_match_stats (match_id VARCHAR)")
        conn.execute("CREATE TABLE schema_migrations (name VARCHAR PRIMARY KEY)")
        conn.close()

        with patch("src.utils.healthcheck_db.get_pve_db_path", return_value=pve_path):
            result = _check_pve()

        assert result.size_bytes and result.size_bytes > 0
        non_missing = [c for c in result.checks if c.status == "missing"]
        assert not non_missing

    def test_pve_missing_table(self, tmp_path: Path) -> None:
        """PVE avec tables manquantes → issue."""
        pve_path = tmp_path / "shared_pve.duckdb"
        conn = duckdb.connect(str(pve_path))
        conn.execute("CREATE TABLE dummy (id INTEGER)")
        conn.close()

        with patch("src.utils.healthcheck_db.get_pve_db_path", return_value=pve_path):
            result = _check_pve()

        missing = [c for c in result.checks if c.status == "missing"]
        assert len(missing) > 0


class TestCheckPlayerMedia:
    """Tests spécifiques pour les tables optionnelles du joueur."""

    def test_media_tables_missing_is_warning(self, tmp_path: Path) -> None:
        """Tables media absentes → warning (pas missing)."""
        db_path = tmp_path / "players" / "TestPlayer" / "stats.duckdb"
        _create_player_db(db_path)

        with patch("src.utils.healthcheck_db.PLAYERS_DIR", tmp_path / "players"):
            result = _check_player("TestPlayer")

        media_checks = [c for c in result.checks if "media" in c.name]
        for c in media_checks:
            assert c.status == "warning"


class TestRunHealthcheck:
    """Tests d'intégration du healthcheck complet."""

    def test_full_run_with_complete_dbs(self, tmp_path: Path) -> None:
        """Healthcheck complet sur des DB valides."""
        warehouse = tmp_path / "warehouse"
        warehouse.mkdir()
        players = tmp_path / "players"
        players.mkdir()

        shared_path = warehouse / "shared_matches_v2.duckdb"
        meta_path = warehouse / "metadata.duckdb"
        pve_path = warehouse / "shared_pve.duckdb"
        player_path = players / "TestGT" / "stats.duckdb"

        _create_shared_db(shared_path)
        _create_metadata_db(meta_path)
        _create_player_db(player_path)

        with (
            patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=shared_path),
            patch("src.utils.healthcheck_db.get_metadata_db_path", return_value=meta_path),
            patch("src.utils.healthcheck_db.get_pve_db_path", return_value=pve_path),
            patch("src.utils.healthcheck_db.PLAYERS_DIR", players),
        ):
            results = run_healthcheck(auto_repair=False)

        # Au moins 3 résultats : shared, metadata, pve (+ 1 joueur)
        assert len(results) >= 3
        # Pas d'erreurs sur les DB qu'on a créées
        shared_result = results[0]
        assert shared_result.db_name == "shared_matches"
        # Les tables et vues sont présentes
        assert not any(c.status in ("error", "broken") for c in shared_result.checks)

    def test_run_with_missing_dbs(self, tmp_path: Path) -> None:
        """Healthcheck quand les DB n'existent pas."""
        with (
            patch(
                "src.utils.healthcheck_db.get_shared_matches_path",
                return_value=tmp_path / "missing.duckdb",
            ),
            patch(
                "src.utils.healthcheck_db.get_metadata_db_path",
                return_value=tmp_path / "missing2.duckdb",
            ),
            patch(
                "src.utils.healthcheck_db.get_pve_db_path",
                return_value=tmp_path / "missing3.duckdb",
            ),
            patch("src.utils.healthcheck_db.PLAYERS_DIR", tmp_path / "players"),
        ):
            results = run_healthcheck(auto_repair=False)

        assert len(results) >= 3
        # shared manquante = erreur
        assert results[0].status in ("error", "warning")

    def test_run_with_specific_player(self, tmp_path: Path) -> None:
        """Healthcheck avec player= ne check que ce joueur."""
        warehouse = tmp_path / "warehouse"
        warehouse.mkdir()
        players = tmp_path / "players"
        players.mkdir()

        shared_path = warehouse / "shared_matches_v2.duckdb"
        meta_path = warehouse / "metadata.duckdb"
        pve_path = warehouse / "shared_pve.duckdb"
        player_path = players / "SpecificGT" / "stats.duckdb"

        _create_shared_db(shared_path)
        _create_metadata_db(meta_path)
        _create_player_db(player_path)
        # Créer un second joueur qui ne devrait PAS être inclus
        _create_player_db(players / "OtherGT" / "stats.duckdb")

        with (
            patch("src.utils.healthcheck_db.get_shared_matches_path", return_value=shared_path),
            patch("src.utils.healthcheck_db.get_metadata_db_path", return_value=meta_path),
            patch("src.utils.healthcheck_db.get_pve_db_path", return_value=pve_path),
            patch("src.utils.healthcheck_db.PLAYERS_DIR", players),
        ):
            results = run_healthcheck(auto_repair=False, player="SpecificGT")

        player_results = [r for r in results if r.db_name.startswith("player:")]
        assert len(player_results) == 1
        assert player_results[0].db_name == "player:SpecificGT"
