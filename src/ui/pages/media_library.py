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
from datetime import datetime, timedelta

import polars as pl
import streamlit as st

from src.ui.formatting import PARIS_TZ, format_datetime_fr_hm
from src.ui.i18n import t
from src.ui.pages.match_view_helpers import index_media_dir
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


def _epoch_seconds_paris(dt_value: datetime | None) -> float | None:
    if dt_value is None:
        return None
    try:
        aware = (
            PARIS_TZ.localize(dt_value)
            if dt_value.tzinfo is None
            else dt_value.astimezone(PARIS_TZ)
        )
        return float(aware.timestamp())
    except Exception:
        return None


def _to_paris_naive(dt_value: object) -> datetime | None:
    """Convertit une valeur datetime en datetime naïve (fuseau Paris)."""
    try:
        if dt_value is None:
            return None
        if isinstance(dt_value, datetime):
            ts = dt_value
        elif isinstance(dt_value, str):
            s = str(dt_value).strip()
            if not s:
                return None
            if s.endswith("Z"):
                s = s[:-1] + "+00:00"
            ts = datetime.fromisoformat(s)
        else:
            return None
        if ts.tzinfo is None:
            return ts
        return ts.astimezone(PARIS_TZ).replace(tzinfo=None)
    except Exception:
        return None


def _compute_match_windows(df_full: DataFrameLike, settings: AppSettings) -> pl.DataFrame:
    """Construit les fenêtres temporelles des matchs (epoch seconds) pour l'association média.

    Améliorations:
    - Gère les start_time NULL avec un diagnostic
    - Utilise une durée par défaut de 12 minutes si time_played_seconds est NULL
    """
    _empty = pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_epoch": pl.Float64,
            "end_epoch": pl.Float64,
            "start_time": pl.Datetime,
        }
    )
    if df_full is None:
        return _empty
    df_full = ensure_polars(df_full)
    if df_full.is_empty():
        return _empty

    tol_min = int(getattr(settings, "media_tolerance_minutes", 0) or 0)
    tol = timedelta(minutes=max(0, tol_min))

    needed = {"match_id", "start_time"}
    if not needed.issubset(set(df_full.columns)):
        return _empty

    cols = [c for c in ["match_id", "start_time", "time_played_seconds"] if c in df_full.columns]
    base = df_full.select(cols)

    # Calculer les fenêtres via Python (struct temporel complexe)
    rows: list[dict[str, object]] = []
    for rec in base.iter_rows(named=True):
        start = _to_paris_naive(rec.get("start_time"))
        if not isinstance(start, datetime):
            continue
        dur_s = rec.get("time_played_seconds")
        try:
            dur = float(dur_s) if dur_s is not None else None
        except Exception:
            dur = None
        if dur is None or dur <= 0:
            end = start + timedelta(minutes=12)
        else:
            end = start + timedelta(seconds=dur)
        t0 = start - tol
        t1 = end + tol
        se = _epoch_seconds_paris(t0)
        ee = _epoch_seconds_paris(t1)
        mid = rec.get("match_id")
        if mid is None or se is None or ee is None:
            continue
        rows.append(
            {
                "match_id": str(mid),
                "start_epoch": se,
                "end_epoch": ee,
                "start_time": start,
            }
        )

    if not rows:
        return _empty

    return pl.DataFrame(rows).sort("start_epoch")


