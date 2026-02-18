# Synthèse visuelle : Problèmes des filtres sidebar

> **Diagrammes et schémas** pour comprendre les bugs en un coup d'œil

---

## 🔴 Problème #1 : Listing dynamique = Options instables

### Comportement actuel (BUGGY)

```
┌─────────────────────────────────────────────────────────────┐
│ Étape 1 : Mode "Sessions" - Dernière session (3 matchs)    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Matchs du joueur :                                         │
│  ┌──────────┬──────────────────┬──────────────┐           │
│  │ Match ID │ Playlist         │ Mode         │           │
│  ├──────────┼──────────────────┼──────────────┤           │
│  │ m1       │ Partie rapide    │ Assassin     │           │
│  │ m2       │ Partie rapide    │ Fiesta       │           │
│  │ m3       │ Arène classée    │ Assassin     │           │
│  └──────────┴──────────────────┴──────────────┘           │
│                                                             │
│  Playlists disponibles (calculées depuis m1-m3) :          │
│  ☑ Partie rapide                                           │
│  ☑ Arène classée                                           │
│                                                             │
│  Utilisateur coche les deux ✅                              │
└─────────────────────────────────────────────────────────────┘

                         ⬇️ Utilisateur change de période

┌─────────────────────────────────────────────────────────────┐
│ Étape 2 : Mode "Période" - Tous les matchs (150 matchs)    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Matchs du joueur :                                         │
│  ┌──────────┬──────────────────┬──────────────┐           │
│  │ Match ID │ Playlist         │ Mode         │           │
│  ├──────────┼──────────────────┼──────────────┤           │
│  │ m1-m3    │ (voir ci-dessus) │ ...          │           │
│  │ m4-m50   │ Assassin classé  │ ...          │           │
│  │ m51-m100 │ BTB              │ ...          │           │
│  │ m101-150 │ Firefight        │ ...          │           │
│  └──────────┴──────────────────┴──────────────┘           │
│                                                             │
│  Playlists disponibles (recalculées depuis m1-m150) :      │
│  ☑ Partie rapide        ← Encore sélectionnée (OK)         │
│  ☑ Arène classée        ← Encore sélectionnée (OK)         │
│  ☐ Assassin classé      ← NOUVELLE, décochée par défaut ❌ │
│  ☐ BTB                  ← NOUVELLE, décochée par défaut ❌ │
│  ☐ Firefight            ← NOUVELLE, décochée par défaut ❌ │
│                                                             │
│  Problème : Options nouvelles = décochées                   │
└─────────────────────────────────────────────────────────────┘

                         ⬇️ Utilisateur retourne en mode Session

┌─────────────────────────────────────────────────────────────┐
│ Étape 3 : Mode "Sessions" - Retour à dernière session      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Matchs de la session (m1-m3) :                             │
│  Playlists disponibles (recalculées) :                      │
│  ☑ Partie rapide        ← Toujours sélectionnée (OK)       │
│  ☑ Arène classée        ← Toujours sélectionnée (OK)       │
│                                                             │
│  Mais... le système nettoie les options obsolètes :         │
│  - "Assassin classé" supprimé de la sélection              │
│  - "BTB" supprimé de la sélection                          │
│  - "Firefight" supprimé de la sélection                    │
│                                                             │
│  ⚠️ Si l'utilisateur avait coché "BTB" à l'étape 2,        │
│     ce choix est PERDU silencieusement                     │
└─────────────────────────────────────────────────────────────┘
```

### Solution recommandée (OPTIONS STATIQUES)

