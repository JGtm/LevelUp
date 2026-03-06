"""Rendu Streamlit de la section Historique des rencontres.

S'affiche directement sous le scoreboard sur la page Match View.
Réutilise les classes CSS du scoreboard (os-sb-*) pour la cohérence visuelle.
"""

from __future__ import annotations

import contextlib
import html
import logging
from datetime import datetime
from typing import Any

import streamlit as st

from src.data.repositories._encounter_loader import load_encounter_stats
from src.data.repositories.duckdb_repo import DuckDBRepository
from src.ui.i18n import t
from src.ui.pages.match_table_html import gamertag_link
from src.ui.pages.match_view_encounters_logic import (
    Badge,
    EncounterStats,
    _relative_date,
    build_friends_set,
    compute_encounter_badges,
    filter_encounter_xuids,
    ordinal,
)
from src.utils.db import duckdb_read_only

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Helpers HTML — tous < 20 lignes
# ---------------------------------------------------------------------------


def _badge_html(badge: Badge) -> str:
    """Génère un <span> inline pour un badge."""
    esc_label = html.escape(t(badge.label_key))
    esc_tip = html.escape(badge.tooltip)
    if badge.css_class == "os-sb-td--best":
        style = "color:#33ffbf;background:rgba(0,158,115,0.22);padding:1px 5px;border-radius:3px;font-size:0.75em;font-weight:700;"
    elif badge.css_class == "os-sb-td--worst":
        style = "color:#ff9e6b;background:rgba(213,94,0,0.22);padding:1px 5px;border-radius:3px;font-size:0.75em;font-weight:700;"
    else:  # amber
        style = "color:#FFB703;background:rgba(255,183,3,0.18);padding:1px 5px;border-radius:3px;font-size:0.75em;font-weight:700;"
    return f'<span style="{style}" title="{esc_tip}">{esc_label}</span>'


def _ordinal_badge_html(n: int) -> str:
    """Génère le badge ordinal grisé ('12e rencontre' / '12th encounter')."""
    label = html.escape(t("encounter_ordinal", ordinal=ordinal(n)))
    return f'<span style="color:#888;font-size:0.75em;margin-left:6px;">{label}</span>'


def _role_cell_html(side: str) -> str:
    """Génère la cellule Rôle colorée selon le côté (allié / ennemi)."""
    if side == "allié":
        style = "color:#a8d4f5;background:rgba(0,114,178,0.28);padding:2px 7px;border-radius:3px;font-size:0.8em;font-weight:600;"
        label = t("role_ally")
    else:
        style = "color:#f5c8a8;background:rgba(213,94,0,0.24);padding:2px 7px;border-radius:3px;font-size:0.8em;font-weight:600;"
        label = t("role_enemy")
    return f'<span style="{style}">{label}</span>'


def _wr_cell_html(wr: float | None, n_matches: int) -> str:
    """Formate un win rate avec highlight si extrême."""
    if wr is None or n_matches == 0:
        return "—"
    pct = round(wr * 100)
    if wr >= 0.65:
        css = "color:#33ffbf;font-weight:700;"
    elif wr <= 0.35:
        css = "color:#ff9e6b;font-weight:700;"
    else:
        css = ""
    return f'<span style="{css}">{pct}%</span>'


def _kd_cell_html(kills: int, deaths: int) -> str:
    """Formate le ratio K/D croisé avec highlight."""
    if kills == 0 and deaths == 0:
        return "—"
    ratio_str = f"{kills}/{deaths}"
    if deaths == 0:
        css = "color:#33ffbf;font-weight:700;"
    elif kills == 0:
        css = "color:#ff9e6b;font-weight:700;"
    else:
        ratio = kills / deaths
        if ratio >= 1.5:
            css = "color:#33ffbf;font-weight:700;"
        elif ratio <= 0.5:
            css = "color:#ff9e6b;font-weight:700;"
        else:
            css = ""
    return f'<span style="{css}">{ratio_str}</span>'


