# Plan : Axes d'amélioration architecturale

> **Date** : 2026-05-28
> **Statut** : À implémenter

---

## Ordre recommandé

| # | Axe | Pourquoi cet ordre |
|---|-----|--------------------|
| 1 | SQL helpers → package neutre | 30min, zéro risque, débloque un import propre avant le reste |
| 2 | Repos DuckDB → port interfaces | Cœur du refacto archi Go, s'appuie sur les fondations propres |
| 3 | `TokenStore` interface (auth) | Même pattern que l'Axe 2, couche auth |
| 4 | Types OpenAPI générés | Migration progressive frontend, peut se faire en parallèle feature par feature |
| 5 | Cohérence erreurs HTTP | Nettoyage localisé, pas d'impact sur les autres axes |
| 6 | Feature flags dead code + Prestige deadline | Hygiène, risque zéro, peut se faire à tout moment |

---

## Principe transverse : tests d'abord (characterization tests)

Pour chaque axe, la séquence est la même :

1. **Écrire les tests qui capturent le contrat actuel** (ce qui entre, ce qui sort, les erreurs
   possibles) — ces tests doivent **passer en vert** sur le code existant.
2. **Faire le refacto**.
3. **Vérifier que les mêmes tests passent toujours** — si l'un échoue, la modification a changé
   un comportement, ce qui n'était pas l'intention.

L'objectif n'est pas de tester la logique métier (qui ne bouge pas), mais de **verrouiller la
surface exposée** (signatures, valeurs retournées, effets de bord observables) pour que le refacto
soit prouvablement neutre.

---

## Axe 1 — SQL helpers → package neutre (30min)

### Tests avant modification

`Placeholders` et `ToAnySlice` sont des fonctions pures — cas idéal pour les characterization
tests.

Créer `internal/sqlex/helpers_test.go` **avant** de déplacer le code :

```go
// Verrouille le contrat : n entrées → n placeholders séparés par virgule
func TestPlaceholders(t *testing.T) {
    assert.Equal(t, "$1", Placeholders(1))
    assert.Equal(t, "$1, $2, $3", Placeholders(3))
    assert.Equal(t, "", Placeholders(0))
}

// Verrouille le contrat : []string → []any, ordre et valeurs préservés
func TestToAnySlice(t *testing.T) {
    in := []string{"a", "b", "c"}
    out := ToAnySlice(in)
    require.Len(t, out, 3)
    assert.Equal(t, "a", out[0])
    assert.Equal(t, "c", out[2])
}
```

Ces tests échouent d'abord (le package n'existe pas encore), puis passent une fois le code déplacé.

### Modification

Déplacer `Placeholders` et `ToAnySlice` de `internal/platform/duckdb/shared_query_helpers.go`
vers `internal/sqlex/helpers.go`.

Mettre à jour l'import dans `internal/api/handlers/patterns.go` (seul consommateur externe).

### Commits

```
test(sqlex): characterization tests for Placeholders and ToAnySlice
refactor(sqlex): move SQL-agnostic helpers out of platform/duckdb
```

---

## Axe 2 — Repos DuckDB → port interfaces ⭐

### Tests avant modification

Deux types de tests à écrire **avant** de toucher au câblage.

**a) Tests de contrat sur les repos concrets**

Verrouiller ce que `SkillV2Repo` et `SquadOffsetRepo` retournent réellement.
Ces tests s'exécutent contre une base DuckDB in-memory initialisée avec le schéma réel.

```go
// internal/platform/duckdb/skill_v2_repo_test.go
// Contrat : LoadState sur un xuid inexistant retourne nil, nil (pas d'erreur)
func TestSkillV2Repo_LoadState_notFound(t *testing.T) { ... }

// Contrat : UpsertState puis LoadState retourne le même état
func TestSkillV2Repo_roundtrip(t *testing.T) { ... }

// Contrat : LoadHyperparams retourne une map vide (pas nil) si aucun hyperparamètre
func TestSkillV2Repo_LoadHyperparams_empty(t *testing.T) { ... }
```

