"""Tests pour l'architecture i18n v6 — metadata.duckdb + v_match_full.

Commit 11 : vérifie que la suppression des JSON/dicts playlists n'a pas
cassé la chaîne de traduction et que les nouvelles sources (v_match_full,
metadata.duckdb) fonctionnent correctement.
"""

from __future__ import annotations

import logging
from pathlib import Path

import duckdb
import polars as pl
import pytest

from src.analysis.filters import mark_firefight
from src.ui.translations import translate_playlist_name

# ---------------------------------------------------------------------------
# 1. v_match_full — playlist_name en anglais (source de vérité métier)
# ---------------------------------------------------------------------------


def test_v_match_full_playlist_name_is_english() -> None:
    """v_match_full.playlist_name doit contenir des noms EN, jamais des traductions FR."""
    v2 = Path("data/warehouse/shared_matches_v2.duckdb")
    if not v2.exists():
        pytest.skip("shared_matches_v2.duckdb non disponible")

    conn = duckdb.connect(str(v2), read_only=True)
    try:
        sample = conn.execute(
            "SELECT DISTINCT playlist_name FROM v_match_full "
            "WHERE playlist_name IS NOT NULL LIMIT 20"
        ).fetchall()
    finally:
        conn.close()

    names = [r[0] for r in sample]
    assert names, "Aucun match avec playlist_name non NULL dans v_match_full"
    # Les noms EN attendus dans ces données
    for name in names:
        assert "Baptême" not in name, f"Nom FR inattendu dans playlist_name : '{name}'"
        assert "Arène" not in name, f"Nom FR inattendu dans playlist_name : '{name}'"


# ---------------------------------------------------------------------------
# 2. v_match_full — playlist_name_fr peuplé depuis metadata.duckdb
# ---------------------------------------------------------------------------


def test_v_match_full_playlist_name_fr_populated(tmp_path: Path) -> None:
    """Quand metadata.duckdb a une table playlists, v_match_full expose playlist_name_fr."""
    # Créer metadata.duckdb avec playlists
    meta_path = tmp_path / "metadata.duckdb"
    meta = duckdb.connect(str(meta_path))
    meta.execute(
        "CREATE TABLE playlists ("
        "  asset_id VARCHAR PRIMARY KEY,"
        "  name_en VARCHAR,"
        "  name_fr VARCHAR,"
        "  playlist_canonical_en VARCHAR,"
        "  playlist_canonical_fr VARCHAR"
        ")"
    )
    meta.execute(
        "INSERT INTO playlists VALUES (?, ?, ?, ?, ?)",
        ["pl-001", "Quick Play", "Partie rapide", "Quick Play", "Partie rapide"],
    )
    meta.close()

    # Créer shared mini avec match_registry + recréer la vue
    shared_path = tmp_path / "shared.duckdb"
    conn = duckdb.connect(str(shared_path))
    conn.execute(
        "CREATE TABLE match_registry ("
        "  match_id VARCHAR PRIMARY KEY,"
        "  playlist_id VARCHAR,"
        "  playlist_name VARCHAR,"
        "  start_time TIMESTAMP,"
        "  duration_seconds INTEGER,"
        "  map_id VARCHAR, pair_id VARCHAR, game_variant_id VARCHAR,"
        "  team_0_score INTEGER, team_1_score INTEGER,"
        "  is_firefight BOOLEAN, is_ranked BOOLEAN,"
        "  backfill_completed BOOLEAN, sync_spnkr_version VARCHAR,"
        "  map_name VARCHAR, pair_name VARCHAR, game_variant_name VARCHAR"
        ")"
    )
    conn.execute(
        "INSERT INTO match_registry (match_id, playlist_id, playlist_name) "
        "VALUES ('m1', 'pl-001', 'Quick Play')"
    )
    conn.execute(f"ATTACH '{meta_path}' AS meta (READ_ONLY)")
    conn.execute(
        "CREATE VIEW v_match_full AS "
        "SELECT mr.*, "
        "NULL AS map_name_fr, "
        "COALESCE(p.name_fr, mr.playlist_name) AS playlist_name_fr, "
        "NULL AS pair_name_fr, NULL AS game_variant_name_fr, "
        "NULL AS mode_name, NULL AS mode_name_fr, "
        "NULL AS playlist_canonical_en, p.playlist_canonical_fr "
        "FROM match_registry mr "
        "LEFT JOIN meta.playlists p ON mr.playlist_id = p.asset_id"
    )
    row = conn.execute(
        "SELECT playlist_name, playlist_name_fr FROM v_match_full WHERE match_id='m1'"
    ).fetchone()
    conn.close()

    assert row is not None
    assert row[0] == "Quick Play", "playlist_name EN doit être préservé"
    assert row[1] == "Partie rapide", "playlist_name_fr doit être résolu depuis metadata"


# ---------------------------------------------------------------------------
# 3. mark_firefight() fonctionne avec les noms EN de v_match_full
# ---------------------------------------------------------------------------


def test_mark_firefight_still_works_after_v_match_full() -> None:
    """mark_firefight() détecte is_firefight=True depuis playlist_name EN 'Firefight'."""
    df = pl.DataFrame({"playlist_name": ["Firefight", "Quick Play", "Ranked Arena"]})
    result = mark_firefight(df)
    ff_col = result["is_firefight"].to_list()
    assert ff_col[0] is True, "Firefight → is_firefight doit être True"
    assert ff_col[1] is False, "Quick Play → is_firefight doit être False"
    assert ff_col[2] is False, "Ranked Arena → is_firefight doit être False"


# ---------------------------------------------------------------------------
# 4. translate_playlist_name() UUID → warning + label "Inconnue"
# ---------------------------------------------------------------------------


