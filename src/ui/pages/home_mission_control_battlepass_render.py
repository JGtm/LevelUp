"""Rendu battle pass de la home Mission Control V7."""

from __future__ import annotations

import base64
import hashlib
from collections.abc import Callable, Sequence
from html import escape

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.home_mission_control_api import HomeBattlepassInfo
from src.ui.pages.home_mission_control_battlepass import BattlepassTierPreview

RenderCard = Callable[[str], None]
_TIER_WINDOW_REWARD_BUDGET = 10


def render_battlepass_card(info: HomeBattlepassInfo | None, render_card: RenderCard) -> None:
    """Affiche la carte de progression du pass de combat."""
    title = t("v7_home_battlepass")
    if info is None:
        render_card(
            "".join(
                [
                    f"<div class='v7-subshell-title'>{escape(title)}</div>",
                    f"<div class='v7-inline-note'>{escape(t('v7_home_api_unavailable'))}</div>",
                ]
            )
        )
        return

    if info.track_image_bytes:
        st.image(info.track_image_bytes, width="stretch")

    tiers = _resolve_browser_tiers(info)
    focus_index = _resolve_focus_index(info, tiers)
    tier_label = t("v7_home_battlepass_premium") if info.is_owned else t("v7_home_battlepass_free")
    rows = [
        f"<div class='v7-subshell-title'>{escape(title)}</div>",
        f"<div class='v7-home-meta'>{escape(info.track_name)} · {escape(tier_label)}</div>",
        "<div class='v7-home-stats'>",
        f"<span class='v7-home-stat'><strong>Niv. {info.op_rank}</strong></span>",
        "</div>",
        _battlepass_progress_html(info),
    ]
    rows.extend(_battlepass_detail_rows(info, tiers=tiers, focus_index=focus_index))
    render_card("".join(rows))
    _render_battlepass_navigation(info, tiers, focus_index)


def _battlepass_detail_rows(
    info: HomeBattlepassInfo,
    *,
    tiers: Sequence[BattlepassTierPreview],
    focus_index: int,
) -> list[str]:
    if not tiers:
        return []
    return [
        f"<div class='v7-home-action-kicker'>{escape(t('v7_home_battlepass_tiers'))}</div>",
        _battlepass_browser_panel_html(info, tiers, focus_index),
    ]


def _render_battlepass_navigation(
    info: HomeBattlepassInfo,
    tiers: Sequence[BattlepassTierPreview],
    focus_index: int,
) -> None:
    if len(tiers) <= 1:
        return
    state_key = _browser_state_key(info, tiers)
    prev_disabled = focus_index <= 0
    next_disabled = focus_index >= len(tiers) - 1
    prev_cols = st.columns([1.0, 1.4, 1.0])
    with prev_cols[0]:
        if st.button(
            t("v7_home_battlepass_prev"),
            key=f"{state_key}_prev",
            disabled=prev_disabled,
            width="stretch",
        ):
            st.session_state[state_key] = tiers[focus_index - 1].rank
            st.rerun()
    with prev_cols[1]:
        st.markdown(
            (
                "<div style='padding-top:6px;text-align:center;'>"
                f"{_battlepass_browser_status_html(tiers, focus_index)}"
                "</div>"
            ),
            unsafe_allow_html=True,
        )
    with prev_cols[2]:
        if st.button(
            t("v7_home_battlepass_next"),
            key=f"{state_key}_next",
            disabled=next_disabled,
            width="stretch",
        ):
            st.session_state[state_key] = tiers[focus_index + 1].rank
            st.rerun()


def _battlepass_browser_panel_html(
    info: HomeBattlepassInfo,
    tiers: Sequence[BattlepassTierPreview],
    focus_index: int,
) -> str:
    window = _select_tier_window(tiers, focus_index)
    parts = [
        "<div class='bp-browser-window' style='display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px;'>"
    ]
    for tier in window:
        parts.append(
            _battlepass_tier_html(
                tier,
                is_focus=tier.rank == tiers[focus_index].rank,
                is_current=tier.rank == info.op_rank,
            )
        )
    parts.append("</div>")
    return "".join(parts)


