package livesync

import (
	"context"
	"fmt"

	"levelup/go-api/internal/persist"
	syncpkg "levelup/go-api/internal/sync"
)

// loadKnownMatchIDs lit les match_id déjà persistés du shared d'un titre (delta-stop).
// Lease RW COURT, path-keyé (relâché AVANT le fetch réseau). Halo 5 → provider nil
// (chemin legacy AcquireSharedWriterStandalone) : VÉRIFIÉ sûr — aucun pool ne tient
// le shared h5 en RO en Phase 1, donc pas de conflit RO+RW mono-process (cf. recon
// T3b + mémoire bswap). Le lease est indépendant du shared Infinite (clé = chemin).
func loadKnownMatchIDs(ctx context.Context, sharedDBPath string) (map[string]bool, error) {
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctx, nil, sharedDBPath)
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
func loadXUIDAliasesSeed(ctx context.Context, sharedDBPath string) map[string]string {
	seed := make(map[string]string, 1024)
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctx, nil, sharedDBPath)
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
func persistBatches(ctx context.Context, sharedDBPath string, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
	if len(batches) == 0 {
		return nil, nil
	}
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctx, nil, sharedDBPath)
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
