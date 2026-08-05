# Journal de decisions — archive 2025-Q3

> Rotation trimestrielle (regle CLAUDE.md).

## [2025-07-15] feat(home): snippets de citations dans MatchCard

**Statut** : ✅ Complété

**Décision technique** :
- Architecture full-stack en 9 phases : migration DDL (steps_metadata), seed réécrit, domain structs, SQL constants (Q26i+Q26j), repo `LoadMatchCitations` (dégradation silencieuse, aucun xuid — match_citations est par-player DB), analysis `BuildCitationSnippets` (pur, sans DB), port interface + noop, service `enrichMatchesWithCitations` (après medals), TS types, composant SVG ring, MatchCard.
- Filtre clé : si `cumul - delta >= lastTier`, la citation était déjà masterisée avant le match → ignorée.
- `buildLookupQuery` ne supporte que `[]int64` → placeholders manuels pour les norm strings.
- Mock test `mockHomeRepo` mis à jour avec `LoadMatchCitations` (stub vide) pour satisfaire l'interface.

**Résultats** : `go build ./...` ✅ · `go vet` (packages modifiés) ✅

**Fichiers modifiés** : steps_metadata.go, ops/seed.go, domain/home.go, queries_home_citations.go, home_repo.go, analysis/citation_snippets.go (NEW), port/repository.go, service/home_service.go, service/home_service_test.go (stub), types.ts, citation-progress-ring.tsx (NEW), match-card.tsx

## [2025-07-25] feat(social): migration shared_social.duckdb — Phases 1→4

**Statut** : Complété

**Décision technique** : Création d'une DB dédiée `shared_social.duckdb` (`data/warehouse/`) pour centraliser toutes les données sociales/contenu utilisateur : `media_files`, `media_match_associations`, `media_likes`, `match_favorites`. Rationale : ces données ne sont ni des statistiques de match (shared_matches_v2) ni des enrichissements joueur (stats.duckdb).

**Résultats** :
- Phase 1 : `TargetSharedSocial` dans registry, `PlayerDB.SharedSocial`, `steps_shared_social.go` (DDL), `main.go`, `LegacySharedSocialDBPath()`, `player_resolver.go`
- Phase 2 : Script one-shot `cmd/migrate-to-shared-social/main.go` (idempotent, `--dry-run`)
- Phase 3 : Bascule `media_repo.go` → `socialDB()` helper (fallback Player si nil), `ops/media.go` → `SharedSocialDBPath`, `notify/notifiers.go` commentaire mis à jour, handler `match_favorite.go` + route `PATCH /players/{slug}/matches/{id}/favorite`, `port.SocialRepository`, `port.SocialService`, `service.SocialService`, `platform/duckdb.SocialRepo`
- Phase 4 : Migrations `drop_media_from_player_db` (stats.duckdb) et `drop_media_likes_from_shared` (shared_matches_v2.duckdb). ATTACH `shared_matches_v2` sur connexion SharedSocial pour que Q37 continue de joindre `shared.match_registry`
- Build : `go build ./...` ✅, `go vet` : sortie propre

**Conclusion** : Architecture sociale proprement isolée. Les vrais DROP ne s'exécutent qu'au prochain démarrage via `launcher.py → _run_migrations()`, après que le script de migration manuelle a copié les données.



### Décisions techniques

1. **Faire un incrément léger mais visible** — la hauteur du visuel passe de `h-32 / sm:h-44 / lg:h-52` à `h-36 / sm:h-48 / lg:h-56`, soit environ +8 à +12% selon le breakpoint.
2. **Conserver le reste du comportement inchangé** — sticky, radius et recouvrement par le contenu restent identiques ; seule l'emprise verticale de l'image augmente.

## [2025-07-17] feat(arch): Sprint 38+39 — DRY, split fichiers >500L, tests couverture

**Statut** : Complété

**Décision technique** :
- Sprint 38 T1-T6 : découpage de 5 fichiers >500L en modules à responsabilité unique ; `domain/outcomes.go` centralise les constantes outcome ; double switch `feature_flags` éliminé via `surfaceFields()` retournant `map[Surface]*Backend` (pivot unique)
- Sprint 39 T1 : 5 fichiers `*_test.go` handlers ajoutés (patterns OK/404/500 avec mocks DI) ; T3 : tests TrueSkill purs + transforms helpers (8 fonctions couvertes)
- Contrainte DuckDB Windows pré-existante : packages `sync` et `handlers` ne compilent pas en local (build constraint `windows-amd64`) mais les tests sont prêts pour CI

**Résultats** :
- `go build ./internal/analysis/...` → clean ✅
- `go test ./internal/analysis/...` → OK (0.193s) ✅
- `go vet ./internal/sync/... ./cmd/levelup/...` → seule contrainte DuckDB attendue ✅
- Aucun fichier créé > 310L ; `queries.go` de 731L → 5 fichiers < 280L chacun
- Double switch feature_flags → 1 seule map lookup dans `surfaceFields()`

**Conclusion** : Sprint 38 100% complété. Sprint 39 : T1+T3 ✅, T2/T4/T5 restants (DuckDB in-memory fixtures + FastAPI + coverage ≥ 50%).

---

**Statut** : Complété

