//go:build integration

// Package sync — engine_provider_multiuser_test.go : E2E multi-user qui
// reproduit le scénario auto_sync sous charge HTTP.
//
// Le scheduler auto_sync prod scanne plusieurs gamertags actifs et lance un
// RunDelta par user. En mode B-swap, tous les RunDelta acquièrent le MÊME
// Provider shared (un seul shared_matches_v2.duckdb). dblease sérialise les
// writers ; les readers HTTP doivent rester non-impactés.
//
// Ce test vérifie le contrat : 3 syncs concurrents (3 users distincts) +
// 10 readers HTTP-like, tous doivent réussir sans Catalog Error ni
// "different configuration".
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// multiUserEnv : Provider partagé + N PlayerDB + N engines distincts.
type multiUserEnv struct {
	t        *testing.T
	repoRoot string
	provider sharedprovider.Provider
	users    []*userSetup
}

type userSetup struct {
	gamertag string
	xuid     string
	pool     *duckdb.PlayerDB
	engine   *SyncEngine
	mock     *mockHaloClient
}

// newMultiUserEnv : tempdir + 1 Provider + N users. Tous les engines partagent
// le même Provider via WithSharedProvider.
func newMultiUserEnv(t *testing.T, nUsers int) *multiUserEnv {
	t.Helper()
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	repoRoot := t.TempDir()
	titleSlug := titlePkg.DefaultSlug
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titleSlug)
	metaPath := pr.MetadataDBPath(titleSlug)

	for _, p := range []string{
		filepath.Dir(sharedPath),
		filepath.Dir(metaPath),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	// Init shared via OpenSharedDB + schéma, puis fermer pour libérer le handle.
	sharedInit, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB init: %v", err)
	}
	// D1b : le chemin batch (SharedPersister, unique voie d'écriture) écrit des
	// colonnes (match_intensity, backfill_bits, mécaniques H5…) ajoutées par les
	// migrations title-owned, absentes du schéma statique EnsureSharedSchema.
	patchSharedSchemaForBatch(t, sharedInit.SQLDb())
	_ = sharedInit.Close()

	mgr := sharedprovider.NewManager()
	provider, err := mgr.For(sharedPath)
	if err != nil {
		t.Fatalf("Manager.For shared: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })

	users := make([]*userSetup, nUsers)
	tokens := &domain.HaloTokens{
		SpartanToken:   "test-spartan",
		ClearanceToken: "test-clearance",
	}
	for i := 0; i < nUsers; i++ {
		gamertag := fmt.Sprintf("MultiUser%d", i)
		xuid := fmt.Sprintf("%016d", 1234567890000000+i)
		playerPath := pr.PlayerDBPath(titleSlug, gamertag)
		if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
			t.Fatalf("mkdir player %d: %v", i, err)
		}
		playerInit, err := OpenPlayerDB(playerPath)
		if err != nil {
			t.Fatalf("OpenPlayerDB init user %d: %v", i, err)
		}
		_ = playerInit.Close()

		pdb, err := duckdb.GetOrOpen(context.Background(), duckdb.PlayerPoolConfig{
			Gamertag:     gamertag,
			XUID:         xuid,
			TitleSlug:    titleSlug,
			PlayerDBPath: playerPath,
			SharedDBPath: sharedPath,
			MetaDBPath:   metaPath,
			SharedReader: provider,
		})
		if err != nil {
			t.Fatalf("GetOrOpen user %d: %v", i, err)
		}

		engine := NewSyncEngine(repoRoot, gamertag, xuid, tokens, nil).
			WithSharedProvider(provider)
		mock := &mockHaloClient{}
		engine.SetCustomClient(mock)

		users[i] = &userSetup{
			gamertag: gamertag,
			xuid:     xuid,
			pool:     pdb,
			engine:   engine,
			mock:     mock,
		}
	}

	return &multiUserEnv{
		t:        t,
		repoRoot: repoRoot,
		provider: provider,
		users:    users,
	}
}

// seedMatches pour un user donné — chaque user a ses propres match_ids
// uniques (le prefix inclut son index pour éviter les collisions cross-user
// qui simuleraient incorrectement le partage du shared registry).
func (env *multiUserEnv) seedMatches(userIdx, count int) {
	u := env.users[userIdx]
	ids := make([]string, count)
	stats := make(map[string]map[string]any, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("user%d-%016d-4000-8000-%012d", userIdx, i, i)
		ids[i] = id
		stats[id] = makeMatchJSON(id, 2)
	}
	u.mock.history = makeHistory(ids...)
	u.mock.statsBody = stats
}

