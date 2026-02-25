# -*- coding: utf-8 -*-
"""Script fixup final : corrige les 35 strings FR restants dans les pages."""
from pathlib import Path


def patch(path: str, replacements: list[tuple[str, str]]) -> None:
    p = Path(path)
    if not p.exists():
        print(f"  SKIP (not found) {p.name}")
        return
    c = p.read_text(encoding="utf-8")
    if "from src.ui.i18n import t" not in c:
        c = c.replace(
            "import streamlit as st\n",
            "import streamlit as st\n\nfrom src.ui.i18n import t\n",
            1,
        )
    count = 0
    for old, new in replacements:
        if old in c:
            c = c.replace(old, new, 1)
            count += 1
        else:
            print(f"  NOT FOUND [{p.name}]: {old[:70]!r}")
    print(f"  {p.name}: {count}/{len(replacements)}")
    p.write_text(c, encoding="utf-8")


# objective_analysis.py — string restants
patch("src/ui/pages/objective_analysis.py", [
    (
        "    st.warning(f\"\u26a0\ufe0f Aucune donnée pour le joueur (XUID: {xuid}).\")",
        '    st.warning(t("obj_no_player_data", xuid=xuid))',
    ),
    (
        "    st.info(f\"**Profil détecté:** {profile}\\n\\n{profile_desc}\")",
        "    st.info(f\"**{t('obj_profile_label')}:** {profile}\\n\\n{profile_desc}\")",
    ),
    (
        "    st.markdown(\"## \U0001f4c8 Analyse détaillée\")",
        "    st.markdown(f\"## {t('obj_analysis_detailed')}\")",
    ),
    (
        "        st.markdown(\"### Corrélation Objectifs / Kills\")",
        "        st.markdown(f\"### {t('obj_correlation_title')}\")",
    ),
    (
        "            st.markdown(\"### Répartition par Catégorie\")",
        "            st.markdown(f\"### {t('obj_breakdown_title')}\")",
    ),
    (
        "        st.markdown(\"### Évolution dans le temps\")",
        "        st.markdown(f\"### {t('obj_trend_title')}\")",
    ),
    (
        "            st.markdown(\"### Détail par type\")",
        "            st.markdown(f\"### {t('obj_assist_detail')}\")",
    ),
    (
        "    st.markdown(\"## \U0001f3c5 Awards les plus fréquents\")",
        "    st.markdown(f\"## {t('obj_awards_frequent')}\")",
    ),
    (
        "    st.markdown(\"## \U0001f4a1 Conseils personnalisés\")",
        "    st.markdown(f\"## {t('obj_tips')}\")",
    ),
    (
        "    st.error(\"\u26a0\ufe0f Veuillez d'abord sélectionner un profil joueur.\")",
        "    st.error(t(\"obj_no_player_selected\"))",
    ),
])

# teammates_charts.py
patch("src/ui/pages/teammates_charts.py", [
    (
        "            st.warning(\"Impossible de charger les stats du coéquipier sur les matchs partagés.\")",
        "            st.warning(t(\"error_chart\", error=\"charger les stats du coéquipier\"))",
    ),
    (
        "        st.info(\"Aucune donnée de folie meurtrière (max) sur ces matchs.\")",
        "        st.info(t(\"insufficient_data_chart\"))",
    ),
    (
        "        st.info(\"Aucune donnée de tirs à la tête sur ces matchs.\")",
        "        st.info(t(\"insufficient_data_chart\"))",
    ),
    (
        "        st.info(\"Aucune donnée de frags parfaits sur ces matchs.\")",
        "        st.info(t(\"insufficient_data_chart\"))",
    ),
])

# match_view_charts.py
patch("src/ui/pages/match_view_charts.py", [
    (
        "    st.subheader(\"Réel vs attendu\")",
        "    st.subheader(t(\"mv_vs_expected\"))",
    ),
    (
        '        st.info("Données insuffisantes pour le graphique F/M/A.")',
        '        st.info(t("insufficient_data_chart"))',
    ),
    (
        '    st.subheader("Folie meurtrière / Tirs à la tête / Frags parfaits")',
        '    st.subheader(t("ts_spree"))',
    ),
    (
        '                st.info("Données insuffisantes pour le graphique Spree/Headshots.")',
        '                st.info(t("insufficient_data_chart"))',
    ),
])

