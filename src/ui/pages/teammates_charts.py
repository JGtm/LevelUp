"""Graphiques pour la page Coéquipiers.

Fonctions de visualisation pour comparer les performances avec les coéquipiers.
"""

from __future__ import annotations

import logging

import plotly.graph_objects as go
import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.config import OKABE_ITO_PALETTE
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import get_lang, t
from src.ui.i18n.viz import viz_t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization import plot_trio_metric
from src.visualization._chart_series import SquadRecordSet
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._plot_options import PlotOptions
from src.visualization.teammates_hs_pk import plot_hs_pk_stacked
from src.visualization.trio import plot_trio_kills_deaths


def _hide_legend(fig: go.Figure) -> go.Figure:
    """Masque la légende Plotly — les noms apparaissent dans le panneau latéral fixe."""
    fig.update_layout(showlegend=False)
    return fig


@fragment_if_available
def render_metric_bar_charts(  # noqa: PLR0913
    series: list[tuple[str, DataFrameLike]],
    colors_by_name: dict[str, str],
    show_smooth: bool,
    key_suffix: str,
    plot_fn,
    squad_records: SquadRecordSet | None = None,
    hspk_records: dict | None = None,
) -> None:
    """Affiche les graphes killing spree et HS+PK pour une escouade."""
    _lang = get_lang()
    for metric_col, label, key_prefix in [
        ("max_killing_spree", t("tm_killing_spree"), "friend_spree_multi"),
    ]:
        fig = plot_fn(
            series,
            metric_col=metric_col,
            y_axis_title=label,
            hover_label=label,
            colors=colors_by_name,
            smooth_window=10,
            show_smooth_lines=show_smooth,
            lang=_lang,
            squad_records=squad_records,
        )
        if fig is None:
            st.info(t("insufficient_data_chart"))
        else:
            st.subheader(label)
            st.plotly_chart(
                _hide_legend(fig),
                width="stretch",
                key=f"{key_prefix}_{key_suffix}",
                config=PLOTLY_CLEAN_CONFIG,
            )

    # ── Graphe combiné HS + PK ──────────────────────────────────────────────
    fig_combined = plot_hs_pk_stacked(
        series,
        colors=colors_by_name,
        lang=_lang,
        records=hspk_records,
    )
    if fig_combined is None:
        st.info(t("insufficient_data_chart"))
    else:
        st.subheader(viz_t("hs_pk_combined_title", _lang))
        st.plotly_chart(
            _hide_legend(fig_combined),
            width="stretch",
            key=f"friend_hs_pk_stacked_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
        )


def render_outcome_bar_chart(dfr: DataFrameLike) -> None:
    """Affiche le graphe de distribution des résultats.

    Args:
        dfr: DataFrame avec colonne 'my_outcome'.
    """
    dfr = ensure_polars(dfr)
    outcome_map = {
        2: t("outcome_win"),
        3: t("outcome_loss"),
        1: t("outcome_draw"),
        4: t("outcome_dnf"),
    }
    dfr = dfr.with_columns(
        pl.col("my_outcome")
        .replace_strict(outcome_map, default=t("outcome_unknown"), return_dtype=pl.Utf8)
        .alias("my_outcome_label")
    )
    ordered_labels = [
        t("outcome_win"),
        t("outcome_loss"),
        t("outcome_draw"),
        t("outcome_dnf"),
        t("outcome_unknown"),
    ]
    counts_df = dfr.group_by("my_outcome_label").len().rename({"len": "count"})
    all_labels = pl.DataFrame({"my_outcome_label": ordered_labels})
    counts_df = all_labels.join(counts_df, on="my_outcome_label", how="left").fill_null(0)
    colors = OKABE_ITO_PALETTE
    fig = go.Figure(
        data=[
            go.Bar(
                x=counts_df["my_outcome_label"].to_list(),
                y=counts_df["count"].to_list(),
                marker_color=colors[0],
            )
        ]
    )
    fig.update_layout(height=300, margin={"l": 40, "r": 20, "t": 30, "b": 40})
    st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)


