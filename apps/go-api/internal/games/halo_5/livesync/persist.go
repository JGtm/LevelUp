package livesync

import (
	"context"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// loadKnownMatchIDs lit les match_id déjà persistés du shared d'un titre (delta-stop).
// Lease RW COURT, path-keyé (relâché AVANT le fetch réseau).
//
// provider : provider per-titre du shared h5 (résolu via le Manager au câblage).
// Quand non-nil, AcquireSharedWriterStandalone route par provider.AcquireWriter
// (PreSwap → drain RO → RW → reopen RO au release) : c'est ce qui rend le B-swap
// sûr dès qu'un pool joueur lit AUSSI le shared h5 en RO (sinon RO+RW non
// coordonnés → "different configuration"). provider == nil → fallback legacy
// (mode kill-switch / Manager absent).
func loadKnownMatchIDs(ctx context.Context, provider sharedprovider.Provider, sharedDBPath string) (map[string]bool, error) {
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "h5_livesync_known"), provider, sharedDBPath)
	if err != nil {
		return nil, fmt.Errorf("h5 known-set: acquire shared: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, "SELECT match_id FROM match_registry")
	if err != nil {
		return nil, fmt.Errorf("h5 known-set: query match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := make(map[string]bool, 512)
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr == nil {
			known[id] = true
		}
	}
	return known, rows.Err()
}

// loadXUIDAliasesSeed lit le mapping gamertag→xuid déjà connu (shared.xuid_aliases,
// alimenté par les runs précédents) pour AMORCER le CachingResolver : un joueur déjà
// résolu n'est PAS re-résolu via PeopleHub → anti rate-limit (le storm 429 venait de
// re-résoudre tout le roster à chaque run). Best-effort : toute erreur → graine vide.
func loadXUIDAliasesSeed(ctx context.Context, provider sharedprovider.Provider, sharedDBPath string) map[string]string {
	seed := make(map[string]string, 1024)
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "h5_livesync_aliases_seed"), provider, sharedDBPath)
	if err != nil {
		return seed
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		"SELECT gamertag, xuid FROM xuid_aliases WHERE gamertag IS NOT NULL AND xuid IS NOT NULL")
	if err != nil {
		return seed
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var gt, xuid string
		if rows.Scan(&gt, &xuid) == nil && gt != "" && xuid != "" {
			seed[gt] = xuid
		}
	}
	return seed
}

// persistBatches écrit les batches sur le shared d'un titre sous UN lease RW (acquis
// APRÈS le fetch — on ne tient jamais le write-lease pendant l'I/O réseau).
// Idempotence : SharedPersister no-op un match_id déjà présent (INSERT-only, ART-safe).
// Retourne les batches RÉELLEMENT persistés (sans erreur) + les messages d'erreur.
func persistBatches(ctx context.Context, provider sharedprovider.Provider, sharedDBPath string, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
	if len(batches) == 0 {
		return nil, nil
	}
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "h5_livesync_persist"), provider, sharedDBPath)
	if err != nil {
		return nil, []string{fmt.Sprintf("h5 persist: acquire shared: %v", err)}
	}
	defer release()

	sp := persist.NewSharedPersister(db)
	var (
		done []*persist.MatchBatch
		errs []string
	)
	for _, b := range batches {
		persist.SanitizeBatch(b)
		if perr := sp.Persist(ctx, b); perr != nil {
			errs = append(errs, fmt.Sprintf("h5 persist match %s: %v", batchMatchID(b), perr))
			continue
		}
		done = append(done, b)
	}
	return done, errs
}

func batchMatchID(b *persist.MatchBatch) string {
	if b != nil && b.Shared.Match != nil {
		return b.Shared.Match.MatchID
	}
	return "?"
}
