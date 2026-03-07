"""Page Carrière — Progression du rang Halo Infinite."""

from __future__ import annotations

import html
import logging
from datetime import datetime, timedelta
from pathlib import Path

import streamlit as st

from src.ui.career_ranks import (
    format_career_rank_label_fr,
    get_rank_icon_path,
)
from src.ui.career_ranks import (
    get_rank_info as get_meta_rank_info,
)
from src.ui.chart_utils import safe_chart_render
from src.ui.components.career_progress_circle import (
    RANK_MAX,
    XP_HERO_TOTAL,
    compute_hero_progress,
    create_career_progress_gauge,
    create_hero_progress_gauge,
)
from src.ui.i18n import t
from src.ui.pages.career_data import (
    _load_career_data,
    _load_career_history,
    _load_lusr_history,
    _load_lusr_snapshot,
    _load_other_players_histories,
    _load_pre_sync_match_dates,
)
from src.ui.pages.career_encounters_render import render_encounters_section
from src.ui.pages.career_logic import (
    _compute_active_xp_per_day,
    _compute_estimated_xp_curve,
    _compute_hero_projections,
    _create_xp_history_chart,
    _get_pg_labels,
)
from src.ui.player_assets import ensure_local_image_path
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available

logger = logging.getLogger(__name__)

_PG_ICONS: dict[str, str] = {
    "ranked": "🏆",
    "arena": "⚔️",
    "btb": "💥",
    "tactical": "🎯",
    "social": "🎮",
    "fun": "🎉",
}
_PG_ORDER = ["ranked", "arena", "btb", "tactical", "social", "fun"]


def _render_lusr_rank_cards(ordered: list, project_root: Path) -> None:  # noqa: PLR0912
    """Rend la grille de cartes visuelles LUSR/CSR (rang, tier, delta)."""
    from src.analysis.skill_rating_config import get_rank_image_path

    pg_labels = _get_pg_labels()
    n_per_row = 3
    for row_start in range(0, len(ordered), n_per_row):
        batch = ordered[row_start : row_start + n_per_row]
        cols = st.columns(len(batch))
        for col, snap in zip(cols, batch, strict=False):
            pg = snap["playlist_group"] or "?"
            tier_label = snap["tier_label"] or t("unranked")
            r_value = float(snap["rating_value"] or 0.0)
            delta = snap["rating_delta"]
            r_type = (
                "CSR" if snap.get("playlist_group") == "ranked" else (snap["rating_type"] or "LUSR")
            )
            with col, st.container(border=True):
                pg_label = pg_labels.get(pg, pg.capitalize())
                pg_icon = _PG_ICONS.get(pg, "🎮")
                img_b64 = ""
                img_path_rel = get_rank_image_path(r_value)
                if img_path_rel:
                    img_full = project_root / img_path_rel
                    if img_full.exists():
                        import base64

                        img_b64 = base64.b64encode(img_full.read_bytes()).decode()
                img_html = (
                    f"<img src='data:image/png;base64,{img_b64}' "
                    f"style='width:90px;height:90px;object-fit:contain' />"
                    if img_b64
                    else "<div style='width:90px;height:90px'></div>"
                )
                badge_bg = "#00B7EB" if r_type == "LUSR" else "#FFD700"
                badge_fg = "#000"
                if delta is not None:
                    _d = round(delta)
                    if _d == 0:
                        delta_html = (
                            "<div style='color:#888888;font-size:0.9em;margin-top:4px'>= 0</div>"
                        )
                    else:
                        d_color = "#00C853" if _d > 0 else "#FF5252"
                        d_arrow = "▲" if _d > 0 else "▼"
                        d_sign = "+" if _d > 0 else "-"
                        delta_html = (
                            f"<div style='color:{d_color};font-size:0.9em;margin-top:4px'>"
                            f"{d_arrow} {d_sign}{abs(_d)}</div>"
                        )
                else:
                    delta_html = "<div style='color:#888;font-size:0.9em;margin-top:4px'>—</div>"
                st.markdown(
                    f"""<div style='display:flex;flex-direction:column;
                                    align-items:center;justify-content:center;
                                    text-align:center;padding:4px 0;gap:4px'>
                        <div style='font-size:1.25em;font-weight:700;
                                    letter-spacing:0.02em;line-height:1.2'>
                            {html.escape(pg_icon)} {html.escape(pg_label)}
                        </div>
                        <div style='display:flex;align-items:center;
                                    justify-content:center;height:96px'>
                            {img_html}
                        </div>
                        <div style='font-size:1.05em;font-weight:700'>
                            {html.escape(tier_label)}
                        </div>
                        <div>
                            <span style='background:{badge_bg};color:{badge_fg};
                                         padding:1px 7px;border-radius:10px;
                                         font-size:0.72em;font-weight:600'>
                                {html.escape(r_type)}
                            </span>
                            &nbsp;<span style='font-size:1.1em;font-weight:700'>
                                {r_value:.0f}
                            </span>
                        </div>
                        {delta_html}
                    </div>""",
                    unsafe_allow_html=True,
                )


