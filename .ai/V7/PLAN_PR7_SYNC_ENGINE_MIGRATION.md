# PR 7: Migrate Sync Engine to dblease.AcquireWriterCtx

**Statut** : Plan documenté (implémentation déféré post-merge leased-writer-enforcement)

**Objective**: Remplacer tous les appels `sync.AcquireLeaseCtx` (legacy API) par `dblease.AcquireWriter` (nouvelle API unifiée) dans le sync engine pour bénéficier de la visibilité observabilité (compteurs `dblease_acquire_total{kind,status}`).

**Avantages** :
- Observabilité centralisée : tous les DB writes (HTTP + sync) sont tracées par le même système de lease
- Suppression de la double implémentation du mutex (legacy sync.leaseMutex vs nouveau dblease.leaseMutex)
- Audit trail unifié pour détecter les contentions et les timeouts

**Risque** : ~63 tests du sync qui doivent restér bit-à-bit identiques. Migration sensible, à faire avec recette testée.

---

## Sites à migrer (12 total)

### Group 1: engine.go (9 sites)

#### Site 1 & 2 : SyncDelta (lines 137-143)
```go
// AVANT (legacy)
relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)

// APRÈS (nouveau)
writerPlayer, err := dblease.AcquireWriter(playerDB, e.playerDBPath, dblease.KindPlayer, timeout)
writerShared, err := dblease.AcquireWriter(sharedDB, e.sharedDBPath, dblease.KindShared, timeout)
defer writerPlayer.Release()
defer writerShared.Release()
```

#### Site 3 & 4 : SyncFull (lines 185-191)
Même pattern que SyncDelta.

#### Site 5 & 6 : RunPostSyncHook (lines 213-219)
```go
relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
```

#### Site 7 & 8 : RecomputeRatings (lines 358-366)
```go
relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
```

#### Site 9 : StoreSession (line 935)
```go
relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
```

### Group 2: backfill_weapons.go (1 site)

#### Site 10 : BackfillWeaponKillsForMatch (line 120)
```go
// AVANT
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)

// APRÈS
writerShared, err := dblease.AcquireWriter(sharedDB, e.sharedDBPath, dblease.KindShared, timeout)
defer writerShared.Release()
```

### Group 3: friends_recompute.go (2 sites)

#### Site 11 & 12 : ComputeTeammatesAsync (lines 52, 58)
```go
relPlayer, err := AcquireLeaseCtx(ctx, playerDBPath)
relShared, err := AcquireLeaseCtx(ctx, sharedDBPath)
```

---

## Plan d'exécution (étapes)

### Phase 1 : Préparation
1. Créer une branche `refactor/pr7-sync-lease-migration` depuis leased-writer-enforcement
2. Ajouter un helper dans sync/lease.go :
```go
// AcquireLeaseCtxLegacy — pour les sites qui n'ont pas accès à la *sql.DB
// (déféré au futur refactor où on passera les DBs au sync engine)
func AcquireLeaseCtxLegacy(ctx context.Context, path string) (func(), error) {
	// Wrapper compatibility jusqu'à migration complète
}
```

### Phase 2 : Migration par groupe
1. **Group 1 (SyncDelta/Full)** :
   - Migrer les 4 sites dans SyncDelta
   - Vérifier : `go test -tags=integration ./internal/sync/ -run TestSyncDelta`
   - Ajouter 1 test: TestSyncDeltaUsesNewLeasingAPI (vérifie appels dblease)

2. **Group 2 (RunPostSyncHook/RecomputeRatings)** :
   - Migrer 4 sites
   - Vérifier : tests existants du hook + ratings

3. **Group 3 (StoreSession)** :
   - Migrer 1 site
   - Vérifier : tests enrichissement

4. **Group 4 (backfill_weapons)** :
   - Migrer 1 site
   - Vérifier : TestBackfillWeaponKills*

5. **Group 5 (friends_recompute)** :
   - Migrer 2 sites
   - Vérifier : tests recomputation