**Décision technique** : Pattern DI avec types génériques `ServiceFactory[S]` et `ContextFactory[S]` définis dans le package handlers. Le `ServiceRegistry` (api/registry.go) centralise la construction des services à partir du `PlayerResolver`. Les handlers reçoivent des fonctions factory typées — aucun couplage direct avec `config`, `platform/duckdb` ou `service`.

**Résultats** :
- 16/21 handlers convertis au DI (tous les player-scoped). 5 handlers non convertis (infrastructure : health, bootstrap, auth, settings, sync) — ont déjà une injection propre.
- 18 interfaces service créées dans `port/services.go`
- `ProfileService` extrait de `setup.go` → `service/profile_service.go`
- `server.go` câblé via `ServiceRegistry` — les handlers ne connaissent plus que les interfaces `port.*`
- Test mock `career_test.go` démontre le pattern (3 cas : OK, 404, 500)
- Gamertag handler reçoit directement un `port.GamertagSearchService` (service global, pas de résolution joueur)
- Explorer handler utilise 2 factories (ExplorerService + MatchHistoryService)

**Prochaine étape** : Sprint 38 — DRY + split fichiers >500L

## [2025-07-18] docs(go-migration): Architecture hexagonale formelle — GO_ARCHITECTURE_RULES.md

**Statut** : Complété

**Tâche** : Créer un document d'architecture logicielle contraignant pour le backend Go, suite à l'audit qui a révélé que l'architecture hexagonale n'était pas formalisée dans le corpus.

**Décisions techniques principales** :

1. **5 couches formalisées** : `domain/` (pur, 0 IO) → `port/` (interfaces) → `service/` (orchestration) → `api/` (transport) ← `platform/` (implémentations) ← `cmd/` (composition root).
2. **Matrice d'imports** : direction des dépendances enforced par linter `depguard` en CI. Règle fondamentale : les dépendances pointent vers l'intérieur.
3. **5 interfaces Go obligatoires** mappées depuis les 3 protocols Python : `PlayerRepository` (DataRepository), `SharedRepository` (nouveau), `HaloClient` (HaloAPIPort), `SyncEngine` (_SyncProtocol), `TokenStore` (éclaté MSAL+sync_meta). Plus 3 additionnelles : `MigrationRunner`, `JobStore`, `MediaIndexer`.
4. **Constructor injection stricte** : zéro globales métier, mocks via constructeurs, `cmd/` seul point d'instanciation concrète.
5. **Config `.golangci.yml`** prête avec rules `depguard` par couche (domain-purity, port-purity, service-no-platform, api-no-platform).
6. **Layout Go révisé** : `cmd/levelup/` (binaire unique), `internal/{domain,port,service,api,platform}/`.
7. **Exceptions documentées** : uniquement via `// ARCH-EXCEPTION: <raison>` — toute dérogation doit modifier le doc.

