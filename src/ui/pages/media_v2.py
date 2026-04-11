"""Page Médias V2 — réécriture depuis zéro, architecture claire et stable."""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl
import streamlit as st

from src.data.media_indexer import MediaIndexer
from src.ui.i18n import t
from src.ui.pages.media_v2_filters import (
    MediaFilterState,
    apply_media_filters,
    render_media_filters,
)
from src.ui.pages.media_v2_grid import render_lightbox_if_pending, render_media_grid
from src.ui.settings import AppSettings

logger = logging.getLogger(__name__)

# Colonnes issues de df_full utiles pour l'affichage (carte, mode, résultat)
_ENRICH_COLS: list[str] = ["match_id", "map_ui", "mode_ui", "outcome"]


def render_media_v2(
    *,
    df_full: pl.DataFrame | None = None,
    settings: AppSettings | None = None,
) -> None:
    """Rend la page Médias V2.

    Point d'entrée appelé par streamlit_app.py à la place de render_media_tab.
    """
    # Lightbox en premier : doit précéder tout autre rendu st.*
    render_lightbox_if_pending()

    if settings is None:
        settings = AppSettings()
    if not getattr(settings, "media_enabled", True):
        st.info(t("media_disabled"))
        return

    db_path = st.session_state.get("db_path", "")
    if not db_path or not str(db_path).endswith(".duckdb"):
        st.info(t("no_data_filter"))
        return

    current_xuid = _resolve_current_xuid(db_path)
    media_df = MediaIndexer.load_media_for_ui(Path(db_path), current_xuid)
    if media_df.is_empty():
        st.info(t("media_no_indexed"))
        return

    media_df = _enrich_with_match_data(media_df, df_full)
    media_df = _enrich_from_enrichment_table(media_df, db_path)
    state = render_media_filters(media_df)
    _render_sections(media_df, state)


# Colonne de regroupement pour chaque clé group_by
_GROUP_COL: dict[str, str] = {
    "date": "capture_end_utc",
    "map": "map_ui",
    "mode": "mode_ui",
    "session": "session_label",
    "experience": "_experience_label",
}

# ── Sections ─────────────────────────────────────────────────────────────────


def _render_sections(media_df: pl.DataFrame, state: MediaFilterState) -> None:
    """Applique filtres puis rend soit sections auteur, soit groupes."""
    if state.group_by == "author":
        _render_by_auteur(media_df, state)
        return

    # Construire le pool filtré selon l'auteur
    pool = _build_pool(media_df, state)
    if pool.is_empty():
        return

    if state.group_by:
        _render_grouped(pool, state)
    else:
        # Aucun groupement : liste plate triée
        rendered = apply_media_filters(pool, state)
        if not rendered.is_empty():
            render_media_grid(rendered, state)


def _build_pool(media_df: pl.DataFrame, state: MediaFilterState) -> pl.DataFrame:
    """Retourne le sous-ensemble correspondant au filtre auteur, dédupliqué."""
    show_all = state.auteur_key is None
    if show_all:
        df = media_df
    elif state.auteur_key == "mine":
        df = (
            media_df.filter(pl.col("section") == "mine")
            if "section" in media_df.columns
            else media_df
        )
    else:
        if "gamertag" in media_df.columns:
            df = media_df.filter(pl.col("gamertag") == state.auteur_key)
        else:
            df = media_df
    return df.unique(subset=["file_path"], keep="first", maintain_order=True)


def _render_grouped(df: pl.DataFrame, state: MediaFilterState) -> None:
    """Rend la grille regroupée par date / carte / mode."""
    # Calculer _experience_label avant de chercher la colonne
    if state.group_by == "experience" and "is_with_friends" in df.columns:
        df = df.with_columns(
            pl.when(pl.col("is_with_friends") == True)  # noqa: E712
            .then(pl.lit(t("media_experience_squad")))
            .otherwise(pl.lit(t("media_experience_solo")))
            .alias("_experience_label")
        )

    group_col = _GROUP_COL.get(state.group_by or "", "capture_end_utc")
    actual_col = group_col if group_col in df.columns else "capture_end_utc"

    # Calculer la clé de groupe : date = jour, map/mode = valeur brute
    if state.group_by == "date" and actual_col in df.columns:
        df = df.with_columns(pl.col(actual_col).cast(pl.Date).alias("_group_key"))
    elif actual_col in df.columns:
        df = df.with_columns(pl.col(actual_col).fill_null("—").alias("_group_key"))
    else:
        df = df.with_columns(pl.lit("—").alias("_group_key"))

    filtered = apply_media_filters(df, state)
    if filtered.is_empty():
        return

    group_keys = (
        filtered.sort("_group_key", descending=state.sort_desc, nulls_last=True)["_group_key"]
        .unique(maintain_order=True)
        .to_list()
    )

    for key in group_keys:
        sub = filtered.filter(pl.col("_group_key") == key)
        if sub.is_empty():
            continue
        label = _format_group_label(state.group_by, key)
        st.markdown(f"#### {label}")
        render_media_grid(sub, state)


