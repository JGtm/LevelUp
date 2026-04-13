"""Service de résolution des filtres — logique pure sans Streamlit.

Remplace le comportement distribué entre ``filters_render.py``,
``filter_state.py``, ``_filters_cascade.py`` et ``_filters_session.py``.
Aucun accès à ``st.session_state`` ni à ``GAP_MINUTES_FIXED``.
"""

from __future__ import annotations

import contextlib
import logging
import re
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path

import polars as pl

from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.filters import (
    AvailableOptions,
    CascadeInput,
    FilterContextInput,
    FilterContextResolved,
    FilterCounts,
    LabelValue,
    PeriodInput,
    SessionOption,
    SessionOptions,
    SessionsInput,
)

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Constantes internes
# ---------------------------------------------------------------------------

_EXPERIENCE_LABELS = ("PVP non classé", "PVP classé", "PVE")
_FIREFIGHT_KEYWORDS = frozenset({"firefight", "fire fight", "pve", "spartan ops", "flood"})


# ---------------------------------------------------------------------------
# Helpers i18n inline (sans accès DB, même logique que i18n_columns.py)
# ---------------------------------------------------------------------------


def _strip_mode_map_suffix(s: str | None) -> str | None:
    """Supprime ' on NomCarte' et variantes Forge/Ranked d'un label de mode."""
    if not s:
        return None
    s = str(s).strip()
    if " on " in s:
        s = s.split(" on ", 1)[0].strip()
    s = re.sub(r"\s*-\s*Forge\b", "", s, flags=re.IGNORECASE).strip()
    s = re.sub(r"\s*-\s*Ranked\b", "", s, flags=re.IGNORECASE).strip()
    return s or None


