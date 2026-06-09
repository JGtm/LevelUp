//go:build integration

// Package persist — e2e_test.go : test de bout-en-bout de l'architecture
// Collect→Persist. Valide que le wiring complet (BatchQueue + Worker +
// SharedPersister) fonctionne avec une vraie DuckDB.
//
// Couverture :
//
//  1. Submit de N batches → tous persistés dans la DB.
//  2. WAL files supprimés post-ACK (durabilité OK).
//  3. Property INSERT-only : re-submit d'un même match_id ne provoque pas
//     d'erreur, ne modifie pas les rows existantes.
//  4. Crash recovery : pré-seed WAL files, RecoverPending() les re-push,
//     worker les persiste comme attendu.

package persist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
)

// ─── Helper E2E ────────────────────────────────────────────────────────────

// e2eSetup prépare un environnement complet : DuckDB shared + queue + worker.
// Le worker est démarré sur une goroutine, terminé via cancel() à la fin du test.
func e2eSetup(t *testing.T) (db *sql.DB, q *BatchQueue, persister *SharedPersister, walDir string, cancel context.CancelFunc, okCounter, errCounter *atomic.Int64) {
	t.Helper()
	db = openSharedTestDB(t)

	walDir = t.TempDir()
	var err error
	q, err = NewBatchQueue(BatchQueueConfig{WALDir: walDir, ChanBufSize: 64})
	if err != nil {
		t.Fatalf("NewBatchQueue: %v", err)
	}

	persister = NewSharedPersister(db)
	w := NewWorker("e2e-shared", q, TargetShared, persister)

	okCounter = &atomic.Int64{}
	errCounter = &atomic.Int64{}
	w.OnPersistOK = func() { okCounter.Add(1) }
	w.OnPersistError = func(error) { errCounter.Add(1) }

	ctx, ccl := context.WithCancel(context.Background())
	cancel = ccl
	go func() { _ = w.Run(ctx) }()

	t.Cleanup(func() {
		cancel()
		_ = q.Close()
	})
	return db, q, persister, walDir, cancel, okCounter, errCounter
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("timeout en attendant %s", msg)
	}
}

// ─── Test 1 : E2E flow nominal — N batches → DB ───────────────────────────