def _format_group_label(group_by: str | None, key: object) -> str:
    """Formate le titre d'un groupe."""
    if group_by == "date":
        try:
            import datetime

            if isinstance(key, datetime.date):
                from src.ui.pages.media_v2_grid import _fmt_date_long

                return _fmt_date_long(key)
        except Exception:
            pass
        return str(key)
    if group_by == "session":
        raw = str(key) if key else "—"
        if raw != "—":
            from src.ui.pages.media_v2_grid import _fmt_session_label

            return _fmt_session_label(raw)
        return raw
    if group_by in {"experience", "map", "mode"}:
        return str(key) if key and str(key) != "—" else "—"
    return str(key) if key else "—"


def _render_by_auteur(media_df: pl.DataFrame, state: MediaFilterState) -> None:
    """Mode sans groupement : sections mine / coéquipiers / sans correspondance."""
    show_all = state.auteur_key is None
    mine = _dedup_section(media_df, "mine")
    teammates = _dedup_section(media_df, "teammate")
    unassigned = _dedup_section(media_df, "unassigned")

    if not mine.is_empty() and (show_all or state.auteur_key == "mine"):
        rendered = apply_media_filters(mine, state)
        if not rendered.is_empty():
            st.markdown(f"### {t('media_my_captures')}")
            render_media_grid(rendered, state)

    if not teammates.is_empty() and "gamertag" in teammates.columns:
        gamertags = sorted(g for g in teammates["gamertag"].drop_nulls().unique().to_list() if g)
        for gamertag in gamertags:
            if not (show_all or state.auteur_key == str(gamertag)):
                continue
            sub = teammates.filter(pl.col("gamertag") == gamertag)
            rendered = apply_media_filters(sub, state)
            if not rendered.is_empty():
                st.markdown(f"### {t('media_captures_of', gamertag=gamertag)}")
                render_media_grid(rendered, state)

    if not unassigned.is_empty() and show_all:
        rendered = apply_media_filters(unassigned, state, match_filters=False)
        if not rendered.is_empty():
            st.markdown(f"### {t('media_unmatched')}")
            render_media_grid(rendered, state)


# ── Helpers privés ────────────────────────────────────────────────────────────


def _dedup_section(df: pl.DataFrame, section: str) -> pl.DataFrame:
    """Filtre une section et déduplique sur file_path."""
    if "section" not in df.columns:
        return pl.DataFrame()
    return df.filter(pl.col("section") == section).unique(
        subset=["file_path"], keep="first", maintain_order=True
    )


def _enrich_with_match_data(
    media_df: pl.DataFrame,
    df_full: pl.DataFrame | None,
) -> pl.DataFrame:
    """Enrichit media_df avec map_ui / mode_ui depuis df_full via match_id."""
    if df_full is None or df_full.is_empty():
        return media_df
    if "match_id" not in media_df.columns or "match_id" not in df_full.columns:
        return media_df
    available = [c for c in _ENRICH_COLS if c in df_full.columns]
    if len(available) <= 1:
        return media_df
    existing = set(media_df.columns) - {"match_id"}
    cols = ["match_id"] + [c for c in available[1:] if c not in existing]
    enrich = df_full.select(cols).unique(subset=["match_id"])
    return media_df.join(enrich, on="match_id", how="left")


def _enrich_from_enrichment_table(
    media_df: pl.DataFrame,
    db_path: str,
) -> pl.DataFrame:
    """Enrichit media_df avec session_label et is_with_friends depuis player_match_enrichment."""
    if "match_id" not in media_df.columns:
        return media_df
    needed = {"session_label", "is_with_friends"} - set(media_df.columns)
    if not needed:
        return media_df
    try:
        import duckdb

        cols = ", ".join(["match_id"] + sorted(needed))
        with duckdb.connect(db_path, read_only=True) as con:
            rows = con.execute(
                f"SELECT {cols} FROM player_match_enrichment"  # noqa: S608
            ).fetchall()
        col_names = ["match_id"] + sorted(needed)
        enrich = pl.DataFrame(
            {name: [r[i] for r in rows] for i, name in enumerate(col_names)}
        ).unique(subset=["match_id"])
        existing = set(media_df.columns) - {"match_id"}
        join_cols = ["match_id"] + [
            c for c in enrich.columns if c != "match_id" and c not in existing
        ]
        return media_df.join(enrich.select(join_cols), on="match_id", how="left")
    except Exception:
        logger.debug("_enrich_from_enrichment_table: impossible d'enrichir", exc_info=True)
        return media_df


def _resolve_current_xuid(db_path: str) -> str | None:
    """Résout le XUID courant depuis l'identité active."""
    xuid_input = st.session_state.get("xuid_input", "")
    try:
        from src.app.profile import get_identity_from_secrets, resolve_xuid

        identity = get_identity_from_secrets()
        return (
            resolve_xuid(xuid_input or identity.gamertag or "", db_path, identity) or identity.xuid
        )
    except Exception:
        return None
