# Plan d'implémentation v7 — Cockpit analytique

> Référence plan : `.ai/PLAN_V7_ALTERNATIVE_COCKPIT.md`
> Déploiement : sous-domaine dédié — pas de contrainte de migration de service
> Branche cible : `v7/cockpit`
> Développement : local jusqu'à Phase F — aucune dépendance infra pour A–E

---

## Vue d'ensemble des phases

| Phase | Contenu | Niveau de détail |
|---|---|---|
| A | Shell, routing, design system | Tâche par tâche |
| B | Contextes analytiques, L2, KPI bar | Détaillé + fichiers sources + risques |
| C | Hubs d'analyse | Détaillé + mapping composants + risques |
| D | Vues nouvelles D1–D5 | Spécifié par vue |
| E | Validation parité + retrait sidebar | Critères et checklist |
| F | Infrastructure preview + rebranching main | Infra, données partagées, mise en prod |

---

## Phase A — Fondations shell, routing & design system

> Objectif : une app qui tourne avec la nouvelle navigation, le nouveau design, et zéro fonctionnalité cassée. La sidebar est masquée, les pages existantes restent accessibles.

### A0 — Initialisation ✅

- [x] Créer la branche de travail depuis `main` — `v7/cockpit`
- [x] Créer `streamlit_app_v7.py` à la racine
- [x] Vérifier que l'app démarre sans erreur sur ce nouveau point d'entrée
- [x] **Créer le package `src/ui/layout/`** — `header_l1.py`, `header_l2.py`, `kpi_bar.py`, `filter_chips.py` + `__init__.py`
- [x] **Palette : fork propre confirmé** — `THEME_COLORS_V7` dans `src/config.py`, CSS dans `src/ui/theme/v7_theme.css`

### A1 — Design system CSS ✅

- [x] Créer `src/ui/theme/v7_theme.css` avec toutes les variables CSS de la palette
- [x] Appliquer via `load_v7_theme_css()` dans le shell
- [x] Variables définies : `--v7-bg-main` · `--v7-bg-surface` · `--v7-bg-surface-alt` · `--v7-border-subtle` · `--v7-border-strong` · `--v7-text-primary` · `--v7-text-secondary` · `--v7-text-muted` · `--v7-accent` · `--v7-accent-soft` · `--v7-success` · `--v7-danger` · `--v7-neutral` · `--v7-radius` · `--v7-mono` · `--v7-sans`
- [x] Règles palette v7 : aucun fond translucide, aucun gradient décoratif, accent unique visible
- [x] `PLOTLY_V7_CONFIG` + `get_v7_plotly_layout()` dans `src/ui/streamlit_modern.py` — branché sur `THEME_COLORS_V7`
- [x] `THEME_COLORS_V7` dans `src/config.py` — source de vérité partagée CSS+Python

### A1bis — Polish visuel des graphes 🔲 en cours

- [x] `get_v7_plotly_layout()` définie dans `streamlit_modern.py` — charte Plotly légère (fond, police, hover labels)
- [ ] **Corriger les deux graphes à fond clair hérités** :
  - `src/visualization/friends_impact_heatmap.py` L244 : `plot_bgcolor="rgba(245, 245, 245, 0.5)"` → thème sombre
  - `src/visualization/friends_impact_scatter.py` L196 : `plot_bgcolor="rgba(248,250,252,0.92)"` → thème sombre
- [ ] Vérifier rendu sur graphes représentatifs après correction

### A2 — Shell global L1 ✅

- [x] `src/ui/layout/header_l1.py` — navigation 6 sections via `st.segmented_control`
- [x] Sync : `render_sync_indicator` compact dans le shell
- [x] Sélecteur Gamertag : single player ou multi-player via `render_player_selector_unified`
- [x] Logo / clic → Accueil
- [x] Section active dans `SK.V7_CURRENT_SECTION`

### A3 — Routing 6 sections ✅

- [x] `page_router.py` : `V7_SECTION_KEYS`, `_V7_SECTION_I18N_KEYS`, `normalize_v7_section`, `get_v7_section_label`, `map_legacy_page_to_v7_location`
- [x] Dispatch v7 dans `v7_sections.py` — 6 sections (home/stats/squad/explorer/media/profile)
- [x] Compatibilité deep links : `map_legacy_page_to_v7_location` + `_hydrate_v7_navigation`
- [x] Fallback : section inconnue → home