# ---------------------------------------------------------------------------
# Construction des lignes HTML
# ---------------------------------------------------------------------------


def _compact_row_html(gamertag: str, side: str) -> str:
    """Génère une ligne compacte pour un joueur rencontré pour la 1ère fois."""
    gt_html = gamertag_link(gamertag) if gamertag and gamertag != "—" else "—"
    role = _role_cell_html(side)
    ordinal = _ordinal_badge_html(1)
    return (
        f"<tr class='os-sb-row'>"
        f"<td class='os-sb-td'>{gt_html}{ordinal}</td>"
        f"<td class='os-sb-td'>{role}</td>"
        f"<td class='os-sb-td' colspan='5' style='color:#666;font-style:italic;'>{html.escape(t('encounter_ordinal', ordinal=ordinal(1)))}</td>"
        f"</tr>"
    )


def _full_row_html(stats: EncounterStats, badges: list[Badge]) -> str:
    """Génère une ligne complète avec toutes les métriques et badges."""
    gt_raw = stats.gamertag or stats.xuid[:8] or "—"
    gt_html = gamertag_link(gt_raw) if gt_raw != "—" else "—"
    ordinal = _ordinal_badge_html(stats.total_encounters)
    badges_html = " ".join(_badge_html(b) for b in badges)
    role = _role_cell_html(stats.current_side)

    enc_detail = f"A:{stats.ally_count} | E:{stats.enemy_count}"
    wr_ally = _wr_cell_html(stats.winrate_as_ally, stats.ally_count)
    wr_enemy = _wr_cell_html(stats.winrate_vs_enemy, stats.enemy_count)
    kd = _kd_cell_html(stats.kills_dealt, stats.deaths_suffered)
    last_str = html.escape(_relative_date(stats.last_seen)) if stats.last_seen else "—"

    return (
        f"<tr class='os-sb-row'>"
        f"<td class='os-sb-td'>{gt_html}{ordinal} {badges_html}</td>"
        f"<td class='os-sb-td'>{role}</td>"
        f"<td class='os-sb-td'>{stats.total_encounters} <span style='color:#888;font-size:0.8em;'>({enc_detail})</span></td>"
        f"<td class='os-sb-td'>{wr_ally}</td>"
        f"<td class='os-sb-td'>{wr_enemy}</td>"
        f"<td class='os-sb-td'>{kd}</td>"
        f"<td class='os-sb-td' style='color:#aaa;font-size:0.85em;'>{last_str}</td>"
        f"</tr>"
    )


def _build_encounter_table_html(rows_html: list[str]) -> str:
    """Assemble le tableau HTML complet."""
    n_cols = 7
    header_label = html.escape(t("mv_encounter_history"))
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
        f"<tr><th class='os-sb-team' colspan='{n_cols}'>{header_label}</th></tr>"
        f"<tr>{col_headers}</tr>"
        f"</thead>"
    )
    tbody = "<tbody>" + "".join(rows_html) + "</tbody>"
    return (
        "<div class='os-table-wrap os-sb-wrap'>"
        "<table class='os-table os-scoreboard'>"
        f"{thead}{tbody}"
        "</table>"
        "</div>"
    )


# ---------------------------------------------------------------------------
# Chargement des friends_xuids depuis player_match_enrichment
# ---------------------------------------------------------------------------


def _fetch_friends_xuids_csv(db_path: str, match_id: str) -> str | None:
    """Lit la colonne friends_xuids depuis player_match_enrichment pour ce match."""
    try:
        with duckdb_read_only(db_path) as conn:
            row = conn.execute(
                "SELECT friends_xuids FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                [match_id],
            ).fetchone()
        return str(row[0]) if row and row[0] else None
    except Exception:
        logger.debug("Lecture friends_xuids échouée pour match %s", match_id)
        return None


# ---------------------------------------------------------------------------
# Point d'entrée public
# ---------------------------------------------------------------------------


