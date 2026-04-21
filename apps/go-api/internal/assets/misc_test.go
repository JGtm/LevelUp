package assets

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NoopMetrics — vérifie que les méthodes ne paniquent pas.
// ---------------------------------------------------------------------------

func TestNoopMetrics_AllMethodsNoPanic(t *testing.T) {
	m := NoopMetrics{}
	m.IncHit(KindMedalImage, SourceLocalFile)
	m.IncMiss(KindMedalImage)
	m.IncFetchError(KindMedalImage)
	m.IncIndexUnavailable()
	m.IncIndexWriteDropped(KindMedalImage)
	m.IncIndexWriteOverflow()
	m.ObserveLatency(KindMedalImage, SourceRemote, time.Millisecond)
}

// ---------------------------------------------------------------------------
// entryToPayload — branche par type de payload et kind.
// ---------------------------------------------------------------------------

func TestEntryToPayload_URLEntry_NonBinaryKind(t *testing.T) {
	store := &stubBinaryStore{hit: nil}
	r := NewDefaultResolver(store, nil, nil, nil, nil)

	ref := Ref{Kind: KindMedalMetadata, TitleID: "hi", ID: "meta"} // non-binary
	entry := &IndexEntry{URL: "https://cdn.example.com/meta.json"}
	p := r.entryToPayload(ref, entry)
	u, ok := p.(URLPayload)
	if !ok {
		t.Fatalf("attendu URLPayload, got %T", p)
	}
	if u.URL != "https://cdn.example.com/meta.json" {
		t.Errorf("URL inattendue: %q", u.URL)
	}
}

func TestEntryToPayload_JSONEntry(t *testing.T) {
	r := NewDefaultResolver(nil, nil, nil, nil, nil)
	ref := Ref{Kind: KindMedalMetadata, TitleID: "hi", ID: "meta"}
	entry := &IndexEntry{RawJSON: []byte(`{"medals":[]}`)}
	p := r.entryToPayload(ref, entry)
	if _, ok := p.(JSONPayload); !ok {
		t.Fatalf("attendu JSONPayload, got %T", p)
	}
}

func TestEntryToPayload_BinaryURLFallback(t *testing.T) {
	// Kind binaire, LocalPath vide, URL non vide → URLPayload.
	store := &stubBinaryStore{hit: nil}
	r := NewDefaultResolver(store, nil, nil, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"} // binary kind
	entry := &IndexEntry{URL: "https://cdn.example.com/medal.png"}
	p := r.entryToPayload(ref, entry)
	u, ok := p.(URLPayload)
	if !ok {
		t.Fatalf("attendu URLPayload, got %T", p)
	}
	if u.URL != "https://cdn.example.com/medal.png" {
		t.Errorf("URL inattendue: %q", u.URL)
	}
}

// ---------------------------------------------------------------------------
// WriteQueue — retry sur lock error.
// ---------------------------------------------------------------------------

// lockRetryIndex est un IndexStore qui retourne une lock error les N premières fois.
type lockRetryIndex struct {
	failCount int
	calls     atomic.Int32
	base      *stubIndex
}

func (l *lockRetryIndex) Available(_ context.Context) bool { return true }
func (l *lockRetryIndex) LookupIndex(_ context.Context, _ Ref) (*IndexEntry, error) {
	return nil, nil
}
func (l *lockRetryIndex) PersistIndex(_ context.Context, _ Ref, e IndexEntry) error {
	n := int(l.calls.Add(1))
	if n <= l.failCount {
		return errors.New("database is locked — test lock error")
	}
	return l.base.PersistIndex(context.Background(), Ref{}, e)
}
func (l *lockRetryIndex) EnsureTable(_ context.Context) error { return nil }

func TestWriteQueue_RetryOnLockError_EventuallySucceeds(t *testing.T) {
	base := &stubIndex{available: true}
	lockIdx := &lockRetryIndex{failCount: 2, base: base}
	q := NewWriteQueue(lockIdx, nil)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "retry-test"}
	q.Enqueue(ref, IndexEntry{Ref: ref, URL: "https://cdn.example.com/1.png"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q.Shutdown(ctx)

	if len(base.persisted) != 1 {
		t.Errorf("attendu 1 entrée persistée après retry, got %d", len(base.persisted))
	}
}

func TestWriteQueue_MaxRetries_DropsJob(t *testing.T) {
	// Toujours une lock error → job droppé après writeMaxRetry tentatives.
	alwaysLocked := &lockRetryIndex{failCount: writeMaxRetry + 1, base: &stubIndex{available: true}}
	q := NewWriteQueue(alwaysLocked, nil)

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "max-retry"}
	q.Enqueue(ref, IndexEntry{Ref: ref})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	q.Shutdown(ctx)

	// Le job est droppé : aucune écriture dans base.persisted.
	if len(alwaysLocked.base.persisted) != 0 {
		t.Errorf("attendu 0 entrées (job droppé), got %d", len(alwaysLocked.base.persisted))
	}
}

func TestWriteQueue_Shutdown_ContextTimeout(t *testing.T) {
	// Shutdown avec un context déjà expiré ne doit pas paniquer.
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	q.Shutdown(ctx) // doit retourner immédiatement sans paniquer.
}
