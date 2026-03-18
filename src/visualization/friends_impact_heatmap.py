"""Visualisation heatmap d'impact des coéquipiers (Sprint 12).

Génère une heatmap montrant les événements clés par joueur/match :
- 🟢 First Blood (Premier sang)
- 🟡 Clutch Finisher (Finisseur)
- 🔴 Last Casualty (Boulet)

Et un tableau de ranking "Taquinerie" avec MVP et Boulet du groupe.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import plotly.graph_objects as go
import polars as pl

from src.analysis.friends_impact import ImpactEvent
from src.config import HALO_COLORS
from src.data.domain.refdata import Outcome
from src.ui.i18n.viz import viz_t
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style

if TYPE_CHECKING:
    pass

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   first_blood:    #2ecc71 (vert)      → #009E73 (vert bleuté Okabe-Ito)
#   clutch_finisher:#f39c12 (or/orange) → #E69F00 (orange Okabe-Ito)
#   last_casualty:  #e74c3c (rouge)     → #D55E00 (vermillon Okabe-Ito)
#   win:            #10b981 (vert)      → #009E73 (vert bleuté Okabe-Ito)
#   loss:           #ef4444 (rouge)     → #D55E00 (vermillon Okabe-Ito)
#   tie:            #8b5cf6 (violet)    → #CC79A7 (rose mauve Okabe-Ito)
IMPACT_COLORS = {
    "first_blood": "#009E73",  # Vert bleuté Okabe-Ito
    "clutch_finisher": "#E69F00",  # Orange Okabe-Ito
    "last_casualty": "#D55E00",  # Vermillon Okabe-Ito
    "none": "rgba(100, 100, 100, 0.1)",  # Gris transparent
}

# Couleurs pour les outcomes (Win/Loss/Tie) — palette Okabe-Ito
OUTCOME_COLORS = {
    "win": "#009E73",  # Vert bleuté Okabe-Ito (victories)
    "loss": "#D55E00",  # Vermillon Okabe-Ito (defeats)
    "tie": "#CC79A7",  # Rose mauve Okabe-Ito (ties)
    "unknown": "rgba(100, 100, 100, 0.3)",  # Gris
}

# Labels d'événements (symboles simplifiés sans emoji redondant)
EVENT_LABELS = {
    "first_blood": "⚡",  # Premier sang
    "clutch_finisher": "🎯",  # Finisseur
    "last_casualty": "💀",  # Boulet
    "last_group_kill": "🐌",  # Touriste (dernier à obtenir son 1er kill)
    "first_group_death": "🪦",  # Première victime
}


def plot_friends_impact_heatmap(  # noqa: C901, PLR0912, PLR0915
    impact_matrix: pl.DataFrame,
    *,
    title: str | None = None,
    max_matches: int = 50,
    height: int | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Crée une heatmap des événements d'impact par joueur et match.

    Args:
        impact_matrix: DataFrame Polars avec colonnes :
            - match_id, gamertag, events (liste de tuples), outcome
        title: Titre optionnel.
        max_matches: Nombre maximum de matchs à afficher.
        height: Hauteur optionnelle.

    Returns:
        Figure Plotly avec la heatmap.
    """
    colors = HALO_COLORS.as_dict()

    if impact_matrix.is_empty():
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_impact_events", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 14, "color": colors.get("slate", "#64748b")},
        )
        fig.update_layout(height=height or 300)
        return apply_halo_plot_style(fig, title=title, height=height)

    # Récupérer les valeurs uniques
    all_gamertags = sorted(impact_matrix["gamertag"].unique().to_list())
    match_ids = impact_matrix["match_id"].unique().to_list()

    # Séparer la ligne "Résultat" des joueurs et extraire les outcomes
    gamertags = [g for g in all_gamertags if g != "Résultat"]

    # Créer un mapping des outcomes par match_id
    match_outcomes_map = {}
    if "Résultat" in all_gamertags:
        outcome_events = impact_matrix.filter(pl.col("gamertag") == "Résultat")
        for row in outcome_events.iter_rows(named=True):
            match_id = str(row["match_id"])
            if row["outcome"] is not None:
                match_outcomes_map[match_id] = int(row["outcome"])

    # Limiter le nombre de matchs
    if len(match_ids) > max_matches:
        # Prendre les plus récents (on suppose triés par date descendante)
        match_ids = match_ids[:max_matches]
        impact_matrix = impact_matrix.filter(pl.col("match_id").is_in(match_ids))

    n_matches = len(match_ids)

    if n_matches == 0:
        fig = go.Figure()
        fig.update_layout(height=height or 300)
        return apply_halo_plot_style(fig, title=title, height=height)

    # Pivoter pour créer la matrice Z
    # Structure : [Joueur1, Joueur2, ...] × match_ids
    z_matrix = []
    text_matrix = []
    y_labels = []

    # Lignes joueurs uniquement
    for gamertag in gamertags:
        y_labels.append(gamertag)
        z_row = []
        text_row = []

        player_events = impact_matrix.filter(pl.col("gamertag") == gamertag)

        for match_id in match_ids:
            match_event = player_events.filter(pl.col("match_id") == match_id)

            if match_event.is_empty():
                z_row.append(0)
                text_row.append("")
            else:
                # Récupérer la liste d'événements (dicts avec "event" et "value")
                events = match_event["events"][0]
                if events is not None and len(events) > 0:
                    # Concaténer tous les symboles
                    symbols = [EVENT_LABELS.get(evt["event"], "") for evt in events]
                    text_row.append(" ".join(symbols))
                    # Utiliser la valeur du premier événement pour z (colorscale neutre)
                    z_row.append(0)  # Toujours 0 pour les joueurs (fond neutre)
                else:
                    z_row.append(0)
                    text_row.append("")

        z_matrix.append(z_row)
        text_matrix.append(text_row)

    # Créer la colorscale custom (simplifié, juste fond neutre)
    colorscale = [
        [0.0, IMPACT_COLORS["none"]],  # Gris transparent
        [1.0, IMPACT_COLORS["none"]],  # Gris transparent
    ]

    # Labels des matchs (afficher index ou raccourci)
    match_labels = [f"#{i + 1}" for i in range(n_matches)]

    fig = go.Figure(
        data=go.Heatmap(
            z=z_matrix,
            x=match_labels,
            y=y_labels,
            text=text_matrix,
            texttemplate="%{text}",
            textfont={"size": 19},  # Emojis réduits de 25% (25 → 19)
            colorscale=colorscale,
            zmin=0,
            zmax=1,
            showscale=False,
            hovertemplate=("<b>%{y}</b><br>Match %{x}<br>%{text}<extra></extra>"),
            xgap=3,  # Bordures verticales entre les matchs
            ygap=2,  # Bordures horizontales entre les joueurs
        )
    )

    # Ajouter des rectangles semi-transparents en arrière-plan pour les outcomes
    shapes = []
    for y_idx in range(len(gamertags)):
        for x_idx, match_id in enumerate(match_ids):
            outcome = match_outcomes_map.get(match_id, 0)

            # Définir la couleur selon l'outcome
            if outcome == Outcome.WIN:
                fillcolor = OUTCOME_COLORS["win"]
            elif outcome == Outcome.LOSS:
                fillcolor = OUTCOME_COLORS["loss"]
            elif outcome == Outcome.TIE:
                fillcolor = OUTCOME_COLORS["tie"]
            else:
                continue  # Pas de rectangle si outcome inconnu

            # Ajouter un rectangle semi-transparent en arrière-plan
            shapes.append(
                {
                    "type": "rect",
                    "x0": x_idx - 0.5,
                    "x1": x_idx + 0.5,
                    "y0": y_idx - 0.5,
                    "y1": y_idx + 0.5,
                    "fillcolor": fillcolor,
                    "opacity": 0.15,  # Très transparent pour ne pas surcharger
                    "line": {"width": 0},
                    "layer": "below",
                }
            )

    # Ajouter des lignes verticales entre chaque match pour séparer visuellement
    n_rows = len(y_labels)
    for x_idx in range(n_matches - 1):
        shapes.append(
            {
                "type": "line",
                "x0": x_idx + 0.5,
                "x1": x_idx + 0.5,
                "y0": -0.5,
                "y1": n_rows - 0.5,
                "line": {"color": "rgba(255, 255, 255, 0.55)", "width": 1.5},
                "layer": "above",
            }
        )

    fig.update_layout(shapes=shapes)

    # Calculer la hauteur dynamique
    calc_height = height or max(300, 50 * n_rows + 100)

    fig.update_layout(
        height=calc_height,
        margin={"l": 120, "r": 40, "t": 60 if title else 30, "b": 50},
        xaxis_title=viz_t("axis_matches", lang),
        yaxis_title="",
        plot_bgcolor="rgba(245, 245, 245, 0.5)",  # Background clair
    )
    fig.update_yaxes(autorange="reversed")  # Premier en haut

    return apply_halo_plot_style(fig, title=title, height=calc_height)


