"""Sections contextuelles additionnelles — comparaison de sessions.

Fonctions :
- render_modes_breakdown  : répartition des modes joués (barres groupées)
- render_match_highlights : meilleur/pire match par F/D
- render_map_table        : tableau wins/défaites par carte
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.data.domain.refdata import Outcome
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars

_COLOR_A = "#E74C3C"
_COLOR_B = "#3498DB"


def _extract_mode(pair_name: str | None) -> str:
    """Extrait le nom de mode depuis pair_name.

    Exemple : "Arena:Ranked Slayer on Aquarius" → "Ranked Slayer"
    """
    if not pair_name:
        return "?"
    s = str(pair_name).strip()
    if ":" in s:
        s = s.split(":", 1)[1].strip()
    lo = s.lower()
    if " on " in lo:
        s = s[: lo.index(" on ")].strip()
    return s or "?"


# ════════════════════════════════════════════════════════════════════════════
# Modes joués
# ════════════════════════════════════════════════════════════════════════════


def render_modes_breakdown(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
) -> None:
    """Affiche la répartition des modes joués dans chaque session."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)

    if "pair_name" not in df_a.columns and "pair_name" not in df_b.columns:
        return

    def _count(df: pl.DataFrame) -> dict[str, int]:
        if df.is_empty() or "pair_name" not in df.columns:
            return {}
        counts: dict[str, int] = {}
        for v in df.get_column("pair_name").to_list():
            m = _extract_mode(v)
            counts[m] = counts.get(m, 0) + 1
        return counts

    counts_a = _count(df_a)
    counts_b = _count(df_b)
    all_modes = sorted(set(counts_a) | set(counts_b))
    if not all_modes:
        return

    st.markdown(t("sc_modes_breakdown"))
    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            name="Session A",
            y=all_modes,
            x=[counts_a.get(m, 0) for m in all_modes],
            orientation="h",
            marker_color=_COLOR_A,
            hovertemplate="%{y}: %{x} partie(s)<extra>Session A</extra>",
        )
    )
    fig.add_trace(
        go.Bar(
            name="Session B",
            y=all_modes,
            x=[counts_b.get(m, 0) for m in all_modes],
            orientation="h",
            marker_color=_COLOR_B,
            hovertemplate="%{y}: %{x} partie(s)<extra>Session B</extra>",
        )
    )
    fig.update_layout(
        barmode="group",
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font={"color": "#E0E0E0"},
        xaxis={
            "title": t("sc_match_count"),
            "dtick": 1,
            "showgrid": True,
            "gridcolor": "rgba(255,255,255,0.08)",
        },
        yaxis={"showgrid": False},
        height=max(180, len(all_modes) * 48),
        showlegend=True,
        legend={"orientation": "h", "y": 1.05, "x": 0.5, "xanchor": "center", "yanchor": "bottom"},
        margin={"l": 130, "r": 20, "t": 40, "b": 40},
    )
    with safe_chart_render():
        st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)


# ════════════════════════════════════════════════════════════════════════════
# Highlights meilleur/pire match
# ════════════════════════════════════════════════════════════════════════════


def render_match_highlights(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
) -> None:
    """Affiche le meilleur et le pire match (par ratio F/D) pour chaque session."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)

    def _best_worst(df: pl.DataFrame) -> tuple[str | None, str | None]:
        if df.is_empty() or "kills" not in df.columns or "deaths" not in df.columns:
            return None, None
        rows = list(df.iter_rows(named=True))
        kds = [
            float(r.get("kills") or 0) / float(r.get("deaths") or 1)
            if float(r.get("deaths") or 0) > 0
            else float(r.get("kills") or 0)
            for r in rows
        ]
        if not kds:
            return None, None

        def _fmt(r: dict, kd: float) -> str:
            mode = _extract_mode(r.get("pair_name"))[:24]
            k, d = int(r.get("kills") or 0), int(r.get("deaths") or 0)
            return f"{k}F/{d}D · F/D {kd:.2f} — {mode}"

        best = _fmt(rows[kds.index(max(kds))], max(kds))
        worst = _fmt(rows[kds.index(min(kds))], min(kds)) if len(rows) > 1 else None
        return best, worst

    best_a, worst_a = _best_worst(df_a)
    best_b, worst_b = _best_worst(df_b)
    if best_a is None and best_b is None:
        return

    st.markdown(t("sc_match_highlights"))
    col_a, col_b = st.columns(2)
    for col, best, worst in [(col_a, best_a, worst_a), (col_b, best_b, worst_b)]:
        with col:
            if best:
                st.markdown(f"{t('sc_best_match')} : {best}")
            if worst:
                st.markdown(f"{t('sc_worst_match')} : {worst}")


# ════════════════════════════════════════════════════════════════════════════
# Tableau des cartes
# ════════════════════════════════════════════════════════════════════════════


def render_map_table(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
) -> None:
    """Affiche un tableau comparatif des stats par carte pour chaque session."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)

    if "map_name" not in df_a.columns and "map_name" not in df_b.columns:
        return

    def _map_counts(df: pl.DataFrame) -> dict[str, dict]:
        if df.is_empty() or "map_name" not in df.columns:
            return {}
        out: dict[str, dict] = {}
        for row in df.iter_rows(named=True):
            m = str(row.get("map_name") or "?")
            if m not in out:
                out[m] = {"n": 0, "w": 0, "l": 0}
            out[m]["n"] += 1
            o = row.get("outcome")
            if o == int(Outcome.WIN):
                out[m]["w"] += 1
            elif o == int(Outcome.LOSS):
                out[m]["l"] += 1
        return out

    mc_a = _map_counts(df_a)
    mc_b = _map_counts(df_b)
    all_maps = sorted(set(mc_a) | set(mc_b))
    if not all_maps:
        return

    st.markdown(t("sc_map_table"))
    rows = []
    for m in all_maps:
        a = mc_a.get(m, {})
        b = mc_b.get(m, {})
        rows.append(
            {
                t("sc_map_col"): m,
                "A — Parties": a.get("n", 0) if a else "—",
                "A — V/D": f"{a.get('w', 0)}V/{a.get('l', 0)}D" if a else "—",
                "B — Parties": b.get("n", 0) if b else "—",
                "B — V/D": f"{b.get('w', 0)}V/{b.get('l', 0)}D" if b else "—",
            }
        )
    st.dataframe(pl.DataFrame(rows), hide_index=True)


__all__ = [
    "render_map_table",
    "render_match_highlights",
    "render_modes_breakdown",
]
