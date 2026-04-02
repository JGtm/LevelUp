"""Composants UI pour les scores de performance.

Ce module contient les fonctions de calcul et d'affichage
des scores de performance pour les sessions de jeu.
"""

from __future__ import annotations

import html as html_mod
from collections.abc import Callable

import polars as pl
import streamlit as st

from src.analysis.performance_config import SCORE_THRESHOLDS, get_score_labels
from src.analysis.performance_score import (
    compute_session_performance_score_v1,
    compute_session_performance_score_v2,
)


def compute_session_performance_score(df_session: pl.DataFrame) -> dict:
    """Calcule un score de performance (0-100) pour une session.

    Le score est une moyenne pondérée de :
    - K/D ratio normalisé (30%)
    - Win rate (25%)
    - Précision moyenne (25%)
    - Score moyen par partie normalisé (20%)

    Args:
        df_session: DataFrame Polars contenant les matchs de la session.

    Returns:
        Dict avec score global et composantes détaillées.
    """
    return compute_session_performance_score_v1(df_session)


def compute_session_performance_score_v2_ui(
    df_session: pl.DataFrame,
    *,
    include_mmr_adjustment: bool = True,
) -> dict:
    """Calcule un score de performance v2 (0–100) pour une session.

    Version pensée pour être réutilisée ailleurs que dans la page de comparaison.

    Args:
        df_session: DataFrame Polars contenant les matchs de la session.
        include_mmr_adjustment: Inclure l'ajustement MMR.
    """
    return compute_session_performance_score_v2(
        df_session,
        include_mmr_adjustment=include_mmr_adjustment,
    )


def get_score_color(score: float | None) -> str:
    """Retourne la couleur CSS selon le score de performance."""
    if score is None:
        return "var(--color-neutral)"
    if score >= SCORE_THRESHOLDS["excellent"]:
        return "var(--color-excellent)"
    if score >= SCORE_THRESHOLDS["good"]:
        return "var(--color-good)"
    if score >= SCORE_THRESHOLDS["average"]:
        return "var(--color-average)"
    if score >= SCORE_THRESHOLDS["below_average"]:
        return "var(--color-poor)"
    return "var(--color-bad)"


def get_score_class(score: float | None) -> str:
    """Retourne la classe CSS selon le score de performance."""
    if score is None:
        return "text-neutral"
    if score >= SCORE_THRESHOLDS["excellent"]:
        return "text-excellent"
    if score >= SCORE_THRESHOLDS["good"]:
        return "text-good"
    if score >= SCORE_THRESHOLDS["average"]:
        return "text-average"
    if score >= SCORE_THRESHOLDS["below_average"]:
        return "text-poor"
    return "text-bad"


def get_score_label(score: float | None, lang: str | None = None) -> str:
    """Retourne le label textuel traduit selon le score."""
    if score is None:
        return "N/A"
    labels = get_score_labels(lang)
    if score >= SCORE_THRESHOLDS["excellent"]:
        return labels["excellent"]
    if score >= SCORE_THRESHOLDS["good"]:
        return labels["good"]
    if score >= SCORE_THRESHOLDS["average"]:
        return labels["average"]
    if score >= SCORE_THRESHOLDS["below_average"]:
        return labels["below_average"]
    return labels["bad"]


def render_performance_score_card(
    label: str,
    perf: dict,
    is_better: bool | None = None,
    compact: bool = False,
) -> None:
    """Affiche une carte avec le score de performance.

    Args:
        label: Titre de la carte (ex: "Session A").
        perf: Dict retourné par compute_session_performance_score.
        is_better: True si cette session est meilleure, False si pire, None si pas de comparaison.
        compact: Si True, affichage réduit sans nb de matchs.
    """
    from src.ui.i18n import get_lang, t

    lang = get_lang()
    score = perf.get("score")
    score_class = get_score_class(score)
    score_label = get_score_label(score, lang=lang)
    score_display = f"{score:.0f}" if score is not None else "—"

    # Indicateur de comparaison
    badge = ""
    if is_better is True:
        badge = "<span class='text-positive text-lg' style='margin-left: 8px;'>▲</span>"
    elif is_better is False:
        badge = "<span class='text-negative text-lg' style='margin-left: 8px;'>▼</span>"

    compact_class = " os-perf-card--compact" if compact else ""
    meta_html = (
        ""
        if compact
        else (
            f"<div class='os-perf-card__meta'>"
            f"{perf.get('matches', 0)} {t('perf_matches_count', lang=lang)}"
            f"</div>"
        )
    )

    st.markdown(
        f"""
        <div class="os-perf-card{compact_class}">
            <div class="os-perf-card__label">{html_mod.escape(str(label))}</div>
            <div class="os-perf-card__score {score_class}">{score_display}{badge}</div>
            <div class="os-perf-card__status {score_class}">{score_label}</div>
            {meta_html}
        </div>
        """,
        unsafe_allow_html=True,
    )