def _index_all_media(settings: AppSettings) -> pl.DataFrame:
    """Indexe les médias configurés (captures + vidéos)."""
    dirs = _coerce_dirs(settings)
    frames: list[pl.DataFrame] = []

    if dirs.screens_dir and os.path.isdir(dirs.screens_dir):
        img_df = index_media_dir(dirs.screens_dir, ("png", "jpg", "jpeg", "webp"))
        if not img_df.is_empty():
            img_df = img_df.with_columns(pl.lit("image").alias("kind"))
            frames.append(img_df)

    if dirs.videos_dir and os.path.isdir(dirs.videos_dir):
        vid_df = index_media_dir(dirs.videos_dir, ("mp4", "webm", "mkv", "mov"))
        if not vid_df.is_empty():
            vid_df = vid_df.with_columns(pl.lit("video").alias("kind"))
            frames.append(vid_df)

    if not frames:
        return pl.DataFrame(
            schema={"path": pl.Utf8, "mtime": pl.Float64, "ext": pl.Utf8, "kind": pl.Utf8}
        )

    df = pl.concat(frames)
    if df.is_empty():
        return df

    df = df.with_columns(
        [
            pl.col("path").cast(pl.Utf8),
            # Vectorisation: str ops au lieu de map_elements(os.path.basename)
            pl.col("path")
            .cast(pl.Utf8)
            .str.replace_all(r"\\", "/")
            .str.split("/")
            .list.last()
            .alias("basename"),
        ]
    ).drop_nulls(subset=["mtime"])
    return df.sort("mtime", descending=True)


def _associate_media_to_matches(media_df: pl.DataFrame, windows_df: pl.DataFrame) -> pl.DataFrame:
    """Associe chaque média à un match (best-effort) via join_asof + check de fenêtre.

    Amélioration: utilise strategy="nearest" pour capturer les médias pris
    légèrement AVANT le match (ex: pendant le chargement) ou APRÈS.
    Vérifie ensuite que le média est bien dans la fenêtre [start_epoch, end_epoch].
    """
    if media_df is None or media_df.is_empty():
        extra_cols = {"match_id": pl.Utf8, "match_start_time": pl.Datetime}
        if media_df is not None:
            schema = {
                **{c: media_df.dtypes[i] for i, c in enumerate(media_df.columns)},
                **extra_cols,
            }
            return pl.DataFrame(schema=schema)
        return pl.DataFrame()

    if windows_df is None or windows_df.is_empty():
        return media_df.with_columns(
            [
                pl.lit(None).cast(pl.Utf8).alias("match_id"),
                pl.lit(None).cast(pl.Datetime).alias("match_start_time"),
            ]
        )

    # S'assurer que mtime est numérique et non-null
    m = media_df.drop_nulls(subset=["mtime"]).sort("mtime")
    w = windows_df.sort("start_epoch")

    # join_asof : strategy="nearest" pour trouver le match le plus proche
    joined = m.join_asof(
        w,
        left_on="mtime",
        right_on="start_epoch",
        strategy="nearest",
    )

    # Vérifier que le média est dans la fenêtre [start_epoch, end_epoch]
    joined = joined.with_columns(
        [
            pl.when(
                pl.col("start_epoch").is_not_null()
                & pl.col("end_epoch").is_not_null()
                & (pl.col("mtime") >= pl.col("start_epoch"))
                & (pl.col("mtime") <= pl.col("end_epoch"))
            )
            .then(pl.col("match_id"))
            .otherwise(pl.lit(None))
            .alias("match_id"),
            pl.when(
                pl.col("start_epoch").is_not_null()
                & pl.col("end_epoch").is_not_null()
                & (pl.col("mtime") >= pl.col("start_epoch"))
                & (pl.col("mtime") <= pl.col("end_epoch"))
            )
            .then(pl.col("start_time"))
            .otherwise(pl.lit(None))
            .alias("start_time"),
        ]
    )

    # Renommer et nettoyer
    drop_cols = [c for c in ["start_epoch", "end_epoch"] if c in joined.columns]
    joined = joined.drop(drop_cols)
    if "start_time" in joined.columns:
        joined = joined.rename({"start_time": "match_start_time"})

    return joined.sort("mtime", descending=True)


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


