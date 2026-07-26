# ADR 0021 — Récupération automatique d'un WAL orphelin sur shared_social.duckdb

> **Note de renumérotation** : une collision de numéro 0021 a existé avec `template-synthesis`. Résolue : le template-synthesis a été renuméroté **0028** (`0028-template-synthesis.md`). Ce document conserve le numéro **0021**.

**Statut** : Accepté — 2026-05-27
**Stakeholders** : pool DuckDB (Guillaume), runtime API Go
**Liens** : ADR 0016 (no-ATTACH cross-DB), ADR 0019 (Collect→Persist), ADR 0022 (SocialPersister)

## Contexte

Le 27/05/2026 à 13:38, l'utilisateur signale que la page Media est vide pour tous les comptes (Chocoboflor, Madina97294, JGtm, XxDaemonGamerxX). Les logs montrent en boucle :

```
[WARN]  pool: ouverture SharedSocial échouée (dégradation: socialDB=nil)
[ERROR] err="INTERNAL Error: Failure while replaying WAL file
        ...shared_social.duckdb.wal: Calling DatabaseManager::GetDefaultDatabase"
```

Diagnostic : bug DuckDB upstream #7659 (`WAL Replay fails when attach alias changes`). Un ATTACH/DDL legacy s'est écrit dans `shared_social.duckdb.wal` sans CHECKPOINT. Le header de la base principale marque "WAL replay needed" mais le replay échoue par assertion sur `DatabaseManager::GetDefaultDatabase`. DuckDB refuse l'open RW. Cascade :