```go
// internal/platform/duckdb/squad_offset_repo_test.go
// Contrat : LoadSquadOffsets est mémoïsé (2 appels = 1 seule requête SQL)
func TestSquadOffsetRepo_memoization(t *testing.T) { ... }

// Contrat : UpsertSquadOffset puis LoadSquadOffsets retourne la valeur insérée
func TestSquadOffsetRepo_roundtrip(t *testing.T) { ... }
```

**b) Test de comportement de `RunLUSRV2Shadow` via mock**

Ce test vérifie que `RunLUSRV2Shadow` appelle bien les bonnes méthodes sur le repo avec les bons
arguments — sans toucher à DuckDB.

```go
// internal/sync/skill_v2_shadow_test.go
type mockSkillV2Repo struct {
    loadStateCalled     bool
    upsertStateCalled   bool
    lastUpsertedState   domain.SkillV2State
}
func (m *mockSkillV2Repo) LoadState(...) (*domain.SkillV2State, error) { ... }
// ... implémente port.SkillV2Repository

func TestRunLUSRV2Shadow_callsRepoMethods(t *testing.T) {
    mock := &mockSkillV2Repo{}
    engine := NewSyncEngine(...).WithSkillV2Repo(mock)
    // ...
    assert.True(t, mock.loadStateCalled)
    assert.True(t, mock.upsertStateCalled)
}
```

Ce test **échoue d'abord** (le champ `skillV2Repo` et le wither n'existent pas encore), puis
passe une fois le refacto fait.

### Modification

**Étape 1** — Créer les interfaces dans `port/` :

```go
type SkillV2Repository interface {
    LoadState(ctx context.Context, xuid, playlistGroup string) (*domain.SkillV2State, error)
    LoadAllStates(ctx context.Context, xuid string) ([]domain.SkillV2State, error)
    LoadHyperparams(ctx context.Context, playlistGroup string) (map[string]float64, error)
    LoadStateHistory(ctx context.Context, xuid, playlistGroup string) ([]domain.SkillV2State, error)
    UpsertState(ctx context.Context, state domain.SkillV2State) error
    UpsertHyperparam(ctx context.Context, h domain.LUSRHyperparam) error
}

type SquadOffsetRepository interface {
    LoadSquadOffsets(ctx context.Context, xuid, playlistGroup string) (map[string]float64, error)
    UpsertSquadOffset(ctx context.Context, o domain.SquadOffset) error
}
```

Ajouter les noops inline (pattern établi dans le fichier).

**Étape 2** — Champs + withers sur `SyncEngine` :

```go
// engine.go
skillV2Repo     port.SkillV2Repository
squadOffsetRepo port.SquadOffsetRepository

// engine_options.go
func (e *SyncEngine) WithSkillV2Repo(r port.SkillV2Repository) *SyncEngine { ... }
func (e *SyncEngine) WithSquadOffsetRepo(r port.SquadOffsetRepository) *SyncEngine { ... }
```

**Étape 3** — `skill_v2_shadow.go` : supprimer les instanciations locales, lire depuis `e.*`.
Garde en début de fonction :

```go
if e.skillV2Repo == nil {
    return fmt.Errorf("RunLUSRV2Shadow: skillV2Repo not injected")
}
```

**Étape 4** — Câblage au point d'entrée :

```go
engine.
    WithSkillV2Repo(duckdb.NewSkillV2Repo(sharedDB)).
    WithSquadOffsetRepo(duckdb.NewSquadOffsetRepo(sharedDB))
```

### Commits

```
test(sync): characterization tests for SkillV2Repo and SquadOffsetRepo contracts
test(sync): mock-based test for RunLUSRV2Shadow repo interactions
feat(port): add SkillV2Repository and SquadOffsetRepository interfaces
refactor(sync): inject skill repos via port interfaces instead of direct duckdb instantiation
```

