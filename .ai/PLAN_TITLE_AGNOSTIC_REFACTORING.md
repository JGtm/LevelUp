# Plan — Refactoring vers une architecture title-agnostic complète (v2)

**Objectif** : retirer/ajouter un champ d'un titre = **1 ligne dans `fields.toml` (mapping) + 1 modif dans le `TitleDataAdapter` correspondant**, sans toucher aux services, repos cross-titre, OpenAPI, ni frontend.

**Critère de succès opérationnel** : une CI step « build + tests pour `synthetic_test_title` » s'exécute avec un dossier `internal/games/synthetic_test_title/` minimal + un set de TOML mappings réduit. Toutes les routes répondent (200 ou 404 capability) et tous les tests existants passent. Aucun changement requis dans `internal/service/`, `internal/api/`, `apps/web/` au-delà d'une route capability-gated.

**Branche cible** : `refactor/title-agnostic-services` (créée depuis `main` après merge de `feat/token-pool-parallel-sync`).

**Statut** : v2.2 (2026-05-06) — décisions D1-D6 actées + Exit Gate strict par phase (DONE/NOT DONE daté), seuils de couverture chiffrés et bloquants, lints logging CI bloquants, validation finale via PR E2E `synthetic_test_title`. Effort total : 42-57 j.

---

## 0. Doctrine — alignement avec l'ADR 0011

L'ADR 0011 a tranché : **`canonical.*` reste minimal**, et trois adapters distincts collaborent côté service :

| Adapter | Rôle | Ce qu'il NE FAIT PAS |
|---|---|---|
| `TitleDataAdapter` | Charge la data brute → `canonical.*` | Pas de label i18n, pas d'URL |
| `TitleSemanticAdapter` | Labels FR/EN, RankCatalog, Outcomes | Pas de data brute |
| `TitleAssetURLAdapter` | URLs map / medal / CSR rank | Pas de DB |

Ce plan **respecte cette frontière** : il ne pousse PAS `canonical.PlayerMatchRow` directement dans le DTO HTTP. Le DTO HTTP reste un **view-model service** qui combine les 3 sources. Ce que le plan vise, c'est :

1. Que les **column names DB** ne fuitent plus dans les services (Phase 2).
2. Que les **types Halo-specific** dans `domain/match_view.go` (`MatchExpectedStats`, `ExpectedStatsRaw`, etc.) soient remplacés par des types canonical reasonably nullable (Phase 3).
3. Que les **flags sync** ne soient plus enumérés en dur par champ Halo (Phase 4).
4. Que le **schéma DB** soit isolé par titre, pas en silo dans `migration/steps_shared.go` partagé (Phase 1.5).

Différence clé avec v1 : le plan v2 **conserve les types domain de view-model** (header, summary tab, scoreboard row…) — ils sont la composition canonique × semantic × assetURL. Ce qu'on retire de `domain/`, c'est uniquement les types `*Raw` et les champs Halo-only sans pendant canonical.

---

## 1. Diagnostic — fuites actuelles de l'abstraction

Sur la modif récente `drop assists_expected/assists_stddev` (Halo Infinite ne renvoie pas ces champs), j'ai dû toucher 14 fichiers dans 8 couches.

| Couche | Fichier | Type de fuite |
|---|---|---|
| **DDL shared** | `migration/steps_shared.go` | `CREATE/ALTER COLUMN` Halo-specific dans la DB partagée multi-titres |
| **SQL inline** | `platform/duckdb/queries_match.go` (Q12, Q26, Q26MatchExpectedStats) | `SELECT mp.assists_expected, ...` codé en dur |
| **Magic constants Halo** | `queries_match.go` Q12 (`medal_name_id = 1512363953` pour Perfect Kills) | Constante Halo-only inline dans une query cross-titre |
| **Scan** | `platform/duckdb/match_view_repo.go` | `row.Scan(&s.AssistsExpected)` couplé à l'ordre du SELECT |
| **Domain (view-model)** | `domain/match_view.go` (`MatchExpectedStats`, `MatchScoreboardRow.ExpectedKills`) | Champs Halo-specific exposés au DTO HTTP |
| **Domain (raw)** | `domain/match_view.go` (`ExpectedStatsRaw`, `ScoreboardRaw`, `MatchHistAvgRow`…) | Types frontière repo↔service mais hébergés dans `domain/` |
| **OpenAPI** | `api/openapi.yaml` | Schema `MatchExpectedStats.expected_assists` édité manuellement (~80 routes maintenues à la main) |
| **Generated** | `internal/api/gen/types.gen.go` | Auto-généré depuis OpenAPI manuel — divergence inévitable |
| **Handlers chi** | `internal/api/handlers/*.go` (~80 fichiers) | Style `func(w, r)` avec validation manuelle (regex, parse query) — pas d'inférence OpenAPI |
| **Service** | `service/match_view_service.go` | `out.ExpectedAssists = e.AssistsExpected` |
| **Sync flags** | `sync/scope.go`, `sync/backfill_flags.go`, `sync/backfill_cli.go` | `PBitAssistsExp`, `--assists-expected`, `scope.AssistsExpected` |
| **Tests** | 4 fichiers | refs à `PBitAssistsExp` / `scope.AssistsExpected` / `assists_expected` |

**Causes racines** :

1. `domain/match_view.go` contient à la fois **les view-models DTO** (légitimes) et **des types `*Raw` repo-frontière** (devraient être en `platform/duckdb/`).
2. **SQL inline dans `queries_match.go`** au lieu de passer par un repo abstrait paramétré par `[]canonical.FieldKey`. Le service dépend implicitement de l'ordre des colonnes.
3. **Sync flags Halo-Infinite-specific** (`PBitAssistsExp`, `--assists-expected`) — bitmask et CLI codés en dur sur un set fixe de champs Halo.
4. **Schéma DB partagé** (`shared_matches_v2.duckdb`, table `match_participants` à 31 colonnes) bake in la liste des champs Halo. Un titre avec d'autres champs casse les DDL ou doit accepter des colonnes nil.
5. **OpenAPI YAML manuel** au lieu d'auto-généré depuis les types Go. Le contrat front fuit la sémantique title-specific.
6. **Constantes magiques Halo** (medal IDs, mode prefixes) inline dans queries cross-titre.

---

## 2. Architecture cible

### 2.1 Topologie (alignée ADR 0011)

```
┌──────────────────────────────────────────────────────────────────┐
│ apps/web/                                                        │
│  - lit JSON DTO via TanStack Query                               │
│  - useFieldLabel(FieldKey, locale) / useCapability(cap)          │
│  - omet/grise les sections quand field nil ou capability absente │
└─────────────────────────────────┬────────────────────────────────┘
                                  │ JSON DTO (view-model)
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│ internal/api/handlers/                                           │
│  - reçoit Request, appelle Service via port.*Service             │
│  - sérialise le DTO retourné                                     │
│  - 0 logique métier, 0 SQL                                       │
└─────────────────────────────────┬────────────────────────────────┘
                                  │ domain.*ViewResponse
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│ internal/service/                                                │
│  compose 3 sources :                                             │
│   - port.*Repository    → canonical.* (data brute)               │
│   - games.TitleSemantic → labels FR/EN, RankCatalog, Outcomes    │
│   - games.TitleAssetURL → URLs map / medal / CSR rank            │
│  retourne domain.*ViewResponse (view-model composé)              │
│  dégrade gracieusement si capability absente                     │
└──────┬─────────────────────────┬──────────────────────────────┬──┘
       ▼                         ▼                              ▼
┌──────────────┐        ┌────────────────┐         ┌────────────────────┐
│ analysis/    │        │ port/          │         │ internal/games/    │
│  algos purs  │        │  *Repository   │         │  halo_infinite/    │
│  0 IO        │        │  *Service      │         │  adapter_data.go   │
│  prend       │        │  interfaces    │         │  adapter_semantic  │
│  canonical   │        └────────┬───────┘         │  adapter_asset_url │
└──────────────┘                 │ impl            └────────────────────┘
                                 ▼
                ┌────────────────────────────────────┐
                │ internal/platform/duckdb/          │
                │  - implémente port.*Repository     │
                │  - construit SQL via FieldKey TOML │
                │  - DDL par titre (steps_*_<slug>)  │
                │  - 0 SQL exposé en dehors          │
                └────────────────────────────────────┘
```