def _build_encounter_rows(
    records: list[dict[str, Any]],
    xuid_to_team: dict[str, Any],
    my_team_id: Any,
) -> list[str]:
    """Construit la liste de lignes HTML triées depuis les enregistrements DuckDB.

    Args:
        records: Dicts retournés par load_encounter_stats.
        xuid_to_team: Mapping xuid → team_id du match courant.
        my_team_id: team_id du joueur principal.

    Returns:
        Liste de chaînes HTML (une par ligne du tableau).
    """
    records.sort(
        key=lambda r: (
            0 if xuid_to_team.get(str(r.get("xuid") or "")) != my_team_id else 1,
            -(r.get("total_encounters") or 0),
        )
    )
    rows: list[str] = []
    for record in records:
        xuid = str(record.get("xuid") or "").strip()
        gamertag = str(record.get("gamertag") or xuid[:8] or "—").strip()
        total = int(record.get("total_encounters") or 0)
        side = "allié" if xuid_to_team.get(xuid) == my_team_id else "ennemi"

        if total <= 1:
            rows.append(_compact_row_html(gamertag, side))
            continue

        raw_last = record.get("last_seen")
        last_seen: datetime | None = None
        if isinstance(raw_last, datetime):
            last_seen = raw_last
        elif raw_last is not None:
            with contextlib.suppress(Exception):
                last_seen = datetime.fromisoformat(str(raw_last))

        stats = EncounterStats(
            xuid=xuid,
            gamertag=gamertag,
            total_encounters=total,
            ally_count=int(record.get("ally_count") or 0),
            enemy_count=int(record.get("enemy_count") or 0),
            winrate_as_ally=_to_float(record.get("winrate_as_ally")),
            winrate_vs_enemy=_to_float(record.get("winrate_vs_enemy")),
            kills_dealt=int(record.get("kills_dealt") or 0),
            deaths_suffered=int(record.get("deaths_suffered") or 0),
            last_seen=last_seen,
            current_side=side,
        )
        rows.append(_full_row_html(stats, compute_encounter_badges(stats)))
    return rows


def _to_float(value: Any) -> float | None:
    """Convertit une valeur nullable en float."""
    return float(value) if value is not None else None


def render_encounter_section(
    *,
    match_id: str,
    self_xuid: str,
    db_path: str,
) -> None:
    """Affiche la section Historique des rencontres sous le scoreboard.

    Ne s'affiche que si le match est valide et qu'au moins un joueur
    non-ami est présent dans la partie.

    Args:
        match_id: Identifiant du match courant.
        self_xuid: XUID du joueur principal.
        db_path: Chemin du stats.duckdb joueur.
    """
    if not match_id or not self_xuid:
        return

    try:
        repo = DuckDBRepository(db_path, xuid=self_xuid, read_only=True)
        players: list[dict[str, Any]] = repo.load_match_scoreboard(match_id.strip())
    except Exception:
        logger.debug("render_encounter_section : échec chargement scoreboard", exc_info=True)
        return

    if not players:
        return

    friends_csv = _fetch_friends_xuids_csv(db_path, match_id)
    friends_set = build_friends_set(self_xuid, friends_csv)
    target_xuids = filter_encounter_xuids(players, self_xuid, friends_set)
    if not target_xuids:
        return

    df = load_encounter_stats(self_xuid, target_xuids, db_path)
    if df.is_empty():
        return

    self_norm = str(self_xuid).strip()
    my_team_id: Any = next(
        (p.get("team_id") for p in players if str(p.get("xuid") or "").strip() == self_norm),
        None,
    )
    xuid_to_team: dict[str, Any] = {
        str(p.get("xuid") or "").strip(): p.get("team_id") for p in players
    }

    rows_html = _build_encounter_rows(df.to_dicts(), xuid_to_team, my_team_id)
    if rows_html:
        st.markdown(_build_encounter_table_html(rows_html), unsafe_allow_html=True)
        st.caption(t("mv_encounter_legend"))


__all__ = ["render_encounter_section"]
