"""Graphiques pour visualisation des antagonistes (némésis/souffre-douleur).

Sprint 5 - Visualisations Plotly pour :
- Barres empilées killer-victim
- Timeseries K/D par minute
- Heatmap de duels
- Graphiques de tendance

Références :
- src/analysis/killer_victim.py : Fonctions d'analyse Polars
- src/visualization/theme.py : Style Halo pour les graphiques
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

# =============================================================================
# Configuration des couleurs
# =============================================================================

# Couleurs accessibles (palette Okabe-Ito) pour les graphiques d'antagonistes.
# Remplacement des couleurs néon précédentes (#00ff00, #ff4444) non accessibles
# aux daltoniens (deuteranopie/protanopie).
COLORS = {
    "kills": "#009E73",  # Vert bleuté Okabe-Ito — Mes kills
    "deaths": "#D55E00",  # Vermillon Okabe-Ito — Mes morts
    "nemesis": "#E69F00",  # Orange Okabe-Ito — Némésis
    "victim": "#0072B2",  # Bleu Okabe-Ito — Victime
    "neutral": "#888888",  # Gris — Neutre
    "positive_kd": "#009E73",  # Vert bleuté Okabe-Ito — K/D positif
    "negative_kd": "#D55E00",  # Vermillon Okabe-Ito — K/D négatif
    "team_alpha": "#0072B2",  # Bleu Okabe-Ito — équipe alpha
    "team_bravo": "#E69F00",  # Orange Okabe-Ito — équipe bravo
    "highlight": "#F0E442",  # Jaune Okabe-Ito — Highlight
}

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   #00ff00 (vert néon), #ff4444 (rouge néon), #00aaff (bleu vif), #ff6600 (orange vif),
#   #aa00ff (violet), #ffff00 (jaune saturé), #00ffaa (cyan-vert), #ff00aa (fuchsia),
#   #aaffaa (vert pâle), #ffaaaa (rose pâle), #aaaaff (lavande), #ffaaff (orchidée)
# Palette de couleurs daltonien-friendly pour les joueurs (jusqu'à 12)
# Basée sur la palette Okabe-Ito + extensions accessibles (Wong 2011)
PLAYER_COLORS = [
    "#0072B2",  # Bleu
    "#E69F00",  # Orange
    "#009E73",  # Vert bleuté
    "#D55E00",  # Vermillon
    "#56B4E9",  # Bleu ciel
    "#CC79A7",  # Rose mauve
    "#F0E442",  # Jaune
    "#999999",  # Gris
    "#332288",  # Indigo
    "#44AA99",  # Sarcelle
    "#DDCC77",  # Sable
    "#AA4499",  # Violet
]


# =============================================================================
# Graphiques principaux
# =============================================================================


def plot_killer_victim_stacked_bars(  # noqa: PLR0913
    pairs_df: pl.DataFrame,
    match_id: str | None = None,
    *,
    me_xuid: str | None = None,
    rank_by_xuid: dict[str, int] | None = None,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Graphique barres empilées : une ligne par tueur, segments = victimes (plus tuée en premier), lignes triées par rang.

    Args:
        pairs_df: DataFrame Polars avec colonnes killer_xuid, killer_gamertag,
                  victim_xuid, victim_gamertag, kill_count.
        match_id: Si fourni, filtre pour ce match uniquement.
        me_xuid: XUID du joueur principal (conservé pour compatibilité).
        rank_by_xuid: Mapping xuid → rang (1 = meilleur). Si None, tri par total kills décroissant.
        title: Titre du graphique.
        height: Hauteur de base en pixels (ajustée selon le nombre de tueurs).

    Returns:
        Figure Plotly avec barres empilées (une par tueur, segments = victimes).
    """
    if title is None:
        title = viz_t("title_elim_victim", lang)
    fig = go.Figure()

    if pairs_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": COLORS["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    filtered_df = pairs_df
    if match_id:
        filtered_df = filtered_df.filter(pl.col("match_id") == match_id)

    if filtered_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_match_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": COLORS["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    # Agrégation par (killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
    agg_df = filtered_df.group_by(
        "killer_xuid", "killer_gamertag", "victim_xuid", "victim_gamertag"
    ).agg(pl.col("kill_count").sum().alias("count"))

    # Total kills par tueur (pour tri et pour ordre des barres)
    killer_totals = agg_df.group_by("killer_xuid", "killer_gamertag").agg(
        pl.col("count").sum().alias("total")
    )
    killers_sorted: list[tuple[str, str, int]] = [
        (row["killer_xuid"], row["killer_gamertag"], row["total"])
        for row in killer_totals.iter_rows(named=True)
    ]
    rank_map = rank_by_xuid or {}
    killers_sorted.sort(
        key=lambda k: (
            rank_map.get(k[0], 999),
            -k[2],
            k[1],
        )
    )
    killer_xuids = [k[0] for k in killers_sorted]
    killer_labels = [k[1] for k in killers_sorted]
    n_killers = len(killer_labels)

    # Ordre des victimes : par total kills global décroissant (premier segment = victime la plus tuée au global)
    victim_totals = (
        agg_df.group_by("victim_xuid", "victim_gamertag")
        .agg(pl.col("count").sum().alias("total"))
        .sort("total", descending=True)
    )
    victim_order: list[tuple[str, str]] = [
        (row["victim_xuid"], row["victim_gamertag"]) for row in victim_totals.iter_rows(named=True)
    ]

    # Matrice (killer_idx, victim_idx) -> count
    kv_lookup: dict[tuple[str, str], int] = {}
    for row in agg_df.iter_rows(named=True):
        kv_lookup[(row["killer_xuid"], row["victim_xuid"])] = int(row["count"])

    # Une trace par victime (ordre = premier segment en bas du stack = plus tuée)
    for idx, (v_xuid, v_label) in enumerate(victim_order):
        color = PLAYER_COLORS[idx % len(PLAYER_COLORS)]
        x_vals = [kv_lookup.get((k_xuid, v_xuid), 0) for k_xuid in killer_xuids]
        if sum(x_vals) == 0:
            continue
        safe_v = str(v_label).replace("&", "&amp;").replace("<", "&lt;")
        fig.add_trace(
            go.Bar(
                name=v_label,
                y=killer_labels,
                x=x_vals,
                orientation="h",
                marker={"color": color},
                hovertemplate=f"<b>%{{y}}</b> → <b>{safe_v}</b><br>{viz_t('axis_kills', lang)}: %{{x}}<extra></extra>",
            )
        )

    fig.update_layout(
        barmode="stack",
        xaxis_title=viz_t("axis_frag_count", lang),
        yaxis_title=viz_t("axis_killer", lang),
        yaxis={"categoryorder": "array", "categoryarray": killer_labels, "autorange": "reversed"},
        margin={"l": 140},
        showlegend=True,
        legend={
            "orientation": "v",
            "yanchor": "top",
            "y": 1,
            "xanchor": "left",
            "x": 1.02,
        },
    )

    # Hauteur adaptée au nombre de tueurs
    plot_height = max(height, 80 + 24 * n_killers)
    return apply_halo_plot_style(fig, title=title, height=plot_height)