def _battlepass_browser_status_html(
    tiers: Sequence[BattlepassTierPreview],
    focus_index: int,
) -> str:
    max_rank = tiers[-1].rank if tiers else 0
    return (
        "<span style='font-size:12px;color:rgba(250,250,250,0.68);'>"
        f"{escape(t('v7_home_battlepass_focus', rank=tiers[focus_index].rank, max_rank=max_rank))}"
        "</span>"
    )


def _battlepass_progress_html(info: HomeBattlepassInfo) -> str:
    if info.xp_per_rank is None:
        return ""
    if info.has_reached_max_rank:
        label = escape(t("v7_home_battlepass_max"))
        return (
            "<div style='margin:10px 0 4px;'>"
            f"<div style='font-size:12px;color:rgba(250,250,250,0.72);margin-bottom:6px;'>{label}</div>"
            "<div style='height:10px;border-radius:999px;background:rgba(255,255,255,0.14);overflow:hidden;'>"
            "<div style='width:100%;height:100%;background:linear-gradient(90deg,#ffffff 0%,#f1f5f9 100%);'></div>"
            "</div></div>"
        )
    current_xp = max(0, min(info.partial_progress or 0, info.xp_per_rank))
    fill_pct = (current_xp / info.xp_per_rank) * 100 if info.xp_per_rank > 0 else 0
    label = escape(t("v7_home_battlepass_tier_progress"))
    return (
        "<div style='margin:10px 0 4px;'>"
        f"<div style='display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:6px;'>"
        f"<span style='font-size:12px;color:rgba(250,250,250,0.72);'>{label}</span>"
        f"<span style='font-size:12px;color:#ffffff;'><strong>{current_xp}/{info.xp_per_rank}</strong> XP</span>"
        "</div>"
        "<div style='height:10px;border-radius:999px;background:rgba(255,255,255,0.12);overflow:hidden;box-shadow:inset 0 0 0 1px rgba(255,255,255,0.06);'>"
        f"<div style='width:{fill_pct:.1f}%;height:100%;background:linear-gradient(90deg,#ffffff 0%,#e2e8f0 100%);'></div>"
        "</div></div>"
    )


def _resolve_browser_tiers(info: HomeBattlepassInfo) -> tuple[BattlepassTierPreview, ...]:
    if info.tiers:
        return info.tiers
    by_rank: dict[int, BattlepassTierPreview] = {}
    for tier in (*info.recent_unlocks, *((info.next_unlock,) if info.next_unlock else ())):
        by_rank.setdefault(tier.rank, tier)
    return tuple(by_rank[rank] for rank in sorted(by_rank))


def _resolve_focus_index(
    info: HomeBattlepassInfo,
    tiers: Sequence[BattlepassTierPreview],
) -> int:
    if not tiers:
        return 0
    default_index = _find_focus_index(tiers, info.op_rank)
    state_key = _browser_state_key(info, tiers)
    selected_rank = st.session_state.get(state_key)
    if selected_rank is None:
        selected_rank = tiers[default_index].rank
        st.session_state[state_key] = selected_rank
    for index, tier in enumerate(tiers):
        if tier.rank == selected_rank:
            return index
    st.session_state[state_key] = tiers[default_index].rank
    return default_index


def _find_focus_index(tiers: Sequence[BattlepassTierPreview], current_rank: int) -> int:
    for index, tier in enumerate(tiers):
        if tier.rank >= current_rank:
            return index
    return max(len(tiers) - 1, 0)


def _select_tier_window(
    tiers: Sequence[BattlepassTierPreview],
    focus_index: int,
) -> tuple[BattlepassTierPreview, ...]:
    if not tiers:
        return ()
    start = max(focus_index - 1, 0)
    end = min(len(tiers), focus_index + 2)
    selected = list(tiers[start:end])
    reward_budget = sum(_tier_weight(tier) for tier in selected)
    cursor = end
    while cursor < len(tiers):
        candidate = tiers[cursor]
        candidate_weight = _tier_weight(candidate)
        if reward_budget + candidate_weight > _TIER_WINDOW_REWARD_BUDGET:
            break
        selected.append(candidate)
        reward_budget += candidate_weight
        cursor += 1
    return tuple(selected)