```
┌─────────────────────────────────────────────────────────────┐
│ Toutes les étapes : Options STABLES                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Playlists disponibles (FIXES, depuis metadata.duckdb) :   │
│  ☑ Partie rapide        ← Sélectionnée                     │
│  ☑ Arène classée        ← Sélectionnée                     │
│  ☐ Assassin classé      ← Disponible, non sélectionnée     │
│  ☐ BTB                  ← Disponible, non sélectionnée     │
│  ☐ Firefight            ← Disponible, non sélectionnée     │
│                                                             │
│  Si l'utilisateur change de période/session :               │
│  - Les options restent IDENTIQUES                          │
│  - Les sélections sont CONSERVÉES                           │
│  - Pas de désorientation                                   │
│                                                             │
│  Si une playlist n'a pas de matchs dans la période :        │
│  ☐ BTB (i)              ← GRISÉ avec tooltip               │
│     "Aucun match dans cette période"                       │
│                                                             │
│  ✅ Utilisateur peut garder sa sélection même si           │
│     temporairement indisponible                            │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔴 Problème #2 : Clés globales = Corruption entre joueurs

### Flux actuel (BUGGY)

```
┌──────────────────────────────────────────────────────────────┐
│ État initial : Joueur A                                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  session_state :                                             │
│  {                                                           │
│    "filter_playlists": {"Partie rapide", "Arène classée"}   │
│    "filter_playlists_cb_Partie_rapide": True   ← Widget     │
│    "filter_playlists_cb_Arène_classée": True   ← Widget     │
│    "filter_playlists_cb_Assassin_classé": False ← Widget    │
│  }                                                           │
│                                                              │
│  Fichier : player_A.json                                     │
│  {                                                           │
│    "playlists_selected": ["Partie rapide", "Arène classée"] │
│  }                                                           │
└──────────────────────────────────────────────────────────────┘

                    ⬇️ Changement vers Joueur B

┌──────────────────────────────────────────────────────────────┐
│ Nettoyage INCOMPLET (streamlit_app.py)                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Supprimé :                                                  │
│  - "filter_playlists" ✅                                     │
│                                                              │
│  NON supprimé (BUG ❌) :                                     │
│  - "filter_playlists_cb_Partie_rapide"  ← Reste en mémoire  │
│  - "filter_playlists_cb_Arène_classée"  ← Reste en mémoire  │
│  - "filter_playlists_cb_Assassin_classé" ← Reste en mémoire │
│                                                              │
│  Application des préférences de B :                          │
│  session_state["filter_playlists"] = {"Assassin classé"}    │
└──────────────────────────────────────────────────────────────┘

                    ⬇️ Rendu pour Joueur B

┌──────────────────────────────────────────────────────────────┐
│ Affichage INCOHÉRENT (filters_render.py)                     │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Streamlit réutilise les clés widgets existantes :          │
│                                                              │
│  st.checkbox(                                                │
│    "Partie rapide",                                          │
│    value="Partie rapide" in current_selection,  ← False     │
│    key="filter_playlists_cb_Partie_rapide"  ← Key exists!   │
│  )                                                           │
│                                                              │
│  ⚠️ Streamlit affiche la valeur STOCKÉE (True pour A)       │
│     au lieu de la valeur calculée (False pour B)            │
│                                                              │
│  Utilisateur voit :                                          │
│  ☑ Partie rapide        ← COCHÉ mais B ne l'a pas sélectionné! │
│  ☐ Arène classée                                            │
│  ☑ Assassin classé      ← Seul vraiment sélectionné par B  │
│                                                              │
│  Utilisateur clique pour "corriger" → Modifie la sélection  │
└──────────────────────────────────────────────────────────────┘

                    ⬇️ Sauvegarde automatique

┌──────────────────────────────────────────────────────────────┐
│ Corruption du fichier B (filters_render.py, ligne 206)       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  save_filter_preferences(xuid="B", db_path=...)             │
│                                                              │
│  Lit depuis session_state (état CORROMPU) :                  │
│  {                                                           │
│    "filter_playlists": {"Partie rapide", "Assassin classé"} │
│  }                                                           │
│                                                              │
│  Écrit dans player_B.json :                                  │
│  {                                                           │
│    "playlists_selected": ["Partie rapide", "Assassin classé"] │
│  }                                                           │
│                                                              │
│  ❌ Les préférences de B ont été écrasées par un mélange A+B │
└──────────────────────────────────────────────────────────────┘

                    ⬇️ Retour à Joueur A

┌──────────────────────────────────────────────────────────────┐
│ Dégradation progressive                                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Les widgets sont maintenant dans un état mixte A+B+...      │
│  Chaque aller-retour aggrave la corruption                   │
│                                                              │
│  Résultat : "encore plus de filtres désélectionnés"          │
└──────────────────────────────────────────────────────────────┘
```

### Solution recommandée (CLÉS SCOPÉES)

```
┌──────────────────────────────────────────────────────────────┐
│ Joueur A : Clés scopées                                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  session_state :                                             │
│  {                                                           │
│    "filter_playlists_player_A": {"Partie rapide"}           │
│    "filter_playlists_player_A_cb_Partie_rapide": True       │
│    "filter_playlists_player_A_cb_Arène_classée": False      │
│  }                                                           │
└──────────────────────────────────────────────────────────────┘

                    ⬇️ Changement vers Joueur B