### 2.2 Règle d'or

> **Aucune fonction dans `service/`, `api/handlers/` ou `apps/web/` ne doit jamais voir un nom de colonne DB, un type Halo-specific, ou un enum Halo (medal_id, mode_prefix). Tout passe par `canonical.*` + `FieldKey` + `Capability` ou par un des 3 adapters.**

### 2.3 Comportement nullable cohérent

- Tous les fields canonical optionnels sont **`*T` pointer**.
- Le frontend gère `null` partout (`omitempty` côté Go, `T | null` côté TS, fallback texte ou skeleton dans la UI).
- Une capability absente → field jamais renvoyé (`omitempty`) → frontend ne l'affiche pas.
- Distinction explicite côté repo : « field absent du mapping TOML » vs « valeur NULL en DB » (cf. décision D2 ci-dessous).

---

## 3. Phase 0 — Décisions techniques + alignement ADR (BLOQUANTES)

**Effort** : 2-3 jours
**Livrable** : 6 décisions tranchées + ADR mise à jour + branche créée + plan v2 figé.

### Décisions tranchées (session 2026-05-06)

| ID | Sujet | Décision | Justification |
|---|---|---|---|
| **D1** | `canonical.Value` (Phase 2) | Wrapper typé `Value{Kind, Int, Float, Str, Bool, Time}` | Compromis lisibilité/perf, pattern-match côté service via switch sur Kind, évite le runtime cast |
| **D2** | Field absent vs NULL | `map[FieldKey]*Value` : présent dans le map = field supporté ; `*Value = nil` = NULL en DB ; absent du map = capability non supportée | Sémantique explicite, testable, permet de vérifier la dégradation gracieuse |
| **D3** | Schéma DB multi-titre (Phase 1.5) | DB physique par titre (`data/titles/{slug}/warehouse/...`) | Cohérent ADR 0008 (isolation par chemin FS). Le `PathResolver` retourne déjà la bonne DB. DDL dans `internal/games/{slug}/ddl/` |
| **D4** | OpenAPI gen (Phase 3) | **Huma intégré au plan** : migrer ~80 handlers chi vers Huma. OpenAPI 3.1 auto-généré par construction, validation des inputs auto, plus jamais de YAML manuel | Décision ambitieuse : élargit le scope du plan mais évite un refactor handlers ultérieur. Coût total révisé à 50-65 j |
| **D5** | Codegen TS canonical (Phase 7) | Script `tools/codegen/canonical-ts/` : lit `canonical/fields.go` (go/ast) → écrit `apps/web/src/lib/canonical/fields.ts`. Single source Go, CI lint vérifie l'idempotence | Évite la dérive entre Go et TS. Compatible avec l'output OpenAPI Huma (qui génère son propre client TS pour les DTO) |
| **D6** | Stratégie de migration progressive (Phase 2) | Service par service en PR atomique : ancien path supprimé dans la même PR que la migration | Pas de feature flag (évite la dette « deux paths à maintenir »). Critère de mergeabilité par phase = service migré + tests passent |

### Tâches Phase 0

- [ ] **Réviser ADR 0011** : ajouter section « v2 — confirmation post-plan title-agnostic » qui acte que canonical reste minimal et que le plan title-agnostic ne le contredit pas. Alternative : nouvel **ADR 0014 — title-agnostic services + Huma migration + DDL isolation** qui complète 0011.
- [ ] **ADR 0015 (à créer)** : « Adoption de Huma pour OpenAPI 3.1 auto-généré ». Documente le choix vs swag/kin-openapi/Fuego, le coût (~80 handlers), les bénéfices long terme (validation auto, plus de YAML manuel).
- [ ] Créer la branche `refactor/title-agnostic-services` depuis `main`.
- [ ] Ajouter ce plan en référence dans `CLAUDE.md` § Décisions architecturales.
- [ ] Entrée `thought_log.md` documentant les 6 décisions et leur justification.

---

## 4. Phases d'exécution

### Phase 1 — Étendre `canonical/fields.go` à 100% des champs services (rapide)

**Effort** : 2-3 jours
**Risque** : faible (additif uniquement)
**Livrable** : tous les FieldKeys que les services lisent existent dans `canonical/fields.go`, mappés vers les colonnes DB Halo Infinite via TOML, pour **toutes les tables shared**, pas seulement `match_participants`.

- [ ] Inventaire exhaustif :
  ```bash
  rg "p\.\w+|mp\.\w+|mr\.\w+|me\.\w+|w\.\w+|kvp\.\w+" \
     apps/go-api/internal/platform/duckdb/queries_*.go > /tmp/cols.txt
  ```
- [ ] Pour chaque colonne référencée, vérifier que le `canonical.FieldKey` existe ; sinon, ajouter dans `canonical/fields.go`.
- [ ] Étendre `config/titles/halo_infinite/mappings/fields.toml` avec le mapping `field_key → db_column → table` pour les 5 tables :
  - `match_participants` (~31 colonnes)
  - `medals_earned` (medal_id, count)
  - `highlight_events` (event_type, time_ms, actor_xuid)
  - `killer_victim_pairs` (killer_xuid, victim_xuid, kill_count)
  - `weapon_kills` (weapon_id, kills)
- [ ] Pour les **constantes Halo magiques** (`medal_name_id = 1512363953` pour Perfect Kills, mode prefixes Assassin/Fiesta/BTB…) : créer `config/titles/halo_infinite/constants.toml` (`perfect_kill_medal_id`, `mode_prefixes`) lu au boot par l'adapter Halo.
- [ ] Test garde-fou : `canonical/fields_test.go` vérifie l'exhaustivité des FieldKeys référencés par les TOMLs (lint custom déjà existant `Lint multi-titres`).

**Critère de complétion** : `grep -rE "p\.\w+|mp\.\w+|mr\.\w+" internal/platform/duckdb/queries_*.go` produit une liste 100% couverte par `fields.toml`. Aucun magic number Halo dans `queries_*.go`.

### Phase 1.5 — DDL et schéma multi-titre (moyen, NOUVELLE)

**Effort** : 4-5 jours
**Risque** : moyen (touche les migrations existantes)
**Livrable** : chaque titre possède ses propres DDL ; le `MigrationRunner` est paramétré par `titleSlug`.

- [ ] Déplacer `migration/steps_shared.go` → `internal/games/halo_infinite/ddl/steps_shared.sql` + `steps_player.sql` + `steps_pve.sql`.
- [ ] `MigrationRunner` lit les steps depuis le `TitleDataAdapter` (méthode `MigrationSteps()`). Plus de DDL hardcoded dans `platform/duckdb/migration/`.
- [ ] Schema versioning par titre : `meta.schema_version` dans chaque DB, par titre.
- [ ] Test : ajouter un titre fixture `synthetic_test_title` avec un schéma minimal (3 colonnes : kills, deaths, match_id) et vérifier que le `MigrationRunner` crée la DB sans toucher au code partagé.

**Critère de complétion** : `internal/platform/duckdb/migration/` ne contient plus aucune DDL Halo-specific. Ajouter un titre = créer son dossier `ddl/` + l'enregistrer.

### Phase 2 — Repository abstrait par FieldKey (moyen)

**Effort** : 5-7 jours (révisé depuis v1)
**Risque** : moyen (refactor des repos existants, mais migration progressive service-par-service)
**Livrable** : nouvelle interface `port.MatchFieldRepository` qui prend des `[]FieldKey`, retourne `map[FieldKey]*canonical.Value`. Implémentation DuckDB qui résout le SELECT via TOML mapping.

