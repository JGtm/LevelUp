"""Module application - Orchestration du dashboard.

Ce module contient la logique d'orchestration extraite de streamlit_app.py :
- state.py : Gestion centralisée du session_state
- routing.py : Navigation entre pages
- sidebar.py : Logique et rendu de la sidebar
- filters.py : Logique des filtres globaux
- helpers.py : Fonctions utilitaires génériques
- profile.py : Gestion du profil joueur
- kpis.py : Calcul et affichage des KPIs
- data_loader.py : Chargement et initialisation des données
- navigation.py : Rendu des pages

Importer directement depuis le sous-module voulu :
    from src.app.state import AppState          # \u2705
    from src.app import AppState                # ❌ charge tout le package
"""

from __future__ import annotations
