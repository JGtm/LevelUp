"""Section Némésis / Souffre-douleur et graphique Killer-Victim.

Extraite de match_view_players.py — regroupe render_nemesis_section
et les helpers d'affichage antagonistes.
"""

from __future__ import annotations

import html
import logging
import os
import re
from collections.abc import Callable

import streamlit as st

from src.analysis import compute_personal_antagonists
from src.config import BOT_MAP
from src.ui import display_name_from_xuid
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import get_lang, t
from src.ui.pages.match_table_html import gamertag_link
from src.ui.pages.match_view_helpers import os_card
from src.ui.pages.match_view_players_data import (
    has_table_duckdb as _has_table_duckdb,
)
from src.ui.pages.match_view_players_data import (
    load_match_players_stats as _load_match_players_stats,
)
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.utils import parse_xuid_input

logger = logging.getLogger(__name__)


def _display_name_for_chart(
    xuid: str,
    gamertag: str | None,
    gt_map: dict[str, str] | None,
) -> str:
    """Nom d'affichage pour le graphe killer-victime (même logique que le roster)."""
    xu_s = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    if xu_s:
        bot_key = xu_s.strip()
        if bot_key.lower().startswith("bid("):
            bot_name = BOT_MAP.get(bot_key)
            if isinstance(bot_name, str) and bot_name.strip():
                return bot_name.strip()

    if xu_s and isinstance(gt_map, dict):
        mapped = gt_map.get(xu_s)
        if isinstance(mapped, str) and mapped.strip():
            return mapped.strip()

    g = str(gamertag or "").strip()
    if g and g != "?" and (not g.isdigit()) and (not g.lower().startswith("xuid(")):
        return g

    if xu_s:
        return display_name_from_xuid(xu_s)
    return "-"


def _render_antagonist_chart(  # noqa: C901, PLR0912, PLR0913
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    load_match_gamertags_fn: Callable | None = None,
    highlight_events: list | None = None,
) -> None:
    """Affiche le graphique des interactions Killer-Victim du match."""
    if not match_id or not match_id.strip():
        return

    gt_map = None
    if load_match_gamertags_fn is not None:
        try:
            gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)
        except Exception:
            logger.warning("antagonist chart: erreur chargement gamertags match=%s", match_id)
            gt_map = None

    pairs_df = None
    if db_path and str(db_path).endswith(".duckdb"):
        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            repo = DuckDBRepository(db_path, str(xuid).strip())
            pairs_df = repo.load_killer_victim_pairs_as_polars(match_id=match_id.strip())
        except Exception:
            logger.warning("antagonist chart: erreur chargement KV pairs match=%s", match_id)
            pairs_df = None

    # Fallback : construire depuis highlight_events
    if (pairs_df is None or pairs_df.is_empty()) and highlight_events:
        try:
            import polars as pl

            from src.analysis import compute_killer_victim_pairs

            kv_pairs = compute_killer_victim_pairs(highlight_events, tolerance_ms=5)
            if kv_pairs:
                pairs_df = pl.DataFrame(
                    {
                        "match_id": [match_id] * len(kv_pairs),
                        "killer_xuid": [p.killer_xuid for p in kv_pairs],
                        "killer_gamertag": [p.killer_gamertag or "?" for p in kv_pairs],
                        "victim_xuid": [p.victim_xuid for p in kv_pairs],
                        "victim_gamertag": [p.victim_gamertag or "?" for p in kv_pairs],
                        "kill_count": [1] * len(kv_pairs),
                        "time_ms": [p.time_ms for p in kv_pairs],
                    }
                )
        except Exception:
            logger.warning(
                "antagonist chart: erreur fallback KV highlight_events match=%s", match_id
            )

    if pairs_df is not None and not pairs_df.is_empty():
        try:
            import polars as pl

            from src.visualization.antagonist_charts import plot_killer_victim_stacked_bars

            killer_displays = [
                _display_name_for_chart(row[0], row[1], gt_map)
                for row in pairs_df.select("killer_xuid", "killer_gamertag").iter_rows()
            ]
            victim_displays = [
                _display_name_for_chart(row[0], row[1], gt_map)
                for row in pairs_df.select("victim_xuid", "victim_gamertag").iter_rows()
            ]
            pairs_df = pairs_df.with_columns(
                pl.Series("killer_gamertag", killer_displays),
                pl.Series("victim_gamertag", victim_displays),
            )

            official_stats = _load_match_players_stats(db_path, match_id.strip())
            rank_by_xuid = (
                {s["xuid"]: s["rank"] for s in official_stats} if official_stats else None
            )

            me_xuid = str(
                parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()
            ).strip()
            with safe_chart_render():
                fig = plot_killer_victim_stacked_bars(
                    pairs_df,
                    match_id=match_id,
                    me_xuid=me_xuid,
                    rank_by_xuid=rank_by_xuid,
                    title=t("mv_killer_victim_title"),
                    height=400,
                    lang=get_lang(),
                )
                if fig is not None:
                    st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
                else:
                    st.info(t("mv_interactions_no_data"))
        except Exception:
            logger.warning(
                "antagonist chart: erreur rendu graphique KV match=%s", match_id, exc_info=True
            )


