# Handoff — Audit ouvertures DuckDB & fix shared_social

**Date** : 2026-05-20
**Auteur** : Claude (session continuée post-commit 20 ConsoleHandler)
**Branche d'origine du fix** : `fix/auto-sync-different-configuration`
**Branche courante au moment du fix** : `fix/csr-placement-alignment` (mélange involontaire)
**Statut global** : Code écrit ✅ — Tests runtime non exécutés ❌ — Non-commit (décision user)

---

## 1. Contexte

Suite à l'élimination du bug *"Can't open a connection to same database file with a different configuration than existing connections"* sur `shared_matches_v2.duckdb` via `SharedDBProvider` B-swap (ADR 0016, commits 1-9), l'utilisateur a demandé :

> "Et est-ce qu'on n'a pas oublié metadata et shared_social en BDD avec ce mécanisme ?"

Puis a précisé :
- Vérifier la sécurité sous écritures concurrentes (likes, favoris, ascension simultanés).
- Comprendre pourquoi des erreurs *"can't open connection while..."* ont été observées sur metadata sur un autre PC.
- Étendre l'audit aux player DBs (`data/players/{gamertag}/stats.duckdb`).

---

## 2. Découverte clé — Cache `openCachedDB`

Au cœur de [apps/go-api/internal/platform/duckdb/db.go](apps/go-api/internal/platform/duckdb/db.go) :

```go
// OpenReadOnly      → cache key = "ro:" + path  (access_mode=read_only)
// OpenReadWrite     → cache key = "rw:" + path  (maxOpenConns=1)
// OpenReadWriteShared → cache key = "rw:" + path  (maxOpenConns=4)
```

**`OpenReadWrite` et `OpenReadWriteShared` partagent la MÊME clé cache**. Le 2e appel sur le même path récupère le `*DB` cached du 1er (Ping OK → refcount++), même si les `maxOpenConns` demandés diffèrent. **Ce n'est pas une cause de crash "different configuration"**.

**Vraie cause du crash** : `OpenReadOnly` (`"ro:"+path`) + n'importe quel `OpenReadWrite*` (`"rw:"+path`) sur le même fichier dans le même process → 2 vrais `duckdb_open()` avec des `access_mode` différents → DuckDB refuse.

---

## 3. Verdict d'audit

