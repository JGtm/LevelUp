# Audit Go API — Parité fonctionnelle & qualité d'architecture

> Date : 2026-04-16 (v2 — réécriture complète)
> Périmètre audité :
> - `apps/go-api/` (~17 500 LOC Go) — backend cible
> - `apps/api/` (~5 000 LOC Python/FastAPI) — backend transitoire effectivement en production
> - `apps/web/` — frontend React consommant le contrat FastAPI
> - `LevelUp/src/` (~55 000 LOC Python) — legacy Streamlit de référence
> - Infra racine : Dockerfile, docker-compose.yml, Makefile

---

## Table des matières

- [Partie 1 — Parité fonctionnelle](#partie-1--parité-fonctionnelle)
  - [1.0 Cadrage : quel est le bon référentiel ?](#10-cadrage--quel-est-le-bon-référentiel-)
  - [1.1 Matrice de correspondance routes (Go vs FastAPI/React)](#11-matrice-de-correspondance-routes-go-vs-fastapireact)
  - [1.2 Écarts de DTO champ par champ](#12-écarts-de-dto-champ-par-champ)
  - [1.3 Sync Engine](#13-sync-engine)
  - [1.4 Algorithmes d'analyse](#14-algorithmes-danalyse)
  - [1.5 Authentification et onboarding](#15-authentification-et-onboarding)
  - [1.6 Migrations, Notifications, Ops](#16-migrations-notifications-ops)
  - [1.7 Runtime et infrastructure](#17-runtime-et-infrastructure)
  - [1.8 Synthèse des écarts critiques](#18-synthèse-des-écarts-critiques)
- [Partie 2 — Audit qualité code Go](#partie-2--audit-qualité-code-go)
  - [2.1 Architecture hexagonale — design vs réalité](#21-architecture-hexagonale--design-vs-réalité)
  - [2.2 Bugs et risques sécurité (P0)](#22-bugs-et-risques-sécurité-p0)
  - [2.3 Gestion d'erreur HTTP incohérente](#23-gestion-derreur-http-incohérente)
  - [2.4 Dépassements de taille (P1)](#24-dépassements-de-taille-p1)
  - [2.5 Violations DRY / architecture (P2)](#25-violations-dry--architecture-p2)
  - [2.6 Stubs et workarounds encore présents](#26-stubs-et-workarounds-encore-présents)
  - [2.7 Points forts](#27-points-forts)
- [Plan d'action recommandé](#plan-daction-recommandé)

---

## Partie 1 — Parité fonctionnelle

### 1.0 Cadrage : quel est le bon référentiel ?

La parité ne se mesure **pas** contre le legacy Streamlit. Elle se mesure contre le **contrat effectivement consommé aujourd'hui par le frontend React** (`apps/web/`) via le backend FastAPI (`apps/api/`).

Le frontend React appelle **31 endpoints** via `fetch` natif avec `credentials: "include"` (cookies httpOnly). Le schéma de référence est défini par les modèles Pydantic de `apps/api/app/schemas/` et consommé par les hooks TanStack Query de `apps/web/src/features/*/queries.ts`.

Un backend Go fonctionnellement complet mais non-substituable au backend FastAPI ne constitue **pas** une migration terminée.

---

### 1.1 Matrice de correspondance routes (Go vs FastAPI/React)

| # | Route FastAPI (contrat React) | Méthode | Route Go | Méthode Go | Statut |
|---|-------------------------------|---------|----------|------------|--------|
| 1 | `/api/v1/health` | GET | `/health` | GET | ⚠️ **Path différent** — casse les healthchecks Docker/compose |
| 2 | `/api/v1/bootstrap` | GET | `/api/v1/bootstrap` | GET | ✅ Conforme |
| 3 | `/api/v1/players` | GET | `/api/v1/players` | GET | ✅ Conforme |
| 4 | `/api/v1/session/context` | POST | `/api/v1/session/context` | POST | ✅ Conforme |
| 5 | `/api/v1/players/{slug}/filters/resolve` | POST | `/api/v1/players/{slug}/filters/resolve` | POST | ✅ Conforme |
| 6 | `/api/v1/auth/device-flow/start` | POST | `/api/v1/auth/device-flow/start` | POST | ✅ Conforme |
| 7 | `/api/v1/auth/device-flow/{id}` | GET | `/api/v1/auth/device-flow/{id}` | GET | ✅ Conforme |
| 8 | `/api/v1/setup/players` | POST | `/api/v1/setup/players` | POST | ✅ Conforme |
| 9 | `/api/v1/setup/smoke-test` | POST | `/api/v1/setup/smoke-test` | POST | ✅ Conforme |
| 10 | `/api/v1/sync/initial` | POST | `/api/v1/sync/initial` | POST | ✅ Conforme |
| 11 | `/api/v1/jobs/{job_id}` | GET | `/api/v1/jobs/{job_id}` | GET | ✅ Conforme |
| 12 | `/api/v1/settings` | GET | `/api/v1/settings` | GET | ✅ Conforme |
| 13 | `/api/v1/settings` | PATCH | `/api/v1/settings` | PATCH | ✅ Conforme |
| 14 | `/api/v1/settings/media/reset-index` | POST | `/api/v1/settings/media/reset-index` | POST | ✅ Conforme |
| 15 | `.../pages/career` | GET | `.../pages/career` | GET | ✅ Conforme |
| 16 | `.../pages/career/top-matches` | GET | `.../pages/career/top-matches` | GET | ✅ Conforme |
| 17 | `.../pages/career/encounters` | GET | `.../pages/career/encounters` | GET | ✅ Conforme |
| 18 | `.../pages/match-history/query` | POST | `.../pages/match-history/query` | POST | ✅ Conforme |
| 19 | `.../pages/match-history/export` | POST | — | — | ❌ **ABSENT** |
| 20 | `.../matches/{match_id}` | GET | `.../matches/{match_id}` | GET | ✅ Route OK, **DTO incomplet** |
| 21 | `/api/v1/directory/gamertags/search` | GET | `/api/v1/directory/gamertags/search` | GET | ✅ Conforme |
| 22 | `.../pages/explorer/player-query` | POST | `.../pages/explorer/player-query` | POST | ⚠️ **Nom de champ** : `target_gamertag` (FastAPI) vs `other_gamertag` (Go) |
| 23 | `.../pages/explorer/matches-query` | POST | — | — | ❌ **ABSENT** |
| 24 | `.../pages/last-match/resolve` | POST | — | — | ❌ **ABSENT** |
| 25 | `.../pages/home` | GET | `.../pages/home` | GET | ✅ Conforme |
| 26 | `.../battlepass` | GET | `.../battlepass` | GET | ✅ Conforme |
| 27 | `.../challenges` | GET | `.../challenges` | GET | ✅ Conforme |
| 28 | `.../pages/citations` | **POST** | `.../pages/citations` | **GET** | ❌ **Méthode incompatible** + filtres perdus |
| 29 | `.../pages/teammates` | **POST** | `.../pages/squad` | **GET** | ❌ **Route + méthode + DTO incompatibles** |
| 30 | `.../pages/synthesis` | **POST** | `.../pages/synthesis` | **GET** | ❌ **Méthode + DTO incompatibles** |
| 31 | `.../pages/media` | **POST** | `.../pages/media` | **GET** | ❌ **Méthode incompatible** + pagination/filtres perdus |
| 32 | `.../pages/timeseries` | **POST** | `.../pages/stats/query` | **POST** | ❌ **Route + DTO complètement différents** |
| 33 | `.../pages/session-compare` | POST | — | — | ❌ **ABSENT** |
| — | — | — | `.../pages/sessions` | GET | 🔵 Go-only (pas dans FastAPI) |
| — | — | — | `.../pages/commendations` | GET | 🔵 Go-only (pas dans FastAPI) |

**Résultat : sur 33 endpoints frontend, 16 sont conformes, 5 sont absents, 6 ont des incompatibilités de route/méthode/DTO, et 4 Go-only n'existent pas côté FastAPI.**

---

### 1.2 Écarts de DTO champ par champ

#### Timeseries (`POST .../pages/timeseries`) vs Stats (`POST .../pages/stats/query`)

| FastAPI (contrat React) | Go |
|------------------------|----|
| Request : `{ filters: FilterContextInput }` | Request : `{ tab: "win_loss\|accuracy\|...", mode: "period\|sessions" }` |
| Response : `TimeseriesPageResponse` avec 5 onglets : `summary_tab`, `cumul_tab`, `form_tab`, `intensity_tab`, `distributions_tab` | Response : `StatsPageResponse` orienté datasets bruts (`win_loss`, `accuracy`, `objective`, `form`, `lusr`) |
| Chaque onglet contient des `PlotlyFigurePayload` (charts Plotly sérialisés) | Pas de Plotly — données brutes uniquement |

→ **Incompatibilité totale**. Le frontend attend des charts Plotly, le Go renvoie des séries de points.

#### Teammates (`POST .../pages/teammates`) vs Squad (`GET .../pages/squad`)

| FastAPI | Go |
|--------|----|
| Route : `.../pages/teammates` | Route : `.../pages/squad` |
| Méthode : POST | Méthode : GET |
| Request body : `{ selected_gamertags, filters }` | Query param : `?teammate=xuid` |
| Response : `{ options, teammates: [TeammateRow], solo_reference, total_matches }` | Response : `{ top_teammates, selected_teammate, solo_stats, squad_stats }` |
| `TeammateRow` : `with_kpis` + `without_kpis` (comparaison avec/sans) | `TopTeammate` : structure plate différente |

→ **Incompatibilité route + méthode + payload + structure de réponse.**

#### Synthesis (`POST .../pages/synthesis`) vs Synthesis (`GET .../pages/synthesis`)

| FastAPI | Go |
|--------|----|
| Request body : `{ period: "all\|2y\|1y\|1m\|1w", filters }` | Pas de body (GET) |
| Response : `{ solo_kpis, squad_kpis, comparison_metrics, heatmap_data, top_weeks }` | Response : `{ heatmap, top_weeks, total_matches, overall_win_rate }` |
| `comparison_metrics` : liste de métriques comparatives solo vs squad | **Absent en Go** |
| `solo_kpis` / `squad_kpis` : 8 KPIs chacun | **Absent en Go** |

→ **Incompatibilité méthode + 60% du payload manquant.**

#### Media (`POST .../pages/media`) vs Media (`GET .../pages/media`)

| FastAPI | Go |
|--------|----|
| Request body riche : `sort`, `kind_filter`, `section_filter`, `map_filter`, `mode_filter`, `group_by`, `pagination` | Pas de filtres (GET) |
| Response : `{ items: PaginatedResponse[MediaItemRow], total_mine, total_teammates, total_unassigned }` | Response : `{ items, total }` (simplifiée) |

→ **Incompatibilité méthode + filtres + compteurs par catégorie manquants.**

#### Citations (`POST .../pages/citations`) vs Citations (`GET .../pages/citations`)

| FastAPI | Go |
|--------|----|
| Request body : `{ filters: FilterContextInput }` | Pas de body (GET) |
| Response : `{ commendations, medals_summary, deltas, distribution_chart }` | Response : `{ commendations, medals }` |
| Inclut `deltas` (variations filtres) et `distribution_chart` (Plotly) | **Absents** |

→ **Incompatibilité méthode + champs manquants.**

#### Match View (même route, DTO incomplet)

| Champ FastAPI `MatchScoreboardRow` | Présent en Go |
|------------------------------------|:-------------:|
| `shots_fired`, `shots_hit`, `accuracy` | ❌ |
| `damage_dealt`, `damage_taken`, `damage_efficiency` | ❌ |
| `headshot_kills`, `max_killing_spree` | ❌ |
| `perfect_kills`, `power_weapon_kills`, `melee_kills` | ❌ |
| `objectives_stolen` | ❌ |
| `dominance_flag`, `had_bot_teammate` (header) | ❌ |

→ **13+ colonnes scoreboard absentes.**

#### Explorer Player Query (même route, noms de champs différents)

| FastAPI | Go |
|--------|----|
| `target_gamertag` | `other_gamertag` |
| Response : `{ target, summary, allies_table, enemies_table, common_matches }` | Response simplifiée |

→ **Incompatibilité JSON clé du request body + réponse simplifiée.**

---

### 1.3 Sync Engine

**Verdict : ~85% porté — cœur fonctionnel, quelques modules avancés manquants.**

| Composant | Python | Go | Statut |
|-----------|--------|-----|--------|
| Delta/Full sync + SyncScope (96 champs) | `engine.py` + 8 mixins + `scope.py` | `engine.go` + `scope.go` | ✅ |
| Backfill detection + CLI | `scripts/backfill/` | `backfill.go` + `backfill_cli.go` + `backfill_flags.go` | ✅ |
| API client Halo | `api_client.py` (wrapper SPNKr) | `halo_client.go` (HTTP natif) | ✅ **Modernisé** |
| Transformers, writes, career, LUSR, perf, aggregates, PvE | 15+ fichiers | 12 fichiers | ✅ |
| **Weapon kills film parsing (NS timeline)** | `_engine_weapon_kills.py` | — | ❌ **Manquant** |
| **Fanout enrichment multi-joueur** | `_engine_fanout.py` | — | ❌ **Manquant** |
| **Batch audit/columns** | `_batch_audit.py`, `_batch_columns.py` | — | ❌ |
| **Challenge migrations, asset langs** | 2 fichiers | — | ❌ |

---

### 1.4 Algorithmes d'analyse

**Verdict : ~40% porté — algorithmes cœur OK, analyses UI absentes (attendu).**

Portés : citations, performance_score, skill_rating (TrueSkill 2), killer_victim, sessions, spawn_detection, squad, weapon_*, home.

Non portés (15+ modules) : cumulative, comeback, friends_impact, match_cadence, match_intensity, objective_participation, participation_radar, win_streaks, maps, stats, player_index, first_events, medal_verdicts, mode_categories, playlist_groups, global_correlation.

→ C'est attendu : ces algos alimentaient exclusivement Streamlit. Cependant, **certains sont maintenant requis par le contrat FastAPI** (ex: `TimeseriesPageResponse` requiert cumul, form trends, intensity, distributions). Le Go ne les a pas car il n'a pas été porté contre ce contrat.

---

### 1.5 Authentification et onboarding

**Verdict : chaîne d'échange complète, mais incohérence applicative dans le parcours onboarding.**

#### Ce qui fonctionne

- Device Code Flow MSAL → access_token : ✅
- Chaîne d'échange Halo (XBL → XSTS → Spartan → Clearance) : ✅ implémentation native
- AttemptStore thread-safe multi-session : ✅ (nouveau, adapté au web)

#### Problèmes d'onboarding

| Problème | Détail |
|----------|--------|
| **Gamertag/XUID jamais récupérés** | `pollDeviceFlow` marque l'attempt `authorized` après l'échange Halo, mais ne renseigne jamais `Gamertag` ni `XUID`. L'attempt documente un état `provisioned` (avec gamertag récupéré) mais aucun code ne fait la transition `authorized → provisioned`. |
| **Identité Halo non propagée en session** | `GetDeviceFlowStatus` copie `LinkedHaloIdentity` en session seulement si `snapshot.Gamertag != ""` — or il est toujours vide. La session n'a donc jamais d'identité liée. |
| **Bootstrap figé en `"missing"`** | `bootstrap_service.go` L54 : `AuthState: "missing"` en dur — commenté `Sprint 0 : auth pas encore portée`. |
| **SetupState ne vérifie pas l'auth** | `resolveSetupState()` ne dépend que de la présence de joueurs, pas de l'état d'authentification. |
| **CreatePlayer bloque sans identité** | `SetupHandler.CreatePlayer` force `profile_mode = xbox` et refusera la création sans identité Halo liée en session — qui n'y arrive jamais (cf. ci-dessus). |
| **Pas de persistance MSAL cache** | Chaque restart serveur force un nouveau Device Code Flow complet (pas de refresh_token DuckDB). |

**Conséquence** : le parcours setup Go fait semblant de fonctionner mais ne peut pas aboutir de bout en bout.

---

### 1.6 Migrations, Notifications, Ops

| Domaine | Verdict |
|---------|---------|
| Migrations DuckDB | **~100%** — 36 steps Go vs 34 Python. Framework idempotent conforme. |
| Notifications Discord | **~100%** — Embeds riches, anti-spam, i18n. |
| Ops CLI | **~90%** — 14 sous-commandes dans `cmd/levelup/`. |

---

### 1.7 Runtime et infrastructure

**Constat critique : toute l'infra est câblée sur Python. Le Go n'est branché nulle part.**

| Composant | Pointe vers |
|-----------|-------------|
| `Dockerfile` | Python 3.12 + FastAPI (`uvicorn apps.api.app.main:app`) |
| `docker-compose.yml` | FastAPI sur port 8000, healthcheck Python |
| `Makefile` (racine) | Cibles `api`, `dev`, `test-api` → uvicorn FastAPI |
| `package.json` `generate-types` | `http://127.0.0.1:8000/api/openapi.json` → schéma FastAPI |
| `apps/web/src/lib/api/types.ts` | Types manuels alignés sur « schémas Pydantic du backend » |
| `apps/web/src/lib/api/client.ts` | `BASE_URL = /api/v1` (servi par FastAPI) |

Le Go-API possède son propre `apps/go-api/Makefile` avec `build`, `run`, `gen`, `lint`, mais **aucun fichier d'infra racine ne le référence**.

→ `docker compose up` exécute Python. Le Go est un sous-projet isolé.

---

### 1.8 Synthèse des écarts critiques

| # | Écart | Sévérité | Impact |
|---|-------|----------|--------|
| **C1** | **5 endpoints absents** : export, matches-query, last-match, session-compare, timeseries | BLOQUANT | Pages frontend non fonctionnelles |
| **C2** | **6 endpoints incompatibles** : citations, teammates, synthesis, media, stats, explorer/player-query | BLOQUANT | Routes/méthodes/DTO divergents |
| **C3** | **Onboarding cassé** : Gamertag jamais récupéré, AuthState toujours `"missing"` | BLOQUANT | Setup impossible bout en bout |
| **C4** | **Infra 100% Python** : Docker, compose, Makefile, generate-types | BLOQUANT | Le Go n'est pas déployable |
| **C5** | **Pas de Plotly** : les onglets timeseries/career attendent des `PlotlyFigurePayload` | HAUTE | Charts non rendables côté front |
| **C6** | **Pas de CSRF** : FastAPI vérifie `Origin`/`Referer` sur POST/PATCH/DELETE, Go n'a pas de middleware CSRF | HAUTE | Sécurité dégradée |
| **C7** | **Pas de persistance MSAL cache** | HAUTE | Re-auth à chaque restart |
| **C8** | **13+ colonnes scoreboard manquantes** dans match_view | MOYENNE | Données incomplètes |
| **C9** | **Weapon kills film parsing absent** | MOYENNE | Extraction armes impossible |
| **C10** | **Fanout enrichment absent** | MOYENNE | Multi-joueur limité en post-sync |

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

**Mais la réalité diverge dans les handlers.** Sur 22 handlers, **15 violent la règle hexagonale** :

| Violation | Handlers concernés | Pattern |
|-----------|-------------------|---------|
| Import `internal/platform/duckdb` | career, citations, explorer, filters, gamertag, home, match_history, match_view, media, sessions, squad, stats | Handler → platform (court-circuite port/service) |
| Appel `config.ResolvePlayer` | Mêmes 12 + settings | Résolution infra dans le transport |
| Construction inline `duckdb.New*Repo` + `service.New*Service` | Mêmes 12 | Le handler est un mini-composition root |

Le pattern dupliqué dans chaque handler ressemble à :

```go
func (h *XHandler) HandleX(w http.ResponseWriter, r *http.Request) {
    pdb, err := h.resolvePlayer(r)           // config.ResolvePlayer
    repo := duckdb.NewXRepo(pdb)             // import platform
    svc := service.NewXService(repo)          // assemblage inline
    result, err := svc.DoX(r.Context(), ...)
    writeJSON(w, http.StatusOK, result)
}
```

→ Ce n'est pas de l'hexagonal. C'est du **service locator pattern** avec un mini-wiring ad hoc dans chaque handler. Les ports ne sont pas utilisés pour l'injection dans les handlers de pages.

**Exception positive** : `BootstrapHandler`, `PlayersHandler` et `AuthHandler` reçoivent leurs dépendances par injection dans `NewRouter()` — c'est le pattern correct.

---

### 2.2 Bugs et risques sécurité (P0)

| # | Fichier | Problème | Sévérité |
|---|---------|----------|----------|
| **P0-1** | `platform/duckdb/pool.go` L55 | **Fuite de connexion** : quand deux goroutines ouvrent simultanément la même PlayerDB, le doublon est détecté et `pdb.Player.Close()` est appelé, mais `pdb.Shared` et `pdb.Metadata` du doublon ne sont **jamais fermés**. Même pattern dans `CloseAll()` qui ne ferme que `Player`. | CRITIQUE |
| **P0-2** | `sync/backfill.go` L175+ | **Injection SQL** : `playerDoneGuard()` construit `NOT IN ('id1','id2'...)` par concaténation de strings sans échappement. Risque quasi-nul (UUIDs) mais pattern fautif — à corriger par paramètres liés. | HAUTE |
| **P0-3** | `service/match_view_service.go` L47-54 | **7 erreurs silencieusement ignorées** : `stats, _ := s.repo.Get*(...)`. DB corrompue → onglets vides sans diagnostic. | HAUTE |
| **P0-4** | Middleware CSRF | **Absent.** FastAPI vérifie `Origin`/`Referer` sur toutes les mutations. Le Go n'a aucun middleware équivalent — les POST sont acceptés de n'importe quelle origine. | HAUTE |

---

### 2.3 Gestion d'erreur HTTP incohérente

Le contrat FastAPI renvoie des erreurs JSON structurées : `{ code, message, retryable, details }`.

Le Go possède un helper `writeError()` qui respecte ce contrat. **Mais 3 handlers utilisent `http.Error()` (texte brut)** :

| Handler | Situation | Code envoyé |
|---------|-----------|-------------|
| `home.go` GetHomePage | Player non trouvé / erreur interne | `http.Error(w, err.Error(), 404/500)` |
| `stats.go` GetPage | Player non trouvé / erreur service | `http.Error(w, err.Error(), 400/500)` |
| `sessions.go` GetSessions | Player non trouvé / erreur service | `http.Error(w, err.Error(), 400/500)` |

Le frontend React parse les erreurs JSON via `ApiError { code, message, retryable }`. Un `http.Error` texte brut → `code = "unknown_error"`, perte du message structuré.

---

### 2.4 Dépassements de taille (P1)

| # | Fichier | Lignes | Dépassement | Action recommandée |
|---|---------|:------:|:-----------:|-------------------|
| **P1-1** | `analysis/squad.go` | **812** | +62% | Split en 6 fichiers + unifier 4 fonctions dupliquées via generics/interface |
| **P1-2** | `sync/skill_rating.go` | **731** | +46% | Extraire les fonctions SQL (`loadLUSR*`, `upsertLUSR*`) dans `skill_rating_db.go` |
| **P1-3** | `platform/duckdb/queries.go` | **714** | +43% | Split par domaine fonctionnel |
| **P1-4** | `sync/transforms.go` | **570** | +14% | Extraire helpers dans `transforms_helpers.go` |
| **P1-5** | `cmd/levelup/main.go` | **532** | +6% | Extraire sous-commandes en fichiers séparés |

#### Duplication critique dans `squad.go`

| Paire dupliquée | Overlap |
|----------------|---------|
| `ComputeParticipationProfile(SquadMatchRow)` / `ComputeTeammateProfile(TeammateMatchRow)` | ~90% identique |
| `ComputeSquadRecords(SquadMatchRow)` / `ComputeTeammateRecords(TeammateMatchRow)` | ~95% identique |

---

### 2.5 Violations DRY / architecture (P2)

| # | Problème | Fichier(s) | Action |
|---|----------|-----------|--------|
| **P2-1** | **15 handlers dupliquent** le pattern `resolvePlayer → NewRepo → NewService` | `handlers/*.go` | Factory abstraite ou injection dans `NewRouter` |
| **P2-2** | SQL queries inline dans un fichier d'algorithme | `sync/skill_rating.go` L460-582 | Extraire dans un repository |
| **P2-3** | Double switch identique sur toutes les surfaces | `config/feature_flags.go` | Refactorer en map lookup |
| **P2-4** | Logique métier dans le handler | `handlers/setup.go` L168-250 (`createPlayerInProfiles`) | Extraire dans un `ProfileService` |
| **P2-5** | Double cache DB | `duckdb/db.go` `openDBs` + `duckdb/pool.go` `globalPool` | Unifier |
| **P2-6** | SQL quasi-identiques | `queries.go` Q4/Q4MV/Q5 | Factoriser |
| **P2-7** | Magic number `case 2: winScore = 1.0` | `sync/skill_rating.go` L130-135 | Constantes nommées |
| **P2-8** | ~180L de noop impls (50% du fichier) | `port/repository.go` | Déplacer dans `port_check_test.go` |
| **P2-9** | Multiple sources de vérité contrat API | OpenAPI Go, schémas Pydantic, types.ts manuels, generated.ts | Figer **une** seule source |

---

### 2.6 Stubs et workarounds encore présents

| Fichier | Stub | Impact visible |
|---------|------|---------------|
| `bootstrap_service.go` L54 | `AuthState: "missing"` en dur | Frontend affiche toujours l'auth comme absente |
| `bootstrap_service.go` L115 | `DiscordConfigured: false` en dur | Indicateur toujours faux |
| `bootstrap_service.go` L116 | `TailscaleEnabled: false` en dur | Indicateur toujours faux |
| `handlers/settings.go` L105 | Reset index médias = goroutine stub | Pas d'action réelle |
| `platform/halo/provider.go` L116, L124 | `TODO Sprint 15` × 2 | Battle Pass et Challenges = best-effort ou erreur |
| `auth.go` `pollDeviceFlow` | Ne récupère jamais Gamertag/XUID | Onboarding cassé |

Ces stubs touchent des **parcours produit visibles** — pas des détails internes.

---

### 2.7 Points forts

| Aspect | Évaluation |
|--------|-----------|
| **Structure packages** | Les 12 packages internes sont bien découpés et nommés (domain, port, service, platform, analysis, sync, migration, notify, ops, config, api, validation). |
| **`analysis/` strictement pur** | 0 import IO — algorithmes testables de manière isolée. Bonne pratique. |
| **Middlewares** | CORS, rate-limit, request-id, session cookie, structured logging — bien composés. |
| **Feature flags** | Bascule granulaire Go↔Python par surface — excellent pour migration progressive. |
| **Migrations idempotentes** | Framework robuste : registre, `schema_migrations`, compile-time checks, tests. |
| **OpenAPI + codegen** | Types générés depuis `openapi.yaml` — contrat versionné dans le repo Go. |
| **Toolchain qualité** | `.golangci.yml` : gocyclo 12, funlen 80L, lll 100, bodyclose, noctx, errcheck, goconst. |
| **Tests** | Golden values JSON, tests analysis, migration, services, backfill_flags. |
| **Seulement 4 TODOs** dans 17.5K LOC | Codebase remarquablement propre pour son volume. |
| **Sync engine Go** | Porté fidèlement avec des modernisations légitimes (HTTP natif au lieu de SPNKr, sync.Mutex au lieu de asyncio.Lock). |

---

## Plan d'action recommandé

### Priorité 0 — Prérequis bascule (BLOQUANT)

| # | Action | Effort |
|---|--------|--------|
| 1 | **Choisir une source de vérité unique** pour le contrat API. Le plus rationnel : figer l'OpenAPI FastAPI comme référence, puis réaligner Go dessus. | 1-2j cadrage |
| 2 | **Corriger l'onboarding Go** : récupérer Gamertag/XUID après l'échange Halo, propager en session, corriger `AuthState`/`SetupState` dans bootstrap. | 2-3j |
| 3 | **Aligner les 6 endpoints incompatibles** sur le contrat FastAPI (routes, méthodes, DTO) : citations, teammates, synthesis, media, timeseries, explorer/player-query. | 5-7j |
| 4 | **Implémenter les 5 endpoints absents** : export CSV, matches-query, last-match, session-compare, timeseries (vrai). | 5-8j |
| 5 | **Brancher l'infra sur Go** : Dockerfile multi-stage (Go + assets web), docker-compose, Makefile racine, generate-types depuis Go. | 3-5j |

### Priorité 1 — Bugs / sécurité

| # | Action |
|---|--------|
| 6 | Corriger fuite de connexion (`pool.go`) — fermer Shared + Metadata du doublon, utiliser `singleflight.Group` |
| 7 | Sécuriser `playerDoneGuard` (`backfill.go`) — paramètres SQL liés |
| 8 | Logger les erreurs dans `match_view_service.go` — ne pas ignorer 7 erreurs |
| 9 | Ajouter middleware CSRF (vérification Origin/Referer comme FastAPI) |
| 10 | Remplacer `http.Error()` par `writeError()` dans home, stats, sessions |

### Priorité 2 — Architecture / qualité

| # | Action |
|---|--------|
| 11 | Refactorer les handlers : injecter les services depuis `NewRouter` au lieu de construire repo+service inline |
| 12 | Split des 5 fichiers >500L (squad.go, skill_rating.go, queries.go, transforms.go, main.go) |
| 13 | Unifier les 4 fonctions dupliquées dans squad.go via generics |
| 14 | Extraire SQL de sync/skill_rating.go dans un repository |
| 15 | Refactorer double-switch feature_flags.go en map lookup |
| 16 | Implémenter persistance MSAL cache (refresh_token DuckDB) |

### Priorité 3 — Évolution fonctionnelle

| # | Action |
|---|--------|
| 17 | Porter weapon kills film parsing si nécessaire |
| 18 | Porter fanout enrichment multi-joueur |
| 19 | Porter les algorithmes d'analyse UI au fur et à mesure des besoins du frontend |

---

## Méthodologie

Fichiers principalement inspectés :

- **Backend Go** : tous les fichiers de `internal/api/handlers/`, `internal/api/server.go`, `internal/service/*.go`, `internal/domain/*.go`, `internal/platform/duckdb/pool.go`, `internal/platform/auth/*.go`, `internal/sync/backfill.go`, `internal/sync/skill_rating.go`, `internal/analysis/squad.go`
- **Backend FastAPI** : `apps/api/app/main.py`, tous les 16 routeurs, tous les 17 schémas Pydantic, `apps/api/app/deps/`, `apps/api/app/core/`
- **Frontend React** : `apps/web/src/features/*/queries.ts` (12 features), `apps/web/src/lib/api/client.ts`, `types.ts`, `generated.ts`, `apps/web/src/routes/`
- **Infrastructure** : `Dockerfile`, `docker-compose.yml`, `Makefile` (racine), `apps/go-api/Makefile`, `apps/web/package.json`
- **Gouvernance** : `SPRINT_ROADMAP.md`, `GO_ARCHITECTURE_RULES.md`

Cet audit est statique. Aucune suite de tests ni validation end-to-end n'a été exécutée.
