# -*- coding: utf-8 -*-
"""Script temporaire : câblage i18n pour toutes les pages restantes."""

from pathlib import Path


def patch_file(path: str, replacements: list[tuple[str, str]], import_needed: bool = True) -> None:
    """Applique les remplacements à un fichier et insère l'import t si besoin."""
    p = Path(path)
    c = p.read_text(encoding="utf-8")

    if import_needed and "from src.ui.i18n import t" not in c:
        # Chercher l'import streamlit ou le premier from src
        if "import streamlit as st\n" in c:
            c = c.replace(
                "import streamlit as st\n",
                "import streamlit as st\n\nfrom src.ui.i18n import t\n",
                1,
            )
        elif "import streamlit as st" in c:
            c = c.replace(
                "import streamlit as st",
                "import streamlit as st\n\nfrom src.ui.i18n import t",
                1,
            )

    count = 0
    for old, new in replacements:
        if old in c:
            c = c.replace(old, new, 1)
            count += 1
        else:
            print(f"  NOT FOUND [{p.name}]: {old[:70]!r}")

    print(f"  {p.name}: {count}/{len(replacements)} replacements")
    p.write_text(c, encoding="utf-8")


# ═══════════════════════════════════════════════════════════════════════════════
# teammates_views.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/teammates_views.py",
    [
        # Spinners
        (
            'with st.spinner("Chargement des matchs avec ce coéquipier\u2026"):',
            'with st.spinner(t("tm_computing_teammate")):',
        ),
        (
            "with st.spinner(\"Calcul du ratio par carte (coéquipiers)\u2026\"):",
            'with st.spinner(t("tm_computing_map")):',
        ),
        (
            "with st.spinner(\"Chargement des stats des coéquipiers\u2026\"):",
            'with st.spinner(t("tm_computing_stats")):',
        ),
        (
            "with st.spinner(\"Agrégation des médailles (moi + coéquipier)\u2026\"):",
            'with st.spinner(t("tm_computing_medals")):',
        ),
        (
            "with st.spinner(\"Agrégation des médailles\u2026\"):",
            'with st.spinner(t("tm_computing_medals_all")):',
        ),
        # Warnings / infos
        (
            "st.warning(\"Aucun match trouvé avec ce coéquipier (selon le filtre).\")",
            'st.warning(t("tm_no_matches_teammate"))',
        ),
        (
            "st.info(\"Pas assez de matchs avec tes coéquipiers (selon le filtre actuel).\")",
            'st.info(t("tm_not_enough_matches"))',
        ),
        (
            "st.info(\"Données insuffisantes pour le ratio par carte.\")",
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher le ratio par carte : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
        (
            "st.info(\"Aucun match trouvé avec tes coéquipiers (selon le filtre actuel).\")",
            'st.info(t("tm_no_matches_filter"))',
        ),
        (
            "st.info(\"Aucun match partagé pour calculer les médailles.\")",
            'st.info(t("tm_no_shared_medals"))',
        ),
        (
            "st.info(\"Impossible de déterminer la liste des matchs pour l'agrégation des médailles.\")",
            'st.info(t("tm_no_medals_aggregate"))',
        ),
        (
            "st.info(\"Données insuffisantes pour les stats/min.\")",
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher les stats/min : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
        (
            "st.warning(\"Impossible d'aligner les stats des 3 joueurs sur ces matchs.\")",
            'st.warning(t("tm_trio_warning"))',
        ),
        # Subheaders
        (
            'st.subheader("Par carte \u2014 avec mes coéquipiers")',
            'st.subheader(t("tm_by_map"))',
        ),
        (
            'st.subheader("Historique \u2014 matchs avec mes coéquipiers")',
            'st.subheader(t("tm_history"))',
        ),
        (
            "st.subheader(\"Médailles (matchs partagés)\")",
            'st.subheader(t("tm_medals"))',
        ),
        (
            'st.subheader("Stats par minute")',
            'st.subheader(t("tm_per_minute"))',
        ),
        (
            'st.subheader("Médailles")',
            'st.subheader(t("tm_medals_all"))',
        ),
        # Dynamic subheader trio
        (
            "st.subheader(f\"Tous les trois \u2014 {f1_name} + {f2_name}\")",
            'st.subheader(t("tm_trio_header", f1=f1_name, f2=f2_name))',
        ),
        # Captions
        (
            "st.caption(f\"Dernière session trio détectée : {latest_label}.\")",
            'st.caption(t("tm_trio_session", label=latest_label))',
        ),
        (
            "st.caption(\"Impossible de déterminer une session trio (données insuffisantes).\")",
            'st.caption(t("tm_trio_session_unknown"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# match_view.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/match_view.py",
    [
        ('st.info("Modules LUSR non disponibles.")', 'st.info(t("mv_lusr_modules_missing"))'),
        ('st.info("Aucun rating disponible pour ce match.")', 'st.info(t("mv_lusr_no_data"))'),
        (
            'st.caption(f"Groupe : {playlist_group.capitalize()}")',
            'st.caption(t("mv_playlist_group_caption", group=playlist_group.capitalize()))',
        ),
        (
            'st.caption("Référentiel Citations indisponible.")',
            'st.caption(t("mv_citations_unavailable"))',
        ),
        (
            'st.caption("Citations indisponibles pour ce match.")',
            'st.caption(t("mv_citations_no_data"))',
        ),
        (
            "st.info(\"Aucune citation n'a progressé dans ce match.\")",
            'st.info(t("citations_no_progress"))',
        ),
        (
            'with st.spinner("Lecture des stats détaillées (attendu vs réel, médailles)\u2026"):',
            'with st.spinner(t("mv_loading")):',
        ),
        (
            'st.subheader("Citations")',
            'st.subheader(t("mv_citations"))',
        ),
        (
            'st.subheader("Médailles")',
            'st.subheader(t("mv_medals"))',
        ),
        (
            "st.info(\"Médailles indisponibles pour ce match (ou aucune médaille).\")",
            'st.info(t("mv_medals_no_data"))',
        ),
        (
            'st.info("MatchId manquant.")',
            'st.info(t("mv_match_id_missing"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# match_view_players.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/match_view_players.py",
    [
        (
            'st.subheader("Dynamique du match")',
            'st.subheader(t("mv_match_dynamics"))',
        ),
        (
            'with st.spinner("Analyse de la dynamique\u2026"):',
            'with st.spinner(t("mv_dynamics_computing")):',
        ),
        (
            'st.info("Données insuffisantes pour afficher la dynamique du match.")',
            'st.info(t("mv_dynamics_no_data"))',
        ),
        (
            'st.info("Roster introuvable \u2014 frise de dominance indisponible.")',
            'st.info(t("mv_dynamics_no_roster"))',
        ),
        (
            "st.info(\"Équipe introuvable pour ce joueur \u2014 frise de dominance indisponible.\")",
            'st.info(t("mv_dynamics_no_team"))',
        ),
        (
            'st.info("Aucun kill enregistré pour ce match.")',
            'st.info(t("mv_dynamics_no_kills"))',
        ),
        (
            'st.info("Données insuffisantes pour la frise de dominance.")',
            'st.info(t("mv_dynamics_no_dominance"))',
        ),
        (
            'st.subheader("Antagonistes du match")',
            'st.subheader(t("mv_antagonists_title"))',
        ),
        (
            'with st.spinner("Chargement des highlight events (film)\u2026"):',
            'with st.spinner(t("mv_highlight_loading")):',
        ),
        (
            "st.info(\"Impossible de déterminer Némésis/Souffre-douleur (timeline insuffisante).\")",
            'st.info(t("mv_nemesis_no_data"))',
        ),
        (
            'st.info("Données insuffisantes pour les interactions eliminateur-victime.")',
            'st.info(t("mv_interactions_no_data"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher les interactions eliminateur-victime : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
        (
            'st.subheader("Frags au fil du match")',
            'st.subheader(t("mv_kills_over_time"))',
        ),
        (
            'st.subheader("Tableau des scores")',
            'st.subheader(t("mv_scoreboard"))',
        ),
        (
            'st.info("Statistiques des joueurs indisponibles pour ce match.")',
            'st.info(t("mv_scoreboard_no_data"))',
        ),
        (
            'st.subheader("Joueurs")',
            'st.subheader(t("mv_players_title"))',
        ),
        (
            'st.subheader("Impact du match")',
            'st.subheader(t("mv_impact_title"))',
        ),
        (
            'st.caption("Données d\'impact indisponibles (highlight events manquants).")',
            'st.caption(t("mv_impact_no_events"))',
        ),
        (
            'with st.spinner("Analyse de la timeline\u2026"):',
            'with st.spinner(t("mv_impact_computing")):',
        ),
        (
            'st.info("Aucun événement enregistré pour ce match.")',
            'st.info(t("mv_impact_no_events_match"))',
        ),
        (
            "st.info(\"Pas assez de données pour afficher la timeline.\")",
            'st.info(t("mv_impact_too_few"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# media_library.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/media_library.py",
    [
        (
            'st.subheader("Bibliothèque médias")',
            'st.subheader(t("media_library_title"))',
        ),
        (
            'st.info("Les médias sont désactivés dans Paramètres \u2192 Médias.")',
            'st.info(t("media_disabled"))',
        ),
        (
            'st.info("Configure au moins un dossier dans Paramètres \u2192 Médias (captures et/ou vidéos).")',
            'st.info(t("media_no_folder"))',
        ),
        (
            'with st.spinner("Indexation en cours..."):',
            'with st.spinner(t("media_scanning")):',
        ),
        (
            'st.error(f"Erreur lors de l\'indexation: {e}")',
            'st.error(t("media_error_indexing", error=e))',
        ),
        (
            'with st.spinner("Génération des thumbnails..."):',
            'with st.spinner(t("media_generating_thumbnails")):',
        ),
        (
            'st.warning("Configure un dossier vidéos dans Paramètres \u2192 Médias.")',
            'st.warning(t("media_configure_video"))',
        ),
        (
            'st.subheader("Non associés")',
            'st.subheader(t("media_unassociated"))',
        ),
        (
            'st.info("Aucun média trouvé.")',
            'st.info(t("media_no_files"))',
        ),
        # There are 2 identical occurrences, handle both
        (
            'st.info("Aucun média à afficher avec ces filtres.")',
            'st.info(t("media_no_filter_result"))',
        ),
        (
            'st.caption("Match inconnu")',
            'st.caption(t("media_unknown_match"))',
        ),
        (
            'st.caption("(pas de miniature générée)")',
            'st.caption(t("media_no_thumbnail"))',
        ),
        (
            'st.caption("Match: non associé")',
            'st.caption(t("media_unassociated_match"))',
        ),
        (
            'st.error(f"Erreur: {e}")',
            'st.error(t("error_loading", error=e))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# citations.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/citations.py",
    [
        (
            'st.warning("Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.")',
            'st.warning(t("no_matches"))',
        ),
        (
            'st.subheader("Citations (Commendations Halo 5)")',
            'st.subheader(t("citations_halo5_title"))',
        ),
        (
            'with st.spinner("Agrégation des médailles\u2026"):',
            'with st.spinner(t("tm_computing_medals_all")):',
        ),
        (
            'st.subheader("Médailles (Halo Infinite)")',
            'st.subheader(t("citations_medals_title"))',
        ),
        (
            'st.caption("Médailles sur la sélection/filtres actuels.")',
            'st.caption(t("citations_medals_caption"))',
        ),
        (
            "st.info(\"Aucun match disponible avec les filtres actuels.\")",
            'st.info(t("no_data_filter"))',
        ),
        (
            "st.info(\"Aucune médaille trouvée (ou payload médailles absent).\")",
            'st.info(t("citations_no_medals"))',
        ),
        (
            'st.subheader("Distribution des médailles")',
            'st.subheader(t("citations_medals_distribution"))',
        ),
        (
            "st.info(\"Données insuffisantes pour la distribution des médailles.\")",
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher la distribution des médailles : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
        (
            'st.subheader("Grille de médailles")',
            'st.subheader(t("citations_medals_grid"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# settings.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/settings.py",
    [
        (
            'st.subheader("Paramètres")',
            'st.subheader(t("settings_title"))',
        ),
        (
            'st.error(f"Erreur: {e}")',
            'st.error(t("error_loading", error=e))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# last_match.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/last_match.py",
    [
        (
            'st.caption("Dernière partie selon la sélection/filtres actuels.")',
            'st.caption(t("last_match_caption"))',
        ),
        (
            "st.info(\"Aucun match disponible avec les filtres actuels.\")",
            'st.info(t("no_data_filter"))',
        ),
        (
            'st.caption("Afficher un match précis via un MatchId, une date/heure, ou une sélection.")',
            'st.caption(t("last_match_select_caption"))',
        ),
        (
            'st.warning("Aucune date exploitable dans la DB.")',
            'st.warning(t("last_match_no_date"))',
        ),
        (
            'st.warning("MatchId introuvable dans la DB actuelle.")',
            'st.warning(t("last_match_not_found"))',
        ),
        (
            'st.info("Renseigne un MatchId ou utilise la sélection/recherche ci-dessus.")',
            'st.info(t("last_match_enter_id"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# teammates_impact.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/teammates_impact.py",
    [
        (
            'st.subheader("⚡ Impact")',
            'st.subheader(t("tm_impact_header"))',
        ),
        (
            "st.info(\"Sélectionnez au moins 2 coéquipiers pour voir l'analyse d'impact.\")",
            'st.info(t("tm_impact_select_two"))',
        ),
        (
            "st.info(\"Aucun match à analyser.\")",
            'st.info(t("tm_impact_no_matches"))',
        ),
        (
            "st.info(\"Aucun événement trouvé pour les matchs sélectionnés.\")",
            'st.info(t("tm_impact_no_events_matches"))',
        ),
        (
            "st.info(\"Aucun événement d'impact trouvé pour les joueurs sélectionnés.\")",
            'st.info(t("tm_impact_no_events_players"))',
        ),
        (
            'st.subheader("Heatmap d\'Impact")',
            'st.subheader(t("tm_impact_heatmap"))',
        ),
        (
            'st.subheader("🏆 Classement")',
            'st.subheader(t("tm_impact_ranking"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# teammates.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/teammates.py",
    [
        (
            "st.info(\"Aucun match disponible avec les filtres actuels.\")",
            'st.info(t("no_data_filter"))',
        ),
        (
            'st.warning("Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.")',
            'st.warning(t("no_matches"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# media_tab.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/media_tab.py",
    [
        (
            'st.info("Aucun média trouvé.")',
            'st.info(t("media_no_files"))',
        ),
        (
            'st.info("Aucun média à afficher avec ces filtres.")',
            'st.info(t("media_no_filter_result"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# match_view_helpers.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/match_view_helpers.py",
    [
        (
            "st.info(\"Aucune citation n'a progressé dans ce match.\")",
            'st.info(t("citations_no_progress"))',
        ),
        (
            'st.info("Données insuffisantes.")',
            'st.info(t("insufficient_data_chart"))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# match_view_participation.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/match_view_participation.py",
    [
        (
            'st.info("Données insuffisantes pour la participation.")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher la participation : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
    ],
)

# ═══════════════════════════════════════════════════════════════════════════════
# teammates_synergy.py
# ═══════════════════════════════════════════════════════════════════════════════
patch_file(
    "src/ui/pages/teammates_synergy.py",
    [
        (
            'st.info("Données insuffisantes pour la synergie.")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher la synergie : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
    ],
)

print("\nDone!")
