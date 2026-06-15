# Tracker — Title-agnostic (fenêtre minimale viable 0→3a)

> **Rôle** : suivi opérationnel **traçable** de la fenêtre minimale viable du
> [PLAN_TITLE_AGNOSTIC_REFACTORING.md](PLAN_TITLE_AGNOSTIC_REFACTORING.md) (master, v2.5).
> Ce fichier = vue « qu'est-ce qui est fait ou non », pas un re-plan. Détails/justifications dans le master.
>
> **Créé** : 2026-06-02 · **Méthode** : audit par phase dans le code réel (6 agents Explore).
> **Légende statut** : ✅ done · 🟡 partial · ⬜ todo.

## Décisions cadrantes (validées avec Guillaume, 2026-06-02)

- **Voie A** : un 2e titre est au backlog → finir le chemin canonique est l'investissement ROI-positif.
- **Fenêtre = minimale viable (Phases 0→3a)** : « services title-agnostic + DTO propres + feature-matrix », **Huma (3b) différé**.
- **OpenAPI** : [PLAN_WEB_API_TYPES_MIGRATION.md](PLAN_WEB_API_TYPES_MIGRATION.md) est **absorbé dans la Phase 3b (Huma)** — on **ne** grind **pas** la réconciliation manuelle d'`openapi.yaml` (Huma auto-génère le contrat depuis les types Go). PLAN_WEB_API_TYPES reste « foundation posée » et redevient actif seulement si 3b glisse >2 trimestres.

## Réconciliations vs master (le master date du 2026-05-18, le code a avancé)

1. **ADR renumérotés** : le master cite 0014-0017 mais ces numéros sont **pris** (et 0020/0021/0024 ont des doublons). → **0025 (title-agnostic, ✅ créé), 0026 (Huma), 0027 (feature-matrix), 0028 (title-diagnostic)**. Cf. [ADR 0025](../docs/adr/0025-title-agnostic-minimal-viable-window.md).
2. **Phase 2 — `MatchFieldRepository` (FieldKey-map D1/D2) SUPERSÉDÉ ✅ acté (ADR 0025 D-MV2)** : cible = **repos canonical-typés** (`PlayerMatchesRepository.LoadPlayerMatches → canonical.PlayerMatchRow`, alias « P4.3 »). 5 services le consomment déjà, **aucun service n'importe `platform/duckdb` pour la data** (sauf `home_service` pour le type `PersistSink`, non-data). On **ne construit pas** le FieldKey-map. Phase 2 résiduelle = finir explorer/career + isoler le PersistSink de home_service + `Load*` stubs au cas par cas.
3. **Phase 1 `fields.toml` sémantique ≠ physique** : le master voulait un mapping physique table→colonne pour alimenter le FieldKey-map. Avec (2), cet item devient **caduc** ; `fields.toml` (labels/units/format) suffit.

## Dashboard

| Phase | Objet | Statut | ~% | Bloque la suite ? |
|---|---|:-:|:-:|---|
| 0 | Décisions + setup (ADR/branche/lints/datasets) | 🟡 | 55 | non — reste = items **prématurés** (datasets/parity/chi-lint → leurs phases) + branche (bloquée user) |
| 1 | FieldKey + `fields.toml` (+ constants.toml) | 🟡 | 80 | non — reste = SQL de-magic, **reclassé Phase 2** |
| 1.5 | DDL par titre (sortir `migration/steps_*`) | 🟡 | 58 | **oui** (2e titre) — mécanisme B complet + **6 steps migrés** (PvE + 5 Shared additifs/backfill) ; **tier A Shared épuisé**, reste = tier B (cœur + relocation tests) |
| 1.6 | Pool tokens clé `(titleSlug,gamertag)` | ✅ | 100 | **oui** (2e titre) — livré : clé composite + garde anti-cross-title |
| 1.7a | `capabilities.toml` + loader + endpoint | ✅ | 100 | non — TOML + loader + adapter consomme + endpoint, livré |
| 1.7b | Feature-matrix 3 états + cascade | ✅ | 100 | non — cascade pure + endpoint, livré (revue adversariale 5 lentilles) |
| 2 | Services title-agnostic (canonical-typé) | 🟡 | 70 | non |
| 3a | Cleanup DTO (`*Raw` hors domain, nullable) | 🟡 | 50 | non |
| 1.8 | Outillage diag Lab | ⬜ | 0 | **différé** (hors fenêtre) |
| 1.9 | Watcher multi-title routing (présence→poll→sync) | ⬜ | 0 | **oui** (2e titre, runtime) — détection déjà title-agnostic ; reste = threader `titleSlug` (fetcher/PlayerWatcher/CoordinatorRequest) |

