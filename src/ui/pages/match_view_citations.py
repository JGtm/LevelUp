"""Section Citations (progressées dans un match) pour la page Match View."""

from __future__ import annotations

import html
import logging
from typing import Any

import polars as pl
import streamlit as st

from src.ui.i18n import get_lang, t
from src.ui.medals import load_medal_description_map, load_medal_name_maps, render_medals_grid

logger = logging.getLogger(__name__)


def render_match_citations_section(  # noqa: C901, PLR0912, PLR0915
    *,
    match_id: str,
    db_path: str,
    xuid: str,
) -> None:
    """Affiche les citations qui ont progressé dans ce match, avec compteur delta."""
    import os as _os

    from src.analysis.citations.engine import CitationEngine
    from src.data.citation_definitions import load_citation_definitions
    from src.ui.commendations import (
        _compute_mastery_display,
        _img_data_uri,
        _img_src,
        _parse_tier_targets,
    )

    logger.debug("Chargement citations match=%s xuid=%s", match_id, xuid)

    citations_db = load_citation_definitions()
    if not citations_db:
        st.caption(t("mv_citations_unavailable"))
        return

    try:
        engine = CitationEngine(db_path, xuid)
        delta_map = engine.aggregate_for_display(match_ids=[match_id])
    except Exception:
        logger.warning("citations: erreur agrégation match=%s", match_id, exc_info=True)
        st.caption(t("mv_citations_no_data"))
        return

    active_norms = {norm for norm, val in delta_map.items() if val > 0}
    if not active_norms:
        st.info(t("citations_no_progress"))
        return

    try:
        full_map = engine.aggregate_for_display(match_ids=None)
    except Exception:
        logger.warning("citations: erreur agrégation complète xuid=%s", xuid)
        full_map = {}

    items = []
    for cit in citations_db:
        norm = cit["citation_name_norm"]
        if norm not in active_norms:
            continue
        tiers = _parse_tier_targets(cit.get("tier_targets"))
        current_full = full_map.get(norm, 0)
        delta = delta_map.get(norm, 0)
        _, _, is_master, _ = _compute_mastery_display(current_full, tiers)
        # Si on est maître ACTUELLEMENT, vérifier si on le was AVANT ce match.
        # Si on ne l'était pas avant (count_before < seuil maître), c'est que
        # ce match a fait atteindre le niveau maître → on l'affiche quand même.
        if is_master:
            count_before = current_full - delta
            _, _, was_master_before, _ = _compute_mastery_display(count_before, tiers)
            if was_master_before:
                continue  # Déjà maître avant ce match → on ne l'affiche pas
        items.append(
            {
                "name": cit["citation_name_display"],
                "norm": norm,
                "description": cit.get("description") or "",
                "image_path": cit.get("image_path"),
                "tiers": tiers,
                "current_full": current_full,
                "delta": delta,
            }
        )

    if not items:
        st.info(t("citations_no_progress"))
        return

    logger.debug("citations: %d items à afficher pour match=%s", len(items), match_id)

    # Grille centrée : padding symétrique si moins de 8 éléments
    n_cols = min(len(items), 8)
    if n_cols < 8:
        _pad = max(1, (8 - n_cols) // 2)
        _all_cols = st.columns([_pad] + [1] * n_cols + [_pad])
        display_cols = _all_cols[1 : 1 + n_cols]
    else:
        display_cols = st.columns(8)

    for i, item in enumerate(items):
        col = display_cols[i % n_cols]
        name = item["name"]
        tiers = item["tiers"]
        current_full = item["current_full"]
        delta = item["delta"]

        level_label, counter_label, is_master, progress_ratio = _compute_mastery_display(
            current_full, tiers
        )

        img = _img_src(item["image_path"])
        data_uri = None
        if img:
            try:
                mtime = _os.path.getmtime(img)
            except OSError:
                mtime = None
            data_uri = _img_data_uri(img, mtime)

        desc = item["description"]
        tip = html.escape(desc) if desc else html.escape(name)

        with col:
            st.markdown("<div class='os-citation-top-gap'></div>", unsafe_allow_html=True)

            if data_uri:
                ring_class = (
                    "os-citation-ring os-citation-ring--master" if is_master else "os-citation-ring"
                )
                ring_color = "#d6b35a" if is_master else "#41d6ff"
                st.markdown(
                    "<div class='"
                    + ring_class
                    + "' title='"
                    + tip
                    + "' style=\"--p:"
                    + str(float(progress_ratio))
                    + ";--ring-color:"
                    + ring_color
                    + ";--img:url('"
                    + data_uri
                    + "')\"></div>",
                    unsafe_allow_html=True,
                )
            else:
                st.markdown(
                    "<div class='os-medal-missing' title='" + tip + "'>?</div>",
                    unsafe_allow_html=True,
                )

            st.markdown(
                "<div class='os-citation-name' title='" + tip + "'>" + html.escape(name) + "</div>",
                unsafe_allow_html=True,
            )
            level_class = (
                "os-citation-level os-citation-level--master" if is_master else "os-citation-level"
            )
            st.markdown(
                f"<div class='{level_class}'>{html.escape(level_label)}</div>",
                unsafe_allow_html=True,
            )

            delta_html = ""
            if delta > 0:
                delta_html = f" <span style='color: #4CAF50; font-weight: bold;'>+{delta}</span>"
            st.markdown(
                "<div class='os-citation-counter'>"
                + html.escape(counter_label)
                + delta_html
                + "</div>",
                unsafe_allow_html=True,
            )


def render_medals_tab(medals_last: list[dict[str, Any]] | None) -> None:
    """Affiche la grille de médailles dans l'onglet Citations & Médailles."""
    st.subheader(t("mv_medals"))
    if not medals_last:
        st.info(t("mv_medals_no_data"))
        return
    md_df = pl.DataFrame(medals_last)
    _fr_map, _en_map = load_medal_name_maps()
    _medal_map = {
        **{str(k): v for k, v in _en_map.items()},
        **{str(k): v for k, v in _fr_map.items()},
    }
    md_df = md_df.with_columns(
        pl.col("name_id")
        .cast(pl.Utf8)
        .replace_strict(_medal_map, default=None, return_dtype=pl.Utf8)
        .fill_null(pl.lit(t("mv_medal_fallback", n="") + " ") + pl.col("name_id").cast(pl.Utf8))
        .alias("label")
    )
    md_df = md_df.sort(["count", "label"], descending=[True, False])
    _lang = get_lang()
    _desc_map = load_medal_description_map(_lang)
    _descriptions = {int(k): v for k, v in _desc_map.items()}
    render_medals_grid(
        md_df.select(["name_id", "count"]).to_dicts(),
        cols_per_row=8,
        center=True,
        lang=_lang,
        descriptions=_descriptions,
    )


__all__ = ["render_match_citations_section", "render_medals_tab"]