def _plot_trio_metric_chart(  # noqa: PLR0913
    *,
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike | None,
    d_f3: DataFrameLike | None,
    names: tuple[str, ...],
    lang: str,
    colors_by_name: dict[str, str] | None,
    metric: str,
    y_title: str,
    key: str,
    y_suffix: str = "",
    y_format: str = "",
    is_inverse: bool = False,
    squad_records: SquadRecordSet | None = None,
) -> None:
    """Rend un seul graphique trio_metric via st.plotly_chart."""
    st.plotly_chart(
        _hide_legend(
            plot_trio_metric(
                d_self,
                d_f1,
                d_f2,
                metric=metric,
                names=names,
                y_title=y_title,
                y_suffix=y_suffix,
                y_format=y_format,
                lang=lang,
                d_f3=d_f3,
                colors_by_name=colors_by_name,
                is_inverse=is_inverse,
                squad_records=squad_records,
            )
        ),
        width="stretch",
        key=key,
        config=PLOTLY_CLEAN_CONFIG,
    )


# (metric, title_key, ytitle_key, key_prefix, extra_kwargs)
_TRIO_METRIC_SPECS: list[tuple[str, str, str, str, dict]] = [
    ("assists", "tm_assists", "tm_assists", "trio_assists", {}),
    ("ratio", "tm_kda", "tm_kda", "trio_ratio", {"y_format": ".3f"}),
    ("accuracy", "tm_accuracy", None, "trio_accuracy", {"y_suffix": "%", "y_format": ".2f"}),
    ("average_life_seconds", "tm_avg_life", "tm_seconds", "trio_life", {"y_format": ".1f"}),
    ("performance", "tm_performance", "tm_score", "trio_performance", {"y_format": ".1f"}),
]


@fragment_if_available
def render_trio_charts(  # noqa: PLR0913
    d_self: DataFrameLike,
    d_f1: DataFrameLike,
    d_f2: DataFrameLike | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    f1_xuid: str,
    f2_xuid: str | None,
    *,
    d_f3: DataFrameLike | None = None,
    f3_name: str | None = None,
    f3_xuid: str | None = None,
    colors_by_name: dict[str, str] | None = None,
    squad_records: SquadRecordSet | None = None,
) -> None:
    """Affiche les graphes de métriques pour une escouade de 2, 3 ou 4 joueurs."""
    names: tuple[str, ...] = (me_name, f1_name)
    if f2_name:
        names += (f2_name,)
    if f3_name:
        names += (f3_name,)
    key_suffix = f1_xuid + (f"_{f2_xuid}" if f2_xuid else "") + (f"_{f3_xuid}" if f3_xuid else "")
    _lang = get_lang()

    # Graphe combiné kills↑/morts↓ (remplace les deux graphes séparés)
    with safe_chart_render():
        st.subheader(t("tm_kills_deaths"))
        st.plotly_chart(
            _hide_legend(
                plot_trio_kills_deaths(
                    d_self,
                    d_f1,
                    d_f2,
                    names=names,
                    opts=PlotOptions(lang=_lang),
                    d_f3=d_f3,
                    colors_by_name=colors_by_name,
                    squad_records=squad_records,
                )
            ),
            width="stretch",
            key=f"trio_kd_{key_suffix}",
            config=PLOTLY_CLEAN_CONFIG,
        )

    _shared: dict = {
        "d_self": d_self,
        "d_f1": d_f1,
        "d_f2": d_f2,
        "d_f3": d_f3,
        "names": names,
        "lang": _lang,
        "colors_by_name": colors_by_name,
        "squad_records": squad_records,
    }
    for metric, title_key, ytitle_key, key_prefix, extra in _TRIO_METRIC_SPECS:
        with safe_chart_render():
            st.subheader(t(title_key))
            _plot_trio_metric_chart(
                **_shared,
                metric=metric,
                y_title=t(ytitle_key) if ytitle_key else "%",
                key=f"{key_prefix}_{key_suffix}",
                **extra,
            )


# ---------------------------------------------------------------------------
# Graphe butterfly — premier frag (haut) / première mort (bas)
# ---------------------------------------------------------------------------

_BIN_SIZE_S: int = 15