def _render_lusr_rating_chart(db_path: str) -> None:
    """Rend le graphe d'évolution LUSR/CSR avec filtres période et groupe."""
    import polars as pl

    from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG
    from src.visualization.timeseries_combat import plot_lusr_timeseries

    history_all = _load_lusr_history(db_path)
    if not history_all:
        return
    df_all = pl.DataFrame(history_all)
    st.markdown(f"#### {t('career_lusr_rating_evolution')}")
    _PERIOD_KEYS = ["all", "2y", "1y", "1m", "1w"]
    selected_period = st.segmented_control(
        t("encounters_period_label"),
        options=_PERIOD_KEYS,
        format_func=lambda k: t(f"encounters_period_{k}"),
        default="all",
        key="lusr_period",
    )
    _period_offsets = {"2y": 730, "1y": 365, "1m": 30, "1w": 7}
    _selected = selected_period or "all"
    if _selected in _period_offsets:
        _since = datetime.utcnow() - timedelta(days=_period_offsets[_selected])
        df_all = df_all.filter(pl.col("start_time") >= _since)
    _PERIOD_GRANULARITY: dict[str, str | None] = {
        "1w": None,
        "1m": "1d",
        "1y": "1w",
        "2y": "1w",
        "all": "1w",
    }
    _trunc = _PERIOD_GRANULARITY.get(_selected)
    if _trunc is not None:
        df_all = (
            df_all.with_columns(pl.col("start_time").dt.truncate(_trunc).alias("_bucket"))
            .sort("start_time", descending=True)
            .unique(subset=["_bucket", "playlist_group"], keep="first")
            .drop("_bucket")
            .sort("start_time")
        )
    available_groups = sorted(df_all["playlist_group"].drop_nulls().unique().to_list())
    if not available_groups:
        return
    group_options: dict[str, str | None] = {t("career_lusr_all_groups"): None}
    pg_labels = _get_pg_labels()
    for g in _PG_ORDER:
        if g in available_groups:
            group_options[f"{_PG_ICONS.get(g, '🎮')} {pg_labels.get(g, g.capitalize())}"] = g
    for g in available_groups:
        if g not in _PG_ORDER:
            group_options[f"🎮 {g.capitalize()}"] = g
    selected_label = st.selectbox(
        t("career_lusr_group_select"),
        options=list(group_options.keys()),
        key="lusr_group_select",
    )
    selected_group = group_options[selected_label]
    chart_title = f"LUSR / CSR — {selected_label}"
    with safe_chart_render("career_lusr_group_error"):
        fig = plot_lusr_timeseries(df_all, title=chart_title, playlist_group=selected_group)
        st.plotly_chart(
            fig, key=f"lusr_chart_{selected_label}", width="stretch", config=PLOTLY_CLEAN_CONFIG
        )


def _render_lusr_section(*, db_path: str, xuid: str) -> None:
    """Rend la section LUSR/CSR sur la page Carrière."""
    try:
        from src.analysis.skill_rating_config import LUSR_SUBTITLE, LUSR_TITLE
    except ImportError:
        logger.debug("Modules LUSR non disponibles pour career.py")
        return

    st.subheader(f"🏅 {LUSR_TITLE} — {LUSR_SUBTITLE}")
    snapshot = _load_lusr_snapshot(db_path)
    if not snapshot:
        st.info(t("career_lusr_no_rating"))
        return

    snap_by_group = {s["playlist_group"]: s for s in snapshot}
    ordered = [snap_by_group[g] for g in _PG_ORDER if g in snap_by_group]
    ordered += [s for s in snapshot if s["playlist_group"] not in _PG_ORDER]

    project_root = Path(__file__).parents[3]
    st.markdown(
        "<style>[data-testid='stVerticalBlockBorderWrapper']{border-radius:0!important}</style>",
        unsafe_allow_html=True,
    )
    _render_lusr_rank_cards(ordered, project_root)
    _render_lusr_rating_chart(db_path)


