//go:build integration

// Package sync — engine_provider_e2e_test.go : E2E intégration sync engine ↔
// SharedDBProvider ↔ pool joueur ↔ readers HTTP-like, sous charge concurrente.
//
// Ce test reproduit le scénario exact qui a déclenché le bug initial
// "different configuration" sur Madina97294 :
//   - 1 SyncEngine en mode B-swap (sharedProvider injecté)
//   - 1 PlayerDB pool consommant le Provider via SharedReader
//   - N goroutines readers tapant les repos cross-DB pendant que le sync écrit
//
// Le contrat post-sprint B1 : zéro erreur "different configuration", zéro
// "Catalog Error", aucune friction même quand sync et readers se chevauchent.
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// e2eTestEnv encapsule toute la plomberie d'un test E2E : tempdir, fichiers DB,
// Provider, Pool, Engine, mock client. Permet de partager le setup entre les
// scénarios sans copier 80 lignes par test.
type e2eTestEnv struct {
	t        *testing.T
	repoRoot string
	gamertag string
	xuid     string

	sharedPath string
	playerPath string
	metaPath   string

	provider sharedprovider.Provider
	pool     *duckdb.PlayerDB
	engine   *SyncEngine
	mock     *mockHaloClient
}

// newE2EEnv : tempdir + DBs initialisées + Provider + Pool + Engine wired.
// Le mock retourne une history vide par défaut (à compléter par le test).
func newE2EEnv(t *testing.T) *e2eTestEnv {
	t.Helper()
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	repoRoot := t.TempDir()
	gamertag := "E2ETestPlayer"
	xuid := "1234567890000001"
	titleSlug := titlePkg.DefaultSlug

	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titleSlug)
	playerPath := pr.PlayerDBPath(titleSlug, gamertag)
	metaPath := pr.MetadataDBPath(titleSlug)

	for _, p := range []string{
		filepath.Dir(sharedPath),
		filepath.Dir(playerPath),
		filepath.Dir(metaPath),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	// Initialise shared via OpenSharedDB (crée + EnsureSharedSchema, ferme).
	sharedInit, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB init: %v", err)
	}
	// Depuis D1b, le chemin batch (SharedPersister) est l'UNIQUE voie d'écriture :
	// il écrit des colonnes (match_intensity, backfill_bits, mécaniques H5…) ajoutées
	// par les migrations title-owned et absentes du schéma statique EnsureSharedSchema.
	// On les patche ici, sinon le persist casse sur "column match_intensity does not exist".
	patchSharedSchemaForBatch(t, sharedInit.SQLDb())
	_ = sharedInit.Close()

	// Initialise player via OpenPlayerDB (crée + EnsurePlayerSchema, ferme).
	// Le pool réouvrira via openCachedDB juste après.
	playerInit, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB init: %v", err)
	}
	_ = playerInit.Close()

	// Provider : owner du handle shared, swap RO↔RW autour des sync writes.
	mgr := sharedprovider.NewManager()
	provider, err := mgr.For(sharedPath)
	if err != nil {
		t.Fatalf("Manager.For shared: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })

	// Pool joueur : consomme Provider via SharedReader. En mode B-swap,
	// pdb.Shared reste nil et toutes les queries shared passent par
	// pdb.SharedReadDB().Get() qui délègue au Provider.
	pdb, err := duckdb.GetOrOpen(context.Background(), duckdb.PlayerPoolConfig{
		Gamertag:     gamertag,
		XUID:         xuid,
		TitleSlug:    titleSlug,
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
		SharedReader: provider, // ← mode B-swap actif
	})
	if err != nil {
		t.Fatalf("Pool.GetOrOpen: %v", err)
	}

	// Engine : NewSyncEngine + WithSharedProvider injecte le mode B-swap.
	// Sans .WithSharedProvider, l'engine ferait OpenSharedDB direct (legacy).
	tokens := &domain.HaloTokens{
		SpartanToken:   "test-token-spartan",
		ClearanceToken: "test-token-clearance",
	}
	engine := NewSyncEngine(repoRoot, gamertag, xuid, tokens, nil).
		WithSharedProvider(provider)

	// Mock client injecté — pas d'appel réseau.
	mock := &mockHaloClient{}
	engine.SetCustomClient(mock)

	return &e2eTestEnv{
		t:          t,
		repoRoot:   repoRoot,
		gamertag:   gamertag,
		xuid:       xuid,
		sharedPath: sharedPath,
		playerPath: playerPath,
		metaPath:   metaPath,
		provider:   provider,
		pool:       pdb,
		engine:     engine,
		mock:       mock,
	}
}

// seedMockMatches alimente le mockHaloClient avec N matchs uniques pour permettre
// au sync engine de processer un batch complet. matchIDPrefix permet de
// différencier les batches successifs.
func (env *e2eTestEnv) seedMockMatches(matchIDPrefix string, count int) {
	ids := make([]string, count)
	stats := make(map[string]map[string]any, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s-%016d-4000-8000-%012d", matchIDPrefix, i, i)
		ids[i] = id
		stats[id] = makeMatchJSON(id, 2)
	}
	env.mock.history = makeHistory(ids...)
	env.mock.statsBody = stats
}

// TestE2E_SyncEngine_HTTPReaders_Concurrency_integration : scénario phare —
// vrai sync engine en mode B-swap + readers concurrents qui interrogent le
// pool joueur via SharedReader pendant que sync écrit.
//
// Le test :
//  1. Démarre 10 goroutines reader qui poll `COUNT(*) FROM match_registry`
//     via pool.SharedReadDB().Get() (chemin HTTP-like).
//  2. Démarre 1 goroutine sync qui appelle engine.RunDelta() avec 5 batches
//     de 4 matchs (mock client réinitialisé entre chaque batch pour
//     simuler 5 cycles de swap RO→RW→RO).
//  3. Attend la fin du sync, puis 200ms de readers supplémentaires.
//  4. Assert : zéro erreur "different configuration" ou "Catalog Error",
//     ≥ 20 matchs écrits, readers majoritairement OK.
func TestE2E_SyncEngine_HTTPReaders_Concurrency_integration(t *testing.T) {
	env := newE2EEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	goroutinesBefore := runtime.NumGoroutine()

	const (
		readers       = 10
		batches       = 5
		matchPerBatch = 4
	)
	var (
		readerOK            atomic.Int64
		readerErr           atomic.Int64
		readerCatalogErr    atomic.Int64
		readerDiffConfigErr atomic.Int64
		syncOK              atomic.Int64
		syncErr             atomic.Int64
		stopReaders         atomic.Bool
		readersWG           sync.WaitGroup
	)

	// Readers : poll match_registry en boucle via SharedReader (chemin pool).
	for i := 0; i < readers; i++ {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			for !stopReaders.Load() {
				db, release, err := env.pool.SharedReadDB().Get(ctx)
				if err != nil {
					readerErr.Add(1)
					classifyErr(err, &readerCatalogErr, &readerDiffConfigErr)
					time.Sleep(1 * time.Millisecond)
					continue
				}
				var count int
				queryErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&count)
				release()
				if queryErr != nil {
					readerErr.Add(1)
					classifyErr(queryErr, &readerCatalogErr, &readerDiffConfigErr)
				} else {
					readerOK.Add(1)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Sync : 5 batches de 4 matchs = 20 matchs total + 5 cycles de swap.
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        matchPerBatch,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100, // pas de rate limit dans les tests
	}
	for b := 0; b < batches; b++ {
		env.seedMockMatches(fmt.Sprintf("batch%d", b), matchPerBatch)
		result, err := env.engine.RunDelta(ctx, opts)
		if err != nil {
			syncErr.Add(1)
			t.Logf("RunDelta batch %d erreur : %v", b, err)
			continue
		}
		syncOK.Add(1)
		if result.MatchesInserted == 0 {
			t.Logf("RunDelta batch %d : 0 match inséré (result=%+v)", b, result)
		}
	}

	// Laisse les readers tourner 200ms supplémentaires après le sync (sanity check
	// qu'on ne plante pas en idle après les swaps successifs).
	time.Sleep(200 * time.Millisecond)
	stopReaders.Store(true)
	readersWG.Wait()

	// ─── Assertions ─────────────────────────────────────────────────────────────

	t.Logf("E2E concurrence : sync OK=%d err=%d | readers OK=%d err=%d (Catalog=%d, DiffConfig=%d)",
		syncOK.Load(), syncErr.Load(),
		readerOK.Load(), readerErr.Load(),
		readerCatalogErr.Load(), readerDiffConfigErr.Load())

	if syncErr.Load() > 0 {
		t.Errorf("%d erreur(s) sync (attendu 0)", syncErr.Load())
	}
	if syncOK.Load() < int64(batches) {
		t.Errorf("syncOK=%d (attendu %d)", syncOK.Load(), batches)
	}
	if readerCatalogErr.Load() > 0 {
		t.Errorf("%d Catalog Error reader (attendu 0 — Provider doit isoler des swaps)",
			readerCatalogErr.Load())
	}
	if readerDiffConfigErr.Load() > 0 {
		t.Errorf("%d \"different configuration\" reader (attendu 0 — bug initial doit être éteint)",
			readerDiffConfigErr.Load())
	}
	if readerOK.Load() < 100 {
		t.Errorf("readerOK=%d trop faible (attendu ≥ 100 sur ~5s de sync × 10 goroutines)",
			readerOK.Load())
	}

	// Vérifier que les 20 matchs ont bien atterri dans shared via le Provider.
	finalDB, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get final: %v", err)
	}
	defer release()
	var finalCount int
	if err := finalDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&finalCount); err != nil {
		t.Fatalf("final count: %v", err)
	}
	expected := batches * matchPerBatch
	if finalCount != expected {
		t.Errorf("match_registry final = %d (attendu %d)", finalCount, expected)
	}

	// Provider doit être de retour à StateRO après le dernier swap.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State final = %v (attendu StateRO)", state)
	}

	// Détection grossière de fuite de goroutines.
	time.Sleep(50 * time.Millisecond) // laisse le runtime nettoyer
	goroutinesAfter := runtime.NumGoroutine()
	if delta := goroutinesAfter - goroutinesBefore; delta > 5 {
		t.Errorf("fuite goroutines détectée : avant=%d, après=%d (delta=%d)",
			goroutinesBefore, goroutinesAfter, delta)
	}
}

// classifyErr identifie les classes d'erreur critiques pour les compteurs.
func classifyErr(err error, catalog, diffConfig *atomic.Int64) {
	if err == nil {
		return
	}
	s := err.Error()
	if strings.Contains(s, "Catalog Error") || strings.Contains(s, "Table with name") ||
		strings.Contains(s, "Table not found") {
		catalog.Add(1)
	}
	if strings.Contains(s, "different configuration") ||
		strings.Contains(s, "Unique file handle") {
		diffConfig.Add(1)
	}
}