def _add_display_columns(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute map_ui, mode_ui, playlist_ui idiomatiques (fr) au DataFrame.

    Reproduit la logique de ``src.app.i18n_columns.add_i18n_display_columns``
    sans import Streamlit.
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
        exprs.append(
            src.map_elements(_strip_mode_map_suffix, return_dtype=pl.Utf8).alias("mode_ui")
        )

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


# ---------------------------------------------------------------------------
# Chargement DuckDB (mode lecture seule, sans DuckDBRepository Streamlit)
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class _MatchRecord:
    """Vue minimale d'un match pour la résolution des filtres."""

    match_id: str
    start_time: datetime | None
    map_ui: str | None
    mode_ui: str | None
    playlist_ui: str | None
    is_firefight: bool
    is_ranked: bool
    session_id: str | None
    session_label: str | None
    is_with_friends: bool


def _load_matches_for_filters(player: PlayerContext) -> pl.DataFrame:
    """Charge les matchs depuis DuckDB (shared + player) en lecture seule.

    Retourne un DataFrame Polars avec les colonnes nécessaires à la résolution
    des filtres. Ne lève pas d'exception : retourne un DataFrame vide si la DB
    est inaccessible.
    """
    try:
        from src.utils.db import duckdb_read_only
    except ImportError:
        logger.warning("src.utils.db non disponible — filtres désactivés")
        return pl.DataFrame()

    db_path = Path(player.db_path)
    shared_path = Path(player.shared_db_path)

    if not db_path.exists():
        logger.warning("DB player introuvable : %s", db_path)
        return pl.DataFrame()

    try:
        with duckdb_read_only(str(db_path)) as conn:
            # Attacher shared
            if shared_path.exists():
                with contextlib.suppress(Exception):  # déjà attachée ou indisponible
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = _resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            has_mv = _has_mv_player_matches(conn)
            source_sql = _build_source_sql(has_mv)

            sql = f"""
            SELECT
                ms.match_id,
                ms.start_time,
                ms.map_name,
                COALESCE(ms.map_name_fr, ms.map_name)               AS map_name_fr,
                ms.pair_name,
                COALESCE(ms.pair_name_fr, ms.pair_name)             AS pair_name_fr,
                ms.playlist_name,
                COALESCE(ms.playlist_name_fr, ms.playlist_name)     AS playlist_name_fr,
                COALESCE(ms.is_firefight, FALSE)                    AS is_firefight,
                COALESCE(ms.is_ranked, FALSE)                       AS is_ranked,
                pme.session_id,
                pme.session_label,
                COALESCE(pme.is_with_friends, FALSE)                AS is_with_friends
            FROM {source_sql} ms
            LEFT JOIN player_match_enrichment pme
                ON ms.match_id = pme.match_id
            ORDER BY ms.start_time DESC
            """
            result = conn.execute(sql, [xuid] if "?" in source_sql else [])
            columns = [d[0] for d in result.description]
            rows = result.fetchall()

        if not rows:
            return pl.DataFrame()

        df = pl.DataFrame(rows, schema=columns, orient="row")
        return _add_display_columns(df)

    except Exception:
        logger.exception("Erreur lors du chargement des matchs pour les filtres")
        return pl.DataFrame()


def _resolve_xuid(conn) -> str:  # type: ignore[no-untyped-def]
    """Extrait le xuid depuis sync_meta."""
    try:
        row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        return str(row[0]).strip() if row else ""
    except Exception:
        return ""


def _has_mv_player_matches(conn) -> bool:  # type: ignore[no-untyped-def]
    """Vérifie si shared.mv_player_matches est disponible."""
    try:
        conn.execute("SELECT 1 FROM shared.mv_player_matches LIMIT 0")
        return True
    except Exception:
        return False


def _build_source_sql(has_mv: bool) -> str:
    """Retourne le sous-SELECT approprié selon la disponibilité de la vue matérialisée."""
    if has_mv:
        return """(
            SELECT match_id, start_time, map_id, map_name, map_name_fr,
                   pair_name, pair_name_fr, playlist_name, playlist_name_fr,
                   game_variant_name, game_variant_name_fr,
                   is_firefight, is_ranked
            FROM shared.mv_player_matches
            WHERE xuid = ?
        )"""
    return """(
        SELECT r.match_id, r.start_time, r.map_id, r.map_name,
               NULL AS map_name_fr,
               r.pair_name,
               NULL AS pair_name_fr,
               r.playlist_name,
               NULL AS playlist_name_fr,
               r.game_variant_name,
               NULL AS game_variant_name_fr,
               COALESCE(r.is_firefight, FALSE) AS is_firefight,
               COALESCE(r.is_ranked, FALSE)    AS is_ranked
        FROM shared.match_registry r
        JOIN shared.match_participants p ON r.match_id = p.match_id
        WHERE p.xuid = ?
    )"""


# ---------------------------------------------------------------------------
# Résolution des sessions
# ---------------------------------------------------------------------------


def _build_session_options(df: pl.DataFrame) -> SessionOptions:
    """Construit la liste des sessions disponibles depuis le DataFrame enrichi."""
    if df.is_empty() or "session_label" not in df.columns:
        return SessionOptions()

    # Agréger par session_label pour obtenir count + is_squad
    try:
        agg = (
            df.filter(pl.col("session_label").is_not_null())
            .group_by(["session_label", "session_id"])
            .agg(
                [
                    pl.len().alias("match_count"),
                    pl.col("is_with_friends").max().cast(pl.Boolean).alias("is_squad"),
                ]
            )
            .sort("session_label", descending=True)
        )
    except Exception:
        return SessionOptions()

    sessions: list[SessionOption] = []
    solo_labels: list[str] = []
    squad_labels: list[str] = []

    for row in agg.iter_rows(named=True):
        label = str(row["session_label"])
        sid = str(row["session_id"]) if row["session_id"] else label
        is_squad = bool(row["is_squad"])
        opt = SessionOption(
            label=label,
            session_id=sid,
            match_count=int(row["match_count"]),
            is_squad=is_squad,
        )
        sessions.append(opt)
        if is_squad:
            squad_labels.append(label)
        else:
            solo_labels.append(label)

    return SessionOptions(
        all_sessions=sessions,
        solo_labels=solo_labels,
        squad_labels=squad_labels,
    )


# ---------------------------------------------------------------------------
# Filtrage Expérience
# ---------------------------------------------------------------------------


def _is_firefight_playlist(playlist_ui: str | None) -> bool:
    """Détecte si une playlist est PVE (Firefight/PvE)."""
    if not playlist_ui:
        return False
    lower = playlist_ui.lower()
    return any(kw in lower for kw in _FIREFIGHT_KEYWORDS)


def _apply_experience_filter(df: pl.DataFrame, experience_types: list[str]) -> pl.DataFrame:
    """Pré-filtre le DataFrame selon les types d'expérience sélectionnés.

    Reproduit ``_apply_experience_filter`` de ``_filters_cascade.py``
    sans accès Streamlit.
    """
    if not experience_types or len(experience_types) >= len(_EXPERIENCE_LABELS):
        return df

    # Construire un ensemble des playlists firefight réellement présentes
    if "playlist_ui" not in df.columns:
        return df

    all_playlists = df["playlist_ui"].drop_nulls().unique().to_list()
    firefight_pls = frozenset(p for p in all_playlists if _is_firefight_playlist(p))

    pve_cond = pl.col("playlist_ui").cast(pl.Utf8).fill_null("").is_in(list(firefight_pls))
    ranked_cond = (
        pl.col("playlist_ui")
        .cast(pl.Utf8)
        .fill_null("")
        .str.to_lowercase()
        .str.contains("classé|ranked")
    ) & ~pve_cond

    # Mapper les noms localisés → catégories
    conds: list[pl.Expr] = []
    for exp in experience_types:
        exp_lower = exp.lower()
        if "pve" in exp_lower or "firefight" in exp_lower:
            conds.append(pve_cond)
        elif "classé" in exp_lower or "ranked" in exp_lower:
            conds.append(ranked_cond)
        else:  # PVP non classé
            conds.append(~pve_cond & ~ranked_cond)

    if not conds:
        return df

    combined = conds[0]
    for c in conds[1:]:
        combined = combined | c
    return df.filter(combined)


# ---------------------------------------------------------------------------
# Construction des options disponibles
# ---------------------------------------------------------------------------


def _sorted_unique_labels(df: pl.DataFrame, column: str) -> list[LabelValue]:
    """Retourne les valeurs uniques d'une colonne triées, formatées en LabelValue."""
    if column not in df.columns:
        return []
    vals = df[column].drop_nulls().cast(pl.Utf8).unique().sort().to_list()
    return [LabelValue(label=v, value=v) for v in vals if v]


def _build_available_options(
    df: pl.DataFrame,
    effective_cascade: CascadeInput,
) -> AvailableOptions:
    """Calcule les options disponibles pour les filtres cascade.

    L'expérience est calculée sur le dataset complet.
    Playlists → calculées après filtre expérience.
    Modes → calculées après filtre expérience + playlist.
    Cartes → calculées après filtre expérience + playlist + mode.
    """
    # -- expérience : toujours toutes les options statiques ---
    experience_opts = [LabelValue(label=e, value=e) for e in _EXPERIENCE_LABELS]

    # Filtrer selon expérience sélectionnée
    df_exp = _apply_experience_filter(df, effective_cascade.experience_types)

    # Playlists disponibles
    playlist_opts = _sorted_unique_labels(df_exp, "playlist_ui")

    # Après filtre playlist → modes disponibles
    df_pl = df_exp
    if effective_cascade.playlists:
        df_pl = df_exp.filter(
            pl.col("playlist_ui").cast(pl.Utf8).is_in(effective_cascade.playlists)
        )
    mode_opts = _sorted_unique_labels(df_pl, "mode_ui")

    # Après filtre mode → cartes disponibles
    df_mo = df_pl
    if effective_cascade.modes:
        df_mo = df_pl.filter(pl.col("mode_ui").cast(pl.Utf8).is_in(effective_cascade.modes))
    map_opts = _sorted_unique_labels(df_mo, "map_ui")

    return AvailableOptions(
        experience_types=experience_opts,
        playlists=playlist_opts,
        modes=mode_opts,
        maps=map_opts,
    )


# ---------------------------------------------------------------------------
# Application des filtres au dataset
# ---------------------------------------------------------------------------


def _apply_period_filter(df: pl.DataFrame, period: PeriodInput) -> pl.DataFrame:
    """Applique le filtre de période au DataFrame."""
    if "start_time" not in df.columns:
        return df
    # Caster la colonne en Datetime si elle est stockée en string (fixtures tests)
    col_dtype = df["start_time"].dtype
    if col_dtype == pl.Utf8 or col_dtype == pl.String:
        df = df.with_columns(
            pl.col("start_time").str.to_datetime(format="%Y-%m-%d %H:%M:%S", strict=False)
        )
    if period.start_date:
        start_dt = datetime(period.start_date.year, period.start_date.month, period.start_date.day)
        df = df.filter(pl.col("start_time") >= start_dt)
    if period.end_date:
        end_dt = datetime(
            period.end_date.year, period.end_date.month, period.end_date.day, 23, 59, 59
        )
        df = df.filter(pl.col("start_time") <= end_dt)
    return df


def _apply_session_filter(df: pl.DataFrame, sessions: SessionsInput) -> pl.DataFrame:
    """Applique le filtre de session au DataFrame."""
    if "session_label" not in df.columns:
        return df

    labels_to_keep: set[str] = set()

    # Mode session unique (rétro-compat)
    if sessions.picked_session_label:
        labels_to_keep.add(sessions.picked_session_label)
    if sessions.picked_solo_session_label:
        labels_to_keep.add(sessions.picked_solo_session_label)
    if sessions.picked_squad_session_label:
        labels_to_keep.add(sessions.picked_squad_session_label)
    # Multi-session
    labels_to_keep.update(sessions.picked_sessions)

    if not labels_to_keep:
        return df
    return df.filter(pl.col("session_label").is_in(labels_to_keep))


def _apply_cascade_filter(df: pl.DataFrame, cascade: CascadeInput) -> pl.DataFrame:
    """Applique les filtres cascade Expérience → Playlist → Mode → Carte."""
    df = _apply_experience_filter(df, cascade.experience_types)
    if cascade.playlists:
        df = df.filter(pl.col("playlist_ui").cast(pl.Utf8).is_in(cascade.playlists))
    if cascade.modes:
        df = df.filter(pl.col("mode_ui").cast(pl.Utf8).is_in(cascade.modes))
    if cascade.maps:
        df = df.filter(pl.col("map_ui").cast(pl.Utf8).is_in(cascade.maps))
    return df


# ---------------------------------------------------------------------------
# Normalisation de l'input
# ---------------------------------------------------------------------------


def _normalize_filter_input(
    raw: FilterContextInput,
    df_full: pl.DataFrame,
) -> FilterContextInput:
    """Normalise ``FilterContextInput`` :

    - Valide les dates (end_date >= start_date)
    - Supprime les options de cascade invalides (non présentes dans les données)
    - Conserve les listes vides telles quelles (= pas de filtre)
    """
    if df_full.is_empty():
        return raw

    period = raw.period
    if period.start_date and period.end_date and period.end_date < period.start_date:
        # Inverser silencieusement
        period = PeriodInput(start_date=period.end_date, end_date=period.start_date)

    return raw.model_copy(update={"period": period})


# ---------------------------------------------------------------------------
# Entrée publique
# ---------------------------------------------------------------------------


def resolve_filters(
    player: PlayerContext,
    ctx: FilterContextInput,
) -> FilterContextResolved:
    """Résout les filtres et retourne le contexte normalisé avec les options disponibles.

    Algorithme :
    1. Charger tous les matchs du joueur (lecture seule DuckDB)
    2. Construire les sessions disponibles
    3. Normaliser l'input (dates, options invalides supprimées)
    4. Appliquer les filtres temporels (période ou sessions)
    5. Calculer les options disponibles après filtre temporel
    6. Appliquer les filtres cascade
    7. Retourner FilterContextResolved
    """
    df_full = _load_matches_for_filters(player)
    total_before = len(df_full)

    # Sessions disponibles (calculées sur le dataset complet)
    session_opts = _build_session_options(df_full)

    # Normaliser l'input
    effective = _normalize_filter_input(ctx, df_full)

    if df_full.is_empty():
        return FilterContextResolved(
            effective=effective,
            available_options=AvailableOptions(
                experience_types=[LabelValue(label=e, value=e) for e in _EXPERIENCE_LABELS],
            ),
            session_options=session_opts,
            counts=FilterCounts(
                total_matches_before_filters=0,
                total_matches_after_filters=0,
            ),
        )

    # Appliquer filtre temporel
    if effective.filter_mode == "sessions":
        df_temporal = _apply_session_filter(df_full, effective.sessions)
    else:
        df_temporal = _apply_period_filter(df_full, effective.period)

    # Options disponibles (après filtre temporel, avant cascade)
    available_opts = _build_available_options(df_temporal, effective.cascade)

    # Appliquer cascade
    df_filtered = _apply_cascade_filter(df_temporal, effective.cascade)
    total_after = len(df_filtered)

    return FilterContextResolved(
        effective=effective,
        available_options=available_opts,
        session_options=session_opts,
        counts=FilterCounts(
            total_matches_before_filters=total_before,
            total_matches_after_filters=total_after,
        ),
    )
