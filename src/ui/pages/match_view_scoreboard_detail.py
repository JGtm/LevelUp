"""Détails joueur intégrés au scoreboard — panneau inline cliquable.

Chaque ligne joueur est cliquable (▶ / ▼) et révèle une ligne cachée
directement sous elle dans le tableau HTML — pas d'accordéon séparé.

Fonctions publiques
-------------------
preload_match_extra_data(main_db_path, match_id, players, lang) → dict
    Pré-charge médailles + armes pour tous les joueurs en 2 requêtes.

build_team_table_html_with_details(...) → str
    Génère le HTML complet d'une équipe avec lignes de détail intégrées.

SCOREBOARD_JS : str
    Fonction JS osToggle() à injecter une fois par rendu.
"""

from __future__ import annotations

import base64
import html
import logging
import os
from typing import Any

from src.config import TEAM_MAP, get_bot_name
from src.ui.i18n import t
from src.ui.medals import get_local_medals_icons_dir, load_medal_name_maps
from src.utils import parse_xuid_input
from src.utils.paths import get_player_db_path

logger = logging.getLogger(__name__)

# Nombre max de médailles affichées dans le panneau inline
_MAX_MEDALS = 10
# Nombre max d'armes affichées
_MAX_WEAPONS = 5
# Nombre max d'awards
_MAX_AWARDS = 6

# =============================================================================
# JavaScript unique pour tous les tableaux de la page
# =============================================================================

SCOREBOARD_JS = """
<script>
window.osToggle = window.osToggle || function(id, row) {
  var dr = document.getElementById(id);
  if (!dr) return;
  var open = dr.style.display !== '' && dr.style.display !== 'none';
  dr.style.display = open ? 'none' : 'table-row';
  var btn = row ? row.querySelector('.os-sb-expand-btn') : null;
  if (btn) btn.textContent = open ? '\u25b6' : '\u25bc';
};
</script>
"""


# =============================================================================
# Chargement de données (batch)
# =============================================================================


def _load_all_medals_for_match(
    main_db_path: str, match_id: str
) -> dict[str, list[dict[str, int]]]:
    """Charge TOUTES les médailles de tous les joueurs en une seule requête.

    Returns:
        ``{xuid: [{name_id: int, count: int}, ...]}``.
    """
    if not main_db_path or not match_id:
        return {}
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(main_db_path, xuid="", read_only=True)
        conn = repo._get_connection()
        rows = conn.execute(
            "SELECT xuid, medal_name_id, count FROM shared.medals_earned WHERE match_id = ?",
            [match_id],
        ).fetchall()
        result: dict[str, list[dict[str, int]]] = {}
        for xuid, nid, cnt in rows:
            result.setdefault(str(xuid), []).append({"name_id": int(nid), "count": int(cnt)})
        return result
    except Exception:
        logger.debug("Médailles batch indisponibles match=%s", match_id)
        return {}


def _load_all_weapons_for_match(
    main_db_path: str, match_id: str
) -> dict[str, list[tuple[int, int]]]:
    """Charge les kills par arme pour tous les joueurs en une seule requête.

    Returns:
        ``{xuid: [(weapon_id, kills), ...]}``, triées par kills DESC.
    """
    if not main_db_path or not match_id:
        return {}
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(main_db_path, xuid="", read_only=True)
        df = repo.load_weapon_kills_for_match(match_id)
        if df.is_empty():
            return {}
        result: dict[str, list[tuple[int, int]]] = {}
        for row in df.to_dicts():
            xu = str(row["xuid"])
            result.setdefault(xu, []).append((int(row["weapon_id"]), int(row["kills"])))
        return result
    except Exception:
        logger.debug("Armes batch indisponibles match=%s", match_id)
        return {}


