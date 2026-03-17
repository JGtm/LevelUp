"""Helpers de détails inline pour le scoreboard de Match View."""

from __future__ import annotations

import functools
import html
import logging
import re
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path

import polars as pl

from src.analysis._medal_data import resolve_medal_name
from src.analysis._weapon_data import resolve_weapon_display
from src.config import get_bot_name, get_repo_root
from src.ui.i18n import get_lang, t
from src.ui.i18n.data_labels import label_obj
from src.utils import parse_xuid_input
from src.utils.paths import get_player_db_path, player_db_exists

logger = logging.getLogger(__name__)

_DETAIL_LIMIT_WEAPONS = 4
_DETAIL_LIMIT_MEDALS = 5
_DETAIL_LIMIT_CITATIONS = 4
_INVALID_GAMERTAGS = {"", "-", "?", "\u2014"}


@dataclass(slots=True)
class MedalDetailItem:
    """Médaille affichable dans le panneau scoreboard."""

    name: str
    count: int
    icon_url: str | None = None


@dataclass(slots=True)
class ScoreboardPlayerExtraData:
    """Données complémentaires affichées dans la ligne dépliée."""

    weapons: list[tuple[str, int]] = field(default_factory=list)
    medals: list[MedalDetailItem] = field(default_factory=list)
    citations: list[tuple[str, int]] = field(default_factory=list)
    performance_score: float | None = None
    had_bot_teammate: bool = False
    has_local_db: bool = False


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
    extra = ScoreboardPlayerExtraData()

    if not normalized_xuid:
        return extra

    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        shared_repo = DuckDBRepository(db_path, xuid=normalized_xuid, read_only=True)
        extra.weapons = _load_weapon_items(shared_repo, normalized_xuid, match_id, lang)
        extra.medals = _load_medal_items(shared_repo, match_id, lang)
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
    except Exception:
        logger.debug(
            "scoreboard detail: chargement local impossible match=%s xuid=%s db=%s",
            match_id,
            normalized_xuid,
            player_db_path,
            exc_info=True,
        )

    return extra


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
        sections.append(
            _render_items_section(
                title=t("mv_scoreboard_detail_weapons"),
                items=[(name, str(count)) for name, count in extra.weapons],
            )
        )
    if extra.medals:
        sections.append(_render_medals_section(extra.medals))
    local_items = _build_local_items(extra)
    if local_items:
        sections.append(
            _render_items_section(
                title=t("mv_scoreboard_detail_local"),
                items=local_items,
            )
        )

    return (
        "<div class='os-sb-detail-panel'>"
        "<div class='os-sb-detail-head'>"
        f"<div class='os-sb-detail-badge'>{html.escape(badge_text)}</div>"
        "</div>"
        "<div class='os-sb-detail-grid'>" + "".join(sections) + "</div>" + "</div>"
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


def _load_weapon_items(repo: object, xuid: str, match_id: str, lang: str) -> list[tuple[str, int]]:
    df = repo.load_weapon_kills_for_player(xuid, [match_id])
    if not isinstance(df, pl.DataFrame) or df.is_empty():
        return []
    aggregated = (
        df.group_by("weapon_id")
        .agg(pl.col("kills").sum().alias("kills"))
        .sort(["kills", "weapon_id"], descending=[True, False])
        .head(_DETAIL_LIMIT_WEAPONS)
    )
    items: list[tuple[str, int]] = []
    for row in aggregated.iter_rows(named=True):
        weapon_id = row.get("weapon_id")
        if weapon_id is None:
            continue
        weapon_name = resolve_weapon_display(int(weapon_id), lang=lang) or "-"
        if weapon_name == "-":
            continue
        items.append((weapon_name, int(row.get("kills") or 0)))
    return items


def _load_medal_items(repo: object, match_id: str, lang: str) -> list[MedalDetailItem]:
    rows = repo.load_match_medals(match_id)
    items = [
        MedalDetailItem(
            name=resolve_medal_name(int(row["name_id"]), lang=lang),
            count=int(row["count"]),
            icon_url=_medal_icon_url(int(row["name_id"])),
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

    items = []
    for norm, value in delta_map.items():
        if norm == "_processed" or int(value or 0) <= 0:
            continue
        label = label_obj("citations", norm, lang=lang) or {}
        items.append((str(label.get("name") or norm), int(value)))
    items.sort(key=lambda item: (-item[1], item[0]))
    return items[:_DETAIL_LIMIT_CITATIONS]


def _build_local_items(extra: ScoreboardPlayerExtraData) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    if extra.performance_score is not None:
        items.append((t("mv_performance"), f"{float(extra.performance_score):.1f}"))
    if extra.had_bot_teammate:
        items.append((t("mv_scoreboard_detail_bot_note"), t("mv_bot_teammate_note")))
    items.extend((name, str(count)) for name, count in extra.citations)
    return items


def _render_items_section(
    *,
    title: str,
    items: list[tuple[str, str]],
    pill_class: str = "os-sb-detail-item",
    value_class: str = "os-sb-detail-item-value",
) -> str:
    if not items:
        return ""
    rows = []
    for label, value in items:
        rows.append(
            f"<div class='{pill_class}'>"
            f"<span class='os-sb-detail-item-label'>{html.escape(str(label))}</span>"
            f"<span class='{value_class}'>{html.escape(str(value))}</span>"
            "</div>"
        )
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(title)}</div>"
        f"<div class='os-sb-detail-list'>{''.join(rows)}</div>"
        "</section>"
    )


@functools.cache
def _build_medal_icon_url_index() -> dict[int, str]:
    """Construit l'index medal_id -> URL statique Streamlit."""
    icons_dir = Path(get_repo_root()) / "static" / "medals" / "icons"
    index: dict[int, str] = {}
    if not icons_dir.exists():
        return index
    for icon_file in icons_dir.iterdir():
        if not icon_file.is_file() or icon_file.suffix.lower() != ".png":
            continue
        try:
            medal_id = int(icon_file.stem)
        except ValueError:
            continue
        index[medal_id] = f"/app/static/medals/icons/{icon_file.name}"
    return index


def _medal_icon_url(medal_id: int) -> str | None:
    """Retourne l'URL statique d'une icône de médaille si disponible."""
    return _build_medal_icon_url_index().get(int(medal_id))


def _render_medals_section(medals: Sequence[MedalDetailItem]) -> str:
    """Construit la section médailles avec icônes compactes."""
    rows = []
    for medal in medals:
        icon_html = ""
        if medal.icon_url:
            icon_html = (
                f"<img class='os-sb-detail-medal-icon' src='{html.escape(medal.icon_url)}' "
                f"alt='{html.escape(medal.name)}'>"
            )
        rows.append(
            "<div class='os-sb-detail-item os-sb-detail-item--medal'>"
            f"{icon_html}"
            f"<span class='os-sb-detail-item-label'>{html.escape(medal.name)}</span>"
            f"<span class='os-sb-detail-item-value'>{html.escape(str(medal.count))}</span>"
            "</div>"
        )
    return (
        "<section class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-title'>{html.escape(t('mv_medals'))}</div>"
        f"<div class='os-sb-detail-list'>{''.join(rows)}</div>"
        "</section>"
    )


__all__ = [
    "MedalDetailItem",
    "ScoreboardPlayerExtraData",
    "load_scoreboard_player_extra_data",
    "render_scoreboard_player_detail_html",
    "scoreboard_toggle_id",
]
