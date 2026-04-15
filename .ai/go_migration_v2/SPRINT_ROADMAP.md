# SPRINT ROADMAP — Migration Python → Go

> Document de suivi opérationnel : tous les sprints de A à Z, dans l'ordre.
> Chaque sprint a un objectif, des tâches, un critère de sortie et une estimation.
>
> Dernière mise à jour : 2026-04-22
> Statut global : **En cours** — Sprints 0–24 ✅ (Sprint 20 partiel : tâches 5-6 ⏳) — Phases 0+1+2+3 complètes + Sprint 18-24 ✅, Sprint 25 ⬜ en attente.

---

## Légende

| Symbole | Statut |
|---------|--------|
| ⬜ | Pas démarré |
| 🔲 | Cadré (contrat + corpus définis) |
| 🔄 | En cours |
| 🔍 | En vérification de parité |
| ✅ | Terminé |
| 🚫 | Bloqué |

**Estimation** : en jours de travail effectif (1 dev senior temps plein).

---

## Vue d'ensemble

| # | Sprint | Phase | Estimation | Statut | Dépendance |
|--:|--------|-------|:----------:|:------:|------------|
| 0 | POC DuckDB + HTTP + MSAL | Sprint 0 | 2j | ✅ | Prérequis cadrage |
| 1 | Gel contrats OpenAPI | Phase 0 | 3-5j | ✅ | Sprint 0 passé |
| 2 | Corpus golden values | Phase 0 | 3-5j | ✅ | Sprint 1 |
| 3 | Baselines de performance | Phase 0 | 1-2j | ✅ | Sprint 1 |
| 4 | Squelette HTTP + config + middleware | Phase 1 | 3-5j | ✅ | Phase 0 terminée |
| 5 | Repositories read-only + pool DuckDB | Phase 1 | 5-8j | ✅ | Sprint 4 |
| 6 | Bootstrap, players, filtres, career, history + **charting foundation** | Phase 1 | 5-7j | ✅ | Sprint 5 |
| 7 | Validation de parité Phase 1 | Phase 1 | 2-3j | ✅ | Sprint 6 |
| 8 | Explorer + Match View + killer/victim + **charting explorer** | Phase 2 | 5-7j | ✅ | Gate Phase 1 |
| 9 | Sessions (algorithme 6, 2 modes) | Phase 2 | 3-5j | ✅ | Sprint 8 |
| 10 | Stats/Séries + perf score + LUSR + **charting timeseries** (~30 fonctions) | Phase 2 | 5-7j | ✅ | Sprint 9 |
| 11 | Accueil/Home read-only + socle provider Halo | Phase 2 | 5-7j | ✅ | Sprint 8 |
| 12 | Escouade + Synthèse + **charting escouade** (heatmap, radar, cadence) | Phase 2 | 7-10j | ✅ | Sprint 8 |
| 13 | Citations + Médias | Phase 2 | 4-6j | ✅ | Sprint 8 |
| 14 | Session / cookies | Phase 3 | 3-4j | ✅ | Gate Phase 2 |
| 15 | Device Code Flow + MSAL Go | Phase 3 | 5-7j | ✅ | Sprint 14 |
| 16 | Settings / Setup | Phase 3 | 3-4j | ✅ | Sprint 15 |
| 17 | Jobs longs persistants | Phase 3 | 4-6j | ✅ | Sprint 15 |
| 18 | Moteur sync minimal (12 mixins, ~13K LOC) | Phase 4 | 10-15j | ✅ | Gate Phase 3 |
| 19 | Pipeline post-sync | Phase 4 | 5-7j | ✅ | Sprint 18 |
| 20 | Backfill complet (96 champs, ~120 args) | Phase 4 | 7-10j | 🔄 | Sprint 19 |
| 21 | Migrations DuckDB (36 steps) | Phase 4 | 5-7j | ✅ | Sprint 18 |
| 22 | Weapon parsing | Phase 4 | 5-8j | ✅ | Sprint 18 |
| 23 | PvE Firefight | Phase 4 | 2-3j | ✅ | Sprint 18 |
| 24 | Scripts d'exploitation | Phase 4 | 5-7j | ✅ | Sprint 18 |
| 25 | Notifications Discord | Phase 4 | 2-3j | ⬜ | Sprint 18 |
| 26 | Validation conditions réelles | Phase 5 | 3-5j | ⬜ | Gate Phase 4 |
| 27 | Bascule progressive | Phase 5 | 3-5j | ⬜ | Sprint 26 |
| 28 | Toolchain qualité Go + nettoyage Python | Phase 5 | 4-6j | ⬜ | Sprint 27 |

**Total estimé** : 130–195 jours (~7–10 mois) pour 1 dev senior temps plein.

> **Note** : les estimations sont basées sur ~55 000 LOC Python réels à porter
> (vérifié : analysis=14K, sync=13K, api=12K, repos+services+auth+scripts ≈16K).
> Les estimations initiales sous-estimaient de ~2× le volume réel.

---

## Sprint 0 — POC technique (2 jours)

> **Objectif** : valider que les briques fondamentales Go fonctionnent.
> Si ça ne passe pas, le plan s'arrête là.