def _load_match_windows_from_db(db_path: str) -> pl.DataFrame:
    """Charge les fenêtres temporelles des matchs depuis la DB pour le diagnostic.

    V5 : Utilise shared_matches.duckdb si disponible (requête unique vs N DBs).
    Fallback v4 : Parcourt les DBs joueurs individuelles.
    """
    _empty = pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_epoch": pl.Float64,
            "end_epoch": pl.Float64,
            "start_time": pl.Datetime,
        }
    )
    try:
        from src.utils.db import duckdb_read_only
        from src.utils.paths import PLAYERS_DIR

        # --- V5 : requête unique via shared_matches.duckdb ---
        shared_db = PLAYERS_DIR.parent / "warehouse" / "shared_matches.duckdb"
        if shared_db.exists():
            try:
                with duckdb_read_only(shared_db) as conn:
                    matches = conn.execute(
                        """
                        SELECT match_id, start_time, duration_seconds
                        FROM match_registry
                        WHERE start_time IS NOT NULL
                        """
                    ).fetchall()

                if matches:
                    all_windows: list[dict[str, object]] = []
                    for match_id, start_time, duration in matches:
                        try:
                            if isinstance(start_time, datetime):
                                dt_start = start_time
                            elif isinstance(start_time, str):
                                if start_time.endswith("Z"):
                                    dt_start = datetime.fromisoformat(start_time[:-1] + "+00:00")
                                elif "+" in start_time or start_time.count("-") > 2:
                                    dt_start = datetime.fromisoformat(start_time)
                                else:
                                    dt_start = datetime.fromisoformat(start_time + "+00:00")
                            else:
                                continue

                            start_epoch = _epoch_seconds_paris(dt_start)
                            if start_epoch is None:
                                continue

                            dur = float(duration or 0) if duration else 12 * 60
                            end_epoch = start_epoch + dur

                            all_windows.append(
                                {
                                    "match_id": str(match_id),
                                    "start_epoch": start_epoch,
                                    "end_epoch": end_epoch,
                                    "start_time": dt_start,
                                }
                            )
                        except Exception:
                            continue

                    if all_windows:
                        return pl.DataFrame(all_windows).sort("start_epoch")
            except Exception:
                pass  # Fallback vers v4

        # --- Fallback V4 : parcourir les DBs joueurs ---
        from src.utils.paths import PLAYER_DB_FILENAME

        all_windows = []

        # Parcourir toutes les DBs joueurs
        if PLAYERS_DIR.exists():
            for player_dir in PLAYERS_DIR.iterdir():
                if not player_dir.is_dir():
                    continue

                player_db = player_dir / PLAYER_DB_FILENAME
                if not player_db.exists():
                    continue

                try:
                    with duckdb_read_only(player_db) as conn:
                        # Vérifier si la table existe
                        tables = conn.execute(
                            """
                            SELECT table_name
                            FROM information_schema.tables
                            WHERE table_schema = 'main'
                            AND table_name = 'match_stats'
                            """
                        ).fetchall()

                        if not tables:
                            continue

                        # Charger les matchs avec start_time
                        matches = conn.execute(
                            """
                            SELECT match_id, start_time, time_played_seconds
                            FROM match_stats
                            WHERE start_time IS NOT NULL
                            """
                        ).fetchall()

                        if matches:
                            for match_id, start_time, time_played in matches:
                                try:
                                    # Convertir start_time
                                    if isinstance(start_time, datetime):
                                        dt_start = start_time
                                    elif isinstance(start_time, str):
                                        if start_time.endswith("Z"):
                                            dt_start = datetime.fromisoformat(
                                                start_time[:-1] + "+00:00"
                                            )
                                        elif "+" in start_time or start_time.count("-") > 2:
                                            dt_start = datetime.fromisoformat(start_time)
                                        else:
                                            dt_start = datetime.fromisoformat(start_time + "+00:00")
                                    else:
                                        continue

                                    # Convertir en epoch Paris
                                    start_epoch = _epoch_seconds_paris(dt_start)
                                    if start_epoch is None:
                                        continue

                                    # Calculer fin
                                    duration = float(time_played or 0) if time_played else 12 * 60
                                    end_epoch = start_epoch + duration

                                    all_windows.append(
                                        {
                                            "match_id": str(match_id),
                                            "start_epoch": start_epoch,
                                            "end_epoch": end_epoch,
                                            "start_time": dt_start,
                                        }
                                    )
                                except Exception:
                                    continue
                except Exception:
                    continue

        if not all_windows:
            return _empty

        return pl.DataFrame(all_windows).sort("start_epoch")

    except Exception:
        return _empty


