"""Tests E2E — pipeline sync complet avec API mockée.

Teste le flux sync_delta() et sync_full() de bout en bout :
API mockée → transformers → insertions shared + player DB → post-sync pipeline.

Ces tests vérifient que le pipeline end-to-end produit les bonnes données
dans les bonnes tables, sans tester les composants isolément.

Scénarios couverts :
- sync_full insère N matchs dans shared + enrichments dans player DB
- sync_delta avec match déjà connu → short-circuit
- sync_delta avec nouveau match → insertion
- Erreur API → résultat avec erreur, pas de crash
- Post-sync pipeline : perf scores calculés, sync_meta peuplé
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any
from unittest.mock import AsyncMock, patch

import duckdb

from tests.conftest_sync import (
    BASE_TIME,
    GT_OPPONENT,
    GT_PLAYER_A,
    XUID_OPPONENT,
    XUID_PLAYER_A,
    count_rows,
    make_engine,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _build_match_stats_json(match_id: str, start_time: datetime) -> dict[str, Any]:
    """Construit un JSON stats_json réaliste pour un match."""
    return {
        "MatchId": match_id,
        "MatchInfo": {
            "StartTime": start_time.isoformat(),
            "EndTime": (start_time + timedelta(minutes=10)).isoformat(),
            "Duration": "PT10M0S",
            "Playlist": {
                "AssetId": "playlist-001",
                "VersionId": "v1",
                "PublicName": "Quick Play",
            },
            "MapVariant": {
                "AssetId": "map-001",
                "VersionId": "v1",
                "PublicName": "Recharge",
            },
            "PlaylistMapModePair": {
                "AssetId": "pair-001",
                "VersionId": "v1",
                "PublicName": "Recharge - Slayer",
            },
            "UgcGameVariant": {
                "AssetId": "variant-001",
                "VersionId": "v1",
                "PublicName": "Slayer",
            },
            "GameVariantCategory": 1,
            "PlayableDuration": "PT9M55S",
            "LifecycleMode": 3,
            "LevelId": "level-001",
            "Clearance": {"PlaylistClearanceId": "unknown"},
        },
        "Teams": [
            {"TeamId": 0, "TotalPoints": 50},
            {"TeamId": 1, "TotalPoints": 45},
        ],
        "Players": [
            {
                "PlayerId": f"xuid({XUID_PLAYER_A})",
                "PlayerGamertag": GT_PLAYER_A,
                "Outcome": 2,
                "LastTeamId": 0,
                "Rank": 1,
                "PlayerTeamStats": [
                    {
                        "Stats": {
                            "CoreStats": {
                                "Kills": 15,
                                "Deaths": 8,
                                "Assists": 5,
                                "KDA": 1.875,
                                "Accuracy": 0.45,
                                "HeadshotKills": 7,
                                "MaxKillingSpree": 5,
                                "AverageLifeSeconds": 45.0,
                                "DamageDealt": 3500.0,
                                "DamageTaken": 2800.0,
                                "ShotsFired": 200,
                                "ShotsHit": 90,
                                "MeleeKills": 1,
                                "GrenadeKills": 2,
                                "PowerWeaponKills": 3,
                                "PersonalScore": 1500,
                                "Score": 1500,
                                "Medals": [
                                    {"NameId": 100, "Count": 3},
                                    {"NameId": 200, "Count": 1},
                                ],
                            },
                            "BombStats": None,
                            "CaptureTheFlagStats": None,
                            "ZonesStats": None,
                        },
                        "PersonalScores": [
                            {"NameId": 1, "Count": 5, "TotalPersonalScoreAwarded": 500},
                            {"NameId": 2, "Count": 3, "TotalPersonalScoreAwarded": 300},
                        ],
                    }
                ],
                "ParticipationInfo": {
                    "TimePlayed": "PT10M0S",
                    "JoinedInProgress": False,
                    "ConfirmedParticipation": True,
                },
            },
            {
                "PlayerId": f"xuid({XUID_OPPONENT})",
                "PlayerGamertag": GT_OPPONENT,
                "Outcome": 3,
                "LastTeamId": 1,
                "Rank": 2,
                "PlayerTeamStats": [
                    {
                        "Stats": {
                            "CoreStats": {
                                "Kills": 8,
                                "Deaths": 15,
                                "Assists": 3,
                                "KDA": 0.53,
                                "Accuracy": 0.38,
                                "HeadshotKills": 3,
                                "MaxKillingSpree": 2,
                                "AverageLifeSeconds": 30.0,
                                "DamageDealt": 2500.0,
                                "DamageTaken": 3500.0,
                                "ShotsFired": 180,
                                "ShotsHit": 68,
                                "MeleeKills": 0,
                                "GrenadeKills": 1,
                                "PowerWeaponKills": 0,
                                "PersonalScore": 800,
                                "Score": 800,
                                "Medals": [
                                    {"NameId": 100, "Count": 1},
                                ],
                            },
                            "BombStats": None,
                            "CaptureTheFlagStats": None,
                            "ZonesStats": None,
                        },
                        "PersonalScores": [
                            {"NameId": 1, "Count": 2, "TotalPersonalScoreAwarded": 200},
                        ],
                    }
                ],
                "ParticipationInfo": {
                    "TimePlayed": "PT10M0S",
                    "JoinedInProgress": False,
                    "ConfirmedParticipation": True,
                },
            },
        ],
    }


@dataclass
class FakeMatchHistoryItem:
    """Simule un MatchHistoryItem retourné par l'API."""

    match_id: str
    start_time: str = ""
    match_type: str = "matchmaking"