def build_impact_ranking_df(
    scores: dict[str, int],
    first_blood_counts: dict[str, int] | None = None,
    clutch_counts: dict[str, int] | None = None,
    casualty_counts: dict[str, int] | None = None,
) -> pl.DataFrame:
    """Construit un DataFrame de ranking pour le tableau de taquinerie.

    Args:
        scores: Dict {gamertag: score_total}.
        first_blood_counts: Nombre de FB par joueur (optionnel).
        clutch_counts: Nombre de Clutch par joueur (optionnel).
        casualty_counts: Nombre de Boulet par joueur (optionnel).

    Returns:
        DataFrame avec colonnes : rang, gamertag, score, fb, clutch, boulet, badge.
    """
    if not scores:
        return pl.DataFrame(
            schema={
                "rang": pl.Int64,
                "gamertag": pl.Utf8,
                "score": pl.Int64,
                "fb": pl.Int64,
                "clutch": pl.Int64,
                "boulet": pl.Int64,
                "badge": pl.Utf8,
            }
        )

    first_blood_counts = first_blood_counts or {}
    clutch_counts = clutch_counts or {}
    casualty_counts = casualty_counts or {}

    # Trier par score décroissant
    sorted_players = sorted(scores.items(), key=lambda x: x[1], reverse=True)

    rows = []
    for rank, (gamertag, score) in enumerate(sorted_players, start=1):
        badge = ""
        if rank == 1:
            badge = "🏆 MVP"
        elif rank == len(sorted_players) and score < 0:
            badge = "🍌 Boulet"
        elif rank == len(sorted_players):
            badge = "📉 Dernier"

        rows.append(
            {
                "rang": rank,
                "gamertag": gamertag,
                "score": score,
                "fb": first_blood_counts.get(gamertag, 0),
                "clutch": clutch_counts.get(gamertag, 0),
                "boulet": casualty_counts.get(gamertag, 0),
                "badge": badge,
            }
        )

    return pl.DataFrame(rows)


