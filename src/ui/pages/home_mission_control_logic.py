"""Logique pure de l'accueil Mission Control V7."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import polars as pl

from src.app.kpis import KPIStats, compute_kpi_stats
from src.app.session_keys import SK
from src.ui.formatting import format_datetime_fr_hm, format_duration_dhm
from src.ui.i18n import get_lang, t
from src.visualization._compat import ensure_polars


@dataclass(frozen=True)
class HomeSessionSummary:
    """Résumé compact d'une session récente."""

    session_label: str
    started_at: object | None
    match_count: int
    kpis: KPIStats


@dataclass(frozen=True)
class HomeMediaEntry:
    """Entrée compacte de média récent."""

    basename: str
    match_id: str | None
    match_start_time: object | None


@dataclass(frozen=True)
class HomeTrendSnapshot:
    """Variation récente sur une fenêtre glissante."""

    current_kpis: KPIStats
    ratio_delta: float | None
    accuracy_delta: float | None
    win_rate_delta: float | None


@dataclass(frozen=True)
class HomeActionCard:
    """Carte d'action rapide depuis l'accueil."""

    title: str
    description: str
    button_label: str
    button_key: str
    target_section: str
    stats_view: str | None = None
    session_label: str | None = None
    squad_mode: bool | None = None
    pending_match_id: str | None = None


@dataclass(frozen=True)
class HomeHighlight:
    """Fait saillant synthétique pour la zone centrale."""

    title: str
    value: str
    detail: str


@dataclass(frozen=True)
class HomeRecentMatch:
    """Match récent affiché dans la timeline de l'accueil."""

    match_id: str
    title: str
    detail: str
    started_at: object | None
    outcome_label: str
    outcome_tone: str


@dataclass(frozen=True)
class SessionCardConfig:
    """Configuration de rendu d'une carte session."""

    title: str
    empty_text: str
    button_label: str
    button_key: str
    target_section: str
    squad_mode: bool


def _format_ratio(value: float | None) -> str:
    """Formate un ratio avec fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.2f}"


def _format_percent(value: float | None) -> str:
    """Formate un pourcentage avec fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.0f}%"


def _format_signed(
    value: float | None,
    *,
    precision: int,
    suffix: str = "",
    scale: float = 1.0,
) -> str:
    """Formate une variation signée."""
    if value is None:
        return "-"
    scaled = value * scale
    sign = "+" if scaled > 0 else ""
    return f"{sign}{scaled:.{precision}f}{suffix}"


def _coerce_float(value: Any) -> float | None:
    """Convertit une valeur en float si possible."""
    if value is None:
        return None
    try:
        coerced = float(value)
    except Exception:
        return None
    if coerced != coerced:
        return None
    return coerced


def _pick_label(row: dict[str, Any], *keys: str) -> str:
    """Retourne le premier libellé non vide parmi plusieurs colonnes."""
    for key in keys:
        value = str(row.get(key) or "").strip()
        if value:
            return value
    return "-"


def _format_outcome(outcome: Any) -> tuple[str, str]:
    """Retourne le libellé et le ton d'issue d'un match."""
    lang = get_lang()
    labels = {
        "win": ("Victoire", "Victory"),
        "loss": ("Défaite", "Loss"),
        "tie": ("Égalité", "Tie"),
        "dnf": ("DNF", "DNF"),
    }
    normalized = str(outcome).strip().lower()
    if normalized in {"2", "win", "victory", "victoire"}:
        key = "win"
    elif normalized in {"3", "loss", "defeat", "défaite"}:
        key = "loss"
    elif normalized in {"1", "tie", "draw", "égalité"}:
        key = "tie"
    else:
        key = "dnf"
    return labels[key][0 if lang == "fr" else 1], key


def _build_navigation_state(
    section: str,
    *,
    stats_view: str | None = None,
    session_label: str | None = None,
    squad_mode: bool | None = None,
    pending_match_id: str | None = None,
) -> dict[str, object]:
    """Construit les mutations de session_state nécessaires à une navigation V7."""
    updates: dict[str, object] = {SK.V7_CURRENT_SECTION: section}
    if stats_view is not None:
        updates[SK.V7_STATS_VIEW] = stats_view
    if session_label:
        updates[SK.FILTER_MODE] = "Sessions"
        updates[SK.PICKED_SESSION_LABEL] = session_label
        updates[SK.PICKED_SESSIONS] = [session_label]
        if squad_mode is True:
            updates[SK.PICKED_SQUAD_SESSION_LABEL] = session_label
            updates[SK.PICKED_SOLO_SESSION_LABEL] = "(toutes)"
        elif squad_mode is False:
            updates[SK.PICKED_SOLO_SESSION_LABEL] = session_label
            updates[SK.PICKED_SQUAD_SESSION_LABEL] = "(toutes)"
    if pending_match_id:
        updates[SK.PENDING_MATCH_ID] = pending_match_id
    return updates