┌──────────────────────────────────────────────────────────────┐
│ Joueur B : Clés scopées (ISOLATION)                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  session_state :                                             │
│  {                                                           │
│    # Clés de A restent en mémoire (pas de conflit)          │
│    "filter_playlists_player_A": {"Partie rapide"}           │
│    "filter_playlists_player_A_cb_Partie_rapide": True       │
│                                                              │
│    # Nouvelles clés pour B                                   │
│    "filter_playlists_player_B": {"Assassin classé"}         │
│    "filter_playlists_player_B_cb_Partie_rapide": False      │
│    "filter_playlists_player_B_cb_Assassin_classé": True     │
│  }                                                           │
│                                                              │
│  ✅ Aucune collision, chaque joueur a son espace             │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔴 Problème #3 : Cascade fragile

### Architecture actuelle (COUPLAGE FORT)

```
┌─────────────────────────────────────────────────────────────┐
│                    Filtres en cascade                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Données brutes (150 matchs)                                │
│       ↓ filtre période/session                             │
│  Données filtrées (3 matchs)                                │
│       ↓ calcul options                                      │
│  Playlists disponibles : [P1, P2]                           │
│       ↓ utilisateur sélectionne P1                          │
│  Données filtrées par playlist (2 matchs)                   │
│       ↓ calcul options                                      │
│  Modes disponibles : [M1, M2]                               │
│       ↓ utilisateur sélectionne M1                          │
│  Données filtrées par mode (1 match)                        │
│       ↓ calcul options                                      │
│  Cartes disponibles : [C1]                                  │
│       ↓ utilisateur sélectionne C1                          │
│  Données finales (1 match)                                  │
│                                                             │
│  Problème : Chaque niveau dépend du précédent               │
│  → Changement en haut = recalcul de toute la cascade        │
│  → États intermédiaires incohérents                         │
└─────────────────────────────────────────────────────────────┘
```

### Architecture recommandée (DÉCOUPLAGE)