### Phase 3 : Vérification
```bash
# Suite baseline pré-migration doit rester verte
bash scripts/check_test_baseline.sh tests

# Couverture du sync ne doit pas baisser
go test -tags=integration -coverprofile=cov_sync.out ./internal/sync/...
go tool cover -func=cov_sync.out | tail -1  # doit rester >= 75%
```

### Phase 4 : Observabilité
1. Vérifier que `dblease_acquire_total{kind=player,status=success}` et
   `dblease_acquire_total{kind=shared,status=timeout}` sont bien incrémentés
2. Ajouter monitoring sur le ratio timeout/success par kind

---

## Modèle de refactoring (exemple Site 1)

**Avant** (engine.go:137-143) :
```go
func (e *Engine) SyncDelta(ctx context.Context, ...) error {
	relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
	if err != nil {
		return fmt.Errorf("failed to acquire player lease: %w", err)
	}
	defer relPlayer()

	relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
	if err != nil {
		return fmt.Errorf("failed to acquire shared lease: %w", err)
	}
	defer relShared()

	// ... rest of sync logic
}
```

**Après** (avec nouvelle API):
```go
func (e *Engine) SyncDelta(ctx context.Context, ...) error {
	const leaseTimeout = 30 * time.Second

	writerPlayer, err := dblease.AcquireWriter(
		e.playerDB, e.playerDBPath, dblease.KindPlayer, leaseTimeout)
	if err != nil {
		return fmt.Errorf("failed to acquire player writer: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriter(
		e.sharedDB, e.sharedDBPath, dblease.KindShared, leaseTimeout)
	if err != nil {
		return fmt.Errorf("failed to acquire shared writer: %w", err)
	}
	defer writerShared.Release()

	// ... rest of sync logic (uses writerPlayer/writerShared.Executor for DB ops)
}
```

**Clé de migration** :
- `AcquireLeaseCtx` → `AcquireWriter` (retourne *dblease.LeasedWriter, pas func())
- Ajouter `defer writer.Release()` pour chaque writer
- Utiliser `writer.Executor` (*sql.Tx / *sql.DB) pour les opérations DB
- Les erreurs et le timeout restent identiques (même comportement)

---

## Tests de validation

```go
// Ajouter à sync/lease_test.go pour valider la migration
func TestSyncEngineSitesUseNewLeasingAPI(t *testing.T) {
	// Vérifier via reflection que sync.Engine._fields ne contient pas
	// de références à l'ancienne API (compile-time verification impossible,
	// donc smoke test via exécution)
}

func TestSyncDeltaNewAPIBehaviorIdentical(t *testing.T) {
	// Fixture: run legacy vs new API, compare match_registry/participants rows
	// Doit être 100% identique
}
```

---

## Déploiement & rollback

**Go/no-go** :
1. Tous les tests baseline passent (esp. TestSyncDelta*, TestBackfillWeapon*)
2. Couverture du sync >= seuil baseline
3. Observabilité : au moins 100 appels `dblease_acquire_total` tracés en QA

**Rollback** (si bug détecté) :
1. Revenir à `AcquireLeaseCtx` sur les sites affectés
2. Cherryomit le revert sur main → déployer hotfix
3. Relancer PR7 après root cause analysis

---

## Gains post-migration

| Métrique | Avant | Après |
|----------|-------|-------|
| Mutex coordonnés | 2 (sync.leaseMutex + dblease.leaseMutex) | 1 (dblease.leaseMutex) |
| Visibilité achat de lease | 2 APIs | 1 API |
| Monitoring contentions | Aucune | `dblease_acquire_total{status=timeout}` |
| Audit trail DB writes | Fragmenté (sync vs HTTP) | Unifié |

---

## Timeline estimée

- Phase 1 (prep): 1h
- Phase 2 (migration x5 groupes): 4h
- Phase 3 (vérification): 2h
- Phase 4 (observabilité): 1h
- **Total**: ~8h (à faire post-merge leased-writer-enforcement)

Non-bloquant pour la résolution fonctionnelle de P1/P2/P3 (leased-writer-enforcement).