def _render_rank_section(  # noqa: C901, PLR0912, PLR0915
    *,
    career_data: dict,
    db_path: str,
    xuid: str,
) -> None:
    """Rend les sections rang, gauge, progression Héros et historique XP."""
    rank_number = career_data.get("rank", 0)
    rank_name = career_data.get("rank_name", "")
    rank_tier = career_data.get("rank_tier", "")
    current_xp = career_data.get("current_xp", 0) or 0
    xp_for_next = career_data.get("xp_for_next_rank", 1) or 1
    xp_total = career_data.get("xp_total", 0) or 0
    is_max = career_data.get("is_max_rank", False)

    # Calcul progression
    if is_max:
        progress_pct = 100.0
    elif xp_for_next > 0:
        progress_pct = min(100.0, (current_xp / xp_for_next) * 100)
    else:
        progress_pct = 0.0

    # Label FR du rang — utiliser les métadonnées officielles depuis metadata.duckdb
    _meta = get_meta_rank_info(rank_number)
    if _meta:
        rank_label_fr = _meta.full_label_fr or t("career_rank_n", n=rank_number)
    else:
        logger.warning(
            "Rang %d introuvable dans metadata.duckdb — fallback sur rank_name DB ('%s')",
            rank_number,
            rank_name,
        )
        rank_label_fr = format_career_rank_label_fr(tier=rank_tier, title=rank_name, grade=None)
        if not rank_label_fr:
            rank_label_fr = rank_name or t("career_rank_n", n=rank_number)

    _render_rank_header(
        career_data,
        rank_number,
        rank_label_fr,
        is_max,
        current_xp,
        xp_for_next,
        xp_total,
        progress_pct,
    )

    st.divider()
    _render_hero_section(xp_total, rank_number, is_max)

    st.divider()
    _render_xp_history(db_path, xuid, is_max, xp_total)


def _render_rank_header(  # noqa: PLR0913
    career_data: dict,
    rank_number: int,
    rank_label_fr: str,
    is_max: bool,
    current_xp: int,
    xp_for_next: int,
    xp_total: int,
    progress_pct: float,
) -> None:
    """Rend le header avec icône de rang, métriques et gauge."""
    col_icon, col_info, col_gauge = st.columns([1, 2, 2])

    with col_icon:
        adornment_db_path = career_data.get("adornment_path") or ""
        displayed_icon = False
        if adornment_db_path:
            local_adornment = ensure_local_image_path(
                adornment_db_path,
                prefix="adornment",
                download_enabled=True,
                auto_refresh_hours=0,
            )
            if local_adornment:
                st.image(local_adornment, width=140)
                displayed_icon = True
        if not displayed_icon:
            icon_path = get_rank_icon_path(rank_number) if rank_number else None
            if icon_path and icon_path.exists():
                st.image(str(icon_path), width=120)
            else:
                st.markdown(f"### {rank_number}")

    with col_info:
        st.subheader(rank_label_fr)
        m1, m2 = st.columns(2)
        with m1:
            st.metric(t("career_metric_rank"), f"{rank_number} / 272")
            st.metric(t("career_metric_xp_total"), f"{xp_total:,}")
        with m2:
            if is_max:
                st.metric(t("career_rank_max"), t("career_rank_max"))
            else:
                st.metric(t("career_metric_current_xp"), f"{current_xp:,}")
                st.metric(t("career_metric_next_rank_xp"), f"{xp_for_next:,}")

    with col_gauge, safe_chart_render("career_gauge_error"):
        gauge_fig = create_career_progress_gauge(
            current_xp=current_xp,
            xp_for_next_rank=xp_for_next,
            progress_pct=progress_pct,
            rank_name_fr=rank_label_fr,
            is_max_rank=is_max,
        )
        if gauge_fig is not None:
            st.plotly_chart(
                gauge_fig,
                key="career_gauge",
                width="stretch",
                config=PLOTLY_STATIC_CONFIG,
            )
        else:
            st.info(t("career_gauge_generate_error"))