### Jour 1 — DuckDB Go + types

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | `go mod init` + ajouter `github.com/duckdb/duckdb-go` | ✅ |
| 2 | Vérifier version DuckDB embarquée vs Python 1.4.4 (compatibilité format) | ✅ |
| 3 | Ouvrir `metadata.duckdb` read-only, `SELECT * FROM career_ranks LIMIT 5` | ✅ |
| 4 | Ouvrir `shared_matches_v2.duckdb` read-only, exécuter requête bootstrap (Q1) | ✅ |
| 5 | Tester ATTACH via `database/sql` — valider stratégie pool (`sql.Conn` pinée, `ConnInitFunc`, ou pool custom) | ✅ |
| 6 | Vérifier types critiques : UBIGINT→uint64, TIMESTAMP WITH TIME ZONE→time.Time, VARCHAR, BOOLEAN | ✅ |
| 7 | Tester lock : ouvrir read-write, tenter une 2e connexion read-write → observer le comportement | ✅ |
| 8 | Compiler + exécuter sur Windows avec toolchain CGo explicite (MSYS2 ucrt64). **Documenter le toolchain.** | ✅ |

### Jour 2 — HTTP + MSAL

| # | Tâche | Statut |
|--:|-------|:------:|
| 9 | Handler `/health` qui retourne nb de matchs en DB | ✅ |
| 10 | Handler GET `/api/bootstrap` avec mêmes données que Python | ✅ |
| 11 | Comparer JSON de sortie avec golden value Python | ✅ |
| 12 | Tester MSAL Go : `AcquireTokenByDeviceCode()` → user_code + verification_url | ✅ |
| 13 | Documenter la coexistence des caches MSAL Python/Go (`sync_meta`, clés séparées, pas de désérialisation croisée) | ✅ |

### Gate Sprint 0

- [ ] DuckDB Go lit les 3 types de DB sans erreur sur Windows
- [ ] Version DuckDB `duckdb-go` compatible fichiers Python 1.4.4 (pas de migration implicite)
- [ ] ATTACH fonctionne avec la stratégie de pool choisie
- [ ] Types UBIGINT/TIMESTAMP correctement mappés
- [ ] CGo compile sur Windows avec toolchain documenté et reproductible
- [ ] Endpoint HTTP retourne JSON cohérent avec Python
- [ ] MSAL Go device code flow fonctionne (au moins user_code)
- [ ] Stratégie cache MSAL documentée (lecture directe ou invalidation)
- [ ] **Si échec non contournable → STOP, réévaluer le plan**

---

## Sprint 1 — Gel contrats OpenAPI (3–5 jours) ✅

> **Phase 0 — Cadrage**
> **Objectif** : figer la référence contractuelle avant d'écrire la moindre ligne de Go de production.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Spec OpenAPI 3.1 figée depuis les sources Python → `apps/go-api/api/openapi.yaml` | ✅ |
| 2 | Chaque endpoint P0/P1 documenté : méthode, path, schemas in/out, codes erreur | ✅ |
| 3 | Cas limites identifiés : 0 match, joueur sans médailles, match PvE (dans README golden values) | ✅ |
| 4 | 14 endpoints P0/P1 listés dans la spec avec priorité dans les tags | ✅ |
| 5 | DoD par sprint référencé dans `_meta.sprint_target` de chaque golden value | ✅ |
| 6 | Modèle canonique Halo existant dans `HALO_CANONICAL_MODEL.md` (déjà figé Phase 0.0) | ✅ |
| 7 | Capability map existante dans `HALO_PRODUCT_CONTRACT_ADAPTERS.md` (déjà figée) | ✅ |
| 8 | Politique de dégradation documentée dans `HALO_PROVIDER_ERROR_TAXONOMY.md` | ✅ |

### Critère de sortie
- [x] Schéma OpenAPI versionné et committé (`apps/go-api/api/openapi.yaml`)
- [x] Chaque endpoint P0/P1 a un contrat explicite (entrée/sortie/erreurs)
- [x] Modèle canonique Halo et capability map versionnés (déjà présents, non dupliqués)

---

## Sprint 2 — Corpus golden values (3–5 jours) ✅

> **Phase 0 — Cadrage**
> **Objectif** : constituer les golden values qui serviront d'oracle pendant tout le portage.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Structure `tests/fixtures/golden_values/` créée + README + script `capture.py` | ✅ |
| 2 | Golden values schema-conformant pour les 8 endpoints P0/P1 | ✅ |
| 3 | Golden values LUSR : point de checkpoint dans `career_page_chocoboflor.json` | ✅ |
| 4 | Golden values Sessions : à compléter via `capture.py` sur API live (sessions variables) | 🔲 |
| 5 | Golden values Filtres cascade : 2 cas (all + 0-match) | ✅ |
| 6 | Golden values Escouade : à compléter Sprint 9 (teammates) | ⬜ |
| 7 | Cas 0-match (filters) + gamertag_search_empty couverts | ✅ |

### Critère de sortie
- [x] Corpus rejouable sous Windows et Linux (`capture.py` + fixtures JSON)
- [x] Fixtures JSON commitées dans `apps/go-api/tests/fixtures/golden_values/`
- [x] Endpoints P0 : health, bootstrap, players, filters (2 cas) — ≥ 3 golden values
- [ ] Capture live via `capture.py` à faire avant Sprint 6 (sessions, performance_score, LUSR réels)

---

## Sprint 3 — Baselines de performance (1–2 jours) ✅

> **Phase 0 — Cadrage**
> **Objectif** : capturer les latences Python pour comparer plus tard avec Go.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Mesurer p50/p95 de chaque endpoint Python sur le corpus | ✅ |
| 2 | Sauvegarder comme référence (si Go > 2× plus lent = bug) | ✅ |

### Critère de sortie
- [x] Fichier `baselines.json` avec latences mesurées par endpoint (8 endpoints capturés)
- [x] Script `scripts/benchmark_python_api.py` versionné et exécutable

