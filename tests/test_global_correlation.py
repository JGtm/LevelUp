"""Tests pour _global_correlation.py — claim-and-remove global."""

from __future__ import annotations

from src.analysis._global_correlation import (
    _attribution_from_event,
    _make_sentinel_global,
    correlate_kills_global,
)
from src.analysis._kill_attribution import KillAttribution
from src.analysis._parser_logging import MatchLogCollector
from src.analysis._weapon_data import (
    GRENADE_WEAPON_ID,
    MELEE_WEAPON_ID,
    WEAPON_BYTES_TO_INT,
)

# ── Fixtures ─────────────────────────────────────────────────────────────────

# Premier weapon_bytes connu dans WEAPON_BYTES_TO_INT
_KNOWN_WEAPON_BYTES, _KNOWN_WEAPON_ID = next(iter(WEAPON_BYTES_TO_INT.items()))
_UNKNOWN_WEAPON_BYTES = bytes(8)  # 8 zéros, absent du dictionnaire


def _kill(
    xuid: str = "x1",
    time_ms: int = 5000,
    is_melee: bool = False,
    is_grenade: bool = False,
) -> dict:
    return {
        "xuid": xuid,
        "time_ms": time_ms,
        "is_melee": is_melee,
        "is_grenade": is_grenade,
        "match_id": "m",
    }


def _event(timestamp_ms: int, weapon_bytes: bytes = _KNOWN_WEAPON_BYTES) -> dict:
    return {
        "timestamp_ms": timestamp_ms,
        "weapon_bytes": weapon_bytes,
        "player_index": 1,
        "chunk_idx": 0,
    }


# ── Tests correlate_kills_global ─────────────────────────────────────────────


class TestCorrelateKillsGlobal:
    def test_empty_kills(self):
        result = correlate_kills_global([], [], {}, {}, {}, [], [])
        assert result == []

    def test_fire_event_claimed(self):
        """Un kill normal clame l'event qui précède dans la fenêtre."""
        kills = [_kill(time_ms=5000)]
        events = [_event(timestamp_ms=4800)]
        result = correlate_kills_global(kills, events, {"x1": 1}, {}, {}, [], [])
        assert len(result) == 1
        assert result[0].attribution_path == "fire_event"
        assert result[0].weapon_id == _KNOWN_WEAPON_ID
        assert result[0].delta_ms == 200

    def test_event_outside_window_uses_formula_a(self):
        """Un event hors fenêtre (>5000ms avant) tombe en formula_a."""
        kills = [_kill(time_ms=5000)]
        # KILL_WINDOW_MS=5000 : fenêtre = [0, 5000]. Event à -1ms est hors fenêtre.
        events = [_event(timestamp_ms=-1)]
        result = correlate_kills_global(kills, events, {"x1": None}, {}, {}, [], [])
        assert len(result) == 1
        assert result[0].attribution_path == "formula_a"

    def test_no_event_uses_formula_a(self):
        """Sans event disponible → formula_a."""
        kills = [_kill(time_ms=5000)]
        result = correlate_kills_global(kills, [], {"x1": None}, {}, {}, [], [])
        assert result[0].attribution_path == "formula_a"

    def test_melee_sentinel(self):
        """Kill melee → sentinel MELEE_WEAPON_ID, path=none."""
        kills = [_kill(time_ms=1000, is_melee=True)]
        events = [_event(timestamp_ms=900)]  # doit être ignoré
        result = correlate_kills_global(kills, events, {}, {}, {}, [], [])
        assert result[0].weapon_id == MELEE_WEAPON_ID
        assert result[0].attribution_path == "none"
        assert result[0].confidence == "none"

    def test_grenade_sentinel(self):
        """Kill grenade → sentinel GRENADE_WEAPON_ID."""
        kills = [_kill(time_ms=1000, is_grenade=True)]
        result = correlate_kills_global(kills, [], {}, {}, {}, [], [])
        assert result[0].weapon_id == GRENADE_WEAPON_ID
        assert result[0].attribution_path == "none"

    def test_bijection_two_players(self):
        """Claim-and-remove : chaque event n'est claimé qu'une seule fois."""
        kills = [
            _kill(xuid="x1", time_ms=3000),
            _kill(xuid="x2", time_ms=5000),
        ]
        events = [
            _event(timestamp_ms=2900),  # pour x1
            _event(timestamp_ms=4900),  # pour x2
        ]
        result = correlate_kills_global(kills, events, {"x1": 1, "x2": 2}, {}, {}, [], [])
        assert all(r.attribution_path == "fire_event" for r in result)
        # weapon_id identique (même arme en fixture), mais deux events distincts
        assert result[0].delta_ms == 100  # kill 3000 - event 2900
        assert result[1].delta_ms == 100  # kill 5000 - event 4900

    def test_claim_priority_earliest_kill_first(self):
        """Kill trié par temps : le kill le plus tôt claim l'event le plus proche."""
        kills = [
            _kill(xuid="x1", time_ms=5000),
            _kill(xuid="x2", time_ms=5010),
        ]
        # Un seul event à 4900ms — doit aller au kill x1 (le plus précoce)
        events = [_event(timestamp_ms=4900)]
        result = correlate_kills_global(kills, events, {"x1": 1, "x2": 2}, {}, {}, [], [])
        fire = [r for r in result if r.attribution_path == "fire_event"]
        fallback = [r for r in result if r.attribution_path == "formula_a"]
        assert len(fire) == 1
        assert fire[0].xuid == "x1"
        assert len(fallback) == 1
        assert fallback[0].xuid == "x2"

    def test_best_candidate_is_closest_in_time(self):
        """Parmi plusieurs events dans la fenêtre, prend le plus proche (le plus récent)."""
        kills = [_kill(time_ms=5000)]
        events = [
            _event(timestamp_ms=2000),  # 3000ms avant — plus loin
            _event(timestamp_ms=4800),  # 200ms avant — plus proche → doit être claimé
        ]
        result = correlate_kills_global(kills, events, {"x1": 1}, {}, {}, [], [])
        assert result[0].delta_ms == 200

    def test_melee_does_not_consume_event(self):
        """Un kill melee ne consomme pas d'event — l'event reste disponible."""
        kills = [
            _kill(xuid="x1", time_ms=3000, is_melee=True),
            _kill(xuid="x2", time_ms=5000),
        ]
        events = [_event(timestamp_ms=4800)]
        result = correlate_kills_global(kills, events, {"x1": 1, "x2": 2}, {}, {}, [], [])
        assert result[0].attribution_path == "none"  # sentinel
        assert result[1].attribution_path == "fire_event"  # event non consommé

    def test_with_log_collector(self):
        """Le log_collector reçoit bien les décisions."""
        log = MatchLogCollector("test")
        kills = [
            _kill(xuid="x1", time_ms=5000),
            _kill(xuid="x2", time_ms=6000, is_melee=True),
        ]
        events = [_event(timestamp_ms=4800)]
        correlate_kills_global(kills, events, {"x1": 1}, {}, {}, [], [], log_collector=log)
        assert len(log.kill_decisions) == 2
        # candidates_count correct pour fire_event
        fe_decision = next(d for d in log.kill_decisions if d["attribution_path"] == "fire_event")
        assert fe_decision["candidates_count"] == 1
        # candidates_count = 0 pour sentinel
        sentinel_decision = next(d for d in log.kill_decisions if d["attribution_path"] == "none")
        assert sentinel_decision["candidates_count"] == 0

    def test_preserves_xuid_in_result(self):
        """Le xuid du kill est propagé dans l'attribution."""
        kills = [_kill(xuid="targetplayer", time_ms=3000)]
        events = [_event(timestamp_ms=2900)]
        result = correlate_kills_global(kills, events, {"targetplayer": 1}, {}, {}, [], [])
        assert result[0].xuid == "targetplayer"

    def test_returns_one_result_per_kill(self):
        """Toujours autant de résultats que de kills en entrée."""
        kills = [_kill(xuid=f"x{i}", time_ms=i * 1000) for i in range(1, 6)]
        result = correlate_kills_global(kills, [], {}, {}, {}, [], [])
        assert len(result) == len(kills)


