"""Tests unitaires — weapon_parser (domaine pur).

Couvre : constantes, find_frame_positions, build_frame_estimator,
correlate_kills_to_weapons, count_kills_by_api_weapon.
"""

from __future__ import annotations

from collections import Counter

import pytest

from src.analysis.weapon_parser import (
    COMMON_WEAPON_SUFFIX,
    FILM_BYTES_TO_API_ID,
    FILM_NAME_TO_API_ID,
    FRAME_MARKER,
    GRENADE_API_ID,
    GRENADE_MEDALS,
    KILL_WINDOW_MS,
    MELEE_API_ID,
    MELEE_MEDALS,
    POV_PLAYER_INDEX,
    WEAPON_ID_MAP,
    WEAPON_IDS_INT,
    WEAPON_INT_TO_NAME,
    build_frame_estimator,
    correlate_kills_to_weapons,
    count_kills_by_api_weapon,
    find_frame_positions,
    scan_fire_events,
)

# ═══════════════════════════════════════════════════════════════════════════
# Constantes
# ═══════════════════════════════════════════════════════════════════════════


class TestConstants:
    """Vérifie la cohérence des constantes."""

    def test_weapon_id_map_has_entries(self):
        assert len(WEAPON_ID_MAP) >= 40

    def test_all_weapon_ids_are_8_bytes(self):
        for wid in WEAPON_ID_MAP:
            assert len(wid) == 8, f"Clé {wid.hex()} n'est pas 8 bytes"

    def test_weapon_ids_int_matches_map(self):
        assert len(WEAPON_IDS_INT) == len(WEAPON_ID_MAP)

    def test_weapon_int_to_name_matches_map(self):
        assert len(WEAPON_INT_TO_NAME) == len(WEAPON_ID_MAP)

    def test_common_suffix_is_4_bytes(self):
        assert len(COMMON_WEAPON_SUFFIX) == 4

    def test_most_weapons_have_common_suffix(self):
        """La majorité des armes utilisent le suffixe commun."""
        with_suffix = sum(1 for w in WEAPON_ID_MAP if w[4:] == COMMON_WEAPON_SUFFIX)
        assert with_suffix >= 35

    def test_film_name_to_api_id_complete(self):
        """Chaque nom dans FILM_NAME_TO_API_ID est un entier positif ou 0 (grenades)."""
        for name, api_id in FILM_NAME_TO_API_ID.items():
            assert isinstance(api_id, int), f"{name}: {api_id}"
            assert api_id >= 0, f"{name}: valeur négative"

    def test_film_bytes_mapping_covers_map(self):
        """Chaque entrée WEAPON_ID_MAP a une correspondance dans FILM_BYTES_TO_API_ID."""
        for wbytes in WEAPON_ID_MAP:
            assert wbytes in FILM_BYTES_TO_API_ID

    def test_pov_player_index(self):
        assert POV_PLAYER_INDEX == 1

    def test_melee_api_id(self):
        assert MELEE_API_ID == 1

    def test_grenade_api_id(self):
        assert GRENADE_API_ID == 0

    def test_kill_window_ms(self):
        assert KILL_WINDOW_MS == 2000

    def test_melee_medals_is_frozenset(self):
        assert isinstance(MELEE_MEDALS, frozenset)
        assert "Pummel" in MELEE_MEDALS

    def test_grenade_medals_is_frozenset(self):
        assert isinstance(GRENADE_MEDALS, frozenset)
        assert "Stick" in GRENADE_MEDALS


# ═══════════════════════════════════════════════════════════════════════════
# find_frame_positions
# ═══════════════════════════════════════════════════════════════════════════


class TestFindFramePositions:
    def test_empty_data(self):
        assert find_frame_positions(b"") == []

    def test_no_markers(self):
        assert find_frame_positions(b"\x00" * 100) == []

    def test_single_marker(self):
        data = b"\x00\x00" + FRAME_MARKER + b"\x00\x00"
        assert find_frame_positions(data) == [2]

    def test_multiple_markers(self):
        data = FRAME_MARKER + b"\xff" + FRAME_MARKER + b"\xff\xff" + FRAME_MARKER
        assert find_frame_positions(data) == [0, 4, 9]

    def test_adjacent_markers(self):
        data = FRAME_MARKER + FRAME_MARKER
        assert find_frame_positions(data) == [0, 3]


# ═══════════════════════════════════════════════════════════════════════════
# build_frame_estimator
# ═══════════════════════════════════════════════════════════════════════════


class TestBuildFrameEstimator:
    def test_basic_estimation(self):
        # 4 frames dans 400 ms → 100 ms/frame
        data = (
            FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
        )  # noqa: E501
        estimate = build_frame_estimator(data, chunk_start_ms=1000, chunk_duration_ms=400)
        # La position 0 est dans la première frame
        ts0 = estimate(0)
        assert ts0 == 1000.0

    def test_later_positions_give_later_timestamps(self):
        data = FRAME_MARKER + b"\x00" * 20 + FRAME_MARKER + b"\x00" * 20 + FRAME_MARKER
        estimate = build_frame_estimator(data, chunk_start_ms=0, chunk_duration_ms=300)
        ts_early = estimate(0)
        ts_late = estimate(25)
        assert ts_late >= ts_early

    def test_no_frames_fallback(self):
        data = b"\x00" * 50
        estimate = build_frame_estimator(data, chunk_start_ms=500, chunk_duration_ms=1000)
        # Pas de frame → frame_idx = 0, tjs chunk_start
        ts = estimate(25)
        assert ts == 500.0