- [ ] **Bench préliminaire** (1 j, BLOQUANT) : implémenter une version POC de `LoadMatchFields` avec SELECT dynamique sur Q12 (scoreboard, 4 joins). Comparer perf vs Q12 actuel sur un dataset de 1000 matchs. Si > 15% slowdown → garder Q12 spécialisé pour les hot paths, n'utiliser le SELECT dynamique que pour les routes peu sollicitées.
- [ ] Créer `internal/port/match_field_repository.go` :
  ```go
  type MatchFieldRepository interface {
      LoadMatchSummary(ctx, matchID) (*canonical.MatchSummary, error)
      LoadMatchParticipant(ctx, matchID, xuid) (*canonical.PlayerMatchRow, error)
      LoadMatchFields(ctx, matchID, xuid, []FieldKey) (map[FieldKey]*canonical.Value, error)
      LoadScoreboardFields(ctx, matchID, []FieldKey) ([]map[FieldKey]*canonical.Value, error)
  }
  ```
- [ ] Implémenter `internal/platform/duckdb/match_field_repo.go` :
  - Construit le SELECT dynamique depuis le TOML mapping.
  - Map les rows vers `*canonical.Value` (D1 = wrapper typé).
  - **Sémantique D2** : FieldKey absente du map = capability non supportée ; FieldKey présente avec `*Value = nil` = NULL en DB ; FieldKey présente avec valeur = OK.
- [ ] Tests integration `match_field_repo_integration_test.go` :
  - Halo Infinite : tous les FieldKeys retournent valeurs ou NULL cohérents.
  - `synthetic_test_title` : seules les FieldKeys déclarées dans son TOML sont résolues, les autres absentes du map.
- [ ] Migration progressive (D6) : chaque service migré en PR séparée :
  1. `match_view_service.go` (pilote, plus complexe)
  2. `synthesis_service.go`
  3. `home_service.go`
  4. `explorer_service.go`
  5. `match_history_service.go`
  6. `career_service.go`
  7. `timeseries_service.go`

**Critère de complétion** : aucun service n'importe `internal/platform/duckdb` directement. Tous dépendent de `port.MatchFieldRepository` ou des autres ports existants.

### Phase 3 — Nettoyage DTO + migration vers Huma (lourd, fusion ex-Phases 3+4)

**Effort** : 18-25 jours
**Risque** : élevé (impact contrat front + rewrite ~80 handlers)
**Livrable** : `domain/match_view.go` ne contient plus que les view-model DTO propres ; tous les handlers chi sont migrés vers Huma ; `api/openapi.yaml` est auto-généré et le client TS front aussi.

> **Pourquoi fusionner Phase 3 (OpenAPI) et ex-Phase 4 (DTO clean)** : Huma génère l'OpenAPI depuis les types Input/Output. Si on migre les handlers vers Huma AVANT de nettoyer les DTOs, on définit les Output structs sur les types `domain` actuels (Halo-specific) puis on les rewrite en Phase 4 — soit deux passes sur 80 handlers. Fusionner = un seul passage par handler.

#### Phase 3a — Cleanup DTO (5-7 j)

- [ ] Déplacer les types `*Raw` (`MatchMetaRaw`, `ScoreboardRaw`, `ExpectedStatsRaw`, `MatchHistAvgRow`, `BulkMedalRaw`, `BulkWeaponKillRaw`, etc.) de `domain/match_view.go` vers `platform/duckdb/raw_types.go`. Ils ne traversent plus la frontière service.
- [ ] Réécrire `MatchExpectedStats` : tous les champs `*float64 omitempty`. Pas de `HasExpectedData bool` (le front teste `expected_kills !== null`).
- [ ] Idem `MatchScoreboardRow` : les 30+ champs deviennent `*T omitempty`.
- [ ] Le service `match_view_service.go` consulte le `TitleDataAdapter` (via `port.MatchFieldRepository`) puis compose le DTO ; les champs absents → nil → JSON omit.

#### Phase 3b — Migration Huma (13-18 j)

- [ ] **Setup Huma sur chi existant** : créer `huma.NewAPI(chi)` adapter, sans toucher aux routes existantes. Phase 3b démarre avec 0 handler migré, OpenAPI vide.
- [ ] **Pattern de migration handler** : pour chaque handler, créer un struct `Input` (path/query/body params via tags `path:`, `query:`, `body:`, `header:`) et un struct `Output` (réponse). Le corps du handler devient `func(ctx, *Input) (*Output, error)`.
  ```go
  type MatchViewInput struct {
      PlayerSlug string `path:"player_slug"`
      MatchID    string `path:"match_id"`
      Playlist   string `query:"playlist,omitempty"`
  }
  type MatchViewOutput struct {
      Body domain.MatchViewResponse
  }
  func (h *Handler) MatchView(ctx context.Context, in *MatchViewInput) (*MatchViewOutput, error) {
      resp, err := h.svc.GetMatchView(ctx, in.PlayerSlug, in.MatchID, in.Playlist)
      if err != nil { return nil, mapErrorToHuma(err) }
      return &MatchViewOutput{Body: *resp}, nil
  }
  ```
- [ ] **Migration progressive par groupe de handlers** (ordre de risque croissant) :
  1. Handlers simples GET sans body (~30 handlers : health, bootstrap, settings, gamertag, prestige, achievements, etc.) — ~3 j
  2. Handlers GET avec query params filtrés (~25 handlers : match_history, sessions, explorer, timeseries, etc.) — ~5 j
  3. Handlers POST/PUT avec body (~15 handlers : sync, watcher, match_favorite, match_exclusion, etc.) — ~3 j
  4. Handlers complexes (~10 handlers : match_view, synthesis, season_pass, squad_v2 — gros DTO, params imbriqués) — ~4 j
- [ ] **Validation des inputs** : les tags `minLength:`, `maxLength:`, `pattern:`, `enum:` dans Input sont vérifiés par Huma avant d'appeler le handler. Supprimer les regex manuels (`playlistOrSessionPattern` etc.) au profit de tags Huma.
- [ ] **Mapping erreurs** : créer `mapErrorToHuma(err) error` qui convertit `port.ErrCapabilityNotSupported` → `huma.Error404NotFound`, `port.ErrInvalidInput` → `huma.Error400BadRequest`, etc.
- [ ] **Suppression `api/openapi.yaml` manuel** : le YAML est désormais généré par `huma.NewAPI(...)` au boot, exposé sur `/openapi.yaml` et committé via `go generate ./tools/openapi-export`. CI fail si diff.
- [ ] **Régénération client TS frontend** : `apps/web/src/lib/api/types.gen.ts` régénéré depuis le nouveau YAML via `openapi-typescript`. Régénération automatique en CI.
- [ ] **Tests de contrat snapshot** : pour chaque route, JSON de réponse comparé à un golden file (fixtures Halo + synthetic_test_title).
- [ ] **Smoke test E2E** : 7 routes critiques (home, match-view, match-history, sessions, palmares, season-pass, synthesis) renvoient toujours du JSON valide vs le schema généré.

**Critère de complétion** :
- `domain/match_view.go` ne contient plus aucun type `*Raw` ni aucun champ marqué « Halo Infinite : pas d'expected_assists ».
- `api/openapi.yaml` est généré + git-tracked ; aucune édition manuelle ; CI fail si diff vs `huma.NewAPI` au boot.
- 100% des handlers utilisent le pattern `func(ctx, *Input) (*Output, error)` Huma.
- Aucun handler n'utilise plus directement `chi.URLParam`, `r.URL.Query()`, ou `json.NewEncoder(w).Encode(...)`.

### Phase 4 — Sync flags génériques (moyen)

**Effort** : 5-6 jours (révisé)
**Risque** : moyen (touche le CLI utilisateur)
**Livrable** : `SyncScope` est partiellement FieldKey-based (pour les champs stats) et garde des flags top-level pour les concerns non-FieldKey (sessions, citations, engagement, dry-run, etc.).

- [ ] **Découpler** :
  - **Champs FieldKey-based** : `TeamMMR`, `KillsExpected`, `DeathsExpected`, `Damage`, `AvgLife`, `GrenadeKills`, `MeleeKills`, `PowerWeaponKills`, `HeadshotKills`, `MaxSpree`, `KDARecalc`, `TimePlayed` → remplacés par `Fields []canonical.FieldKey`.
  - **Champs « stratégie de sync » non-FieldKey** : `Medals`, `Events`, `Skill`, `KillerVictim`, `Sessions`, `Citations`, `EngagementScores`, `LUSR`, `CSR`, `SkillRank`, `ComebackBadges`, `PlayableDuration`, `Aliases`, `Assets`, `PVEStats`, `Weapons` → restent flags top-level (groupés dans `SyncScope.Operations`).
  - **Options générales** : `DryRun`, `MaxMatches`, `RequestsPerSec`, `DetectionMode` → inchangés.