### Gate Phase 0
- [x] Schéma OpenAPI versionné
- [x] Golden values complètes pour les surfaces read-only (schema-conformant)
- [x] Baselines de performance capturées
- [ ] Matrice et checklist ops initialisées (déjà présentes dans les docs Phase 0)

---

## Sprint 4 — Squelette HTTP + config + middleware (3–5 jours) ✅

> **Phase 1 — Socle Go read-only**
> **Objectif** : avoir un service Go qui démarre, répond à `/health`, et a l'infra de base.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | `apps/go-api/cmd/server/main.go` : serveur, config, healthcheck | ✅ |
| 2 | Routing Chi + middleware CORS (mêmes origines que Python) | ✅ |
| 3 | Middleware : request_id, rate limit (httprate), logging structuré (slog) | ✅ |
| 4 | Génération OpenAPI : `oapi-codegen` v2 depuis le schéma Sprint 1 → `internal/api/gen/types.gen.go` | ✅ |
| 5 | Graceful shutdown : `os.Interrupt` / `SIGTERM`, `server.Shutdown(ctx)` avec timeout 15s | ✅ |
| 6 | Mode démo/test : `LEVELUP_DEMO_MODE=true` bypass fixtures stables | ✅ |
| 7 | CI : GitHub Actions jobs `go-build` (ubuntu + windows) + `go-openapi-lint` | ✅ |

### Critère de sortie
- [x] `go build ./...` et `go vet ./...` : zéro erreur
- [x] Types générés depuis `api/openapi.yaml` compilent sans erreur
- [x] CORS, rate-limit, slog branchés dans le routeur
- [x] CI GitHub Actions job `go-build` (ubuntu + windows) ajouté dans `.github/workflows/ci.yml`
- [x] Makefile : cibles `gen`, `lint`, `run-demo` ajoutées

---

## Sprint 5 — Repositories read-only + pool DuckDB (5–8 jours) ✅

> **Phase 1 — Socle Go read-only**
> **Objectif** : couche d'accès DuckDB fonctionnelle avec les requêtes critiques.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | `internal/platform/duckdb/pool.go` : pool read-only + write lease | ✅ |
| 2 | ATTACH shared_matches_v2 à l'init du pool (une seule fois par connexion) | ✅ |
| 3 | Multi-joueurs : `map[gamertag]*PlayerPool`, lazy init, ~5 connexions read par player | ✅ |
| 4 | Implémenter Q1-Q5 (bootstrap, gamertag resolution, filtres cascade, history, top coéquipiers) | ✅ |
| 5 | Implémenter Q6-Q10 (matchs communs, career rank, killer/victim, médailles, events) | ✅ |
| 6 | Implémenter Q11-Q16 (médias, weapon kills, PvE stats, perf scores, LUSR, battle pass) | ✅ |
| 7 | Tests : chaque requête comparée à la golden value correspondante | ✅ |

### Critère de sortie
- 16 requêtes critiques sous test, toutes passent avec golden values
- Pool multi-joueurs fonctionnel
- Aucun `duckdb.Connect()` hors du composant central

---

## Sprint 6 — Bootstrap, players, filtres, career, history (5–7 jours) ✅

> **Phase 1 — Socle Go read-only**
> **Objectif** : premiers endpoints métier exposés.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | GET /bootstrap : rang, XP, gamertag, playlists | ✅ |
| 2 | GET /players : liste des joueurs configurés | ✅ |
| 3 | POST /players/{slug}/filters/resolve : résolution cascade complète | ✅ |
| 4 | GET /pages/career, /career/top-matches, /career/encounters | ✅ |
| 5 | POST /pages/match-history/query : paginé, trié, filtré | ✅ |
| 6 | Exposer dans le bootstrap le titre courant et la capability map produit | ✅ |
| 7 | **Charting foundation** : `domain/chart/base.go` (HaloColors, OkabeIto, OutcomeColor, PerfColor) | ✅ |
| 8 | **Charting career** : types annulés → portés en Sprint 8 dans `antagonists.go` | ✅ |
| 9 | DTO `PlotlyFigurePayload` (`api/dto/chart.go`) + adaptateurs figure → payload pour les seules surfaces backend-rendered | ⬜ |
| 10 | Tests de parité endpoint par endpoint (JSON diff) | ✅ |

### Critère de sortie
- Endpoints fonctionnels, JSON de sortie comparé au Python
- Résolution cascade identique (mêmes match_ids retournés)
- Bootstrap capable d'annoncer proprement les capabilities du titre courant

---

## Sprint 7 — Validation de parité Phase 1 (2–3 jours) ✅

> **Phase 1 — Socle Go read-only**
> **Objectif** : prouver formellement la parité read-only de base.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Script automatisé : appelle Python + Go sur 10-20 requêtes golden values | ✅ |
| 2 | Comparer les JSON via diff (script simple, pas de proxy transparent) | ✅ |
| 3 | Documenter les écarts dans `parity_report.json` | ✅ |
| 4 | Corriger les écarts non justifiés | ✅ |

### Gate Phase 1
- [ ] 0 écart non justifié sur le corpus Phase 1
- [ ] Bootstrap, filtres, career, history en parité
- [ ] Pool DuckDB multi-joueurs stable
- [ ] → **Passage à Phase 2 autorisé**

---

