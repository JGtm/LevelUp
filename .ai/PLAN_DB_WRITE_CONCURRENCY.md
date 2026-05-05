# Plan — Concurrence DB : writes HTTP vs sync

> Créé : 2026-05-04
> Mis à jour : 2026-05-05 (v2 — corrections post-revue : deadlock sync↔Prestige, notifications déjà existant, ADR 0013, abstraction `PlayerDB.AcquireWriter`, séparation `DBWriter`/`DBExecutor`)
> Contexte : analyse des collisions potentielles entre les goroutines HTTP et le sync engine sur DuckDB
> Branche de référence : `feat/seasons-as-asset-kind` (mais applicable à tout)

---

## Pourquoi ce document existe

Gemini a suggéré le pattern "writer unique + channel buffer + ticker" pour DuckDB.
L'analyse de l'architecture a montré que le projet a déjà plusieurs mécanismes de sérialisation
(`dblease`, `WriteQueue`, `indexMu`, `sync.RWMutex`) — mais pas uniformément appliqués.

Le document poursuit deux objectifs :
1. **Tactique** — recenser les risques de collision actuels (P1 critique, P2 modéré, P3 faible)
2. **Stratégique** — diagnostiquer pourquoi ces oublis sont structurellement inévitables
   aujourd'hui, et proposer une refonte de typage qui rend la règle inviolable

**Approche retenue :** viser directement la cible architecturale (`LeasedWriter` type)
plutôt qu'un fix court-terme manuel suivi d'une refonte. Une seule branche, commits séquentiels,
un seul design.

> **Note sur PRESTIGE_ENABLED** — `prestige.IsEnabled()` retourne `true` par défaut
> (false uniquement si la var vaut explicitement `0/false/no/off`). Si l'env prod ne fixe pas
> la var, **P1 est déjà actif en prod** dès lors que les routes Prestige sont câblées (cf. ADR-0005).
> À vérifier dans le déploiement avant de calibrer l'urgence des commits 1-3.

---

## Architecture de sérialisation existante

| Mécanisme | Fichier | Scope | Protège |
|---|---|---|---|
| `dblease` | `internal/platform/dblease/lease.go` | Par chemin DB | Sync engine sur player DB + shared matches |
| `WriteQueue` | `internal/assets/write_queue.go` | Asset index | Écritures `asset_index` (metadata.duckdb) |
| `indexMu` | `internal/platform/duckdb/media_repo.go` | Par chemin DB | ATTACH/DETACH DuckDB pendant media indexing |
| `sync.RWMutex` | settings store | Process-wide | Lectures/écritures `app_settings.json` |
| Pool `sql.DB` (RW=1) | `internal/platform/duckdb/db.go` | Par chemin DB | Sérialise les `Exec()` concurrents intra-process |

**Note clé sur le pool :** `OpenReadWrite()` retourne une instance `*sql.DB` mise en cache par chemin.
Avec `maxOpenConns=1`, les `Exec()` concurrents depuis la même process sont sérialisés automatiquement
par `database/sql`. Le risque de collision *intra-process* est donc plus faible qu'il n'y paraît à première
lecture. Le `dblease` protège surtout contre les ouvertures concurrentes *avant* que le cache soit établi,
et contre les process externes (scripts Python, migrations).

**Abstractions par-DB existantes** (à réutiliser, pas recréer) :
- `*duckdb.PlayerDB` — abstraction par-joueur (cf. `internal/api/registry_notifications.go:49`)
  → on lui ajoute `AcquireWriter(ctx) (*LeasedWriter, error)` au commit 0
- `*duckdb.SharedSocialDB` / `*duckdb.SharedMatchesDB` — équivalents partagés (à vérifier nommage exact)
- Pas de package `paths` à créer — la DB connaît son propre chemin

---

## Diagnostic architectural

L'absence de lease dans Prestige HTTP n'est pas une simple erreur d'inattention — c'est la conséquence
prévisible d'une **convention non matérialisée dans le type system**.

### Couches actuelles

```
Layer 1 — DuckDB file lock                       (forcé par DuckDB)
Layer 2 — dblease (sync.Mutex par chemin)        ← OPT-IN, par convention
Layer 3 — sql.DB pool (1 conn RW)                ← sérialise auto les Exec()
Layer 4 — Repository.Exec(...)                   ← n'a pas connaissance du lease
```

### Responsabilité orpheline

| Couche | Responsabilité | Connaît le lease ? |
|---|---|---|
| `prestige.Service` | Orchestration métier | Devrait — ne le fait pas actuellement |
| `prestige_player_repo.go` | CRUD DuckDB | Non |
| `duckdb.OpenReadWrite()` | Ouverture connexion | Non |
| `dblease` | Mutex par path | Oui (mais standalone, opt-in) |

Le repo expose `db.Exec()` directement. Le service appelle le repo. Le lease vit dans son coin.
Aucune des trois couches ne porte la responsabilité d'imposer son passage.

C'est le pattern du **"savoir tribal"** : la règle existe et est documentée dans le package doc,
mais elle n'est pas portée par l'API. Le sync engine s'en souvient parce qu'il a été écrit en
sachant la contrainte ; Prestige HTTP a été écrit plus tard, sans ce contexte. **Le bug est
structurellement inévitable tant que la règle reste advisory.**

### Options architecturales évaluées

| Option | Principe | Avantage | Coût |
|---|---|---|---|
| **A — `LeasedWriter` type** | Méthodes repo prennent un type uniquement constructible via `Acquire()` | Garantie compile-time, ergonomie synchrone | Refactor signatures repo, gestion explicite des cas réentrance |
| **B — Lease dans le repo** | Chaque méthode repo acquiert le lease elle-même | Service ne sait plus rien | Pas de réentrance `sync.Mutex` → token contexte requis (debug pénible) |
| **C — Single-writer goroutine** | Type `WriteQueue` étendu aux player DBs | Élimine le lease par construction | Perd la propagation synchrone des erreurs HTTP |

**Choix retenu :** Option A. La réentrance n'est pas requise car on **propage explicitement** le
`*LeasedWriter` au sync hook Prestige (cf. §Réentrance ci-dessous), au lieu de tenter de le ré-acquérir.

### Réentrance — résolution du deadlock sync hook ↔ Prestige

