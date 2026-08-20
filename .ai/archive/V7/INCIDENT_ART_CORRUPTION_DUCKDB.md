# INCIDENT — Corruption d'index ART DuckDB (récurrent)

**Status** : ✅ **CLOSED** — Phase 4 (post-sync batch INSERT-only) validée empiriquement 2026-05-24 : 4 cycles consécutifs × 4 joueurs = **16 syncs, 0 FATAL ART** sous `LEVELUP_PERSIST_BATCH=1 LEVELUP_POSTSYNC_INSERT_ONLY=1` (post `force_rebuild_art --all true` initial).
**Premier signalement** : 2026-05-20 (`docs/INCIDENT_2026-05-20_match_participants_index.md`).
**Reproduit en prod** : 2026-05-23.
**Phase 1-2 (path INSERT)** : 2026-05-23 — refactor Collect→Persist livré (ADR 0019).
**Phase 4 (post-sync UPDATE)** : 2026-05-24 — 5 sites batch INSERT-only + RebuildMatchSkillRankART (commits 14dfd135, 4ef122b7).
**Phase 5 (cleanup anti-ART)** : 2026-05-24 — singleflight + CHECKPOINT + auto-heal supprimés.
**Dernière mise à jour** : 2026-05-24.

## Pré-requis prod (héritage corruption)

`force_rebuild_art --all true` est requis avant déploiement Phase 4 sur une DB qui a vécu en mode legacy (ART probablement corrompue). Le CLI rebuild les **3 tables critiques** : `shared.match_participants`, `player_match_enrichment`, `match_skill_rank`.

Phase 4 batch INSERT-only **prévient** les futures corruptions mais ne **défait pas** une corruption pré-existante.

## RÉSOLUTION (2026-05-23)

Le refactor **Collect → Persist** est livré : couche persistance INSERT-only sur 4 DBs (Shared/Player/PVE/Metadata) + orchestrateur sync `submitMatchAsBatch` (engine_batch_path.go) + feature flag `LEVELUP_PERSIST_BATCH=1`.

**3 modes opérationnels** :
1. Legacy (default) — `insertFetchedMatch` UPSERT direct sur `shared.match_participants` → ART bug actif.
2. **Sync batch** (`LEVELUP_PERSIST_BATCH=1`) — `submitMatchAsBatch` + `persist.SharedPersister/PlayerPersister.Persist()` (INSERT-only, pas de WAL). **ART bug supprimé par construction**.
3. **Async batch** (`+ WithBatchQueue`) — queue + worker + WAL durable + Drain. À activer Phase 4 si bénéfice observé.

