# Plan — Refactoring vers une architecture title-agnostic complète

**Objectif** : retirer/ajouter un champ d'un titre = **1 ligne dans `capabilities.toml` + 1 modif dans le `TitleDataAdapter`**, sans toucher aux services, repos, schemas DB, OpenAPI, frontend.

**Critère de succès** : pour valider le refactoring, on simule l'ajout d'un 2e titre (ex. `halo_5_guardians` minimal) avec un sous-ensemble de fields. Aucune modif requise dans `internal/service/`, `internal/api/`, `apps/web/` au-delà d'une route capability-gated.

**Branche cible** : `refactor/title-agnostic-services` (créer depuis `main` après merge de `feat/token-pool-parallel-sync`).

**Statut** : À démarrer. Document rédigé après l'audit du sync atomique (cf. `.ai/thought_log.md` 2026-05-06 « Sync atomic »).

---

## 1. Diagnostic — fuites actuelles de l'abstraction

Sur la modif récente `drop assists_expected/assists_stddev` (Halo Infinite ne renvoie pas ces champs), j'ai dû toucher 14 fichiers dans 8 couches. Inventaire des fuites :

| Couche | Fichier | Type de fuite |
|--------|---------|---------------|
| **DB schema** | `migration/steps_shared.go` | Schema initial + ALTER → DROP COLUMN explicite par champ |
| **SQL direct** | `platform/duckdb/queries_match.go` (Q26, Q26MatchExpectedStats) | `SELECT mp.assists_expected, ...` codé en dur |
| **Scan** | `platform/duckdb/match_view_repo.go` | `row.Scan(&s.AssistsExpected)` couplé à l'ordre du SELECT |
| **Domain** | `domain/match_view.go` (`MatchExpectedStats`, `ExpectedStatsRaw`, `MatchScoreboardRow`) | Champs Halo-specific exposés à l'API publique |
| **OpenAPI** | `api/openapi.yaml` | Schema `MatchExpectedStats.expected_assists` |
| **Generated** | `internal/api/gen/types.gen.go` | Auto-généré depuis OpenAPI |
| **Service** | `service/match_view_service.go` | `out.ExpectedAssists = e.AssistsExpected` |
| **Sync flags** | `sync/scope.go`, `sync/backfill_flags.go`, `sync/backfill_cli.go` | `PBitAssistsExp`, `--assists-expected`, `scope.AssistsExpected` |
| **Tests** | 4 fichiers | refs à `PBitAssistsExp` / `scope.AssistsExpected` / `assists_expected` |

**Causes racines** identifiées :

1. **`domain.*` n'est PAS canonical** — c'est un type service-à-frontend qui mirror la structure DB Halo Infinite (1:1 avec les colonnes de `match_participants`). Donc retirer une colonne → casse l'API publique.
2. **SQL inline dans `queries_match.go`** au lieu de passer par un repo abstrait qui prend des `canonical.FieldKey`. Le service dépend des column names physiques.
3. **Sync flags Halo-Infinite-specific** (`PBitAssistsExp`, `--assists-expected`) — bitmask et CLI codés en dur sur le set de champs Halo Infinite. Pas de générique.
4. **OpenAPI YAML manuel** au lieu d'auto-généré depuis `canonical.*`. Le contrat front fuit la sémantique title-specific.

---

## 2. Architecture cible

### 2.1 Topologie

```
┌───────────────────────────────────────────────────────┐
│ apps/web/                                             │
│  - lit canonical.* via TanStack Query                 │
│  - useFieldLabel(FieldKey) / useCapability(cap)       │
│  - omet/grise les UI quand field nil ou cap absente   │
└───────────────────────────┬───────────────────────────┘
                            │ JSON canonical
                            ▼
┌───────────────────────────────────────────────────────┐
│ internal/api/handlers/                                │
│  - reçoit Request, appelle Service, sérialise         │
│  - 0 logique métier, 0 SQL                            │
└───────────────────────────┬───────────────────────────┘
                            │ canonical.MatchSummary, etc.
                            ▼
┌───────────────────────────────────────────────────────┐
│ internal/service/                                     │
│  - orchestre repo + analysis                          │
│  - retourne canonical.* uniquement                    │
│  - dégrade gracieusement si capability absente        │
└─────────────┬─────────────────────────────┬───────────┘
              │                             │
              ▼                             ▼
┌─────────────────────┐       ┌─────────────────────────┐
│ internal/analysis/  │       │ internal/port/          │
│  - algos purs       │       │  Repository interface   │
│  - 0 IO, 0 DuckDB   │       │  LoadMatch(matchID)     │
│  - prend canonical  │       │    → canonical.Match    │
│    en input/output  │       │  LoadField(FieldKey)    │
└─────────────────────┘       │    → canonical.Value    │
                              └─────────────┬───────────┘
                                            │ implémentation
                                            ▼
              ┌─────────────────────────────────────────┐
              │ internal/platform/duckdb/               │
              │  - lit shared/player DBs                │
              │  - mappe colonnes DB → canonical        │
              │    via TitleDataAdapter                 │
              │  - 0 SQL exposé en dehors               │
              └─────────────────────────────────────────┘
                                            ▲
                                            │
              ┌─────────────────────────────┴───────────┐
              │ internal/games/halo_infinite/           │
              │  adapter_data.go (TitleDataAdapter)     │
              │   - Load*() retourne canonical.*        │
              │   - extract from API JSON               │
              │  adapter_semantic.go (TitleSemanticAdp) │
              │   - labels FR/EN, asset URLs            │
              │  capabilities.toml                      │
              │   - liste exhaustive des FieldKeys      │
              │     supportées par ce titre             │
              └─────────────────────────────────────────┘
```

