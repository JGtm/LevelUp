"""Fonctions de style pour les tableaux de la page Victoires/Défaites."""

from __future__ import annotations

import html as html_lib

from src.config import HALO_COLORS


def _to_float(v: object) -> float | None:
    """Convertit une valeur en float, ou None si impossible."""
    try:
        if v is None:
            return None
        x = float(v)
        return x if x == x else None
    except Exception:
        return None


def _styler_map(styler, func, subset):
    """Applique un style en mode compatible pandas 1.x et 2.x."""
    try:
        return styler.map(func, subset=subset)
    except AttributeError:
        return styler.applymap(func, subset=subset)


# ---------------------------------------------------------------------------
# Constantes couleur
# ---------------------------------------------------------------------------

_GREEN = str(getattr(HALO_COLORS, "green", "#2ECC71"))
_RED = str(getattr(HALO_COLORS, "red", "#E74C3C"))
_VIOLET = "#8E6CFF"


# ---------------------------------------------------------------------------
# Helpers HTML pour les cellules du tableau par carte
# (remplacent l'ancienne API pandas .style)
# ---------------------------------------------------------------------------


def map_name_cell_html(map_name: str | None) -> str:
    """Retourne une <td> avec tooltip thumbnail pour le nom de carte."""
    from src.ui.pages.match_table_html import map_thumb_url

    raw = str(map_name or "").strip() or "-"
    val = html_lib.escape(raw)
    url = map_thumb_url(map_name)
    if url:
        esc_url = html_lib.escape(url)
        return (
            f"<td><span class='map-hover'>{val}"
            f"<img class='map-popup' src='{esc_url}' alt='' /></span></td>"
        )
    return f"<td>{val}</td>"


def win_rate_cell_html(win_pct: float | None, loss_pct: float | None, display: str) -> str:
    """Retourne une <td> colorée pour le taux de victoire."""
    esc = html_lib.escape(display)
    if win_pct is None or loss_pct is None:
        return f"<td>{esc}</td>"
    if win_pct > loss_pct:
        return f"<td style='color:{_GREEN};font-weight:800'>{esc}</td>"
    if win_pct < loss_pct:
        return f"<td style='color:{_RED};font-weight:800'>{esc}</td>"
    return f"<td style='color:{_VIOLET};font-weight:800'>{esc}</td>"


def loss_rate_cell_html(win_pct: float | None, loss_pct: float | None, display: str) -> str:
    """Retourne une <td> colorée pour le taux de défaite (miroir du taux de victoire)."""
    esc = html_lib.escape(display)
    if win_pct is None or loss_pct is None:
        return f"<td>{esc}</td>"
    if win_pct > loss_pct:
        return f"<td style='color:{_RED};font-weight:800'>{esc}</td>"
    if win_pct < loss_pct:
        return f"<td style='color:{_GREEN};font-weight:800'>{esc}</td>"
    return f"<td style='color:{_VIOLET};font-weight:800'>{esc}</td>"


def ratio_cell_html(ratio_val: float | None, display: str) -> str:
    """Retourne une <td> colorée pour le ratio (vert >1, rouge <1, violet =1)."""
    esc = html_lib.escape(display)
    if ratio_val is None:
        return f"<td>{esc}</td>"
    if ratio_val > 1.0:
        return f"<td style='color:{_GREEN};font-weight:800'>{esc}</td>"
    if ratio_val < 1.0:
        return f"<td style='color:{_RED};font-weight:800'>{esc}</td>"
    return f"<td style='color:{_VIOLET};font-weight:800'>{esc}</td>"


def perf_cell_html(perf_val: object) -> str:
    """Retourne une <td> colorée pour la performance (même échelle que match_history)."""
    from src.ui.components.performance import get_score_class

    if perf_val is None:
        return "<td>-</td>"
    try:
        f = float(perf_val)  # type: ignore[arg-type]
        if f != f:  # NaN
            return "<td>-</td>"
        css_class = get_score_class(f)
        return f"<td class='{css_class}'>{f:.1f}</td>"
    except Exception:
        return f"<td>{html_lib.escape(str(perf_val))}</td>"


def plain_cell_html(val: object, fmt: str | None = None) -> str:
    """Retourne une <td> simple, avec formatage optionnel."""
    if val is None:
        return "<td>-</td>"
    if fmt:
        try:
            return f"<td>{fmt % float(val)}</td>"  # type: ignore[arg-type]
        except Exception:
            pass
    s = str(val).strip()
    return f"<td>{html_lib.escape(s) if s else '-'}</td>"
