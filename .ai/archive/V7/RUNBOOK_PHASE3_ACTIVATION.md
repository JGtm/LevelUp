# Runbook Phase 3 — Activation progressive Collect→Persist

**Branche** : `refactor/collect-persist`
**Pré-requis** : Phase 1 + 2 mergées (commits 65a63900 → 04b56f96)
**ADR** : `docs/adr/0019-collect-persist-architecture.md`

---

## Objectif

Activer le chemin `submitMatchAsBatch` (INSERT-only via Persisters) en lieu et place du chemin legacy `insertFetchedMatch` (UPSERT direct sur `shared.match_participants`). Cible : supprimer définitivement les `FATAL Error: Invalid Input Error: Failed to delete all rows from index` qui se produisent en concurrence multi-joueurs.

**Mode initial activé** : sync (path Persister direct, sans WAL). L'async (queue + worker) reste désactivé en Phase 3 ; activable en Phase 4 si bénéfice observé.

---

## Étape 1 — Staging : 1 joueur, 1 cycle

### Préparation

```powershell
# Sur la machine staging (Windows PowerShell)
$env:LEVELUP_PERSIST_BATCH = "1"
```

Sur Linux :
```bash
export LEVELUP_PERSIST_BATCH=1
```

### Activation

Redémarrer le serveur Go avec la variable en place. Vérifier au boot :

```
INFO sync: utilisation client personnalisé (pool)
INFO sync: DBs ouvertes
```

Pas d'INFO/WARN nouvelle attendue au boot — le flag s'active seulement quand un sync démarre.

### Déclenchement d'un sync

Soit attendre le scheduler auto-sync (intervalle config), soit déclencher manuellement :

```bash
curl -X POST http://localhost:8080/api/sync/Madina97294
```

### Observation

Logs attendus pendant le cycle :

```
INFO submitMatchAsBatch: match persisté gamertag=Madina97294 match_id=... participants=N medals=N
```

Au lieu de l'ancien :

```
INFO sync: match traité (parallèle) ...
```

Métriques expvar (HTTP GET sur `/debug/vars`) — chercher les nouveaux compteurs :

```json
{
  "persist_shared_total_ok": 12,
  "persist_player_total_ok": 12,
  "persist_batch_committed_total": 12,
  "persist_shared_total_error": 0,
  "persist_player_total_error": 0
}
```

### Critères Go/No-Go (étape 1)

| Critère | Cible |
|---|---|
| FATAL ART pendant le cycle | **0** (vs 1+ en mode legacy sous concurrence) |
| `persist_shared_total_error` | 0 |
| `persist_player_total_error` | 0 |
| Cycle complet | < 8 min |
| Rows en DB cohérentes vs legacy | Sanity check : `SELECT COUNT(*) FROM match_registry WHERE first_sync_at > NOW() - INTERVAL '5 minutes'` |

Si Go → étape 2. Si No-Go → désactiver `LEVELUP_PERSIST_BATCH`, redémarrer, ouvrir incident.

---

## Étape 2 — Staging : 5 cycles consécutifs

Laisser le scheduler auto-sync tourner pendant ~30 min (5+ cycles selon l'intervalle).

### Critères Go/No-Go (étape 2)

| Critère | Cible |
|---|---|
| 5 cycles consécutifs sans FATAL ART | OK |
| `persist_shared_total_error` cumulé | 0 |
| Aucun WARN `submitMatchAsBatch` répété | OK |
| Temps moyen cycle | ≤ legacy (±20%) |

---

## Étape 3 — Prod : 4 joueurs, observation 24h

```bash
# Sur la machine prod
export LEVELUP_PERSIST_BATCH=1
```

Redémarrer le serveur. Observer pendant 24h.

### Critères Go/No-Go (étape 3)

| Critère | Cible |
|---|---|
| FATAL ART sur 24h | **0** |
| `art_corruption_detected_*` sur 24h | 0 |
| Tous les enrichments présents post-cycle | sanity SQL |
| Pas de plainte utilisateur sur des stats manquantes | feedback |

---

## Étape 4 — Flip default à on

Une fois étape 3 validée :

1. Modifier le code pour que `WithBatchPersistMode(true)` soit le défaut.
2. Inverser la variable d'environnement : `LEVELUP_PERSIST_BATCH=0` pour désactiver (rollback rapide).
3. Communication au reste de l'équipe.
4. Phase 5 cleanup démarre (suppression des anti-patterns : singleflight, CHECKPOINT, UPDATE-then-INSERT, RebuildART).

---

## Rollback rapide

Pendant les étapes 1-3, en cas d'incident :

```bash
unset LEVELUP_PERSIST_BATCH
# ou
export LEVELUP_PERSIST_BATCH=0
```

Redémarrer le serveur. Le path legacy `insertFetchedMatch` reprend immédiatement (zéro changement de comportement vs pre-refactor).

---

## Activation Phase 4 (async, plus tard)

L'async path (`BatchQueue` + Worker) est livré en code mais pas câblé côté serveur. Pour l'activer :

1. Au boot du serveur (`cmd/server/main.go`) :
   - Créer une `*persist.BatchQueue`
   - Créer les Workers Shared+Player (background goroutines)
   - `queue.RecoverPending()` au boot (re-pousse les WAL non-ACKés post-crash)
2. Injecter la queue dans le `SyncEngine` via `WithBatchQueue` (en plus de `WithBatchPersistMode`).
3. Le `submitMatchAsBatch` route automatiquement vers `queue.Submit` quand `batchQueue != nil`.
4. `run()` appelle `queue.Drain(ctx)` (timeout 60s) entre la fin de la boucle pagination et `runConditionalPostSync` — garantit que post-sync compute lit la DB peuplée.

Critères Go/No-Go async :
- Crash mid-cycle → reprise propre au boot (vérifier WAL recovery)
- WAL files supprimés après ACK (no leak)
- Pas de timeout Drain en nominal (< 60s par cycle)

---

## Métriques expvar à monitorer

| Métrique | Description | Interprétation |
|---|---|---|
| `persist_shared_total_ok` | Persists shared OK | Doit augmenter à chaque match inséré |
| `persist_shared_total_error` | Persists shared en erreur | **Doit rester à 0** |
| `persist_player_total_ok/error` | Idem pour player DB | Idem |
| `persist_batch_committed_total` | Batches complets (shared + player OK) | = nb matches inserted |
| `persist_batch_submitted_total` | Async only — Submit dans la queue | Doit converger vers `_committed_total` |
| `persist_batch_submit_error` | Async only — Submit en échec | **0** |

Accès : HTTP GET `/debug/vars` (expvar standard).

---

## Critères de succès Phase 3 (récap)

1. ✅ 10 cycles consécutifs prod sans `FATAL Error: Invalid Input Error`
2. ✅ `art_corruption_detected_*` à 0 sur 24h
3. ✅ Tous les enrichments présents post-cycle
4. ✅ Cycle 3 joueurs < 8 min
5. ✅ Pas de plainte utilisateur

→ Une fois ces 5 critères verts, Phase 5 cleanup peut démarrer.
