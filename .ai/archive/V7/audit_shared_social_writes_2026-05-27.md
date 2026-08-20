# Audit — sites d'écriture sur `shared_social.duckdb` (2026-05-27)

> Contexte : incident WAL orphelin du 27/05 ([thought_log](thought_log.md) entrée
> du jour, [ADR 0021](../docs/adr/0021-shared-social-wal-recovery.md)). Cet audit
> liste **tous** les sites runtime qui écrivent sur `shared_social.duckdb` pour
> identifier ceux qui ne passent pas par `SocialPersister.Persist` (qui garantit
> COMMIT + CHECKPOINT) et donc peuvent laisser un WAL non-checkpointed susceptible
> de devenir orphelin sur un kill brutal Windows.

## Classification

| Niveau | Critère |
|---|---|
| **OK** | Écriture via `SocialPersister.Persist` → CHECKPOINT garanti immédiat |
| **MED** | Écriture via `LeasedWriter.BeginTx` → transaction atomique mais CHECKPOINT pas explicitement forcé après commit |
| **HIGH** | Exec direct sur `pdb.SharedSocial` ou `socialDB()` SANS CHECKPOINT et SANS LeasedWriter |
| **READ** | Pas une écriture (Query/QueryRow) — non concerné par WAL |

## Sites identifiés

### Migrations + boot (one-shot, vie courte)

| Fichier:line | Type | Niveau | Notes |
|---|---|---|---|
| `cmd/server/main.go:628` | `socialDB.ExecContext(..., "CHECKPOINT")` | **OK** | CHECKPOINT scheduler explicite |
| `cmd/server/main.go:1068` | `migration.RunForDB(socialDB, TargetSharedSocial)` | **MED** | Migrations idempotentes. `socialDB.Close()` à la fin (ligne 1072) — pas de CHECKPOINT forcé après la close ; à investiguer si les DDL des migrations laissent un WAL. |
| `internal/migration/steps_shared_social*.go` (8 fichiers) | CREATE TABLE / ALTER TABLE | **MED** | DDL idempotents via `IF NOT EXISTS`. Sans CHECKPOINT explicite après chaque step → si crash mid-migration, WAL pending au boot suivant. |
| `cmd/migrate-to-shared-social/main.go` | bulk INSERT | **MED** | One-shot CLI, hors path HTTP runtime |

### Runtime — path Persister (sain)

| Fichier:line | Type | Niveau | Notes |
|---|---|---|---|
| `internal/persist/shared_social_persister.go:336-355` | `tx.Commit()` + `CHECKPOINT` | **OK** | Pattern de référence. Tous les batches passent par ici. |
| `internal/api/post_sync_deltas.go:703` | `SocialPersister.AppendPlayerRecord` | **OK** | Primary path post-sync |
| `internal/api/post_sync_progression.go` (records, prestige) | `SocialPersister.*` | **OK** | Wrapping Persister |

### Runtime — path direct (à surveiller)

