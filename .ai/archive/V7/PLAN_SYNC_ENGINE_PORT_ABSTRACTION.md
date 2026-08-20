# Plan : Réduire le couplage à DuckDB (portabilité de stack)

> **Date** : 2026-05-29
> **Statut** : À implémenter
> **Branche de travail** : `refactor/arch-port-abstractions`

## But originel — ne pas le perdre de vue

Question de départ : *« si je voulais quitter DuckDB, mon archi est-elle prête pour que ce soit le plus simple possible ? »*

Réponse : ~70-80% portable grâce au pattern ports/adapters (services métier déjà derrière
`port.*`, types domaine en Go pur). Ce plan attaque les **points de couplage fort restants** où du
code de logique dépend encore directement du type concret `*duckdb.*` au lieu d'une interface.
Chaque axe doit **réduire le couplage** — pas faire du rangement cosmétique.

> ⚠️ **Leçon de la v1 de ce plan** : les détails d'exécution avaient été écrits depuis des résumés
> d'exploration, pas depuis le code réel. Résultat : Axe 1 reposait sur une conclusion fausse et
> Axe 2 décrivait une mécanique inexistante. **Tout le contenu ci-dessous a été vérifié dans le code
> réel le 2026-05-29.** Règle : reconfronter chaque étape au fichier réel juste avant de coder.

## Statut vérifié par axe

| Axe | Couplage visé | Verdict (lu dans le code) |
|-----|---------------|---------------------------|
| **1. patterns.go** | Handler → `*duckdb.PlayerDB` + SQL brut | Réel. Le déplacement de helpers seul ne découple PAS (le type `*duckdb.PlayerDB` reste). Vrai fix = repo derrière interface. |
| **2. skill_v2 sync** | Logique → `*duckdb.SkillV2Repo` | Réel. Mais PAS de `SyncEngine.WithX` : `RunLUSRV2Shadow` est une fonction autonome, repo threadé via `shadowRunContext`. |
| **3. auth TokenStore** | Handlers → `*MultiUserTokenStore` | Réel et solide. Type concret injecté en 2 points + 1 instanciation locale. |
| **5. erreurs HTTP** | Shape non uniforme | Réel. `settings_backup.go` confirmé (l.24 `http.Error`, l.32 `{"error":...}`). |
| **6. feature flags morts** | Dead scaffolding | Réel. 12 surfaces consommées uniquement par `surface-status`. |
| **4. types OpenAPI (front)** | Sync manuelle types | Prémisse issue de l'exploration, **non re-vérifiée ligne par ligne**. Concerne le front, pas le couplage DuckDB. |

---

## Principe transverse : tests d'abord

Pour chaque axe : tests verrouillant le comportement observable **en vert sur le code actuel** →
refacto → mêmes tests toujours verts. Un test qui casse = comportement changé involontairement.

**Bonne nouvelle confirmée** : les filets de sécurité existent déjà pour les 2 axes les plus
sensibles :
- Axe 1 : `internal/platform/duckdb/shared_query_helpers_test.go` (couvre `Placeholders`/`ToAnySlice`).
- Axe 2 : `internal/sync/skill_v2_shadow_test.go` (**1200+ lignes**, ~15 cas contre DuckDB réel :
  flag off, flow 2v2, watermark idempotent, canonical, squad offset, cross-mode…). Tout refacto
  Axe 2 doit garder cette suite verte — c'est la preuve de neutralité.

---

## Axe 1 — Découpler patterns.go de DuckDB (handler → interface)

### Couplage réel (vérifié)

`internal/api/handlers/patterns.go` :
- Importe `internal/platform/duckdb` (l.26).
- Prend `*duckdb.PlayerDB` comme **type de paramètre** dans 5 fonctions (l.108, 174, 242, 296).
- Exécute du **SQL brut** dans le handler via `pdb.Player.Query(...)` (l.253, 306).

➡️ Déplacer `Placeholders`/`ToAnySlice` ne change rien : l'import reste à cause du type. Le vrai
couplage, c'est que la couche handler fait de l'accès données concret au lieu de dépendre d'une
interface — exactement le smell que les autres handlers évitent déjà via `port.*`.

### Objectif de découplage

Le handler ne doit plus connaître ni `*duckdb.PlayerDB` ni le SQL. Il dépend d'une interface
`port.PatternsRepository` ; l'implémentation DuckDB vit dans `internal/platform/duckdb/`.

### Étapes

1. **Tests d'abord** : caractériser ce que `loadPatternRows` / `loadPatternShared` /
   `loadPatternEnrichments` / `loadPatternSkillRanks` retournent aujourd'hui (round-trip sur
   DuckDB in-memory + cas vide). Ces tests doivent passer sur le code actuel.

