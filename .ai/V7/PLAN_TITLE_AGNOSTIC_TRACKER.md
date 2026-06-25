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
| 1 | FieldKey + `fields.toml` (+ constants.toml) | 🟡 | 80 | non — reste = SQL de-magic médaille Perfect, **reclassé adapter Halo 5** (résolution par titre, pas un Go-const à mi-chemin) |
| 1.5 | DDL par titre (sortir `migration/steps_*`) | ✅ | 100 | **DÉBLOQUÉ** — relocation voie B COMPLÈTE (b3→b26 : 150 steps statiques + 2 dynamiques title-owned) + reorder inversion skill_v2/tier_boundaries `18f3adae2` livré. **PMT-2 store cutover `873637195` ANNULÉ (2026-06-25)** : tokens account-level partagés inter-titres, store global `data/auth/watcher_tokens/` rétabli — cf. branche `fix/auth-tokens-title-agnostic`. Seule exception globale délibérée : `rebuild_match_participants_defeat_art_corruption` (boot-path cycle). PMT-9 (`743f9467c`) ajoute le routage `RunForTitleDB(slug)` + ledger par titre |
| 1.6 | Pool tokens clé `(titleSlug,gamertag)` | ✅ | 100 | **oui** (2e titre) — livré : clé composite + garde anti-cross-title |
| 1.7a | `capabilities.toml` + loader + endpoint | ✅ | 100 | non — TOML + loader + adapter consomme + endpoint, livré |
| 1.7b | Feature-matrix 3 états + cascade | ✅ | 100 | non — cascade pure + endpoint, livré (revue adversariale 5 lentilles) |
| 2 | Services title-agnostic (canonical-typé) | ✅ | 100 | **CLOSE 2026-06-17** — critère IMPORT verrouillé (lint) ; les 3 services lourds canonical-typés via l'adapter : **HIGH-C career** ✅ + **HIGH-B explorer** ✅ + **HIGH-A match_view** ✅ (events T0 complets : cadence+rôles+kill-feed+badges recalés sur le vrai début de match). Le reste (médailles/armes i18n/scoreboard/citations) est **enrichment-boundary par conception** (ADR 0011), title-specific légitime — PAS du travail en attente |
| 3a | Cleanup DTO (`*Raw` hors domain, nullable) | ✅ | 100 | **CLOSE 2026-06-19 — les 2 volets résolus comme non-actions décidées.** Volet « *Raw hors domain » = NO-OP acté (cycle d'import Go `port↔duckdb` ; restent en `domain` couche feuille). Volet **nullabilité** = **WON'T-DO/écarté** (pointer-iser `HasHistAvg`/`HistMatchCount`/`HistModeCategory`/`MatchScoreboardRow` = NO-OP front + churn-sans-valeur, interdit CLAUDE.md). `HasExpectedData` retiré (3a-B). Plus rien en attente. |
| 1.8 | Outillage diag Lab | 🟢 | — | **amorce livrée** (re-vérifié 2026-06-19) : `admin_title_diagnostic.go` (handler Huma) + Lab title-aware monté/testé (`lab_routes_mounted_test`). L'ancien « 0% différé » sous-estimait. Reste cosmétique (queryKeys Lab sans slug, vocab Waypoint) |
| 1.9 | Watcher multi-title routing (présence→poll→sync) | 🟢 | 85 | non — **threading LIVRÉ 2026-06-17** (`b265a01d9`) : `td.Slug`/`PlayerSummary.TitleSlug` → `pollerCtx` (host PMT-1) + `MatchRequest.TitleSlug` (write-path PMT-3), byte-identique mono-titre. Reste **PMT-2 auth par titre** (pool non title-scopé, inexerçable sans 2e titre) |

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
| 0 `medal_name_id=<int>` inline (SQL) | 🟡→**Halo 5 adapter** | **scopé au 2e titre** (re-statué 2026-06-19). Le `1512363953` (médaille Perfect) est baké dans ~10 SQL const strings / templates / inline builders **avec un commentaire à chaque site**. Le sortir = rendre la couche requêtes **title-aware** (résoudre depuis `constants.toml` par titre) — c'est exactement le travail de l'**adapter data du 2e titre** (Halo 5), où l'ID devient title-correct. Un Go-const à mi-chemin serait du churn (toujours Halo-spécifique) + hasard de désync (les requêtes ont des `%`/LIKE → `fmt.Sprintf` non-sûr ; paramétrer = risque de position sur 10 sites). **Décision : NE PAS dé-magicker en Go-const ; résoudre par titre lors de l'écriture de l'adapter Halo 5.** Const local `perfectKillMedalID` déjà présent (`match_view_builders_summary.go:53`). |

## Phase 1.5 — DDL par titre · ✅ (100%) — **PREMIER GROS MORCEAU LIVRÉ, prérequis 2e titre débloqué**

**Relocation 1.5.1 (voie B) COMPLÈTE (b3→b26, autonome)** : **150 steps title-owned statiques** (+ 2 dynamiques) dans `internal/games/halo_infinite/migrations/`. Les 5 tiers (metadata/player/shared/shared_social/pve) entièrement déplacés. **Seule exception globale délibérée** : `rebuild_match_participants_defeat_art_corruption` (couplé à RebuildMatchParticipantsART, boot-path — cycle). `order_audit_test.go` prouve la complétude global+title. **Reste UNIQUEMENT le reorder inversion 132/133** (le 2e irréversible — PMT-2 store cutover — a été **ANNULÉ le 2026-06-25** : tokens account-level partagés inter-titres, store global conservé, cf. branche `fix/auth-tokens-title-agnostic`). dans `games/halo_infinite/migrations/steps.go` + **2 dynamiques** (`seed_prestige_catalog_v1`, `seed_milestone_catalog_v1` via `Register*SeedMigration`, pattern de réf pour les seeds TOML — enregistrés au boot car ils lisent `config/titles/`). Tier metadata quasi épuisé (leaves + familles citation/medal/xbox/mode_name_tr/playlist_fr/weapon_labels/prestige/milestones/csr/assists/catalogue+ranked). **Reste metadata** : `add_asset_translations` + `fix_super_fiesta` (root, à faire en dernier) + world_csr_leaderboard. Puis tier-B CŒUR Shared (cluster `steps_shared.go` 982L — effort focalisé multi-fichiers) + tiers player/shared_social/pve. Irréversible restant : reorder inversion 132/133 (le PMT-2 store-cutover a été **ANNULÉ le 2026-06-25** — cf. branche `fix/auth-tokens-title-agnostic`).

| Item | Statut | Evidence / next action |
|---|:-:|---|
| **1.5.0 — Ordre d'exécution EXPLICITE (garde-fou)** | ✅ | `migration/order.go` (`canonicalOrder` 142 migrations) + `RunForDB` trie dessus + `order_test.go` (complétude + no-op prouvé, sanity vérifié). Déplacer un fichier ne réordonne plus. Commit `407a41c54` |
| **1.5.1 — voie B (MigrationSteps explicite, ADR 0025)** | 🟡 | **Mécanisme COMPLET (no-op prouvé)** : helpers exportés (`519f2e7c4`) + `RunSteps` primitive (`a40f28c53`) + provider `SetTitleStepsProvider`/`combineSteps`, `RunForDB` combine global+title dédup par Name, package `halo_infinite/migrations` (vide), wiring boot serveur (`ad44a5ede`). 14 appelants inchangés. **b3 en cours** : 6 steps migrés (pilote `add_pve_schema` `cd80688b9` + batch Shared additif `d1b76b7d1` + seed_tier_boundaries `29f3d5eb9`). CLIs Shared câblés (diag_bot, apply_shared_migrations). DSL exportée : 7 helpers DDL + `BootCtx` (backfills). **Tier A Shared épuisé** (les restants ont test dédié / sont cœur). **Pattern** : copier dans `Steps()` (`execScript`→`migration.ExecScript`), `git rm`, câbler les CLIs du target, `order_audit` valide. **Reste = 2 TIERS** : **(A) additif (facile)** — `ADD COLUMN`/index/table secondaire : aucun test ne les asserte → déplacement direct (CLIs Shared déjà câblés). **(B) CŒUR = CLUSTER tightly-coupled (tentative faite + revert propre 2026-06-02)** : `steps_shared.go` (982L, 34 migrations) + `steps_shared_rebuild_match_participants.go` (336L) **partagent** les helpers privés `applyResolutionViews`/`applyMvPlayerMatchesView` ; ils utilisent aussi les consts `colSmallInt`… (à exporter). Et leurs **tests dédiés** (`steps_shared_rebuild_*_test.go`, `migration_test.go`) sont en `package migration` → **cycle d'import** empêche d'y poser le provider Halo. **Stratégie validée** : (1) exporter les consts `Col*` ; (2) déplacer le cluster `steps_shared.go` + rebuild **ensemble** (les `applyXxx` co-localisés) via le pattern `registerShared` (find-replace, compilo = filet) ; (3) câbler le provider dans les tests non-cycle (persist/sync/scheduler) + relocaliser/adapter les tests `package migration` (intégration-tagués). C'est un **effort focalisé multi-fichiers**, pas un push autonome. Câblage restant par target : Metadata (seed-assists, populate-playlists, cmd/levelup), Player (repair_data_consistency + provider global via mains), SharedSocial (pool via mains) |
| `internal/migration/` sans DDL Halo | ✅ | **Fait** (re-vérifié 2026-06-19). Les vraies migrations Halo sont relocalisées dans `internal/games/halo_infinite/migrations/` (33 fichiers). Les ~55 `steps_*.go` restant dans `internal/migration/` sont des **stubs-breadcrumb de 5-7 L** + 2 fichiers légitimes : `steps_shared_rebuild_match_participants.go` (exception ART boot-path documentée) + `steps_shared.go` (helpers exportés retenus pour éviter un cycle d'import migration→titre, ce ne sont PAS des migrations). L'ancien « 53 restants ~72% » était PÉRIMÉ (contradisait la ligne 28/63 du même tracker, déjà ✅ 100%). |
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

## Phase 2 — Services title-agnostic · 🟡 (~78%) — **mécanisme canonical-typé (FieldKey-map supersédé)**

> **Maj 2026-06-17 (workflow readiness 9 agents + SAFE-1/2/3 + lint)** : le critère
> de complétion d'IMPORT « 0 service n'importe `platform/duckdb` pour la data » est
> ATTEINT et VERROUILLÉ (`internal/service/no_duckdb_import_test.go`). Cleared :
> home (`PersistSink`→`port.HomePersistSink`, `ca38d98b8`), skill_v2 (→`port.SkillV2Repository`,
> `39d5eeb29`), career_live ×3 (DTOs→`domain`, `4f607df65`). media ×2 (`OpenReadWrite`
> write-IO) allowlistés (`895af83f4`). **Reste = canonical-TYPING profond** (HIGH,
> décisions requises) : explorer (types cross-joueur NET-NEW), match_view (split
> canonical vs enrichment), career XP/LUSR/CSR. « pas d'import duckdb » ≠ « canonical ».

| Service | Statut | Evidence / next action |
|---|:-:|---|
| `synthesis_service` | 🟡 | `playerMatchesRepo.LoadPlayerMatches` (canonical) ; n'importe pas duckdb |
| `home_service` | 🟡 | idem ; importe duckdb **uniquement** pour le type `PersistSink` (non-data) — à isoler |
| `match_history_service` | 🟡 | canonical via `playerMatchesRepo` ; OK |
| `timeseries_service` | 🟡 | canonical via `playerMatchesRepo` ; OK |
| `career_service` | ✅ | **HIGH-C COMPLET 2026-06-17** : rang+encounters (déjà) + XP (`e1c1570ad`) + LUSR (`bfe33f625`, new `canonical.LUSRCheckpoint`) + TopMatches (`392c252f2`, new `canonical.CareerTopMatch`) canonical via l'adapter, golden parity par chemin |
| `explorer_service` | ✅ | **HIGH-B COMPLET 2026-06-17** : 3 chemins cross-titre canonical via l'adapter — recent (`d99dec588`, `canonical.RecentMatchRow`) + participant-sum (`85dea3c99`, `PlayerMatchSetStats`) + intersection (`492bf77cd`, `PlayerIntersection`/`CommonMatchRow`/`CrossKillTally`) ; câblés en prod (DI). 5 chemins enrichment-boundary (medals/weapons/i18n/primitifs) documentés non-canonicalisés |
| `match_view_service` | 🟡 | **HIGH-A 2026-06-17 — events T0 complets** : (1) `WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb))` câblé (seam Timeseries via `highlightEventsLoader`) → cadence + 8 rôles sur `canonical.HighlightEvent` **T0-corrigés** ; (2) extension `correctMatchViewEventsT0` (point unique) : `d.events` recalé via nouveau `timeline.CorrectEventRaws` → `event_time_ms` (axe X des charts KD-cumul/frag-diff/tug-scatter) + badges d'impact sur le vrai début de match ; skips `<0` (countdown) sur evtList + badges. `d.kvPairs` NON corrigé (inerte affichage : kd_timeline mort front, axe tug dérivé de la durée — vérifié par workflow front). Golden `TestMatchViewService_T0_EventsHonorRealMatchStart` + tests `CorrectEventRaws`. Reste enrichment-boundary (medals/weapons/i18n/scoreboard) |
| ~~`port.MatchFieldRepository` (FieldKey-map)~~ | ⛔ | **supersédé** (réconciliation 2) — ne pas construire |
| 0 service importe `platform/duckdb` pour la data | 🟡 | vrai sauf `home_service` (PersistSink, non-data) |
| 7 stubs `Load*` de l'adapter | ⬜ | scoreboard/highlight = low effort ; summaries/playerstats = medium ; matchDetail/timeseries/friends = high — **à retenir ou non selon (2)** |

## Phase 3a — Cleanup DTO · 🟡 (~50%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `domain/match_view.go` sans `*Raw` | ✅ | `*Raw` isolés dans `match_view_raw.go` (split 2026-06-02) |
| `domain/match_view.go` ≤ 500 L | ✅ | 487 L |
| 16 `*Raw` déplacés **hors `domain/`** (→ `platform/duckdb`) | ⛔ | **ABANDONNÉ — impossible par design.** Les *Raw croisent `port` (DTO de contrat) ; les mettre dans `duckdb` ferait `port→duckdb` alors que `duckdb→port` = cycle d'import Go. Restent canoniquement en `domain` (couche partagée). Un package `internal/dto` neutre = churn sans valeur (domain a déjà la propriété feuille). Décision 2026-06-17 |
| `MatchExpectedStats` 100% nullable | ⛔ **WON'T-DO** | **Écarté (churn-sans-valeur)** — re-confirmé 2026-06-19 vs thought_log 2026-06-18. Les stats `*float64`/`*int` sont DÉJÀ `*T omitempty`. Reste 3 champs (`HasHistAvg` bool, `HistMatchCount` int, `HistModeCategory` string) : les pointer-iser = **NO-OP front** (lu en `?? false`/truthy) + risque d'ambiguïté → interdit par CLAUDE.md (« churn-sans-valeur »). `HasExpectedData` a été RETIRÉ (Phase 3a-B, 2026-06-17). |
| `MatchScoreboardRow` 100% nullable | ⛔ **WON'T-DO** | **Écarté** (même raison) — `XUID`/`Gamertag`/`IsMe`/`OutcomeLabel` non-nullable car toujours peuplés ; les pointer-iser = NO-OP front + churn. |

## Phase 1.9 — Watcher multi-title routing · 🟢 (85%) — **threading LIVRÉ 2026-06-17 (byte-identique mono-titre) ; reste PMT-2 auth par titre (différé)**

> Ajoutée après audit du poller de présence en jeu (2026-06-13). **Re-vérif 2026-06-17 (workflow ultracode)** : le write-path (CoordinatorRequest.TitleSlug + gateKey composite + WithTitleSlug ctx moteur) ET le host read-path (`endpoint_resolver.hostFor(ctx)` ctx-driven, fallback const byte-identique) étaient DÉJÀ livrés (PMT-3 + PMT-1). Le seul gap réel : `td.Slug` n'était jamais propagé jusqu'à `MatchRequest.TitleSlug` (toujours `""` → halo_infinite). Threading livré.

| Item | Statut | Evidence / next action |
|---|:-:|---|
| Détection présence title-agnostic | ✅ | `watcher/daemon.go::makePresenceHandler` → `titleReg.MatchPresence(titleID)` → `title/matcher.go::MatchByXboxTitleID` itère tous les titres |
| `MatchFetcher` routé par titre | ✅ | **Pas un resolver per-fetcher** : 1 client partagé, routé au RUNTIME par le ctx (`halo_client.hostFor(ctx)` → `endpoint_resolver` lit `ctxkeys.TitleSlug`). Devient title-aware dès que `pollerCtx` porte le slug (livré ci-dessous) — rien à recâbler côté fetcher |
| `PlayerWatcher.titleSlug` | ✅ | **Livré 2026-06-17** : champ `titleSlug` + `SetTitleSlug` (intrinsèque, posé à la construction depuis `PlayerSummary.TitleSlug` — robuste au broadcast de présence) ; `startPoller` pose le slug sur `pollerCtx` → fetch routé par titre |
| `TitleSlug` dans `MatchRequest`/`CoordinatorRequest`/`TriggerSync` | ✅ | **Livré 2026-06-17** : CoordinatorRequest.TitleSlug (PMT-3, déjà) + `queueSyncTrigger.TriggerSync` lit `ctxkeys.TitleSlug(ctx)` → `MatchRequest.TitleSlug` ; bug `Enqueue` qui droppait `TitleSlug` dans `filtered` corrigé. Threading PAR ctx (zéro changement d'interface `SyncTrigger`) |
| Garde compat `DefaultSlug` + non-régression mono-titre | ✅ | registre mono-entrée → `td.Slug == halo_infinite` ; `gateKey(halo_infinite, gt) == normGT(gt)` (court-circuit) → clé/host inchangés. Tests : `title_routing_test.go` (3) + suite watcher existante verte |
| **PMT-2 auth par titre (pool)** | ⬜ **différé** | `NewPooledHaloClient` sert des tokens Halo round-robin non title-scopés (`main.go` pool `PolicyAnyPublic`). **Inexerçable sans 2e titre** (pas de périmètre auth distinct à valider) → seul axe réellement absent, à brancher au 2e titre |
| Steam fallback title-aware (`MatchBySteamAppID`) | ⬜→**optionnel** | `SteamPoller` non câblé (note W8) → hors scope tant que non activé |

---

## Hors fenêtre minimale (différé, tracké pour mémoire)

- **Phase 3b — Huma** : ✅ **CONTRAT JSON 100% MIGRÉ — FINALISÉ 2026-06-19**. Le tail des dernières routes JSON inline (media×5, watcher `/auth/start`, asset-stubs fallback×2, device-flow×2, home×3 dont `/pages/home` ETag/304 byte-exact) a été migré (commit `0d334d77a`). **Garde-fou `TestNoJSONRouteBypassesHuma`** (`internal/api/handlers/json_huma_coverage_test.go`) : échoue si une route JSON contourne Huma (allowlist = `writeError` + 2 routes multipart). **Bug produit corrigé au passage** : session anonyme non persistée (`e09ef2737`) cassait le login device-flow → fix `SessionData.DeviceFlowAttemptID` (2 E2E rouges → verts). Restent sur chi **par conception** (non-JSON) : multipart upload/import, binaire (images/serve), redirects OAuth, CSV export, `/static`, SPA. **Bascule openapi.yaml généré = DESCOPÉE** (bloquant archi, investigué 2026-06-18 : ~133 `humacore.NewAPI`, chemins relatifs, `Components.Schemas`=Registry, ~20 routes non-JSON) → **YAML manuel gardé** + drift-detector `openapi_schema_drift_test.go` (gate MISSING==0). **`PLAN_WEB_API_TYPES` n'est PLUS « foundation posée » : MIGRÉ** — openapi.yaml complété (MISSING 332→0), `types.ts`→`generated.ts` (228 shims + ratchet anti-doublon `lint-contract-ratchet.mjs`), cf. thought_log 2026-06-18/19.
- **Phase 4** — sync flags génériques (FieldKey-based) : ⛔ **CLÔTURÉE PAR SUPPRESSION 2026-06-18** (pas par câblage). Le refactor FieldKey restait NO-GO (fausse affordance, zéro consommateur prod). Les 12 flags granulaires `SyncScope` + 5 groupes alias + leurs `Force*` + 34 flags CLI `NewBackfillFlagSet` ont été retirés (byte-identique fetch prouvé ; `--accuracy`/`--shots`/`--enemy-mmr` directs conservés). cf. thought_log 2026-06-18.
- **Phase 5** — frontend canonical-aware + `<FeatureGate>` : 🟡 **GATING PAR CAPABILITY LIVRÉ 2026-06-18** (worktree `feat/multititre-peripherie`, 2 workflows ultracode : map 99 gates → reconcile per-capability 35 vérifiés). Socle `apps/web/src/lib/capabilities/` : `useCapability` / `useTitleCapabilities` / `hasCapabilityIn` (fail-open bootstrap) + `<FeatureGate capability>` + `<RouteCapabilityGate>` (+ `<FeatureUnavailable>` placeholder gracieux). Câblage NO-OP pour halo_infinite (déclare les 11 capabilities) : **nav** (NavL1 filtre `visibleSections` + onglets : media→media, career→career, ascension→lusr, onglet Classements→world.leaderboard ; NavL2 career/community) ; **route-page** (media, career_, citations, season-pass, palmares/index, ascension) ; **section/intra-composant** (AchievementsCareerSection→achievements ×2, CareerLusrEvolutionChart→lusr, colonnes CSR/LUSR de CareerRankingBlock→ranked/lusr, bloc Médias match-view→media, EngagementMatchSection + 3 autres sites engagement→engagement, Home playlists récentes + skill-peaks CSR/LUSR→ranked/lusr). Décision tranchée : **Ascension = `lusr`** (les 3 onglets consomment skill_rating/lusr_components — résout le désaccord initial achievements/engagement/lusr). **firefight/forge = 0 gate** (firefight self-hide via 422 pve_not_supported + agrégat synthèse correct ; aucune surface Forge front). Vérifs : typecheck ✅, eslint 0 err, vitest **1902 pass / 14 skip**. Reste **canonical-aware** (labels via field-mappings TOML) = déjà en place (`useFieldLabel`/`useOutcomeLabel`). **CORRECTION 2026-06-18 (audit UI vérifié + adversarial)** : le « ✅ » était surévalué — 2 surfaces rang-Halo avaient été oubliées par le gating. **FIXÉES ce jour** : (a) **Explorer** — bloc CSR saison (`ExplorerTargetProfileCard` → `useCapability('ranked')`, reflow matchs pleine largeur) + filtre paliers skill-tier (`ExplorerPage.matchesMode` masqué si !ranked) ; (b) **Timeseries Progression** — `TimeseriesSkillProgression` + `TimeseriesSkillRankPerformance` gatés `ranked||lusr` (RankScore=score+placement reste générique). typecheck+eslint+vitest 1902 verts. **Reste (non bloquant) = vocabulaire Halo en dur** non gaté : admin Sync sections « API Halo »/« Watcher présence Xbox », Lab libellés « Waypoint », HomeHeroBanner images, `csrRankImageURL` motif `HINF-CSR` (cosmétique : fallback Halo pour un 2e titre, à sortir vers adapter par-titre). **Gating front = ~95%** (cosmétique restant). cf. thought_log 2026-06-18 + audit workflow.
- **Phase 1.8** — outillage diag Lab (3-4 j, différable même dans la fenêtre).

## Périphérie multi-titre (audit 2026-06-14) — registre B (hors chemin data-lecture)

> Index complet : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md) (`MT-01..MT-26`). Specs : [PLAN_MULTITITRE_PERIPHERY.md](PLAN_MULTITITRE_PERIPHERY.md) (`PMT-1..14` + `EXT-1.5/2/5`). **⚠ Pointeurs datés — RE-VÉRIFIER avant exécution.** Méthode : `expand → parity-gate → contract` + oracle double (parité Halo golden + `synthetic_test_title`).

| Phase | Objet | Axes | Sév. | Statut | Prérequis |
|---|---|---|:-:|:-:|---|
| PMT-1 | Hosts d'ingestion title-aware | MT-01 | blocker | ⬜ | racine |
| PMT-2 | Acquisition auth par titre | MT-02 | blocker | ⬜ | racine |
| PMT-3 | Scheduler/sync titleSlug threading | MT-11 | blocker | ⬜ | PMT-1/2 |
| PMT-4 | Settings par titre + config Discord | MT-04, MT-26 | ✅ | **COMPLET 2026-06-18** : PR-0 `ResolveForTitle`+`TitleSettingsPath`, PR-1 CSR season overlay, PR-2 Discord `LoadNotifyConfigForTitle`, PR-3a/b CSR+session per-titre, **PR-3c GET/PATCH /settings overlay** (GET résout l'overlay ; PATCH écrit les champs valeur per-titre — ShowProgression/OutcomeExclude* — dans `data/titles/<slug>/settings.json` via `SaveTitleOverlay` sparse, titre par défaut byte-identique, amis/verrou/lang restent globaux). FriendGamertags = global PAR DÉCISION (personnes ≠ titre). Tests : overlay sparse + global jamais modifié + merge incrémental |
| PMT-5 | Canonicalisation Outcome | MT-06 | major | ✅ | **COMPLET 2026-06-18** : Contract Go (archlint rawOutcomeAllowlist VIDE) + infra SQL resolver (`SQLEqExpr` 4 issues, port `games.OutcomeResolver`+`SetDefaultOutcomeResolver`, helper `duckdb.outcomeSQLEq`/`outcomeSQLEqSlug`, fallback byte-identique Halo) + **TOUS les repos SQL migrés** (explorer, compare, match_history, career_encounters, match_detail, squad ×3 templates) + **dead consts retirés** (`SQLIsWin`/`SQLWinRateExpr`, `Q42MapStatsForSquadTemplate`). 0 littéral d'issue actif restant. Validé : duckdb `-tags integration` (67s) + archlint verts |
| PMT-6 | Achievements par titre | MT-08 | major | ⬜ | PMT-1/2 |
| PMT-7 | World-stats / leaderboard par titre | MT-03 | major | ⬜ | PMT-3 |
| PMT-8 | Cycle de vie du titre (Status) | MT-22 | major | ✅ | branche `feat/multititre-peripherie` (cf27ff85f) |
| PMT-9 | Registre migrations + schema_version par titre | MT-23 | major | ⬜ | PMT-3 |
| PMT-10 | Observabilité — dimension titre | MT-05 | major | ✅ | logs + 3 collecteurs + endpoints ?title= + émission sync + LogsDir (PR-1→4) |
| PMT-11 | Discord notifications (contenu) | MT-26 | ✅ | **COMPLET 2026-06-18** : seam `NotifyLabels` (Outcome + TitleName, halo/semantic, oracle double 29 tests) + **câblage** : resolver partagé `notify.SetDefaultLabelsResolver`/`LabelsForSlug` (closure boot via adapter sémantique + nom registre, failsafe Halo) + injection `cfg.Labels = LabelsForSlug(ctx slug)` aux call-sites serveur (settings friend-added, friends_orchestrator). Embeds (outcomes + footer « LevelUp · {titre} Stats ») suivent le titre ; byte-identique Halo. CLIs = fallback Halo (resolver non câblé hors serveur) |
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

## État au 2026-06-19 — l'infra est livrée ; il ne reste QUE le payload 2e titre

**Re-vérifié contre le code (audit « que reste-t-il du master plan »).** Toutes les phases de la fenêtre minimale (0→3a) + 1.5/1.6/1.7/1.8/1.9 + 2 + 3b + 5 sont livrées et prouvées. Le multi-titre est branchable day-one côté données ET navigation (switcher câblé).

**Ce qui reste, ordonné :**
1. **LE vrai gros morceau = adapter data + vraies sources d'un 2e titre réel (Halo 5).** Intrinsèquement title-spécifique, HORS infra. `internal/games/synthetic_title_b/adapter.go` est un stub (`ErrCapabilityNotSupported` sur tous les `Load*`). Handoff : [HANDOFF_HALO5_EXPERIMENTAL.md](HANDOFF_HALO5_EXPERIMENTAL.md). **Embarque au passage** : (a) résolution title-aware de la médaille Perfect (`1512363953`) via `constants.toml` ; (b) externalisation du vocabulaire Halo cosmétique vers les adapters par-titre.
2. **PMT-2 jambe finale** (pool d'auth keyé par titre) — inexerçable sans 2e titre au périmètre auth distinct.

**Dette résiduelle = NÉANT actionnable hors 2e titre** (re-statué 2026-06-19) :
- Phase 3a nullabilité (`MatchExpectedStats`/`MatchScoreboardRow`) → **WON'T-DO/écarté** (NO-OP front + churn-sans-valeur, interdit CLAUDE.md).
- Phase 1 SQL Perfect-medal → **scopé adapter Halo 5** (résolution par titre ; un Go-const à mi-chemin = churn + risque, `%`/LIKE rendent `fmt.Sprintf` non-sûr).
- Vocabulaire Halo cosmétique → **YAGNI** jusqu'au 2e titre (abstraction sans consommateur).

> **Note de tracking** : mettre à jour les statuts (✅/🟡/⬜) + evidence à chaque PR. Doctrine : carte datée ≠ vérité, RE-VÉRIFIER contre le code (cf. `PLAN_MULTITITRE_INDEX.md`).
