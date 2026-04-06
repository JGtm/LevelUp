"""Visualisations additionnelles — comparaison de sessions.

Extrait de session_compare_charts.py pour respecter la limite de 500 lignes.

Fonctions :
- _build_participation_bar_chart  : barres horiz. profil participation (interne)
- render_session_temporal_header  : résumé temporel (date, nombre de parties)
- render_outcomes_distribution    : répartition résultats (donuts W/L/T)
- render_kd_progression           : évolution F/M par partie (courbe)
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.data.domain.refdata import Outcome
from src.ui.chart_utils import render_chart_or_info
from src.ui.i18n import t
from src.ui.pages.session_compare_logic import format_date_with_weekday
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization._compat import DataFrameLike, ensure_polars

# Couleurs locales (évite une dépendance circulaire avec session_compare_charts)
_COLOR_A = "#E74C3C"  # Rouge corail — Session A
_COLOR_B = "#3498DB"  # Bleu vif    — Session B

_OUTCOME_COLORS: dict[int, str] = {
    int(Outcome.WIN): "#27AE60",
    int(Outcome.LOSS): "#E74C3C",
    int(Outcome.TIE): "#95A5A6",
    int(Outcome.DID_NOT_FINISH): "#555555",
}

# ════════════════════════════════════════════════════════════════════════════
# Profil de participation — barres horizontales (remplacement du radar superposé)
# ════════════════════════════════════════════════════════════════════════════


def _build_participation_bar_chart(profiles: list[dict]) -> go.Figure | None:
    """Construit un graphique en barres horizontales comparant les profils A vs B.

    Remplace le radar superposé/opaque par une représentation plus lisible :
    6 axes en lignes, Session A et B en colonnes groupées, valeurs en %.
    """
    if not profiles:
        return None

    axes = ["objectifs", "combat", "support", "score", "impact", "survie"]
    labels = [
        t("radar_objectives"),
        t("radar_combat"),
        t("radar_support"),
        t("col_score"),
        t("radar_impact"),
        t("radar_survival"),
    ]

    fig = go.Figure()
    for profile in profiles:
        values = [(profile.get(f"{ax}_norm") or 0.0) * 100 for ax in axes]
        fig.add_trace(
            go.Bar(
                name=profile.get("name", ""),
                y=labels,
                x=values,
                orientation="h",
                marker_color=profile.get("color"),
                hovertemplate="%{y}: %{x:.0f}%<extra>%{fullData.name}</extra>",
            )
        )

    fig.update_layout(
        barmode="group",
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font={"color": "#E0E0E0"},
        xaxis={
            "title": "%",
            "range": [0, 110],
            "showgrid": True,
            "gridcolor": "rgba(255,255,255,0.1)",
        },
        yaxis={"showgrid": False},
        showlegend=True,
        legend={"orientation": "h", "y": -0.18, "x": 0.5, "xanchor": "center"},
        height=300,
        margin={"l": 90, "r": 20, "t": 10, "b": 55},
    )
    return fig


# ════════════════════════════════════════════════════════════════════════════
# Résumé temporel compact
# ════════════════════════════════════════════════════════════════════════════


def render_session_temporal_header(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
) -> None:
    """Affiche le contexte temporel de chaque session : date de début et nombre de parties."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)

    def _summary(df: pl.DataFrame, color: str) -> str:
        if df.is_empty() or "start_time" not in df.columns:
            return "—"
        times = df.get_column("start_time").drop_nulls().sort()
        if times.is_empty():
            return "—"
        n = len(df)
        date_str = format_date_with_weekday(times[0])
        s = "s" if n > 1 else ""
        return (
            f"<span style='color:{color};font-weight:600'>{date_str}</span>"
            f"&nbsp;·&nbsp;<b>{n} partie{s}</b>"
        )

    col_a, col_b = st.columns(2)
    with col_a:
        st.markdown(_summary(df_a, _COLOR_A), unsafe_allow_html=True)
    with col_b:
        st.markdown(_summary(df_b, _COLOR_B), unsafe_allow_html=True)


# ════════════════════════════════════════════════════════════════════════════
# Répartition des résultats — donuts WIN / LOSS / TIE / DNF
# ════════════════════════════════════════════════════════════════════════════


