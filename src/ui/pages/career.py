"""Page Carrière — Progression du rang Halo Infinite.

Affiche le rang actuel, une gauge de progression XP, et l'historique
de progression dans le temps.
"""

from __future__ import annotations

import logging

import plotly.graph_objects as go
import streamlit as st

from src.config import THEME_COLORS
from src.ui.career_ranks import (
    format_career_rank_label_fr,
    get_rank_icon_path,
)
from src.ui.components.career_progress_circle import (
    RANK_MAX,
    XP_HERO_TOTAL,
    compute_hero_progress,
    create_career_progress_gauge,
    create_hero_progress_gauge,
)
from src.ui.player_assets import ensure_local_image_path
from src.ui.streamlit_modern import fragment_if_available
from src.visualization.theme import apply_halo_plot_style

logger = logging.getLogger(__name__)


def _load_career_data(db_path: str, xuid: str) -> dict | None:
    """Charge les dernières données de rang carrière depuis DuckDB.

    Returns:
        Dict avec rank, rank_name, rank_tier, current_xp, etc. ou None.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            result = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          adornment_path, recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at DESC
                   LIMIT 1""",
                (xuid,),
            ).fetchone()

            if result:
                return {
                    "rank": result[0],
                    "rank_name": result[1],
                    "rank_tier": result[2],
                    "current_xp": result[3],
                    "xp_for_next_rank": result[4],
                    "xp_total": result[5],
                    "is_max_rank": bool(result[6]),
                    "adornment_path": result[7],
                    "recorded_at": result[8],
                }
    except Exception as e:
        logger.debug(f"Impossible de charger career_progression: {e}")

    return None