### A4 — Masquage sidebar ✅

- [x] Sidebar masquée via CSS dans `v7_theme.css` : `[data-testid="stSidebar"] { display: none !important; }`
- [ ] **Isoler les règles CSS sidebar orphelines dans `static/styles.css`** — baliser avec `/* SIDEBAR — orphan en v7, retrait en Phase E3 */` (différé Phase E3)

### A5 — Smoke tests Phase A

- [ ] Navigation entre les 6 sections fonctionne
- [ ] Rechargement page → section active restaurée
- [ ] Aucune régression sur les pages existantes
- [ ] Design appliqué uniformément (backgrounds, typographie, Plotly)

**Risques Phase A :**
- Le sélecteur Gamertag est probablement dans la sidebar actuellement — identifier sa logique et la déplacer dans L1 sans casser l'état joueur actif

---

## Phase B — Contextes analytiques

> Objectif : Stats et Escouade ont leur bandeau L2 fonctionnel avec filtres, navigation de scope et KPI bar. Le reste des sections est inchangé.

**Fichiers sources à lire avant de commencer Phase B :**
- `src/app/filters_render.py` — logique de rendu des filtres (actuellement sidebar-first)
- `src/ui/filter_state.py` — persistance des états de filtre
- `src/app/session_keys.py` — clés session_state existantes
- `src/utils/paths.py` — pour comprendre les chemins de données utilisés par les contextes

### B1 — Bandeau de contexte L2 ✅

- [x] `src/ui/layout/header_l2.py` — s'affiche uniquement dans Stats et Escouade
- [x] Bouton Réinitialiser filtres + `render_filter_chips()`
- [ ] Navigation session (◀ Précédente / ⚡ Dernière) — différé Phase C
- [ ] Raccourci "Modifier mon escouade" dans L2 Escouade — différé Phase C

### B2 — Filtres découplés de la sidebar

**Prérequis B2 :**
- `filters_render.py` fait 494L. Découpage `_filters_logic.py` + scope dimension à faire avant d'aller plus loin.

- [ ] Lire `filters_render.py` en entier avant de toucher quoi que ce soit
- [ ] Identifier tous les widgets qui écrivent dans `session_state` depuis la sidebar
- [ ] Déplacer le rendu des filtres dans un panneau accessible depuis L2 (drawer ou expander compact)
- [ ] Conserver les mêmes clés `session_state` pour ne pas casser les pages existantes
- [ ] `filter_state.py` : scope (solo/squad/all) comme nouvelle dimension

### B3 — KPI bar ✅

- [x] `src/ui/layout/kpi_bar.py` — KD · Précision · V% · volume matchs · durée (HTML compact)
- [x] Valeurs calculées depuis `compute_kpi_stats(dff)`
- [x] Affiché uniquement dans Stats et Escouade (`if active_section in {"stats", "squad"}`)

### B4 — Filter chips ✅

- [x] `src/ui/layout/filter_chips.py` — chips filtres actifs avec suppression unitaire
- [x] Synchronisation avec `filter_state.py`

---

## Prérequis Phase C — Découpe des god files

> À traiter avant d'intégrer quoi que ce soit dans les hubs. Ces 5 fichiers dépassent ou frôlent 500L et seront réintégrés dans plusieurs hubs — les intégrer tels quels traînerait la dette dans la nouvelle archi.

| Fichier | Lignes | Action |
|---|---|---|
| `src/ui/pages/match_view_scoreboard_detail.py` | 624L | Extraire la logique de scoring en `match_view_scoreboard_logic.py` |
| `src/ui/pages/teammates_charts.py` | 535L | Séparer fragment wrappers / builders de figures / helpers en sous-modules |
| `src/ui/pages/session_compare.py` | 529L | Logique state inline → extraire dans `_session_compare_state.py` |
| `src/ui/pages/match_view_helpers.py` | 505L | Séparer helpers rendu / helpers données |
| `src/ui/pages/session_compare_charts.py` | 501L | Extraire les builders figure dans `_session_compare_figures.py` |

- [ ] Chaque découpe doit laisser le fichier source sous 400L après split
- [ ] Vérifier que les tests existants passent après chaque split (aucun test cassé)
- [ ] Ne pas modifier la logique fonctionnelle — seule la répartition de code change

---

## Phase C — Hubs d'analyse

