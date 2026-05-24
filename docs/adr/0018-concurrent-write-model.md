# ADR 0018 — Concurrent Write Model pour shared.match_participants (singleflight)

**Date** : 2026-05-23
**Status** : ✅ **CLOSED (2026-05-24)** — la singleflight a été SUPPRIMÉE de `internal/sync/writes.go::InsertParticipants` le 2026-05-24 (Phase 5 cleanup), suite à validation empirique de Phase 4 batch INSERT-only (16 syncs / 0 FATAL). Superseded par ADR 0019 + Phase 4 qui résolvent le bug ART par construction (path INSERT-only sur shared + batch INSERT-only sur post-sync compute).
**Branch** : `chore/post-stabilisation-debt`
**Related** : ADR 0013 (LeasedWriter), ADR 0016 (B-swap RO↔RW), **ADR 0019 (Collect→Persist, qui rend ce pattern obsolète)**
**Plan** : `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md` Phase 1

## Context

Crash `duckdb::FatalException` observé en prod le 2026-05-22 :

```
INTERNAL Error: Failed to append to PRIMARY_match_participants_match_id_xuid:
Constraint Error: PRIMARY KEY or UNIQUE constraint violation:
duplicate key 0941d737-1fb4-4a11-8a9a-169624911729, 2533274828226170
```

Cause racine identifiée (cf. audit `HANDOFF_SYNC_CONCURRENCY_AUDIT.md` §2) : **corruption silencieuse de l'index ART** (Adaptive Radix Tree) de DuckDB sur `shared.match_participants(match_id, xuid)`, déclenchée par des UPSERTs concurrents sur la même clé via `INSERT ... ON CONFLICT DO UPDATE`.