def _tier_weight(tier: BattlepassTierPreview) -> int:
    return tier.total_rewards if tier.total_rewards > 0 else 1


def _browser_state_key(
    info: HomeBattlepassInfo,
    tiers: Sequence[BattlepassTierPreview],
) -> str:
    payload = f"{info.track_name}:{info.op_rank}:{len(tiers)}:{tiers[-1].rank if tiers else 0}"
    digest = hashlib.sha1(payload.encode("utf-8")).hexdigest()[:10]
    return f"bp-browser-{digest}"


def _battlepass_tier_html(
    tier: BattlepassTierPreview,
    *,
    is_focus: bool,
    is_current: bool,
) -> str:
    container_border = "rgba(255,255,255,0.30)" if is_focus else "rgba(255,255,255,0.10)"
    container_bg = "rgba(255,255,255,0.09)" if is_focus else "rgba(255,255,255,0.03)"
    badges = []
    if is_current:
        badges.append(
            f"<span style='display:inline-flex;align-items:center;height:18px;padding:0 7px;border-radius:999px;background:rgba(255,255,255,0.14);color:#ffffff;font-size:10px;font-weight:700;'>{escape(t('v7_home_battlepass_current'))}</span>"
        )
    rewards_html = []
    for reward in tier.free_rewards:
        rewards_html.append(_battlepass_reward_html(reward, premium=False))
    for reward in tier.premium_rewards:
        rewards_html.append(_battlepass_reward_html(reward, premium=True))
    if not rewards_html:
        rewards_html.append(
            f"<div style='font-size:11px;color:rgba(250,250,250,0.58);padding:4px 0;'>{escape(t('v7_home_battlepass_empty'))}</div>"
        )
    return "".join(
        [
            f"<div style='padding:10px;border-radius:14px;border:1px solid {container_border};background:{container_bg};min-width:0;'>",
            "<div style='display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:8px;'>",
            f"<div style='font-size:12px;color:rgba(250,250,250,0.78);'><strong>Palier {tier.rank}</strong></div>",
            f"<div style='display:flex;align-items:center;gap:6px;'>{''.join(badges)}</div>",
            "</div>",
            "<div style='display:flex;gap:8px;flex-wrap:wrap;padding:2px 0 0;'>",
            *rewards_html,
            "</div>",
            "</div>",
        ]
    )


def _battlepass_reward_html(reward: object, *, premium: bool) -> str:
    border = "rgba(241, 196, 15, 0.85)" if premium else "rgba(96, 165, 250, 0.8)"
    background = "rgba(241, 196, 15, 0.10)" if premium else "rgba(96, 165, 250, 0.10)"
    badge_background = "rgba(241, 196, 15, 0.92)" if premium else "rgba(96, 165, 250, 0.92)"
    badge_text = "P" if premium else "F"
    tooltip = escape(_battlepass_reward_tooltip(reward), quote=True)
    if reward.image_bytes:
        encoded = base64.b64encode(reward.image_bytes).decode()
        body = (
            f"<img src='data:image/png;base64,{encoded}' alt='' "
            "style='width:100%;height:100%;display:block;object-fit:cover;'>"
        )
    else:
        label = escape(reward.tile_label or reward.label)
        body = (
            "<div style='width:100%;height:100%;display:flex;align-items:center;justify-content:center;"
            "padding:6px;font-size:11px;font-weight:700;line-height:1.05;text-align:center;color:#f8fafc;'>"
            f"{label}</div>"
        )
    return (
        f"<div title='{tooltip}' style='position:relative;flex:0 0 auto;width:56px;height:56px;border-radius:12px;"
        f"overflow:hidden;border:1px solid {border};background:{background};box-shadow:inset 0 0 0 1px rgba(255,255,255,0.04);'>"
        f"<div style='position:absolute;top:4px;right:4px;z-index:1;min-width:16px;height:16px;padding:0 4px;border-radius:999px;background:{badge_background};color:#081018;font-size:9px;font-weight:800;display:flex;align-items:center;justify-content:center;'>{badge_text}</div>"
        f"{body}</div>"
    )


def _battlepass_reward_tooltip(reward: object) -> str:
    parts = [reward.label]
    if reward.description:
        parts.append(reward.description)
    return "\n".join(part for part in parts if part)