```
┌─────────────────────────────────────────────────────────────┐
│               Filtres indépendants                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Données brutes (150 matchs)                                │
│       │                                                     │
│       ├─→ Filtre période/session                           │
│       ├─→ Filtre playlists                                 │
│       ├─→ Filtre modes                                     │
│       └─→ Filtre cartes                                    │
│       ↓ application simultanée                              │
│  Données finales (intersection)                             │
│                                                             │
│  Options pour chaque filtre :                               │
│  - Toutes les playlists du jeu (statiques)                 │
│  - Tous les modes du jeu (statiques)                       │
│  - Toutes les cartes du jeu (statiques)                    │
│                                                             │
│  Feedback visuel :                                          │
│  - Options grisées si 0 matchs avec cette sélection         │
│  - Compteur "X matchs disponibles" en temps réel            │
│                                                             │
│  ✅ Aucune dépendance entre filtres                         │
│  ✅ Changement local = pas de recalcul global               │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔴 Problème #4 : Persistance non validée

### Flux actuel (DANGEREUX)

```
┌─────────────────────────────────────────────────────────────┐
│ Chaque rendu → Sauvegarde automatique                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Utilisateur modifie un filtre                           │
│       ↓ st.rerun()                                          │
│  2. Rendu des filtres                                       │
│       ↓ fin de render_filters_sidebar()                     │
│  3. save_filter_preferences()                               │
│     - Lit tout depuis session_state (AUCUNE VALIDATION)     │
│     - Écrit dans player_{gamertag}.json                     │
│       ↓                                                     │
│  4. Si session_state corrompu → JSON corrompu              │
│                                                             │
│  Cas problématiques :                                        │
│  - Sélection vide (bug d'affichage)                        │
│  - Options invalides (playlist supprimée)                   │
│  - États intermédiaires (mid-rerun)                        │
│  - Mélange entre joueurs (clés globales)                   │
│                                                             │
│  ❌ Pas de rollback, préférences perdues définitivement     │
└─────────────────────────────────────────────────────────────┘
```

### Flux recommandé (SÉCURISÉ)

```
┌─────────────────────────────────────────────────────────────┐
│ Sauvegarde explicite avec validation                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Utilisateur modifie des filtres                         │
│     - Changements en mémoire uniquement                     │
│     - Aucune écriture disque                                │
│                                                             │
│  2. Utilisateur clique "💾 Sauvegarder les préférences"     │
│       ↓ validation                                          │
│  3. Validation des préférences :                            │
│     - Playlists sélectionnées existent ?                    │
│     - Dates cohérentes (start ≤ end) ?                      │
│     - Session label existe ?                                │
│       ↓ si OK                                               │
│  4. Sauvegarde dans player_{gamertag}.json                  │
│     - Backup de l'ancien fichier                            │
│     - Écriture atomique (tmp + rename)                      │
│       ↓                                                     │
│  5. Feedback à l'utilisateur :                              │
│     st.success("✅ Préférences sauvegardées")               │
│                                                             │
│  Si validation échoue :                                      │
│     st.error("❌ Impossible de sauvegarder : [raison]")     │
│     - Pas d'écriture disque                                 │
│     - Préférences précédentes conservées                    │
│                                                             │
│  Option : Auto-save avec debounce (2-3s après dernier changement) │
└─────────────────────────────────────────────────────────────┘
```

---

## ✅ Architecture cible (Vision globale)

```
┌─────────────────────────────────────────────────────────────────┐
│                    COUCHE PRÉSENTATION                          │
│  (filters_sidebar.py)                                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐               │
│  │  Période   │  │ Playlists  │  │   Modes    │               │
│  │  / Session │  │  (static)  │  │  (static)  │               │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘               │
│        │               │               │                        │
│        └───────────────┴───────────────┘                        │
│                        │                                        │
│                 Événements UI                                   │
│                        │                                        │
└────────────────────────┼────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    COUCHE MÉTIER                                │
│  (filter_manager.py)                                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────┐         │
│  │  FilterState (immutable dataclass)                │         │
│  │  - mode: FilterMode                               │         │
│  │  - start_date, end_date                           │         │
│  │  - playlists_selected: frozenset[UUID]           │         │
│  │  - modes_selected: frozenset[UUID]               │         │
│  │  - maps_selected: frozenset[UUID]                │         │
│  └───────────────────────────────────────────────────┘         │
│                         │                                       │
│  ┌───────────────────────────────────────────────────┐         │
│  │  FilterDispatcher                                 │         │
│  │  - handle_period_change()                         │         │
│  │  - handle_playlist_change()                       │         │
│  │  - handle_save_request()                          │         │
│  │  → Retourne nouveau FilterState                   │         │
│  └───────────────────────────────────────────────────┘         │
│                                                                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    COUCHE DONNÉES                               │
│  (filter_repository.py + filter_options_provider.py)           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────┐  ┌──────────────────────────┐   │
│  │ FilterRepository         │  │ FilterOptionsProvider    │   │
│  │ - load_preferences()     │  │ - get_all_playlists()    │   │
│  │ - save_preferences()     │  │ - get_all_modes()        │   │
│  │ - validate_preferences() │  │ - get_all_maps()         │   │
│  │                          │  │ (depuis metadata.duckdb)  │   │
│  │ (JSON par joueur)        │  │                          │   │
│  └──────────────────────────┘  └──────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📊 Impact des solutions

### Comparaison quantitative

| Métrique | Actuel | Après Phase 1 (Nettoyage) | Après Phase 2 (Options statiques) | Après Phase 4 (Refactoring) |
|----------|--------|---------------------------|-----------------------------------|------------------------------|
| **Bugs de corruption** | 🔴 Fréquents | 🟡 Rares | 🟢 Aucun | 🟢 Aucun |
| **Désorientation UX** | 🔴 Élevée | 🔴 Élevée | 🟢 Nulle | 🟢 Nulle |
| **Lignes de code** | ~1800 | ~1800 | ~2000 | ~1500 |
| **Complexité cyclomatique** | 🔴 15-20 | 🟡 12-15 | 🟢 8-10 | 🟢 5-8 |
| **Tests coverage** | 🟡 60% | 🟡 65% | 🟢 80% | 🟢 90% |
| **Temps de rendu** | 🟡 200-300ms | 🟡 200-300ms | 🟢 100-150ms | 🟢 80-120ms |
| **Effort implémentation** | - | 1-2j | 5-7j | 2-3 sem |

---

**🎯 Recommandation finale** : Phase 1 + Phase 2 (options statiques)

- Résout les 2 problèmes critiques (corruption + désorientation)
- Effort raisonnable (1 semaine)
- ROI maximal pour l'utilisateur