---

## Phase 0 — Décisions + setup · 🟡

| Item | Statut | Evidence / next action |
|---|:-:|---|
| ADR title-agnostic (**0025**) | ✅ | `docs/adr/0025-title-agnostic-minimal-viable-window.md` (acte D-MV1..4 : fenêtre 0→3a, Phase 2 canonical-typé, Huma absorbe WEB_API_TYPES) |
| ADR Huma (**0026**), Feature-Matrix (**0027**), Title-Diagnostic (**0028**) | ⬜ | à rédiger au démarrage de leurs phases respectives (3b / 1.7b / 1.8) |
| Branche `refactor/title-agnostic-services` | ⬜ | différée : recherche parallèle user en cours (CLAUDE.md règle 4) ; à créer quand la branche se libère |
| Plan référencé dans `CLAUDE.md` § ADR | ✅ | ADR 0025 + master + tracker listés (commit `2b9d9aaae`) |
| Lint `no_slug_comparison` (cœur title-agnostic) | ✅ | `internal/archlint/no_slug_comparison_test.go` (ratchet, 2 hard-gates allowlistés, sanity vérifié — commit `801d7444f`) |
| Lints `slog-context` / `no-new-chi-handler` | ⬜→**différés** | `no-new-chi-handler` = Phase 3b (tag `phase-3b-start`). `slog-context` = 283 sites existants + besoin d'AST pour être correct (ctx dispo ?) ; baseline fragile, non-spécifique title-agnostic → différé |
| Job CI `synthetic_test_title-parity` | ⬜→**Phase 1.5/2** | **prématuré** : exige le titre synthétique câblé pour tourner la suite sous `LEVELUP_TITLE=...` (dépend Phase 1.5 DDL + 2) |
| Datasets `testdata/integration/{halo_full,synthetic,openspartan_primed}` | ⬜→**Phase 2** | **prématuré** : consommés par les tests Phase 2 (snapshots, continuité OpenSpartan) ; créés quand ces tests existent (pas de scaffold vide) |

## Phase 1 — FieldKey + fields.toml · 🟡 (~60%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `canonical/fields.go` couvre les colonnes services | ✅* | 59 FieldKeys ; `match_participants` OK. `killer_victim_pairs` (killer/victim_xuid, kill_count) = **données de relation/événement, pas de stat joueur → pas de FieldKey requis** (acté) |
| `fields.toml` (loader `internal/games/mappings/`) | ✅ | existe + 59 sections + loader complet + `loader_smoke_test` valide la couverture |
| Mapping **physique** table→colonne (5 tables) | ⛔ | **caduc** (ADR 0025 D-MV2 : FieldKey-map abandonné, `fields.toml` sémantique suffit) |
| `constants.toml` (source title-agnostic) | ✅ | `config/titles/halo_infinite/constants.toml` créé : medal `perfect_kill` + emplacement mode prefixes |
| Test exhaustivité FieldKeys↔TOML | 🟡 | `loader_smoke_test.go` couvre TOML⊇FieldKeys ; sens inverse optionnel |
| 0 `medal_name_id=<int>` inline (SQL) | 🟡→**Phase 2** | **couplé Phase 2** : le `1512363953` est baké dans des SQL const strings ; le sortir = rendre la couche requêtes title-aware (consommer `constants.toml`). 7 sites documentés pointant vers `constants.toml` |

## Phase 1.5 — DDL par titre · 🟡 (~25%) — **PREMIER GROS MORCEAU, prérequis 2e titre**

