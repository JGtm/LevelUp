# Audit Consolidé — Go API vs FastAPI : Parité, Architecture & Plan d'Action

> Date : 2026-04-16 — Fusion des deux audits (parité champ-par-champ + qualité architecture)
>
> Périmètre audité :
> - `apps/go-api/` (~17 500 LOC Go) — backend cible
> - `apps/api/` (~5 000 LOC Python/FastAPI) — backend transitoire en production
> - `apps/web/` — frontend React consommant le contrat FastAPI
> - `LevelUp/src/` (~55 000 LOC Python) — legacy Streamlit de référence
> - Infra racine : Dockerfile, docker-compose.yml, Makefile

---

## Table des matières

- [Résumé exécutif](#résumé-exécutif)
- [Partie 1 — Parité fonctionnelle](#partie-1--parité-fonctionnelle)
  - [1.1 Cadrage : le bon référentiel](#11-cadrage--le-bon-référentiel)
  - [1.2 Matrice de correspondance des routes](#12-matrice-de-correspondance-des-routes)
  - [1.3 Résumé comptable](#13-résumé-comptable)
  - [1.4 Différences DTO champ par champ](#14-différences-dto-champ-par-champ)
  - [1.5 Sync Engine](#15-sync-engine)
  - [1.6 Algorithmes d'analyse](#16-algorithmes-danalyse)
  - [1.7 Authentification et onboarding](#17-authentification-et-onboarding)
  - [1.8 Migrations, Notifications, Ops](#18-migrations-notifications-ops)
  - [1.9 Runtime et infrastructure](#19-runtime-et-infrastructure)
  - [1.10 Compatibilité sessions/cookies](#110-compatibilité-sessionscookies)
  - [1.11 Synthèse des écarts critiques](#111-synthèse-des-écarts-critiques)
- [Partie 2 — Audit qualité code Go](#partie-2--audit-qualité-code-go)
  - [2.1 Architecture hexagonale — design vs réalité](#21-architecture-hexagonale--design-vs-réalité)
  - [2.2 Violations handlers — matrice détaillée](#22-violations-handlers--matrice-détaillée)
  - [2.3 Bugs et risques sécurité (P0)](#23-bugs-et-risques-sécurité-p0)
  - [2.4 Gestion d'erreur HTTP incohérente](#24-gestion-derreur-http-incohérente)
  - [2.5 Dépassements de taille (P1)](#25-dépassements-de-taille-p1)
  - [2.6 Violations DRY / architecture (P2)](#26-violations-dry--architecture-p2)
  - [2.7 Stubs et workarounds encore présents](#27-stubs-et-workarounds-encore-présents)
  - [2.8 Points forts](#28-points-forts)
- [Décision stratégique](#décision-stratégique)
- [Partie 3 — Plan d'action consolidé](#partie-3--plan-daction-consolidé)
  - [Estimation d'effort total](#estimation-deffort-total)
  - [Ordre d'exécution recommandé (P0)](#ordre-dexécution-recommandé-p0)
  - [P0 — Prérequis bascule (BLOQUANT)](#p0--prérequis-bascule-bloquant)
  - [P1 — Bugs / sécurité](#p1--bugs--sécurité)
  - [P2 — Architecture / qualité](#p2--architecture--qualité)
  - [P3 — Évolution fonctionnelle](#p3--évolution-fonctionnelle)
  - [P4 — Améliorations UX / produit](#p4--améliorations-ux--produit)
- [Partie 4 — Audit Tests & Observabilité](#partie-4--audit-tests--observabilité)
  - [4.1 Inventaire de l'existant](#41-inventaire-de-lexistant)
  - [4.2 Matrice de couverture par couche](#42-matrice-de-couverture-par-couche)
  - [4.3 Analyse des lacunes critiques](#43-analyse-des-lacunes-critiques)
  - [4.4 Observabilité et monitoring](#44-observabilité-et-monitoring)
  - [4.5 CI/CD — ce qui tourne vs ce qui devrait tourner](#45-cicd--ce-qui-tourne-vs-ce-qui-devrait-tourner)
  - [4.6 Recommandations priorisées](#46-recommandations-priorisées)
  - [P0 — Prérequis bascule (BLOQUANT)](#p0--prérequis-bascule-bloquant)
  - [P1 — Bugs / sécurité](#p1--bugs--sécurité)
  - [P2 — Architecture / qualité](#p2--architecture--qualité)
  - [P3 — Évolution fonctionnelle](#p3--évolution-fonctionnelle)
  - [P4 — Améliorations UX / produit](#p4--améliorations-ux--produit)

---

## Résumé exécutif

1. **Le portage Go est substantiel sur le noyau technique.**
   Le moteur de sync, les migrations DuckDB, le Device Code Flow, l'échange Halo, une partie importante de l'analyse métier et les opérations CLI existent réellement côté Go. Le projet n'est pas un simple squelette.

2. **Ce portage ne suffit pas encore à remplacer la chaîne Python en production.**
   Docker, compose, Makefile de développement et génération de types frontend restent arrimés à FastAPI. La cible d'exécution effective n'est donc pas Go aujourd'hui.

3. **Le vrai blocage est le contrat produit.**
  Le frontend React appelle plusieurs routes absentes ou incompatibles côté Go. Il existe aussi un drift entre le runtime, les types générés, les fixtures MSW/E2E et les backends, notamment autour de `GET /setup/status`.

4. **L'onboarding Go reste incohérent de bout en bout.**
   L'authentification Microsoft/Halo est partiellement portée, mais bootstrap, provisioning de l'identité Halo et création de profil ne sont pas alignés.

5. **L'architecture Go est bonne au niveau des packages, mais incomplète dans les dépendances réelles.**
   Les handlers composent encore directement repos et services, importent l'infrastructure, et dupliquent le pattern `resolvePlayer → NewRepo → NewService`.

6. **Plusieurs défauts internes concrets.**
   Fuite potentielle de connexions dans le pool DuckDB, concaténation SQL dans le backfill, erreurs silencieusement ignorées dans MatchView, fichiers trop gros, logique métier dans `setup.go` et divergence de contrat d'erreur HTTP.

7. **L'architecture actuelle est mono-titre par construction.**
   Tout le système est calibré Halo Infinite : endpoints d'échange Halo, URLs publiques, métadonnées, conventions DuckDB, wording produit. Si le support multi-titres devient un objectif, il faudra introduire un namespace explicite par titre au niveau stockage, configuration, contrats API et auth. Ce n'est pas un blocage pour la bascule HI-only, mais c'est un point de design non traité.

8. **La surface d'automatisation reste Python-first au-delà du seul `ci.yml`.**
   `release.yml`, `bump-version.yml`, `deploy.yml`, `test-deploy-precheck.yml` et le packaging portable produisent encore des artefacts Python/uvicorn. Le portage Go n'est donc pas seulement incomplet côté runtime ; il ne dispose pas encore d'une chaîne de build/release/deploy cohérente.

**Verdict global : la base Go est sérieuse et crédible, mais la dernière ligne droite n'est pas une formalité. Elle demande un réalignement de contrat, un nettoyage de composition et une fermeture de plusieurs faux-semblants runtime.**

### Ce que la seconde passe confirme côté Go

Cette table corrige deux lectures trop pessimistes : certains sous-systèmes existent bel et bien, mais leur intégration produit reste incomplète.

| Domaine | État après relecture | Nuance importante |
|---------|----------------------|-------------------|
| Sync/backfill | largement porté | le cœur existe, mais ne garantit pas la parité du produit web |
| Migrations DuckDB | réellement portées | framework Go crédible et déjà alimenté |
| Device Code Flow + échange Halo | présents | la faiblesse porte sur la persistance et le provisioning, pas sur l'absence totale d'auth |
| MSAL silent helper | présent | AcquireTokenSilent et sérialisation mémoire existent, mais le cache n'est pas persisté dans sync_meta |
| Weapon parser | présent | les briques weapon_parser / correlation / reconciliation existent, mais la chaîne complète legacy n'est pas prouvée branchée sur les bons flux |
| Discord / CLI / ops | largement portés | bonne couverture technique, mais hors du chemin critique de bascule React → Go |
| Analyses UI avancées | partielles | le manque est tolérable tant que le frontend React ne les consomme pas |

**Conclusion intermédiaire** : le constat principal n'est pas "Go est vide", mais "Go est riche techniquement sans être encore prêt comme backend de référence du produit actuel". **Le portage Go n'est pas seulement incomplet, il est parfois aligné sur une autre cible fonctionnelle que celle d'`apps/web`.**

### Ce qui n'est PAS considéré comme une régression problématique

- Le remplacement de Streamlit par React/TanStack Router
- La redistribution des anciennes pages vers des routes plus explicites
- L'existence temporaire de scripts Python ou de surfaces techniques encore non utilisées par `apps/web`
- L'absence de certains algorithmes purement UI tant que le frontend actuel ne les consomme pas

Le problème commence quand le statut projet laisse entendre que l'application Go remplace déjà le backend réel.

---

## Partie 1 — Parité fonctionnelle

### 1.1 Cadrage : le bon référentiel

La parité ne se mesure **pas** contre le legacy Streamlit. Elle se mesure contre le **contrat effectivement consommé aujourd'hui par le frontend React** (`apps/web/`) via le backend FastAPI (`apps/api/`).

L'inventaire du frontend React et de ses artefacts d'intégration expose **31 surfaces d'appel** via `fetch` natif avec `credentials: "include"` (cookies httpOnly). Le schéma de référence reste défini par les modèles Pydantic de `apps/api/app/schemas/` et consommé côté runtime par les hooks TanStack Query de `apps/web/src/features/*/queries.ts`.

Un backend Go fonctionnellement complet mais non-substituable au backend FastAPI ne constitue **pas** une migration terminée.

---

### 1.2 Matrice de correspondance des routes

> **Colonnes React** : fichier `queries.ts`, hook TanStack Query, et page/route qui consomme l'endpoint.
> Les endpoints marqués 🔶 sont définis dans `generated.ts` (OpenAPI) mais **aucun hook React ne les appelle**.
>
> **Important** : cette matrice mélange volontairement quatre catégories différentes pour mesurer le drift global : runtime réellement consommé, hooks/keys/types non consommés, artefacts de test/codegen (MSW, Playwright, `generated.ts`) et endpoints Go-only. Les totaux qui suivent ne doivent donc pas être lus comme un simple relevé du trafic runtime utilisateur.

| # | Route FastAPI (sous `/api/v1`) | Méthode | Route Go | Méthode Go | Statut | React queries.ts | React Hook | React Page | Écarts |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `/health` | GET | `/health` (hors `/api/v1`) | GET | ⚠️ | 🔶 queryKey défini | — | — | Préfixe `/api/v1` manquant — casse les healthchecks Docker |
| 2 | `/bootstrap` | GET | `/bootstrap` | GET | ✅ | `routes/__root.tsx` (inline) | `useQuery` (inline) | RootLayout | |
| 3 | `/players` | GET | `/players` | GET | ✅ | 🔶 queryKey défini | — | — | |
| 4 | `/session/context` | POST | `/session/context` | POST | ✅ | 🔶 types définis | — | — | |
| 5 | `/auth/device-flow/start` | POST | `/auth/device-flow/start` | POST | ✅ | `features/setup/queries.ts` | `useStartDeviceFlow()` | `routes/setup.tsx` | |
| 6 | `/auth/device-flow/{attempt_id}` | GET | `/auth/device-flow/{attempt_id}` | GET | ✅ | `features/setup/queries.ts` | `useDeviceFlowStatus()` (poll 5s) | `routes/setup.tsx` | |
| 7 | `/settings` | GET | `/settings` | GET | ✅ | `features/setup/queries.ts` | `useSettings()` | `routes/settings.tsx` | |
| 8 | `/settings` | PATCH | `/settings` | PATCH | ✅ | `features/setup/queries.ts` | `useUpdateSettings()` | `routes/settings.tsx` | |
| 9 | `/settings/media/reset-index` | POST | `/settings/media/reset-index` | POST | ✅ | 🔶 types définis | — | — | Go: stub (TODO Sprint 19) |
| 10 | `/setup/players` | POST | `/setup/players` | POST | ✅ | `features/setup/queries.ts` | `useCreatePlayer()` | `routes/setup.tsx` | |
| 11 | — | — | `/setup/smoke-test` | POST | 🟦 GO-ONLY | `features/setup/queries.ts` | `useStartSmokeTest()` | `routes/setup.tsx` | Absent en FastAPI **mais câblé dans React** |
| 12 | `/jobs/{job_id}` | GET | `/jobs/{job_id}` | GET | ✅ | `features/setup/queries.ts` | `useJobStatus()` (poll 3s) | `routes/setup.tsx` | |
| 13 | `/sync/initial` | POST | `/sync/initial` | POST | ✅ | `features/setup/queries.ts` | `useStartInitialSync()` | `routes/setup.tsx` | |
| 14 | `.../filters/resolve` | POST | `.../filters/resolve` | POST | ✅ | 🔶 queryKey dans `globalFilterStore` | — (hook non implémenté) | — | |
| 15 | `.../pages/match-history/query` | POST | `.../pages/match-history/query` | POST | ⚠️ | `features/match-history/queries.ts` | `useMatchHistory()` | `routes/.../stats/history.tsx` | Champ `columns` absent Go |
| 16 | `.../pages/match-history/export` | POST | — | — | ❌ ABSENT | `features/match-history/queries.ts` | `useMatchHistoryExport()` | `routes/.../stats/history.tsx` | Export CSV non implémenté |
| 17 | `.../pages/career` | GET | `.../pages/career` | GET | ✅ | `features/career/queries.ts` | `useCareerPage()` | `routes/.../career.tsx` | |
| 18 | `.../pages/career/top-matches` | GET | `.../pages/career/top-matches` | GET | ✅ | `features/career/queries.ts` | `useCareerTopMatches()` (lazy) | `routes/.../career.tsx` | |
| 19 | `.../pages/career/encounters` | GET | `.../pages/career/encounters` | GET | ✅ | `features/career/queries.ts` | `useCareerEncounters()` (lazy) | `routes/.../career.tsx` | |
| 20 | `.../pages/citations` | **POST** | `.../pages/citations` | **GET** | ❌ MÉTHODE | `features/citations/queries.ts` | `useCitationsPage()` | `routes/.../profile/citations.tsx` | POST+body filtres → GET sans body |
| 21 | — | — | `.../pages/commendations` | GET | 🟦 GO-ONLY | — | — | — | Python fusionne dans citations |
| 22 | `.../matches/{match_id}` | GET | `.../matches/{match_id}` | GET | ⚠️ | `features/match-view/queries.ts` | `useMatchView()` | `routes/.../matches/$matchId.tsx` | 13+ colonnes scoreboard manquantes |
| 23 | `.../pages/explorer/matches-query` | POST | — | — | ❌ ABSENT | `features/explorer/queries.ts` | `useExplorerMatches()` | `routes/.../explorer/index.tsx` | Explorer filtré non implémenté |
| 24 | `.../pages/explorer/player-query` | POST | `.../pages/explorer/player-query` | POST | ⚠️ | `features/explorer/queries.ts` | `useExplorerPlayer()` | `routes/.../explorer/index.tsx` | `target_gamertag` (FastAPI) vs `other_gamertag` (Go) + réponse simplifiée |
| 25 | `.../pages/last-match/resolve` | POST | — | — | ❌ ABSENT | `features/match-view/queries.ts` | `useLastMatchResolve()` | `routes/.../last-match.tsx` | |
| 26 | — | — | `.../pages/sessions` | GET | 🟦 GO-ONLY | — | — | — | |
| 27 | `.../pages/timeseries` | **POST** | `.../pages/stats/query` | **POST** | ❌ ROUTE+DTO | `features/timeseries/queries.ts` | `useTimeseriesPage()` | `routes/.../stats/timeseries.tsx` | Contrat radicalement différent |
| 28 | `.../pages/session-compare` | POST | — | — | ❌ ABSENT | `features/session-compare/queries.ts` | `useSessionComparePage()` | `routes/.../stats/sessions.tsx` | |
| 29 | `.../pages/home` | GET | `.../pages/home` | GET | ✅ | `features/home/queries.ts` + `KPIBar.tsx` | `useHomePage()` | `routes/.../home.tsx` + toutes les pages joueur | Bug `http.Error()` plain text |
| 30 | `.../battlepass` | GET | `.../battlepass` | GET | ✅ | `features/home/queries.ts` | `useBattlePass()` (stale 4h) | `routes/.../home.tsx` | |
| 31 | `.../challenges` | GET | `.../challenges` | GET | ✅ | `features/home/queries.ts` | `useChallenges()` (stale 1h) | `routes/.../home.tsx` | |
| 32 | `.../pages/teammates` | **POST** | `.../pages/squad` | **GET** | ❌ ROUTE+MÉTHODE+DTO | `features/squad/queries.ts` | `useTeammates()` | `routes/.../squad.tsx` | Route renommée + POST→GET + contrat diff |
| 33 | `.../pages/synthesis` | **POST** | `.../pages/synthesis` | **GET** | ❌ MÉTHODE+DTO | `features/synthesis/queries.ts` | `useSynthesisPage()` | `routes/.../synthesis.tsx` | POST→GET, body filtres perdu, 60% payload absent |
| 34 | `.../pages/media` | **POST** | `.../pages/media` | **GET** | ❌ MÉTHODE+CONTRAT | `features/media/queries.ts` | `useMediaPage()` | `routes/.../media.tsx` | POST body filtres/tri → GET ?page=N |
| 35 | `/directory/gamertags/search` | GET | `/directory/gamertags/search` | GET | ⚠️ | `features/explorer/queries.ts` | `useGamertagSearch()` | `routes/.../explorer/index.tsx` | Go ne set pas `query` dans la réponse |

#### Endpoints non câblés côté React (5)

| Endpoint OpenAPI | Statut React |
|---|---|
| `GET /health` | queryKey défini dans `keys.ts`, aucun hook consommateur |
| `GET /players` | queryKey défini dans `keys.ts`, aucun hook consommateur |
| `POST /session/context` | Types définis dans `generated.ts`, aucun hook |
| `POST /filters/resolve` | queryKey référencé dans `globalFilterStore`, hook non implémenté |
| `POST /settings/media/reset-index` | Types définis dans `generated.ts`, aucun hook |

#### Chiffres clés React

| Métrique | Valeur |
|---|---|
| Endpoints câblés (appel `api.get`/`api.post`/`api.patch`) | **24** |
| Hooks TanStack Query (`useQuery`) | 17 |
| Mutations TanStack (`useMutation`) | 7 |
| Query keys définis dans `keys.ts` | 22 |
| Endpoints OpenAPI dans `generated.ts` | 29 |
| Endpoints non câblés | 5 |
| Features folders | 13 |
| Route files | 15 |

#### Observation critique : `GET /setup/status`

Le code frontend conserve un hook `useSetupStatus()` dans `features/setup/queries.ts`, mais `SetupPage` ne l'utilise pas et l'orchestration runtime du setup dépend aujourd'hui de `GET /bootstrap` (`setup_state`) dans `__root.tsx` + `SetupPage`.

En revanche, `generated.ts`, les handlers MSW et plusieurs specs Playwright référencent encore `/setup/status`. **Cet endpoint n'existe ni dans FastAPI ni dans Go.** La bonne lecture n'est donc pas "blocage runtime confirmé", mais **artefact mort / drift documentaire et test** à purger explicitement avant de recalculer certains compteurs de parité.

---

### 1.3 Résumé comptable

| Métrique | Valeur |
|---|---|
| Endpoints FastAPI total | **28** |
| Endpoints Go total | **27** |
| Routes identiques (route + méthode + contrat conforme) | **16** |
| Routes avec méthode HTTP incompatible | **4** (citations, teammates/squad, synthesis, media) |
| Endpoints absents en Go | **5** (export, matches-query, last-match, session-compare, timeseries) |
| Endpoints Go-only | **4** (smoke-test, commendations, sessions, stats/query) |
| Réponses appauvries en Go | **5** (citations, synthesis, media, explorer/player-query, match scoreboard) |
| Réponses enrichies en Go | **1** (squad vs teammates) |
| Handlers avec `http.Error()` plain text | **3** (home, sessions, stats) |
| Handlers avec violation hexagonale | **15/21** |

> **Note de calcul** : la matrice §1.2 liste 35 lignes (toutes surfaces confondues : runtime câblé, hooks/types non consommés, artefacts test/codegen, endpoints Go-only). Les totaux FastAPI=28 et Go=27 comptent les endpoints réels de chaque backend (hors endpoints de l'autre). Le ratio handlers hexagonaux est calculé sur 21 fichiers handler Go (20 handlers + `helpers.go` utilitaire, hors `sync_handler.go` qui est correctement injecté).

**Résultat global : sur la surface d'API inventoriée autour du frontend, de ses artefacts et des endpoints Go-only, le tableau fait apparaître 16 routes conformes, 5 absentes, 6 incompatibles route/méthode/DTO et 4 Go-only.**

Lu strictement runtime, le cas `/setup/status` doit être déclassé : il illustre surtout une dette de purge des hooks/tests/codegen et non un call produit actuellement branché dans `SetupPage`.

---

### 1.4 Différences DTO champ par champ

#### 1.4.1 `pages/timeseries` (FastAPI) vs `pages/stats/query` (Go) — ❌ INCOMPATIBLE

**FastAPI `TimeseriesPageResponse`** :
```
total_matches: int
summary_tab:
  kpi_cards: [{key, label, value, delta, color}]
  win_rate_chart: PlotlyFigurePayload | null
  score_chart: PlotlyFigurePayload | null
  kda_dist_chart: PlotlyFigurePayload | null
cumul_tab:
  cumul_net_chart: PlotlyFigurePayload | null
  cumul_kd_chart: PlotlyFigurePayload | null
  rolling_kd_chart: PlotlyFigurePayload | null
form_tab:
  ewma_kd_chart: PlotlyFigurePayload | null
  regression_chart: PlotlyFigurePayload | null
  net_score_per_hour_chart: PlotlyFigurePayload | null
  regression_stats: {kd_slope, winrate_slope, r_squared, has_enough_for_trend, trend}
intensity_tab:
  intensity_heatmap: PlotlyFigurePayload | null
  score_per_minute_chart: PlotlyFigurePayload | null
distributions_tab:
  kda_distribution: PlotlyFigurePayload | null
  first_kill_dist: PlotlyFigurePayload | null
  correlations: [PlotlyFigurePayload]
```

→ Chaque chart = `PlotlyFigurePayload` (data+layout JSON Plotly, rendu server-side).

**Go `StatsPageResponse`** :
```json
{
  "win_loss": {"points": [WinLossPoint], "win_rate": float, "total_matches": int,
               "rolling_win_rate": [float], "cumulative_kd": [CumulativePoint],
               "cumulative_net": [CumulativePoint]},
  "accuracy": {"points": [AccuracyPoint], "mean": float, "has_data": bool,
               "score_per_min": [float]},
  "objective": {"points": [ObjectivePoint], "total_score": int, "avg_assists": float,
                "has_data": bool},
  "form":     {"points": [PerformancePoint], "mean": *float, "has_enough_data": bool},
  "lusr":     {"points": [LUSRPoint], "current_rating": *float, "has_data": bool},
  "bucket_info": {"type": string, "label": string},
  "total_matches": int
}
```

→ Points de données bruts — **AUCUN Plotly** — le frontend devrait construire les charts lui-même.

**Verdict** : ❌ Contrats radicalement différents. Le frontend attend des charts Plotly sérialisés, le Go renvoie des séries de points. Pas d'adaptation possible sans réécriture.

> **Recommandation architecture** : le pattern Go (data points bruts → charts construits côté frontend) est **objectivement meilleur** pour un SPA React. Il sépare les responsabilités (backend = données, frontend = présentation), réduit la taille des payloads, et élimine la dépendance Plotly côté serveur. **Exception légitime à la règle "Go s'aligne sur FastAPI"** : pour `timeseries` et à terme les autres endpoints avec Plotly, la bonne stratégie serait d'adapter le frontend React à consommer des data points (via recharts, visx ou nivo) et d'adopter le contrat Go plutôt que de sérialiser du Plotly JSON en Go. Cela implique cependant une modification du frontend — à planifier comme un workstream distinct post-bascule, ou en parallèle si les ressources le permettent.

---

#### 1.4.2 `pages/teammates` (FastAPI) vs `pages/squad` (Go) — ❌ INCOMPATIBLE

| Aspect | FastAPI | Go |
|---|---|---|
| Route | `pages/teammates` | `pages/squad` |
| Méthode | POST body `{selected_gamertags[], filters}` | GET query `?teammate=xuid` |
| Structure réponse | `{options, teammates[{with_kpis, without_kpis}], solo_reference}` | `{top_teammates, selected_teammate, solo_stats, squad_stats}` |
| KPIs comparés | `with_kpis` vs `without_kpis` par coéquipier | solo_stats vs squad_stats global |
| `kd_ratio, accuracy, kills_per_game, assists_per_game` | ✅ | ❌ remplacé par `avg_kda, avg_kills` |
| Squad score / radar / impact / records / timeseries | ❌ | ✅ (Go plus riche sur ce point) |
| Filtres cascade | ✅ via FilterContextInput body | ❌ |

**Verdict** : ❌ Incompatibilité route + méthode + payload + structure de réponse. Go plus riche sur certains axes, mais non substituable.

---

#### 1.4.3 `pages/citations` — ❌ MÉTHODE + CONTRAT

| Champ | FastAPI (POST + body filtres) | Go (GET, pas de body) |
|---|---|---|
| Fusion commendations+médailles | 1 endpoint | 2 endpoints séparés |
| `count_filtered` vs `count_total` | ✅ (filtres appliqués) | ❌ 1 seul `total` |
| `deltas` (variations filtres) | ✅ | ❌ ABSENT |
| `distribution_chart` (Plotly) | ✅ | ❌ ABSENT |
| `mastery_pct`, `tier_label` | ✅ | ❌ ABSENT |
| Filtres cascade | ✅ POST body | ❌ Aucun |

---

#### 1.4.4 `pages/synthesis` — ❌ MÉTHODE + CONTRAT

| Champ | FastAPI (POST + body `{period, filters}`) | Go (GET, pas de body) |
|---|---|---|
| `solo_kpis` + `squad_kpis` (8 champs chacun) | ✅ | ❌ ABSENT |
| `comparison_metrics[]` | ✅ | ❌ ABSENT |
| `period` filtrable | ✅ | ❌ ABSENT |
| Heatmap format | `{dow, hour, count}` (temporel) | `{row_key, col_key, value, count}` (map×mode) |
| `top_weeks.kd_ratio` | ✅ | ❌ `avg_kills` à la place |

→ ~60% du payload manquant côté Go.

---

#### 1.4.5 `pages/media` — ❌ MÉTHODE + CONTRAT

| Champ | FastAPI (POST + body riche) | Go (GET `?page=N`) |
|---|---|---|
| Naming item | `basename` | `file_name` |
| `section` (mine/teammates/unassigned) | ✅ | ❌ ABSENT |
| `owner_gamertag`, `map_name` | ✅ | ❌ ABSENT |
| `total_mine/teammates/unassigned` | ✅ (3 compteurs) | ❌ 1 seul `total_count` |
| Sort, kind_filter, group_by | ✅ POST body | ❌ ABSENT |
| Pagination | `{total, page, page_size, has_next, has_prev}` | `{total_count, page, page_size, has_more}` |

---

#### 1.4.6 `matches/{match_id}` — ⚠️ DTO INCOMPLET

**13+ colonnes scoreboard absentes en Go** :

| Champ FastAPI | Présent Go |
|---|:---:|
| `rank.icon_url` | ❌ |
| `combat_tab.weapon_kills[].effective_weapon_id` | ❌ |
| `combat_tab.highlight_events.event_time_ms` | ❌ (Go: `tick_count`) |
| `combat_tab.highlight_events.target_xuid` | ❌ |
| `combat_tab.highlight_events.weapon_id` | ❌ |
| `team_tab.scoreboard[].is_bot` | ❌ |
| `team_tab.scoreboard[].betrayals` | ❌ |
| `team_tab.scoreboard[].suicides` | ❌ |
| `team_tab.scoreboard[].shots_accuracy` | ❌ |
| `team_tab.scoreboard[].damage_efficiency` | ❌ |
| `team_tab.scoreboard[].average_life` | ❌ |
| `team_tab.scoreboard[].objectives_stolen` | ❌ |
| `team_tab.scoreboard[].headshot_kills` | ❌ |
| `team_tab.scoreboard[].max_killing_spree` | ❌ |
| `team_tab.scoreboard[].perfect_kills` | ❌ |
| `team_tab.scoreboard[].power_weapon_kills` | ❌ |
| `team_tab.scoreboard[].melee_kills` | ❌ |

---

#### 1.4.7 `pages/match-history/query` — ⚠️ DIFFS MINEURES

**Request** :

| Champ | FastAPI | Go |
|---|---|---|
| `columns` | ✅ `list[str] | None` | ❌ ABSENT |
| Sort | via `PaginationRequest` inherited | `sort_field + sort_dir` (flat) |

**Response** :

| Champ | FastAPI | Go |
|---|---|---|
| `outcome_code` | `int | None` | `int` (non-nullable) |
| `map_ui`, `mode_ui` | non-nullable | nullable |
| `pagination.freshness` | ❌ | ✅ (Go-only) |
| `export_hint` | ✅ | ❌ (export absent) |

---

#### 1.4.8 `pages/explorer/player-query` — ⚠️ DIFFS MAJEURS

**Request** :

| Champ | FastAPI | Go |
|---|---|---|
| Champ gamertag | `target_gamertag` | `other_gamertag` (**nom différent**) |
| `filters` | ✅ `FilterContextInput | None` | ❌ ABSENT |
| `limit` | ❌ | ✅ `int` (Go-only) |

**Response** :

| Champ | FastAPI | Go |
|---|---|---|
| `target: {gamertag, xuid}` | ✅ objet structuré | `other_gamertag` + `other_xuid` (flat) |
| `summary: {matches_together, wins_together, losses_together, last_seen_at}` | ✅ | ❌ ABSENT |
| `allies_table: [ExplorerEncounterRow]` | ✅ | ❌ ABSENT |
| `enemies_table: [ExplorerEncounterRow]` | ✅ | ❌ ABSENT |
| `common_matches` | 10 champs par match (labels formatés) | 5 champs (données brutes) |

---

#### 1.4.9 `pages/home` — ✅ CONFORME

Champs JSON identiques : `hero`, `highlights`, `recent_matches`, `recent_media`, `solo_session`, `squad_session`.

**Bug** : `home.go` utilise `http.Error()` (plain text) au lieu de `writeError()` (JSON structuré) sur 3 paths d'erreur.

---

#### 1.4.10 `directory/gamertags/search` — ⚠️ BUG MINEUR

Struct identique : `{query, items: [{gamertag, xuid, score, exact_match}]}`.

**Bug Go** : le handler ne set pas le champ `query` dans la réponse → toujours `""`.

---

#### 1.4.11 `filters/resolve` — ✅ CONFORME

`FilterContextResolved` a la même structure : `effective`, `available_options`, `session_options`, `counts`.

---

### 1.5 Sync Engine

**Verdict : ~85% porté — cœur fonctionnel, quelques modules avancés manquants.**

| Composant | Python | Go | Statut |
|-----------|--------|-----|--------|
| Delta/Full sync + SyncScope (~50 champs) | `engine.py` + 8 mixins + `scope.py` | `engine.go` + `scope.go` | ✅ |
| Backfill detection + CLI | `scripts/backfill/` | `backfill.go` + `backfill_cli.go` + `backfill_flags.go` | ✅ |
| API client Halo | `api_client.py` (wrapper SPNKr) | `halo_client.go` (HTTP natif) | ✅ **Modernisé** |
| Transformers, writes, career, LUSR, perf, aggregates, PvE | 15+ fichiers | 12 fichiers | ✅ |
| **Weapon kills film parsing (NS timeline)** | `_engine_weapon_kills.py` | — | ❌ **Manquant** |
| **Fanout enrichment multi-joueur** | `_engine_fanout.py` | — | ❌ **Manquant** |
| **Batch audit/columns** | `_batch_audit.py`, `_batch_columns.py` | — | ❌ |
| **Challenge migrations, asset langs** | 2 fichiers | — | ❌ |

---

### 1.6 Algorithmes d'analyse

**Verdict : ~40% porté — algorithmes cœur OK, analyses UI absentes (attendu… en partie).**

**Portés** : citations, performance_score, skill_rating (TrueSkill 2), killer_victim, sessions, spawn_detection, squad, weapon_*, home.

**Non portés** (15+ modules) : cumulative, comeback, friends_impact, match_cadence, match_intensity, objective_participation, participation_radar, win_streaks, maps, stats, player_index, first_events, medal_verdicts, mode_categories, playlist_groups, global_correlation.

→ C'est attendu pour les algos purement Streamlit. Cependant, **certains sont maintenant requis par le contrat FastAPI** (ex: `TimeseriesPageResponse` requiert cumul, form trends, intensity, distributions). Le Go ne les a pas car il n'a pas été porté contre ce contrat.

---

### 1.7 Authentification et onboarding

**Verdict : chaîne d'échange complète, mais onboarding cassé de bout en bout.**

#### Ce qui fonctionne

- Device Code Flow MSAL → access_token : ✅
- Chaîne d'échange Halo (XBL → XSTS → Spartan → Clearance) : ✅ implémentation native
- AttemptStore thread-safe multi-session : ✅ (nouveau, adapté au web)
- AcquireTokenSilent et sérialisation mémoire : ✅ (mais cache non persisté)

#### Problèmes d'onboarding

| Problème | Détail |
|----------|--------|
| **Gamertag/XUID jamais récupérés** | `pollDeviceFlow` marque l'attempt `authorized` après l'échange Halo, mais ne renseigne jamais `Gamertag` ni `XUID`. L'état `provisioned` existe mais aucun code n'y transite. |
| **Identité Halo non propagée en session** | `GetDeviceFlowStatus` copie `LinkedHaloIdentity` seulement si `snapshot.Gamertag != ""` — toujours vide. |
| **Bootstrap figé en `"missing"`** | `bootstrap_service.go` L54 : `AuthState: "missing"` en dur — commenté `Sprint 0`. |
| **SetupState ne vérifie pas l'auth** | `resolveSetupState()` ne dépend que de la présence de joueurs, pas de l'état d'authentification. |
| **CreatePlayer bloque sans identité** | `SetupHandler.CreatePlayer` force `profile_mode = xbox` et refuse sans identité Halo liée. |
| **Pas de persistance MSAL cache** | Chaque restart serveur force un nouveau Device Code Flow complet. |

**Conséquence** : le parcours setup Go fait semblant de fonctionner mais ne peut pas aboutir de bout en bout. Ce n'est pas un simple TODO — c'est un défaut de cohérence applicative dans le parcours d'entrée.

---

### 1.8 Migrations, Notifications, Ops

| Domaine | Verdict |
|---------|---------|
| Migrations DuckDB | **~100%** — 36 steps Go vs 34 Python. Framework idempotent conforme. |
| Notifications Discord | **~100%** — Embeds riches, anti-spam, i18n. |
| Ops CLI | **~90%** — 14 sous-commandes dans `cmd/levelup/`. |

---

### 1.9 Runtime et infrastructure

**Constat critique : toute l'infra est câblée sur Python. Le Go n'est branché nulle part.**

| Composant | Pointe vers |
|-----------|-------------|
| `Dockerfile` | Python 3.12 + FastAPI (`uvicorn apps.api.app.main:app`) |
| `docker-compose.yml` | FastAPI sur port 8000, healthcheck Python |
| `Makefile` (racine) | Cibles `api`, `dev`, `test-api` → uvicorn FastAPI |
| `package.json` `generate-types` | `http://127.0.0.1:8000/api/openapi.json` → schéma FastAPI |
| `apps/web/src/lib/api/types.ts` | Types manuels alignés sur schémas Pydantic |
| `apps/web/src/lib/api/client.ts` | `BASE_URL = /api/v1` (servi par FastAPI) |

Le Go-API possède son propre `apps/go-api/Makefile` mais **aucun fichier d'infra racine ne le référence**.

→ `docker compose up` exécute Python. Le Go est un sous-projet isolé.

Nuance importante : le runtime React importe aujourd'hui `apps/web/src/lib/api/types.ts`. `generated.ts` existe toujours, mais n'est pas importé par le code applicatif. Le problème n'est donc pas un basculement partiel réussi, mais une coexistence non maîtrisée entre contrats manuels, générés et fixtures de test.

#### 1.9.1 Multi-titres / modèle de stockage

**Constat : le système est mono-titre de bout en bout.**

Indices concrets :

- les descriptions produit et OpenAPI parlent explicitement de **Halo Infinite** ;
- l'échange Halo cible un endpoint `titles/hi/...` ;
- les URLs publiques de match history pointent vers `halowaypoint.com/halo-infinite/...` ;
- les chemins de stockage sont globaux et non namespacés par titre : `metadata.duckdb`, `shared_matches_v2.duckdb`, `shared_pve.duckdb`, `players/{gamertag}/stats.duckdb`.

**Conclusion architecture** : si LevelUp doit rester **Halo Infinite uniquement**, rien n'impose de refondre tout de suite le schéma de stockage. En revanche, si le produit doit supporter plusieurs titres Halo, **la bonne réponse n'est pas de mélanger plusieurs jeux dans les mêmes DB par défaut**.

La recommandation la plus sûre est un **namespace par titre** :

- `data/titles/{title_slug}/warehouse/metadata.duckdb`
- `data/titles/{title_slug}/warehouse/shared_matches_v2.duckdb`
- `data/titles/{title_slug}/warehouse/shared_pve.duckdb` si pertinent
- `data/titles/{title_slug}/players/{gamertag}/stats.duckdb`

Ce choix évite de devoir rendre `title_slug` premier citoyen dans **toutes** les PK, vues SQL, migrations, caches, URLs Waypoint, tables metadata, résolutions de labels, profils joueurs et jobs de sync dès maintenant.

**Alternative non recommandée à court terme** : une seule famille de DB multi-jeux avec `title_slug` ou `title_id` partout. C'est faisable, mais cela oblige à requalifier massivement :

- les PK et index des tables partagées ;
- les vues `v_*` et leurs jointures metadata ;
- les migrations `schema_migrations` ;
- `db_profiles.json`, la session courante, le bootstrap et la sélection du joueur ;
- les liens publics, l'auth exchange et les réglages dépendants du titre ;
- les datasets de démo, fixtures, golden values et tests E2E.

**Verdict** : ce sujet n'est **pas** un P0 pour une bascule Go sur Halo Infinite, mais il doit être documenté comme **décision d'architecture explicite**. Sans cela, le premier ajout d'un second titre cassera le modèle de stockage implicite actuel.

#### 1.9.2 GitHub Actions, CI et releases

**Constat : l'automatisation ne doit pas être regardée uniquement à travers `ci.yml`.**

Aujourd'hui :

- `ci.yml` lance bien des jobs Go, mais les tests Go sont en grande partie hors chemin DuckDB réel (`CGO_ENABLED=0` pour `go test`) et la matrice ne couvre pas macOS ;
- `release.yml` construit encore des artefacts **Python** (`python embeddable` Windows + archive Unix source) via `packaging/build_release.py` ;
- `bump-version.yml` ne versionne que `pyproject.toml` ;
- `deploy.yml` et `test-deploy-precheck.yml` valident et déploient une image Python/FastAPI ;
- `e2e-browser-optional.yml` vise encore les E2E navigateur Streamlit, pas le couple React + backend cible.

**Implication** : si Go devient le backend de référence, il faut revoir **toute la surface GitHub Actions**, pas seulement ajouter 2 jobs dans `ci.yml`.

**Concernant les releases multi-plateformes Go** : oui, Go permet de produire des builds par plateforme, mais **pas de manière magique** dans ce projet.

Le point bloquant n'est pas Go lui-même, c'est l'usage de `duckdb-go` avec bindings natifs et contraintes CGo. Le `go.mod` montre des bindings embarqués pour :

- `linux-amd64`
- `linux-arm64`
- `darwin-amd64`
- `darwin-arm64`
- `windows-amd64`

En pratique, cela implique une **build matrix native par OS cible** pour la partie réellement supportée, et pas une promesse de cross-compilation triviale depuis une seule machine.

**Recommandation de distribution** :

1. **Canal serveur / prod** : publier d'abord une ou plusieurs images OCI/Docker comme artefacts de référence.
2. **Canal self-host / local** : publier des archives par OS contenant le binaire Go + `apps/web/dist` + fichiers de config templates.
3. **Canal desktop utilisateur final** : à décider explicitement ; si conservé, il faut empaqueter backend + frontend + wrappers OS, pas simplement remplacer le zip Python par un binaire nu.

**Cibles minimales réalistes** pour une chaîne Go mature :

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64
- Windows amd64

**Points non couverts par l'existant** :

- signature Windows / SmartScreen ;
- notarization / signature macOS ;
- source de version unifiée pour Go + web + image + release notes ;
- healthchecks et prechecks sur le vrai runtime Go ;
- validation du chemin CGo/DuckDB en CI, pas seulement un build partiel.

---

### 1.10 Compatibilité sessions/cookies

**Point non couvert dans les sections précédentes** : FastAPI utilise des cookies httpOnly pour la gestion de session. Le Go utilise son propre store de sessions (`gorilla/sessions` ou équivalent via `middleware.WithSession`).

**Risques à valider avant bascule** :

- **Format de cookie** : si le nom, le domaine, le path ou la sérialisation diffèrent entre les deux backends, la bascule invalidera toutes les sessions utilisateurs existantes (re-login forcé).
- **Session store** : FastAPI utilise potentiellement un store en mémoire ou un backend externe ; Go utilise un `CookieStore` ou `FilesystemStore`. Si les deux ne partagent pas le même format de sérialisation, un rollback Go→FastAPI pendant la transition casserait les sessions.
- **SameSite / Secure flags** : vérifier que le middleware Go configure ces flags de manière identique à FastAPI pour éviter les rejets de cookies par le navigateur.

**Recommandation** : ajouter un test de compatibilité cookies avant P0-5. Concrètement : démarrer les deux backends, obtenir un cookie de chacun, et vérifier que le format est interchangeable ou documenter que la bascule impose un re-login one-shot.

---

### 1.11 Synthèse des écarts critiques

| # | Écart | Sévérité | Impact |
|---|-------|----------|--------|
| **C1** | **5 endpoints absents** : export, matches-query, last-match, session-compare, timeseries | BLOQUANT | Pages frontend non fonctionnelles |
| **C2** | **6 endpoints incompatibles** : citations, teammates, synthesis, media, stats, explorer/player-query | BLOQUANT | Routes/méthodes/DTO divergents |
| **C3** | **Onboarding cassé** : Gamertag jamais récupéré, AuthState toujours `"missing"` | BLOQUANT | Setup impossible bout en bout |
| **C4** | **Infra, CI/CD et releases encore Python-first** : Docker, compose, Makefile, `release.yml`, `deploy.yml`, `test-deploy-precheck.yml`, versioning et generate-types | BLOQUANT | Le Go n'est ni déployable ni publiable proprement comme runtime de référence |
| **C5** | **Pas de Plotly** : timeseries/career attendent `PlotlyFigurePayload` | HAUTE | Charts non rendables côté front — **mais le pattern Go (data points) est meilleur à long terme** ; envisager d'adapter le frontend plutôt que de sérialiser du Plotly en Go |
| **C6** | **Pas de CSRF** : FastAPI vérifie `Origin`/`Referer`, Go n'a pas de middleware CSRF | HAUTE | Sécurité dégradée |
| **C7** | **Pas de persistance MSAL cache** | HAUTE | Re-auth à chaque restart |
| **C8** | **13+ colonnes scoreboard manquantes** dans match_view | MOYENNE | Données incomplètes |
| **C9** | **Weapon kills film parsing absent** | MOYENNE | Extraction armes impossible |
| **C10** | **Fanout enrichment absent** | MOYENNE | Multi-joueur limité en post-sync |
| **C11** | **Sources de vérité multiples** : schémas Pydantic, OpenAPI Go, `types.ts` manuels, `generated.ts`, MSW/E2E | HAUTE | Le drift n'est plus théorique ; `GET /setup/status` illustre surtout un artefact mort et des surfaces désalignées |
| **C12** | **Documentation et fixtures contradictoires** : docs de migration, codegen, tests, hooks morts | HAUTE | Risque de faux positifs / faux négatifs dans la priorisation ; on peut réparer le mauvais problème en premier |
| **C13** | **Architecture mono-titre implicite** : auth, URLs publiques, metadata et chemins DuckDB sont hardcodés Halo Infinite | HAUTE si multi-titres devient un objectif | Le premier second jeu imposera un namespace de stockage/config/API avant toute extension sérieuse |
| **C14** | **Compatibilité sessions/cookies non vérifiée** | MOYENNE | La bascule peut invalider les sessions existantes si le format de cookie ou le store diffère |

#### Matrice de risque : « que se passe-t-il si on bascule maintenant ? »

Cette matrice quantifie l'impact utilisateur concret d'une bascule Go immédiate sans résoudre les P0 :

| Page frontend | Route(s) impactée(s) | Comportement actuel Go | Conséquence UX |
|---|---|---|---|
| **Setup** | `/bootstrap`, `/auth/device-flow/*`, `/setup/players` | AuthState toujours `"missing"`, Gamertag jamais récupéré | **Page blanche** — impossible de terminer l'onboarding |
| **Timeseries** | `/pages/timeseries` | Route absente (Go a `/pages/stats/query` différent) | **404** → page blanche |
| **Session Compare** | `/pages/session-compare` | Absent | **404** → page blanche |
| **Last Match** | `/pages/last-match/resolve` | Absent | **404** → page blanche |
| **Explorer (matches)** | `/pages/explorer/matches-query` | Absent | **Fonctionnalité inopérante** — recherche par matchs KO |
| **Match History (export)** | `/pages/match-history/export` | Absent | **Bouton Export mort** |
| **Citations** | `/pages/citations` | GET au lieu de POST → 405 Method Not Allowed | **Erreur 405** → page inutilisable |
| **Teammates/Squad** | `/pages/teammates` | Route renommée `squad` + GET vs POST → 404/405 | **Erreur** → page inutilisable |
| **Synthesis** | `/pages/synthesis` | GET sans body vs POST attendu + 60% payload absent | **Page partiellement vide** avec erreurs |
| **Media** | `/pages/media` | GET vs POST attendu + section/tri absents | **Page dégradée** — médias sans catégories ni tri |
| **Match View** | `/matches/{id}` | Fonctionne mais 13+ colonnes scoreboard manquantes | **Données incomplètes** — onglets team appauvris |
| **Home** | `/pages/home` | Fonctionne, bug `http.Error()` sur erreur | **OK nominal**, erreur non structurée si player manquant |
| **Career** | `/pages/career/*` | Conforme | **OK** |
| **Settings** | `/settings` | Conforme | **OK** |

**Bilan** : sur les 15 pages principales du frontend, **6 seraient totalement cassées** (page blanche ou 404/405), **3 seraient dégradées** (données partielles ou fonctionnalités manquantes), et **6 fonctionneraient correctement**. La bascule sans P0 est donc exclue.

---

## Partie 2 — Audit qualité code Go

### 2.1 Architecture hexagonale — design vs réalité

Le design cible est correct :

```
cmd/             → Entry points
internal/
  domain/        → Types purs, 0 dépendance
  port/          → Interfaces abstraites (8 repos)
  service/       → Logique métier / orchestration (13 services)
  analysis/      → Algorithmes stateless (0 DB, 0 IO)
  platform/      → Adaptateurs (DuckDB, MSAL, Halo, sessions, settings, jobs)
  api/           → HTTP handlers + middleware (chi)
  sync/          → Moteur de synchronisation autonome
  config/        → Configuration + feature flags
  migration/     → Migrations DuckDB
```

**Mais la réalité diverge dans les handlers.** Le pattern dupliqué :

```go
func (h *XHandler) HandleX(w http.ResponseWriter, r *http.Request) {
    pdb, err := h.resolvePlayer(r)           // config.ResolvePlayer
    repo := duckdb.NewXRepo(pdb)             // import platform
    svc := service.NewXService(repo)          // assemblage inline
    result, err := svc.DoX(r.Context(), ...)
    writeJSON(w, http.StatusOK, result)
}
```

→ Ce n'est pas de l'hexagonal mais du **service locator pattern** ad hoc dans chaque handler. Les ports ne sont pas utilisés pour l'injection.

**Exception positive** : `BootstrapHandler`, `PlayersHandler` et `AuthHandler` reçoivent leurs dépendances par injection dans `NewRouter()` — pattern correct.

---

### 2.2 Violations handlers — matrice détaillée

| Handler | Importe `platform/duckdb` | Appelle `config.ResolvePlayer` | Construit repos inline | `http.Error()` plain text |
|---|:---:|:---:|:---:|:---:|
| `auth.go` | ❌ | ❌ | ❌ | ❌ |
| `bootstrap.go` | ❌ | ❌ | ❌ | ❌ |
| `career.go` | ✅ | ✅ | ✅ | ❌ |
| `citations.go` | ✅ | ✅ | ✅ | ❌ |
| `explorer.go` | ✅ | ✅ | ✅ | ❌ |
| `filters.go` | ✅ | ✅ | ✅ | ❌ |
| `gamertag.go` | ✅ (`OpenReadOnly` + `NewGamertagRepo`) | ❌ (`config.SharedDBPath`) | ✅ | ❌ |
| `health.go` | ❌ | ❌ | ❌ | ❌ |
| **`home.go`** | ✅ | ✅ | ✅ | **✅ 3×** |
| `jobs.go` | ❌ | ❌ | ❌ | ❌ |
| `match_history.go` | ✅ | ✅ | ✅ | ❌ |
| `match_view.go` | ✅ | ✅ | ✅ | ❌ |
| `media.go` | ✅ | ✅ | ✅ | ❌ |
| **`sessions.go`** | ✅ | ✅ | ✅ | **✅ 2×** |
| `settings.go` | ❌ | ❌ | ❌ | ❌ |
| `setup.go` | ❌ | ❌ | ❌ | ❌ |
| `squad.go` | ✅ | ✅ | ✅ | ❌ |
| **`stats.go`** | ✅ | ✅ | ✅ | **✅ 2×** |
| `sync_handler.go` | ❌ | ❌ | ❌ | ❌ |
| `session_context.go` | ❌ | ❌ | ❌ | ❌ |

**Bilan** : 15/21 violent l'hexagone (20 handlers + `helpers.go`), 3 utilisent `http.Error()` plain text. 6 handlers correctement injectés (`auth`, `bootstrap`, `health`, `jobs`, `settings`, `setup`).

---

### 2.3 Bugs et risques sécurité (P0)

| # | Fichier | Problème | Sévérité |
|---|---------|----------|----------|
| **P0-1** | `platform/duckdb/pool.go` L55 | **Fuite de connexion** : doublon PlayerDB → `pdb.Player.Close()` appelé mais `pdb.Shared` et `pdb.Metadata` **jamais fermés**. Même pattern dans `CloseAll()`. | CRITIQUE |
| **P0-2** | `sync/backfill.go` L175+ | **Concaténation SQL** : `playerDoneGuard()` construit `NOT IN ('id1','id2'...)` par concaténation sans échappement. Risque concret quasi-nul (les IDs proviennent de DuckDB, pas de l'utilisateur) mais pattern fautif à corriger par principe. | MOYENNE |
| **P0-3** | `service/match_view_service.go` L47-54 | **7 erreurs silencieusement ignorées** : `GetPlayerMatchStats`, `GetMatchEnrichment`, `GetMatchScoreboard`, `GetMatchMedals`, `GetMatchEvents`, `GetMatchWeaponKills`, `GetMatchKVPairs` — tous évalués avec `_, _`. DB corrompue → onglets vides sans diagnostic. | HAUTE |
| **P0-4** | Middleware CSRF | **Absent.** FastAPI vérifie `Origin`/`Referer` sur toutes les mutations. Go n'a aucun middleware équivalent. | HAUTE |

---

### 2.4 Gestion d'erreur HTTP incohérente

Le contrat FastAPI renvoie des erreurs JSON : `{ code, message, retryable, details }`. Le Go possède `writeError()` qui respect ce contrat, mais 3 handlers utilisent `http.Error()` (texte brut) :

| Handler | Situation | Code envoyé |
|---------|-----------|-------------|
| `home.go` | Player non trouvé / erreur interne | `http.Error(w, err.Error(), 404/500)` |
| `stats.go` | Player non trouvé / erreur service | `http.Error(w, err.Error(), 400/500)` |
| `sessions.go` | Player non trouvé / erreur service | `http.Error(w, err.Error(), 400/500)` |

Le frontend React parse `ApiError { code, message, retryable }` → `http.Error` texte brut → perte du message structuré.

**Autres problèmes transport** :
- `StatsHandler` accepte un JSON malformé et retombe silencieusement sur des defaults `win_loss / period`
- Plusieurs réponses JSON ignorent l'erreur d'`Encode` via `nolint:errcheck` ou affectation muette

---

### 2.5 Dépassements de taille (P1)

| # | Fichier | Lignes | Dépassement | Action recommandée |
|---|---------|:------:|:-----------:|-------------------|
| **P1-1** | `analysis/squad.go` | **812** | +62% | Split en 6 fichiers + unifier 4 fonctions dupliquées via generics |
| **P1-2** | `sync/skill_rating.go` | **731** | +46% | Extraire SQL dans `skill_rating_db.go` |
| **P1-3** | `platform/duckdb/queries.go` | **714** | +43% | Split par domaine fonctionnel |
| **P1-4** | `sync/transforms.go` | **570** | +14% | Extraire helpers dans `transforms_helpers.go` |
| **P1-5** | `cmd/levelup/main.go` | **532** | +6% | Extraire sous-commandes en fichiers séparés |

**Duplication critique dans `squad.go`** :

| Paire dupliquée | Overlap |
|----------------|---------|
| `ComputeParticipationProfile(SquadMatchRow)` / `ComputeTeammateProfile(TeammateMatchRow)` | ~90% identique |
| `ComputeSquadRecords(SquadMatchRow)` / `ComputeTeammateRecords(TeammateMatchRow)` | ~95% identique |

---

### 2.6 Violations DRY / architecture (P2)

| # | Problème | Fichier(s) | Action |
|---|----------|-----------|--------|
| **P2-1** | 15 handlers dupliquent `resolvePlayer → NewRepo → NewService` | `handlers/*.go` | Factory abstraite ou injection dans `NewRouter` |
| **P2-2** | SQL queries inline dans un fichier d'algorithme | `sync/skill_rating.go` L460-582 | Extraire dans un repository |
| **P2-3** | Double switch identique sur toutes les surfaces | `config/feature_flags.go` | Refactorer en map lookup |
| **P2-4** | Logique métier dans le handler | `handlers/setup.go` L168-250 | Extraire dans `ProfileService` |
| **P2-5** | Double cache DB | `duckdb/db.go` `openDBs` + `duckdb/pool.go` `globalPool` | Unifier |
| **P2-6** | SQL quasi-identiques | `queries.go` Q4/Q4MV/Q5 | Factoriser |
| **P2-7** | Magic number `case 2: winScore = 1.0` | `sync/skill_rating.go` L130-135 | Constantes nommées |
| **P2-8** | ~180L de noop impls (50% du fichier) | `port/repository.go` | Déplacer dans `port_check_test.go` |
| **P2-9** | Multiple sources de vérité contrat API | OpenAPI Go, schémas Pydantic, types.ts, generated.ts | Figer **une** seule source |

---

### 2.7 Stubs et workarounds encore présents

| Fichier | Stub | Impact visible |
|---------|------|---------------|
| `bootstrap_service.go` L54 | `AuthState: "missing"` en dur | Frontend affiche toujours l'auth comme absente |
| `bootstrap_service.go` L115 | `DiscordConfigured: false` en dur | Indicateur toujours faux |
| `bootstrap_service.go` L116 | `TailscaleEnabled: false` en dur | Indicateur toujours faux |
| `handlers/settings.go` L105 | Reset index médias = goroutine stub | Pas d'action réelle |
| `platform/halo/provider.go` L116, L124 | `TODO Sprint 15` × 2 | Battle Pass et Challenges = best-effort |
| `auth.go` `pollDeviceFlow` | Ne récupère jamais Gamertag/XUID | Onboarding cassé |

---

### 2.8 Points forts

**Contexte** : 17 500 LOC Go fonctionnel constitue un portage substantiel. L'architecture `analysis/service/handler` est fondamentalement saine, même si l'injection est incomplète dans les handlers. Le moteur de sync, composant le plus complexe du projet (~85% porté), démontre une maîtrise du domaine métier, pas seulement de la technique Go.

| Aspect | Évaluation |
|--------|-----------|
| **Structure packages** | 12 packages internes bien découpés et nommés — l'organisation hexagonale est correcte au niveau macro. |
| **`analysis/` strictement pur** | 0 import IO — algorithmes testables isolément. C'est le package le mieux conçu du projet. |
| **Middlewares** | CORS, rate-limit, request-id, session cookie, structured logging — bien composés et idiomatiques Go. |
| **Feature flags** | Bascule granulaire Go↔Python par surface — excellent pour migration progressive et réversible. |
| **Migrations idempotentes** | Framework robuste : registre, `schema_migrations`, compile-time checks, tests. Plus propre que la version Python. |
| **OpenAPI + codegen** | Types générés depuis `openapi.yaml` — contrat versionné dans le repo Go. |
| **Toolchain qualité** | `.golangci.yml` : gocyclo 12, funlen 80L, lll 100, bodyclose, noctx, errcheck, goconst — discipline de qualité supérieure au projet Python historique. |
| **Tests** | Golden values JSON, tests analysis, migration, services, backfill_flags — couverture ciblée et pertinente sur les algos. |
| **Seulement 4 TODOs** dans 17.5K LOC | Codebase remarquablement propre pour son volume — peu de dette technique intentionnelle. |
| **Sync engine Go** | Porté fidèlement avec modernisations légitimes (HTTP natif, sync.Mutex). Le composant le plus complexe, et le mieux porté. |
| **Halo client natif** | Remplacement de SPNKr par un client HTTP natif — élimine une dépendance Python et simplifie le déploiement. |
| **Modèle de concurrence** | `sync.Mutex`, goroutines pour les polls device flow, rate limiter intégré — idiome Go bien exploité. |

---

## Décision stratégique

### Option retenue : **Option A — Réaligner Go sur FastAPI**

**Décision prise le 2025-07-17.**

Le backend Go doit s'aligner sur le contrat FastAPI existant (routes, méthodes HTTP, DTOs), et non l'inverse. Cela signifie :

1. **Le contrat de référence est celui de FastAPI** (`apps/api/`) tel que consommé par le frontend React (`apps/web/`).
2. **Le Go s'adapte** : les 6 endpoints incompatibles doivent adopter les routes, méthodes et structures DTO de FastAPI.
3. **Les 5 endpoints absents** doivent être implémentés en Go avec le même contrat que FastAPI.
4. **Les 4 endpoints Go-only** restent en l'état tant que le frontend ne les consomme pas (sauf `smoke-test` déjà câblé en React — à conserver).
5. **Aucune modification du frontend React** n'est nécessaire pour la bascule — c'est le critère de succès : substituer Go à FastAPI sans toucher `apps/web/`.

### Justification

- Le frontend React est le produit en production ; casser son contrat impliquerait une double migration (backend + frontend) = risque élevé.
- FastAPI a un contrat éprouvé et testé par l'usage réel. Le Go a des choix de contrat divergents non validés en prod.
- L'Option B (réaligner FastAPI sur Go) n'a pas de sens : personne ne consomme le contrat Go aujourd'hui.

### Ce que cette décision implique pour le plan d'action

- **P0-1** (purger les artefacts morts et les contradictions de surface) devient le prérequis du reste : `setup/status`, hooks/query keys non consommés, `generated.ts`, MSW, Playwright et docs de migration doivent être soit réalignés, soit supprimés.
- Les garde-fous automatiques minimum doivent être branchés **avant** le portage lourd des endpoints : contract tests OpenAPI/chi, golden tests CI, Playwright React en CI, lint OpenAPI bloquant.
- **P0-5** (aligner les 6 endpoints) et **P0-6** (implémenter les 5 absents) restent les travaux de fond prioritaires après assainissement de la surface.
- Le contrat Go-only (`commendations`, `sessions`, `stats/query`) peut être conservé comme endpoints bonus mais ne doit **pas** remplacer les endpoints FastAPI correspondants.
- L'`openapi.yaml` Go devra être mis à jour pour refléter le contrat FastAPI, pas l'inverse.

### Exception Plotly

Pour les endpoints `timeseries` et à terme les autres surfaces avec `PlotlyFigurePayload`, le pattern Go (data points bruts) est architecturalement supérieur. La recommandation est de **ne pas** sérialiser du Plotly JSON en Go, mais de planifier à moyen terme l'adaptation du frontend React vers une lib de charts JS (recharts, visx, nivo). Pour la bascule initiale, une couche de compatibilité mince côté Go (ou un endpoint temporaire) peut servir de pont.

### Critère de bascule mesurable

La bascule Go → production est validée quand **toutes** ces conditions sont remplies :

1. **Parité contract** : 0 diff dans `parity_check.py` sur les 24 endpoints câblés par React (tolérance float, champs ignorés configurés)
2. **E2E vert** : les 15 specs Playwright React passent sur le backend Go en CI
3. **Onboarding complet** : un test E2E peut dérouler le flow setup (auth → create player → initial sync → home page) de bout en bout
4. **Sécurité** : middleware CSRF actif, `http.Error()` éliminé, pool.go leak corrigé
5. **Infra** : `docker compose up` démarre le Go, healthcheck passe, `make dev` fonctionne
6. **Cookie compat** : compatibilité session documentée (ou re-login one-shot accepté explicitement)

### Coût de la double stack pendant la transition

Tant que Go n'est pas prêt, chaque évolution produit du frontend React doit être implémentée dans les deux backends (FastAPI en production + Go en préparation). Ce coût de maintenance double est un argument fort pour :

- **Option 1 — Gel features FastAPI** : ne plus développer de nouvelles routes FastAPI, traiter les bugs uniquement. Toute évolution va directement en Go. Risque : retarde les features produit.
- **Option 2 — Timeline serrée** : concentrer les ressources pour terminer P0 rapidement et basculer. Le gel implicite est tolérable si la fenêtre est < 4-6 semaines.
- **Option 3 — Implémentation double** : coûteux mais sans risque de gel. Acceptable uniquement si le volume de nouvelles features est faible.

**Recommandation** : Option 2, avec la contrainte que P0-1 à P0-4 soient traités en premier (1-2 semaines), puis P0-5/P0-6 en parallèle (3-4 semaines)

---

## Partie 3 — Plan d'action consolidé

### Estimation d'effort total

| Priorité | Effort estimé | Commentaire |
|----------|:---:|---|
| **P0** (bloquants bascule) | **18-30j** | Chemin critique — incompressible |
| **P1** (bugs / sécurité) | **3-5j** | Peut être traité en parallèle de P0-5/P0-6 |
| **P2** (architecture / qualité) | **8-12j** | Post-bascule acceptable |
| **P3** (évolution fonctionnelle) | **10-15j** | À la demande du frontend |
| **P4** (améliorations UX / produit) | **5-8j** | Post-bascule, profite de la refonte React |
| **Total P0+P1** | **~21-35j** | Effort minimum pour une bascule Go sécurisée |
| **Total P0+P1+P2** | **~29-47j** | Effort pour un backend Go propre et maintenable |
| **Total complet P0-P4** | **~52-82j** | Vision complète incluant dette technique et UX |

### Ordre d'exécution recommandé (P0)

Les 8 actions P0 ont des dépendances entre elles. L'ordre ci-dessous minimise les blocages :

```
Phase 1 — Assainissement (semaine 1-2)
├── P0-1 Purger artefacts morts ──────────────────┐
├── P0-2 Fixer source de vérité contrat ──────────┤
├── P0-3 Brancher garde-fous CI ──────────────────┤ (parallélisable)
└── P0-7 Décider stratégie distribution ──────────┘

Phase 2 — Réparation (semaine 2-4)
├── P0-4 Corriger onboarding Go ──────────────────┐
├── P0-5+P0-6 Contrat API par page ──────────────┤ (après P0-1/P0-2)
│   ├── Lot 1 : Home, Career, Settings ──────────┤ (déjà conformes → valider)
│   ├── Lot 2 : Citations, Media, Synthesis ──────┤ (POST + DTO → 3-4j)
│   ├── Lot 3 : Explorer, Match History ──────────┤ (ajouts + fixes → 2-3j)
│   ├── Lot 4 : Teammates/Squad, Timeseries ──────┤ (réécriture contrat → 3-5j)
│   └── Lot 5 : Last-match, Session-compare ──────┤ (endpoints abs. → 2-3j)
└── P1-* Bugs/sécurité ──────────────────────────┘ (en parallèle de Lot 2+)

Phase 3 — Infrastructure (semaine 4-6)
└── P0-8 Rebaser infra/CI/releases ──────────────── (après P0-7)

Validation — Critère de bascule
└── parity_check.py = 0 diff + Playwright vert + onboarding E2E OK
```

> **Note** : P0-5 et P0-6 sont fusionnés en un seul workstream "Contrat API" organisé par page frontend (des plus visitées aux moins critiques) plutôt que par catégorie (absent vs incompatible). Cela permet de livrer des pages complètes progressivement.

### P0 — Prérequis bascule (BLOQUANT)

| # | Action | Effort estimé |
|---|--------|:---:|
| **P0-1** | **Purger les artefacts morts et contradictions de surface** : décider explicitement du sort de `/setup/status`, supprimer ou rétablir hooks/query keys morts, réaligner `generated.ts`, handlers MSW, specs Playwright et docs de migration. | 1-2j |
| **P0-2** | **Fixer une source de vérité unique** pour le contrat API. Recommandation : figer l'OpenAPI FastAPI comme référence, réaligner Go dessus, puis converger `types.ts`/codegen sur cette décision. | 1-2j |
| **P0-3** | **Brancher d'abord les garde-fous automatiques minimum** : contract tests OpenAPI/chi, golden diff CI, Playwright React en CI, retrait du `continue-on-error` sur le lint OpenAPI. | 2-4j |
| **P0-4** | **Corriger l'onboarding Go** : récupérer Gamertag/XUID après échange Halo, propager en session, corriger AuthState/SetupState dans bootstrap. | 2-3j |
| **P0-5** | **Aligner les 6 endpoints incompatibles** sur le contrat FastAPI (routes, méthodes, DTO) : citations POST, teammates POST, synthesis POST, media POST, timeseries route+DTO, explorer/player-query rename. | 5-7j |
| **P0-6** | **Implémenter les 5 endpoints absents** : export CSV, matches-query, last-match/resolve, session-compare, timeseries (vrai). | 5-8j |

> **Exécution recommandée** : fusionner P0-5 et P0-6 en un seul workstream "Contrat API" organisé par lot/page (voir §Ordre d'exécution). Livrer des pages complètes plutôt que traiter séparément "incompatibles" vs "absents".

| **P0-7** | **Décider la stratégie de distribution cible** : serveur/container only, self-host natif par OS, ou vraie distribution desktop. Sans cette décision, impossible de corriger proprement `release.yml` et `deploy.yml`. | 1j |
| **P0-8** | **Rebaser toute l'automatisation sur l'architecture cible** : Dockerfile multi-stage Go, docker-compose, Makefile racine, `ci.yml`, `release.yml`, `bump-version.yml`, `deploy.yml`, `test-deploy-precheck.yml`, `e2e-browser-optional.yml`, `generate-types`. | 3-6j |

### P1 — Bugs / sécurité

| # | Action |
|---|--------|
| **P1-1** | Corriger fuite de connexion `pool.go` — fermer Shared + Metadata du doublon, utiliser `singleflight.Group`. |
| **P1-2** | Sécuriser `playerDoneGuard` `backfill.go` — paramètres SQL liés. |
| **P1-3** | Logger les erreurs dans `match_view_service.go` — ne pas ignorer 7 erreurs. |
| **P1-4** | Ajouter middleware CSRF (vérification Origin/Referer comme FastAPI). |
| **P1-5** | Remplacer `http.Error()` par `writeError()` dans home, stats, sessions. |
| **P1-6** | Corriger `StatsHandler` qui accepte silencieusement du JSON malformé. |
| **P1-7** | Corriger `gamertag.go` qui ne set pas `query` dans la réponse. |

### P2 — Architecture / qualité

| # | Action |
|---|--------|
| **P2-1** | Refactorer les handlers : injecter les services depuis `NewRouter` au lieu de construire repo+service inline. |
| **P2-2** | Split des 5 fichiers >500L (squad.go, skill_rating.go, queries.go, transforms.go, main.go). |
| **P2-3** | Unifier les 4 fonctions dupliquées dans squad.go via generics. |
| **P2-4** | Extraire SQL de sync/skill_rating.go dans un repository. |
| **P2-5** | Refactorer double-switch feature_flags.go en map lookup. |
| **P2-6** | Implémenter persistance MSAL cache (refresh_token DuckDB). |
| **P2-7** | Extraire `createPlayerInProfiles` de setup.go dans un `ProfileService`. |
| **P2-8** | Unifier double cache DB (`duckdb/db.go` openDBs + `pool.go` globalPool). |
| **P2-9** | Écrire une ADR explicite sur le périmètre mono-titre vs multi-titres. Si multi-titres est retenu, définir `title_slug` comme dimension d'architecture avant le premier second jeu. |

### P3 — Évolution fonctionnelle

| # | Action |
|---|--------|
| **P3-1** | Porter weapon kills film parsing si nécessaire. |
| **P3-2** | Porter fanout enrichment multi-joueur. |
| **P3-3** | Ajouter les 13+ colonnes scoreboard manquantes dans match_view. |
| **P3-4** | Porter les algorithmes d'analyse UI au fur et à mesure des besoins du frontend (cumul, form trends, intensity, distributions). |
| **P3-5** | Compléter le healthcheck Go sous `/api/v1/health`. |
| **P3-6** | Si le support multi-titres devient un besoin produit, introduire un namespace de stockage/configuration par titre (`data/titles/{title_slug}/...`) au lieu d'étendre les DB globales actuelles sans clé de partition explicite. |

### P4 — Améliorations UX / produit

> Ces items ne sont pas bloquants pour la bascule Go, mais le frontend React est le bon moment pour les traiter (refonte UI en cours). Ils améliorent la compréhension utilisateur et corrigent des approximations héritées du legacy Streamlit.

| # | Action | Contexte | Effort estimé |
|---|--------|---------|:---:|
| **P4-1** | **Graphe bipolaire solo vs escouade (Synthesis)** — vérifier que le chart diverging bar (`buildBipolaireChart()` dans `SynthesisPage.tsx`) est bien porté côté Go et que `comparison_metrics[]` est renvoyé avec les bons KPIs (`solo_kpis` vs `squad_kpis`). Le FastAPI expose ce contrat mais le Go en §1.4.4 montre que ~60% du payload synthesis est absent, incluant potentiellement ce graphe. | Existant côté React (barres horizontales cyan solo / vert escouade, axe zéro central). Absent côté Go — à implémenter dans le lot P0-5 synthesis. | Inclus P0-5 |
| **P4-2** | **Tooltips explicatifs pour LUSR et Performance Score** — ajouter une icône `(?)` avec tooltip contextuel à côté de chaque occurrence de ces métriques dans le frontend. Le tooltip doit expliquer en 1-2 phrases que c'est un calcul custom LevelUp (pas une stat officielle Halo) et contenir un lien vers une page de documentation in-app. | Aucun tooltip existant. L'utilisateur voit « Performance: 72 » ou « LUSR: 1420 » sans contexte. Ces métriques semblent opaques sans explication. | 2-3j |
| **P4-3** | **Page "Quoi de neuf" / historique des versions in-app** — créer une page accessible depuis le menu ou le footer qui affiche un changelog condensé (features majeures, dates). Alimentée soit depuis un fichier statique `CHANGELOG.md`, soit depuis les tags Git. Le tooltip P4-2 pointerait vers la section pertinente de cette page pour expliquer quand et pourquoi LUSR/Performance Score ont été introduits. | Aucun historique in-app aujourd'hui. Le `CHANGELOG.md` existe dans le repo mais n'est pas exposé au frontend. Reconstituer les dates approximatives depuis l'historique Git. | 2-3j |
| **P4-4** | **Calcul de durée de jeu — passer du span brut au découpage par session** — actuellement la durée est calculée comme `(dernier_match_end - premier_match_start)` par session groupée. Ce calcul est correct quand les matchs s'enchaînent, mais faux quand un joueur fait un match le matin et un le soir (ex: 14h de "durée" pour 2 matchs de 15 min). **Deux approches** : (a) sommer les durées individuelles des matchs (`match_duration_seconds` dans `match_registry`) pour obtenir le temps de jeu effectif ; (b) garder le span session mais afficher les deux métriques (temps de jeu effectif + durée de la session). L'approche (b) est plus riche car elle distingue temps joué vs temps connecté. | `session_compare_service.py` L112 utilise `times.min()` / `times.max()`. Le champ `duration_seconds` existe dans `match_registry` et peut être sommé. Les sessions sont déjà groupées par l'algo `analysis/sessions.py`. | 1-2j |

---

## Partie 4 — Audit Tests & Observabilité

> **Constat central** : l'audit des Parties 1-3 a révélé 6 incompatibilités de méthode HTTP, 5 endpoints absents, une surface morte autour de `GET /setup/status` (hook/types/tests conservés alors que le runtime setup est piloté par `/bootstrap`), 13+ colonnes scoreboard manquantes et 7 erreurs silencieusement ignorées. **Aucun de ces problèmes n'a été détecté automatiquement.** Cette partie analyse pourquoi et propose un plan de remédiation.

### 4.1 Inventaire de l'existant

#### Go API — 8 fichiers test, 1 716 lignes

| Fichier test | Lignes | Type | Ce qui est couvert |
|---|:---:|---|---|
| `analysis/squad_test.go` | 266 | Unit | `ComputeSquadPerformanceScore`, `resolveSquadGrade`, `ComputeParticipationProfile`, records, impact, breakdown, heatmap, top weeks |
| `analysis/sessions_test.go` | 214 | Unit | `ComputeSessions` (gap-based), `ComputeSessionsWithContext` (teammates/ranked breaks), `BuildSessionGroups`, `GetBucketInfo` |
| `analysis/home_test.go` | 262 | Unit (external) | `ComputeKPIs`, `ComputeTrend`, `BuildRecentMatches`, `BuildRecentMedia`, `BuildSessionSummary` |
| `analysis/citations_test.go` | 163 | Unit | `MergeCitationTotals`, `ExtractCategories`, `MergeMedalSummary`, `GroupCommendationsByCategory` |
| `sync/backfill_flags_test.go` | 190 | Unit (contrat) | Parité numérique bitmasks Go vs Python — chaque bit individuel + groupes |
| `service/service_test.go` | 330 | Unit | `ResolveFiltersFromRows`, `formatLifeSeconds`, `computeMapWinRates`, `paginate`, `buildHeroProgress`, `convertMedals` |
| `migration/migration_test.go` | 171 | Integration | Idempotence `RunForDB(Metadata)` sur 3 passes — build tag `integration` |
| `config/feature_flags_test.go` | 120 | Unit | Feature flags défaut, parse, env var override, fichier absent |

**Verdict** : bonne couverture des **algorithmes purs** (`analysis/`, `service/` fonctions pures). **Zéro couverture** des handlers HTTP, repositories DuckDB, middleware et error paths.

#### Fixtures et golden files Go

| Ressource | Contenu | Automatisé en CI ? |
|---|---|:---:|
| `tests/fixtures/golden_values/` (10 fichiers) | Golden JSON : health, bootstrap, players, filters, career, match_history, gamertag_search, match_view | ❌ Manuel uniquement (`parity_check.py`) |
| `tests/fixtures/baselines.json` | Baseline perf Python (p50/p95/p99, 10 samples) | ❌ Non vérifié |
| `scripts/parity_check.py` (314L) | Deep diff Go live vs golden values (tolérance float, ignore fields) | ❌ Jamais dans CI |
| `scripts/capture_golden_values.py` | Capture golden values depuis API Python live | ❌ Manuel |

#### FastAPI — 0 test

`apps/api/` contient 17 routers, 17 schémas Pydantic, 8 services — **aucun fichier test**, pas de `tests/` directory. Le backend de production n'a littéralement zéro test.

Les 5 tests Python en racine (`test_data_contract_*.py`, ~570L) sont des **contrats DuckDB** (tables/colonnes) — ils valident le schéma de la base, pas l'API.

#### React — 11 Vitest + 15 Playwright E2E

**Tests unitaires Vitest (639L)** :

| Type | Fichiers | Profondeur |
|---|:---:|---|
| Store tests | 2 | ✅ Bon — `globalFilterStore` (9 tests), `appShellStore` |
| Page smoke tests | 9 | ⚠️ Superficiel — vérifient que le composant monte sans crash, pas d'interaction utilisateur |

Infrastructure Vitest bien configurée : MSW (Mock Service Worker) avec handlers pour chaque endpoint, `renderWithProviders()` avec `QueryClient`, mock `react-plotly.js`.

**Tests E2E Playwright (15 fichiers, 958L)** :

Couvrent toutes les pages/slices. Certains vérifient la **structure JSON** de la réponse API (match_view vérifie `header`, `summary_tab`, `combat_tab`, `team_tab`). Lancés avec `make test-e2e` contre l'API Python.

**Problème** : les E2E Playwright React ne sont **pas dans la CI GitHub Actions**. Ils existent uniquement en `workflow_dispatch` manuel (et encore, le workflow cible les tests Streamlit, pas React).

---

### 4.2 Matrice de couverture par couche

| Couche | Unit | Integration | Contract | E2E | Observabilité |
|---|:---:|:---:|:---:|:---:|:---:|
| **Go — analysis/** | ✅ 4 fichiers | — | — | — | — |
| **Go — service/** | ✅ Fonctions pures | ❌ Orchestration non testée | — | — | — |
| **Go — handlers/** | ❌ **0 test** | ❌ **0 httptest** | ❌ **0 vs OpenAPI** | — | ✅ slog middleware |
| **Go — repository/DuckDB** | ❌ **0 test** | ❌ | — | — | ❌ |
| **Go — middleware** | ❌ **0 test** | — | — | — | ✅ Existe mais non vérifié |
| **Go — migrations** | — | ✅ Metadata uniquement | — | — | — |
| **Go — sync engine** | ✅ Bitmasks | ❌ Orchestration non testée | — | — | — |
| **FastAPI — routers/** | ❌ **0 test** | ❌ | ❌ | — | ❌ |
| **FastAPI — schemas/** | ❌ **0 test** | — | ❌ | — | — |
| **FastAPI — services/** | ❌ **0 test** | — | — | — | — |
| **React — stores** | ✅ 2 fichiers | — | — | — | — |
| **React — pages** | ⚠️ Smoke only | — | — | ✅ Playwright (15 slices) | — |
| **Cross-layer** | — | — | ❌ **0 Go vs FastAPI** | ❌ **0 onboarding flow** | ❌ **0 shadow mode** |

**Lecture** : la diagonale vert clair (analysis, stores, E2E pages) fonctionne. Tout le reste est un trou noir.

---

### 4.3 Analyse des lacunes critiques

#### Lacune 1 — Zéro test de handler HTTP Go (impact : P0)

Les 21 handlers Go n'ont aucun test `httptest`. Concrètement :

| Problème non détecté | Pourquoi |
|---|---|
| 4 endpoints avec mauvaise méthode HTTP (POST→GET) | Un test `router.Routes()` aurait listé toutes les routes avec leurs méthodes et comparé à l'OpenAPI spec |
| `http.Error()` plain text dans 3 handlers | Un test `httptest.NewRecorder` + assertion `Content-Type: application/json` l'attrape instantanément |
| JSON malformé accepté silencieusement par `StatsHandler` | Un test envoyant `{"garbage": true}` au handler vérifierait la réponse 400 |
| 7 erreurs ignorées dans `match_view_service.go` | Un test injectant un repo qui retourne `error` vérifierait que le handler répond 500, pas 200 |

**Cause racine** : les handlers construisent `repo+service` inline (`duckdb.NewXxxRepo(pdb)` → `service.NewXxxService(repo)`), ce qui les rend **non-mockables** sans refactoring (cf. P2-1). L'absence de tests est autant un symptôme qu'une cause.

#### Lacune 2 — Zéro contract testing OpenAPI (impact : P0)

L'`openapi.yaml` Go (2 121 lignes) est linté par Spectral en CI (`continue-on-error: true` — même les erreurs de lint sont ignorées), mais **jamais comparé aux routes réellement enregistrées dans chi**.

Ce qui manque :
- Un test parsant `openapi.yaml` et vérifiant que chaque `path+method` a un handler chi correspondant → aurait détecté les 5 endpoints absents
- Un test envoyant une requête à chaque route et validant la réponse JSON contre le schéma OpenAPI → aurait détecté les DTOs appauvris (synthesis 60% absent, scoreboard 13+ colonnes)
- Une comparaison automatique `openapi.yaml` Go vs schémas Pydantic FastAPI → aurait détecté les 6 incompatibilités de méthode

**Note** : le `continue-on-error: true` sur le job `go-openapi-lint` est un signal en soi — l'OpenAPI spec a des erreurs connues mais tolérées.

#### Lacune 3 — Zéro test FastAPI (impact : P0)

17 routers de production avec 0 test. Le backend qui sert effectivement le frontend React n'a aucune couverture. Conséquences :

- Pas de détection si un router change sa shape de réponse (régression silencieuse pour le frontend)
- Pas de snapshot des réponses Pydantic sérialisées (base de comparaison pour le Go)
- Pas de vérification que les schémas Pydantic sont cohérents avec l'OpenAPI consommé par le frontend

**Ironie** : les 5 tests `test_data_contract_*.py` en racine vérifient que DuckDB a les bonnes colonnes, mais personne ne vérifie que FastAPI expose correctement ces colonnes au frontend.

#### Lacune 4 — Golden values non automatisées (impact : P1)

Le tooling existe — `parity_check.py` est un script sophistiqué (deep diff avec tolérance float, champs ignorés, seuils de perf). Le problème est qu'il n'est **jamais exécuté automatiquement** :

- Pas dans CI (nécessite les deux backends live)
- Pas de `make check-parity` dans le workflow
- Les `baselines.json` (seuils perf « si Go > 2× p95 Python = bug ») ne sont vérifiés nulle part

C'est un filet de sécurité entièrement construit mais **pas branché**.

#### Lacune 5 — E2E Playwright déconnectés de la CI (impact : P1)

15 specs Playwright E2E React existent et couvrent toutes les pages. Mais :

- Le workflow CI `e2e-browser-optional.yml` est en `workflow_dispatch` (manuel) et cible les tests **Streamlit**, pas React
- Les tests React E2E (`apps/web/e2e/`) n'ont **aucun workflow CI**
- `make test-e2e` fonctionne localement mais n'est jamais appelé en CI

Ces tests auraient pu détecter automatiquement les artefacts morts autour de `/setup/status` (spec E2E incohérente, handlers MSW obsolètes), ainsi que les vraies 404/500 sur les pages branchées.

#### Lacune 6 — Zéro test repository DuckDB (impact : P2)

La couche `internal/port/` (interfaces) et `internal/platform/duckdb/` (implémentation) ne sont jamais testées. Risques non couverts :

- Que se passe-t-il quand une table attendue n'existe pas ? (migration non appliquée)
- Que se passe-t-il quand une colonne a changé de type ? (drift de schéma)
- Le pool de connexions (`pool.go`) — fuite détectée dans l'audit, jamais détectable sans stress test
- Les requêtes SQL — concaténation dans `backfill.go` (injection potentielle)

#### Lacune 7 — Zéro test d'error path (impact : P2)

Aucun test ne simule :
- DuckDB inaccessible → le handler devrait retourner 503
- Player non trouvé → 404 (pas 500)
- Token MSAL expiré → 401 propre
- JSON body malformé → 400 avec message explicite

Résultat : les 7 erreurs silencieusement ignorées dans `match_view_service.go` sont passées inaperçues.

#### Lacune 8 — Zéro test de migration player/shared/shared_pve (impact : P2)

`migration_test.go` teste uniquement `TargetMetadata`. Les 3 autres targets (`player`, `shared`, `shared_pve`) n'ont aucun test d'idempotence. Si une migration `shared` casse, la détection est 100% manuelle.

---

### 4.4 Observabilité et monitoring

#### Ce qui existe

| Composant | Technologie | Couverture |
|---|---|---|
| Logger Go | `log/slog` (stdlib) | JSON en prod, texte en dev |
| Middleware request logging | `slog_logger.go` | method, path, status, duration_ms, request_id, remote_addr |
| Request ID | `request_id.go` | `X-Request-ID` (UUID, propagé ou généré) |
| Rate limiting | `go-chi/httprate` | En place |
| Panic recovery | `chi/middleware.Recoverer` | En place |
| Notifications | Discord embeds (4 fichiers `internal/notify/`) | Sync results, errors |
| Healthcheck | `GET /health` | `{status, match_count, db_version}` |

#### Ce qui manque

| Lacune | Impact | Aurait détecté |
|---|---|---|
| **0 métriques Prometheus/OpenTelemetry** | Pas de dashboard latence/erreurs par route | Les endpoints Go 3× plus lents que Python (sans `baselines.json` en CI) |
| **0 tracing distribué** | Impossible de suivre une requête bootstrap → auth → sync → response | L'onboarding cassé de bout en bout |
| **0 error tracking (Sentry)** | Les erreurs silencieuses (`nolint:errcheck`) sont perdues | Les 7 erreurs ignorées dans MatchView |
| **0 response body logging** | Le middleware logue status/duration mais **pas la taille ni la shape** de la réponse | Un endpoint renvoyant 200 avec un body appauvri (synthesis 60% absent) passe complètement inaperçu |
| **0 contract validation runtime** | Pas de middleware validant les réponses JSON contre l'OpenAPI spec | DTOs appauvris en Go (`scoreboard` 13+ colonnes manquantes) détectables dès le premier appel |
| **0 shadow/dual monitoring** | Le feature flag Go↔Python existe mais sans comparaison automatique des réponses | **Toutes** les incompatibilités de contrat de la Partie 1 — en activant le shadow mode avec diff logging, chaque divergence serait visible dans les logs |
| **0 alerting sur error rate** | Pas de seuil d'erreur par route qui déclenche une notification | Un handler qui passe de 0% à 5% d'erreurs après un déploiement |
| **FastAPI : 0 logging structuré** | Pas de middleware de logging dédié dans `apps/api/` | Le backend de production n'a aucune observabilité propre |

---

### 4.5 CI/CD — ce qui tourne vs ce qui devrait tourner

#### État actuel de la CI (`ci.yml`)

| Job CI | Cible | Tourne ? | Utile pour la migration ? |
|---|---|:---:|:---:|
| `fast-data-contracts` | 5 tests DuckDB (tables/colonnes) | ✅ | ⚠️ Partiel — valide le schéma DB, pas l'API |
| `test` | pytest Python (3.11 + 3.12) | ✅ | ⚠️ Teste le code Streamlit legacy, pas FastAPI |
| `frontend` | npm ci + typecheck + lint + build + vitest | ✅ | ✅ Attrape les erreurs de type React |
| `lint` | ruff, black, isort | ✅ | ❌ Ne lint pas FastAPI spécifiquement |
| `quality` | `enforce_size_limits` (lignes/fonctions) | ✅ | ❌ Code legacy uniquement |
| `go-build` | `go vet` + `go test` (sans integration) | ✅ | ✅ Tests analysis + service + flags |
| `go-lint` | golangci-lint v1.62 | ✅ | ✅ Qualité code Go |
| `go-coverage` | Seuil min **30%** | ✅ | ⚠️ Seuil très bas — masque les 0% sur handlers/repos |
| `go-openapi-lint` | Spectral sur `openapi.yaml` | ⚠️ `continue-on-error` | ❌ Erreurs tolérées, pas bloquant |

**Limites importantes du chemin Go actuel dans `ci.yml`** :

- `go build` prouve un build de base, mais pas une release distribuable ;
- `go test` et `go-coverage` tournent avec `CGO_ENABLED=0`, donc hors du chemin DuckDB réel ;
- la matrice ne couvre pas macOS, alors que les bindings DuckDB embarquent aussi des libs Darwin ;
- aucun job ne publie ou ne valide un artefact Go + web dist consommable.

#### Workflows annexes à revoir hors `ci.yml`

| Workflow | État actuel | Problème pour l'architecture cible |
|---|---|---|
| `release.yml` | Construit un zip Windows Python portable + une archive Unix source via `packaging/build_release.py` | Ne produit aucun artefact Go/web aligné sur le runtime cible |
| `bump-version.yml` | Modifie seulement `pyproject.toml` | Pas de source de version unifiée pour Go, web, images et releases |
| `deploy.yml` | Déploie `deploy.sh` sur une stack Python/FastAPI | Ne valide pas le chemin de déploiement Go |
| `test-deploy-precheck.yml` | Vérifie l'image et le compose Python actuels | Ne couvre pas la future image Go ni la chaîne release associée |
| `e2e-browser-optional.yml` | Lance des E2E navigateur Streamlit | Ne couvre pas le frontend React avec le backend cible |

**Jobs absents critiques** :

| Job manquant | Ce qu'il attraperait |
|---|---|
| **`go-contract-test`** | Routes Go vs OpenAPI spec → 5 endpoints absents, 4 méthodes incompatibles |
| **`go-golden-test`** | Réponse Go vs golden values → DTOs appauvris (synthesis, scoreboard) |
| **`go-handler-test`** | Status codes + Content-Type + body shape → `http.Error()` plain text, JSON malformé accepté |
| **`fastapi-test`** | Routers + schémas Pydantic → le backend de production est non testé |
| **`e2e-react`** | Playwright React sur CI → artefacts morts `/setup/status`, onboarding flow, navigation complète |
| **`parity-check`** | Go vs Python response diff → toutes les incompatibilités de contrat |
| **`go-integration`** | Migrations player/shared/shared_pve + queries DuckDB réelles → drift de schéma |

#### Seuil de couverture Go : 30% — analyse

Le seuil de 30% en CI est **trompeur**. La couverture réelle par zone :

| Zone | Couverture estimée | Pondération dans le total |
|---|:---:|---|
| `analysis/` (~3 000L) | ~60-70% | Tire le chiffre vers le haut |
| `service/` fonctions pures (~500L) | ~40-50% | Contribue modérément |
| `config/` (~300L) | ~80% | Petit volume mais bien couvert |
| `sync/` bitmasks (~200L) | ~90% | Petit volume, quasi complet |
| `migration/` (~400L) | ~30% (metadata only) | 3 targets sur 4 non testées |
| `handlers/` (~4 000L) | **0%** | **23% du code total, zéro test** |
| `platform/duckdb/` (~2 000L) | **0%** | Repository + pool non testés |
| `service/` orchestration (~2 000L) | **0%** | Fonctions avec dépendances DB non testées |

Le seuil de 30% est atteint grâce aux algorithmes purs (analysis, config, bitmasks) qui compensent le 0% de la couche HTTP + DB. **Monter le seuil à 50% forcerait mécaniquement l'écriture de tests handlers et repository.**

---

### 4.6 Recommandations priorisées

#### R0 — Purge des surfaces mortes et doubles sources de vérité (effort : 1-2j, ROI immédiat)

1. Décider explicitement du sort de `/setup/status` : soit le supprimer partout, soit le réintroduire volontairement.
2. Distinguer dans la doc et le code ce qui relève du runtime, du codegen, des fixtures de test et des endpoints Go-only.
3. Aligner `MIGRATION_MASTER`, `generated.ts`, `types.ts`, les handlers MSW et les specs E2E sur cette décision.

→ Évite de prioriser des faux positifs et rend le reste de l'audit mécaniquement plus fiable.

#### R1 — Contract tests Go (effort : 2-3j, ROI immédiat)

Test automatisé en CI qui :
1. Parse `openapi.yaml` et liste tous les `path + method`
2. Itère les routes enregistrées dans le routeur chi
3. Vérifie que chaque route OpenAPI a un handler avec la bonne méthode HTTP
4. Vérifie que chaque handler renvoie `Content-Type: application/json` (pas `text/plain`)

→ Aurait détecté : 5 endpoints absents, 4 méthodes incompatibles, 3 `http.Error()` plain text.

#### R2 — Golden tests automatisés en CI (effort : 2-3j, ROI immédiat)

Intégrer `parity_check.py` dans un job CI :
1. Démarrer le backend Go en mode test (DuckDB fixtures)
2. Appeler chaque endpoint avec des paramètres fixes
3. Comparer la réponse JSON à un golden file (deep diff avec tolérance)
4. Échec CI si la shape diffère (clés manquantes ou en trop)

→ Aurait détecté : synthesis 60% payload absent, scoreboard 13+ colonnes manquantes, `query` absent dans gamertag_search.

#### R3 — Tests handlers `httptest` (effort : 3-5j, ROI élevé)

Pour chaque handler Go :
1. Créer un mock repository (interface `port.XxxRepository` → mock)
2. Injecter via constructeur (nécessite P2-1 — refactoring injection)
3. Tester : status code, Content-Type, body JSON shape, error paths (repo retourne erreur → 500 JSON)

**Prérequis** : P2-1 (injection de dépendances dans handlers). Sans ça, les handlers ne sont pas testables unitairement.

**Alternative rapide** (sans P2-1) : tests d'intégration HTTP avec `httptest.NewServer(router)` + DuckDB en mémoire. Moins isolé mais testable immédiatement.

#### R4 — Shadow mode avec diff logging (effort : 3-5j, ROI élevé pendant la migration)

Le feature flag Go↔Python existe déjà. Ajouter un mode `"both"` qui :
1. Appelle les deux backends en parallèle
2. Compare les réponses JSON (clés, types, shape — pas les valeurs exactes)
3. Logue les diffs en `slog.Warn` avec le path de la route
4. Retourne la réponse du backend de référence (Python) au frontend

→ Détecterait **en temps réel** toute divergence de contrat pendant la migration.

#### R5 — E2E Playwright React en CI (effort : 1-2j, ROI élevé)

Les 15 specs Playwright existent déjà. Il suffit de :
1. Ajouter un job `e2e-react` dans `ci.yml`
2. Démarrer `make dev` (API + Vite) en mode demo
3. Exécuter `npx playwright test` avec Chromium headless

→ Aurait détecté : artefacts morts `/setup/status`, onboarding cassé, toute page qui plante sur une 404/500.

#### R6 — Tests FastAPI minimal (effort : 2-3j, ROI modéré)

Créer `apps/api/tests/` avec :
1. `TestClient` pour chaque router (status code + shape de réponse)
2. Snapshot tests : capturer la réponse sérialisée et la comparer à un golden file
3. Ces golden files deviennent la **source de vérité** pour le Go

→ Sécurise le backend de production ET crée la baseline de comparaison pour `parity_check.py`.

#### R7 — Observabilité minimale (effort : 2j, ROI long terme)

1. **Response size middleware** Go : ajouter `response_bytes` au log slog → détecter les réponses anormalement petites
2. **Contract validation middleware** (dev mode) : valider chaque réponse JSON contre `openapi.yaml` avec `kin-openapi` → alerter sur les fields manquants
3. **Monter le seuil de couverture** Go de 30% à 50% → forcer mécaniquement l'écriture de tests handlers
4. **Retirer `continue-on-error`** de `go-openapi-lint` → rendre le lint OpenAPI bloquant

#### R8 — Rebaser versioning, releases et déploiement sur le runtime cible (effort : 3-5j, ROI élevé)

1. Choisir explicitement le ou les canaux de distribution : OCI image, self-host natif, desktop.
2. Remplacer la logique `pyproject.toml`-only par une source de version commune (tag Git ou fichier `VERSION`) consommée par Go, web et release notes.
3. Construire les artefacts Go sur une matrice native par OS cible, avec validation CGo/DuckDB sur les plateformes réellement supportées.
4. Réécrire `release.yml`, `deploy.yml` et `test-deploy-precheck.yml` pour produire et valider les artefacts du runtime cible, pas ceux du runtime Python historique.
5. Ajouter la publication des images OCI et, si nécessaire, des archives par OS contenant binaire Go + `apps/web/dist` + fichiers de config.

#### Matrice effort / impact

| Recommandation | Effort | Impact | Prérequis |
|---|:---:|:---:|---|
| **R0** Purge surfaces mortes / vérité unique | 1-2j | 🔴 Critique | Aucun |
| **R1** Contract tests OpenAPI | 2-3j | 🔴 Critique | Aucun |
| **R2** Golden tests en CI | 2-3j | 🔴 Critique | DuckDB fixtures ou API live |
| **R5** Playwright React en CI | 1-2j | 🟠 Élevé | Aucun (specs existent) |
| **R7** Observabilité minimale | 2j | 🟠 Élevé | Aucun |
| **R8** Releases/deploy/versioning cible | 3-5j | 🟠 Élevé | Décision explicite sur le canal de distribution |
| **R3** Tests handlers httptest | 3-5j | 🟠 Élevé | P2-1 (injection) ou integ |
| **R4** Shadow mode diff | 3-5j | 🟠 Élevé | Feature flags existants |
| **R6** Tests FastAPI | 2-3j | 🟡 Modéré | Aucun |

**Effort total** : ~15-23 jours-dev pour passer d'une couverture structurellement aveugle à un filet de sécurité complet pour la migration.

**Quick wins immédiats** (< 1 jour chacun) :
- Décider explicitement du sort de `/setup/status` et nettoyer les artefacts associés
- Retirer `continue-on-error` du lint OpenAPI
- Décider explicitement si la distribution cible est serveur/container only ou s'il faut maintenir de vraies releases natives par OS
- Monter le seuil de couverture Go de 30% à 50%
- Ajouter le job Playwright React dans la CI (les specs existent déjà)

---

## Méthodologie

**Fichiers inspectés** :

- **Backend Go** : `internal/api/handlers/*.go` (22 fichiers), `internal/api/server.go`, `internal/service/*.go`, `internal/domain/*.go`, `internal/platform/duckdb/pool.go`, `internal/platform/auth/*.go`, `internal/sync/backfill.go`, `internal/sync/skill_rating.go`, `internal/analysis/squad.go`, `cmd/levelup/main.go`
- **Backend FastAPI** : `apps/api/app/main.py`, 16 routeurs, 17 schémas Pydantic, `apps/api/app/deps/`, `apps/api/app/core/`
- **Frontend React** : `apps/web/src/features/*/queries.ts` (12 features), `apps/web/src/lib/api/client.ts`, `types.ts`, `generated.ts`, `apps/web/src/routes/`
- **Infrastructure** : `Dockerfile`, `docker-compose.yml`, `Makefile` (racine), `apps/go-api/Makefile`, `apps/web/package.json`, `packaging/build_release.py`
- **Gouvernance** : `SPRINT_ROADMAP.md`, `GO_ARCHITECTURE_RULES.md`, `MATRIX.md`
- **Tests Go** : `internal/analysis/*_test.go` (4), `internal/sync/backfill_flags_test.go`, `internal/service/service_test.go`, `internal/migration/migration_test.go`, `internal/config/feature_flags_test.go`
- **Tests React** : `apps/web/src/features/*/Page.test.tsx` (9), `apps/web/src/stores/*.test.ts` (2), `apps/web/e2e/slice-*.spec.ts` (15)
- **Fixtures Go** : `tests/fixtures/golden_values/` (10 fichiers), `tests/fixtures/baselines.json`, `scripts/parity_check.py`, `scripts/capture_golden_values.py`
- **CI/CD** : `.github/workflows/ci.yml`, `release.yml`, `deploy.yml`, `bump-version.yml`, `e2e-browser-optional.yml`, `test-deploy-precheck.yml`, `.pre-commit-config.yaml`, `.golangci.yml`

**Nuance de calibration** : le runtime React importe aujourd'hui `apps/web/src/lib/api/types.ts` et pilote le setup via `/bootstrap` ; `generated.ts` et certaines specs/fixtures restent partiellement désalignés, en particulier autour de `/setup/status`.

**Audit statique uniquement.** Aucune suite de tests, aucun build end-to-end et aucune validation navigateur n'ont été exécutés.
