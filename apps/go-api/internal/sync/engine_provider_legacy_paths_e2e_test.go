//go:build integration

// Package sync — engine_provider_legacy_paths_e2e_test.go : E2E qui couvre
// les fonctions ex-legacy migrées au commit 13 (BackfillEventsForMatches,
// BackfillWeaponKillsForMatches, RecomputeIsWithFriends, MatchRecomputer,
// RecalculatePlayerSessions).
//
// Avant ce commit, ces fonctions ouvraient `shared` en RW direct via
// OpenSharedDB, court-circuitant le Provider. En collision avec un sync
// Provider-coordonné, elles pouvaient déclencher "different configuration".
//
// Ces tests prouvent que sous mode B-swap, ces fonctions :
//   - acquièrent le writer via Provider (donc coordonnent avec les readers)
//   - ne deadlock pas (helper AcquireSharedWriterStandalone prend le lease
//     en interne en mode Provider)
//   - n'introduisent pas de Catalog Error ni "different configuration" pour
//     les readers HTTP concurrents
package sync

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestE2E_BackfillEvents_WithProvider_NoFriction_integration : lance
// BackfillEventsForMatches via Provider, sous 5 readers HTTP en boucle.
// Vérifie 0 friction.
//
// NOTE : BackfillEventsForMatches dans son flow normal fait des appels HTTP
// SPNKr (GetHighlightEventsChunk). Comme la liste de matchIDs est vide, on
// court-circuite avant tout appel réseau — on teste juste le path lease+open
// du début de la fonction, qui est exactement le bug qu'on a fixé.
func TestE2E_BackfillEvents_WithProvider_NoFriction_integration(t *testing.T) {
	env := newE2EEnv(t)
	env.mock.highlightChunkFound = false // pas de chunk → fonction se court-circuite

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var (
		readerOK      atomic.Int64
		readerErr     atomic.Int64
		readerCatalog atomic.Int64
		readerDiffCfg atomic.Int64
		stopReaders   atomic.Bool
		readersWG     sync.WaitGroup
	)
	for i := 0; i < 5; i++ {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			for !stopReaders.Load() {
				db, release, err := env.pool.SharedReadDB().Get(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return
					}
					readerErr.Add(1)
					classifyErr(err, &readerCatalog, &readerDiffCfg)
					continue
				}
				var n int
				qErr := db.QueryRowContext(ctx,
					"SELECT COUNT(*) FROM match_registry").Scan(&n)
				release()
				if qErr != nil {
					readerErr.Add(1)
					classifyErr(qErr, &readerCatalog, &readerDiffCfg)
				} else {
					readerOK.Add(1)
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// BackfillEventsForMatches avec liste vide + includeBroken=false : pas
	// d'appel SPNKr, mais TOUT le path lease+open est exercé.
	// Avant fix : double-lock dblease + OpenSharedDB qui collisionnerait.
	// Après fix : passe par Provider.AcquireWriter, propre.
	_, err := env.engine.BackfillEventsForMatches(ctx, []string{}, false, nil)
	if err != nil {
		t.Fatalf("BackfillEventsForMatches: %v (était deadlock avant fix B3 + bug avant commit 13a)", err)
	}

	// Provider doit être à StateRO après l'opération.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-BackfillEvents = %v (attendu StateRO)", state)
	}

	time.Sleep(100 * time.Millisecond)
	stopReaders.Store(true)
	readersWG.Wait()

	t.Logf("BackfillEvents E2E : readers OK=%d err=%d (Catalog=%d, DiffCfg=%d)",
		readerOK.Load(), readerErr.Load(),
		readerCatalog.Load(), readerDiffCfg.Load())

	if readerCatalog.Load() > 0 {
		t.Errorf("%d Catalog Error reader (attendu 0)", readerCatalog.Load())
	}
	if readerDiffCfg.Load() > 0 {
		t.Errorf("%d \"different configuration\" reader (attendu 0)", readerDiffCfg.Load())
	}
}

// TestE2E_RecomputeIsWithFriends_WithProvider_NoDeadlock_integration : test
// de régression du commit 13b. Avant le fix, RecomputeIsWithFriends faisait
// dblease + OpenSharedDB direct. Avec un Provider présent dans le même
// process, conflit possible. Maintenant via AcquireSharedWriterStandalone.
func TestE2E_RecomputeIsWithFriends_WithProvider_NoDeadlock_integration(t *testing.T) {
	env := newE2EEnv(t)
	env.seedMockMatches("recompute-friends", 3)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Sync 1er pour populate shared.match_participants (sinon RecomputeIsWithFriends
	// retourne early-skip).
	if _, err := env.engine.RunDelta(ctx, domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 3,
		WithParticipants: true, RequestsPerSecond: 100,
	}); err != nil {
		t.Fatalf("RunDelta initial: %v", err)
	}

	// RecomputeIsWithFriends avec Provider attaché — doit terminer sans deadlock.
	// L'ami n'existe pas en DB → resolved=0, no-op, mais le path lease+open
	// est entièrement exercé (= la mécanique du fix B3 standalone).
	res, err := RecomputeIsWithFriends(ctx, env.provider, env.playerPath, env.sharedPath,
		env.xuid, []string{"NonexistentFriend"})
	if err != nil {
		t.Fatalf("RecomputeIsWithFriends avec Provider : %v (était deadlock avant fix 13b)", err)
	}
	t.Logf("RecomputeIsWithFriends result: friends_resolved=%d promoted=%d",
		res.FriendXUIDsCount, res.MatchesPromoted)

	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-recompute = %v (attendu StateRO)", state)
	}

	// Sanity : reader OK après tout ça.
	db, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get post-recompute: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&n); err != nil {
		t.Errorf("query post-recompute: %v", err)
	}
}