def _load_career_history(db_path: str, xuid: str, limit: int = 50) -> list[dict]:
    """Charge l'historique de progression depuis DuckDB.

    Returns:
        Liste de dicts ordonnés par date croissante.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at ASC
                   LIMIT ?""",
                (xuid, limit),
            ).fetchall()

            return [
                {
                    "rank": r[0],
                    "rank_name": r[1],
                    "rank_tier": r[2],
                    "current_xp": r[3],
                    "xp_for_next_rank": r[4],
                    "xp_total": r[5],
                    "is_max_rank": bool(r[6]),
                    "recorded_at": r[7],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger career_history: {e}")
        return []


def _create_xp_history_chart(history: list[dict]) -> go.Figure | None:
    """Crée un graphique d'historique XP total dans le temps."""
    if len(history) < 2:
        return None

    dates = [h["recorded_at"] for h in history]
    xp_totals = [h["xp_total"] or 0 for h in history]

    # Texte au survol avec le rang
    hover_texts = []
    for h in history:
        name = h.get("rank_name", "")
        tier = h.get("rank_tier", "")
        label = format_career_rank_label_fr(tier=tier, title=name, grade=None)
        hover_texts.append(f"Rang {h['rank']}: {label}<br>XP total: {h['xp_total']:,}")

    bg_rgb = THEME_COLORS.bg_plot
    bg_color = f"rgb({bg_rgb[0]}, {bg_rgb[1]}, {bg_rgb[2]})"

    fig = go.Figure()

    fig.add_trace(
        go.Scatter(
            x=dates,
            y=xp_totals,
            mode="lines+markers",
            name="XP total",
            line={"color": THEME_COLORS.accent, "width": 2},
            marker={"size": 6, "color": THEME_COLORS.accent},
            hovertext=hover_texts,
            hoverinfo="text",
        )
    )

    fig.update_layout(
        title="Progression XP",
        xaxis_title="Date",
        yaxis_title="XP total",
        paper_bgcolor=bg_color,
        plot_bgcolor=bg_color,
        font={"color": "white"},
        height=350,
        margin={"t": 40, "b": 40, "l": 60, "r": 20},
        xaxis={"gridcolor": "rgba(255,255,255,0.05)"},
        yaxis={"gridcolor": "rgba(255,255,255,0.1)"},
    )

    apply_halo_plot_style(fig)

    return fig


def _load_lusr_snapshot(db_path: str) -> list[dict]:
    """Charge le dernier rating LUSR/CSR par playlist_group depuis match_skill_rank.

    Returns:
        Liste de dicts : rating_type, rating_value, tier_label, rating_delta, playlist_group.
        Triée par playlist_group alphabétique.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            # Dernier rating par playlist_group = ligne avec start_time la plus récente
            rows = conn.execute(
                """
                WITH ranked AS (
                    SELECT
                        msr.match_id, msr.rating_type, msr.rating_value,
                        msr.tier_label, msr.sub_tier, msr.tier, msr.tier_fr,
                        msr.rating_delta, msr.playlist_group,
                        ROW_NUMBER() OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at) DESC
                        ) AS rn
                    FROM match_skill_rank msr
                )
                SELECT rating_type, rating_value, tier_label, sub_tier,
                       tier, tier_fr, rating_delta, playlist_group
                FROM ranked
                WHERE rn = 1
                ORDER BY playlist_group
                """
            ).fetchall()

            if not rows:
                return []
            return [
                {
                    "rating_type": r[0],
                    "rating_value": r[1],
                    "tier_label": r[2],
                    "sub_tier": r[3] or 0,
                    "tier": r[4],
                    "tier_fr": r[5],
                    "rating_delta": r[6],
                    "playlist_group": r[7],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger match_skill_rank: {e}")
        return []


def _load_lusr_history(db_path: str, playlist_group: str | None = None) -> list[dict]:
    """Charge l'historique LUSR/CSR pour le graphe d'évolution.

    Args:
        db_path: Chemin vers stats.duckdb.
        playlist_group: Filtrer par groupe (None = tous).

    Returns:
        Liste de dicts avec : match_id, rating_value, rating_deviation,
        rating_type, playlist_group, start_time, tier_label.
    """
    try:
        from src.utils.db import duckdb_read_only

        pg_filter = "AND msr.playlist_group = ?" if playlist_group else ""
        params: list = [playlist_group] if playlist_group else []

        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                f"""
                SELECT msr.match_id, msr.rating_value, msr.rating_deviation,
                       msr.rating_type, msr.playlist_group, msr.tier_label,
                       COALESCE(msr.start_time, msr.created_at) AS start_time
                FROM match_skill_rank msr
                WHERE 1=1 {pg_filter}
                ORDER BY COALESCE(msr.start_time, msr.created_at) ASC
                """,
                params,
            ).fetchall()

            return [
                {
                    "match_id": r[0],
                    "rating_value": r[1],
                    "rating_deviation": r[2],
                    "rating_type": r[3],
                    "playlist_group": r[4],
                    "tier_label": r[5],
                    "start_time": r[6],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger l'historique LUSR: {e}")
        return []


_PG_LABELS: dict[str, str] = {
    "ranked": "Classé",
    "arena": "Arena",
    "btb": "Big Team Battle",
    "tactical": "Tactique",
    "social": "Social",
    "fun": "Fun",
}
_PG_ICONS: dict[str, str] = {
    "ranked": "🏆",
    "arena": "⚔️",
    "btb": "💥",
    "tactical": "🎯",
    "social": "🎮",
    "fun": "🎉",
}
# Ordre d'affichage des groupes (du plus compétitif au plus détendu)
_PG_ORDER = ["ranked", "arena", "btb", "tactical", "social", "fun"]


def _render_lusr_section(*, db_path: str, xuid: str) -> None:
    """Rend la section LUSR/CSR sur la page Carrière.

    Affiche :
    1. Cartes visuelles par groupe (rang, image, delta)
    2. Graphe d'évolution avec sélecteur de groupe
    """
    from pathlib import Path

    try:
        import polars as pl

        from src.analysis.skill_rating_config import LUSR_SUBTITLE, LUSR_TITLE, get_rank_image_path
        from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG
        from src.visualization.timeseries_combat import plot_lusr_timeseries
    except ImportError:
        logger.debug("Modules LUSR non disponibles pour career.py")
        return

    st.subheader(f"🏅 {LUSR_TITLE} — {LUSR_SUBTITLE}")

    snapshot = _load_lusr_snapshot(db_path)
    if not snapshot:
        st.info(
            "Aucun rating LUSR/CSR calculé. "
            "Utilisez `--lusr` (non classé) ou `--csr` (classé) pour calculer."
        )
        return

    # ── Cartes visuelles — triées par ordre de compétitivité ──
    snap_by_group = {s["playlist_group"]: s for s in snapshot}
    ordered = [snap_by_group[g] for g in _PG_ORDER if g in snap_by_group]
    # Groupes non listés dans _PG_ORDER (futur) en queue
    ordered += [s for s in snapshot if s["playlist_group"] not in _PG_ORDER]

    project_root = Path(db_path).parents[2]
    n_per_row = 3
    for row_start in range(0, len(ordered), n_per_row):
        batch = ordered[row_start : row_start + n_per_row]
        cols = st.columns(len(batch))
        for col, snap in zip(cols, batch, strict=False):
            pg = snap["playlist_group"] or "?"
            tier_label = snap["tier_label"] or "Non classé"
            r_value = float(snap["rating_value"] or 0.0)
            delta = snap["rating_delta"]
            r_type = snap["rating_type"] or "LUSR"

            with col:
                # ── En-tête du groupe ──
                pg_label = _PG_LABELS.get(pg, pg.capitalize())
                pg_icon = _PG_ICONS.get(pg, "🎮")
                st.markdown(
                    f"<div style='text-align:center;font-weight:600;margin-bottom:4px'>"
                    f"{pg_icon} {pg_label}</div>",
                    unsafe_allow_html=True,
                )

                # ── Image du rang (centrée) ──
                img_path_rel = get_rank_image_path(r_value)
                if img_path_rel:
                    img_full = project_root / img_path_rel
                    if img_full.exists():
                        # Centrage via colonnes vides
                        _, c_img, _ = st.columns([1, 2, 1])
                        c_img.image(str(img_full), width=90)

                # ── Tier label ──
                st.markdown(
                    f"<div style='text-align:center;font-size:1.05em;font-weight:700;"
                    f"margin:4px 0'>{tier_label}</div>",
                    unsafe_allow_html=True,
                )

                # ── Badge type + rating ──
                badge_bg = "#00B7EB" if r_type == "LUSR" else "#FFD700"
                badge_fg = "#000"
                st.markdown(
                    f"<div style='text-align:center;margin:2px 0'>"
                    f"<span style='background:{badge_bg};color:{badge_fg};"
                    f"padding:1px 7px;border-radius:10px;font-size:0.72em;"
                    f"font-weight:600'>{r_type}</span>"
                    f"&nbsp;<span style='font-size:1.1em;font-weight:700'>"
                    f"{r_value:.0f}</span></div>",
                    unsafe_allow_html=True,
                )

                # ── Delta (dernier match) ──
                if delta is not None:
                    d_color = "#00C853" if delta >= 0 else "#FF5252"
                    d_arrow = "▲" if delta >= 0 else "▼"
                    d_sign = "+" if delta >= 0 else ""
                    st.markdown(
                        f"<div style='text-align:center;color:{d_color};"
                        f"font-size:0.95em'>{d_arrow} {d_sign}{delta:.0f}</div>",
                        unsafe_allow_html=True,
                    )
                else:
                    st.markdown(
                        "<div style='text-align:center;color:#888;font-size:0.9em'>—</div>",
                        unsafe_allow_html=True,
                    )

    # ── Graphe d'évolution avec sélecteur de groupe ──
    history_all = _load_lusr_history(db_path)
    if not history_all:
        return

    df_all = pl.DataFrame(history_all)
    available_groups = sorted(df_all["playlist_group"].drop_nulls().unique().to_list())
    if not available_groups:
        return

    st.markdown("#### 📈 Évolution du rating")

    # Sélecteur de groupe : "Tous" + un par groupe disponible
    group_options: dict[str, str | None] = {"Tous les groupes": None}
    for g in _PG_ORDER:
        if g in available_groups:
            group_options[f"{_PG_ICONS.get(g, '🎮')} {_PG_LABELS.get(g, g.capitalize())}"] = g
    # Groupes hors _PG_ORDER en queue
    for g in available_groups:
        if g not in _PG_ORDER:
            group_options[f"🎮 {g.capitalize()}"] = g

    selected_label = st.selectbox(
        "Groupe :",
        options=list(group_options.keys()),
        key="lusr_group_select",
    )
    selected_group = group_options[selected_label]

    chart_title = f"LUSR / CSR — {selected_label}"
    try:
        fig = plot_lusr_timeseries(
            df_all,
            title=chart_title,
            playlist_group=selected_group,
        )
        st.plotly_chart(
            fig,
            key=f"lusr_chart_{selected_label}",
            width="stretch",
            config=PLOTLY_CLEAN_CONFIG,
        )
    except Exception as e:
        st.warning(f"Impossible d'afficher le graphe : {e}")


@fragment_if_available
def render_career_page(
    *,
    db_path: str,
    xuid: str,
    db_key: str | None = None,
) -> None:
    """Rend la page Carrière avec rang actuel, gauge et historique."""
    st.header("Carrière")

    # Charger les données
    career_data = _load_career_data(db_path, xuid)

    if career_data is None:
        st.info(
            "Aucune donnée de carrière disponible. "
            "Synchronisez vos données pour voir votre progression de rang."
        )
        return

    rank_number = career_data.get("rank", 0)
    rank_name = career_data.get("rank_name", "")
    rank_tier = career_data.get("rank_tier", "")
    current_xp = career_data.get("current_xp", 0) or 0
    xp_for_next = career_data.get("xp_for_next_rank", 1) or 1
    xp_total = career_data.get("xp_total", 0) or 0
    is_max = career_data.get("is_max_rank", False)

    # Calcul progression
    if is_max:
        progress_pct = 100.0
    elif xp_for_next > 0:
        progress_pct = min(100.0, (current_xp / xp_for_next) * 100)
    else:
        progress_pct = 0.0

    # Label FR du rang
    rank_label_fr = format_career_rank_label_fr(tier=rank_tier, title=rank_name, grade=None)
    if not rank_label_fr:
        rank_label_fr = rank_name or f"Rang {rank_number}"

    # --- Header avec icone + metriques ---
    col_icon, col_info, col_gauge = st.columns([1, 2, 2])

    with col_icon:
        # Priorité: adornment (DB) > icône locale du rang (10C.3.4)
        adornment_db_path = career_data.get("adornment_path") or ""
        displayed_icon = False

        if adornment_db_path:
            # L'adornment_path peut être une URL ou un chemin CMSlocal
            local_adornment = ensure_local_image_path(
                adornment_db_path,
                prefix="adornment",
                download_enabled=True,
                auto_refresh_hours=0,
            )
            if local_adornment:
                st.image(local_adornment, width=140)
                displayed_icon = True

        if not displayed_icon:
            # Fallback: icône locale standard
            icon_path = get_rank_icon_path(rank_number) if rank_number else None
            if icon_path and icon_path.exists():
                st.image(str(icon_path), width=120)
            else:
                st.markdown(f"### {rank_number}")

    with col_info:
        st.subheader(rank_label_fr)

        # Métriques
        m1, m2 = st.columns(2)
        with m1:
            st.metric("Rang", f"{rank_number} / 272")
            st.metric("XP total", f"{xp_total:,}")
        with m2:
            if is_max:
                st.metric("Statut", "Rang maximum")
            else:
                st.metric("XP actuel", f"{current_xp:,}")
                st.metric("XP prochain rang", f"{xp_for_next:,}")

    with col_gauge:
        # Gauge de progression
        try:
            gauge_fig = create_career_progress_gauge(
                current_xp=current_xp,
                xp_for_next_rank=xp_for_next,
                progress_pct=progress_pct,
                rank_name_fr=rank_label_fr,
                is_max_rank=is_max,
            )
            if gauge_fig is not None:
                st.plotly_chart(
                    gauge_fig, key="career_gauge", width="stretch", config={"staticPlot": True}
                )
            else:
                st.info("Impossible de générer la jauge de progression.")
        except Exception as e:
            st.warning(f"Impossible d'afficher la jauge de progression : {e}")

    # --- Progression vers Héros ---
    st.divider()
    st.subheader("Progression vers Héros")

    hero_data = compute_hero_progress(xp_total=xp_total, rank=rank_number, is_max_rank=is_max)
    hero_pct = hero_data["percentage"]
    xp_remaining = hero_data["xp_remaining"]

    col_hero_metrics, col_hero_gauge = st.columns([3, 2])

    with col_hero_metrics:
        m1, m2, m3, m4 = st.columns(4)
        with m1:
            st.metric("XP gagnée", f"{xp_total:,}")
        with m2:
            st.metric("XP restante", f"{xp_remaining:,}")
        with m3:
            st.metric("Total requis", f"{XP_HERO_TOTAL:,}")
        with m4:
            st.metric("Rang", f"{rank_number} / {RANK_MAX}")

    with col_hero_gauge:
        try:
            hero_gauge = create_hero_progress_gauge(
                hero_pct=hero_pct,
                xp_total=xp_total,
                xp_remaining=xp_remaining,
                is_max_rank=is_max,
            )
            st.plotly_chart(
                hero_gauge,
                key="hero_progress_gauge",
                width="stretch",
                config={"staticPlot": True},
            )
        except Exception as e:
            st.warning(f"Impossible d'afficher la progression vers Héros : {e}")

    # --- Historique de progression ---
    st.divider()

    history = _load_career_history(db_path, xuid)

    if history:
        try:
            history_fig = _create_xp_history_chart(history)
            if history_fig:
                st.plotly_chart(
                    history_fig,
                    key="career_xp_history",
                    width="stretch",
                    config={"displayModeBar": False},
                )
            else:
                st.info("Pas assez de données pour afficher l'historique.")
        except Exception as e:
            st.warning(f"Impossible d'afficher l'historique de progression : {e}")

        # Tableau récapitulatif des derniers snapshots
        with st.expander("Historique détaillé", expanded=False):
            # Afficher les 10 derniers snapshots (du plus récent au plus ancien)
            recent = list(reversed(history[-10:]))
            for snap in recent:
                snap_label = format_career_rank_label_fr(
                    tier=snap.get("rank_tier", ""),
                    title=snap.get("rank_name", ""),
                    grade=None,
                )
                date_str = str(snap.get("recorded_at", ""))[:19]
                xp_t = snap.get("xp_total", 0) or 0
                st.text(f"{date_str}  |  Rang {snap['rank']}: {snap_label}  |  XP: {xp_t:,}")
    else:
        st.info(
            "Pas encore d'historique de progression. Les données seront collectées à chaque synchronisation."
        )

    # --- LUSR / CSR — LevelUp Skill Rank ---
    st.divider()
    _render_lusr_section(db_path=db_path, xuid=xuid)
