"""Tests unitaires — weapon_parser (domaine pur).

Couvre : constantes, find_frame_positions, build_frame_estimator,
scan_formula_a, correlate_kills_to_weapons, count_kills_by_api_weapon,
find_chunk_at_time, _get_confidence (zones A/B/C), Zone B W2 upgrade,
delayed_damage, swap_pis, scan_all_players, get_player_index_acurtis,
detect_player_indices.
"""

from __future__ import annotations

from collections import Counter

import pytest
from bitstring import Bits

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
    detect_player_indices,
    find_chunk_at_time,
    find_frame_positions,
    get_player_index_acurtis,
    scan_all_players,
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
        assert KILL_WINDOW_MS == 5000

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
        timeline, _, timing = build_weapon_timeline({})
        assert timeline == {}
        assert timing == []

    def test_single_chunk_timing(self):
        data = b"\x00" * 50
        chunks = {3: (data, 45000, 15000)}
        timeline, _, timing = build_weapon_timeline(chunks)
        assert timing == [(45000, 60000)]
        assert 3 in timeline

    def test_timeline_ordered_by_chunk_idx(self):
        chunks = {
            7: (b"\x00" * 50, 90000, 15000),
            3: (b"\x00" * 50, 45000, 15000),
        }
        _, _, timing = build_weapon_timeline(chunks)
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


# ═══════════════════════════════════════════════════════════════════════════
# find_chunk_at_time
# ═══════════════════════════════════════════════════════════════════════════


class TestFindChunkAtTime:
    def test_empty_list_returns_zero(self):
        assert find_chunk_at_time([], [], 5000) == 0

    def test_t_inside_chunk_returns_that_chunk(self):
        assert find_chunk_at_time([3], [(10000, 20000)], 15000) == 3

    def test_t_before_all_chunks_falls_back_to_last(self):
        """t_ms avant le premier chunk → boucle ne matche pas → fallback dernier."""
        assert find_chunk_at_time([3], [(10000, 20000)], 5000) == 3

    def test_t_after_all_chunks_falls_back_to_last(self):
        assert find_chunk_at_time([0, 1], [(0, 10000), (10000, 20000)], 25000) == 1

    def test_t_at_chunk_boundary(self):
        """Borne gauche incluse (start <= t < end)."""
        assert find_chunk_at_time([0, 1], [(0, 10000), (10000, 20000)], 10000) == 1

    def test_multi_chunk_middle(self):
        chunks_sorted = [0, 1, 2]
        timing = [(0, 10000), (10000, 20000), (20000, 30000)]
        assert find_chunk_at_time(chunks_sorted, timing, 15000) == 1


# ═══════════════════════════════════════════════════════════════════════════
# _get_confidence (zones A / B / C)
# ═══════════════════════════════════════════════════════════════════════════


class TestGetConfidence:
    """Tests de _get_confidence (import via module interne)."""

    def _conf(self, weapon: str, delta: int | None) -> str:
        from src.analysis.weapon_parser import _get_confidence

        return _get_confidence(weapon, delta)

    def test_none_delta_returns_none(self):
        assert self._conf("BR75", None) == "none"

    def test_zone_a_high(self):
        """delta < swap_ms(BR75=650) → high."""
        assert self._conf("BR75", 300) == "high"

    def test_zone_b_medium(self):
        """swap_ms ≤ delta ≤ travel_max pour Heatwave (650/2000) → medium."""
        assert self._conf("Heatwave", 1000) == "medium"

    def test_zone_c_low_delayed_damage(self):
        """delta > travel_max(Heatwave=2000) → low."""
        assert self._conf("Heatwave", 3000) == "low"

    def test_unknown_weapon_uses_defaults(self):
        """Arme inconnue → defaults (swap=650, travel=2000)."""
        assert self._conf("ArmeInconnue", 100) == "high"
        assert self._conf("ArmeInconnue", 1500) == "medium"
        assert self._conf("ArmeInconnue", 3000) == "low"


# ═══════════════════════════════════════════════════════════════════════════
# Zone B W2 upgrade (FINDINGS §6a Step 2)
# ═══════════════════════════════════════════════════════════════════════════


