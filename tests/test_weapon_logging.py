"""Tests logging structuré (_parser_logging.py)."""

from __future__ import annotations

import json
import logging

from src.analysis._kill_attribution import KillAttribution
from src.analysis._parser_logging import MatchLogCollector


def _attr(conf: str = "high", path: str = "fire_event"):
    return KillAttribution(
        match_id="m",
        xuid="x",
        time_ms=1000,
        weapon_id=42,
        reconciled_as=None,
        delta_ms=500,
        confidence=conf,
        attribution_path=path,
        swap_detected=False,
        delayed_damage=False,
        player_index=1,
        source_chunk_idx=0,
    )


class TestMatchLogCollector:
    def test_empty_summary(self):
        log = MatchLogCollector("test")
        s = log.summary()
        assert s["match_id"] == "test"
        assert s["kills_decided"] == 0
        assert s["steps_count"] == 0

    def test_record_step(self):
        log = MatchLogCollector("test")
        log.record_step("scan_fire", chunk=0, events=5)
        assert len(log.steps) == 1
        assert log.steps[0]["step"] == "scan_fire"

    def test_kill_decision(self):
        log = MatchLogCollector("test")
        kill = {"xuid": "x1", "time_ms": 1000}
        attr = _attr()
        log.kill_decision(kill, attr, {"candidates_count": 3})
        assert len(log.kill_decisions) == 1
        assert log.kill_decisions[0]["weapon_id"] == 42

    def test_reconciliation_decision(self):
        log = MatchLogCollector("test")
        log.reconciliation_decision(
            "assign_sentinel",
            1000,
            "x1",
            {"weapon_id": None},
            {"reconciled_as": 0},
        )
        assert len(log.reconciliation_decisions) == 1
        assert log.reconciliation_decisions[0]["action"] == "assign_sentinel"

    def test_warn(self):
        log = MatchLogCollector("test")
        log.warn("unresolved_player", xuid="x1")
        assert len(log.warnings) == 1

    def test_confidence_distribution(self):
        log = MatchLogCollector("test")
        for conf in ["high", "high", "medium", "low"]:
            kill = {"xuid": "x", "time_ms": 0}
            log.kill_decision(kill, _attr(conf=conf), {})
        s = log.summary()
        assert s["confidence_distribution"]["high"] == 2
        assert s["confidence_distribution"]["medium"] == 1

    def test_confidence_sentinel_vs_no_weapon(self):
        """sentinel (melee/grenade) et no_weapon (formula_a sans arme) sont distingués."""
        log = MatchLogCollector("test")
        kill = {"xuid": "x", "time_ms": 0}
        # Sentinel : attribution_path="none", confidence="none"
        log.kill_decision(kill, _attr(conf="none", path="none"), {})
        log.kill_decision(kill, _attr(conf="none", path="none"), {})
        # No-weapon : confidence="none", attribution_path="formula_a"
        log.kill_decision(kill, _attr(conf="none", path="formula_a"), {})
        s = log.summary()
        dist = s["confidence_distribution"]
        assert dist.get("sentinel") == 2
        assert dist.get("no_weapon") == 1
        assert "none" not in dist  # l'ancienne clé brute ne doit plus apparaître

    def test_path_distribution(self):
        log = MatchLogCollector("test")
        kill = {"xuid": "x", "time_ms": 0}
        log.kill_decision(kill, _attr(path="fire_event"), {})
        log.kill_decision(kill, _attr(path="formula_a"), {})
        s = log.summary()
        assert s["path_distribution"]["fire_event"] == 1
        assert s["path_distribution"]["formula_a"] == 1

    def test_flush_logs_to_logger(self, caplog):
        log = MatchLogCollector("test")
        kill = {"xuid": "x", "time_ms": 0}
        log.kill_decision(kill, _attr(), {})
        with caplog.at_level(logging.INFO, logger="levelup.weapon_parser"):
            log.flush()
        assert any("COMPLETE" in r.message for r in caplog.records)

    def test_flush_compact_format(self, caplog):
        """La ligne COMPLETE est au format compact lisible."""
        log = MatchLogCollector("abcdef1234")
        kill = {"xuid": "x", "time_ms": 0}
        log.kill_decision(kill, _attr(conf="high", path="fire_event"), {})
        log.kill_decision(kill, _attr(conf="none", path="none"), {})  # sentinel
        log.kill_decision(kill, _attr(conf="none", path="formula_a"), {})  # no_weapon
        with caplog.at_level(logging.INFO, logger="levelup.weapon_parser"):
            log.flush()
        msg = next(r.message for r in caplog.records if "COMPLETE" in r.message)
        assert "match=abcdef12…" in msg
        assert "k=3" in msg
        assert "H=1" in msg
        assert "s=1" in msg  # sentinel dans conf
        assert "?=1" in msg  # no_weapon
        assert "fe=1" in msg
        assert "s=1" in msg  # sentinel dans paths
        assert "warn=0" in msg

    def test_serializable_json(self):
        log = MatchLogCollector("test")
        log.record_step("load", count=5)
        log.warn("test warning")
        # Doit être sérialisable sans erreur
        json.dumps(log.summary())

    def test_parser_works_without_log(self):
        """Parser fonctionne avec log_collector=None."""
        from src.analysis.weapon_parser import correlate_kills

        kills = [
            {"match_id": "m", "xuid": "x", "time_ms": 1000, "is_melee": False, "is_grenade": False}
        ]
        result = correlate_kills(
            kills=kills,
            fire_events_by_pi={},
            pi_mapping={},
            timeline={},
            swap_pis={},
            timing=[],
            chunks_sorted=[],
            match_id="test",
            log_collector=None,
        )
        assert len(result) == 1