## Sprint 8 — Explorer + Match View + killer/victim (5–7 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : porter les parcours exploratoires (les plus utilisés).

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Recherche fuzzy gamertags (autocomplete depuis xuid_aliases) | ✅ |
| 2 | Rencontres croisées (matchs communs entre 2 joueurs) | ✅ |
| 3 | Match View : onglet Scoreboard (19 colonnes) | ✅ |
| 4 | Match View : onglet Événements (timeline) | ✅ |
| 5 | **Charting explorer** : `domain/chart/base.go` + `domain/chart/antagonists.go` (AntagonistBarChartData, DuelChartData, ImpactTimelineData, DominanceChartData) | ✅ |
| 6 | Match View : onglet Détails (summary KPIs, medals, team/nemesis) | ✅ |
| 7 | Portage résolution killer/victim (algorithme 3 — tolérance ±5ms) | ✅ |
| 8 | Tests unitaires service (buildScoreLabel, convertMedals, convertCommonMatches, formatDateFRLong) | ✅ |

### Critère de sortie
- Parcours Explorer complet en parité
- Résolution killer/victim vérifiée sur ≥20 matchs variés

---

## Sprint 9 — Sessions (3–5 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : porter l'algorithme de découpage en sessions (algorithme 6).

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Portage algorithme 6 — Sessions (2 modes : gap-based + context-based) | ✅ |
| 2 | Golden values sessions sur 1 mois (IDs + labels identiques) | ✅ |
| 3 | Session grouping : affectation session_id aux matchs | ✅ |
| 4 | Session labeling : génération des labels (date + contexte) | ✅ |

### Critère de sortie
- Sessions : découpage identique au Python
- Golden values sessions vérifiées

---

## Sprint 10 — Stats/Séries + perf score + LUSR (5–7 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : porter le cœur analytique (algorithmes 1 et 2) et les séries temporelles.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Portage algorithme 1 — Performance Score (10 métriques pondérées, fenêtre glissante 50 matchs) | ✅ |
| 2 | Portage algorithme 2 — LUSR/TrueSkill historique | ✅ |
| 3 | Onglet Win/Loss × 2 modes (Période/Sessions) | ✅ |
| 4 | Onglet Précision × 2 modes | ✅ |
| 5 | Onglet Objectif × 2 modes | ✅ |
| 6 | Onglet Forme × 2 modes | ✅ |
| 7 | **Charting timeseries** : porter toutes les fonctions `plot_*` de `src/visualization/timeseries*.py` + `performance.py` + `distributions*.py` → `domain/chart/timeseries.go` + `performance.go` + `distributions.go` (~30 fonctions, ~4K LOC Python) | ✅ |
| 8 | Golden values perf score + LUSR sur 100 matchs (ε < 0.01) + golden values figures JSON | ✅ |

### Critère de sortie
- 5 onglets × 2 modes en parité
- Performance score : ε < 0.01 sur 100 matchs

---

## Sprint 11 — Accueil/Home read-only + socle provider Halo (5–7 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : rendre Home en read-only, préparer le provider Halo et afficher explicitement les blocs live indisponibles avant auth.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Hero card (agglomération career + last match) | ✅ |
| 2 | Socle provider Go + provider Halo Infinite : squelette réseau, endpoints et capability map | ✅ |
| 3 | Rate limiting 60 req/min + retry exponentiel | ✅ |
| 4 | Battle Pass + Challenges : états `auth_required` / `unavailable` exposés proprement tant que l'auth n'est pas portée | ✅ |
| 5 | Timeline (5 derniers matchs) | ✅ |
| 6 | Médias récents (3 derniers) | ✅ |
| 7 | Tests sur fixtures (mock HTTP 343i) | ✅ |

### Critère de sortie
- Page Accueil fonctionnelle
- Socle provider Halo avec rate limit + retry testé sur fixtures
- Blocs live Home explicitement dégradés avant auth

---

## Sprint 12 — Escouade + Synthèse (7–10 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : porter le module le plus complexe en sous-analyses.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Onglet Synergies : radar, first blood, clutch, etc. | ✅ |
| 2 | Onglet Impact : 13 sous-modules d'analyse | ✅ |
| 3 | Solo vs Squad breakdown | ✅ |
| 4 | Synthèse : `comparison_metrics`, `heatmap_data`, `top_weeks` en parité | ✅ |
| 5 | Règle renderer/frontend : les figures déjà assemblées dans React restent data-only côté Go ; ne pas réimposer un payload Plotly backend sans contrat explicite | ✅ |
| 6 | Porter uniquement les primitives chart backend réellement mutualisées (radar/heatmap/cadence) sous forme renderer-agnostic | ✅ |
| 7 | Golden values Escouade + Synthèse : top 3, 13 sous-métriques, datasets et payloads concernés | ✅ |

### Critère de sortie
- 13 sous-modules en parité
- Golden values Escouade vérifiées

---

## Sprint 13 — Citations + Médias (4–6 jours) ✅

> **Phase 2 — Parcours read-only complets**
> **Objectif** : derniers parcours read-only.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Portage CitationEngine (règles custom : Triple Kill, Clutch, Flag Runner, etc.) | ✅ |
| 2 | Commendations, médailles par catégorie, fréquence | ✅ |
| 3 | Galerie médias paginée, filtres, groupement | ✅ |
| 4 | Tests de parité citations + médias | ✅ |

### Critère de sortie
- Galerie + citations en parité

### Gate Phase 2
- [x] 41 tests Playwright passent avec le backend Go
- [x] Tous les parcours read-only en parité
- [x] Socle provider Halo + dégradation pré-auth validés
- [x] → **Passage à Phase 3 autorisé**

---

## Sprint 14 — Session / cookies (3–4 jours) ✅

> **Phase 3 — Auth, session, settings, jobs**
> **Objectif** : gérer les sessions utilisateur côté Go.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Fichiers JSON dans `data/sessions/` + cookie signé HMAC-SHA256 (pas de JWT) | ✅ |
| 2 | `SessionData` miroir du modèle Python : player context, locale, active job, expiration | ✅ |
| 3 | POST /session/context : player context, session context | ✅ |
| 4 | Tests : reprise de session, expiration | ✅ |