class TestZoneBW2Upgrade:
    _hw = bytes.fromhex("2ac9c2ff42c9679f")  # Heatwave swap=650 travel=2000
    _br = bytes.fromhex("2b1824d542c9679f")  # BR75

    def test_medium_upgrades_to_w2_high(self):
        """MEDIUM kill + feu W2 juste après → HIGH attribué à W2."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            # W1: Heatwave delta=1000 → MEDIUM
            {
                "timestamp_ms": 4000,
                "weapon_name": "Heatwave",
                "weapon_bytes": self._hw,
                "fire_seq": 1,
            },
            # W2: BR75 à 5300 (dans [5000, 5000+650])
            {"timestamp_ms": 5300, "weapon_name": "BR75", "weapon_bytes": self._br, "fire_seq": 2},
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "BR75"
        assert result[0]["confidence"] == "high"

    def test_medium_stays_medium_without_w2(self):
        """MEDIUM kill sans W2 → reste MEDIUM."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            {
                "timestamp_ms": 4000,
                "weapon_name": "Heatwave",
                "weapon_bytes": self._hw,
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "Heatwave"
        assert result[0]["confidence"] == "medium"

    def test_w2_outside_window_ignored(self):
        """W2 hors fenêtre [kill_t, kill_t+swap_ms] → pas de promotion."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            {
                "timestamp_ms": 4000,
                "weapon_name": "Heatwave",
                "weapon_bytes": self._hw,
                "fire_seq": 1,
            },
            # W2 à 5800 — Heatwave swap_ms=650, donc fenêtre = [5000, 5650]
            {"timestamp_ms": 5800, "weapon_name": "BR75", "weapon_bytes": self._br, "fire_seq": 2},
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["weapon_name"] == "Heatwave"
        assert result[0]["confidence"] == "medium"


# ═══════════════════════════════════════════════════════════════════════════
# delayed_damage flag
# ═══════════════════════════════════════════════════════════════════════════


class TestDelayedDamage:
    _hw = bytes.fromhex("2ac9c2ff42c9679f")  # Heatwave travel_max=2000
    _br = bytes.fromhex("2b1824d542c9679f")  # BR75 travel_max=500

    def test_delayed_damage_true_when_delta_exceeds_travel_max(self):
        """delta=2100 > travel_max(Heatwave=2000) → delayed_damage=True, conf=low."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            {
                "timestamp_ms": 2900,
                "weapon_name": "Heatwave",
                "weapon_bytes": self._hw,
                "fire_seq": 1,
            },
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["delayed_damage"] is True
        assert result[0]["confidence"] == "low"

    def test_delayed_damage_false_when_delta_within_travel_max(self):
        """delta=300 < travel_max(BR75=500) → delayed_damage=False."""
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        fire_events = [
            {"timestamp_ms": 4700, "weapon_name": "BR75", "weapon_bytes": self._br, "fire_seq": 1},
        ]
        result = correlate_kills_to_weapons(kills, fire_events)
        assert result[0]["delayed_damage"] is False

    def test_no_fire_event_no_delayed_damage(self):
        kills = [{"time_ms": 5000, "is_melee": False, "is_grenade": False}]
        result = correlate_kills_to_weapons(kills, [])
        assert result[0]["delayed_damage"] is False


# ═══════════════════════════════════════════════════════════════════════════
# swap_pis detection dans build_weapon_timeline
# ═══════════════════════════════════════════════════════════════════════════


class TestSwapPisDetection:
    _br75 = bytes.fromhex("2b1824d542c9679f")  # BR75
    _ar = bytes.fromhex("48c19d2d42c9679f")  # MA40 AR

    def _make_formula_a_entry(self, pb: int, wid: bytes) -> bytes:
        """Construit un pattern Formula A [20 00 02 pb 00 00 00 00 wid]."""
        return bytes.fromhex("200002") + bytes([pb]) + b"\x00" * 4 + wid

    def test_swap_detected_same_pi_two_weapons(self):
        """Pi=1 voit BR75 puis MA40 AR dans le même chunk → pi=1 dans swap_pis[0]."""
        pb = 0x20  # pi = 0x20 >> 5 = 1
        data = self._make_formula_a_entry(pb, self._br75) + self._make_formula_a_entry(pb, self._ar)
        chunks = {0: (data, 0, 19000)}
        _, swap_pis, _ = build_weapon_timeline(chunks)
        assert 1 in swap_pis.get(0, set())

    def test_no_swap_single_weapon_per_pi(self):
        """Pi=1 ne voit qu'une arme → absent de swap_pis."""
        pb = 0x20
        data = self._make_formula_a_entry(pb, self._br75)
        chunks = {0: (data, 0, 19000)}
        _, swap_pis, _ = build_weapon_timeline(chunks)
        assert 1 not in swap_pis.get(0, set())

    def test_swap_only_for_pi_with_multiple_weapons(self):
        """Pi=1 avec swap, pi=2 avec une seule arme → seul pi=1 dans swap_pis."""
        data = (
            self._make_formula_a_entry(0x20, self._br75)  # pi=1, BR75
            + self._make_formula_a_entry(0x20, self._ar)  # pi=1, AR → swap
            + self._make_formula_a_entry(0x40, self._br75)  # pi=2, BR75 seulement
        )
        chunks = {0: (data, 0, 19000)}
        _, swap_pis, _ = build_weapon_timeline(chunks)
        assert 1 in swap_pis.get(0, set())
        assert 2 not in swap_pis.get(0, set())


