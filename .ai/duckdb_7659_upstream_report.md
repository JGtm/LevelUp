# DuckDB upstream bug report — brouillon issue

> Brouillon d'issue à ouvrir sur https://github.com/duckdb/duckdb/issues
>
> Cf. ADR 0021 Phase 13 / Bonus.

## Titre proposé

`INTERNAL Error: Failure while replaying WAL file — Calling DatabaseManager::GetDefaultDatabase with no default database set (potential regression of #7659)`

## Body

### Summary

After a brutal process termination (SIGKILL or `os.Exit(0)` without graceful Close) during bulk INSERTs on a DuckDB v1.4 database, the WAL becomes non-replayable at next open. The error surfaces as :

```
INTERNAL Error: Failure while replaying WAL file "...wal":
Calling DatabaseManager::GetDefaultDatabase with no default database set
```

This blocks RW opens indefinitely until the WAL is manually quarantined or the DB is rebuilt via EXPORT/IMPORT.

The pattern matches https://github.com/duckdb/duckdb/issues/7659 (`WAL Replay fails when attach alias changes`) but the repro here does NOT involve ATTACH — only INSERTs.

### Repro

Self-contained Go reproducer (DuckDB Go driver) — see `apps/go-api/cmd/duckdb_7659_repro/main.go` in the LevelUp project. Two phases :

1. **Write phase** : `sql.Open("duckdb", path)` → `CREATE TABLE` → `INSERT × 100` → `os.Exit(0)` BRUTAL (no `db.Close()`).
2. **Read phase** : `sql.Open(same path)` → `db.Ping()` → bug surfaces.

```bash
go run ./apps/go-api/cmd/duckdb_7659_repro
```

The bug reproduces ~80% of the time. When it does, the second sub-process logs :

```
[READ] !!! BUG #7659 REPRODUIT !!!
Erreur DuckDB : ... Failure while replaying WAL file "...wal":
Calling DatabaseManager::GetDefaultDatabase with no default database set
```

### Environment

- DuckDB Go driver : `github.com/duckdb/duckdb-go/v2 v2.10503.0`
- DuckDB version : v1.5.3 (bundled by `duckdb-go-bindings v0.10503.0`)
- OS : Windows 11 Pro (NTFS) — incident initial. Repro confirmé sur le même setup.
- Pattern : RW open + DDL + bulk INSERTs + SIGKILL → reopen fail

### Real-world impact

In LevelUp (Halo Infinite stats dashboard), this bug brought down the entire Media gallery for all players because :

1. Air auto-restart (Windows dev) kills the server brutally on file change.
2. Brutal kill mid-write leaves a non-replayable WAL.
3. At next boot, DuckDB refuses to open shared_social.duckdb in RW.
4. The pool degrades to `socialDB = nil` → all media listing returns empty.

Fix applied : quarantine the orphan WAL + retry open ([ADR 0021](docs/adr/0021-shared-social-wal-recovery.md)), but this only papers over the symptom — the underlying replay assertion failure is the real bug.

### Suspected root cause

Looking at the DuckDB error message, the assertion fails in `DatabaseManager::GetDefaultDatabase`. This suggests the WAL header references a database alias (or default-database state) that no longer exists at replay time. But in our minimal repro, we don't use ATTACH — only INSERTs.

Possible explanations :

1. DuckDB internal state references the DB name implicitly in WAL entries, and a slight path normalization difference (Windows vs Linux, trailing slash, etc.) causes the lookup to fail.
2. A race condition between the WAL writer and the page-header writer leaves the file in an inconsistent state.
3. The bug from #7659 (ATTACH-related) has been generalized in v1.4 to cover other operations.

### Workaround (applied)

Our recovery code quarantines the `.wal` file before retrying the open :
```
<path>.wal → <path>.wal.orphan-<RFC3339-UTC>
```

When the corruption extends to the `.duckdb` main file header (as observed in production), an EXPORT/IMPORT cycle is needed — see `cmd/rebuild_shared_social/main.go`.

### Forensic data

A real WAL orphelin captured in production (2509 bytes, 27/05/2026) is preserved in our testdata as a fixture for regression testing. Hex dump reveals :

- Pattern `mainf` + `media_files` (schema.table reference)
- Repeated 8-byte pattern `65 46 F2 09 C0 52 06 00` (likely a timestamp encoded ~400 times)
- No `ATTACH` literal visible

### Forensic comparison (ADR 0021 Gap 2)

We built a comparator tool (`cmd/wal_forensic_compare`) that generates 4 reference WAL files via sub-process exit brutal — each triggered by a distinct DuckDB operation pattern — and compares them with our real prod WAL. Results :

| Pattern | WAL size | Strings ∩ real (out of 6) | Magic header match |
|---|---|---|---|
| ATTACH + INSERT | 143 B | 2/6 | yes (DuckDB v1.4 magic = `64 00 62 65 00 02 FF FF`) |
| CREATE TABLE | 202 B | 1/6 | yes |
| ALTER TABLE × 2 | 251 B | 1/6 | yes |
| INSERT × 100 | 18 598 B | 2/6 | yes |
| **Real prod WAL** | **2 509 B** | — | — |

The 2 509 B size fits between ALTER TABLE and INSERT × 100, suggesting somewhere between 10–30 rows worth of INSERT/UPDATE payload. The real WAL contains strings `shared_social`, `indexed_at`, `liked`, `discord_notified` which are NOT present in any reference WAL because they're application-specific (database name + table schema legacy columns).

