// Package persist — queue_test.go : tests TDD pour BatchQueue.
//
// Contrat à valider AVANT impl :
//
//  1. Submit écrit le batch dans WAL JSON sur disque AVANT de pousser dans
//     le channel (durabilité crash-safe).
//  2. RecoverPending lit walDir/*.json au boot, re-pousse dans les
//     channels, batches existants traités au prochain cycle worker.
//  3. ACK supprime le WAL file (preuve que persist a réussi).
//  4. WAL corrompu (JSON invalide) → déplacé dans walDir/corrupted/ + log.
//  5. Submit bloque si channel plein (backpressure naturelle).
//  6. Submit retourne erreur si queue déjà fermée.
//  7. RecoverPending est idempotent (peut être appelé plusieurs fois).

package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// helperNewBatch crée un MatchBatch minimal pour les tests (1 match, 1
// participant). Réutilisé partout pour ne pas dupliquer la création.
func helperNewBatch(t *testing.T, batchID, gamertag string) *MatchBatch {
	t.Helper()
	intPtr := func(v int) *int { return &v }
	b := NewBatchBuilder("halo_infinite", gamertag, "xuid_"+gamertag, "test")
	b.AddParticipants([]domain.MatchParticipantRow{
		{MatchID: "m1", XUID: "xuid_" + gamertag, Kills: intPtr(10), Deaths: intPtr(5)},
	})
	out := b.Build()
	out.BatchID = batchID // override pour test reproductible
	return out
}

// ─── Régression D1-1 : Submit concurrent à Close ne panique jamais ─────────
//
// Avant le fix, Submit relâchait le RLock AVANT le send `q.chMain <- batch` ;
// Close pouvait close(chMain) dans cette fenêtre → panic "send on closed
// channel" (atteignable au shutdown si un cycle de sync straggler Submit après
// le Wait scheduler borné à 3s). Le fix (sendWG attendu par Close avant close)
// garantit qu'aucun send ne touche un channel fermé. Lancer sous -race.
func TestBatchQueue_ConcurrentSubmitClose_NoPanic(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 16})
	if err != nil {
		t.Fatalf("NewBatchQueue: %v", err)
	}

	// Drain continu pour que les Submit ne bloquent pas indéfiniment sur un
	// channel plein ; s'arrête quand Close ferme chMain (range termine).
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range q.Channel() {
		}
	}()

	// Pré-construire les batches hors goroutines (helperNewBatch touche t).
	const n = 256
	batches := make([]*MatchBatch, n)
	for i := range batches {
		batches[i] = helperNewBatch(t, fmt.Sprintf("race_%03d", i), "Alice")
	}

	var wg sync.WaitGroup
	unexpected := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(b *MatchBatch) {
			defer wg.Done()
			// Seul ErrQueueClosed est toléré ; une panic ferait planter le test.
			if err := q.Submit(b); err != nil && !errors.Is(err, ErrQueueClosed) {
				unexpected <- err
			}
		}(batches[i])
	}

	// Fermer pendant que les Submit sont en vol — c'est la fenêtre de course.
	time.Sleep(500 * time.Microsecond)
	if err := q.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	wg.Wait()
	<-drained
	close(unexpected)
	for e := range unexpected {
		t.Errorf("Submit a retourné une erreur inattendue : %v", e)
	}

	// Close idempotent.
	if err := q.Close(); err != nil {
		t.Errorf("2e Close doit être no-op, got %v", err)
	}
}

// ─── Test 1 : Submit écrit le WAL avant de pousser ────────────────────────

func TestBatchQueue_Submit_WritesWALBeforeChannelPush(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir:      dir,
		ChanBufSize: 10,
	})
	if err != nil {
		t.Fatalf("NewBatchQueue: %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })

	batch := helperNewBatch(t, "batch001", "Alice")
	if err := q.Submit(batch); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// WAL file doit exister immédiatement après Submit.
	walPath := filepath.Join(dir, "batch001.json")
	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("WAL file absent post-Submit (durabilité cassée) : %v", err)
	}

	// Le content du WAL doit être un MatchBatch valide.
	data, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatal(err)
	}
	var reread MatchBatch
	if err := json.Unmarshal(data, &reread); err != nil {
		t.Fatalf("WAL content invalide : %v", err)
	}
	if reread.BatchID != "batch001" {
		t.Errorf("WAL batch_id mismatch : %q", reread.BatchID)
	}
}

// ─── Test 2 : RecoverPending lit le WAL et re-pousse ──────────────────────

