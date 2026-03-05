"""Génération de tableaux HTML pour les résultats de matchs.

Module partagé entre match_history.py et explorer.py pour éviter
la duplication du rendu de tableaux de matchs (anti-pattern ≤2 copies).

Fournit aussi ``app_url()`` (URL interne) et ``gamertag_link()`` (lien joueur).
"""

from __future__ import annotations

import html as html_lib
import urllib.parse
from collections.abc import Callable

import polars as pl

from src.config import HALO_COLORS, OUTCOME_CODES
from src.ui.components.performance import get_score_class
from src.ui.i18n import t

# ---------------------------------------------------------------------------
# URL interne — centralisée (était dupliquée dans streamlit_app + match_history)
# ---------------------------------------------------------------------------


def app_url(page: str, **params: str) -> str:
    """Génère une URL interne vers une page de l'app.

    Args:
        page: Slug de la page cible (ex: ``"Explorer"``).
        **params: Paramètres query string supplémentaires.

    Returns:
        URL relative (ex: ``/?page=Explorer&match_id=abc``).
    """
    qp: dict[str, str] = {"page": page, **{k: v for k, v in params.items() if v}}
    return "/?" + urllib.parse.urlencode(qp)


def gamertag_link(gt: str) -> str:
    """Génère un lien HTML vers la page Explorer pour un gamertag.

    Le style est sobre (souligné, couleur héritée) pour s'intégrer partout.

    Args:
        gt: Gamertag du joueur.

    Returns:
        Chaîne HTML ``<a>`` cliquable.
    """
    href = app_url("Explorer", gamertag=gt)
    esc = html_lib.escape(gt)
    return (
        f"<a href='{html_lib.escape(href)}' target='_self' "
        f"style='text-decoration:underline;color:inherit;'>{esc}</a>"
    )


# ---------------------------------------------------------------------------
# Helpers de formatage (réutilisables)
# ---------------------------------------------------------------------------


def fmt_value(v: object) -> str:
    """Formate une valeur générique pour affichage tableau."""
    if v is None:
        return "-"
    s = str(v)
    return s if s.strip() else "-"


def fmt_float(v: object, decimals: int = 2) -> str:
    """Formate un float avec décimales."""
    if v is None:
        return "-"
    try:
        f = float(v)  # type: ignore[arg-type]
        return "-" if f != f else f"{f:.{decimals}f}"
    except Exception:
        return "-"


def fmt_mmr_int(v: object) -> str:
    """Formate un MMR en entier arrondi."""
    if v is None:
        return "-"
    try:
        f = float(v)  # type: ignore[arg-type]
        return "-" if f != f else str(int(round(f)))
    except Exception:
        return "-"


# ---------------------------------------------------------------------------
# Styles de cellules
# ---------------------------------------------------------------------------

_COLORS = HALO_COLORS.as_dict()


def outcome_style(outcome: object, label: str) -> str:
    """Retourne le style CSS inline pour un outcome (victoire/défaite/égalité)."""
    try:
        code = int(outcome)  # type: ignore[arg-type]
        if code == int(OUTCOME_CODES.WIN):
            return f"color:{_COLORS['green']}; font-weight:800"
        if code == int(OUTCOME_CODES.LOSS):
            return f"color:{_COLORS['red']}; font-weight:800"
        if code in (int(OUTCOME_CODES.TIE), int(OUTCOME_CODES.NO_FINISH)):
            return f"color:{_COLORS['violet']}; font-weight:800"
    except Exception:
        pass
    return "opacity:0.92"


def mmr_gap_style(v: object) -> str:
    """Retourne le style CSS inline pour un delta MMR."""
    try:
        f = float(v)  # type: ignore[arg-type]
        if f != f:
            return ""
        if f > 0:
            return f"color:{_COLORS['green']}; font-weight:600"
        if f < 0:
            return f"color:{_COLORS['red']}; font-weight:600"
    except Exception:
        pass
    return ""


# ---------------------------------------------------------------------------
# Construction du tableau HTML
# ---------------------------------------------------------------------------

# Colonnes par défaut pour le tableau de matchs
_DEFAULT_COLUMNS: list[tuple[Callable[[], str], str]] = []


