# ADR 0021 — Recovery automatique d'un WAL orphelin sur shared_social.duckdb

> **Note de renumérotation** : une collision de numéro 0021 a existé avec `template-synthesis`. Résolue : le template-synthesis a été renuméroté **0028** (`0028-template-synthesis.md`). Ce document conserve le numéro **0021** ; toutes les références code « ADR 0021 » relatives au WAL recovery / SocialPersister / shared_social désignent ce document.

**Status** : Accepted — 2026-05-27
**Stakeholders** : pool DuckDB (Guillaume), runtime API Go
**Related** : ADR 0016 (no-ATTACH cross-DB), ADR 0019 (Collect→Persist), ADR 0022 (SocialPersister)

## Contexte

Le 27/05/2026 à 13:38, l'utilisateur signale que la page Media est vide pour tous les comptes (Chocoboflor, Madina97294, JGtm, XxDaemonGamerxX). Les logs montrent en boucle :

```
[WARN]  pool: ouverture SharedSocial échouée (dégradation: socialDB=nil)
[ERROR] err="INTERNAL Error: Failure while replaying WAL file
        ...shared_social.duckdb.wal: Calling DatabaseManager::GetDefaultDatabase"
```

Diagnostic : bug DuckDB upstream #7659 (`WAL Replay fails when attach alias changes`). Un ATTACH/DDL legacy s'est écrit dans `shared_social.duckdb.wal` sans CHECKPOINT. Le header de la DB principale marque "WAL replay needed" mais le replay assertion-fail dans `DatabaseManager::GetDefaultDatabase`. DuckDB refuse l'open RW. Cascade :

