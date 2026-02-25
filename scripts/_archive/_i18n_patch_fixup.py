# -*- coding: utf-8 -*-
"""Script correctif pour les fichiers avec NOT FOUND."""
from pathlib import Path


def patch_file(path: str, replacements: list[tuple[str, str]]) -> None:
    p = Path(path)
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
    print(f"  {p.name}: {count}/{len(replacements)} replacements")
    p.write_text(c, encoding="utf-8")


# media_tab.py — strings différents de ce qu'on attendait
patch_file(
    "src/ui/pages/media_tab.py",
    [
        (
            "st.info(\"Les médias sont désactivés dans Paramètres \u2192 Médias.\")",
            'st.info(t("media_disabled"))',
        ),
        (
            "st.info(\"Sélectionne un profil joueur (DB DuckDB) pour afficher les médias.\")",
            'st.info(t("no_data_filter"))',
        ),
    ],
)

# match_view_participation.py — strings différents
patch_file(
    "src/ui/pages/match_view_participation.py",
    [
        (
            'st.info("Impossible de générer le radar de participation.")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher le radar de participation : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
        (
            'st.info("Impossible de générer le radar de comparaison.")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher le radar de comparaison : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
    ],
)

# teammates_synergy.py — strings différents
patch_file(
    "src/ui/pages/teammates_synergy.py",
    [
        (
            'st.info("Données de participation indisponibles (PersonalScores manquants).")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            'st.info("Impossible de générer le radar de participation.")',
            'st.info(t("insufficient_data_chart"))',
        ),
        (
            "st.warning(f\"Impossible d'afficher le radar de participation : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
    ],
)

# teammates_impact.py — strings différents
patch_file(
    "src/ui/pages/teammates_impact.py",
    [
        (
            'st.warning("Aucun match à analyser.")',
            'st.warning(t("tm_impact_no_matches"))',
        ),
        (
            "st.warning(f\"Impossible de charger les données d'impact : {e}\")",
            'st.warning(t("error_chart", error=e))',
        ),
    ],
)

print("Done!")
