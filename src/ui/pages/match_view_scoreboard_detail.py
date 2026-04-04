"""Helpers de détails inline pour le scoreboard de Match View."""

from __future__ import annotations

import html
import logging
import re
from collections.abc import Sequence
from dataclasses import dataclass, field

import polars as pl

from src.analysis import compute_personal_antagonists_from_pairs_polars
from src.analysis._medal_data import resolve_medal_description, resolve_medal_name
from src.analysis._weapon_data import resolve_weapon_display
from src.config import get_bot_name
from src.ui.components.performance import get_score_class
from src.ui.i18n import get_lang, t
from src.ui.pages._scoreboard_asset_urls import medal_icon_url, weapon_asset_url
from src.ui.pages.match_table_html import app_url
from src.utils import parse_xuid_input
from src.utils.paths import get_player_db_path, player_db_exists

logger = logging.getLogger(__name__)

_DETAIL_LIMIT_MEDALS = 5
_DETAIL_LIMIT_CITATIONS = 4
_INVALID_GAMERTAGS = {"", "-", "?", "\u2014"}


@dataclass(slots=True)
class MedalDetailItem:
    """Médaille affichable dans le panneau scoreboard."""

    name: str
    count: int
    icon_url: str | None = None
    description: str | None = None


@dataclass(slots=True)
class WeaponDetailItem:
    """Arme affichable dans le panneau scoreboard."""

    name: str
    count: int
    asset_url: str | None = None


@dataclass(slots=True)
class ScoreboardPlayerExtraData:
    """Données complémentaires affichées dans la ligne dépliée."""

    weapons: list[WeaponDetailItem] = field(default_factory=list)
    medals: list[MedalDetailItem] = field(default_factory=list)
    antagonist_items: list[tuple[str, str]] = field(default_factory=list)
    citations: list[tuple[str, int, str | None]] = field(default_factory=list)
    # (label, actual, expected, higher_is_better)
    expected_items: list[tuple[str, float, float]] = field(default_factory=list)
    performance_score: float | None = None
    skill_rank: tuple[str, str] | None = None   # (label affiché, rating_type)
    had_bot_teammate: bool = False
    has_local_db: bool = False
    profile_link_html: str | None = None


def load_scoreboard_player_extra_data(
    *,
    db_path: str,
    match_id: str,
    xuid: str,
    gamertag: str | None,
) -> ScoreboardPlayerExtraData:
    """Charge les détails additionnels d'une ligne du scoreboard."""
    lang = get_lang()
    normalized_xuid = _normalize_xuid(xuid)
    clean_gamertag = _clean_gamertag(gamertag)
    extra = ScoreboardPlayerExtraData(profile_link_html=_build_profile_link(clean_gamertag))

    if not normalized_xuid:
        return extra

    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        shared_repo = DuckDBRepository(db_path, xuid=normalized_xuid, read_only=True)
        extra.weapons = _load_weapon_items(shared_repo, normalized_xuid, match_id, lang)
        extra.medals = _load_medal_items(shared_repo, match_id, lang)
        extra.antagonist_items = _load_antagonist_items(shared_repo, normalized_xuid, match_id)
        extra.expected_items = _load_expected_items(shared_repo, match_id)
    except Exception:
        logger.debug(
            "scoreboard detail: chargement shared impossible match=%s xuid=%s",
            match_id,
            normalized_xuid,
            exc_info=True,
        )

    player_db_path = _resolve_player_db_path(clean_gamertag)
    if player_db_path is None:
        return extra

    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        local_repo = DuckDBRepository(player_db_path, xuid=normalized_xuid, read_only=True)
        had_bot, performance_score, _dominance_flag = local_repo.load_player_match_enrichment(
            match_id
        )
        extra.has_local_db = True
        extra.had_bot_teammate = had_bot
        extra.performance_score = performance_score
        extra.citations = _load_citation_items(player_db_path, normalized_xuid, match_id, lang)
        extra.skill_rank = _load_skill_rank_item(player_db_path, match_id)
    except Exception:
        logger.debug(
            "scoreboard detail: chargement local impossible match=%s xuid=%s db=%s",
            match_id,
            normalized_xuid,
            player_db_path,
            exc_info=True,
        )

    return extra


