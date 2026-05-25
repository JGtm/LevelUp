// Package v2 — e2e_test.go : tests d'intégration end-to-end du pipeline V2.
//
// Approche pragmatique : monte l'orchestrator avec les vraies
// implémentations adapter (KnownLoader DuckDB-réel, fetchers HaloClient
// avec mock contrôlé, persister + BatchQueue réel), et un faux worker
// qui ACK les batches immédiatement (vide le WAL). On observe les
// invariants à la frontière sans dépendre des Persisters V1 (testés
// dans le package persist).
//
// Pas de build tag : tourne dans `go test ./...`. Coût ~1s par test.
package v2

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/persist"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// e2eEnv encapsule les ressources d'un test E2E.
type e2eEnv struct {
	playerDBs map[string]*duckdbpkg.DB // gamertag → DB ouverte
	sharedDB  *duckdbpkg.DB
	queue     *persist.BatchQueue
	stopAcker func()
}

// setupE2EEnv prépare l'env : DuckDB temp pour shared + N players, schéma
// minimal, BatchQueue + worker fake qui ACK tout immédiatement.
func setupE2EEnv(t *testing.T, gamertags []string) *e2eEnv {
	t.Helper()
	tmpDir := t.TempDir()

	// Shared DB avec table match_participants (knownLoader cross-player).
	sharedPath := filepath.Join(tmpDir, "shared.duckdb")
	sharedDB, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	if _, err := sharedDB.SQLDb().Exec(`
		CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)
	`); err != nil {
		t.Fatalf("create match_participants: %v", err)
	}

	// Player DBs avec table player_match_enrichment (knownLoader source 1).
	playerDBs := make(map[string]*duckdbpkg.DB, len(gamertags))
	for _, gt := range gamertags {
		path := filepath.Join(tmpDir, "player_"+gt+".duckdb")
		db, err := duckdbpkg.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("open player %s: %v", gt, err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if _, err := db.SQLDb().Exec(`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY)`); err != nil {
			t.Fatalf("create player_match_enrichment %s: %v", gt, err)
		}
		playerDBs[gt] = db
	}

	// BatchQueue + faux worker qui ACK chaque batch immédiatement.
	// On test la SUBMISSION + drain (PendingCount→0), pas la persistence
	// DuckDB (testée dans persist package).
	walDir := filepath.Join(tmpDir, "wal")
	q, err := persist.NewBatchQueue(persist.BatchQueueConfig{
		WALDir:      walDir,
		ChanBufSize: 100,
	})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}

	ackerCtx, ackerCancel := context.WithCancel(context.Background())
	ackerDone := make(chan struct{})
	go func() {
		defer close(ackerDone)
		ch := q.Channel(persist.TargetShared)
		for {
			select {
			case <-ackerCtx.Done():
				return
			case batch, ok := <-ch:
				if !ok {
					return
				}
				_ = q.ACK(batch.BatchID)
				q.RecordPersistResult(true)
			}
		}
	}()
	stopAcker := func() {
		ackerCancel()
		select {
		case <-ackerDone:
		case <-time.After(2 * time.Second):
			t.Log("e2e acker shutdown timeout")
		}
		_ = q.Close()
	}
	t.Cleanup(stopAcker)

	return &e2eEnv{
		playerDBs: playerDBs,
		sharedDB:  sharedDB,
		queue:     q,
		stopAcker: stopAcker,
	}
}