> Objectif : chaque section est un vrai hub avec ses onglets et blocs. Les composants existants sont réintégrés, recomposés ou remplacés selon la règle du plan (réutiliser / recomposer / créer).

**Principe de travail Phase C :** lire chaque module source en entier avant de l'intégrer dans un hub. Ne pas supposer ce qu'il fait.

### C1 — Accueil Mission Control

**Fichiers sources à lire :** `src/ui/pages/last_match.py` · loaders sessions · `src/data/repositories/_match_relations.py` · `src/ui/pages/media_library.py` (pour A6)

- [ ] Créer `src/ui/pages/home_mission_control.py`
- [ ] **A1 Hero dernier match** (synchrone) — réutilise `render_match_view()` + navigation ◀▶ déjà existante
- [ ] **A5 Raccourcis** (synchrone) — 4 cartes : Stats · Escouade · Explorer · Médias
- [ ] **A2 Session solo récente** (fragment) — volume + KD/Précision/V-D + 1 insight + CTA vers Stats
- [ ] **A3 Session escouade récente** (fragment) — idem en scope squad + CTA vers Escouade
- [ ] **A4 Faits saillants** (fragment, après A2/A3) — masqué si aucune session comparable ; placeholder height fixe si masqué pour éviter layout shift
- [ ] **A6 Derniers médias liés** (fragment) — 3 à 6 items, miniature + date + lien match ; transmet un filtre via `session_state` à l'ouverture du hub Médias

**Risque C1 :** `render_match_view()` peut être lourd à charger. S'assurer qu'il n'y a pas de chargement DB synchrone bloquant dans A1 qui retarde l'affichage des raccourcis A5.

**Risque C1 :** A4 (faits saillants) nécessite un historique de sessions comparables — si l'utilisateur a peu de sessions, le bloc sera souvent masqué. Prévoir un message sobre (pas d'erreur, pas de spinner vide).

### C2 — Stats hub

**Fichiers sources à lire :** `src/ui/pages/timeseries.py` (5 onglets internes : Résumé, Cartes & Modes, Distributions, Progression, Avancé) · `src/ui/pages/match_history.py` · `src/ui/pages/session_compare.py`

- [ ] Créer `src/ui/pages/stats_hub.py`
- [ ] 5 onglets : Vue d'ensemble · Performance · Historique · Sessions · Matchs

**Mapping composants → onglets :**

| Onglet | Composants sources | Action |
|---|---|---|
| Vue d'ensemble | Blocs D1 + D5 (Phase D) | Nouveau — assemblage |
| Performance | `timeseries.py` (onglets Résumé, Progression, Avancé) · `render_performance_info` | Réintégrer |
| Historique | `timeseries.py` (onglets Cartes & Modes, Distributions) · `match_history.py` | Réintégrer |
| Sessions | `session_compare.py` · bloc D2 (Phase D) | Réintégrer + nouveau |
| Matchs | Table matchs filtrée · `render_match_view()` drill-down inline | Recomposer |

- [ ] Onglet Matchs : prototyper le drill-down inline — si l'expander dans la table provoque un layout shift, basculer sur split-view (table en haut, détail en dessous avec `st.container`)

**Risque C2 :** `timeseries.py` a 5 onglets internes — les insérer dans les onglets du hub Stats peut créer des onglets imbriqués (Streamlit gère mal les `st.tabs` dans `st.tabs`). Lire la structure de `timeseries.py` avant de décider comment l'intégrer — peut nécessiter de le réfactorer pour exposer ses sections indépendamment.

### C3 — Escouade hub

**Fichiers sources à lire :** `src/ui/pages/teammates.py` · `src/ui/pages/teammates_views.py` · `src/ui/pages/teammates_synergy.py` · `src/ui/pages/teammates_impact.py` · `src/ui/pages/_teammates_trio.py` · `src/ui/pages/_teammates_trio_helpers.py`

- [ ] Créer `src/ui/pages/squad_hub.py`
- [ ] 4 onglets : Groupe · Synergies · Mes stats · Sessions d'escouade

**Mapping composants → onglets :**

| Onglet | Composants sources | Action |
|---|---|---|
| Groupe | `teammates.py` (coéquipiers fréquents) | Réintégrer |
| Synergies | `render_synergy_radar` · `render_trio_synergy_radar` · `render_impact_taquinerie` · `render_squad_cadence_section` | Réorganiser — pas de création |
| Mes stats | Stats hub en scope squad (mêmes composants, filtre squad) + bloc D3 | Recomposer |
| Sessions d'escouade | `render_squad_timeline` · `plot_squad_performance_timeline` · `_detect_trio_session` · `render_trio_view` + Q11 cartes session (nouveau léger) | Réorganiser + 1 nouveau |