2. **Définir l'interface** dans `internal/port/` :
   ```go
   type PatternsRepository interface {
       LoadRows(ctx context.Context, limit int) ([]patterns.MatchRow, error)
       LoadShared(ctx context.Context, limit int) ([]SharedPatternRow, error)
       LoadEnrichments(ctx context.Context, matchIDs []string) (map[string]EnrichmentRow, error)
       LoadSkillRanks(ctx context.Context, matchIDs []string) (map[string]SkillRankRow, error)
   }
   ```
   (Les types de retour `SharedPatternRow`/`EnrichmentRow`/`SkillRankRow` migrent de `handlers/`
   vers un package neutre — `domain` ou `port` — pour ne pas créer de dépendance inverse.)

3. **Implémenter** `duckdb.PatternsRepo` : déplacer le SQL brut de `patterns.go` (l.108-320) dans
   ce repo. C'est là, et seulement là, que `Placeholders`/`ToAnySlice` sont utilisés — pas besoin
   de les déplacer, ils restent légitimement dans `platform/duckdb`.

4. **Câbler** le handler sur l'interface (via `ProgressionResolver` ou un nouveau champ injecté),
   supprimer l'import `platform/duckdb` de `patterns.go`.

5. Vérifier que les tests de l'étape 1 (re-câblés sur le repo) passent toujours.

### Sous-étape optionnelle (cosmétique, faible valeur)

Si on veut quand même un package SQL neutre : `Placeholders`/`ToAnySlice` ont ~25 callers internes
au package `duckdb`. Les déplacer vers `internal/sqlex` est un churn de ~25 fichiers pour **zéro
gain de découplage** (ils sont déjà dans la couche infra). À ne faire que si un besoin neutre
émerge ailleurs. Sinon : laisser tel quel.

### Commits

```
test(patterns): characterization tests for pattern data loaders
feat(port): add PatternsRepository interface
refactor(duckdb): implement PatternsRepo, move raw SQL out of handler
refactor(handlers): depend on PatternsRepository, drop platform/duckdb import
```

**Effort** : 3-5h (déplacement SQL + types + câblage). Risque modéré (déplacement de SQL réel).

---

## Axe 2 — Découpler la logique LUSR v2 de `*duckdb.SkillV2Repo`

### Couplage réel (vérifié)

`internal/sync/skill_v2_shadow.go` :
- `RunLUSRV2Shadow(ctx, playerDB, sharedDB *sql.DB, xuid)` — **fonction autonome**, pas de méthode
  `SyncEngine`. (La v1 du plan inventait des `WithSkillV2Repo` sur un `SyncEngine` — **faux**.)
- Instancie le repo en ligne 89 : `repo := duckdb.NewSkillV2Repo(sharedDB)` et conditionnellement
  `squadRepo = duckdb.NewSquadOffsetRepo(sharedDB)` (l.94).
- Le repo concret est threadé via la struct `shadowRunContext` (champs `repo *duckdb.SkillV2Repo`,
  `squadRepo *duckdb.SquadOffsetRepo`) et passé à ~6 fonctions : `processOneShadowMatch`,
  `resolveGroupParams`, `applyMatchToSkillV2`, `loadStatesOrSeed`, `persistTeamSkillV2`,
  `propagateCrossModeLeak`, `computeTeamSquadOffsets`.
- **Appelé depuis** : `engine_postsync.go:379` (prod), `cmd/lusr_v2_replay/main.go:88` (CLI) + ~15
  tests — tous avec des `*sql.DB` bruts.

### Objectif de découplage

Toute la **logique de calcul** (les ~6 fonctions ci-dessus) doit dépendre d'une interface, pas du
type concret DuckDB. Ainsi l'algorithme devient testable avec un mock et indépendant du moteur.

### Option A — Découplage interne (recommandé, faible churn)

Garde la signature `RunLUSRV2Shadow(ctx, playerDB, sharedDB *sql.DB, xuid)` et l'instanciation
ligne 89 (seul point qui touche `duckdb`). Change **uniquement les types internes** :

1. **Tests d'abord** : la suite `skill_v2_shadow_test.go` existante EST le filet. La lire, la
   lancer en vert avant de toucher quoi que ce soit.

