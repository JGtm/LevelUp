"""Tests pour les transformers PVE (v5.2).

Couvre :
- _is_firefight_match : détection par GameVariantCategory et playlist_name
- _extract_enemy_kills_by_type : extraction des kills par type d'ennemi
- extract_pve_stats : extraction complète pour un match Firefight
- PveMatchStatsRow : création et champs
- batch_insert_pve_stats : insertion en base (in-memory DuckDB)
- ensure_pve_schema : idempotence du DDL
"""

from __future__ import annotations

import pytest

# =============================================================================
# Fixtures JSON
# =============================================================================

_XUID_A = "100000000000111"  # 15 digits — format valide pour XUID_RE
_XUID_B = "100000000000222"
_XUID_C = "100000000000333"
_XUID_D = "100000000000999"


def _make_firefight_json(
    match_id: str = "ff-match-001",
    category_id: int = 9,
    player_xuid: str = _XUID_A,
    waves: int = 3,
    boss_kills: int = 2,
    grunt_kills: int = 10,
    elite_kills: int = 5,
) -> dict:
    """Construit un JSON minimal d'un match Firefight."""
    return {
        "MatchId": match_id,
        "MatchInfo": {
            "GameVariantCategory": category_id,
        },
        "Players": [
            {
                "PlayerId": f"xuid({player_xuid})",
                "PlayerTeamStats": [
                    {
                        "Stats": {
                            "PveStats": {
                                "WavesCompleted": waves,
                                "MaxWaveReached": waves,
                                "BossKills": boss_kills,
                                "MythicBossKills": 0,
                                "TotalEnemyKills": grunt_kills + elite_kills,
                                "GruntKills": grunt_kills,
                                "EliteKills": elite_kills,
                            }
                        }
                    }
                ],
            }
        ],
    }


def _make_pvp_json(match_id: str = "pvp-match-001") -> dict:
    """Construit un JSON minimal d'un match PvP (non-Firefight)."""
    return {
        "MatchId": match_id,
        "MatchInfo": {
            "GameVariantCategory": 1,  # Catégorie PvP standard
        },
        "Players": [
            {
                "PlayerId": f"xuid({_XUID_B})",
                "PlayerTeamStats": [{"Stats": {}}],
            }
        ],
    }


# =============================================================================
# Tests _is_firefight_match
# =============================================================================