def _load_enrichment_for_gamertag(
    match_id: str, gamertag: str, xuid: str
) -> dict[str, Any]:
    """Charge l'enrichissement depuis la DB propre du joueur si disponible."""
    result: dict[str, Any] = {
        "has_db": False,
        "performance_score": None,
        "session_label": None,
        "awards": [],
    }
    if not gamertag or gamertag == "?":
        return result

    player_db = get_player_db_path(gamertag)
    if not player_db.exists():
        return result

    result["has_db"] = True
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(player_db) as conn:
            try:
                row = conn.execute(
                    "SELECT performance_score, session_label"
                    " FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                    [match_id],
                ).fetchone()
                if row:
                    result["performance_score"] = row[0]
                    result["session_label"] = row[1]
            except Exception:
                pass

            try:
                rows_a = conn.execute(
                    "SELECT award_name, award_category, award_count, award_score"
                    " FROM personal_score_awards WHERE match_id = ?"
                    " ORDER BY award_score DESC",
                    [match_id],
                ).fetchall()
                result["awards"] = [
                    {"name": r[0] or r[1] or "?", "count": r[2]}
                    for r in rows_a
                ]
            except Exception:
                pass
    except Exception:
        logger.debug("Enrichissement DB indisponible gamertag=%s", gamertag)
    return result


def preload_match_extra_data(
    main_db_path: str,
    match_id: str,
    players: list[dict[str, Any]],
    lang: str = "fr",
) -> dict[str, dict[str, Any]]:
    """Pré-charge médailles, armes et enrichissement pour tous les joueurs.

    Args:
        main_db_path: DB du joueur courant (accès shared).
        match_id: ID du match.
        players: Liste des dicts joueur du scoreboard.
        lang: Langue ("fr" ou "en").

    Returns:
        ``{xuid: {medals, weapons, enrichment}}``
    """
    all_medals = _load_all_medals_for_match(main_db_path, match_id)
    all_weapons = _load_all_weapons_for_match(main_db_path, match_id)

    extra: dict[str, dict[str, Any]] = {}
    for p in players:
        xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip())
            or str(p.get("xuid") or "").strip()
        ).strip()
        gamertag = str(p.get("gamertag") or "").strip()

        medals = all_medals.get(xu, [])
        # Trier par count décroissant
        medals = sorted(medals, key=lambda m: m.get("count", 0), reverse=True)[:_MAX_MEDALS]

        weapons_raw = all_weapons.get(xu, [])
        weapons = weapons_raw[:_MAX_WEAPONS]

        enrichment = _load_enrichment_for_gamertag(match_id, gamertag, xu)

        extra[xu] = {
            "medals": medals,
            "weapons": weapons,
            "enrichment": enrichment,
        }
    return extra


# =============================================================================
# Helpers HTML (icônes, formatage)
# =============================================================================


def _medal_icon_b64(nid: int) -> str | None:
    """Retourne le data-URI base64 d'une icône de médaille, ou None."""
    icons_dir = get_local_medals_icons_dir()
    path = os.path.join(icons_dir, f"{int(nid)}.png")
    if not os.path.exists(path):
        return None
    try:
        with open(path, "rb") as f:
            data = f.read()
        return "data:image/png;base64," + base64.b64encode(data).decode("ascii")
    except Exception:
        return None


def _fmt_detail(key: str, value: Any) -> str:
    """Formate une valeur pour le panneau de détail (version concise)."""
    if value is None:
        return "—"
    if key == "kda":
        try:
            return f"{float(value):.2f}"
        except (ValueError, TypeError):
            return str(value)
    if key == "accuracy":
        try:
            v = float(value)
            if v <= 1.0:
                v *= 100.0
            return f"{v:.1f}%"
        except (ValueError, TypeError):
            return str(value)
    if key == "avg_life_seconds":
        try:
            secs = int(float(value))
            return f"{secs // 60}:{secs % 60:02d}"
        except (ValueError, TypeError):
            return str(value)
    if key in ("damage_dealt", "damage_taken"):
        try:
            return f"{int(round(float(value))):,}"
        except (ValueError, TypeError):
            return str(value)
    return str(value)