Issues DuckDB upstream pertinentes :
- [#18782](https://github.com/duckdb/duckdb/issues/18782) — ART + INSERT-in-txn comportement incorrect (OPEN août 2025)
- [#16520](https://github.com/duckdb/duckdb/issues/16520) — Duplicate key during data insert
- Changelog v1.4.1 : *"ART index could omit rows non-deterministically when running on multiple threads"*

ART corruption observée chiffrée : ~2% des matchs (`indexed != scan`), ~32 matchs sur 1610 dans la prod actuelle. Données silencieusement perdues (ex : LUSR de Madina figé Argent IV au lieu de Platine).

### Cartographie des writers sur `shared.match_participants`

| # | Caller | Fichier:Ligne | Op | Concurrence interne | Protection actuelle |
|---|---|---|---|---|---|
| 1 | Sync engine pagination | [`sync/engine_fetch.go:141`](../../apps/go-api/internal/sync/engine_fetch.go) | UPSERT via `InsertParticipants` | Séquentiel (Phase 3 du run après errgroup) | `dblease.AcquireWriter` |
| 2 | Sync engine processMatch (legacy) | [`sync/engine_process_match.go:108`](../../apps/go-api/internal/sync/engine_process_match.go) | UPSERT | Séquentiel | `dblease.AcquireWriter` |
| 3 | Heal skill | [`sync/skill_heal.go:116`](../../apps/go-api/internal/sync/skill_heal.go) | UPSERT | **errgroup 8 goroutines** | `dblease.AcquireWriter` (mais writes concurrents intra-lease) |
| 4 | Heal stats | [`sync/stats_heal.go:94`](../../apps/go-api/internal/sync/stats_heal.go) | UPSERT | **errgroup 8 goroutines** | `dblease.AcquireWriter` (mais writes concurrents intra-lease) |
| 5 | Import OpenSpartan | [`service/openspartan_import_service.go:313`](../../apps/go-api/internal/service/openspartan_import_service.go) | UPSERT | One-shot CLI | `dblease.AcquireWriter` |
| 6 | `MarkSkillLoaded` | [`sync/writes.go:547`](../../apps/go-api/internal/sync/writes.go) | UPDATE bitmask | Appelé séquentiellement | `dblease.AcquireWriter` |
| 7 | Migration `fix_bot_xuid` | [`migration/steps_shared.go:355`](../../apps/go-api/internal/migration/steps_shared.go) | UPDATE | Once-only au boot | Sentinel migration |
| 8 | CLI `reset_bitmasks` | [`cmd/levelup/cmd_reset_bitmasks.go:127`](../../apps/go-api/cmd/levelup/cmd_reset_bitmasks.go) | UPDATE | Manuel one-shot | N/A (admin) |

**Sources de concurrence problématiques** :

- **Intra-cycle** : `skill_heal` et `stats_heal` lancent 8 goroutines parallèles qui UPSERT chacune un match différent. Pour les heals filtrés par `WHERE xuid = ?`, deux goroutines du MÊME heal ne touchent pas la même `(match_id, xuid)` — mais l'index ART est PARTAGÉ entre toutes les clés et le bug DuckDB se manifeste même sur clés différentes (cf. handoff §2 risque 1).
- **Inter-joueurs** (futur Phase 3.4) : si le scheduler `RunOnce` est parallélisé sur 3 joueurs simultanément, Madina écrit `(matchA, MadinaXUID)` et Choco écrit `(matchA, ChocoXUID)` sur la même table en parallèle. Si matchA est partagé, c'est encore plus probable (3 syncs concurrents sur les mêmes match_ids).

Le `dblease.AcquireWriter` sérialise au niveau **process** (1 lease global par path DB) mais ne sérialise PAS les goroutines à l'intérieur d'un même lease. Les errgroups 8 goroutines tournent sous UN seul lease et hammer le même index ART.

## Decision

Adopter le pattern **singleflight par clé naturelle** pour tous les writes sur `shared.match_participants` :

```go
import "golang.org/x/sync/singleflight"

var participantsSF singleflight.Group

// Avant chaque UPSERT/UPDATE, déduplicer par (match_id, xuid).
func InsertParticipantsSafe(ctx context.Context, db *sql.DB, rows []ParticipantRow) error {
    for _, row := range rows {
        key := row.MatchID + "|" + row.XUID
        _, err, _ := participantsSF.Do(key, func() (any, error) {
            return nil, insertSingleParticipant(ctx, db, row)
        })
        if err != nil {
            return err
        }
    }
    return nil
}
```

### Sémantique

- Deux goroutines qui appellent `InsertParticipantsSafe` avec la même `(match_id, xuid)` au même moment : **un seul** exécute le statement SQL ; les autres attendent et reçoivent son résultat.
- Sur des clés différentes : aucune sérialisation (parallèle, normal).
- Coût : un lookup map mémoire + une attente conditionnelle. **Négligeable** vs un RTT API ou un INSERT DuckDB.

### Périmètre

Le singleflight s'applique à :
- `match_participants` PK `(match_id, xuid)` ← cible principale (ce ADR)

Le singleflight ne s'applique PAS aux tables append-only (INSERT OR IGNORE / DELETE+INSERT par clé naturelle) :
- `shared.match_registry` → `INSERT IF NOT EXISTS` natif, déjà sûr
- `shared.medals_earned` → `INSERT OR IGNORE`, déjà sûr
- `shared.weapon_kills` → append-only sans PK, déjà sûr
- `shared.highlight_events` → append-only sans PK, déjà sûr

### Ce qui RESTE protégé par `dblease.AcquireWriter`

Le singleflight est ADDITIONNEL au lease, pas un remplacement :
- `dblease` garantit qu'un seul process (et un seul lease writer) tape la DB à un instant T.
- `singleflight` garantit qu'une seule goroutine intra-process tape une `(match_id, xuid)` à un instant T.

Les deux ensembles protègent contre :
1. Multi-process concurrent writes (via lease)
2. Multi-goroutine concurrent UPSERT sur même clé (via singleflight)

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| `sync.Mutex` global sur toutes les writes shared | Trop coarse, tue la parallélisation des autres tables (medals append-only, weapon_kills append-only) |
| Transaction `BEGIN ... COMMIT` autour de chaque UPSERT | Ne résout rien : DuckDB sérialise déjà les commits, le bug est dans l'ART pas dans le commit log |
| Retry automatique sur `Constraint Error` | Ne marche pas : le crash est `FatalException` C++ non-récupérable depuis Go |
| Batch INSERT `INSERT ... VALUES (...), (...), ...` | Issue DuckDB #8147 confirme conflits intra-statement non supportés — aggrave |
| Sharding par `match_id` (1 worker par shard) | Équivalent fonctionnel au singleflight mais plus lourd à implémenter, perd la flexibilité |
| `healParallelism = 1` (revert Action B) | Régression perf — sync Madina passerait de 8 min à ~30 min |

## Consequences

### Positives

- Plus de race ART au niveau applicatif (élimine la cause racine du crash `FatalException`).
- Compatible avec la parallélisation future du scheduler (Phase 3.4 du plan) : 3 joueurs simultanés peuvent UPSERT sans risque sur match_participants.
- Pattern réutilisable pour d'autres tables PK-indexées si besoin futur.
- Coût mémoire et CPU négligeable.

### Négatives / limites

- **Ne résout pas le bug DuckDB upstream**. Une corruption ART peut encore survenir si :
  - Un UPSERT solitaire arrive sur une PK déjà corrompue (Phase 4.1 ART rebuild requise pour les matchs déjà corrompus).
  - Le bug ART touche aussi les writes inter-clés (issue #18782 suggère que oui dans certains cas).
- **Multi-process** : si jamais deux instances du serveur écrivent dans la même DB (rare, mais possible si un cron parallèle est lancé), singleflight n'aide pas. Le lease DuckDB (ADR 0013) couvre ce cas.
- **Sémantique singleflight = dedupe, pas serialize** : si 5 callers demandent la même clé en même temps, 1 seul exécute et les 4 autres reçoivent son résultat (y compris son `error`). Acceptable car les inputs sont équivalents (même row à UPSERT).

### Risques résiduels après application

Cf. handoff §2 risque #1-7 :
1. Le bug DuckDB persiste sur les writes inter-clés.
2. Une corruption peut survenir hors writes (DELETE+reinsert intra-txn — issue #16520).
3. `SIGABRT` handler (Phase 4.2) peut arriver trop tard pour catcher le crash C++.
4. Rebuild swap-table (Phase 4.1) prend un lock writer ~1s sur 50k rows.
5. Les batchs computed sur rows partiellement visibles ont produit des résultats faux (LUSR Madina) → recompute force=true requis (Phase 4.4).

## Implementation

Voir Phase 1 du plan `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md` :
- Phase 1.3 : implémentation `InsertParticipantsSafe` avec wrapper singleflight.
- Phase 5.1 : test stress concurrent UPSERT (TDD avant Phase 1.3).
- Phase 4.1 : ART rebuild runtime pour les matchs déjà corrompus.
- Phase 4.4 : recompute force=true post-rebuild pour les batchs invalidés.

## References

- Plan stabilisation : `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md`
- Audit handoff : `.ai/HANDOFF_SYNC_CONCURRENCY_AUDIT.md`
- ADR 0013 — LeasedWriter (lock applicatif par path DB)
- ADR 0016 — SharedDBProvider B-swap (RO↔RW)
- `apps/go-api/internal/platform/duckdb/art_probe.go` — `BootARTGuard` détection
- `apps/go-api/internal/migration/steps_shared_rebuild_match_participants.go` — rebuild one-shot existant
- `apps/go-api/internal/migration/steps_shared_social_purge_data_health.go` — pattern swap-table déjà utilisé sur `player_notifications`
- DuckDB issues : [#18782](https://github.com/duckdb/duckdb/issues/18782), [#16520](https://github.com/duckdb/duckdb/issues/16520), [#8147](https://github.com/duckdb/duckdb/issues/8147)
- Go stdlib : [`golang.org/x/sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight)
