"""Construction et rendu du bloc rang LUSR/CSR pour la vue match."""

from __future__ import annotations

import html
import logging

import streamlit as st

from src.data.domain.refdata import Outcome
from src.ui.i18n import t

logger = logging.getLogger(__name__)

__all__ = ["_build_match_rank_html", "_render_match_rank_tab"]


def _build_match_rank_html(  # noqa: C901, PLR0912, PLR0915
    *,
    match_id: str,
    db_path: str,
    db_key: tuple[int, int] | None = None,
    had_bot_teammate: bool = False,
    bot_outcome: int | None = None,
) -> str | None:
    """Construit le HTML du bloc rang LUSR/CSR pour un match.

    Args:
        had_bot_teammate: Si True, ajoute une note contextuelle bot sous le delta.
        bot_outcome: Outcome du joueur (2 = victoire) pour choisir la note bot.

    Returns:
        Chaîne HTML prête à injecter, ou None si pas de données.
    """
    logger.debug("Construction rang HTML match=%s", match_id)
    try:
        from pathlib import Path

        from src.analysis.skill_rating_config import (
            UNRANKED_COLOR,
            UNRANKED_LABEL,
            get_rank_image_path,
            get_sub_tier_start,
            get_tier_for_rating,
            get_tier_size,
        )
        from src.ui.cache_loaders import cached_get_match_skill_rank
    except ImportError:
        return None

    row_rank = cached_get_match_skill_rank(db_path, match_id, db_key=db_key)
    if row_rank is None:
        _msg = html.escape(t("mv_rating_pending"))
        _hint = html.escape(t("mv_rating_pending_hint"))
        return (
            "<div style='display:flex;align-items:center;gap:16px'>"
            "<span style='color:#555;font-size:3.2em;line-height:1'>?</span>"
            "<div style='flex:1;min-width:0'>"
            f"<div style='font-size:1.1em;font-weight:600;color:#888'>{_msg}</div>"
            f"<div style='font-size:0.82em;color:#555;margin-top:4px'>{_hint}</div>"
            "</div>"
            "</div>"
        )

    (
        rating_type,
        rating_value,
        rating_deviation,
        tier_label,
        sub_tier,
        tier_name,
        tier_fr,
        rating_delta,
        playlist_group,
    ) = row_rank

    from src.ui.player_assets import file_to_data_url

    tier_display = html.escape(tier_label or UNRANKED_LABEL)
    rating_type_badge = html.escape("CSR" if rating_type == "CSR" else "LUSR")
    rating_val_str = f"{rating_value:.0f}" if rating_value is not None else "-"

    # Groupe traduit
    group_translated = ""
    if playlist_group:
        group_key = f"mv_pg_{playlist_group.lower()}"
        group_translated = t(group_key)
        if group_translated == group_key:
            group_translated = playlist_group.capitalize()

    rating_line = html.escape(
        f"{rating_type_badge} {group_translated} : {rating_val_str}"
        if group_translated
        else f"{rating_type_badge} • {rating_val_str}"
    )

    # Image du rang en data URI
    img_html = f"<span style='color:{UNRANKED_COLOR};font-size:3em'>◆</span>"
    img_path_rel = get_rank_image_path(rating_value) if rating_value else None
    if img_path_rel:
        img_full = Path(__file__).resolve().parents[3] / img_path_rel
        if img_full.exists():
            data_url = file_to_data_url(str(img_full))
            if data_url:
                img_html = (
                    f"<img src='{data_url}' style='width:110px;height:110px;object-fit:contain'>"
                )

    # Delta — formatage avec 1 décimale si |delta| < 5, entier sinon
    delta_html = ""
    if rating_delta is not None:
        _abs = abs(rating_delta)
        if _abs < 0.05:
            delta_html = "<div style='color:#888888;font-size:1.05em;margin-top:4px'>= 0 pts</div>"
        else:
            _delta_color = "#50C878" if rating_delta > 0 else "#FF4444"
            _delta_sign = "+" if rating_delta > 0 else "-"
            # 1 décimale si petit delta (< 5 pts), entier sinon
            _delta_fmt = f"{_abs:.1f}" if _abs < 5 else f"{round(_abs)}"
            delta_html = (
                f"<div style='color:{_delta_color};font-size:1.05em;margin-top:4px'>"
                f"{_delta_sign}{_delta_fmt} pts</div>"
            )

    # Barre de progression avec marqueur de delta
    progress_html = ""
    if rating_value is not None:
        tier_obj, sub = get_tier_for_rating(rating_value)
        if tier_obj and tier_obj.sub_tiers > 1:
            tier_size = get_tier_size(rating_value)
            sub_start = get_sub_tier_start(rating_value)
            if tier_size > 0:
                pct = min(100.0, max(0.0, (rating_value - sub_start) / tier_size * 100))
                tier_sz = f"{tier_size:.0f}"

                # Calcul du marqueur delta sur la barre
                bar_inner = ""
                if rating_delta is not None and abs(rating_delta) >= 0.05:
                    # Largeur min garantie de 1 % pour qu'1 pixel soit visible
                    _MIN_PCT = 1.0
                    delta_pct = abs(rating_delta) / tier_size * 100
                    delta_width = max(_MIN_PCT, min(delta_pct, 100.0))
                    _band_color = "#50C878" if rating_delta > 0 else "#FF4444"
                    if rating_delta > 0:
                        # Gain : remplissage bleu jusqu'à (pct - delta), puis bande verte
                        base_pct = max(0.0, pct - delta_pct)
                        bar_inner = (
                            f"<div style='position:absolute;left:0;top:0;"
                            f"width:{base_pct:.2f}%;height:8px;background:#33d6ff'></div>"
                            f"<div style='position:absolute;left:{base_pct:.2f}%;top:0;"
                            f"width:{delta_width:.2f}%;height:8px;background:{_band_color}'></div>"
                        )
                    else:
                        # Perte : remplissage bleu jusqu'à pct, puis bande rouge
                        bar_inner = (
                            f"<div style='position:absolute;left:0;top:0;"
                            f"width:{pct:.2f}%;height:8px;background:#33d6ff'></div>"
                            f"<div style='position:absolute;left:{pct:.2f}%;top:0;"
                            f"width:{delta_width:.2f}%;height:8px;background:{_band_color}'></div>"
                        )
                else:
                    # Pas de delta : barre simple
                    bar_inner = (
                        f"<div style='position:absolute;left:0;top:0;"
                        f"width:{pct:.2f}%;height:8px;background:#33d6ff'></div>"
                    )

                progress_html = f"""
<div style='display:flex;align-items:center;gap:6px;margin-top:8px'>
  <span style='font-size:0.75em;color:#888;min-width:14px;text-align:right'>0</span>
  <div style='position:relative;flex:1;height:8px;background:rgba(255,255,255,0.12);border-radius:4px;overflow:hidden'>
    {bar_inner}
  </div>
  <span style='font-size:0.75em;color:#888;min-width:14px'>{tier_sz}</span>
</div>"""

    # Note contextuelle si coéquipier bot détecté
    bot_note_html = ""
    if had_bot_teammate:
        if bot_outcome == Outcome.WIN:
            _note_text = html.escape(t("mv_lusr_bot_win"))
            bot_note_html = (
                f"<div style='color:#50C878;font-size:0.82em;margin-top:6px'>💪 {_note_text}</div>"
            )
        else:
            _note_text = html.escape(t("mv_lusr_bot_loss"))
            bot_note_html = (
                f"<div style='color:#FFB347;font-size:0.82em;margin-top:6px'>⚠️ {_note_text}</div>"
            )

    return (
        f"<div style='display:flex;align-items:center;gap:16px'>"
        f"{img_html}"
        f"<div style='flex:1;min-width:0'>"
        f"<div style='font-size:1.4em;font-weight:700;line-height:1.2'>{tier_display}</div>"
        f"{progress_html}"
        f"{delta_html}"
        f"{bot_note_html}"
        f"<div style='color:#ffffff;font-size:1.2em;font-weight:bold;margin-top:4px'>{rating_line}</div>"
        f"</div>"
        f"</div>"
    )


def _render_match_rank_tab(
    *,
    match_id: str,
    db_path: str,
    db_key: tuple[int, int] | None = None,
    had_bot_teammate: bool = False,
    bot_outcome: int | None = None,
) -> None:
    """Affiche l'onglet 🏅 Rang pour un match (LUSR ou CSR).

    Wrapper qui appelle ``_build_match_rank_html`` et rend le résultat.
    """
    rank_html = _build_match_rank_html(
        match_id=match_id,
        db_path=db_path,
        db_key=db_key,
        had_bot_teammate=had_bot_teammate,
        bot_outcome=bot_outcome,
    )
    if rank_html is None:
        st.info(t("mv_no_rating"))
        return
    st.markdown(rank_html, unsafe_allow_html=True)
