# ADR 0013 — Sérialisation des écritures DuckDB par lease

**Date** : 2026-05-05
**Statut** : Accepté
**Branche** : `refactor/leased-writer-enforcement` (commits 0-8)

## Contexte

L'API Go fait tourner en parallèle dans le même processus :
- un sync engine (ingestion Halo) qui écrit dans `stats.duckdb` (par joueur)
  et les DBs partagées (`shared_social`, `shared_matches_v2`, `metadata`) ;
- des handlers HTTP (Prestige, Notifications, Social, Media) qui écrivent
  dans les mêmes fichiers.

Avant ce refactor, seul le sync engine acquérait un lease via
`sync.AcquireLeaseCtx(ctx, path)` (un `sync.Mutex` indexé par chemin de
fichier). Les handlers HTTP écrivaient directement via leurs repositories,
sans coordination explicite. C'était une convention "savoir tribal" non
matérialisée dans le type system — et les handlers Prestige (ajoutés plus
tard) ne la suivaient pas.

**Risque concret (P1)** : pendant un sync long (~30s en delta, plusieurs
minutes en full), un `POST /challenges` pouvait :
- déclencher un lock DuckDB-level (autre process ou pool de connexions),
- entrer en course avec les écritures `match_participants` du sync,
- échouer en HTTP 500 ("database is locked") côté driver.

**P2** (notifications) et **P3** (media likes / favoris match) : aucune
garantie d'atomicité entre les paires d'écritures (`media_files.liked` vs
`media_likes`).

## Décision

**Matérialiser le lease comme un type `*LeasedWriter` et router toutes les
écritures HTTP via lui — par pattern Wrapper ou Option — sans modifier les
interfaces existantes des repositories.**

### Architecture

Acquisition au-dessus du repo : le service acquiert le `*LeasedWriter`
(timeout court côté HTTP), defer Release(), puis délègue au repo dont la
signature reste inchangée. Le mutex partagé `dblease.leaseMutex(path)`
sérialise toutes les écritures sur le même chemin de fichier — sync engine
inclus, qui continue d'utiliser l'API legacy `sync.AcquireLeaseCtx`.

Deux patterns coexistent selon la surface API du service :

1. **Wrapper** (commits 2-3, Prestige) : couche API qui wrappe
   `prestige.Service`. Justifié pour les services à grande surface (16
   méthodes — un wrapper évite la duplication d'options).

2. **Option** (commits 4-6, Notifications/Social/Media) : option
   `WithWriterAcquirer(...)` au constructeur. Justifié pour les services à
   petite surface — l'acquéreur est invoqué en interne avant chaque write.

### Type system

- `port.DBExecutor` (Exec / Query / QueryRow) — satisfait par `*sql.DB` ET
  `*sql.Tx`.
- `port.DBWriter` (`DBExecutor + BeginTx`) — satisfait par `*sql.DB` et
  `*dblease.LeasedWriter`. Pas par `*sql.Tx` (vérification compile-time
  dans `writer_test.go`).

### Atomicité (commit 6)

Les likes média (P3) ouvrent une transaction via
`tx := writer.BeginTx(ctx, nil)` puis appellent
`repo.SetMediaLikeAtomic(ctx, tx, ...)` qui exécute les deux SQL via le
même `*sql.Tx`. Rollback si l'un échoue.

### Observabilité

Métriques `expvar` exposées sur `/debug/vars`, **cardinalité bornée** par
`Kind` (player / shared_matches / shared_social / metadata) — pas par chemin
individuel (sinon explosion en multi-user, cf. ADR-0009).

## Alternatives rejetées

- **A. Garantie compile-time stricte par cascade de signatures** : modifier
  chaque méthode write de repo pour qu'elle prenne `port.DBExecutor` —
  cascaderait dans 5+ mocks et 146 tests Prestige. Coût trop élevé pour le
  bénéfice. La garantie compile-time est délivrée plus tard via le lint
  script du commit 8.
- **B. Réentrance via token de contexte** : aurait permis au hook Prestige
  d'ignorer le lease déjà tenu par le sync engine. Mais propagation de
  tokens via contexte plus dure à debugger qu'une signature explicite.
- **C. Single-writer goroutine** : aurait éliminé le lease via une
  goroutine unique par DB. Perd la propagation synchrone d'erreurs HTTP.

## Conséquences

### Positives

- P1 (Prestige), P2 (Notifications), P3 (atomicité Media) tous résolus.
- Handlers HTTP retournent `503 + Retry-After: 5` proprement pendant un
  sync, au lieu de 500 ou état corrompu.
- Tests existants préservés bit-à-bit (pas de modification d'interfaces
  partagées).
- Intégration sync engine automatique via la map de mutex partagée (aucun
  changement aux 11 sites `AcquireLeaseCtx` du sync).
- Observabilité : métriques expvar à cardinalité bornée + helper anti-fuite
  (`dblease.AssertNoLeasedWriters(t)`).

### Négatives / dette

- **Garantie compile-time non-stricte** : un futur dev pourrait ajouter un
  nouveau service qui écrit via `db.Exec` sans acquérir de lease. Mitigé
  par l'ADR + le script CI grep du commit 8.
- **Sync engine encore en API legacy** : `sync.AcquireLeaseCtx` ne
  contribue pas aux compteurs `dblease_acquire_total{kind}`. Migration 1:1
  faisable mais touche 11 sites — différée pour ne pas casser ~63 tests.
- **Tests atomic cgo-only** : nécessitent une vraie DuckDB `:memory:` —
  documentés comme tests d'intégration en attente.

## Mise en œuvre

8 commits séquentiels sur `refactor/leased-writer-enforcement` (cf. la
version anglaise pour le détail commit par commit). 41 nouveaux tests + 3
benchmarks ajoutés. Aucun test existant supprimé.

## Références

- Plan complet : `.ai/PLAN_DB_WRITE_CONCURRENCY.md` v3
- ADR-0009 (expvar monitoring multi-user) — cardinalité bornée
- ADR-0005 (Prestige phased activation) — contexte d'urgence P1
- `internal/sync/lease.go` — invariant deadlock-free documenté
- [apps/go-api/scripts/check_lease_enforcement.sh](../../../apps/go-api/scripts/check_lease_enforcement.sh) — script CI grep
