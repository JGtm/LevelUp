"""Tests pour le script de migration vers IDs techniques."""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

from scripts.migration.migrate_to_technical_ids import (
    AWARD_NAME_MAP,
    CITATION_NORM_MAP,
    migrate_award_names,
    migrate_citation_norms,
    migrate_composite_children,
    table_exists,
)


@pytest.fixture
def metadata_conn(tmp_path: Path) -> duckdb.DuckDBPyConnection:
    """Crée une metadata DB avec citation_mappings legacy."""
    conn = duckdb.connect(str(tmp_path / "metadata.duckdb"))
    conn.execute("""
        CREATE TABLE citation_mappings (
            citation_name_norm TEXT PRIMARY KEY,
            citation_name_display TEXT NOT NULL,
            mapping_type TEXT NOT NULL,
            award_name TEXT,
            composite_children TEXT
        )
    """)
    conn.executemany(
        "INSERT INTO citation_mappings VALUES (?, ?, ?, ?, ?)",
        [
            ("pilote", "Pilote", "medal", None, None),
            ("defenseur du drapeau", "Défenseur du drapeau", "award", "Zone capturée", None),
            (
                "destructeur de covenants",
                "Destructeur de Covenants",
                "composite",
                None,
                '["tueur de grognards", "tueur d\'elites", "tueur de rapaces"]',
            ),
            ("bulldozer", "Bulldozer", "custom", None, None),
        ],
    )
    yield conn
    conn.close()