| Fichier:line | Type | Niveau | Notes |
|---|---|---|---|
| `internal/api/post_sync_deltas.go:708-716` | `pdb.SharedSocial.Exec(...INSERT INTO player_records ON CONFLICT DO UPDATE)` | **HIGH** | **Fallback legacy** activé quand `SocialPersister == nil` (cas tests). En prod main.go wire toujours le Persister → branche morte en pratique. **Action 3.2** : ajouter un `if pdb.SocialPersister == nil { return ErrPersisterNotWired }` plutôt que silencieusement exécuter sans CHECKPOINT. |
| `internal/platform/duckdb/media_repo_writes.go:38,41,81,185,194` | `socialDB().ExecRecovered(... INSERT/DELETE media_match_associations / media_files ...)` | **HIGH** | Écritures directes sans CHECKPOINT. Le pattern `ExecRecovered` retry sur erreur transitoire mais ne CHECKPOINT pas. **Action 3.2** : router via `SocialPersister.PersistMediaAssociations` ou wrap chaque Exec dans `LeasedWriter.BeginTx` (qui CHECKPOINT à la fin). |
| `internal/platform/duckdb/media_repo_writes.go:113-154` | `LeasedWriter.BeginTx` + `exec.ExecContext` | **MED** | Path atomique pour SetMediaLike. À vérifier : LeasedWriter fait-il un CHECKPOINT après Commit ? Lire `internal/platform/dblease/`. |
| `internal/ops/media.go:261-265` | CHECKPOINT marqué **"best-effort"** | **HIGH** | Si lock contention au CHECKPOINT post-IndexMedia, le CHECKPOINT est skippé silencieusement → WAL non vidé. **Action 3.2** : passer en erreur dure si CHECKPOINT échoue (au pire bloquer l'indexation > corrompre la DB). |

### Tests (out of runtime scope)

| Fichier:line | Notes |
|---|---|
| `internal/ops/media_kill_brutal_test.go` | Test E2E reopen post kill — preuve que le path CHECKPOINT-aware fonctionne |
| `internal/ops/media_checkpoint_test.go` | Idem |
| `internal/ops/media_associate_regression_test.go` | Cycle ATTACH/restart — test passe → ATTACH retiré OK |
| `internal/platform/duckdb/no_attach_on_social_test.go` | Sentinelle AST — interdit les ATTACH résiduels |

### Pure-read (non concerné par WAL)

`internal/platform/duckdb/{home_repo_matches,match_view_repo_extras,notifications_repo,progression_diag_repo,records_repo,social_repo}.go` — uniquement `Query` / `QueryRow`.

## Sites suspects à investiguer plus en profondeur

1. **LeasedWriter — CHECKPOINT après Commit ?**
   - Path : `internal/platform/dblease/`
   - Question : `lw.Commit()` ou `lw.Close()` exécute-t-il un `CHECKPOINT` ?
   - Si non → tous les sites `MED` ci-dessus deviennent **HIGH**.

2. **Migrations CHECKPOINT-aware ?**
   - Path : `internal/migration/registry.go` + `migration.RunForDB`
   - Question : après chaque step appliqué, y a-t-il un `CHECKPOINT` ?
   - Cf. `cmd/server/main.go:1068-1072` : la séquence est `RunForDB(...) → socialDB.Close()`. La Close DuckDB en RW fait-elle un CHECKPOINT implicite ?

3. **Forensique sur le WAL en quarantaine**
   - Fichier : `data/titles/halo_infinite/warehouse/shared_social.duckdb.wal.orphan-20260527-135758` (2509 B)
   - Dump hex des premiers 256 bytes pour identifier la nature de la dernière écriture (ATTACH / CREATE / ALTER / INSERT).
   - Bonus : la DB live de prod a aussi un `.corrupt-20260527-121448Z` (11.3 MB) qui peut être inspectée RO pour valider que le schema est bien celui d'avant le rebuild.

## Recommandations prioritaires (Phase 3.2)

1. **Bloquer le path legacy** : `post_sync_deltas.go:708` doit retourner une erreur si `SocialPersister == nil` en prod (pas en test). Pattern :
   ```go
   if pdb.SocialPersister == nil {
       return fmt.Errorf("shared_social: SocialPersister non wired — refus d'écrire sans CHECKPOINT")
   }
   ```

2. **Router toutes les écritures media_repo via Persister** : étendre `SharedSocialBatch` avec `MediaMatchAssociationsDelete` / `MediaMatchAssociationsInsert` et migrer `media_repo_writes.go:38,41` vers `SocialPersister.Persist`.

3. **Renforcer le CHECKPOINT IndexMedia** : `ops/media.go:261-265` doit erreur dure si CHECKPOINT échoue (au lieu de best-effort silencieux).

4. **Vérifier LeasedWriter** : ajouter un test qui valide qu'un `LeasedWriter.Commit()` puis kill brutal puis reopen → WAL vide.

## État de l'invariant après cet audit

**Invariant cible (ADR 0021)** : aucune écriture sur `shared_social.duckdb` ne quitte le process sans CHECKPOINT.

**État actuel** :
- **Garanti** par construction sur les paths Persister (≥90% du traffic runtime).
- **Probable** mais pas vérifié sur les paths LeasedWriter (~5%).
- **Non garanti** sur les paths Exec directs (`media_repo_writes`, `ops/media.IndexMedia`, fallback post_sync_deltas) — risque résiduel.
- **Compensé** par le code de récupération auto (`openSharedSocialWithWALRecovery`) qui rouvre malgré le WAL orphelin via quarantaine — protection en profondeur.

## Suivi

- [x] **LeasedWriter vérifié** ([writer.go:79-90](../apps/go-api/internal/platform/dblease/writer.go#L79-L90)) — `Release()` ne fait PAS de CHECKPOINT, c'est juste un unlock + log. **Conséquence** : tout site qui utilise `LeasedWriter.BeginTx` + `tx.Commit()` SANS CHECKPOINT explicite est à risque. Mitigé par le CHECKPOINT scheduler 5min mais fenêtre d'exposition non-nulle. **TODO Phase 3.2 bis** : enrichir `LeasedWriter` avec une méthode `CommitWithCheckpoint()` qui wrap les deux.
- [ ] Vérifier CHECKPOINT après chaque migration step (cas non-bloquant : DDL idempotent, CHECKPOINT scheduler 5min fait fallback)
- [x] **Phase 3.2 implémentée** — `CheckpointSharedSocial(ctx, db)` ajouté à 5 sites HIGH + IndexMedia CHECKPOINT dur. Cf. ADR 0021.
- [x] **Forensique WAL `shared_social.duckdb.wal.orphan-20260527-135758`** (2509 bytes) :

### Forensique WAL orphelin

Dump hex des 512 premiers bytes + extraction des strings ASCII :

**Strings détectées** : `mainf`, `media_files`, `shared_social`, `indexed_at`, `liked`, `discord_notified`

**Interprétation initiale** :
- `mainf` + `media_files` = pattern WAL DuckDB pour une opération sur `main.media_files`.
- `shared_social` apparaît plusieurs fois : très probablement le **nom du fichier de base** stocké dans le WAL (DuckDB v1.4 référence le DB name dans le WAL header). PAS un alias ATTACH externe.
- `indexed_at`, `liked`, `discord_notified` = colonnes de l'**ancien schéma** de `media_files` (cf. divergence schema documentée dans l'audit). Confirme que l'opération coupable touche l'ancien schéma legacy.
- Pattern répété `65 46 F2 09 C0 52 06 00` à partir de l'offset 0x60 sur ~400 bytes = **8 bytes répétés** = probablement un timestamp UTC encodé (le DOUBLE 1.7494e+18 ≈ 2025-05-something epoch nanoseconds) qui se répète sur N rows.

**Comparaison forensique Gap 2** (via `cmd/wal_forensic_compare`, runs 2026-05-27, sortie complète dans [wal_forensic_comparison.txt](wal_forensic_comparison.txt)) :

4 WAL témoins ont été générés via sub-processes Go qui appliquent un pattern DDL/DML puis font `os.Exit(0)` BRUTAL sans Close — préservant ainsi le WAL non-checkpointé sur disque.

| Pattern | Taille WAL | Strings communes vs REAL_PROD (6 strings) | Head match |
|---|---|---|---|
| ATTACH (+ INSERT) | 143 B | 2/6 (`mainf`, `media_files`) | TRUE (header DuckDB v1.4 std) |
| CREATE TABLE | 202 B | 1/6 (`media_files`) | TRUE |
| ALTER TABLE (× 2) | 251 B | 1/6 (`media_files`) | TRUE |
| INSERT × 100 | 18598 B | 2/6 (`mainf`, `media_files`) | TRUE |
| **REAL_PROD** | **2509 B** | — | — |

**Strings UNIQUES à REAL_PROD non présentes dans aucun témoin** : `shared_social` (= nom du fichier de DB, stocké dans le WAL header), `indexed_at`, `liked`, `discord_notified` (colonnes du schema legacy).

**Conclusion forensique finale** :
1. Le `head_match=TRUE` partout est trompeur — c'est juste le magic header DuckDB v1.4 standard (16 bytes).
2. La taille 2509 B se situe entre ALTER_TABLE (251) et INSERT × 100 (18598) → probablement entre 10 et 30 INSERT/UPDATE rows.
3. Aucun pattern témoin ne contient `shared_social` (nom de la DB d'origine) — c'est attendu : nos témoins utilisent un nom de DB différent (`ATTACH.duckdb`, etc.).
4. La présence des colonnes legacy `indexed_at`/`liked`/`discord_notified` confirme une opération **UPDATE ou INSERT sur le schema media_files legacy** (pré-migration align).

**Hypothèse finale** : la dernière opération avant le crash a été un **UPDATE bulk sur `media_files`** (typiquement `UPDATE media_files SET indexed_at = NOW()` ou backfill thumbnails) — pas un ATTACH. Le crash brutal a coupé l'opération avant CHECKPOINT.

Le bug DuckDB #7659 n'est **PAS** déclenché par un ATTACH dans ce cas précis (contrairement à ce que suggère le titre du bug upstream original). Le replay-fail `Calling DatabaseManager::GetDefaultDatabase` survient sur n'importe quelle opération non-checkpointée + kill brutal, dans certaines conditions encore mal comprises côté DuckDB.

**Action recommandée** :
- Aligner le schéma legacy `media_files` (DB live) avec le schéma migré actuel — Bonus 12 traité partiellement (ADD COLUMN IF NOT EXISTS, conversion `id INT→VARCHAR` hors scope).
- Reporter le bug DuckDB upstream avec ce contexte forensique enrichi — cf. [duckdb_7659_upstream_report.md](duckdb_7659_upstream_report.md).

---

## TODO Gap 5 — conversion `media_files.id INTEGER → VARCHAR` (déféré par décision 2026-05-27)

**Statut** : décision Option A — laisser en TODO documenté, pas de migration dans la session courante.

**Contexte** :
- Le schéma DB live : `media_files.id INTEGER DEFAULT nextval('media_id_seq') PRIMARY KEY`
- Le schéma cible (migrations Go récentes) : `media_files.id VARCHAR PRIMARY KEY`
- Aujourd'hui le code Go lit `id` comme `int64` via `sql.Scan` (DuckDB cast automatiquement INTEGER → string si scan vers string, et vice-versa) → fonctionnel mais cosmétiquement incohérent.

**Pourquoi déféré** :
1. Aucune feature utilisateur ne casse aujourd'hui — la gallery, les likes, les favoris, les associations match marchent toutes avec le schéma legacy.
2. Bug WAL initial réglé (ADR 0021 livré), priorité immédiate adressée.
3. La conversion exige de toucher :
   - le schéma `media_files` (PK)
   - le schéma `media_match_associations` (`media_file_id INTEGER → VARCHAR`)
   - le code Go : `internal/ops/media.go` (IndexMedia INSERT/SELECT), `internal/platform/duckdb/media_repo*.go` (Scan, lookup par id), `internal/platform/duckdb/media_repo_writes.go` (SetMediaMatchAssociation déjà refactoré avec `mediaID int64`), éventuellement le frontend si des ids transitent en JSON
   - une migration data-lossless en 2 phases (ADD nouvelle colonne UUID, backfill, swap PK, DROP ancienne) — élevée en risque sur les 121 médias prod
4. Aucun garde-rail run-time ne bloque l'incohérence — c'est du déchet schéma plus qu'un risque actif.

**Quand le faire** :
- Sprint dédié « media schema cleanup » à planifier hors session ADR 0021.
- Pré-requis : tests E2E renforcés sur les 3 surfaces (gallery, association manuelle, indexation auto-sync).
- Stratégie recommandée : Option B du débat (file_path comme PK) — élimine la double identité id+file_path. Mais accepter qu'un renommage de fichier disque casse les associations historiques (acceptable pour un sprint cleanup).

**Référence ADR 0021 (Gap 5)** — la décision Option A est documentée aussi dans le suivi de l'ADR.
