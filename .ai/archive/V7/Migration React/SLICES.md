# SLICES.md — Backlog de vertical slices

> Backlog ordonné de la migration, exprimé en parcours métier complets.
> Chaque slice doit produire quelque chose de testable de bout en bout.
> Source : PLAN_MIGRATION_FASTAPI_REACT.md § Backlog d'implémentation exécutable

---

## ⚠ Prérequis avant Slice 0 — Corpus de données de référence

> **À faire avant d'écrire la première ligne de code de migration.**

Sans un corpus figé, les tests de parité deviennent des comparaisons sur des données qui bougent à chaque sync. Impossible alors de distinguer une régression d'un nouveau match synchronisé.

### Ce qu'il faut figer maintenant

1. **Choisir un joueur de référence** — un joueur avec un historique stable, idéalement pas en sync active pendant la migration.
2. **Créer un dump DuckDB figé** — copie des DBs pertinentes (`shared_matches_v2.duckdb`, `stats.duckdb` du joueur, `metadata.duckdb`) dans un répertoire `tests/fixtures/ref_player/`. Ce dump ne doit jamais être écrasé par une sync réelle.
3. **Figer plusieurs scopes de filtres** — au minimum :
   - Période complète (aucun filtre)
   - Une période courte connue (ex : dernier mois, dates fixes)
   - Une session solo connue
   - Une session escouade connue
4. **Figer une liste de `match_id` de référence** — 5 à 10 matchs couvrant des cas variés : victoire, défaite, ranked, BTB, match avec bot, match avec armes inhabituelles.
5. **Capturer les valeurs de référence Streamlit** — pour chaque scope figé, noter les valeurs critiques actuelles : rang, XP total, LUSR, nombre de lignes Match History, score d'un match de référence. Ce sont les golden values contre lesquelles les tests de parité compareront.

### Où vivent ces fixtures

```
tests/
  fixtures/
    ref_player/
      shared_matches_v2.duckdb   ← copie figée, jamais écrasée
      stats.duckdb               ← copie figée
      metadata.duckdb            ← copie figée
    scopes/
      full_period.json           ← FilterContextInput scope complet
      last_month.json            ← FilterContextInput période courte
      session_solo.json          ← FilterContextInput session solo
      session_squad.json         ← FilterContextInput session escouade
    golden_values/
      career.json                ← rang, XP, LUSR attendus
      match_history_full.json    ← cardinalité, premier match_id attendu
      match_ref_<id>.json        ← score, roster, médailles d'un match connu
```

### Règle d'usage

Les tests de parité backend (`tests/parity/`) pointent exclusivement sur `tests/fixtures/ref_player/`. Jamais sur les DBs de production.

### Taille des fixtures et stockage

Les DBs DuckDB de production peuvent peser des centaines de Mo. Options par ordre de préférence :

1. **Corpus réduit** (recommandé) : extraire un sous-ensemble représentatif (~500 matchs, couvrant tous les cas de test) dans des DBs allégées. Script `scripts/create_test_corpus.py` à créer.
2. **Git LFS** : si le corpus réduit dépasse 50 Mo, utiliser Git LFS pour les fichiers `.duckdb` du corpus.
3. **Stockage hors-repo** : bucket S3/B2 avec script de téléchargement `make fetch-fixtures`. Dernier recours uniquement.

**Règle** : les fixtures ne doivent jamais dépasser 100 Mo au total. Au-delà, réduire le corpus ou passer à Git LFS.

### Blocages explicites avant tout preview React

Les points suivants sont bloquants avant d'ouvrir une première surface React en état `preview` :

1. Le corpus `tests/fixtures/ref_player/` est commité et versionné.
2. `tests/parity/` existe réellement et couvre au minimum `filters/resolve`, Setup/Settings et le premier écran MVP branché.
3. `DEMO_MODE` retourne les mêmes schémas que le mode normal sur `/bootstrap`, `/players` et les premiers endpoints métier.
4. Le shell React sait consommer et normaliser les anciens deep links sans dépendre d'un `page=` interne.

---

## Règle de découpage

1. Partir d'une intention utilisateur observable
2. Identifier le minimum de contexte nécessaire : joueur courant, filtres, deep link, auth
3. Exposer un contrat d'API suffisant pour fermer la page — pas pour "finir la couche data"
4. Construire le rendu React complet de la route, loading/empty/error inclus
5. Ajouter les tests de parité et l'instrumentation minimale avant de déclarer le slice livrable