**Durée estimée** : 2-3h (tests inclus). Risque faible.

---

## Axe 3 — `TokenStore` interface (auth)

### Contexte

`MultiUserTokenStore` est une struct concrète injectée par type concret dans les handlers et la
`ServiceRegistry` :

```go
// handlers/auth_xbox_oauth.go
type XboxOAuthHandler struct {
    authStore *auth_platform.MultiUserTokenStore  // type concret
}

// api/registry.go
type ServiceRegistry struct {
    authStore *auth.MultiUserTokenStore  // type concret
}
```

Conséquence : tester un handler auth nécessite créer un vrai répertoire sur disque. Les tests du
store lui-même sont excellents (831 lignes, `t.TempDir()`, concurrence), mais les handlers qui
le consomment ne peuvent pas utiliser de mock.

**Ce qui est déjà bien abstrait :** `TokenProvider` (MSAL/OAuth) est une interface ✅. Le package
`auth` n'a aucune dépendance DuckDB ✅ (ADR 0023 respecté).

**Ce qui reste concret :** `MultiUserTokenStore` + les fonctions globales
`halo.GetCachedPlayerTokens` / `halo.InvalidateCachedPlayerTokens` (singleton process).

### Tests avant modification

Écrire un test de handler auth qui utilise un mock du store **avant** de créer l'interface —
ce test échoue d'abord, puis passe une fois l'interface en place :

```go
// internal/api/handlers/auth_xbox_oauth_test.go
type mockTokenStore struct {
    upsertCalled bool
    loadResult   *auth.UserTokens
}
func (m *mockTokenStore) Load(xuid string) (*auth.UserTokens, error) { return m.loadResult, nil }
func (m *mockTokenStore) Upsert(tokens *auth.UserTokens) error       { m.upsertCalled = true; return nil }
func (m *mockTokenStore) LoadByGamertag(g string) (*auth.UserTokens, error) { ... }
func (m *mockTokenStore) UpdateOAuthRefreshToken(xuid, rt string) error     { ... }

func TestXboxOAuthHandler_callback_persistsTokens(t *testing.T) {
    store := &mockTokenStore{}
    h := NewXboxOAuthHandler(...).WithAuthStore(store)
    // ... simule le callback OAuth
    assert.True(t, store.upsertCalled)
}
```

### Modification

**Étape 1** — Créer l'interface dans `internal/auth/` (ou `internal/port/`) :

```go
type TokenStore interface {
    Load(xuid string) (*UserTokens, error)
    Upsert(tokens *UserTokens) error
    LoadByGamertag(gamertag string) (*UserTokens, error)
    UpdateOAuthRefreshToken(xuid, refreshToken string) error
}
```

`MultiUserTokenStore` implémente déjà toutes ces méthodes — aucune modification du store concret.

**Étape 2** — Remplacer les types concrets dans les consommateurs :

```go
// Avant
authStore *auth.MultiUserTokenStore

// Après
authStore auth.TokenStore
```

**Étape 3** — `halo.GetCachedPlayerTokens` / `halo.InvalidateCachedPlayerTokens` (priorité basse)

Ces fonctions globales sont un singleton process. Si le besoin de les mocker émerge, créer une
interface `TokenCache` et l'injecter — mais ce n'est pas urgent car c'est un cache interne sans
état partagé entre tests.

### Commits

```
test(auth): mock-based characterization tests for XboxOAuthHandler
feat(auth): introduce TokenStore interface
refactor(auth): inject TokenStore interface instead of concrete MultiUserTokenStore
```

**Durée estimée** : 1-2h. Risque faible — `MultiUserTokenStore` implémente déjà tout.

---

## Axe 4 — Types OpenAPI générés (frontend) ⭐ impact long terme

### Contexte

`apps/web/src/lib/api/types.ts` = **~3500 lignes** de types TypeScript maintenus manuellement.
Un fichier `generated.ts` (via `openapi-typescript`) existe déjà mais n'est pas branché.