def _render_expected_section(items: list[tuple[str, float, float]]) -> str:
    """Section Attendu vs Réel avec symbole ↑/↓ et delta coloré."""
    if not items:
        return ""
    rows = []
    for label, actual, expected, higher_is_better in items:
        delta = actual - expected
        better = (delta > 0) if higher_is_better else (delta < 0)
        symbol = "↑" if delta > 0 else ("↓" if delta < 0 else "→")
        color_class = "os-ev-better" if better else ("os-ev-worse" if delta != 0 else "os-ev-neutral")
        sign = "+" if delta > 0 else ""
        rows.append(
            "<div class='os-sb-detail-item--kv'>"
            f"<span class='os-sb-detail-item-label'>{html.escape(str(label))}</span>"
            f"<span class='os-sb-detail-item-value'>"
            f"<span class='os-ev-expected'>{expected:.1f} vs </span>"
            f"{actual:.0f} <span class='{color_class}'>{symbol} {sign}{delta:.1f}</span>"
            f"</span>"
            "</div>"
        )
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(t('mv_scoreboard_detail_expected'))}</div>"
        f"<div class='os-sb-detail-list--kv'>{''.join(rows)}</div>"
        "</section>"
    )


def render_scoreboard_player_detail_html(
    *,
    extra: ScoreboardPlayerExtraData,
) -> str:
    """Construit le panneau HTML affiché sous une ligne du scoreboard."""
    badge_text = (
        t("mv_scoreboard_detail_player_db")
        if extra.has_local_db
        else t("mv_scoreboard_detail_shared_only")
    )
    sections: list[str] = []
    if extra.weapons:
        sections.append(_render_weapons_section(extra.weapons))
    if extra.medals or extra.citations:
        sections.append(_render_medals_and_citations_section(extra.medals, extra.citations))
    if extra.expected_items:
        sections.append(_render_expected_section(extra.expected_items))
    if extra.antagonist_items:
        sections.append(
            _render_items_section(
                title=t("mv_scoreboard_detail_antagonist"),
                items=extra.antagonist_items,
            )
        )
    local_section = _render_local_section(extra)
    if local_section:
        sections.append(local_section)

    return (
        "<div class='os-sb-detail-panel'>"
        "<div class='os-sb-detail-grid'>"
        + "".join(sections)
        + "</div>"
        + _render_profile_footer(extra.profile_link_html, badge_text)
        + "</div>"
    )


def scoreboard_toggle_id(team_id: object, xuid: str, row_index: int) -> str:
    """Génère un identifiant HTML stable pour une ligne du scoreboard."""
    raw = f"{team_id}-{xuid or row_index}"
    slug = re.sub(r"[^a-z0-9_-]+", "-", raw.lower()).strip("-")
    return f"os-sb-toggle-{slug or row_index}"


def _normalize_xuid(xuid: str | None) -> str:
    raw = str(xuid or "").strip()
    normalized = parse_xuid_input(raw)
    return str(normalized or raw).strip()


def _clean_gamertag(gamertag: str | None) -> str:
    value = str(gamertag or "").strip()
    return "" if value in _INVALID_GAMERTAGS or value.isdigit() else value


def _resolve_player_db_path(gamertag: str) -> str | None:
    if not gamertag or get_bot_name(gamertag) is not None:
        return None
    if not player_db_exists(gamertag):
        return None
    return str(get_player_db_path(gamertag))


def _build_profile_link(gamertag: str) -> str | None:
    if not gamertag or get_bot_name(gamertag) is not None:
        return None
    href = html.escape(app_url("Explorer", gamertag=gamertag))
    label = html.escape(t("mv_scoreboard_detail_explore_player", player=gamertag))
    return f"<a class='os-sb-detail-link' href='{href}' target='_self'>{label}</a>"