| Item | Statut | Evidence / next action |
|---|:-:|---|
| **1.5.0 — Ordre d'exécution EXPLICITE (garde-fou)** | ✅ | `migration/order.go` (`canonicalOrder` 142 migrations) + `RunForDB` trie dessus + `order_test.go` (complétude + no-op prouvé, sanity vérifié). Déplacer un fichier ne réordonne plus. Commit `407a41c54` |
| **1.5.1 — voie B (MigrationSteps explicite, ADR 0025)** | 🟡 | **Mécanisme COMPLET (no-op prouvé)** : helpers exportés (`519f2e7c4`) + `RunSteps` primitive (`a40f28c53`) + provider `SetTitleStepsProvider`/`combineSteps`, `RunForDB` combine global+title dédup par Name, package `halo_infinite/migrations` (vide), wiring boot serveur (`ad44a5ede`). 14 appelants inchangés. **b3 en cours** : 6 steps migrés (pilote `add_pve_schema` `cd80688b9` + batch Shared additif `d1b76b7d1` + seed_tier_boundaries `29f3d5eb9`). CLIs Shared câblés (diag_bot, apply_shared_migrations). DSL exportée : 7 helpers DDL + `BootCtx` (backfills). **Tier A Shared épuisé** (les restants ont test dédié / sont cœur). **Pattern** : copier dans `Steps()` (`execScript`→`migration.ExecScript`), `git rm`, câbler les CLIs du target, `order_audit` valide. **Reste = 2 TIERS** : **(A) additif (facile)** — `ADD COLUMN`/index/table secondaire : aucun test ne les asserte → déplacement direct (CLIs Shared déjà câblés). **(B) CŒUR = CLUSTER tightly-coupled (tentative faite + revert propre 2026-06-02)** : `steps_shared.go` (982L, 34 migrations) + `steps_shared_rebuild_match_participants.go` (336L) **partagent** les helpers privés `applyResolutionViews`/`applyMvPlayerMatchesView` ; ils utilisent aussi les consts `colSmallInt`… (à exporter). Et leurs **tests dédiés** (`steps_shared_rebuild_*_test.go`, `migration_test.go`) sont en `package migration` → **cycle d'import** empêche d'y poser le provider Halo. **Stratégie validée** : (1) exporter les consts `Col*` ; (2) déplacer le cluster `steps_shared.go` + rebuild **ensemble** (les `applyXxx` co-localisés) via le pattern `registerShared` (find-replace, compilo = filet) ; (3) câbler le provider dans les tests non-cycle (persist/sync/scheduler) + relocaliser/adapter les tests `package migration` (intégration-tagués). C'est un **effort focalisé multi-fichiers**, pas un push autonome. Câblage restant par target : Metadata (seed-assists, populate-playlists, cmd/levelup), Player (repair_data_consistency + provider global via mains), SharedSocial (pool via mains) |
| `internal/migration/` sans DDL Halo | ⬜ | **53 `steps_*.go` Halo-specific** restants (~72% du dossier) — recoupe le `steps_shared.go` 982L que j'avais déféré |
| `MigrationRunner` via `TitleDataAdapter.MigrationSteps()` | ⬜ | méthode inexistante ; registre global `init()`/`Register()` (ordre = exécution, cf. `registry.go:38`) |
| `synthetic_test_title/ddl/` minimal | ⬜ | `synthetic_title_b` = stub sans DDL |
| `ops/` (backup/restore/diagnose) multi-titre | 🟡 | `backup_service` itère via PathResolver ; `restore.go`/`diagnose.go` prennent un chemin direct |

## Phase 1.6 — Pool tokens multi-titre · ✅ — **prérequis 2e titre, livré**

