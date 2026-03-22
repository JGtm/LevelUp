"""Logging structuré pour le weapon parser v2.

Chaque décision d'attribution est traçable sans relancer le parser.
Le collecteur est optionnel — le parser fonctionne sans (log_callback=None).
"""

from __future__ import annotations

import logging
from collections import Counter
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from src.analysis._kill_attribution import KillAttribution

logger = logging.getLogger("levelup.weapon_parser")


@dataclass
class MatchLogCollector:
    """Collecteur de logs structurés pour un match."""

    match_id: str
    steps: list[dict] = field(default_factory=list)
    kill_decisions: list[dict] = field(default_factory=list)
    reconciliation_decisions: list[dict] = field(default_factory=list)
    warnings: list[dict] = field(default_factory=list)

    def record_step(self, step_name: str, **kwargs: object) -> None:
        """Enregistre une étape du pipeline."""
        entry = {"step": step_name, **kwargs}
        self.steps.append(entry)
        logger.debug("match=%s step=%s %s", self.match_id, step_name, kwargs)

    def kill_decision(
        self,
        kill: dict,
        attribution: KillAttribution,
        details: dict,
    ) -> None:
        """Enregistre la décision d'attribution pour un kill."""
        entry = {
            "xuid": kill.get("xuid", ""),
            "time_ms": kill.get("time_ms", 0),
            "weapon_id": attribution.weapon_id,
            "confidence": attribution.confidence,
            "attribution_path": attribution.attribution_path,
            "delta_ms": attribution.delta_ms,
            "candidates_count": details.get("candidates_count", 0),
            "claimed_event_ts": details.get("claimed_event_ts"),
            "fallback_used": details.get("fallback_used", False),
        }
        self.kill_decisions.append(entry)
        logger.debug(
            "match=%s kill xuid=%s t=%dms → weapon=%s conf=%s path=%s delta=%s",
            self.match_id,
            kill.get("xuid", "?"),
            kill.get("time_ms", 0),
            attribution.weapon_id,
            attribution.confidence,
            attribution.attribution_path,
            attribution.delta_ms,
        )

    def reconciliation_decision(  # noqa: PLR0913
        self,
        action: str,
        kill_time_ms: int,
        xuid: str,
        before: dict,
        after: dict,
    ) -> None:
        """Enregistre une décision de réconciliation."""
        entry = {
            "action": action,
            "xuid": xuid,
            "time_ms": kill_time_ms,
            "before": before,
            "after": after,
        }
        self.reconciliation_decisions.append(entry)
        logger.info(
            "match=%s reconcile %s xuid=%s t=%dms: %s → %s",
            self.match_id,
            action,
            xuid,
            kill_time_ms,
            before,
            after,
        )

    def warn(self, message: str, **context: object) -> None:
        """Enregistre un warning."""
        entry = {"message": message, **context}
        self.warnings.append(entry)
        logger.warning("match=%s %s %s", self.match_id, message, context)

    def debug(self, message: str, **context: object) -> None:
        """Log un message debug (sans stocker — diagnostic léger)."""
        logger.debug("match=%s %s %s", self.match_id, message, context)

    def summary(self) -> dict:
        """Résumé JSON-sérialisable du traitement."""
        return {
            "match_id": self.match_id,
            "steps_count": len(self.steps),
            "kills_decided": len(self.kill_decisions),
            "reconciliations": len(self.reconciliation_decisions),
            "warnings": len(self.warnings),
            "confidence_distribution": self._confidence_dist(),
            "path_distribution": self._path_dist(),
            "b2_dispatch": self._b2_dispatch_stats(),
        }

    def _b2_dispatch_stats(self) -> dict | None:
        """Extrait les stats de dispatch b2→pi depuis les steps."""
        for step in self.steps:
            if step.get("step") == "b2_dispatch":
                return {
                    "total": step.get("total_events", 0),
                    "dispatched": step.get("dispatched_events", 0),
                    "dropped": step.get("dropped_events", 0),
                    "resolved_b2": step.get("resolved_b2", 0),
                }
        return None

    def _confidence_dist(self) -> dict[str, int]:
        """Distribution des confiances — distingue sentinel (melee/grenade) et no_weapon.

        - ``sentinel`` : kills melee/grenade (attribution_path='none', arme connue).
        - ``no_weapon`` : kills formula_a sans arme trouvée dans la timeline (weapon_id=None).
        """
        result: dict[str, int] = {}
        for d in self.kill_decisions:
            if d["attribution_path"] == "none":
                key = "sentinel"
            elif d["confidence"] == "none":
                key = "no_weapon"
            else:
                key = d["confidence"]
            result[key] = result.get(key, 0) + 1
        return result

    def _path_dist(self) -> dict[str, int]:
        return dict(Counter(d["attribution_path"] for d in self.kill_decisions))

    def flush(self) -> None:
        """Écrit le résumé compact dans le logger INFO.

        Format : ``match=XXXXXXXX… COMPLETE k=N | H=X M=X L=X s=X ?=X | fe=X fa=X s=X | warn=N``

        Sections :
        - conf   : H=high  M=medium  L=low  s=sentinel  ?=no_weapon
        - paths  : fe=fire_event  fa=formula_a  s=sentinel (attribution_path='none')
        - b2     : optionnel, raw/ok/drop
        """
        s = self.summary()
        conf = s["confidence_distribution"]
        paths = s["path_distribution"]
        b2 = s.get("b2_dispatch") or {}

        conf_parts = [
            f"{abbr}={conf[key]}"
            for key, abbr in (
                ("high", "H"),
                ("medium", "M"),
                ("low", "L"),
                ("sentinel", "s"),
                ("no_weapon", "?"),
            )
            if conf.get(key, 0)
        ]
        path_parts = [
            f"{abbr}={paths[key]}"
            for key, abbr in (("fire_event", "fe"), ("formula_a", "fa"), ("none", "s"))
            if paths.get(key, 0)
        ]
        b2_str = f" b2=[raw={b2['total']} ok={b2['dispatched']} drop={b2['dropped']}]" if b2 else ""
        logger.info(
            "match=%s\u2026 COMPLETE k=%d | %s | %s | warn=%d%s",
            self.match_id[:8],
            s["kills_decided"],
            " ".join(conf_parts) or "-",
            " ".join(path_parts) or "-",
            s["warnings"],
            b2_str,
        )
