# PLAN — B-swap shared : retirer (Option A) ou atténuer (Option B)

**Date** : 2026-06-10
**Branche cible proposée** : `perf/shared-swap-window` (Option B) ou `refactor/remove-bswap-confine-shared-writer` (Option A)
**Méthode** : workflow multi-agents (35 agents — inventaire lecture, état B-swap, invariants écriture, synthèse + revue adversariale). Constats vérifiés sur le code réel, pas sur l'audit périmé.

---

## 0. Constat de départ — VÉRIFIÉ (corrige le diagnostic initial)

Trois faits établis par l'inventaire (croisé code réel) :

1. **La migration lecture est DÉJÀ FAITE.** `attachShared` a été entièrement retiré des connexions player (commits `9c.5`/`9feb07e1e`, `9c.4`/`772119b92`). ~113 sites de lecture passent par `pdb.SharedReadDB().Get()` avec split 2-phases + merge Go (squad/career/leaderboard inclus). En mode B-swap, `pool.go` n'ouvre même pas `pdb.Shared` (= nil). **Zéro site de lecture cross-DB restant à migrer** (`confirmed: []`).
   → L'audit `.ai/V7/AUDIT_SHARED_READER_GAPS.md` (2026-05-20) et la note `main.go:410-414` (« attachShared reste en place pour squad_repo ») sont **PÉRIMÉS**.

