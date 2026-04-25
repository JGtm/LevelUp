package assets

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// stubIndex est un IndexStore de test.
type stubIndex struct {
	available  bool
	entry      *IndexEntry
	lookupErr  error
	persisted  []IndexEntry
	persistErr error
}

func (s *stubIndex) Available(_ context.Context) bool { return s.available }

func (s *stubIndex) LookupIndex(_ context.Context, _ Ref) (*IndexEntry, error) {
	return s.entry, s.lookupErr
}

func (s *stubIndex) PersistIndex(_ context.Context, _ Ref, e IndexEntry) error {
	if s.persistErr != nil {
		return s.persistErr
	}
	s.persisted = append(s.persisted, e)
	return nil
}

func (s *stubIndex) EnsureTable(_ context.Context) error { return nil }

func TestWriteQueue_Enqueue_Persists(t *testing.T) {
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	entry := IndexEntry{Ref: ref, URL: "http://cdn.example.com/1.png"}

	q.Enqueue(ref, entry)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	q.Shutdown(ctx)

	if len(idx.persisted) != 1 {
		t.Errorf("attendu 1 entrée persistée, got %d", len(idx.persisted))
	}
}

func TestWriteQueue_Enqueue_MultipleJobs(t *testing.T) {
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)

	const n = 10
	for i := range n {
		ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: string(rune('A' + i))}
		q.Enqueue(ref, IndexEntry{Ref: ref})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	q.Shutdown(ctx)

	if len(idx.persisted) != n {
		t.Errorf("attendu %d entrées, got %d", n, len(idx.persisted))
	}
}

func TestWriteQueue_Overflow_DropsJob(t *testing.T) {
	// Index très lent pour saturer le channel.
	var processed atomic.Int32
	slowIdx := &slowStubIndex{delay: 10 * time.Millisecond, base: &stubIndex{available: true}, processed: &processed}
	q := NewWriteQueue(slowIdx, nil)

	// Envoyer plus de jobs que la capacité du channel (writeQueueCap=256)
	// Tous doivent être acceptés ou droppés sans paniquer.
	for i := range 300 {
		ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: string(rune('A' + (i % 26)))}
		q.Enqueue(ref, IndexEntry{Ref: ref})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q.Shutdown(ctx)
}

// slowStubIndex simule un index lent pour tester l'overflow.
type slowStubIndex struct {
	delay     time.Duration
	base      *stubIndex
	processed *atomic.Int32
}

func (s *slowStubIndex) Available(_ context.Context) bool { return true }

func (s *slowStubIndex) LookupIndex(ctx context.Context, ref Ref) (*IndexEntry, error) {
	return s.base.LookupIndex(ctx, ref)
}

func (s *slowStubIndex) PersistIndex(ctx context.Context, ref Ref, e IndexEntry) error {
	time.Sleep(s.delay)
	s.processed.Add(1)
	return s.base.PersistIndex(ctx, ref, e)
}

func (s *slowStubIndex) EnsureTable(_ context.Context) error { return nil }

func TestWriteQueue_Shutdown_Idempotent(t *testing.T) {
	// Shutdown sur une queue vide ne doit pas bloquer.
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	q.Shutdown(ctx)
	// Pas de panique = test réussi.
}

func TestWriteQueue_PersistError_Drops(t *testing.T) {
	idx := &stubIndex{available: true, persistErr: errors.New("disk full")}
	q := NewWriteQueue(idx, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	q.Enqueue(ref, IndexEntry{Ref: ref})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	q.Shutdown(ctx)
	// Pas de panique, job droppé après tentatives.
}

func TestBackoffDuration_Progression(t *testing.T) {
	prev := backoffDuration(0)
	for i := 1; i < writeMaxRetry; i++ {
		next := backoffDuration(i)
		if next <= prev && next < writeBackoffMax {
			t.Errorf("backoff[%d]=%v <= backoff[%d]=%v", i, next, i-1, prev)
		}
		prev = next
	}
}

func TestBackoffDuration_Cap(t *testing.T) {
	// À un grand attempt, le backoff ne dépasse pas writeBackoffMax.
	d := backoffDuration(20)
	if d > writeBackoffMax {
		t.Errorf("backoff dépassé : %v > %v", d, writeBackoffMax)
	}
}