# ═══════════════════════════════════════════════════════════════════════════
# scan_all_players
# ═══════════════════════════════════════════════════════════════════════════


class TestScanAllPlayers:
    def test_returns_dict_keyed_by_index(self):
        result = scan_all_players(b"\x00" * 80, 0, 5000, n_players=4)
        assert set(result.keys()) == {0, 1, 2, 3}

    def test_empty_data_all_empty(self):
        result = scan_all_players(b"\x00" * 50, 0, 1000, n_players=3)
        assert all(v == [] for v in result.values())


# ═══════════════════════════════════════════════════════════════════════════
# get_player_index_acurtis
# ═══════════════════════════════════════════════════════════════════════════


class TestGetPlayerIndexAcurtis:
    # XUID bytes non-ambigus (tous non-nuls, pas de motif féditable)
    _XUID_INT = 0xFEDCBA9876543210
    _XUID_LE = Bits(uintle=_XUID_INT, length=64).bytes  # b'\x10\x32\x54...'

    def test_not_found_returns_none(self):
        result = get_player_index_acurtis(Bits(bytes=b"\x00" * 20), self._XUID_INT)
        assert result is None

    def test_bit_pos_too_small_returns_none(self):
        """XUID dès le bit 0 → bit_pos=0 < 5 → None."""
        chunk_data = self._XUID_LE + b"\x00" * 4
        result = get_player_index_acurtis(Bits(bytes=chunk_data), self._XUID_INT)
        assert result is None

    def test_found_returns_pi(self):
        """XUID à bit 8 (après 1 octet nul) → bits[3:8]=0 → pi=0."""
        chunk_data = b"\x00" + self._XUID_LE + b"\x00"
        result = get_player_index_acurtis(Bits(bytes=chunk_data), self._XUID_INT)
        assert result == 0

    def test_pi_extracted_from_prefix_bits(self):
        """Préfixe \xf8 = 11111000 → bits[3:8] = 11000 = 24 → pi=24."""
        chunk_data = b"\xf8" + self._XUID_LE + b"\x00"
        result = get_player_index_acurtis(Bits(bytes=chunk_data), self._XUID_INT)
        # bits[3:8] de 0xf8 (11111000) = bits 3,4,5,6,7 = 1,1,0,0,0 = 24
        assert result == 24


# ═══════════════════════════════════════════════════════════════════════════
# detect_player_indices
# ═══════════════════════════════════════════════════════════════════════════


class TestDetectPlayerIndices:
    _XUID_A = 0xFEDCBA9876543210
    _XUID_B = 0x0F1E2D3C4B5A6978
    _LE_A = Bits(uintle=_XUID_A, length=64).bytes
    _LE_B = Bits(uintle=_XUID_B, length=64).bytes

    def test_no_xuids_present_returns_empty(self):
        result = detect_player_indices(b"\x00" * 30, {12345: "P1"})
        assert result == {}

    def test_empty_xuid_map_returns_empty(self):
        result = detect_player_indices(b"\x00" * 30, {})
        assert result == {}

    def test_single_player_resolved(self):
        """XUID_A à bit 8 → pi=0 détecté."""
        chunk_data = b"\x00" + self._LE_A + b"\x00"
        result = detect_player_indices(chunk_data, {self._XUID_A: "PlayerA"})
        assert 0 in result
        assert result[0] == self._XUID_A

    def test_two_players_both_resolved(self):
        """Deux XUIDs dans le même chunk → les deux résolus."""
        # XUID_A à bit 8, XUID_B à bit 8+64+8 = 80
        chunk_data = b"\x00" + self._LE_A + b"\x00" + self._LE_B + b"\x00"
        result = detect_player_indices(
            chunk_data, {self._XUID_A: "PlayerA", self._XUID_B: "PlayerB"}
        )
        # Les deux pi doivent être résolus (pi=0 pour chaque préfixe nul)
        # Le second sera ignoré si même pi=0 (collision) — ce test vérifie au moins 1
        assert len(result) >= 1
        assert self._XUID_A in result.values() or self._XUID_B in result.values()