// buildE2EOrchestrator monte un CycleOrchestrator V2 avec les 6 adapters
// connectés à l'env e2e + le mockNarrowClient passé en argument.
func buildE2EOrchestrator(t *testing.T, env *e2eEnv, client *mockNarrowClient, players []PlayerProfile) (*CycleOrchestratorImpl, *mockPostSyncRunner) {
	t.Helper()
	playerBySlug := make(map[string]PlayerProfile, len(players))
	for _, p := range players {
		playerBySlug[p.PlayerSlug] = p
	}

	playerDBOpener := func(_ context.Context, gamertag string) (*sql.DB, func(), error) {
		d, ok := env.playerDBs[gamertag]
		if !ok {
			return nil, nil, fmt.Errorf("playerDB %s absent", gamertag)
		}
		return d.SQLDb(), func() {}, nil
	}

	knownLoader := NewKnownLoader(playerDBOpener, func() *sql.DB { return env.sharedDB.SQLDb() })
	clientFactory := func(_, _ string) HaloClient { return client }
	matchListProvider := NewMatchListProvider(clientFactory, "matchmaking", 25, 20)
	sharedFetcher := NewSharedMatchFetcher(clientFactory)
	playerEnr := NewPlayerEnrichmentFetcher()
	persister := NewCycleBatchPersister("halo_infinite", env.queue, 5*time.Second)

	postSyncMock := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{},
	}
	for _, p := range players {
		postSyncMock.resultFor[p.PlayerSlug] = PlayerPostSyncResult{CitationsComputed: 0}
	}

	orch := NewCycleOrchestrator(knownLoader, matchListProvider, sharedFetcher,
		playerEnr, persister, postSyncMock, CycleConfig{
			FetchSharedParallelism: 4,
			FetchPlayerParallelism: 2,
			PostSyncParallelism:    2,
		})
	return orch, postSyncMock
}

// histEntry construit une syncpkg.MatchHistoryEntry à partir d'un match_id.
func histEntry(matchID string) syncpkg.MatchHistoryEntry {
	return syncpkg.MatchHistoryEntry{MatchID: matchID, StartTime: "2026-05-25T12:00:00.000Z"}
}

// histList construit une slice de MatchHistoryEntry depuis une liste d'IDs.
func histList(matchIDs ...string) []syncpkg.MatchHistoryEntry {
	out := make([]syncpkg.MatchHistoryEntry, len(matchIDs))
	for i, m := range matchIDs {
		out[i] = histEntry(m)
	}
	return out
}

// ─── Tests E2E ───────────────────────────────────────────────────────

func TestE2E_V2_FullCycleOrchestratesAllPhases(t *testing.T) {
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
		{Gamertag: "bob", XUID: "1000000000000002", PlayerSlug: "bob"},
	}
	env := setupE2EEnv(t, []string{"alice", "bob"})

	// alice : m1, m2, m3 ; bob : m2, m4 (m2 partagé → 4 uniques)
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m1", "m2", "m3"),
			"xuid(1000000000000002)": histList("m2", "m4"),
		},
		statsByMatch: map[string]map[string]any{
			"m1": {"placeholder": 1},
			"m2": {"placeholder": 2},
			"m3": {"placeholder": 3},
			"m4": {"placeholder": 4},
		},
	}

	orch, postSyncMock := buildE2EOrchestrator(t, env, client, players)
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("orch.Run err = %v", err)
	}

	// Invariant 1 — UniqueMatches dedup : exactement 4 (m1,m2,m3,m4).
	if res.UniqueMatches != 4 {
		t.Errorf("UniqueMatches = %d, want 4", res.UniqueMatches)
	}

	// Invariant 2 — Phase 3 : exactement 4 GetMatchStats (1/unique match).
	// C'est l'invariant CENTRAL de V2 (cross-player dedup).
	if got := client.statsCallCount.Load(); got != 4 {
		t.Errorf("GetMatchStats calls = %d, want 4 (cross-player dedup)", got)
	}

	// Invariant 3 — Phase 6 : exactement 1 post-sync/joueur.
	if got := postSyncMock.totalCalls.Load(); got != 2 {
		t.Errorf("PostSync calls = %d, want 2", got)
	}

	// Invariant 4 — Toutes les phases reportées.
	for _, ph := range []string{PhaseDiscovery, PhaseDedup, PhaseFetchShared, PhaseFetchPlayer, PhasePersist, PhasePostSync} {
		if _, ok := res.PhaseDurations[ph]; !ok {
			t.Errorf("PhaseDurations[%s] absent", ph)
		}
	}

	// Invariant 5 — PerPlayer : 2 joueurs présents avec status ok.
	if len(res.PerPlayer) != 2 {
		t.Errorf("PerPlayer len = %d, want 2", len(res.PerPlayer))
	}
	for slug, out := range res.PerPlayer {
		if out.Status != "ok" {
			t.Errorf("PerPlayer[%s].Status = %q, want ok", slug, out.Status)
		}
	}
}

