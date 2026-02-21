"""Tests unitaires pour populate_playlist_translations et MetadataResolver.

Ce module teste :
- Création des tables playlist_translations et mode_translations
- UPSERT idempotent
- Nettoyage des entrées obsolètes
- Résolution MetadataResolver via playlist_translations
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import duckdb
import pytest

# Ajouter la racine du projet au path pour les imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from scripts.populate_playlist_translations import (
    cleanup_obsolete,
    ensure_schema,
    populate,
)
from src.data.sync.metadata_resolver import MetadataResolver

FAKE_JSON = {
    "playlists": [
        {
            "en": "Big Team Battle",
            "fr": "Grande bataille en équipe",
            "uuid": "2825d417-0000-0000-0000-000000000000",
        },
        {
            "en": "Quick Play",
            "fr": "Partie rapide",
            "uuid": "1b1691dc-0000-0000-0000-000000000000",
        },
    ],
    "modes": [
        {"en": "Arena:Slayer", "fr": "Arène : Assassin", "category": "Arena"},
        {"en": "BTB:CTF", "fr": "BTB : Capture du drapeau", "category": "BTB"},
        {"en": "Firefight:KOTH", "fr": "Baptême du feu : KOTH", "category": "Firefight"},
    ],
}


@pytest.fixture
def fake_json(tmp_path: Path) -> Path:
    """Écrit un JSON minimal dans un répertoire temporaire."""
    p = tmp_path / "Playlist_modes_translations.json"
    p.write_text(json.dumps(FAKE_JSON), encoding="utf-8")
    return p


@pytest.fixture
def empty_metadata_db(tmp_path: Path) -> Path:
    """Base metadata.duckdb vide (sans tables)."""
    db_path = tmp_path / "metadata.duckdb"
    duckdb.connect(str(db_path)).close()
    return db_path


class TestPopulatePlaylistTranslations:
    """Tests du script populate_playlist_translations."""

    def test_ensure_schema_idempotent(self, empty_metadata_db: Path) -> None:
        """ensure_schema() appelé deux fois ne lève pas d'erreur."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            ensure_schema(conn)
            tables = {
                row[0]
                for row in conn.execute(
                    "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
                ).fetchall()
            }
            assert "playlist_translations" in tables
            assert "mode_translations" in tables
        finally:
            conn.close()

    def test_populate_playlists_count(self, empty_metadata_db: Path) -> None:
        """populate() insère exactement N playlists issues du JSON."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            n_pl, _ = populate(conn, FAKE_JSON)
            assert n_pl == 2
            actual = conn.execute("SELECT COUNT(*) FROM playlist_translations").fetchone()[0]
            assert actual == 2
        finally:
            conn.close()

    def test_populate_modes_count(self, empty_metadata_db: Path) -> None:
        """populate() insère exactement N modes issus du JSON."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            _, n_modes = populate(conn, FAKE_JSON)
            assert n_modes == 3
            actual = conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0]
            assert actual == 3
        finally:
            conn.close()

    def test_populate_playlist_values(self, empty_metadata_db: Path) -> None:
        """Valeurs uuid, name_en, name_fr correctement insérées."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)
            row = conn.execute(
                "SELECT uuid, name_en, name_fr FROM playlist_translations "
                "WHERE uuid = '2825d417-0000-0000-0000-000000000000'"
            ).fetchone()
            assert row is not None
            assert row[0] == "2825d417-0000-0000-0000-000000000000"
            assert row[1] == "Big Team Battle"
            assert row[2] == "Grande bataille en équipe"
        finally:
            conn.close()

    def test_populate_mode_values(self, empty_metadata_db: Path) -> None:
        """Valeurs name_en, name_fr, category correctement insérées."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)
            row = conn.execute(
                "SELECT name_en, name_fr, category FROM mode_translations "
                "WHERE name_en = 'Arena:Slayer'"
            ).fetchone()
            assert row is not None
            assert row[0] == "Arena:Slayer"
            assert row[1] == "Arène : Assassin"
            assert row[2] == "Arena"
        finally:
            conn.close()

    def test_upsert_idempotent(self, empty_metadata_db: Path) -> None:
        """Double appel populate() : même nombre de lignes, pas de doublons."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)
            populate(conn, FAKE_JSON)
            n_pl = conn.execute("SELECT COUNT(*) FROM playlist_translations").fetchone()[0]
            n_modes = conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0]
            assert n_pl == 2
            assert n_modes == 3
        finally:
            conn.close()

    def test_cleanup_removes_obsolete_playlists(self, empty_metadata_db: Path) -> None:
        """cleanup_obsolete() supprime les playlists absentes du JSON."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)

            # Insérer une playlist qui n'existe pas dans le JSON
            conn.execute(
                "INSERT INTO playlist_translations VALUES (?, ?, ?)",
                ["ffffffff-0000-0000-0000-000000000000", "Obsolete", "Obsolète"],
            )
            assert conn.execute("SELECT COUNT(*) FROM playlist_translations").fetchone()[0] == 3

            n_pl_del, _ = cleanup_obsolete(conn, FAKE_JSON)
            assert n_pl_del == 1
            assert conn.execute("SELECT COUNT(*) FROM playlist_translations").fetchone()[0] == 2
        finally:
            conn.close()

    def test_cleanup_removes_obsolete_modes(self, empty_metadata_db: Path) -> None:
        """cleanup_obsolete() supprime les modes absents du JSON."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)

            # Insérer un mode qui n'existe pas dans le JSON
            conn.execute(
                "INSERT INTO mode_translations VALUES (?, ?, ?)",
                ["Obsolete:Mode", "Mode obsolète", "Other"],
            )
            assert conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0] == 4

            _, n_mode_del = cleanup_obsolete(conn, FAKE_JSON)
            assert n_mode_del == 1
            assert conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0] == 3
        finally:
            conn.close()

    def test_reset_drops_and_recreates(self, empty_metadata_db: Path) -> None:
        """Simuler --reset : DROP + ensure_schema + populate repart à zéro."""
        conn = duckdb.connect(str(empty_metadata_db))
        try:
            ensure_schema(conn)
            populate(conn, FAKE_JSON)

            # Simuler un --reset
            conn.execute("DROP TABLE IF EXISTS playlist_translations")
            conn.execute("DROP TABLE IF EXISTS mode_translations")

            ensure_schema(conn)
            n_pl, n_modes = populate(conn, FAKE_JSON)
            assert n_pl == 2
            assert n_modes == 3
        finally:
            conn.close()


class TestMetadataResolverWithPlaylistTranslations:
    """Tests d'intégration MetadataResolver + playlist_translations."""

    @pytest.fixture
    def db_with_playlist_translations(self, tmp_path: Path) -> Path:
        """Base avec table playlist_translations uniquement."""
        db_path = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute(
            "CREATE TABLE playlist_translations "
            "(uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)"
        )
        conn.execute(
            "INSERT INTO playlist_translations VALUES (?,?,?)",
            [
                "2825d417-0000-0000-0000-000000000000",
                "Big Team Battle",
                "Grande bataille en équipe",
            ],
        )
        conn.close()
        return db_path

    def test_resolve_playlist_by_uuid(self, db_with_playlist_translations: Path) -> None:
        """MetadataResolver résout un UUID playlist depuis playlist_translations."""
        resolver = MetadataResolver(db_with_playlist_translations)
        name = resolver.resolve("playlist", "2825d417-0000-0000-0000-000000000000")
        assert name == "Grande bataille en équipe"
        resolver.close()

    def test_resolve_unknown_uuid_returns_none(self, db_with_playlist_translations: Path) -> None:
        """UUID inconnu retourne None sans lever d'exception."""
        resolver = MetadataResolver(db_with_playlist_translations)
        assert resolver.resolve("playlist", "ffffffff-0000-0000-0000-000000000000") is None
        resolver.close()

    def test_playlists_table_has_priority(self, tmp_path: Path) -> None:
        """Si table playlists existe, elle est interrogée avant playlist_translations."""
        db_path = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db_path))
        # Créer playlists (discovery) avec valeur différente
        conn.execute(
            "CREATE TABLE playlists "
            "(asset_id VARCHAR PRIMARY KEY, version_id VARCHAR, public_name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO playlists VALUES (?,?,?)",
            ["2825d417-0000-0000-0000-000000000000", "1", "BTB (Discovery)"],
        )
        # Créer playlist_translations avec valeur différente
        conn.execute(
            "CREATE TABLE playlist_translations "
            "(uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)"
        )
        conn.execute(
            "INSERT INTO playlist_translations VALUES (?,?,?)",
            ["2825d417-0000-0000-0000-000000000000", "Big Team Battle", "Grande bataille"],
        )
        conn.close()

        resolver = MetadataResolver(db_path)
        name = resolver.resolve("playlist", "2825d417-0000-0000-0000-000000000000")
        # playlists (discovery) a la priorité
        assert name == "BTB (Discovery)"
        resolver.close()

    def test_fallback_to_playlist_translations(self, tmp_path: Path) -> None:
        """Si playlists existe mais ne contient pas l'UUID, fallback sur playlist_translations."""
        db_path = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db_path))
        # playlists vide
        conn.execute(
            "CREATE TABLE playlists "
            "(asset_id VARCHAR PRIMARY KEY, version_id VARCHAR, public_name VARCHAR)"
        )
        # playlist_translations avec une entrée
        conn.execute(
            "CREATE TABLE playlist_translations "
            "(uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)"
        )
        conn.execute(
            "INSERT INTO playlist_translations VALUES (?,?,?)",
            ["abcdef00-0000-0000-0000-000000000000", "Quick Play", "Partie rapide"],
        )
        conn.close()

        resolver = MetadataResolver(db_path)
        name = resolver.resolve("playlist", "abcdef00-0000-0000-0000-000000000000")
        assert name == "Partie rapide"
        resolver.close()