2. **Définir les interfaces** dans `internal/port/` :
   ```go
   type SkillV2Repository interface {
       LoadState(ctx context.Context, xuid, playlistGroup string) (*domain.SkillV2State, error)
       LoadHyperparams(ctx context.Context, playlistGroup string) (map[string]float64, error)
       UpsertState(ctx context.Context, state domain.SkillV2State) error
       // + les méthodes réellement appelées (à confirmer en lisant skill_v2_repo.go :
       //   LoadAllStates, LoadStateHistory, UpsertHyperparam selon usage réel)
   }
   type SquadOffsetRepository interface {
       LoadSquadOffsets(ctx context.Context, xuid, playlistGroup string) (map[string]float64, error)
       UpsertSquadOffset(ctx context.Context, o domain.SquadOffset) error
   }
   ```
   ⚠️ Avant d'écrire l'interface : lire `internal/platform/duckdb/skill_v2_repo.go` pour la liste
   EXACTE des méthodes appelées (ne pas deviner). `*duckdb.SkillV2Repo` doit la satisfaire sans
   modification.

3. **Retyper** `shadowRunContext.repo` / `.squadRepo` et les ~6 signatures de fonction de
   `*duckdb.SkillV2Repo` → `port.SkillV2Repository` (idem squad). Ligne 89 reste concrète : la
   variable est juste affectée à un champ d'interface.

4. Lancer la suite de tests → doit rester 100% verte (aucun comportement changé).

**Effet** : la logique de calcul est découplée de DuckDB et mockable. `skill_v2_shadow.go` importe
encore `duckdb` pour la seule instanciation ligne 89 — couplage résiduel d'1 ligne au lieu de
diffus dans tout l'algorithme.

### Option B — Découplage complet (churn élevé, optionnel)

Injecter le repo depuis les appelants : signature
`RunLUSRV2Shadow(ctx, skillRepo port.SkillV2Repository, squadRepo port.SquadOffsetRepository, playerDB *sql.DB, xuid)`.
Supprime l'import `duckdb` de `skill_v2_shadow.go` entièrement, mais oblige à toucher
`engine_postsync.go`, le CLI replay, et ~15 tests. À ne faire que si l'Option A ne suffit pas.

> **Recommandation** : Option A. Elle atteint le but (logique stack-agnostique, testable) pour un
> coût faible. L'import résiduel d'1 ligne ne bloque pas un changement de stack — il se remplace
> trivialement le jour où on swappe le repo.

### Commits (Option A)

```
feat(port): add SkillV2Repository and SquadOffsetRepository interfaces
refactor(sync): type LUSR v2 logic against port interfaces, not concrete DuckDB repo
```

**Effort** : 1-2h. Risque faible — filet de test massif existant, zéro changement de call-site.

---

## Axe 3 — `TokenStore` interface (auth) ✅ solide

### Couplage réel (vérifié)

- `handlers/auth_xbox_oauth.go:35` : `authStore *auth_platform.MultiUserTokenStore` (concret),
  wither `WithAuthStore(*MultiUserTokenStore)`. Méthodes utilisées : `UpdateOAuthRefreshToken`.
- `api/registry.go:76` : `authStore *auth.MultiUserTokenStore` (concret). Méthodes : `Load`,
  `UpdateOAuthRefreshToken` (registry_auth.go).
- `handlers/admin_auto_sync.go:162` : instancie `NewMultiUserTokenStore(...)` **en local dans le
  handler** + `LoadByGamertag`, `UpdateOAuthRefreshToken`.
- Test e2e existant `auth_xbox_e2e_test.go` : crée un vrai store sur tempdir → confirme qu'on ne
  peut pas tester sans disque aujourd'hui.

### Objectif

Une interface `TokenStore` couvrant les méthodes réellement consommées, pour mocker en test.

### Étapes

1. **Tests d'abord** : test de `XboxOAuthHandler` avec un mock store (échoue tant que le wither
   prend un type concret) → vert après.
2. **Définir l'interface** dans `internal/platform/auth/` :
   ```go
   type TokenStore interface {
       Load(xuid string) (*UserTokens, error)
       Upsert(tokens *UserTokens) error
       LoadByGamertag(gamertag string) (*UserTokens, error)
       UpdateOAuthRefreshToken(xuid, refreshToken string) error
   }
   ```
   `MultiUserTokenStore` l'implémente déjà — aucune modif du concret.
3. **Remplacer** `*auth.MultiUserTokenStore` → `auth.TokenStore` dans `registry.go`,
   `auth_xbox_oauth.go` (champ + wither). Pour `admin_auto_sync.go` : injecter le store plutôt que
   l'instancier en local (sinon le couplage reste).

### Commits

```
test(auth): mock-based test for XboxOAuthHandler token persistence
feat(auth): introduce TokenStore interface
refactor(auth): inject TokenStore interface instead of concrete MultiUserTokenStore
```

**Effort** : 1-2h. Risque faible.

---

## Axe 5 — Cohérence des erreurs HTTP ✅ solide (nettoyage)

