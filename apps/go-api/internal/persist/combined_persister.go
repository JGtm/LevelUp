// Package persist — combined_persister.go : BatchPersister qui écrit shared + player
// en un seul appel, avec gestion des leases par batch.
//
// Utilisé par le Worker async (Phase 3 du refactor Collect→Persist) pour écrire
// les batches dans les deux DBs sans que le caller (main.go) ait à gérer les
// connexions lui-même.
//
// Pour éviter un cycle d'import (persist ↔ sync), l'acquisition de la connexion
// shared est injectée via SharedWriterFn — le caller (main.go) fournit la closure
// appropriée selon le mode (Provider B-swap ou legacy dblease).
//
// Chaque appel Persist ouvre ses propres connexions et les ferme à la fin.
// Acceptable pour notre volume (<50 matchs / cycle) — la durée d'une connexion
// DuckDB ≈ 1ms sur SSD local.

package persist

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// SharedWriterFn acquiert une connexion RW sur shared_matches_v2.duckdb avec
// le lease exclusif approprié (Provider B-swap ou dblease legacy).
// Le caller doit appeler la fonction release() retournée via defer.
type SharedWriterFn func(ctx context.Context) (*sql.DB, func(), error)

// CombinedPersister implémente BatchPersister pour shared + player en un seul
// appel. Conçu pour être utilisé par un Worker goroutine unique.
type CombinedPersister struct {
	acquireShared  SharedWriterFn
	playerDBPathFn func(gamertag string) string
}

// NewCombinedPersister construit un CombinedPersister.
//
//   - acquireShared  : fonction qui ouvre shared_matches_v2.duckdb en RW avec lease.
//     Construire depuis main.go avec le Provider (B-swap) ou OpenSharedDB (legacy).
//   - playerDBPathFn : retourne le chemin complet de stats.duckdb pour un gamertag.
func NewCombinedPersister(acquireShared SharedWriterFn, playerDBPathFn func(gamertag string) string) *CombinedPersister {
	return &CombinedPersister{
		acquireShared:  acquireShared,
		playerDBPathFn: playerDBPathFn,
	}
}

// Persist écrit le batch dans shared puis dans player, chacun avec son propre
// lease et sa propre connexion. Atomique par DB (1 TX / DB), pas cross-DB.
//
// Ordre : shared AVANT player — si le shared fail, on ne touche pas player
// (le Worker loggue l'erreur et ne supprime pas le WAL → retry au prochain boot).
func (p *CombinedPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return fmt.Errorf("CombinedPersister.Persist: batch nil")
	}

	// ── 1. Shared DB ─────────────────────────────────────────────────────────
	sharedDB, releaseShared, err := p.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("CombinedPersister: acquire shared: %w", err)
	}

	sharedErr := NewSharedPersister(sharedDB).Persist(ctx, batch)
	releaseShared()

	if sharedErr != nil {
		slog.ErrorContext(ctx, "CombinedPersister: shared persist échoué",
			"batch_id", batch.BatchID, "player", batch.Player, "err", sharedErr)
		return fmt.Errorf("CombinedPersister: shared: %w", sharedErr)
	}

	// ── 2. Player DB ──────────────────────────────────────────────────────────
	playerPath := p.playerDBPathFn(batch.Player)
	if playerPath == "" {
		slog.WarnContext(ctx, "CombinedPersister: playerDBPath vide, skip player persist",
			"batch_id", batch.BatchID, "player", batch.Player)
		return nil
	}

	playerCtx, playerCancel := context.WithTimeout(ctx, 30*time.Second)
	defer playerCancel()

	playerLease, leaseErr := dblease.AcquireLeaseCtx(playerCtx, playerPath)
	if leaseErr != nil {
		return fmt.Errorf("CombinedPersister: player lease %s: %w", batch.Player, leaseErr)
	}
	defer playerLease()

	playerDB, openErr := sql.Open("duckdb", playerPath+"?access_mode=READ_WRITE")
	if openErr != nil {
		return fmt.Errorf("CombinedPersister: open player %s: %w", batch.Player, openErr)
	}
	defer func() { _ = playerDB.Close() }()

	if playerErr := NewPlayerPersister(playerDB).Persist(playerCtx, batch); playerErr != nil {
		slog.ErrorContext(ctx, "CombinedPersister: player persist échoué",
			"batch_id", batch.BatchID, "player", batch.Player, "err", playerErr)
		return fmt.Errorf("CombinedPersister: player: %w", playerErr)
	}

	return nil
}