### 2.2 Règle d'or

> **Aucune fonction dans `service/`, `api/handlers/`, ou `apps/web/` ne doit jamais voir un nom de colonne DB ou un type Halo-specific.** Tout passe par `canonical.*` + `FieldKey` + `Capability`.

### 2.3 Comportement nullable cohérent

- Tous les fields canonicals optionnels sont **`*T` pointer**.
- Le frontend gère `null` partout (omitempty côté Go, `T | null` côté TS, fallback texte ou skeleton dans la UI).
- Une capability absente → field jamais renvoyé (omitempty) → frontend ne l'affiche pas.

---

## 3. Phases (ordre de risque/effort croissant)

### Phase 0 — Préparation (rapide, pas de code)

**Effort** : 1 jour
**Livrable** : ce document approuvé + branche créée + ADR mise à jour.

- [ ] Relire `docs/adr/0011-canonical-vs-semantic-adapter-separation.md` et identifier les écarts.
- [ ] Créer `docs/adr/0012-canonical-end-to-end.md` documentant la cible.
- [ ] Créer la branche `refactor/title-agnostic-services` depuis `main`.
- [ ] Ajouter ce plan en référence dans `CLAUDE.md` § Décisions architecturales.

### Phase 1 — Étendre `canonical/fields.go` à 100% des fields utilisés (rapide)

**Effort** : 1-2 jours
**Risque** : faible (additif uniquement, pas de breaking)
**Livrable** : tous les FieldKeys que les services lisent existent dans canonical, mappés vers les colonnes DB Halo Infinite via TOML.

- [ ] Lister tous les `SELECT mp.<col>` / `SELECT mr.<col>` dans `platform/duckdb/queries_*.go`.
- [ ] Pour chaque colonne, vérifier que le `canonical.FieldKey` existe.
  - Si oui : noter le mapping.
  - Si non : ajouter dans `internal/games/canonical/fields.go`.
- [ ] Mettre à jour `config/titles/halo_infinite/mappings/fields.toml` avec le mapping `field_key → db_column`.
- [ ] Tests : `internal/games/canonical/fields_test.go` vérifie l'exhaustivité des FieldKeys référencés par les TOMLs (lint custom déjà existant `Lint multi-titres`).

**Critère de complétion** : `grep -rE "p\\.\w+|mp\\.\w+|mr\\.\w+" internal/platform/duckdb/queries_*.go | sort -u` produit une liste 100% couverte par `fields.toml`.

### Phase 2 — Repository abstrait par FieldKey (moyen)

**Effort** : 4-6 jours
**Risque** : moyen (refactor des repos existants, mais migration progressive possible)
**Livrable** : nouvelle interface `port.MatchRepository` qui prend des `FieldKey[]`, retourne `map[FieldKey]canonical.Value`. Implémentation DuckDB qui résout via TOML mapping.

- [ ] Créer `internal/port/match_repository.go` :
  ```go
  type MatchRepository interface {
      LoadMatchSummary(ctx, matchID) (*canonical.MatchSummary, error)
      LoadMatchParticipant(ctx, matchID, xuid) (*canonical.PlayerMatchRow, error)
      LoadMatchFields(ctx, matchID, xuid, []FieldKey) (map[FieldKey]canonical.Value, error)
  }
  ```