@fragment_if_available
def render_nemesis_section(  # noqa: C901, PLR0913, PLR0915
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    colors: dict,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Némésis / Souffre-douleur."""
    st.subheader(t("mv_antagonists_title"))
    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        logger.debug("némésis: table highlight_events absente pour match=%s", match_id)
        st.caption(t("mv_nemesis_unavailable"))
        return

    with st.spinner(t("mv_highlight_loading")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    match_gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    official_stats = _load_match_players_stats(db_path, match_id.strip())

    res = compute_personal_antagonists(
        he, me_xuid=me_xuid, tolerance_ms=5, official_stats=official_stats
    )
    if (res.nemesis is None) and (res.bully is None):
        st.info(t("mv_nemesis_no_data"))

    def _debug_enabled() -> bool:
        env_flag = str(os.environ.get("LEVELUP_DEBUG_ANTAGONISTS") or "").strip().lower()
        if env_flag in {"1", "true", "yes", "y", "on"}:
            return True

        env_flag2 = str(os.environ.get("LEVELUP_DEBUG") or "").strip().lower()
        if env_flag2 in {"1", "true", "yes", "y", "on"}:
            return True

        try:
            if bool(st.session_state.get("ui_debug_antagonists", False)):
                return True
        except Exception:
            pass

        try:
            if hasattr(st, "query_params"):
                qp = st.query_params
                v = qp.get("debug_antagonists") or qp.get("debug")
            else:
                qp = st.experimental_get_query_params()
                v = (qp.get("debug_antagonists") or qp.get("debug") or [""])[0]
            if isinstance(v, list | tuple):
                v = v[0] if v else ""
            if str(v or "").strip().lower() in {"1", "true", "yes", "y", "on"}:
                return True
        except Exception:
            pass

        return False

    def _display_name_from_kv(xuid_value, gamertag_value) -> str:
        gt = str(gamertag_value or "").strip()
        xu_raw = str(xuid_value or "").strip()
        xu = parse_xuid_input(xu_raw) or xu_raw

        xu_key = str(xu).strip() if xu is not None else ""
        if xu_key and isinstance(match_gt_map, dict):
            mapped = match_gt_map.get(xu_key)
            if isinstance(mapped, str) and mapped.strip():
                return mapped.strip()

        if (not gt) or gt == "?" or gt.isdigit() or gt.lower().startswith("xuid("):
            if xu:
                return display_name_from_xuid(str(xu).strip())
            return "-"
        return gt

    if (res.nemesis is not None) or (res.bully is not None):
        nemesis_name = "-"
        nemesis_killed_me: int | None = None
        nemesis_killed_me_approx = False
        me_killed_nemesis: int | None = None
        me_killed_nemesis_approx = False
        if res.nemesis is not None:
            nemesis_name = _display_name_from_kv(res.nemesis.xuid, res.nemesis.gamertag)
            nemesis_killed_me = int(res.nemesis.opponent_killed_me.total)
            nemesis_killed_me_approx = bool(res.nemesis.opponent_killed_me.has_estimated)
            me_killed_nemesis = int(res.nemesis.me_killed_opponent.total)
            me_killed_nemesis_approx = bool(res.nemesis.me_killed_opponent.has_estimated)

        bully_name = "-"
        bully_killed_me: int | None = None
        bully_killed_me_approx = False
        me_killed_bully: int | None = None
        me_killed_bully_approx = False
        if res.bully is not None:
            bully_name = _display_name_from_kv(res.bully.xuid, res.bully.gamertag)
            bully_killed_me = int(res.bully.opponent_killed_me.total)
            bully_killed_me_approx = bool(res.bully.opponent_killed_me.has_estimated)
            me_killed_bully = int(res.bully.me_killed_opponent.total)
            me_killed_bully_approx = bool(res.bully.me_killed_opponent.has_estimated)

        def _clean_name(v: str) -> str:
            s = str(v or "")
            s = s.replace("\ufffd", "")
            s = re.sub(r"[\x00-\x1f\x7f]", "", s)
            s = re.sub(r"\s+", " ", s).strip()
            return s or "-"

        nemesis_name = _clean_name(nemesis_name)
        bully_name = _clean_name(bully_name)

        def _cmp_color(deaths_: int | None, kills_: int | None) -> str:
            if deaths_ is None or kills_ is None:
                return colors["slate"]
            if int(deaths_) > int(kills_):
                return colors["red"]
            if int(deaths_) < int(kills_):
                return colors["green"]
            return colors["violet"]

        def _fmt_count(label: str, value: int | None, approx: bool) -> str:
            if value is None:
                return "-"
            prefix = "≈ " if approx else ""
            if label == "deaths":
                return t("mv_deaths_count", prefix=prefix, n=int(value))
            return t("mv_killed_count", prefix=prefix, n=int(value))

        def _fmt_two_lines(
            deaths_: int | None, deaths_approx: bool, kills_: int | None, kills_approx: bool
        ) -> str:
            d = _fmt_count("deaths", deaths_, deaths_approx)
            k = _fmt_count("kills", kills_, kills_approx)
            return html.escape(d) + "<br/>" + html.escape(k)

        c = st.columns(2)
        with c[0]:
            os_card(
                t("lbl_nemesis"),
                gamertag_link(nemesis_name) if nemesis_name != "-" else "-",
                _fmt_two_lines(
                    nemesis_killed_me,
                    nemesis_killed_me_approx,
                    me_killed_nemesis,
                    me_killed_nemesis_approx,
                ),
                accent=_cmp_color(nemesis_killed_me, me_killed_nemesis),
                kpi_is_html=True,
                sub_style="color: rgba(245, 248, 255, 0.92); font-weight: 800; font-size: 16px; line-height: 1.15;",
                min_h=110,
            )
        with c[1]:
            os_card(
                t("lbl_victim"),
                gamertag_link(bully_name) if bully_name != "-" else "-",
                _fmt_two_lines(
                    bully_killed_me, bully_killed_me_approx, me_killed_bully, me_killed_bully_approx
                ),
                accent=_cmp_color(bully_killed_me, me_killed_bully),
                kpi_is_html=True,
                sub_style="color: rgba(245, 248, 255, 0.92); font-weight: 800; font-size: 16px; line-height: 1.15;",
                min_h=110,
            )

    if _debug_enabled():
        deaths_missing = max(0, int(res.my_deaths_total) - int(res.my_deaths_assigned_total))
        deaths_est = max(0, int(res.my_deaths_assigned_total) - int(res.my_deaths_assigned_certain))
        kills_missing = max(0, int(res.my_kills_total) - int(res.my_kills_assigned_total))
        kills_est = max(0, int(res.my_kills_assigned_total) - int(res.my_kills_assigned_certain))

        validation_icon = "✓" if res.is_validated else "⚠"
        validation_label = t("lbl_validated") if res.is_validated else t("lbl_not_validated")

        deaths_part = t(
            "mvp_attribution_deaths",
            assigned=res.my_deaths_assigned_total,
            total=res.my_deaths_total,
            certain=res.my_deaths_assigned_certain,
            estimated=deaths_est,
            missing=deaths_missing,
        )
        kills_part = t(
            "mvp_attribution_kills",
            assigned=res.my_kills_assigned_total,
            total=res.my_kills_total,
            certain=res.my_kills_assigned_certain,
            estimated=kills_est,
            missing=kills_missing,
        )
        st.caption(
            f"Debug antagonistes {validation_icon} {validation_label} — "
            f"{deaths_part} · {kills_part}"
        )

        if res.validation_notes:
            st.caption(f"Validation: {res.validation_notes}")

    _render_antagonist_chart(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        load_match_gamertags_fn=load_match_gamertags_fn,
        highlight_events=he,
    )