**Anti-patterns à éviter** :
- Une "phase API" qui produit des endpoints sans écran utilisable
- Une "phase composants" qui produit une bibliothèque UI sans parcours branché
- Un epic défini comme "terminer auth front" ou "terminer tous les endpoints" sans route métier cible
- La duplication temporaire de calculs dans le front pour compenser un backend pas encore tranché

---

## Statuts

| Statut | Signification |
|--------|--------------|
| `todo` | Pas encore commencé |
| `in-progress` | En cours d'implémentation |
| `preview` | Code présent, tests de parité en cours |
| `canonical` | Parité validée, React est le front officiel |
| `retired` | Streamlit décommissionné pour cette surface |

---

## Slice 0a — Shell, bootstrap et plomberie

**Statut** : `canonical` ✅ — 2026-04-13  
**Dépendances** : aucune (mais corpus de référence prérequis)

> **Objectif** : valider que la plomberie bout en bout fonctionne — FastAPI démarre, le shell React se monte, le bootstrap renvoie un joueur, le DEMO_MODE est utilisable.

### Backend
- Créer l'app FastAPI, la convention `/api/v1` et le socle Pydantic des schémas transverses
- Implémenter `GET /api/v1/bootstrap`
- Implémenter `GET /api/v1/players`
- Implémenter `GET /api/v1/health`
- Mettre en place la session web backend (cookie signé + stockage fichier)
- Définir un middleware `request_id` + enveloppe d'erreurs unifiée (`ApiError`)
- Configurer `structlog` avec output JSON
- Implémenter `DEMO_MODE` : bypass auth, pointage sur fixtures, mêmes schémas que le mode normal
- Exporter le schéma OpenAPI + pipeline `openapi-typescript` → `apps/web/src/lib/api/generated.ts`

### Frontend
- Créer le shell Vite/React/Router/Query/Zustand
- Poser les routes `/setup`, `/settings`, `/players/:playerSlug/*`
- Créer `useAppShellStore`
- Configurer MSW pour les tests et le dev offline
- Ajouter une couche de compatibilité qui redirige les anciens deep links Streamlit vers les routes canoniques React
- Brancher bootstrap + hydration du joueur courant + langue

### Tests obligatoires
- Tests unitaires backend : `bootstrap`, `players`, `health`, `DEMO_MODE`
- Tests composant front : shell mount, store hydration, redirect deep links
- Tests contrat : le schéma OpenAPI exporté est cohérent avec les réponses
- E2E Playwright : navigation shell de base en DEMO_MODE

### Stores / query keys
- `useAppShellStore`
- `['bootstrap']`
- `['players']`

### Sortie tangible
- Shell React monté en DEMO_MODE
- Joueur courant sélectionnable
- Types TS générés et consommés
- Logging structuré opérationnel

### Critères de recette
- Le corpus de référence et `DEMO_MODE` existent réellement dans le repo
- `make dev` lance API + front en une commande
- Le shell React démarre avec le même joueur courant que le shell Streamlit quand le contexte est connu
- Un changement de joueur recharge proprement le contexte
- Les anciens deep links sont soit supportés, soit redirigés proprement vers la forme canonique
- Les types TS sont générés et à jour (`make check-types` passe)
- Les tests unitaires / composant / E2E sont verts

---

## Slice 0b — Contrat de filtres (spike dédié)

**Statut** : `canonical` ✅ — 2026-04-13  
**Dépendances** : Slice 0a

> ⚠ **C'est le vrai test d'architecture de toute la migration, pas bootstrap.**
>
> `POST /filters/resolve` est le contrat qui remplace les shadow keys, les reruns, `GAP_MINUTES_FIXED`, toute la logique de cohérence du scope actuel dans `streamlit_app.py` + `filters_render.py` + `filter_state.py`. Si ce contrat est mal défini ou instable, tous les écrans MVP qui en dépendent régressent simultanément.

### Lecture obligatoire avant implémentation
- `src/app/filters_render.py` (logique de résolution des sessions, `GAP_MINUTES_FIXED`, shadow keys)
- `src/app/filter_state.py` (état des filtres)
- `src/app/state.py` + `session_keys.py` (AppState, clés session)
- `src/analysis/sessions.py` (algorithme de groupement en sessions)

