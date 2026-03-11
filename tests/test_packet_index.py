"""Tests unitaires — packet_index (index structurel REPLICATION_DATA).

Couvre : PacketType enum, index_chunk, build_packet_estimator,
extract_metadata_payload, detect_pi_from_metadata.
"""

from __future__ import annotations

import struct

import pytest

from src.analysis.packet_index import (
    HEADER_STRUCT,
    Packet,
    PacketType,
    build_packet_estimator,
    detect_pi_from_metadata,
    extract_metadata_payload,
    index_chunk,
)


# ═══════════════════════════════════════════════════════════════════════════
# Helpers
# ═══════════════════════════════════════════════════════════════════════════


def _make_packet_bytes(
    pkt_type: int, size: int, microseconds: int, payload: bytes = b"",
) -> bytes:
    """Construit un paquet (header 16B + payload)."""
    header = HEADER_STRUCT.pack(pkt_type, 0, 0, size, microseconds)
    # Pad payload to exact size
    padded = payload.ljust(size, b"\x00")[:size]
    return header + padded


def _make_chunk(packets_spec: list[tuple[int, int, int, bytes]]) -> bytes:
    """Construit un chunk complet depuis une liste de (type, size, µs, payload)."""
    parts = [_make_packet_bytes(t, s, us, p) for t, s, us, p in packets_spec]
    return b"".join(parts)


# ═══════════════════════════════════════════════════════════════════════════
# PacketType enum
# ═══════════════════════════════════════════════════════════════════════════


class TestPacketType:
    def test_frame_is_zero(self):
        assert PacketType.FRAME == 0

    def test_end_chunk_is_seven(self):
        assert PacketType.END_CHUNK == 7

    def test_player_metadata_is_eight(self):
        assert PacketType.PLAYER_METADATA == 8

    def test_all_values_unique(self):
        values = [e.value for e in PacketType]
        assert len(values) == len(set(values))


# ═══════════════════════════════════════════════════════════════════════════
# HEADER_STRUCT
# ═══════════════════════════════════════════════════════════════════════════


class TestHeaderStruct:
    def test_size_is_16(self):
        assert HEADER_STRUCT.size == 16

    def test_roundtrip(self):
        packed = HEADER_STRUCT.pack(0, 1, 2, 258, 123456789)
        unpacked = HEADER_STRUCT.unpack(packed)
        assert unpacked == (0, 1, 2, 258, 123456789)


# ═══════════════════════════════════════════════════════════════════════════
# index_chunk
# ═══════════════════════════════════════════════════════════════════════════


class TestIndexChunk:
    def test_empty_data(self):
        assert index_chunk(b"") == []

    def test_too_short(self):
        assert index_chunk(b"\x00" * 15) == []

    def test_single_end_chunk(self):
        data = _make_packet_bytes(PacketType.END_CHUNK, 0, 1000)
        packets = index_chunk(data)
        assert len(packets) == 1
        assert packets[0].type == PacketType.END_CHUNK

    def test_minimal_chunk_structure(self):
        """START_CHUNK → FRAME → END_CHUNK."""
        data = _make_chunk([
            (PacketType.START_CHUNK, 4, 100, b"\x00" * 4),
            (PacketType.FRAME, 16, 200, b"\xff" * 16),
            (PacketType.END_CHUNK, 0, 300, b""),
        ])
        packets = index_chunk(data)
        assert len(packets) == 3
        assert packets[0].type == PacketType.START_CHUNK
        assert packets[1].type == PacketType.FRAME
        assert packets[2].type == PacketType.END_CHUNK

    def test_packet_offsets_correct(self):
        data = _make_chunk([
            (PacketType.START_CHUNK, 10, 100, b"\x00" * 10),
            (PacketType.FRAME, 20, 200, b"\xff" * 20),
            (PacketType.END_CHUNK, 0, 300, b""),
        ])
        packets = index_chunk(data)
        assert packets[0].offset == 16       # header size
        assert packets[1].offset == 16 + 10 + 16  # prev header + payload + this header
        assert packets[2].offset == 16 + 10 + 16 + 20 + 16

    def test_microseconds_preserved(self):
        data = _make_chunk([
            (PacketType.FRAME, 4, 999_888_777, b"\x00" * 4),
            (PacketType.END_CHUNK, 0, 999_888_800, b""),
        ])
        packets = index_chunk(data)
        assert packets[0].microseconds == 999_888_777

    def test_stops_at_end_chunk(self):
        """Data after END_CHUNK is ignored."""
        chunk = _make_chunk([
            (PacketType.FRAME, 4, 100, b"\x00" * 4),
            (PacketType.END_CHUNK, 0, 200, b""),
        ])
        data = chunk + b"\x00" * 100  # trailing garbage
        packets = index_chunk(data)
        assert len(packets) == 2

    def test_realistic_types(self):
        """Simule la structure réelle : START → TYPE_2 → METADATA → FRAME×3 → END."""
        data = _make_chunk([
            (PacketType.START_CHUNK, 4, 1000, b"\x00" * 4),
            (PacketType.INIT_STATE, 100, 1000, b"\xab" * 100),
            (PacketType.PLAYER_METADATA, 50, 1000, b"\xcd" * 50),
            (PacketType.FRAME, 32, 2000, b"\x01" * 32),
            (PacketType.FRAME, 32, 18000, b"\x02" * 32),
            (PacketType.FRAME, 32, 34000, b"\x03" * 32),
            (PacketType.END_CHUNK, 0, 50000, b""),
        ])
        packets = index_chunk(data)
        assert len(packets) == 7
        types = [p.type for p in packets]
        assert types == [1, 2, 8, 0, 0, 0, 7]