- [ ] Implémenter `internal/platform/duckdb/match_repo_canonical.go` :
  - Construit le SELECT dynamique depuis le TOML mapping des FieldKey demandées.
  - Map les rows vers `canonical.Value` (typed wrapper sur `interface{}`).
  - Skip silencieusement les FieldKeys absentes du TOML (capability absente).
- [ ] Tests integration `match_repo_canonical_integration_test.go` :
  - Halo Infinite : tous les FieldKeys retournent valeurs ou nil cohérent.
  - Mock title (synthetic_title_b) : seules les FieldKeys déclarées dans son TOML sont résolues.
- [ ] Migration progressive : un service à la fois passe de `Q5/Q26/...` à `MatchRepository.LoadMatchFields(...)`.

**Critère de complétion** : `match_view_service.go` n'importe plus `internal/platform/duckdb`. Il dépend uniquement de `port.MatchRepository`.

### Phase 3 — Domain types → canonical (moyen)

**Effort** : 3-5 jours
**Risque** : moyen (impact sur OpenAPI → frontend)
**Livrable** : `domain/match_view.go` réécrit en termes de canonical. Plus de `MatchExpectedStats`, `ExpectedStatsRaw`, `MatchScoreboardRow` Halo-specific.

- [ ] Map chaque field de `MatchExpectedStats` à un FieldKey canonical.
- [ ] Réécrire `MatchViewResponse` :
  ```go
  type MatchViewResponse struct {
      Header   MatchViewHeader            `json:"header"`
      Rank     *canonical.SkillSnapshot   `json:"rank,omitempty"`
      Stats    canonical.PlayerMatchRow   `json:"stats"`     // contient kills, deaths, expected_*, etc.
      Teams    []canonical.MatchTeam      `json:"teams"`
      Medals   []canonical.PlayerMedal    `json:"medals"`
      // ...
  }
  ```
- [ ] Tous les champs `*T` (pointer) avec `omitempty` JSON.
- [ ] Si Halo Infinite ne supporte pas un field → adapter retourne nil → JSON omit → front ne l'affiche pas.
- [ ] OpenAPI : générer depuis canonical (cf. Phase 5).

**Critère de complétion** : `domain/match_view.go` ne contient plus aucun champ Halo-Infinite-specific (pas de `assists_expected`, `team_mmr`, etc. — tout via canonical).

### Phase 4 — Sync flags génériques (moyen)

**Effort** : 3-4 jours
**Risque** : moyen (touche le CLI utilisateur)
**Livrable** : sync scope et CLI prennent des `FieldKey[]` au lieu de booleans hardcodés.

- [ ] Remplacer `SyncScope.AssistsExpected/KillsExpected/...` par `SyncScope.Fields []canonical.FieldKey`.
- [ ] CLI : `--field <key>` répétable, ex. `--field kills_expected --field deaths_expected`.
- [ ] Aliases historiques préservés via map (`--mmr` → `[team_mmr, enemy_mmr]`, `--skill` → `PBitSkill` set).
- [ ] `backfill_flags.go::PBit*` devient un mapping dynamique chargé au boot depuis le TOML capabilities.
- [ ] FindMatchesMissingData prend `[]FieldKey` et construit le SQL `WHERE col IS NULL OR ...` dynamiquement.

**Critère de complétion** : ajouter un nouveau field stats = 1 ligne dans le TOML + 1 ligne dans le DataAdapter (extract depuis JSON). Le CLI le détecte automatiquement.

### Phase 5 — OpenAPI auto-généré depuis canonical (lourd)

**Effort** : 5-7 jours
**Risque** : élevé (impact contrat front, validation Redocly)
**Livrable** : `api/openapi.yaml` est généré depuis les types Go canonical. Un script `make openapi-gen` regénère.

- [ ] Choix outil : `swaggo/swag` ou `getkin/kin-openapi` ou `oapi-codegen` reverse mode.
- [ ] Annoter les handlers Go avec les schemas canonical correspondants.
- [ ] Pipeline CI : `go generate ./...` doit produire un YAML idempotent.
- [ ] Front régénère son client TS depuis le nouveau YAML.
- [ ] Smoke test E2E : 5 routes critiques (home, match-view, sessions, palmares, season-pass) renvoient toujours du JSON valide vs schema.

**Critère de complétion** : `api/openapi.yaml` n'est plus committé manuellement — généré + git-ignored, ou committé mais avec hook `pre-commit` qui regen.

**Note** : peut être différé après les autres phases si trop coûteux. Tant qu'on ne change pas le schema public, le manuel reste viable.

### Phase 6 — Schema DB title-agnostic (lourd, optionnel)

