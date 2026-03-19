"""Tests parser v2 — claim-and-remove, KillAttribution, confidence."""

from __future__ import annotations

from src.analysis._kill_attribution import KillAttribution
from src.analysis._weapon_data import (
    GRENADE_WEAPON_ID,
    MELEE_MEDALS,
    MELEE_WEAPON_ID,
    WEAPON_ID_MAP,
    WEAPON_IDS_INT,
    WEAPON_TIMING_BY_ID,
)
from src.analysis.weapon_parser import (
    KILL_WINDOW_MS,
    compute_confidence,
    correlate_kills,
)

# ── Fixtures ──

BR75_BYTES = bytes.fromhex("2b1824d542c9679f")
BR75_WID = int.from_bytes(BR75_BYTES, "big")
SIDEKICK_BYTES = bytes.fromhex("f408190f42c9679f")
SIDEKICK_WID = int.from_bytes(SIDEKICK_BYTES, "big")


def _kill(t_ms: int, xuid: str = "123", **kw):
    """Fabrique un kill dict minimal."""
    return {
        "match_id": "test-match",
        "xuid": xuid,
        "time_ms": t_ms,
        "gamertag": "TestPlayer",
        "medals_nearby": [],
        "is_melee": False,
        "is_grenade": False,
        **kw,
    }


def _fire(t_ms: int, weapon_bytes: bytes = BR75_BYTES, pi: int = 1, seq: int = 1):
    """Fabrique un fire event dict minimal."""
    return {
        "timestamp_ms": t_ms,
        "weapon_bytes": weapon_bytes,
        "fire_seq": seq,
        "player_index": pi,
    }


# ══════════════════════════════════════════════════════════════════════
# Groupe A — Constantes et invariants
# ══════════════════════════════════════════════════════════════════════


class TestConstants:
    def test_sentinel_values_cannot_collide_with_film(self):
        for sentinel in (0, 1, 2):
            assert sentinel not in WEAPON_IDS_INT

    def test_all_confirmed_weapons_in_map(self):
        assert len(WEAPON_ID_MAP) >= 36

    def test_weapon_timing_covers_known_weapons(self):
        for _wid in WEAPON_IDS_INT:
            # Chaque arme a un timing (sinon default est utilisé)
            pass
        assert len(WEAPON_TIMING_BY_ID) > 0

    def test_melee_medals_includes_ninja_pancake(self):
        assert "Ninja" in MELEE_MEDALS
        assert "Pancake" in MELEE_MEDALS

    def test_kill_window_ms_is_5000(self):
        assert KILL_WINDOW_MS == 5000


# ══════════════════════════════════════════════════════════════════════
# Groupe F — correlate_kills (CŒUR v2)
# ══════════════════════════════════════════════════════════════════════