def _load_first_events_data(
    db_path: str,
    xuids_by_name: dict[str, str],
    match_ids: list[str],
) -> pl.DataFrame:
    """Charge les instants du premier frag et de la première mort par joueur par match.

    Args:
        db_path: Chemin vers la DB (shared accessible).
        xuids_by_name: Mapping nom joueur → xuid.
        match_ids: Liste de match_ids à interroger.

    Returns:
        DataFrame avec first_kill_s et first_death_s (en secondes) ou vide.
    """
    from src.data.repositories.duckdb_repo import DuckDBRepository
    from src.data.services._teammates_first_events_queries import query_first_events

    if not xuids_by_name or not match_ids:
        return pl.DataFrame()

    xuids = list(xuids_by_name.values())
    xuid_to_name = {v: k for k, v in xuids_by_name.items()}

    try:
        repo = DuckDBRepository(db_path, "")
        conn = repo._get_connection()
        df = query_first_events(conn, xuids, match_ids)
    except Exception:
        logger.debug("_load_first_events_data: erreur connexion DB", exc_info=True)
        return pl.DataFrame()

    if df.is_empty():
        return df

    df = df.with_columns(pl.col("xuid").replace(xuid_to_name).alias("player_name"))
    return df


def _format_bin_label(start_s: int) -> str:
    """Formate le début d'une tranche en 'Xs' ou 'XmYYs'."""
    m, s = divmod(start_s, 60)
    if m == 0:
        return f"{s}s"
    return f"{m}m{s:02d}s"