**Bonne nouvelle (investigation Go) :** l'architecture est déjà **contract-first** côté Go.
`apps/go-api/api/openapi.yaml` est la source de vérité (~1000 lignes), et `make gen` génère les
types Go depuis ce fichier. La pipeline frontend peut pointer directement sur ce YAML statique —
pas besoin d'endpoint HTTP dynamique.

```json
// apps/web/package.json
"generate:types": "openapi-typescript ../go-api/api/openapi.yaml -o src/lib/api/generated.ts"
```

Note : un endpoint `/api/v1/lab/contracts` existe dans le code Go mais n'est pas encore monté
dans le routeur. Non bloquant pour cet axe.

### Tests avant migration

Avant de basculer une feature, écrire des **tests de compatibilité de types** avec `expect-type` :

```typescript
// src/lib/api/__tests__/types-compat.test-d.ts
import { expectType } from 'tsd'
import type { PlayerMatchRow } from '../types'      // manuel actuel
import type { components } from '../generated'      // généré OpenAPI

type GeneratedRow = components['schemas']['PlayerMatchRow']

// Doit passer EN VERT avant de basculer les imports
// Échoue si un champ manque ou diffère → documente la divergence à corriger
declare const g: GeneratedRow
expectType<PlayerMatchRow>(g)
```

### Modification (progressive, feature par feature)

1. **Câbler la génération** (voir commande ci-dessus)
2. **Pour chaque feature** :
   - Écrire le test de compatibilité de type
   - Corriger les divergences (dans `openapi.yaml` ou dans le type manuel)
   - Basculer les imports `types.ts` → `generated.ts`
3. **Supprimer `types.ts`** une fois toutes les features migrées

### Commits (par feature)

```
chore(web): wire openapi-typescript generation from go-api/api/openapi.yaml
test(web/types): type-compat tests for PlayerMatch types before migration
refactor(web/engagement): migrate to generated OpenAPI types
refactor(web/explorer): migrate to generated OpenAPI types
...
chore(web): remove manual types.ts
```

**Durée estimée** : 30min pour le câblage + ~1h par feature migrée.
Risque modéré — les divergences entre types manuels et générés peuvent être surprenantes.

---

## Axe 5 — Cohérence des erreurs HTTP (nettoyage)

### Contexte

Le helper `writeError()` existe et est utilisé correctement dans la grande majorité des handlers,
avec une shape uniforme `{code, message, retryable}` et des codes HTTP cohérents. C'est solide.

Trois fichiers dévient du standard et un problème transverse sur les middlewares :

| Fichier | Problème |
|---------|----------|
| `handlers/settings_backup.go` | `http.Error()` (text/plain) + `map[string]string{"error": ...}` au lieu de `writeError()` |
| `handlers/health_home.go` | Shape custom `{error, checks}` sur le 503 au lieu de `{code, message, retryable}` |
| `middleware/require_auth.go` + `require_admin.go` + `require_capability.go` | `json.NewEncoder(w).Encode()` direct — risque de double `WriteHeader` |
| Panic recovery (`chi.Recoverer`) | Retourne du texte brut, pas du JSON standardisé |

### Tests avant modification

Les tests existants (`contract_validate` middleware en mode `LEVELUP_CONTRACT_VALIDATE=1`)
vérifient déjà la shape en dev. Avant de corriger, ajouter des tests unitaires sur les endpoints
concernés qui assertent la shape JSON et le Content-Type :

```go
func TestSettingsBackupHandler_errorShape(t *testing.T) {
    // provoque une erreur 503
    resp := callHandler(...)
    assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
    var body map[string]any
    json.Unmarshal(resp.Body.Bytes(), &body)
    assert.Contains(t, body, "code")
    assert.Contains(t, body, "message")
    assert.Contains(t, body, "retryable")
}
```

### Modification