| BDD | Risque "different configuration" | Raison |
|---|---|---|
| `shared_matches_v2.duckdb` | ✅ Aucun | `SharedDBProvider` B-swap (RO↔RW dynamique). Cf. ADR 0016. |
| `metadata.duckdb` | ✅ Aucun | 100% `OpenReadWriteShared` au runtime serveur ([main.go:339](apps/go-api/cmd/server/main.go#L339), [pool.go:217](apps/go-api/internal/platform/duckdb/pool.go#L217), [server.go:285](apps/go-api/internal/api/server.go#L285), [prestige_setup.go:69](apps/go-api/internal/api/prestige_setup.go#L69), [lab/provider.go:59](apps/go-api/internal/platform/lab/provider.go#L59)). Aucun `OpenReadOnly`. |
| Player `stats.duckdb` | ✅ Aucun | Mix `OpenReadWrite` ([pool.go:193](apps/go-api/internal/platform/duckdb/pool.go#L193), [engine.go:168](apps/go-api/internal/sync/engine.go#L168)) + `OpenReadWriteShared` ([main.go:664](apps/go-api/cmd/server/main.go#L664) onRotated, `engine_acquire.go:118` AcquirePlayerWriterStandalone) → MÊME clé cache `"rw:"+path` → cache merge transparent. |
| `shared_social.duckdb` | ⚠️ Présent **avant ce fix** | `OpenReadOnly` ([registry_notifications.go:121](apps/go-api/internal/api/registry_notifications.go#L121)) + `OpenReadWrite`/`OpenReadWriteShared` ailleurs → clés cache différentes → 2 `duckdb_open()` → crash sur collision media upload + post-sync. |

### Hypothèse pour les erreurs metadata vues sur l'autre PC user

Collision **inter-process** avec un CLI tool (`cmd/refresh-metadata`, `cmd/seed-weapon-labels`, `cmd/migrate-static-maps`, `cmd/levelup/cmd_sync_achievements`) lancé pendant que le serveur tournait. DuckDB ne supporte pas le partage file-lock entre processus distincts. **Pas un bug serveur** — un problème ops.

**Mitigation recommandée** : runbook ops disant "ne pas lancer ces CLI pendant que le serveur est actif", ou wrapper ces commandes dans un check `lsof`/équivalent.

---

## 4. Changements code (fix shared_social)

5 fichiers modifiés, **non commités** sur la branche courante `fix/csr-placement-alignment` (qui contient déjà du travail career/home en cours de l'utilisateur — autre sujet).

### 4.1 `internal/api/registry_notifications.go` (CRITIQUE)

`loadRecentMediaMatchIDs` : `OpenReadOnly` → `OpenReadWriteShared` (ligne 121).

→ Élimine la seule clé cache `"ro:"+sharedSocialPath` du process serveur. Plus de mix RO/RW possible sur shared_social.

### 4.2 `internal/api/post_sync_deltas.go`

`upsertPlayerRecord` (ligne 624) : drop le `OpenReadWrite + defer Close` redondant, utilise `pdb.SharedSocial.Exec` direct.

→ Le fichier mélangeait `pdb.SharedSocial.QueryRow` (ligne 607, lecture) et `OpenReadWrite` séparé (écriture) sur le même DB — incohérence interne corrigée.

### 4.3 `internal/platform/duckdb/pool.go`

`openPlayerDB` (ligne 230) : `OpenReadWrite(cfg.SharedSocialDBPath)` → `OpenReadWriteShared`.

→ Intent explicite : lectures concurrentes HTTP + écritures occasionnelles (favoris/likes/ascension/post-sync records). Commentaire bloc mis à jour pour documenter le contrat partagé avec les autres call sites.

### 4.4 `internal/platform/duckdb/notifications_repo_test.go`

`TestPlayerRecords_UpsertAndLoad` (ligne 336) : fixture alignée — utilise `pdb.SharedSocial.Exec` direct au lieu de ré-ouvrir un `*DB` séparé. Empêche un futur copy-paste du pattern obsolète dans le code prod.

### 4.5 `.ai/thought_log.md`

Entrée *"Commit 21 : éliminer le mix RO/RW sur shared_social"* ajoutée en tête.

---

## 5. Sécurité sous écritures concurrentes

Question explicite user : plusieurs utilisateurs likent/favorise/ascension en même temps, on est safe ?

**Réponse** : oui sans rien d'autre à faire. DuckDB sérialise les writes via MVCC (single writer commit lock au niveau DB). Pour des **micro-INSERTs 1 ligne** — ce que fait shared_social (player_records, favorites, likes, ascension completions) — la sérialisation est ms-scale, imperceptible.

L'infra `dblease.KindSharedSocial` + `pdb.AcquireSharedSocialWriterTimeout` existe déjà ([pool_writers.go:40-52](apps/go-api/internal/platform/duckdb/pool_writers.go#L40-L52)) pour les sites qui veulent une arbitration applicative explicite — utilisée par :

- [registry_notifications.go:63](apps/go-api/internal/api/registry_notifications.go#L63)
- [registry.go:515](apps/go-api/internal/api/registry.go#L515), [registry.go:585](apps/go-api/internal/api/registry.go#L585)
- [prestige_lazy_service.go:124](apps/go-api/internal/api/prestige_lazy_service.go#L124)

**Pas besoin** d'un Provider B-swap pour shared_social — pattern trop lourd pour des micro-écritures non bloquantes.

---

## 6. Ce qui A été testé ✅

- **`go vet ./...`** : clean sur tout le module. Syntaxe + types corrects.
- **IDE diagnostics** : aucune erreur après tous les edits (1 warning intermédiaire `undefined: err` résolu en passant à `if err := ... ; err != nil`).
- **Audits Explore subagent** : 2 audits parallèles (metadata + player DBs) avec lecture des call sites, vérifiés manuellement contre le code réel.

---

## 7. Ce qui N'A PAS été testé ❌

**Aucun test runtime n'a tourné** sur les packages modifiés. Cause : link CGO DuckDB cassé sur ce poste (MSYS2 GCC 16 incompatible `libduckdb_static`, erreurs `undefined reference to __stdio_common_vsnprintf_s` + `seekpos`).

Concrètement, non-vérifié :
- [ ] Build serveur runtime
- [ ] `TestPlayerRecords_UpsertAndLoad` (test que j'ai modifié)
- [ ] Suite `internal/api/...` (notamment notification flow + post-sync deltas)
- [ ] Suite `internal/platform/duckdb/...` (pool tests, integration tests)
- [ ] Dry-run boot (vérifier que le pool ouvre shared_social en RWShared sans warning)
- [ ] Scénario manuel : upload media simultané à un end-of-sync (le scénario qui aurait crashé avant le fix)

**Risque résiduel** : le refactor `upsertPlayerRecord` (utilise `pdb.SharedSocial.Exec` directement) est très simple et hautement probable de fonctionner. Le changement de mode dans `pool.go` repose sur la sémantique cache du `*DB` qui est claire à la lecture du code. Mais "probable" ≠ "vérifié".

---

## 8. État des commits

### Commit 20 — `feat(observability): ConsoleHandler compact + skip verbose attrs + truncation`

- **SHA** : `8f928ee7`
- **Branche** : `fix/auto-sync-different-configuration` (52+1 commits ahead of origin)
- **Statut** : ✅ Commité. Pas pushé.
- **Tests** : 33 tests passés (`internal/observability/...`) + preview visuel confirmé.
- **Fichiers** :
  - `apps/go-api/internal/observability/logging/console_handler.go` (new)
  - `apps/go-api/internal/observability/logging/console_handler_test.go` (new)
  - `apps/go-api/internal/observability/logging/preview_visual_test.go` (new)
  - `apps/go-api/internal/observability/logging/config.go` (modified)
  - `apps/go-api/internal/observability/logging/README.md` (modified)
  - `apps/go-api/cmd/server/main.go` (modified)
  - `.ai/thought_log.md` (modified)

### "Commit 21" — `fix(db): éliminer le mix RO/RW sur shared_social.duckdb`

- **SHA** : n/a (non commité)
- **Branche** : aucune choisie — l'utilisateur a dit "on commit pas pour le moment"
- **Statut** : ⏸️ En attente de décision utilisateur (cf. § 9)
- **Tests** : non exécutés (CGO link cassé localement)
- **Fichiers modifiés** (working tree current branch `fix/csr-placement-alignment`) :
  - `apps/go-api/internal/api/registry_notifications.go`
  - `apps/go-api/internal/api/post_sync_deltas.go`
  - `apps/go-api/internal/platform/duckdb/pool.go`
  - `apps/go-api/internal/platform/duckdb/notifications_repo_test.go`
  - `.ai/thought_log.md`

⚠️ La branche courante `fix/csr-placement-alignment` contient **aussi** du travail non-commit de l'utilisateur (career_repo, home_repo_playlist_ranks, CareerRankingBlock, HomeRecentPlaylistsCard) qui n'est pas le mien.

---

## 9. À faire

### Côté utilisateur (décisions)

1. **Décider la stratégie commit pour le fix shared_social** :
   - Option A : stash le travail career/home, switch sur `fix/auto-sync-different-configuration`, commit le fix là (cohérent thématiquement avec le sprint), revenir sur `fix/csr-placement-alignment` et restore.
   - Option B : créer une branche dédiée `fix/shared-social-rwshared` depuis `main`, commit isolé (PR indépendant — recommandé si le sprint auto-sync va être PRed séparément).
   - Option C : commit sur la branche courante (viole CLAUDE.md "1 tâche = 1 branche").

2. **Pusher commit 20** quand prêt : `git push origin fix/auto-sync-different-configuration` (52+1 commits à pousser).

### Côté validation (avant merge)

3. **Sur un poste avec CGO fonctionnel** (autre PC user, CI Linux, ou WSL) :
   ```bash
   cd apps/go-api
   go test -count=1 ./internal/api/... ./internal/platform/duckdb/...
   ```
   En particulier vérifier que `TestPlayerRecords_UpsertAndLoad` passe avec le nouveau pattern `pdb.SharedSocial.Exec`.

4. **Dry-run serveur sous trafic simulé** :
   - Boot serveur → vérifier dans `logs/duckdb.log` que shared_social ouvre en RWShared (commentaire bloc updated).
   - Lancer simultanément un upload media (`POST /api/v1/players/{slug}/media/upload`) + un end-of-sync (`/api/v1/players/{slug}/sync`).
   - Avant fix : crash `"different configuration"` attendu.
   - Après fix : pas d'erreur, les 2 ops réussissent.

5. **Stress concurrent likes/favoris** (validation de la sérialisation MVCC) :
   - 10 goroutines POST `/api/v1/media/{id}/like` en parallèle → vérifier que tous les likes sont enregistrés, pas d'erreur DuckDB.

### Côté ops (non bloquant, mais recommandé)

6. **Documenter dans le runbook** que les CLI tools modifient des DBs partagées :
   - `cmd/refresh-metadata`
   - `cmd/seed-weapon-labels`, `cmd/seed-rank-translations`, `cmd/seed-assists-model`
   - `cmd/migrate-static-maps`, `cmd/migrate-to-shared-social`
   - `cmd/populate-career-rank-images`, `cmd/populate-assets`
   - `cmd/refresh-career-ranks`
   - `cmd/backfill_*`
   - `cmd/levelup/cmd_sync_achievements`
   - `cmd/levelup/cmd_backfill`

   → ne pas lancer pendant que le serveur tourne (collision file-lock OS). Si besoin, soit arrêter le serveur, soit ajouter un check `lsof` au début de ces commandes.

### Côté toolchain (point bonus)

7. **Fixer le link CGO MSYS2 sur ce poste** (chronophage, indirect au sujet). Symptômes :
   - `__stdio_common_vsnprintf_s` undefined → MSVC runtime mismatch
   - `std::basic_streambuf::seekpos` undefined → libstdc++ trop récent (GCC 16) vs libduckdb_static compilée avec libstdc++ plus ancien.

   Hypothèses : downgrade GCC vers 13.x, ou recompiler libduckdb_static avec le toolchain courant, ou utiliser un autre toolchain (Zig CC ? TDM-GCC ?).

---

## 10. Documents liés

- [ADR 0016 — SharedDBProvider B-swap](docs/adr/0016-shared-db-provider-b-swap.md) — le fix d'origine pour shared_matches_v2
- [.ai/thought_log.md](.ai/thought_log.md) — entrées commit 20 + commit 21 (en attente)
- [apps/go-api/internal/observability/logging/README.md](apps/go-api/internal/observability/logging/README.md) — doc logging commit 20

---

## 11. TL;DR pour la prochaine session

Si tu reprends cette tâche dans une nouvelle session :

1. Lis ce fichier en entier.
2. Vérifie l'état git : `git status`. Si les 5 fichiers de § 4 sont toujours modifiés non commités, demande à l'utilisateur la décision § 9.1.
3. Si commit fait : passe directement à § 9.3 (validation runtime sur autre poste).
4. Si questions sur l'audit : tout est en § 3 + § 5.
5. Le fix est **fonctionnellement complet et cohérent** — le seul blocage est la validation runtime + la décision de stratégie git.