def _kpi_html(label: str, value: str, highlight: str = "") -> str:
    """Génère un chip KPI pour le panneau de détail."""
    cls = f" os-sb-kpi--{highlight}" if highlight else ""
    return (
        f"<div class='os-sb-kpi{cls}'>"
        f"<span class='os-sb-kpi-val'>{html.escape(value)}</span>"
        f"<span class='os-sb-kpi-lbl'>{html.escape(label)}</span>"
        f"</div>"
    )


def _build_stats_html(player: dict[str, Any]) -> str:
    """Génère la section KPI du panneau de détail."""
    kpis = [
        _kpi_html(t("col_score"), _fmt_detail("score", player.get("score"))),
        _kpi_html(t("col_kills"), _fmt_detail("kills", player.get("kills"))),
        _kpi_html(t("col_deaths"), _fmt_detail("deaths", player.get("deaths"))),
        _kpi_html(t("col_assists_short"), _fmt_detail("assists", player.get("assists"))),
        _kpi_html(t("col_kda"), _fmt_detail("kda", player.get("kda"))),
        _kpi_html(t("col_accuracy"), _fmt_detail("accuracy", player.get("accuracy"))),
        _kpi_html(t("col_headshots"), str(player.get("headshot_kills") or 0)),
        _kpi_html(t("col_melee"), str(player.get("melee_kills") or 0)),
        _kpi_html(t("col_power_weapon"), str(player.get("power_weapon_kills") or 0)),
        _kpi_html(t("col_dmg_dealt"), _fmt_detail("damage_dealt", player.get("damage_dealt"))),
        _kpi_html(t("col_dmg_taken"), _fmt_detail("damage_taken", player.get("damage_taken"))),
        _kpi_html(
            t("mv_scoreboard_avg_life"),
            _fmt_detail("avg_life_seconds", player.get("avg_life_seconds")),
        ),
    ]
    return (
        f"<div class='os-sb-detail-section'>"
        f"<div class='os-sb-detail-kpis'>{''.join(kpis)}</div>"
        f"</div>"
    )


def _build_weapons_html(weapons: list[tuple[int, int]], lang: str) -> str:
    """Génère la section armes du panneau de détail."""
    if not weapons:
        return ""

    parts = []
    for wid, kills in weapons:
        try:
            from src.analysis._weapon_data import resolve_weapon_display

            name = resolve_weapon_display(int(wid), lang=lang) or f"#{wid}"
        except Exception:
            name = f"#{wid}"
        parts.append(
            f"<span class='os-sb-weapon-tag'>{html.escape(name)} ({kills})</span>"
        )
    title = html.escape(t("sb_detail_top_weapons"))
    return (
        f"<div class='os-sb-detail-section'>"
        f"<span class='os-sb-det-title'>{title}</span>"
        f"<div class='os-sb-weapon-list'>{''.join(parts)}</div>"
        f"</div>"
    )


def _build_medals_html(medals: list[dict[str, int]], lang: str) -> str:
    """Génère la section médailles du panneau de détail."""
    if not medals:
        return ""

    fr_map, en_map = load_medal_name_maps()
    chips = []
    for m in medals:
        nid = int(m.get("name_id", 0))
        cnt = int(m.get("count", 0))
        key = str(nid)
        name = (fr_map if lang == "fr" else en_map).get(key) or fr_map.get(key) or f"#{nid}"
        img_uri = _medal_icon_b64(nid)
        if img_uri:
            img_tag = (
                f"<img src='{img_uri}' width='24' height='24' "
                f"title='{html.escape(name)}' loading='lazy' />"
            )
        else:
            img_tag = f"<span title='{html.escape(name)}'>🏅</span>"
        chips.append(
            f"<div class='os-sb-medal-chip' title='{html.escape(name)}'>"
            f"{img_tag}"
            f"<span class='os-sb-medal-chip-cnt'>×{cnt}</span>"
            f"</div>"
        )
    title = html.escape(t("sb_detail_medals"))
    return (
        f"<div class='os-sb-detail-section'>"
        f"<span class='os-sb-det-title'>{title}</span>"
        f"<div class='os-sb-medal-row'>{''.join(chips)}</div>"
        f"</div>"
    )


