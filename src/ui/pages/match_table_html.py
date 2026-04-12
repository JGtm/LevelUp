"""Génération de tableaux HTML pour les résultats de matchs.

Module partagé entre match_history.py et explorer.py pour éviter
la duplication du rendu de tableaux de matchs (anti-pattern ≤2 copies).

Fournit aussi ``app_url()`` (URL interne) et ``gamertag_link()`` (lien joueur).
"""

from __future__ import annotations

import functools
import html as html_lib
import logging
import urllib.parse
from collections.abc import Callable
from pathlib import Path

_log = logging.getLogger(__name__)

import polars as pl

from src.config import HALO_COLORS, OUTCOME_CODES, get_repo_root
from src.ui.components.performance import get_score_class
from src.ui.i18n import get_lang, t

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


def win_rate_style(v: object) -> str:
    """Retourne le style CSS inline pour un taux de victoire cumulatif."""
    try:
        f = float(v)  # type: ignore[arg-type]
        if f != f:
            return ""
        if f >= 55.0:
            return f"color:{_COLORS['green']}; font-weight:700"
        if f <= 45.0:
            return f"color:{_COLORS['red']}; font-weight:700"
        return f"color:{_COLORS['cyan']}; font-weight:700"
    except Exception:
        return ""


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
# Miniature de carte (tooltip hover)
# ---------------------------------------------------------------------------


@functools.cache
def _build_map_url_index() -> dict[str, str]:
    """Scanne static/maps/ une fois et retourne {stem_lower: url_statique}."""
    import unicodedata

    maps_dir = Path(get_repo_root()) / "static" / "maps"
    index: dict[str, str] = {}
    if not maps_dir.exists():
        _log.warning(
            "Répertoire static/maps/ introuvable — thumbnails de cartes désactivés (%s)", maps_dir
        )
        return index
    for f in maps_dir.iterdir():
        if f.is_file() and f.suffix.lower() in (".jpg", ".jpeg", ".png", ".webp"):
            stem = unicodedata.normalize("NFC", f.stem.lower())
            url = f"/app/static/maps/{f.name}"
            index[stem] = url
            index[stem.replace(" ", "_")] = url
    _log.debug("Index thumbnails cartes : %d entrées chargées depuis %s", len(index), maps_dir)
    return index


def map_thumb_url(map_name: str | None) -> str | None:
    """Retourne l'URL statique Streamlit de la miniature pour un nom de carte (EN)."""
    if not map_name:
        return None
    key = str(map_name).strip().lower()
    idx = _build_map_url_index()
    return idx.get(key) or idx.get(key.replace(" ", "_"))


@functools.cache
def _build_map_id_index() -> dict[str, str]:
    """Construit {asset_id: url} en joignant asset_translations (lang=en-US) + index fichiers."""
    import unicodedata

    from src.utils.db import duckdb_read_only

    name_idx = _build_map_url_index()
    if not name_idx:
        return {}
    meta_path = Path(get_repo_root()) / "data" / "warehouse" / "metadata.duckdb"
    if not meta_path.exists():
        return {}
    try:
        with duckdb_read_only(meta_path) as conn:
            rows = conn.execute(
                "SELECT asset_id, name FROM asset_translations"
                " WHERE asset_type='map' AND lang='en-US' AND name IS NOT NULL"
            ).fetchall()
    except Exception as exc:  # noqa: BLE001
        _log.warning("Impossible de charger l'index map_id→thumbnail : %s", exc)
        return {}
    idx: dict[str, str] = {}
    for asset_id, name_en in rows:
        key = unicodedata.normalize("NFC", name_en.lower())
        url = name_idx.get(key) or name_idx.get(key.replace(" ", "_"))
        if url:
            idx[str(asset_id)] = url
    return idx


def map_thumb_url_by_id(map_id: str | None) -> str | None:
    """Retourne l'URL statique de la miniature via asset_id (indépendant de la langue)."""
    if not map_id:
        return None
    return _build_map_id_index().get(str(map_id).strip())


# ---------------------------------------------------------------------------
# Construction du tableau HTML
# ---------------------------------------------------------------------------

