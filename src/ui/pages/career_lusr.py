"""Page Carrière — Section LUSR/CSR (cartes, graphe, snapshot)."""

from __future__ import annotations

import html
import logging
from datetime import datetime, timedelta
from pathlib import Path

import streamlit as st

from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.pages.career_data import _load_lusr_history, _load_lusr_snapshot
from src.ui.pages.career_logic import _get_pg_labels

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