| Item | Statut | Evidence / next action |
|---|:-:|---|
| Clé pool `(titleSlug, gamertag)` | ✅ | `CredentialSource.TitleSlug` (stampé par `discovery.go` depuis `d.titleSlug`) → `poolImpl.titleSlug` (source unique, dérivé des sources) → `slotsByKey map[string]int` clé `gtKey(titleSlug,gamertag)` (NUL-séparé). Signatures publiques `Acquire/HasPlayer/MarkUnhealthy` **inchangées** (le pool compose la clé en interne) → zéro ripple sur les 11 callers. **Garde anti-cross-title** dans `AddOrUpdateSource` (refuse une source d'un autre titre). Tests : `pool_title_key_test.go` (3 cas : title-scoped lookup, cross-title rejeté, same-title OK). `titleSlug` vide (legacy/tests) dégrade vers clé gamertag-only = comportement historique. |

## Phase 1.7a — capabilities.toml · ✅ (100%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `capabilities.toml` par titre + loader | ✅ | `config/titles/halo_infinite/mappings/capabilities.toml` (9 caps, clés quotées) + `mappings.CapabilityMappingSet` + `LoadCapabilities*` (valide statuts, title-agnostic sur clés) + `Registry.GetCapabilities` (chargement optionnel dans `LoadFromConfigDir`). `games.CapabilityMapFromMappings` (vocabulaire clés). Commit `56dc835b7`. |
| Adapters consomment le loader (0 hardcode) | ✅ | `adapter_data.go` : `WithCapabilities(caps)` + `Capabilities()` prefer-TOML, fallback codé = **filet sécurité boot** (parité TOML⟷fallback testée → pas de drift). Câblé aux 3 sites (`server.go` boot + `ServiceRegistry.dataAdapterForPDB`/career via `WithCapabilities`). `capCareer` supprimé (downgrade runtime inline). Commit `2ea1bcd13`. |
| Endpoint `GET /title/capabilities` | ✅ | `handlers.CapabilitiesHandler` → `GET /api/v1/titles/{slug}/capabilities` (gated `MULTI_TITLE_API_ENABLED`), sert les statuts statiques du titre + ETag/Cache-Control. Tests httptest (success/404/304). |

## Phase 1.7b — Feature-matrix 3 états + cascade · ✅ (100%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `internal/domain/feature/` (Key/Status/Matrix) | ✅ | `domain/feature/feature.go` : `Key` + 8 consts, `Status` (available/degraded/unavailable) + `Available()`, `Matrix` + `Get()`. **Pur** (0 import games/DB/HTTP). |
| `port.FeatureChecker` | ⛔→**écarté (YAGNI)** | Dérivation *pure stateless* (caps→cascade) : pas d'état à mocker. Cascade = `games.ComputeFeatureMatrix(caps) → feature.Matrix` (`games/feature.go`, `featureDefinitions` partagé entre titres, primary+enhancements). Handler chi mince réutilise `CapabilitiesRegistry`. Port ajoutable si consommateur stateful émerge. |
| Loader + handler `/title/feature_matrix` | ✅ | Pas de loader `[feature]` séparé (définitions produit en Go, partagées ; la variation par titre vient de SA `capabilities.toml` 1.7a — plus simple, pas de drift). Handler `GET /api/v1/titles/{slug}/feature-matrix` (gated `MULTI_TITLE_API_ENABLED`, ETag/Cache-Control/schema_version cohérents avec frères). |
| Qualité | ✅ | **Revue adversariale 5 lentilles** (workflow `wgki7z18p` : layering/multi-titres/cascade/tests/intégration — 0 blocker, 3 major + 5 minor **tous traités**). Tests : cascade (tous états + cap absente = dégradation gracieuse + enrichissement degraded), handler httptest (success+headers/404/304/slug-vide/caps-invalides). |

## Phase 2 — Services title-agnostic · 🟡 (~70%) — **mécanisme canonical-typé (FieldKey-map supersédé)**

| Service | Statut | Evidence / next action |
|---|:-:|---|
| `synthesis_service` | 🟡 | `playerMatchesRepo.LoadPlayerMatches` (canonical) ; n'importe pas duckdb |
| `home_service` | 🟡 | idem ; importe duckdb **uniquement** pour le type `PersistSink` (non-data) — à isoler |
| `match_history_service` | 🟡 | canonical via `playerMatchesRepo` ; OK |
| `timeseries_service` | 🟡 | canonical via `playerMatchesRepo` ; OK |
| `career_service` | 🟡 | consomme `dataAdapter.LoadCareerSnapshot/LoadEncounters` ; reste XP/LUSR/CSR à canoniser |
| `explorer_service` | 🟡 | `dataAdapter` pour capability-gating seulement ; data via `ExplorerRepository` (pas encore canonical) |
| `match_view_service` | 🟡 | `dataAdapter` injecté mais data via `MatchViewRepository` (legacy) ; bascule = la plus complexe |
| ~~`port.MatchFieldRepository` (FieldKey-map)~~ | ⛔ | **supersédé** (réconciliation 2) — ne pas construire |
| 0 service importe `platform/duckdb` pour la data | 🟡 | vrai sauf `home_service` (PersistSink, non-data) |
| 7 stubs `Load*` de l'adapter | ⬜ | scoreboard/highlight = low effort ; summaries/playerstats = medium ; matchDetail/timeseries/friends = high — **à retenir ou non selon (2)** |

## Phase 3a — Cleanup DTO · 🟡 (~50%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `domain/match_view.go` sans `*Raw` | ✅ | `*Raw` isolés dans `match_view_raw.go` (split 2026-06-02) |
| `domain/match_view.go` ≤ 500 L | ✅ | 487 L |
| 16 `*Raw` déplacés **hors `domain/`** (→ `platform/duckdb`) | ⬜ | encore dans `domain/match_view_raw.go` |
| `MatchExpectedStats` 100% nullable | 🟡 | `HasExpectedData`/`HasHistAvg` (bool), `HistMatchCount` (int), `HistModeCategory` (string) non-nullable |
| `MatchScoreboardRow` 100% nullable | 🟡 | `XUID`/`Gamertag`/`IsMe`/`OutcomeLabel`/... non-nullable |

## Phase 1.9 — Watcher multi-title routing · ⬜ (0%) — **NOUVELLE 2026-06-13, prérequis 2e titre (runtime)**

> Ajoutée après audit du poller de présence en jeu (2026-06-13). Spec détaillée dans le master ([PLAN_TITLE_AGNOSTIC_REFACTORING.md](PLAN_TITLE_AGNOSTIC_REFACTORING.md) §Phase 1.9). Hors fenêtre minimale 0→3a : à exécuter quand le 2e titre est enregistré, mais la plomberie peut se poser dès maintenant derrière une garde `DefaultSlug` (testée avec un Registry à 2 titres fixtures).

| Item | Statut | Evidence / next action |
|---|:-:|---|
| Détection présence title-agnostic | ✅ | `watcher/daemon.go::makePresenceHandler` → `titleReg.MatchPresence(titleID)` → `title/matcher.go::MatchByXboxTitleID` itère tous les titres ; `TitleDescriptor` porte `XboxTitleID` + `SteamAppID` |
| `MatchFetcher` par titre (resolver) | ⬜ | 1 seul `HaloMatchFetcher` partagé (`cmd/server/main.go` ~l.1728 → `DaemonConfig.MatchFetcher`) → API Halo only. Cible : `MatchFetcherResolver.FetcherFor(slug)`, fallback Warn+Idle si titre sans fetcher |
| `PlayerWatcher.activeTitleSlug` | ⬜ | pas de champ titre ; `OnPresenceActive` ne propage pas le `td.Slug` matché |
| `TitleSlug` dans `MatchRequest`/`CoordinatorRequest`/`TriggerSync` | ⬜ | `{Gamertag, XUID, MatchIDs}` sans titre → sync sur titre défaut |
| Garde compat `DefaultSlug` + non-régression mono-titre | ⬜ | titleSlug vide → `halo_infinite` (même garde que 1.6) ; `WatcherStatus` byte-identique à 1 titre |
| Steam fallback title-aware (`MatchBySteamAppID`) | ⬜→**optionnel** | `SteamPoller` non câblé (note W8) → hors scope tant que non activé |

---

## Hors fenêtre minimale (différé, tracké pour mémoire)

- **Phase 3b — Huma** (17-25 j) : migre ~113 handlers chi → Huma, auto-génère `openapi.yaml` + client TS. **Absorbe PLAN_WEB_API_TYPES.** Démarre par le tag `phase-3b-start` + lint D13.
- **Phase 4** — sync flags génériques (FieldKey-based) (5-6 j).
- **Phase 5** — frontend canonical-aware + `<FeatureGate>` (7-9 j).
- **Phase 1.8** — outillage diag Lab (3-4 j, différable même dans la fenêtre).

## Périphérie multi-titre (audit 2026-06-14) — registre B (hors chemin data-lecture)

> Index complet : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md) (`MT-01..MT-26`). Specs : [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md) (`PMT-1..14` + `EXT-1.5/2/5`). **⚠ Pointeurs datés — RE-VÉRIFIER avant exécution.** Méthode : `expand → parity-gate → contract` + oracle double (parité Halo golden + `synthetic_test_title`).