`socialDB = nil` → `MediaRepo.loadMediaCandidates` retourne `(nil, nil)` ([media_repo_q37_pipeline.go:78-82](../../apps/go-api/internal/platform/duckdb/media_repo_q37_pipeline.go#L78-L82)) → galerie vide pour TOUS les joueurs, pas seulement les coéquipiers.

Le code a déjà été audité : depuis 2026-05-25, aucun ATTACH n'est exécuté sur `shared_social.duckdb` RW dans le runtime. Pourtant un WAL non-rejouable est réapparu — preuve que **soit** un site d'écriture non-checkpointed reste à identifier (cf. [audit](../../.ai/V7/audit_shared_social_writes_2026-05-27.md)), **soit** une migration past laisse un état latent dans le header.

## Décision

Mettre en place une **récupération automatique en deux temps** au boot du pool :

### 1. Quarantaine + retry en-process

Dans `openPlayerDB`, l'ouverture de `shared_social.duckdb` passe par `openSharedSocialWithWALRecovery` ([pool_shared_social_recovery.go](../../apps/go-api/internal/platform/duckdb/pool_shared_social_recovery.go)) qui :

1. Tente `OpenReadWriteShared`.
2. Si l'erreur contient `"Failure while replaying WAL file"` :
   - Renomme atomiquement `<path>.wal` en `<path>.wal.orphan-<RFC3339-utc>` (preuve forensique).
   - Incrémente le compteur expvar `levelup.wal_orphan_quarantine.shared_social`.
   - Émet un `slog.ErrorContext` (PAS Warn — événement à alerting).
   - Retry **une seule fois** `OpenReadWriteShared`.
3. Si retry échoue → `socialDB = nil` (dégradation graceful pré-existante), avec `slog.Error` pointant vers `cmd/rebuild_shared_social` pour intervention manuelle.

**Pas de boucle, pas de delete, pas de modification du `.duckdb`** — le code de récupération auto ne traite que le cas où la corruption est isolée dans le `.wal`.

### 2. Outil de reconstruction (cas extrême)

Quand la corruption atteint le header du fichier `.duckdb` (cas constaté le 27/05 : même après quarantaine du `.wal`, l'open RW continue à échouer), l'outil one-shot `cmd/rebuild_shared_social` ([main.go](../../apps/go-api/cmd/rebuild_shared_social/main.go)) effectue :

1. Snapshot baseline `COUNT(*)` par table (mode RO — DuckDB skip le replay en RO).
2. `EXPORT DATABASE` vers un répertoire temporaire (parquets + `schema.sql` + `load.sql`).
3. Renomme le fichier corrompu en `<path>.corrupt-<ts>` (preuve forensique).
4. Crée une DB vide et fait `IMPORT DATABASE` qui rejoue **le schema exporté tel quel** (pas les migrations Go — qui peuvent diverger pour cause d'historique). 
5. CHECKPOINT explicite.
6. Vérification post : snapshot final vs baseline → fail si `delta != 0` sur tables non-`bak_*`.

L'outil est testé sur sandbox avant production, et tous les helpers (`extractTableName`, `extractParquetPath`, `diffCounts`) ont des tests unitaires ([parser_test.go](../../apps/go-api/cmd/rebuild_shared_social/parser_test.go)).

## Conséquences

### Positives

- **Idempotent au boot** : un crash brutal Windows qui laisse un WAL orphelin ne casse plus la galerie média au reboot suivant. Le pool quarantine et continue.
- **Observabilité** : compteur expvar + slog ERROR sur chaque quarantaine → alerting possible.
- **Forensique** : les fichiers `.wal.orphan-*` et `.duckdb.corrupt-*` restent sur disque pour analyse post-incident.
- **Pas de perte de données** : le WAL orphelin est par définition non-rejouable, donc son contenu est déjà inaccessible. Les écritures validées (likes, favoris, médias) sont dans le fichier principal grâce au CHECKPOINT systématique du `SocialPersister` ([shared_social_persister.go:283-285](../../apps/go-api/internal/persist/shared_social_persister.go#L283-L285)).
- **Compatibilité** : la signature publique de `openPlayerDB` est inchangée. Seul le bloc d'ouverture social est refactoré.

### Négatives

- **Risque résiduel non-éliminé** : les sites d'écriture directs (cf. [audit Phase 3.1](../../.ai/V7/audit_shared_social_writes_2026-05-27.md)) peuvent toujours produire un WAL non-checkpointed. La recovery auto compense mais ne supprime pas la cause.
- **Cas extrême non-récupérable en-process** : si le `.duckdb` header est corrompu, seul `cmd/rebuild_shared_social` peut récupérer. Intervention manuelle requise.
- **Compatibilité IMPORT DATABASE** : si DuckDB upstream change le format de `schema.sql` / `load.sql`, le rebuild peut échouer. Mitigé par les tests sandbox.

### Invariant à respecter pour préserver le bénéfice

> **Aucune écriture sur `shared_social.duckdb` ne quitte le process sans CHECKPOINT.**

Garde-rails :
- Toutes les écritures **runtime** passent par `SocialPersister.Persist` qui CHECKPOINT après chaque batch.
- Sentinelle AST [`no_attach_on_social_test.go`](../../apps/go-api/internal/platform/duckdb/no_attach_on_social_test.go) interdit les `ATTACH` sur la conn social.
- Tests `media_kill_brutal_test.go` + `media_checkpoint_test.go` + `media_associate_regression_test.go` couvrent les cycles kill brutal → reopen.
- Le `cmd/server/main.go:628` exécute un CHECKPOINT scheduler périodique pour défense en profondeur.

## Alternatives considérées

### Option A — Skip le WAL replay via flag DuckDB

Recherche dans DuckDB v1.4 : aucun flag `?skip_wal_replay=true` ni équivalent. DuckDB refuse l'ouverture si le replay échoue.

### Option B — Patch DuckDB upstream

Bug #7659 est tracké côté DuckDB mais sans timeline de fix. Notre cycle de release Go-API est incompatible avec l'attente. Recovery applicative = mitigation pragmatique.

### Option C — Désactiver complètement le WAL

`PRAGMA wal_autocheckpoint = 0` ferait du checkpointing manuel obligatoire. Trop intrusif pour le runtime (chaque transaction devrait CHECKPOINT). Rejeté.

### Option D — Détecter et delete le WAL plutôt que quarantaine

Le delete perd définitivement la preuve forensique. La quarantaine garde un fichier audit. Choisi : quarantaine.

## Vérification

- Tests unitaires : `go test -race -run "TestIsWALReplayFailure_Pattern|TestQuarantineOrphanWAL|TestOpenSharedSocial" ./internal/platform/duckdb/` (9 tests, 100% pass).
- Test sentinelle AST : `go test -run TestNoATTACHOnSocialDB ./internal/platform/duckdb/` (pass).
- Test outil rebuild : `go test ./cmd/rebuild_shared_social/` (parser + diff helpers).
- Validation sandbox manuelle : copie de la DB live, run `rebuild_shared_social`, vérifier que les 22 tables et tous les counts sont préservés.
- Validation production : rebuild exécuté 2026-05-27 14:14 sur la DB live, restart Air, snapshot post-rebuild = baseline, pas de WARN SharedSocial dans `logs/duckdb.log` depuis le boot.

## Suivi

- [x] **Phase 3.2** : sites d'écriture direct refactorés — `CheckpointSharedSocial` helper appliqué aux fallbacks + nouvelles méthodes `SocialPersister.SetMediaMatchAssociation` / `SetMediaLiked` (TX atomique + CHECKPOINT immédiat).
- [x] **Forensique** : dump hex du `.wal.orphan-20260527-135758` effectué (Phase 3.3) + comparaison avec 4 WAL témoins via `cmd/wal_forensic_compare` (Gap 2). Conclusion : opération coupable = UPDATE/INSERT bulk sur le schema legacy, PAS un ATTACH.
- [ ] **Schéma divergence `media_files.id INT → VARCHAR`** (Gap 5) : **déféré par décision 2026-05-27 (Option A)** — laissé en TODO documenté dans [.ai/V7/audit_shared_social_writes_2026-05-27.md](../../.ai/V7/audit_shared_social_writes_2026-05-27.md). Aucune feature ne casse en l'état, le décalage est cosmétique. À traiter dans un sprint « media schema cleanup » séparé.
- [x] **CI** : `.github/workflows/shared-social-gate.yml` câblé + coverage ratchet via `scripts/check_coverage_ratchet.sh` (Gap 4) avec baseline versionnée dans `scripts/coverage_baseline.txt`.
- [x] **Issue upstream DuckDB #7659** (Gap 6) : comment posté 2026-05-27 sur [#19712](https://github.com/duckdb/duckdb/issues/19712#issuecomment-4555562539) (issue OPEN avec label `needs reproducible example` — nous fournissons exactement ça). Brouillon de référence dans [.ai/duckdb_7659_upstream_report.md](../../.ai/duckdb_7659_upstream_report.md).
- [x] **Hook local** (Gap 3.5) : le gate tourne au **pre-push** via lefthook — `lefthook.yml`, commande `shared-social-gate` (glob calqué sur le filtre `paths` du workflow) → `scripts/git-hooks/lefthook/shared-social-gate.sh` → `make go-api-test-shared-social-gate`. Installation : `make install-git-hooks` (= `lefthook install`), une fois après chaque clone. **Correctif 2026-07-26** : le hook `pre-commit` d'origine (`scripts/git-hooks/pre-commit`, copié dans `.git/hooks/`) ne s'exécutait plus depuis le passage à lefthook — le shim lefthook l'avait renommé en `pre-commit.old` — et la cible `install-git-hooks` écrasait ce shim. Le script et le `.pre-commit-config.yaml` mort ont été supprimés ; lefthook est le seul système de hooks.
