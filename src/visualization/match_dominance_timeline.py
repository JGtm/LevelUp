"""Visualisation de la timeline de domination du match.

Ce module crée des frises chronologiques pour visualiser qui domine le match
selon le mode de jeu :
- Slayer : Progression des kills
- CTF : Progression des captures
- Strongholds : Contrôle des bases
- Oddball : Contrôle de la balle

La visualisation peut inclure un overlay des kills/deaths.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Literal

import polars as pl
import plotly.graph_objects as go
from plotly.subplots import make_subplots

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository


# =============================================================================
# Timeline Slayer : Progression des Kills
# =============================================================================


def create_slayer_dominance_timeline(
    repo: DuckDBRepository,
    match_id: str,
    show_kills_deaths: bool = True,
) -> go.Figure:
    """Crée une timeline de domination pour le mode Slayer.
    
    Visualise la progression des kills par équipe au fil du temps.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        show_kills_deaths: Si True, affiche les kills/deaths en overlay.
        
    Returns:
        Figure Plotly avec timeline de domination.
    """
    # Charger les highlight_events avec timestamps
    query = """
    SELECT 
        time_ms,
        killer_xuid,
        victim_xuid,
        event_type
    FROM highlight_events
    WHERE match_id = ?
      AND event_type = 'kill'
    ORDER BY time_ms ASC
    """
    
    events = repo.query_df(query, [match_id])
    
    if events.is_empty():
        return _create_empty_figure("Aucun événement de combat disponible")
    
    # Récupérer les team_ids des joueurs
    team_mapping = _get_player_team_mapping(repo, match_id)
    
    # Ajouter les team_ids aux événements
    events = events.with_columns([
        pl.col("killer_xuid").map_dict(team_mapping, default=None).alias("killer_team"),
        pl.col("victim_xuid").map_dict(team_mapping, default=None).alias("victim_team"),
    ])
    
    # Compter les kills cumulés par équipe
    team_0_kills = []
    team_1_kills = []
    timestamps = []
    
    team_0_count = 0
    team_1_count = 0
    
    for row in events.iter_rows(named=True):
        killer_team = row["killer_team"]
        time_ms = row["time_ms"]
        
        if killer_team == 0:
            team_0_count += 1
        elif killer_team == 1:
            team_1_count += 1
        
        timestamps.append(time_ms)
        team_0_kills.append(team_0_count)
        team_1_kills.append(team_1_count)
    
    # Créer le graphique
    fig = go.Figure()
    
    # Convertir timestamps en secondes pour affichage
    timestamps_s = [t / 1000 for t in timestamps]
    
    # Ligne Team 0
    fig.add_trace(go.Scatter(
        x=timestamps_s,
        y=team_0_kills,
        mode='lines',
        name='Équipe 0',
        fill='tozeroy',
        line=dict(color='#0A84FF', width=2),
        fillcolor='rgba(10, 132, 255, 0.3)',
        hovertemplate='%{y} kills<br>%{x:.0f}s<extra></extra>',
    ))
    
    # Ligne Team 1
    fig.add_trace(go.Scatter(
        x=timestamps_s,
        y=team_1_kills,
        mode='lines',
        name='Équipe 1',
        fill='tozeroy',
        line=dict(color='#FF453A', width=2),
        fillcolor='rgba(255, 69, 58, 0.3)',
        hovertemplate='%{y} kills<br>%{x:.0f}s<extra></extra>',
    ))
    
    # Layout
    fig.update_layout(
        title="Timeline de Domination - Slayer",
        xaxis_title="Temps (secondes)",
        yaxis_title="Kills Cumulés",
        hovermode='x unified',
        template='plotly_dark',
        height=400,
        legend=dict(
            orientation="h",
            yanchor="bottom",
            y=1.02,
            xanchor="right",
            x=1
        ),
    )
    
    return fig


# =============================================================================
# Timeline CTF : Progression des Captures
# =============================================================================


def create_ctf_dominance_timeline(
    repo: DuckDBRepository,
    match_id: str,
) -> go.Figure:
    """Crée une timeline de domination pour le mode CTF.
    
    Visualise la progression des captures de drapeaux par équipe.
    Utilise la timeline approximative basée sur les kills/deaths.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        
    Returns:
        Figure Plotly avec timeline de domination.
    """
    from src.data.services.objective_timeline_service import (
        estimate_objective_captures_timeline,
    )
    
    # Estimer la timeline des captures
    timeline = estimate_objective_captures_timeline(repo, match_id, mode="CTF")
    
    if timeline.is_empty():
        return _create_empty_figure("Aucune capture de drapeau détectée")
    
    # Préparer les données pour le graphique
    team_0_data = timeline.filter(pl.col("team_id") == 0).sort("estimated_time_ms")
    team_1_data = timeline.filter(pl.col("team_id") == 1).sort("estimated_time_ms")
    
    # Calculer les captures cumulées
    team_0_cumulative = list(range(1, team_0_data.height + 1))
    team_1_cumulative = list(range(1, team_1_data.height + 1))
    
    # Créer le graphique
    fig = go.Figure()
    
    # Team 0
    if not team_0_data.is_empty():
        fig.add_trace(go.Scatter(
            x=[t / 1000 for t in team_0_data["estimated_time_ms"].to_list()],
            y=team_0_cumulative,
            mode='lines+markers',
            name='Équipe 0',
            line=dict(color='#0A84FF', width=3),
            marker=dict(size=10, symbol='star'),
            fill='tozeroy',
            fillcolor='rgba(10, 132, 255, 0.2)',
            hovertemplate='Capture #%{y}<br>%{x:.0f}s<extra></extra>',
        ))
    
    # Team 1
    if not team_1_data.is_empty():
        fig.add_trace(go.Scatter(
            x=[t / 1000 for t in team_1_data["estimated_time_ms"].to_list()],
            y=team_1_cumulative,
            mode='lines+markers',
            name='Équipe 1',
            line=dict(color='#FF453A', width=3),
            marker=dict(size=10, symbol='star'),
            fill='tozeroy',
            fillcolor='rgba(255, 69, 58, 0.2)',
            hovertemplate='Capture #%{y}<br>%{x:.0f}s<extra></extra>',
        ))
    
    # Layout
    fig.update_layout(
        title="Timeline de Domination - Capture The Flag",
        xaxis_title="Temps (secondes)",
        yaxis_title="Captures Cumulées",
        hovermode='x unified',
        template='plotly_dark',
        height=400,
        legend=dict(
            orientation="h",
            yanchor="bottom",
            y=1.02,
            xanchor="right",
            x=1
        ),
    )
    
    return fig


# =============================================================================
# Timeline Strongholds : Contrôle des Bases
# =============================================================================


def create_strongholds_dominance_timeline(
    repo: DuckDBRepository,
    match_id: str,
) -> go.Figure:
    """Crée une timeline de domination pour le mode Strongholds.
    
    Visualise le contrôle estimé des bases par équipe au fil du temps.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        
    Returns:
        Figure Plotly avec timeline de domination.
    """
    from src.data.services.objective_timeline_service import (
        estimate_objective_captures_timeline,
        estimate_team_base_control,
    )
    
    # Estimer la timeline des captures
    timeline = estimate_objective_captures_timeline(repo, match_id, mode="Strongholds")
    
    if timeline.is_empty():
        return _create_empty_figure("Aucune capture de base détectée")
    
    # Calculer le contrôle par équipe
    control = estimate_team_base_control(timeline, mode="Strongholds")
    
    # Créer le graphique avec les captures au fil du temps
    team_0_data = timeline.filter(pl.col("team_id") == 0).sort("estimated_time_ms")
    team_1_data = timeline.filter(pl.col("team_id") == 1).sort("estimated_time_ms")
    
    fig = go.Figure()
    
    # Team 0
    if not team_0_data.is_empty():
        # Compter les bases progressivement (estimation)
        team_0_bases_over_time = []
        for i in range(team_0_data.height):
            # Estimation simple : min(captures, 3 bases max)
            team_0_bases_over_time.append(min(i + 1, 3))
        
        fig.add_trace(go.Scatter(
            x=[t / 1000 for t in team_0_data["estimated_time_ms"].to_list()],
            y=team_0_bases_over_time,
            mode='lines+markers',
            name='Équipe 0',
            line=dict(color='#0A84FF', width=3, shape='hv'),
            marker=dict(size=8, symbol='square'),
            fill='tozeroy',
            fillcolor='rgba(10, 132, 255, 0.3)',
            hovertemplate='~%{y} bases<br>%{x:.0f}s<extra></extra>',
        ))
    
    # Team 1
    if not team_1_data.is_empty():
        team_1_bases_over_time = []
        for i in range(team_1_data.height):
            team_1_bases_over_time.append(min(i + 1, 3))
        
        fig.add_trace(go.Scatter(
            x=[t / 1000 for t in team_1_data["estimated_time_ms"].to_list()],
            y=team_1_bases_over_time,
            mode='lines+markers',
            name='Équipe 1',
            line=dict(color='#FF453A', width=3, shape='hv'),
            marker=dict(size=8, symbol='square'),
            fill='tozeroy',
            fillcolor='rgba(255, 69, 58, 0.3)',
            hovertemplate='~%{y} bases<br>%{x:.0f}s<extra></extra>',
        ))
    
    # Ajouter annotation du contrôle final
    if not control.is_empty():
        team_0_control = control.filter(pl.col("team_id") == 0)
        team_1_control = control.filter(pl.col("team_id") == 1)
        
        team_0_bases = team_0_control["estimated_unique_bases"][0] if not team_0_control.is_empty() else 0
        team_1_bases = team_1_control["estimated_unique_bases"][0] if not team_1_control.is_empty() else 0
        
        annotation_text = f"Contrôle final estimé : Équipe 0 ({team_0_bases} bases) - Équipe 1 ({team_1_bases} bases)"
        
        fig.add_annotation(
            text=annotation_text,
            xref="paper", yref="paper",
            x=0.5, y=-0.15,
            showarrow=False,
            font=dict(size=10, color="gray"),
            xanchor="center",
        )
    
    # Layout
    fig.update_layout(
        title="Timeline de Domination - Strongholds",
        xaxis_title="Temps (secondes)",
        yaxis_title="Bases Contrôlées (estimation)",
        yaxis=dict(range=[0, 3.5], dtick=1),
        hovermode='x unified',
        template='plotly_dark',
        height=400,
        legend=dict(
            orientation="h",
            yanchor="bottom",
            y=1.02,
            xanchor="right",
            x=1
        ),
    )
    
    return fig


# =============================================================================
# Timeline avec Kills/Deaths Overlay
# =============================================================================


def create_dominance_timeline_with_overlay(
    repo: DuckDBRepository,
    match_id: str,
    mode: Literal["Slayer", "CTF", "Strongholds", "Oddball"] = "Slayer",
) -> go.Figure:
    """Crée une timeline de domination avec overlay kills/deaths.
    
    Affiche la timeline principale selon le mode, avec un graphique secondaire
    montrant les kills en bâtons au-dessus et les morts en dessous.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        mode: Mode de jeu.
        
    Returns:
        Figure Plotly avec 2 sous-graphiques (domination + kills/deaths).
    """
    # Créer les sous-graphiques
    fig = make_subplots(
        rows=2, cols=1,
        row_heights=[0.6, 0.4],
        subplot_titles=("Domination du Match", "Kills & Deaths"),
        vertical_spacing=0.12,
    )
    
    # Graphique principal selon le mode
    if mode == "Slayer":
        main_fig = create_slayer_dominance_timeline(repo, match_id, show_kills_deaths=False)
    elif mode == "CTF":
        main_fig = create_ctf_dominance_timeline(repo, match_id)
    elif mode == "Strongholds":
        main_fig = create_strongholds_dominance_timeline(repo, match_id)
    else:
        main_fig = create_slayer_dominance_timeline(repo, match_id, show_kills_deaths=False)
    
    # Copier les traces du graphique principal
    for trace in main_fig.data:
        fig.add_trace(trace, row=1, col=1)
    
    # Charger les kills/deaths pour l'overlay
    query = """
    SELECT 
        time_ms,
        killer_xuid,
        victim_xuid
    FROM highlight_events
    WHERE match_id = ?
      AND event_type = 'kill'
    ORDER BY time_ms ASC
    """
    
    events = repo.query_df(query, [match_id])
    
    if not events.is_empty():
        team_mapping = _get_player_team_mapping(repo, match_id)
        
        events = events.with_columns([
            pl.col("killer_xuid").map_dict(team_mapping, default=None).alias("killer_team"),
            pl.col("victim_xuid").map_dict(team_mapping, default=None).alias("victim_team"),
        ])
        
        # Regrouper par tranches de 30 secondes
        window_size = 30000  # 30 secondes
        
        events_with_bin = events.with_columns([
            (pl.col("time_ms") // window_size).alias("time_bin")
        ])
        
        # Compter kills et deaths par équipe et par bin
        team_0_kills_by_bin = (
            events_with_bin
            .filter(pl.col("killer_team") == 0)
            .group_by("time_bin")
            .agg(pl.count().alias("kills"))
        )
        
        team_1_kills_by_bin = (
            events_with_bin
            .filter(pl.col("killer_team") == 1)
            .group_by("time_bin")
            .agg(pl.count().alias("kills"))
        )
        
        team_0_deaths_by_bin = (
            events_with_bin
            .filter(pl.col("victim_team") == 0)
            .group_by("time_bin")
            .agg(pl.count().alias("deaths"))
        )
        
        team_1_deaths_by_bin = (
            events_with_bin
            .filter(pl.col("victim_team") == 1)
            .group_by("time_bin")
            .agg(pl.count().alias("deaths"))
        )
        
        # Barres Team 0 (kills en haut, deaths en bas)
        if not team_0_kills_by_bin.is_empty():
            fig.add_trace(go.Bar(
                x=[(b * window_size + window_size / 2) / 1000 for b in team_0_kills_by_bin["time_bin"].to_list()],
                y=team_0_kills_by_bin["kills"].to_list(),
                name='Équipe 0 Kills',
                marker_color='#0A84FF',
                showlegend=False,
                hovertemplate='%{y} kills<extra></extra>',
            ), row=2, col=1)
        
        if not team_0_deaths_by_bin.is_empty():
            fig.add_trace(go.Bar(
                x=[(b * window_size + window_size / 2) / 1000 for b in team_0_deaths_by_bin["time_bin"].to_list()],
                y=[-d for d in team_0_deaths_by_bin["deaths"].to_list()],  # Négatif pour afficher en bas
                name='Équipe 0 Deaths',
                marker_color='rgba(10, 132, 255, 0.5)',
                showlegend=False,
                hovertemplate='%{y} deaths<extra></extra>',
            ), row=2, col=1)
        
        # Barres Team 1
        if not team_1_kills_by_bin.is_empty():
            fig.add_trace(go.Bar(
                x=[(b * window_size + window_size / 2) / 1000 for b in team_1_kills_by_bin["time_bin"].to_list()],
                y=team_1_kills_by_bin["kills"].to_list(),
                name='Équipe 1 Kills',
                marker_color='#FF453A',
                showlegend=False,
                hovertemplate='%{y} kills<extra></extra>',
            ), row=2, col=1)
        
        if not team_1_deaths_by_bin.is_empty():
            fig.add_trace(go.Bar(
                x=[(b * window_size + window_size / 2) / 1000 for b in team_1_deaths_by_bin["time_bin"].to_list()],
                y=[-d for d in team_1_deaths_by_bin["deaths"].to_list()],
                name='Équipe 1 Deaths',
                marker_color='rgba(255, 69, 58, 0.5)',
                showlegend=False,
                hovertemplate='%{y} deaths<extra></extra>',
            ), row=2, col=1)
    
    # Layout
    fig.update_xaxes(title_text="Temps (secondes)", row=2, col=1)
    fig.update_yaxes(title_text="Progression", row=1, col=1)
    fig.update_yaxes(title_text="Kills (+) / Deaths (-)", row=2, col=1)
    
    fig.update_layout(
        template='plotly_dark',
        height=700,
        hovermode='x unified',
        barmode='relative',
        showlegend=True,
        legend=dict(
            orientation="h",
            yanchor="bottom",
            y=1.05,
            xanchor="right",
            x=1
        ),
    )
    
    return fig


# =============================================================================
# Helpers
# =============================================================================


def _get_player_team_mapping(repo: DuckDBRepository, match_id: str) -> dict[str, int]:
    """Récupère le mapping xuid → team_id pour un match.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        
    Returns:
        Dictionnaire {xuid: team_id}.
    """
    query = """
    SELECT xuid, team_id
    FROM match_participants
    WHERE match_id = ?
    """
    
    result = repo.query_df(query, [match_id])
    
    if result.is_empty():
        return {}
    
    return dict(zip(result["xuid"].to_list(), result["team_id"].to_list()))


def _create_empty_figure(message: str) -> go.Figure:
    """Crée une figure vide avec un message.
    
    Args:
        message: Message à afficher.
        
    Returns:
        Figure Plotly vide.
    """
    fig = go.Figure()
    
    fig.add_annotation(
        text=message,
        xref="paper", yref="paper",
        x=0.5, y=0.5,
        showarrow=False,
        font=dict(size=16, color="gray"),
        xanchor="center", yanchor="middle",
    )
    
    fig.update_layout(
        template='plotly_dark',
        height=400,
        xaxis=dict(visible=False),
        yaxis=dict(visible=False),
    )
    
    return fig


# =============================================================================
# Fonction Principale Adaptative
# =============================================================================


def create_match_dominance_timeline(
    repo: DuckDBRepository,
    match_id: str,
    game_mode: str | None = None,
    show_kills_overlay: bool = True,
) -> go.Figure:
    """Crée une timeline de domination adaptée au mode de jeu.
    
    Détecte automatiquement le mode de jeu et génère la visualisation appropriée.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        game_mode: Mode de jeu (optionnel, auto-détecté si None).
        show_kills_overlay: Si True, affiche l'overlay kills/deaths.
        
    Returns:
        Figure Plotly avec timeline de domination.
    """
    # Auto-détecter le mode de jeu si non fourni
    if game_mode is None:
        game_mode = _detect_game_mode(repo, match_id)
    
    # Normaliser le mode
    mode_normalized = game_mode.lower() if game_mode else "slayer"
    
    # Sélectionner la visualisation appropriée
    if "ctf" in mode_normalized or "flag" in mode_normalized:
        if show_kills_overlay:
            return create_dominance_timeline_with_overlay(repo, match_id, mode="CTF")
        else:
            return create_ctf_dominance_timeline(repo, match_id)
    
    elif "stronghold" in mode_normalized or "base" in mode_normalized or "zone" in mode_normalized:
        if show_kills_overlay:
            return create_dominance_timeline_with_overlay(repo, match_id, mode="Strongholds")
        else:
            return create_strongholds_dominance_timeline(repo, match_id)
    
    elif "oddball" in mode_normalized:
        # Oddball pas encore implémenté, fallback sur Slayer
        if show_kills_overlay:
            return create_dominance_timeline_with_overlay(repo, match_id, mode="Slayer")
        else:
            return create_slayer_dominance_timeline(repo, match_id)
    
    else:
        # Slayer ou mode inconnu
        if show_kills_overlay:
            return create_dominance_timeline_with_overlay(repo, match_id, mode="Slayer")
        else:
            return create_slayer_dominance_timeline(repo, match_id)


def _detect_game_mode(repo: DuckDBRepository, match_id: str) -> str:
    """Détecte le mode de jeu d'un match.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match.
        
    Returns:
        Mode de jeu (ex: "Slayer", "CTF", "Strongholds").
    """
    query = """
    SELECT game_variant_category
    FROM match_registry
    WHERE match_id = ?
    """
    
    result = repo.query_df(query, [match_id])
    
    if result.is_empty():
        return "Slayer"
    
    return result["game_variant_category"][0] or "Slayer"
