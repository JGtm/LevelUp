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
	"os"
	"path/filepath"
	"strings"
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
		case batch := <-q.Channel(TargetShared):
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
	<-q.Channel(TargetShared)
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
	<-q.Channel(TargetShared)

	// 2e appel — ne doit pas re-push (le WAL n'a pas été ACKé, donc il
	// devrait être re-pushed... mais c'est une politique à définir).
	// Pour ce test, on assume que RecoverPending est appelé UNIQUEMENT au
	// boot. Multiple calls = ouvre la même fenêtre de récupération.
	if err := q.RecoverPending(); err != nil {
		t.Errorf("2e RecoverPending devrait OK : %v", err)
	}
	// Devrait y avoir à nouveau le batch dans le channel (WAL toujours là)
	select {
	case b := <-q.Channel(TargetShared):
		if b.BatchID != "idem001" {
			t.Errorf("2e recovery a renvoyé batch_id=%q", b.BatchID)
		}
	case <-time.After(200 * time.Millisecond):
		// Acceptable aussi si on dédupe en mémoire — à clarifier en impl.
		// Pour ce test, on tolère soit "re-push" soit "dédupe silencieux".
		t.Log("2e recovery n'a pas re-pushed (dedupe interne, OK)")
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