- [ ] CLI : `--field <key>` répétable + `--operation <name>` répétable. Aliases historiques préservés via map (`--mmr` → `--field team_mmr --field enemy_mmr`, `--skill` → `--field` set + `--operation skill`).
- [ ] `backfill_flags.go::PBit*` : générer le bitmask dynamiquement depuis le TOML capabilities au boot. Le mapping `bit_position ↔ FieldKey` est versionné (changement = bump schema_version + script de migration de la table `backfill_completed`).
- [ ] `FindMatchesMissingData` prend `[]FieldKey` + `[]Operation` et construit le `WHERE` dynamiquement. Bench (cf. Phase 2) pour vérifier que les index DuckDB existants tiennent.
- [ ] Deprecation warning sur les vieilles options CLI : 2 versions, retrait à v6.5.

**Critère de complétion** : ajouter un nouveau field stats = 1 ligne dans le TOML + 1 ligne dans le `TitleDataAdapter`. Le CLI le détecte automatiquement. Les opérations restent enumérées (pas de scope creep).

### Phase 5 — Frontend canonical-aware (moyen)

**Effort** : 6-8 jours (révisé)
**Risque** : moyen (ajout d'abstractions front, mais OpenAPI gen capture les changes)
**Livrable** : composants UI utilisent `useFieldLabel(FieldKey)` / `useCapability(cap)` au lieu d'accéder directement aux propriétés JSON par hardcoded path.

- [ ] **Codegen TS depuis canonical Go** (D5) : script `tools/codegen/canonical-ts/` qui lit `canonical/fields.go` et écrit `apps/web/src/lib/canonical/fields.ts`. CI lint vérifie que le fichier généré est à jour.
- [ ] **Client API TS auto-généré** depuis l'OpenAPI Huma (Phase 3b) via `openapi-typescript`. Remplace le client manuel actuel.
- [ ] Hook `useFieldLabel(field, locale)` lit le manifest i18n exposé via API `/api/v1/title/manifest` (déjà existant côté back via TitleSemanticAdapter).
- [ ] Composants `<StatRow field="kills_expected" value={...} />` qui se masquent automatiquement si value undefined.
- [ ] Capability gating au routeur : `<Route capability="lusr">` n'affiche pas la route si titre n'expose pas LUSR.
- [ ] Tests Vitest sur les hooks + tests Playwright sur la dégradation `synthetic_test_title`.

**Critère de complétion** : ajouter un titre = uploader son `i18n manifest` + `capabilities.toml` côté back. Le front l'utilise sans nouveau code TS.

---

## 5. Tests — exigences strictes (BLOQUANTES)

> **Règle absolue** : chaque seuil ci-dessous est BLOQUANT pour l'exit d'une phase. Aucune dérogation. Si un seuil n'est pas tenu, la phase n'est PAS close — le travail continue jusqu'à ce que le seuil soit atteint OU une dérogation est documentée dans `thought_log.md` AVEC date de remediation engagée.

### 5.1 Seuils de couverture par couche (mesurés via `go test -coverprofile`)

| Couche | Couverture min | Type de test | Métrique de qualité supplémentaire |
|---|:-:|---|---|
| `internal/games/canonical/` | **95%** | Unitaire pur (zero IO) | 100% des FieldKey référencés dans TOMLs ont un test de round-trip |
| `internal/analysis/` | **90%** | Unitaire pur | Property-based (gopter ou rapid) sur ratios, KDA, accuracy |
| `internal/games/{slug}/adapter_*.go` | **85%** | Unitaire avec fixtures JSON réelles | Chaque `Load*()` testé sur ≥3 fixtures (best/typical/edge case) |
| `internal/games/{slug}/ddl/` | **100%** des steps | Test MigrationRunner sur DB vide | Schema produit comparé à un golden file via `pragma_table_info` |
| `internal/port/` | N/A (interfaces) | Mocks utilisables | Lint : aucune impl directe de `port.*` n'est référencée hors `platform/` |
| `internal/platform/duckdb/match_field_repo.go` | **85%** | Integration `:memory:` + dataset réaliste | Test pour CHAQUE FieldKey du TOML : présent+valeur, présent+NULL, absent du TOML |
| `internal/service/` | **80%** | Tests avec mocks `port.*` + fakes `games.*` | Tous les chemins de dégradation `ErrCapabilityNotSupported` testés |
| `internal/api/handlers/` (Huma) | **85%** | `huma.NewTestAPI` + golden snapshot JSON | 100% des routes : test happy path + 1 test d'erreur (404 / 400 / 500 mappé) |
| `apps/web/` (hooks + features) | **75%** | Vitest unit + Playwright E2E | Critical path : home, match-view, sessions, season-pass — Playwright systématique |

**Outils de mesure obligatoires** :
- Go : `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` — CI fail si seuil non tenu par package.
- TS : `vitest --coverage` avec `coverage.thresholds` configurés dans `vitest.config.ts` — CI fail si seuil non tenu.

### 5.2 Tests de non-régression OBLIGATOIRES (par phase)

#### Phase 2 — migration service par service vers `port.MatchFieldRepository`

Pour CHAQUE PR migrant un service :

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON avant/après** | Pour 10 matchs réels (5 PVP + 5 PVE), capturer la réponse JSON avant la migration (sur `main`) et après. | `diff -u before.json after.json` = vide OU diff documenté ligne par ligne dans la PR avec justification |
| **Snapshot SQL avant/après** | Logger les queries DuckDB exécutées (via `slog` Debug) avant/après. | Plan d'exécution comparable (pas de `Sequential Scan` introduit là où il n'y en avait pas) |
| **Bench latence** | `go test -bench` sur le handler complet pour 100 matchs. | p95 latency post-migration ≤ p95 pre-migration × 1.10 (slowdown max 10%) |
| **Test de parité multi-titre** | Sur `synthetic_test_title` (subset fields), même handler répond 200 + JSON valide. | Pas de panic, pas de 500, FieldKeys absents → omitempty respecté |

#### Phase 3a — cleanup DTO

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON 100% routes** | Pour 5 fixtures par route (1 par profil joueur représentatif), avant/après. | Diff = uniquement des champs nouvellement nullables qui passent de `0`/`""` à `null` (omitempty), JAMAIS un champ qui disparaît |
| **Test typescript front** | Le client TS regen passe `tsc --noEmit` sur tout `apps/web/src/`. | 0 erreur TS |
| **Snapshot OpenAPI YAML** | Diff entre YAML manuel actuel et YAML produit par les types Go nouvellement nettoyés. | Diff documenté champ par champ dans la PR |

#### Phase 3b — migration Huma

Pour CHAQUE groupe de handlers migré :

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot JSON full** | Toutes les routes du groupe testées via `httptest` (via Huma) ET via `chi` (route legacy si encore présente) sur les mêmes fixtures. | Bytes identiques OU diff documenté |
| **Test des middlewares** | Pour chaque middleware existant (auth, CSRF, slog HTTP, rate-limit, title extractor), test `httptest` confirme l'invocation sur une route Huma. | 5/5 middlewares passent |
| **Test des erreurs** | Chaque erreur mappée (`ErrCapabilityNotSupported` → 404, `ErrInvalidInput` → 400, etc.) testée. | Status code + body conformes au schema OpenAPI Huma |
| **Test client TS regen** | `openapi-typescript` régénère le client TS sans erreur, `apps/web/` compile encore. | Build front passe, snapshot des types front committé |
| **Test validation Huma** | Pour chaque route, 3 inputs invalides (manque param, format faux, valeur hors enum). | Réponse 400 + body lisible, pas de panic, pas de 500 |

#### Phase 4 — sync flags