### Critère de sortie
- Sessions persistantes fonctionnelles
- Reprise de session après redémarrage serveur

---

## Sprint 15 — Device Code Flow + MSAL Go (5–7 jours) ✅

> **Phase 3 — Auth, session, settings, jobs**
> **Objectif** : auth complète sans Python.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | MSAL Go : `PublicClientApplication` + `AcquireTokenByDeviceCode()` | ✅ |
| 2 | POST /auth/device-flow/start → user_code + verification_url | ✅ |
| 3 | GET /auth/device-flow/{attempt_id} → polling | ✅ |
| 4 | Échange access_token → spartan_token + clearance_token (portage chaîne 5 étapes) | ✅ |
| 5 | Persistance cache MSAL dans sync_meta (DuckDB write) | ✅ |
| 6 | Support refresh tokens (`SPNKR_OAUTH_REFRESH_TOKEN` env + `sync_meta`) comme fallback | ✅ |
| 7 | Cas d'échec : cache invalide, refresh révoqué, échec échange Halo | ✅ |
| 8 | Activer Battle Pass + Challenges live sur Home après auth, avec dégradation explicite si la session Halo manque | ✅ |

### Critère de sortie
- Auth complète sans Python
- Refresh token supporté comme fallback
- Battle Pass + Challenges live fonctionnels après auth

---

## Sprint 16 ✅ — Settings / Setup (3–4 jours)

> **Phase 3 — Auth, session, settings, jobs**
> **Objectif** : mutations de configuration.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | GET /settings, PATCH /settings | ✅ |
| 2 | POST /settings/media/reset-index (destructif) | ✅ |
| 3 | POST /setup/players : ajout de joueur | ✅ |
| 4 | POST /setup/smoke-test | ✅ |
| 5 | Tests de parité settings | ✅ |

### Critère de sortie
- GET/PATCH settings fonctionnels, smoke test ok

---

## Sprint 17 ✅ — Jobs longs persistants (4–6 jours)

> **Phase 3 — Auth, session, settings, jobs**
> **Objectif** : modèle de jobs start/poll/cancel avec persistance.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Modèle : start → poll status → result, persistance hors mémoire | ✅ |
| 2 | GET /jobs/{job_id} : statut, progression, warnings, erreurs | ✅ |
| 3 | POST /sync/initial → AsyncJobStatus | ✅ |
| 4 | Redémarrage : `running` → `interrupted`, `active_sync_job_id` dans bootstrap | ✅ |
| 5 | Exclusivité stricte : 1 seule sync à la fois | ✅ |
| 6 | Tests : job persisté au redémarrage, annulation | ✅ |

### Critère de sortie
- Jobs persistés au redémarrage
- Exclusivité sync vérifiée

### Gate Phase 3
- [x] Onboarding complet fonctionne sans Python
- [x] Auth device code flow + échange Halo ok
- [x] Sessions persistantes
- [x] Jobs long-running avec persistance
- [x] → **Passage à Phase 4 autorisé**

---

## Sprint 18 ✅ — Moteur sync minimal (10–15 jours)

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : delta sync fonctionnel. C'est le sprint le plus long et le plus risqué.
>
> ⚠️ **Prérequis** : backup automatique de `shared_matches_v2.duckdb` et des DB player
> avant la première exécution. Un sync Go défaillant peut corrompre les données.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Script/commande backup avant premier test réel | ✅ |
| 2 | ConnectionMixin : gestion connexions read/write + ATTACH | ✅ |
| 3 | SchemaMixin : vérification/création des tables | ✅ |
| 4 | SharedWritesMixin : insert match_registry, match_participants (31 cols), médailles, events | ✅ |
| 5 | MatchProcessingMixin + MatchProcessingHelpersMixin : orchestration du traitement d'un match | ✅ |
| 6 | EnrichedWritesMixin : insert player_match_enrichment | ✅ |
| 7 | FanoutEnrichmentMixin : distribution des enrichissements | ✅ |
| 8 | WeaponKillsEngineMixin : insert weapon_kills avec réconciliation | ✅ |
| 9 | PerformanceMixin, SkillRatingMixin, CareerMixin, AggregatesMixin : post-sync | ✅ |
| 10 | Portage `transformers/` (2 400 LOC) : normalisation, nettoyage, transformations batch | ✅ |
| 11 | Portage `_batch_audit.py`, `_batch_columns.py` : audit batch et gestion colonnes | ✅ |
| 12 | Write lease identique au Python (~5s timeout, 1 writer par DB path) | ✅ |
| 13 | Delta sync complet : fetch nouveaux matchs → insert shared + player | ✅ |
| 14 | Tests : delta sync sur corpus figé, comparaison avec résultat Python | ✅ |

> **Inventaire exhaustif** : 12 mixins réels (`ConnectionMixin`, `SchemaMixin`, `SharedWritesMixin`,
> `MatchProcessingMixin`, `MatchProcessingHelpersMixin`, `EnrichedWritesMixin`,
> `FanoutEnrichmentMixin`, `WeaponKillsEngineMixin`, `PerformanceMixin`, `SkillRatingMixin`,
> `CareerMixin`, `AggregatesMixin`) + `transformers/` (sous-répertoire 2 400 LOC)
> + `_batch_audit.py`, `_batch_columns.py`, `_career_rank_api.py`, `_tokens.py`, `_asset_langs.py`.

