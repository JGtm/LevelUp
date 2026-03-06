"""Page Carrière — Tableaux HTML des rencontres et antagonistes."""

from __future__ import annotations

import contextlib
import html
from datetime import datetime

from src.ui.i18n import t
from src.ui.pages.match_table_html import gamertag_link
from src.ui.pages.match_view_encounters import (
    badge_html,
    kd_cell_html,
    ordinal_badge_html,
    role_cell_html,
    wr_cell_html,
)
from src.ui.pages.match_view_encounters_logic import (
    EncounterStats,
    _relative_date,
    compute_encounter_badges,
    ordinal,
)


def _to_opt_float(value: object) -> float | None:
    """Convertit une valeur nullable en float."""
    return float(value) if value is not None else None


def _kd_style(kills: int, deaths: int) -> str:
    """Retourne un style CSS selon le ratio K/D."""
    if deaths == 0:
        return "color:#33ffbf;font-weight:700;" if kills > 0 else ""
    ratio = kills / deaths
    if ratio >= 1.5:
        return "color:#33ffbf;font-weight:700;"
    if ratio <= 0.5:
        return "color:#ff9e6b;font-weight:700;"
    return ""


def _parse_last_seen(raw: object) -> datetime | None:
    """Convertit une valeur brute en datetime (ou None)."""
    if isinstance(raw, datetime):
        return raw
    if raw is not None:
        with contextlib.suppress(Exception):
            return datetime.fromisoformat(str(raw))
    return None


def _encounter_row_html(r: dict, gt_html: str, side: str) -> str:
    """Construit la ligne HTML d'un joueur croisé ≥2 fois."""
    total = r.get("total_encounters", 0)
    gamertag = r.get("gamertag") or r.get("xuid", "—")[:12]
    stats = EncounterStats(
        xuid=r.get("xuid", ""),
        gamertag=gamertag,
        total_encounters=total,
        ally_count=r.get("ally_count", 0),
        enemy_count=r.get("enemy_count", 0),
        winrate_as_ally=_to_opt_float(r.get("winrate_as_ally")),
        winrate_vs_enemy=_to_opt_float(r.get("winrate_vs_enemy")),
        kills_dealt=int(r.get("kills_dealt") or 0),
        deaths_suffered=int(r.get("deaths_suffered") or 0),
        last_seen=_parse_last_seen(r.get("last_seen")),
        current_side=side,
    )
    badges = compute_encounter_badges(stats)
    badges_html = " ".join(badge_html(b) for b in badges)
    enc_detail = f"A:{stats.ally_count} | E:{stats.enemy_count}"
    last_str = html.escape(_relative_date(stats.last_seen)) if stats.last_seen else "—"
    return (
        f"<tr class='os-sb-row'>"
        f"<td class='os-sb-td'>{gt_html}{ordinal_badge_html(total)} {badges_html}</td>"
        f"<td class='os-sb-td'>{role_cell_html(side)}</td>"
        f"<td class='os-sb-td'>{total} <span style='color:#888;font-size:0.8em;'>({enc_detail})</span></td>"
        f"<td class='os-sb-td'>{wr_cell_html(stats.winrate_as_ally, stats.ally_count)}</td>"
        f"<td class='os-sb-td'>{wr_cell_html(stats.winrate_vs_enemy, stats.enemy_count)}</td>"
        f"<td class='os-sb-td'>{kd_cell_html(stats.kills_dealt, stats.deaths_suffered)}</td>"
        f"<td class='os-sb-td' style='color:#aaa;font-size:0.85em;'>{last_str}</td>"
        f"</tr>"
    )