| Test | Modalité | Critère pass |
|---|---|---|
| **Compatibility CLI** | Tous les anciens flags (`--mmr`, `--skill`, `--assists-expected`, etc.) sont parsés et produisent le même bitmask `backfill_completed` qu'avant. | Test snapshot bitmask sur 20 combinaisons de flags |
| **Idempotence backfill** | Lancer backfill 2 fois avec le même scope → la 2e run ne fait aucun appel API ni écriture DB. | `slog` capture confirme 0 appel API en 2e run |

#### Phase 5 — frontend canonical-aware

| Test | Modalité | Critère pass |
|---|---|---|
| **Snapshot rendu Playwright** | Capture screenshot par route critique (home, match-view, sessions, season-pass) sur Halo Infinite + synthetic_test_title. | Diff visuel automatique (pixelmatch) ≤ 0.5% sur Halo. synthetic_test_title : sections capability-gated absentes |
| **Test capability gating** | 4 routes sans capability requise → 404 propre côté back, route absente du routeur front. | Pas d'erreur console, pas de blank page |
| **Test codegen TS** | `make canonical-ts-gen` produit un fichier identique au commit. | CI fail si diff |

### 5.3 Tests de parité multi-titre (exécutés à CHAQUE phase)

Le job CI `synthetic_test_title-parity` lance la suite COMPLÈTE des tests avec `LEVELUP_TITLE=synthetic_test_title`. Critères :

- Toutes les routes répondent (200 ou 404 capability) — JAMAIS 500.
- Toutes les FieldKeys absentes du TOML synthetic sont omises du JSON via `omitempty`.
- Aucun test n'utilise `if titleSlug == "halo_infinite"` (lint custom `tests/lint/no_slug_comparison_test.go`).
- Le frontend chargé avec `LEVELUP_TITLE=synthetic_test_title` n'a aucune erreur console.

### 5.4 Cas de dégradation explicitement testés

Pour chaque phase, les cas suivants sont obligatoirement couverts par un test (pas implicites, écrits noir sur blanc) :

- Title sans `expected_kills` → service retourne `*MatchScoreboardRow.ExpectedKills = nil` → JSON omit → UI masque la case → pas d'erreur 500.
- Title sans capability `match_film` → endpoint `/match/{id}/film` retourne 404 + body `{"capability":"match_film","supported":false}`.
- Title sans capability `lusr` → home page n'affiche pas le panneau LUSR (sans casser le reste).
- Title sans capability `firefight` → routes PVE absentes du routeur front, 404 côté back.
- Repo retourne `ErrCapabilityNotSupported` → handler dégrade au lieu de paniquer.
- Adapter retourne `nil` → service compose un DTO partiel sans crash.

### 5.5 Datasets de tests OBLIGATOIRES

Pour les tests d'intégration et de non-régression, **deux datasets obligatoires** (cf. mémoire « tests d'intégration avec datasets réalistes ») :

- **Halo réaliste** (`testdata/integration/halo_full/`) : 50 matchs hétérogènes (PVP + PVE, ranked + social, plusieurs maps/playlists, plusieurs joueurs, ≥1 match avec données partielles/NULL).
- **Synthetic minimal** (`testdata/integration/synthetic/`) : 10 matchs avec subset de FieldKeys (kills, deaths, match_id, durations uniquement).

CI échoue si un test d'intégration tourne sur fixtures < ces datasets.

---

## 6. Logging — exigences strictes (BLOQUANTES)

> **Règle absolue** : tout code introduit par ce plan respecte 100% des règles ci-dessous. Lint CI fail si violation. Aucune dérogation.

### 6.1 Lint CI obligatoires (à activer en Phase 0)

| Lint | Cible | Action si violation |
|---|---|---|
| `forbidigo` ban `fmt.Println`, `fmt.Printf`, `fmt.Print` | tout `internal/` et `cmd/` | CI fail |
| `forbidigo` ban `log.Print*`, `log.Fatal*`, `log.Panic*` (stdlib `log`) | tout `internal/` et `cmd/` | CI fail |
| `forbidigo` ban `panic(` hors `init()` ou tests | tout `internal/` et `cmd/` | CI fail |
| `revive` règle `unused-parameter` sur les handlers Huma | `internal/api/handlers/` | CI fail |
| Lint custom `slog-context-required` | toute fonction qui prend `ctx context.Context` doit utiliser `slog.*Context` (pas `slog.*` sans contexte) | CI fail |
| Lint custom `error-must-be-logged-or-returned` | tout `err != nil` doit soit `return err` (avec wrap), soit `slog.ErrorContext` | CI fail |

### 6.2 Standards de logging par opération

**Erreur non-triviale** (toute erreur autre que `io.EOF`, `sql.ErrNoRows` quand attendu, `context.Canceled`) :
```go
slog.ErrorContext(ctx, "match_field_repo: scan failed",
    "err", err,
    "title", titleSlug,
    "match_id", matchID,
    "xuid", xuid,
    "operation", "load_match_fields")
return fmt.Errorf("scan match %s: %w", matchID, err)
```

**Opération significative** (DB query > 100ms, API call externe, sync de plus de 1 match, migration step) :
```go
slog.InfoContext(ctx, "sync: backfill batch completed",
    "title", titleSlug,
    "player", gamertag,
    "match_count", len(matches),
    "duration", time.Since(start),
    "operation", "backfill_batch")
```

**Capability absente** (titre ne supporte pas un field/feature) — émis 1 fois par boot via `sync.Once` par couple `(title, capability)` :
```go
slog.WarnContext(ctx, "title_data_adapter: capability unsupported",
    "title", titleSlug,
    "capability", capName,
    "consumer", "match_view_service")
```

**Trace de debug** (utilisateur final ne voit pas, mais utile pour diag) :
```go
slog.DebugContext(ctx, "match_field_repo: load fields",
    "title", titleSlug,
    "match_id", matchID,
    "field_count", len(fields),
    "fields_requested", fieldKeys)
```

### 6.3 Clés structurées normalisées (whitelist exhaustive)

Toute nouvelle clé hors whitelist nécessite mise à jour de cette section + entrée `thought_log.md`.

| Clé | Type | Usage |
|---|---|---|
| `err` | error | Toujours présent en cas d'erreur |
| `title` | string (slug) | Identifiant titre courant |
| `match_id` | string | UUID du match |
| `xuid` | string | XUID Xbox du joueur |
| `player` | string (gamertag) | Display name du joueur |
| `field` | string (FieldKey) | FieldKey canonical concerné |
| `capability` | string (CapabilityKey) | Capability concernée |
| `operation` | string | Nom de l'opération (`load_match_fields`, `backfill_batch`, `migration_step`, etc.) |
| `duration` | time.Duration | Durée d'opération significative |
| `match_count` | int | Nombre de matchs (batch) |
| `field_count` | int | Nombre de FieldKeys |
| `route` | string | Path HTTP (Huma) |
| `status` | int | HTTP status code |
| `consumer` | string | Service qui consomme (pour traces inter-couches) |

### 6.4 Métriques expvar obligatoires (par phase)

L'observabilité ne se résume pas aux logs. Pour chaque nouvelle couche introduite, exposer via `expvar` (cohérent ADR 0009) :

| Métrique | Phase d'introduction | Format |
|---|---|---|
| `match_field_repo.load_duration_ms` | Phase 2 | Histogram (p50/p95/p99) |
| `match_field_repo.fields_unsupported_count` | Phase 2 | Counter par couple `(title, field)` |
| `huma.request_count_by_status` | Phase 3b | Counter par couple `(route, status)` |
| `huma.request_duration_ms` | Phase 3b | Histogram par route |
| `huma.validation_failures_count` | Phase 3b | Counter par route |
| `migration_runner.steps_applied` | Phase 1.5 | Counter par titre |
| `sync.field_backfill_count` | Phase 4 | Counter par FieldKey |

**Test obligatoire** : un test `expvar_smoke_test.go` par phase qui vérifie que les métriques sont exposées et incrémentent correctement après une opération.

### 6.5 Vérifications automatiques en fin de phase