@pytest.fixture
def player_conn(tmp_path: Path) -> duckdb.DuckDBPyConnection:
    """Crée une player DB avec match_citations et personal_score_awards legacy."""
    conn = duckdb.connect(str(tmp_path / "stats.duckdb"))
    conn.execute("""
        CREATE TABLE match_citations (
            match_id TEXT NOT NULL,
            citation_name_norm TEXT NOT NULL,
            value INTEGER NOT NULL,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)
    conn.execute("""
        CREATE TABLE personal_score_awards (
            match_id TEXT,
            award_name TEXT,
            award_count INTEGER,
            award_score INTEGER
        )
    """)
    conn.execute(
        "INSERT INTO match_citations VALUES "
        "('m1', 'pilote', 2), ('m1', 'defenseur du drapeau', 3), "
        "('m2', 'tueur de grognards', 5), ('m2', 'bulldozer', 1)"
    )
    conn.execute(
        "INSERT INTO personal_score_awards VALUES "
        "('m1', 'Zone capturée', 3, 150), "
        "('m1', 'DESTROYED_BANSHEE', 1, 100), "
        "('m2', 'Wraith Destroyed', 2, 200)"
    )
    yield conn
    conn.close()


class TestMigrationMappings:
    """Vérifie que les mappings couvrent les données connues."""

    def test_citation_map_covers_all_pvp(self):
        """Le mapping citation couvre tous les noms FR connus."""
        assert "pilote" in CITATION_NORM_MAP
        assert "defenseur du drapeau" in CITATION_NORM_MAP
        assert "carnage de spartans" in CITATION_NORM_MAP
        assert CITATION_NORM_MAP["pilote"] == "driver"
        assert CITATION_NORM_MAP["oeil de lynx"] == "eagle_eye"

    def test_citation_map_covers_pve(self):
        """Le mapping citation couvre les PVE."""
        assert "tueur de grognards" in CITATION_NORM_MAP
        assert CITATION_NORM_MAP["tueur de grognards"] == "grunt_slayer"
        assert CITATION_NORM_MAP["tueur de chasseurs"] == "hunter_slayer"

    def test_award_map_covers_vehicles(self):
        """Le mapping award couvre les véhicules destroy/hijack."""
        assert "DESTROYED_BANSHEE" in AWARD_NAME_MAP
        assert "HIJACKED_GHOST" in AWARD_NAME_MAP
        assert AWARD_NAME_MAP["DESTROYED_BANSHEE"] == "destroyed_banshee"

    def test_award_map_covers_objectives(self):
        """Le mapping award couvre les objectifs FR."""
        assert "Zone capturée" in AWARD_NAME_MAP
        assert AWARD_NAME_MAP["Zone capturée"] == "zone_captured"
        assert AWARD_NAME_MAP["Porteur arrêté"] == "runner_stopped"


class TestMigrateCitationNorms:
    """Tests de la migration des citation_name_norm."""

    def test_migrates_known_norms(self, metadata_conn):
        """Migre les noms FR connus vers les IDs techniques."""
        count = migrate_citation_norms(metadata_conn, "citation_mappings", dry_run=False)
        assert count >= 3  # pilote + defenseur du drapeau + destructeur de covenants

        # Vérifier les nouvelles valeurs
        row = metadata_conn.execute(
            "SELECT citation_name_norm FROM citation_mappings "
            "WHERE citation_name_display = 'Pilote'"
        ).fetchone()
        assert row[0] == "driver"

        row2 = metadata_conn.execute(
            "SELECT citation_name_norm FROM citation_mappings "
            "WHERE citation_name_display = 'Défenseur du drapeau'"
        ).fetchone()
        assert row2[0] == "flag_defender"

    def test_keeps_unknown_norms(self, metadata_conn):
        """Ne touche pas les norms déjà correctes ou inconnues."""
        migrate_citation_norms(metadata_conn, "citation_mappings", dry_run=False)
        # bulldozer n'est pas dans le mapping → inchangé
        row = metadata_conn.execute(
            "SELECT citation_name_norm FROM citation_mappings "
            "WHERE citation_name_display = 'Bulldozer'"
        ).fetchone()
        assert row[0] == "bulldozer"

    def test_dry_run_does_not_modify(self, metadata_conn):
        """En dry-run, aucune donnée n'est modifiée."""
        migrate_citation_norms(metadata_conn, "citation_mappings", dry_run=True)

        row = metadata_conn.execute(
            "SELECT citation_name_norm FROM citation_mappings "
            "WHERE citation_name_display = 'Pilote'"
        ).fetchone()
        assert row[0] == "pilote"  # Pas modifié


class TestMigrateAwardNames:
    """Tests de la migration des award_name."""

    def test_migrates_french_awards(self, player_conn):
        """Migre les award_name FR vers les IDs techniques."""
        count = migrate_award_names(player_conn, "personal_score_awards", dry_run=False)
        assert count >= 2  # Zone capturée + DESTROYED_BANSHEE

        rows = player_conn.execute(
            "SELECT award_name FROM personal_score_awards ORDER BY award_name"
        ).fetchall()
        names = [r[0] for r in rows]
        assert "destroyed_banshee" in names
        assert "zone_captured" in names
        assert "destroyed_wraith" in names

    def test_dry_run_does_not_modify(self, player_conn):
        """En dry-run, aucune donnée n'est modifiée."""
        migrate_award_names(player_conn, "personal_score_awards", dry_run=True)

        rows = player_conn.execute(
            "SELECT award_name FROM personal_score_awards WHERE award_name = 'Zone capturée'"
        ).fetchall()
        assert len(rows) == 1  # Pas modifié


class TestMigrateCompositeChildren:
    """Tests de la migration composite_children."""

    def test_migrates_children_list(self, metadata_conn):
        """Migre les enfants composites vers les nouveaux noms."""
        # D'abord migrer les norms (pour que le PK soit correct)
        migrate_citation_norms(metadata_conn, "citation_mappings", dry_run=False)
        count = migrate_composite_children(metadata_conn, dry_run=False)
        assert count == 1

        row = metadata_conn.execute(
            "SELECT composite_children FROM citation_mappings "
            "WHERE citation_name_norm = 'covenant_destroyer'"
        ).fetchone()
        # Le résultat est un JSON array avec les IDs techniques
        import json

        children = json.loads(row[0])
        assert children == ["grunt_slayer", "elite_slayer", "jackal_slayer"]


class TestMigratePlayerCitations:
    """Tests intégrés sur la player DB."""

    def test_migrates_match_citations(self, player_conn):
        """Migre les citation_name_norm dans match_citations."""
        count = migrate_citation_norms(player_conn, "match_citations", dry_run=False)
        assert count >= 3  # pilote + defenseur du drapeau + tueur de grognards

        rows = player_conn.execute(
            "SELECT citation_name_norm FROM match_citations ORDER BY citation_name_norm"
        ).fetchall()
        norms = [r[0] for r in rows]
        assert "driver" in norms
        assert "flag_defender" in norms
        assert "grunt_slayer" in norms
        assert "bulldozer" in norms  # Inchangé (pas dans le mapping)

    def test_table_exists_check(self, player_conn):
        """table_exists retourne True/False correctement."""
        assert table_exists(player_conn, "match_citations") is True
        assert table_exists(player_conn, "nonexistent_table") is False