def _render_squad_score_block(squad_result: dict, lang: str | None) -> None:
    """Affiche le bloc score d'équipe + grade (sans Streamlit dans compute_*)."""
    from src.ui.i18n import t

    score = squad_result.get("score")
    grade = squad_result.get("grade", "N/A")
    score_class = get_score_class(score)
    score_display = f"{score:.0f}" if score is not None else "—"

    st.markdown(
        f"""
        <div class="os-perf-card" style="margin-top:1rem;">
            <div class="os-perf-card__label">{t("squad_score_header", lang=lang)}</div>
            <div class="os-perf-card__score {score_class}">{score_display}</div>
            <div class="os-perf-card__status {score_class}"
                 style="font-size:1.6rem;font-weight:700;letter-spacing:.05em;">
                {html_mod.escape(grade)}
            </div>
        </div>
        """,
        unsafe_allow_html=True,
    )


def _render_compact_team_card(team_perf: dict, grade: str, lang: str | None) -> None:
    """Affiche la carte équipe compacte (score + grade + détail bonus) dans la rangée des joueurs."""
    from src.ui.i18n import t

    score = team_perf.get("score")
    score_class = get_score_class(score)
    score_display = f"{score:.0f}" if score is not None else "—"

    base_avg = team_perf.get("components", {}).get("base_avg")
    bonus_html = ""
    if score is not None and base_avg is not None:
        bonus = round(score - base_avg)
        if bonus > 0:
            label = html_mod.escape(
                t("squad_score_bonus", lang=lang, base=f"{base_avg:.0f}", bonus=bonus)
            )
        else:
            label = html_mod.escape(
                t("squad_score_base_only", lang=lang, base=f"{base_avg:.0f}")
            )
        bonus_html = f"<div class='os-perf-card__detail'>{label}</div>"

    st.markdown(
        f"""
        <div class="os-perf-card os-perf-card--compact">
            <div class="os-perf-card__label">{t("squad_score_header", lang=lang)}</div>
            <div class="os-perf-card__score {score_class}">{score_display}</div>
            <div class="os-perf-card__status {score_class}">{html_mod.escape(grade)}</div>
            {bonus_html}
        </div>
        """,
        unsafe_allow_html=True,
    )


def render_squad_session_header(
    players_data: list[tuple[str, object]],
    lang: str | None = None,
) -> None:
    """Affiche l'en-tête de session escouade : cartes individuelles + score collectif.

    Args:
        players_data: liste de (name, df_session) pour chaque joueur.
        lang: langue active pour les traductions.
    """

    from src.analysis._performance_squad import compute_squad_performance_score
    from src.visualization._compat import ensure_polars

    perf_list = [
        (name, compute_session_performance_score_v2_ui(ensure_polars(df)))
        for name, df in players_data
        if df is not None
    ]
    if not perf_list:
        return

    squad_result = compute_squad_performance_score([p for _, p in perf_list])
    avg_score = squad_result.get("score")
    grade = squad_result.get("grade", "N/A")

    cols = st.columns(len(perf_list) + 1)
    with cols[0]:
        team_perf = {**squad_result, "score": avg_score}
        _render_compact_team_card(team_perf, grade, lang)

    for col, (name, perf) in zip(cols[1:], perf_list, strict=False):
        with col:
            player_score = perf.get("score")
            is_better: bool | None = None
            if avg_score is not None and player_score is not None:
                is_better = player_score > avg_score
            render_performance_score_card(label=name, perf=perf, is_better=is_better, compact=True)


def render_metric_comparison_row(
    label: str,
    val_a,
    val_b,
    fmt: str | Callable = "{}",
    higher_is_better: bool = True,
) -> None:
    """Affiche une ligne de métrique avec comparaison colorée.

    Args:
        label: Nom de la métrique.
        val_a: Valeur pour la session A.
        val_b: Valeur pour la session B.
        fmt: Format string pour l'affichage OU une fonction callable.
        higher_is_better: Si True, la valeur la plus haute est verte.
    """
    col1, col2, col3 = st.columns([2, 1, 1])

    # Déterminer les classes CSS
    class_a, class_b = "os-metric-value--neutral", "os-metric-value--neutral"
    if val_a is not None and val_b is not None:
        if higher_is_better:
            if val_a > val_b:
                class_a = "os-metric-value--better"
            elif val_b > val_a:
                class_b = "os-metric-value--better"
        else:
            if val_a < val_b:
                class_a = "os-metric-value--better"
            elif val_b < val_a:
                class_b = "os-metric-value--better"

    # Fonction de formatage
    def _format_value(val):
        if val is None:
            return "—"
        if callable(fmt):
            return fmt(val)
        return fmt.format(val)

    with col1:
        st.markdown(f"**{label}**")
    with col2:
        display_a = _format_value(val_a)
        st.markdown(
            f"<span class='os-metric-value {class_a}'>{display_a}</span>",
            unsafe_allow_html=True,
        )
    with col3:
        display_b = _format_value(val_b)
        st.markdown(
            f"<span class='os-metric-value {class_b}'>{display_b}</span>",
            unsafe_allow_html=True,
        )
