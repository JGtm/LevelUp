"""Tests unitaires — scan_fire_events_b5 (player_index = b5>>4, inv134).

Valide que :
  - le marker universel détecte les fire events de tous les joueurs
  - player_index = b5 >> 4 est extrait correctement
  - slot = b5 & 0x03 est extrait correctement
  - le filtre weapon (WEAPON_IDS_INT ou COMMON_WEAPON_SUFFIX) fonctionne
  - la déduplication par byte_pos proximity est correcte
  - tous les champs du dict résultant sont présents
"""

from __future__ import annotations

from bitstring import BitArray, Bits

from src.analysis._weapon_scanners import (
    _UNIVERSAL_MARKER,
    scan_fire_events_b5,
)

# Armes connues avec COMMON_WEAPON_SUFFIX = 42c9679f
BR75_BYTES = bytes.fromhex("2b1824d542c9679f")
SIDEKICK_BYTES = bytes.fromhex("f408190f42c9679f")

_TS_CONST: float = 1000.0
_TS_FN_CONST = lambda bp: _TS_CONST  # noqa: E731
_TS_FN_POS = lambda bp: float(bp) * 10  # noqa: E731


def _make_event_chunk(
    pi: int,
    weapon_bytes: bytes = BR75_BYTES,
    fire_seq: int = 1,
    fire_counter: int = 5,
    slot: int = 1,
    prefix_bits: int = 5,
) -> bytes:
    """Construit un chunk binaire minimal contenant un fire event valide.

    Layout depuis le début du BitArray :
      [prefix_bits zeros] [11b marker] [8b fire_seq] [8b b3=0x40]
      [8b fire_counter] [8b b5=(pi<<4)|slot] [64b weapon] [32b post] [pad]

    L'invariant event_start = marker_pos + 3 est respecté :
      b5_start  = marker_pos + 3 + 32 = marker_pos + 35
      wid_start = marker_pos + 3 + 40 = marker_pos + 43
    """
    ba = BitArray()
    ba.append(Bits(uint=0, length=prefix_bits))
    ba.append(_UNIVERSAL_MARKER)                            # 11 bits
    ba.append(Bits(uint=fire_seq, length=8))                # [event_start+8]
    ba.append(Bits(uint=0x40, length=8))                    # [event_start+16] b3
    ba.append(Bits(uint=fire_counter, length=8))            # [event_start+24] b4
    ba.append(Bits(uint=(pi << 4) | slot, length=8))        # [event_start+32] b5
    ba.append(Bits(bytes=weapon_bytes))                     # [event_start+40] weapon
    ba.append(Bits(uint=0, length=32))                      # post_bytes
    pad = (8 - len(ba) % 8) % 8
    if pad:
        ba.append(Bits(uint=0, length=pad))
    return ba.bytes


# ══════════════════════════════════════════════════════════════════════
# Groupe A — Cas de base / vide
# ══════════════════════════════════════════════════════════════════════


class TestScanFireEventsB5Empty:
    def test_empty_bytes_returns_empty(self):
        assert scan_fire_events_b5(b"", _TS_FN_CONST) == []

    def test_zero_chunk_returns_empty(self):
        assert scan_fire_events_b5(b"\x00" * 32, _TS_FN_CONST) == []

    def test_truncated_chunk_no_crash(self):
        """Un chunk trop court pour contenir weapon+post ne doit pas crasher."""
        chunk = _make_event_chunk(pi=1)[:8]
        result = scan_fire_events_b5(chunk, _TS_FN_CONST)
        assert isinstance(result, list)


# ══════════════════════════════════════════════════════════════════════
# Groupe B — player_index et slot depuis b5
# ══════════════════════════════════════════════════════════════════════


class TestScanFireEventsB5PlayerIndex:
    def test_pi_0(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=0), _TS_FN_CONST)
        assert len(events) == 1
        assert events[0]["player_index"] == 0

    def test_pi_1(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=1), _TS_FN_CONST)
        assert events[0]["player_index"] == 1

    def test_pi_7(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=7), _TS_FN_CONST)
        assert events[0]["player_index"] == 7

    def test_pi_8(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=8), _TS_FN_CONST)
        assert events[0]["player_index"] == 8

    def test_pi_15(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=15), _TS_FN_CONST)
        assert events[0]["player_index"] == 15

    def test_slot_primary(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=2, slot=1), _TS_FN_CONST)
        assert events[0]["slot"] == 1

    def test_slot_secondary(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=2, slot=3), _TS_FN_CONST)
        assert events[0]["slot"] == 3

    def test_b5_encodes_pi_and_slot_coherently(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=5, slot=3), _TS_FN_CONST)
        ev = events[0]
        assert ev["b5"] == (5 << 4) | 3
        assert ev["player_index"] == 5
        assert ev["slot"] == 3