# ═══════════════════════════════════════════════════════════════════════════
# build_packet_estimator
# ═══════════════════════════════════════════════════════════════════════════


class TestBuildPacketEstimator:
    def test_no_frames_returns_chunk_start(self):
        packets = [Packet(type=PacketType.END_CHUNK, size=0, microseconds=100, offset=16)]
        est = build_packet_estimator(packets, chunk_start_ms=5000.0)
        assert est(0) == 5000.0

    def test_single_frame(self):
        packets = [
            Packet(type=PacketType.FRAME, size=100, microseconds=1_000_000, offset=16),
            Packet(type=PacketType.END_CHUNK, size=0, microseconds=1_100_000, offset=132),
        ]
        est = build_packet_estimator(packets, chunk_start_ms=0.0)
        # byte_pos within the frame → timestamp = 0.0 + (1_000_000 - 1_000_000)/1000 = 0.0
        assert est(50) == 0.0

    def test_two_frames_progression(self):
        # Frame1 at µs=0, Frame2 at µs=16000 (16ms later)
        packets = [
            Packet(type=PacketType.FRAME, size=100, microseconds=0, offset=16),
            Packet(type=PacketType.FRAME, size=100, microseconds=16000, offset=132),
            Packet(type=PacketType.END_CHUNK, size=0, microseconds=32000, offset=248),
        ]
        est = build_packet_estimator(packets, chunk_start_ms=1000.0)
        # byte_pos in frame 1 range → 1000.0 + 0/1000 = 1000.0
        assert est(20) == 1000.0
        # byte_pos in frame 2 range → 1000.0 + 16000/1000 = 1016.0
        assert est(140) == 1016.0

    def test_later_position_gives_later_time(self):
        packets = [
            Packet(type=PacketType.FRAME, size=50, microseconds=0, offset=16),
            Packet(type=PacketType.FRAME, size=50, microseconds=16000, offset=82),
            Packet(type=PacketType.FRAME, size=50, microseconds=32000, offset=148),
            Packet(type=PacketType.END_CHUNK, size=0, microseconds=48000, offset=214),
        ]
        est = build_packet_estimator(packets, chunk_start_ms=0.0)
        t1 = est(20)   # in frame 1
        t2 = est(90)   # in frame 2
        t3 = est(160)  # in frame 3
        assert t1 < t2 < t3

    def test_non_frame_packets_ignored(self):
        packets = [
            Packet(type=PacketType.INIT_STATE, size=1000, microseconds=0, offset=16),
            Packet(type=PacketType.FRAME, size=50, microseconds=5000, offset=1032),
            Packet(type=PacketType.END_CHUNK, size=0, microseconds=10000, offset=1098),
        ]
        est = build_packet_estimator(packets, chunk_start_ms=100.0)
        # byte_pos before the FRAME → should still map to frame 0 (first frame found)
        assert est(1040) == 100.0  # base_us = 5000, (5000-5000)/1000 = 0


# ═══════════════════════════════════════════════════════════════════════════
# extract_metadata_payload
# ═══════════════════════════════════════════════════════════════════════════


class TestExtractMetadataPayload:
    def test_no_metadata_returns_none(self):
        data = _make_chunk([
            (PacketType.FRAME, 10, 0, b"\x00" * 10),
            (PacketType.END_CHUNK, 0, 100, b""),
        ])
        packets = index_chunk(data)
        assert extract_metadata_payload(data, packets) is None

    def test_extracts_correct_payload(self):
        metadata_content = b"\xAB\xCD\xEF" * 10  # 30 bytes
        data = _make_chunk([
            (PacketType.START_CHUNK, 4, 0, b"\x00" * 4),
            (PacketType.PLAYER_METADATA, 30, 0, metadata_content),
            (PacketType.END_CHUNK, 0, 100, b""),
        ])
        packets = index_chunk(data)
        payload = extract_metadata_payload(data, packets)
        assert payload == metadata_content

    def test_returns_first_metadata(self):
        """Si multiple METADATA (théoriquement impossible), retourne le premier."""
        data = _make_chunk([
            (PacketType.PLAYER_METADATA, 4, 0, b"\x01\x02\x03\x04"),
            (PacketType.PLAYER_METADATA, 4, 100, b"\x05\x06\x07\x08"),
            (PacketType.END_CHUNK, 0, 200, b""),
        ])
        packets = index_chunk(data)
        payload = extract_metadata_payload(data, packets)
        assert payload == b"\x01\x02\x03\x04"