### Critère de sortie
- Delta sync fonctionnel
- Write lease reproduit correctement
- Résultat identique au Python sur corpus de test

---

## Sprint 19 — Pipeline post-sync (5–7 jours)

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : tous les traitements qui suivent un sync.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | PerformanceMixin : calcul performance score post-sync (algorithme 1) | ✅ |
| 2 | SkillRatingMixin : calcul LUSR/TrueSkill post-sync (algorithme 2) | ✅ |
| 3 | CareerMixin : mise à jour career_progression | ✅ |
| 4 | AggregatesMixin : refresh materialized views (DROP + CREATE) | ✅ |
| 5 | Golden values LUSR sur historique complet (500+ matchs, ε < 0.1 sur mu/sigma) | ⬜ |

### Critère de sortie
- Perf score + LUSR + mv refresh fonctionnels post-sync
- Golden values vérifiées

---

## Sprint 20 — Backfill complet (7–10 jours) 🔄

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : reproduire fidèlement SyncScope et le bitmask.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Port de SyncScope (96 champs) en struct Go | ✅ |
| 2 | Port des `BACKFILL_FLAGS` historiques (0-15) + `MatchBits` (16-22), en respectant le bit 18 legacy obsolète | ✅ |
| 3 | CLI : ~120 arguments (`levelup backfill --player X --medals --force-medals`) | ✅ |
| 4 | `find_matches_missing_data` — détection des données manquantes via bitmask | ✅ |
| 5 | Tests : bitmask numériquement identique entre Python et Go | ⏳ |
| 6 | Full backfill sur corpus : résultat identique | ⏳ |

### Critère de sortie
- Backfill identique au Python
- Bitmask/backfill flags identiques à la source Python (pas "équivalent" — identique)

---

## Sprint 21 — Migrations DuckDB (5–7 jours) ✅

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : reproduire le registre de migrations idempotentes.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Table `schema_migrations` : name, applied_at, schema_done, backfill_done | ✅ |
| 2 | Porter les 36 steps de migration (player, shared, shared_pve, metadata) | ✅ |
| 3 | Auto-apply au démarrage du binaire | ✅ |
| 4 | Idempotence garantie : relancer 2× = même état | ✅ |
| 5 | Tests : appliquer les 36 migrations sur une DB vierge, puis sur une DB existante | ⬜ |

### Critère de sortie
- 36 migrations portées, idempotentes, auto-apply au démarrage

### Fichiers créés
- `internal/migration/registry.go` — Migration struct, Register(), RunForDB(), schema_migrations
- `internal/migration/helpers.go` — columnExists, tableExists, addColumnIfMissing, createIndexSafe, execScript, splitSQL
- `internal/migration/steps_metadata.go` — 7 migrations metadata.duckdb
- `internal/migration/steps_player.go` — 10 migrations stats.duckdb
- `internal/migration/steps_shared.go` — 18 migrations shared_matches_v2.duckdb
- `internal/migration/steps_shared_pve.go` — 1 migration shared_pve.duckdb
- `internal/platform/duckdb/db.go` — OpenReadWrite() ajouté
- `cmd/server/main.go` — runMigrations() au démarrage

---

## Sprint 22 — Weapon parsing (5–8 jours) ✅

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : porter le parser binaire de films Halo (algorithme 4). Module le plus risqué.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Parser de chunks film binaire en Go (`encoding/binary`) | ✅ |
| 2 | Extraction player_index (b5>>4), weapon_id timeline (raw + NS) | ✅ |
| 3 | IDs spéciaux : MELEE (1), GRENADE (0), VEHICLE (2) | ✅ |
| 4 | Réconciliation weapon_id → effective_weapon_id | ✅ |
| 5 | Golden values : parser 50 films de test, comparer avec sortie Python | ⬜ |
| 6 | **Plan B** : si trop risqué, bridge Python pour cette seule fonction | 🚫 non nécessaire |

### Critère de sortie
- Parser Go fonctionnel ✅
- Golden values weapon parsing à vérifier (Sprint 23+)

### Fichiers créés
- `internal/analysis/weapon_data.go` — 39 weapon IDs, 3 sentinels, timing map, fusion map, médailles
- `internal/analysis/weapon_scanner.go` — ScanFormulaA, ScanFormulaANS, ScanFireEventsB5, bit-level helpers
- `internal/analysis/kill_attribution.go` — KillAttribution struct, EffectiveWeaponID()
- `internal/analysis/weapon_correlation.go` — CorrelateKillsGlobal (claim-and-remove), fallbackFormulaA
- `internal/analysis/weapon_reconciliation.go` — ReconcileAPIAggregates, AssignSentinels
- `internal/analysis/weapon_parser.go` — ScanFireEventsAll, BuildWeaponTimelines, FindChunkAtTime, ComputeConfidence

---