1. `settings_backup.go` → remplacer `http.Error()` et la map custom par `writeError()`
2. `health_home.go` → aligner la shape 503 sur `{code, message, retryable}`
3. Middlewares → extraire un helper `writeJSONError()` accessible depuis `middleware/` sans
   import circulaire (ex: dans `internal/api/apierror/` partagé)
4. Panic recovery → wrapper `chi.Recoverer` avec un middleware custom qui écrit du JSON standard

### Commits

```
test(api): assert JSON error shape on settings_backup and health_home endpoints
fix(api): align error responses to {code, message, retryable} standard
refactor(middleware): extract shared JSON error writer to avoid direct json.NewEncoder
```

**Durée estimée** : 1-2h. Risque très faible — pur nettoyage de surface.

---

## Axe 6 — Feature flags : dead scaffolding et deadline Prestige

### Contexte

Deux problèmes distincts identifiés dans `internal/config/`.

**Ce qui est bien :** centralisation dans `internal/config/`, injection via `AppConfig`,
testabilité avec `t.Setenv()`. Pattern propre.

### Problème A — 12 surface flags jamais routés (dead scaffolding)

Les flags `Career`, `History`, `Explorer`, `MatchView`... (`FeatureFlags` struct, 12 champs) ont
été préparés pour un routing Go/Python. Le seul consommateur réel est la commande diagnostic
`surface-status` qui les *affiche* — aucun code ne les utilise jamais pour router une requête.

C'est du dead code au sens du CLAUDE.md (infrastructure préparée, jamais activée).

**Action :** deux options selon l'intention :
- Si la migration Go/Python est abandonnée → supprimer `feature_flags.go` et `AllSurfaces`
- Si elle est différée → documenter explicitement dans le fichier avec une date de révision

### Problème B — Deadline Prestige non gardée en code

L'ADR 0005 stipule : *"si non activé en prod avant fin Q3 2026 → archiver ou supprimer le
module Prestige"*. Il n'existe aucun mécanisme dans le code qui signale cette échéance.

**Action :** ajouter un test de garde à date fixe :

```go
// internal/config/prestige_expiry_test.go
func TestPrestigeFlag_expiryReminder(t *testing.T) {
    deadline := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
    if time.Now().After(deadline) {
        t.Errorf("Prestige deadline reached (ADR 0005): decide to activate or delete the module")
    }
}
```

Ce test échoue au CI à partir du 2026-10-01 si personne n'a agi — c'est intentionnel.

### Tests avant modification

Pour le dead scaffolding, vérifier d'abord qu'aucun code non trouvé par l'analyse n'utilise
`BackendFor()` autrement que dans `cmd_ops.go` :

```bash
grep -r "BackendFor\|BackendGo\|BackendPython\|AllSurfaces" apps/go-api/ --include="*.go"
```

### Commits

```
test(config): add expiry guard test for Prestige flag per ADR 0005
chore(config): document surface flags as deferred scaffolding with review date
# ou, si migration abandonnée :
chore(config): remove unused surface backend switching scaffolding
```

**Durée estimée** : 30min–1h. Risque très faible.

---

## Récapitulatif

| Axe | Tests avant | Impact | Effort total | Risque |
|-----|-------------|--------|--------------|--------|
| **1. SQL helpers** | Tests unitaires fonctions pures | Propreté imports | ~1h | Très faible |
| **2. Repos → port interfaces** | Contrat repos + mock shadow | Architecture, testabilité Go | 2-3h | Faible |
| **3. TokenStore interface** | Mock-based handler tests | Testabilité auth | 1-2h | Faible |
| **4. Types OpenAPI** | Type-compat tests par feature | Maintenabilité long terme | 3-6h | Modéré |
| **5. Cohérence erreurs HTTP** | Assert shape JSON sur 3 endpoints | Fiabilité frontend | 1-2h | Très faible |
| **6. Feature flags dead code + Prestige deadline** | Grep + test expiry | Hygiène codebase | 30min–1h | Très faible |