| Phase | Objet | Axes | Sév. | Statut | Prérequis |
|---|---|---|:-:|:-:|---|
| PMT-1 | Hosts d'ingestion title-aware | MT-01 | blocker | ⬜ | racine |
| PMT-2 | Acquisition auth par titre | MT-02 | blocker | ⬜ | racine |
| PMT-3 | Scheduler/sync titleSlug threading | MT-11 | blocker | ⬜ | PMT-1/2 |
| PMT-4 | Settings par titre + config Discord | MT-04, MT-26 | major | ⬜ | PMT-3 |
| PMT-5 | Canonicalisation Outcome | MT-06 | major | 🟡 | Expand ✅ (seam int↔canon, 4bc694fd7) ; Contract (migration ~20 sites) en session dédiée |
| PMT-6 | Achievements par titre | MT-08 | major | ⬜ | PMT-1/2 |
| PMT-7 | World-stats / leaderboard par titre | MT-03 | major | ⬜ | PMT-3 |
| PMT-8 | Cycle de vie du titre (Status) | MT-22 | major | ✅ | branche `feat/multititre-peripherie` (cf27ff85f) |
| PMT-9 | Registre migrations + schema_version par titre | MT-23 | major | ⬜ | PMT-3 |
| PMT-10 | Observabilité — dimension titre | MT-05 | major | ⬜ | — |
| PMT-11 | Discord notifications (contenu) | MT-26 | major | ⬜ | — |
| PMT-12 | Garde-fous & validateurs | MT-21, MT-09, MT-12 | major | ✅ | MT-21 ✅ + MT-09 ✅ (factory player-scoped, allowlist archlint vide) + lint MT-12 ✅ (warn) |
| PMT-13 | Mineurs & bénins (décision documentée) | MT-24, MT-25, MT-20 | minor | ⬜ | — |
| PMT-14 | Admin : gestion des titres (+ réhab. Lab cassé) | MT-22 (+1.7a/b, 1.8) | major | ✅ | vol.A ✅ ; vol.C ✅ (Lab monté 2026-06-14) ; vol.B = 0 dup (atoms feature-local corrects) |
| EXT-1.5 | Extension Phase 1.5 (metadata/ops/seed/notif) | MT-16/10/18/17 | major | ⬜ | PMT-3 |
| EXT-2 | Extension Phase 2 (career/LUSR/extraction/prestige) | MT-07/15/14/19 | major | ⬜ | PMT-3 |
| EXT-5 | Extension Phase 5 (slug constants + tables Halo front) | MT-12/13 | major | ⬜ | Phase 5 |