def _build_mock_client(
    match_ids: list[str],
    stats_jsons: dict[str, dict],
) -> AsyncMock:
    """Construit un mock API client complet.

    Args:
        match_ids: IDs retournés par get_match_history (du plus récent au plus ancien).
        stats_jsons: Dict match_id → stats JSON.
    """
    client = AsyncMock()

    async def _get_history(player, match_type="matchmaking", start=0, count=25):
        batch = match_ids[start : start + count]
        return [FakeMatchHistoryItem(mid) for mid in batch]

    client.get_match_history = AsyncMock(side_effect=_get_history)

    async def _get_stats(match_id):
        return stats_jsons.get(match_id)

    client.get_match_stats = AsyncMock(side_effect=_get_stats)
    client.get_skill_stats = AsyncMock(return_value=None)
    client.get_highlight_events = AsyncMock(return_value=[])
    client.get_asset = AsyncMock(return_value=None)

    client.__aenter__ = AsyncMock(return_value=client)
    client.__aexit__ = AsyncMock(return_value=None)
    return client


# ===========================================================================
# Tests sync_full E2E
# ===========================================================================


class TestSyncFullE2E:
    """Tests end-to-end pour sync_full."""

    def test_sync_full_inserts_matches(self, tmp_path: Path) -> None:
        """sync_full insère les matchs dans shared + enrichments dans player DB."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-e2e-001", "match-e2e-002"]
            stats = {
                mid: _build_match_stats_json(mid, BASE_TIME + timedelta(hours=i))
                for i, mid in enumerate(match_ids)
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=10,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=False))

            assert result.matches_inserted == 2
            assert len(result.errors) == 0

            # Vérifier shared DB
            shared = engine._get_shared_connection()
            assert count_rows(shared, "match_registry") == 2
            assert count_rows(shared, "match_participants") == 4  # 2 joueurs × 2 matchs

            # Vérifier player DB
            conn = engine._get_connection()
            pme_count = count_rows(conn, "player_match_enrichment")
            assert pme_count == 2  # 1 PME par match
        finally:
            engine.close()

    def test_sync_full_with_medals(self, tmp_path: Path) -> None:
        """sync_full insère aussi les médailles dans shared."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-medals-e2e"]
            stats = {
                "match-medals-e2e": _build_match_stats_json("match-medals-e2e", BASE_TIME),
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=False))

            assert result.matches_inserted >= 1

            shared = engine._get_shared_connection()
            medal_count = count_rows(shared, "medals_earned")
            # Le match a 3 médailles (2 pour player A, 1 pour opponent)
            assert medal_count >= 3
        finally:
            engine.close()

    def test_sync_full_with_personal_scores(self, tmp_path: Path) -> None:
        """sync_full insère les personal_score_awards dans player DB."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-psa-e2e"]
            stats = {
                "match-psa-e2e": _build_match_stats_json("match-psa-e2e", BASE_TIME),
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=False))  # noqa: F841

            conn = engine._get_connection()
            psa_count = count_rows(conn, "personal_score_awards")
            # Player A a 2 PersonalScores dans le JSON
            assert psa_count >= 2
        finally:
            engine.close()

    def test_sync_full_api_error(self, tmp_path: Path) -> None:
        """Erreur API → résultat avec erreur, pas d'exception."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-err-001"]
            # get_match_stats retourne None pour simuler l'erreur
            mock_client = _build_mock_client(match_ids, {})

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=False))

            # Pas d'exception levée, mais warnings
            assert result.matches_inserted == 0
        finally:
            engine.close()


# ===========================================================================
# Tests sync_delta E2E
# ===========================================================================