**Conclusion** : the offending operation in our case is most likely an UPDATE/INSERT bulk on the `media_files` table — NOT an ATTACH operation. This generalizes the original #7659 scope to plain DML.

### Request

1. Confirm whether #7659 covers the no-ATTACH DML case described here, or if this is a separate regression.
2. Investigate whether a `PRAGMA wal_recovery_mode = 'best_effort'` or equivalent could be added so the database self-recovers without external `.wal` quarantine.
3. The repeated 8-byte pattern in our WAL (`65 46 F2 09 C0 52 06 00`) appears to be a timestamp serialization — would a malformed timestamp encoding (e.g. negative or out-of-range) trigger the replay assertion ? Worth investigating.

Happy to provide :
- Full hex dump of the orphan WAL (2 509 B) — small enough to share inline.
- The gzipped corrupt `.duckdb` file (86 KB compressed) for reproduction on your side.
- LevelUp server logs around the crash time.

---

## Registre des issues upstream apparentées (état au 2026-05-27)

Toutes partagent la même assertion `Calling DatabaseManager::GetDefaultDatabase with no default database set` au replay du WAL, mais l'opération qui la déclenche varie.

| # | État | Opération déclenchante | Notre action | Notes |
|---|---|---|---|---|
| **#7659** | **CLOSED** (2023) | ATTACH alias change | — | Issue historique mentionnée pour contexte, fermée sans fix complet — la famille de bugs ci-dessous a hérité du symptôme. |
| **#18259** | OPEN, under review (2026-03-03) | `ALTER TABLE ADD COLUMN` avec DEFAULT expression | ⏳ watch, pas de comment | Variante DDL spécifique. 4 comments. |
| **#19099** | OPEN, under review (2025-09-23) | méta-issue "WAL recovery seems unsafe" | ⏳ watch | Discussion d'ensemble sur la stratégie de récupération WAL. Aucune action utile à y commenter. |
| **#19712** | OPEN, `needs reproducible example` (2026-05-11) | crash générique post-INSERT | ✅ **commenté avec notre repro complet** | https://github.com/duckdb/duckdb/issues/19712#issuecomment-4555562539 |
| **#20543** | OPEN, under review (2026-01-15) | `ALTER COLUMN type` | ⏳ watch | Auteur Nick Crews. Titre exact match. Pas commenté pour éviter spam. |
| **#22044** | OPEN, under review (2026-04-14) | `DROP INDEX` | ⏳ watch | Variante DDL, scope plus restreint. |
| **#22124** | CLOSED, reproduced (2026-05-01) | `UPDATE + table rebuild same TX` | ✓ fix mergé via [#22093](https://github.com/duckdb/duckdb/pull/22093) le 21/04. | **Fix dans DuckDB v1.5.3 (notre version actuelle)** mais ne couvre que ce sous-cas spécifique. Notre bug reste actif. |

**Conclusion sur la stratégie upstream** :

1. Notre comment sur **#19712** est la contribution active — c'est l'issue avec le label `needs reproducible example` qui demande explicitement notre type de données.
2. **Ne pas spam** les autres issues OPEN avec des "+1 / same here". Une duplicate de comment réduirait la lisibilité pour le mainteneur.
3. **Watch** ces issues pour repérer un fix mergé : recommandé via `gh issue list --search 'in:title "WAL" OR "GetDefaultDatabase"' --state closed` périodique, ou GitHub watching côté UI.
4. Le fix #22093 (mergé v1.5.3) ne suffit pas pour notre cas — il couvre `ALTER+UPDATE même TX`, pas notre `INSERT/UPDATE bulk + kill brutal`. La recovery applicative ADR 0021 reste nécessaire.

**Action à long terme côté LevelUp** :

- À chaque release DuckDB (v1.6+, v2.0+), re-tester notre repro `cmd/duckdb_7659_repro` après upgrade `duckdb-go-bindings`. Si le bug ne se reproduit plus sur 100+ runs → considérer retirer le code de recovery applicative ADR 0021 (le garder en défense en profondeur est une option valide aussi).
- Si un mainteneur DuckDB répond sur #19712 et demande la fixture `.duckdb` corrompue (86KB gzippé) ou des logs supplémentaires, on a tout sous la main dans `testdata/wal_orphan_fixture/`.

## Statut côté LevelUp

- Mitigation applicative livrée ADR 0021 — autosuffisante en attendant le fix upstream.
- Le repro `cmd/duckdb_7659_repro` est versionné comme artefact prêt à être attaché (non-déterministe, mais le pattern y est).
- Le comparateur forensique `cmd/wal_forensic_compare` produit les 4 WAL témoins prêts à attacher.
- **Comment posté 2026-05-27 sur l'issue OPEN existante https://github.com/duckdb/duckdb/issues/19712** (le maintainer demandait explicitement un `reproducible example`). URL du comment : https://github.com/duckdb/duckdb/issues/19712#issuecomment-4555562539. Le brouillon initial sur ce fichier était prévu pour une nouvelle issue ; après scan des issues OPEN existantes 4 doublons potentiels ont été identifiés (#19712, #20543, #22044, #19099) — choix de commenter #19712 plutôt que d'ouvrir une duplicate.