def _load_weapon_items(repo: object, xuid: str, match_id: str, lang: str) -> list[WeaponDetailItem]:
    from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS, WEAPON_FUSION_MAP_ID

    df = repo.load_weapon_kills_for_player(xuid, [match_id])
    if not isinstance(df, pl.DataFrame) or df.is_empty():
        return []

    fused = (
        df.filter(pl.col("kills") > 0)
        .filter(~pl.col("weapon_id").is_in(list(EXCLUDED_WEAPON_IDS)))
        .with_columns(pl.col("weapon_id").replace(WEAPON_FUSION_MAP_ID).alias("weapon_id"))
        .group_by("weapon_id")
        .agg(pl.col("kills").sum().alias("kills"))
        .sort(["kills", "weapon_id"], descending=[True, False])
    )

    items: list[WeaponDetailItem] = []
    for row in fused.iter_rows(named=True):
        weapon_id = row.get("weapon_id")
        if weapon_id is None:
            continue
        weapon_name = resolve_weapon_display(int(weapon_id), lang=lang) or "-"
        if weapon_name == "-":
            continue
        items.append(
            WeaponDetailItem(
                name=weapon_name,
                count=int(row.get("kills") or 0),
                asset_url=weapon_asset_url(weapon_name),
            )
        )

    # Enrichissement grenade / mêlée depuis match_participants (API).
    # Limité au remainder (api_total - film_kills) pour éviter le double-comptage
    # des melee kills filmés sous le weapon_id de l'arme tenue.
    try:
        grenade, melee = repo.load_grenade_melee_kills(str(xuid).strip(), [match_id])
        film_kills = sum(item.count for item in items)
        api_total = repo.load_total_kills_for_player(str(xuid).strip(), [match_id])
        remainder = max(0, api_total - film_kills)
        melee_net = min(melee, remainder)
        grenade_net = min(grenade, max(0, remainder - melee_net))
        if grenade_net > 0:
            items.append(
                WeaponDetailItem(
                    name=t("col_grenade_kills"),
                    count=grenade_net,
                    asset_url=weapon_asset_url("Grenade"),
                )
            )
        if melee_net > 0:
            items.append(
                WeaponDetailItem(
                    name=t("col_melee"),
                    count=melee_net,
                    asset_url=weapon_asset_url("Melee"),
                )
            )
    except Exception:
        pass

    return items


def _load_medal_items(repo: object, match_id: str, lang: str) -> list[MedalDetailItem]:
    rows = repo.load_match_medals(match_id)
    items = [
        MedalDetailItem(
            name=resolve_medal_name(int(row["name_id"]), lang=lang),
            count=int(row["count"]),
            icon_url=medal_icon_url(int(row["name_id"])),
            description=resolve_medal_description(int(row["name_id"]), lang=lang),
        )
        for row in rows
        if row.get("name_id") is not None and int(row.get("count") or 0) > 0
    ]
    items.sort(key=lambda item: (-item.count, item.name))
    return items[:_DETAIL_LIMIT_MEDALS]


def _load_citation_items(
    player_db_path: str,
    xuid: str,
    match_id: str,
    lang: str,
) -> list[tuple[str, int]]:
    try:
        from src.analysis.citations.engine import CitationEngine

        engine = CitationEngine(player_db_path, xuid)
        delta_map = engine.aggregate_for_display(match_ids=[match_id])
    except Exception:
        logger.debug(
            "scoreboard detail: citations indisponibles match=%s xuid=%s",
            match_id,
            xuid,
            exc_info=True,
        )
        return []

    items: list[tuple[str, int, str | None]] = []
    from src.data.citation_definitions import load_citation_definitions

    citations_db = load_citation_definitions()
    norm_to_meta: dict[str, dict] = {c["citation_name_norm"]: c for c in citations_db}
    for norm, value in delta_map.items():
        if norm == "_processed" or int(value or 0) <= 0:
            continue
        meta = norm_to_meta.get(norm, {})
        name = meta.get("citation_name_display", norm)
        raw_path = meta.get("image_path") or ""
        icon_url = f"/app/{raw_path}" if raw_path else None
        items.append((name, int(value), icon_url))
    items.sort(key=lambda item: (-item[1], item[0]))
    return items[:_DETAIL_LIMIT_CITATIONS]


def _load_expected_items(
    repo: object,
    match_id: str,
) -> list[tuple[str, float, float]]:
    """Charge kills/deaths/assists attendus vs réels depuis shared.match_participants.

    Returns:
        Liste de (label, actual, expected, higher_is_better).
    """
    try:
        skill_data = repo.load_match_skill_data(match_id)
    except Exception:
        logger.debug("scoreboard detail: expected indisponible match=%s", match_id, exc_info=True)
        return []
    if not skill_data:
        return []
    items: list[tuple[str, float, float]] = []
    for key, label_key, higher_is_better in (
        ("kills", "tm_kills", True),
        ("deaths", "tm_deaths", False),
        ("assists", "tm_assists", True),
    ):
        stat = skill_data.get(key) or {}
        actual = stat.get("count")
        expected = stat.get("expected")
        if actual is None or expected is None:
            continue
        items.append((t(label_key), float(actual), float(expected), higher_is_better))  # type: ignore[arg-type]
    return items


