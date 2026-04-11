"""Tests unitaires pour les extensions SPNKr (PR: challenges + reward tracks + item metadata).

Pattern de mock : unittest.mock.AsyncMock sur _get de BaseService.
Pas besoin d'aioresponses — on isole complètement la couche HTTP.

Chaque test :
  1. Charge la fixture JSON
  2. Crée une réponse aiohttp mockée dont .json() retourne la fixture
  3. Appelle la méthode du service
  4. Appelle .parse() sur le JsonResponse
  5. Vérifie les champs clés du modèle Pydantic
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any
from unittest.mock import AsyncMock, MagicMock

import pytest

# Assure que spnkr_pr est importable depuis la racine du repo.
_ROOT = Path(__file__).resolve().parents[2]
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from spnkr_pr.models.economy_additions import (
    Challenge,
    ChallengeCategory,
    PlayerChallenges,
    PlayerRewardTrackProgression,
    PlayerRewardTracksSummary,
    SpartanPoints,
)
from spnkr_pr.models.gamecms_hacs_additions import InventoryItem, OperationRewardTrackMeta
from spnkr_pr.services.economy_additions import EconomyServiceExtension
from spnkr_pr.services.gamecms_hacs_additions import GameCmsHacsServiceExtension

_FIXTURES = Path(__file__).parent / "fixtures"

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────


def _load_fixture(name: str) -> dict[str, Any]:
    return json.loads((_FIXTURES / name).read_text(encoding="utf-8"))


def _make_mock_response(payload: dict[str, Any]) -> AsyncMock:
    """Crée une réponse aiohttp fictive dont .json() retourne payload."""
    response = AsyncMock()
    response.json = AsyncMock(return_value=payload)
    response.read = AsyncMock(return_value=b"")
    response.raise_for_status = MagicMock()
    # Pas de .from_cache → sera rate-limité (sans effet en test car _get est mocké)
    del response.from_cache
    return response


def _make_service(service_cls, mock_response):
    """Instancie un service avec sa session mockée."""
    session = MagicMock()
    svc = service_cls.__new__(service_cls)
    svc._session = session
    svc._get = AsyncMock(return_value=mock_response)
    return svc


# ─────────────────────────────────────────────────────────────────────────────
# EconomyServiceExtension — get_player_challenges
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_challenges_returns_player_challenges():
    payload = _load_fixture("player_challenges.json")
    mock_resp = _make_mock_response(payload)
    svc = _make_service(EconomyServiceExtension, mock_resp)

    result = await svc.get_player_challenges("1234567890123456")
    parsed = await result.parse()

    assert isinstance(parsed, PlayerChallenges)


@pytest.mark.asyncio
async def test_get_player_challenges_deck_id():
    payload = _load_fixture("player_challenges.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_challenges("1234567890123456")).parse()

    assert parsed.challenge_deck_id == "ChallengeDecks/Season6-Weekly-2024-11-12.json"


@pytest.mark.asyncio
async def test_get_player_challenges_categories_count():
    payload = _load_fixture("player_challenges.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_challenges("1234567890123456")).parse()

    assert len(parsed.categories) == 2  # daily + weekly
    assert all(isinstance(c, ChallengeCategory) for c in parsed.categories)


@pytest.mark.asyncio
async def test_get_player_challenges_daily_category():
    payload = _load_fixture("player_challenges.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_challenges("1234567890123456")).parse()
    daily = parsed.categories[0]

    assert daily.campaign_id == "ChallengeDecks/Daily"
    assert len(daily.challenges) == 2
    assert all(isinstance(c, Challenge) for c in daily.challenges)


@pytest.mark.asyncio
async def test_get_player_challenges_completed_flag():
    payload = _load_fixture("player_challenges.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_challenges("1234567890123456")).parse()
    daily = parsed.categories[0].challenges

    assert daily[0].is_completed is False
    assert daily[1].is_completed is True
    assert daily[1].participation_challenge is True


@pytest.mark.asyncio
async def test_get_player_challenges_weekly_has_item_reward():
    payload = _load_fixture("player_challenges.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_challenges("1234567890123456")).parse()
    weekly_challenge = parsed.categories[1].challenges[0]

    assert len(weekly_challenge.item_rewards) == 1
    assert "Coatings" in weekly_challenge.item_rewards[0].inventory_item_path


@pytest.mark.asyncio
async def test_get_player_challenges_calls_correct_url():
    payload = _load_fixture("player_challenges.json")
    mock_resp = _make_mock_response(payload)
    svc = _make_service(EconomyServiceExtension, mock_resp)

    await svc.get_player_challenges("1234567890123456")

    svc._get.assert_called_once()
    called_url = svc._get.call_args[0][0]
    assert "xuid(1234567890123456)" in called_url
    assert "challenges" in called_url


# ─────────────────────────────────────────────────────────────────────────────
# EconomyServiceExtension — get_player_reward_tracks
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_reward_tracks_returns_summary():
    payload = _load_fixture("player_reward_tracks.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()

    assert isinstance(parsed, PlayerRewardTracksSummary)


@pytest.mark.asyncio
async def test_get_player_reward_tracks_operation_count():
    payload = _load_fixture("player_reward_tracks.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()

    assert len(parsed.operation_reward_tracks) == 2
    assert all(isinstance(t, PlayerRewardTrackProgression) for t in parsed.operation_reward_tracks)


@pytest.mark.asyncio
async def test_get_player_reward_tracks_current_operation_progress():
    payload = _load_fixture("player_reward_tracks.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()
    current_op = parsed.operation_reward_tracks[0]

    assert current_op.current_progress == 28500
    assert current_op.is_owned is True
    assert current_op.track_type == "Operations"


@pytest.mark.asyncio
async def test_get_player_reward_tracks_empty_events():
    payload = _load_fixture("player_reward_tracks.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()

    assert len(parsed.event_reward_tracks) == 0


@pytest.mark.asyncio
async def test_get_player_reward_tracks_career_rank():
    payload = _load_fixture("player_reward_tracks.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()
    cr = parsed.career_rank

    assert cr is not None
    assert cr.current_progress == 87  # rang entier, pas de l'XP
    assert cr.track_type == "CareerRanks"


@pytest.mark.asyncio
async def test_get_player_reward_tracks_no_career_rank_field():
    """career_rank est optionnel — doit accepter son absence."""
    payload = _load_fixture("player_reward_tracks.json")
    payload_no_cr = {k: v for k, v in payload.items() if k != "CareerRank"}
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload_no_cr))

    parsed = await (await svc.get_player_reward_tracks("1234567890123456")).parse()

    assert parsed.career_rank is None


# ─────────────────────────────────────────────────────────────────────────────
# EconomyServiceExtension — get_player_reward_track (single)
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_reward_track_single_returns_progression():
    payload = _load_fixture("player_reward_track_single.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_player_reward_track("1234567890123456", "operations", "Season6Operations-1")
    ).parse()

    assert isinstance(parsed, PlayerRewardTrackProgression)
    assert parsed.current_progress == 28500


@pytest.mark.asyncio
async def test_get_player_reward_track_single_url():
    payload = _load_fixture("player_reward_track_single.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    await svc.get_player_reward_track("1234567890123456", "operations", "Season6Operations-1")

    url = svc._get.call_args[0][0]
    assert "rewardtracks/operations/Season6Operations-1" in url
    assert "xuid(1234567890123456)" in url


# ─────────────────────────────────────────────────────────────────────────────
# EconomyServiceExtension — get_player_spartan_points
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_spartan_points_returns_model():
    payload = _load_fixture("spartan_points.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_spartan_points("1234567890123456")).parse()

    assert isinstance(parsed, SpartanPoints)
    assert parsed.balance == 500
    assert parsed.total_earned == 1250


@pytest.mark.asyncio
async def test_get_player_spartan_points_no_total_earned():
    """total_earned est optionnel."""
    payload = {"Balance": 200}
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    parsed = await (await svc.get_player_spartan_points("1234567890123456")).parse()

    assert parsed.balance == 200
    assert parsed.total_earned is None


@pytest.mark.asyncio
async def test_get_player_spartan_points_url():
    payload = _load_fixture("spartan_points.json")
    svc = _make_service(EconomyServiceExtension, _make_mock_response(payload))

    await svc.get_player_spartan_points("1234567890123456")

    url = svc._get.call_args[0][0]
    assert "currency/spartan-points" in url
    assert "xuid(1234567890123456)" in url


# ─────────────────────────────────────────────────────────────────────────────
# GameCmsHacsServiceExtension — get_item_metadata
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_item_metadata_returns_inventory_item():
    payload = _load_fixture("inventory_item.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_item_metadata("Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json")
    ).parse()

    assert isinstance(parsed, InventoryItem)


@pytest.mark.asyncio
async def test_get_item_metadata_common_data_id():
    payload = _load_fixture("inventory_item.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_item_metadata("Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json")
    ).parse()

    assert parsed.common_data.id == "013-001-olympus-c13d0f2f"
    assert parsed.common_data.item_type == "ArmorCore"
    assert parsed.common_data.title == "Mark VII"
    assert parsed.common_data.quality == "Common"


@pytest.mark.asyncio
async def test_get_item_metadata_image_path():
    payload = _load_fixture("inventory_item.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_item_metadata("Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json")
    ).parse()

    # Chemin à passer à get_image() pour récupérer le PNG
    image_url = parsed.common_data.display_path.display_path.media_url
    assert image_url == "career_rank/ArmorCores/olympus_preview.png"


@pytest.mark.asyncio
async def test_get_item_metadata_strips_leading_slash():
    """get_item_metadata doit accepter un chemin avec ou sans slash initial."""
    payload = _load_fixture("inventory_item.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    await svc.get_item_metadata("/Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json")

    url = svc._get.call_args[0][0]
    # Le double slash doit être absent
    assert "//Inventory" not in url
    assert "Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json" in url


@pytest.mark.asyncio
async def test_get_item_metadata_extra_fields_ignored():
    """Les champs inconnus ne doivent pas faire planter le parsing (extra=allow)."""
    payload = _load_fixture("inventory_item.json")
    # Le JSON de fixture contient déjà "SomeExtraField" — doit parser sans erreur
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_item_metadata("Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json")
    ).parse()

    assert parsed is not None


# ─────────────────────────────────────────────────────────────────────────────
# GameCmsHacsServiceExtension — get_operation_reward_track
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_operation_reward_track_returns_meta():
    payload = _load_fixture("operation_reward_track.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_operation_reward_track("Operations/Season6Operations-1.json")
    ).parse()

    assert isinstance(parsed, OperationRewardTrackMeta)


@pytest.mark.asyncio
async def test_get_operation_reward_track_summary_image_path():
    payload = _load_fixture("operation_reward_track.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_operation_reward_track("Operations/Season6Operations-1.json")
    ).parse()

    assert parsed.summary_image_path == "career_rank/Operations/Season6/summary_background.png"


@pytest.mark.asyncio
async def test_get_operation_reward_track_url():
    payload = _load_fixture("operation_reward_track.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    await svc.get_operation_reward_track("Operations/Season6Operations-1.json")

    url = svc._get.call_args[0][0]
    assert "Progression/file/Operations/Season6Operations-1.json" in url


@pytest.mark.asyncio
async def test_get_operation_reward_track_extra_fields_ignored():
    """Les champs inconnus (Ranks, Name, etc.) ne font pas planter le parsing."""
    payload = _load_fixture("operation_reward_track.json")
    svc = _make_service(GameCmsHacsServiceExtension, _make_mock_response(payload))

    parsed = await (
        await svc.get_operation_reward_track("Operations/Season6Operations-1.json")
    ).parse()

    assert parsed is not None