# ── Tests _make_sentinel_global ───────────────────────────────────────────────


class TestMakeSentinelGlobal:
    def test_melee_sentinel(self):
        kill = _kill(time_ms=1500)
        attr = _make_sentinel_global(kill, "match1", MELEE_WEAPON_ID)
        assert isinstance(attr, KillAttribution)
        assert attr.weapon_id == MELEE_WEAPON_ID
        assert attr.attribution_path == "none"
        assert attr.confidence == "none"
        assert attr.delta_ms is None
        assert attr.match_id == "match1"
        assert attr.time_ms == 1500

    def test_grenade_sentinel(self):
        kill = _kill(time_ms=2000)
        attr = _make_sentinel_global(kill, "m", GRENADE_WEAPON_ID)
        assert attr.weapon_id == GRENADE_WEAPON_ID


# ── Tests _attribution_from_event ─────────────────────────────────────────────


class TestAttributionFromEvent:
    def _compute_conf(self, wid: int, delta: int) -> str:
        from src.analysis.weapon_parser import compute_confidence

        return compute_confidence(wid, delta)

    def test_known_weapon_bytes(self):
        """weapon_bytes connu → weapon_id via WEAPON_BYTES_TO_INT."""
        kill = _kill(time_ms=5000)
        ev = _event(timestamp_ms=4800, weapon_bytes=_KNOWN_WEAPON_BYTES)
        attr = _attribution_from_event(kill, ev, "m", self._compute_conf)
        assert attr.weapon_id == _KNOWN_WEAPON_ID
        assert attr.attribution_path == "fire_event"
        assert attr.delta_ms == 200

    def test_unknown_weapon_bytes_fallback(self):
        """weapon_bytes inconnu → fallback int.from_bytes (ligne 128)."""
        kill = _kill(time_ms=5000)
        ev = _event(timestamp_ms=4900, weapon_bytes=_UNKNOWN_WEAPON_BYTES)
        attr = _attribution_from_event(kill, ev, "m", self._compute_conf)
        assert attr.weapon_id == 0  # 8 zéros → 0
        assert attr.attribution_path == "fire_event"
        assert attr.delta_ms == 100

    def test_propagates_player_index(self):
        """player_index de l'event est propagé."""
        kill = _kill(time_ms=3000)
        ev = {
            "timestamp_ms": 2900,
            "weapon_bytes": _KNOWN_WEAPON_BYTES,
            "player_index": 7,
            "chunk_idx": 2,
        }
        attr = _attribution_from_event(kill, ev, "m", self._compute_conf)
        assert attr.player_index == 7
        assert attr.source_chunk_idx == 2

    def test_swap_detected_on_large_delta(self):
        """swap_detected=True si delta >= swap threshold de l'arme."""
        kill = _kill(time_ms=10000)
        ev = _event(timestamp_ms=0, weapon_bytes=_KNOWN_WEAPON_BYTES)
        # delta=10000ms, très largement au-delà de tout threshold
        attr = _attribution_from_event(kill, ev, "m", self._compute_conf)
        assert attr.swap_detected is True