# ══════════════════════════════════════════════════════════════════════
# Groupe C — Filtre weapon
# ══════════════════════════════════════════════════════════════════════


class TestScanFireEventsB5WeaponFilter:
    def test_br75_accepted(self):
        assert len(scan_fire_events_b5(_make_event_chunk(pi=1, weapon_bytes=BR75_BYTES), _TS_FN_CONST)) == 1

    def test_sidekick_accepted(self):
        assert len(scan_fire_events_b5(_make_event_chunk(pi=1, weapon_bytes=SIDEKICK_BYTES), _TS_FN_CONST)) == 1

    def test_unknown_weapon_without_suffix_rejected(self):
        """Weapon inconnue sans COMMON_WEAPON_SUFFIX = filtré."""
        bad = bytes.fromhex("0102030405060708")  # suffix 05060708 ≠ 42c9679f
        assert scan_fire_events_b5(_make_event_chunk(pi=1, weapon_bytes=bad), _TS_FN_CONST) == []

    def test_weapon_bytes_in_output(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=1, weapon_bytes=SIDEKICK_BYTES), _TS_FN_CONST)
        assert events[0]["weapon_bytes"] == SIDEKICK_BYTES


# ══════════════════════════════════════════════════════════════════════
# Groupe D — Champs du dict résultant
# ══════════════════════════════════════════════════════════════════════


class TestScanFireEventsB5Fields:
    def _event(self, **kw):
        return scan_fire_events_b5(_make_event_chunk(**kw), _TS_FN_CONST)[0]

    def test_all_expected_keys_present(self):
        ev = self._event(pi=1)
        expected = {
            "timestamp_ms", "player_index", "slot", "b5", "weapon_name",
            "weapon_bytes", "fire_seq", "fire_counter", "byte_pos",
            "post_bytes", "burst_end", "hit_likely",
        }
        assert expected.issubset(ev.keys())

    def test_fire_seq_field(self):
        assert self._event(pi=1, fire_seq=77)["fire_seq"] == 77

    def test_fire_counter_field(self):
        assert self._event(pi=1, fire_counter=200)["fire_counter"] == 200

    def test_timestamp_from_estimator(self):
        events = scan_fire_events_b5(_make_event_chunk(pi=1), lambda bp: 42.5)
        assert events[0]["timestamp_ms"] == 42.5

    def test_byte_pos_is_int(self):
        assert isinstance(self._event(pi=1)["byte_pos"], int)

    def test_burst_end_is_bool(self):
        assert isinstance(self._event(pi=1)["burst_end"], bool)


# ══════════════════════════════════════════════════════════════════════
# Groupe E — Déduplication et tri
# ══════════════════════════════════════════════════════════════════════


class TestScanFireEventsB5Dedup:
    def test_single_event_kept(self):
        assert len(scan_fire_events_b5(_make_event_chunk(pi=3), _TS_FN_CONST)) == 1

    def test_two_events_far_apart_both_kept(self):
        """Deux events séparés par >>2 bytes → aucune dédup ne les élimine."""
        e1 = _make_event_chunk(pi=1)
        e2 = _make_event_chunk(pi=4)
        # 0xff*30 : aucun pattern marker possible (all-ones ≠ marker)
        chunk = e1 + b"\xff" * 30 + e2
        events = scan_fire_events_b5(chunk, _TS_FN_CONST)
        pis = {ev["player_index"] for ev in events}
        assert 1 in pis
        assert 4 in pis

    def test_sorted_by_timestamp_ascending(self):
        """Résultat trié par timestamp_ms croissant."""
        e1 = _make_event_chunk(pi=1)
        e2 = _make_event_chunk(pi=4)
        chunk = e1 + b"\xff" * 30 + e2
        events = scan_fire_events_b5(chunk, _TS_FN_POS)
        ts = [ev["timestamp_ms"] for ev in events]
        assert ts == sorted(ts)

    def test_multiple_pi_values_all_returned(self):
        """Plusieurs pi distincts dans un chunk → tous retournés."""
        chunks = b"".join(_make_event_chunk(pi=pi) + b"\xff" * 20 for pi in [0, 2, 5, 7])
        events = scan_fire_events_b5(chunks, _TS_FN_CONST)
        found_pis = {ev["player_index"] for ev in events}
        assert {0, 2, 5, 7}.issubset(found_pis)