# session_compare_charts.py
patch("src/ui/pages/session_compare_charts.py", [
    (
        '            st.info("Impossible de générer le radar de comparaison.")',
        '            st.info(t("insufficient_data_chart"))',
    ),
    (
        '            st.info("Données insuffisantes pour le graphique comparatif.")',
        '            st.info(t("insufficient_data_chart"))',
    ),
    (
        '                    st.info("Impossible de générer le radar de participation.")',
        '                    st.info(t("insufficient_data_chart"))',
    ),
])

# match_view_helpers.py — subheader Médias x2 + caption Vidéos
patch("src/ui/pages/match_view_helpers.py", [
    (
        '    st.subheader("Médias")',
        '    st.subheader(t("mv_media_title"))',
    ),
    (
        '    st.caption("Vidéos")',
        '    st.caption(t("mv_videos_title"))',
    ),
])
# Second occurrence of st.subheader("Médias") — same string, second replace
with open("src/ui/pages/match_view_helpers.py", encoding="utf-8") as f:
    c = f.read()
if '    st.subheader("Médias")' in c:
    c = c.replace('    st.subheader("Médias")', '    st.subheader(t("mv_media_title"))', 1)
    with open("src/ui/pages/match_view_helpers.py", "w", encoding="utf-8") as f:
        f.write(c)
    print("  match_view_helpers.py: second Médias occurrence fixed")

# teammates.py
patch("src/ui/pages/teammates.py", [
    (
        '        st.caption("F/M 1ère moitié \u2192 2nde moitié des matchs affichés.")',
        '        st.caption(t("tm_kd_half_caption"))',
    ),
    (
        '        st.info("Sélectionne au moins un coéquipier.")',
        '        st.info(t("tm_select_teammate"))',
    ),
])

# session_compare.py
patch("src/ui/pages/session_compare.py", [
    (
        '        st.markdown("**Métrique**")',
        '        st.markdown(t("sc_metric_label"))',
    ),
    (
        '        st.markdown("**Métrique MMR**")',
        '        st.markdown(t("sc_mmr_metric_label"))',
    ),
])

# timeseries.py
patch("src/ui/pages/timeseries.py", [
    (
        '        st.info(f"Colonne {column} non disponible dans les données.")',
        '        st.info(t("cannot_display"))',
    ),
])

# teammates_impact.py
patch("src/ui/pages/teammates_impact.py", [
    (
        '    st.caption("\u26a1 Premier sang | \U0001f3af Finisseur | \U0001f480 Boulet | \U0001f40c Plus lent | \U0001fab6 Première victime")',
        '    st.caption(t("tm_impact_legend"))',
    ),
])

# settings.py
patch("src/ui/pages/settings.py", [
    (
        '        st.markdown("**Données à backfill :**")',
        '        st.markdown(f"**{t(\'settings_backfill_data_label\')}**")',
    ),
])

# media_tab.py
patch("src/ui/pages/media_tab.py", [
    (
        '    st.subheader("Médias")',
        '    st.subheader(t("mv_media_title"))',
    ),
])

# media_library.py — second occurrence of Aucun média
with open("src/ui/pages/media_library.py", encoding="utf-8") as f:
    c = f.read()
if 'st.info("Aucun média à afficher avec ces filtres.")' in c:
    c = c.replace(
        'st.info("Aucun média à afficher avec ces filtres.")',
        'st.info(t("media_no_filter_result"))',
        1,
    )
    with open("src/ui/pages/media_library.py", "w", encoding="utf-8") as f:
        f.write(c)
    print("  media_library.py: second occurrence fixed")

# match_view.py — second occurrence citations_no_progress
with open("src/ui/pages/match_view.py", encoding="utf-8") as f:
    c = f.read()
if "st.info(\"Aucune citation n'a progressé dans ce match.\")" in c:
    c = c.replace(
        "st.info(\"Aucune citation n'a progressé dans ce match.\")",
        'st.info(t("citations_no_progress"))',
        1,
    )
    with open("src/ui/pages/match_view.py", "w", encoding="utf-8") as f:
        f.write(c)
    print("  match_view.py: citations_no_progress fixed")

# match_history.py
patch("src/ui/pages/match_history.py", [
    (
        '        st.warning("Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.")',
        '        st.warning(t("no_matches"))',
    ),
])

print("\nAll done!")