### Backend
- Implémenter `POST /api/v1/players/{player_slug}/filters/resolve`
- Valider contre les 4 scopes figés du corpus : période complète, période courte, session solo, session escouade
- Les options et compteurs retournés doivent correspondre exactement aux valeurs Streamlit

### Frontend
- Créer `useGlobalFilterStore`
- Implémenter le cycle URL → store → API → queries (voir INVARIANTS.md § 2)
- Synchronisation URL seulement pour les filtres partageables

### Tests obligatoires
- **Tests de parité** : 4 scopes × comparaison avec golden values Streamlit (options, compteurs, effective)
- Tests unitaires backend : parseFilterContextInput, edge cases (filtres videsiles, sessions inconnues, cascade vide)
- Tests composant front : FilterStore initialisation, sync URL, transition entre scopes

### Stores / query keys
- `useGlobalFilterStore`
- `['filters-resolve', playerSlug, filterContextHash]`

### Sortie tangible
- Filtres résolus côté API avec parité démontrée
- Le front consomme les filtres sans logique de résolution locale

### Critères de recette
- Le résolveur de filtres renvoie des options et compteurs **identiques** à l'état Streamlit pour les 4 scopes figés
- Aucune shadow key ou rerun nécessaire côté front
- Les tests de parité sont verts et versionné

---

## Slice 1 — Setup / Auth / Settings

**Statut** : `canonical` ✅ — 2026-04-13  
**Dépendances** : Slice 0a + 0b

### Backend
- Implémenter `/api/v1/setup/status`
- Implémenter `/api/v1/auth/device-flow/start` et `/api/v1/auth/device-flow/{attempt_id}`
- Implémenter `/api/v1/setup/players`
- Implémenter `/api/v1/setup/smoke-test` + `/api/v1/jobs/{job_id}`
- Implémenter `GET/PATCH /api/v1/settings` et `POST /api/v1/settings/media/reset-index`
- Formaliser la machine d'état `choose_mode -> auth -> player -> smoke_test -> done`

### Frontend
- Écran setup en plusieurs étapes
- Polling du Device Code Flow
- Écran smoke test avec progression temps réel
- Page settings avec formulaires groupés par section
- Reprise propre après refresh navigateur pendant un Device Code Flow ou un smoke test

### Stores / query keys
- `useSetupFlowStore`
- `useSettingsDraftStore`
- `['setup-status']`
- `['device-flow', attemptId]`
- `['settings']`
- `['job', jobId]`

### Tests obligatoires
- Tests unitaires backend : machine d'état setup (transitions, états terminaux), device flow lifecycle, settings CRUD, smoke test job
- Tests de non-régression : snapshots `SetupStatusResponse`, `SettingsResponse`
- Tests composant front : wizard steps, device code polling, settings form validation
- E2E Playwright : parcours setup complet en DEMO_MODE, modification settings + vérification persistance

### Sortie tangible
- Un utilisateur neuf peut configurer l'app sans passer par Streamlit
- Un utilisateur configuré peut modifier ses settings depuis React

### Critères de recette
- Setup bloque l'accès aux routes protégées tant qu'il n'est pas terminé
- Settings persistants après refresh navigateur
- Un attempt de Device Code Flow expiré réapparaît explicitement comme expiré
- Smoke test affichant les mêmes conclusions que la page Streamlit

---

