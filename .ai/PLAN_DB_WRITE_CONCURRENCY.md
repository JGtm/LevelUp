# Plan — Concurrence DB : writes HTTP vs sync

> Créé : 2026-05-04  
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
plutôt qu'un fix court-terme manuel suivi d'une refonte. Justification : `PRESTIGE_ENABLED=false`
en prod retire la pression temporelle, et un fix court-terme sur `prestige.Service` serait
de toute façon jeté lors de la migration `LeasedWriter`. Une seule branche, commits séquentiels,
un seul design.

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
| **A — `LeasedWriter` type** | Méthodes repo prennent un type uniquement constructible via `Acquire()` | Garantie compile-time, ergonomie synchrone | Refactor signatures repo |
| **B — Lease dans le repo** | Chaque méthode repo acquiert le lease elle-même | Service ne sait plus rien | Pas de réentrance `sync.Mutex` → token contexte requis |
| **C — Single-writer goroutine** | Type `WriteQueue` étendu aux player DBs | Élimine le lease par construction | Perd la propagation synchrone des erreurs HTTP |

**Choix retenu :** Option A en cible moyen terme, fix court-terme manuel pour débloquer Prestige prod.

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

- **Masqué aujourd'hui par :** `PRESTIGE_ENABLED=false` en prod (ADR-0005)
- **Devient bloquant à :** activation Phase 2 en prod
- **Résolu par :** commits 2 + 3 du plan d'implémentation (LeasedWriter sur `prestige_player_repo` + `prestige_social_repo`)

### P2 — Notifications multi-sources : contention potentielle `[MODÉRÉ]`

`NotificationsRepo` écrit dans `shared_social.duckdb` sans coordination explicite. Le pool
`sql.DB` (1 connexion) sérialise les `Exec()`, mais plusieurs sources peuvent émettre
simultanément (sync, HTTP, post_sync_deltas, startup). Risque de head-of-line blocking si
une opération longue (CapAndSweep sur 500+ rows) bloque les courtes (MarkRead).

- **Résolu par :** commit 4 du plan (création couche `notifications.Service` + LeasedWriter)

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

---

## Plan d'implémentation — une branche, commits séquentiels

**Branche :** `refactor/leased-writer-enforcement`  
**Stratégie :** viser directement la cible architecturale (Option A `LeasedWriter`),
sans fix court-terme intermédiaire. Chaque commit est reviewable indépendamment et laisse
le code dans un état cohérent (compile + tests verts).

**Effort total estimé :** 3-4 jours.

### Commit 1 — Introduire le type `LeasedWriter` + interface `DBWriter` `[rapide, 3-4 h]`

**Fichiers :**
- `internal/port/dbwriter.go` (nouveau) — interface `port.DBWriter` exposant `ExecContext`,
  `QueryContext`, `QueryRowContext`, `BeginTx`. Permet aux services de prendre l'interface
  en paramètre et d'être mockés dans les tests sans DuckDB réel.
- `internal/platform/dblease/writer.go` (nouveau) — type concret `LeasedWriter` qui implémente
  `port.DBWriter` + wrappe `*sql.DB` + `release()`
- `internal/platform/dblease/metrics.go` (nouveau) — compteurs `expvar` (cohérent avec ADR-0009) :
  `dblease_acquire_total`, `dblease_acquire_timeout_total`, `dblease_wait_duration_ms`
  (par chemin DB)

**API :**
- Deux constructeurs :
  - `AcquireWriter(path, timeout) (*LeasedWriter, error)` — writes HTTP courts
  - `AcquireWriterCtx(ctx, path) (*LeasedWriter, error)` — sync long, piloté par contexte
- Path obtenu via `paths.PlayerDBPath(...)` / `paths.SharedSocialDBPath(...)` côté caller
  (jamais de `filepath.Join` direct — règle `PathResolver`)
- Erreur typée `dblease.ErrDBLocked` exportée (consommée par les handlers pour mapper en 503)

**Logging :**
- `slog.DebugContext(ctx, "dblease acquired", "path", path, "wait_ms", waitMs)` à l'acquisition
- `slog.WarnContext(ctx, "dblease timeout", "path", path, "timeout", timeout, "err", err)` au timeout
- `slog.DebugContext(ctx, "dblease released", "path", path, "held_ms", heldMs)` au release

