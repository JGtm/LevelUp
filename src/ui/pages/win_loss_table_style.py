"""Fonctions de style pour les tableaux de la page Victoires/Défaites."""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import pandas as pd

from src.config import HALO_COLORS
from src.ui.i18n import t


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


def _style_map_table_row(row: pd.Series) -> pd.Series:
    """Style les lignes du tableau par carte."""
    import pandas as pd  # requis pour l'API .style de Streamlit

    green = str(getattr(HALO_COLORS, "green", "#2ECC71"))
    red = str(getattr(HALO_COLORS, "red", "#E74C3C"))
    violet = "#8E6CFF"

    col_win = t("wl_col_win_rate")
    col_loss = t("wl_col_loss_rate")
    col_ratio = t("wl_col_ratio")

    win_pct = _to_float(row.get(col_win))
    loss_pct = _to_float(row.get(col_loss))
    ratio_val = _to_float(row.get(col_ratio))

    styles: dict[str, str] = {str(c): "" for c in row.index}

    if win_pct is not None and loss_pct is not None:
        if win_pct > loss_pct:
            styles[col_win] = f"color: {green}; font-weight: 800;"
            styles[col_loss] = f"color: {red}; font-weight: 800;"
        elif win_pct < loss_pct:
            styles[col_win] = f"color: {red}; font-weight: 800;"
            styles[col_loss] = f"color: {green}; font-weight: 800;"
        else:
            styles[col_win] = f"color: {violet}; font-weight: 800;"
            styles[col_loss] = f"color: {violet}; font-weight: 800;"

    if ratio_val is not None:
        if ratio_val > 1.0:
            styles[col_ratio] = f"color: {green}; font-weight: 800;"
        elif ratio_val < 1.0:
            styles[col_ratio] = f"color: {red}; font-weight: 800;"
        else:
            styles[col_ratio] = f"color: {violet}; font-weight: 800;"

    return pd.Series(styles)