# Colonnes par défaut pour le tableau de matchs
_DEFAULT_COLUMNS: list[tuple[Callable[[], str], str]] = []


def _build_default_columns() -> list[tuple[str, str]]:
    """Construit la liste (label, clé) des colonnes standard."""
    return [
        (t("col_start_date"), "start_time_fr"),
        (t("col_map"), "map_ui"),
        (t("col_playlist"), "playlist_fr"),
        (t("col_mode"), "mode_ui"),
        (t("col_result"), "outcome_label"),
        (t("col_win_rate_hist"), "win_rate_hist"),
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


def render_match_table_html(  # noqa: PLR0913 — kwargs keyword-only, interface stable
    view: pl.DataFrame,
    *,
    waypoint_player: str | None = None,
    header_css_class: str = "",
    page_slug: str = "Explorer",
    page_params: dict[str, str] | None = None,
    max_rows: int = 250,
    start_row: int = 0,
    hide_empty_cols: bool = False,
) -> str:
    """Génère le HTML d'un tableau de matchs.

    Args:
        view: DataFrame avec les colonnes enrichies (start_time_fr, outcome_label, etc.).
        waypoint_player: Nom Waypoint pour les liens Halo Waypoint (None = pas de colonne).
        header_css_class: Classe CSS additionnelle pour le ``<thead>`` (ex: couleur d'équipe).
        page_slug: Slug de la page cible pour le lien "Ouvrir".
        page_params: Paramètres query string supplémentaires pour les liens internes.
        max_rows: Nombre maximum de lignes à afficher.
        start_row: Décalage 0-based à appliquer après tri pour paginer le tableau.
        hide_empty_cols: Si True, masque les colonnes entièrement nulles (inconnus).

    Returns:
        Chaîne HTML complète (``<div class='os-table-wrap'>…</div>``).
    """
    start_row = max(0, int(start_row or 0))
    view = view.sort("start_time", descending=True).slice(start_row, max_rows)
    cols = _build_default_columns()
    if hide_empty_cols:
        cols = [
            (lbl, key)
            for lbl, key in cols
            if key not in view.columns or view[key].drop_nulls().len() > 0
        ]
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
        _render_row(r, cols, lbl_open, waypoint_player, page_slug, page_params)
        for r in view.to_dicts()
    ]
    body = "<tbody>" + "".join(rows_html) + "</tbody>"

    return (
        "<div class='os-table-wrap os-table-wrap--map-hover'>"
        f"<table class='os-table'>{head}{body}</table>"
        "</div>"
    )


def _render_row(  # noqa: PLR0913
    r: dict,
    cols: list[tuple[str, str]],
    lbl_open: str,
    waypoint_player: str | None,
    page_slug: str,
    page_params: dict[str, str] | None,
) -> str:
    """Génère une ligne ``<tr>`` pour un match."""
    mid = str(r.get("match_id") or "").strip()
    extra_params = page_params or {}
    url = app_url(page_slug, match_id=mid, **extra_params)
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

    if key == "win_rate_hist":
        val = r.get(key)
        total = r.get("win_rate_hist_total")
        style = win_rate_style(val)
        try:
            pct = f"{int(round(float(val)))}%"  # type: ignore[arg-type]
            if total is not None:
                n_label = "matches" if get_lang() == "en" else "matchs"
                pct = f"{pct} ({int(total)} {n_label})"
        except Exception:
            pct = "-"
        return f"<td style='{style}'>{html_lib.escape(pct)}</td>"

    if key in ("map_name", "map_ui"):
        # Afficher map_ui (traduit) si disponible, sinon map_name (EN fallback)
        display_name = r.get("map_ui") or r.get("map_name")
        val = fmt_value(display_name)
        url = map_thumb_url_by_id(r.get("map_id")) or map_thumb_url(r.get("map_name"))
        if url:
            esc_url = html_lib.escape(url)
            esc_val = html_lib.escape(val)
            return (
                f"<td><span class='map-hover'>{esc_val}"
                f"<img class='map-popup' src='{esc_url}' alt='' /></span></td>"
            )
        return f"<td>{html_lib.escape(val)}</td>"

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