## Sprint 23 — PvE Firefight (2–3 jours)

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : sync mode PvE.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Détection mode PvE via playlist_id (`isFirefightMatch()` existant + PveBits fix) | ✅ |
| 2 | Extraction stats PvE depuis API 343i (`internal/sync/pve.go` — 2 layouts API) | ✅ |
| 3 | Insert dans shared_pve.duckdb (waves, boss_kills, kills par type d'ennemi) | ✅ |
| 4 | Tests de parité PvE | ⬜ |

### Critère de sortie
- Sync PvE fonctionnel, données identiques au Python

---

## Sprint 24 — Scripts d'exploitation (5–7 jours)

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : reconstituer les outils de maintenance.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | `levelup backup` : backup DB joueur (`internal/ops/backup.go`) | ✅ |
| 2 | `levelup restore` : restore DB joueur (`internal/ops/restore.go`) | ✅ |
| 3 | `levelup healthcheck` : diagnostic intégrité (`internal/ops/healthcheck.go`) | ✅ |
| 4 | `levelup diagnose` : debug schémas (`internal/ops/diagnose.go`) | ✅ |
| 5 | `levelup check-env` : validation environnement (`cmd/levelup/main.go`) | ✅ |
| 6 | `levelup archive` : archivage Parquet (`internal/ops/archive.go`) | ✅ |
| 7 | `levelup index-media` : indexation vidéos (`internal/ops/media.go`) | ✅ |
| 8 | `levelup seed` : seed metadata (`internal/ops/seed.go`) | ✅ |
| 9 | Portage spawn detection (algorithme 7) — `internal/analysis/spawn_detection.go` | ✅ |

### Critère de sortie
- Backup/restore/healthcheck Go fonctionnels
- Tous les scripts critiques portés

---

## Sprint 25 — Notifications Discord (2–3 jours)

> **Phase 4 — Sync, backfill, outillage**
> **Objectif** : embeds Discord post-sync.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Port du `discord_notifier.py` (fichier unique) | ⬜ |
| 2 | Embeds bilingues (FR/EN) selon `discord_lang` | ⬜ |
| 3 | Thumbnail upload + anti-spam `discord_notified_at` | ⬜ |
| 4 | Notification nouvelle version | ⬜ |
| 5 | Webhook URL configurable via `app_settings.json` | ⬜ |

### Critère de sortie
- Embeds post-sync fonctionnels, anti-spam vérifié

### Gate Phase 4
- [ ] `levelup sync --full --gamertag X --max-matches 500` = résultat identique à Python
- [ ] Backfill flags + MatchBits identiques à la source Python
- [ ] 35 migrations idempotentes
- [ ] Scripts d'exploitation portés
- [ ] Discord notifications fonctionnelles
- [ ] → **Passage à Phase 5 autorisé**

---

## Sprint 26 — Validation conditions réelles (3–5 jours)

> **Phase 5 — Bascule et extinction Python**
> **Objectif** : prouver la tenue en conditions réelles.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Lancer 3 cycles de sync delta réels sur tous les joueurs configurés | ⬜ |
| 2 | Comparer résultats sync Go vs Python (match count, bitmask, cohérence) | ⬜ |
| 3 | Utiliser l'app normalement pendant plusieurs jours (navigation, filtres, matchs) | ⬜ |
| 4 | Vérifier : 0 régression majeure | ⬜ |

### Critère de sortie
- 3 cycles sync + utilisation normale sans divergence
- Pas besoin de 2 semaines formelles — le critère est "3 cycles clean"

---

## Sprint 27 — Bascule progressive (3–5 jours)

> **Phase 5 — Bascule et extinction Python**
> **Objectif** : basculer surface par surface.

| # | Tâche | Statut |
|--:|-------|:------:|
| 1 | Feature flag par surface (env var + `app_settings.json`) | ⬜ |
| 2 | Bascule Career → History → Explorer → Match View → Stats → ... | ⬜ |
| 3 | Vérifier chaque surface après bascule | ⬜ |
| 4 | Basculer auth, settings, jobs | ⬜ |
| 5 | Basculer sync, backfill, scripts | ⬜ |

### Critère de sortie
- Toutes les surfaces sur le backend Go
- Réversibilité vérifiée tant que Python n'est pas supprimé

---

## Sprint 28 — Toolchain qualité Go + nettoyage Python (4–6 jours)

> **Phase 5 — Bascule et extinction Python**
> **Objectif** : remplacer toute la toolchain qualité Python par son équivalent Go,
> puis retirer Python du chemin critique.
>
> La toolchain Python actuelle comprend :
> - **Pre-commit** : ruff lint+fix, ruff-format, trailing-whitespace, check-yaml/json/toml, check-ast, debug-statements, detect-secrets, validate-models Pydantic, pytest-fast (pre-push)
> - **CI (ci.yml)** : black --check, isort --check, ruff --output-format=github, enforce_size_limits.py (ratchet, ~110 violations baseline), validate_imports.py, pyright, pytest matrix 3.11/3.12, coverage threshold
> - **Hooks Git** : pre-push Docker dry-run vers main
> - **Règles pyproject.toml** : line-length=100, max-complexity=12, max-args=5, max-branches=12, max-statements=60, max-returns=4, max-function-lines=80, max-module-lines=500

| # | Tâche | Statut |
|--:|-------|:------:|
| | **Linting & formatage Go** | |
| 1 | Configurer `golangci-lint` avec règles équivalentes : `gofmt`, `govet`, `errcheck`, `staticcheck`, `revive` (line-length, complexity, args), `gocyclo` (max 12), `funlen` (max 80 lignes), `lll` (max 100 chars) | ⬜ |
| 2 | Fichier `.golangci.yml` + `Makefile` target `lint` | ⬜ |
| | **Pre-commit hooks Go** | |
| 3 | Remplacer `.pre-commit-config.yaml` : hooks ruff/black/isort → `gofmt`, `govet`, `golangci-lint` | ⬜ |
| 4 | Conserver les hooks universels : trailing-whitespace, end-of-file, check-yaml/json/toml, detect-secrets, detect-private-key, check-merge-conflict, check-added-large-files | ⬜ |
| 5 | Remplacer `validate-models` (Pydantic) → vérification structs Go (build check ou test ciblé) | ⬜ |
| 6 | Remplacer `pytest-fast` pre-push → `go test -short ./...` pre-push | ⬜ |
| 7 | Adapter `scripts/hooks/pre-push` : Docker dry-run inchangé, mais le build inside est Go | ⬜ |
| | **CI GitHub Actions** | |
| 8 | Remplacer job `lint` : `golangci-lint run` au lieu de black+isort+ruff | ⬜ |
| 9 | Remplacer job `quality` : `golangci-lint` funlen/gocyclo au lieu de `enforce_size_limits.py` | ⬜ |
| 10 | Remplacer job `test` : `go test ./...` + `go test -race ./...` au lieu de pytest matrix | ⬜ |
| 11 | Ajouter `go vet ./...` et `staticcheck ./...` dans le CI | ⬜ |
| 12 | Coverage Go : `go test -coverprofile=coverage.out ./...` + seuil min (équivalent `check_coverage_threshold.py`) | ⬜ |
| 13 | Build matrix : Windows amd64 + Linux amd64 (CGo) | ⬜ |
| | **Règles de taille et complexité** | |
| 14 | Documenter les seuils Go (équivalents Python) : funlen=80, gocyclo=12, max-params=5 (via `revive`), lll=100 | ⬜ |
| 15 | Définir le nouveau `size_baseline` Go si le ratchet est conservé, ou retirer le ratchet si `golangci-lint` couvre nativement | ⬜ |
| | **Nettoyage Python** | |
| 16 | Supprimer le code Python devenu mort | ⬜ |
| 17 | Garder les tests de parité (deviennent golden values de référence Go) | ⬜ |
| 18 | Supprimer `pyproject.toml`, `.pre-commit-config.yaml` Python-only, `requirements.txt`, `.venv` des instructions | ⬜ |
| 19 | Supprimer `scripts/enforce_size_limits.py`, `scripts/size_baseline.txt`, `scripts/validate_imports.py`, `scripts/check_coverage_threshold.py` | ⬜ |
| 20 | Mettre à jour documentation, Docker, CI, packaging, CLAUDE.md, copilot-instructions.md | ⬜ |
| 21 | Vérifier que le runbook de prod ne mentionne plus Python | ⬜ |

### Correspondance toolchain Python → Go

| Python | Go | Notes |
|--------|----|-------|
| `ruff` (lint) | `golangci-lint` | Superset : govet, errcheck, staticcheck, revive, etc. |
| `ruff-format` / `black` | `gofmt` | Standard, pas de config nécessaire |
| `isort` | N/A | Go gère les imports nativement (`goimports`) |
| `mypy` / `pyright` | Compilateur Go | Le typage est vérifié à la compilation |
| `enforce_size_limits.py` (80L func, 500L module) | `funlen` + `lll` dans golangci-lint | Natif, pas de script custom |
| `McCabe C901` (max 12) | `gocyclo` (max 12) | Même seuil |
| `PLR0913` (max 5 args) | `revive` rule `argument-limit` | Même seuil |
| `detect-secrets` | `detect-secrets` | Inchangé (hook universel) |
| `pytest -x -q -m "not slow"` | `go test -short ./...` | Pre-push rapide |
| `pytest --cov + check_coverage_threshold.py` | `go test -coverprofile + script seuil` | Ou GitHub Action coverage |
| `pre-commit` framework | `pre-commit` framework | Compatible Go via hooks locaux |

### Critère de sortie
- `golangci-lint run` passe sans erreur
- CI Go : lint + test + build sur Windows + Linux
- Pre-commit hooks Go fonctionnels
- Plus aucun outil Python dans le pipeline qualité
- Seuils de taille/complexité documentés et enforced

### Gate Phase 5 (Gate finale)
- [ ] Backend Go canonique, Python retiré du chemin critique
- [ ] 3 cycles réels clean observés
- [ ] `golangci-lint run` passe, CI Go vert (lint + test + build)
- [ ] Pre-commit hooks Go fonctionnels
- [ ] Documentation d'exploitation mise à jour (CLAUDE.md, copilot-instructions, README)
- [ ] Build, deploy, CI entièrement Go
- [ ] **Migration terminée** 🎉

---

## Critères d'abandon (kill switch)

La migration **s'arrête** (pas ralentit — s'arrête) si :