def _render_hero_section(xp_total: int, rank_number: int, is_max: bool) -> None:
    """Rend la section progression vers Héros."""
    st.subheader(t("career_progression_to_hero"))
    hero_data = compute_hero_progress(xp_total=xp_total, rank=rank_number, is_max_rank=is_max)
    hero_pct = hero_data["percentage"]
    xp_remaining = hero_data["xp_remaining"]

    col_hero_metrics, col_hero_gauge = st.columns([3, 2])
    with col_hero_metrics:
        m1, m2, m3, m4 = st.columns(4)
        with m1:
            st.metric(t("career_metric_xp_earned"), f"{xp_total:,}")
        with m2:
            st.metric(t("career_metric_xp_remaining"), f"{xp_remaining:,}")
        with m3:
            st.metric(t("career_metric_xp_required"), f"{XP_HERO_TOTAL:,}")
        with m4:
            st.metric(t("career_metric_rank"), f"{rank_number} / {RANK_MAX}")

    with col_hero_gauge, safe_chart_render("career_hero_progress_error"):
        hero_gauge = create_hero_progress_gauge(
            hero_pct=hero_pct,
            xp_total=xp_total,
            xp_remaining=xp_remaining,
            is_max_rank=is_max,
        )
        st.plotly_chart(
            hero_gauge,
            key="hero_progress_gauge",
            width="stretch",
            config=PLOTLY_STATIC_CONFIG,
        )


def _render_xp_history(  # noqa: C901, PLR0912
    db_path: str,
    xuid: str,
    is_max: bool,
    xp_total: int,
) -> None:
    """Rend la section historique de progression XP."""
    history = _load_career_history(db_path, xuid)
    if not history:
        st.info(t("career_computing"))
        return

    estimated_curve: list[tuple[datetime, int]] | None = None
    try:
        first_sync_at = history[0]["recorded_at"]
        if first_sync_at and len(history) >= 2:
            pre_sync_dates = _load_pre_sync_match_dates(db_path, xuid, first_sync_at)
            if pre_sync_dates:
                estimated_curve = _compute_estimated_xp_curve(history, pre_sync_dates)
    except Exception as e:
        logger.warning("Estimation pré-sync échouée: %s", e)

    hero_proj: list[tuple[datetime, int]] | None = None
    optimistic_proj: list[tuple[datetime, int]] | None = None
    if not is_max:
        try:
            xp_per_day = _compute_active_xp_per_day(history)
            if xp_per_day > 0:
                last_date = history[-1]["recorded_at"]
                hero_proj, optimistic_proj = _compute_hero_projections(
                    xp_total,
                    last_date,
                    xp_per_day,
                )
        except Exception as e:
            logger.warning("Projection Héros échouée: %s", e)

    try:
        other_players_data = _load_other_players_histories(xuid)
    except Exception as e:
        logger.warning("Chargement autres joueurs échoué: %s", e)
        other_players_data = []

    with safe_chart_render("career_history_error"):
        history_fig = _create_xp_history_chart(
            history,
            estimated_curve=estimated_curve,
            hero_projection=hero_proj,
            optimistic_projection=optimistic_proj,
            is_max_rank=is_max,
            other_players=other_players_data or None,
        )
        if history_fig:
            st.plotly_chart(
                history_fig,
                key="career_xp_history",
                width="stretch",
                config=PLOTLY_CLEAN_CONFIG,
            )
        else:
            st.info(t("career_rank_history_no_data"))

    # Tableau récapitulatif des derniers snapshots
    with st.expander(t("career_rank_history_title"), expanded=False):
        recent = list(reversed(history[-10:]))
        for snap in recent:
            _snap_meta = get_meta_rank_info(snap.get("rank", 0))
            snap_label = (
                _snap_meta.full_label_fr
                if _snap_meta
                else format_career_rank_label_fr(
                    tier=snap.get("rank_tier", ""),
                    title=snap.get("rank_name", ""),
                    grade=None,
                )
            )
            date_str = str(snap.get("recorded_at", ""))[:19]
            xp_t = snap.get("xp_total", 0) or 0
            st.text(
                f"{date_str}  |  {t('career_rank_n', n=snap['rank'])}: {snap_label}  |  XP: {xp_t:,}"
            )


@fragment_if_available
def render_career_page(  # noqa: C901, PLR0912, PLR0915
    *,
    db_path: str,
    xuid: str,
    db_key: str | None = None,
) -> None:
    """Rend la page Carrière avec rang actuel, gauge et historique."""
    st.header(t("career_header"))
    career_data = _load_career_data(db_path, xuid)

    if career_data is not None:
        _render_rank_section(career_data=career_data, db_path=db_path, xuid=xuid)
    else:
        st.info(t("career_no_data"))

    st.divider()
    _render_lusr_section(db_path=db_path, xuid=xuid)

    st.divider()
    render_encounters_section(db_path=db_path, xuid=xuid)
