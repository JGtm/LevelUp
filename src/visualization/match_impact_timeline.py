"""Timeline des kills/deaths d'un match avec annotations d'impact.

Affiche un graphe chronologique montrant l'accumulation des kills et deaths
du joueur principal au cours du match, avec des marqueurs annotés pour :
- ⚡ Premier sang (premier kill du match, tous joueurs confondus)
- 🎯 Finisseur (dernier kill d'une victoire)
- 🐌 Plus lent (le joueur le plus lent à obtenir son 1er kill)
- 🪦 Première victime (première mort du match, tous joueurs confondus)
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go

from src.ui.formatting import format_time_ms as _format_time
from src.ui.i18n.viz import viz_t
from src.visualization._match_impact_events import (
    MatchImpactEvent,
    compute_single_match_impact,
    get_impact_labels,
)
from src.visualization.theme import apply_halo_plot_style, get_default_layout_kwargs

# =============================================================================
# Construction du graphe Timeline
# =============================================================================


def plot_match_kill_death_timeline(  # noqa: C901, PLR0912, PLR0915
    highlight_events: list[dict[str, Any]],
    me_xuid: str,
    impact_events: list[MatchImpactEvent],
    *,
    height: int = 340,
    lang: str = "fr",
) -> go.Figure | None:
    """Crée un graphe timeline des kills/deaths cumulés avec annotations d'impact.

    Affiche deux courbes (kills et deaths cumulées) du joueur principal,
    avec des marqueurs annotés pour les événements d'impact.

    Args:
        highlight_events: Liste de dicts {event_type, time_ms, xuid, gamertag, ...}.
        me_xuid: XUID du joueur principal.
        impact_events: Liste des événements d'impact à annoter.
        height: Hauteur du graphe en pixels.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if not highlight_events:
        return None

    me_xuid = str(me_xuid).strip()

    # Filtrer les événements du joueur principal
    my_kills = sorted(
        [
            e
            for e in highlight_events
            if str(e.get("event_type", "")).lower() == "kill"
            and str(e.get("xuid", "")).strip() == me_xuid
            and e.get("time_ms") is not None
        ],
        key=lambda e: int(e["time_ms"]),
    )
    my_deaths = sorted(
        [
            e
            for e in highlight_events
            if str(e.get("event_type", "")).lower() == "death"
            and str(e.get("xuid", "")).strip() == me_xuid
            and e.get("time_ms") is not None
        ],
        key=lambda e: int(e["time_ms"]),
    )

    if not my_kills and not my_deaths:
        return None

    # Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
    # Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
    #   kill_color:        #3DFFB5 (vert néon Halo) → #0072B2 (bleu Okabe-Ito)
    #   death_color:       #FF4D6D (rose-rouge)      → #D55E00 (vermillon Okabe-Ito)
    #   annotation_color:  #FFB703 (ambre chaud)     → #E69F00 (orange Okabe-Ito)
    kill_color = "#0072B2"  # Bleu Okabe-Ito — kills
    death_color = "#D55E00"  # Vermillon Okabe-Ito — morts
    annotation_color = "#E69F00"  # Orange Okabe-Ito — annotations

    fig = go.Figure()

    # --- Courbe kills cumulées ---
    if my_kills:
        kill_times = [int(e["time_ms"]) for e in my_kills]
        kill_cum = list(range(1, len(kill_times) + 1))
        kill_labels = [_format_time(t) for t in kill_times]

        fig.add_trace(
            go.Scatter(
                x=kill_times,
                y=kill_cum,
                mode="lines+markers",
                name=viz_t("trace_kills", lang),
                line={"color": kill_color, "width": 2.5},
                marker={"size": 5, "color": kill_color},
                hovertemplate=viz_t("hover_kill_cumul", lang),
                text=kill_labels,
            )
        )

    # --- Courbe deaths cumulées ---
    if my_deaths:
        death_times = [int(e["time_ms"]) for e in my_deaths]
        death_cum = list(range(1, len(death_times) + 1))
        death_labels = [_format_time(t) for t in death_times]

        fig.add_trace(
            go.Scatter(
                x=death_times,
                y=death_cum,
                mode="lines+markers",
                name=viz_t("trace_deaths", lang),
                line={"color": death_color, "width": 2.5, "dash": "dot"},
                marker={"size": 5, "color": death_color},
                hovertemplate=viz_t("hover_death_cumul", lang),
                text=death_labels,
            )
        )

    # --- Annotations d'impact ---
    # Préparer les données d'annotations, triées par temps pour éviter superposition
    sorted_impacts = sorted(impact_events, key=lambda e: e.time_ms)

    # Calcul des décalages verticaux pour éviter superposition
    # Seuil de proximité temporelle (30 secondes)
    PROXIMITY_THRESHOLD_MS = 30_000
    ay_levels = [-40, -90, -140]  # Niveaux de décalage vertical

    annotation_data: list[tuple[MatchImpactEvent, int]] = []
    for ie in sorted_impacts:
        # Trouver le niveau de décalage approprié
        ay_level_idx = 0
        for prev_ie, prev_ay_idx in annotation_data:
            # Si proche du précédent, monter d'un niveau
            if abs(ie.time_ms - prev_ie.time_ms) < PROXIMITY_THRESHOLD_MS:
                ay_level_idx = max(ay_level_idx, prev_ay_idx + 1)
        # Boucler si on dépasse les niveaux disponibles
        ay_level_idx = ay_level_idx % len(ay_levels)
        annotation_data.append((ie, ay_level_idx))

    impact_labels = get_impact_labels(lang)
    annotations = []
    for ie, ay_level_idx in annotation_data:
        label_info = impact_labels.get(ie.event_type)
        if not label_info:
            continue

        icon, label_fr = label_info
        display_name = ie.gamertag
        annotation_text = f"{icon} {label_fr}"

        # Trouver la position Y et X d'ancrage sur la courbe concernée.
        # On clamp arrow_x dans [premier_event, dernier_event] de la courbe pour
        # garantir que la flèche pointe toujours sur un segment existant :
        #   - trop tôt (ex: first_group_death avant ma 1ère mort) → clamp au début
        #   - trop tard (ex: last_casualty après ma dernière mort) → clamp à la fin
        # y_val est recalculé sur la position clampée.
        t_ms = ie.time_ms
        event_base = ie.event_type

        if event_base in ("first_blood", "clutch_finisher", "last_group_kill"):
            if my_kills:
                first_t = int(my_kills[0]["time_ms"])
                last_t = int(my_kills[-1]["time_ms"])
                arrow_x = max(first_t, min(t_ms, last_t))
                y_val = sum(1 for e in my_kills if int(e["time_ms"]) <= arrow_x)
            else:
                arrow_x = t_ms
                y_val = 0.5
        else:
            if my_deaths:
                first_t = int(my_deaths[0]["time_ms"])
                last_t = int(my_deaths[-1]["time_ms"])
                arrow_x = max(first_t, min(t_ms, last_t))
                y_val = sum(1 for e in my_deaths if int(e["time_ms"]) <= arrow_x)
            else:
                arrow_x = t_ms
                y_val = 0.5

        # Couleur selon si c'est moi ou un autre
        ann_bg = "rgba(61, 255, 181, 0.15)" if ie.is_me else "rgba(255, 183, 3, 0.15)"
        ann_border = kill_color if ie.is_me else annotation_color

        # Décalage vertical selon le niveau attribué
        ay_offset = ay_levels[ay_level_idx]

        annotations.append(
            {
                "x": arrow_x,
                "y": y_val,
                "text": f"<b>{annotation_text}</b><br><span style='font-size:11px'>{display_name}</span>",
                "showarrow": True,
                "arrowhead": 2,
                "arrowsize": 1,
                "arrowwidth": 1.5,
                "arrowcolor": ann_border,
                "ax": 0,
                "ay": ay_offset,
                "bordercolor": ann_border,
                "borderwidth": 1,
                "borderpad": 4,
                "bgcolor": ann_bg,
                "font": {"size": 12, "color": "#F5F8FF"},
                "align": "center",
            }
        )

    # --- Layout ---
    layout_kwargs = get_default_layout_kwargs(height=height)

    fig.update_layout(
        **layout_kwargs,
        xaxis={
            "title": viz_t("axis_match_time", lang),
            "tickmode": "auto",
            "nticks": 10,
            "tickformat": "d",
            "ticksuffix": "",
        },
        yaxis={
            "title": viz_t("label_cumul", lang),
            "dtick": 1,
        },
        legend={
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "left",
            "x": 0,
        },
        annotations=annotations,
        hovermode="x unified",
    )

    # Formater les ticks X en M:SS
    all_times = []
    if my_kills:
        all_times.extend(int(e["time_ms"]) for e in my_kills)
    if my_deaths:
        all_times.extend(int(e["time_ms"]) for e in my_deaths)

    if all_times:
        min(all_times)
        max_t = max(all_times)
        # Créer des ticks réguliers toutes les 60 secondes
        tick_interval = 60_000  # 1 minute
        tick_vals = list(range(0, max_t + tick_interval, tick_interval))
        tick_texts = [_format_time(t) for t in tick_vals]
        fig.update_xaxes(
            tickmode="array",
            tickvals=tick_vals,
            ticktext=tick_texts,
        )

    apply_halo_plot_style(fig)

    return fig


