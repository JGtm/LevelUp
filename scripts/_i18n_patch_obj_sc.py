# -*- coding: utf-8 -*-
"""Script temporaire : câblage i18n objective_analysis.py + session_compare.py."""

# ── objective_analysis.py ────────────────────────────────────────────────────
with open("src/ui/pages/objective_analysis.py", encoding="utf-8") as f:
    c = f.read()

if "from src.ui.i18n import t" not in c:
    c = c.replace(
        "import streamlit as st\n",
        "import streamlit as st\n\nfrom src.ui.i18n import t\n",
        1,
    )
    print("obj Import OK")

repl = [
    ('st.title("📊 Analyse des Objectifs")', 'st.title(t("obj_analysis_title"))'),
    (
        'st.error("⚠️ Cette page nécessite Polars. " "Installez-le avec: `pip install polars`")',
        'st.error(t("obj_polars_missing"))',
    ),
    ('with st.spinner("Chargement des données..."):', 'with st.spinner(t("obj_loading")):'),
    (
        'st.error(f"Erreur lors du chargement des données: {e}")',
        'st.error(t("error_loading", error=e))',
    ),
    (
        'st.info("Données insuffisantes pour la corrélation objectifs/kills.")',
        'st.info(t("insufficient_data_chart"))',
    ),
    (
        "st.warning(f\"Impossible d'afficher la corrélation objectifs/kills : {e}\")",
        'st.warning(t("error_chart", error=e))',
    ),
    (
        'st.info("Données insuffisantes pour la répartition.")',
        'st.info(t("insufficient_data_chart"))',
    ),
    (
        "st.warning(f\"Impossible d'afficher la répartition : {e}\")",
        'st.warning(t("error_chart", error=e))',
    ),
    ('st.info("Impossible de générer la jauge.")', 'st.info(t("insufficient_data_chart"))'),
    (
        "st.warning(f\"Impossible d'afficher le ratio objectifs : {e}\")",
        'st.warning(t("error_chart", error=e))',
    ),
    (
        'st.info("Données insuffisantes pour la tendance.")',
        'st.info(t("insufficient_data_chart"))',
    ),
    (
        "st.warning(f\"Impossible d'afficher la tendance : {e}\")",
        'st.warning(t("error_chart", error=e))',
    ),
    (
        'st.info("Pas assez de données pour afficher la tendance.")',
        'st.info(t("insufficient_data_chart"))',
    ),
    (
        "st.caption(\"Décomposition des différents types d'assistances.\")",
        'st.caption(t("obj_assists_caption"))',
    ),
    (
        "st.info(\"Aucune donnée d'assistance disponible.\")",
        'st.info(t("obj_no_assists"))',
    ),
    (
        'st.info("Aucun award objectif enregistré.")',
        'st.info(t("obj_no_awards"))',
    ),
    (
        'st.info("Aucun award enregistré.")',
        'st.info(t("obj_no_awards_generic"))',
    ),
    (
        'st.caption("Top joueurs rencontrés par contribution aux objectifs.")',
        'st.caption(t("obj_top_opponents_caption"))',
    ),
    (
        'with st.expander("🔜 Comparaison (à venir)", expanded=False):',
        'with st.expander(t("obj_comparison_coming_soon"), expanded=False):',
    ),
    (
        "st.error(f\"Erreur lors de l'ouverture de la base: {e}\")",
        'st.error(t("error_loading", error=e))',
    ),
]

count = 0
for old, new in repl:
    if old in c:
        c = c.replace(old, new, 1)
        count += 1
    else:
        print(f"obj NOT FOUND: {old[:70]!r}")

print(f"obj Done: {count}/{len(repl)}")
with open("src/ui/pages/objective_analysis.py", "w", encoding="utf-8") as f:
    f.write(c)


# ── session_compare.py ───────────────────────────────────────────────────────
with open("src/ui/pages/session_compare.py", encoding="utf-8") as f:
    c = f.read()

if "from src.ui.i18n import t" not in c:
    c = c.replace(
        "import streamlit as st\n",
        "import streamlit as st\n\nfrom src.ui.i18n import t\n",
        1,
    )
    print("sc Import OK")

repl2 = [
    (
        '    st.caption("Compare les performances entre deux sessions de jeu.")',
        '    st.caption(t("sc_loading_caption"))',
    ),
    (
        '        st.info("Aucune session disponible.")',
        '        st.info(t("sc_no_sessions"))',
    ),
    (
        '        st.warning("Il faut au moins 2 sessions pour comparer.")',
        '        st.warning(t("sc_need_two_sessions"))',
    ),
    (
        '            st.markdown("#### Net score cumulé par session")',
        '            st.markdown(t("sc_net_score_cumul"))',
    ),
    (
        '                    st.info("Données insuffisantes pour le net score cumulé.")',
        '                    st.info(t("insufficient_data_chart"))',
    ),
    (
        "                st.warning(f\"Impossible d'afficher le net score cumulé : {e}\")",
        '                st.warning(t("error_chart", error=e))',
    ),
    (
        '    st.markdown("### 🏆 Score de performance")',
        '    st.markdown(t("sc_performance_score"))',
    ),
    (
        '    st.markdown("### 📊 Métriques détaillées")',
        '    st.markdown(t("sc_detailed_metrics"))',
    ),
    (
        '    st.markdown("### 🎯 Comparaison MMR")',
        '    st.markdown(t("sc_mmr_comparison"))',
    ),
    (
        '    st.markdown("### 📈 Graphiques comparatifs")',
        '    st.markdown(t("sc_comparative_charts"))',
    ),
    (
        '    st.markdown("### 📋 Historique des parties")',
        '    st.markdown(t("sc_match_history"))',
    ),
    (
        '    st.markdown("#### 🎯 Évolution du profil de participation")',
        '    st.markdown(t("sc_participation_profile"))',
    ),
    (
        '    st.markdown("#### Vue radar")',
        '    st.markdown(t("sc_radar_view"))',
    ),
    (
        '    st.markdown("#### Comparaison par métrique")',
        '    st.markdown(t("sc_metric_comparison"))',
    ),
]

count2 = 0
for old, new in repl2:
    if old in c:
        c = c.replace(old, new, 1)
        count2 += 1
    else:
        print(f"sc NOT FOUND: {old[:70]!r}")

print(f"sc Done: {count2}/{len(repl2)}")
with open("src/ui/pages/session_compare.py", "w", encoding="utf-8") as f:
    f.write(c)