class TestIsFirefightMatch:
    def test_detection_by_category_id_9(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {"GameVariantCategory": 9}
        assert _is_firefight_match(match_info) is True

    def test_detection_by_category_id_24(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {"GameVariantCategory": 24}
        assert _is_firefight_match(match_info) is True

    def test_not_firefight_pvp_category(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {"GameVariantCategory": 1}
        assert _is_firefight_match(match_info) is False

    def test_detection_by_playlist_name_firefight(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {
            "GameVariantCategory": 99,  # Catégorie inconnue
            "Playlist": {"PublicName": "Firefight Heroic"},
        }
        assert _is_firefight_match(match_info) is True

    def test_detection_by_playlist_name_case_insensitive(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {"Playlist": {"PublicName": "FIREFIGHT HEROIC"}}
        assert _is_firefight_match(match_info) is True

    def test_detection_by_playlist_name_survive(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {"Playlist": {"PublicName": "Survive the storm"}}
        assert _is_firefight_match(match_info) is True

    def test_not_firefight_empty_dict(self):
        from src.data.sync.transformers import _is_firefight_match

        assert _is_firefight_match({}) is False

    def test_not_firefight_normal_pvp_playlist(self):
        from src.data.sync.transformers import _is_firefight_match

        match_info = {
            "GameVariantCategory": 3,
            "Playlist": {"PublicName": "Ranked Arena"},
        }
        assert _is_firefight_match(match_info) is False


# =============================================================================
# Tests _extract_enemy_kills_by_type
# =============================================================================


class TestExtractEnemyKillsByType:
    def test_direct_fields(self):
        from src.data.sync.transformers import _extract_enemy_kills_by_type

        pve_dict = {
            "GruntKills": 15,
            "EliteKills": 7,
            "JackalKills": 3,
            "BruteKills": 2,
            "HunterKills": 1,
        }
        result = _extract_enemy_kills_by_type(pve_dict)
        assert result["grunt"] == 15
        assert result["elite"] == 7
        assert result["jackal"] == 3
        assert result["brute"] == 2
        assert result["hunter"] == 1

    def test_sub_dict_structure(self):
        from src.data.sync.transformers import _extract_enemy_kills_by_type

        pve_dict = {
            "EnemyKillsByType": {
                "Grunt": 8,
                "Elite": 4,
                "Crawler": 12,
                "Knight": 1,
            }
        }
        result = _extract_enemy_kills_by_type(pve_dict)
        assert result["grunt"] == 8
        assert result["elite"] == 4
        assert result["crawler"] == 12
        assert result["knight"] == 1

    def test_empty_dict(self):
        from src.data.sync.transformers import _extract_enemy_kills_by_type

        result = _extract_enemy_kills_by_type({})
        assert result == {}

    def test_alternative_key_name(self):
        from src.data.sync.transformers import _extract_enemy_kills_by_type

        pve_dict = {"Grunts": 5}  # Alias
        result = _extract_enemy_kills_by_type(pve_dict)
        assert result["grunt"] == 5

    def test_direct_overrides_sub_dict(self):
        """Les champs directs ont priorité sur le sous-dictionnaire."""
        from src.data.sync.transformers import _extract_enemy_kills_by_type

        pve_dict = {
            "GruntKills": 20,
            "EnemyKillsByType": {"Grunt": 5},  # Ignoré si direct trouvé
        }
        result = _extract_enemy_kills_by_type(pve_dict)
        assert result["grunt"] == 20


# =============================================================================
# Tests extract_pve_stats
# =============================================================================


class TestExtractPveStats:
    def test_firefight_match_returns_rows(self):
        from src.data.sync.transformers import extract_pve_stats

        match_json = _make_firefight_json(
            match_id="ff-001",
            player_xuid=_XUID_A,
            waves=3,
            boss_kills=2,
            grunt_kills=10,
            elite_kills=5,
        )
        rows = extract_pve_stats(match_json)

        assert len(rows) == 1
        row = rows[0]
        assert row.match_id == "ff-001"
        assert row.xuid == _XUID_A
        assert row.waves_completed == 3
        assert row.boss_kills == 2
        assert row.grunt_kills == 10
        assert row.elite_kills == 5

    def test_pvp_match_returns_empty(self):
        from src.data.sync.transformers import extract_pve_stats

        rows = extract_pve_stats(_make_pvp_json())
        assert rows == []

    def test_missing_match_id_returns_empty(self):
        from src.data.sync.transformers import extract_pve_stats

        match_json = _make_firefight_json()
        del match_json["MatchId"]
        rows = extract_pve_stats(match_json)
        assert rows == []

    def test_no_players_returns_empty(self):
        from src.data.sync.transformers import extract_pve_stats

        match_json = _make_firefight_json()
        match_json["Players"] = []
        rows = extract_pve_stats(match_json)
        assert rows == []

    def test_multiple_players(self):
        from src.data.sync.transformers import extract_pve_stats

        match_json = {
            "MatchId": "ff-multi",
            "MatchInfo": {"GameVariantCategory": 9},
            "Players": [
                {
                    "PlayerId": f"xuid({_XUID_A})",
                    "PlayerTeamStats": [
                        {"Stats": {"PveStats": {"WavesCompleted": 2, "GruntKills": 8}}}
                    ],
                },
                {
                    "PlayerId": f"xuid({_XUID_B})",
                    "PlayerTeamStats": [
                        {"Stats": {"PveStats": {"WavesCompleted": 2, "EliteKills": 3}}}
                    ],
                },
            ],
        }
        rows = extract_pve_stats(match_json)
        assert len(rows) == 2
        xuids = {r.xuid for r in rows}
        assert _XUID_A in xuids
        assert _XUID_B in xuids

    def test_player_without_pve_stats_skipped(self):
        from src.data.sync.transformers import extract_pve_stats

        match_json = {
            "MatchId": "ff-partial",
            "MatchInfo": {"GameVariantCategory": 9},
            "Players": [
                {
                    "PlayerId": f"xuid({_XUID_A})",
                    "PlayerTeamStats": [{"Stats": {"PveStats": {"WavesCompleted": 2}}}],
                },
                {
                    "PlayerId": f"xuid({_XUID_B})",
                    "PlayerTeamStats": [{"Stats": {}}],  # Pas de PveStats
                },
            ],
        }
        rows = extract_pve_stats(match_json)
        # Le joueur sans stats PvE devrait être sauté
        assert len(rows) == 1
        assert rows[0].xuid == _XUID_A

    def test_playlist_name_detection_firefight(self):
        """Détection via playlist_name sans GameVariantCategory."""
        from src.data.sync.transformers import extract_pve_stats

        match_json = {
            "MatchId": "ff-playlist",
            "MatchInfo": {
                "GameVariantCategory": 99,
                "Playlist": {"PublicName": "Firefight Standard"},
            },
            "Players": [
                {
                    "PlayerId": f"xuid({_XUID_C})",
                    "PlayerTeamStats": [
                        {"Stats": {"PveStats": {"WavesCompleted": 1, "TotalEnemyKills": 20}}}
                    ],
                }
            ],
        }
        rows = extract_pve_stats(match_json)
        assert len(rows) == 1
        assert rows[0].total_enemy_kills == 20


# =============================================================================
# Tests batch_insert_pve_stats + ensure_pve_schema
# =============================================================================


class TestPveSchemaAndInsert:
    @pytest.fixture
    def pve_conn(self):
        """Connexion DuckDB in-memory avec schéma PVE."""
        import duckdb

        from src.data.sync.migrations import ensure_pve_schema

        conn = duckdb.connect(":memory:")
        ensure_pve_schema(conn)
        yield conn
        conn.close()

    def test_ensure_pve_schema_creates_table(self, pve_conn):
        result = pve_conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables " "WHERE table_name = 'pve_match_stats'"
        ).fetchone()
        assert result[0] == 1

    def test_ensure_pve_schema_idempotent(self):
        """ensure_pve_schema doit être idempotente (appelable plusieurs fois)."""
        import duckdb

        from src.data.sync.migrations import ensure_pve_schema

        conn = duckdb.connect(":memory:")
        ensure_pve_schema(conn)
        ensure_pve_schema(conn)  # Deuxième appel ne doit pas lever d'erreur
        conn.close()

    def test_batch_insert_pve_stats_returns_count(self, pve_conn):
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.models import PveMatchStatsRow

        rows = [
            PveMatchStatsRow(match_id="ff-001", xuid=_XUID_A, grunt_kills=10, elite_kills=5),
            PveMatchStatsRow(match_id="ff-001", xuid=_XUID_B, grunt_kills=7, elite_kills=2),
        ]
        inserted = batch_insert_pve_stats(pve_conn, rows)
        assert inserted == 2

    def test_batch_insert_pve_stats_empty_list(self, pve_conn):
        from src.data.sync.batch_insert import batch_insert_pve_stats

        inserted = batch_insert_pve_stats(pve_conn, [])
        assert inserted == 0

    def test_batch_insert_pve_stats_persists(self, pve_conn):
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.models import PveMatchStatsRow

        row = PveMatchStatsRow(
            match_id="ff-persist",
            xuid=_XUID_D,
            waves_completed=4,
            boss_kills=3,
            grunt_kills=15,
        )
        batch_insert_pve_stats(pve_conn, [row])

        result = pve_conn.execute(
            "SELECT match_id, xuid, waves_completed, boss_kills, grunt_kills "
            "FROM pve_match_stats WHERE match_id = ?",
            ["ff-persist"],
        ).fetchone()

        assert result is not None
        assert result[0] == "ff-persist"
        assert result[1] == _XUID_D
        assert result[2] == 4
        assert result[3] == 3
        assert result[4] == 15

    def test_batch_insert_pve_stats_insert_or_replace(self, pve_conn):
        """Un second insert avec même (match_id, xuid) doit remplacer."""
        from src.data.sync.batch_insert import batch_insert_pve_stats
        from src.data.sync.models import PveMatchStatsRow

        row1 = PveMatchStatsRow(match_id="ff-dup", xuid=_XUID_A, grunt_kills=5)
        row2 = PveMatchStatsRow(match_id="ff-dup", xuid=_XUID_A, grunt_kills=99)

        batch_insert_pve_stats(pve_conn, [row1])
        batch_insert_pve_stats(pve_conn, [row2])

        count = pve_conn.execute(
            "SELECT COUNT(*) FROM pve_match_stats WHERE match_id = ?", ["ff-dup"]
        ).fetchone()[0]
        assert count == 1

        val = pve_conn.execute(
            "SELECT grunt_kills FROM pve_match_stats WHERE match_id = ?", ["ff-dup"]
        ).fetchone()[0]
        assert val == 99


# =============================================================================
# Tests PveMatchStatsRow
# =============================================================================


class TestPveMatchStatsRow:
    def test_default_values(self):
        from src.data.sync.models import PveMatchStatsRow

        row = PveMatchStatsRow(match_id="m1", xuid="u1")
        assert row.match_id == "m1"
        assert row.xuid == "u1"
        assert row.waves_completed is None  # Optional field (None par défaut)
        assert row.grunt_kills == 0  # Kills default à 0 (pas None)
        assert row.pve_bits == 0

    def test_all_fields(self):
        from src.data.sync.models import PveMatchStatsRow

        row = PveMatchStatsRow(
            match_id="m2",
            xuid="u2",
            waves_completed=5,
            max_wave_reached=5,
            boss_kills=3,
            mythic_boss_kills=1,
            total_enemy_kills=100,
            grunt_kills=40,
            elite_kills=20,
            jackal_kills=10,
            brute_kills=8,
            hunter_kills=2,
            skimmer_kills=5,
            crawler_kills=6,
            soldier_kills=4,
            knight_kills=3,
            warden_kills=2,
            pve_bits=7,
        )
        assert row.waves_completed == 5
        assert row.pve_bits == 7
        assert row.warden_kills == 2


# =============================================================================
# Tests PveBits et MatchBits
# =============================================================================


class TestPveBitsConstants:
    def test_pve_bits_values(self):
        from src.data.sync.constants import PveBits

        assert PveBits.WAVES == 1
        assert PveBits.BOSS_KILLS == 2
        assert PveBits.TOTAL_KILLS == 4
        assert PveBits.GRUNT == 8
        assert PveBits.WARDEN == 4096

    def test_pve_bits_combinations(self):
        from src.data.sync.constants import PveBits

        assert PveBits.BANISHED_FULL == (
            PveBits.GRUNT
            | PveBits.ELITE
            | PveBits.JACKAL
            | PveBits.BRUTE
            | PveBits.HUNTER
            | PveBits.SKIMMER
        )
        assert PveBits.FULL_PVE == (
            PveBits.WAVES
            | PveBits.BOSS_KILLS
            | PveBits.TOTAL_KILLS
            | PveBits.BANISHED_FULL
            | PveBits.FORERUNNER_ANY
        )

    def test_match_bits_pve_stats(self):
        from src.data.sync.constants import MatchBits

        assert MatchBits.PVE_STATS == 1 << 20
        assert MatchBits.PVE_STATS == 1048576

    def test_match_bits_no_collision_with_events(self):
        """PVE_STATS ne doit pas entrer en collision avec EVENTS (1<<16)."""
        from src.data.sync.constants import MatchBits

        assert MatchBits.PVE_STATS != MatchBits.EVENTS
        assert (MatchBits.PVE_STATS & MatchBits.EVENTS) == 0


# =============================================================================
# Tests SyncScope PVE
# =============================================================================


class TestSyncScopePve:
    def test_pve_stats_field_exists(self):
        from src.data.sync.scope import SyncScope

        scope = SyncScope(pve_stats=True)
        assert scope.pve_stats is True

    def test_force_pve_stats_implies_pve_stats(self):
        from src.data.sync.scope import SyncScope

        scope = SyncScope(force_pve_stats=True)
        scope.resolve()
        assert scope.pve_stats is True

    def test_pve_stats_in_all_data(self):
        from src.data.sync.scope import SyncScope

        scope = SyncScope(all_data=True)
        scope.resolve()
        assert scope.pve_stats is True

    def test_pve_stats_not_in_scope_by_default(self):
        from src.data.sync.scope import SyncScope

        scope = SyncScope()
        assert scope.pve_stats is False
        assert scope.force_pve_stats is False