# =============================================================================
# K/D cumulé de tous les joueurs (timeline match)
# =============================================================================


def plot_all_players_frags_timeline(  # noqa: C901, PLR0912, PLR0913, PLR0915
    highlight_events: list[dict[str, Any]],
    me_xuid: str,
    *,
    gt_map: dict[str, str] | None = None,
    player_color_map: dict[str, str] | None = None,
    height: int = 380,
    lang: str = "fr",
) -> go.Figure | None:
    """Graphe chronologique du score K/D différentiel cumulé de tous les joueurs.

    Chaque kill ajoute +1, chaque mort retranche -1. Les courbes peuvent
    descendre en dessous de zéro, révélant les phases de domination et de
    remontée au fil du match.

    Args:
        highlight_events: Events {event_type, time_ms, xuid, gamertag, ...}.
        me_xuid: XUID du joueur principal (ligne mise en avant).
        gt_map: Mapping xuid → gamertag résolu.
        player_color_map: Mapping xuid → couleur hex.  Si fourni, les
            couleurs seront cohérentes avec d'autres graphes du même match.
        height: Hauteur du graphe en pixels.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if not highlight_events:
        return None

    from src.config import get_bot_name
    from src.visualization.antagonist_charts import PLAYER_COLORS

    me_xuid = str(me_xuid).strip()

    # Collecter uniquement les events kill/death avec time_ms
    events: list[tuple[int, str, str, str | None]] = []  # (time_ms, event_type, xuid, gamertag)
    for e in highlight_events:
        etype = str(e.get("event_type", "")).lower()
        if etype not in ("kill", "death"):
            continue
        t = e.get("time_ms")
        if t is None:
            continue
        xu = str(e.get("xuid", "")).strip()
        if not xu:
            continue
        events.append((int(t), etype, xu, e.get("gamertag")))

    if not events:
        return None

    events.sort(key=lambda x: x[0])

    # Accumuler le différentiel K/D (+1 kill, -1 mort) par joueur au fil du temps
    player_kd: dict[str, int] = {}
    # Stocker (time_ms, kd_cumulé) par joueur
    player_series: dict[str, list[tuple[int, int]]] = {}
    player_gt: dict[str, str] = {}  # xuid → meilleur gamertag connu

    for t_ms, etype, xu, gt_raw in events:
        player_kd.setdefault(xu, 0)
        player_series.setdefault(xu, [])

        if etype == "kill":
            player_kd[xu] += 1
        elif etype == "death":
            player_kd[xu] -= 1
        else:
            continue

        player_series[xu].append((t_ms, player_kd[xu]))

        # Garder le gamertag le plus récent
        if gt_raw and str(gt_raw).strip():
            player_gt[xu] = str(gt_raw).strip()

    if not player_series:
        return None

    # Résolution des noms d'affichage
    def _resolve_gt(xu: str) -> str:
        # gt_map (source fraîche)
        if gt_map and xu in gt_map:
            g = str(gt_map[xu]).strip()
            if g and g != "?" and not g.isdigit():
                return g
        # Bots
        if xu.lower().startswith("bid("):
            bot = get_bot_name(xu)
            if isinstance(bot, str) and bot.strip():
                return bot.strip()
        # Gamertag de l'event
        if xu in player_gt:
            g = player_gt[xu]
            if g and g != "?" and not g.isdigit() and not g.lower().startswith("xuid("):
                return g
        return xu[:8] + "…"

    # Trier les joueurs pour l'assignation des couleurs :
    # joueur principal en premier, puis par nombre total d'events décroissant
    sorted_xuids = sorted(
        player_series.keys(),
        key=lambda xu: (xu != me_xuid, -len(player_series[xu])),
    )

    # Construire le color_map si non fourni
    if player_color_map is None:
        player_color_map = {}
        for idx, xu in enumerate(sorted_xuids):
            player_color_map[xu] = PLAYER_COLORS[idx % len(PLAYER_COLORS)]

    fig = go.Figure()

    for xu in sorted_xuids:
        series = player_series[xu]
        if not series:
            continue

        # Insérer (0, 0) pour que toutes les courbes partent du même axe
        times = [0] + [s[0] for s in series]
        kds = [0] + [s[1] for s in series]
        labels = ["00:00"] + [_format_time(t) for t in times[1:]]
        name = _resolve_gt(xu)
        color = player_color_map.get(xu, "#999999")
        is_me = xu == me_xuid

        fig.add_trace(
            go.Scatter(
                x=times,
                y=kds,
                mode="lines+markers",
                name=name,
                line={
                    "color": color,
                    "width": 4.5 if is_me else 2.8,
                },
                marker={
                    "size": 7 if is_me else 4,
                    "color": color,
                },
                opacity=1.0 if is_me else 0.65,
                hovertemplate=(
                    f"<b>{name}</b><br>" f"Score: %{{y:+.0f}}<br>" "%{text}<extra></extra>"
                ),
                text=labels,
            )
        )

    # Ligne de référence à y=0 (équilibre kills/deaths)
    fig.add_hline(
        y=0,
        line_dash="dot",
        line_color="rgba(200, 200, 200, 0.5)",
        line_width=1,
    )

    fig.update_layout(
        **get_default_layout_kwargs(height),
        xaxis_title="",
        yaxis_title=viz_t("axis_kd_score", lang),
        showlegend=True,
        legend={
            "orientation": "h",
            "yanchor": "top",
            "y": -0.15,
            "xanchor": "left",
            "x": 0,
            "font": {"size": 10},
        },
    )

    # Calibrer l'axe Y sur les valeurs réelles (potentiellement négatives)
    all_kd_values = [0] + [v for xu in player_series for _, v in player_series[xu]]
    y_min = min(all_kd_values)
    y_max = max(all_kd_values)
    margin = max(1, round(max(abs(y_min), abs(y_max)) * 0.1))
    fig.update_yaxes(
        range=[y_min - margin, y_max + margin],
    )

    # Formater l'axe X en MM:SS
    all_times = [t for xu in player_series for t, _ in player_series[xu]]
    if all_times:
        max_t = max(all_times)
        tick_interval = 60_000
        tick_vals = list(range(0, max_t + tick_interval, tick_interval))
        tick_texts = [_format_time(t) for t in tick_vals]
        fig.update_xaxes(tickmode="array", tickvals=tick_vals, ticktext=tick_texts)

    apply_halo_plot_style(fig)
    return fig


# =============================================================================
# Exports
# =============================================================================

__all__ = [
    "MatchImpactEvent",
    "get_impact_labels",
    "compute_single_match_impact",
    "plot_match_kill_death_timeline",
    "plot_all_players_frags_timeline",
]