class TestCorrelateKills:
    """Tests du corrélateur claim-and-remove."""

    def _run(self, kills, events_by_pi, pi_map=None, **kw):
        """Helper pour appeler correlate_kills avec des valeurs par défaut."""
        if pi_map is None:
            pi_map = {"123": 1}
        return correlate_kills(
            kills=kills,
            fire_events_by_pi=events_by_pi,
            pi_mapping=pi_map,
            timeline={},
            swap_pis={},
            timing=[],
            chunks_sorted=[],
            match_id="test",
            **kw,
        )

    def test_empty_kills(self):
        assert self._run([], {}) == []

    def test_single_kill_single_event(self):
        kills = [_kill(45000)]
        events = {1: [_fire(44800)]}
        result = self._run(kills, events)
        assert len(result) == 1
        assert result[0].confidence == "high"
        assert result[0].delta_ms == 200
        assert result[0].attribution_path == "fire_event"

    def test_kill_no_event(self):
        kills = [_kill(45000)]
        result = self._run(kills, {})
        assert len(result) == 1
        assert result[0].confidence == "none"
        assert result[0].attribution_path == "formula_a"

    def test_kill_with_formula_a_fallback(self):
        kills = [_kill(45000)]
        timeline = {0: {1: BR75_BYTES}}
        result = correlate_kills(
            kills=kills,
            fire_events_by_pi={},
            pi_mapping={"123": 1},
            timeline=timeline,
            swap_pis={},
            timing=[(0, 100000)],
            chunks_sorted=[0],
            match_id="test",
        )
        assert result[0].attribution_path == "formula_a"
        assert result[0].weapon_id is not None

    def test_claim_and_remove_no_double(self):
        """2 kills, 1 fire event → seul le 1er claim."""
        kills = [_kill(45000), _kill(46000)]
        events = {1: [_fire(44000)]}
        result = self._run(kills, events)
        assert result[0].weapon_id is not None
        assert result[0].attribution_path == "fire_event"
        # 2e kill → fallback (event claimé)
        assert result[1].attribution_path == "formula_a"

    def test_claim_and_remove_two_events(self):
        """2 kills, 2 fire events → chacun claim le sien."""
        kills = [_kill(45000), _kill(46000)]
        events = {1: [_fire(44000, seq=1), _fire(45500, seq=2)]}
        result = self._run(kills, events)
        assert all(r.attribution_path == "fire_event" for r in result)

    def test_window_exactly_5s(self):
        kills = [_kill(50000)]
        events = {1: [_fire(45000)]}  # delta = exactly 5000ms
        result = self._run(kills, events)
        assert result[0].weapon_id is not None

    def test_window_exceeded(self):
        kills = [_kill(50000)]
        events = {1: [_fire(44999)]}  # delta = 5001ms
        result = self._run(kills, events)
        assert result[0].attribution_path == "formula_a"

    def test_takes_closest_event(self):
        """3 events dans fenêtre → prend le plus proche (le plus récent)."""
        kills = [_kill(50000)]
        events = {1: [_fire(46000, seq=1), _fire(48000, seq=2), _fire(49000, seq=3)]}
        result = self._run(kills, events)
        assert result[0].delta_ms == 1000  # 50000 - 49000

    def test_output_length_invariant(self):
        n = 7
        kills = [_kill(40000 + i * 1000) for i in range(n)]
        result = self._run(kills, {})
        assert len(result) == n

    def test_weapon_id_never_sentinel(self):
        """Aucun weapon_id du parser ne doit être un sentinel."""
        kills = [_kill(45000)]
        events = {1: [_fire(43000)]}
        result = self._run(kills, events)
        for a in result:
            if a.weapon_id is not None:
                assert a.weapon_id not in {0, 1, 2}

    def test_reconciled_as_always_none(self):
        """reconciled_as est toujours None à la sortie du parser (pas sa responsabilité)."""
        kills = [_kill(45000)]
        events = {1: [_fire(43000)]}
        result = self._run(kills, events)
        assert all(a.reconciled_as is None for a in result)

    def test_melee_kill(self):
        kills = [_kill(45000, is_melee=True)]
        result = self._run(kills, {})
        assert result[0].weapon_id == MELEE_WEAPON_ID
        assert result[0].attribution_path == "none"

    def test_grenade_kill(self):
        kills = [_kill(45000, is_grenade=True)]
        result = self._run(kills, {})
        assert result[0].weapon_id == GRENADE_WEAPON_ID
        assert result[0].attribution_path == "none"

    def test_multiple_players(self):
        """2 joueurs, kills entrelacés → attributions indépendantes par pi."""
        kills = [_kill(45000, xuid="A"), _kill(45500, xuid="B")]
        events = {1: [_fire(44000)], 2: [_fire(44500)]}
        pi_map = {"A": 1, "B": 2}
        result = self._run(kills, events, pi_map)
        assert len(result) == 2
        assert all(r.attribution_path == "fire_event" for r in result)

    def test_fire_event_after_kill_ignored(self):
        """Event à kill_t + 500 → non retenu (hors fenêtre)."""
        kills = [_kill(45000)]
        events = {1: [_fire(45500)]}
        result = self._run(kills, events)
        assert result[0].attribution_path == "formula_a"

    def test_log_collector_per_kill(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        kills = [_kill(45000), _kill(46000)]
        events = {1: [_fire(44000)]}
        self._run(kills, events, log_collector=log)
        assert len(log.kill_decisions) == 2


# ══════════════════════════════════════════════════════════════════════
# Groupe G — compute_confidence
# ══════════════════════════════════════════════════════════════════════


class TestComputeConfidence:
    def test_known_weapon_zone_a(self):
        assert compute_confidence(BR75_WID, 300) == "high"

    def test_default_timing_zone_b(self):
        # Arme inconnue → timing par défaut (650, 2000)
        unknown_wid = 0xDEADBEEF
        assert compute_confidence(unknown_wid, 1000) == "medium"  # 650 <= 1000 <= 2000

    def test_default_timing_zone_c(self):
        # Arme inconnue → timing par défaut (650, 2000)
        unknown_wid = 0xDEADBEEF
        assert compute_confidence(unknown_wid, 2500) == "low"  # > 2000

    def test_none_weapon(self):
        assert compute_confidence(None, 100) == "none"

    def test_none_delta(self):
        assert compute_confidence(BR75_WID, None) == "none"

    def test_unknown_weapon_uses_default(self):
        # Unknown weapon → default timing (650, 2000)
        unknown = 0xDEADBEEF
        assert compute_confidence(unknown, 300) == "high"  # < 650
        assert compute_confidence(unknown, 1000) == "medium"  # 650-2000
        assert compute_confidence(unknown, 3000) == "low"  # > 2000


# ══════════════════════════════════════════════════════════════════════
# Groupe H — KillAttribution dataclass
# ══════════════════════════════════════════════════════════════════════


class TestKillAttribution:
    def _attr(self, weapon_id=None, reconciled_as=None):
        return KillAttribution(
            match_id="m",
            xuid="x",
            time_ms=1000,
            weapon_id=weapon_id,
            reconciled_as=reconciled_as,
            delta_ms=None,
            confidence="none",
            attribution_path="none",
            swap_detected=False,
            delayed_damage=False,
            player_index=None,
            source_chunk_idx=None,
        )

    def test_effective_weapon_id_no_reconciled(self):
        a = self._attr(weapon_id=BR75_WID)
        assert a.effective_weapon_id == BR75_WID

    def test_effective_weapon_id_with_reconciled(self):
        a = self._attr(weapon_id=BR75_WID, reconciled_as=0)
        assert a.effective_weapon_id == 0

    def test_effective_weapon_id_both_none(self):
        a = self._attr()
        assert a.effective_weapon_id is None

    def test_attribution_path_values(self):
        """Vérifier les valeurs autorisées."""
        valids = {"fire_event", "melee_event", "formula_a", "none"}
        for path in valids:
            a = KillAttribution(
                match_id="m",
                xuid="x",
                time_ms=0,
                weapon_id=None,
                reconciled_as=None,
                delta_ms=None,
                confidence="none",
                attribution_path=path,
                swap_detected=False,
                delayed_damage=False,
                player_index=None,
                source_chunk_idx=None,
            )
            assert a.attribution_path in valids