def _build_outcome_donut(df: pl.DataFrame, label: str) -> go.Figure | None:
    """Construit un donut chart de répartition Victoire/Défaite/Égalité."""
    if df.is_empty() or "outcome" not in df.columns:
        return None

    counts: dict[int, int] = {}
    for val in df.get_column("outcome").drop_nulls().to_list():
        key = int(val)
        counts[key] = counts.get(key, 0) + 1

    if not counts:
        return None

    ordered = [Outcome.WIN, Outcome.LOSS, Outcome.TIE, Outcome.DID_NOT_FINISH]
    label_map = {
        int(Outcome.WIN): t("outcome_win"),
        int(Outcome.LOSS): t("outcome_loss"),
        int(Outcome.TIE): t("outcome_draw"),
        int(Outcome.DID_NOT_FINISH): t("outcome_dnf"),
    }

    pie_labels = [label_map[int(o)] for o in ordered if counts.get(int(o), 0) > 0]
    pie_values = [counts[int(o)] for o in ordered if counts.get(int(o), 0) > 0]
    pie_colors = [_OUTCOME_COLORS[int(o)] for o in ordered if counts.get(int(o), 0) > 0]

    wins = counts.get(int(Outcome.WIN), 0)
    total = sum(pie_values)

    fig = go.Figure(
        go.Pie(
            labels=pie_labels,
            values=pie_values,
            hole=0.55,
            marker={"colors": pie_colors, "line": {"color": "rgba(0,0,0,0.25)", "width": 1}},
            textinfo="label+percent",
            hovertemplate="%{label}: %{value} partie(s)<extra></extra>",
            showlegend=False,
            sort=False,
        )
    )
    fig.add_annotation(
        text=f"<b>{wins}/{total}</b>",
        x=0.5,
        y=0.5,
        font={"size": 14, "color": "#E0E0E0"},
        showarrow=False,
    )
    fig.update_layout(
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        font={"color": "#E0E0E0"},
        title={"text": label, "x": 0.5, "xanchor": "center", "font": {"size": 13}},
        height=220,
        margin={"l": 10, "r": 10, "t": 35, "b": 10},
    )
    return fig


