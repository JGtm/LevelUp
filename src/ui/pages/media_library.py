"""Page Bibliothèque médias.

**Note** : L’onglet principal est désormais « Médias » (media_tab.py), qui charge
les données depuis la BDD (media_files, media_match_associations) et affiche
les sections Mes captures / Captures de XXX / Sans correspondance. Ce module
reste disponible pour compatibilité (dispatch « Bibliothèque médias » → render_media_tab)
et pour options avancées (re-scan manuel, etc.) si besoin.

Objectif: proposer une vue "bibliothèque" qui scanne les dossiers de médias
(configurés dans les paramètres) et permet d'ouvrir rapidement le match associé.

L'association média → match se fait par proximité temporelle:
- on indexe les fichiers (mtime)
- on calcule pour chaque match une fenêtre [start - tol ; end + tol]
- on associe un média au match dont la fenêtre contient son mtime

Note: cette page ne dépend pas de métadonnées dans les noms de fichiers.
"""

from __future__ import annotations

import contextlib
import hashlib
import html
import os
import urllib.parse
from dataclasses import dataclass

import polars as pl
import streamlit as st

from src.ui.formatting import format_datetime_fr_hm
from src.ui.i18n import t
from src.ui.pages.match_view_helpers import index_media_dir
from src.ui.pages.media_library_data import (
    associate_media_to_matches as _associate_media_to_matches,
)
from src.ui.pages.media_library_data import (
    compute_match_windows as _compute_match_windows,
)
from src.ui.pages.media_library_data import (
    gamertag_from_db_path as _gamertag_from_db_path,
)
from src.ui.pages.media_library_data import (
    index_all_media as _index_all_media,
)
from src.ui.pages.media_library_data import (
    load_match_windows_from_db as _load_match_windows_from_db,
)
from src.ui.pages.media_library_data import (
    load_media_from_db as _load_media_from_db,
)
from src.ui.settings import AppSettings
from src.visualization._compat import DataFrameLike, ensure_polars


@dataclass(frozen=True)
class _MediaDirs:
    screens_dir: str
    videos_dir: str


def _coerce_dirs(settings: AppSettings) -> _MediaDirs:
    screens_dir = str(getattr(settings, "media_screens_dir", "") or "").strip()
    videos_dir = str(getattr(settings, "media_videos_dir", "") or "").strip()
    return _MediaDirs(screens_dir=screens_dir, videos_dir=videos_dir)


def _build_app_url(page: str, **params: str) -> str:
    qp: dict[str, str] = {"page": str(page)}
    for k, v in params.items():
        s = str(v or "").strip()
        if s:
            qp[str(k)] = s
    return "?" + urllib.parse.urlencode(qp)


def _open_match_button(match_id: str, *, unique_suffix: str | None = None) -> None:
    """Affiche un bouton pour ouvrir la page Match.

    Args:
        match_id: ID du match à ouvrir
        unique_suffix: Suffixe optionnel pour rendre la clé unique (ex: path_hash ou stable_id)
    """
    mid = str(match_id or "").strip()
    if not mid:
        st.caption(t("media_unknown_match"))
        return

    # Rendre la clé unique en incluant le suffixe si fourni
    # Cela évite les clés dupliquées quand plusieurs médias ont le même match_id
    button_key = f"open_match_{mid}_{unique_suffix}" if unique_suffix else f"open_match_{mid}"

    # Utiliser _pending_page au lieu de modifier directement "page"
    # car le widget segmented_control avec key="page" est déjà instancié
    # consume_pending_page() s'occupera de mettre à jour "page" au prochain rendu
    if st.button(t("ml_open_match"), key=button_key, width="stretch"):
        st.session_state["_pending_page"] = "Match"
        st.session_state["_pending_match_id"] = mid
        st.rerun()


def _placeholder_html(base: str, hint: str | None = None) -> str:
    """HTML du placeholder vidéo (sans charger la miniature)."""
    if hint is None:
        hint = t("ml_click_thumbnail")
    return (
        "<div style='padding:18px;border-radius:12px;border:1px solid rgba(255,255,255,0.12);'>"
        "<div style='font-size:34px;line-height:1'>🎬</div>"
        "<div style='opacity:0.85;margin-top:6px'>" + html.escape(base) + "</div>"
        "<div style='font-size:11px;opacity:0.6;margin-top:4px'>" + html.escape(hint) + "</div>"
        "</div>"
    )