def count_events_by_player(
    events: dict[str, ImpactEvent],
) -> dict[str, int]:
    """Compte le nombre d'événements par joueur.

    Args:
        events: Dict {match_id: ImpactEvent}.

    Returns:
        Dict {gamertag: count}.
    """
    counts: dict[str, int] = {}
    for event in events.values():
        gamertag = event.gamertag
        counts[gamertag] = counts.get(gamertag, 0) + 1
    return counts


def render_impact_summary_stats(
    first_bloods: dict[str, ImpactEvent],
    clutch_finishers: dict[str, ImpactEvent],
    last_casualties: dict[str, ImpactEvent],
) -> dict[str, int]:
    """Calcule les statistiques résumées des événements d'impact.

    Args:
        first_bloods: Dict des premiers kills.
        clutch_finishers: Dict des finisseurs.
        last_casualties: Dict des boulets.

    Returns:
        Dict avec total_fb, total_clutch, total_casualty, total_matches.
    """
    # Compter les matchs uniques
    all_match_ids = (
        set(first_bloods.keys()) | set(clutch_finishers.keys()) | set(last_casualties.keys())
    )

    return {
        "total_fb": len(first_bloods),
        "total_clutch": len(clutch_finishers),
        "total_casualty": len(last_casualties),
        "total_matches": len(all_match_ids),
    }


# ─── Feature 1 — Heatmap joueur × carte ─────────────────────────────────────


def plot_squad_map_heatmap(
    series: list[tuple[str, DataFrameLike]],
    lang: str = "fr",
) -> go.Figure | None:
    """Heatmap de performance par joueur × carte.

    Chaque cellule = performance_avg du joueur sur cette carte (top 15 cartes par fréquence).

    Args:
        series: Liste de (nom_joueur, df_matchs) pour chaque membre de l'escouade.
        lang: Langue.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    from src.analysis import compute_map_breakdown  # noqa: PLC0415

    if not series:
        return None

    player_bds: list[pl.DataFrame] = []
    for name, df in series:
        df_pl = ensure_polars(df)
        if df_pl.is_empty():
            continue
        bd = compute_map_breakdown(df_pl)
        if not bd.is_empty():
            player_bds.append(bd.with_columns(pl.lit(name).alias("_player")))

    if not player_bds:
        return None

    all_bd = pl.concat(player_bds, how="diagonal_relaxed")
    top_maps = (
        all_bd.group_by("map_name")
        .agg(pl.col("matches").sum())
        .sort("matches", descending=True)
        .head(15)["map_name"]
        .to_list()
    )
    all_bd = all_bd.filter(pl.col("map_name").is_in(top_maps))
    if all_bd.is_empty():
        return None

    player_names = [n for n, _ in series]
    matrix = []
    for name in player_names:
        row_bd = all_bd.filter(pl.col("_player") == name)
        perf_map = dict(
            zip(row_bd["map_name"].to_list(), row_bd["performance_avg"].to_list(), strict=False)
        )
        matrix.append([perf_map.get(m) for m in top_maps])

    c = HALO_COLORS.as_dict()
    colorscale = [
        [0.0, c["red"]],
        [0.4, c["amber"]],
        [0.7, c["cyan"]],
        [1.0, c["green"]],
    ]
    perf_lbl = "Perf"
    height = max(180, len(player_names) * 60 + 100)
    fig = go.Figure(
        go.Heatmap(
            z=matrix,
            x=top_maps,
            y=player_names,
            colorscale=colorscale,
            zmid=0,
            hovertemplate=f"%{{y}} — %{{x}}<br>{perf_lbl}=%{{z:.1f}}<extra></extra>",
            colorbar={"title": perf_lbl, "thickness": 15},
        )
    )
    fig.update_layout(
        height=height,
        margin={"l": 40, "r": 80, "t": 30, "b": 120},
        xaxis={"tickangle": -35},
    )
    return apply_halo_plot_style(fig, height=height)