## Slice 2 — Profil [V7 §7 : Carrière + Citations]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 0a, 0b et 1  
**Ref specs** : [FUNCTIONAL_SPECS.md § 7 Profil](FUNCTIONAL_SPECS.md#7-profil)

> La section V7 "Profil" regroupe deux sous-vues : Carrière et Citations.
> L'implémentation est phasée pour permettre une livraison MVP rapide.

### Phase A — Carrière (MVP P1)

#### Backend
- Implémenter `GET /api/v1/players/{player_slug}/pages/career`
- Optionnel : `/top-matches` et `/encounters` si le payload devient trop lourd

#### Frontend
- Route `/players/:playerSlug/profile/career`
- Cards résumé, jauges Plotly, historique XP, section LUSR
- Liens vers top matches et match detail

#### Tests obligatoires
- Tests de parité backend : rang, XP total, progression Hero, LUSR, top matches contre golden values
- Tests de non-régression : snapshot `CareerPageResponse`
- Tests composant front : rendu Career page + loading/error states + figures Plotly
- E2E Playwright : navigation vers Career, vérification rang affiché

### Phase B — Citations (post-MVP P2)

**Statut Phase B** : `canonical` ✅ — 2026-04-13

#### Backend
- Implémenter l'endpoint page-oriented citations avec filtered vs full

#### Frontend
- Sous-vue citations accessible depuis `/players/:playerSlug/profile/citations`
- Grille commendations + médailles + distribution Plotly

#### Tests obligatoires
- Tests de parité : mêmes totaux de citations, mêmes médailles, mêmes deltas filtre vs complet

### Stores / query keys
- `['career', playerSlug]`
- `['career', playerSlug, 'top-matches']`
- `['career', playerSlug, 'encounters']`
- `['citations', playerSlug, filterContextHash]`

### Sortie tangible
- Première page métier React démontrable en parité forte (Phase A)
- Section Citations complète (Phase B)

### Critères de recette
- Même rang et mêmes valeurs XP qu'en Streamlit
- Figures chargées sans adaptation métier côté front
- Mêmes totaux et mêmes deltas filtre vs complet (citations)

**Note** : `useCareerPageStore` n'est pas nécessaire si l'état local (panels, tabs) reste dans les composants via `useState`. Ne créer un store Zustand que si l'état doit survivre à un changement de route.

---

## Slice 3 — Stats [V7 §2 : Séries + Sessions + Historique]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 0a et 0b  
**Ref specs** : [FUNCTIONAL_SPECS.md § 2 Stats](FUNCTIONAL_SPECS.md#2-stats)

> La section V7 "Stats" regroupe 3 sous-vues : Séries temporelles (5 onglets),
> Comparaison de sessions (15 composants) et Historique des parties (17 colonnes).
> L'Historique est le candidat MVP ; les deux autres sont post-MVP.

### Phase A — Historique des parties (MVP P1)

#### Backend
- Implémenter `POST /api/v1/players/{player_slug}/pages/match-history/query`
- Implémenter `POST /api/v1/players/{player_slug}/pages/match-history/export`

#### Frontend
- Route `/players/:playerSlug/stats/history`
- Table riche AG Grid ou équivalente
- Pagination, tri, export, colonnes configurables
- Synchronisation avec `useGlobalFilterStore`

#### Tests obligatoires
- Tests de parité backend : cardinalité lignes, ordre, valeurs calculées (win_rate_hist, performance_score_relative)
- Tests de non-régression : snapshot `MatchHistoryPageResponse` (première page)
- Tests composant front : table rendering, pagination, sort, export button
- E2E Playwright : chargement Match History, tri, export CSV

### Phase B — Séries temporelles (post-MVP P3)

**Statut Phase B** : `canonical` ✅ — 2026-04-13

#### Backend
- Endpoint page-oriented timeseries + figures Plotly JSON
- Extraire au préalable toute logique métier encore piégée dans `src/ui/pages/timeseries.py` si elle n'est pas déjà couverte par un service stable

#### Frontend
- Route `/players/:playerSlug/stats/timeseries`
- Page analytics à 5 onglets (KPIs, Cumul, Forme, Intensité, Distributions), sans recalcul métier client

### Phase C — Comparaison de sessions (post-MVP P3)

**Statut Phase C** : `canonical` ✅ — 2026-04-13

#### Backend
- Endpoint page-oriented compare sessions + sélection A/B + contexte historique
- Stabiliser avant cela le modèle de sessions et l'état A/B pour éviter toute logique client cachée

#### Frontend
- Route `/players/:playerSlug/stats/sessions`
- Sélection A/B, radars, historiques, breakdowns (15 composants)

### Stores / query keys
- `useMatchHistoryTableStore` : pagination, tri, colonnes
- `useTimeseriesStore`
- `useSessionCompareStore`
- `['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]`
- `['timeseries', playerSlug, filterContextHash]`
- `['session-compare', playerSlug, filterContextHash, compareStateHash]`

### Sortie tangible
- Première grande table web réactive branchée sur le backend Python (Phase A)
- Analytics complètes V7 avec 3 sous-vues (Phase B+C)

### Critères de recette
- Parité ligne à ligne sur un échantillon critique (Historique)
- Export identique au scope affiché
- Mêmes séries et mêmes agrégats sur un scope donné (Séries)
- Même choix par défaut et mêmes deltas qu'en Streamlit (Sessions)

---

## Slice 4 — Explorer [V7 §5 : Filtres + Match View + Last Match]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 0a, 0b et 3  
**Ref specs** : [FUNCTIONAL_SPECS.md § 5 Explorer](FUNCTIONAL_SPECS.md#5-explorer)

> La section V7 "Explorer" intègre la recherche, les filtres cascade, les résultats
> **et** le détail Match View (4 onglets) dans la même section. La navigation Last Match
> est une vue dérivée de Match View + scope filtré.

### Phase A — Recherche + filtres cascade

#### Backend
- Implémenter `GET /api/v1/directory/gamertags/search`
- Implémenter `POST /api/v1/players/{player_slug}/pages/explorer/matches-query`
- Implémenter `POST /api/v1/players/{player_slug}/pages/explorer/player-query`

#### Frontend
- Route `/players/:playerSlug/explorer`
- Mode recherche joueur (fuzzy, top 8 suggestions)
- Mode filtres match (cascade 4 dimensions)
- Deep links vers un match ou un gamertag

### Phase B — Match View (4 onglets)

**Statut Phase B** : `canonical` ✅ — 2026-04-13

#### Backend
- Implémenter `GET /api/v1/players/{player_slug}/matches/{match_id}`
- Payload composé : header, rank, résumé, combat, équipe, médias, citations

#### Frontend
- Route `/players/:playerSlug/explorer/matches/:matchId` (ou modal/drawer dans Explorer)
- Tabs détail match : Résumé / Combat / Équipe / Médias / Citations
- Scoreboard 19 colonnes, armes via `v_weapon_kills`, labels via `weapon_labels`

### Phase C — Last Match (navigation prev/next)

**Statut Phase C** : `canonical` ✅ — 2026-04-13

#### Backend
- Implémenter `POST /api/v1/players/{player_slug}/pages/last-match/resolve`

#### Frontend
- Route `/players/:playerSlug/last-match` ou redirection logique depuis Accueil
- Navigation prev/next sur le scope filtré courant

### Stores / query keys
- `useExplorerStore` : searchMode, playerSearchInput, selectedMatchId, localMatchFilters, pagination
- `useMatchViewStore` : activeTab, selectedScoreboardRow, mediaLightboxIndex
- `useLastMatchStore` : resolvedMatchId, currentIndex, total
- `['gamertag-search', q]`
- `['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]`
- `['explorer', 'player', playerSlug, targetGamertag, filterContextHash]`
- `['match-view', playerSlug, matchId]`
- `['last-match', playerSlug, filterContextHash]`

### Tests obligatoires
- Tests unitaires backend : fuzzy search, gamertag resolution, cascade de filtres
- Tests de parité backend : score, roster, médailles, armes, citations sur matchs de référence
- Tests composant front : search input, results table, tabs rendering, scoreboard
- E2E Playwright : recherche joueur → résultats → clic match → Match View → navigation tabs → prev/next

### Sortie tangible
- Parcours complet recherche → résultat → détail match → navigation dans le même shell V7

### Critères de recette
- Même comportement de recherche et de cascade que Streamlit
- Même score, même outcome, même roster, mêmes armes, mêmes médailles
- Last Match pointe vers le même match que Streamlit pour un scope donné
- prev/next navigue sur la même liste ordonnée

---

## Slice 5 — Accueil [V7 §1 : Home Mission Control]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 2, 4 (Phase B) et 8  
**Ref specs** : [FUNCTIONAL_SPECS.md § 1 Accueil](FUNCTIONAL_SPECS.md#1-accueil-home-mission-control)

> La Home est une route composée qui agrège les résultats de plusieurs endpoints déjà exposés
> (Career, Match View, Media) plus les endpoints live (Battle Pass, Challenges).
> À traiter une fois les dépendances prêtes.

### Backend
- Endpoint agrégateur `/api/v1/players/{player_slug}/pages/home`
- Sous-resource Battle Pass : `GET /api/v1/players/{player_slug}/battlepass` (données **live API Halo**, cache process 4h)
- Sous-resource Challenges : `GET /api/v1/players/{player_slug}/challenges` (données **live API Halo**, cache process 1h)
- Summaries sessions, recent matches, recent media, dernier match

> ⚠ Battle Pass et Challenges sont des données **live API** avec cache process-level.
> Les autres widget de la Home sont dérivés de données locales.
> Cette distinction impacte le design des query keys et des loading states.

### Frontend
- Route `/` (racine)
- Hero Card, signaux de tendance, quick actions
- Widgets Battle Pass + Challenges
- Timeline activité, résumé sessions, médias récents
- Embed dernier match (réutilise Match View)

### Stores / query keys
- `useHomeStore`
- `['home', playerSlug]`
- `['home', playerSlug, 'battlepass']`
- `['home', playerSlug, 'challenges']`

### Tests obligatoires
- Tests de parité : même contenu battle pass/défis, mêmes highlights, même ordre matchs récents
- E2E Playwright : chargement Home en DEMO_MODE

### Critères de recette
- Même contenu battle pass/défis et mêmes highlights que Streamlit V7

---

## Slice 6 — Escouade [V7 §3 : Synergies + Contributions]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 0a, 0b, 4 (Phase B)  
**Ref specs** : [FUNCTIONAL_SPECS.md § 3 Escouade](FUNCTIONAL_SPECS.md#3-escouade)

> Section la plus complexe de l'app : 13 sous-modules, 2 onglets riches (Synergies / Contributions),
> sélecteur multi-coéquipiers (max 3), données cross-player via shared.

### Backend
- Endpoints teammates pour sélection, overview, synergy, impact, weapons
- Clarifier avant implémentation le modèle d'accès multi-joueur et le périmètre exact des lectures cross-player depuis `shared`
- Données PersonalScores API pour le radar complémentarité (6 axes)
- Données highlight_events pour heatmap impact + ranking (8 emojis, scoring MVP/LVP)

### Frontend
- Route `/players/:playerSlug/squad`
- Sélecteur multi (max 3) + panneau légende fixe
- Onglet Synergies : maps charts, form score, timeline, impact heatmap + ranking
- Onglet Contributions : stats/min, radar 6-axes, heatmap intensité kills, métriques trio, armes top-12, butterfly frag/mort, médailles
- Tableau historique 12 colonnes (250 rows)

### Stores / query keys
- `useTeammatesStore`
- `['teammates', playerSlug, filterContextHash, teammatesSelectionHash]`

### Tests obligatoires
- Tests de parité : même set coéquipiers, mêmes radars, mêmes stats impact, mêmes enrichissements armes
- Tests composant : sélecteur multi, tabs rendering, heatmap

### Critères de recette
- Même set de coéquipiers, mêmes radars, mêmes stats d'impact et mêmes enrichissements armes

---

## Slice 7 — Synthèse [V7 §4 : Solo vs Escouade]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slices 0a, 0b  
**Ref specs** : [FUNCTIONAL_SPECS.md § 4 Synthèse](FUNCTIONAL_SPECS.md#4-synthèse)

> Objective Analysis (ancienne page autonome) est **absorbée** dans l'Escouade (radar complémentarité
> PersonalScores) et potentiellement ici pour les trends objectifs.

### Backend
- Endpoint `/api/v1/players/{player_slug}/pages/synthesis`
- Optional : sous-resource objective analysis si les trends objectifs méritent un endpoint séparé

### Frontend
- Route `/players/:playerSlug/synthesis`
- Sélecteur période (5 options : all, 2y, 1y, 1m, 1w)
- Breakdown carte/mode, heatmap temporelle, top semaine
- Bipolaire Solo vs Escouade (6 métriques, Cyan/Vert)

### Stores / query keys
- `['synthesis', playerSlug, filterContextHash, period]`

### Tests obligatoires
- Tests de parité : mêmes agrégats solo/escouade, même bipolaire, mêmes heatmaps

### Critères de recette
- Mêmes agrégats solo/escouade et mêmes ratios objectifs

### Décision : Objective Analysis

L'ancienne page autonome Objective Analysis (P4) **n'est pas migrée en tant que section V7 distincte**.
- Les données d'awards objectifs (PersonalScores API) sont consommées par le **radar complémentarité** de l'Escouade (§3.7.2)
- Si des trends objectifs spécifiques sont nécessaires, ils peuvent être ajoutés ici dans Synthèse ou comme onglet futur
- La query key `['objective-analysis', ...]` est supprimée — les données transitent via `['teammates', ...]` et `['synthesis', ...]`

---

## Slice 8 — Médias [V7 §6 : Galerie + Lightbox]

**Statut** : `canonical` ✅ — 2026-04-13 (Phase A)  
**Dépendances** : Slice 4 (Phase B)  
**Ref specs** : [FUNCTIONAL_SPECS.md § 6 Médias](FUNCTIONAL_SPECS.md#6-médias)

### Backend
- Exposer l'index media, les enrichissements, les thumbs et les jobs de reset/reindex
- Endpoint galerie : `POST /api/v1/players/{player_slug}/pages/media` (body `MediaQueryRequest`)
- Endpoint reindex : `POST /api/v1/settings/media/reset-index`

### Frontend
- Route `/players/:playerSlug/media`
- Grille media avec 8 contrôles toolbar (tri, filtre, groupement, affichage)
- 2 modes de groupement (série/session ou catégorie)
- Lightbox avec navigation ◀ ▶ + métadonnées match
- Likes persistés en **localStorage** uniquement (pas de sync serveur)
  - Clé : `levelup_liked_media` → `Set<media_id>`
  - Migration depuis l'ancien `mv2_liked_media` (session_state) vers localStorage au premier chargement React

### Stores / query keys
- `useMediaStore`
- `['media', playerSlug, mediaFilterHash]`

### Tests obligatoires
- Tests de parité : même cardinalité, mêmes groupements, même navigation vers match
- Tests composant : lightbox navigation, likes toggle, filtres toolbar

### Critères de recette
- Même cardinalité, mêmes groupements, même navigation vers match
- Likes persistés en localStorage et restaurés au refresh

---

## Slice 9 — Décommission Streamlit UI

**Statut** : `canonical` ✅  
**Date** : 2026-04-13  
**Dépendances** : Toutes les slices MVP/P2/P3 canonical

### Backend / produit
- Supprimer les endpoints temporaires devenus redondants si besoin
- Verrouiller la compatibilité des contrats utiles restants
- Basculer les routes finales
- Mettre en place redirects legacy
- Retirer les pages Streamlit absorbées ou non exposées

### Critères de recette
- Aucun parcours MVP/P1/P2 ne dépend encore d'un rendu Streamlit
- La documentation d'installation et de lancement est à jour

### Définition of done globale — `canonical` ✅ 2026-04-14

> **DoD globale satisfaite** — migration React/FastAPI terminée.

1. ✅ **Toutes les sections V7 du MVP/P1 sont en état `canonical`** : Setup, Settings, Profil, Stats (Historique), Explorer (+ Match View)
2. ✅ **Les sections P2/P3 sont soit `canonical`, soit explicitement dépriorisées** avec une décision écrite dans `MIGRATION_MASTER.md` : Accueil, Escouade, Synthèse, Médias, Stats Phases B+C
3. ✅ **Streamlit ne délivre plus aucune surface active** — `streamlit_app.py` et `streamlit_app_v7.py` archivés (headers ARCHIVED ajoutés, launcher bascule vers React/FastAPI)
4. ✅ **Les tests de parité sont tous verts** — suite complète (hors integration) sans failure
5. ✅ **La documentation est à jour** : INSTALL.md, FR/INSTALL.md, CONFIGURATION.md, FR/CONFIGURATION.md, FR/README.md, README_FR.md — sans mention de Streamlit comme front principal
6. ✅ **Les imports `src/ui/pages/` dans les services FastAPI** sont intentionnels (logique métier réutilisée, pas de rendu Streamlit) — aucun rendu Streamlit actif
7. ⚪ **Validation FUNCTIONAL_SPECS.md** — optionnel / dépriorisé (peut être réalisé par section lors du polish P2/P3)

**Ce que "terminée" ne signifie pas** : le code Streamlit peut rester présent dans le repo pour référence ou archivage — ce qui change c'est qu'il n'est plus le front actif. La décommission complète du code est optionnelle et peut être faite séparément.

---

## Modèle de cohabitation

| État surface | Front canonique | Front secondaire | Règle |
|---|---|---|---|
| Legacy seule | Streamlit | aucun | surface non commencée côté React |
| Preview React | Streamlit | React | accès React réservé au dev, au flag ou à une URL dédiée |
| Bascule canonique | React | Streamlit | React devient l'entrée principale, Streamlit reste rollback court terme |
| Décommissionnée | React | aucun | la route Streamlit est retirée ou redirigée |

**Règles de bascule** :
1. Ouvrir une surface React en preview interne seulement après validation des tests de parité critiques
2. Passer en canonique React uniquement quand la navigation, l'instrumentation et le rollback sont prêts
3. Garder la version Streamlit seulement comme filet de sécurité court terme, pas comme double maintenance indéfinie
4. Retirer la version Streamlit dès qu'une surface React a stabilisé ses chiffres sur une période observée

### ⚠ Règle de gel Streamlit — à appliquer dès le premier écran en preview

> **Dès qu'un écran passe en état `preview` React, la version Streamlit de cet écran est gelée.**

Concrètement :
- **Aucun nouveau feature** sur la version Streamlit d'un écran en preview React
- **Les bugs critiques uniquement** peuvent être corrigés côté Streamlit si la version React n'est pas encore canonique — et la même correction doit être appliquée côté React dans la foulée
- **Aucune modification de logique métier** dans le module Python source uniquement pour servir la version Streamlit — si la logique change, elle change pour les deux et les tests de parité sont mis à jour

**Pourquoi cette règle est critique** : sans elle, la pression naturelle pendant la cohabitation est de "juste patcher Streamlit" sur chaque bug signalé, au lieu de corriger dans le nouveau front. Au bout de quelques semaines, la version Streamlit accumule des comportements que la version React n'a pas, la parité se dégrade, et la migration devient indéfiniment bloquée.

**Opérationnellement** : tenir un tableau simple dans `MIGRATION_MASTER.md` (section État courant) avec le statut de gel par écran.

---

## Stores front minimaux à créer dès maintenant

- `useAppShellStore` : joueur courant, locale, feature flags, état bootstrap
- `useGlobalFilterStore` : filter context, options résolues, hash de contexte, dirty state
- `useSetupFlowStore` : wizard, attempt auth, jobs
- `useSettingsDraftStore` : brouillon settings + ui prefs locales
- `useMatchHistoryTableStore` : pagination, tri, colonnes (Stats/Historique)
- `useExplorerStore` : mode de recherche, input, selectedMatchId, pagination locale (Explorer)
- `useMatchViewStore` : tab active, sous-sélection UI (Explorer/Match View)
- `useLastMatchStore` : index courant et voisins (Explorer/Last Match)
- `useTeammatesStore` : coéquipiers sélectionnés, cache, scope (Escouade)
- `useMediaStore` : filtres, tri, groupement, likes localStorage (Médias)
- `useHomeStore` : battle pass focus, prefetch (Accueil)
- `useTimeseriesStore` : onglet actif (Stats/Séries)
- `useSessionCompareStore` : sélection A/B (Stats/Sessions)

---

## Correspondance Slices ↔ Sections V7

> Référence rapide pour les agents et les revues de PR.

| Slice | Section V7 | Ref specs | Priorité |
|-------|------------|-----------|----------|
| 0a | Shell + Bootstrap | §0 | P0 |
| 0b | Filtres resolve | §0.3 | P0 |
| 1 | Setup / Auth / Settings | §8 | P0 |
| 2 | **Profil** (Carrière + Citations) | §7 | P1 / P2 |
| 3 | **Stats** (Historique + Séries + Sessions) | §2 | P1 / P3 |
| 4 | **Explorer** (Filtres + Match View + Last Match) | §5 | P1 |
| 5 | **Accueil** (Home Mission Control) | §1 | P2 |
| 6 | **Escouade** (Synergies + Contributions) | §3 | P2 |
| 7 | **Synthèse** (Solo vs Escouade) | §4 | P3 |
| 8 | **Médias** (Galerie + Lightbox) | §6 | P2 |
| 9 | Décommission Streamlit | — | Final | `canonical` ✅ 2026-04-13 |

## Query keys TanStack normalisées (référence complète)

```
['bootstrap']
['players']
['filters-resolve', playerSlug, filterContextHash]
['settings']
['setup-status']
['job', jobId]
['device-flow', attemptId]
['career', playerSlug]
['career', playerSlug, 'top-matches']
['career', playerSlug, 'encounters']
['citations', playerSlug, filterContextHash]
['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]
['gamertag-search', q]
['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]
['explorer', 'player', playerSlug, targetGamertag, filterContextHash]
['match-view', playerSlug, matchId]
['last-match', playerSlug, filterContextHash]
['media', playerSlug, mediaFilterHash]
['home', playerSlug]
['home', playerSlug, 'battlepass']
['home', playerSlug, 'challenges']
['timeseries', playerSlug, filterContextHash]
['session-compare', playerSlug, filterContextHash, compareStateHash]
['teammates', playerSlug, filterContextHash, teammatesSelectionHash]
['synthesis', playerSlug, filterContextHash, period]
```