**Garanties post-activation Phase 3** :
- Aucun UPDATE/UPSERT concurrent sur `shared.match_participants` (les workers persisters utilisent INSERT en TX atomique avec check `EXISTS(match_id)` pour l'idempotence).
- Pas de DELETE-side index access → pas de stress sur l'ART → corruption impossible par construction.

**Référence** : ADR 0019 (`docs/adr/0019-collect-persist-architecture.md`), runbook activation (`.ai/RUNBOOK_PHASE3_ACTIVATION.md`).

## VERDICT EMPIRIQUE FINAL (2026-05-23 19h33)

Migrations UPDATE-then-INSERT (commit `acad4603`) ont été testées en prod : **AUCUN effet**. Le FATAL revient exactement de la même façon :

```
19:33:20 upsertLUSRRatings: exec LUSR update failed
FATAL Error: Invalid Input Error: Failed to delete all rows from index.
```

**Cause root** : DuckDB étant **columnar**, les UPDATE sont implémentés comme **DELETE+INSERT en interne** (rewrite de la row entière). Donc UPDATE consulte aussi l'index ART en mode DELETE → bug ART déclenché par UPDATE.

**Implications** :
- Aucun pattern SQL (UPSERT, UPDATE, INSERT OR REPLACE) ne contourne le bug
- La seule solution = **éliminer les UPDATEs concurrents** sur les tables critiques
- → Architecture Collect → Persist avec INSERT-only batch

**Doc dédié** : [REFACTOR_COLLECT_PERSIST.md](REFACTOR_COLLECT_PERSIST.md) — design complet validé 2026-05-23.

---

## Résumé exécutif

DuckDB embedded (driver `github.com/duckdb/duckdb-go/v2`) souffre d'un bug de corruption d'index ART (Adaptive Radix Tree) sur les tables avec PK VARCHAR soumises à des UPSERTs concurrents intensifs. Le bug se manifeste sous **deux facettes** :

1. **SELECT-side** (mai 2020) : `WHERE pk = ?` retourne un sous-ensemble strict des rows réelles (l'index ART point vers moins d'entries qu'il n'en existe).
2. **DELETE-side** (mai 2023 incident) : `ON CONFLICT DO UPDATE` / `INSERT OR REPLACE` exécute en interne DELETE+INSERT ; la phase DELETE consulte un index ART contenant des entries fantômes → `FATAL Error: Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of N rows`.

Le bug **régénère après un rebuild** : usage prolongé → corruption revient. Le driver bump v1.5.2 → v1.5.3 **réduit la facette SELECT mais ne fixe pas DELETE**.

**Solution actuelle (combinaison)** :
- ✅ Rebuild swap CTAS via CLI `cmd/force_rebuild_art/` (manuel, on-demand).
- ✅ Auto-heal au boot via `BootARTGuard` (uniquement SELECT-side, ne détecte pas DELETE-side).
- 🟡 **Plan J en validation** : `PRAGMA CHECKPOINT` après chaque post-sync pour évacuer le WAL et éviter l'accumulation d'entries fantômes (commit `ae82901e`).

---

## Symptômes observés

### Facette SELECT-side (mai 2020 historique)

```
loadLUSRParticipants chargeait 1-2 participants au lieu de 8-16
→ LUSR Madina figé à Argent IV au lieu de Platine attendu
```

Pattern de détection (BootARTGuard) :
```sql
SELECT COUNT(*) FROM match_participants WHERE match_id = ?       -- ART pushdown, ratele bug
SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?  -- Table scan, bon résultat
```
→ Divergence non-nulle = corruption ART détectée.

### Facette DELETE-side (mai 2023 — cette session)

```
17:07:38 [WARN] post-sync: sessions échoué gamertag=Chocoboflor
err="recalculateSessionsInline write: WriteSessionAssignments(ec938fb4-...):
FATAL Error: Invalid Input Error: Failed to delete all rows from index.
Only deleted 0 out of 1 rows.
Chunk: Chunk - [23 Columns]
- FLAT VARCHAR: 1 = [ ec938fb4-a1ad-42c2-aa64-72df11b7256f]
- CONSTANT FLOAT: 1 = [ NULL]
..."
```

Cascade immédiate sur la même connexion DuckDB :
```
17:07:38 [WARN] post-sync: perf scores échoué err="...FATAL Error: Failed: database has been invalidated because of a previous fatal error. The database must be restarted prior to being used again."
17:07:38 [WARN] post-sync: LUSR échoué        err="...invalidated"
17:07:38 [WARN] post-sync: friends recompute    err="...invalidated"
17:07:38 [WARN] aggregates: échec vue mv_player_matches err="drop mv_player_matches: ...invalidated"
17:07:43 [WARN] achievements: sync échouée    err="...invalidated"
```

**Pattern signature à grep** : `Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of N rows`

---

## Tables affectées (catalogue)

| DB | Table | PK | Première observation | Statut |
|---|---|---|---|---|
| `shared_matches_v2.duckdb` | `match_participants` | `(match_id, xuid)` VARCHAR | 2026-05-20 | Rebuild OK 2026-05-23 |
| `shared_matches_v2.duckdb` | `player_notifications` | `?` VARCHAR | 2026-05-20 (handler NotificationsRepo.List) | À investiguer |
| `data/players/{gt}/stats.duckdb` | `player_match_enrichment` | `match_id` VARCHAR | 2026-05-23 17h07 | Rebuild OK 2026-05-23 18h11 (4 joueurs) |
| `data/players/{gt}/stats.duckdb` | `streaks` | `?` VARCHAR | 2026-05-23 04h21 (StreaksRepo.List handler) | À investiguer |

**Pattern commun** : toutes ces tables ont une **PK VARCHAR + UPSERT `ON CONFLICT DO UPDATE`** (ou `INSERT OR REPLACE` équivalent).

Le code Go qui les manipule :
- `internal/sync/writes.go::InsertParticipants` (singleflight wrappé, phase 2.3)
- `internal/sync/skill_rating_loaders.go::upsertLUSRRatings` → `match_skill_rank`
- `internal/sync/engine_postsync.go::recalculateSessionsInline` → `player_match_enrichment`
- Handlers HTTP qui retournent 500 sur cascade `database has been invalidated`

---

## Causes identifiées (analyse)

### Hypothèses validées

1. ✅ **Le bug est dans DuckDB upstream**, pas dans le code applicatif.
2. ✅ **`INSERT OR REPLACE` et `ON CONFLICT DO UPDATE` partagent le même code path C++** (DuckDB documente le premier comme sugar du second). Confirmé par test TDD `art_upsert_patterns_test.go` (49 erreurs `TransactionContext Error: Conflict on update!` identiques sur les 3 patterns testés en `:memory:`, mais le FATAL ART lui-même n'est **pas reproductible** en `:memory:` — il faut un état disque dégradé).
3. ✅ **Le bug régénère après rebuild** : sentinel `match_participants_rebuilt_v1` déjà posé en prod (rebuild précédent), pourtant la corruption est revenue. Le rebuild swap CTAS répare l'index mais ne prévient pas la nouvelle corruption.
4. ✅ **Le driver v1.5.3 réduit la facette SELECT** (changelog DuckDB 1.4.1 : *"ART index could omit rows non-deterministically when running on multiple threads"*) **mais ne fixe pas DELETE-side**.

### Hypothèses non validées

- **Plan J — Accumulation WAL** : la corruption pourrait être des entries ART fantômes dans le WAL (Write-Ahead Log) qui s'accumulent sans CHECKPOINT régulier. À VALIDER (test en cours).
- **Pattern data spécifique** : certaines combinaisons de match_id / xuid (préfixes hexa similaires ?) pourraient stresser le radix tree davantage. Pas vérifié.
- **Threading concurrent DuckDB** : le bug pourrait dépendre du nombre de cores / parallelism interne DuckDB. Pas vérifié.

### Hypothèses écartées

- ❌ **Race condition dans le code Go** : non, le singleflight (Phase 2.3, commit `aef47968`) sérialise les UPSERTs sur la même clé. Le bug persiste même avec cette protection.
- ❌ **Crash C++ FatalException qui tue le process** : non, le serveur survit aux FATAL. Pas de SIGABRT déclenché (Phase 4.2 inactive). C'est un FATAL DuckDB côté Go, géré gracieusement.
- ❌ **Migration manquante** : non, toutes les migrations applied=0 au boot (DB à jour).

---

## Pistes explorées

### Pistes appliquées

| # | Piste | Statut | Commit | Effet |
|---|---|---|---|---|
| 0a | Driver bump v1.5.2 → v1.5.3 | ✅ Appliquée | `25b56846` | Réduit facette SELECT, ne fixe pas DELETE |
| 2.3 | Singleflight sur `InsertParticipants` | ✅ Appliquée | `aef47968` | Protège contre races applicatives (nécessaire mais pas suffisant) |
| 4.1 | Auto-heal ART au boot (BootARTGuard → rebuild) | ✅ Appliquée | `d2ca98ce` + `20c23eda` | Détecte + répare la facette SELECT au boot. Aveugle à DELETE-side. |
| 4.1.b | `RebuildPlayerMatchEnrichmentART` + CLI étendu `--all` | ✅ Appliquée 2026-05-23 | `487eea4e` | Couvre les player DBs (shared seule ne suffit pas) |
| 4.4 | Recompute force=true post-rebuild | ✅ Appliquée (opt-in) | `b65e0417` + `5d9984fb` | Répare LUSR/perf/dominance/is_with_friends post-rebuild |
| **4.1.c** | **`CHECKPOINT` post-sync (Plan J)** | 🟡 **EN VALIDATION** | `ae82901e` | **Hypothèse : compaction WAL évite régénération corruption** |

### Pistes envisagées non retenues / invalidées

| Piste | Verdict |
|---|---|
| **A. `INSERT OR REPLACE` au lieu de `ON CONFLICT`** | ❌ INVALIDE — Test TDD `art_upsert_patterns_test.go` démontre que les 2 patterns ont exactement le même comportement (49/10000 erreurs sous concurrence). DuckDB implémente l'un comme sugar de l'autre. |
| **B. Auto-heal runtime périodique** | ⏳ Plan B candidat si J échoue. Étendre BootARTGuard pour aussi sonder DELETE-side (probe `BEGIN; DELETE FROM t WHERE pk=non-existant; ROLLBACK`). Plus complexe. |
| **C. Close+Reopen DB sur FATAL via pool interceptor** | ⏳ Combiné avec B. Évite la cascade `database invalidated`. |
| **D. Migration vers PostgreSQL** | ❌ Disproportionné. Refactor majeur (semaines). |
| **E. Reporter upstream à DuckDB** | À faire (court terme, post-validation J). Issue avec repro pattern. |
| **F. Append-only sur match_participants** | ⏳ Plan radical si J échoue. Plus de DELETE → plus de bug ART. Mais refactor des reads + cleanup background. |
| **G. Sharded tables per-player** | ❌ Pas applicable (matchs partagés entre coéquipiers, table partagée nécessaire). |
| **H. SQLite pour tables hot + DuckDB pour analytics** | ❌ Double stack DB trop complexe. |
| **I. Skip no-op UPSERT** (SELECT préalable pour comparer) | ⏳ Combinable avec J. Réduit le volume d'UPSERT → moins de pression sur ART. |
| **K. Désactiver ART pour tables hot** | ❌ Pas exposé par DuckDB ; et lookups O(N) inacceptables. |

---

## Impacts observés

### Impact fonctionnel

- ❌ Cycle auto-sync échoue : 74 erreurs FATAL vs 10 UPSERT OK sur le 1er cycle observé (15h34).
- ❌ Post-sync pipeline cassé : sessions, perf scores, LUSR, dominance, is_with_friends, achievements tous KO en cascade.
- ❌ Vues matérialisées non rafraîchies : `mv_player_matches`, `mv_map_stats`.
- ❌ Handlers HTTP retournent 500 sur les endpoints touchant les tables affectées (`/api/v1/.../streaks`, etc.).

### Impact data

- ⚠️ LUSR figé : Madina97294 affiché Argent IV au lieu de Platine attendu (cause documentée dans `docs/INCIDENT_2026-05-20_match_participants_index.md`).
- ⚠️ Performance scores non recalculés sur les nouveaux matchs.
- ⚠️ Sessions/dominance flags figés.
- ✅ **Pas de perte de data** : la table sous-jacente reste correcte ; seuls les index sont corrompus. Le rebuild swap CTAS préserve toutes les rows (validé : 25069 / 410 / 815 / 1106 / 22 rows préservées sur les 5 DBs rebuilées).

### Impact service

- ✅ Le serveur **ne crash pas** (Phase 4.2 SIGABRT pas déclenchée). Le FATAL est géré côté Go : la connexion DuckDB est marquée invalidated, les requêtes suivantes échouent jusqu'au redémarrage.
- ⚠️ Heartbeat reste vivant (uptime 1h3m20s observé pendant 4 cycles foireux), 23 goroutines stables = pas de leak.
- ❌ Mais les cycles consécutifs échouent tous : pas de progression, pas de nouveau match inséré, snapshot stale.

---

## Diagnostic & outils

### Commandes utiles pour diag à chaud

```bash
# 1. Probe ART sur shared (lit ART vs table scan)
cd apps/go-api
go build -o bin/diag_art_probe.exe ./cmd/diag_art_probe/    # si pas déjà compilé
./bin/diag_art_probe.exe                                     # liste divergences

# 2. Vérifier l'état LUSR vs CSR par joueur (post-rebuild validation)
go build -o bin/diag_csr_check.exe ./cmd/diag_csr_check/
./bin/diag_csr_check.exe

# 3. Inspecter le log structuré pour les FATAL d'aujourd'hui
grep '"2026-05-23' logs/general.log | grep -i 'invalid input' | head -3

# 4. Force rebuild si BootARTGuard est aveugle (DELETE-side)
go build -o bin/force_rebuild_art.exe ./cmd/force_rebuild_art/
./bin/force_rebuild_art.exe --all true                       # shared + 4 player DBs

# 5. Monitorer les FATAL en live pendant un cycle
tail -F logs/server.boot.log | grep -E 'FATAL Error|Invalid Input|database has been invalidated'

# 6. Métriques expvar (post Phase 4.3)
curl -s http://127.0.0.1:8000/debug/vars | grep -oE '"[a-z_]*(art_|singleflight|upsert_match_|checkpoint_)[a-z_]*":\s*[0-9]+'
```

### Métriques expvar à surveiller

| Métrique | Sens | Seuil d'alerte |
|---|---|---|
| `art_corruption_detected_*` | BootARTGuard a détecté divergence SELECT-side | > 0 = corruption présente au boot |
| `art_rebuild_runs_total_ok` | Auto-heal a réparé | > 0 = rebuild tiré |
| `art_rebuild_runs_total_still_diverged` | Rebuild fait mais probe post-rebuild toujours KO | > 0 = bug très profond, investigation manuelle |
| `singleflight_dedupe_total` | Concurrence réelle dédupliquée | > 0 = preuve que la protection sert |
| `upsert_match_participants_total_error` | FATAL côté UPSERT | > 0 = bug ART en cours, debug requis |
| `checkpoint_runs_total_ok_player` / `_shared` | CHECKPOINT a tiré (Plan J) | Doit incrémenter à chaque cycle |
| `checkpoint_runs_total_error_player` / `_shared` | CHECKPOINT a échoué | > 0 = investigation requise |

---

## Solutions appliquées (timeline)

### 2026-05-20 — Incident initial
- `docs/INCIDENT_2026-05-20_match_participants_index.md` documente la facette SELECT-side.
- Workaround applicatif : `WHERE pk || '' = ?` dans certains loaders critiques (forcé table-scan).

### 2026-05-22 — Plan stabilisation
- `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md` v1.

### 2026-05-23 (cette session)
- **Phase 0a** driver bump v1.5.3 (commit `25b56846`) — réduit SELECT-side.
- **Phase 2.3** singleflight (commit `aef47968`) — protection concurrence Go.
- **Phase 4.1** auto-heal ART boot (commits `d2ca98ce` + `20c23eda`) — répare au démarrage si BootARTGuard détecte divergence.
- **Phase 4.4** recompute force=true post-rebuild opt-in (commits `b65e0417` + `5d9984fb`).
- **Migration** `applyRebuildMatchParticipants` historique (référence : `steps_shared_rebuild_match_participants.go`).
- **Validation prod** : rebuild via CLI `force_rebuild_art` exécuté avec succès (25069 rows shared OK).
- **Incident DELETE-side découvert** : 1h après rebuild, 1er cycle auto-sync re-FATAL sur `player_match_enrichment` → BootARTGuard aveugle à cette facette.
- **Phase 4.1.b** rebuild player DBs (commit `487eea4e`) — étend `RebuildPlayerMatchEnrichmentART` + CLI mode `--all`.
- **Phase 4.1.c** Plan J — CHECKPOINT post-sync (commit `ae82901e`) — **EN VALIDATION**.

---

## TODO / Suites possibles

### Si Plan J fonctionne (CHECKPOINT évite régénération)

- [ ] Observer N cycles sans FATAL → confirmer.
- [ ] Documenter la solution comme définitive Phase 4.1.c.
- [ ] Reporter le bug upstream à DuckDB avec repro pattern.
- [ ] Cleanup : retirer le sentinel obsolète `match_participants_rebuilt_v1` (rebuild est désormais runtime).

### Si Plan J échoue (FATAL revient)

- [ ] Plan B : étendre `BootARTGuard.ProbeARTDivergences` pour aussi tester DELETE-side via probe `BEGIN; DELETE FROM t WHERE pk=non-existant; ROLLBACK` (capturer le code d'erreur).
- [ ] Plan C : interceptor sur le pool DB qui détecte `database has been invalidated` → Close+Reopen handle + appelle rebuild auto.
- [ ] Plan I (combinable) : skip no-op UPSERT via SELECT préalable. Réduit le volume = moins de pression sur ART.
- [ ] Plan F (radical) : append-only sur les tables critiques. Plus de DELETE → bug ART impossible. Refactor majeur.

### À investiguer (pas urgent)

- [ ] Caractériser les patterns de keys VARCHAR qui stressent le radix tree (préfixes hexa UUID similaires ?).
- [ ] Tester DuckDB v1.5.4+ quand disponible (peut-être fix upstream).
- [ ] Évaluer la possibilité de migrer les tables hot vers une autre techno (SQLite WAL ? Postgres ? FoundationDB ?). Très bas dans la priorité.

---

## Références

- `docs/INCIDENT_2026-05-20_match_participants_index.md` — incident initial SELECT-side.
- `docs/adr/0018-concurrent-write-model.md` — décision singleflight match_participants.
- `apps/go-api/internal/migration/steps_shared_rebuild_match_participants.go` — rebuild shared (origine).
- `apps/go-api/internal/migration/steps_player_rebuild_match_enrichment.go` — rebuild player (extension 2026-05-23).
- `apps/go-api/cmd/force_rebuild_art/main.go` — CLI standalone (`--db` / `--player-db` / `--all`).
- `apps/go-api/internal/platform/duckdb/art_probe.go` — `ProbeARTDivergences` + `BootARTGuard`.
- `apps/go-api/internal/sync/engine_postsync.go::runCheckpoint` — Plan J (CHECKPOINT post-sync).
- `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md` — plan stabilisation général.