func TestE2E_Pipeline_PersistsAllBatches(t *testing.T) {
	db, q, _, walDir, _, okCounter, errCounter := e2eSetup(t)

	const N = 5
	for i := 0; i < N; i++ {
		id := "e2e_" + string(rune('a'+i))
		batch := helperBuildSampleBatch(id, "1111", "Alice")
		if err := q.Submit(batch); err != nil {
			t.Fatalf("Submit[%d]: %v", i, err)
		}
	}

	waitFor(t, func() bool { return okCounter.Load() == N }, 3*time.Second,
		"worker persiste les 5 batches")

	if errCounter.Load() != 0 {
		t.Errorf("OnPersistError appelé %d fois (devrait être 0)", errCounter.Load())
	}

	// Vérifier que les match_id sont en DB.
	var dbCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id LIKE 'e2e_%'`).Scan(&dbCount); err != nil {
		t.Fatal(err)
	}
	if dbCount != N {
		t.Errorf("match_registry a %d rows, want %d", dbCount, N)
	}

	// Vérifier que les WAL files ont été supprimés (ACK).
	entries, _ := os.ReadDir(walDir)
	jsonCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			jsonCount++
		}
	}
	if jsonCount != 0 {
		t.Errorf("WAL dir contient encore %d .json files (devraient être ACKés)", jsonCount)
	}
}

// ─── Test 2 : INSERT-only — re-submit d'un même match_id ──────────────────

func TestE2E_Pipeline_INSERTOnlyOnReSubmit(t *testing.T) {
	db, q, _, _, _, okCounter, _ := e2eSetup(t)

	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	// 1er submit
	b1 := helperBuildSampleBatch("e2e_dup", "1111", "Alice")
	if err := q.Submit(b1); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return okCounter.Load() == 1 }, 2*time.Second,
		"1er batch persisté")

	// 2e submit du même match_id, kills modifiés
	b2 := helperBuildSampleBatch("e2e_dup", "1111", "Alice")
	b2.Shared.Participants = []domain.MatchParticipantRow{
		{MatchID: "e2e_dup", XUID: "1111", Gamertag: strPtr("Alice"), Kills: intPtr(99999)},
	}
	if err := q.Submit(b2); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return okCounter.Load() == 2 }, 2*time.Second,
		"2e batch traité (skip silencieux car déjà persisté)")

	// Le kills initial (10) doit avoir survécu.
	var kills int
	if err := db.QueryRow(`SELECT kills FROM match_participants WHERE match_id = ? AND xuid = ?`,
		"e2e_dup", "1111").Scan(&kills); err != nil {
		t.Fatal(err)
	}
	if kills != 10 {
		t.Errorf("kills = %d, want 10 (INSERT-only)", kills)
	}
}

// ─── Test 3 : Crash recovery via RecoverPending ────────────────────────────

func TestE2E_Pipeline_CrashRecovery(t *testing.T) {
	// Setup : créer un WAL dir pré-seed avant de démarrer la queue/worker.
	// Le nom du fichier WAL DOIT matcher batch.BatchID (contrat ACK).
	walDir := t.TempDir()
	preSeeded := []string{"recov_001", "recov_002", "recov_003"}
	for _, id := range preSeeded {
		batch := helperBuildSampleBatch(id, "1111", "Alice")
		batch.BatchID = id // override pour matcher le filename
		data, err := json.Marshal(batch)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(walDir, id+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	db := openSharedTestDB(t)
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: walDir, ChanBufSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	persister := NewSharedPersister(db)
	w := NewWorker("e2e-recovery", q, TargetShared, persister)
	okCount := &atomic.Int64{}
	w.OnPersistOK = func() { okCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = w.Run(ctx) }()

	// Au boot : recover les WAL pré-seed
	if err := q.RecoverPending(); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}

	waitFor(t, func() bool { return okCount.Load() == int64(len(preSeeded)) },
		3*time.Second, "tous les WAL recoverés persistés")

	// Vérifier les rows en DB
	for _, id := range preSeeded {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("match %s : %d rows registry, want 1", id, n)
		}
	}

	// WAL files supprimés
	for _, id := range preSeeded {
		if _, err := os.Stat(filepath.Join(walDir, id+".json")); !os.IsNotExist(err) {
			t.Errorf("WAL %s doit être supprimé post-ACK", id)
		}
	}
}

// ─── Test [G] : FATAL mid-batch → ROLLBACK (0 row partielle) + WAL survit + recovery ──
//
// fatalMidTxPersister simule un FATAL DuckDB APRÈS insertion partielle dans la
// même transaction : il ouvre une vraie TX, INSERT la row match_registry, puis
// retourne une erreur FATAL sans COMMIT → le defer Rollback doit tout annuler.
// Couvre l'invariant central post-incident ART (2026-05-24) : un crash mid-TX
// ne laisse AUCUNE row partielle, le WAL survit (pas d'ACK), et la recovery
// rejoue le batch proprement une fois la persistance rétablie.
type fatalMidTxPersister struct {
	mu       sync.Mutex
	db       txBeginner
	real     *SharedPersister
	failNext bool
}

func (p *fatalMidTxPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	p.mu.Lock()
	fail := p.failNext
	p.mu.Unlock()
	if !fail {
		return p.real.Persist(ctx, batch)
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // doit annuler l'INSERT partiel ci-dessous
	if batch.Shared.Match != nil {
		if err := persistMatchRegistry(ctx, tx, batch.Shared.Match, false); err != nil {
			return err
		}
	}
	// FATAL mid-batch : on retourne SANS COMMIT → rollback.
	return errors.New("FATAL: database has been invalidated")
}

func (p *fatalMidTxPersister) setFailNext(v bool) {
	p.mu.Lock()
	p.failNext = v
	p.mu.Unlock()
}

func TestE2E_FatalMidBatch_RollbackThenRecovery(t *testing.T) {
	db := openSharedTestDB(t)
	walDir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: walDir, ChanBufSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	p := &fatalMidTxPersister{db: db, real: NewSharedPersister(db), failNext: true}
	w := NewWorker("fatal-midtx", q, TargetShared, p)
	w.retryBaseDelay = time.Millisecond
	var okCount, errCount atomic.Int64
	w.OnPersistOK = func() { okCount.Add(1) }
	w.OnPersistError = func(error) { errCount.Add(1) }

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = w.Run(ctx) }()

	batch := helperBuildSampleBatch("fatal_mid", "1111", "Alice")
	walFile := filepath.Join(walDir, batch.BatchID+".json")
	if err := q.Submit(batch); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool { return errCount.Load() == 1 }, 3*time.Second,
		"le FATAL mid-batch remonte une erreur")

	// ROLLBACK : aucune row partielle malgré l'INSERT registry dans la TX.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, "fatal_mid").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("rollback raté : %d row(s) partielle(s) en DB, want 0", n)
	}
	// WAL survit (pas d'ACK sur échec).
	if _, err := os.Stat(walFile); err != nil {
		t.Errorf("WAL doit survivre au FATAL : %v", err)
	}

	// Persistance rétablie → recovery rejoue le batch jusqu'au succès complet.
	p.setFailNext(false)
	if err := q.RecoverPending(); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}
	waitFor(t, func() bool { return okCount.Load() == 1 }, 3*time.Second,
		"recovery persiste le batch précédemment FATAL")

	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, "fatal_mid").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("après recovery : %d row registry, want 1", n)
	}
	if _, err := os.Stat(walFile); !os.IsNotExist(err) {
		t.Errorf("WAL doit être ACKé après recovery réussie")
	}
}

// ─── Test 4 : Volumetric — 200 batches concurrents soumis rapidement ──────

func TestE2E_Pipeline_VolumetricSubmit(t *testing.T) {
	db, q, _, _, _, okCounter, errCounter := e2eSetup(t)

	const N = 200
	for i := 0; i < N; i++ {
		batch := helperBuildSampleBatch(
			fmt.Sprintf("vol_%05d", i),
			fmt.Sprintf("xuid_%05d", i%5), // 5 joueurs cycliques
			fmt.Sprintf("Player%d", i%5),
		)
		if err := q.Submit(batch); err != nil {
			t.Fatalf("Submit[%d]: %v", i, err)
		}
	}

	waitFor(t, func() bool { return okCounter.Load() == N }, 10*time.Second,
		"200 batches persistés")

	if errCounter.Load() != 0 {
		t.Errorf("OnPersistError appelé %d fois (devrait être 0)", errCounter.Load())
	}

	var dbCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id LIKE 'vol_%'`).Scan(&dbCount); err != nil {
		t.Fatal(err)
	}
	if dbCount != N {
		t.Errorf("match_registry vol_* : %d rows, want %d", dbCount, N)
	}
}