**Tests :** `internal/platform/dblease/writer_test.go`
- Acquisition + release simple, vérifier métriques `acquire_total +1`
- Double-Acquire séquentielle (releaser entre les deux)
- Double-Acquire concurrente : `goroutine A acquiert, goroutine B attend, A release, B acquiert`
  (utiliser `sync.WaitGroup` + canal de signalisation)
- `AcquireWriter` avec timeout dépassé → `ErrDBLocked` + métrique `acquire_timeout_total +1`
- `AcquireWriterCtx` avec ctx annulé → wrap `context.Canceled`
- `AcquireWriterCtx` avec parent ctx avec deadline dépassé → wrap `context.DeadlineExceeded`
- Acquisition pour 2 paths différents en parallèle → ne se bloquent pas mutuellement
- Test de stress : 100 goroutines acquérant/releasing le même path → toutes finissent,
  pas de deadlock, métriques cohérentes

**Done :** type + interface compilent, tests verts, métriques exposées sur `/debug/vars`,
aucun caller pour l'instant.

---

### Commit 2 — Migrer `prestige_player_repo` + `prestige.Service` + handler `[moyen, 4-6 h]` `[résout P1]`

**Fichiers :**
- `internal/platform/duckdb/prestige_player_repo.go` — signatures Write* prennent
  `port.DBWriter` (interface) au lieu d'ouvrir leur propre `*sql.DB`
- `internal/prestige/service.go` (couche `service/`) — acquérir `*LeasedWriter` via
  `dblease.AcquireWriter(playerDBPath, dblease.PlayerLeaseTimeout)` au début de
  `CreateChallenge`, `UpdateChallenge`, `AbandonChallenge`. Le `playerDBPath` provient de
  `paths.PlayerDBPath(ctxkeys.TitleSlug(ctx), gamertag)`
- `internal/api/handlers/prestige.go` — mapper `errors.Is(err, dblease.ErrDBLocked)` →
  HTTP 503 + `Retry-After: 5`

**Logging :**
```go
// Service — succès en debug, blocage en warn
slog.DebugContext(ctx, "prestige write start", "op", "CreateChallenge", "user", userID)
slog.WarnContext(ctx, "prestige write blocked by lease",
    "err", err, "playerDBPath", playerDBPath, "op", "CreateChallenge", "user", userID)

// Handler — sur erreur retournée
slog.ErrorContext(ctx, "prestige handler failed",
    "err", err, "endpoint", "POST /challenges", "user", userID)
```

**Tests par couche :**
- `internal/platform/duckdb/prestige_player_repo_test.go` — DuckDB `:memory:` + `LeasedWriter`
  réel. Couvre toutes les méthodes Write* migrées, vérifie idempotence INSERT OR REPLACE,
  vérifie qu'une méthode appelée sans `LeasedWriter` ne compile pas (test de régression
  via build tag).
- `internal/prestige/service_test.go` — mock `port.Repository` + mock `port.DBWriter` :
  - cas nominal : Create/Update/Abandon réussissent, vérifie séquence d'appels au repo
  - cas lease tenu : test acquiert le mutex via `dblease.AcquireWriter(path, 1*time.Second)`
    avant l'appel service → vérifier retour `ErrDBLocked`
  - cas lease libéré entre temps : service réussit après attente
  - non-régression : tous les tests existants restent verts (CreateChallenge / UpdateChallenge /
    AbandonChallenge / CreateArc continuent de produire le même comportement métier)
- `internal/api/handlers/prestige_test.go` — `httptest.NewRecorder` + mock `port.PrestigeService` :
  - cas succès : 200 + body JSON
  - cas `ErrDBLocked` : 503 + header `Retry-After: 5` + body JSON erreur typée
  - non-régression : tests existants verts
- Test d'intégration : `internal/prestige/integration_test.go` (build tag `integration`) — DuckDB
  `:memory:`, lance un sync long en goroutine A + appel service en goroutine B → vérifier
  ordonnancement (B attend A puis succeed, ou B 503 si A trop long)

**Done :** P1 critique résolu. Prestige Player désormais inviolable au compile-time.
Tous les tests existants restent verts.

---

### Commit 3 — Migrer `prestige_social_repo` `[moyen, 3-4 h]` `[résout P1 squad/PP]`

**Fichiers :**
- `internal/platform/duckdb/prestige_social_repo.go` — `EmitEvent`, `UpsertUserPrestige`,
  `Create*`, `AddMember`, `RemoveMember`, `AddParticipant` prennent `port.DBWriter`
- `internal/prestige/service.go` — acquérir le lease sur `paths.SharedSocialDBPath()` pour
  les opérations squad / PP

