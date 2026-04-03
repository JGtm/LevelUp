"""Graphiques en barres pour les métriques par match.

Ce module contient les fonctions pour afficher des graphiques
en barres chronologiques des statistiques de match.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.ui.date_formats import FMT_SHORT_DATETIME_FR, FMT_TICK_DATETIME
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom


def plot_metric_bars_by_match(  # noqa: PLR0913
    df_: DataFrameLike,
    *,
    metric_col: str,
    title: str | None = None,
    y_axis_title: str,
    hover_label: str,
    bar_color: str,
    smooth_color: str,
    smooth_window: int = 10,
    lang: str = "fr",
) -> go.Figure | None:
    """Graphique en barres d'une métrique par match avec courbe de moyenne lissée.

    Args:
        df_: DataFrame avec colonnes 'start_time' et la métrique.
        metric_col: Nom de la colonne de métrique.
        title: Titre du graphique.
        y_axis_title: Titre de l'axe Y.
        hover_label: Label pour le hover.
        bar_color: Couleur des barres.
        smooth_color: Couleur de la courbe lissée.
        smooth_window: Fenêtre de lissage (défaut: 10).

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if df_ is None:
        return None
    df_pl = ensure_polars(df_)
    if df_pl.is_empty():
        return None
    if metric_col not in df_pl.columns or "start_time" not in df_pl.columns:
        return None

    extra_cols = [c for c in ("map_name",) if c in df_pl.columns]
    d = df_pl.select(["start_time", metric_col, *extra_cols])
    # Convertir start_time en Datetime si nécessaire
    st_dtype = d.schema["start_time"]
    if st_dtype == pl.String or st_dtype == pl.Utf8:
        d = d.with_columns(pl.col("start_time").str.to_datetime(strict=False))
    elif st_dtype == pl.Date:
        d = d.with_columns(pl.col("start_time").cast(pl.Datetime))
    d = d.drop_nulls("start_time").sort("start_time")
    if d.is_empty():
        return None

    d = d.with_columns(pl.col(metric_col).cast(pl.Float64, strict=False))
    y = d.get_column(metric_col).to_list()
    x_idx = list(range(len(d)))
    _map_tick_col = "map_ui" if "map_ui" in d.columns else "map_name"
    labels = (
        [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(d.get_column(_map_tick_col).fill_null("").to_list())
        ]
        if _map_tick_col in d.columns
        else d.get_column("start_time").dt.strftime(FMT_TICK_DATETIME).to_list()
    )
    step = max(1, len(labels) // 10) if labels else 1

    w = int(smooth_window) if smooth_window else 0
    if w and w > 1:
        smooth = (
            d.get_column(metric_col).rolling_mean(window_size=max(1, w), min_samples=1).to_list()
        )
    else:
        smooth = y

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=x_idx,
            y=y,
            name=y_axis_title,
            marker_color=bar_color,
            opacity=0.70,
            hovertemplate=f"{hover_label}=%{{y}}<extra></extra>",
        )
    )

    fig.add_trace(
        go.Scatter(
            x=x_idx,
            y=smooth,
            mode="lines",
            name=viz_t("trace_avg_smoothed", lang),
            line={"width": 3, "color": smooth_color},
            hovertemplate=viz_t("hover_avg_smoothed", lang),
        )
    )
    layout_kwargs: dict = {
        "margin": {"l": 40, "r": 20, "t": 40 if title else 10, "b": 90},
        "hovermode": "x unified",
        "legend": get_legend_horizontal_bottom(),
    }
    if title is not None:
        layout_kwargs["title"] = title
    fig.update_layout(**layout_kwargs)
    fig.update_yaxes(title_text=y_axis_title, rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=x_idx[::step],
        ticktext=labels[::step],
        tickangle=-45,
        type="category",
    )

    return apply_halo_plot_style(fig, height=320)