# ═══════════════════════════════════════════════════════════════════════════
# detect_pi_from_metadata
# ═══════════════════════════════════════════════════════════════════════════


class TestDetectPiFromMetadata:
    @staticmethod
    def _embed_xuid_with_pi(xuid_int: int, pi: int) -> bytes:
        """Crée un fragment de metadata avec un XUID précédé de 5 bits de pi."""
        from bitstring import Bits

        pi_bits = Bits(uint=pi, length=5)
        xuid_bits = Bits(uintle=xuid_int, length=64)
        combined = pi_bits + xuid_bits
        # Align to bytes
        pad = 8 - (len(combined) % 8) if len(combined) % 8 != 0 else 0
        return (Bits(uint=0, length=pad) + combined).bytes

    def test_empty_payload(self):
        result = detect_pi_from_metadata(b"", {12345: "player1"})
        assert result == {}

    def test_xuid_not_found(self):
        result = detect_pi_from_metadata(b"\x00" * 100, {12345: "player1"})
        assert result == {}

    def test_single_player_non_zero_pi(self):
        xuid = 0x0009_1234_5678_ABCD
        payload = b"\x00" * 50 + self._embed_xuid_with_pi(xuid, 3) + b"\x00" * 50
        result = detect_pi_from_metadata(payload, {xuid: "player1"})
        assert result == {3: xuid}

    def test_pi_zero_skipped(self):
        """Le POV player (pi=0) n'est pas retourné."""
        xuid = 0x0009_1234_5678_ABCD
        payload = b"\x00" * 50 + self._embed_xuid_with_pi(xuid, 0) + b"\x00" * 50
        result = detect_pi_from_metadata(payload, {xuid: "player1"})
        assert result == {}

    def test_multiple_players(self):
        xuid1 = 0x0009_AAAA_BBBB_CCCC
        xuid2 = 0x0009_DDDD_EEEE_FFFF
        payload = (
            b"\x00" * 30
            + self._embed_xuid_with_pi(xuid1, 2)
            + b"\x00" * 30
            + self._embed_xuid_with_pi(xuid2, 5)
            + b"\x00" * 30
        )
        result = detect_pi_from_metadata(payload, {xuid1: "p1", xuid2: "p2"})
        assert result == {2: xuid1, 5: xuid2}

    def test_no_pi_collision(self):
        """Deux XUIDs ne peuvent pas avoir le même pi."""
        xuid1 = 0x0009_1111_2222_3333
        xuid2 = 0x0009_4444_5555_6666
        # Both embedded with pi=3 — second should not be returned
        payload = (
            b"\x00" * 30
            + self._embed_xuid_with_pi(xuid1, 3)
            + b"\x00" * 30
            + self._embed_xuid_with_pi(xuid2, 3)
            + b"\x00" * 30
        )
        result = detect_pi_from_metadata(payload, {xuid1: "p1", xuid2: "p2"})
        # Only first one (pi=3 → xuid1) should be kept
        assert 3 in result
        assert result[3] == xuid1


# ═══════════════════════════════════════════════════════════════════════════
# scan_fire_events avec packets (intégration)
# ═══════════════════════════════════════════════════════════════════════════


class TestScanFireEventsWithPackets:
    """Vérifie que scan_fire_events accepte le kwarg packets."""

    def test_packets_none_falls_back(self):
        """packets=None → comportement identique à l'ancien code."""
        from src.analysis.weapon_parser import scan_fire_events

        # Données vides → aucun fire event
        result = scan_fire_events(b"\x00" * 100, 1, 0, 1000, packets=None)
        assert result == []

    def test_packets_provided_uses_index(self):
        """packets fourni → utilise build_packet_estimator."""
        from src.analysis.weapon_parser import scan_fire_events

        packets = [
            Packet(type=PacketType.FRAME, size=100, microseconds=0, offset=16),
            Packet(type=PacketType.END_CHUNK, size=0, microseconds=100000, offset=132),
        ]
        # Données vides → aucun fire event, mais ne doit pas crasher
        result = scan_fire_events(b"\x00" * 200, 1, 0, 1000, packets=packets)
        assert result == []