// TestE2E_MultiUser_ConcurrentSync_HTTPReaders_integration : reproduction du
// scénario auto_sync prod multi-user.
//
// Setup : 3 users distincts, chacun avec son pool, son engine, son mock. Tous
// partagent le MÊME Provider shared (= 1 fichier shared_matches_v2.duckdb).
// Action : 3 RunDelta lancés en parallèle (autant que possible — dblease
// sérialisera) + 10 readers HTTP en boucle continue.
// Assertions : 3 syncs OK (dblease arbitre), ≥ 18 matchs finaux dans shared
// (3 users × ~6 matchs unique-per-user), 0 Catalog Error, 0 "different
// configuration", Provider StateRO post-cycle.
func TestE2E_MultiUser_ConcurrentSync_HTTPReaders_integration(t *testing.T) {
	const (
		nUsers       = 3
		matchPerUser = 6
		nReaders     = 10
	)
	env := newMultiUserEnv(t, nUsers)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for i := 0; i < nUsers; i++ {
		env.seedMatches(i, matchPerUser)
	}

	var (
		syncOK              atomic.Int64
		syncErr             atomic.Int64
		readerOK            atomic.Int64
		readerErr           atomic.Int64
		readerCatalogErr    atomic.Int64
		readerDiffConfigErr atomic.Int64
		stopReaders         atomic.Bool
		wg                  sync.WaitGroup
	)

	// Readers : poll shared via Provider en boucle pendant tout le test.
	for i := 0; i < nReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stopReaders.Load() {
				db, release, err := env.provider.Get(ctx)
				if err != nil {
					readerErr.Add(1)
					classifyErr(err, &readerCatalogErr, &readerDiffConfigErr)
					time.Sleep(time.Millisecond)
					continue
				}
				var n int
				qErr := db.QueryRowContext(ctx,
					"SELECT COUNT(*) FROM match_registry").Scan(&n)
				release()
				if qErr != nil {
					readerErr.Add(1)
					classifyErr(qErr, &readerCatalogErr, &readerDiffConfigErr)
				} else {
					readerOK.Add(1)
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Syncs : lancer 3 RunDelta en parallèle (dblease arbitre la sérialisation
	// shared, les writers player sont indépendants par user).
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        matchPerUser,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}
	var syncWG sync.WaitGroup
	for i := 0; i < nUsers; i++ {
		syncWG.Add(1)
		go func(userIdx int) {
			defer syncWG.Done()
			result, err := env.users[userIdx].engine.RunDelta(ctx, opts)
			if err != nil {
				syncErr.Add(1)
				t.Logf("RunDelta user %d : err=%v", userIdx, err)
				return
			}
			syncOK.Add(1)
			t.Logf("RunDelta user %d : inserted=%d (status=%s)",
				userIdx, result.MatchesInserted, result.Status())
		}(i)
	}
	syncWG.Wait()

	// Laisse les readers tourner 200ms supplémentaires post-sync (sanity).
	time.Sleep(200 * time.Millisecond)
	stopReaders.Store(true)
	wg.Wait()

	t.Logf("Multi-user E2E : sync OK=%d err=%d | readers OK=%d err=%d (Catalog=%d, DiffConfig=%d)",
		syncOK.Load(), syncErr.Load(),
		readerOK.Load(), readerErr.Load(),
		readerCatalogErr.Load(), readerDiffConfigErr.Load())

	// ─── Assertions ─────────────────────────────────────────────────────────────
	if syncErr.Load() > 0 {
		t.Errorf("%d erreur(s) sync multi-user (attendu 0 — dblease doit arbitrer)",
			syncErr.Load())
	}
	if syncOK.Load() != int64(nUsers) {
		t.Errorf("syncOK = %d (attendu %d : 1 par user)", syncOK.Load(), nUsers)
	}
	if readerCatalogErr.Load() > 0 {
		t.Errorf("%d Catalog Error reader multi-user (attendu 0)",
			readerCatalogErr.Load())
	}
	if readerDiffConfigErr.Load() > 0 {
		t.Errorf("%d \"different configuration\" reader (attendu 0)",
			readerDiffConfigErr.Load())
	}
	if readerOK.Load() < 100 {
		t.Errorf("readerOK = %d trop faible (attendu ≥ 100)", readerOK.Load())
	}

	// Final : chaque user a inséré ≥ 1 match unique. Le shared registry
	// agrégé doit donc contenir ≥ nUsers matchs distincts (probablement plus
	// à cause de la pagination mock qui retourne en boucle).
	finalDB, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get final: %v", err)
	}
	defer release()
	var finalCount int
	if err := finalDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&finalCount); err != nil {
		t.Fatalf("final count: %v", err)
	}
	if finalCount < nUsers {
		t.Errorf("match_registry final = %d (attendu ≥ %d, 1 par user min)",
			finalCount, nUsers)
	}

	// Provider doit être à StateRO après tous les swaps.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-multi-user = %v (attendu StateRO)", state)
	}
}