@fragment_if_available
def render_outcomes_distribution(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
) -> None:
    """Affiche la répartition Victoire/Défaite/Égalité pour les deux sessions."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)

    if "outcome" not in df_a.columns and "outcome" not in df_b.columns:
        return

    st.markdown(t("sc_outcomes_distribution"))
    col_a, col_b = st.columns(2)

    with col_a:
        fig = _build_outcome_donut(df_a, "Session A")
        render_chart_or_info(
            fig,
            key="sc_outcome_donut_a",
            config=PLOTLY_STATIC_CONFIG,
            info_key="insufficient_data_chart",
        )

    with col_b:
        fig = _build_outcome_donut(df_b, "Session B")
        render_chart_or_info(
            fig,
            key="sc_outcome_donut_b",
            config=PLOTLY_STATIC_CONFIG,
            info_key="insufficient_data_chart",
        )


# ════════════════════════════════════════════════════════════════════════════
# Évolution du ratio Frags/Morts par partie
# ════════════════════════════════════════════════════════════════════════════


def _compute_match_series(df: pl.DataFrame) -> dict:
    """Calcule les séries par partie (F/D, précision, durée de vie) triées par start_time."""
    empty: dict = {"kds": [], "accuracies": [], "avg_lifes": [], "hover_labels": []}
    if df.is_empty() or "kills" not in df.columns or "deaths" not in df.columns:
        return empty
    df_s = df.sort("start_time") if "start_time" in df.columns else df
    kds, accs, lifes, labels = [], [], [], []
    for i, row in enumerate(df_s.iter_rows(named=True), start=1):
        k = float(row.get("kills") or 0)
        d = float(row.get("deaths") or 0)
        kds.append(round(k / d if d > 0 else k, 2))
        acc = row.get("accuracy")
        accs.append(float(acc) if acc is not None else None)
        life = row.get("avg_life_seconds")
        lifes.append(float(life) if life is not None else None)
        pair = str(row.get("pair_fr") or row.get("mode_ui") or row.get("pair_name") or "")
        short = pair.split(":")[-1].strip()[:18] if ":" in pair else pair[:18]
        lbl = t("sc_match_index") + f" {i}"
        labels.append(lbl + (f" — {short}" if short else ""))
    return {"kds": kds, "accuracies": accs, "avg_lifes": lifes, "hover_labels": labels}


def _kd_layout(has_acc: bool) -> dict:
    """Retourne le layout Plotly pour le graphe F/D + précision."""
    layout: dict = {
        "paper_bgcolor": "rgba(0,0,0,0)",
        "plot_bgcolor": "rgba(0,0,0,0)",
        "font": {"color": "#E0E0E0"},
        "xaxis": {
            "title": t("sc_match_index"),
            "tickmode": "linear",
            "dtick": 1,
            "showgrid": True,
            "gridcolor": "rgba(255,255,255,0.08)",
        },
        "yaxis": {
            "title": t("sc_fd_ratio"),
            "showgrid": True,
            "gridcolor": "rgba(255,255,255,0.08)",
            "rangemode": "tozero",
        },
        "height": 310,
        "showlegend": True,
        "legend": {"orientation": "h", "y": -0.28, "x": 0.5, "xanchor": "center"},
        "margin": {"l": 50, "r": 55 if has_acc else 20, "t": 20, "b": 75},
    }
    if has_acc:
        layout["yaxis2"] = {
            "title": t("col_accuracy"),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
            "range": [0, 100],
        }
    return layout


def _build_kd_progression_figure(
    s_a: dict,
    s_b: dict,
    label_a: str,
    label_b: str,
    has_acc: bool,
) -> go.Figure:
    """Construit la figure Plotly F/D + précision pour deux sessions."""
    fig = go.Figure()
    fig.add_hline(
        y=1.0,
        line_dash="dot",
        line_color="rgba(255,255,255,0.25)",
        annotation_text="1.0",
        annotation_position="right",
        annotation_font_color="rgba(255,255,255,0.4)",
    )

    for series, label, color in [(s_a, label_a, _COLOR_A), (s_b, label_b, _COLOR_B)]:
        if not series["kds"]:
            continue
        x = list(range(1, len(series["kds"]) + 1))
        hover_texts = [
            f"{hl}<br>F/D: {kd:.2f}"
            + (f"<br>{t('col_accuracy')}: {acc:.1f}%" if acc is not None else "")
            + (f"<br>{t('col_avg_life')}: {life:.0f}s" if life is not None else "")
            for hl, kd, acc, life in zip(
                series["hover_labels"],
                series["kds"],
                series["accuracies"],
                series["avg_lifes"],
                strict=False,
            )
        ]
        fig.add_trace(
            go.Scatter(
                x=x,
                y=series["kds"],
                mode="lines+markers",
                name=label,
                line={"color": color, "width": 2},
                marker={"size": 7},
                hovertext=hover_texts,
                hovertemplate="%{hovertext}<extra></extra>",
            )
        )
        if has_acc:
            x_acc = [x[i] for i, a in enumerate(series["accuracies"]) if a is not None]
            y_acc = [a for a in series["accuracies"] if a is not None]
            if x_acc:
                fig.add_trace(
                    go.Scatter(
                        x=x_acc,
                        y=y_acc,
                        mode="lines+markers",
                        name=f"{label} — {t('col_accuracy')}",
                        line={"color": color, "width": 1.5, "dash": "dash"},
                        marker={"size": 5, "symbol": "circle-open"},
                        yaxis="y2",
                        hovertemplate=f"Préc. %{{y:.1f}}%<extra>{label}</extra>",
                        opacity=0.75,
                    )
                )

    fig.update_layout(**_kd_layout(has_acc))
    return fig


@fragment_if_available
def render_kd_progression(
    df_session_a: DataFrameLike,
    df_session_b: DataFrameLike,
    label_a: str = "Session A",
    label_b: str = "Session B",
) -> None:
    """Affiche l'évolution du ratio F/D + précision par partie pour les deux sessions."""
    df_a = ensure_polars(df_session_a)
    df_b = ensure_polars(df_session_b)
    s_a = _compute_match_series(df_a)
    s_b = _compute_match_series(df_b)
    if not s_a["kds"] and not s_b["kds"]:
        return

    has_acc = any(v is not None for v in (s_a["accuracies"] + s_b["accuracies"]))
    st.markdown(t("sc_kd_progression"))
    fig = _build_kd_progression_figure(s_a, s_b, label_a, label_b, has_acc)
    render_chart_or_info(
        fig,
        key="sc_kd_progression",
        config=PLOTLY_STATIC_CONFIG,
        info_key="insufficient_data_chart",
    )


__all__ = [
    "_build_participation_bar_chart",
    "render_session_temporal_header",
    "render_outcomes_distribution",
    "render_kd_progression",
]