def _get_scope_sessions(sessions_df: Any, *, squad_mode: bool) -> pl.DataFrame:
    """Retourne les sessions d'un scope donné."""
    if sessions_df is None:
        return pl.DataFrame()
    sessions_pl = ensure_polars(sessions_df)
    if sessions_pl.is_empty():
        return sessions_pl

    required = {"match_id", "session_label", "start_time"}
    if not required.issubset(set(sessions_pl.columns)):
        return pl.DataFrame()

    if "is_with_friends" not in sessions_pl.columns:
        return pl.DataFrame() if squad_mode else sessions_pl.drop_nulls(subset=["session_label"])

    return sessions_pl.filter(pl.col("is_with_friends") == squad_mode).drop_nulls(
        subset=["session_label"]
    )


def _build_session_summary(
    matches_df: Any,
    sessions_df: Any,
    *,
    squad_mode: bool,
) -> HomeSessionSummary | None:
    """Construit le résumé de la dernière session d'un scope."""
    matches_pl = ensure_polars(matches_df)
    scope_sessions = _get_scope_sessions(sessions_df, squad_mode=squad_mode)
    if matches_pl.is_empty() or scope_sessions.is_empty() or "match_id" not in matches_pl.columns:
        return None

    latest_session = (
        scope_sessions.group_by("session_label")
        .agg(pl.col("start_time").max().alias("last_start"))
        .sort("last_start", descending=True)
    )
    if latest_session.is_empty():
        return None

    latest_label = latest_session.row(0, named=True).get("session_label")
    if latest_label is None:
        return None

    session_match_ids = (
        scope_sessions.filter(pl.col("session_label") == latest_label)
        .get_column("match_id")
        .drop_nulls()
        .unique()
        .to_list()
    )
    if not session_match_ids:
        return None

    session_matches = matches_pl.filter(pl.col("match_id").is_in(session_match_ids))
    if session_matches.is_empty():
        return None

    started_at = None
    if "start_time" in session_matches.columns:
        started_at = session_matches.select(pl.col("start_time").min()).item()

    return HomeSessionSummary(
        session_label=str(latest_label),
        started_at=started_at,
        match_count=len(session_matches),
        kpis=compute_kpi_stats(session_matches),
    )


def _compute_trend_snapshot(matches_df: Any, window_size: int = 5) -> HomeTrendSnapshot | None:
    """Calcule une variation récente par fenêtre glissante."""
    matches_pl = ensure_polars(matches_df)
    if matches_pl.is_empty():
        return None
    if "start_time" in matches_pl.columns:
        matches_pl = matches_pl.sort("start_time", descending=True)

    current_window = matches_pl.head(window_size)
    current_kpis = compute_kpi_stats(current_window)
    previous_window = matches_pl.slice(window_size, window_size)
    if previous_window.is_empty():
        return HomeTrendSnapshot(current_kpis, None, None, None)

    previous_kpis = compute_kpi_stats(previous_window)
    ratio_delta = None
    if current_kpis.global_ratio is not None and previous_kpis.global_ratio is not None:
        ratio_delta = current_kpis.global_ratio - previous_kpis.global_ratio
    accuracy_delta = None
    if current_kpis.avg_accuracy is not None and previous_kpis.avg_accuracy is not None:
        accuracy_delta = current_kpis.avg_accuracy - previous_kpis.avg_accuracy
    win_rate_delta = current_kpis.win_rate - previous_kpis.win_rate
    return HomeTrendSnapshot(current_kpis, ratio_delta, accuracy_delta, win_rate_delta)


def _select_recent_media(media_df: Any, limit: int = 3) -> list[HomeMediaEntry]:
    """Sélectionne les médias récents à afficher dans l'accueil."""
    media_pl = ensure_polars(media_df)
    if media_pl.is_empty() or "basename" not in media_pl.columns:
        return []

    sort_column = "mtime_paris_epoch" if "mtime_paris_epoch" in media_pl.columns else None
    if sort_column is None and "match_start_time" in media_pl.columns:
        sort_column = "match_start_time"

    if sort_column:
        media_pl = media_pl.sort(sort_column, descending=True)

    media_pl = media_pl.unique(
        subset=["path"] if "path" in media_pl.columns else ["basename"],
        keep="first",
        maintain_order=True,
    )
    rows = media_pl.head(limit).iter_rows(named=True)
    return [
        HomeMediaEntry(
            basename=str(row.get("basename") or "-"),
            match_id=str(row.get("match_id") or "").strip() or None,
            match_start_time=row.get("match_start_time"),
        )
        for row in rows
    ]