- [ ] Onglet "Mes stats" : libellé court pour les onglets Streamlit — utiliser `Mes stats` (pas "Mes stats avec l'escouade")
- [ ] Q11 cartes de session : sur `compute_squad_records()` existant — résumé par session : volume + roster dominant + issue — seul vrai nouveau composant de cet onglet

**Risque C3 :** `teammates_synergy.py` et `teammates_impact.py` ont peut-être des dépendances sur un état de sélection coéquipier géré dans la sidebar. Identifier ces dépendances avant de les déplacer.

### C4 — Profil hub

**Fichiers sources à lire :** `src/ui/pages/career.py` · `src/ui/pages/citations.py` · `src/ui/pages/settings.py`

- [ ] Créer `src/ui/pages/profile_hub.py`
- [ ] 3 onglets : Carrière · Citations · Paramètres

**Mapping composants → onglets :**

| Onglet | Composants sources | Action |
|---|---|---|
| Carrière | `career.py` | Réintégrer |
| Citations | `citations.py` | Réintégrer |
| Paramètres | `settings.py` + P6 Mon escouade (nouveau) | Réintégrer + nouveau bloc |

- [ ] P6 Mon escouade : formulaire de configuration du roster squad — persisté dans `filter_state.py` ou `app_settings`

### C5 — Explorer amélioré

**Fichiers sources à lire :** `src/ui/pages/explorer.py`

- [ ] Lire `explorer.py` en entier, cartographier filtres / résultats / détail
- [ ] Améliorer la hiérarchie visuelle : barre de recherche + facettes → résultats → détail inline
- [ ] Ajouter titre de section sobre (harmonisation visuelle avec les autres hubs — pas de L2 mais pas de page nue non plus)
- [ ] Actions contextuelles E4 : "Ouvrir dans Stats" · "Ouvrir dans Médias" · "Copier match id" — transmis via `session_state`

**Risque C5 :** les actions "Ouvrir dans Stats" et "Ouvrir dans Médias" nécessitent que le routing v7 accepte un état initial (filtre ou match_id préchargé). Vérifier que `page_router.py` supporte ce passage de contexte dès Phase A3.

### C6 — Médias connecté

**Fichiers sources à lire :** `src/ui/pages/media_library.py`

- [ ] Vérifier la réception du filtre transmis depuis A6 (Accueil) et E4 (Explorer)
- [ ] Si un filtre est présent dans `session_state` à l'ouverture, l'appliquer automatiquement
- [ ] Préserver le comportement existant si aucun filtre transmis

---

## Phase D — Vues nouvelles (liste fermée)

> Règle : uniquement D1–D5. Tout ajout hors liste est différé en v8.

### D1 — Forme récente (`Stats > Vue d'ensemble`)

**Question :** Suis-je dans une bonne ou mauvaise dynamique récente ?
**Type :** Sparklines synchronisées — 2 métriques visibles max (KD + V%)
**Données :** 10 à 20 derniers matchs du contexte `ctx_stats` filtré
**Règles :**
- 1 variation textuelle à côté des sparklines (ex: `+0.12 vs session précédente`)
- Pas plus de 2 métriques affichées simultanément — pas de multiselect
- Risque : surcharge si on ajoute une 3e métrique — tenir la limite

### D2 — Stabilité par session (`Stats > Sessions`)

**Question :** Mes sessions sont-elles homogènes ou très volatiles ?
**Type :** Bandes min/médiane/max par session (ou boxplot simple)
**Données :** Agrégats session sur `ctx_stats` — limiter aux 8 à 12 dernières sessions
**Règles :**
- Illisible au-delà de 12 sessions sur un écran standard — forcer la limite
- Métrique principale : KD (option secondaire : précision)
- Risque : sessions très courtes (< 3 matchs) produisent des bandes sans sens — filtrer ou signaler

### D3 — Delta perf escouade vs globale (`Escouade > Mes stats`)