def build_encounters_table_html(rows: list[dict], title: str) -> str:
    """Construit un tableau HTML top joueurs croisés (style Encounter History)."""
    n_cols = 7
    col_headers = "".join(
        f"<th class='os-sb-th'>{html.escape(h)}</th>"
        for h in [
            t("col_player"),
            t("col_role"),
            t("col_encounters"),
            t("col_wr_ally"),
            t("col_wr_enemy"),
            t("col_kd_cross"),
            t("col_last_seen"),
        ]
    )
    thead = (
        f"<thead>"
        f"<tr><th class='os-sb-team' colspan='{n_cols}'>{html.escape(title)}</th></tr>"
        f"<tr>{col_headers}</tr>"
        f"</thead>"
    )
    body_rows: list[str] = []
    for r in rows:
        total = r.get("total_encounters", 0)
        gamertag = r.get("gamertag") or r.get("xuid", "—")[:12]
        gt_html = gamertag_link(gamertag) if gamertag and gamertag != "—" else "—"
        ally_count = r.get("ally_count", 0)
        enemy_count = r.get("enemy_count", 0)
        side = "allié" if ally_count > enemy_count else "ennemi"
        if total <= 1:
            body_rows.append(
                f"<tr class='os-sb-row'>"
                f"<td class='os-sb-td'>{gt_html}{ordinal_badge_html(1)}</td>"
                f"<td class='os-sb-td'>{role_cell_html(side)}</td>"
                f"<td class='os-sb-td' colspan='5' style='color:#666;font-style:italic;'>"
                f"{html.escape(t('encounter_ordinal', ordinal=ordinal(1)))}</td></tr>"
            )
            continue
        body_rows.append(_encounter_row_html(r, gt_html, side))
    tbody = "<tbody>" + "".join(body_rows) + "</tbody>"
    return (
        f"<div class='os-table-wrap os-sb-wrap'>"
        f"<table class='os-table os-scoreboard'>"
        f"{thead}{tbody}</table></div>"
    )


def build_antagonist_table_html(
    rows: list[dict],
    title: str,
    *,
    mode: str,
) -> str:
    """Construit un tableau HTML top némésis ou souffre-douleurs."""
    col_main = t("col_times_killed_by") if mode == "nemesis" else t("col_times_killed")
    col_sec = t("col_times_killed") if mode == "nemesis" else t("col_times_killed_by")
    header = (
        "<thead><tr>"
        f"<th class='os-sb-th' style='text-align:left'>#</th>"
        f"<th class='os-sb-th' style='text-align:left'>{t('col_player')}</th>"
        f"<th class='os-sb-th'>{col_main}</th>"
        f"<th class='os-sb-th'>{col_sec}</th>"
        f"<th class='os-sb-th'>{t('col_net_kills')}</th>"
        f"<th class='os-sb-th'>{t('col_matches_against')}</th>"
        "</tr></thead>"
    )
    body_rows = []
    for i, r in enumerate(rows, 1):
        opp_gt = r.get("opponent_gamertag") or ""
        gt = gamertag_link(opp_gt) if opp_gt else "—"
        killed = r.get("times_killed", 0)
        killed_by = r.get("times_killed_by", 0)
        net = r.get("net_kills", killed - killed_by)
        matches = r.get("matches_against", 0)
        main_val = killed_by if mode == "nemesis" else killed
        sec_val = killed if mode == "nemesis" else killed_by
        net_style = _kd_style(killed, killed_by)
        net_sign = "+" if net > 0 else ""
        body_rows.append(
            f"<tr class='os-sb-row'><td class='os-sb-td'>{i}</td>"
            f"<td class='os-sb-td'>{gt}</td>"
            f"<td class='os-sb-td' style='text-align:center;font-weight:700'>{main_val}</td>"
            f"<td class='os-sb-td' style='text-align:center'>{sec_val}</td>"
            f"<td class='os-sb-td' style='text-align:center;{net_style}'>{net_sign}{net}</td>"
            f"<td class='os-sb-td' style='text-align:center;color:#aaa'>{matches}</td></tr>"
        )
    tbody = "<tbody>" + "".join(body_rows) + "</tbody>"
    return (
        f"<div class='os-table-wrap os-sb-wrap'>"
        f"<table class='os-table os-scoreboard'>"
        f"<thead><tr><th class='os-sb-team' colspan='6'>{title}</th></tr></thead>"
        f"{header}{tbody}</table></div>"
    )