def _load_antagonist_items(repo: object, xuid: str, match_id: str) -> list[tuple[str, str]]:
    pairs_df = repo.load_killer_victim_pairs_as_polars(match_id=match_id)
    if not isinstance(pairs_df, pl.DataFrame) or pairs_df.is_empty():
        return []
    result = compute_personal_antagonists_from_pairs_polars(pairs_df, xuid)
    items: list[tuple[str, str]] = []
    if result.nemesis_times_killed_by > 0:
        items.append(
            (
                t("mv_scoreboard_detail_nemesis"),
                _format_antagonist_value(result.nemesis_gamertag, result.nemesis_times_killed_by),
            )
        )
    if result.victim_times_killed > 0:
        items.append(
            (
                t("mv_scoreboard_detail_bully"),
                _format_antagonist_value(result.victim_gamertag, result.victim_times_killed),
            )
        )
    return items


def _load_skill_rank_item(
    player_db_path: str,
    match_id: str,
) -> tuple[str, str] | None:
    """Retourne (label_affiché, rating_type) depuis match_skill_rank, ou None."""
    from src.utils.db import duckdb_read_only

    try:
        with duckdb_read_only(player_db_path) as conn:
            row = conn.execute(
                "SELECT rating_type, rating_value, tier_label, rating_delta "
                "FROM match_skill_rank WHERE match_id = ?",
                (match_id,),
            ).fetchone()
        if row is None:
            return None
        rating_type, rating_value, tier_label, rating_delta = row
        label = tier_label or f"{rating_value:.0f}"
        if rating_delta is not None:
            sign = "+" if rating_delta >= 0 else ""
            label = f"{label} ({sign}{rating_delta:.0f} pts)"
        return (label, str(rating_type))
    except Exception:
        logger.debug("scoreboard detail: skill_rank indisponible match=%s", match_id, exc_info=True)
        return None


def _format_antagonist_value(gamertag: str | None, count: int) -> str:
    name = _clean_gamertag(gamertag) or "—"
    return f"{name} ({int(count)})"


def _build_local_items(extra: ScoreboardPlayerExtraData) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    if extra.skill_rank is not None:
        label, rating_type = extra.skill_rank
        key = "mv_scoreboard_detail_lusr" if rating_type == "LUSR" else "mv_scoreboard_detail_csr"
        items.append((t(key), label))
    if extra.had_bot_teammate:
        items.append((t("mv_scoreboard_detail_bot_note"), t("mv_bot_teammate_note")))
    return items


def _render_local_section(extra: ScoreboardPlayerExtraData) -> str:
    rows: list[str] = []
    if extra.performance_score is not None:
        score_value = f"{float(extra.performance_score):.1f}"
        score_class = get_score_class(extra.performance_score)
        rows.append(
            "<div class='os-sb-detail-item--kv'>"
            f"<span class='os-sb-detail-item-label'>{html.escape(str(t('mv_performance')))}</span>"
            f"<span class='os-sb-detail-item-value {html.escape(score_class)}'>{html.escape(score_value)}</span>"
            "</div>"
        )
    for label, value in _build_local_items(extra):
        rows.append(
            "<div class='os-sb-detail-item--kv'>"
            f"<span class='os-sb-detail-item-label'>{html.escape(str(label))}</span>"
            f"<span class='os-sb-detail-item-value'>{html.escape(str(value))}</span>"
            "</div>"
        )
    if not rows:
        return ""
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(t('mv_scoreboard_detail_local'))}</div>"
        f"<div class='os-sb-detail-list--kv'>{''.join(rows)}</div>"
        "</section>"
    )


def _render_items_section(
    *,
    title: str,
    items: list[tuple[str, str]],
) -> str:
    if not items:
        return ""
    rows = []
    for label, value in items:
        rows.append(
            "<div class='os-sb-detail-item--kv'>"
            f"<span class='os-sb-detail-item-label'>{html.escape(str(label))}</span>"
            f"<span class='os-sb-detail-item-value'>{html.escape(str(value))}</span>"
            "</div>"
        )
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(title)}</div>"
        f"<div class='os-sb-detail-list--kv'>{''.join(rows)}</div>"
        "</section>"
    )