def _compute_bin_counts(
    df: pl.DataFrame,
    player_names: list[str],
    metric: str,
) -> dict[str, dict[str, int]]:
    """Compte les matchs par tranche de _BIN_SIZE_S secondes pour chaque joueur.

    Args:
        df: DataFrame avec colonnes player_name et metric (float secondes).
        player_names: Liste des noms de joueurs.
        metric: Colonne à agréger (first_kill_s ou first_death_s).

    Returns:
        Dict player_name → {label_bin: count}.
    """
    counts: dict[str, dict[str, int]] = {name: {} for name in player_names}
    for name in player_names:
        vals = df.filter(pl.col("player_name") == name)[metric].drop_nulls().to_list()
        for v in vals:
            b = int(v // _BIN_SIZE_S) * _BIN_SIZE_S
            label = _format_bin_label(b)
            counts[name][label] = counts[name].get(label, 0) + 1
    return counts


def _build_first_events_fig(
    df: pl.DataFrame,
    colors_by_name: dict[str, str],
    lang: str,
) -> go.Figure:
    """Construit le butterfly histogram premier frag (haut) / première mort (bas).

    X = tranches de 15 s. Barres positives = frags, barres négatives = morts.
    Barres groupées par joueur (une couleur par joueur). Axe X en gras blanc.

    Args:
        df: DataFrame avec colonnes player_name, first_kill_s, first_death_s.
        colors_by_name: Palette de couleurs par nom de joueur.
        lang: Code langue courant.

    Returns:
        Figure Plotly butterfly groupée.
    """
    label_frag = t("tm_first_frag", lang=lang)
    label_death = t("tm_first_death", lang=lang)

    player_names = df["player_name"].unique().to_list()
    kill_counts = _compute_bin_counts(df, player_names, "first_kill_s")
    death_counts = _compute_bin_counts(df, player_names, "first_death_s")

    all_bins: set[str] = set()
    for d in list(kill_counts.values()) + list(death_counts.values()):
        all_bins.update(d.keys())

    def _bin_key(label: str) -> int:
        """Retourne le nombre de secondes du début du bin pour le tri."""
        part = label.rstrip("s")
        if "m" in part:
            m_str, s_str = part.split("m")
            return int(m_str) * 60 + int(s_str)
        return int(part)

    sorted_bins = sorted(all_bins, key=_bin_key)

    # Tick labels positionnés aux bordures des colonnes (i-0.5) plutôt qu'au centre
    n_bins = len(sorted_bins)
    border_tickvals = [i - 0.5 for i in range(n_bins + 1)]
    border_ticktext = [
        _format_bin_label(
            _bin_key(sorted_bins[i]) if i < n_bins else _bin_key(sorted_bins[-1]) + _BIN_SIZE_S
        )
        for i in range(n_bins + 1)
    ]

    fig = go.Figure()
    for name in player_names:
        color = colors_by_name.get(name, "#888888")
        y_frag = [kill_counts[name].get(b, 0) for b in sorted_bins]
        y_death = [-death_counts[name].get(b, 0) for b in sorted_bins]

        fig.add_trace(
            go.Bar(
                x=sorted_bins,
                y=y_frag,
                name=name,
                legendgroup=name,
                showlegend=True,
                marker_color=color,
                customdata=y_frag,
                hovertemplate=(
                    f"<b>{name}</b><br>%{{x}}<br>{label_frag} : %{{customdata}} matchs<extra></extra>"
                ),
            )
        )
        fig.add_trace(
            go.Bar(
                x=sorted_bins,
                y=y_death,
                name=name,
                legendgroup=name,
                showlegend=False,
                marker_color=color,
                customdata=[-v for v in y_death],
                hovertemplate=(
                    f"<b>{name}</b><br>%{{x}}<br>{label_death} : %{{customdata}} matchs<extra></extra>"
                ),
            )
        )

    max_kill = max((max(d.values(), default=0) for d in kill_counts.values()), default=1)
    max_death = max((max(d.values(), default=0) for d in death_counts.values()), default=1)
    max_count = max(max_kill, max_death, 1)
    step = max(1, max_count // 5)
    tickvals = sorted(set(range(-(max_count + step), max_count + step + 1, step)))
    ticktext = [str(abs(v)) for v in tickvals]

    # Séparateurs verticaux entre chaque tranche de 15 s
    col_shapes = [
        {
            "type": "line",
            "x0": i - 0.5,
            "x1": i - 0.5,
            "y0": 0,
            "y1": 1,
            "xref": "x",
            "yref": "paper",
            "line": {"color": "rgba(255,255,255,0.18)", "width": 1, "dash": "dot"},
        }
        for i in range(1, len(sorted_bins))
    ]

    fig.update_layout(
        title=t("tm_first_events_title", lang=lang),
        barmode="group",
        height=420,
        margin={"l": 50, "r": 20, "t": 60, "b": 80},
        paper_bgcolor="rgba(0,0,0,0)",
        plot_bgcolor="rgba(0,0,0,0)",
        shapes=col_shapes,
        legend={
            "orientation": "h",
            "yanchor": "top",
            "y": -0.18,
            "xanchor": "center",
            "x": 0.5,
        },
        xaxis={
            "gridcolor": "rgba(0,0,0,0)",
            "tickfont": {"color": "white", "size": 13, "family": "Arial"},
            "title_font": {"color": "white"},
            "tickmode": "array",
            "tickvals": border_tickvals,
            "ticktext": border_ticktext,
        },
        yaxis={
            "gridcolor": "rgba(255,255,255,0.08)",
            "zerolinecolor": "rgba(255,255,255,0.5)",
            "zerolinewidth": 2,
            "tickvals": tickvals,
            "ticktext": ticktext,
            "title_text": t("col_matches", lang=lang),
        },
    )
    fig.add_annotation(
        text=f"▲ {label_frag}",
        x=0.01,
        y=1.0,
        xref="paper",
        yref="paper",
        showarrow=False,
        font={"color": "rgba(255,255,255,0.65)", "size": 11},
        xanchor="left",
        yanchor="top",
    )
    fig.add_annotation(
        text=f"▼ {label_death}",
        x=0.01,
        y=0.0,
        xref="paper",
        yref="paper",
        showarrow=False,
        font={"color": "rgba(255,255,255,0.65)", "size": 11},
        xanchor="left",
        yanchor="bottom",
    )
    return fig


@fragment_if_available
def render_first_events_chart(
    db_path: str,
    xuids_by_name: dict[str, str],
    match_ids: list[str],
    colors_by_name: dict[str, str],
    key_suffix: str,
) -> None:
    """Affiche la tendance chronologique du premier frag et de la première mort.

    Args:
        db_path: Chemin vers la DB (shared accessible).
        xuids_by_name: Mapping nom joueur → xuid.
        match_ids: Matchs d'escouade à considérer.
        colors_by_name: Palette de couleurs par nom.
        key_suffix: Suffixe unique pour la clé Streamlit.
    """
    _lang = get_lang()
    df = _load_first_events_data(db_path, xuids_by_name, match_ids)
    if df.is_empty() or "player_name" not in df.columns:
        st.caption(t("insufficient_data_chart"))
        return

    fig = _build_first_events_fig(df, colors_by_name, _lang)
    st.plotly_chart(
        _hide_legend(fig),
        width="stretch",
        key=f"first_events_{key_suffix}",
        config=PLOTLY_CLEAN_CONFIG,
    )
