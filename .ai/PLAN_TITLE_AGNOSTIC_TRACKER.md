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
| 0 | Décisions + setup (ADR/branche/lints/datasets) | 🟡 | 15 | non (additif) |
| 1 | FieldKey + `fields.toml` (+ constants.toml) | 🟡 | 60 | non |
| 1.5 | DDL par titre (sortir `migration/steps_*`) | ⬜ | 5 | **oui** (2e titre) |
| 1.6 | Pool tokens clé `(titleSlug,gamertag)` | ⬜ | 0 | **oui** (2e titre) |
| 1.7a | `capabilities.toml` + loader + endpoint | 🟡 | 30 | non |
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
| Branche `refactor/title-agnostic-services` | ⬜ | absente ; à créer depuis `main` quand on démarre l'exécution |
| Plan référencé dans `CLAUDE.md` § ADR | ⬜ | grep vide |
| Lints CI (`no_slug_comparison`, `slog-context`, `no-new-chi-handler`) | ⬜ | `tests/lint/` n'existe pas ; `no-new-chi-handler` reste désactivé jusqu'à Phase 3b |
| Job CI `synthetic_test_title-parity` (vide OK en P0) | ⬜ | absent de `.github/workflows/ci.yml` (10 jobs) |
| Datasets `testdata/integration/{halo_full,synthetic,openspartan_primed}` | ⬜ | absent ; `synthetic_title_b` existe (stub), pas `synthetic_test_title` |

## Phase 1 — FieldKey + fields.toml · 🟡 (~60%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `canonical/fields.go` couvre les colonnes services | 🟡 | 59 FieldKeys ; `match_participants` OK ; **`killer_victim_pairs` (killer/victim_xuid, kill_count) non mappées** |
| `fields.toml` (loader `internal/games/mappings/`) | ✅ | existe + 59 sections + loader complet + `loader_smoke_test` valide la couverture |
| Mapping **physique** table→colonne (5 tables) | 🟡→**caduc** | `fields.toml` est sémantique ; item caduc si FieldKey-map supersédé (réconciliation 2) |
| `constants.toml` (medal IDs, mode prefixes) | ⬜ | **absent** ; magic `1512363953` (Perfect) inline dans **7 fichiers** |
| Test exhaustivité FieldKeys↔TOML | 🟡 | `loader_smoke_test.go` couvre le sens TOML⊇FieldKeys ; manque sens inverse |
| 0 `medal_name_id=<int>` inline dans `queries_*.go` | ⬜ | `queries_match.go:38,245,306`, `queries_home_citations.go:26`, `queries_squad.go:79,145` + 4 repos |

## Phase 1.5 — DDL par titre · ⬜ (~5%) — **PREMIER GROS MORCEAU, prérequis 2e titre**

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `internal/games/halo_infinite/ddl/*.sql` | ⬜ | dossier inexistant |
| `internal/migration/` sans DDL Halo | ⬜ | **53 `steps_*.go` Halo-specific** restants (~72% du dossier) — recoupe le `steps_shared.go` 982L que j'avais déféré |
| `MigrationRunner` via `TitleDataAdapter.MigrationSteps()` | ⬜ | méthode inexistante ; registre global `init()`/`Register()` (ordre = exécution, cf. `registry.go:38`) |
| `synthetic_test_title/ddl/` minimal | ⬜ | `synthetic_title_b` = stub sans DDL |
| `ops/` (backup/restore/diagnose) multi-titre | 🟡 | `backup_service` itère via PathResolver ; `restore.go`/`diagnose.go` prennent un chemin direct |

## Phase 1.6 — Pool tokens multi-titre · ⬜ — **prérequis 2e titre**

| Item | Statut | Evidence / next action |
|---|:-:|---|
| Clé pool `(titleSlug, gamertag)` | ⬜ | `pool.go` : `slotsByGt map[string]int` (gamertag seul) ; `Discovery` a déjà `titleSlug` mais juste pour le path |

## Phase 1.7a — capabilities.toml (binaire) · 🟡 (~30%)

| Item | Statut | Evidence / next action |
|---|:-:|---|
| `capabilities.toml` par titre + loader | 🟡 | `CapabilityMap` existe **en code** (`games/adapter.go:36-70`) ; **0 `.toml`**, 0 loader ; extraire vers TOML |
| Adapters consomment le loader (0 hardcode) | ⬜ | `halo_infinite/adapter_data.go:61-80` hardcode `Capabilities()` |
| Endpoint `GET /title/capabilities` | ⬜ | seul `/titles/{slug}/field-mappings` existe |

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
