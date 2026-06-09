# Plan — Robustesse persist (gaps post-Phase 3 Collect→Persist)

> **Créé le** : 2026-06-09
> **Statut** : Proposé — dégraissé après analyse de recouvrement avec la convergence autonome
> **Priorité** : 🟡 Moyenne (ops/confiance), non bloquant
> **Origine backlog** : `[persist/safety] Garde-fous restants post-Phase 3` (gaps A, B, D, E, G)

## Contexte & cadrage

Audit safety du chemin `submitMatchAsBatch` (ADR 0019). Sur les 6 gaps initiaux :
- **[C] timeout par Persist call** — déjà fixé (commit 2026-05-24).
- **[F] RecoverPending au boot** — déjà câblé (`cmd/server/main.go`, 2026-06-03).
- **[D] circuit breaker** — ✅ **déjà livré** : `internal/persist/queue.go:69-79` (`consecutiveFailures` + seuil 5) + `queue_circuit_breaker_test.go` (5 tests verts). **Retiré du périmètre.**

Restent A, B, E, G — recalibrés, **mais un trou réel a été identifié** (révision 2026-06-09, cf. ci-dessous).

**Mécanismes existants** :
1. **Worker continu** (`main.go:644-647`) : persiste les nouveaux batches en flux. OK en régime normal.
2. **Convergence autonome** (`internal/sync/convergence.go`, `engine_postsync.go:73-81`) rattrape à chaque cycle sync les matchs **insérés mais non enrichis** — cas **distinct** de l'échec d'écriture persist (ne rejoue PAS un batch persist échoué).

⚠️ **Trou réel — recovery boot-only sur webapp rarement redémarrée** :
- `RecoverPending()` (re-soumission des WAL pending) n'est appelé qu'**au boot** (`main.go:652` ; cf. `queue_test.go:325` « appelé UNIQUEMENT au boot »). Un batch qui **échoue** en cours d'exécution laisse un fichier WAL **jamais re-tenté tant que le process ne redémarre pas**.
- Une webapp tourne des semaines sans reboot → un batch transitoirement échoué reste bloqué d'autant.
- **Pire** : le janitor 24h `PurgeOldWAL` (`queue.go:370-432`) supprime **tout** WAL > 7 j par `mtime`, **sans vérifier s'il a été persisté** → un batch bloqué est **silencieusement effacé après 7 j** = perte de données potentielle (le code parie sur « un re-sync delta couvrira », vrai seulement pour les matchs encore dans la fenêtre API).

➡️ **Conséquence** : le vrai correctif n'est PAS le retry intra-cycle [A], mais une **recovery périodique en cours d'exécution** (Phase 1 ci-dessous). [A] devient un complément (réduit le churn), pas le cœur.

## Périmètre retenu

### Phase 1 — Recovery périodique en cours d'exécution (NOUVEAU, cœur, ~45min)

**Pourquoi cœur** : c'est le correctif du trou « boot-only » ci-dessus — il supprime la dépendance au reboot et le risque de perte à 7 j, sans la complexité d'un retry par appel.

**Tâches** :
1. Appeler `RecoverPending()` **périodiquement** en runtime, pas seulement au boot. Deux options (à trancher) :
   - **(a) recommandé** : le câbler dans le janitor existant (`main.go:663-695`), **avant** `PurgeOldWAL`, et descendre l'intervalle (ex. ticker dédié 10-15 min, ou hook au début de chaque cycle de sync). Re-soumet tout WAL pending dans le channel → le Worker continu le persiste.
   - (b) ticker dédié court (5-10 min) dans la goroutine persist.