def _gamertag_from_db_path(db_path: str) -> str | None:
    """Extrait le gamertag (nom du dossier joueur) depuis le chemin de la DB."""
    if not db_path:
        return None
    try:
        from pathlib import Path

        p = Path(db_path)
        # data/players/JGtm/stats.duckdb -> JGtm
        if p.name and p.name.endswith(".duckdb"):
            return p.parent.name or None
        return None
    except Exception:
        return None


def _load_media_from_db(
    db_path: str,
    xuid: str | None = None,
    gamertag: str | None = None,
) -> pl.DataFrame:
    """Charge les médias depuis la BDD DuckDB.

    Args:
        db_path: Chemin vers la DB DuckDB.
        xuid: XUID du joueur pour filtrer les associations (optionnel).
        gamertag: Gamertag (nom du dossier) pour inclure les associations stockées
                  avec le nom du dossier quand sync_meta n'a pas de xuid.

    Returns:
        DataFrame avec colonnes: path, mtime, ext, kind, basename, match_id, match_start_time, xuid
    """
    _col_names = [
        "path",
        "mtime",
        "mtime_paris_epoch",
        "ext",
        "kind",
        "basename",
        "thumbnail_path",
        "match_id",
        "match_start_time",
        "association_confidence",
        "xuid",
    ]
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            # Vérifier si les tables existent
            tables = conn.execute(
                """
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = 'main'
                AND table_name = 'media_files'
                """
            ).fetchall()

            if not tables:
                return pl.DataFrame()

            # Charger les médias avec leurs associations.
            # Si xuid est fourni, on accepte mma.xuid = xuid OU mma.xuid = gamertag
            # (l'indexeur peut stocker le gamertag en fallback quand sync_meta n'a pas de xuid).
            if xuid or gamertag:
                # Inclure les associations que l'indexeur a stockées avec xuid OU gamertag (fallback).
                uids = [u for u in (xuid, gamertag) if u]
                uids = list(dict.fromkeys(uids))  # dédupliquer
                if not uids:
                    uid_filter = "1=0"
                    params: list[str] = []
                elif len(uids) == 1:
                    uid_filter = "mma.xuid = ?"
                    params = [uids[0]]
                else:
                    uid_filter = "(mma.xuid = ? OR mma.xuid = ?)"
                    params = list(uids[:2])
                result = conn.execute(
                    f"""
                    SELECT DISTINCT
                        mf.file_path AS path,
                        mf.mtime,
                        mf.mtime_paris_epoch,
                        mf.file_ext AS ext,
                        mf.kind,
                        mf.file_name AS basename,
                        mf.thumbnail_path,
                        mma.match_id,
                        mma.match_start_time,
                        mma.association_confidence,
                        mma.xuid
                    FROM media_files mf
                    LEFT JOIN media_match_associations mma
                        ON mf.file_path = mma.media_path
                        AND ({uid_filter})
                    ORDER BY mf.mtime_paris_epoch DESC
                    """,
                    params,
                ).fetchall()
            else:
                # Charger tous les médias avec toutes leurs associations
                result = conn.execute(
                    """
                    SELECT DISTINCT
                        mf.file_path AS path,
                        mf.mtime,
                        mf.mtime_paris_epoch,
                        mf.file_ext AS ext,
                        mf.kind,
                        mf.file_name AS basename,
                        mf.thumbnail_path,
                        mma.match_id,
                        mma.match_start_time,
                        mma.association_confidence,
                        mma.xuid
                    FROM media_files mf
                    LEFT JOIN media_match_associations mma
                        ON mf.file_path = mma.media_path
                    ORDER BY mf.mtime_paris_epoch DESC
                    """
                ).fetchall()

            if not result:
                return pl.DataFrame()

            # Construire un pl.DataFrame à partir des tuples
            rows = [dict(zip(_col_names, row, strict=False)) for row in result]
            return pl.DataFrame(rows)

    except Exception:
        return pl.DataFrame()


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
        resolve_xuid(xuid_input or "JGtm", db_path, identity)
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