2. **Le B-swap n'existe PAS à cause de l'ATTACH** (qui n'existe plus). Il existe à cause d'une contrainte **driver DuckDB-Go mono-process** : un handle RO et un handle RW ne peuvent pas coexister sur le même fichier dans le même process (`Can't open ... with a different configuration`). Le serveur tient shared en RO permanent ; pour écrire, le `sharedprovider.Provider` : draine les lecteurs (`drainTimeout=5s`) → ferme le RO → ouvre RW → écrit (append-only via persist) → ferme RW → rouvre RO. Les lectures arrivant pendant la fenêtre bloquent sur `<-ready` (`readyTimeout=30s` → 503).
   **Le « stall pendant le sync » = cette fenêtre de drain**, pas l'ATTACH.

3. **Retirer le swap est un problème d'ÉCRITURE / cycle de vie du handle, pas de lecture.** Le travail résiduel n'est pas côté lecteur (fait) mais côté writer.

### Assurance ART (vérifiée)
Un refactor **strictement lecture** ne peut pas réintroduire l'ART : la corruption ART est exclusivement un phénomène d'**écriture** (UPSERT/UPDATE/DELETE concurrents sur index ART). Les 4 garde-fous (`TestNoARTPatternsOnProtectedTables`, `TestNoBulkMultiRowUpdateOnCriticalTables`, `TestSharedMatchTablesWrittenViaPersistOrAllowlist`, `TestNoDirectCombatWritesOutsidePersist`) + `BootARTGuard` + `dblease` ne référencent pas le B-swap : le retirer ne retire aucun garde-fou ART. **MAIS** (voir Option A / F2) externaliser le writer SORT du mono-process et peut, lui, réintroduire un risque ART par concurrence cross-process.

---

## 1. La bifurcation

| | **Option A — externaliser le writer** | **Option B — garder le swap, tuer le stall** |
|---|---|---|
| Objectif | Supprimer le B-swap : le serveur n'ouvre jamais shared en RW | Réduire la fenêtre RW (drain) ressentie |
| Touche l'écriture ? | Oui (déplacement + délégation des endpoints) | Non (juste la **cadence** des flushes) |
| Nouveau process | Oui (`cmd/sync-worker`) + lock inter-process | Non |
| Effort | **Lourd** (plusieurs jours) | **Faible/moyen** |
| Risque ART | **Réel** (cf. F2 : dblease non cross-process) | Nul (aucune écriture modifiée) |
| Risque dispo | Coexistence RO+RW Windows non prouvée (F3), staleness RO (F5) | Faible |
| Quand | Seulement si B prouve que le swap est le plafond à l'échelle cible | **Maintenant** |

**Recommandation : Option B d'abord (mesurer + resserrer), Option A en réserve.**

---

## 2. Option B (RECOMMANDÉE) — réduire la fenêtre RW

Hypothèse à valider : le ralenti dure « tout le sync » parce que le RW est tenu trop longtemps **ou** que le swap se déclenche trop souvent (un drain par batch). Rappel : l'audit perf (PLAN_SYNC_CONCURRENCY §0) a mesuré les writes DB **sub-second** ; donc un stall « de plusieurs minutes » trahit une **cadence** de swap, pas un débit.

### Phase B1 — Diagnostic (lecture seule, risque nul)
- Lire la granularité acquire/release du writer shared : `internal/persist/combined_persister.go` (Worker live), `internal/sync/engine_acquire.go:51`, `engine.go:251/531`. Combien d'`AcquireWriter`/`Release` par cycle de sync ?
- Compteurs expvar swap (`/debug/vars`) : nombre de swaps/cycle, durée moyenne RW, occurrences `ErrSwapTimeout`/503.
- **DoD** : un chiffre clair « X swaps/cycle, RW tenu Y ms median/cycle ». Décide si le stall est un réglage (cheap) ou un plafond (→ Option A).

### Phase B2 — Resserrer la fenêtre RW (si B1 montre trop de swaps / RW trop long)
- Coalescer les écritures : moins de flushes plus gros par cycle (le `BatchBuilder` est conçu pour ça) → moins de drains, chacun bref. **Aucune modif du SQL d'écriture** (sécurité ART intacte), seulement la cadence/taille de batch.
- Borne haute du RW : garantir que le RW est relâché immédiatement après le flush (jamais tenu sur la durée réseau du cycle).
- **DoD** : swaps/cycle réduits, RW median < seuil ; golden lecture inchangé ; 4 garde-fous ART verts.

### Phase B3 — Résilience lecture pendant la fenêtre (optionnel)
- Ajuster `drainTimeout`/`readyTimeout` ; vérifier la dégradation propre (jamais d'avalage silencieux ; 503 borné et observable).
- Métrique `shared_swap_wait_ms` (p99 d'attente lecteur sur `<-ready`).

---

## 3. Option A (RÉSERVE) — confiner l'écriture hors-process

Plan complet **corrigé des 12 findings adversariaux**. À ne lancer que si B1 prouve que le swap reste le goulot à l'échelle cible. NON réalisable « tel quel » sans les corrections F1–F3.

### Pré-requis bloquants (sinon ART/indispo)
- **F1** — Inventaire EXHAUSTIF des writers shared : ce ne sont **pas 2** (sync + cron) mais ~8 surfaces, dont des **endpoints HTTP** (`service/openspartan_import_service.go`, `openspartan_post_import_service.go`, `api/handlers/backfill.go`, `sync_handler.go`) + le Worker persist (`combined_persister.go`). Déplacer « sync + cron » ne suffit pas : les imports/backfills HTTP écrivent shared dans le process serveur. → Les **déléguer** au worker (enqueue de job), pas écrire en direct. Effort **lourd**.
- **F2** — `dblease` est un **mutex in-process** (`map[string]*sync.Mutex`), PAS un lock cross-process. Séparer en 2 process casse la sérialisation writer-unique → 2 RW concurrents possibles → **risque ART**. Exige un **lock inter-process** (fichier `.lock` advisory) + garantie « exactement 1 process RW » + test « 2e writer échoue proprement ».
- **F3** — **Spike Windows GO/NO-GO en Phase 0** (avant tout) : prouver qu'un RO **permanent** dans le serveur coexiste avec un RW d'**un autre process** (le file-lock OS est cross-process — il pourrait re-bloquer le RO à chaque écriture, ramenant le stall) ET que la visibilité WAL/checkpoint est correcte sous charge.

### Findings justesse à intégrer
- **F5** — `SharedReaderRO` permanent peut servir un **snapshot périmé** (matchs récents invisibles jusqu'au restart) → `Reopen()` conditionnel sur invalidation (mtime/IPC) via `atomic.Pointer[sql.DB]` existant. Métrique `shared_ro_staleness_seconds` (canari).
- **F4** — Étendre inventaire/sentinelles à `global`/xuid_aliases et `shared_social` (ATTACH **process-wide**), pas seulement `shared_matches_v2`.
- **F6** — Goldens sur **tous** les sites 2-phases avec tri/pagination/filtre/dedup/NULL/TZ (pas 3) : `match_history_repo` (Q5, pagination+tri sur champ shared), `media_repo` (Q37), `explorer` (Q19 intersection), pièges `start_time NULL` + `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`.
- **F7** — Phase « swap=0 » dépend de F1 (sinon les endpoints HTTP déclenchent encore le swap).
- **F8** — Refactor `NewWorldLeaderboardCron` (lit la fraîcheur via `provider.Get` + écrit) sur une abstraction worker-local **avant** la suppression du package `sharedprovider`.
- **F9** — Sentinelle « pas de RW shared dans le serveur » doit être **runtime** (packages `internal/` partagés serveur/worker → un scan AST par package ne distingue pas) + test sur le graphe d'imports de `cmd/server`.
- **F10** — Topologie 2-process = runbook (superviseur worker, ordre de boot, recovery WAL #7659 Windows, comportement worker-down). Irréversible sans redeploy.
- **F11** — Sortir le dead-code cleanup (Q4/Q30/Q42/Q15/Q26e, `checkXuidAliasMissing`) du commit atomique de retrait Provider ; valider par build+test, pas grep.
- **F12** — Métrique de fraîcheur RO obligatoire.

### Séquence (corrigée)
0. Spike inter-process Windows (gate dur F3). 1. Inventaire writers + sentinelles runtime (F1/F9). 2. Lock inter-process (F2). 3. `cmd/sync-worker` + délégation endpoints HTTP (F1) sous flag OFF. 4. `SharedReaderRO` avec Reopen (F5) + goldens étendus (F6). 5. Défaut ON + soak 48h + runbook (F10). 6. Suppression Provider **en dernier** derrière sentinelle ; ADR 0016 → Superseded, ADR 0019 MAJ, ADR 0026 nouveau. Dead-code en commit séparé (F11).

---

## 4. Références
- `docs/adr/0016-shared-db-provider-b-swap.md`, `0018-concurrent-write-model.md` (CLOSED), `0019-collect-persist-architecture.md`
- `internal/platform/duckdb/sharedprovider/`, `pool.go`, `pool_swap_hook.go`, `db.go` (`OpenReadForQuery` L118)
- `internal/sync/engine_acquire.go`, `no_art_patterns_test.go`, `shared_write_guard_test.go`, `combat_write_guard_test.go`
- `internal/scheduler/world_leaderboard_cron.go`, `cmd/server/main.go` (câblage L392-466, cron L966-976)
- PÉRIMÉ : `.ai/V7/AUDIT_SHARED_READER_GAPS.md`