**Logging :**
```go
slog.DebugContext(ctx, "prestige social write start",
    "op", "EmitEvent", "user", userID, "pp", pp)
slog.WarnContext(ctx, "prestige social write blocked by lease",
    "err", err, "sharedSocialPath", path, "op", "EmitEvent", "user", userID)
slog.ErrorContext(ctx, "prestige social write failed",
    "err", err, "op", "EmitEvent", "user", userID)
```

**Tests par couche :**
- `internal/platform/duckdb/prestige_social_repo_test.go` — DuckDB `:memory:` + `LeasedWriter`
  réel : `EmitEvent` (insert + upsert atomique), création squad, leaderboard, idempotence
  ON CONFLICT
- `internal/prestige/service_test.go` — étendre les tests du commit 2 pour couvrir squad/PP :
  cas nominal, lease tenu, lease libéré
- Non-régression : tests existants `prestige_social` verts, leaderboard inchangé

**Done :** Prestige Social également migré. Toute la classe Prestige est désormais protégée.

---

### Commit 4 — Créer `notifications.Service` + migrer `notifications_repo` `[moyen, 4-5 h]` `[résout P2]`

**Fichiers :**
- `internal/notifications/service.go` (nouveau) — couche `service/`, expose `Emit`, `MarkRead`,
  `MarkUnread`, `Delete`, `UpsertPreferences`, `CapAndSweep`. Acquiert `*LeasedWriter` via
  `paths.SharedSocialDBPath()`
- `internal/port/notifications.go` (nouveau) — interface `port.NotificationsService`
- `internal/platform/duckdb/notifications_repo.go` — méthodes write prennent `port.DBWriter`,
  reste un CRUD pur (couche `platform/duckdb/`)
- Callers à migrer vers le service :
  - `internal/sync/post_sync_deltas.go`
  - `internal/api/handlers/notifications.go`
  - `internal/notifications/boot.go` (`EmitAppReleaseForAllPlayers`)

**Comportement par méthode :**
- `Emit()` / `CapAndSweep()` — best-effort : timeout `PlayerLeaseTimeout`, si KO →
  `slog.WarnContext(ctx, "notification dropped by lease timeout", ...)` + `return nil`
- `MarkRead/Unread/Delete/UpsertPreferences` — synchrones HTTP : timeout standard,
  si KO → `dblease.ErrDBLocked` propagée → handler mappe en 503

**Logging :**
```go
// Best-effort (Emit, CapAndSweep)
slog.DebugContext(ctx, "notification emit", "category", cat, "xuid", xuid)
slog.WarnContext(ctx, "notification dropped by lease timeout",
    "err", err, "category", cat, "xuid", xuid, "source", source)

// Synchrone (MarkRead, etc.)
slog.WarnContext(ctx, "notification write blocked by lease",
    "err", err, "op", "MarkRead", "xuid", xuid)
slog.ErrorContext(ctx, "notification write failed",
    "err", err, "op", "MarkRead", "xuid", xuid)
```

**Tests par couche :**
- `internal/platform/duckdb/notifications_repo_test.go` — DuckDB `:memory:` + `LeasedWriter`
  réel : Insert / MarkRead / MarkUnread / Delete / UpsertPreferences / CapAndSweep (purge
  au-delà de 500). Vérifie scope par xuid.
