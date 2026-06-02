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
| 1.7b | Feature-matrix 3 états + cascade | ⬜ | 0 | non |
| 2 | Services title-agnostic (canonical-typé) | 🟡 | 70 | non |
| 3a | Cleanup DTO (`*Raw` hors domain, nullable) | 🟡 | 50 | non |
| 1.8 | Outillage diag Lab | ⬜ | 0 | **différé** (hors fenêtre) |

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

## Phase 1.7b — Feature-matrix 3 états + cascade · ⬜

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `internal/domain/feature/` (FeatureKey/Status/Matrix) | ⬜ | inexistant (seuls `domain/chart`, `domain/title`) |
| `port.FeatureChecker` | ⬜ | inexistant |
| Loader `[data]`+`[feature]` + handler `/title/feature_matrix` | ⬜ | inexistant |

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

---

## Hors fenêtre minimale (différé, tracké pour mémoire)

- **Phase 3b — Huma** (17-25 j) : migre ~113 handlers chi → Huma, auto-génère `openapi.yaml` + client TS. **Absorbe PLAN_WEB_API_TYPES.** Démarre par le tag `phase-3b-start` + lint D13.
- **Phase 4** — sync flags génériques (FieldKey-based) (5-6 j).
- **Phase 5** — frontend canonical-aware + `<FeatureGate>` (7-9 j).
- **Phase 1.8** — outillage diag Lab (3-4 j, différable même dans la fenêtre).

## Prochaines 3 actions concrètes (quand on démarre l'exécution)

1. **Acter les réconciliations** (commit doc) : créer ADR 0024 (+ noter dans 0024 que Phase 2 = canonical-typé, FieldKey-map abandonné) ; référencer le master + ce tracker dans `CLAUDE.md`. *(Phase 0, faible risque, débloque la cohérence.)*
2. **Finir Phase 1** (additif, faible risque) : `constants.toml` (sortir `1512363953` + mode prefixes des 7 fichiers) + mapper `killer_victim_pairs` ou acter qu'elles n'ont pas de FieldKey. 
3. **Cadrer Phase 1.5** (le vrai prérequis 2e titre + gros risque ordre `init()`) : décider `ddl/*.sql` vs garder des steps Go par-titre, **avec garde-rail d'ordre de migration**. À faire en passe supervisée.

> **Note de tracking** : mettre à jour les statuts (✅/🟡/⬜) + evidence à chaque PR. Une phase n'est « close » que quand son Exit Gate (master §8) est 100% DONE + tag `phase-{N}-exit`.