# ═══════════════════════════════════════════════════════════════════════════
# scan_fire_events — test minimal (nécessite des données réelles)
# ═══════════════════════════════════════════════════════════════════════════


class TestScanFireEvents:
    def test_empty_data_returns_empty(self):
        result = scan_fire_events(
            chunk_data=b"\x00" * 100,
            player_index=1,
            chunk_start_ms=0,
            chunk_duration_ms=1000,
        )
        assert result == []

    def test_tiny_data_returns_empty(self):
        result = scan_fire_events(
            chunk_data=b"\x00" * 10,
            player_index=1,
            chunk_start_ms=0,
            chunk_duration_ms=100,
        )
        assert result == []


# ═══════════════════════════════════════════════════════════════════════════
# correlate_kills_to_weapons
# ═══════════════════════════════════════════════════════════════════════════


class TestCorrelateKillsToWeapons:
    """Test la corrélation kill→fire event."""

    @pytest.fixture()
    def br75_bytes(self):
        return bytes.fromhex("2b1824d542c9679f")

    def test_empty_kills(self):
        assert correlate_kills_to_weapons([], []) == []

    def test_melee_kill_excluded(self):
        kills = [
            {"time_ms": 5000, "is_melee": True, "is_grenade": False},
        ]
        result = correlate_kills_to_weapons(kills, [])
        assert len(result) == 1
        assert result[0]["weapon_name"] == "MELEE (exclu)"
        assert result[0]["matched_fire_event"] is None

    def test_grenade_kill_excluded(self):
        kills = [
            {"time_ms": 5000, "is_melee": False, "is_grenade": True},
        ]
        result = correlate_kills_to_weapons(kills, [])
        assert len(result) == 1
        assert result[0]["weapon_name"] == "GRENADE (exclu)"

    def test_kill_matches_closest_fire_event(self, br75_bytes):
        kills = [
            {"time_ms": 5000, "is_melee": False, "is_grenade": False},
        ]
        fire_events = [
            {
                "timestamp_ms": 3500,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 1,
            },
            {
                "timestamp_ms": 4800,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 2,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert len(result) == 1
        assert result[0]["weapon_name"] == "BR75"
        assert result[0]["delta_ms"] == 200  # 5000 - 4800

    def test_kill_no_fire_event_in_window(self, br75_bytes):
        kills = [
            {"time_ms": 10000, "is_melee": False, "is_grenade": False},
        ]
        fire_events = [
            {
                "timestamp_ms": 2000,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "NON TROUVE"
        assert result[0]["delta_ms"] is None

    def test_fire_event_after_kill_ignored(self, br75_bytes):
        kills = [
            {"time_ms": 5000, "is_melee": False, "is_grenade": False},
        ]
        fire_events = [
            {
                "timestamp_ms": 5500,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "NON TROUVE"

    def test_multiple_kills_independent(self, br75_bytes):
        sidekick_bytes = bytes.fromhex("f408190f42c9679f")
        kills = [
            {"time_ms": 5000, "is_melee": False, "is_grenade": False},
            {"time_ms": 12000, "is_melee": False, "is_grenade": False},
        ]
        fire_events = [
            {
                "timestamp_ms": 4500,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 1,
            },
            {
                "timestamp_ms": 11500,
                "weapon_name": "Mk51 Sidekick",
                "weapon_bytes": sidekick_bytes,
                "fire_seq": 2,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "BR75"
        assert result[1]["weapon_name"] == "Mk51 Sidekick"


# ═══════════════════════════════════════════════════════════════════════════
# count_kills_by_api_weapon
# ═══════════════════════════════════════════════════════════════════════════


class TestCountKillsByApiWeapon:
    """Test l'agrégation en Counter {api_weapon_id: n_kills}."""

    @pytest.fixture()
    def br75_bytes(self):
        return bytes.fromhex("2b1824d542c9679f")

    def test_empty_correlated(self):
        assert count_kills_by_api_weapon([]) == Counter()

    def test_melee_counted_as_api_id_1(self):
        correlated = [
            {"is_melee": True, "is_grenade": False, "matched_fire_event": None},
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[MELEE_API_ID] == 1

    def test_grenade_counted_as_api_id_0(self):
        correlated = [
            {"is_melee": False, "is_grenade": True, "matched_fire_event": None},
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[GRENADE_API_ID] == 1

    def test_weapon_mapped_to_api_id(self, br75_bytes):
        correlated = [
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": br75_bytes},
            },
        ]
        counts = count_kills_by_api_weapon(correlated)
        # BR75 → API ID 41533
        assert counts[41533] == 1

    def test_unknown_weapon_ignored(self):
        unknown_bytes = b"\x00" * 8
        correlated = [
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": unknown_bytes},
            },
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert len(counts) == 0

    def test_no_fire_event_ignored(self):
        correlated = [
            {"is_melee": False, "is_grenade": False, "matched_fire_event": None},
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert len(counts) == 0

    def test_mixed_kills_aggregation(self, br75_bytes):
        correlated = [
            {"is_melee": True, "is_grenade": False, "matched_fire_event": None},
            {"is_melee": False, "is_grenade": True, "matched_fire_event": None},
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": br75_bytes},
            },
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": br75_bytes},
            },
            {"is_melee": False, "is_grenade": False, "matched_fire_event": None},
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[MELEE_API_ID] == 1
        assert counts[GRENADE_API_ID] == 1
        assert counts[41533] == 2  # 2 BR75 kills
        assert sum(counts.values()) == 4  # 5e kill ignoré (pas de fire event)