def _build_enrichment_html(enrichment: dict[str, Any]) -> str:
    """Génère la section enrichissement DB joueur si disponible."""
    if not enrichment.get("has_db"):
        return ""

    perf = enrichment.get("performance_score")
    session = enrichment.get("session_label")
    awards = (enrichment.get("awards") or [])[:_MAX_AWARDS]

    has_any = perf is not None or session or awards
    if not has_any:
        return ""

    chips = []
    if perf is not None:
        try:
            chips.append(
                f"<span class='os-sb-enrich-chip'>"
                f"{html.escape(t('col_performance'))} {float(perf):.0f}"
                f"</span>"
            )
        except (ValueError, TypeError):
            pass
    if session:
        chips.append(
            f"<span class='os-sb-enrich-chip'>"
            f"📅 {html.escape(str(session))}"
            f"</span>"
        )
    for aw in awards:
        name = str(aw.get("name") or "?")
        cnt = aw.get("count", 0)
        chips.append(
            f"<span class='os-sb-enrich-chip'>{html.escape(name)} ×{cnt}</span>"
        )
    title = html.escape(t("sb_detail_player_db"))
    return (
        f"<div class='os-sb-detail-section'>"
        f"<span class='os-sb-det-title'>📊 {title}</span>"
        f"<div class='os-sb-enrichment'>{''.join(chips)}</div>"
        f"</div>"
    )


def _build_detail_cell_html(
    player: dict[str, Any],
    extra: dict[str, Any],
    n_cols: int,
    lang: str,
) -> str:
    """Construit la cellule de détail (colspan) pour un joueur."""
    stats_html = _build_stats_html(player)
    weapons_html = _build_weapons_html(extra.get("weapons", []), lang)
    medals_html = _build_medals_html(extra.get("medals", []), lang)
    enrichment_html = _build_enrichment_html(extra.get("enrichment", {}))

    inner = stats_html + weapons_html + medals_html + enrichment_html
    return (
        f"<td colspan='{n_cols}' class='os-sb-detail-td'>"
        f"<div class='os-sb-detail-grid'>{inner}</div>"
        f"</td>"
    )


# =============================================================================
# Rendu du tableau d'équipe avec lignes de détail intégrées
# =============================================================================


def _team_display_name(tid: Any) -> str:
    """Retourne le nom d'affichage d'une équipe."""
    if tid is None:
        return t("mv_team_unknown")
    try:
        return TEAM_MAP.get(int(tid), t("mv_team_n", n=tid))
    except (ValueError, TypeError):
        return t("mv_team_n", n=tid)