**Risque :** [sync_hook.go:48](apps/go-api/internal/prestige/sync_hook.go#L48) `RunPostSyncHook` est appelé par
le sync engine **alors que celui-ci tient déjà le `*LeasedWriter` sur la player DB**.
`RunPostSyncHook` → `Service.EvaluateForUser` → `prestige_player_repo` qui re-acquiert le même
writer ⇒ deadlock (`sync.Mutex` non réentrant).

**Solution retenue :** propagation explicite via signature.

```go
// internal/prestige/service.go
type Service interface {
    // Variante HTTP : acquiert son propre writer.
    EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]EvaluationOutcome, error)
    // Variante sync : reçoit le writer déjà tenu par l'appelant.
    EvaluateForUserWithWriter(ctx context.Context, userID, titleSlug string, w *dblease.LeasedWriter) ([]EvaluationOutcome, error)
    // ... autres méthodes inchangées
}
```

- Le sync engine appelle `EvaluateForUserWithWriter(ctx, userID, slug, playerWriter)` au lieu de la version
  sans writer. `RunPostSyncHook` propage le writer en paramètre.
- Les handlers HTTP continuent d'appeler `EvaluateForUser(ctx, ...)` qui acquiert lui-même son writer.
- Aucun token contexte, aucune réentrance, `sync.Mutex` reste non réentrant et sûr.

**Coût :** une méthode supplémentaire dans l'interface `Service`, deux dans la struct concrète,
une signature étendue pour `RunPostSyncHook`. Refactor ciblé, sans token magique.

---

## Cartographie complète des writers par DB

### `metadata.duckdb`
| Writer | Déclencheur | Sérialisation |
|---|---|---|
| `DuckDBIndexStore.PersistIndex()` | Asset resolver (nav user) | ✅ `WriteQueue` (goroutine unique) |
| Sync engine (career enrichment) | Post-sync | ✅ `dblease` |
| Migrations / EnsureTable | Startup | Startup séquentiel |

### `shared_matches_v2.duckdb`
| Writer | Déclencheur | Sérialisation |
|---|---|---|
| Sync engine (matchs, participants, médailles…) | Sync périodique | ✅ `dblease` |

### `stats.duckdb` (par joueur)
| Writer | Déclencheur | Sérialisation |
|---|---|---|
| Sync engine (enrichment, career, sessions…) | Sync périodique | ✅ `dblease` |
| **`prestige.Service` — CreateChallenge** | **HTTP temps réel** | ❌ **aucun lease** |
| **`prestige.Service` — UpdateChallenge** | **HTTP temps réel** | ❌ **aucun lease** |
| **`prestige.Service` — AbandonChallenge** | **HTTP temps réel** | ❌ **aucun lease** |
| **`prestige.Service` — CreateArc** | **HTTP temps réel** | ❌ **aucun lease** |
| **`prestige.Service` — EvaluateForUser (post-sync)** | **Sync hook (writer déjà tenu)** | ⚠️ doit recevoir le writer (pas le ré-acquérir) |

### `shared_social.duckdb`
| Writer | Table | Déclencheur | Sérialisation |
|---|---|---|---|
| `NotificationsRepo.Insert/Emit()` | `player_notifications` | Sync + HTTP + startup | ⚠️ Pool sql.DB (1 conn) |
| `NotificationsRepo.MarkRead/Unread/Delete` | `player_notifications` | HTTP | ⚠️ Pool sql.DB |
| `NotificationsRepo.UpsertPreferences` | `notification_preferences` | HTTP | ⚠️ Pool sql.DB |
| `MediaRepo.insertMediaFile` | `media_files` | HTTP upload + startup | ✅ `indexMu` |
| `MediaRepo.AssociateMediaWithMatches` | `media_match_associations` | HTTP reset-index + startup | ✅ `indexMu` |
| `MediaRepo.BackfillThumbnailPaths` | `media_files` | HTTP reset-index | ✅ `indexMu` |
| **`MediaRepo.SetMediaLike`** | **`media_files`** | **HTTP temps réel** | ⚠️ Pool sql.DB |
| **`MediaRepo.ToggleSharedLike`** | **`media_likes`** | **HTTP temps réel** | ⚠️ Pool sql.DB |
| **`SocialRepo.ToggleMatchFavorite`** | **`match_favorites`** | **HTTP temps réel** | ⚠️ Pool sql.DB |
| `PrestigeSocialRepo.EmitEvent` | `prestige_events`, `user_prestige` | Sync/Prestige API | ⚠️ Pool sql.DB |
| `PrestigeSocialRepo.UpsertUserPrestige` | `user_prestige` | Prestige API | ⚠️ Pool sql.DB |
| `PrestigeSocialRepo.Create/AddMember…` | `squad*` | Prestige API | ⚠️ Pool sql.DB |

**Légende ⚠️ Pool sql.DB :** pas de collision de corruption (pool=1 sérialise les Exec),
mais risque de contention / head-of-line blocking si une opération longue bloque les courtes.

---

## Problèmes identifiés

### P1 — Prestige & Challenges : HTTP sans dblease `[CRITIQUE]`

Les méthodes write HTTP du service Prestige (`CreateChallenge`, `UpdateChallenge`,
`AbandonChallenge`, `CreateArc`) ouvrent `stats.duckdb` en ReadWrite sans acquérir le `dblease`.
Si le sync engine tient le lease sur ce fichier (DuckDB-level lock), la tentative d'ouverture
depuis le handler HTTP peut échouer avec "database is locked".

- **État réel :** vérifier `PRESTIGE_ENABLED` en prod. Si la var n'est pas posée explicitement,
  Prestige est actif (default `true`) → P1 est **bloquant immédiat** dès que les routes sont câblées.
- **Résolu par :** commits 2 + 3 du plan (LeasedWriter sur `prestige_player_repo` + `prestige_social_repo`)
  + `EvaluateForUserWithWriter` pour le sync hook.

### P2 — Notifications multi-sources : contention potentielle `[MODÉRÉ]`

`NotificationsRepo` écrit dans `shared_social.duckdb` sans coordination explicite. Le pool
`sql.DB` (1 connexion) sérialise les `Exec()`, mais plusieurs sources peuvent émettre
simultanément (sync, HTTP, post_sync_deltas, startup). Risque de head-of-line blocking si
une opération longue (CapAndSweep sur 500+ rows) bloque les courtes (MarkRead).

- **Résolu par :** commit 4 du plan (migration du `notifications.Service` existant + LeasedWriter
  côté repo).

### P3 — Media likes & match favorites : incohérence possible `[FAIBLE]`

`SetMediaLike` (écrit `media_files.liked`) et `ToggleSharedLike` (écrit `media_likes`)
sont deux `Exec()` séquentiels non atomiques. Si `ToggleSharedLike` échoue, `media_files.liked`
est mis à jour sans que `media_likes` le reflète.

- **Résolu par :** commit 6 du plan (transaction atomique au passage de la migration likes)

---

## Ce qui est déjà correct — ne pas toucher

| Domaine | Mécanisme | Statut |
|---|---|---|
| Asset kinds (images, JSONs) | `WriteQueue` goroutine unique | ✅ Correct |
| Media indexing (upload, reset-index) | `indexMu` par path | ✅ Correct |
| Sync engine (matchs, career, citations) | `dblease` | ✅ Correct |
| Settings (`app_settings.json`) | `sync.RWMutex` | ✅ Correct |
| Career rank / Spartan ID / adornment | Sync only, sous dblease | ✅ Correct |
| Seasons as asset kind (cette branche) | Aucune écriture DB (TOML statique) | ✅ Correct |
| `notifications.Service` (existe déjà) | Couche service propre | ⚠️ À étendre, pas recréer |

---

## Stratégie de tests — non-régression blindée

Le refactor touche ~600 tests existants (prestige 146, notifications 19, sync 400+, duckdb
repos 80, handlers 50, dblease 10) et introduit de la concurrence DB. Une simple "tests verts"
ne suffit pas — il faut un **contrat explicite de non-régression** appliqué à chaque commit.

### Baseline pré-migration (commit 0, avant tout autre changement)

Capturer l'état initial comme **contrat figé** :

```bash
# Sur main, juste avant de créer la branche
go test -count=3 -race ./... -json > .ai/baselines/tests_pre_migration.jsonl
go test -coverprofile=.ai/baselines/coverage_pre_migration.out -covermode=atomic ./...
go tool cover -func=.ai/baselines/coverage_pre_migration.out > .ai/baselines/coverage_pre_migration.txt
```

Ces fichiers sont commités sur la branche dès le commit 0. À chaque commit suivant :
- **Tous les tests présents dans la baseline doivent rester verts** (`go test -run` sur la liste extraite).
- **Aucun test ne doit être supprimé** sauf justification documentée dans le commit message.
- **Le coverage par package ne doit pas baisser** de plus de 1 % (vérifié en CI).
- Les tests **renommés** (ex. signature changée) doivent l'être par migration mécanique
  (`gofmt -r` ou `gopls rename`), pas par réécriture libre.

Un script `scripts/check_test_baseline.sh` (à ajouter au commit 0) automatise ces 4 vérifs et
échoue le CI si un seul critère est violé.

### Invariants critiques à vérifier par property-based tests

Quatre invariants doivent être testés via `testing/quick` ou `pgregory.net/rapid` (déjà utilisé
par certains projets Go internes — sinon table-driven sur N=100 cas générés) :

1. **Release garanti** — pour toute séquence d'acquisitions/releases d'un `*LeasedWriter`,
   le compteur `acquire_total - release_total` retombe à 0 quand toutes les goroutines terminent.
2. **Pas de double-release** — release() est idempotent (un second appel est no-op, pas une panic).
3. **Idempotence des writes Prestige** — pour toute séquence de `CreateChallenge` /
   `UpdateChallenge` / `AbandonChallenge` exécutée 1× ou 2× (replay), l'état final est identique.
4. **Ordre de verrouillage stable** — si un acquireur prend `(player, shared_matches)` et un
   autre prend les mêmes deux DBs dans l'ordre inverse → le test détecte le risque de deadlock
   et échoue. Sert à blinder l'ordre documenté dans `engine.go`.

### Tests de fuites de ressources

Ajouter un helper de test `dblease.AssertNoLeasedWriters(t)` qui inspecte le compteur global :
- Au début et à la fin de **chaque test** des packages `prestige`, `notifications`, `social`, `media`,
  `sync` qui touche un writer → aucune fuite tolérée.
- Test dédié `TestNoLeakOnPanic` : provoque un panic au milieu d'une opération service →
  vérifier que `defer release()` libère bien le writer avant de remonter.
- Test dédié `TestNoLeakOnCtxCancel` : annule le ctx pendant une acquisition longue → writer libéré.

### Tests de chaos / scénarios catastrophes

À ajouter en build tag `integration` :
- **Panic au milieu d'une transaction atomique** (commit 6) — vérifier rollback DB + release writer.
- **Ctx canceled pendant `EvaluateForUserWithWriter`** — vérifier release writer + état Prestige cohérent.
- **Kill brutal du process** (signal SIGKILL simulé via `os.Exit(1)` sous-process) — au redémarrage,
  vérifier que les leases ne sont pas tenus (mutexes intra-process disparaissent avec le process,
  mais valider qu'aucun fichier `.lock` résiduel ne reste).
- **DuckDB-level lock externe** (autre process tient `stats.duckdb`) — vérifier que `OpenReadWrite`
  retourne une erreur propre et que le service mappe en `ErrDBLocked` → 503.

### Tests de performance — pas de régression de latence

Benchmarks à ajouter au commit 1 (baseline) puis à valider à chaque commit :

```go
// internal/platform/dblease/writer_bench_test.go
func BenchmarkAcquireRelease_Uncontended(b *testing.B)  // 1 goroutine
func BenchmarkAcquireRelease_Contended_2(b *testing.B)   // 2 goroutines
func BenchmarkAcquireRelease_Contended_10(b *testing.B)  // 10 goroutines
```

Cibles :
- Uncontended : < 1 µs par acquire+release (overhead du wrapping `*sql.DB`).
- Contended_2 : < 100 µs en moyenne (2 goroutines alternent).
- Contended_10 : pas de starvation visible (P99 < 10× P50).

Si une régression > 20 % apparaît à un commit, investiguer avant de continuer.

### Matrice de couverture par invariant métier

| Invariant | Test responsable | Type | Commit |
|---|---|---|---|
| Aucun double-write corrompu | `TestSyncVsPrestigeConcurrent` | Intégration | 2 |
| Pas de deadlock sync↔hook | `TestSyncHookNoDeadlock` | Intégration | 3 |
| `Emit` reste best-effort | `TestNotificationsServiceEmit_LeaseTimeoutSilent` | Unitaire | 4 |
| `MarkRead` propage `ErrDBLocked` | `TestNotificationsServiceMarkRead_LeaseTimeout` | Unitaire | 4 |
| Match favorite toggle bidirectionnel | `TestSocialService_ToggleMatchFavorite_Bidirectional` | Unitaire | 5 |
| Like atomique = rollback sur échec | `TestMediaLikeRollback` | Intégration | 6 |
| Sync ingère le même nb de matchs | `TestSyncDeltaProducesSameOutput` | Intégration | 7 |
| Pas de fuite de writer sur panic | `TestNoLeakOnPanic` | Unitaire | 1 |
| Pas de fuite de writer sur ctx cancel | `TestNoLeakOnCtxCancel` | Unitaire | 1 |
| Idempotence CreateChallenge | `TestPrestigeCreateChallengeIdempotent` (rapid) | Property | 2 |
| Release idempotent | `TestLeasedWriterReleaseIdempotent` | Unitaire | 1 |
| Ordre de verrouillage respecté | `TestSyncEngineLockOrder` | Intégration | 7 |

Tout test absent de cette matrice qui couvrirait un invariant équivalent doit être ajouté à
la matrice (la matrice est le contrat de revue, pas une approximation).

---

## Plan d'implémentation — une branche, commits séquentiels

**Branche :** `refactor/leased-writer-enforcement`
**Stratégie :** viser directement la cible architecturale (Option A `LeasedWriter`),
sans fix court-terme intermédiaire. Chaque commit est reviewable indépendamment et laisse
le code dans un état cohérent (compile + tests verts + baseline non-régressée).

**Effort total ré-estimé :** **8-9 jours** (révisé après ajout de la grille de tests blindée :
+ 0.5 j commit 0 baseline, + 0.5 j sur chaque commit pour les tests d'invariants).

### Commit 0 — Baseline de tests `[rapide, 0.5 j]`

**Découverte au démarrage :** `*duckdb.DB` expose déjà `Path()` (cf. `internal/platform/duckdb/db.go:211`).
Donc **aucune modification de `pool.go` n'est nécessaire au commit 0**. Le commit 1 pourra
ajouter directement les méthodes `AcquireWriter*` à `*PlayerDB` en utilisant `pdb.Player.Path()`,
`pdb.SharedSocial.Path()`, etc.

Conséquence : le commit 0 est une pure capture de baseline + script CI, **aucun code Go modifié**.

**Fichiers — baseline (capturés depuis `apps/go-api/` avec `make test` flags) :**
- `.ai/baselines/tests_pre_migration.jsonl` — sortie de `go test -tags=integration -count=1
  -timeout=300s -p 1 -json ./...` (aligné sur les flags du Makefile, pas `-race` ni `-count=3`
  qui ne sont pas la convention du projet et coûteraient en CI).
- `.ai/baselines/tests_pre_migration.exitcode` / `.stderr` — diagnostic.
- `.ai/baselines/coverage_pre_migration.raw` — profil binaire.
- `.ai/baselines/coverage_pre_migration.txt` — `go tool cover -func` (lisible).
- `scripts/check_test_baseline.sh` — vérifie : (a) tous les tests baseline existent encore,
  (b) coverage global ne baisse pas de plus de 1 point. Échoue le CI sur violation.
- (Optionnel) `.github/workflows/test_baseline.yml` ou cible Makefile `test-baseline` — à
  câbler dans une PR séparée sur l'infra CI ; pour cette branche, le script est exécuté
  manuellement avant chaque commit.

**Tests :**
- Smoke test : `bash scripts/check_test_baseline.sh tests` passe sur le commit lui-même
  (auto-test : la baseline est strictement incluse dans le run courant, ce qui est trivialement
  vrai au commit 0 puisque c'est le même code).

**Invariants protégés par ce commit :**
- Aucun comportement runtime modifié (zéro fichier Go touché).
- Baseline figée et auditable.

**Done :** baseline figée, script de vérification disponible, prêt pour commit 1.

---

### Commit 1 — Introduire `LeasedWriter` + interfaces `DBExecutor`/`DBWriter` `[moyen, 1 j]`

**Fichiers :**
- `internal/port/dbexecutor.go` (nouveau) — interface `port.DBExecutor` :
  `ExecContext`, `QueryContext`, `QueryRowContext`. Satisfaite par `*sql.DB` ET `*sql.Tx`.
  C'est le type que prennent les méthodes write des repos (pour accepter une transaction).
- `internal/port/dbwriter.go` (nouveau) — interface `port.DBWriter` : `port.DBExecutor + BeginTx`.
  Satisfaite uniquement par `*sql.DB` et `*LeasedWriter`. Utilisée par les services qui ouvrent
  des transactions atomiques (commit 6).
- `internal/platform/dblease/writer.go` (nouveau) — type concret `LeasedWriter` qui implémente
  `port.DBWriter` + wrappe `*sql.DB` + `release()`
- `internal/platform/dblease/metrics.go` (nouveau) — compteurs `expvar` (cohérent avec ADR-0009),
  groupés par **`kind`** (pas par chemin) pour borner la cardinalité :
  - `dblease_acquire_total{kind="player|shared_matches|shared_social|metadata"}`
  - `dblease_acquire_timeout_total{kind=...}`
  - `dblease_wait_duration_ms_total{kind=...}` (cumulé) + `dblease_acquire_total` ⇒ moyenne calculée
    côté observabilité (pas d'histogramme expvar).

**API :**
- Constructeurs internes au package, exposés via les abstractions DB du commit 0 :
  ```go
  // internal/platform/dblease/writer.go
  func acquireWriter(path string, kind Kind, timeout time.Duration) (*LeasedWriter, error)
  func acquireWriterCtx(ctx context.Context, path string, kind Kind) (*LeasedWriter, error)
  ```
- Erreur typée `dblease.ErrDBLocked` (sentinel) exportée — consommée par les handlers pour mapper en 503.
- Note documentée : la fairness n'est pas garantie par `TryLock+sleep` — risque de starvation
  sous très forte concurrence à tracer en backlog (post-merge).

**Logging :**
- `slog.DebugContext(ctx, "dblease acquired", "kind", kind, "path", path, "wait_ms", waitMs)` à l'acquisition
- `slog.WarnContext(ctx, "dblease timeout", "kind", kind, "path", path, "timeout", timeout, "err", err)` au timeout
- `slog.DebugContext(ctx, "dblease released", "kind", kind, "path", path, "held_ms", heldMs)` au release

**Tests :** `internal/platform/dblease/writer_test.go`
- Acquisition + release simple, vérifier métriques `acquire_total{kind=player} +1`
- Double-Acquire séquentielle (releaser entre les deux)
- Double-Acquire concurrente : `goroutine A acquiert, goroutine B attend, A release, B acquiert`
  (utiliser `sync.WaitGroup` + canal de signalisation)
- Acquire avec timeout dépassé → `ErrDBLocked` + métrique `acquire_timeout_total{kind} +1`
- AcquireCtx avec ctx annulé → wrap `context.Canceled`
- AcquireCtx avec parent ctx avec deadline dépassé → wrap `context.DeadlineExceeded`
- Acquisition pour 2 paths différents en parallèle → ne se bloquent pas mutuellement
- Test de stress : 100 goroutines acquérant/releasing le même path → toutes finissent,
  pas de deadlock, métriques cohérentes (fairness non testée — documentée comme non garantie)
- `*LeasedWriter` satisfait `port.DBWriter` (compile-time check)
- `*sql.Tx` satisfait `port.DBExecutor` mais pas `port.DBWriter` (compile-time check)

**Tests d'invariants critiques (cf. matrice §Stratégie de tests) :**
- `TestLeasedWriterReleaseIdempotent` — `release()` appelé 2× ne panique pas et le compteur
  ne descend pas sous 0.
- `TestNoLeakOnPanic` — fonction qui acquiert un writer puis panic au milieu → `recover()` dans
  le test, vérifier que `dblease.LeasedWritersInUse() == 0` après la panic.
- `TestNoLeakOnCtxCancel` — goroutine A tient le writer 100 ms, goroutine B fait `AcquireCtx`
  puis cancel le ctx à 50 ms → B retourne `context.Canceled`, n'a pas tenu le writer.
- Property-based `TestAcquireReleaseBalance` (rapid ou table-driven N=100) — pour toute séquence
  aléatoire de N acquisitions/releases sur K paths concurrents, le compteur global retombe à 0
  quand toutes les goroutines terminent.
- Helper `dblease.AssertNoLeasedWriters(t)` exporté pour usage par les autres packages.

**Bench (baseline) :**
- `BenchmarkAcquireRelease_Uncontended` — cible < 1 µs.
- `BenchmarkAcquireRelease_Contended_2` — cible < 100 µs moyen.
- `BenchmarkAcquireRelease_Contended_10` — P99 < 10× P50.
- Profils stockés dans `.ai/baselines/bench_dblease_pre.txt` pour comparaison aux commits suivants.

**Invariants protégés par ce commit :**
- Aucun release manqué, même sur panic ou ctx cancel.
- Compteurs métriques cohérents sous concurrence (vérifié par stress test 100 goroutines).
- Compile-time : `*sql.Tx` ne peut pas être passé là où `port.DBWriter` est attendu.

**Done :** types + interfaces compilent, tests verts, métriques exposées sur `/debug/vars`,
abstractions DB du commit 0 implémentent désormais `AcquireWriter` réellement, baseline bench
figée.

---

### Commit 2 — Migrer `prestige_player_repo` + `prestige.Service` HTTP `[moyen, 1 j]` `[résout P1 HTTP]`

**Fichiers :**
- `internal/platform/duckdb/prestige_player_repo.go` — signatures Write* prennent
  `port.DBExecutor` (interface) au lieu d'ouvrir leur propre `*sql.DB`. Reste un CRUD pur.
- `internal/prestige/service.go` (struct `service`) — méthodes HTTP `CreateChallenge`,
  `UpdateChallenge`, `AbandonChallenge`, `CreateArc` acquièrent `*LeasedWriter` via
  `pdb.AcquireWriterTimeout(dblease.PlayerLeaseTimeout)` au début, defer release. Le `*PlayerDB`
  est résolu via `r.PlayerDB(ctx, gamertag)` (pattern existant).
- `internal/api/handlers/prestige.go` — mapper `errors.Is(err, dblease.ErrDBLocked)` →
  HTTP 503 + `Retry-After: 5`

**Logging :**
```go
// Service — succès en debug, blocage en warn
slog.DebugContext(ctx, "prestige write start", "op", "CreateChallenge", "user", userID)
slog.WarnContext(ctx, "prestige write blocked by lease",
    "err", err, "kind", "player", "op", "CreateChallenge", "user", userID)

// Handler — sur erreur retournée
slog.ErrorContext(ctx, "prestige handler failed",
    "err", err, "endpoint", "POST /challenges", "user", userID)
```

**Tests par couche :**
- `internal/platform/duckdb/prestige_player_repo_test.go` — DuckDB `:memory:` + `*sql.DB` brut OU
  `*sql.Tx` (vérifier que les deux satisfont `port.DBExecutor`). Couvre toutes les méthodes Write*
  migrées, vérifie idempotence INSERT OR REPLACE.
- `internal/prestige/service_test.go` — mock `port.Repository` + `*PlayerDB` réel (DuckDB `:memory:`) :
  - cas nominal : Create/Update/Abandon réussissent
  - cas lease tenu : test acquiert le writer avant l'appel service → vérifier retour `ErrDBLocked`
  - cas lease libéré entre temps : service réussit après attente
  - non-régression : tous les tests existants restent verts (CreateChallenge / UpdateChallenge /
    AbandonChallenge / CreateArc continuent de produire le même comportement métier)
- `internal/api/handlers/prestige_test.go` — `httptest.NewRecorder` + mock `port.PrestigeService` :
  - cas succès : 200 + body JSON
  - cas `ErrDBLocked` : 503 + header `Retry-After: 5` + body JSON erreur typée
  - non-régression : tests existants verts
- Test d'intégration : `internal/prestige/integration_test.go` (build tag `integration`) — DuckDB
  `:memory:`, lance un sync long en goroutine A + appel HTTP service en goroutine B → vérifier
  ordonnancement (B attend A puis succeed, ou B 503 si A trop long)

**Tests d'invariants critiques :**
- `TestPrestigeCreateChallengeIdempotent` (rapid / property-based, N=100) — pour toute
  séquence `Create(req)` exécutée 1× ou 2× avec le même `req.idempotency_key`, l'état final
  des tables `challenges` et `user_prestige` est identique. Sert à blinder INSERT OR REPLACE.
- `TestPrestigeNoLeakOnHandlerPanic` — handler qui panic après acquisition writer → vérifier
  `dblease.AssertNoLeasedWriters(t)` après la requête.
- Helper `prestige.AssertNoLeakedWriters(t)` appelé en `t.Cleanup()` de **chaque test**
  des packages `internal/prestige/...` qui touche un writer.

**Tests de non-régression baseline :**
- Tous les **45 tests** de `service_coverage_test.go` doivent rester verts après adaptation
  des signatures. Les outputs de `EvaluateForUser_*`, `CreateChallenge_*`, etc. doivent être
  bit-à-bit identiques (vérifier par `cmp.Diff` sur les valeurs de retour).
- Tous les **12 tests** de `service_full_test.go` idem.
- Tous les **19 tests** de `baseline_lifecycle_test.go` idem.
- `scripts/check_test_baseline.sh` doit valider que la liste de tests `prestige/*` est
  identique à `main` (ou justifier les renommages mécaniques).

**⚠️ Différé au commit 3 :** la résolution du deadlock sync hook (`EvaluateForUserWithWriter`).
Le commit 2 ne résout que les écritures HTTP synchrones. Le sync hook reste cassé entre commit 2
et commit 3 → ne pas activer Prestige en prod entre les deux.

**Invariants protégés par ce commit :**
- Comportement métier `CreateChallenge` / `UpdateChallenge` / `AbandonChallenge` / `CreateArc`
  inchangé (vérifié par diff bit-à-bit des sorties).
- Aucun chemin HTTP ne tient un writer après réponse (vérifié par `AssertNoLeakedWriters`).
- 503 + `Retry-After: 5` est le seul nouveau comportement HTTP visible.

**Done :** P1 HTTP résolu. Prestige Player désormais inviolable au compile-time côté HTTP.
146 tests Prestige existants restent verts.

---

### Commit 3 — Migrer `prestige_social_repo` + `EvaluateForUserWithWriter` `[moyen, 1 j]` `[résout P1 sync hook + squad/PP]`

**Fichiers :**
- `internal/platform/duckdb/prestige_social_repo.go` — `EmitEvent`, `UpsertUserPrestige`,
  `Create*`, `AddMember`, `RemoveMember`, `AddParticipant` prennent `port.DBExecutor`
- `internal/prestige/service.go` — ajouter `EvaluateForUserWithWriter(ctx, userID, slug, w *LeasedWriter)`
  à l'interface `Service` et à la struct concrète. La version sans writer (`EvaluateForUser`) acquiert
  son propre writer puis délègue ; la version avec writer reçoit celui de l'appelant et passe le
  même `port.DBExecutor` au repo.
- `internal/prestige/sync_hook.go` — `RunPostSyncHook(ctx, svc, userID, slug, w *LeasedWriter)`
  prend désormais le writer en paramètre et appelle `svc.EvaluateForUserWithWriter(ctx, userID, slug, w)`.
- `internal/api/prestige_setup.go:162` — l'appel à `RunPostSyncHook` reçoit le writer du sync engine
  (sera passé via le commit 7 quand le sync engine acquerra son writer en début de pipeline).
  Entre commit 3 et commit 7, le sync engine acquiert et release localement avant l'appel hook
  pour ne pas régresser (le hook acquiert son propre writer dans cet intervalle).

**Logging :**
```go
slog.DebugContext(ctx, "prestige social write start",
    "op", "EmitEvent", "user", userID, "pp", pp)
slog.WarnContext(ctx, "prestige social write blocked by lease",
    "err", err, "kind", "shared_social", "op", "EmitEvent", "user", userID)
slog.ErrorContext(ctx, "prestige social write failed",
    "err", err, "op", "EmitEvent", "user", userID)
```

**Tests par couche :**
- `internal/platform/duckdb/prestige_social_repo_test.go` — DuckDB `:memory:` + `*sql.DB` :
  `EmitEvent` (insert + upsert atomique), création squad, leaderboard, idempotence ON CONFLICT.
- `internal/prestige/service_test.go` — étendre les tests pour couvrir `EvaluateForUserWithWriter` :
  - cas nominal avec writer fourni : aucune nouvelle acquisition (vérifier via mock writer ou
    compteur de leases)
  - cas `EvaluateForUser` (sans writer) : acquisition interne + release en fin
  - lease tenu sur `shared_social` : `EmitEvent` retourne `ErrDBLocked` propagée
- `internal/prestige/sync_hook_test.go` — adapter les tests existants à la nouvelle signature ;
  ajouter un cas où le writer est `nil` (devrait acquérir lui-même, pour permettre les tests legacy
  ou décider de l'interdire — à trancher).
- Non-régression critique : les **7 tests** de `sync_hook_test.go` existants
  (`TestRunPostSyncHook_DisabledSkipsService`, `_EnabledCallsService`, `_NilServiceSurvives`,
  `_ServiceErrorIsLoggedNotPropagated`, etc.) restent verts après adaptation à la nouvelle
  signature avec writer.
- Test d'intégration concurrentiel : sync engine + hook en pipeline, vérifier qu'aucun deadlock
  ne survient (timeout court sur le test pour détecter un blocage).

**Tests d'invariants critiques :**
- `TestSyncHookNoDeadlock` (build tag `integration`) — pipeline complet :
  - sync engine acquiert le writer player puis appelle `RunPostSyncHook(ctx, svc, ..., w)`,
  - le hook propage `w` à `EvaluateForUserWithWriter`,
  - aucune seconde acquisition n'est tentée,
  - timeout court (5 s) → le test échoue par timeout si deadlock.
- `TestEvaluateForUser_AcquiresOwnWriter_HTTPPath` — variante HTTP, le service acquiert et
  release son propre writer (vérifier via `dblease.LeasedWritersInUse()` avant/pendant/après).
- `TestEvaluateForUserWithWriter_DoesNotAcquire_SyncPath` — variante sync, vérifier qu'aucun
  appel à `AcquireWriter` n'est effectué (mock `*PlayerDB` qui compte les appels).
- Property-based `TestEvaluateForUserOutputStable` (N=50 fixtures de matchs aléatoires) —
  les deux variantes (avec/sans writer) produisent les mêmes `[]EvaluationOutcome`.

**Tests de non-régression baseline :**
- Tests `service_evaluate.go` correspondants (`EvaluateForUser_NoActive`, `_TargetReachedCreditsPP`,
  `_DeadlineExpired`, `_FetchErrorReturnsUnchanged`) restent verts.
- Aucune table `prestige_events` / `user_prestige` ne diverge en contenu vs baseline (test
  de fixture : insérer N matchs, comparer dump SQL avant/après).

**Invariants protégés par ce commit :**
- Pas de deadlock sync↔hook (par construction, vérifié par test timeout).
- Mêmes `EvaluationOutcome` produits par les deux variantes (équivalence sémantique).
- Aucun double-write de `user_prestige` (idempotence ON CONFLICT).

**Done :** P1 entièrement résolu (HTTP + sync hook). Toute la classe Prestige est protégée,
pas de deadlock possible. Le squad/PP est également migré.

---

### Commit 4 — Migrer le `notifications.Service` existant `[moyen, 1 j]` `[résout P2]`

**⚠️ Le service existe déjà** (`internal/notifications/service.go`, `port.go`). Ce commit
**l'étend**, ne le crée pas.

**Fichiers :**
- `internal/notifications/port.go` — étendre l'interface `Repository` : les méthodes write
  (`Insert`, `MarkRead`, `MarkUnread`, `MarkAllRead`, `Delete`, `UpsertPreferences`, `CapAndSweep`)
  prennent désormais un `port.DBExecutor` en premier paramètre après `ctx`.
- `internal/notifications/service.go` — `Service` acquiert `*LeasedWriter` via `pdb.AcquireWriter(ctx)`
  au début de chaque méthode write, defer release, puis passe l'executor au repo.
- `internal/platform/duckdb/notifications_repo.go` — les méthodes write reçoivent et utilisent
  le `port.DBExecutor` au lieu de `r.db`.
- Callers à mettre à jour (signatures uniquement, comportement identique) :
  - `internal/sync/post_sync_deltas.go` — passe par le service existant
  - `internal/api/handlers/notifications.go` — déjà via service, juste mapper `ErrDBLocked` → 503
  - `internal/notifications/boot.go` (`EmitAppReleaseForAllPlayers`) — déjà via service
  - `internal/api/registry_notifications.go` — `notifServiceFor` doit recevoir `*PlayerDB` (pour
    `AcquireWriter`), pas seulement `Repository` — vérifier et adapter.

**Contrat public — décision à valider :**
- `Service.Emit()` : maintenir le contrat **best-effort** (lease bloqué → log warn + return nil).
  C'est le comportement attendu par les hooks sync (jamais casser le pipeline pour une notif).
- `Service.MarkRead/Delete/UpsertPreferences` : nouveau comportement → `ErrDBLocked` propagée
  vers le handler qui mappe en 503. Cohérent avec Prestige HTTP.
- `Service.CapAndSweep` : best-effort, log warn + return nil.

**Logging :**
```go
// Best-effort (Emit, CapAndSweep)
slog.DebugContext(ctx, "notification emit", "category", cat, "xuid", xuid)
slog.WarnContext(ctx, "notification dropped by lease timeout",
    "err", err, "kind", "shared_social", "category", cat, "xuid", xuid, "source", source)

// Synchrone (MarkRead, etc.)
slog.WarnContext(ctx, "notification write blocked by lease",
    "err", err, "kind", "shared_social", "op", "MarkRead", "xuid", xuid)
slog.ErrorContext(ctx, "notification write failed",
    "err", err, "op", "MarkRead", "xuid", xuid)
```

**Tests par couche :**
- `internal/platform/duckdb/notifications_repo_test.go` — DuckDB `:memory:` + `*sql.DB` :
  Insert / MarkRead / MarkUnread / Delete / UpsertPreferences / CapAndSweep (purge au-delà de 500).
  Vérifie scope par xuid. Adapter aux nouvelles signatures.
- `internal/notifications/service_test.go` — étendre les tests existants :
  - `Emit()` lease tenu : log warn capturé + retour `nil` (contrat best-effort préservé)
  - `MarkRead()` lease tenu : retour `ErrDBLocked` (nouveau)
  - `CapAndSweep()` lease tenu : silencieux (best-effort)
  - tous les tests `*_test.go` existants doivent rester verts après mise à jour des signatures
- `internal/api/handlers/notifications_test.go` — ajouter cas `ErrDBLocked` → 503 + `Retry-After: 5`
  pour `mark-read`, `delete`, `update-preferences`. Les ~15 tests existants restent verts.
- Non-régression : `internal/sync/post_sync_deltas_test.go` mis à jour, vérifier que les
  notifications de season_pass_level / objective_completed / friend_added / media_added sont
  toujours émises avec les mêmes catégories.
- Non-régression : `EmitAppReleaseForAllPlayers` au boot émet toujours pour chaque joueur une
  seule fois par version.

**Tests d'invariants critiques :**
- `TestNotificationsServiceEmit_LeaseTimeoutSilent` — lease tenu pendant l'`Emit`, vérifier
  retour `nil` + log warn capturé via `slog` test handler (pas d'erreur propagée au caller).
- `TestNotificationsServiceMarkRead_LeaseTimeout` — lease tenu pendant `MarkRead`, vérifier
  `errors.Is(err, dblease.ErrDBLocked)`.
- `TestNotificationsServiceCapAndSweep_BestEffort` — lease tenu pendant `CapAndSweep`, vérifier
  pas d'erreur propagée et purge réessayée au prochain `Emit`.
- Property-based `TestNotificationsEmitMarkReadCycle` (N=50) — pour toute séquence
  `Emit → MarkRead → MarkUnread → Delete` répétée K fois, l'état final est cohérent (pas de
  doublon, ID unique, scope xuid respecté).
- `TestNotifNoLeakAfterBurst` — émission de 100 notifications en parallèle, vérifier
  `dblease.AssertNoLeasedWriters(t)` après la rafale.

**Tests de non-régression baseline :**
- Les **19 tests** de `service_test.go` existants restent verts. Outputs de `Emit`, `List`,
  `UnreadCount`, `MarkRead`, `MarkAllRead`, `Delete`, `Get/UpdatePreferences` strictement
  identiques (diff bit-à-bit).
- Les **15+ tests** de `handlers/notifications_test.go` restent verts. Le seul nouveau cas
  est `ErrDBLocked → 503`, ajouté en plus.
- Comportement `ErrCategoryDisabled` (drop silencieux) **inchangé** : test dédié de
  régression.

**Invariants protégés par ce commit :**
- Contrat best-effort `Emit()` préservé (pas de breaking change pour les callers sync/boot).
- `MarkRead/Delete/UpsertPreferences` peuvent retourner `ErrDBLocked` (extension d'API,
  non-breaking si les callers existants ignorent l'erreur — à valider).
- `CapAndSweep` reste totalement silencieux.

**Done :** P2 résolu. La couche service existante est préservée, son comportement public est
quasi-inchangé (sauf nouveau retour `ErrDBLocked` pour les opérations synchrones).
19 tests service + 15 tests handlers restent verts.

---

### Commit 5 — Migrer `social_repo` (match favorites) `[rapide, 0.5 j]`

**Fichiers :**
- `internal/platform/duckdb/social_repo.go` — `ToggleMatchFavorite` prend `port.DBExecutor`
- `internal/social/service.go` (existant ou à créer si absent) — acquérir `*LeasedWriter` via
  `sharedSocialDB.AcquireWriter(ctx)` avant délégation au repo
- `internal/api/handlers/match_favorite.go` — mapper `ErrDBLocked` → 503 (idem prestige)

**Logging :**
```go
slog.DebugContext(ctx, "favorite toggle", "match_id", matchID, "xuid", xuid, "favorited", fav)
slog.WarnContext(ctx, "favorite write blocked by lease",
    "err", err, "kind", "shared_social", "match_id", matchID, "xuid", xuid)
slog.ErrorContext(ctx, "favorite write failed", "err", err, "match_id", matchID)
```

**Tests par couche :**
- `internal/platform/duckdb/social_repo_test.go` — DuckDB `:memory:` :
  toggle ON, toggle OFF, idempotence, scope par xuid
- `internal/social/service_test.go` — mock repo : nominal, lease tenu, propagation erreur
- `internal/api/handlers/match_favorite_test.go` — `httptest` : succès, `ErrDBLocked` → 503
- Non-régression : tests existants `match_favorite` adaptés à la nouvelle signature, comportement
  métier identique (toggle bidirectionnel, lecture de l'état favori inchangée)

**Done :** match favorites protégés. Comportement utilisateur inchangé.

---

### Commit 6 — Migrer `media_repo` (likes) + transaction atomique `[moyen, 1 j]` `[résout P3]`

**Fichiers :**
- `internal/platform/duckdb/media_repo.go` — `SetMediaLike` et `ToggleSharedLike` prennent
  `port.DBExecutor` (peut donc être appelé avec un `*sql.Tx`).
- Nouvelle méthode service `SetMediaLikeAtomic(ctx, mediaID, xuid, liked)` qui :
  1. Acquiert `*LeasedWriter` via `sharedSocialDB.AcquireWriter(ctx)` (`port.DBWriter`, donc
     a accès à `BeginTx`)
  2. Ouvre une transaction `tx, err := w.BeginTx(ctx, nil)` (tx satisfait `port.DBExecutor`)
  3. Exécute `repo.SetMediaLike(ctx, tx, ...)` puis `repo.ToggleSharedLike(ctx, tx, ...)`
  4. Commit ou rollback en cas d'erreur
- `internal/media/service.go` — orchestrer via `SetMediaLikeAtomic`, plus jamais d'appels
  séparés `SetMediaLike` + `ToggleSharedLike`
- `internal/api/handlers/media.go` — mapper `ErrDBLocked` → 503

**Logging :**
```go
slog.DebugContext(ctx, "media like atomic start",
    "media_id", mediaID, "xuid", xuid, "liked", liked)
slog.WarnContext(ctx, "media like atomic blocked by lease",
    "err", err, "kind", "shared_social", "media_id", mediaID, "xuid", xuid)
slog.ErrorContext(ctx, "media like atomic rollback",
    "err", err, "phase", "ToggleSharedLike", "media_id", mediaID, "xuid", xuid)
```

**Tests par couche :**
- `internal/platform/duckdb/media_repo_test.go` — DuckDB `:memory:` :
  - `SetMediaLikeAtomic` succès → `media_files.liked = true` + ligne dans `media_likes`
  - `SetMediaLikeAtomic` échec sur `ToggleSharedLike` (mock retournant erreur) → rollback
    de `media_files.liked` vérifié (état initial restauré)
  - `SetMediaLikeAtomic` toggle off : suppression de la ligne `media_likes` + `liked = false`
- `internal/media/service_test.go` — mock repo : nominal, lease tenu, rollback observable
- `internal/api/handlers/media_test.go` — `PATCH /media/likes` succès, `ErrDBLocked` → 503,
  rollback transparent côté client
- Non-régression : `MediaRepo.insertMediaFile`, `AssociateMediaWithMatches`, `BackfillThumbnailPaths`
  conservent leur protection `indexMu` (pas migrés vers `LeasedWriter` ici — différent scope :
  protègent ATTACH/DETACH, pas le write). Vérifier que le test d'indexation media existant reste vert.
- Non-régression : `GetMediaLikers` (lecture) reste en accès direct `*sql.DB` (pas de write,
  pas de migration nécessaire)

**Tests d'invariants critiques :**
- `TestMediaLikeRollback` (build tag `integration`) — DuckDB `:memory:` :
  - état initial : `media_files.liked = false`, aucune ligne dans `media_likes`,
  - mock du repo qui fait échouer `ToggleSharedLike` après que `SetMediaLike` ait écrit,
  - appeler `service.SetMediaLikeAtomic` → vérifier `liked = false` après rollback (état initial),
  - vérifier 0 ligne dans `media_likes` (aucune insertion partielle).
- `TestMediaLikeAtomic_PanicMidTx` — provoquer un panic après `SetMediaLike` mais avant
  `ToggleSharedLike` (via mock qui panic) → vérifier rollback DB + writer relâché.
- `TestMediaLikeNoLeakOnLeaseTimeout` — lease tenu, `SetMediaLikeAtomic` → `ErrDBLocked`
  retourné, aucune transaction ouverte, aucun writer fuité.
- Property-based `TestMediaLikeIdempotent` (N=50) — pour toute séquence
  `Like → Unlike → Like → ...` de longueur K, l'état final dépend uniquement de la parité de K.

**Tests de non-régression baseline :**
- Test d'upload media existant (`MediaRepo.insertMediaFile` via `indexMu`) reste vert sans
  modification.
- `media_repo_filters_test.go` reste vert.
- Endpoint `GET /media/:id/likers` (lecture) inchangé.

**Invariants protégés par ce commit :**
- **Atomicité** : `media_files.liked` et `media_likes` sont toujours cohérents (une seule des
  deux ne peut jamais être à jour seule).
- Rollback observable via le test `TestMediaLikeRollback`.
- Aucune fuite de writer ni de transaction sur panic.

**Done :** P3 résolu. Likes désormais atomiques, indexation media inchangée.

---

### Commit 7 — Sync engine adopte `LeasedWriter` + propage au hook Prestige `[lourd, 1.5 j]`

**Fichiers :**
- `internal/sync/engine.go` — `run()` acquiert plusieurs `*LeasedWriter` via
  `pdb.AcquireWriterCtx(ctx)` au début (player, shared matches, metadata, shared social),
  les passe à toutes les méthodes repo en cascade. Remplace les anciens
  `dblease.AcquireLeaseCtx()` standalone.
- `internal/sync/writes.go` — toutes les fonctions write prennent `port.DBExecutor`.
- `internal/sync/career.go`, `internal/sync/citations.go`, `internal/sync/post_sync_deltas.go` — idem.
- `internal/sync/lease.go` — déprécier les façades (laisser un commentaire pointant vers
  `pdb.AcquireWriter*`) ; supprimer après vérification qu'aucun caller ne les utilise plus.
- `internal/api/prestige_setup.go:162` — `RunPostSyncHook` reçoit le `*LeasedWriter` player
  acquis par le sync engine, plus de double acquisition.
- `internal/sync/coordinator.go` — inchangé (le sémaphore parallélisme reste).

**Subtilité :** le sync utilise plusieurs DBs (player + shared matches + metadata + shared social).
Il faut donc 2-4 `*LeasedWriter` selon le pipeline. Acquisition au début (avec ordre de
verrouillage stable pour éviter tout deadlock cross-DB futur), release deferré
à la fin du pipeline (un seul lease par DB pour toute la durée du sync). Si une acquisition
échoue (ctx canceled, shutdown), libérer ceux déjà acquis avant return.

**Ordre de verrouillage recommandé :** `player → shared_matches → shared_social → metadata`
(documenter dans `engine.go`). Tout futur multi-acquireur doit suivre le même ordre.

**Logging :**
```go
slog.InfoContext(ctx, "sync writers acquired",
    "gamertag", gt, "title", title, "duration_acquire_ms", waitMs)
slog.InfoContext(ctx, "sync writers released",
    "gamertag", gt, "title", title, "duration_held_ms", heldMs)
slog.WarnContext(ctx, "sync writer acquire timeout",
    "err", err, "gamertag", gt, "phase", "player_db")
```

**Tests par couche :**
- `internal/sync/engine_test.go` — pipeline sync complet :
  - vérifier acquisition de tous les writers nécessaires en début de `run()`
  - vérifier release deferred dans tous les chemins (succès, erreur, panic)
  - cas écart : si `AcquireWriterCtx` échoue sur le 3ème writer, les 2 premiers sont relâchés
  - cas ctx annulé pendant l'acquisition → propagation propre, aucun lease tenu
  - **deadlock-free :** sync engine + sync hook en pipeline, vérifier que `RunPostSyncHook`
    réutilise le writer player et ne tente jamais une seconde acquisition
- `internal/sync/writes_test.go` — chaque fonction write prend bien `port.DBExecutor`, pas de
  `OpenReadWrite` direct (vérification par grep dans le test ou static analysis)
- Non-régression critique : tous les tests existants du sync engine restent verts. Vérifier
  notamment :
  - `TestRunDelta_*` : sync delta produit le même résultat (idempotence)
  - `TestRunFull_*` : sync full produit le même résultat
  - `TestPostSyncPipeline_*` : pipeline post-sync (career, citations, prestige hook) inchangé
  - Tests d'écriture `match_participants`, `medals_earned`, `highlight_events` verts
- Test concurrence : sync en cours + tentative de write HTTP (Prestige) → HTTP attend ou 503,
  jamais de double-write corrompu

**Tests d'invariants critiques :**
- `TestSyncEngineLockOrder` (build tag `integration`) — deux goroutines acquièrent les writers
  dans le même ordre stable (player → shared_matches → shared_social → metadata). Une goroutine
  malicieuse qui tente l'ordre inverse doit être détectée par un helper de test (compteur de
  séquence d'acquisitions par goroutine, comparé à l'ordre canonique).
- `TestSyncEngineReleaseOnFailure` — si `AcquireWriterCtx` échoue sur le 3ème writer (ex. via
  un mock `*PlayerDB` qui retourne erreur), vérifier que les 2 premiers writers sont libérés
  avant le return (`AssertNoLeasedWriters`).
- `TestSyncEngineReleaseOnPanic` — panic provoquée au milieu du pipeline → tous les writers
  libérés via `defer release()`.
- `TestSyncEngineReleaseOnCtxCancel` — ctx canceled pendant le pipeline → release propre.
- `TestSyncDeltaProducesSameOutput` (build tag `integration`) — fixture identique à un sync
  pre-migration : run sur l'engine migré, comparer dump SQL des tables `match_participants`,
  `medals_earned`, `highlight_events`, `career_progression`, `prestige_events` → identique
  bit-à-bit (utiliser `cmp.Diff` sur les rows ordonnées).
- `TestSyncFullProducesSameOutput` idem pour le sync full.

**Tests de non-régression baseline (le plus critique) :**
- Les **23 tests** de `engine_e2e_test.go` doivent rester verts. Adapter signatures, pas la logique.
- Les **13 tests** de `engine_mock_test.go` idem.
- Les **27 tests** de `backfill_missing_test.go` idem.
- Les **45 tests** de `halo_client_extra_test.go` (purement client, pas de write) restent
  verts sans modification.
- Toute la suite `*_integration_test.go` du package sync (achievements, backfill, career, pve,
  performance, schema) reste verte.
- `scripts/check_test_baseline.sh` doit valider la liste complète des tests sync vs baseline.
- Coverage `internal/sync/` : aucune baisse > 1 % par fichier (vérifier `engine.go`,
  `writes.go`, `career.go`, `citations.go`, `post_sync_deltas.go`).

**Tests de fuites systématiques :**
- Helper `sync.AssertNoLeasedWriters(t)` appelé en `t.Cleanup()` de **chaque test** sync
  qui touche un writer.
- Test de stress `TestSyncBurstNoLeak` — lance 10 syncs séquentiels (mais courts) sur 5
  joueurs différents, vérifier 0 fuite de writer après le burst.

**Invariants protégés par ce commit :**
- **Aucune régression métier** : les tables produites par le sync sont identiques bit-à-bit
  (vérifié par `TestSyncDeltaProducesSameOutput` + `TestSyncFullProducesSameOutput`).
- Ordre de verrouillage stable (vérifié par test).
- Tous les writers libérés sur panic / ctx cancel / erreur d'acquisition.
- Pas de double-acquisition (commit 3 a déjà câblé `EvaluateForUserWithWriter`).

**Done :** sync engine fully aligned. Plus aucun caller n'ouvre `OpenReadWrite` directement
pour des writes. La règle est désormais respectée par construction sur tout le code de prod.
Le deadlock sync↔Prestige est résolu par construction. ~400 tests sync existants restent verts.

---

### Commit 8 — ADR + lint analyzer `[optionnel, différable, 0.5 j]`

**Fichiers :**
- `docs/adr/0013-leased-writer-enforcement.md` (note : 0012 déjà pris par
  `0012-halo-only-adapters-extraction.md`) — décision : "Toute écriture sur player/shared DB
  passe par `*LeasedWriter`. Le `*sql.DB` brut n'est exposé que pour les lectures."
- `docs/FR/adr/0013-leased-writer-enforcement.md` — synchronisation FR (règle CLAUDE.md)
- `tools/lintwriter/` (nouveau) — analyzer Go custom : interdit `db.Exec()` /
  `db.ExecContext()` dans les fonctions du package `internal/platform/duckdb/` si le
  premier paramètre n'est pas `port.DBExecutor` ou `*sql.Tx`
- Activation en CI via `golangci-lint` custom plugin ou commande dédiée

**Done :** la règle est gravée dans l'ADR + impossible à violer en CI. Récidive future
prévenue compile-time + lint-time.

---

### Tests d'intégration & non-régression — vue globale

**Principe :** chaque commit ajoute ses tests, mais la branche entière doit être validée par
une suite de non-régression couvrant les flux end-to-end. Aucun comportement utilisateur ne
doit changer sauf le mapping `ErrDBLocked` → 503 (qui est une amélioration).

**Suite à exécuter en CI sur chaque commit de la branche :**

| Niveau | Suite | Couverture |
|---|---|---|
| Unitaire | `go test ./internal/platform/dblease/...` | Type `LeasedWriter`, métriques par kind, concurrence |
| Unitaire | `go test ./internal/port/...` | Interfaces `DBExecutor` / `DBWriter` (compile-time satisfaction) |
| Unitaire | `go test ./internal/platform/duckdb/...` | Repos avec DuckDB `:memory:` + `*sql.DB` réel |
| Unitaire | `go test ./internal/prestige/... ./internal/notifications/... ./internal/social/... ./internal/media/...` | Services avec mocks `port.Repository` |
| Unitaire | `go test ./internal/api/handlers/...` | Handlers `httptest` + mocks services, mapping erreurs |
| Unitaire | `go test ./internal/sync/...` | Pipeline sync, acquisition/release de writers, deadlock-free vs hook |
| Intégration | `go test -tags=integration ./...` | Scénarios concurrents sync + HTTP |
| Performance | benchmark optionnel : `go test -bench=. ./internal/platform/dblease/` | Latence acquisition lease sous charge |

**Tests d'intégration concurrentiels à ajouter (build tag `integration`) :**

*Concurrence sync ↔ HTTP*
1. **`TestSyncVsPrestigeConcurrent`** — démarre un sync long en goroutine A, lance 10 appels
   `CreateChallenge` HTTP en goroutine B → vérifier qu'aucun ne corrompt la DB, que B attend
   ou reçoit 503 proprement, que les challenges créés sont cohérents.
2. **`TestSyncVsNotificationsConcurrent`** — sync long + 20 `MarkRead` HTTP en parallèle →
   tous succèdent ou retournent `ErrDBLocked`/503, aucune notif corrompue.
3. **`TestSyncVsMediaLikeConcurrent`** — sync long + 10 toggle likes HTTP → cohérence
   `media_files.liked` ↔ `media_likes` en fin de sync.
4. **`TestSyncVsMatchFavoriteConcurrent`** — sync long + 10 toggles favorite HTTP → idem.

*Deadlock-freedom*
5. **`TestSyncHookNoDeadlock`** — sync engine + RunPostSyncHook en pipeline complet, timeout
   court (5 s) → vérifier qu'aucun deadlock ne survient (le test échoue par timeout sinon).
6. **`TestSyncEngineLockOrder`** — deux acquireurs concurrents qui prennent les writers dans
   l'ordre canonique vs ordre inverse → le test détecte et signale la violation d'ordre.

*Atomicité / cohérence*
7. **`TestMediaLikeRollback`** — provoque un échec sur `ToggleSharedLike` au milieu de la
   transaction → vérifier que `media_files.liked` est bien rollback.
8. **`TestMediaLikeAtomic_PanicMidTx`** — panic au milieu de la tx → rollback + writer libéré.
9. **`TestPrestigeEmitEventAtomicity`** — `EmitEvent` insère `prestige_events` + upsert
   `user_prestige` ; mock un échec sur la 2ᵉ étape → vérifier rollback complet.

*Non-régression de sortie sync (ISO bit-à-bit)*
10. **`TestSyncDeltaProducesSameOutput`** — fixture pré-migration vs post-migration, dump SQL
    identique sur toutes les tables touchées.
11. **`TestSyncFullProducesSameOutput`** idem pour sync full.
12. **`TestSyncEngineFullPipeline`** — sync delta complet pour un joueur, vérifier que toutes
    les tables (match_participants, medals_earned, career_progression, etc.) sont peuplées
    identiquement à avant la migration.

*Bursts / charge*
13. **`TestNotificationsBurst`** — émet 50 notifications en parallèle (sync + HTTP + boot) →
    vérifier que toutes sont insérées (ou droppées avec log si lease saturé), aucune corruption.
14. **`TestSyncBurstNoLeak`** — 10 syncs séquentiels courts → 0 fuite de writer après le burst.
15. **`TestPrestigeBurst`** — 50 `CreateChallenge` HTTP en parallèle → tous insérés (ou 503
    propres), aucune fuite, idempotence respectée.

*Crash recovery*
16. **`TestProcessKillNoStaleLock`** — sous-process Go (`os.StartProcess`) qui ouvre une DB
    en write puis exit brutal → vérifier que le process parent peut ré-ouvrir immédiatement
    (mutexes intra-process ne survivent pas, mais valider qu'aucun fichier `.lock` DuckDB
    résiduel ne reste).
17. **`TestExternalProcessLock`** — autre process Go ouvre `stats.duckdb` en RW, le service
    Prestige tente une écriture → `ErrDBLocked` propre, 503 côté HTTP.

**Suite de non-régression métier (à valider manuellement après merge) :**

*Flux principaux*
- [ ] Sync delta sur un joueur réel : durée et nombre de matchs ingérés identiques (±5 %)
- [ ] Sync full sur un joueur réel : nombre de matchs et de médailles identique vs baseline
- [ ] Création/édition/abandon de défi via UI : flow inchangé
- [ ] Page Prestige `/api/v1/prestige/me` : leaderboard et PP cohérents
- [ ] Création d'arc + escouade : flow inchangé
- [ ] Notifications in-app : émission, mark-read, mark-all-read, suppression OK
- [ ] Préférences notifications : update OK, persisté après refresh
- [ ] Like/unlike d'un média : état persistant après refresh, badge cohérent
- [ ] Favoris match : toggle visible immédiatement, persisté
- [ ] Indexation media (settings/reset-index) : aucun changement, `indexMu` toujours actif
- [ ] Endpoints lecture (palmares, season-pass, career) : aucun impact

*Comportements concurrents observables*
- [ ] Pendant un sync long, un `POST /challenges` retourne soit 200 soit 503 (jamais 500)
- [ ] Un 503 reçu : retry après 5s succède (header `Retry-After` respecté côté front)
- [ ] Un upload media pendant un sync : pas de blocage HTTP > 10 s
- [ ] `/debug/vars` expose `dblease_acquire_total{kind=...}` non-nul après quelques opérations

*Comparaison data avant/après*
- [ ] Dump des tables critiques (`match_participants`, `medals_earned`, `prestige_events`,
      `user_prestige`, `notifications`, `media_files`, `media_likes`, `match_favorites`)
      avant et après le déploiement → diff SQL vide ou justifié

---

### Gate d'activation Prestige prod (ADR-0005)

À cocher avant `PRESTIGE_ENABLED=true` en prod (ou avant câblage des routes si Prestige est
déjà actif par défaut) :
- [ ] Commits 1-3 mergés (P1 résolu : Prestige Player + Social sous LeasedWriter, sync hook
      utilise `EvaluateForUserWithWriter`)
- [ ] Tests unitaires + intégration verts en CI (tableau ci-dessus)
- [ ] `TestSyncVsPrestigeConcurrent` ET `TestSyncHookNoDeadlock` verts
- [ ] Test manuel : sync long + POST `/challenges` simultané → 503 propre, pas 500
- [ ] Métriques `dblease_acquire_total{kind}` et `dblease_acquire_timeout_total{kind}` exposées
      et observables sur `/debug/vars`
- [ ] État réel de `PRESTIGE_ENABLED` confirmé en prod (var explicite ou défaut `true` ?)
- [ ] Entrée `thought_log.md` consolidée pour la branche

Note : commits 4-7 ne sont pas bloquants pour l'activation Prestige (ils protègent d'autres
domaines). Ils peuvent être livrés dans la même branche ou différés au sprint suivant.

---

### Done definition globale

*Code & architecture*
- [ ] Tous les commits 0-7 effectués sur `refactor/leased-writer-enforcement`
- [ ] CI verte sur tous les niveaux (unitaire + intégration)
- [ ] Vérifier seuils CLAUDE.md transverses : aucun fichier touché ne dépasse 500 L, aucune
      fonction 80 L, complexité cyclomatique ≤ 12
- [ ] Vérifier qu'aucun `OpenReadWrite()` n'est appelé directement depuis un service ou un
      handler (grep ciblé en CI)
- [ ] Vérifier qu'aucun service ou handler ne manipule de `string path` de DB — passage par
      `*PlayerDB.AcquireWriter()` / équivalents partagés

*Tests — non-régression blindée*
- [ ] **Baseline figée** : `.ai/baselines/tests_pre_migration.jsonl` + `coverage_pre_migration.txt`
      commités au commit 0, immuables ensuite
- [ ] **`scripts/check_test_baseline.sh` vert sur chaque commit** de la branche (en CI)
- [ ] **Aucun test baseline supprimé** sans justification dans le commit message
- [ ] **Coverage par package ne baisse pas** de plus de 1 % (vérifié par le script)
- [ ] **Tous les tests d'invariants** de la matrice §Stratégie de tests sont implémentés et verts
- [ ] **Helpers `AssertNoLeasedWriters(t)`** appelés en `t.Cleanup()` de chaque test qui touche
      un writer (prestige, notifications, social, media, sync) — vérifié par `grep`
- [ ] **17 tests d'intégration** ajoutés (build tag `integration`), tous verts
- [ ] **4 invariants property-based** implémentés (release balance, idempotence Prestige,
      idempotence Notifications, idempotence Media like)
- [ ] **Bench dblease** : aucune régression > 20 % vs `.ai/baselines/bench_dblease_pre.txt`
- [ ] Coverage : aucune méthode write des repos migrés n'est sans test (vérifier via
      `go test -cover ./internal/platform/duckdb/...` — coverage stable ou en hausse)
- [ ] Suite de non-régression métier validée manuellement (tous les points de la liste)

*Observabilité*
- [ ] Logging vérifié : pour chaque write, au moins un `slog.DebugContext` (succès) et un
      `slog.WarnContext` ou `slog.ErrorContext` (échec) avec `ctx` propagé
- [ ] Métriques `expvar` exposées : `dblease_acquire_total`, `dblease_acquire_timeout_total`,
      `dblease_wait_duration_ms_total` par **kind** (pas par chemin individuel)
- [ ] Compteurs `dblease_writers_in_use{kind}` exposés (pour détecter les fuites en prod)

*Documentation*
- [ ] Entrée `thought_log.md` ajoutée pour la branche (règle CLAUDE.md OBLIGATOIRE)
- [ ] Commit 8 (ADR 0013 + lint) : tracker en backlog si différé
- [ ] Section §Stratégie de tests référencée par les futurs PRs touchant des writers DB
      (devient le contrat permanent du projet)

---

## Conformité aux règles du projet

| Règle | Comment le plan la respecte |
|---|---|
| Couches Go (`api/handlers/` ↔ `service/` ↔ `platform/duckdb/`) | Acquisition du writer dans `service/`, jamais dans le repo. Repos restent CRUD purs et prennent `port.DBExecutor`. |
| Pas de `filepath.Join` direct | Aucun service ne manipule de `string path`. L'abstraction `*PlayerDB` connaît son chemin et expose `AcquireWriter()`. Pas de package `paths` créé. |
| Logging `slog.*Context` | Snippets explicites par commit : `Debug` (succès), `Warn` (lease bloqué), `Error` (échec inattendu). Clés standard `err`, `op`, `kind`, identifiants métier. |
| Tests par couche (platform / service / handler / sync / intégration) | Chaque commit liste ses tests par couche. Section globale "Tests d'intégration & non-régression" avec tableau et scénarios concurrentiels. |
| Coverage stable ou en hausse | Vérifié dans Done definition globale via `go test -cover`. |
| `thought_log.md` obligatoire avant commit (CLAUDE.md) | Mentionné dans Done definition globale (entrée consolidée pour la branche). |
| `docs/FR/` synchronisé avec `docs/` | ADR commit 8 créé en EN + FR. |
| Capabilities / feature flags | Gate `PRESTIGE_ENABLED` explicitement géré (commits 1-3 bloquants si Prestige actif). |
| Taille fonctions ≤ 80 L / fichiers ≤ 500 L / complexité ≤ 12 | Vérification dans Done definition globale. |
| Métriques (`expvar`, ADR-0009) | Compteurs `dblease_*{kind}` ajoutés au commit 1 (cardinalité bornée par kind, pas par path), observabilité validée dans la gate Prestige. |
| Stratégie Git : 1 tâche = 1 branche, N commits | ✅ Une seule branche `refactor/leased-writer-enforcement`, 8-9 commits. |
| Pas de `OpenReadWrite()` direct depuis service/handler | Grep ciblé en CI dans Done definition. |

## Références

- `internal/platform/dblease/lease.go` — API dblease existante (timeouts standards :
  `PlayerLeaseTimeout=5s`, `MetadataLeaseTimeout=10s`, `SharedLeaseTimeout=45s`)
- `internal/platform/duckdb/pool.go` — abstraction `*PlayerDB` (à étendre avec `AcquireWriter`)
- `internal/api/registry_notifications.go:49` — pattern de cache `*PlayerDB` → service
- `internal/notifications/service.go` — service existant (cible commit 4 = migration)
- `internal/prestige/sync_hook.go:48` — `RunPostSyncHook` (cible commit 3 = propagation writer)
- `internal/api/prestige_setup.go:162` — appel sync→hook (cible commit 7)
- `internal/assets/write_queue.go` — modèle de référence pour queue async (Option C non retenue)
- `internal/platform/duckdb/db.go` — cache `OpenReadWrite` + pool config (RW=1)
- `internal/prestige/service.go` — cible Phase 1 P1 + ajout `EvaluateForUserWithWriter`
- `docs/adr/0005-prestige-phased-activation.md` — contexte activation Prestige
- `docs/adr/0009-expvar-monitoring-multi-user.md` — contraintes cardinalité métriques
- `docs/adr/0012-halo-only-adapters-extraction.md` — ADR 0012 déjà pris
- ADR à créer : `docs/adr/0013-leased-writer-enforcement.md` + `docs/FR/adr/0013-*.md` (commit 8)