func TestBatchQueue_RecoverPending_RepushesExistingWAL(t *testing.T) {
	dir := t.TempDir()

	// Pré-seed : écrire 2 batches directement dans le WAL (simule crash avant ACK)
	for _, id := range []string{"recov001", "recov002"} {
		batch := helperNewBatch(t, id, "Bob")
		data, _ := json.Marshal(batch)
		if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	q, err := NewBatchQueue(BatchQueueConfig{
		WALDir:      dir,
		ChanBufSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	if err := q.RecoverPending(); err != nil {
		t.Fatalf("RecoverPending: %v", err)
	}

	// Les 2 batches doivent être lisibles depuis le channel.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case batch := <-q.Channel():
			seen[batch.BatchID] = true
		case <-ctx.Done():
			t.Fatalf("timeout en lisant le channel (vu %d/2)", len(seen))
		}
	}
	if !seen["recov001"] || !seen["recov002"] {
		t.Errorf("batches manqués post-recovery : %+v", seen)
	}
}

// ─── Test 3 : ACK supprime le WAL ──────────────────────────────────────────

func TestBatchQueue_ACK_RemovesWALFile(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	batch := helperNewBatch(t, "ack001", "Carol")
	_ = q.Submit(batch)

	walPath := filepath.Join(dir, "ack001.json")
	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("WAL doit exister pré-ACK : %v", err)
	}

	if err := q.ACK("ack001"); err != nil {
		t.Fatalf("ACK: %v", err)
	}

	if _, err := os.Stat(walPath); !os.IsNotExist(err) {
		t.Errorf("WAL doit être supprimé post-ACK, err=%v", err)
	}
}

// ─── Test 4 : WAL corrompu → dossier corrupted/ ────────────────────────────

func TestBatchQueue_RecoverPending_CorruptedWAL_MovesToCorruptedDir(t *testing.T) {
	dir := t.TempDir()

	// Écrire un fichier JSON invalide.
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	if err := q.RecoverPending(); err != nil {
		t.Fatalf("RecoverPending devrait skip les corrompus sans erreur : %v", err)
	}

	// Vérifier que le fichier est dans corrupted/
	corruptedPath := filepath.Join(dir, "corrupted", "broken.json")
	if _, err := os.Stat(corruptedPath); err != nil {
		t.Errorf("WAL corrompu non déplacé : %v", err)
	}
	// Original n'existe plus
	if _, err := os.Stat(filepath.Join(dir, "broken.json")); !os.IsNotExist(err) {
		t.Errorf("WAL corrompu original toujours présent")
	}
}

// ─── Test 5 : Submit bloque si channel plein ──────────────────────────────

func TestBatchQueue_Submit_BlocksWhenChannelFull(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Remplir le channel (size=2)
	for i := 0; i < 2; i++ {
		if err := q.Submit(helperNewBatch(t, "bp"+string(rune('0'+i)), "Dave")); err != nil {
			t.Fatal(err)
		}
	}

	// 3e Submit doit bloquer. Tester via timeout court.
	done := make(chan error, 1)
	go func() {
		done <- q.Submit(helperNewBatch(t, "bp_extra", "Dave"))
	}()

	select {
	case err := <-done:
		t.Errorf("Submit n'a pas bloqué : err=%v (back-pressure cassée)", err)
	case <-time.After(100 * time.Millisecond):
		// OK : Submit bloque comme attendu
	}

	// Drain pour débloquer le Submit en attente
	<-q.Channel()
	<-done // Le Submit en attente termine maintenant
}

// ─── Test 6 : Submit après Close retourne erreur ───────────────────────────

func TestBatchQueue_Submit_AfterClose_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	_ = q.Close()

	err = q.Submit(helperNewBatch(t, "after_close", "Eve"))
	if err == nil {
		t.Error("Submit après Close devrait retourner une erreur")
	}
	if !errors.Is(err, ErrQueueClosed) {
		t.Errorf("err = %v, want ErrQueueClosed", err)
	}
}

// ─── Test 7 : RecoverPending idempotent ───────────────────────────────────

func TestBatchQueue_RecoverPending_Idempotent(t *testing.T) {
	dir := t.TempDir()
	batch := helperNewBatch(t, "idem001", "Frank")
	data, _ := json.Marshal(batch)
	_ = os.WriteFile(filepath.Join(dir, "idem001.json"), data, 0o644)

	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// 1er appel
	if err := q.RecoverPending(); err != nil {
		t.Fatal(err)
	}
	// Lire le batch
	<-q.Channel()

	// 2e appel — ne doit pas re-push (le WAL n'a pas été ACKé, donc il
	// devrait être re-pushed... mais c'est une politique à définir).
	// Pour ce test, on assume que RecoverPending est appelé UNIQUEMENT au
	// boot. Multiple calls = ouvre la même fenêtre de récupération.
	if err := q.RecoverPending(); err != nil {
		t.Errorf("2e RecoverPending devrait OK : %v", err)
	}
	// Devrait y avoir à nouveau le batch dans le channel (WAL toujours là)
	select {
	case b := <-q.Channel():
		if b.BatchID != "idem001" {
			t.Errorf("2e recovery a renvoyé batch_id=%q", b.BatchID)
		}
	case <-time.After(200 * time.Millisecond):
		// Acceptable aussi si on dédupe en mémoire — à clarifier en impl.
		// Pour ce test, on tolère soit "re-push" soit "dédupe silencieux".
		t.Log("2e recovery n'a pas re-pushed (dedupe interne, OK)")
	}
}

