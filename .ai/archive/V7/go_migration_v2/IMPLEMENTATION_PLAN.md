# Plan d'implémentation — Post-audit consolidé

> Référence : `AUDIT_CONSOLIDE.md` (2026-04-16)
> Sprints 29–44 — Phases 6 à 9
>
> Ce plan détaille chaque sprint issu des recommandations P0–P4 et R0–R8 de l'audit.
> Les sprints 0–28 (Phases 0–5) sont dans `SPRINT_ROADMAP.md`.

---

## Table des matières

- [Vue d'ensemble](#vue-densemble)
- [Phase 6 — Réalignement contrat & sécurité](#phase-6--réalignement-contrat--sécurité)
  - [Sprint 29 — Assainissement surface + garde-fous CI](#sprint-29--assainissement-surface--garde-fous-ci)
  - [Sprint 30 — Bugs sécurité & error handling](#sprint-30--bugs-sécurité--error-handling)
  - [Sprint 31 — Onboarding Go & cookies session](#sprint-31--onboarding-go--cookies-session)
  - [Sprint 32 — Contrat API : Lots 1-3 (conformes + POST)](#sprint-32--contrat-api--lots-1-3-conformes--post)
  - [Sprint 33 — Contrat API : Lots 4-5 (réécriture + absents)](#sprint-33--contrat-api--lots-4-5-réécriture--absents)
- [Phase 7 — Infrastructure & bascule production](#phase-7--infrastructure--bascule-production)
  - [Sprint 34 — Infra release/deploy Go](#sprint-34--infra-releasedeploy-go)
  - [Sprint 35 — Golden tests CI + shadow mode](#sprint-35--golden-tests-ci--shadow-mode)
  - [Sprint 36 — Validation & bascule production](#sprint-36--validation--bascule-production)
- [Phase 8 — Qualité & dette technique (post-bascule)](#phase-8--qualité--dette-technique-post-bascule)
  - [Sprint 37 — Architecture handlers & injection](#sprint-37--architecture-handlers--injection)
  - [Sprint 38 — DRY + split fichiers >500L](#sprint-38--dry--split-fichiers-500l)
  - [Sprint 39 — Tests couches manquantes + couverture 50%](#sprint-39--tests-couches-manquantes--couverture-50)
  - [Sprint 40 — Observabilité & monitoring](#sprint-40--observabilité--monitoring)
- [Phase 9 — Évolutions fonctionnelles & UX](#phase-9--évolutions-fonctionnelles--ux)
  - [Sprint 41 — Scoreboard complet + weapon film parsing](#sprint-41--scoreboard-complet--weapon-film-parsing)
  - [Sprint 42 — Analyse UI avancée + fanout multi-joueur](#sprint-42--analyse-ui-avancée--fanout-multi-joueur)
  - [Sprint 43 — Améliorations UX produit](#sprint-43--améliorations-ux-produit)
  - [Sprint 44 — Implémentation multi-titres + ADR + polish final](#sprint-44--implémentation-multi-titres--adr--polish-final)
- [Estimation totale & risques](#estimation-totale--risques)

---

## Vue d'ensemble

| # | Sprint | Phase | Estimation | Statut | Dépendance | Réf. audit |
|--:|--------|-------|:----------:|:------:|------------|:----------:|
| 29 | Assainissement surface + garde-fous CI | Phase 6 | 5-8j | ⬜ | Sprint 28 | P0-1/P0-2/P0-3, R0/R1/R5 |
| 30 | Bugs sécurité & error handling | Phase 6 | 3-5j | ⬜ | Sprint 29 | P1-1→P1-7 |
| 31 | Onboarding Go & cookies session | Phase 6 | 3-4j | ⬜ | Sprint 29 | P0-4, §1.10 |
| 32 | Contrat API : Lots 1-3 | Phase 6 | 5-8j | ⬜ | Sprint 29 | P0-5/P0-6 (lots 1-3) |
| 33 | Contrat API : Lots 4-5 | Phase 6 | 5-8j | ⬜ | Sprint 32 | P0-5/P0-6 (lots 4-5) |
| 34 | Infra release/deploy Go | Phase 7 | 5-8j | ⬜ | Sprint 33 | P0-7/P0-8, R8 |
| 35 | Golden tests CI + shadow mode | Phase 7 | 4-6j | ⬜ | Sprint 34 | R2/R4/R7 |
| 36 | Validation & bascule production | Phase 7 | 3-5j | ⬜ | Sprint 35 | Gate bascule |
| 37 | Architecture handlers & injection | Phase 8 | 4-6j | ⬜ | Sprint 36 | P2-1/P2-7 |
| 38 | DRY + split fichiers >500L | Phase 8 | 4-6j | ⬜ | Sprint 37 | P2-2→P2-6/P2-8 |
| 39 | Tests couches manquantes + couverture 50% | Phase 8 | 4-6j | ⬜ | Sprint 37 | R3/R6 |
| 40 | Observabilité & monitoring | Phase 8 | 2-3j | ⬜ | Sprint 36 | R7 |
| 41 | Scoreboard + weapon film parsing + healthcheck | Phase 9 | 5-8j | ⬜ | Sprint 36 | P3-1/P3-3/P3-5 |
| 42 | Analyse UI avancée + fanout multi-joueur | Phase 9 | 5-8j | ⬜ | Sprint 41 | P3-2/P3-4 |
| 43 | Améliorations UX produit | Phase 9 | 5-8j | ⬜ | Sprint 36 | P4-1→P4-4 |
| 44 | Implémentation multi-titres + ADR + polish final | Phase 9 | 10-14j | ⬜ | Sprint 36 | P2-9/P3-6 |

**Total Phases 6-9** : ~72-111 jours (~3-5 mois) pour 1 dev senior temps plein.

### Diagramme de dépendances

```
Phase 6 — Réalignement contrat & sécurité (sem 1-6)
                                    ┌──── Sprint 30 (sécurité) ────┐
Sprint 29 (assainissement+CI) ──────┤                               │
                                    ├──── Sprint 31 (onboarding) ───┤
                                    │                               │
                                    └──── Sprint 32 (lots 1-3) ─────┤
                                                │                   │
                                          Sprint 33 (lots 4-5) ─────┘
                                                │
Phase 7 — Infrastructure & bascule (sem 6-10)   │
                                          Sprint 34 (infra) ────────┐
                                                │                   │
                                          Sprint 35 (golden+shadow) │
                                                │                   │
                                          Sprint 36 (BASCULE) ──────┘
                                                │
Phase 8 — Qualité (post-bascule)                │
                    ┌──── Sprint 37 (injection) ─┤──── Sprint 38 (DRY)
                    │                            │
                    ├──── Sprint 39 (tests)      │
                    │                            │
                    └──── Sprint 40 (observ.)    │
                                                 │
Phase 9 — Évolutions                             │
                    ┌──── Sprint 41 (scoreboard) ─┤──── Sprint 42 (analyse)
                    │                             │
                    ├──── Sprint 43 (UX) ─────────┤
                    │                             │
                    └──── Sprint 44 (ADR) ────────┘
```

---

## Phase 6 — Réalignement contrat & sécurité

> **Objectif** : assainir la surface d'API, brancher les garde-fous automatiques,
> corriger les bugs sécurité, puis réaligner méthodiquement chaque endpoint Go
> sur le contrat FastAPI. C'est le chemin critique vers la bascule.

---

### Sprint 29 — Assainissement surface + garde-fous CI (5–8 jours)

> **Phase 6 — Réalignement contrat**
> **Objectif** : partir d'une base propre — purger les artefacts contradictoires,
> figer la source de vérité contrat, brancher les premiers tests automatiques.
>
> **Réf. audit** : P0-1, P0-2, P0-3, R0, R1, R5

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Purge des artefacts morts (P0-1 / R0)** | | |
| 1 | Décider explicitement du sort de `GET /setup/status` : supprimer partout ou réintroduire volontairement. Documenter la décision. | ⬜ | P0-1 |
| 2 | Supprimer ou mettre à jour le hook `useSetupStatus()` dans `features/setup/queries.ts` selon la décision §1 | ⬜ | P0-1 |
| 3 | Nettoyer les handlers MSW obsolètes liés à `/setup/status` dans `apps/web/` | ⬜ | P0-1 |
| 4 | Nettoyer les specs Playwright qui référencent `/setup/status` dans `apps/web/e2e/` | ⬜ | P0-1 |
| 5 | Réaligner `generated.ts` avec les endpoints réellement consommés (retirer les types morts, annoter les Go-only) | ⬜ | P0-1 |
| 6 | Supprimer les 5 query keys non consommés dans `keys.ts` ou les annoter comme futurs/dormants | ⬜ | P0-1 |
| | **Source de vérité contrat (P0-2)** | | |
| 7 | Figer l'OpenAPI FastAPI (`apps/api/`) comme contrat de référence unique pour la bascule | ⬜ | P0-2 |
| 8 | Générer un export OpenAPI automatique depuis les schémas Pydantic FastAPI si ce n'est pas déjà fait | ⬜ | P0-2 |
| 9 | Diffusion automatique : script qui compare l'OpenAPI FastAPI vs l'OpenAPI Go et produit un rapport de parité (routes + méthodes + schemas) | ⬜ | P0-2 |
| 10 | Mettre à jour `apps/go-api/api/openapi.yaml` pour refléter le contrat FastAPI cible (routes, méthodes, DTOs) | ⬜ | P0-2 |
| | **Garde-fous CI (P0-3 / R1)** | | |
| 11 | Créer un test Go `contract_test.go` qui parse `openapi.yaml` et compare avec les routes enregistrées dans chi : chaque `path+method` OpenAPI → handler chi existant | ⬜ | R1 |
| 12 | Ajouter au `contract_test.go` : vérifier que chaque handler renvoie `Content-Type: application/json` (pas `text/plain`) via `httptest.NewRecorder` | ⬜ | R1 |
| 13 | Retirer `continue-on-error: true` du job `go-openapi-lint` dans `ci.yml` → rendre le lint OpenAPI bloquant | ⬜ | P0-3 |
| 14 | Ajouter le job `contract-test` dans `ci.yml` : exécute `contract_test.go` sur chaque PR | ⬜ | P0-3 |
| | **Playwright React en CI (R5)** | | |
| 15 | Créer un job `e2e-react` dans `ci.yml` ou un workflow dédié | ⬜ | R5 |
| 16 | Le job démarre `make dev` (API + Vite) en mode demo avec DuckDB fixtures | ⬜ | R5 |
| 17 | Exécuter les 15 specs Playwright existantes dans `apps/web/e2e/` avec Chromium headless | ⬜ | R5 |
| 18 | Configurer le timeout et le retry (1 retry pour les flaky) | ⬜ | R5 |

### Critère de sortie Sprint 29
- [ ] Plus aucun artefact mort autour de `/setup/status`
- [ ] OpenAPI FastAPI = source de vérité documentée
- [ ] `contract_test.go` vérifie routes + méthodes + Content-Type, intégré au CI
- [ ] Lint OpenAPI bloquant (plus de `continue-on-error`)
- [ ] 15 specs Playwright React tournent en CI (au minimum en `workflow_dispatch` la première semaine)
- [ ] `generated.ts` et `keys.ts` nettoyés

---

### Sprint 30 — Bugs sécurité & error handling ✅ (complété 2026-07-18)

> **Phase 6 — Réalignement contrat**
> **Objectif** : corriger tous les bugs sécurité et d'error handling identifiés par l'audit.
> Ces corrections sont parallélisables avec les sprints 31-32.
>
> **Réf. audit** : P1-1→P1-7, §2.3, §2.4
> **Commits** : `8b4d70f1`

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Fuite connexion pool (P1-1)** | | |
| 1 | Dans `pool.go`, corriger la fuite : `pdb.Shared.Close()` et `pdb.Metadata.Close()` dans le chemin doublon `PlayerDB` | ✅ | P1-1 |
| 2 | Implémenter `singleflight.Group` pour éviter les créations concurrentes de pools joueur | ✅ | P1-1 |
| 3 | Ajouter un test unitaire `pool_test.go` : ouvrir 2× le même joueur → vérifier qu'un seul pool est créé et que le 2e n'a pas de ressource pendante | ⬜ | P1-1 |
| | **SQL injection pattern (P1-2)** | | |
| 4 | Dans `backfill.go` L175+, remplacer `playerDoneGuard()` concaténation SQL par validation `isValidMatchID` | ✅ | P1-2 |
| 5 | Test unitaire : vérifier que `playerDoneGuard` génère une requête paramétrée correcte avec 0, 1 et 100+ IDs | ⬜ | P1-2 |
| | **Erreurs silencieuses MatchView (P1-3)** | | |
| 6 | Dans `match_view_service.go` L47-54, remplacer les 7 `_, _` par `slog.Warn` + dégradation gracieuse | ✅ | P1-3 |
| 7 | Décider pour chaque erreur : meta = bloquant (404), sub-data = dégradation gracieuse (onglet vide + slog.Warn) | ✅ | P1-3 |
| 8 | Test : injecter un mock repo qui retourne `error` → vérifier la réponse HTTP (status + JSON structuré) | ⬜ | P1-3 |
| | **CSRF middleware (P1-4)** | | |
| 9 | Implémenter un middleware CSRF Go : vérification `Origin`/`Referer` sur toutes les requêtes mutantes (POST, PATCH, DELETE) | ✅ | P1-4 |
| 10 | Configurer la whitelist d'origines depuis les mêmes variables que le CORS (`CORSOrigins`) | ✅ | P1-4 |
| 11 | Test `httptest` : requête POST sans `Origin` → 403 ; avec `Origin` valide → pass-through (5 tests) | ✅ | P1-4 |
| | **http.Error → writeError (P1-5)** | | |
| 12 | Remplacer `http.Error()` par `writeError()` (JSON structuré) dans `home.go` | ✅ | P1-5 |
| 13 | Idem dans `stats.go` | ✅ | P1-5 |
| 14 | Idem dans `sessions.go` | ✅ | P1-5 |
| 15 | Vérifier que `contract_test.go` (Sprint 29) attrape désormais tout `http.Error` résiduel | ⬜ | P1-5 |
| | **JSON malformé (P1-6)** | | |
| 16 | Dans `StatsHandler`, valider le body JSON reçu : rejeter avec 400 + MaxBytesReader 1MB | ✅ | P1-6 |
| 17 | Test : envoyer `{"garbage": true}` → 400 JSON structuré | ⬜ | P1-6 |
| | **Gamertag search response (P1-7)** | | |
| 18 | Dans `gamertag.go`, ajouter le champ `query` dans la réponse JSON (echo du terme de recherche) | ✅ | P1-7 |

### Critère de sortie Sprint 30
- [x] `pool.go` : doublon → fermeture propre + `singleflight`
- [x] `backfill.go` : plus aucune concaténation SQL, validation isValidMatchID
- [x] `match_view_service.go` : 0 erreur ignorée, dégradation documentée par onglet
- [x] Middleware CSRF actif sur toutes les mutations, testé (5 tests)
- [x] Plus aucun `http.Error()` dans les handlers → writeError JSON
- [x] `StatsHandler` rejette les JSON malformés avec 400
- [x] `gamertag.go` retourne `query` dans la réponse

---

### Sprint 31 — Onboarding Go & cookies session ✅ (complété 2026-07-18)

> **Phase 6 — Réalignement contrat**
> **Objectif** : rendre le flow d'onboarding Go fonctionnel de bout en bout
> (auth → identité Halo → session → bootstrap).
>
> **Réf. audit** : P0-4, §1.10, §2.7
> **Commits** : `92709243`

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Device Code Flow → identité Halo (P0-4)** | | |
| 1 | Dans `auth.go` `pollDeviceFlow` : après échange Halo réussi, récupérer Gamertag+XUID (DisplayClaims XSTS) | ✅ | P0-4 |
| 2 | Propager `gamertag` + `xuid` dans la session cookie via attempt store → GetDeviceFlowStatus | ✅ | P0-4 |
| 3 | Corriger `bootstrap_service.go` : `AuthState` dynamique (missing/partial/ready) depuis session | ✅ | P0-4 |
| 4 | Corriger `DiscordConfigured` et `TailscaleEnabled` → lire depuis settings (`discord_webhook_url`, `tailscale_enabled`) | ✅ | §2.7 |
| 5 | Vérifier que `SetupState` dans le bootstrap reflète correctement l'état du profil joueur | ✅ | P0-4 |
| | **Compatibilité cookies session (§1.10)** | | |
| 6 | Inventorier les attributs du cookie session Go : `levelup_session`, Path=/, HttpOnly, SameSite=Lax, Secure (prod), MaxAge 7j | ✅ | §1.10 |
| 7 | Config déjà compatible navigateur (mêmes attributs standards) | ✅ | §1.10 |
| 8 | Stratégie : invalidation one-shot (re-login) — acceptable car premier déploiement Go | ✅ | §1.10 |
| 9 | Message invalidation : géré par le flow bootstrap normal (AuthState=missing → redirect auth) | ✅ | §1.10 |
| | **Test onboarding** | | |
| 10 | `bootstrap_test.go` : 5 cas AuthState + 3 cas LinkedIdentity (tagged cgo) | ✅ | Gate bascule |
| 11 | Test E2E Playwright complet reporté (nécessite stack CGO + mock Halo) | ⬜ | — |

### Critère de sortie Sprint 31
- [x] `pollDeviceFlow` récupère Gamertag + XUID et les propage en session
- [x] `AuthState` dynamique (pas hardcodé) dans le bootstrap
- [x] Cookie session Go compatible navigateur
- [x] Tests bootstrap AuthState + LinkedIdentity passent

---

### Sprint 32 — Contrat API : Lots 1-3 (5–8 jours)

> **Phase 6 — Réalignement contrat**
> **Objectif** : réaligner les endpoints Go sur le contrat FastAPI, page par page,
> en commençant par les plus simples (validation conforme) puis les POST à convertir.
>
> **Réf. audit** : P0-5 / P0-6 (lots 1-3 du §3)

#### Lot 1 — Home, Career, Settings (validation conformité)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 1 | **Home** (`/pages/home`) : comparer champ par champ la réponse Go vs golden value FastAPI. Corriger le bug `http.Error()` (si pas déjà fait Sprint 30) et vérifier la shape complète du JSON. | ⬜ | — |
| 2 | **Career** (`/pages/career`, `/career/top-matches`, `/career/encounters`) : golden diff → documenter les écarts éventuels, corriger si non-conforme. | ⬜ | — |
| 3 | **Settings** (`GET /settings`, `PATCH /settings`) : vérifier que les champs et valeurs par défaut sont identiques. Tester le PATCH avec des valeurs limites. | ⬜ | — |
| 4 | **Battle Pass + Challenges** : vérifier que l'état `auth_required` / `unavailable` est renvoyé correctement jusqu'à la complétion de Sprint 31. | ⬜ | — |

#### Lot 2 — Citations, Media, Synthesis (POST + DTO fix)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 5 | **Citations** (`/pages/citations`) : changer la méthode Go de GET → POST, accepter le body filtres (identique FastAPI `CitationsPageRequest`), retourner le DTO complet. | ⬜ | P0-5 |
| 6 | Aligner le DTO de réponse citations Go sur `CitationsPageResponse` FastAPI (médaille → catégories, fréquence, totaux). | ⬜ | P0-5 |
| 7 | **Media** (`/pages/media`) : changer de GET → POST, accepter le body filtres/tri/pagination (identique FastAPI `MediaPageRequest`). | ⬜ | P0-5 |
| 8 | Aligner le DTO de réponse media Go (inclure filtres appliqués, tri, metadata, pas juste `?page=N`). | ⬜ | P0-5 |
| 9 | **Synthesis** (`/pages/synthesis`) : changer de GET → POST, accepter le body filtres (identique FastAPI `SynthesisPageRequest`). | ⬜ | P0-5 |
| 10 | Compléter le payload synthesis Go : ajouter les ~60% champs manquants identifiés dans §1.4.4 de l'audit (`comparison_metrics`, `heatmap_data`, `top_weeks`, sous-modules solo/squad etc.). | ⬜ | P0-5 |

#### Lot 3 — Explorer, Match History (ajouts + fixes)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 11 | **Explorer — Player Query** (`/pages/explorer/player-query`) : renommer `other_gamertag` → `target_gamertag` dans le DTO Go (aligner sur FastAPI). Enrichir la réponse avec les champs manquants. | ⬜ | P0-5 |
| 12 | **Explorer — Matches Query** (`/pages/explorer/matches-query`) : implémenter l'endpoint absent. Le contrat est un POST avec filtres (maps, modes, dates, outcome) → résultats paginés. Porter depuis le service FastAPI `explorer_service.py`. | ⬜ | P0-6 |
| 13 | **Match History Query** (`/pages/match-history/query`) : ajouter le champ `columns` dans le DTO Go (configuration des colonnes affichées, comme FastAPI). | ⬜ | P0-5 |
| 14 | **Match History Export** (`/pages/match-history/export`) : implémenter l'endpoint absent. Export CSV des matchs filtrés. | ⬜ | P0-6 |

### Critère de sortie Sprint 32
- [ ] Lot 1 : golden diff = 0 écart sur Home, Career (3 endpoints), Settings (2 endpoints)
- [ ] Lot 2 : Citations, Media, Synthesis acceptent POST + body filtres (même DTO que FastAPI)
- [ ] Lot 2 : Synthesis renvoie 100% du payload (plus de ~60% absent)
- [ ] Lot 3 : `target_gamertag` aligné, `matches-query` et `match-history/export` implémentés
- [ ] Tous les endpoints ci-dessus passent le golden diff

---

### Sprint 33 — Contrat API : Lots 4-5 (5–8 jours)

> **Phase 6 — Réalignement contrat**
> **Objectif** : traiter les endpoints les plus divergents (réécriture contrat complète)
> et implémenter les derniers endpoints absents.
>
> **Réf. audit** : P0-5 / P0-6 (lots 4-5 du §3)

#### Lot 4 — Teammates/Squad, Timeseries (réécriture contrat)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Teammates → Squad** | | |
| 1 | Ajouter l'endpoint Go `POST /pages/teammates` (route FastAPI) en parallèle de l'existant `GET /pages/squad` | ⬜ | P0-5 |
| 2 | Accepter le body POST filtres identique `TeammatesPageRequest` FastAPI | ⬜ | P0-5 |
| 3 | Aligner le DTO de réponse Go sur `TeammatesPageResponse` FastAPI — conserver les données enrichies Go comme champs bonus si applicable | ⬜ | P0-5 |
| 4 | Le Go-only `GET /pages/squad` reste comme endpoint bonus mais n'est pas le contrat React | ⬜ | — |
| | **Timeseries** | | |
| 5 | Créer l'endpoint Go `POST /pages/timeseries` (route FastAPI, remplace le Go-only `/pages/stats/query`) | ⬜ | P0-5 |
| 6 | Accepter le body `TimeseriesPageRequest` FastAPI (tab filter, period/session mode, match filters) | ⬜ | P0-5 |
| 7 | **Décision architecturale Plotly** : implémenter une couche de compatibilité qui prend les data points Go et les sérialise en `PlotlyFigurePayload` (data+layout JSON). Alternative : adapter le frontend pour consommer les data points bruts (plus long, hors scope bascule). | ⬜ | Exception Plotly §3 |
| 8 | Compléter le DTO Go `TimeseriesPageResponse` avec les 5 onglets : `summary_tab`, `cumul_tab`, `form_tab`, `intensity_tab`, `distributions_tab` | ⬜ | P0-6 |
| 9 | Chaque onglet expose ses charts comme `PlotlyFigurePayload` (ou data points selon la décision §7) | ⬜ | — |
| 10 | Golden diff : comparer le JSON de sortie Go vs FastAPI sur 3 gamertags × 2 period modes | ⬜ | — |

#### Lot 5 — Endpoints absents restants

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 11 | **Last Match Resolve** (`POST /pages/last-match/resolve`) : implémenter l'endpoint qui résout le dernier match d'un joueur et redirige vers la vue match-view. Porter depuis `last_match_service.py`. | ⬜ | P0-6 |
| 12 | **Session Compare** (`POST /pages/session-compare`) : implémenter l'endpoint complet. Accepte 2+ session_ids, retourne les métriques de comparaison. Porter depuis `session_compare_service.py`. | ⬜ | P0-6 |
| 13 | Aligner le DTO session-compare Go sur `SessionComparePageResponse` FastAPI (KPIs, charts, deltas). | ⬜ | P0-6 |

### Critère de sortie Sprint 33
- [ ] `POST /pages/teammates` implémenté avec contrat FastAPI
- [ ] `POST /pages/timeseries` implémenté avec 5 onglets complets
- [ ] Décision Plotly documentée et implémentée (compatibilité ou data points)
- [ ] `last-match/resolve` et `session-compare` implémentés et conformes
- [ ] Golden diff = 0 écart sur les 4 endpoints (tolérance float)

---

## Phase 7 — Infrastructure & bascule production

> **Objectif** : rebaser toute l'infra (Docker, CI, releases, deploy) sur le
> runtime Go, mettre en place les golden tests automatisés, puis bascule.

---

### Sprint 34 — Infra release/deploy Go (5–8 jours)

> **Phase 7 — Infrastructure & bascule**
> **Objectif** : rebaser l'automatisation sur le runtime Go.
>
> **Réf. audit** : P0-7, P0-8, R8

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Décision stratégie distribution (P0-7)** | | |
| 1 | Documenter la stratégie cible : serveur/container only, self-host natif par OS, ou desktop. Créer une ADR courte. | ⬜ | P0-7 |
| | **Dockerfile & compose (P0-8)** | | |
| 2 | Réécrire `Dockerfile` en multi-stage Go : build CGo (DuckDB) → image finale minimale (distroless ou alpine) | ⬜ | P0-8 |
| 3 | Inclure `apps/web/dist` (build Vite) dans l'image finale | ⬜ | P0-8 |
| 4 | Mettre à jour `docker-compose.yml` pour lancer le binaire Go au lieu d'uvicorn/FastAPI | ⬜ | P0-8 |
| 5 | Vérifier que `docker compose up` démarre correctement, healthcheck passe | ⬜ | P0-8 |
| | **Makefile racine** | | |
| 6 | Mettre à jour les cibles `make dev`, `make build`, `make run` pour le runtime Go | ⬜ | P0-8 |
| 7 | Vérifier que `make dev` lance Go + Vite dev server (HMR) | ⬜ | P0-8 |
| | **CI/CD workflows (R8)** | | |
| 8 | Réécrire `release.yml` : build matrice Go natif (linux-amd64, windows-amd64) + web dist → archive par OS | ⬜ | R8 |
| 9 | Source de version unifiée : tag Git ou fichier `VERSION` consommé par Go (`-ldflags`) + web + release notes | ⬜ | R8 |
| 10 | Réécrire `deploy.yml` pour déployer l'image OCI Go (pas le script Python) | ⬜ | R8 |
| 11 | Réécrire `test-deploy-precheck.yml` pour valider l'image Go (healthcheck + smoke test) | ⬜ | R8 |
| 12 | Réécrire `bump-version.yml` pour mettre à jour `VERSION` (plus seulement `pyproject.toml`) | ⬜ | R8 |
| 13 | Optionnel : publication d'images OCI sur un registry (GHCR / Docker Hub) | ⬜ | R8 |

### Critère de sortie Sprint 34
- [ ] `docker compose up` démarre le runtime Go + sert le frontend
- [ ] Healthcheck Docker passe
- [ ] `make dev` fonctionne (Go + Vite HMR)
- [ ] `release.yml` produit des artefacts Go+web par OS
- [ ] `deploy.yml` et `test-deploy-precheck.yml` ciblent l'image Go
- [ ] Source de version unifiée (plus de `pyproject.toml`-only)

---

### Sprint 35 — Golden tests CI + shadow mode (4–6 jours)

> **Phase 7 — Infrastructure & bascule**
> **Objectif** : automatiser la vérification de parité Go vs golden values en CI,
> et optionnellement activer un shadow mode de comparaison runtime.
>
> **Réf. audit** : R2, R4, R7 (partiel)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Golden tests automatisés (R2)** | | |
| 1 | Créer des fixtures DuckDB légères : base metadata + shared + player avec ~10-20 matchs reproductibles | ⬜ | R2 |
| 2 | Adapter `parity_check.py` pour fonctionner avec le backend Go + fixtures (pas besoin de Python live) | ⬜ | R2 |
| 3 | Ajouter un job CI `golden-test` : démarrer Go avec fixtures → exécuter parity_check → diff JSON → fail si écart | ⬜ | R2 |
| 4 | Configurer les tolérances : champs ignorés (`timestamp`, IDs dynamiques), tolérance float (ε=0.001) | ⬜ | R2 |
| 5 | Capturer de nouvelles golden values pour les endpoints ajoutés (Sprint 32-33) : session-compare, matches-query, export, last-match | ⬜ | R2 |
| | **Shadow mode diff (R4)** | | |
| 6 | Ajouter un mode `"both"` aux feature flags qui appelle les deux backends en parallèle | ⬜ | R4 |
| 7 | Comparer les réponses JSON (clés, types, shape — pas les valeurs exactes) et logger les diffs en `slog.Warn` | ⬜ | R4 |
| 8 | Le frontend reçoit toujours la réponse du backend de référence (Python pendant transition, puis Go) | ⬜ | R4 |
| 9 | Dashboard ou script qui agrège les diffs shadow sur une période (ex: combien de divergences /jour/page) | ⬜ | R4 |
| | **Quick wins observabilité (R7 partiel)** | | |
| 10 | Ajouter `response_bytes` au middleware slog → détecter les réponses anormalement petites (signe de payload appauvri) | ⬜ | R7 |
| 11 | Monter le seuil de couverture Go de 30% → 50% dans `ci.yml` | ⬜ | R7 |

### Critère de sortie Sprint 35
- [ ] Job CI `golden-test` passe sur les 24+ endpoints câblés
- [ ] Golden values existent pour tous les endpoints post-Sprint 33
- [ ] Shadow mode fonctionnel et diffs loggés (au moins 1 semaine d'observation recommandée avant bascule)
- [ ] `response_bytes` loggé pour chaque requête
- [ ] Seuil couverture Go ≥ 50%

---

### Sprint 36 — Validation & bascule production (3–5 jours)

> **Phase 7 — Infrastructure & bascule**
> **Objectif** : vérifier systématiquement tous les critères de bascule de l'audit,
> puis basculer le runtime en production.
>
> **Réf. audit** : Critère de bascule mesurable (§ Décision stratégique)

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Vérification critères de bascule** | | |
| 1 | **Parité contrat** : exécuter `parity_check.py` sur les 24 endpoints → 0 diff (tolérance configurée) | ⬜ | Critère 1 |
| 2 | **E2E vert** : les 15 specs Playwright React passent sur le backend Go en CI | ⬜ | Critère 2 |
| 3 | **Onboarding E2E** : dérouler le flow setup complet → home page (test depuis Sprint 31) | ⬜ | Critère 3 |
| 4 | **Sécurité** : audit rapide — CSRF actif, 0 `http.Error()`, pool.go corrigé, 0 `nolint:errcheck` non justifié | ⬜ | Critère 4 |
| 5 | **Infra** : `docker compose up` + healthcheck + `make dev` fonctionnels | ⬜ | Critère 5 |
| 6 | **Cookie compat** : stratégie documentée et testée (re-login ou compatibilité) | ⬜ | Critère 6 |
| | **Bascule** | | |
| 7 | Basculer le feature flag par défaut de Python → Go pour toutes les routes | ⬜ | — |
| 8 | Monitorer pendant 48h : logs slog, response_bytes, shadow diffs, error rate | ⬜ | — |
| 9 | Si 0 incident critique : retirer le backend Python du `docker-compose.yml` | ⬜ | — |
| 10 | Mettre à jour la documentation d'exploitation, README, CLAUDE.md | ⬜ | — |
| | **Rollback plan** | | |
| 11 | Documenter la procédure de rollback : remettre le feature flag sur Python en < 1 minute | ⬜ | — |
| 12 | Garder le backend FastAPI déployable pendant 2 semaines post-bascule (pas de suppression immédiate) | ⬜ | — |

### Critère de sortie Sprint 36 (= Gate bascule production)
- [ ] **parity_check.py = 0 diff** sur les 24 endpoints câblés
- [ ] **15 specs Playwright = vert** sur backend Go en CI
- [ ] **Onboarding E2E = vert** (auth → player → sync → home)
- [ ] **Sécurité = OK** (CSRF, pool, errors, JSON validation)
- [ ] **Infra = OK** (Docker, docker-compose, Makefile, healthcheck)
- [ ] **Cookie = documenté** et testé
- [ ] **48h de monitoring** sans incident critique
- [ ] **Rollback plan** documenté et testé

---

## Phase 8 — Qualité & dette technique (post-bascule)

> **Objectif** : une fois en production, nettoyer l'architecture interne Go,
> améliorer la testabilité, réduire la dette technique. Non bloquant pour la bascule
> mais important pour la maintenabilité long terme.

---

### Sprint 37 — Architecture handlers & injection (4–6 jours)

> **Phase 8 — Qualité**
> **Objectif** : rendre les handlers testables unitairement via injection de dépendances.
>
> **Réf. audit** : P2-1, P2-7

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 1 | Refactorer `NewRouter` pour accepter un `ServiceRegistry` ou chaque service en paramètre | ⬜ | P2-1 |
| 2 | Chaque handler reçoit son service via le constructeur (plus de `resolvePlayer → NewRepo → NewService` inline) | ⬜ | P2-1 |
| 3 | Créer des interfaces de service pour les 6 principaux handlers : `HomeService`, `CareerService`, `StatsService`, `SquadService`, `SynthesisService`, `ExplorerService` | ⬜ | P2-1 |
| 4 | Convertir les 15 handlers violant l'hexagone (cf. §2.2 de l'audit) | ⬜ | P2-1 |
| 5 | Extraire `createPlayerInProfiles()` de `setup.go` dans un `ProfileService` | ⬜ | P2-7 |
| 6 | Vérifier que tous les handlers compilent et que les tests existants passent | ⬜ | — |
| 7 | Écrire un test handler `home_handler_test.go` pour valider le nouveau pattern (service mocké → JSON attendu) | ⬜ | R3 |

### Critère de sortie Sprint 37
- [ ] 0 handler qui construit `repo+service` inline
- [ ] Interfaces de service définies dans `internal/port/`
- [ ] `setup.go` sans logique métier (déléguée à `ProfileService`)
- [ ] Au moins 1 test handler avec mock service

---

### Sprint 38 — DRY + split fichiers >500L (4–6 jours)

> **Phase 8 — Qualité**
> **Objectif** : réduire la duplication et découper les fichiers trop gros.
>
> **Réf. audit** : P2-2→P2-6, P2-8

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Split fichiers >500L (P2-2)** | | |
| 1 | `analysis/squad.go` (812L) → split en 6 fichiers : `squad_records.go`, `squad_participation.go`, `squad_impact.go`, `squad_heatmap.go`, `squad_top_weeks.go`, `squad.go` (orchestration) | ⬜ | P2-2 |
| 2 | `sync/skill_rating.go` (731L) → extraire les requêtes SQL dans `skill_rating_repo.go` | ⬜ | P2-2/P2-4 |
| 3 | `platform/duckdb/queries.go` (714L) → split par domaine fonctionnel : `queries_career.go`, `queries_match.go`, `queries_medal.go`, etc. | ⬜ | P2-2 |
| 4 | `sync/transforms.go` (570L) → extraire helpers dans `transforms_helpers.go` | ⬜ | P2-2 |
| 5 | `cmd/levelup/main.go` (532L) → extraire sous-commandes en fichiers séparés : `cmd_serve.go`, `cmd_sync.go`, etc. | ⬜ | P2-2 |
| | **DRY squad generics (P2-3)** | | |
| 6 | Unifier `ComputeParticipationProfile(SquadMatchRow)` / `ComputeTeammateProfile(TeammateMatchRow)` via generics (~90% identique) | ⬜ | P2-3 |
| 7 | Unifier `ComputeSquadRecords` / `ComputeTeammateRecords` via generics (~95% identique) | ⬜ | P2-3 |
| | **Autres DRY (P2-5/P2-6/P2-8)** | | |
| 8 | Refactorer double-switch `feature_flags.go` en map lookup | ⬜ | P2-5 |
| 9 | Unifier double cache DB : supprimer `duckdb/db.go` `openDBs` OU `pool.go` `globalPool`, garder un seul mécanisme | ⬜ | P2-6 |
| 10 | Factoriser les requêtes SQL quasi-identiques Q4/Q4MV/Q5 dans `queries.go` | ⬜ | — |
| 11 | Déplacer les ~180L de noop impls de `port/repository.go` dans `port_check_test.go` | ⬜ | P2-8 |
| 12 | Remplacer le magic number `case 2: winScore = 1.0` dans `skill_rating.go` par des constantes nommées | ⬜ | §2.6 |

### Critère de sortie Sprint 38
- [ ] Plus aucun fichier Go >500L (seuil projet)
- [ ] Plus de duplication `SquadMatchRow/TeammateMatchRow` (generics)
- [ ] Feature flags = map lookup (pas double switch)
- [ ] Un seul mécanisme de cache DB
- [ ] 0 magic number dans `skill_rating.go`

---

### Sprint 39 — Tests couches manquantes + couverture 50% (4–6 jours)

> **Phase 8 — Qualité**
> **Objectif** : combler les trous de couverture identifiés par l'audit
> (handlers, repositories, error paths).
>
> **Réf. audit** : R3, R6, §4.3

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Tests handlers httptest (R3)** | | |
| 1 | Écrire des tests `httptest` pour les 6 handlers principaux : home, career, stats, squad, synthesis, explorer | ⬜ | R3 |
| 2 | Chaque test vérifie : status code, Content-Type `application/json`, shape JSON de la réponse (clés attendues) | ⬜ | R3 |
| 3 | Tests error paths : repo retourne `error` → handler renvoie 500 JSON structuré (pas 200 vide) | ⬜ | R3 |
| 4 | Tests 404 : player non trouvé, match non trouvé → 404 JSON structuré | ⬜ | R3 |
| | **Tests repository DuckDB (R6)** | | |
| 5 | Créer un helper `testdb.go` : initialise une DB DuckDB in-memory avec le schéma attendu + fixtures minimales | ⬜ | R6 |
| 6 | Tests unitaires repository : vérifier que chaque méthode retourne les bonnes données sur fixtures connues | ⬜ | R6 |
| 7 | Tests edge cases : table absente → erreur propre (pas panic), colonne manquante → dégradation | ⬜ | R6 |
| | **Tests pool & middleware** | | |
| 8 | Test stress pool : ouvrir/fermer 100 connexions en parallèle → 0 leak, 0 panic | ⬜ | — |
| 9 | Test middleware CSRF (si pas déjà fait Sprint 30) : POST sans `Origin` → 403 | ⬜ | — |
| | **Tests FastAPI minimal (R6)** | | |
| 10 | Créer `apps/api/tests/` avec `TestClient` pour les 5 endpoints les plus critiques (bootstrap, home, career, history, settings) | ⬜ | R6 |
| 11 | Snapshot tests : capturer la réponse sérialisée et la committer comme golden file | ⬜ | R6 |
| | **Couverture 50%** | | |
| 12 | Vérifier que la couverture Go atteint 50% après les tests ajoutés — ajuster si nécessaire | ⬜ | R7 |

### Critère de sortie Sprint 39
- [ ] 6+ handlers avec tests httptest (status, Content-Type, shape, error paths)
- [ ] Repository testés avec DuckDB in-memory + fixtures
- [ ] Pool stress tested : 0 leak
- [ ] FastAPI : au moins 5 endpoints avec TestClient + golden files
- [ ] Couverture Go ≥ 50%

---

### Sprint 40 — Observabilité & monitoring (2–3 jours)

> **Phase 8 — Qualité**
> **Objectif** : ajouter l'observabilité minimale pour opérer le backend Go en production
> avec confiance.
>
> **Réf. audit** : R7 (complet), §4.4

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 1 | **Contract validation middleware (dev mode)** : valider chaque réponse JSON contre `openapi.yaml` avec `kin-openapi` ; logger un warning si fields manquants/en trop | ⬜ | R7 |
| 2 | **Error tracking** : intégrer un reporter d'erreurs (Sentry ou simple webhook Discord pour les 500) | ⬜ | §4.4 |
| 3 | **Alerting error rate** : si > 5% d'erreurs sur une route en 5 minutes → notification Discord | ⬜ | §4.4 |
| 4 | Optionnel : métriques Prometheus (latence p50/p95 par route, error rate, connexions DB actives) — si un dashboard Grafana est déjà en place | ⬜ | §4.4 |
| 5 | Optionnel : tracing distribué (OpenTelemetry) — utile si le flow auth → sync → response doit être debuggé en prod | ⬜ | §4.4 |

### Critère de sortie Sprint 40
- [ ] Contract validation middleware logge les divergences OpenAPI en dev
- [ ] Les erreurs 500 sont reportées (Sentry ou Discord)
- [ ] Alerting error rate fonctionnel (au moins via Discord)

---

## Phase 9 — Évolutions fonctionnelles & UX

> **Objectif** : ajouter les fonctionnalités manquantes et les améliorations UX
> identifiées dans l'audit. Ces sprints ne sont pas bloquants pour la production
> mais enrichissent le produit.

---

### Sprint 41 — Scoreboard complet + weapon film parsing + healthcheck (5–8 jours)

> **Phase 9 — Évolutions**
> **Objectif** : compléter les surfaces de données appauvries en Go.
>
> **Réf. audit** : P3-1, P3-3, P3-5

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Scoreboard complet (P3-3)** | | |
| 1 | Identifier les 13+ colonnes scoreboard manquantes dans `match_view_service.go` par diff avec FastAPI | ⬜ | P3-3 |
| 2 | Ajouter les requêtes DuckDB nécessaires dans le repository match | ⬜ | P3-3 |
| 3 | Compléter le DTO `MatchViewResponse` Go avec toutes les colonnes | ⬜ | P3-3 |
| 4 | Golden diff match_view sur 5+ matchs variés (PvP, PvE, Ranked, BTB) | ⬜ | P3-3 |
| | **Weapon film parsing (P3-1)** | | |
| 5 | Vérifier l'état du weapon parser Go existant : est-il branché sur les bons flux (highlight_events → weapon_kills) ? | ⬜ | P3-1 |
| 6 | Si nécessaire, connecter le weapon parser au pipeline de sync/backfill Go | ⬜ | P3-1 |
| 7 | Vérifier la parité numérique weapon_kills Go vs Python sur 20+ matchs | ⬜ | P3-1 |
| | **Healthcheck complet (P3-5)** | | |
| 8 | Déplacer le healthcheck Go sous `/api/v1/health` (mêmes préfixe que les autres routes) | ⬜ | P3-5 |
| 9 | Enrichir la réponse : `{status, match_count, db_version, players_count, uptime_seconds, last_sync}` | ⬜ | P3-5 |

### Critère de sortie Sprint 41
- [ ] Scoreboard Go = parité complète avec FastAPI (0 colonne manquante)
- [ ] Weapon parser branché et vérifié sur 20+ matchs
- [ ] Healthcheck sous `/api/v1/health` avec infos enrichies

---

### Sprint 42 — Analyse UI avancée + fanout multi-joueur (5–8 jours)

> **Phase 9 — Évolutions**
> **Objectif** : porter les algorithmes d'analyse UI restants et le fanout enrichment.
>
> **Réf. audit** : P3-2, P3-4

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Fanout enrichment multi-joueur (P3-2)** | | |
| 1 | Porter la logique de fanout : quand un match est synced pour le joueur A, enrichir aussi le `match_participants` pour les joueurs B, C, D présents dans le match | ⬜ | P3-2 |
| 2 | Tester avec 2+ joueurs configurés : vérifier que les stats croisées sont à jour | ⬜ | P3-2 |
| | **Algorithmes d'analyse UI (P3-4)** | | |
| 3 | Porter les algorithmes cumul (Onglet Cumul : `cumul_net_chart`, `cumul_kd_chart`, `rolling_kd_chart`) si pas déjà fait dans Sprint 33 | ⬜ | P3-4 |
| 4 | Porter les algorithmes de forme (Onglet Forme : `ewma_kd_chart`, `ewma_winrate_chart`, `current_streak`, `regression_stats`) | ⬜ | P3-4 |
| 5 | Porter les algorithmes d'intensité (Onglet Intensité : `intensity_heatmap`, `score_per_minute_chart`) | ⬜ | P3-4 |
| 6 | Porter les distributions (Onglet Distributions : `kda_distribution`, `score_distribution`, `correlations`) | ⬜ | P3-4 |
| 7 | Golden diff par onglet sur 3 gamertags | ⬜ | P3-4 |

### Critère de sortie Sprint 42
- [ ] Fanout multi-joueur fonctionnel et testé
- [ ] 4 onglets d'analyse UI (Cumul, Forme, Intensité, Distributions) en parité avec FastAPI
- [ ] Golden diff vert par onglet

---

### Sprint 43 — Améliorations UX produit (5–8 jours)

> **Phase 9 — Évolutions**
> **Objectif** : implémenter les améliorations UX/produit identifiées dans le P4 de l'audit.
>
> **Réf. audit** : P4-1→P4-4

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| | **Bipolaire solo/escouade (P4-1)** | | |
| 1 | Vérifier que `buildBipolaireChart()` de `SynthesisPage.tsx` reçoit bien `comparison_metrics[]` avec `solo_kpis`/`squad_kpis` depuis Go (dépend du sprint 32 — synthèse payload 100%) | ⬜ | P4-1 |
| 2 | Si des KPIs manquent côté Go, les ajouter au service synthesis | ⬜ | P4-1 |
| | **Tooltips LUSR & Performance Score (P4-2)** | | |
| 3 | Ajouter un composant React `InfoTooltip` : icône `(?)` avec popover contenant 1-2 phrases explicatives | ⬜ | P4-2 |
| 4 | Placer les `InfoTooltip` à côté de chaque occurrence de « Performance Score » et « LUSR » dans les pages Home, Career, Stats, Synthesis | ⬜ | P4-2 |
| 5 | Le tooltip pointe vers la page Changelog (P4-3) pour plus de détails | ⬜ | P4-2 |
| | **Page "Quoi de neuf" / Changelog (P4-3)** | | |
| 6 | Créer une route `/changelog` dans le frontend React | ⬜ | P4-3 |
| 7 | Alimenter la page depuis un fichier statique `changelog.json` (ou parser `CHANGELOG.md` du repo) | ⬜ | P4-3 |
| 8 | Afficher les versions majeures avec date, features, et liens vers les sections pertinentes | ⬜ | P4-3 |
| 9 | Optionnel : endpoint Go `GET /changelog` qui sert le contenu versionné depuis le binaire | ⬜ | P4-3 |
| | **Calcul de durée : span → durée effective (P4-4)** | | |
| 10 | Identifier les endroits qui calculent la durée de session (backend : `session_compare_service`, frontend : si applicable) | ⬜ | P4-4 |
| 11 | Ajouter le calcul de durée effective : sommer `duration_seconds` de `match_registry` pour chaque session | ⬜ | P4-4 |
| 12 | Exposer les deux métriques dans le DTO : `effective_play_time_seconds` (somme des matchs) + `session_span_seconds` (premier → dernier) | ⬜ | P4-4 |
| 13 | Côté frontend : afficher « X min jouées (session de Y min) » au lieu d'un seul chiffre | ⬜ | P4-4 |

### Critère de sortie Sprint 43
- [ ] Chart bipolaire synthesis fonctionne avec les données Go (solo vs squad complets)
- [ ] Tooltips `(?)` visibles pour LUSR et Performance Score dans toutes les pages concernées
- [ ] Page Changelog accessible et alimentée
- [ ] Durée de session affichée avec les deux métriques (temps joué + span)

---

### Sprint 44 — Implémentation multi-titres + ADR + polish final (10–14 jours)

> **Phase 9 — Évolutions**
> **Objectif** : transformer l'ADR d'architecture déjà acceptée en implémentation propre et fiable. Le Sprint 44 doit être le moment où le support multi-titres devient une capacité réellement introduite dans le runtime Go, avec design durci, migration sûre et validation forte.
>
> **Réf. audit** : P2-9, P3-6
>
> **Documents d'exécution** : [SPRINT_44_WORKPACKAGES.md](SPRINT_44_WORKPACKAGES.md) et [ADR_S44_MULTI_TITLE_NAMESPACE.md](ADR_S44_MULTI_TITLE_NAMESPACE.md)
>
> **Note estimation** : revue de 6–9j à 10–14j après audit du code Go.
> Le refactor touche toutes les couches : 29 références de chemins hardcodés dans 15 fichiers,
> pool DuckDB (13 repos), 23 endpoints OpenAPI, provisioning/setup joueur, commandes ops `levelup` + binaire `server`, routes/query keys/codegen frontend, demo mode.
> La sous-estimation venait principalement de WP3 (migration physique DuckDB sur Windows),
> WP4 (réalignement frontend complet + décision routage OpenAPI) et WP1 (ops/validation/sync/demo paths + provisioning).
> L'auth n'est pas impactée (flow MSAL titre-agnostique).
>
> **Coexistence Python** : le projet Python LevelUp n'est plus maintenu à ce stade.
> Le Go est la seule baseline. Aucune rétrocompatibilité Python n'est requise.

| # | Tâche | Statut | Réf. |
|--:|-------|:------:|:----:|
| 1 | S'appuyer sur l'ADR déjà acceptée et verrouiller son alignement avec l'implémentation multi-titres | ✅ | P2-9 |
| 2 | Introduire `TitleRegistry` / `TitleDescriptor` et `PathResolver` title-aware pour centraliser titres, capabilities et chemins runtime | ⬜ | P2-9/P3-6 |
| 3 | Figer la matrice des chemins globaux vs title-aware (warehouse, players, archive, captures, backups, sessions, jobs, db_profiles, app_settings, demo fixtures) et l'encoder dans `PathResolver` | ⬜ | P2-9/P3-6 |
| 4 | Refactorer `PlayerResolver` pour accepter `(title_slug, player_slug)` et propager au pool DuckDB (clé `{title}:{gamertag}` — impacte 13 fichiers `*_repo.go`) | ⬜ | P3-6 |
| 5 | Rendre config, session, bootstrap, jobs et sélection joueur title-aware (auth non impactée) avec switch via `POST /session/context` | ⬜ | P3-6 |
| 6 | Migrer `db_profiles.json` vers un format v3 title-aware, avec lecture rétro-compatible du format actuel | ⬜ | P3-6 |
| 7 | Rendre `POST /setup/players`, `GET /players` et le provisioning de profils explicitement title-aware | ⬜ | P3-6 |
| 8 | Rendre demo mode title-aware (`DemoFixturesDir` namespacé) + migrer `internal/ops/` (6 fichiers), `validation/gate.go`, `sync/engine.go` vers `PathResolver` | ⬜ | P3-6 |
| 9 | Mettre en place le namespace de stockage `data/titles/{title_slug}/...` | ⬜ | P3-6 |
| 10 | Créer une migration opérable `dry-run / apply / rollback` via manifest JSON (`operations.json` traçant chaque `(source, dest)`), journal d'opérations, backup automatique et vérification d'idempotence | ⬜ | P3-6 |
| 11 | Créer le corpus synthétique second titre (~0.5–1j) : `metadata.duckdb` minimal + `shared_matches_v2.duckdb` avec quelques matchs, schémas compatibles | ⬜ | P3-6/R2 |
| 12 | Adapter contrats API (OpenAPI) : décider routage `{title_slug}` (23 endpoints), middleware Chi d'extraction, fallback anciennes routes, périmètre frontend complet (routes, query keys, hooks, liens, codegen) | ⬜ | P3-6 |
| 13 | Ajouter `--title` aux commandes ops concernées du binaire `levelup` et brancher la résolution de titre au démarrage du binaire `server` | ⬜ | P3-6/R2 |
| 14 | Brancher `appShellStore.currentTitleSlug`, `switchTitle()`, `isTitleSwitching`, `settingsDraftStore.lastPlayerSlug` title-aware, `buildUrl()`, routes/query keys/codegen TS | ⬜ | P3-6/R2 |
| 15 | Mettre à jour fixtures, démo, golden values et CI pour `halo_infinite` namespacé + titre synthétique | ⬜ | P3-6/R2 |
| 16 | Tests unitaires ciblés : `TitleRegistry`, `PathResolver`, `PlayerResolver` (mode réel + démo), config v3, pool keying | ⬜ | R3 |
| 17 | Tests d'intégration : migration dry-run/apply/rollback, dépôt legacy HI-only, dépôt déjà migré, isolement inter-titres (deux titres même gamertag ne partagent pas de pool), absence de fuite de chemins et provisioning title-aware | ⬜ | R3/R6 |
| 18 | Golden tests et smoke E2E : zéro diff HI pré/post migration + smoke React sur changement de titre / titre synthétique | ⬜ | R2/R4 |
| 19 | Observabilité : logs `title_slug` + `response_bytes`, validation contrat bootstrap title-aware en dev | ⬜ | R7 |
| 20 | Documentation finale : README, CLAUDE.md, copilot-instructions, runbook d'exploitation, rollback plan | ⬜ | — |

### Sous-plan de réussite 10/10

#### A. Design

- **Single source of truth titre** : aucun `title_slug` bricolé au fil de l'eau. Tous les titres supportés passent par un registre central décrivant slug, capacités, defaults et résolveurs de chemins.
- **PlayerResolver** : pivot central du refactor. C'est la première brique à modifier car elle résout `player_slug` → gamertag → chemins DB. Le pool DuckDB doit passer d'une clé `gamertag` à `{title}:{gamertag}` (13 fichiers `*_repo.go` impactés, transparent via `PlayerDB` enrichie).
- **Chemins hardcodés** : 29 références dans 15 fichiers (`cmd/server`, `config/player_resolver`, `ops/`, `validation/gate`, `sync/engine`). Toutes doivent passer par le `PathResolver`.
- **Matrice des chemins** : le sprint doit figer ce qui reste global (`app_settings.json`, `db_profiles.json`, `data/sessions`, `data/cache/jobs.json`) et ce qui devient title-aware (`warehouse`, `players`, `archive`, `captures`, `backups`, fixtures démo). Pas de déduction implicite tolérée.
- **Demo mode** : `resolveDemoPlayer()` et `DemoFixturesDir` doivent devenir title-aware.
- **Runtime context explicite** : session, bootstrap et jobs doivent toujours savoir pour quel titre ils opèrent. Aucune déduction implicite depuis le joueur courant ou des chemins par défaut.
- **Config** : `db_profiles.json` v3 title-aware avec rétrocompatibilité lecture du format actuel.
- **Provisioning** : `POST /setup/players`, `GET /players` et la matérialisation du layout joueur doivent suivre le titre courant.
- **OpenAPI** : 23 endpoints avec `{player_slug}` doivent intégrer `{title_slug}` (recommandation : préfixe path + fallback anciennes routes).
- **CLI** : les commandes ops concernées du binaire `levelup` doivent accepter `--title` (défaut `halo_infinite`) et le binaire `server` doit résoudre correctement le titre au démarrage.
- **Frontend** : le blast radius ne se limite pas aux stores. `appShellStore.currentTitleSlug` + `switchTitle()` + `isTitleSwitching` (loader) + stores `reset()`, mais aussi routes TanStack, `routeTree.gen.ts`, `queryKeys`, hooks `features/*/queries.ts`, liens de navigation, codegen OpenAPI, MSW/Playwright et `settingsDraftStore.lastPlayerSlug` doivent devenir title-aware.
- **Switch titre runtime** : préparé structurellement, pas de bouton UI. Flux : `POST /session/context {title_slug}` → validation `TitleRegistry` → invalidation joueur courant → bootstrap complet retourné. Frontend : flush stores → re-hydratation atomique. Lazy pool opening (connexions DuckDB du nouveau titre ouvertes à la demande). Erreur → rollback silencieux côté frontend.
- **Auth hors périmètre** : le flow MSAL est titre-agnostique (confirmé par audit). Aucune modification requise dans `internal/platform/auth/`.

#### B. Migration

- **Migration opérable** : commande Go avec `dry-run`, `apply` et `rollback`. Mécanisme retenu : manifest JSON (`operations.json`) traçant chaque opération `(source, dest)`, rollback = exécution inverse du manifest. Pas de symlinks (problématiques sur Windows).
- **Idempotence** : une migration relancée ne doit ni dupliquer ni corrompre les données, et doit détecter proprement un dépôt déjà migré.
- **Budget corpus synthétique** : prévoir 0.5–1 jour dédié pour un jeu de données minimal mais significatif pour un second titre (metadata + shared_matches avec quelques matchs).
- **Parité Halo Infinite** : la migration ne doit pas introduire de divergence fonctionnelle pour le titre existant ; le namespace n'est pas acceptable si Halo Infinite régresse.
- **Coexistence Python** : non requise. Le Go est la seule baseline à ce stade.

#### C. Validation

- **Tests de modules touchés** : les nouveaux composants title-aware doivent avoir une couverture ciblée élevée ; l'objectif est **≥ 80%** sur les modules modifiés par le Sprint 44, tout en conservant **≥ 50%** de couverture globale.
- **Golden parity** : comparer Halo Infinite avant/après migration sur le corpus de référence, pas seulement sur des tests unitaires.
- **Isolation inter-titres** : vérifier avec le corpus synthétique que deux titres partageant le même gamertag ne fuient pas entre config, sessions, pools DuckDB, jobs et bootstrap.
- **Smoke frontend** : vérifier que le shell React démarre, change de contexte et consomme le bootstrap title-aware sans drift de contrat.
- **Logging structuré** : tous les événements title-aware loggés via slog avec attributs typés (`title_switched`, `legacy_session`, `bootstrap_served`, `job_created`, `title_switch_rejected`). Vérifiable via `slogtest.Handler` dans les tests.
- **Tests quantifiés** : 20 tests WP2 (9 unitaires session/bootstrap + 6 httptest switch + 3 intégration pool/job/persistence + 2 logging slog) + 17 tests WP4 (8 backend middleware/CLI/fallback + 7 frontend store/prefs/client + 2 E2E Playwright) + golden parity + smoke E2E.

### Critère de sortie Sprint 44
- [ ] `TitleRegistry` / `PathResolver` centralisent les titres et chemins runtime (29 refs migrées)
- [ ] `PlayerResolver` title-aware (mode réel + mode démo), pool DuckDB clé `{title}:{gamertag}`
- [ ] `db_profiles.json` v3 title-aware avec rétrocompatibilité lecture
- [ ] `title_slug` introduit comme dimension explicite du runtime Go
- [ ] Namespace `data/titles/{title_slug}/...` branché côté stockage/config/runtime
- [ ] Demo mode title-aware
- [ ] 6 fichiers `internal/ops/` + `validation/gate.go` + `sync/engine.go` passent par `PathResolver`
- [ ] Migration HI-only → namespace via manifest JSON, testée et idempotente
- [ ] `POST /setup/players` + `GET /players` provisionnent/listent correctement dans le titre courant
- [ ] Switch titre runtime fonctionnel : `POST /session/context {title_slug}` → invalidation joueur + re-bootstrap
- [ ] `SessionData.CurrentTitleSlug` non-nul + fallback legacy `"halo_infinite"`
- [ ] `BootstrapResponse` enrichi : `current_title` + `available_titles` (type `TitleSummary`)
- [ ] `JobMeta` structuré avec `TitleSlug` obligatoire, validé via `TitleRegistry`
- [ ] Routage OpenAPI `{title_slug}` décidé et implémenté (23 endpoints + fallback)
- [ ] Commandes ops concernées du binaire `levelup` acceptent `--title` et le binaire `server` résout le titre au démarrage
- [ ] Frontend : `appShellStore.currentTitleSlug` + `switchTitle()` + `isTitleSwitching` + stores `reset()` + `settingsDraftStore.lastPlayerSlug` title-aware
- [ ] API client `buildUrl()` title-aware + routes/query keys/codegen TS alignés + types TS générés `TitleSummary`
- [ ] Logging structuré complet : `title_switched`, `legacy_session`, `bootstrap_served`, `job_created`
- [ ] Zéro diff Halo Infinite sur corpus golden après migration
- [ ] Corpus synthétique second titre valide l'isolement inter-titres
- [ ] Tests : 20 WP2 + 17 WP4 + golden + smoke E2E Playwright
- [ ] Couverture ciblée des modules Sprint 44 ≥ 80% et couverture Go globale ≥ 50%
- [ ] ADR multi-titres déjà acceptée, relue et alignée avec l'implémentation
- [ ] Documentation à jour
- [ ] 0 TODO non-documenté dans le codebase Go
- [ ] `golangci-lint run` clean

---

## Estimation totale & risques

### Effort par phase

| Phase | Sprints | Effort estimé | Nature |
|-------|:-------:|:---:|---|
| **Phase 6** — Réalignement contrat | 29-33 | **21-33j** | Chemin critique — bloquant bascule |
| **Phase 7** — Infrastructure & bascule | 34-36 | **12-19j** | Dépend de Phase 6 |
| **Phase 8** — Qualité post-bascule | 37-40 | **14-21j** | Non bloquant, maintenabilité |
| **Phase 9** — Évolutions fonctionnelles | 41-44 | **25-38j** | À la demande |
| **Total** | 29-44 | **~72-111j** | ~3-5 mois (1 dev senior) |

### Risques identifiés

| Risque | Probabilité | Impact | Mitigation |
|--------|:-----------:|:------:|------------|
| La couche Plotly compat (Sprint 33) est plus complexe que prévu | Moyenne | Élevé | Décider tôt : compat server-side vs adapter le frontend à des data points bruts |
| Les golden values FastAPI dérivent pendant la Phase 6 | Faible | Moyen | Option 2 (gel features FastAPI) + shadow mode |
| L'onboarding Go (Sprint 31) dépend de bugs MSAL non documentés | Moyenne | Élevé | Mock Halo provider pour les tests E2E, debug MSAL en parallèle |
| La bascule révèle des cas edge non couverts par les fixtures | Moyenne | Moyen | Shadow mode pendant 1-2 semaines, rollback plan < 1 min |
| Phase 8 (qualité) est repoussée indéfiniment post-bascule | Élevée | Élevé | Planifier au moins Sprint 37 immédiatement après bascule (injection = prérequis tests) |
| La migration HI-only → namespace par titre casse des chemins implicites ou des jobs existants | Moyenne | Élevé | `dry-run / apply / rollback`, backup manifest, tests d'intégration sur dépôt legacy et golden diff Halo Infinite pré/post |
| Le multi-titres reste nominal car validé sur un seul titre réel | Moyenne | Élevé | Créer un corpus synthétique second titre + tests d'isolement inter-titres + smoke bootstrap/frontend title-aware |

### Critère d'abandon Phase 6-7

Si après 6 semaines de Phase 6, plus de 50% des golden diffs restent en écart, réévaluer :
- La complexité réelle du réalignement
- L'option de réaligner le frontend sur le contrat Go (Option B de l'audit)
- L'option de maintenir FastAPI en production à long terme

---

## Traçabilité audit → sprints

| Réf. audit | Sprint(s) | Statut |
|:----------:|:---------:|:------:|
| P0-1 | 29 | ⬜ |
| P0-2 | 29 | ⬜ |
| P0-3 | 29 | ⬜ |
| P0-4 | 31 | ⬜ |
| P0-5 | 32, 33 | ⬜ |
| P0-6 | 32, 33 | ⬜ |
| P0-7 | 34 | ⬜ |
| P0-8 | 34 | ⬜ |
| P1-1 | 30 | ⬜ |
| P1-2 | 30 | ⬜ |
| P1-3 | 30 | ⬜ |
| P1-4 | 30 | ⬜ |
| P1-5 | 30 | ⬜ |
| P1-6 | 30 | ⬜ |
| P1-7 | 30 | ⬜ |
| P2-1 | 37 | ⬜ |
| P2-2 | 38 | ⬜ |
| P2-3 | 38 | ⬜ |
| P2-4 | 38 | ⬜ |
| P2-5 | 38 | ⬜ |
| P2-6 | 38 | ⬜ |
| P2-7 | 37 | ⬜ |
| P2-8 | 38 | ⬜ |
| P2-9 | 44 | ⬜ |
| P3-1 | 41 | ⬜ |
| P3-2 | 42 | ⬜ |
| P3-3 | 41 | ⬜ |
| P3-4 | 42 | ⬜ |
| P3-5 | 41 | ⬜ |
| P3-6 | 44 | ⬜ |
| P4-1 | 43 | ⬜ |
| P4-2 | 43 | ⬜ |
| P4-3 | 43 | ⬜ |
| P4-4 | 43 | ⬜ |
| R0 | 29 | ⬜ |
| R1 | 29 | ⬜ |
| R2 | 35 | ⬜ |
| R3 | 37, 39 | ⬜ |
| R4 | 35 | ⬜ |
| R5 | 29 | ⬜ |
| R6 | 39 | ⬜ |
| R7 | 35, 40 | ⬜ |
| R8 | 34 | ⬜ |
