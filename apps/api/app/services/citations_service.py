"""Service Citations — construit la réponse `CitationsPageResponse`.

Toutes les importations src.* sont lazy pour permettre le mocking en tests.
"""

from __future__ import annotations

import logging
from typing import Any

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.citations import (
    CitationsDeltas,
    CitationsPageResponse,
    CitationsQueryRequest,
    CommendationSummary,
    MedalSummary,
)

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Point d'entrée public
# ---------------------------------------------------------------------------


def get_citations_page(
    player: PlayerContext, request: CitationsQueryRequest
) -> CitationsPageResponse:
    """Construit la page Citations (commendations + médailles + deltas)."""
    from apps.api.app.services.filter_service import _apply_polars_filters, _load_base_matches
    from src.data.repositories.duckdb_repo import DuckDBRepository

    with DuckDBRepository(
        player.db_path,
        xuid=player.xuid,
        shared_db_path=player.shared_db_path,
        metadata_db_path=player.metadata_db_path,
        read_only=True,
    ) as repo:
        # Charger les matchs filtrés et non filtrés
        df_full = _load_base_matches(player)
        dff = _apply_polars_filters(df_full, player, request.filters)

        all_match_ids = df_full["match_id"].cast(str).to_list() if not df_full.is_empty() else []
        filtered_match_ids = dff["match_id"].cast(str).to_list() if not dff.is_empty() else []

        commendations = _build_commendations(player.db_path, player.xuid, filtered_match_ids)
        medals_filtered = _load_medals_summary(repo, filtered_match_ids, player.metadata_db_path)
        medals_total = _load_medals_summary(repo, all_match_ids, player.metadata_db_path)
        medals = _merge_medal_summaries(medals_filtered, medals_total)

        filtered_total = sum(m.count_filtered for m in medals)
        unfiltered_total = sum(m.count_total for m in medals)
        deltas = CitationsDeltas(
            filtered_total=filtered_total,
            unfiltered_total=unfiltered_total,
            delta_count=unfiltered_total - filtered_total,
        )

    return CitationsPageResponse(
        commendations=commendations,
        medals_summary=medals,
        deltas=deltas,
        distribution_chart=None,
    )


# ---------------------------------------------------------------------------
# Commendations (Halo 5 Guardians via CitationEngine)
# ---------------------------------------------------------------------------


def _build_commendations(
    db_path: str,
    xuid: str,
    filtered_match_ids: list[str],
) -> list[CommendationSummary]:
    """Charge et formate les commendations progressées dans le scope filtré."""
    try:
        from src.analysis.citations.engine import CitationEngine
        from src.data.citation_definitions import load_citation_definitions

        defs = load_citation_definitions()
        engine = CitationEngine(db_path, xuid)

        # Agrégation sur le scope filtré
        delta_map = engine.aggregate_for_display(match_ids=filtered_match_ids)
        # Agrégation totale pour mastery_pct
        full_map = engine.aggregate_for_display(match_ids=None)

    except Exception as exc:
        logger.debug("_build_commendations: erreur engine %s", exc)
        return []

    active = {norm: val for norm, val in delta_map.items() if val > 0}
    if not active:
        return []

    defs_dict: dict[str, Any] = {}
    if isinstance(defs, dict):
        defs_dict = defs
    elif isinstance(defs, list):
        for d in defs:
            if isinstance(d, dict) and "name" in d:
                defs_dict[d["name"]] = d

    result: list[CommendationSummary] = []
    for norm, val in sorted(active.items(), key=lambda x: x[1], reverse=True):
        defn = defs_dict.get(norm, {})
        total_val = full_map.get(norm, 0)

        mastery_pct: float | None = None
        # Calcul du taux de maîtrise si on a une valeur cible
        tier_targets_raw = defn.get("tier_targets") or defn.get("targets") or ""
        try:
            from src.ui.commendations import _parse_tier_targets

            tiers = _parse_tier_targets(str(tier_targets_raw))
            if tiers:
                max_tier = max(t.get("target", 0) for t in tiers)
                if max_tier > 0:
                    mastery_pct = min(round(total_val / max_tier * 100, 1), 100.0)
        except Exception:
            pass

        result.append(
            CommendationSummary(
                key=norm,
                label=str(defn.get("label") or norm),
                category=str(defn.get("category") or ""),
                current_value=int(val),
                color=defn.get("color"),
                icon_path=defn.get("image_path"),
                tier_label=defn.get("tier_label"),
                mastery_pct=mastery_pct,
            )
        )

    return result


# ---------------------------------------------------------------------------
# Médailles (Halo Infinite via medals_earned)
# ---------------------------------------------------------------------------


def _load_medals_summary(
    repo: Any,
    match_ids: list[str],
    metadata_db_path: str,
) -> dict[int, int]:
    """Retourne {medalNameId: total_count} pour les matchs donnés."""
    if not match_ids:
        return {}

    try:
        raw = repo.load_top_medals(match_ids, top_n=None)
        return {int(medal_id): int(count) for medal_id, count in raw}
    except Exception as exc:
        logger.debug("_load_medals_summary: %s", exc)
        return {}


def _merge_medal_summaries(
    filtered: dict[int, int],
    total: dict[int, int],
) -> list[MedalSummary]:
    """Fusionne les compteurs filtrés et totaux, résout les noms."""
    all_ids = set(filtered) | set(total)
    if not all_ids:
        return []

    name_map: dict[int, str] = {}
    desc_map: dict[int, str] = {}
    try:
        from src.ui.medals import load_medal_description_map, load_medal_name_maps

        raw_names, _ = load_medal_name_maps("fr")
        name_map = {int(k): v for k, v in (raw_names or {}).items()}
        desc_raw = load_medal_description_map("fr")
        desc_map = {int(k): v for k, v in (desc_raw or {}).items()}
    except Exception as exc:
        logger.debug("_merge_medal_summaries: erreur noms %s", exc)

    result: list[MedalSummary] = []
    for medal_id in all_ids:
        count_filt = filtered.get(medal_id, 0)
        count_tot = total.get(medal_id, 0)
        result.append(
            MedalSummary(
                medal_name_id=medal_id,
                name=name_map.get(medal_id, str(medal_id)),
                count_filtered=count_filt,
                count_total=count_tot,
                description=desc_map.get(medal_id),
            )
        )

    result.sort(key=lambda m: m.count_filtered, reverse=True)
    return result