def _build_default_columns() -> list[tuple[str, str]]:
    """Construit la liste (label, clé) des colonnes standard."""
    return [
        (t("col_start_date"), "start_time_fr"),
        (t("col_map"), "map_name"),
        (t("col_playlist"), "playlist_fr"),
        (t("col_mode"), "mode_ui"),
        (t("col_result"), "outcome_label"),
        (t("col_score"), "score"),
        (t("mv_performance"), "performance"),
        (t("col_mmr_team"), "team_mmr"),
        (t("col_mmr_enemy"), "enemy_mmr"),
        (t("col_mmr_gap"), "delta_mmr"),
        (t("col_kda"), "kda"),
        (t("col_kills"), "kills"),
        (t("col_deaths"), "deaths"),
        (t("col_max_spree"), "max_killing_spree"),
        (t("col_headshots"), "headshot_kills"),
        (t("col_avg_life"), "average_life_mmss"),
        (t("col_assists"), "assists"),
        (t("col_accuracy"), "accuracy"),
        (t("col_ratio"), "ratio"),
    ]


def render_match_table_html(
    view: pl.DataFrame,
    *,
    waypoint_player: str | None = None,
    header_css_class: str = "",
    page_slug: str = "Explorer",
    max_rows: int = 250,
) -> str:
    """Génère le HTML d'un tableau de matchs.

    Args:
        view: DataFrame avec les colonnes enrichies (start_time_fr, outcome_label, etc.).
        waypoint_player: Nom Waypoint pour les liens Halo Waypoint (None = pas de colonne).
        header_css_class: Classe CSS additionnelle pour le ``<thead>`` (ex: couleur d'équipe).
        page_slug: Slug de la page cible pour le lien "Ouvrir".
        max_rows: Nombre maximum de lignes à afficher.

    Returns:
        Chaîne HTML complète (``<div class='os-table-wrap'>…</div>``).
    """
    view = view.sort("start_time", descending=True).head(max_rows)
    cols = _build_default_columns()
    lbl_open = t("btn_open")

    # En-têtes
    head_cells = [f"<th>{html_lib.escape(lbl_open)}</th>"]
    if waypoint_player:
        head_cells.append("<th>Waypoint</th>")
    head_cells.extend(f"<th>{html_lib.escape(h)}</th>" for h, _ in cols)
    thead_class = f" class='{html_lib.escape(header_css_class)}'" if header_css_class else ""
    head = f"<thead{thead_class}><tr>{''.join(head_cells)}</tr></thead>"

    # Corps
    rows_html = [
        _render_row(r, cols, lbl_open, waypoint_player, page_slug) for r in view.to_dicts()
    ]
    body = "<tbody>" + "".join(rows_html) + "</tbody>"

    return f"<div class='os-table-wrap'><table class='os-table'>{head}{body}</table></div>"


def _render_row(
    r: dict,
    cols: list[tuple[str, str]],
    lbl_open: str,
    waypoint_player: str | None,
    page_slug: str,
) -> str:
    """Génère une ligne ``<tr>`` pour un match."""
    mid = str(r.get("match_id") or "").strip()
    url = app_url(page_slug, match_id=mid)
    match_link = (
        f"<a href='{html_lib.escape(url)}' target='_self'>{html_lib.escape(lbl_open)}</a>"
        if mid
        else "-"
    )
    tds = [f"<td>{match_link}</td>"]

    if waypoint_player:
        hw = f"https://www.halowaypoint.com/halo-infinite/players/{waypoint_player.strip()}/matches/{mid}"
        tds.append(
            f"<td><a href='{html_lib.escape(hw)}' target='_blank' rel='noopener'>"
            f"{html_lib.escape(lbl_open)}</a></td>"
        )

    outcome_code = r.get("outcome")
    for _h, key in cols:
        tds.append(_render_cell(r, key, outcome_code))

    return "<tr>" + "".join(tds) + "</tr>"


def _render_cell(r: dict, key: str, outcome_code: object) -> str:
    """Génère une cellule ``<td>`` formatée selon le type de colonne."""
    if key == "outcome_label":
        val = fmt_value(r.get(key))
        style = outcome_style(outcome_code, val)
        return f"<td style='{style}'>{html_lib.escape(val)}</td>"

    if key == "performance":
        perf_val = r.get("performance")
        css_class = get_score_class(perf_val)
        display = fmt_mmr_int(perf_val) if perf_val is not None else "-"
        return f"<td class='{css_class}'>{html_lib.escape(display)}</td>"

    if key in ("team_mmr", "enemy_mmr"):
        return f"<td>{html_lib.escape(fmt_mmr_int(r.get(key)))}</td>"

    if key == "delta_mmr":
        val = r.get(key)
        style = mmr_gap_style(val)
        try:
            display = f"{int(round(float(val))):+d}"  # type: ignore[arg-type]
        except Exception:
            display = "-"
        return f"<td style='{style}'>{html_lib.escape(display)}</td>"

    if key in ("kda", "accuracy", "ratio"):
        return f"<td>{html_lib.escape(fmt_float(r.get(key)))}</td>"

    return f"<td>{html_lib.escape(fmt_value(r.get(key)))}</td>"
