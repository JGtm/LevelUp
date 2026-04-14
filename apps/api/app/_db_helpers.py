"""Helpers SQL centralisés pour les services API.

Regroupe les fonctions utilitaires DuckDB dupliquées dans 5+ services :
- ``resolve_xuid`` : extrait le xuid depuis ``sync_meta``
- ``has_mv_player_matches`` : détecte la vue matérialisée ``shared.mv_player_matches``
- ``build_match_source_sql`` : génère le sous-SELECT (mv ou fallback)
- ``add_display_columns`` : ajoute ``map_ui``, ``mode_ui``, ``playlist_ui``
- Constantes ``OUTCOME_*``
"""

from __future__ import annotations

import re
from enum import IntEnum

import polars as pl

# ---------------------------------------------------------------------------
# Constantes Outcome
# ---------------------------------------------------------------------------


class Outcome(IntEnum):
    """Codes numériques des résultats de match Halo Infinite."""

    TIE = 1
    WIN = 2
    LOSS = 3
    DNF = 4


OUTCOME_LABELS: dict[int, str] = {
    Outcome.WIN: "Victoire",
    Outcome.LOSS: "Défaite",
    Outcome.TIE: "Égalité",
    Outcome.DNF: "Abandon",
}

OUTCOME_TONES: dict[int, str] = {
    Outcome.WIN: "win",
    Outcome.LOSS: "loss",
    Outcome.TIE: "tie",
    Outcome.DNF: "dnf",
}

FMT_DATETIME_FR = "%d/%m/%Y %H:%M"


# ---------------------------------------------------------------------------
# Helpers SQL bas-niveau
# ---------------------------------------------------------------------------


def resolve_xuid(conn) -> str:  # type: ignore[no-untyped-def]
    """Extrait le xuid du joueur depuis ``sync_meta``.

    Args:
        conn: Connexion DuckDB ouverte sur la DB joueur.

    Returns:
        xuid (str) ou chaîne vide si introuvable.
    """
    try:
        row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        return str(row[0]).strip() if row else ""
    except Exception:
        return ""


def has_mv_player_matches(conn) -> bool:  # type: ignore[no-untyped-def]
    """Vérifie si ``shared.mv_player_matches`` est disponible.

    Args:
        conn: Connexion DuckDB avec shared attaché.
    """
    try:
        conn.execute("SELECT 1 FROM shared.mv_player_matches LIMIT 0")
        return True
    except Exception:
        return False


def build_match_source_sql(has_mv: bool) -> str:
    """Retourne le sous-SELECT approprié selon la vue matérialisée.

    Les appelants doivent ajouter leur propre alias :
    ``FROM {source_sql} ms``.

    Args:
        has_mv: Résultat de ``has_mv_player_matches``.

    Returns:
        Clause SQL ``(...)`` avec un ``?`` paramètre pour le xuid.
    """
    if has_mv:
        return """(
            SELECT match_id, start_time, map_id, map_name, map_name_fr,
                   pair_name, pair_name_fr, playlist_name, playlist_name_fr,
                   is_firefight, is_ranked
            FROM shared.mv_player_matches
            WHERE xuid = ?
        )"""
    return """(
        SELECT r.match_id, r.start_time, r.map_id, r.map_name,
               NULL AS map_name_fr, r.pair_name, NULL AS pair_name_fr,
               r.playlist_name, NULL AS playlist_name_fr,
               COALESCE(r.is_firefight, FALSE) AS is_firefight,
               COALESCE(r.is_ranked, FALSE)    AS is_ranked
        FROM shared.match_registry r
        JOIN shared.match_participants p ON r.match_id = p.match_id
        WHERE p.xuid = ?
    )"""


# ---------------------------------------------------------------------------
# Enrichissement colonnes i18n
# ---------------------------------------------------------------------------

_STRIP_RE_ON = re.compile(r" on .+$", re.IGNORECASE)
_STRIP_RE_FORGE = re.compile(r"\s*-\s*Forge\b", re.IGNORECASE)
_STRIP_RE_RANKED = re.compile(r"\s*-\s*Ranked\b", re.IGNORECASE)


def _strip_mode_suffix(s: str | None) -> str | None:
    """Supprime ``' on NomCarte'`` et variantes Forge/Ranked d'un mode."""
    if not s:
        return None
    s = str(s).strip()
    if " on " in s:
        s = s.split(" on ", 1)[0].strip()
    s = _STRIP_RE_FORGE.sub("", s).strip()
    s = _STRIP_RE_RANKED.sub("", s).strip()
    return s or None


def add_display_columns(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute ``map_ui``, ``mode_ui``, ``playlist_ui`` au DataFrame.

    Reproduit la logique i18n (coalesce FR/EN) sans import Streamlit.
    """
    if df.is_empty():
        return df

    exprs: list[pl.Expr] = []

    if "map_ui" not in df.columns and "map_name" in df.columns:
        src = (
            pl.coalesce([pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)])
            if "map_name_fr" in df.columns
            else pl.col("map_name").cast(pl.Utf8)
        )
        exprs.append(src.alias("map_ui"))

    if "mode_ui" not in df.columns and "pair_name" in df.columns:
        src = (
            pl.coalesce([pl.col("pair_name_fr").cast(pl.Utf8), pl.col("pair_name").cast(pl.Utf8)])
            if "pair_name_fr" in df.columns
            else pl.col("pair_name").cast(pl.Utf8)
        )
        exprs.append(src.map_elements(_strip_mode_suffix, return_dtype=pl.Utf8).alias("mode_ui"))

    if "playlist_ui" not in df.columns and "playlist_name" in df.columns:
        src = (
            pl.coalesce(
                [pl.col("playlist_name_fr").cast(pl.Utf8), pl.col("playlist_name").cast(pl.Utf8)]
            )
            if "playlist_name_fr" in df.columns
            else pl.col("playlist_name").cast(pl.Utf8)
        )
        exprs.append(src.alias("playlist_ui"))

    return df.with_columns(exprs) if exprs else df