2. **Garde-fou anti-perte** : garantir que `RecoverPending()` tourne **avant** toute `PurgeOldWAL` du même cycle (sinon on purge ce qu'on aurait pu rejouer). Idéalement, ne purger un WAL > 7 j que s'il a déjà été tenté ≥ N fois (lien avec [B] DLQ si on l'active).
3. **Idempotence** : `RecoverPending` est déjà idempotent (testé `queue_test.go:304`) — re-soumettre un WAL déjà en vol ne crée pas de doublon (ACK supprime le fichier). Vérifier qu'une re-soumission concurrente d'un batch en cours de persist ne double pas (dédup par `batch_id` / lock fichier).

**Livrable** : recovery runtime câblée + 1 test (WAL pending re-soumis sans reboot, et non purgé avant tentative).

### Phase 2 — [G] Test E2E FATAL DuckDB injecté mid-batch (~1h)

**Pourquoi d'abord** : le moins risqué, le plus rentable en confiance. Couvre l'invariant central post-incident ART 2026-05-24.

**État actuel** (vérifié) :
- ✅ `e2e_test.go::TestE2E_Pipeline_CrashRecovery` couvre la recovery WAL→boot.
- ✅ Rollback garanti par `shared_persister.go:71-75` (`defer tx.Rollback()`).
- ❌ Aucun test n'injecte un **FATAL DuckDB mid-batch** (erreur après N rows insérées) pour vérifier rollback complet + survie WAL + rejouabilité.

**Tâches** :
1. Mock `txBeginner` / `Persister` qui retourne une erreur type `"database has been invalidated"` après K rows insérées dans la TX.
2. Asserter : (a) la TX rollback (aucune row partielle visible), (b) le fichier WAL **survit** (pas d'ACK), (c) `RecoverPending()` rejoue le batch proprement au cycle suivant.
3. Placer sous build tag `integration` si une vraie `*sql.DB` DuckDB `:memory:` est requise pour `BeginTx`.

**Livrable** : 1 test dans `internal/persist/` (ou `worker_test.go`), vert.

### Hors périmètre — [E] Endpoint `/health/persist` (retiré du plan)

**Décision (2026-06-09) : on ne le fait pas.** Raisons :
- La **Phase 1** (recovery périodique) ferme déjà le besoin de durabilité **sans** actionneur externe → l'endpoint n'a plus de rôle de remédiation.
- L'**observabilité existe déjà** via `/debug/vars` (expvar persist) + logs, et le **diagnostic manuel** via le CLI `cmd/diag_replay_wal`.
- Un endpoint ne *fait* rien seul : il **exige un consommateur** (Docker `HEALTHCHECK`, systemd, Uptime Robot…). Aucun n'est câblé sur ce projet → ce serait une jauge passive = poids mort (YAGNI).

➡️ **À ne ressortir QUE si** on introduit un consommateur. **Règle** : endpoint et consommateur dans le **même lot**, jamais l'endpoint seul. Tant qu'il n'y a pas de besoin de supervision externe, l'expvar suffit.

### Complément — [A] Retry+backoff (utile, non prioritaire)

Avec la **Phase 1** (recovery périodique), la durabilité ne dépend plus du reboot. [A] devient un **complément** qui réduit le churn (re-tente immédiatement un batch transitoirement échoué au lieu d'attendre le prochain tick de recovery).
- Si fait : `persist.WithRetry(maxRetries, baseDelay)` dans `worker.handle()`, backoff 1s→2s→4s puis abandon (laisse en WAL → repris par la recovery périodique).
- **Garde-fou** : ne retenter QUE sur erreurs transitoires (lock/busy/IO), **jamais** sur erreur de parse/contrainte (sinon boucle poison).

### Parké — [B] DLQ après N retries

**Reporté jusqu'à observation d'un batch « poison » réel.** Déclencheur : logs montrant le même `batch_id` rejoué ≥ N fois (boot ou recovery périodique). Fix alors : compteur d'attempts dans le nom WAL → `walDir/dlq/` au-delà de N + alerte ERROR. NB : la DLQ devient aussi le garde-fou propre du `PurgeOldWAL` (purger depuis `dlq/` plutôt que d'effacer des batches jamais tentés).

## Ordre & estimation

1. **Phase 1 — recovery périodique** — ~45min (**priorité réelle** : ferme le trou boot-only + risque de perte à 7 j)
2. **Phase 2 — [G] test FATAL** — ~1h (confiance ART)
3. [A] complément / [B] parké — sur déclencheur
4. [E] `/health/persist` — **hors périmètre** (à ressortir uniquement avec un consommateur)

Une seule branche `feat/persist-robustness`, commits par phase.

## Références
- `internal/persist/{worker.go, queue.go, doc.go, shared_persister.go}`
- `internal/sync/{convergence.go, engine_postsync.go}` (recouvrement convergence)
- `internal/api/handlers/health.go` (modèle read-only)
- ADR 0019 — Collect→Persist ; `.ai/V7/audit_art_writes.md` ; `.ai/V7/HANDOFF_ENRICHMENT_CONVERGENCE.md`