def test_translate_playlist_name_uuid_logs_warning(caplog: pytest.LogCaptureFixture) -> None:
    """UUID brut → WARNING loggué, retourne 'Inconnue' (FR) ou 'Unknown' (EN)."""
    uuid_full = "a446725e-b281-414c-a21e-1234567890ab"

    with caplog.at_level(logging.WARNING, logger="src.ui.translations"):
        result_fr = translate_playlist_name(uuid_full, lang="fr")
        result_en = translate_playlist_name(uuid_full, lang="en")

    assert result_fr == "Inconnue", f"attendu 'Inconnue', obtenu '{result_fr}'"
    assert result_en == "Unknown", f"attendu 'Unknown', obtenu '{result_en}'"
    assert any(
        "UUID brut" in r.message or "non r" in r.message for r in caplog.records
    ), "Un WARNING doit être loggué pour un UUID non résolu"


# ---------------------------------------------------------------------------
# 5. game_variants.mode_name — extraction depuis public_name
# ---------------------------------------------------------------------------


def test_game_variants_mode_name_count(tmp_path: Path) -> None:
    """game_variants : SPLIT_PART(public_name, ':', 2) extrait mode_name correctement."""
    import sys

    sys.path.insert(0, str(Path(__file__).parent.parent / "scripts"))
    from _metadata_db import create_metadata_db

    meta_path = tmp_path / "metadata.duckdb"
    conn = duckdb.connect(str(meta_path))
    create_metadata_db(conn)

    # Insérer 3 game_variants avec préfixe:mode
    data = [
        ("gv-01", "v1", "Arena:Slayer"),
        ("gv-02", "v1", "Arena:CTF"),
        ("gv-03", "v1", "BTB:Slayer"),
        ("gv-04", "v1", "Firefight:King of the Hill"),
    ]
    conn.executemany(
        "INSERT INTO game_variants (asset_id, version_id, public_name) VALUES (?, ?, ?)", data
    )
    # Appliquer extraction mode_name
    conn.execute("""
        UPDATE game_variants
        SET mode_name = TRIM(SPLIT_PART(public_name, ':', 2))
        WHERE CONTAINS(public_name, ':') AND mode_name IS NULL
    """)
    modes = conn.execute(
        "SELECT DISTINCT mode_name FROM game_variants WHERE mode_name IS NOT NULL"
    ).fetchall()
    conn.close()

    mode_names = {r[0] for r in modes}
    assert "Slayer" in mode_names
    assert "CTF" in mode_names
    assert "King of the Hill" in mode_names
    assert len(mode_names) == 3  # Slayer apparaît 2x → dédupliqué


# ---------------------------------------------------------------------------
# 6. playlist_canonical_fr regroupe les variantes BTB
# ---------------------------------------------------------------------------


def test_btb_variants_share_canonical_fr(tmp_path: Path) -> None:
    """Les 3 variantes BTB partagent le même playlist_canonical_fr."""
    import sys

    sys.path.insert(0, str(Path(__file__).parent.parent / "scripts"))
    from _metadata_db import create_metadata_db

    meta_path = tmp_path / "metadata.duckdb"
    conn = duckdb.connect(str(meta_path))
    create_metadata_db(conn)

    btb_variants = [
        ("btb-1", "v1", "Big Team Battle", "Grande bataille en équipe"),
        ("btb-2", "v1", "Big Team Battle: Refresh", "Grande bataille en équipe"),
        ("btb-3", "v1", "Big Team Social", "Grande bataille sociale"),
    ]
    conn.executemany(
        "INSERT INTO playlists (asset_id, version_id, public_name, name_fr) VALUES (?, ?, ?, ?)",
        btb_variants,
    )
    # Calculer playlist_canonical logique : on extrait le préfixe avant ':'
    conn.execute("""
        UPDATE playlists
        SET playlist_canonical_en = TRIM(SPLIT_PART(public_name, ':', 1))
        WHERE playlist_canonical_en IS NULL AND public_name IS NOT NULL
    """)
    conn.execute("""
        UPDATE playlists
        SET playlist_canonical_fr = name_fr
        WHERE playlist_canonical_fr IS NULL AND name_fr IS NOT NULL
    """)

    rows = conn.execute(
        "SELECT public_name, playlist_canonical_fr FROM playlists ORDER BY public_name"
    ).fetchall()
    conn.close()

    canonical_fr = {r[0]: r[1] for r in rows}
    assert canonical_fr["Big Team Battle"] == "Grande bataille en équipe"
    assert canonical_fr["Big Team Battle: Refresh"] == "Grande bataille en équipe"
    assert canonical_fr["Big Team Social"] == "Grande bataille sociale"


# ---------------------------------------------------------------------------
# 7. metadata_populate_errors.txt créé si extraction mode_name échoue
# ---------------------------------------------------------------------------


def test_populate_errors_file_created_on_failure(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """_write_enrich_errors() crée le fichier d'erreurs si des erreurs existent."""
    import sys

    sys.path.insert(0, str(Path(__file__).parent.parent / "scripts"))
    import _metadata_db as mdb

    errors_path = tmp_path / "metadata_populate_errors.txt"
    monkeypatch.setattr(mdb, "ERRORS_FILE_PATH", errors_path)

    # Simuler 1 erreur (game_variant sans ':' dans public_name)
    mdb._write_enrich_errors(
        [
            'game_variants | asset_id=gv-99 | public_name="NoColon" | raison=mode_name_extraction_failed'
        ]
    )

    assert errors_path.exists(), "Le fichier d'erreurs doit être créé"
    content = errors_path.read_text(encoding="utf-8")
    assert "gv-99" in content
    assert "mode_name_extraction_failed" in content
