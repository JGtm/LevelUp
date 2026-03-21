"""Chaînes viz — hovertemplates Plotly (52 clés)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Hover templates (parties variables) ─────────────────────────────────
    "hover_wins": {
        "fr": "Victoires",
        "en": "Wins",
    },
    "hover_losses": {
        "fr": "Défaites",
        "en": "Losses",
    },
    "hover_ties": {
        "fr": "Égalités",
        "en": "Ties",
    },
    "hover_unfinished": {
        "fr": "Non terminés",
        "en": "Unfinished",
    },
    "hover_win_rate": {
        "fr": "Win Rate",
        "en": "Win Rate",
    },
    # ── Hovertemplates (Phase 3) ───────────────────────────────────────────────
    "hover_kda_combined": {
        "fr": "frags=%{customdata[0]} morts=%{customdata[1]} assistances=%{customdata[2]}<br>précision=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
        "en": "kills=%{customdata[0]} deaths=%{customdata[1]} assists=%{customdata[2]}<br>accuracy=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
    },
    "hover_assists_combined": {
        "fr": "assistances=%{y}<br>frags=%{customdata[0]} morts=%{customdata[1]}<br>précision=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
        "en": "assists=%{y}<br>kills=%{customdata[0]} deaths=%{customdata[1]}<br>accuracy=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
    },
    "hover_avg_smoothed": {
        "fr": "moyenne=%{y:.2f}<extra></extra>",
        "en": "average=%{y:.2f}<extra></extra>",
    },
    "hover_avg_smoothed_s": {
        "fr": "moyenne=%{y:.2f}s<extra></extra>",
        "en": "average=%{y:.2f}s<extra></extra>",
    },
    "hover_avg_s1": {
        "fr": "moyenne=%{y:.1f}<extra></extra>",
        "en": "average=%{y:.1f}<extra></extra>",
    },
    "hover_perfect_sprees": {
        "fr": "frags parfaits=%{y}<extra></extra>",
        "en": "perfect sprees=%{y}<extra></extra>",
    },
    "hover_avg": {
        "fr": "moy=%{y:.2f}<extra></extra>",
        "en": "avg=%{y:.2f}<extra></extra>",
    },
    "hover_avg0": {
        "fr": "moy=%{y:.0f}<extra></extra>",
        "en": "avg=%{y:.0f}<extra></extra>",
    },
    "hover_kpm": {
        "fr": "frags/min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (frags=%{customdata[1]:.0f})<extra></extra>",
        "en": "kills/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (kills=%{customdata[1]:.0f})<extra></extra>",
    },
    "hover_dpm": {
        "fr": "morts/min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (morts=%{customdata[2]:.0f})<extra></extra>",
        "en": "deaths/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (deaths=%{customdata[2]:.0f})<extra></extra>",
    },
    # hover_dpm_neg : utilisé quand deaths sont tracées en négatif (customdata[5] = valeur absolue)
    "hover_dpm_neg": {
        "fr": "morts/min=%{customdata[5]:.2f}<br>temps joué=%{customdata[0]:.0f}s (morts=%{customdata[2]:.0f})<extra></extra>",
        "en": "deaths/min=%{customdata[5]:.2f}<br>time played=%{customdata[0]:.0f}s (deaths=%{customdata[2]:.0f})<extra></extra>",
    },
    "hover_apm": {
        "fr": "assist./min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (assistances=%{customdata[3]:.0f})<extra></extra>",
        "en": "assists/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (assists=%{customdata[3]:.0f})<extra></extra>",
    },
    "hover_accuracy_pct": {
        "fr": "précision=%{y:.2f}%<extra></extra>",
        "en": "accuracy=%{y:.2f}%<extra></extra>",
    },
    "hover_lifespan": {
        "fr": "durée de vie moy.=%{y:.1f}s<br>morts=%{customdata[0]}<br>temps joué=%{customdata[1]:.0f}s<extra></extra>",
        "en": "avg lifespan=%{y:.1f}s<br>deaths=%{customdata[0]}<br>time played=%{customdata[1]:.0f}s<extra></extra>",
    },
    "hover_killing_spree": {
        "fr": "folie meurtrière=%{y}<extra></extra>",
        "en": "killing spree=%{y}<extra></extra>",
    },
    "hover_headshots": {
        "fr": "tirs à la tête=%{y}<extra></extra>",
        "en": "headshots=%{y}<extra></extra>",
    },
    "hover_streak": {
        "fr": "série=%{y}<br>date=%{customdata}<extra></extra>",
        "en": "streak=%{y}<br>date=%{customdata}<extra></extra>",
    },
    "hover_dmg_dealt": {
        "fr": "infligés=%{y:.0f}<extra></extra>",
        "en": "dealt=%{y:.0f}<extra></extra>",
    },
    "hover_dmg_taken": {
        "fr": "subis=%{y:.0f}<extra></extra>",
        "en": "taken=%{y:.0f}<extra></extra>",
    },
    "hover_shots_fired": {
        "fr": "tirés=%{y:.0f}<extra></extra>",
        "en": "fired=%{y:.0f}<extra></extra>",
    },
    "hover_shots_hit": {
        "fr": "touchés=%{y:.0f}<extra></extra>",
        "en": "hit=%{y:.0f}<extra></extra>",
    },
    "hover_net_kd_cumul": {
        "fr": "Min %{x}<br>Net F/D cumulé: %{y:+d}<extra></extra>",
        "en": "Round %{x}<br>Net K/D cumul: %{y:+d}<extra></extra>",
    },
    "hover_killed_by": {
        "fr": "<b>%{y}</b><br>M'a tué: %{x} fois<extra></extra>",
        "en": "<b>%{y}</b><br>Killed me: %{x} times<extra></extra>",
    },
    "hover_i_killed": {
        "fr": "<b>%{y}</b><br>Tué: %{x} fois<extra></extra>",
        "en": "<b>%{y}</b><br>Killed: %{x} times<extra></extra>",
    },
    "hover_kde": {
        "fr": "FDA=%{x:.2f}<br>densité=%{y:.3f}<extra></extra>",
        "en": "CDF=%{x:.2f}<br>density=%{y:.3f}<extra></extra>",
    },
    "hover_first_event": {
        "fr": "Temps: %{x:.0f}s<br>Matchs: %{y}<extra></extra>",
        "en": "Time: %{x:.0f}s<br>Matches: %{y}<extra></extra>",
    },
    "hover_weapons": {
        "fr": "%{y}<br>Frags: %{x}<br>Headshot: %{customdata[0]:.1f}%<br>Précision: %{customdata[1]:.1f}%<extra></extra>",
        "en": "%{y}<br>Kills: %{x}<br>Headshot: %{customdata[0]:.1f}%<br>Accuracy: %{customdata[1]:.1f}%<extra></extra>",
    },
    "hover_cumul_score": {
        "fr": "<b>%{x}</b><br>Cumulé: %{y:+d}<extra></extra>",
        "en": "<b>%{x}</b><br>Cumul: %{y:+d}<extra></extra>",
    },
    "hover_kd_cumul_line": {
        "fr": "<b>%{x}</b><br>F/D Cumulé: %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>K/D Cumul: %{y:.2f}<extra></extra>",
    },
    "hover_kd_rolling_line": {
        "fr": "<b>%{x}</b><br>F/D Glissant: %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>Rolling K/D: %{y:.2f}<extra></extra>",
    },
    "hover_kd_ewma": {
        "fr": "<b>%{x}</b><br>F/D Lissé (EWMA): %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>Smoothed K/D (EWMA): %{y:.2f}<extra></extra>",
    },
    "hover_nph_raw": {
        "fr": "<b>%{x}</b><br>Net/h: %{y:+.1f}<extra></extra>",
        "en": "<b>%{x}</b><br>Net/h: %{y:+.1f}<extra></extra>",
    },
    "hover_nph_rolling": {
        "fr": "<b>%{x}</b><br>Net/h lissé: %{y:+.1f}<extra></extra>",
        "en": "<b>%{x}</b><br>Smoothed Net/h: %{y:+.1f}<extra></extra>",
    },
    "hover_kd_match": {
        "fr": "<b>%{x}</b><br>F/D Match: %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>Match K/D: %{y:.2f}<extra></extra>",
    },
    "hover_trend_smooth": {
        "fr": "tendance=%{y:.0f}<extra></extra>",
        "en": "trend=%{y:.0f}<extra></extra>",
    },
    "hover_match_cumul": {
        "fr": "<b>{label}</b><br>Match #%{{x}}<br>Cumulé: %{{y:+d}}<extra></extra>",
        "en": "<b>{label}</b><br>Match #%{{x}}<br>Cumul: %{{y:+d}}<extra></extra>",
    },
    # ── Impact timeline hovertemplates ───────────────────────────────────────
    "hover_kill_cumul": {
        "fr": "<b>Frag #%{y}</b><br>%{text}<extra></extra>",
        "en": "<b>Kill #%{y}</b><br>%{text}<extra></extra>",
    },
    "hover_death_cumul": {
        "fr": "<b>Mort #%{y}</b><br>%{text}<extra></extra>",
        "en": "<b>Death #%{y}</b><br>%{text}<extra></extra>",
    },
    # ── Hovertemplates Distributions ─────────────────────────────────────────
    "hover_kda_rug": {
        "fr": "FDA=%{x:.2f}<extra></extra>",
        "en": "KDA=%{x:.2f}<extra></extra>",
    },
    # ── Hovertemplates Antagonistes ───────────────────────────────────────────
    "hover_kills_per_min": {
        "fr": "Min %{x}<br>Frags: %{y}<extra></extra>",
        "en": "Min %{x}<br>Kills: %{y}<extra></extra>",
    },
    "hover_deaths_per_min": {
        "fr": "Min %{x}<br>Morts: %{customdata}<extra></extra>",
        "en": "Min %{x}<br>Deaths: %{customdata}<extra></extra>",
    },
    "hover_heatmap_kills": {
        "fr": "<b>%{y}</b> → <b>%{x}</b><br>Frags: %{z}<extra></extra>",
        "en": "<b>%{y}</b> → <b>%{x}</b><br>Kills: %{z}<extra></extra>",
    },
    # ── Hovertemplates Objectifs ──────────────────────────────────────────────
    "hover_obj_kills_score": {
        "fr": "<b>%{text}</b><br>Frags: %{x}<br>Score Objectifs: %{y}<br><extra>%{customdata[0]}</extra>",
        "en": "<b>%{text}</b><br>Kills: %{x}<br>Objective Score: %{y}<br><extra>%{customdata[0]}</extra>",
    },
    "hover_obj_leaderboard": {
        "fr": "<b>%{y}</b><br>Score Total: %{x:,.0f}<br>Matchs: %{customdata}<extra></extra>",
        "en": "<b>%{y}</b><br>Total Score: %{x:,.0f}<br>Matches: %{customdata}<extra></extra>",
    },
    "hover_obj_score_line": {
        "fr": "<b>%{x}</b><br>Objectifs: %{y}<extra></extra>",
        "en": "<b>%{x}</b><br>Objectives: %{y}<extra></extra>",
    },
    "hover_obj_total_line": {
        "fr": "<b>%{x}</b><br>Total: %{y}<extra></extra>",
        "en": "<b>%{x}</b><br>Total: %{y}<extra></extra>",
    },
    # ── Hovertemplates Participation ──────────────────────────────────────────
    "hover_score_pts_h": {
        "fr": "<b>%{y}</b><br>Score: %{x:,.0f} pts<extra></extra>",
        "en": "<b>%{y}</b><br>Score: %{x:,.0f} pts<extra></extra>",
    },
    "hover_score_pts_v": {
        "fr": "<b>%{x}</b><br>Score: %{y:,.0f} pts<extra></extra>",
        "en": "<b>%{x}</b><br>Score: %{y:,.0f} pts<extra></extra>",
    },
    # ── Hover supplémentaires ─────────────────────────────────────────────────
    "hover_kda": {
        "fr": "Frags: %{customdata[0]}<br>Morts: %{customdata[1]}<br>Assists: %{customdata[2]}",
        "en": "Kills: %{customdata[0]}<br>Deaths: %{customdata[1]}<br>Assists: %{customdata[2]}",
    },
    "hover_kill_ordinal": {"fr": "Frag n°%{x}", "en": "Kill #%{x}"},
    "hover_death_ordinal": {"fr": "Mort n°%{x}", "en": "Death #%{x}"},
}