def _render_media_grid(
    items: DataFrameLike, *, cols_per_row: int, render_context: str = "default"
) -> None:
    """Affiche une grille de médias Streamlit."""
    if items is None:
        st.info(t("media_no_filter_result"))
        return
    items = ensure_polars(items)
    if items.is_empty():
        st.info(t("media_no_filter_result"))
        return

    cols_per_row = int(cols_per_row)
    if cols_per_row < 2:
        cols_per_row = 2
    if cols_per_row > 8:
        cols_per_row = 8

    # Ajouter un identifiant stable au DataFrame AVANT le rendu
    # pour éviter que les clés session_state changent à chaque rendu
    items = items.with_row_index("_stable_id")

    rows = items.to_dicts()
    for i in range(0, len(rows), cols_per_row):
        chunk = rows[i : i + cols_per_row]
        # TOUJOURS créer cols_per_row colonnes, même si len(chunk) < cols_per_row
        # pour éviter que les images prennent toute la largeur
        cols = st.columns(cols_per_row)
        for col_idx in range(cols_per_row):
            with cols[col_idx]:
                if col_idx < len(chunk):
                    rec = chunk[col_idx]
                    path = str(rec.get("path") or "").strip()
                    kind = str(rec.get("kind") or "")
                    base = str(rec.get("basename") or os.path.basename(path))
                    mid = rec.get("match_id")

                    if kind == "image" and path:
                        try:
                            st.image(path, width="stretch")
                        except Exception:
                            st.caption(base)
                    else:
                        # Vidéo : afficher la miniature seulement au clic (évite tout charger à l'ouverture)
                        # Clé stable : hash du path + match_id + contexte + identifiant stable du média
                        # (pas de position dans la grille pour éviter l'instabilité)
                        thumb_path = str(rec.get("thumbnail_path") or "").strip()
                        path_hash = hashlib.md5(path.encode()).hexdigest()
                        match_id_part = (
                            str(mid).strip() if isinstance(mid, str) and mid.strip() else "no_match"
                        )
                        # Utiliser l'ID stable au lieu de i et col_idx
                        stable_id = rec.get("_stable_id", 0)
                        thumb_key = f"thumb_show::{path_hash}::{match_id_part}::{render_context}::{stable_id}"
                        show_thumb = st.session_state.get(thumb_key, False)

                        if show_thumb and thumb_path and os.path.exists(thumb_path):
                            try:
                                st.image(thumb_path, width="stretch")
                            except Exception:
                                st.markdown(_placeholder_html(base), unsafe_allow_html=True)
                            if st.button(t("ml_hide_thumbnail"), key=thumb_key + "::btn"):
                                st.session_state[thumb_key] = False
                                st.rerun()
                        else:
                            st.markdown(
                                _placeholder_html(base),
                                unsafe_allow_html=True,
                            )
                            if thumb_path and os.path.exists(thumb_path):
                                if st.button(t("ml_show_thumbnail"), key=thumb_key + "::btn"):
                                    st.session_state[thumb_key] = True
                                    st.rerun()
                            else:
                                st.caption(t("media_no_thumbnail"))
                        if path:
                            preview_key = f"media_preview::{path_hash}::{match_id_part}::{render_context}::{stable_id}"
                            if st.button(t("ml_preview"), key=preview_key, width="stretch"):
                                st.session_state[preview_key + "::open"] = True
                            if st.session_state.get(preview_key + "::open"):
                                try:
                                    st.video(path)
                                except Exception:
                                    st.caption(path)

                    st.caption(base)
                    # Ne pas afficher le bouton "Ouvrir le match" si on est dans un contexte de groupe
                    # (le bouton est déjà affiché avant la grille dans l'expander)
                    if (
                        isinstance(mid, str)
                        and mid.strip()
                        and not render_context.startswith("match_")
                    ):
                        # Utiliser le stable_id pour rendre la clé unique même si plusieurs médias ont le même match_id
                        stable_id = rec.get("_stable_id", 0)
                        _open_match_button(mid, unique_suffix=str(stable_id))
                    elif isinstance(mid, str) and mid.strip():
                        # Dans un groupe de match, le bouton est déjà affiché avant la grille
                        pass
                    else:
                        st.caption(t("media_unassociated_match"))