- `internal/notifications/service_test.go` — mock `port.Repository` + mock `port.DBWriter` :
  - `Emit()` cas nominal : appel repo, succès
  - `Emit()` lease tenu : `slog.Warn` capturé + retour `nil` (pas d'erreur propagée)
  - `Emit()` repo retourne erreur : `slog.Error` + retour `nil` (best-effort)
  - `MarkRead()` cas nominal, lease tenu (→ `ErrDBLocked`), repo erreur (→ propagée)
  - `CapAndSweep()` cas nominal, lease tenu silencieux, vérifier purge appelée
- `internal/api/handlers/notifications_test.go` — `httptest` :
  - `PATCH /notifications/mark-read` cas succès
  - `PATCH /notifications/mark-read` avec `ErrDBLocked` → 503 + `Retry-After: 5`
  - `DELETE /notifications/{id}` idem
  - `PATCH /settings/notifications` (UpsertPreferences) idem
- Non-régression : `internal/sync/post_sync_deltas_test.go` mis à jour pour utiliser le service ;
  vérifier que les notifications de season_pass_level / objective_completed / friend_added /
  media_added sont toujours émises avec les mêmes catégories
- Non-régression : `EmitAppReleaseForAllPlayers` au boot émet toujours pour chaque joueur
  une seule fois par version

**Done :** P2 résolu. Couche service introduite, repo reste pur. Tous les callers
(sync, HTTP, boot, deltas) passent par le service.

---

### Commit 5 — Migrer `social_repo` (match favorites) `[rapide, 2-3 h]`

**Fichiers :**
- `internal/platform/duckdb/social_repo.go` — `ToggleMatchFavorite` prend `port.DBWriter`
- `internal/social/service.go` (existant ou à créer) — acquérir `*LeasedWriter` via
  `paths.SharedSocialDBPath()` avant délégation au repo
- `internal/api/handlers/match_favorite.go` — mapper `ErrDBLocked` → 503 (idem prestige)

**Logging :**
```go
slog.DebugContext(ctx, "favorite toggle", "match_id", matchID, "xuid", xuid, "favorited", fav)
slog.WarnContext(ctx, "favorite write blocked by lease",
    "err", err, "match_id", matchID, "xuid", xuid)
slog.ErrorContext(ctx, "favorite write failed", "err", err, "match_id", matchID)
```

**Tests par couche :**
- `internal/platform/duckdb/social_repo_test.go` — DuckDB `:memory:` + `LeasedWriter` :
  toggle ON, toggle OFF, idempotence, scope par xuid
- `internal/social/service_test.go` — mock `port.Repository` : nominal, lease tenu, propagation erreur
- `internal/api/handlers/match_favorite_test.go` — `httptest` : succès, `ErrDBLocked` → 503
- Non-régression : tests existants `match_favorite` adaptés à la nouvelle signature, comportement
  métier identique (toggle bidirectionnel, lecture de l'état favori inchangée)

**Done :** match favorites protégés. Comportement utilisateur inchangé.

---

### Commit 6 — Migrer `media_repo` (likes) + transaction atomique `[moyen, 3-4 h]` `[résout P3]`

**Fichiers :**
- `internal/platform/duckdb/media_repo.go` — `SetMediaLike` et `ToggleSharedLike` prennent
  `port.DBWriter`
- Nouvelle méthode `SetMediaLikeAtomic(ctx, w, ...)` qui ouvre une transaction
  (`w.BeginTx(ctx)`) et exécute les deux writes dans cette transaction → rollback automatique
  si l'un échoue
- `internal/media/service.go` — orchestrer via `SetMediaLikeAtomic`, plus jamais d'appels
  séparés `SetMediaLike` + `ToggleSharedLike`
- `internal/api/handlers/media.go` — mapper `ErrDBLocked` → 503

**Logging :**
```go
slog.DebugContext(ctx, "media like atomic start",
    "media_id", mediaID, "xuid", xuid, "liked", liked)
slog.WarnContext(ctx, "media like atomic blocked by lease",
    "err", err, "media_id", mediaID, "xuid", xuid)
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

**Done :** P3 résolu. Likes désormais atomiques, indexation media inchangée.

---

### Commit 7 — Sync engine adopte `LeasedWriter` `[moyen, 1 j]`

**Fichiers :**
- `internal/sync/engine.go` — `run()` acquiert plusieurs `*LeasedWriter` via
  `AcquireWriterCtx(ctx, ...)` au début (player, shared matches, metadata, shared social),
  les passe à toutes les méthodes repo en cascade. Remplace les anciens
  `dblease.AcquireLeaseCtx()` standalone
- `internal/sync/writes.go` — toutes les fonctions write prennent `port.DBWriter`
- `internal/sync/career.go`, `internal/sync/citations.go`, `internal/sync/post_sync_deltas.go` — idem
- `internal/sync/coordinator.go` — inchangé (le sémaphore parallélisme reste)

**Subtilité :** le sync utilise plusieurs DBs (player + shared matches + metadata + shared social).
Il faut donc 2-4 `*LeasedWriter` selon le pipeline. Acquisition au début, release deferré
à la fin du pipeline (un seul lease par DB pour toute la durée du sync). Si une acquisition
échoue (ctx canceled, shutdown), libérer ceux déjà acquis avant return.

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
- `internal/sync/writes_test.go` — chaque fonction write prend bien `port.DBWriter`, pas de
  `OpenReadWrite` direct (vérification par grep dans le test ou static analysis)
- Non-régression critique : tous les tests existants du sync engine restent verts. Vérifier
  notamment :
  - `TestRunDelta_*` : sync delta produit le même résultat (idempotence)
  - `TestRunFull_*` : sync full produit le même résultat
  - `TestPostSyncPipeline_*` : pipeline post-sync (career, citations, prestige hook) inchangé
  - Tests d'écriture `match_participants`, `medals_earned`, `highlight_events` verts
- Test concurrence : sync en cours + tentative de write HTTP (Prestige) → HTTP attend ou 503,
  jamais de double-write corrompu

**Done :** sync engine fully aligned. Plus aucun caller n'ouvre `OpenReadWrite` directement
pour des writes. La règle est désormais respectée par construction sur tout le code de prod.

---

### Commit 8 — ADR + lint analyzer `[optionnel, différable, 1 j]`

**Fichiers :**
- `docs/adr/0012-leased-writer-enforcement.md` — décision : "Toute écriture sur player/shared DB
  passe par `*LeasedWriter`. Le `*sql.DB` brut n'est exposé que pour les lectures."
- `docs/FR/adr/0012-leased-writer-enforcement.md` — synchronisation FR (règle CLAUDE.md)
- `tools/lintwriter/` (nouveau) — analyzer Go custom : interdit `db.Exec()` /
  `db.ExecContext()` dans les fonctions du package `internal/platform/duckdb/` si le
  receveur n'a pas un `*LeasedWriter` en paramètre
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
| Unitaire | `go test ./internal/platform/dblease/...` | Type `LeasedWriter`, métriques, concurrence |
| Unitaire | `go test ./internal/platform/duckdb/...` | Repos avec DuckDB `:memory:` + `LeasedWriter` réel |
| Unitaire | `go test ./internal/prestige/... ./internal/notifications/... ./internal/social/... ./internal/media/...` | Services avec mocks `port.Repository` + `port.DBWriter` |
| Unitaire | `go test ./internal/api/handlers/...` | Handlers `httptest` + mocks services, mapping erreurs |
| Unitaire | `go test ./internal/sync/...` | Pipeline sync, acquisition/release de writers |
| Intégration | `go test -tags=integration ./...` | Scénarios concurrents sync + HTTP |
| Performance | benchmark optionnel : `go test -bench=. ./internal/platform/dblease/` | Latence acquisition lease sous charge |

**Tests d'intégration concurrentiels à ajouter (build tag `integration`) :**

1. **`TestSyncVsPrestigeConcurrent`** — démarre un sync long en goroutine A, lance 10 appels
   `CreateChallenge` HTTP en goroutine B → vérifier qu'aucun ne corrompt la DB, que B attend
   ou reçoit 503 proprement, que les challenges créés sont cohérents.
2. **`TestNotificationsBurst`** — émet 50 notifications en parallèle (sync + HTTP + boot) →
   vérifier que toutes sont insérées (ou droppées avec log si lease saturé), aucune corruption.
3. **`TestMediaLikeRollback`** — provoque un échec sur `ToggleSharedLike` au milieu de la
   transaction → vérifier que `media_files.liked` est bien rollback.
4. **`TestSyncEngineFullPipeline`** — sync delta complet pour un joueur, vérifier que toutes
   les tables (match_participants, medals_earned, career_progression, etc.) sont peuplées
   identiquement à avant la migration.

**Suite de non-régression métier (à valider manuellement après merge) :**

- [ ] Sync delta sur un joueur réel : durée et nombre de matchs ingérés identiques
- [ ] Création/édition de défi via UI : flow inchangé
- [ ] Page Prestige `/api/v1/prestige/me` : leaderboard et PP cohérents
- [ ] Notifications in-app : émission, mark-read, suppression OK
- [ ] Like/unlike d'un média : état persistant après refresh
- [ ] Favoris match : toggle visible immédiatement, persisté
- [ ] Indexation media (settings/reset-index) : aucun changement, `indexMu` toujours actif
- [ ] Endpoints lecture (palmares, season-pass, career) : aucun impact

---

### Gate d'activation Prestige prod (ADR-0005)

À cocher avant `PRESTIGE_ENABLED=true` en prod :
- [ ] Commits 1-3 mergés (P1 résolu : Prestige Player + Social sous LeasedWriter)
- [ ] Tests unitaires + intégration verts en CI (tableau ci-dessus)
- [ ] `TestSyncVsPrestigeConcurrent` vert
- [ ] Test manuel : sync long + POST `/challenges` simultané → 503 propre, pas 500
- [ ] Métriques `dblease_acquire_total` et `dblease_acquire_timeout_total` exposées et
      observables sur `/debug/vars`
- [ ] Entrée `thought_log.md` consolidée pour la branche

Note : commits 4-7 ne sont pas bloquants pour l'activation Prestige (ils protègent d'autres
domaines). Ils peuvent être livrés dans la même branche ou différés au sprint suivant.

---

### Done definition globale

- [ ] Tous les commits 1-7 effectués sur `refactor/leased-writer-enforcement`
- [ ] CI verte sur tous les niveaux (unitaire + intégration)
- [ ] Suite de non-régression métier validée manuellement (8 points ci-dessus)
- [ ] Coverage : aucune méthode write des repos migrés n'est sans test (vérifier via
      `go test -cover ./internal/platform/duckdb/...` — coverage stable ou en hausse)
- [ ] Logging vérifié : pour chaque write, au moins un `slog.DebugContext` (succès) et un
      `slog.WarnContext` ou `slog.ErrorContext` (échec) avec `ctx` propagé
- [ ] Métriques `expvar` exposées : `dblease_acquire_total`, `dblease_acquire_timeout_total`,
      `dblease_wait_duration_ms` par chemin
- [ ] Vérifier seuils CLAUDE.md : aucun fichier touché ne dépasse 500 L, aucune fonction 80 L,
      complexité cyclomatique ≤ 12
- [ ] Vérifier qu'aucun `OpenReadWrite()` n'est appelé directement depuis un service ou un
      handler (grep ciblé en CI)
- [ ] Entrée `thought_log.md` ajoutée pour la branche (règle CLAUDE.md OBLIGATOIRE)
- [ ] Commit 8 (ADR + lint) : tracker en backlog si différé

---

## Conformité aux règles du projet

| Règle | Comment le plan la respecte |
|---|---|
| Couches Go (`api/handlers/` ↔ `service/` ↔ `platform/duckdb/`) | Acquisition du lease dans `service/`, jamais dans le repo. Repos restent CRUD purs et prennent `port.DBWriter`. |
| `PathResolver` (jamais de `filepath.Join`) | Tous les paths DB passent par `paths.PlayerDBPath()` / `paths.SharedSocialDBPath()`. |
| Logging `slog.*Context` | Snippets explicites par commit : `Debug` (succès), `Warn` (lease bloqué), `Error` (échec inattendu). Clés standard `err`, `op`, identifiants métier. |
| Tests par couche (platform / service / handler / sync / intégration) | Chaque commit liste ses tests par couche. Section globale "Tests d'intégration & non-régression" avec tableau et scénarios concurrentiels. |
| Coverage stable ou en hausse | Vérifié dans Done definition globale via `go test -cover`. |
| `thought_log.md` obligatoire avant commit (CLAUDE.md) | Mentionné dans Done definition globale (entrée consolidée pour la branche). |
| `docs/FR/` synchronisé avec `docs/` | ADR commit 8 créé en EN + FR. |
| Capabilities / feature flags | Gate `PRESTIGE_ENABLED` explicitement géré (commits 1-3 bloquants). |
| Taille fonctions ≤ 80 L / fichiers ≤ 500 L / complexité ≤ 12 | Vérification dans Done definition globale. |
| Métriques (`expvar`, ADR-0009) | Compteurs `dblease_*` ajoutés au commit 1, observabilité validée dans la gate Prestige. |
| Stratégie Git : 1 tâche = 1 branche, N commits | ✅ Une seule branche `refactor/leased-writer-enforcement`, 7-8 commits. |
| Pas de `OpenReadWrite()` direct depuis service/handler | Grep ciblé en CI dans Done definition. |

## Références

- `internal/platform/dblease/lease.go` — API dblease existante (timeouts standards : `PlayerLeaseTimeout=5s`, `MetadataLeaseTimeout=10s`, `SharedLeaseTimeout=45s`)
- `internal/assets/write_queue.go` — modèle de référence pour queue async (Option C non retenue)
- `internal/platform/duckdb/db.go` — cache `OpenReadWrite` + pool config (RW=1)
- `internal/prestige/service.go` — cible Phase 1 P1
- `internal/notifications/` (à créer si absent) — cible Phase 1 P2 (couche service)
- `internal/platform/duckdb/notifications_repo.go` — reste CRUD pur, pas de lease ici
- `docs/adr/0005-prestige-phased-activation.md` — contexte activation Prestige
- ADR à créer : `docs/adr/0012-leased-writer-enforcement.md` + `docs/FR/adr/0012-*.md` (Phase 3)