**Bloquants 2ᵉ titre (récap)** : Phase 1.5 (DDL) + PMT-1 (hosts) + PMT-2 (auth) + PMT-3 (écriture par titre). Le reste suit.

**Clôture 2026-06-15 (branche `feat/multititre-peripherie`)** : PMT-8, PMT-12 (MT-21) et PMT-14 volet A livrés à leur Exit Gate (oracles double Halo+synthetic_b verts à chaque axe). Reste : **MT-09** (cutoffs DefaultSlug→lookup) — re-vérifié : faisable maintenant (resolver existe), pas bloqué par PMT-3 ; **lint MT-12** front ✅ livré. **PMT-14 volet C** ✅ (Lab monté + fail-closed + test anti-régression, livré 2026-06-14 sur main ; l'audit le croyait absent car il ne regardait que le diff de branche). **PMT-14 volet B** ✅ par construction : 0 duplication copy-paste d'atom ; les `StatusBadge` admin/lab/ascension sont des composants feature-local distincts (props/i18n propres) — les unifier violerait la modularité par feature. Les 3 phases ont été consolidées sur UNE branche (1 tâche = N commits) après un éclatement initial en 3 branches/worktrees (corrigé).

**Reprise PMT-5 Contract (session dédiée — cartographié 2026-06-15)** : l'Expand (seam `OutcomeMappingSet.Canonical`/`RawCode`/`SQLIsWinExpr` + `raw_code` dans outcomes.toml) est livré. Le Contract migre ~20 littéraux `outcome=2/3/1/4`. **Découverte clé** : ces sites n'ont PAS l'`OutcomeMappingSet` en portée → vrai threading, pas mécanique ; **risque data-path** (assists/LUSR/stats) → golden parité par site obligatoire. Approche d'injection :
- **Sync** : `NewSyncEngine` reçoit `repoRoot` → charger l'`OutcomeMappingSet` DANS le constructeur (`mappings.LoadOutcomesFromFile(repoRoot/config/titles/{DefaultSlug}/mappings/outcomes.toml)`, best-effort nil), stocker `e.outcomes`, threader vers `assists_model.go:70` (SQL `!= 4`) + `skill_v2_quit_penalty.go:51` (`m.outcome==4`, méthode struct par-match → passer le code DNF). Évite de toucher les 8+ appelants de `NewSyncEngine`.
- **Repos lecture (PR-b, le gros)** : `ServiceRegistry` a `titleResolver` → `resolver.Semantic(slug).Outcomes()` ; injecter l'`OutcomeMappingSet` dans les constructeurs de `compare/explorer/match_history/squad/career_encounters/match_detail/mapstats` repos, remplacer `outcome = 2` par `SQLIsWinExpr("outcome")`.
- **Citations (PR-c)** : `citations_custom.go` ×4 → `Outcomes().Canonical(ctx.Outcome)==OutcomeWin`. **Front (PR-d)** : supprimer `OUTCOME_KEY`/int-maps, dériver via `/field-mappings`.
- **Garde** : archlint ratchet `no raw outcome literal` (regex `outcome\s*[=!]=\s*[1-4]`) avec allowlist décroissante par PR. Réf spec : `.ai/PLAN_MULTITITRE_PERIPHERY.md` PMT-5 (l.268-284).

**Constat (PMT-14 vol. C) — RÉSOLU (2026-06-14, vérifié 2026-06-15)** : le Lab était cassé (backend non monté → `/lab/*` 404). Désormais **monté** dans `server.go:680-683` (les 3 routes), `requireAccess` **fail-closed** (refuse si `LoadAppSettings` échoue), test anti-régression `lab_routes_mounted_test.go` (chi.Walk sur le vrai routeur), panneau Contracts marqué « à retirer » (cutover Go fait). L'audit 2026-06-15 l'avait cru absent car il ne lisait que le diff de la branche PMT-14 (le montage vivait déjà sur main).

## Prochaines 3 actions concrètes (quand on démarre l'exécution)

1. **Acter les réconciliations** (commit doc) : créer ADR 0024 (+ noter dans 0024 que Phase 2 = canonical-typé, FieldKey-map abandonné) ; référencer le master + ce tracker dans `CLAUDE.md`. *(Phase 0, faible risque, débloque la cohérence.)*
2. **Finir Phase 1** (additif, faible risque) : `constants.toml` (sortir `1512363953` + mode prefixes des 7 fichiers) + mapper `killer_victim_pairs` ou acter qu'elles n'ont pas de FieldKey. 
3. **Cadrer Phase 1.5** (le vrai prérequis 2e titre + gros risque ordre `init()`) : décider `ddl/*.sql` vs garder des steps Go par-titre, **avec garde-rail d'ordre de migration**. À faire en passe supervisée.

> **Note de tracking** : mettre à jour les statuts (✅/🟡/⬜) + evidence à chaque PR. Une phase n'est « close » que quand son Exit Gate (master §8) est 100% DONE + tag `phase-{N}-exit`.