func TestE2E_V2_KnownMatchesSkippedAtDiscovery(t *testing.T) {
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
	}
	env := setupE2EEnv(t, []string{"alice"})

	// Pré-seed m_old dans player_match_enrichment d'alice.
	if _, err := env.playerDBs["alice"].SQLDb().Exec(
		"INSERT INTO player_match_enrichment (match_id) VALUES ('m_old')",
	); err != nil {
		t.Fatalf("seed m_old: %v", err)
	}

	// API renvoie [m_new, m_old] → delta stop sur m_old → seul m_new unknown.
	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m_new", "m_old"),
		},
		statsByMatch: map[string]map[string]any{
			"m_new": {"placeholder": 1},
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if res.UniqueMatches != 1 {
		t.Errorf("UniqueMatches = %d, want 1", res.UniqueMatches)
	}
	if got := client.statsCallCount.Load(); got != 1 {
		t.Errorf("GetMatchStats calls = %d, want 1", got)
	}
}

func TestE2E_V2_PartialFailureFetchSharedDoesNotBlock(t *testing.T) {
	// m2 retourne stats nil (simule erreur silencieuse au parsing
	// downstream) ; m1 et m3 OK. Le cycle doit terminer sans erreur.
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
	}
	env := setupE2EEnv(t, []string{"alice"})

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m1", "m2", "m3"),
		},
		statsByMatch: map[string]map[string]any{
			"m1": {"placeholder": 1},
			"m3": {"placeholder": 3},
			// m2 absent → mock retourne nil sans erreur (parsing downstream skip).
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("err = %v (m2 manquant ne devrait pas tuer le cycle)", err)
	}
	if res.UniqueMatches != 3 {
		t.Errorf("UniqueMatches = %d, want 3", res.UniqueMatches)
	}
	if got := client.statsCallCount.Load(); got != 3 {
		t.Errorf("GetMatchStats calls = %d, want 3 (tous appelés en parallèle)", got)
	}
}

func TestE2E_V2_IdempotencyRunCycleTwice(t *testing.T) {
	// 2 cycles avec mêmes inputs et mêmes mocks → 2e cycle doit refaire
	// les appels (pas de cache cycle-to-cycle) mais ne doit pas planter.
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
	}
	env := setupE2EEnv(t, []string{"alice"})

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m1"),
		},
		statsByMatch: map[string]map[string]any{
			"m1": {"placeholder": 1},
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)
	_, err1 := orch.Run(context.Background(), players)
	if err1 != nil {
		t.Fatalf("cycle 1 err = %v", err1)
	}
	calls1 := client.statsCallCount.Load()

	_, err2 := orch.Run(context.Background(), players)
	if err2 != nil {
		t.Fatalf("cycle 2 err = %v", err2)
	}
	calls2 := client.statsCallCount.Load()

	// 2e cycle doit refaire au moins 1 appel (re-discovery), mais comme
	// les parsing tests précédents skip silencieusement (Stats sans
	// MatchId), le m1 n'est pas réellement persisté dans la DB. Le 2e
	// cycle redécouvre donc m1 et le re-fetche.
	if calls2 < calls1+1 {
		t.Errorf("cycle 2 calls=%d, cycle 1=%d (au moins 1 nouvel appel attendu)", calls2, calls1)
	}
}

func TestE2E_V2_QueueDrainSuccessOnEmptyCycle(t *testing.T) {
	// 0 matchs à insérer → drain immédiat, pas de timeout.
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
	}
	env := setupE2EEnv(t, []string{"alice"})

	// API renvoie [m_old] qui est dans known → 0 nouveau.
	if _, err := env.playerDBs["alice"].SQLDb().Exec(
		"INSERT INTO player_match_enrichment (match_id) VALUES ('m_old')",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m_old"),
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)
	start := time.Now()
	res, err := orch.Run(context.Background(), players)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.UniqueMatches != 0 {
		t.Errorf("UniqueMatches = %d, want 0", res.UniqueMatches)
	}
	// Le cycle doit être rapide (pas d'API fetch, pas de drain timeout).
	if elapsed > 500*time.Millisecond {
		t.Errorf("cycle trop lent sur 0 matchs : %v", elapsed)
	}
}