**Résultats** :
- [GO_ARCHITECTURE_RULES.md](.ai/go_migration_v2/GO_ARCHITECTURE_RULES.md) créé (~370 lignes, 10 sections)
- Référencé dans PLAN (lecture obligatoire #7 + encart dans Règles de conception), CHARTER (encart dans Architecture cible minimale), README (table source de vérité + liste de lecture #2 + références exhaustives #7)

**Conclusion** : Les 6 lacunes identifiées par l'audit sont désormais comblées. L'architecture hexagonale est contraignante, enforced en CI, et vérifiable par sprint.

## [2025-07-18] docs(go-migration): Revue et correction exhaustive du corpus v2 (19 documents)

**Statut** : Complété

**Tâche** : Revue de l'intégralité du plan de migration Python→Go (19 documents dans `.ai/go_migration_v2/`), identification des erreurs factuelles, et correction masse.

**Décisions techniques principales** :

1. **LOC vérifiés vs codebase réelle** : les estimations initiales étaient sous-évaluées de 2-5×. Total corrigé : ~55K LOC Python (analysis=14K, sync=13K, api=12K vs plan initial ~25K).
2. **12 mixins** (pas 11) : `MatchProcessingHelpersMixin` manquait. **96 champs SyncScope** (pas 94).
3. **Bridge SPNKr supprimé** : décision utilisateur de passer directement au client Go natif dès S11, sans bridge Python transitoire.
4. **Sprint 9 splitté** en S09 (Sessions) + S10 (Stats/Séries + perf score + LUSR). Réindexation S00-S28 (29 sprints).
5. **Config native Go** : struct Go + JSON + env vars, pas de viper. Binary size : 100-200 MB (CGo+DuckDB statique).
6. **9 items manquants ajoutés** : CI/CD (GH Actions build matrix CGo), config native Go, SSE sync progress, pagination cursor-based, CORS, hot reload (Air), binary size, pool multi-joueurs dégradation, versioning `-ldflags`.

**Documents modifiés** : MATRIX.md, SPRINT_ROADMAP.md, GO_MIGRATION_CHECKLIST.md, ZERO_PYTHON_STRATEGY.md, PLAN_MIGRATION_PYTHON_TO_GO_V2.md, OPS_COMPAT_CHECKLIST.md, PROGRAM_CHARTER.md, PORTING_REFERENCE.md, ZERO_PYTHON_TARGET.md, HALO_PROVIDER_ERROR_TAXONOMY.md (10/19).

**Conclusion** : Le corpus est maintenant cohérent avec la codebase réelle et les décisions utilisateur. Prêt pour Sprint 0.

## [2025-07-15] feat(polish): composants natifs react A2/A3/A5/B3/C3/C4/C5/D1/D2/E1

**Statut** : Complété

**Tâche** : Finaliser les composants UI/UX natifs listés dans `NATIVE_COMPONENTS.md` pour le dashboard React (LevelUp-no-streamlit).

**Décisions techniques principales** :
- A2/A3 : `CareerTopMatchesTable` réécrit avec colonnes K/D/A, badges DOM/HUMILIATION/etc., navigation clic, split `variant="best"|"worst"`, thème Halo dark.
- A5 : `MatchScoreboard` créé — highlight min/max (vert=max, rouge=min, inversé pour `deaths`/`damage_taken`), badges MVP/LVP, tri par équipe.
- E1 : `PlayerDetailPanel` — panneau collapsible par clic ligne (▸/▾) dans `MatchScoreboard`, affiche 14 stats + armes/médailles/citations si `is_me=true`.
- C3/C4/C5 : `MatchStatCards` (`StatExpectedCard`, `MatchRankBadge`, `KdIndicatorCard`) — stats attendues vs réelles, badge de rang, ratio K/D vs nemesis.
- B3 : `CitationsPage` — grille responsive CSS (`grid-cols-4 sm:grid-cols-6 lg:grid-cols-8`) remplace la `<table>`, triée par `count_filtered DESC`.
- D1 : `TimeseriesPage` — 3 `DeltaCard` dans l'onglet Forme (pente K/D, pente Win Rate, R²) depuis `regression_stats` calculées dans le service backend.
- D2 : `SessionComparePage` — 4 `DeltaCard` au-dessus du tableau de métriques (K/D, Win Rate, Kills/match, Score).
- Composant transversal `delta-card.tsx` créé dans `components/ui/`.
- Backend : ajout de `MatchExpectedStats` dans `match_view.py`, `TimeseriesRegressionStats` dans `timeseries.py`, population dans `timeseries_api_service.py`.
- Import dupliqué `TimeseriesRegressionStats` nettoyé (retiré du niveau fonction → hissé dans l'import top-level).

**Résultats observés** :
- 12 fichiers TypeScript modifiés/créés — 0 erreur de compilation.
- 3 schémas backend Pydantic mis à jour sans breaking change (champs optionnels avec defaults).
- Task 10 (suppression fichiers Streamlit résiduels) reportée — nécessite validation manuelle.

**Conclusion / prochaine étape** :
- Task 10 (optionnelle) : supprimer `streamlit_app.py`, `streamlit_app_v7.py` à la racine de `LevelUp-no-streamlit` et les pages Streamlit pures dans `src/ui/pages/` (garder les modules `_data.py`, `_logic.py` importés par les services API).
- Prochaine feature : implémenter A4, A6, B1, B2, D3, D4 (non prioritaires, post-MVP).

## [2025-07-14] feat(v7-onboarding): sprints 4.4 + 5.2 — audit logs et cleanup endpoints legacy

**Statut** : Complété

**Tâche** : Finalisation des sprints 4 (hardening) et 5 (cleanup) du plan V7 onboarding.

**Décision technique principale** :
- Sprint 4.1/4.2/4.3 découverts déjà implémentés (CSRF + rate limit + cookie security déjà en place)
- Sprint 4.4 : ajout de `initial_sync_started` et `initial_sync_succeeded` dans `sync_service.py` (les logs device_flow_* étaient déjà présents)
- Sprint 5.2 : suppression complète des endpoints legacy (`GET /setup/status`, `POST /setup/smoke-test`) + functions associées dans service/schema + tests legacy supprimés
- Sprint 5.3 (déféré) : `_has_any_synced_matches()` supprimé naturellement lors du nettoyage de `get_setup_status()`. Le fallback dans `bootstrap_service.py` reste pour la migration.

**Résultats** : 163/163 tests API passent (test_media.py exclu — échec pré-existant).

**Fichiers modifiés** :
- `apps/api/app/services/sync_service.py` — logs `initial_sync_started` + `initial_sync_succeeded`
- `apps/api/app/services/setup_service.py` — supp. `get_setup_status`, `get_setup_status_demo`, `start_smoke_test`, `_run_smoke_test_bg` + helpers privés liés
- `apps/api/app/routers/setup.py` — supp. `GET /setup/status` et `POST /setup/smoke-test`
- `apps/api/app/schemas/setup.py` — supp. `SetupStatusResponse`, `SetupAuthInfo`, `SetupPlayerInfo`, `SmokeTestStartRequest`
- `tests/api/test_setup.py` — supp. tests legacy setup/status + smoke-test

## [2025-07-26] docs(migration): alignement complet des docs migration sur les sections V7

**Statut** : Complété  
**Branche** : `feature/remove-streamlit-ui`

**Décision technique** :
- Audit croisé de 6 docs migration (SLICES, PARITY_MATRIX, API_CONTRACTS, INVARIANTS, DECISIONS, FUNCTIONAL_SPECS) — 13 incohérences identifiées (4 🔴 structurelles, 6 🟡 manquantes, 3 🟠 à clarifier).
- Réalignement systématique de tous les docs sur les 8 sections V7 réelles au lieu de l'ancien découpage par pages Streamlit.

**Résultats** :
- **SLICES.md** : Slices 2-8 restructurés par section V7 avec phases (A/B/C), table de correspondance Slices↔V7, query keys complètes, DoD V7
- **PARITY_MATRIX.md** : matrice synthétique V7, fiches regroupées sous headers V7 (Profil, Stats, Explorer, Accueil, Escouade, Synthèse, Médias), duplicatas supprimés (Citations/Timeseries/Session Compare standalone), Objective Analysis marqué absorbé, tests de parité par section V7
- **API_CONTRACTS.md** : sections renommées V7, Slice 5 fusionné dans Slice 4 (Explorer Phase B/C), 7 contrats placeholder ajoutés (Citations, Timeseries, Session Compare, Accueil+BattlePass+Challenges, Escouade, Synthèse, Médias), note `v_weapon_kills`, décision KPI Bar, `objective-analysis` query key supprimée, `battlepass`/`challenges` query keys ajoutées
- **INVARIANTS.md** : routes canoniques V7 (`/profile/career`, `/stats/history`, `/squad`, `/synthesis`, `/media`)
- **DECISIONS.md** : arbre routes V7 + features V7 dans §4 Structure repo
- **MIGRATION_MASTER.md** : listes de lecture réalignées, scope MVP corrigé (Accueil P2 pas P1), refs post-MVP enrichies

**Issues audit résolues** : 13/13
- 🔴 #1 SLICES old pages → ✅ V7 sections
- 🔴 #2 PARITY_MATRIX old structure → ✅ V7 sections
- 🔴 #3 API_CONTRACTS old slices → ✅ V7 sections
- 🔴 #4 DoD divergence → ✅ unifié V7
- 🟡 #5 Post-MVP no contracts → ✅ 7 placeholders
- 🟡 #6 L2 Header contract → ✅ KPI Bar décision dans API_CONTRACTS
- 🟡 #7 weapon_kills dep → ✅ note v_weapon_kills
- 🟡 #8 Battle Pass/Challenges → ✅ endpoints + query keys
- 🟡 #9 KPI Bar → ✅ décision provisoire (FilterContextResolved)
- 🟡 #10 Likes localStorage → ✅ documenté dans Slice 8
- 🟠 #11 Objective Analysis → ✅ absorbé (Escouade radar + Synthèse)
- 🟠 #12 Explorer includes Match View → ✅ phases A/B/C
- 🟠 #13 Routes V7 → ✅ toutes les routes mises à jour

**Prochaine étape** : constituer le corpus `tests/fixtures/ref_player/` + `tests/parity/`, puis scaffolder `apps/api/` et `apps/web/`

## [2025-07-16] Réécriture test_media_filters_v64.py — Complété

**Statut** : Complété

**Décision technique :** `_apply_media_filters` a perdu les filtres `kinds`, `name`, `outcome_codes` lors du sprint précédent (suppression UI). La suite `TestApplyMediaFilters` référençait ces clés dans `_base_filters()` → 7 tests échouants. Réécriture complète du fichier : nouvelle `_base_filters()` sans les clés obsolètes, remplacement des 7 tests supprimés par 8 tests de filtres actuels (map/mode/squad/apply_match_filters) + 8 tests d'idempotence (`TestApplyMediaFiltersIdempotence`) + 2 tests de constantes (`TestConstants`). Ajout de `maintain_order=True` sur `unique()` + tri secondaire stable sur `file_path`.

**Résultats** : 31/31 tests passent. Suites régressives sprint5/sprint6 : 5/5 OK.

**Conclusion** : Tests alignés avec l'interface réelle de `_apply_media_filters`. Idempotence couverte.

---

## [2025-07-16] Fix navigation Media → Explorer (deep-link match_id) — Complété

**Statut** : Complété

**Décision technique :** Le flux Media → Explorer (bouton "Match" dans la bibliothèque médias) était cassé : `_consume_deep_links()` dans `explorer.py` ne lisait que `_pending_match_id` depuis `session_state`, mais `consume_pending_match_id()` (appelé dans le même run que `st.switch_page`) avait déjà consommé cette clé et créé `match_id_input`. Après le `st.switch_page` (rerun 3 → Explorer), `_pending_match_id` était absent et `match_id_input` non lu → `pending_mid` vide → `show_single_match` jamais appelé.

**Flux complet corrigé (3 reruns)** :
1. Bouton cliqué → `open_match_button()` pose `st.query_params["page"]="Match"` + `st.query_params["match_id"]=mid` → `st.rerun()`
2. Routing : `_parse_query_params()` lit URL → pose `_pending_match_id` + `_pending_page` → vide URL → `consume_pending_match_id()` transfère vers `match_id_input` → `st.switch_page(explorer)`
3. Explorer : `_consume_deep_links()` pop `match_id_input` (ou `_pending_match_id` en fallback) → `show_single_match()`

**Fichiers modifiés** :
- `src/ui/pages/explorer.py` : `_consume_deep_links` — ajout fallback `match_id_input` + `.strip()` inline
- `tests/test_media_to_explorer_navigation.py` : docstring corrigée (flux query_params) + classe `TestOpenMatchButton` (4 tests)

**Résultats** : 18/18 tests passent. Ruff OK.

**Conclusion** : Navigation fonctionnelle. Les 3 chemins sont testés : (1) flux normal via `match_id_input`, (2) fallback via `_pending_match_id` (switch_page interrompt), (3) bouton `open_match_button` avec query_params.

---

## [2025-07-25] Navigation pleine largeur — Complété

**Statut** : Complété

**Décision technique** : Injection CSS via `static/styles.css` (chargé par `load_css()` au démarrage).
Ajout de 3 règles ciblant `div[data-testid="stSegmentedControl"]` :
- conteneur à `width: 100%`
- groupe interne en `display: flex; width: 100%`
- chaque `<label>` avec `flex: 1 1 0` pour répartition égale

**Résultat** : Barre de navigation (st.segmented_control) occupe toute la largeur disponible, onglets équidistants.

**Conclusion** : Modification minimaliste et non-invasive. CSS appliqué globalement via le mécanisme existant.

---

## [2025-07-16] Sprint 9 + Sprint 10 — Sessions + Performance Score + LUSR + Stats Series

**Statut** : Complété

**Décision technique principale** :
Port complet des algorithmes Python en Go dans le package `analysis/` :
- `ComputeSessions` (gap-based) + `ComputeSessionsWithContext` (friends+ranked) depuis `src/analysis/sessions.py`
- `ComputeRelativePerformanceScore` v5-relative (10 métriques, percentile rank) depuis `src/analysis/_performance_relative.py`
- `ComputeSkillRatingsBatch` (TrueSkill-inspired LUSR) depuis `src/analysis/skill_rating.py`
Note critique : utiliser `create_file` plutôt que heredoc bash pour les fichiers Go → les heredoc corrompent les lignes contenant des commentaires français ou des patterns `if v, ok := ...`.

**Fichiers créés** :
- `internal/domain/sessions.go` — SessionMatchRow, SessionComputeOptions (renommé depuis SessionOptions pour éviter conflit avec filters.go), BucketType, SessionsResponse
- `internal/domain/stats.go` — StatsMatchRow, LUSRMatchRating, ParticipantRow, 5 types tab response, StatsPageResponse
- `internal/analysis/sessions.go` — 2 modes de calcul + grouping + labeling + GetBucketInfo
- `internal/analysis/sessions_test.go` — 11 tests unitaires (tous verts)
- `internal/analysis/performance_score.go` — score relatif percentile + fallback KDA
- `internal/analysis/skill_rating.go` — TrueSkill update + composite score + normCDF/PDF/InvCDF
- `internal/platform/duckdb/sessions_repo.go` — LoadSessionMatches (Q22)
- `internal/platform/duckdb/stats_repo.go` — LoadStatsMatches (Q23) + LoadLUSRHistory (Q24) + LoadMatchParticipants (Q25)
- `internal/service/sessions_service.go` — GetSessions (2 modes)
- `internal/service/stats_service.go` — GetPage (5 onglets : win_loss, accuracy, objective, form, lusr)
- `internal/api/handlers/sessions.go` — GET /pages/sessions
- `internal/api/handlers/stats.go` — POST /pages/stats/query

**Fichiers modifiés** :
- `internal/api/server.go` — routes Sprint 9+10 ajoutées
- `internal/platform/duckdb/queries.go` — Q22-Q25 ajoutés
- `internal/port/repository.go` — SessionsRepository + StatsRepository interfaces + noop impls

**Résultats observés** :
- `go build ./...` → PASS (0 erreurs)
- `go test ./internal/analysis/...` → 11/11 PASS
- Commit : `fd721220` sur `feature/go-migration`

**Conclusion** :
Sprint 9+10 complets. Architecture clean layer: domain → analysis → platform/duckdb → service → handlers.
Prochaine étape selon SPRINT_ROADMAP : Sprint 11 (charting timeseries = ~30 fonctions, à planifier séparément).

---

## [2025-07-11] Sprint 43 — Améliorations UX produit

**Statut** : Complété (4/4 tâches)

### T1 — Bipolaire solo/escouade : enrichissement payload synthesis Go
- **Décision** : enrichir SynthesisPageResponse avec SynthesisKPIs (performance_score, accuracy, kills_per_min, avg_life_seconds), ComparisonMetricItem (bipolaire), et TemporalHeatmapCell (dow x hour au lieu de map x mode)
- **Fichiers modifiés** : domain/squad.go, queries_squad.go (Q33b +3 colonnes), squad_repo.go (scan 10 cols), squad_breakdown.go (+3 fonctions), squad_service.go (GetSynthesisPage réécrit)
- **Résultat** : go vet OK, tous tests analysis passent (42/42)

### T2 — Composant InfoTooltip React
- **Décision** : composant pur CSS/state (aucune dépendance Radix/Headless), hover+focus+click
- **Fichiers créés** : components/ui/info-tooltip.tsx
- **Intégré** : SynthesisPage (Performance Score) + CareerPage (LUSR)

### T3 — Page /changelog
- **Décision** : endpoint Go GET /api/v1/changelog lisant docs/CHANGELOG.md avec cache mémoire 5min. Frontend avec react-markdown + @tailwindcss/typography
- **Fichiers créés** : handlers/changelog.go, features/changelog/ (queries.ts + ChangelogPage.tsx), routes/changelog.tsx

### T4 — Durée session : deux métriques
- **Décision** : ajouter DurationSeconds (span) et TotalPlayedSeconds (somme time_played_seconds) à SessionGroup
- **Fichiers modifiés** : domain/sessions.go, analysis/sessions.go (BuildSessionGroups enrichi)
- **Résultat** : go vet OK, tous tests analysis passent (42/42)

**Conclusion** : Sprint 43 complet.

---

## [2025-07-11] Sprint 44 — Implémentation multi-titres (tâches T6-T20)

**Statut** : Complété (T6, T7, T8, T11, T12, T13, T15, T18, T20)

### T6 — db_profiles.json v3 title-aware
- **Décision** : format v3 avec structure `{ "version": "3.0", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }`. Rétrocompatibilité v2.1 via détection automatique de version dans `LoadPlayers()`.
- **Fichiers modifiés** : `config/config.go` (types v3 + version probe + loadPlayersV2/V3), `service/profile_service.go` (réécriture complète v3 : `parseOrMigrateProfiles`, `CreatePlayer` title-scoped), `db_profiles.example.json` (v3), `domain/settings.go` (TitleSlug field)
- **Résultat** : config backward-compatible, profile_service écrit toujours en v3

### T7 — Setup/players handlers title-aware
- **Décision** : `POST /setup/players` lit `titleSlug` depuis le contexte (middleware TitleExtractor), le passe à `ProfileService.CreatePlayer`. PathResolver pour chemins DB joueur.
- **Fichiers modifiés** : `handlers/setup.go` (imports ctxkeys/title, injection titleSlug, PathResolver pour dbPath)

### T8 — Ops PathResolver migration
- **Décision** : migrer `healthcheck.go` et `gate.go` de `filepath.Join(root, "data", ...)` vers `PathResolver.Legacy*` methods (rétrocompatibilité layout plat).
- **Fichiers modifiés** : `ops/healthcheck.go` (PathResolver pour config/warehouse), `validation/gate.go` (PathResolver pour shared DB, metadata DB, player DB)

### T13 — Routage OpenAPI {title_slug}
- **Décision** : approche **header-only** (`X-LevelUp-Title`). Le middleware `TitleExtractor` est déjà en place au niveau du router racine. URLs inchangées. Pas de segments `{title_slug}` dans les routes — architecture moins disruptive et déjà fonctionnelle.
- **Fichiers** : aucune modification nécessaire (middleware déjà appliqué globalement dans `server.go`)

### T15 — Frontend stores/routes title-aware
- **Décision** : `settingsDraftStore.lastPlayerSlug` → `lastPlayerSlugByTitle: Record<string, string | null>` (migration transparente). `appShellStore` enrichi avec `switchTitle()` (POST /session/context + re-bootstrap + resetPlayerData + rollback on error), `isTitleSwitching`, `resetPlayerData()`.
- **Fichiers modifiés** : `stores/settingsDraftStore.ts` (interface + actions + default), `stores/appShellStore.ts` (switchTitle, isTitleSwitching, resetPlayerData, import api)

### T11 — Corpus synthétique second titre
- **Décision** : script Python `create_multititle_fixture.py` créant un titre `halo_mcc` avec 5 matchs dans `data/titles/halo_mcc/warehouse/` + `data/titles/halo_mcc/players/MCCTestPlayer/stats.duckdb`
- **Fichiers créés** : `tests/create_multititle_fixture.py`

### T12 — Fixtures multi-titres
- **Décision** : 3 fichiers tests Go pour l'isolation multi-titre — PathResolver isolation (6 cas : SharedDB, MetadataDB, PlayerDB, PlayerDir, WarehouseDir, BackupDir), structure arborescence, même gamertag dans deux titres, registry isolation. Config tests v3 : title isolation, backward compat v2, empty title.
- **Fichiers créés** : `domain/title/multititle_test.go`, `config/config_test.go`

### T18 — Golden tests skeleton
- **Décision** : squelette Go natif chargeant les fixtures JSON existantes et assertant leur structure (clés requises, types). Tests de validation structurelle (pas de serveur requis) + test `AllFixturesLoadable` itérant tous les .json.
- **Fichiers créés** : `tests/golden/golden_test.go`

### T20 — Documentation finale
- **Décision** : ajout section « Multi-Title Architecture (Sprint 44) » dans `docs/ARCHITECTURE_V6.md` (EN) et `docs/FR/ARCHITECTURE_V6.md` (FR). Arborescence, composants clés, stratégie routage, frontend, rétrocompatibilité.
- **Fichiers modifiés** : `docs/ARCHITECTURE_V6.md`, `docs/FR/ARCHITECTURE_V6.md`

### Roadmap update
- Sprint 44 : 9 tâches passées à ✅ (T6, T7, T8, T11, T12, T13, T15, T18, T20)
- Gate Phase 9 : 22 items cochés sur 24 (restent coverage ≥ 50% et lint clean — à vérifier en CI)

**Conclusion** : Sprint 44 implémentation multi-titres complétée. Toutes les couches sont title-aware : backend (config, handlers, ops, validation), frontend (stores, API client), données (v3 format, PathResolver), tests (isolation, fixtures, golden skeleton), documentation. Les items restants (coverage, lint) dépendent d'un run CI complet.

---

## [2025-07-16] Sprint 49 closure — Gate Sprint 47 : Repos DuckDB ≥ 70%

**Statut** : Complété ✅

### Décision technique principale

Création de 3 fichiers de tests `//go:build integration` (`package duckdb`) pour le package `internal/platform/duckdb` :
- `player_repos_test.go` — HomeRepo, SessionsRepo, StatsRepo, CareerRepo, MediaRepo, ResolveXUID
- `match_repos_test.go` — FiltersRepo, MatchHistoryRepo, CitationsRepo, ExplorerRepo, MatchViewRepo, SquadRepo
- `extra_coverage_test.go` — fonctions 0% restantes : GetLUSRHistory, LoadMedalCitationMappings, DB.SQLDb/Path, GetMatchMedals/Events/WeaponKills/KVPairs, PoolKey, CloseAll, LoadTeammateMatches, LoadImpactEvents, LoadSynthesisHeatmap, LoadSynthesisMatches

Fix correctif : `Q33SynthesisHeatmap` `GROUP BY map_name, mode_name` → `GROUP BY 1, 2` (ambiguïté alias/colonne sous DuckDB quand la table possède un champ `map_name`).

### Résultats observés

| Étape | Couverture |
|-------|-----------|
| Baseline | 13.1% |
| Après player_repos_test.go + match_repos_test.go | 59.4% |
| Après extra_coverage_test.go | **75.4%** ✅ |

### Conclusion

Gate Sprint 47 "Repos DuckDB ≥ 70%" validé à 75.4% avec `-tags integration`. SPRINT_ROADMAP.md mis à jour.

---

## [2025-07-22] Sprint 54 Gate — LeaderboardBlock vitest + Garde-fous medals

### Statut : Complété

### Décision technique
Deux items Gate S54 restaient non implémentés :
1. **LeaderboardBlock vitest** : 8 tests React (vitest + MSW + @testing-library) vérifiant spinner, badge Local, tri par rank, CSR/tier, compteur, état vide, erreur 500, titre.
2. **Garde-fous medals** : package Go `internal/metadata` avec 3 guards purs (cardinalité ±10%, champs requis, images partielles) + `RunAllGuards` orchestrateur. 13 tests Go couvrant tous les cas.

### Résultats
- vitest : 8/8 PASS (1.53s)
- Go tests : 13/13 PASS (0.19s)
- Gate S54 : 2 items cochés supplémentaires

### Fichiers créés
- `apps/web/src/features/leaderboard/LeaderboardBlock.test.tsx` (8 tests)
- `apps/web/src/test/handlers.ts` (handler MSW leaderboard ajouté)
- `apps/go-api/internal/metadata/medals_guard.go` (guards purs)
- `apps/go-api/internal/metadata/medals_guard_test.go` (13 tests)
- `apps/go-api/internal/metadata/medals_guard_test.go` (13 tests)

---

## [2025-07-16] Vérification finale — Xbox/Steam Presence Watcher (feat/match-favorites)

**Statut** : Complété

### Décision technique principale
Finalisation et validation du daemon de présence Xbox/Steam introduit dans la session précédente. Correction d'une entrée `go.sum` manquante pour `gorilla/websocket` et `x/crypto` (détectés uniquement lors du `go test`, pas lors du `go build`).

### Actions réalisées
1. **go.sum** : `go get github.com/gorilla/websocket@v1.5.3` + `go get levelup/go-api/internal/platform/userstore` + `go mod tidy`
2. **7 fichiers de tests créés** (session précédente, validés maintenant) :
   - `internal/presence/event_parser_test.go` — 8 cas
   - `internal/presence/reconnect_test.go` — 5 cas  
   - `internal/watcher/match_queue_test.go` — 6 cas
   - `internal/watcher/state_machine_test.go` — 11 cas
   - `internal/titles/registry_test.go` — 9 cas
   - `internal/auth/token_store_test.go` — 8 cas
   - `internal/sync/coordinator_test.go` — 6 cas
3. **Audit logging** : `slog` standardisé, dead code supprimé dans `presence_notifier.go`

### Résultats
- `go build ./... && go vet ./...` : 0 erreur (packages watcher)
- `go test` sur 10 packages : **100% PASS** (presence, watcher, auth, titles, sync, notify, analysis, domain)
- Packages WIP avec erreurs préexistantes (`config_cache.go`, `reward_tracks_provider.go`, `bootstrap_service.go`) : non touchés, erreurs antérieures à cette session

### Conclusion / prochaine étape
Le daemon de présence Xbox/Steam est complet, compilable et testé. 53 cas de test couvrent toute la logique pure des nouveaux packages.

---

## [2025-07-29] — Indicateur sync en cours dans NavL1 (SyncStatusDot)

**Statut** : Complété

**Décision technique principale** :
Utiliser le système de jobs asynchrones existant (`activeSyncJobId` / `useJobStatus`) plutôt que le watcher de présence Xbox (sur une autre branche). `activeSyncJobId` était déjà dans `appShellStore` et hydraté depuis bootstrap — il manquait uniquement le setter et le composant visuel.

**Fichiers modifiés** :
- `apps/web/src/stores/appShellStore.ts` : ajout de `setActiveSyncJobId(id: string | null) => void` dans l'interface et l'implémentation
- `apps/web/src/components/shell/NavL1.tsx` : import `useJobStatus`, composant `SyncStatusDot` (SVG `animate-spin`), intégration dans les deux zones gamertag (single et multi-joueurs)

**Résultats observés** :
- `tsc --noEmit` : 0 erreur TypeScript
- `vitest run NavL1.test.tsx` : 3/3 tests passent

**Conclusion** :
La feature est 100% terminée. Un spinner tourne à côté du gamertag dans la L1 quand `active_sync_job_id` est non-null (retourné par bootstrap ou par POST /sync). Il disparaît automatiquement quand le job passe en état terminal via `useEffect` + `setActiveSyncJobId(null)`.

---

## [2025-07-29] — Suppression complète Python legacy

**Statut** : Complété

**Décision technique principale** :
Après confirmation que sync DuckDB, auth MSAL/XSTS, API Halo (SPNKr) et launcher sont portés en Go, suppression de tout le Python legacy sur la branche `feat/timeseries-rendering-fixes`.

**Fichiers supprimés** :
- `src/` (code Python métier entier)
- `scripts/` (98 scripts Python)
- `spnkr_pr/` (fork SPNKr)
- `launcher.py` (appelait uvicorn supprimé)
- `tests/` (tests pytest src/)
- `levelup_halo.egg-info/` (build artifact)
- `packaging/build_release.py` + `packaging/__pycache__/`
- `pyproject.toml`
- `apps/go-api/scripts/benchmark_python_api.py`, `export_fastapi_openapi.py`, `diff_openapi.py`, `capture_golden_values.py`, `parity_check.py`
- `.github/workflows/e2e-browser-optional.yml` (testait Streamlit)

**Fichiers modifiés** :
- `Makefile` : suppression variables Python/LAUNCHER, cibles Python (install, run, test, etc.), GO_VERSION lit `VERSION`
- `.pre-commit-config.yaml` : hook check-code-size (enforce_size_limits.py) retiré
- `.github/workflows/ci.yml` : 5 jobs Python supprimés (fast-data-contracts, test, lint, quality, go-golden-test)
- `.github/workflows/bump-version.yml` : réécrit pour utiliser `VERSION` au lieu de pyproject.toml
- `.github/workflows/deploy.yml` : validation Python + prepare_demo_data.py remplacé par placeholder `levelup seed`
- `.github/workflows/test-deploy-precheck.yml` : toutes les étapes `python -c` remplacées par shell
- `.github/workflows/release.yml` : job `build-releases` Python supprimé, `needs` et `files` nettoyés
- `apps/go-api/internal/service/timeseries_service.go` : correction types pointeurs FilterMatchRow (bug WIP branche)

**Résultats observés** :
- `CGO_ENABLED=1 go build ./...` : EXIT 0 (succès)
- Aucune référence Python dans les workflows CI restants

**Conclusion** :
Go est désormais l'unique runtime. ~900 fichiers Python supprimés. Le dépôt est prêt pour un commit de clôture.

---

## 2025-07-13 — Portage timeseries Go — rendu charts client-side

**Contexte** : Sprint 33+. `PlotlyFigurePayload = null` dans toutes les réponses Go (décision architecturale). La page "Stats en solo" n'affichait que des `EmptyStateNotice` dans tous les onglets.

**Décision architecturale** : Construire les charts côté React depuis les données brutes (arrays typés) — pas de régénération de `PlotlyFigurePayload` dans le Go.

**Fichiers modifiés — Go** :
- `apps/go-api/internal/domain/timeseries.go` : `TimeseriesMatchRow` ajouté, `CorrelationDataPair.Outcome *int`, 3 nouveaux champs buckets dans `TimeseriesDistributionsTab`, `MatchRows []TimeseriesMatchRow` dans `TimeseriesPageResponse`
- `apps/go-api/internal/service/timeseries_service.go` : alpha EWMA 0.1→0.20, `buildMatchRows`, `buildAccuracyBuckets`, `buildScorePerMinBuckets`, `buildRollingWRBuckets`, `buildCorrelationPoints` étendu à 6 types avec Outcome, `filterStatsMatchRows` branché dans `GetPage`
- `apps/go-api/internal/service/timeseries_service_test.go` : 6 nouveaux tests (buildMatchRows, buildAccuracyBuckets×2, buildCorrelationPoints, filterStatsMatchRows×2)

**Fichiers modifiés — TypeScript / React** :
- `apps/web/src/lib/api/types.ts` : 5 nouvelles interfaces (CumulativePoint, DistributionBucket, CorrelationDataPair, IntensityHeatmapPoint, TimeseriesMatchRow), interfaces existantes étendues
- `apps/web/src/features/timeseries/TimeseriesPage.tsx` : tous les onglets câblés sur les nouveaux composants (suppression PlotlyChart fallback)
- `apps/web/src/components/ui/timeseries-line-chart.tsx` : CRÉÉ — multi-séries Plotly ligne (cumul/rolling/EWMA)
- `apps/web/src/components/ui/timeseries-histogram.tsx` : CRÉÉ — barres depuis DistributionBucket[]
- `apps/web/src/components/ui/timeseries-heatmap.tsx` : CRÉÉ — heatmap 7×24 jour/heure
- `apps/web/src/components/ui/timeseries-scatter.tsx` : CRÉÉ — scatter multi-type avec sélecteur de label
- `apps/web/src/components/ui/timeseries-kda-bars.tsx` : CRÉÉ — timeline K/D barres+ligne par match

**Résultats observés** :
- `go build ./...` : EXIT 0
- `go test ./internal/service/... -run "Timeseries|BuildMatchRows|BuildAccuracy|BuildCorrelation|FilterStats"` : 13/13 PASS

**Conclusion** : Tous les onglets de la page timeseries ont désormais des composants de rendu fonctionnels. Le Go fournit les données brutes, React construit les figures Plotly client-side.

---