def plot_multi_metric_bars_by_match(  # noqa: C901, PLR0912, PLR0913, PLR0915
    series: list[tuple[str, DataFrameLike]],
    *,
    metric_col: str,
    title: str,
    y_axis_title: str,
    hover_label: str,
    colors: dict[str, str] | list[str] | None,
    smooth_window: int = 10,
    show_smooth_lines: bool = True,
    lang: str = "fr",
    records: dict[str, dict[str, float | None]] | None = None,
    records_per_map: dict[str, dict[str, dict[str, float | None]]] | None = None,
) -> go.Figure | None:
    """Graphique en barres multi-joueurs d'une métrique par match.

    Args:
        series: Liste de tuples (nom, DataFrame).
        metric_col: Nom de la colonne de métrique.
        title: Titre du graphique.
        y_axis_title: Titre de l'axe Y.
        hover_label: Label pour le hover.
        colors: Dict ou liste de couleurs par joueur.
        smooth_window: Fenêtre de lissage (défaut: 10).
        show_smooth_lines: Afficher les courbes lissées.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if not series:
        return None

    # Normaliser tous les DataFrames en Polars
    normalized: list[tuple[str, pl.DataFrame]] = []
    for name, df_ in series:
        if df_ is None:
            continue
        df_pl = ensure_polars(df_)
        if df_pl.is_empty():
            continue
        normalized.append((str(name), df_pl))

    if not normalized:
        return None

    # Vérifier si match_id / map_name sont disponibles dans tous les DataFrames
    has_match_id = all("match_id" in df_pl.columns for _, df_pl in normalized)
    has_map_name = any("map_name" in df_pl.columns for _, df_pl in normalized)

    prepared: list[tuple[str, pl.DataFrame]] = []
    all_match_data: list[pl.DataFrame] = []  # Pour construire l'axe X commun

    for name, df_pl in normalized:
        if metric_col not in df_pl.columns or "start_time" not in df_pl.columns:
            continue

        cols_to_use = ["start_time", metric_col]
        if has_match_id:
            cols_to_use.append("match_id")
        # Inclure map_name si disponible dans ce DataFrame
        if has_map_name and "map_name" in df_pl.columns:
            cols_to_use.append("map_name")

        d = df_pl.select(cols_to_use)
        # Convertir start_time en Datetime si nécessaire
        st_dtype = d.schema["start_time"]
        if st_dtype == pl.String or st_dtype == pl.Utf8:
            d = d.with_columns(pl.col("start_time").str.to_datetime(strict=False))
        elif st_dtype == pl.Date:
            d = d.with_columns(pl.col("start_time").cast(pl.Datetime))
        d = d.drop_nulls("start_time").sort("start_time")

        # Dédoublonner par match_id (ou start_time si pas de match_id)
        # pour éviter les boucles visuelles causées par des duplicates
        if has_match_id:
            d = d.unique(subset=["match_id"], keep="first")
        else:
            d = d.unique(subset=["start_time"], keep="first")

        if d.is_empty():
            continue

        prepared.append((name, d))

        # Collecter les données pour l'axe X commun (vectorisé)
        if has_match_id:
            map_cols = [pl.col("match_id").cast(pl.String).alias("match_id"), pl.col("start_time")]
            _disp_col = "map_ui" if "map_ui" in d.columns else "map_name"
            if _disp_col in d.columns:
                map_cols.append(pl.col(_disp_col).fill_null("").alias("_map_display"))
            match_df = d.select(map_cols)
        else:
            map_cols_no_id = [
                pl.col("start_time").dt.strftime("%Y-%m-%dT%H:%M:%S").alias("match_id"),
                pl.col("start_time"),
            ]
            _disp_col_no_id = "map_ui" if "map_ui" in d.columns else "map_name"
            if _disp_col_no_id in d.columns:
                map_cols_no_id.append(pl.col(_disp_col_no_id).fill_null("").alias("_map_display"))
            match_df = d.select(map_cols_no_id)
        all_match_data.append(match_df)

    if not prepared or not all_match_data:
        return None

    # Construire l'axe X commun (vectorisé)
    combined = pl.concat(
        all_match_data, how="diagonal"
    )  # diagonal pour tolérer colonnes manquantes
    # Garder le premier start_time (et _map_display si dispo) par match_id
    agg_exprs = [pl.col("start_time").min()]
    if "_map_display" in combined.columns:
        agg_exprs.append(pl.col("_map_display").first())
    match_times = combined.group_by("match_id").agg(agg_exprs).sort("start_time")

    match_ids_ordered = match_times.get_column("match_id").to_list()
    idx_map_df = pl.DataFrame(
        {
            "_match_key": match_ids_ordered,
            "_x": list(range(len(match_ids_ordered))),
        }
    )

    # Labels pour l'axe X : #N + map_display si disponible, sinon dates
    n_matches = len(match_ids_ordered)
    if "_map_display" in match_times.columns:
        labels = [
            f"#{i + 1}<br>{mn}" if mn else f"#{i + 1}"
            for i, mn in enumerate(match_times.get_column("_map_display").fill_null("").to_list())
        ]
    else:
        labels = match_times.get_column("start_time").dt.strftime(FMT_SHORT_DATETIME_FR).to_list()
    step = max(1, n_matches // 10) if n_matches else 1

    fig = go.Figure()
    w = int(smooth_window) if smooth_window else 0
    _player_xs: dict[str, list[int]] = {}

    for i, (name, d) in enumerate(prepared):
        if isinstance(colors, dict):
            color = colors.get(name) or "#35D0FF"
        elif isinstance(colors, list) and colors:
            color = colors[i % len(colors)]
        else:
            color = "#35D0FF"

        # Cast métrique en Float64 et filtrer les nulls
        d = d.with_columns(pl.col(metric_col).cast(pl.Float64, strict=False).alias("_y"))
        d = d.filter(pl.col("_y").is_not_null())
        if d.is_empty():
            continue

        # Ajouter la clé de match pour mapper vers l'axe X commun
        if has_match_id:
            d = d.with_columns(pl.col("match_id").cast(pl.String).alias("_match_key"))
        else:
            d = d.with_columns(
                pl.col("start_time").dt.strftime("%Y-%m-%dT%H:%M:%S").alias("_match_key")
            )

        # Joindre pour obtenir l'index X commun
        d = d.join(idx_map_df, on="_match_key", how="inner")
        if d.is_empty():
            continue

        # Trier par _x pour garantir l'ordre correct (évite les boucles visuelles)
        d = d.sort("_x")

        # Calculer la moyenne lissée APRÈS le tri par _x (ordre d'affichage)
        # pour que les valeurs lissées correspondent au bon ordre chronologique visuel
        if bool(show_smooth_lines):
            if w and w > 1:
                d = d.with_columns(
                    pl.col("_y").rolling_mean(window_size=max(1, w), min_samples=1).alias("_smooth")
                )
            else:
                d = d.with_columns(pl.col("_y").alias("_smooth"))

        x = d.get_column("_x").to_list()
        y2_sorted = d.get_column("_y").to_list()

        if not x:
            continue

        _player_xs[name] = x
        fig.add_trace(
            go.Bar(
                x=x,
                y=y2_sorted,
                name=name,
                marker_color=color,
                opacity=0.70,
                hovertemplate=f"{name}<br>{hover_label}=%{{y}}<extra></extra>",
                legendgroup=name,
            )
        )

        if bool(show_smooth_lines):
            smooth_sorted = d.get_column("_smooth").to_list()

            fig.add_trace(
                go.Scatter(
                    x=x,
                    y=smooth_sorted,
                    mode="lines",
                    name=f"{name} — {viz_t('trace_avg_smoothed', lang)}",
                    line={"width": 3, "color": color},
                    opacity=0.95,
                    hovertemplate=f"{name}<br>{viz_t('hover_avg_smoothed', lang)}",
                    legendgroup=name,
                )
            )

    if not fig.data:
        return None

    if records and _player_xs:
        from src.visualization._squad_record_shapes import add_record_shapes
        _names_with_data = list(_player_xs.keys())
        _map_names_x = [
            labels[xi].split("<br>", 1)[1] if xi < len(labels) and "<br>" in labels[xi] else None
            for xi in range(len(match_ids_ordered))
        ]
        _colors: dict[str, str] = colors if isinstance(colors, dict) else {}
        for _pname, _xs in _player_xs.items():
            _pm = (
                {_pname: (records_per_map.get(_pname) or {}).get(metric_col, {})}
                if records_per_map else None
            )
            add_record_shapes(
                fig,
                xs=_xs,
                records={_pname: (records.get(_pname) or {}).get(metric_col)},
                player_names=_names_with_data,
                n_players=len(_names_with_data),
                is_negative=False,
                colors_by_name=_colors,
                per_map_records=_pm,
                map_names_per_x=[_map_names_x[xi] for xi in _xs if xi < len(_map_names_x)],
            )

    fig.update_layout(
        title=title,
        margin={"l": 40, "r": 20, "t": 40, "b": 150},
        hovermode="x unified",
        legend={**get_legend_horizontal_bottom(), "y": -0.52},
        barmode="group",
    )
    fig.update_yaxes(title_text=y_axis_title, rangemode="tozero")
    fig.update_xaxes(
        title_text=viz_t("axis_match_number", lang),
        tickmode="array",
        tickvals=list(range(len(labels)))[::step],
        ticktext=labels[::step],
        tickangle=-45,
    )

    return apply_halo_plot_style(fig, height=320)