def _render_profile_footer(profile_link_html: str | None, badge_text: str) -> str:
    """Construit la ligne footer avec badge DB et lien Explorer."""
    badge_html = f"<div class='os-sb-detail-badge'>{html.escape(badge_text)}</div>"
    if not profile_link_html:
        return f"<div class='os-sb-detail-footer'>{badge_html}</div>"
    return f"<div class='os-sb-detail-footer'>{badge_html}{profile_link_html}</div>"


def _render_medal_item(medal: MedalDetailItem) -> str:
    tooltip_text = html.escape(f"{medal.name} : {medal.description}" if medal.description else medal.name)
    icon_html = (
        f"<div class='os-sb-detail-medal-icon-wrap'>"
        f"<img class='os-sb-detail-medal-icon' src='{html.escape(medal.icon_url)}' "
        f"alt='{html.escape(medal.name)}' title='{tooltip_text}'>"
        f"</div>"
        if medal.icon_url
        else ""
    )
    return (
        f"<div class='os-sb-detail-item os-sb-detail-item--medal' title='{tooltip_text}'>"
        f"{icon_html}"
        f"<span class='os-sb-detail-item-value'>×{html.escape(str(medal.count))}</span>"
        "</div>"
    )


def _render_citation_item(name: str, count: int, icon_url: str | None) -> str:
    tooltip_text = html.escape(name)
    icon_html = (
        f"<div class='os-sb-detail-medal-icon-wrap'>"
        f"<img class='os-sb-detail-medal-icon os-sb-detail-citation-icon' src='{html.escape(icon_url)}' "
        f"alt='{tooltip_text}' title='{tooltip_text}'>"
        f"</div>"
        if icon_url
        else ""
    )
    return (
        f"<div class='os-sb-detail-item os-sb-detail-item--medal' title='{tooltip_text}'>"
        f"{icon_html}"
        f"<span class='os-sb-detail-item-value'>×{html.escape(str(count))}</span>"
        "</div>"
    )


def _render_medals_and_citations_section(
    medals: Sequence[MedalDetailItem],
    citations: list[tuple[str, int, str | None]],
) -> str:
    """Construit la section Médailles & Citations avec icônes compactes."""
    medal_rows = [_render_medal_item(m) for m in medals]
    citation_rows = [_render_citation_item(name, count, icon_url) for name, count, icon_url in citations]
    if not medal_rows and not citation_rows:
        return ""
    # Séparateur invisible pleine largeur pour forcer les citations sur une nouvelle ligne
    separator = "<div class='os-sb-detail-list-break'></div>" if medal_rows and citation_rows else ""
    title = t("mv_medals_and_citations") if citations else t("mv_medals")
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(title)}</div>"
        f"<div class='os-sb-detail-list os-sb-detail-list--medals'>"
        f"{''.join(medal_rows)}{separator}{''.join(citation_rows)}"
        f"</div>"
        "</section>"
    )


def _render_weapons_section(weapons: Sequence[WeaponDetailItem]) -> str:
    """Construit la section armes avec assets, sans effet pastille."""
    rows = []
    for weapon in weapons:
        asset_html = ""
        if weapon.asset_url:
            asset_html = (
                f"<img class='os-sb-detail-weapon-asset' src='{html.escape(weapon.asset_url)}' "
                f"alt='{html.escape(weapon.name)}' title='{html.escape(weapon.name)}'>"
            )
        else:
            asset_html = (
                f"<span class='os-sb-detail-weapon-fallback' title='{html.escape(weapon.name)}'>"
                f"{html.escape(weapon.name)}"
                "</span>"
            )
        rows.append(
            "<div class='os-sb-detail-item os-sb-detail-item--weapon'>"
            f"{asset_html}"
            f"<span class='os-sb-detail-item-value'>{html.escape(str(weapon.count))}</span>"
            "</div>"
        )
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(t('mv_scoreboard_detail_weapons'))}</div>"
        f"<div class='os-sb-detail-list os-sb-detail-list--weapons'>{''.join(rows)}</div>"
        "</section>"
    )


__all__ = [
    "MedalDetailItem",
    "ScoreboardPlayerExtraData",
    "WeaponDetailItem",
    "load_scoreboard_player_extra_data",
    "render_scoreboard_player_detail_html",
    "scoreboard_toggle_id",
]