def plot_kd_timeseries(
    timeseries_df: pl.DataFrame,
    *,
    title: str | None = None,
    show_cumulative: bool = True,
    height: int = 350,
    lang: str = "fr",
) -> go.Figure:
    """Graphique timeseries du K/D par minute.

    Affiche l'évolution du K/D au cours du match, minute par minute.

    Args:
        timeseries_df: DataFrame Polars avec colonnes minute, kills, deaths,
                       net_kd, cumulative_net_kd (depuis compute_kd_timeseries_by_minute_polars).
        title: Titre du graphique.
        show_cumulative: Si True, affiche aussi la courbe cumulative.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec la timeseries.
    """
    title = title or viz_t("title_kd_per_min", lang)
    fig = go.Figure()

    if timeseries_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": COLORS["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    minutes = timeseries_df["minute"].to_list()
    kills = timeseries_df["kills"].to_list()
    deaths = timeseries_df["deaths"].to_list()
    timeseries_df["net_kd"].to_list()

    # Barres pour kills par minute
    fig.add_trace(
        go.Bar(
            name=viz_t("trace_kills", lang),
            x=minutes,
            y=kills,
            marker={"color": COLORS["kills"], "opacity": 0.7},
            hovertemplate=viz_t("hover_kills_per_min", lang),
        )
    )

    # Barres pour morts par minute (négatif)
    fig.add_trace(
        go.Bar(
            name=viz_t("trace_deaths", lang),
            x=minutes,
            y=[-d for d in deaths],
            marker={"color": COLORS["deaths"], "opacity": 0.7},
            hovertemplate=viz_t("hover_deaths_per_min", lang),
            customdata=deaths,
        )
    )

    # Ligne de K/D net cumulatif
    if show_cumulative and "cumulative_net_kd" in timeseries_df.columns:
        cumulative = timeseries_df["cumulative_net_kd"].to_list()

        # Couleur basée sur la valeur finale
        final_color = COLORS["positive_kd"] if cumulative[-1] >= 0 else COLORS["negative_kd"]

        fig.add_trace(
            go.Scatter(
                name=viz_t("trace_kd_cumul", lang),
                x=minutes,
                y=cumulative,
                mode="lines+markers",
                line={"color": final_color, "width": 3},
                marker={"size": 6},
                yaxis="y2",
                hovertemplate=viz_t("hover_net_kd_cumul", lang),
            )
        )

    # Mise en forme avec axe Y secondaire
    fig.update_layout(
        barmode="relative",
        xaxis_title=viz_t("axis_minute", lang),
        yaxis_title=viz_t("axis_kills_deaths", lang),
        yaxis2={
            "title": viz_t("trace_kd_cumul", lang),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
        },
        showlegend=True,
        legend={
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "center",
            "x": 0.5,
        },
    )

    # Ligne horizontale à 0
    fig.add_hline(y=0, line_width=1, line_color=COLORS["neutral"], line_dash="dash")

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_duel_history(  # noqa: PLR0913
    duel_df: pl.DataFrame,
    me_gamertag: str,
    opponent_gamertag: str,
    *,
    title: str | None = None,
    height: int = 300,
    lang: str = "fr",
) -> go.Figure:
    """Graphique de l'historique des duels entre deux joueurs.

    Args:
        duel_df: DataFrame Polars avec colonnes match_id, my_kills, opponent_kills, net
                 (depuis compute_duel_history_polars).
        me_gamertag: Gamertag du joueur principal.
        opponent_gamertag: Gamertag de l'adversaire.
        title: Titre du graphique (auto-généré si None).
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec l'historique des duels.
    """
    if title is None:
        title = f"Historique des duels : {me_gamertag} vs {opponent_gamertag}"

    fig = go.Figure()

    if duel_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_duel", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": COLORS["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    matches = list(range(len(duel_df)))
    my_kills = duel_df["my_kills"].to_list()
    opponent_kills = duel_df["opponent_kills"].to_list()
    net_values = duel_df["net"].to_list()

    # Barres groupées
    fig.add_trace(
        go.Bar(
            name=me_gamertag,
            x=matches,
            y=my_kills,
            marker={"color": COLORS["kills"]},
            hovertemplate=f"<b>{me_gamertag}</b><br>{viz_t('axis_kills', lang)}: %{{y}}<extra></extra>",
        )
    )

    fig.add_trace(
        go.Bar(
            name=opponent_gamertag,
            x=matches,
            y=opponent_kills,
            marker={"color": COLORS["deaths"]},
            hovertemplate=f"<b>{opponent_gamertag}</b><br>{viz_t('axis_kills', lang)}: %{{y}}<extra></extra>",
        )
    )

    # Ligne de net
    fig.add_trace(
        go.Scatter(
            name=viz_t("label_net", lang),
            x=matches,
            y=net_values,
            mode="lines+markers",
            line={"color": COLORS["highlight"], "width": 2},
            yaxis="y2",
            hovertemplate="Net: %{y:+d}<extra></extra>",
        )
    )

    # Calculer les totaux
    total_my_kills = sum(my_kills)
    total_opponent_kills = sum(opponent_kills)
    total_net = total_my_kills - total_opponent_kills

    # Ajouter annotation avec les totaux
    win_status = (
        viz_t("label_win", lang)
        if total_net > 0
        else (viz_t("label_tie", lang) if total_net == 0 else viz_t("label_loss", lang))
    )
    annotation_text = f"Total: {total_my_kills}-{total_opponent_kills} ({win_status})"

    fig.add_annotation(
        text=annotation_text,
        xref="paper",
        yref="paper",
        x=1,
        y=1.15,
        showarrow=False,
        font={
            "size": 14,
            "color": COLORS["positive_kd"] if total_net >= 0 else COLORS["negative_kd"],
        },
        xanchor="right",
    )

    fig.update_layout(
        barmode="group",
        xaxis_title=viz_t("axis_match_number", lang),
        yaxis_title=viz_t("axis_frag_count", lang),
        yaxis2={
            "title": viz_t("label_balance", lang),
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
        },
        showlegend=True,
        legend={
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "center",
            "x": 0.5,
        },
    )

    # Ligne à 0 sur l'axe secondaire
    fig.add_hline(
        y=0,
        line_width=1,
        line_color=COLORS["neutral"],
        line_dash="dash",
        yref="y2",
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_nemesis_victim_summary(
    nemesis_data: dict[str, Any],
    victim_data: dict[str, Any],
    *,
    title: str | None = None,
    height: int = 250,
    lang: str = "fr",
) -> go.Figure:
    """Graphique résumé du némésis et souffre-douleur.

    Affiche les statistiques des deux principaux antagonistes.

    Args:
        nemesis_data: Dict avec gamertag, times_killed_by, matches.
        victim_data: Dict avec gamertag, times_killed, matches.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec le résumé.
    """
    if title is None:
        title = viz_t("title_nemesis_victim", lang)
    fig = make_subplots(
        rows=1,
        cols=2,
        subplot_titles=(viz_t("label_nemesis", lang), viz_t("label_punching_bag", lang)),
        specs=[[{"type": "indicator"}, {"type": "indicator"}]],
    )

    # Némésis
    nemesis_gt = nemesis_data.get("gamertag", "N/A")
    nemesis_count = nemesis_data.get("times_killed_by", 0)

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=nemesis_count,
            number={
                "suffix": f" {viz_t('suffix_deaths', lang)}",
                "font": {"size": 36, "color": COLORS["nemesis"]},
            },
            title={
                "text": f"<b>{nemesis_gt}</b>",
                "font": {"size": 16},
            },
        ),
        row=1,
        col=1,
    )

    # Victime
    victim_gt = victim_data.get("gamertag", "N/A")
    victim_count = victim_data.get("times_killed", 0)

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=victim_count,
            number={
                "suffix": f" {viz_t('suffix_kills', lang)}",
                "font": {"size": 36, "color": COLORS["victim"]},
            },
            title={
                "text": f"<b>{victim_gt}</b>",
                "font": {"size": 16},
            },
        ),
        row=1,
        col=2,
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_killer_victim_heatmap(
    matrix_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 500,
    lang: str = "fr",
) -> go.Figure:
    """Heatmap de la matrice killer-victim.

    Args:
        matrix_df: DataFrame Polars pivotée (killer en index, victim en colonnes)
                   depuis killer_victim_matrix_polars.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec la heatmap.
    """
    if title is None:
        title = viz_t("title_killer_victim_matrix", lang)
    fig = go.Figure()

    if matrix_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": COLORS["neutral"]},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    # Extraire les données
    killers = matrix_df["killer_gamertag"].to_list()
    victim_cols = [c for c in matrix_df.columns if c != "killer_gamertag"]

    # Créer la matrice de valeurs
    z_values = []
    for row in matrix_df.iter_rows(named=True):
        row_values = [row.get(col, 0) or 0 for col in victim_cols]
        z_values.append(row_values)

    fig.add_trace(
        go.Heatmap(
            z=z_values,
            x=victim_cols,
            y=killers,
            colorscale=[
                [0, "#1a1a2e"],
                [0.25, "#16213e"],
                [0.5, "#0f3460"],
                [0.75, "#e94560"],
                [1, "#ff6b6b"],
            ],
            hoverongaps=False,
            hovertemplate=viz_t("hover_heatmap_kills", lang),
        )
    )

    fig.update_layout(
        xaxis_title=viz_t("axis_victim", lang),
        yaxis_title=viz_t("axis_killer", lang),
        xaxis={"side": "bottom"},
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_top_antagonists_bars(  # noqa: PLR0913
    nemeses: list[dict],
    victims: list[dict],
    *,
    top_n: int = 5,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Graphique barres horizontales des top antagonistes.

    Args:
        nemeses: Liste de dicts avec killer_gamertag, times_killed_by.
        victims: Liste de dicts avec victim_gamertag, times_killed.
        top_n: Nombre de joueurs à afficher par catégorie.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec les top antagonistes.
    """
    if title is None:
        title = viz_t("title_top_antagonists", lang)
    fig = make_subplots(
        rows=1,
        cols=2,
        subplot_titles=(
            f"Top {top_n} {viz_t('label_nemesis', lang)}",
            f"Top {top_n} {viz_t('label_punching_bag', lang)}",
        ),
        horizontal_spacing=0.15,
    )

    # Top Némésis
    top_nemeses = nemeses[:top_n] if nemeses else []
    if top_nemeses:
        nem_names = [n.get("killer_gamertag", "?") for n in reversed(top_nemeses)]
        nem_counts = [n.get("times_killed_by", 0) for n in reversed(top_nemeses)]

        fig.add_trace(
            go.Bar(
                y=nem_names,
                x=nem_counts,
                orientation="h",
                marker={"color": COLORS["nemesis"]},
                name=viz_t("trace_deaths", lang),
                hovertemplate=viz_t("hover_killed_by", lang),
            ),
            row=1,
            col=1,
        )

    # Top Victimes
    top_victims = victims[:top_n] if victims else []
    if top_victims:
        vic_names = [v.get("victim_gamertag", "?") for v in reversed(top_victims)]
        vic_counts = [v.get("times_killed", 0) for v in reversed(top_victims)]

        fig.add_trace(
            go.Bar(
                y=vic_names,
                x=vic_counts,
                orientation="h",
                marker={"color": COLORS["victim"]},
                name=viz_t("trace_kills", lang),
                hovertemplate=viz_t("hover_i_killed", lang),
            ),
            row=1,
            col=2,
        )

    fig.update_layout(
        showlegend=False,
    )

    fig.update_xaxes(title_text=viz_t("axis_deaths", lang), row=1, col=1)
    fig.update_xaxes(title_text=viz_t("axis_kills", lang), row=1, col=2)

    return apply_halo_plot_style(fig, title=title, height=height)


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_antagonist_chart_colors() -> dict[str, str]:
    """Retourne le dictionnaire des couleurs pour les graphiques antagonistes.

    Utile pour harmoniser les couleurs dans l'UI.

    Returns:
        Dict avec les codes couleur hex.
    """
    return COLORS.copy()


def create_kd_indicator(
    kills: int,
    deaths: int,
    *,
    title: str = "K/D",
    height: int = 150,
) -> go.Figure:
    """Crée un indicateur K/D simple.

    Args:
        kills: Nombre de kills.
        deaths: Nombre de morts.
        title: Titre de l'indicateur.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec l'indicateur.
    """
    kd_ratio = kills / deaths if deaths > 0 else kills

    # Couleur basée sur le ratio
    if kd_ratio >= 1.5:
        color = COLORS["positive_kd"]
    elif kd_ratio >= 1.0:
        color = COLORS["highlight"]
    else:
        color = COLORS["negative_kd"]

    fig = go.Figure()

    fig.add_trace(
        go.Indicator(
            mode="number+delta",
            value=kd_ratio,
            number={
                "valueformat": ".2f",
                "font": {"size": 48, "color": color},
            },
            delta={
                "reference": 1.0,
                "valueformat": ".2f",
                "increasing": {"color": COLORS["positive_kd"]},
                "decreasing": {"color": COLORS["negative_kd"]},
            },
            title={
                "text": f"<b>{title}</b><br><span style='font-size:12px'>{kills}K / {deaths}D</span>",
                "font": {"size": 14},
            },
        )
    )

    return apply_halo_plot_style(fig, title="", height=height)