def build_team_table_html_with_details(  # noqa: PLR0913
    *,
    team_players: list[dict[str, Any]],
    tid: Any,
    my_team_id: Any,
    me_xu: str,
    mvp_xuid: str,
    lvp_xuid: str,
    extremes: dict[str, tuple[float, float]],
    sb_cols: list[tuple[str, str]],
    extra_data: dict[str, dict[str, Any]],
    match_id_key: str,
    team_index: int,
    lang: str,
    fmt_cell_fn: Any,
    cell_class_fn: Any,
    gamertag_link_fn: Any,
) -> str:
    """Génère le HTML complet d'une équipe avec lignes de détail inline.

    Args:
        team_players: Liste des joueurs de l'équipe.
        tid: team_id (None = inconnu).
        my_team_id: team_id du joueur courant.
        me_xu: XUID du joueur courant.
        mvp_xuid: XUID du MVP.
        lvp_xuid: XUID du LVP.
        extremes: Min/max par colonne (highlight couleur).
        sb_cols: Colonnes du scoreboard [(label, key), ...].
        extra_data: Données pré-chargées {xuid: {medals, weapons, enrichment}}.
        match_id_key: Préfixe unique pour les IDs HTML (ex: match_id[:8]).
        team_index: Indice de l'équipe (pour unicité des IDs).
        lang: Langue.
        fmt_cell_fn: Fonction de formatage des cellules.
        cell_class_fn: Fonction de calcul des classes CSS.
        gamertag_link_fn: Fonction générant un lien HTML gamertag.

    Returns:
        Chaîne HTML complète pour cette équipe.
    """
    raw_name = _team_display_name(tid)
    is_my_team = tid == my_team_id
    team_css_mod = "os-sb-team--mine" if is_my_team else "os-sb-team--enemy"
    team_label = t("mv_team_label", name=html.escape(raw_name))

    n_cols = len(sb_cols)
    th_cells = "".join(
        f"<th class='os-sb-th'>{html.escape(label)}</th>" for label, _ in sb_cols
    )
    thead = (
        f"<thead>"
        f"<tr><th class='os-sb-team {team_css_mod}' colspan='{n_cols}'>{team_label}</th></tr>"
        f"<tr>{th_cells}</tr>"
        f"</thead>"
    )

    body_rows: list[str] = []
    for p_idx, p in enumerate(team_players):
        p_xu = str(
            parse_xuid_input(str(p.get("xuid") or "").strip())
            or str(p.get("xuid") or "").strip()
        ).strip()
        is_me = bool(me_xu and p_xu and p_xu == me_xu)
        row_class = " os-sb-row--me" if is_me else ""
        if p_xu and p_xu == mvp_xuid:
            row_class += " os-sb-row--mvp"
        elif p_xu and p_xu == lvp_xuid:
            row_class += " os-sb-row--lvp"

        # ID unique pour la ligne de détail
        detail_id = f"os-sbdr-{match_id_key}-{team_index}-{p_idx}"

        # Cellules du scoreboard
        cell_parts: list[str] = []
        for col_idx, (_, key) in enumerate(sb_cols):
            css = cell_class_fn(key, p.get(key), extremes)
            raw_val = fmt_cell_fn(key, p.get(key))
            if key == "gamertag" and raw_val != "—" and not is_me:
                inner = (
                    f"<span class='os-sb-expand-btn'>&#9654;</span> "
                    + gamertag_link_fn(raw_val)
                )
            elif key == "gamertag":
                inner = (
                    f"<span class='os-sb-expand-btn'>&#9654;</span> "
                    + html.escape(raw_val)
                )
            else:
                inner = html.escape(raw_val)
            cell_parts.append(f"<td class='os-sb-td{css}'>{inner}</td>")

        cells = "".join(cell_parts)
        # Bot name check — bots ne sont pas clickable
        is_bot = p_xu.lower().startswith("bid(") if p_xu else False
        has_bot_name = get_bot_name(p_xu) is not None if p_xu else False
        clickable_cls = "" if is_bot or has_bot_name else " os-sb-row--clickable"
        onclick_attr = (
            f" onclick=\"osToggle('{detail_id}', this)\""
            if not (is_bot or has_bot_name)
            else ""
        )
        body_rows.append(
            f"<tr class='os-sb-row{row_class}{clickable_cls}'{onclick_attr}>{cells}</tr>"
        )

        # Ligne de détail (cachée par défaut)
        p_extra = extra_data.get(p_xu, {"medals": [], "weapons": [], "enrichment": {}})
        detail_cell = _build_detail_cell_html(p, p_extra, n_cols, lang)
        body_rows.append(
            f"<tr id='{detail_id}' class='os-sb-detail-row' style='display:none'>"
            f"{detail_cell}"
            f"</tr>"
        )

    return (
        "<div class='os-table-wrap os-sb-wrap'>"
        "<table class='os-table os-scoreboard os-sb-expandable'>"
        f"{thead}"
        f"<tbody>{''.join(body_rows)}</tbody>"
        "</table>"
        "</div>"
    )