def render_media_library_page(*, df_full: DataFrameLike, settings: AppSettings) -> None:
    """Rend la page Bibliothèque médias."""
    st.subheader(t("media_library_title"))

    if not bool(getattr(settings, "media_enabled", True)):
        st.info(t("media_disabled"))
        return

    dirs = _coerce_dirs(settings)
    if not dirs.screens_dir and not dirs.videos_dir:
        st.info(t("media_no_folder"))
        return

    # Récupérer le XUID du joueur actuel
    db_path = st.session_state.get("db_path", "")
    xuid_input = st.session_state.get("xuid_input", "")

    # Résoudre le XUID
    from src.app.profile import resolve_xuid
    from src.app.state import get_default_identity

    identity = get_default_identity()
    xuid = (
        resolve_xuid(xuid_input or "SpartanC", db_path, identity)
        or identity.xuid
        or identity.xuid_fallback
    )

    with st.expander(t("ml_options"), expanded=True):
        c1, c2, c3, c4 = st.columns([1, 1, 1, 1])
        group_by_match = c1.toggle(t("ml_group_by_match"), value=True)
        show_unassigned = c2.toggle(t("ml_show_unassociated"), value=True)
        cols_per_row = c3.slider(t("media_columns"), min_value=2, max_value=6, value=4, step=1)
        max_items = c4.slider(t("ml_max_media"), min_value=50, max_value=2000, value=400, step=50)

        kinds = st.multiselect(
            t("ml_types"),
            options=["image", "video"],
            default=["image", "video"],
        )
        name_filter = st.text_input(t("ml_filter_filename"), value="", placeholder="ex: 2026-01")

        col_scan, col_thumbs = st.columns(2)
        with col_scan:
            if st.button(t("ml_rescan"), width="stretch"):
                with contextlib.suppress(Exception):
                    index_media_dir.clear()
                # Forcer re-indexation en BDD
                if "_media_indexing_started" in st.session_state:
                    del st.session_state["_media_indexing_started"]

                # Lancer l'indexation manuellement si DB DuckDB disponible
                if db_path and db_path.endswith(".duckdb"):
                    try:
                        from pathlib import Path

                        from src.data.media_indexer import MediaIndexer

                        videos_path = (
                            Path(dirs.videos_dir)
                            if dirs.videos_dir and os.path.exists(dirs.videos_dir)
                            else None
                        )
                        screens_path = (
                            Path(dirs.screens_dir)
                            if dirs.screens_dir and os.path.exists(dirs.screens_dir)
                            else None
                        )

                        if videos_path or screens_path:
                            with st.spinner(t("media_scanning")):
                                indexer = MediaIndexer(Path(db_path))
                                result = indexer.scan_and_index(
                                    videos_dir=videos_path,
                                    screens_dir=screens_path,
                                    force_rescan=True,
                                )
                                tolerance = int(
                                    getattr(settings, "media_tolerance_minutes", 5) or 5
                                )
                                n_associated = indexer.associate_with_matches(
                                    tolerance_minutes=tolerance
                                )
                                n_thumb_gen, n_thumb_err = 0, 0
                                if videos_path:
                                    n_thumb_gen, n_thumb_err = indexer.generate_thumbnails_for_new(
                                        videos_path
                                    )
                                msg = t(
                                    "ml_indexing_done",
                                    n_new=result.n_new,
                                    n_updated=result.n_updated,
                                    n_associated=n_associated,
                                )
                                if n_thumb_gen or n_thumb_err:
                                    msg += t(
                                        "ml_indexing_thumbnails",
                                        n_gen=n_thumb_gen,
                                        n_err=n_thumb_err,
                                    )
                                st.success(msg)
                    except Exception as e:
                        st.error(t("media_error_indexing", error=e))

                st.rerun()

        with col_thumbs:
            if st.button(
                t("ml_generate_thumbnails"),
                width="stretch",
                help=t("ml_generate_help"),
            ):
                if (
                    db_path
                    and db_path.endswith(".duckdb")
                    and dirs.videos_dir
                    and os.path.exists(dirs.videos_dir)
                ):
                    try:
                        from pathlib import Path

                        from src.data.media_indexer import MediaIndexer

                        with st.spinner(t("media_generating_thumbnails")):
                            indexer = MediaIndexer(Path(db_path))
                            n_gen, n_err = indexer.generate_thumbnails_for_new(
                                Path(dirs.videos_dir)
                            )
                            st.success(
                                t("ml_thumbnails_generated", n_gen=n_gen)
                                + (
                                    t("ml_indexing_thumbnails", n_gen=0, n_err=n_err)
                                    if n_err
                                    else ""
                                )
                            )
                    except Exception as e:
                        st.error(t("error_loading", error=e))
                    st.rerun()
                else:
                    st.warning(t("media_configure_video"))

    # Charger depuis BDD si disponible
    media_df = pl.DataFrame()
    using_db = False
    windows_df = pl.DataFrame()  # Initialiser pour le diagnostic
    if db_path and db_path.endswith(".duckdb"):
        # Charger les médias avec associations pour le joueur actuel (ou tous si xuid=None)
        gamertag = _gamertag_from_db_path(db_path)
        media_df = _load_media_from_db(db_path, xuid=xuid, gamertag=gamertag)
        using_db = not media_df.is_empty()

    # Fallback sur scan disque si BDD vide
    if media_df.is_empty():
        media_df = _index_all_media(settings)
        # Si on a scanné depuis disque, on peut essayer d'associer avec les matchs
        if not media_df.is_empty():
            windows_df = _compute_match_windows(df_full, settings)
            assoc_df = _associate_media_to_matches(media_df, windows_df)
        else:
            assoc_df = pl.DataFrame()
    else:
        # Les associations sont déjà dans la BDD
        assoc_df = media_df.clone()
        # S'assurer que match_id est bien présent même si NULL
        if "match_id" not in assoc_df.columns:
            assoc_df = assoc_df.with_columns(pl.lit(None).alias("match_id"))
        if "match_start_time" not in assoc_df.columns:
            assoc_df = assoc_df.with_columns(pl.lit(None).alias("match_start_time"))
        # Calculer windows_df pour le diagnostic depuis la DB des médias (pas df_full)
        # car l'association se fait depuis toutes les DBs joueurs, pas seulement celle du joueur actuel
        windows_df = _load_match_windows_from_db(db_path) if db_path else pl.DataFrame()

    # Diagnostic : afficher info si médias non associés depuis BDD
    if using_db and not assoc_df.is_empty():
        unassigned_count = assoc_df["match_id"].is_null().sum()
        if unassigned_count > 0:
            st.info(t("ml_unassigned_from_db", count=unassigned_count))

    if assoc_df.is_empty():
        st.info(t("media_no_files"))
        return

    assoc_df = assoc_df.head(int(max_items))

    if kinds:
        assoc_df = assoc_df.filter(pl.col("kind").is_in([str(k) for k in kinds]))

    if name_filter.strip():
        nf = name_filter.strip().lower()
        assoc_df = assoc_df.filter(
            pl.col("basename").cast(pl.Utf8).str.to_lowercase().str.contains(nf, literal=True)
        )

    assigned = assoc_df.filter(pl.col("match_id").is_not_null())
    unassigned = assoc_df.filter(pl.col("match_id").is_null())

    # DÉDUPLIQUER : Un média peut avoir plusieurs associations (multi-joueurs)
    # On garde une seule ligne par média/match pour l'affichage
    if not assigned.is_empty():
        assigned = assigned.unique(subset=["path", "match_id"], keep="first")
    if not unassigned.is_empty():
        unassigned = unassigned.unique(subset=["path"], keep="first")

    # Diagnostic unifié : afficher un seul message informatif
    if not using_db:
        st.info(t("ml_disk_fallback"))
    elif windows_df.is_empty() and assigned.is_empty():
        st.warning(t("ml_no_match_windows"))
    elif assigned.is_empty() and not unassigned.is_empty() and using_db:
        tolerance = int(getattr(settings, "media_tolerance_minutes", 5) or 5)
        st.warning(t("ml_no_associations", tol=tolerance))

    # Affichage
    if not group_by_match:
        _render_media_grid(assoc_df, cols_per_row=int(cols_per_row), render_context="all")
        return

    if not assigned.is_empty():
        # Tri: match le plus récent d'abord, puis médias par ordre chronologique (mtime asc)
        assigned = assigned.with_columns(
            pl.col("match_start_time").cast(pl.Datetime, strict=False).alias("_match_sort")
        )
        assigned = assigned.sort(["_match_sort", "mtime"], descending=[True, False])

        for match_id, g in assigned.group_by("match_id", maintain_order=True):
            match_id_val = match_id[0] if isinstance(match_id, tuple) else match_id
            title_dt = None
            try:
                dt0 = g["match_start_time"][0]
                title_dt = format_datetime_fr_hm(dt0) if dt0 is not None else None
            except Exception:
                title_dt = None

            label = f"Match {match_id_val}" + (" — " + str(title_dt) if title_dt else "")
            with st.expander(label, expanded=False):
                _open_match_button(str(match_id_val))
                g2 = g.sort("mtime", descending=False)
                # Dédupliquer une dernière fois par sécurité (au cas où plusieurs xuid pour même média/match)
                g2 = g2.unique(subset=["path"], keep="first")
                _render_media_grid(
                    g2, cols_per_row=int(cols_per_row), render_context=f"match_{match_id_val}"
                )

    if show_unassigned and not unassigned.is_empty():
        st.divider()
        st.subheader(t("media_unassociated"))
        _render_media_grid(unassigned, cols_per_row=int(cols_per_row), render_context="unassigned")