def _select_recent_matches(matches_df: Any, limit: int = 5) -> list[HomeRecentMatch]:
    """Sélectionne les derniers matchs pour la timeline de l'accueil."""
    matches_pl = ensure_polars(matches_df)
    if matches_pl.is_empty() or "match_id" not in matches_pl.columns:
        return []
    if "start_time" in matches_pl.columns:
        matches_pl = matches_pl.sort("start_time", descending=True)

    entries: list[HomeRecentMatch] = []
    for row in matches_pl.head(limit).iter_rows(named=True):
        match_id = str(row.get("match_id") or "").strip()
        if not match_id:
            continue
        outcome_label, outcome_tone = _format_outcome(row.get("outcome"))
        map_label = _pick_label(row, "map_name_fr", "map_name")
        mode_label = _pick_label(
            row,
            "pair_name",
            "mode_name_fr",
            "mode_name",
            "playlist_name_fr",
            "playlist_name",
        )
        ratio_txt = _format_ratio(_coerce_float(row.get("ratio")))
        accuracy_txt = _format_percent(_coerce_float(row.get("accuracy")))
        entries.append(
            HomeRecentMatch(
                match_id=match_id,
                title=f"{outcome_label} · {map_label}",
                detail=f"{mode_label} · KD {ratio_txt} · {accuracy_txt}",
                started_at=row.get("start_time"),
                outcome_label=outcome_label,
                outcome_tone=outcome_tone,
            )
        )
    return entries


def _build_recent_highlights(matches_df: Any, sessions_df: Any) -> list[HomeHighlight]:
    """Construit les faits saillants récents affichés sur l'accueil."""
    matches_pl = ensure_polars(matches_df)
    if matches_pl.is_empty():
        return []
    if "start_time" in matches_pl.columns:
        matches_pl = matches_pl.sort("start_time", descending=True)

    highlights: list[HomeHighlight] = []
    recent_window = matches_pl.head(8)
    if "ratio" in recent_window.columns:
        best_row = recent_window.sort("ratio", descending=True, nulls_last=True).row(0, named=True)
        best_ratio = _format_ratio(_coerce_float(best_row.get("ratio")))
        best_map = _pick_label(best_row, "map_name_fr", "map_name")
        best_mode = _pick_label(best_row, "pair_name", "mode_name_fr", "mode_name")
        best_date = format_datetime_fr_hm(best_row.get("start_time"), lang=get_lang())
        highlights.append(
            HomeHighlight(
                title=t("v7_home_highlight_peak"),
                value=f"KD {best_ratio}",
                detail=f"{best_map} · {best_mode} · {best_date}",
            )
        )

    trend_snapshot = _compute_trend_snapshot(matches_pl)
    if trend_snapshot and trend_snapshot.ratio_delta is not None:
        highlights.append(
            HomeHighlight(
                title=t("v7_home_highlight_trend"),
                value=f"KD {_format_signed(trend_snapshot.ratio_delta, precision=2)}",
                detail=(
                    f"ACC {_format_signed(trend_snapshot.accuracy_delta, precision=0, suffix='%')}"
                    f" · WR {_format_signed(trend_snapshot.win_rate_delta, precision=0, suffix='%', scale=100)}"
                ),
            )
        )

    squad_summary = _build_session_summary(matches_pl, sessions_df, squad_mode=True)
    if squad_summary is not None:
        highlights.append(
            HomeHighlight(
                title=t("v7_home_highlight_squad"),
                value=f"{squad_summary.match_count} {t('lbl_parties')}",
                detail=f"{squad_summary.session_label} · WR {_format_percent(squad_summary.kpis.win_rate * 100)}",
            )
        )

    if len(highlights) < 3:
        recent_stats = compute_kpi_stats(matches_pl.head(10))
        highlights.append(
            HomeHighlight(
                title=t("v7_home_highlight_volume"),
                value=f"{len(matches_pl.head(10))} {t('lbl_parties')}",
                detail=(
                    f"KD {_format_ratio(recent_stats.global_ratio)}"
                    f" · {format_duration_dhm(recent_stats.total_play_seconds, lang=get_lang())}"
                ),
            )
        )
    return highlights[:3]
