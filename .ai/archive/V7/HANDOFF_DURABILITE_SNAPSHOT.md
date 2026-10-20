# HANDOFF — Durabilité / Snapshot immuable (Phase 2 en cours)

> Doc de reprise après compaction. Lire en entier avant de continuer.
> Date : 2026-06-25.

## 0. TL;DR

- **Branche** : `refactor/durabilite-snapshot-immuable`, poussée sur `origin`. **PAS mergée sur `main`**.
- **Worktree dédié** : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-durabilite-snapshot` (isolé du dépôt principal). Toutes les commandes Go/git ci-dessous s'y exécutent.
- **Fait + testé** : Phase 0 (B-swap instrumentation), Phase 1.b (`CommitThenAdvance`), **Phase 2 producteur+monitoring (`3600007f4`)**, **Phase 3 lecture snapshot + cutover MatchView (`083f22eea`)**, Phase 4 global (`c6214c3ab`) **REVERTÉ par la vérification adversariale** → **lecture-snapshot SCOPED MatchView retenue** (cf. thought_log [2026-06-25] « Vérification finale + remédiation »).
- **Câblage retenu = SCOPED MatchView** : `MatchViewRepo.WithSharedReader` + helper `sharedRead()` + reader snapshot-préféré SINGLETON par titre (`ServiceRegistry.snapReaders`, fermé au shutdown). `OpenSnapshotShared` reconstruit le schéma shared complet (matérialisation + vues canoniques). **GLOBAL abandonné** : cassait le classement mondial (`world_csr_leaderboard_latest` lue via le même SharedReader, hors snapshot, données non-match-immutables). Médias (SharedSocial) + dérivés player (ReadDB) restent live.
- **Remédiation (28 findings adversariaux)** : dead code dérivé retiré (`OpenSnapshotForPlayer`/export dérivé/createSnapshotView) ; logging dédié `logs/snapshot.log` (`ModuleSnapshot`) + fallback/`ErrSnapshotIncomplete`/force-ready loggés (negative-cache anti-spam) ; `rows.Err()` propagé (`ProduceSnapshot` no-op `read_incomplete` au lieu de figer un set tronqué) ; tests ajoutés (change-gate re-cut, rétention/purge, métriques cut, reader fallback-incomplet).
- **Reste** : **déploiement prod = décision utilisateur** (downtime ; merge main = auto-deploy). Au deploy : producteur peuple les snapshots → MatchView bascule (fallback live avant), mesurable `snapshot_read_served/live_fallback_total` + `shared_provider_reader_stall_ns_total`. Revert = binaire précédent (additif, zéro migration destructive). GLOBAL sûr = phase future (reconstruction complete-by-construction de tout le schéma shared). Suivi non bloquant : régénérer `apps/web` generated-types.
- **Commits** : Phase 4 global `c6214c3ab` (poussé), remédiation revert→scoped+fixes `7967f6c16` (poussé, hook gofmt+vet vert). `main` intact.
- **Plan maître** : `.ai/PLAN_DURABILITE_SNAPSHOT_IMMUABLE.md` (Option B = lecture snapshot seule ; readiness marker ; grâce bornée).

## 1. Objectif

Découpler les lectures des écritures : servir les lectures depuis des **snapshots Parquet immuables versionnés** pour que le sync (écritures) ne bloque jamais l'app (problème B-swap : RO+RW interdits sur le même fichier DuckDB in-process). Un match n'entre dans un snapshot que **complet** (toutes dérivations terminales). Contrat de fraîcheur : un nouveau match apparaît ~1s après complétion ; les écritures ne stallent jamais les lectures.

## 2. Commits (branche, du plus récent)

| Hash | Sujet |
|---|---|
| `2e92e640f` | feat(snapshot): orchestration readiness + câblage post-sync étape 6 |
| `6041b93ad` | feat(snapshot): prédicat snapshot-ready + capability CapWeaponKills |
| `4d2592f4a` | feat(snapshot): fondation readiness marker (colonnes snapshot_ready_at/partial_reasons) |
| `8a3f0bb74` | refactor(sync): invariant durable-avant-progrès (CommitThenAdvance) |
| `e3a2cdfef` | fix(test): start_time_utc aux fixtures match_registry (dette pré-existante) |
| `b30282d70` | feat(observability): instrumente le stall lecteur réel du B-swap (Phase 0) |

`3e02aad80` (le plan, sur `main`) est l'ancêtre. Le doc plan existe aussi dans le worktree.

## 3. Détail de ce qui est fait

### Phase 0 — instrumentation B-swap (`internal/platform/duckdb/sharedprovider/`)
5 signaux expvar : `shared_provider_reader_stall_ns_total` (vrai stall lecteur côté Get, ≠ drain moteur `get_wait_ms_total`), `reader_delayed_total`, `rw_window_ms` (avg/max, PAS de p50/p95 — infra observability ne les fait pas), `swap_failures_total{drain_timeout}` (désambiguïsé d'`acquire_writer`). Fichiers : `metrics.go`, `provider.go` (Get + champ rwWindowStart), `provider_writer.go` (releaseWriter), `snapshot.go` (SwapSnapshot étendu) + test `reader_stall_metrics_test.go`. **But** : mesurer en prod si le stall justifie Phases 2-4.

### Phase 1.b — invariant durable-avant-progrès
`internal/sync/durable_progress.go` : helper `CommitThenAdvance(ctx, held, step)` (n'avance le progrès qu'après écriture durable ; sinon tient le groupe). `skill_v2_shadow.go` : `processOneShadowMatch` refactoré via le helper, `canonicalGate` SUPPRIMÉ (comportement identique, prouvé par 2 tests e2e oracle). Test unitaire pur `durable_progress_test.go` (4 cas). **NB** : c'était un refactor de clarté, l'invariant était déjà correct. + dette : `start_time_utc` ajouté aux 3 fixtures (`skill_rating_loaders_test.go`, `skill_rating_dryrun_test.go`, `recompute_after_art_rebuild_test.go`).

### Phase 2a — readiness marker (COMPLET, fonctionnel)
- **Migration** : `snapshot_ready_at` + `partial_reasons` en colonnes append-only via un stage propriétaire dédié `'snapshot'` dans `internal/migration/steps_player_append_only_match_enrichment.go` (`pmeColumnStage` + `ensurePMEColumns`). Le merge-on-read par stage garantit que perf/psa n'écrasent jamais `snapshot_ready_at`. Pas de migration v2 (v1 idempotente).
- **Capability** : `CapWeaponKills` (`registry.go` const + liste Infinite ; `config_loader.go` knownCapabilities). Infinite oui, Halo 5 non. **NE PAS** confondre avec `CapNativeKillMechanics` (Halo-5-only).
- **Prédicat pur** : `internal/sync/snapshot_readiness.go` — `isMatchSnapshotReady(facts, caps, agedOut)` + enum FERMÉ de raisons (`lusr_ineligible`, `lusr_skipped`, `weapons_absent`, `forced`, `blocked_*`). 10 cas unitaires (`snapshot_readiness_test.go`).
- **Orchestration** : `internal/sync/snapshot_readiness_eval.go` — `evaluateSnapshotReadiness(ctx, playerDB, sharedDB, xuid, titleSlug)` : 2 requêtes Go-diff (player facts + shared facts, ZÉRO ATTACH), INSERT pur `stage='snapshot'`, grâce par âge (`LEVELUP_SNAPSHOT_GRACE_HOURS`, défaut 60j). Helpers caps `snapshotReadinessCaps`/`slugProducesWeaponKills`.
- **Câblage** : étape 6 de `runPostSyncPipeline` (`engine_postsync.go`, best-effort) + `SnapshotReadyMarked` dans `domain/sync.go` PostSyncResult + step `snapshot_readiness` dans `postsync_clock.go`.
- **Test** : `snapshot_readiness_integration_test.go` (dataset hétérogène complete/transient/ffa, idempotent, propagation vue `_latest`). Gardes anti-ART verts.

## 4. RESTE À FAIRE — Phase 2 producteur + monitoring (plan 13 étapes, étapes 7-13)

> Design autoritaire = sortie du workflow `wg8fx58bl` (peut ne pas survivre au compact ; résumé ci-dessous, corrections du verdict adversarial INCLUSES).

7. **`PathResolver`** (`internal/domain/title/registry.go`, après `WarehouseDir`) : `SnapshotsDir(slug)` = `data/titles/{slug}/warehouse/snapshots/`, `SnapshotVersionDir(slug, version)` = `Join(SnapshotsDir, fmt.Sprintf("v%020d", version))` (tri lexico=chrono), `SnapshotCurrentManifestPath(slug)` = `Join(SnapshotsDir, "CURRENT.json")`. Pur join.
8. **Briques ops** (NOUVEAUX fichiers `internal/ops/snapshot_*.go`, testables isolément) : `SharedReadOpener` iface `{ OpenSharedRO(ctx)(*sql.DB,func(),error) }` (découple ops de platform/duckdb) ; `snapshot_manifest.go` (struct `SnapshotManifest{Version,TitleSlug,CreatedAt,Watermark,ReadyMatchCount,PartialMatchCount,Partitions[],SchemaVersion}` + `PartitionInfo{Table,Month,RelPath,RowCount,SizeBytes,SHA256}` + writeManifest/readCurrent(0 si absent)/**flipCurrent (write tmp + os.Rename ATOMIQUE)**/sha256File) ; `snapshot_retention.go` (`applyRetention(keep, currentVersion)` garde keep+CURRENT ; `compactMonth` swap .tmp→rename, garde row-count). Tests : flip atomique (lecteur voit N ou N+1 jamais mixte), readCurrent=0 dir vierge, rétention protège CURRENT, compaction préserve row count.
9. **Producteur** (`internal/ops/snapshot.go` + `snapshot_export.go`) : `ProduceSnapshot(ctx, opts)` : version=readCurrent+1 ; mkdir SnapshotVersionDir ; ouvre shared RO via `opts.SharedReader` (**OpenReadForQuery / SharedReadDB — JAMAIS `?access_mode=read_only` direct ni ATTACH player-DB** : incident 2026-06-01) ; **N COPY séparés** des 5 faits shared (match_registry, match_participants, medals_earned, highlight_events, killer_victim_pairs) + 3 dérivés ancrés (player_match_enrichment_latest, match_skill_rank_latest filtrable rating_type='LUSR', match_citations_latest) **filtrés `WHERE snapshot_ready_at IS NOT NULL`** ; partition mensuelle `strftime(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC'),'%Y-%m')` ; `COPY ... (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL 9)` ; checksums ; writeManifest puis **flipCurrent atomique** ; applyRetention. **No-op propre si 0 match ready** (ne PAS flipper CURRENT vers du vide). **PAS une extension d'`archive.go`** (qui fait sql.Open direct + filepath.Join brut + DELETE = violations).
10. **Câblage cycle V2** (`internal/sync/v2/cycle.go` + `cmd/server/sync_v2_wiring.go`) : Phase 6bis après `RunPostSync`, writer-lease shared relâché, UNE fois par titre. Iface `SnapshotProducer.CutSnapshot(ctx, titleSlug)` injectée. **`NewCycleOrchestrator` panic sur dépendance nil** → nil-guard EXPLICITE sur ce seul param. Best-effort (échec n'invalide pas le cycle). Inconditionnel (pas de flag partiel).
11. **Métriques cut** (`internal/sync/snapshot_metrics.go`, miroir skill_v2_metrics.go) : enum FERMÉ reasons + `snapshot_cut_total`/`snapshot_cut_failures_total{reason∈copy_failed,manifest_flip_failed,read_ro_failed}`/`snapshot_cut_duration_ms`/`snapshot_version` (GAUGE SetIntT). JAMAIS de clé expvar dérivée de match_id/xuid.
12. **Gauges pending** (`internal/sync/snapshot_report.go`) : **agrégat GLOBAL-par-titre read-only** (itère LoadPlayers + somme) — **PAS un SetIntT per-joueur** (écraserait la valeur entre joueurs = bug). Prédicat pending = `snapshot_ready_at IS NULL` (PAS le backlog de convergence). GAUGE=SetIntT (overwrite) vs CUMUL=AddIntT.
13. **Section admin** (`internal/api/registry_monitoring.go` `MonitoringOverview` = le vrai zéro-DuckDB-I/O ; DTO `domain/admin_monitoring.go`).

## 5. GARDE-FOUS — erreurs déjà écartées (NE PAS les refaire)

- **`dominance_flag` n'est PAS sur `shared.match_registry`** (il est sur `player_match_enrichment`, déjà append-only stage='dominance'). « Phase 1.a » du plan = tâche FANTÔME, annulée.
- **`CapNativeKillMechanics` = Halo-5-only** → ne JAMAIS l'utiliser pour gater weapon-kills. → `CapWeaponKills` (Infinite only) créée.
- **LUSR** : un match 2-team fact-éligible SANS row LUSR (imbalance/DNF/group-hold) = `lusr_skipped` NON bloquant, JAMAIS un gel.
- **Gauge pending per-joueur ÉCRASE** → agrégat global-par-titre.
- **ATTACH player-DB unsafe** (« different configuration ») → N COPY via OpenReadForQuery.
- **INSERT pur** sur player_match_enrichment (table protégée no_art_patterns) — jamais UPDATE/DELETE indexé.
- **Timezone canonical** : `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')` pour partition + watermark.

## 6. Toolchain / commandes (worktree)

```bash
WT="C:/Users/Guillaume/Downloads/Scripts/LevelUp-durabilite-snapshot/apps/go-api"
# CGO requis (driver DuckDB) : gcc msys64 dans PATH, CGO_ENABLED=1 (défaut OK).
go -C "$WT" vet ./internal/sync/ ./internal/ops/ ./internal/migration/ ./internal/domain/title/
go -C "$WT" test -tags integration -count=1 -run 'Snapshot' ./internal/sync/ ./internal/ops/
# -race exige : -gcflags=all=-d=checkptr=0
```
- **Outil Bash** : lancer les builds/tests/push avec `dangerouslyDisableSandbox=true` (CGO/gcc/réseau).
- **Commits** : `git -C ../LevelUp-durabilite-snapshot ...`. Hook pre-commit lefthook (gofmt/vet) tourne ; le bruit « build constraints exclude all Go files » sur les cmd/ est NORMAL (tags), pas une erreur.
- **Push** : `git -C ../LevelUp-durabilite-snapshot push` — branche, **PAS `main`** (push main = auto-deploy prod).

## 7. Gates / décisions

- **Déploiement prod = merge sur `main` = auto-deploy (downtime)** = DÉCISION UTILISATEUR. Ne PAS merger main sans accord explicite.
- La feature devient « live » (lectures servies depuis snapshot) seulement après producteur + chemin de lecture (Phase 3) + deploy.

## 8. État dépôt (piège)

Le **répertoire principal** `LevelUp-go-migration` est sur la branche `chore/doc-triage-v7` (un autre agent réorganisait `.ai/` → `.ai/V7/`). Mon travail est **ISOLÉ dans le worktree** `refactor/durabilite-snapshot-immuable`. Ne pas mélanger.

## 9. Préférences utilisateur (rappel)

- Être **autonome**, ne pas s'arrêter pour des broutilles ; commit+push+continue.
- Réponses **factuelles, pas de blabla**.
- Escalader seulement les vraies décisions produit/correctness (ex. Option A weapon-caps déjà tranchée par l'user).
- Pas d'emojis dans les fichiers versionnés.