### État réel (vérifié)

Le helper standard existe et domine. Déviations confirmées dans `handlers/settings_backup.go` :
- l.24 : `http.Error(w, "...", 503)` → text/plain au lieu de JSON.
- l.32 : `writeJSON(..., map[string]string{"error": err.Error()})` → clé `error` au lieu du shape
  `{code, message, retryable}`.

Autres cibles citées par l'exploration (à reconfirmer avant de toucher) : `health_home.go` (shape
custom sur 503), middlewares `require_auth/admin/capability` (`json.NewEncoder` direct → risque
double `WriteHeader`), panic recovery `chi.Recoverer` (texte brut).

### Étapes

1. **Tests d'abord** : assert Content-Type `application/json` + présence `code`/`message`/
   `retryable` sur les endpoints concernés.
2. Aligner `settings_backup.go` sur le helper standard.
3. Reconfirmer puis corriger `health_home.go` et les middlewares (extraire un writer JSON partagé
   accessible sans import circulaire).

### Commits

```
test(api): assert JSON error shape on settings_backup endpoint
fix(api): align settings_backup error responses to standard shape
refactor(middleware): shared JSON error writer (after re-verifying each site)
```

**Effort** : 1-2h. Risque très faible.

---

## Axe 6 — Feature flags morts + deadline Prestige ✅ solide

### État réel (vérifié)

- Les 12 surface flags (`FeatureFlags`, `AllSurfaces`, `BackendFor`) ne sont consommés QUE par la
  commande diagnostic `surface-status` (`cmd/levelup`). Aucun routing réel. Dead scaffolding
  confirmé. ⚠️ Plusieurs tests en dépendent (`feature_flags_test.go`, `pure_funcs_test.go`) — à
  supprimer avec, si on retire le scaffolding.
- Deadline Prestige (ADR 0005, fin Q3 2026) : aucun garde-fou en code.

### Étapes

1. **Surface flags** : décision binaire —
   - migration Go/Python abandonnée → supprimer `feature_flags.go` + `AllSurfaces` + tests + la
     branche `surface-status` ;
   - différée → ajouter en tête de `feature_flags.go` un commentaire daté de révision (anti
     "compatibility guard forever" du CLAUDE.md).
2. **Prestige** : test de garde à date fixe qui échoue au CI après le 2026-09-30 :
   ```go
   func TestPrestigeFlag_expiryReminder(t *testing.T) {
       deadline := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
       if time.Now().After(deadline) {
           t.Errorf("Deadline Prestige atteinte (ADR 0005) : activer en prod ou supprimer le module")
       }
   }
   ```

### Commits

```
test(config): expiry guard for Prestige flag per ADR 0005
chore(config): remove dead surface-switching scaffolding   # ou: document deferred with review date
```

**Effort** : 30min-1h. Risque très faible.

---

## Axe 4 — Types OpenAPI générés (frontend)

> **Non re-vérifié ligne par ligne** — prémisse issue de l'exploration. Concerne la maintenabilité
> du front, pas le couplage DuckDB. À valider dans le code avant implémentation.

Prémisse : `apps/web/src/lib/api/types.ts` (~3500 l. manuelles), `generated.ts` existe mais non
branché. Côté Go, `apps/go-api/api/openapi.yaml` est déjà source de vérité (contract-first,
`make gen`) → `openapi-typescript` peut pointer sur ce YAML statique.

Étapes : câbler la génération → tests de compat de types (`expect-type`) par feature → migration
progressive → suppression de `types.ts`.

**Effort** : 30min câblage + ~1h/feature. Risque modéré.

---

## Récapitulatif (ordre conseillé)

| # | Axe | But découplage | Effort | Risque |
|---|-----|----------------|--------|--------|
| 1 | **Axe 3 — TokenStore** | Handlers auth mockables | 1-2h | Faible |
| 2 | **Axe 6 — flags morts + Prestige** | Hygiène, dette documentée | 30min-1h | Très faible |
| 3 | **Axe 5 — erreurs HTTP** | Fiabilité contrat front | 1-2h | Très faible |
| 4 | **Axe 2 — logique LUSR v2 (Option A)** | Algo stack-agnostique | 1-2h | Faible |
| 5 | **Axe 1 — patterns.go → repo** | Handler sans `*duckdb.*` | 3-5h | Modéré |
| 6 | **Axe 4 — types OpenAPI** | Maintenabilité front | 3-6h | Modéré |

Ordre = du plus solide/rapide au plus lourd. Axes 1 et 2 visent explicitement le **découplage
DuckDB** (le but originel) ; 3/5/6 sont des gains de testabilité et d'hygiène attrapés en chemin.
