"""Tests unitaires — weapon_parser (domaine pur).

Couvre : constantes, find_frame_positions, build_frame_estimator,
scan_formula_a, correlate_kills_to_weapons, count_kills_by_api_weapon.
"""

from __future__ import annotations

from collections import Counter

import pytest

from src.analysis.weapon_parser import (
    COMMON_WEAPON_SUFFIX,
    FORMULA_A_PATTERN,
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
    WEAPON_TIMING,
    build_frame_estimator,
    build_weapon_timeline,
    correlate_kills_to_weapons,
    count_kills_by_api_weapon,
    find_frame_positions,
    scan_fire_events,
    scan_formula_a,
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

    def test_weapon_timing_present(self):
        assert "BR75" in WEAPON_TIMING
        assert "Mk51 Sidekick" in WEAPON_TIMING
        assert "Gravity Hammer" in WEAPON_TIMING

    def test_weapon_timing_tuples(self):
        for name, (swap, travel) in WEAPON_TIMING.items():
            assert isinstance(swap, int), f"{name}: swap non entier"
            assert isinstance(travel, int), f"{name}: travel non entier"
            assert swap > 0, f"{name}: swap <= 0"
            assert travel > 0, f"{name}: travel <= 0"

    def test_formula_a_pattern_correct(self):
        assert bytes.fromhex("200002") == FORMULA_A_PATTERN

    def test_confirmed_weapon_ids(self):
        """IDs confirmés liste maître présents dans le map."""
        confirmed = [
            ("6acdc44d42c9679f", "Bandit Evo"),
            ("2b1824d542c9679f", "BR75"),
            ("841ac5e542c9679f", "Gravity Hammer"),
            ("4ff3937e42c9679f", "Energy Sword"),
            ("4ff3937e8978aa7a", "Duelist Energy Sword"),
            ("841ac5e5a730e49f", "Diminisher of Hope"),
            ("841ac5e5d8d07ca1", "Rushdown Hammer"),
            ("f408190f42c9679f", "Mk51 Sidekick"),
        ]
        for hex_id, expected_name in confirmed:
            key = bytes.fromhex(hex_id)
            assert key in WEAPON_ID_MAP, f"{hex_id} absent du map"
            assert (
                WEAPON_ID_MAP[key] == expected_name
            ), f"{hex_id}: attendu {expected_name!r}, got {WEAPON_ID_MAP[key]!r}"


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
        data = (
            FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
            + b"\x00" * 10
            + FRAME_MARKER
        )
        estimate = build_frame_estimator(data, chunk_start_ms=1000, chunk_duration_ms=400)
        ts0 = estimate(0)
        assert ts0 == 1000.0

    def test_later_positions_give_later_timestamps(self):
        data = FRAME_MARKER + b"\x00" * 20 + FRAME_MARKER + b"\x00" * 20 + FRAME_MARKER
        estimate = build_frame_estimator(data, chunk_start_ms=0, chunk_duration_ms=300)
        assert estimate(25) >= estimate(0)

    def test_no_frames_fallback(self):
        data = b"\x00" * 50
        estimate = build_frame_estimator(data, chunk_start_ms=500, chunk_duration_ms=1000)
        assert estimate(25) == 500.0


# ═══════════════════════════════════════════════════════════════════════════
# scan_formula_a
# ═══════════════════════════════════════════════════════════════════════════


class TestScanFormulaA:
    def test_empty_data_returns_empty(self):
        assert scan_formula_a(b"") == []

    def test_no_pattern_returns_empty(self):
        assert scan_formula_a(b"\x00" * 100) == []

    def test_detects_standard_suffix(self):
        """Un pattern [20 00 02 pb ... wid:8B] avec suffixe standard est détecté."""
        # pb = 0xa0 → pi = 0xa0 >> 5 = 5
        wid = bytes.fromhex("2b1824d542c9679f")  # BR75
        data = bytes.fromhex("200002") + b"\xa0" + b"\x00" * 4 + wid
        results = scan_formula_a(data)
        assert len(results) >= 1
        offsets, pis, wids = zip(*results, strict=False)
        assert 5 in pis
        assert wid in wids

    def test_pi_extracted_correctly(self):
        """pi = pb >> 5, donc pb=0x20 → pi=1."""
        wid = bytes.fromhex("f408190f42c9679f")  # Sidekick
        data = bytes.fromhex("200002") + b"\x20" + b"\x00" * 4 + wid
        results = scan_formula_a(data)
        assert any(pi == 1 for _, pi, _ in results)


# ═══════════════════════════════════════════════════════════════════════════
# build_weapon_timeline
# ═══════════════════════════════════════════════════════════════════════════


class TestBuildWeaponTimeline:
    def test_empty_chunks_returns_empty(self):
        timeline, timing = build_weapon_timeline({})
        assert timeline == {}
        assert timing == []

    def test_single_chunk_timing(self):
        data = b"\x00" * 50
        chunks = {3: (data, 45000, 15000)}
        timeline, timing = build_weapon_timeline(chunks)
        assert timing == [(45000, 60000)]
        assert 3 in timeline

    def test_timeline_ordered_by_chunk_idx(self):
        chunks = {
            7: (b"\x00" * 50, 90000, 15000),
            3: (b"\x00" * 50, 45000, 15000),
        }
        _, timing = build_weapon_timeline(chunks)
        # chunk 3 avant chunk 7
        assert timing[0][0] == 45000
        assert timing[1][0] == 90000


# ═══════════════════════════════════════════════════════════════════════════
# scan_fire_events — test minimal
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
    @pytest.fixture()
    def br75_bytes(self):
        return bytes.fromhex("2b1824d542c9679f")

    def test_empty_kills(self):
        assert correlate_kills_to_weapons([], []) == []

    def test_melee_kill_excluded(self):
        kills = [{"time_ms": 5000, "is_melee": True, "is_grenade": False}]
        result = correlate_kills_to_weapons(kills, [])
        assert result[0]["weapon_name"] == "MELEE"
        assert result[0]["matched_fire_event"] is None
        assert result[0]["confidence"] == "none"

    def test_grenade_kill_excluded(self):
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": True}]
        result = correlate_kills_to_weapons(kills, [])
        assert result[0]["weapon_name"] == "GRENADE"
        assert result[0]["confidence"] == "none"

    def test_kill_matches_closest_fire_event(self, br75_bytes):
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
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
        assert result[0]["weapon_name"] == "BR75"
        assert result[0]["delta_ms"] == 200  # 5000 - 4800

    def test_kill_no_fire_event_returns_not_found(self, br75_bytes):
        kills = [{"time_ms": 10000, "is_melee": False, "is_grenade": False}]
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
        assert result[0]["confidence"] == "none"

    def test_confidence_high_for_small_delta(self, br75_bytes):
        """delta < swap_ms(BR75=650) → HIGH."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            {
                "timestamp_ms": 4700,
                "weapon_name": "BR75",
                "weapon_bytes": br75_bytes,
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["confidence"] == "high"
        assert result[0]["delta_ms"] == 300

    def test_confidence_medium_for_ambiguous_delta(self, br75_bytes):
        """swap_ms ≤ delta ≤ travel_max → MEDIUM."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            # delta = 5000 - 4200 = 800ms → > 650 swap_ms, < 500 travel_max? Non → low
            # Pour MEDIUM: swap_ms(650) ≤ delta ≤ travel_max(500)… impossible pour BR75
            # Utilisons Heatwave: swap=650, travel=2000 → delta=1000 → MEDIUM
            {
                "timestamp_ms": 4000,
                "weapon_name": "Heatwave",
                "weapon_bytes": bytes.fromhex("2ac9c2ff42c9679f"),
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["confidence"] == "medium"

    def test_fire_event_after_kill_ignored(self, br75_bytes):
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
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
# count_kills_by_api_weapon (alias de count_kills_by_film_weapon)
# ═══════════════════════════════════════════════════════════════════════════


class TestCountKillsByApiWeapon:
    @pytest.fixture()
    def br75_bytes(self):
        return bytes.fromhex("2b1824d542c9679f")

    def test_empty_correlated(self):
        assert count_kills_by_api_weapon([]) == Counter()

    def test_melee_counted_as_api_id_1(self):
        correlated = [{"is_melee": True, "is_grenade": False, "matched_fire_event": None}]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[MELEE_API_ID] == 1

    def test_grenade_counted_as_api_id_0(self):
        correlated = [{"is_melee": False, "is_grenade": True, "matched_fire_event": None}]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[GRENADE_API_ID] == 1

    def test_weapon_mapped_to_film_uint64(self, br75_bytes):
        """BR75 → uint64 big-endian de ses 8 bytes film."""
        br75_uint64 = int.from_bytes(br75_bytes, "big")
        correlated = [
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": br75_bytes},
            },
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert counts[br75_uint64] == 1

    def test_unknown_weapon_ignored(self):
        correlated = [
            {
                "is_melee": False,
                "is_grenade": False,
                "matched_fire_event": {"weapon_bytes": b"\x00" * 8},
            },
        ]
        counts = count_kills_by_api_weapon(correlated)
        assert len(counts) == 0

    def test_no_fire_event_ignored(self):
        correlated = [{"is_melee": False, "is_grenade": False, "matched_fire_event": None}]
        assert len(count_kills_by_api_weapon(correlated)) == 0

    def test_mixed_kills_aggregation(self, br75_bytes):
        br75_uint64 = int.from_bytes(br75_bytes, "big")
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
        assert counts[br75_uint64] == 2
        assert sum(counts.values()) == 4