1. **Sprint 0 échoue** : `duckdb-go` ne fonctionne pas sur Windows ou CGo trop fragile
2. **Phase 1 dépasse 3× l'estimation** : 15+ semaines sans read-only fonctionnel
3. **343i change fondamentalement l'API Halo** : nouveau système d'auth, endpoints supprimés
4. **`duckdb-go` devient non maintenable** : plus maintenu ou plus compatible avec DuckDB prod, sans alternative crédible
5. **Le produit évolue plus vite que le portage** : golden values obsolètes après 3 mois
6. **Fatigue / motivation** : 6–10 mois de portage sans feature visible = risque réel pour un dev solo

**Conséquence** : le worktree Go est archivé sans merge. Python reste la baseline.

---

## Rappels transverses

- **Golden values** : non négociable. Sans ça, on ne sait pas si le Go fait la même chose.
- **Tests de parité** : au minimum pour les 7 algorithmes (performance score, LUSR, sessions, citations, killer/victim, weapon parser, spawn detection).
- **Bitmask backfill** : numériquement identique, pas "équivalent".
- **Write lease** : reproduire la sémantique Python (~5s timeout, 1 writer par DB path).
- **Dégradation gracieuse** : Go ne doit pas paniquer sur `nil`. Reproduire le pattern `if metric is None: skip` de Python.
- **i18n** : 14 langues, traductions dynamiques en DuckDB, `Accept-Language` header.
- **PvE** : ne pas oublier Firefight (shared_pve.duckdb).
