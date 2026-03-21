"""Tests réconciliation API découplée (reconciliation.py)."""

from __future__ import annotations

from src.analysis._kill_attribution import KillAttribution
from src.analysis._weapon_data import MELEE_WEAPON_ID
from src.analysis.reconciliation import assign_sentinels, reconcile_api_aggregates

BR75_WID = int.from_bytes(bytes.fromhex("2b1824d542c9679f"), "big")
SIDEKICK_WID = int.from_bytes(bytes.fromhex("f408190f42c9679f"), "big")


def _attr(  # noqa: PLR0913
    t_ms: int = 1000,
    weapon_id: int | None = None,
    confidence: str = "none",
    path: str = "none",
    reconciled_as: int | None = None,
    xuid: str = "x1",
):
    return KillAttribution(
        match_id="m",
        xuid=xuid,
        time_ms=t_ms,
        weapon_id=weapon_id,
        reconciled_as=reconciled_as,
        delta_ms=500,
        confidence=confidence,
        attribution_path=path,
        swap_detected=False,
        delayed_damage=False,
        player_index=1,
        source_chunk_idx=0,
    )


class TestReconcileApiAggregates:
    def test_no_api_data_unchanged(self):
        attrs = [_attr(confidence="high", weapon_id=BR75_WID, path="fire_event")]
        result = reconcile_api_aggregates(attrs, {})
        assert result[0].reconciled_as is None

    def test_exact_match_unchanged(self):
        attrs = [_attr(confidence="high", weapon_id=BR75_WID, path="fire_event")]
        api = {BR75_WID: 1}
        result = reconcile_api_aggregates(attrs, api)
        assert result[0].reconciled_as is None

    def test_deficit_assigns_reconciled_as(self):
        # 1 kill low sans weapon → surplus API BR75 → reconciled_as = BR75
        attrs = [_attr(confidence="low", weapon_id=None, path="fire_event")]
        api = {BR75_WID: 1}
        result = reconcile_api_aggregates(attrs, api)
        assert result[0].reconciled_as == BR75_WID

    def test_never_overwrites_weapon_id(self):
        attrs = [_attr(confidence="low", weapon_id=BR75_WID, path="fire_event")]
        api = {SIDEKICK_WID: 1}
        result = reconcile_api_aggregates(attrs, api)
        # weapon_id JAMAIS modifié
        assert result[0].weapon_id == BR75_WID

    def test_never_assigns_on_high(self):
        attrs = [_attr(confidence="high", weapon_id=BR75_WID, path="fire_event")]
        api = {SIDEKICK_WID: 5}
        result = reconcile_api_aggregates(attrs, api)
        assert result[0].reconciled_as is None

    def test_never_assigns_on_medium(self):
        attrs = [_attr(confidence="medium", weapon_id=BR75_WID, path="fire_event")]
        api = {SIDEKICK_WID: 5}
        result = reconcile_api_aggregates(attrs, api)
        assert result[0].reconciled_as is None

    def test_log_callback_called(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        attrs = [_attr(confidence="low", weapon_id=None, path="fire_event")]
        api = {BR75_WID: 1}
        reconcile_api_aggregates(attrs, api, log_collector=log)
        assert len(log.reconciliation_decisions) >= 1


class TestAssignSentinels:
    def test_assign_melee_sentinel(self):
        attrs = [_attr(t_ms=1000, xuid="x1")]
        sentinel_map = {"x1_1000": MELEE_WEAPON_ID}
        result = assign_sentinels(attrs, sentinel_map)
        assert result[0].reconciled_as == MELEE_WEAPON_ID

    def test_does_not_overwrite_existing_reconciled(self):
        attrs = [_attr(t_ms=1000, xuid="x1", reconciled_as=BR75_WID)]
        sentinel_map = {"x1_1000": MELEE_WEAPON_ID}
        result = assign_sentinels(attrs, sentinel_map)
        assert result[0].reconciled_as == BR75_WID  # inchangé

    def test_no_match_leaves_none(self):
        attrs = [_attr(t_ms=1000, xuid="x1")]
        sentinel_map = {"x2_2000": MELEE_WEAPON_ID}
        result = assign_sentinels(attrs, sentinel_map)
        assert result[0].reconciled_as is None

    def test_log_collector_records_step(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        attrs = [_attr(t_ms=1000, xuid="x1")]
        sentinel_map = {"x1_1000": MELEE_WEAPON_ID}
        assign_sentinels(attrs, sentinel_map, log_collector=log)
        steps = [s for s in log.steps if s.get("step") == "assign_sentinels"]
        assert len(steps) == 1
        assert steps[0]["assigned"] == 1

    def test_no_assignment_no_log(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        attrs = [_attr(t_ms=1000, xuid="x1")]
        assign_sentinels(attrs, {}, log_collector=log)
        steps = [s for s in log.steps if s.get("step") == "assign_sentinels"]
        assert len(steps) == 0


class TestSurplusExhaustion:
    """Tests pour le logging quand le surplus API est épuisé."""

    def test_surplus_exhausted_logs_warning(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        # 3 kills low/none, mais surplus API = 1 seule arme
        attrs = [
            _attr(t_ms=1000, confidence="low", path="fire_event"),
            _attr(t_ms=2000, confidence="none", path="fire_event"),
            _attr(t_ms=3000, confidence="low", path="fire_event"),
        ]
        api = {BR75_WID: 1}  # surplus de 1 seulement
        reconcile_api_aggregates(attrs, api, log_collector=log)
        warnings = [
            w for w in log.warnings if w.get("message") == "reconciliation_surplus_exhausted"
        ]
        assert len(warnings) == 1
        assert warnings[0]["remaining_unreconciled"] > 0

    def test_surplus_sufficient_no_warning(self):
        from src.analysis._parser_logging import MatchLogCollector

        log = MatchLogCollector("test")
        attrs = [_attr(t_ms=1000, confidence="low", path="fire_event")]
        api = {BR75_WID: 5}  # largement suffisant
        reconcile_api_aggregates(attrs, api, log_collector=log)
        warnings = [
            w for w in log.warnings if w.get("message") == "reconciliation_surplus_exhausted"
        ]
        assert len(warnings) == 0


class TestResolveWeaponDisplay:
    """Tests pour _weapon_data.resolve_weapon_display()."""

    def test_none_returns_none(self):
        from src.analysis._weapon_data import resolve_weapon_display

        assert resolve_weapon_display(None) is None

    def test_melee_sentinel_fr(self):
        from src.analysis._weapon_data import MELEE_WEAPON_ID, resolve_weapon_display

        assert resolve_weapon_display(MELEE_WEAPON_ID, "fr") == "Corps à corps"

    def test_melee_sentinel_en(self):
        from src.analysis._weapon_data import MELEE_WEAPON_ID, resolve_weapon_display

        assert resolve_weapon_display(MELEE_WEAPON_ID, "en") == "Melee"

    def test_grenade_sentinel(self):
        from src.analysis._weapon_data import GRENADE_WEAPON_ID, resolve_weapon_display

        assert resolve_weapon_display(GRENADE_WEAPON_ID) == "Grenade"

    def test_vehicle_sentinel_fr(self):
        from src.analysis._weapon_data import VEHICLE_WEAPON_ID, resolve_weapon_display

        assert resolve_weapon_display(VEHICLE_WEAPON_ID, "fr") == "Véhicule"

    def test_vehicle_sentinel_en(self):
        from src.analysis._weapon_data import VEHICLE_WEAPON_ID, resolve_weapon_display

        assert resolve_weapon_display(VEHICLE_WEAPON_ID, "en") == "Vehicle"

    def test_known_weapon_fr(self):
        from src.analysis._weapon_data import resolve_weapon_display

        result = resolve_weapon_display(BR75_WID, "fr")
        assert result is not None
        assert isinstance(result, str)
        assert result != f"weapon_{BR75_WID}"

    def test_known_weapon_en(self):
        from src.analysis._weapon_data import resolve_weapon_display

        result = resolve_weapon_display(BR75_WID, "en")
        assert result is not None
        assert isinstance(result, str)

    def test_unknown_weapon_fallback(self):
        from src.analysis._weapon_data import resolve_weapon_display

        unknown = 0xDEADBEEF
        result = resolve_weapon_display(unknown)
        assert result == f"weapon_{unknown}"