class TestSyncDeltaE2E:
    """Tests end-to-end pour sync_delta."""

    def test_delta_short_circuit_when_up_to_date(self, tmp_path: Path) -> None:
        """Delta short-circuit quand le dernier match est déjà en DB."""
        engine = make_engine(tmp_path)
        try:
            # Pre-seed : match-delta-001 déjà complet dans shared + player
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-delta-001", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists) VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                ["match-delta-001", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-delta-001"],
            )
            conn.execute(
                "INSERT INTO personal_score_awards (match_id, xuid, award_name, award_score) "
                "VALUES (?, ?, 'Kill', 100)",
                ["match-delta-001", XUID_PLAYER_A],
            )
            conn.commit()

            # API retourne le même match
            mock_client = _build_mock_client(["match-delta-001"], {})

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=True))

            # Short-circuit : 0 matchs insérés
            assert result.matches_inserted == 0
        finally:
            engine.close()

    def test_delta_processes_new_match(self, tmp_path: Path) -> None:
        """Delta insère un nouveau match quand l'API retourne un match non connu."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-delta-new"]
            stats = {
                "match-delta-new": _build_match_stats_json("match-delta-new", BASE_TIME),
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=True))

            assert result.matches_inserted == 1

            shared = engine._get_shared_connection()
            assert count_rows(shared, "match_registry") == 1
        finally:
            engine.close()

    def test_delta_stops_at_known_match(self, tmp_path: Path) -> None:
        """Delta s'arrête dès qu'un match déjà connu est rencontré dans l'historique."""
        engine = make_engine(tmp_path)
        try:
            # Pre-seed : match-known exists dans shared
            shared = engine._get_shared_connection()
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-known", BASE_TIME],
            )
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, personal_score) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 0)",
                ["match-known", XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.commit()

            conn = engine._get_connection()
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                ["match-known"],
            )
            conn.commit()

            # Historique : nouveau + connu
            all_ids = ["match-new-1", "match-known"]
            stats = {
                "match-new-1": _build_match_stats_json(
                    "match-new-1", BASE_TIME + timedelta(hours=1)
                ),
            }
            mock_client = _build_mock_client(all_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=10,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                result = asyncio.run(engine._sync_internal(options, delta_mode=True))

            # Seul le nouveau match inséré
            assert result.matches_inserted == 1
        finally:
            engine.close()

        # Vérifier via connexion directe (après fermeture du engine)
        shared_db = tmp_path / "data" / "warehouse" / "shared_matches_v2.duckdb"
        with duckdb.connect(str(shared_db), read_only=True) as check_conn:
            assert count_rows(check_conn, "match_registry") == 2


# ===========================================================================
# Tests post-sync pipeline
# ===========================================================================


class TestPostSyncMetadata:
    """Vérifie que sync_meta est peuplé après un sync réussi."""

    def test_sync_meta_updated_after_full(self, tmp_path: Path) -> None:
        """Après sync_full, sync_meta contient les données de la dernière sync."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-meta-001"]
            stats = {
                "match-meta-001": _build_match_stats_json("match-meta-001", BASE_TIME),
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                asyncio.run(engine._sync_internal(options, delta_mode=False))

            assert engine._get_sync_meta("last_sync_mode") == "full"
            assert engine._get_sync_meta("last_sync_matches") == "1"
            assert engine._get_sync_meta("gamertag") == GT_PLAYER_A
        finally:
            engine.close()


# ===========================================================================
# Tests idempotence
# ===========================================================================


class TestSyncIdempotence:
    """Vérifie que lancer sync 2 fois ne duplique pas les données."""

    def test_double_sync_no_duplicates(self, tmp_path: Path) -> None:
        """Deux syncs full identiques → même nombre de lignes."""
        engine = make_engine(tmp_path)
        try:
            match_ids = ["match-idem-001"]
            stats = {
                "match-idem-001": _build_match_stats_json("match-idem-001", BASE_TIME),
            }
            mock_client = _build_mock_client(match_ids, stats)

            from src.data.sync.models import SyncOptions

            options = SyncOptions(
                max_matches=5,
                with_highlight_events=False,
                with_skill=False,
                with_career_rank=False,
                with_weapons=False,
            )

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client

                # 1er sync
                r1 = asyncio.run(engine._sync_internal(options, delta_mode=False))
                assert r1.matches_inserted == 1

            # Pour le 2e sync, il faut un nouveau mock client
            mock_client2 = _build_mock_client(match_ids, stats)

            with patch("src.data.sync.engine.create_api_client") as mock_factory:
                mock_factory.return_value = mock_client2

                # Reset le cache pour forcer le rechargement
                engine._existing_match_ids = None

                r2 = asyncio.run(engine._sync_internal(options, delta_mode=False))  # noqa: F841

            # Le 2e sync doit traiter le match comme "known" (déjà dans registry)
            # mais ne pas créer de duplicat dans match_registry
            shared = engine._get_shared_connection()
            assert count_rows(shared, "match_registry") == 1
        finally:
            engine.close()
