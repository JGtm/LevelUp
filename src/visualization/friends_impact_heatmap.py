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
from src.visualization.theme import apply_halo_plot_style

if TYPE_CHECKING:
    pass

# Couleurs pour les événements d'impact
IMPACT_COLORS = {
    "first_blood": "#2ecc71",  # Vert
    "clutch_finisher": "#f39c12",  # Or/Orange
    "last_casualty": "#e74c3c",  # Rouge
    "none": "rgba(100, 100, 100, 0.1)",  # Gris transparent
}

# Couleurs pour les outcomes (Win/Loss/Tie)
OUTCOME_COLORS = {
    "win": "#10b981",  # Vert (victories)
    "loss": "#ef4444",  # Rouge (defeats)
    "tie": "#8b5cf6",  # Violet (ties)
    "unknown": "rgba(100, 100, 100, 0.3)",  # Gris
}

# Labels d'événements (symboles simplifiés sans emoji redondant)
EVENT_LABELS = {
    "first_blood": "⚡",  # Premier sang
    "clutch_finisher": "🎯",  # Finisseur
    "last_casualty": "💀",  # Boulet
    "last_group_kill": "🐌",  # Dernier à tuer (plus lent)
    "first_group_death": "🎯🔻",  # Première victime
}


def plot_friends_impact_heatmap(
    impact_matrix: pl.DataFrame,
    *,
    title: str | None = None,
    max_matches: int = 50,
    height: int | None = None,
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
            text="Aucun événement d'impact à afficher",
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

    # Séparer la ligne "Résultat" des joueurs
    has_outcome_row = "Résultat" in all_gamertags
    gamertags = [g for g in all_gamertags if g != "Résultat"]

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
    # Structure : ["Résultat", Joueur1, Joueur2, ...] × match_ids
    z_matrix = []
    text_matrix = []
    y_labels = []

    # Ligne 1 : Résultat (outcomes)
    if has_outcome_row:
        y_labels.append("Résultat")
        z_row = []
        text_row = []

        outcome_events = impact_matrix.filter(pl.col("gamertag") == "Résultat")
        for match_id in match_ids:
            match_outcome = outcome_events.filter(pl.col("match_id") == match_id)

            if not match_outcome.is_empty() and match_outcome["outcome"][0] is not None:
                outcome = match_outcome["outcome"][0]
                # Mapper outcome (2=Win, 3=Loss, 1=Tie) vers une valeur pour colorscale
                # Win=10, Loss=-10, Tie=5 pour avoir une échelle séparée
                if outcome == 2:  # Win
                    z_row.append(10)
                    text_row.append("")
                elif outcome == 3:  # Loss
                    z_row.append(-10)
                    text_row.append("")
                elif outcome == 1:  # Tie
                    z_row.append(5)
                    text_row.append("")
                else:
                    z_row.append(0)
                    text_row.append("")
            else:
                z_row.append(0)
                text_row.append("")

        z_matrix.append(z_row)
        text_matrix.append(text_row)

    # Lignes joueurs
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

    # Créer la colorscale custom
    # Échelle pour outcomes : -10 (Loss), 0, 5 (Tie), 10 (Win)
    # Échelle pour joueurs : -1 (Boulet), 0 (rien), 1 (FB), 2 (Clutch)
    # On va normaliser sur [-10, 10] pour tout couvrir
    colorscale = [
        [0.0, OUTCOME_COLORS["loss"]],  # -10 = Loss (rouge)
        [0.25, IMPACT_COLORS["none"]],  # -1 à 0 = Gris transparent
        [0.5, IMPACT_COLORS["none"]],  # 0 = Gris transparent
        [0.55, IMPACT_COLORS["first_blood"]],  # 1 = FB (vert)
        [0.6, IMPACT_COLORS["clutch_finisher"]],  # 2 = Clutch (orange)
        [0.75, OUTCOME_COLORS["tie"]],  # 5 = Tie (violet)
        [1.0, OUTCOME_COLORS["win"]],  # 10 = Win (vert)
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
            textfont={"size": 25},  # Emojis 2.5x plus grands (10 → 25)
            colorscale=colorscale,
            zmin=-10,
            zmax=10,
            showscale=False,
            hovertemplate=("<b>%{y}</b><br>" "Match %{x}<br>" "%{text}<extra></extra>"),
        )
    )

    # Calculer la hauteur dynamique
    n_rows = len(y_labels)
    calc_height = height or max(300, 50 * n_rows + 100)

    fig.update_layout(
        height=calc_height,
        margin={"l": 120, "r": 40, "t": 60 if title else 30, "b": 50},
        xaxis_title="Matchs récents →",
        yaxis_title="",
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