```bash
# 1. Aucun fmt.Println résiduel
rg "fmt\.Print(ln|f)?\(" apps/go-api/internal/ apps/go-api/cmd/ \
  | grep -v "_test.go" \
  | grep -v "// approved:" \
  && echo "FAIL: fmt.Print* found" && exit 1

# 2. Aucun log.Printf résiduel
rg "^[^/]*\blog\.(Print|Fatal|Panic)" apps/go-api/internal/ apps/go-api/cmd/ \
  && echo "FAIL: stdlib log usage found" && exit 1

# 3. Tout slog dans contexte ctx utilise *Context
rg "slog\.(Debug|Info|Warn|Error)\(" apps/go-api/internal/ apps/go-api/cmd/ \
  | grep -v "_test.go" \
  && echo "FAIL: slog without Context found" && exit 1

# 4. Aucune erreur silencieuse
rg "err != nil \{[\s]*\}" apps/go-api/internal/ apps/go-api/cmd/ \
  && echo "FAIL: silent error swallow found" && exit 1
```

Ces 4 vérifications sont en CI à partir de Phase 0 et BLOQUANTES.

---

## 7. Multi-titres — exigences capability

- [ ] Chaque FieldKey est listée dans `config/titles/{slug}/mappings/fields.toml` (ou héritée d'un default si extension future).
- [ ] Si une route HTTP nécessite une capability et le titre actif ne la supporte pas → handler renvoie `404 ErrCapabilityNotSupported` (pas de panic, pas de 500).
- [ ] Le frontend appelle `GET /api/v1/title/capabilities` au boot pour connaître les features dispos.
- [ ] **Aucune** comparaison `if titleSlug == "halo_infinite"` dans le code applicatif. Tout via `HasCapability(cap)`. Lint custom Phase 0 : `tests/lint/no_slug_comparison_test.go`.

---

## 8. Phase Exit Gate — règles strictes (BLOQUANTES)

> **Aucune phase ne peut être déclarée "close" tant que TOUS les items du Exit Gate sont DONE et datés.**
>
> - Pas de "TODO", "report à plus tard", "OK pour MVP", "partiel". Chaque item est binaire : DONE ou NOT DONE.
> - Aucun item n'est optionnel. Si un item ne s'applique pas à un cas, ça doit être DONE-N/A avec justification écrite dans `thought_log.md` AVANT de fermer la phase.
> - Le tag git `phase-{N}-exit` n'est posé que quand le tableau exit gate est 100% DONE.

### 8.1 Format de l'Exit Gate

Chaque phase a son tableau Exit Gate au format suivant :

```
| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| Description courte de l'item | DONE | 2026-05-20 | commit abc123 / PR #42 / CI run #99 | GS |
```

- **Statut** : `DONE` ou `NOT DONE` (jamais "WIP", "PARTIAL", "TBD"). `DONE-N/A` autorisé uniquement avec justification thought_log.
- **Date** : YYYY-MM-DD du jour où l'item est passé en DONE.
- **Evidence** : lien commit hash + PR + CI run qui prouve.
- **Validateur** : initiales du validateur humain (Guillaume = GS).

### 8.2 Items communs à TOUTES les phases (Exit Gate transverse)

Ces 12 items sont obligatoires en fin de chaque phase, en plus des items spécifiques :

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| 1. `go test ./... -race` passe sans erreur | NOT DONE | | | |
| 2. `go vet ./...` sans warning | NOT DONE | | | |
| 3. Lint logging CI (cf. §6.5) passe (4 vérifs) | NOT DONE | | | |
| 4. Couverture par couche ≥ seuil §5.1 (mesurée) | NOT DONE | | | |
| 5. Job CI `synthetic_test_title-parity` passe | NOT DONE | | | |
| 6. Tests de non-régression de la phase passent (cf. §5.2) | NOT DONE | | | |
| 7. Datasets `halo_full` et `synthetic` à jour (cf. §5.5) | NOT DONE | | | |
| 8. Aucun fichier nouveau > 500 lignes | NOT DONE | | | |
| 9. Aucune fonction nouvelle > 80 lignes | NOT DONE | | | |
| 10. Métriques expvar §6.4 exposées + test smoke OK | NOT DONE | | | |
| 11. Entrée `thought_log.md` rédigée (date, décision, résultats, prochaine étape) | NOT DONE | | | |
| 12. Tag git `phase-{N}-exit` posé sur le HEAD de la branche | NOT DONE | | | |

### 8.3 Exit Gate Phase 0 — décisions et setup

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| ADR 0011 mis à jour (section v2) OU ADR 0014 créé | NOT DONE | | | |
| ADR 0015 « Adoption Huma » créé | NOT DONE | | | |
| Branche `refactor/title-agnostic-services` créée | NOT DONE | | | |
| Plan référencé dans `CLAUDE.md` § Décisions architecturales | NOT DONE | | | |
| 6 lints CI §6.1 activés et BLOQUANTS | NOT DONE | | | |
| Job CI `synthetic_test_title-parity` créé (vide est OK pour Phase 0) | NOT DONE | | | |
| Datasets `testdata/integration/halo_full/` + `synthetic/` créés | NOT DONE | | | |
| 12 items de §8.2 (Exit Gate transverse) | NOT DONE | | | |

### 8.4 Exit Gate Phase 1 — FieldKey exhaustifs (5 tables)

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| Inventaire colonnes `queries_*.go` produit (`/tmp/cols.txt` committé) | NOT DONE | | | |
| 100% colonnes inventoriées ont un FieldKey dans `canonical/fields.go` | NOT DONE | | | |
| 100% FieldKeys ont une section dans `fields.toml` (5 tables) | NOT DONE | | | |
| `constants.toml` Halo créé (medal IDs, mode prefixes) | NOT DONE | | | |
| Test `fields_test.go::TestExhaustiveTOMLCoverage` passe | NOT DONE | | | |
| Aucun `medal_name_id = <int>` inline dans `queries_*.go` | NOT DONE | | | |
| Couverture `canonical/` ≥ 95% | NOT DONE | | | |
| Property-based test sur ratios (KDA, accuracy) | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.5 Exit Gate Phase 1.5 — DDL par titre

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `internal/games/halo_infinite/ddl/steps_*.sql` créés | NOT DONE | | | |
| `migration/steps_shared.go` ne contient plus de DDL Halo-specific | NOT DONE | | | |
| `MigrationRunner` accepte un `TitleDataAdapter` paramètre | NOT DONE | | | |
| `synthetic_test_title/ddl/` créé (schema minimal) | NOT DONE | | | |
| Test : `MigrationRunner` crée la DB synthetic from scratch sans toucher au code partagé | NOT DONE | | | |
| Test golden : schema produit = `pragma_table_info` snapshot | NOT DONE | | | |
| `internal/ops/` (backup, restore, diagnose) auditeé et adaptée multi-titre | NOT DONE | | | |
| Couverture `internal/games/{slug}/ddl/` 100% des steps | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.6 Exit Gate Phase 2 — `port.MatchFieldRepository` + 7 services migrés

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `port.MatchFieldRepository` interface créée | NOT DONE | | | |
| `platform/duckdb/match_field_repo.go` impl créée | NOT DONE | | | |
| Bench préliminaire : `LoadMatchFields` p95 ≤ Q12 actuel × 1.15 | NOT DONE | | | |
| Test integration `:memory:` couvre : présent+valeur, présent+NULL, absent du TOML (3 scénarios par FieldKey) | NOT DONE | | | |
| Service 1 migré : `match_view_service.go` (PR atomique avec snapshots avant/après) | NOT DONE | | | |
| Service 2 migré : `synthesis_service.go` | NOT DONE | | | |
| Service 3 migré : `home_service.go` | NOT DONE | | | |
| Service 4 migré : `explorer_service.go` | NOT DONE | | | |
| Service 5 migré : `match_history_service.go` | NOT DONE | | | |
| Service 6 migré : `career_service.go` | NOT DONE | | | |
| Service 7 migré : `timeseries_service.go` | NOT DONE | | | |
| Aucun service n'importe `internal/platform/duckdb` directement (lint custom) | NOT DONE | | | |
| Snapshot JSON pour 10 matchs réels : diff = vide ou justifié | NOT DONE | | | |
| Bench latence : aucun handler avec slowdown > 10% | NOT DONE | | | |
| Couverture `internal/service/` ≥ 80%, `internal/platform/duckdb/match_field_repo.go` ≥ 85% | NOT DONE | | | |
| Métriques `match_field_repo.load_duration_ms` + `.fields_unsupported_count` exposées | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.7 Exit Gate Phase 3a — cleanup DTO

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| Types `*Raw` (12 types) déplacés vers `platform/duckdb/raw_types.go` | NOT DONE | | | |
| `domain/match_view.go` ne contient plus de type `*Raw` | NOT DONE | | | |
| `MatchExpectedStats` 100% nullable (`*float64 omitempty`) | NOT DONE | | | |
| `MatchScoreboardRow` 100% nullable (30+ champs) | NOT DONE | | | |
| `domain/match_view.go` ne contient plus de commentaire « Halo Infinite : pas de... » | NOT DONE | | | |
| Snapshot JSON 100% routes : diff documenté champ par champ | NOT DONE | | | |
| `tsc --noEmit` sur `apps/web/` passe | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.8 Exit Gate Phase 3b — migration Huma

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `huma.NewAPI(chi)` adapter en place | NOT DONE | | | |
| Groupe 1 (handlers GET simples) migré : 30 handlers | NOT DONE | | | |
| Groupe 2 (handlers GET + query params) migré : 25 handlers | NOT DONE | | | |
| Groupe 3 (handlers POST/PUT) migré : 15 handlers | NOT DONE | | | |
| Groupe 4 (handlers complexes) migré : 10 handlers | NOT DONE | | | |
| **100% handlers migrés** (~80) — aucun `func(w http.ResponseWriter, r *http.Request)` métier hors middlewares | NOT DONE | | | |
| Aucun `chi.URLParam` hors middlewares | NOT DONE | | | |
| Aucun `r.URL.Query()` hors middlewares | NOT DONE | | | |
| Aucun `json.NewEncoder(w).Encode` hors middlewares | NOT DONE | | | |
| `mapErrorToHuma` créé avec 100% des erreurs port mappées | NOT DONE | | | |
| `api/openapi.yaml` généré par `huma.NewAPI` au boot, committé via `go generate` | NOT DONE | | | |
| CI fail si `openapi.yaml` diff entre boot et commit | NOT DONE | | | |
| Client TS frontend regen via `openapi-typescript`, build front passe | NOT DONE | | | |
| Snapshot JSON full : 100% routes, bytes identiques OU diff documenté | NOT DONE | | | |
| Test des 5 middlewares chi (auth, CSRF, slog, rate-limit, title) avec route Huma : 5/5 OK | NOT DONE | | | |
| Test validation Huma : 3 inputs invalides par route, 0 panic, 0 status 500 | NOT DONE | | | |
| Couverture `internal/api/handlers/` ≥ 85% | NOT DONE | | | |
| Métriques `huma.request_*` exposées | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.9 Exit Gate Phase 4 — sync flags génériques

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `SyncScope.Fields []FieldKey` introduit | NOT DONE | | | |
| `SyncScope.Operations []string` introduit (flags non-FieldKey) | NOT DONE | | | |
| 12 champs FieldKey-based retirés de `SyncScope` (TeamMMR, KillsExpected, ...) | NOT DONE | | | |
| CLI `--field <key>` répétable + `--operation <name>` répétable | NOT DONE | | | |
| Aliases historiques préservés (test snapshot bitmask sur 20 combinaisons) | NOT DONE | | | |
| Deprecation warnings sur les vieilles options CLI (date butoir v6.5 documentée) | NOT DONE | | | |
| `FindMatchesMissingData` accepte `[]FieldKey` + `[]Operation` | NOT DONE | | | |
| Test idempotence backfill : 2e run = 0 appel API + 0 écriture DB | NOT DONE | | | |
| Métrique `sync.field_backfill_count` par FieldKey exposée | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.10 Exit Gate Phase 5 — frontend canonical-aware

| Item | Statut | Date | Evidence | Validateur |
|------|:-:|------|----------|:-:|
| `tools/codegen/canonical-ts/` créé | NOT DONE | | | |
| `apps/web/src/lib/canonical/fields.ts` généré + committé | NOT DONE | | | |
| CI lint fail si `fields.ts` ≠ sortie codegen | NOT DONE | | | |
| Client API TS regen depuis OpenAPI Huma, build front passe | NOT DONE | | | |
| Hook `useFieldLabel(field, locale)` créé + tests Vitest | NOT DONE | | | |
| Hook `useCapability(cap)` créé + tests Vitest | NOT DONE | | | |
| Composant `<StatRow field=... value=... />` créé + tests | NOT DONE | | | |
| Capability gating au routeur (`<Route capability="...">`) | NOT DONE | | | |
| Snapshot Playwright Halo : diff visuel ≤ 0.5% sur 4 routes critiques | NOT DONE | | | |
| Snapshot Playwright synthetic_test_title : capability-gated sections absentes | NOT DONE | | | |
| Aucun composant ne hardcode un FieldKey en string littéral (lint custom) | NOT DONE | | | |
| Couverture `apps/web/src/lib/canonical/` ≥ 90%, hooks ≥ 80% | NOT DONE | | | |
| 12 items de §8.2 | NOT DONE | | | |

### 8.11 Validation finale (cross-phase)

À la clôture de la Phase 5, ajouter une PR « validate: synthetic_test_title E2E » qui :

1. Crée un dossier `internal/games/synthetic_test_title/` complet (data + semantic + asset_url + ddl).
2. Définit un `fields.toml` avec un subset de FieldKeys (kills, deaths, match_id, durations).
3. Lance la suite COMPLÈTE des tests existants (Go + TS) avec `LEVELUP_TITLE=synthetic_test_title`.

Cette PR est le **critère de réussite final**. Elle doit passer SANS modifier :
- `internal/service/`
- `internal/api/`
- `apps/web/src/{features,components,routes}/`

Si une de ces zones doit être modifiée pour faire passer la PR, c'est que le plan a échoué — retour en Phase {N} pour corriger.

**Validation finale** : ajouter un titre `synthetic_test_title` avec un sous-ensemble de fields (kills, deaths, match_id, durations) ; vérifier que :
- Toutes les routes répondent sans 500.
- Tous les fields canonical absents → JSON omit ou `null`.
- Le front masque les sections concernées sans erreur console.
- Le pipeline CI `synthetic_test_title-build-and-test` est vert.
- Aucune modification dans `internal/service/`, `internal/api/`, `apps/web/src/{features,components}/` n'a été nécessaire au-delà de l'enregistrement du titre dans le Resolver.

---

## 9. Blockers / risques connus

| Risque | Probabilité | Mitigation |
|---|:-:|---|
| Migration Huma sur 80 handlers casse une route en prod | Élevée | Migration par groupes (4 groupes de 25-30 handlers), tests de contrat snapshot par route avant/après, smoke test E2E systématique. Possibilité de revert handler-par-handler si régression. |
| Validation Huma plus stricte qu'avant rejette des inputs auparavant tolérés | Moyenne | Audit des regex/whitelist actuels (ex: `playlistOrSessionPattern`), porter en tags Huma à l'identique. Tests des cas limites avec inputs de prod. |
| Client TS regen change la forme des types (camelCase, optionals) → cascade front | Élevée | Phase 3b ne casse pas le contrat JSON (omitempty conservé). Test snapshot des fixtures front avant/après regen. Coordonner avec Phase 5. |
| Bench Phase 2 montre slowdown > 15% sur SELECT dynamique | Moyenne | Garder `Q12` spécialisé pour les hot paths (match_view), n'utiliser FieldKey-based que pour les routes secondaires. Documenter la décision. |
| Migration des sync flags casse les scripts utilisateur (`--mmr`, `--skill`) | Moyenne | Aliases historiques préservés via map, deprecation warning, retrait à v6.5 |
| ADR 0011 conflictuel avec le plan | Faible (résolu Phase 0) | Phase 0 met à jour ADR 0011 ou crée 0014 |
| Coût total dépassé (50 j vs estimé 65 j) | Élevée | Phasage strict, chaque phase mergeable indépendamment, possibilité de geler après Phase 3a (cible « DTO propres + Huma reporté ») |
| DDL par titre rompt les outils ops existants (backup, restore, diagnose) | Moyenne | Audit `internal/ops/` en Phase 1.5, adapter `BackupRunner` pour itérer sur tous les titres enregistrés |
| Codegen TS canonical (D5) divergence avec Go | Faible | CI lint compare le fichier généré au commit, fail si diff |
| Huma compatibility avec middlewares chi existants (auth, CSRF, slog HTTP) | Moyenne | Huma utilise `huma.NewAPI(chi)` adapter — les middlewares chi continuent de tourner. Tester explicitement chaque middleware en début de Phase 3b. |

---

## 10. Anti-patterns à NE PAS faire

- **God refactor en une seule PR** — chaque phase = 1 ou plusieurs PR mergeables.
- **Casser l'API publique sans deprecation** — tous les changements de schema OpenAPI passent par un cycle deprecate → remove (2 versions).
- **Pousser `canonical.PlayerMatchRow` directement dans le DTO HTTP** — viole ADR 0011. Le DTO HTTP reste un view-model service composé.
- **Schema EAV** — perd les bénéfices colonnar de DuckDB. Préférer une DB par titre (Phase 1.5).
- **Cas particuliers `if title == "halo_infinite"`** — toujours via capability. Lint custom CI.
- **Forcer la canonicalisation des features encore en chantier** (ex. nouvelles features Halo Infinite-only). Phase finale uniquement quand le périmètre est stable.
- **Mélanger flags FieldKey-based et flags non-FieldKey dans `SyncScope` sans découplage explicite** (Phase 5).
- **Régénérer OpenAPI manuellement après Phase 3** — la gen doit être en CI, pas dans la mémoire des devs.

---

## 11. Effort total estimé (révisé v2.1 — Huma intégré)

- Phase 0 : 2-3 j
- Phase 1 : 2-3 j
- Phase 1.5 : 4-5 j
- Phase 2 : 5-7 j (incl. 1 j bench)
- Phase 3a : 5-7 j (cleanup DTO)
- Phase 3b : 13-18 j (migration Huma sur ~80 handlers, 4 groupes)
- Phase 4 : 5-6 j
- Phase 5 : 6-8 j

**Total** : ~42-57 jours-personne, étalable sur 3-4 mois sans blocage du reste du dev.

**Fenêtre minimale viable** : Phases 0 → 3a = ~18-25 j → état « services title-agnostic + DTO propres mais OpenAPI manuel ». Phase 3b (Huma) peut être différée d'un trimestre si le ROI n'est pas clair, MAIS dans ce cas le client TS front reste désynchronisé du back.

**ROI Huma seul** : validation auto, gen permanente, élimination de la dette OpenAPI manuel (qui se cumule à chaque feature). Décisif si le rythme d'ajout de routes reste soutenu (>10 routes/an).

**ROI title-agnostic** : marginal tant qu'il n'y a qu'1 titre. Décisif dès le 2e titre — chaque titre ajouté coûte ~1-2 jours (capabilities.toml + DataAdapter + ddl/) au lieu de plusieurs semaines.

---

## 12. Pré-requis avant démarrage

- [ ] Sync atomique (heals + LocalFilmCache) mergé sur `main` (= état actuel de `feat/token-pool-parallel-sync`).
- [ ] Pas d'autre big refactor en parallèle (notamment côté frontend canonical pipeline).
- [ ] Cap définie sur les fields à inclure dans canonical (pas de scope creep en cours de Phase 1).
- [ ] Validation par toi (Guillaume) du phasage et des 6 décisions Phase 0 — quels risques tu acceptes, lesquels tu repousses.
- [ ] Décision sur la fenêtre minimale viable (Phases 0-4) vs full sweep (0-7).

---

## 13. Premier pas concret quand tu reprendras

```bash
git checkout main
git pull
git checkout -b refactor/title-agnostic-services

# Phase 0 — ADRs et setup (décisions D1-D6 déjà actées)
# 1. Mettre à jour ADR 0011 ou créer ADR 0014 (title-agnostic + DDL isolation)
# 2. Créer ADR 0015 (adoption Huma)
# 3. Entrée thought_log.md actant les 6 décisions
# Commit : "docs(adr): record decisions for title-agnostic refactor + Huma adoption"

# Phase 1 — inventaire FieldKey (5 tables shared)
# rg "p\.\w+|mp\.\w+|mr\.\w+|me\.\w+|w\.\w+|kvp\.\w+" \
#    apps/go-api/internal/platform/duckdb/queries_*.go > /tmp/cols.txt
# Pour chaque ligne, vérifier/ajouter dans canonical/fields.go + halo_infinite/mappings/fields.toml
# Extraire les magic constants Halo (medal_id 1512363953, mode prefixes) vers constants.toml
# Commit : "feat(canonical): exhaustive FieldKey coverage for Halo Infinite (5 tables)"
```

---

## 14. Changelog

- **v2.2 (2026-05-06)** :
  - **§5 Tests refondu** : seuils de couverture chiffrés par couche (BLOQUANTS), 5 sous-sections (seuils, non-régression par phase, parité multi-titre, dégradation, datasets obligatoires)
  - **§6 Logging refondu** : 6 lints CI bloquants, standards par opération, whitelist de clés structurées, métriques expvar par phase, 4 vérifications shell de fin de phase
  - **§8 Phase Exit Gate** : refonte complète. Tableau binaire DONE/NOT DONE daté + validateur + evidence par phase. 12 items transverses + items spécifiques. Aucun item optionnel. Tag git `phase-{N}-exit` sur HEAD.
  - **§8.11 Validation finale** : PR `synthetic_test_title` E2E qui doit passer sans modifier service/api/features front. Critère de réussite ultime.
  - Aucun changement du phasage ni de l'effort total.
- **v2.1 (2026-05-06)** :
  - 6 décisions D1-D6 actées (cf. session interactive 2026-05-06) :
    - D1 : `Value{Kind, Int, Float, Str, Bool, Time}` (wrapper typé)
    - D2 : `map[FieldKey]*Value` (présent+nil = NULL, absent = unsupported)
    - D3 : DB physique par titre
    - D4 : **Huma intégré au plan** (rewrite ~80 handlers) au lieu de swag/kin-openapi
    - D5 : Codegen Go → TS pour canonical
    - D6 : Service par service en PR atomique, pas de feature flag
  - Phase 3 fusionne ex-Phases 3 et 4 (cleanup DTO + migration Huma) en Phase 3a + 3b
  - Renumérotation : ex-Phase 5 → Phase 4, ex-Phase 7 → Phase 5 (Phase 6 EAV skipped retirée)
  - ADR 0015 (Huma) ajouté en Phase 0
  - Effort total révisé : 42-57 j (vs 36-49 j en v2)
  - Fenêtre minimale viable décalée à Phase 3a (~18-25 j)
  - Risques mis à jour avec spécifiques Huma (validation stricte, middlewares chi, client TS regen)
- **v2 (2026-05-06)** :
  - Phase 0 enrichie avec 6 décisions techniques bloquantes (D1-D6) + spike OpenAPI gen
  - Alignement explicite avec ADR 0011 (3 adapters préservés, view-models côté service)
  - Phase 1 étendue à 5 tables shared (pas seulement match_participants) + magic constants Halo
  - Phase 1.5 NOUVELLE : DDL par titre (cohérence ADR 0008)
  - Phases 3 et 4 réordonnées : OpenAPI gen avant réécriture domain (évite cascade YAML manuel)
  - Phase 5 découplée : flags FieldKey-based vs flags « stratégie sync »
  - Phase 7 : codegen TS canonical (D5) au lieu de duplication manuelle
  - Effort total révisé : 36-49 j (vs 21-31 j en v1, sous-estimé d'un facteur ~1.5)
  - Fenêtre minimale viable Phases 0-4 (~25-35 j) introduite
  - CI gate `synthetic_test_title-build-and-test` formalisé
  - Anti-pattern « pousser canonical dans le DTO » ajouté
- **v1 (2026-05-06)** : version initiale, voir historique git.

---

**Auteur** : Claude (session 2026-05-06).
**Revue v2** : après audit plan v1 (cf. session du 2026-05-06).
**À traiter par** : Guillaume, plus tard.