// ─── Test PendingCount + Drain ─────────────────────────────────────────────

func TestBatchQueue_PendingCount_CountsJSONFiles(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Submit 3 batches (Submit écrit le WAL)
	for _, id := range []string{"p1", "p2", "p3"} {
		if err := q.Submit(helperNewBatch(t, id, "Alice")); err != nil {
			t.Fatal(err)
		}
	}

	n, err := q.PendingCount()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("PendingCount = %d, want 3", n)
	}

	// ACK 1 — pending doit baisser à 2
	if err := q.ACK("p2"); err != nil {
		t.Fatal(err)
	}
	n, _ = q.PendingCount()
	if n != 2 {
		t.Errorf("PendingCount post-ACK = %d, want 2", n)
	}
}

func TestBatchQueue_PendingCount_IgnoresCorruptedDir(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Crée 1 WAL valide + 1 dans corrupted/
	_ = q.Submit(helperNewBatch(t, "valid", "Alice"))
	corrDir := filepath.Join(dir, "corrupted")
	_ = os.MkdirAll(corrDir, 0o755)
	_ = os.WriteFile(filepath.Join(corrDir, "bad.json"), []byte(`{`), 0o644)

	n, _ := q.PendingCount()
	if n != 1 {
		t.Errorf("PendingCount = %d, want 1 (exclut corrupted/)", n)
	}
}

func TestBatchQueue_Drain_WaitsForAllACKed(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	_ = q.Submit(helperNewBatch(t, "d1", "Alice"))
	_ = q.Submit(helperNewBatch(t, "d2", "Alice"))

	// ACK en background pour simuler les workers
	go func() {
		time.Sleep(80 * time.Millisecond)
		_ = q.ACK("d1")
		time.Sleep(80 * time.Millisecond)
		_ = q.ACK("d2")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Drain(ctx); err != nil {
		t.Fatalf("Drain: %v", err)
	}

	n, _ := q.PendingCount()
	if n != 0 {
		t.Errorf("post-Drain : PendingCount = %d, want 0", n)
	}
}

func TestBatchQueue_Drain_RespectsContextCancel(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	_ = q.Submit(helperNewBatch(t, "stuck", "Alice"))
	// Pas d'ACK → Drain doit bloquer jusqu'à timeout ctx

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = q.Drain(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Drain devrait retourner ctx.Err() (timeout)")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("Drain a retourné trop tôt (%v)", elapsed)
	}
}

// ─── Test PurgeOldWAL ──────────────────────────────────────────────────────

func TestBatchQueue_PurgeOldWAL_RemovesFilesOlderThanMaxAge(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Crée 2 WAL files : un vieux + un récent.
	oldPath := filepath.Join(dir, "old.json")
	if err := os.WriteFile(oldPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldPath, time.Now().Add(-10*24*time.Hour), time.Now().Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	recentPath := filepath.Join(dir, "recent.json")
	if err := os.WriteFile(recentPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	purged, err := q.PurgeOldWAL(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("PurgeOldWAL: %v", err)
	}
	if purged != 1 {
		t.Errorf("purged = %d, want 1", purged)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("vieux WAL devrait être supprimé, err=%v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Errorf("WAL récent doit être préservé, err=%v", err)
	}
}

func TestBatchQueue_PurgeOldWAL_HandlesCorruptedDir(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Crée corrupted/ avec un vieux file.
	corruptedDir := filepath.Join(dir, "corrupted")
	if err := os.MkdirAll(corruptedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldCorrupted := filepath.Join(corruptedDir, "bad.json")
	if err := os.WriteFile(oldCorrupted, []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldCorrupted, time.Now().Add(-30*24*time.Hour), time.Now().Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	purged, err := q.PurgeOldWAL(7 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if purged != 1 {
		t.Errorf("purged = %d, want 1 (corrupted/bad.json)", purged)
	}
	if _, err := os.Stat(oldCorrupted); !os.IsNotExist(err) {
		t.Errorf("vieux WAL corrompu devrait être supprimé")
	}
}

func TestBatchQueue_PurgeOldWAL_EmptyDir_NoOp(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	purged, err := q.PurgeOldWAL(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("PurgeOldWAL empty dir: %v", err)
	}
	if purged != 0 {
		t.Errorf("purged = %d, want 0 (dir vide)", purged)
	}
}

// ─── Test 8 : ACK d'un batch inexistant → no-op, pas d'erreur ────────────

func TestBatchQueue_ACK_NonExistent_NoError(t *testing.T) {
	dir := t.TempDir()
	q, err := NewBatchQueue(BatchQueueConfig{WALDir: dir, ChanBufSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	if err := q.ACK("nonexistent"); err != nil {
		// Comportement attendu : ACK est idempotent
		if !strings.Contains(err.Error(), "no such file") {
			// Tolérance : peut retourner os.IsNotExist mais pas une autre erreur
			t.Errorf("ACK inexistant a retourné une erreur inattendue : %v", err)
		}
	}
}