`socialDB = nil` → `MediaRepo.loadMediaCandidates` retourne `(nil, nil)` ([media_repo_q37_pipeline.go:78-82](../../../apps/go-api/internal/platform/duckdb/media_repo_q37_pipeline.go#L78-L82)) → galerie vide pour TOUS les joueurs, pas seulement les coéquipiers.

Le code a déjà été audité : depuis 2026-05-25, plus aucun ATTACH n'est exécuté sur `shared_social.duckdb` RW dans le runtime. Pourtant un WAL non-rejouable est réapparu — soit un site d'écriture non-checkpointed reste à identifier (cf. [audit](../../../.ai/V7/audit_shared_social_writes_2026-05-27.md)), soit une migration passée laisse un état latent dans le header.

## Décision

Mettre en place une **récupération automatique en deux temps** au boot du pool :

### 1. Quarantaine + retry dans le process

Dans `openPlayerDB`, l'ouverture de `shared_social.duckdb` passe par `openSharedSocialWithWALRecovery` ([pool_shared_social_recovery.go](../../../apps/go-api/internal/platform/duckdb/pool_shared_social_recovery.go)) qui :

1. Tente `OpenReadWriteShared`.
2. Si l'erreur contient `"Failure while replaying WAL file"` :
   - Renomme atomiquement `<path>.wal` en `<path>.wal.orphan-<RFC3339-utc>` (preuve forensique).
   - Incrémente le compteur expvar `levelup.wal_orphan_quarantine.shared_social`.
   - Émet un `slog.ErrorContext` (PAS Warn — événement à monitorer).
   - Réessaie **une seule fois** `OpenReadWriteShared`.
3. Si le retry échoue → `socialDB = nil` (dégradation graceful déjà existante), avec `slog.Error` pointant vers `cmd/rebuild_shared_social` pour intervention manuelle.

**Pas de boucle, pas de delete, pas de modification du `.duckdb`** — le code de récupération auto ne traite que le cas où la corruption est isolée dans le `.wal`.

### 2. Outil de reconstruction (cas extrême)

Quand la corruption atteint le header du fichier `.duckdb` (cas constaté le 27/05 : même après quarantaine du `.wal`, l'open RW continue à échouer), l'outil one-shot `cmd/rebuild_shared_social` ([main.go](../../../apps/go-api/cmd/rebuild_shared_social/main.go)) effectue :

1. Snapshot baseline `COUNT(*)` par table (mode RO — DuckDB ignore le replay en RO).
2. `EXPORT DATABASE` vers un répertoire temporaire (parquets + `schema.sql` + `load.sql`).
3. Renomme le fichier corrompu en `<path>.corrupt-<ts>` (preuve forensique).
4. Crée une DB vide et lance `IMPORT DATABASE` qui rejoue **le schema exporté tel quel** (pas les migrations Go — qui peuvent diverger pour cause d'historique).
5. CHECKPOINT explicite.
6. Vérification post : snapshot final vs baseline → échec si `delta != 0` sur tables non-`bak_*`.

L'outil est testé en sandbox avant production, et tous les helpers (`extractTableName`, `extractParquetPath`, `diffCounts`) ont des tests unitaires ([parser_test.go](../../../apps/go-api/cmd/rebuild_shared_social/parser_test.go)).

### 3. Helper `CheckpointSharedSocial` + sites HIGH refacto (Phase 3.2)

Pour éliminer la **source** des WAL orphelins (pas seulement compenser après coup), Phase 3.2 ajoute un helper exporté `CheckpointSharedSocial(ctx, db) error` ([pool_shared_social_recovery.go](../../../apps/go-api/internal/platform/duckdb/pool_shared_social_recovery.go)) et l'applique à tous les sites d'écriture directe identifiés dans l'audit :

- `media_repo_writes.go SetMediaMatchAssociation` : CHECKPOINT après le DELETE + INSERT.
- `media_repo_writes.go SetMediaLike` (legacy) : CHECKPOINT après UPDATE.
- `media_repo_writes.go ToggleSharedLike` (fallback) : CHECKPOINT après INSERT/DELETE.
- `post_sync_deltas.go upsertPlayerRecord` (legacy) : CHECKPOINT après l'UPSERT.
- `ops/media.go IndexMedia` : CHECKPOINT en **erreur dure** (return error si échec) au lieu de best-effort.

La fenêtre d'exposition au bug WAL orphelin passe de 5 minutes (scheduler périodique) à 0 seconde sur tous ces sites.

## Conséquences

### Positives

- **Idempotent au boot** : un crash brutal Windows qui laisse un WAL orphelin ne casse plus la galerie média au reboot suivant. Le pool quarantine et continue.
- **Observabilité** : compteur expvar + slog ERROR sur chaque quarantaine → alerting possible.
- **Forensique** : les fichiers `.wal.orphan-*` et `.duckdb.corrupt-*` restent sur disque pour analyse post-incident.
- **Pas de perte de données** : le WAL orphelin est par définition non-rejouable, donc son contenu est déjà inaccessible. Les écritures validées (likes, favoris, médias) sont dans le fichier principal grâce au CHECKPOINT systématique du `SocialPersister` ([shared_social_persister.go:283-285](../../../apps/go-api/internal/persist/shared_social_persister.go#L283-L285)).
- **Compatibilité** : la signature publique de `openPlayerDB` est inchangée. Seul le bloc d'ouverture social est refactoré.

### Négatives

- **Risque résiduel non-éliminé** : les sites d'écriture directs (cf. [audit Phase 3.1](../../../.ai/V7/audit_shared_social_writes_2026-05-27.md)) peuvent toujours produire un WAL non-checkpointed entre deux CHECKPOINT. La recovery auto compense mais la source ne disparaît que si on bascule tout sur SocialPersister (refacto plus large à venir).
- **Cas extrême non-récupérable dans le process** : si le `.duckdb` header est corrompu, seul `cmd/rebuild_shared_social` peut récupérer. Intervention manuelle requise.
- **Compatibilité IMPORT DATABASE** : si DuckDB upstream change le format de `schema.sql` / `load.sql`, le rebuild peut échouer. Atténué par les tests sandbox.

### Invariant à respecter pour préserver le bénéfice

> **Aucune écriture sur `shared_social.duckdb` ne quitte le processus sans CHECKPOINT.**

Garde-rails :
- Toutes les écritures **runtime** passent par `SocialPersister.Persist` qui CHECKPOINT après chaque batch.
- Les writes legacy résiduels (4 sites identifiés) appellent `CheckpointSharedSocial` après l'Exec.
- Sentinelle AST [`no_attach_on_social_test.go`](../../../apps/go-api/internal/platform/duckdb/no_attach_on_social_test.go) interdit les `ATTACH` sur la conn social (Phase 2.4 : scope élargi aux selector chains `pdb.SharedSocial.Exec`, `r.socialDB().Exec`).
- Tests `media_kill_brutal_test.go` + `media_checkpoint_test.go` + `media_associate_regression_test.go` couvrent les cycles kill brutal → reopen.
- `cmd/server/main.go:628` exécute un CHECKPOINT scheduler périodique pour défense en profondeur.
- Target Makefile `go-api-test-shared-social-gate` rejoue l'ensemble des invariants en CI.

## Alternatives considérées

### Option A — Sauter le WAL replay via un flag DuckDB

Recherche dans DuckDB v1.4 : aucun flag `?skip_wal_replay=true` ni équivalent. DuckDB refuse l'ouverture si le replay échoue.

### Option B — Patcher DuckDB upstream

Le bug #7659 est tracké côté DuckDB mais sans calendrier de fix. Notre cycle de release Go-API est incompatible avec l'attente. Récupération applicative = mitigation pragmatique.

### Option C — Désactiver complètement le WAL

`PRAGMA wal_autocheckpoint = 0` ferait du checkpointing manuel obligatoire. Trop intrusif pour le runtime (chaque transaction devrait CHECKPOINT). Rejeté.

### Option D — Détecter et supprimer le WAL plutôt que quarantine

Le delete perd définitivement la preuve forensique. La quarantaine garde un fichier audit. Choix retenu : quarantaine.

## Vérification

- Tests unitaires : `go test -race -run "TestIsWALReplayFailure_Pattern|TestQuarantineOrphanWAL|TestOpenSharedSocial|TestCheckpointSharedSocial|TestSet.*PersistsAfter|TestToggle.*PersistsAfter" ./internal/platform/duckdb/` (14 tests, 100% pass).
- Test sentinelle AST : `go test -run "TestNoATTACHOnSocialDB|TestSocialReceiverLabel|TestSentinel" ./internal/platform/duckdb/` (3 tests + 11 sous-tests, pass).
- Test outil rebuild : `go test ./cmd/rebuild_shared_social/` (parser + diff helpers, 5 tests).
- Validation sandbox manuelle : copie de la DB live, run `rebuild_shared_social`, vérifier que les 22 tables et tous les counts sont préservés.
- Validation production : rebuild exécuté 2026-05-27 14:14 sur la DB live, restart Air, snapshot post-rebuild = baseline, plus de WARN SharedSocial dans `logs/duckdb.log` depuis le boot.
- Gate Makefile : `make go-api-test-shared-social-gate` — race-clean en < 5s sur 19+ tests.
- Script validation manuelle : `scripts/verify_shared_social_recovery.ps1` — exit 0.

## Suivi

- [x] **Phase 3.2 suite** : refacto Persister fait — `SocialPersister.SetMediaMatchAssociation` + `SetMediaLiked` (TX atomique + CHECKPOINT immédiat) câblés dans `media_repo_writes.go`. Fallback legacy préservé pour tests.
- [x] **Forensique** : dump hex + comparaison comparative (Gap 2) via `cmd/wal_forensic_compare`. Conclusion : UPDATE/INSERT bulk sur schema legacy, pas un ATTACH.
- [ ] **Divergence schema `media_files.id INT → VARCHAR`** (Gap 5) : **déféré par décision 2026-05-27 (Option A)** — TODO documenté ; aucune feature ne casse aujourd'hui, à reprendre dans un sprint dédié.
- [x] **CI gate effectif** : `.github/workflows/shared-social-gate.yml` câblé + coverage ratchet (`scripts/check_coverage_ratchet.sh`).
- [x] **Issue upstream DuckDB #7659** (Gap 6) : comment posté 2026-05-27 sur [#19712](https://github.com/duckdb/duckdb/issues/19712#issuecomment-4555562539) (le mainteneur demandait un `reproducible example` — fourni).
- [x] **Hook local** : gate joué au **pre-push** par lefthook (`lefthook.yml`, commande `shared-social-gate` → `scripts/git-hooks/lefthook/shared-social-gate.sh` → `make go-api-test-shared-social-gate`). Installation : `make install-git-hooks` (= `lefthook install`), une fois par dev après clone. **Correctif 2026-07-26** : l'ancien hook `pre-commit` copié dans `.git/hooks/` ne tournait plus (renommé `pre-commit.old` par lefthook) et la cible Makefile écrasait le shim lefthook ; script + `.pre-commit-config.yaml` mort supprimés.