**Effort** : 5-10 jours
**Risque** : élevé (migration de prod)
**Livrable** : DB schema utilise un layout générique (table `match_field_values(match_id, xuid, field_key, value)`) au lieu de colonnes nommées.

- [ ] **Trade-off** : layout EAV (Entity-Attribute-Value) plus flexible mais perd les optimisations colonnar de DuckDB (slow scan, larger DB).
- [ ] **Décision recommandée** : NE PAS faire cette phase. Garder schema colonnar Halo-specific. La phase 2 (repo abstrait) suffit pour découpler le code applicatif du schema.
- [ ] Si un futur titre a un schema TRÈS différent (ex. plus de teams, structures imbriquées) : créer une nouvelle DB par titre (`shared_<title>.duckdb`) avec son propre schema. Le PathResolver retourne déjà la bonne DB par `titleSlug`.

**Critère de complétion** : non applicable (à NE PAS faire dans ce refactoring).

### Phase 7 — Frontend canonical-aware (lourd)

**Effort** : 4-6 jours
**Risque** : moyen (ajout d'abstractions front)
**Livrable** : composants UI utilisent `useField(FieldKey)` / `useCapability(cap)` au lieu d'accéder directement aux propriétés JSON.

- [ ] Créer `apps/web/src/lib/canonical/fields.ts` — mirror TS de `canonical.fields.go`.
- [ ] Hook `useFieldLabel(field, locale)` lit le TitleSemanticAdapter (déjà exposé via API `/i18n/manifest`).
- [ ] Composants `<StatRow field="kills_expected" value={...} />` qui se masquent automatiquement si value undefined.
- [ ] Capability gating au routeur : `<Route capability="lusr">` n'affiche pas si titre n'expose pas LUSR.

**Critère de complétion** : ajouter un titre = uploader son `i18n manifest` + `capabilities.toml` côté back. Le front l'utilise sans nouveau code.

---

## 4. Tests par couche

| Couche | Test type | Couverture cible |
|--------|-----------|------------------|
| `canonical/` | Unitaire pur | 100% des FieldKeys référencées dans TOMLs (lint existant) |
| `internal/games/halo_infinite/adapter_data.go` | Unitaire avec fixtures JSON Halo | Chaque Load*() retourne canonical valide pour fixtures réelles |
| `internal/port/` | Mock implementations | Tous les services testés contre mocks port.* |
| `internal/platform/duckdb/match_repo_canonical.go` | Integration `:memory:` | LoadMatchFields() retourne nil pour FieldKey absente du TOML |
| `internal/service/` | Tests avec mock port.* | Dégradation gracieuse sur capability absente |
| `internal/api/handlers/` | `httptest` | Schema OpenAPI respecté, omitempty correct |
| `apps/web/` | Vitest hook | `useField` retourne null si field absent |

**Cas de dégradation à tester explicitement** :
- Title sans `assists_expected` → service retourne `*PlayerMatchRow.AssistsExpected = nil` → JSON omit → UI masque la case → pas d'erreur 500.
- Title sans capability `match_film` → endpoint `/match/{id}/film` retourne 404 + body `{"capability":"match_film","supported":false}`.
- Title sans capability `lusr` → home page n'affiche pas le panneau LUSR (sans casser le reste).

---

## 5. Logging

Toutes les nouvelles fonctions de repo et adapter :
- `slog.DebugContext(ctx, "match_repo: load fields", "title", slug, "match_id", id, "field_count", len(fields))`
- `slog.WarnContext(ctx, "title_data_adapter: field unsupported", "title", slug, "field", key)` (1 fois par boot, pas par appel)
- `slog.ErrorContext(ctx, "match_repo: scan failed", "err", err, "match_id", id)`

Pas de `fmt.Println`. Clés structurées : `title`, `field`, `match_id`, `xuid`, `capability`.

---

## 6. Multi-titres — exigences capability

- [ ] Chaque FieldKey est listée dans `config/titles/{slug}/capabilities.toml` (ou héritée d'un default).
- [ ] Si une route HTTP nécessite une capability et le titre actif ne la supporte pas → handler renvoie `404 ErrCapabilityNotSupported` (pas de panic, pas de 500).
- [ ] Le frontend appelle `GET /api/v1/title/capabilities` au boot pour connaître les features dispos.
- [ ] **Aucune** comparaison `if titleSlug == "halo_infinite"` dans le code applicatif. Tout via `HasCapability(cap)`.

---

## 7. Délivrabilité — done definition par phase

Chaque phase est livrable indépendamment :

| Phase | Livrable testable | Dépendance |
|-------|-------------------|------------|
| 0 | Doc + branche + ADR | — |
| 1 | TOML capabilities exhaustif + lint pass | Phase 0 |
| 2 | `port.MatchRepository` + impl DuckDB + 1 service migré (ex. match_view) | Phase 1 |
| 3 | `domain/match_view.go` réécrit + tous les services qui le consomment migrés | Phase 2 |
| 4 | CLI `--field` répétable + scope.go générique | Phase 1 (TOML) |
| 5 | OpenAPI gen depuis canonical | Phase 3 |
| 6 | (skipped) | — |
| 7 | Front `useField()` + capability gating routes | Phase 3 + 5 |

**Validation finale** : ajouter un titre `synthetic_test_title` avec un sous-ensemble de fields (ex. seulement kills/deaths/maps), vérifier que :
- Toutes les routes répondent sans 500.
- Tous les fields canonical absents → JSON omit ou `null`.
- Le front masque les sections concernées sans erreur console.

---

## 8. Blockers / risques connus

| Risque | Probabilité | Mitigation |
|--------|:-----------:|------------|
| Frontend casse à cause d'OpenAPI regen (Phase 5) | Élevée | Pin version, regen manuel, smoke test E2E avant merge |
| Performance regression dans MatchRepository.LoadMatchFields (SELECT dynamique) | Moyenne | Bench avant/après. Si > 10% slowdown, garder les Q5/Q26 spécialisés en parallèle pour les hot paths |
| Migration des sync flags casse les scripts utilisateur (`--mmr`, `--skill`) | Moyenne | Garder les aliases historiques pendant 2 versions, deprecation warning |
| ADR 0011 incomplet ou contradictoire avec ce plan | Faible | Phase 0 met à jour l'ADR, ce plan devient annexe |
| Coût total dépassé (semaines au lieu de jours) | Élevée | Phasage strict, chaque phase mergeable indépendamment, pas de big-bang |

---

## 9. Anti-patterns à NE PAS faire

- ❌ **God refactor en une seule PR** — chaque phase = sa propre PR mergeable.
- ❌ **Ne pas casser l'API publique sans deprecation** — tous les changements de schema OpenAPI passent par un cycle deprecate → remove.
- ❌ **Schema EAV** (Entity-Attribute-Value) — perd les bénéfices colonnar de DuckDB. Préférer une DB par titre si vraiment hétérogène.
- ❌ **Cas particuliers `if title == ...`** — toujours via capability.
- ❌ **Forcer la canonicalisation des features encore en chantier** (ex. nouvelles features Halo Infinite-only). Phase finale uniquement quand le périmètre est stable.

---

## 10. Effort total estimé

- Phase 0 : 1 j
- Phase 1 : 1-2 j
- Phase 2 : 4-6 j
- Phase 3 : 3-5 j
- Phase 4 : 3-4 j
- Phase 5 : 5-7 j (peut être différé)
- Phase 6 : (skipped)
- Phase 7 : 4-6 j

**Total** : ~21-31 jours-personne, étalable sur 2-3 mois sans blocage du reste du dev.

**ROI** : marginal tant qu'il n'y a qu'1 titre. Décisif dès le 2e titre — chaque titre ajouté coûte ~1 jour (capabilities.toml + DataAdapter) au lieu de plusieurs semaines.

---

## 11. Pré-requis avant démarrage

- [ ] Sync atomique (heals + LocalFilmCache) mergé sur `main` (= état actuel de `feat/token-pool-parallel-sync`).
- [ ] Pas d'autre big refactor en parallèle (notamment côté frontend canonical pipeline).
- [ ] Cap définie sur les fields à inclure dans canonical (pas de scope creep).
- [ ] Validation par toi (Guillaume) du phasage — quels risques tu acceptes, lesquels tu repousses.

---

## 12. Premier pas concret quand tu reprendras

```bash
git checkout main
git pull
git checkout -b refactor/title-agnostic-services
# Phase 0 : lis docs/adr/0011, écris docs/adr/0012, commit.
# Phase 1 : grep "p\\.\w+\|mp\\.\w+\|mr\\.\w+" internal/platform/duckdb/queries_*.go > /tmp/cols.txt
# Pour chaque ligne, vérifie/ajoute dans canonical/fields.go + halo_infinite/mappings/fields.toml.
# Commit : "feat(canonical): exhaustive FieldKey coverage for Halo Infinite"
```

---

**Auteur** : Claude (session 2026-05-06, après livraison sync atomique).
**À traiter par** : Guillaume, plus tard.