**Question :** Est-ce que je joue mieux, moins bien ou pareil avec mon escouade ?
**Type :** Cartes delta comparatives (valeur squad · valeur globale · delta coloré)
**Données :** `ctx_squad` vs agrégat global — 4 métriques max : KD · Précision · V% · volume
**Règles :**
- Pas de graphe complexe — cartes delta lisibles en un coup d'œil
- Delta positif = accent `#3d8eb9` ou vert · Delta négatif = rouge · Neutre = gris
- Risque : si peu de matchs en squad, le delta est statistiquement non fiable — afficher un indicateur de volume sous chaque carte

### D4 — Résumé session solo / escouade (`Accueil A2 / A3`)

**Question :** Quelle est ma dernière session en résumé ?
**Type :** Bloc compact — volume + KD/Précision/V-D + 1 insight clé + CTA
**Données :** Dernière session du contexte correspondant (solo ou squad)
**Règles :**
- Hauteur fixe identique pour A2 et A3 — alignement horizontal
- Si aucune session squad récente : A3 affiche un état vide sobre (pas d'erreur)
- L'insight clé = variation la plus notable de la session (meilleur KD ? plus de défaites ?)

### D5 — Vue d'ensemble Stats (`Stats > onglet Vue d'ensemble`)

**Question :** Où en suis-je globalement ?
**Type :** Assemblage — D1 (forme récente) + KPI compact + blocs existants sélectionnés
**Contenu recommandé :**
- D1 Forme récente (sparklines)
- Résumé KPI étendu (KD · V% · Précision · meilleure map · mode dominant)
- 1 graphe existant le plus révélateur (à choisir lors de l'implémentation après lecture de `timeseries.py`)
**Règles :**
- 2 à 4 blocs max — pas un dump de tous les graphes
- C'est le premier onglet vu à l'ouverture de Stats — doit charger vite

---

## Phase E — Validation parité & retrait sidebar

> Objectif : confirmer que v7 couvre 100% des informations accessibles en v6, puis retirer la sidebar définitivement.

### E1 — Checklist parité fonctionnelle

Pour chaque item ci-dessous, vérifier que l'information est accessible en v7 :

- [ ] Dernier match + navigation ◀▶
- [ ] Timeseries KD / Précision / V% dans le temps
- [ ] Historique matchs filtrable (map, mode, playlist, période)
- [ ] Sessions : liste, comparaison, stabilité
- [ ] Coéquipiers fréquents + synergies + impact
- [ ] Sessions d'escouade + timeline
- [ ] Explorer : recherche par critères + détail match
- [ ] Médias : bibliothèque + association match
- [ ] Carrière + Citations
- [ ] Paramètres : langue, Mon escouade, préférences UI
- [ ] Filtres : application, persistance, réinitialisation
- [ ] Navigation deep link (URL avec paramètres)

### E2 — Checklist design

- [ ] Palette cohérente sur toutes les sections (backgrounds, textes, accent)
- [ ] Tous les `st.plotly_chart` utilisent `PLOTLY_V7_CONFIG`
- [ ] Police, tailles, ticks, légendes et hover labels cohérents sur les graphes clés
- [ ] Contraste suffisant entre fonds, gridlines, axes et annotations sur desktop et mobile
- [ ] Les exceptions locales au thème global sont documentées et justifiées par la lisibilité
- [ ] Aucun `border-radius` > 8px sur les cards
- [ ] Aucun emoji dans les titres de sections
- [ ] Typographie : 3 niveaux max, pas de hero metric géant
- [ ] Drill-down inline stable (pas de layout shift)
- [ ] Transmission contexte A6 → Médias vérifiée
- [ ] Transmission contexte Explorer → Stats / Médias vérifiée

### E3 — Retrait sidebar

- [ ] Supprimer le CSS de masquage — remplacer par suppression réelle dans `streamlit_app_v7.py`
- [ ] Identifier et supprimer les composants sidebar devenus orphelins
- [ ] Nettoyer `session_keys.py` des clés sidebar legacy
- [ ] Vérifier qu'aucun module importe encore `st.sidebar.*`

---

## Phase F — Infrastructure preview & mise en production

> Objectif : déployer v7 sur un sous-domaine dédié avec données partagées, valider, puis rebasculer le site principal sur v7.

### F0 — Décisions d'architecture

- [x] **Nom de branche confirmé** : `v7/cockpit`
- [x] **Stratégie déploiement** : répertoire séparé `/opt/levelup-preview/` + `docker-compose.override.yml` — ne pas modifier `docker-compose.yml` du service principal
- [ ] **Stratégie données** : le service preview monte `/opt/levelup/data` en **read-only** (`data:ro`) — pas de sync depuis le preview, évite les conflits d'écriture concurrente sur les mêmes fichiers DuckDB
- [ ] **Sync** : reste exclusivement sur le service principal (v6) — le preview consomme les données produites par v6
- [ ] **Bouton Sync UI** : vérifier si le disabler explicitement dans `streamlit_app_v7.py` (le volume `:ro` empêche les écritures DuckDB, mais les appels API Halo se déclencheraient quand même si l'utilisateur clique). Option : réutiliser `LEVELUP_DEMO_MODE=true` qui masque déjà le bouton sync via `is_demo_mode()` — vérifier que cela ne force pas des données de démo dans le contexte preview

### F1 — Mise en place initiale du service preview (VPS)

> À faire une seule fois via SSH sur le VPS.

- [ ] **Clone initial** : `git clone <repo> /opt/levelup-preview && cd /opt/levelup-preview && git checkout v7/cockpit`
- [ ] **Copier les configs depuis la prod** (une seule fois) :
  ```bash
  cp /opt/levelup/.env.local /opt/levelup-preview/
  cp /opt/levelup/app_settings.json /opt/levelup-preview/
  cp /opt/levelup/db_profiles.json /opt/levelup-preview/
  ```
- [ ] **Créer `/opt/levelup-preview/docker-compose.override.yml`** :
  ```yaml
  services:
    levelup:
      container_name: levelup-preview
      command: ["streamlit", "run", "streamlit_app_v7.py", "--server.port=8501"]
      ports:
        - "127.0.0.1:8502:8501"
      volumes:
        - /opt/levelup/data:/app/data:ro
        - ./db_profiles.json:/app/db_profiles.json
        - ./app_settings.json:/app/app_settings.json
        - ./.env.local:/app/.env.local
  ```
- [ ] **Lancer** : `docker compose up -d --build`
- [ ] Vérifier que le container démarre sur `streamlit_app_v7.py` (logs : `docker compose logs -f`)

**Mise à jour du preview après un push :**
```bash
cd /opt/levelup-preview && git pull && docker compose up -d --build
```

### F2 — CI/CD workflow preview

- [ ] Créer `.github/workflows/deploy-preview.yml`
- [ ] Trigger : push sur la branche v7 confirmée
- [ ] Job : SSH → pull branche v7 → `docker compose up -d levelup-preview`
- [ ] Pas de regen données démo pour le preview

### F3 — Reverse proxy sous-domaine

- [ ] Configurer le reverse proxy (Nginx/Caddy selon infra existante) pour pointer le sous-domaine preview vers le port 8503
- [ ] HTTPS si le sous-domaine est exposé publiquement

### F4 — Validation finale sur sous-domaine

- [ ] Rejouer la checklist E1 (parité fonctionnelle) sur le sous-domaine réel
- [ ] Vérifier le chargement des données live (pas des données de démo)
- [ ] Vérifier que le service principal (v6) continue de tourner sans impact

### F5 — Bascule site principal vers v7

- [ ] Fusionner la branche v7 dans `main`
- [ ] Mettre à jour `deploy.yml` : le déploiement principal utilise désormais `streamlit_app_v7.py`
- [ ] Archiver `streamlit_app.py` → `streamlit_app_v6_archive.py`
- [ ] Supprimer le service `levelup-preview` de `docker-compose.yml` (devenu inutile)
- [ ] Supprimer le workflow `deploy-preview.yml`

---

## Dépendances critiques

```
A0 → A1 → A2 → A3 → A4 → A5
               ↓
         B1 → B2 → B3 → B4
                    ↓
              C1 → C2 → C3 → C4 → C5 → C6
                               ↓
                             D1–D5
                               ↓
                               E1 → E2 → E3
                                          ↓
                                    F0 → F1 → F2 → F3 → F4 → F5
```

**Point de synchronisation clé :** A1/A1bis (design system + `PLOTLY_V7_CONFIG` + polish visuel des charts) doivent être stables avant d'attaquer B et C — tous les hubs s'appuient sur ces variables.

**Point de synchronisation clé :** A3 (routing) doit supporter le passage de contexte (filtre ou match_id) dès le départ — C5 (Explorer) et C6 (Médias) en dépendent.
