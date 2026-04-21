package assets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// stubFetcher est un Fetcher de test.
type stubFetcher struct {
	payload   Payload
	err       error
	supported bool
	calls     int
}

func (f *stubFetcher) Supports(_ Kind) bool { return f.supported }

func (f *stubFetcher) Fetch(_ context.Context, _ Ref) (Payload, error) {
	f.calls++
	return f.payload, f.err
}

// stubBinaryStore est un BinaryStore de test.
type stubBinaryStore struct {
	hit        *BinaryPayload
	lookupErr  error
	persisted  []BinaryPayload
	persistErr error
	pathCalled int
}

func (s *stubBinaryStore) LookupBinary(_ context.Context, _ Ref) (*BinaryPayload, error) {
	return s.hit, s.lookupErr
}

func (s *stubBinaryStore) PersistBinary(_ context.Context, _ Ref, p BinaryPayload) error {
	if s.persistErr != nil {
		return s.persistErr
	}
	s.persisted = append(s.persisted, p)
	return nil
}

func (s *stubBinaryStore) Path(_ Ref) string {
	s.pathCalled++
	return "/fake/path.png"
}

// ---------------------------------------------------------------------------
// Tests Get() — flux 6 étapes
// ---------------------------------------------------------------------------

func TestDefaultResolver_Get_FSHit(t *testing.T) {
	// Étape 1 : hit FS → retour direct sans fetch.
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47}
	store := &stubBinaryStore{hit: &BinaryPayload{ContentType: "image/png", Bytes: pngBytes, ETag: "abc"}}
	fetcher := &stubFetcher{supported: true}

	r := NewDefaultResolver(store, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	res, err := r.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if res.Source != SourceLocalFile {
		t.Errorf("source: got %v, want SourceLocalFile", res.Source)
	}
	if fetcher.calls != 0 {
		t.Error("le fetcher ne devrait pas être appelé sur un hit FS")
	}
}

func TestDefaultResolver_Get_IndexHit(t *testing.T) {
	// Étape 2 : miss FS, hit index → URLPayload.
	store := &stubBinaryStore{hit: nil}
	idx := &stubIndex{
		available: true,
		entry: &IndexEntry{
			Ref:       Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"},
			URL:       "https://cdn.example.com/medal.png",
			FetchedAt: time.Now(),
		},
	}
	fetcher := &stubFetcher{supported: true}

	r := NewDefaultResolver(store, idx, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	res, err := r.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if res.Source != SourceLocalDB {
		t.Errorf("source: got %v, want SourceLocalDB", res.Source)
	}
	if fetcher.calls != 0 {
		t.Error("le fetcher ne devrait pas être appelé sur un hit index")
	}
}

func TestDefaultResolver_Get_FetchBinaryPayload(t *testing.T) {
	// Étapes 3-6 : miss FS + miss index → fetch → persist FS → enqueue index.
	dir := t.TempDir()
	store := NewLocalFSStore(dir)
	idx := &stubIndex{available: true}
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01}
	fetcher := &stubFetcher{
		supported: true,
		payload:   BinaryPayload{ContentType: "image/png", Bytes: pngBytes},
	}
	q := NewWriteQueue(idx, nil)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		q.Shutdown(ctx)
	}()

	r := NewDefaultResolver(store, idx, fetcher, q, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "medal-fetch-test"}

	res, err := r.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if res.Source != SourceRemote {
		t.Errorf("source: got %v, want SourceRemote", res.Source)
	}

	// Vérifier que le fichier est bien persisté sur FS.
	path := store.Path(ref)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("fichier non persisté sur FS: %v", err)
	}
}

func TestDefaultResolver_Get_FetchURLPayload(t *testing.T) {
	// Fetch retourne URLPayload → pas d'écriture FS, URL retournée directement.
	store := &stubBinaryStore{hit: nil}
	idx := &stubIndex{available: false}
	fetcher := &stubFetcher{
		supported: true,
		payload:   URLPayload{URL: "https://cdn.example.com/img.png", ContentType: "image/png"},
	}

	r := NewDefaultResolver(store, idx, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "url-test"}

	res, err := r.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if res.Source != SourceRemote {
		t.Errorf("source: got %v, want SourceRemote", res.Source)
	}
	urlp, ok := res.Payload.(URLPayload)
	if !ok {
		t.Fatal("payload devrait être URLPayload")
	}
	if urlp.URL != "https://cdn.example.com/img.png" {
		t.Errorf("URL inattendue: %q", urlp.URL)
	}
}

func TestDefaultResolver_Get_UnsupportedKind(t *testing.T) {
	// Fetcher qui ne supporte rien → ErrUnsupportedKind.
	fetcher := &stubFetcher{supported: false}
	r := NewDefaultResolver(nil, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	_, err := r.Get(context.Background(), ref)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("erreur attendue ErrUnsupportedKind, got %v", err)
	}
}

func TestDefaultResolver_Get_FetchError(t *testing.T) {
	// Fetch échoue → ErrUpstreamUnavailable retourné.
	fetcher := &stubFetcher{supported: true, err: ErrUpstreamUnavailable}
	r := NewDefaultResolver(nil, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	_, err := r.Get(context.Background(), ref)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("erreur attendue ErrUpstreamUnavailable, got %v", err)
	}
}

func TestDefaultResolver_Get_FetchNotFound(t *testing.T) {
	fetcher := &stubFetcher{supported: true, err: ErrNotFound}
	r := NewDefaultResolver(nil, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "missing"}

	_, err := r.Get(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("erreur attendue ErrNotFound, got %v", err)
	}
}

func TestDefaultResolver_Get_PersistFSError_ReturnsError(t *testing.T) {
	// Persist FS échoue → ErrPersistFailed.
	store := &stubBinaryStore{persistErr: fmt.Errorf("%w: disk full", ErrPersistFailed)}
	fetcher := &stubFetcher{
		supported: true,
		payload:   BinaryPayload{ContentType: "image/png", Bytes: []byte{1, 2, 3}},
	}
	r := NewDefaultResolver(store, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	_, err := r.Get(context.Background(), ref)
	if !errors.Is(err, ErrPersistFailed) {
		t.Errorf("erreur attendue ErrPersistFailed, got %v", err)
	}
}

func TestDefaultResolver_Get_NilFetcher_UnsupportedKind(t *testing.T) {
	r := NewDefaultResolver(nil, nil, nil, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	_, err := r.Get(context.Background(), ref)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("erreur attendue ErrUnsupportedKind, got %v", err)
	}
}

func TestDefaultResolver_Get_IndexUnavailable_FallsBackToFetch(t *testing.T) {
	// Index indisponible → ignore index, passe directement au fetch.
	store := &stubBinaryStore{hit: nil}
	idx := &stubIndex{available: false}
	fetcher := &stubFetcher{
		supported: true,
		payload:   URLPayload{URL: "https://cdn.example.com/img.png"},
	}
	r := NewDefaultResolver(store, idx, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	res, err := r.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if res.Source != SourceRemote {
		t.Errorf("source: got %v, want SourceRemote", res.Source)
	}
}

// ---------------------------------------------------------------------------
// Tests Refresh()
// ---------------------------------------------------------------------------

func TestDefaultResolver_Refresh_AlwaysFetches(t *testing.T) {
	// Refresh bypasse le cache et force le fetch.
	fetcher := &stubFetcher{
		supported: true,
		payload:   URLPayload{URL: "https://cdn.example.com/img.png"},
	}
	r := NewDefaultResolver(nil, nil, fetcher, nil, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	_, err := r.Refresh(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if fetcher.calls != 1 {
		t.Errorf("fetcher calls: got %d, want 1", fetcher.calls)
	}
}

// ---------------------------------------------------------------------------
// Tests Warm()
// ---------------------------------------------------------------------------

func TestDefaultResolver_Warm_DoesNotBlock(t *testing.T) {
	// Warm est asynchrone — ne doit pas bloquer.
	fetcher := &stubFetcher{
		supported: true,
		payload:   URLPayload{URL: "https://cdn.example.com/img.png"},
	}
	r := NewDefaultResolver(nil, nil, fetcher, nil, nil)
	refs := []Ref{
		{Kind: KindMedalImage, TitleID: "hi", ID: "1"},
		{Kind: KindMedalImage, TitleID: "hi", ID: "2"},
	}

	done := make(chan struct{})
	go func() {
		r.Warm(context.Background(), refs...)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Warm() a bloqué")
	}
	// Laisser les goroutines se terminer.
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests RegisterLocalFile()
// ---------------------------------------------------------------------------

func TestDefaultResolver_RegisterLocalFile_OK(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "medal.png")
	if err := os.WriteFile(filePath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}

	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)
	r := NewDefaultResolver(nil, idx, nil, q, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	if err := r.RegisterLocalFile(context.Background(), ref, filePath); err != nil {
		t.Fatalf("RegisterLocalFile: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	q.Shutdown(ctx)

	if len(idx.persisted) != 1 {
		t.Errorf("attendu 1 entrée persistée, got %d", len(idx.persisted))
	}
}

func TestDefaultResolver_RegisterLocalFile_MissingFile(t *testing.T) {
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)
	r := NewDefaultResolver(nil, idx, nil, q, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}

	err := r.RegisterLocalFile(context.Background(), ref, "/nonexistent/path.png")
	if err == nil {
		t.Error("erreur attendue pour fichier inexistant")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	q.Shutdown(ctx)
}

func TestDefaultResolver_RegisterLocalFile_NilIndex(t *testing.T) {
	// Sans index, RegisterLocalFile est un no-op sans erreur.
	r := NewDefaultResolver(nil, nil, nil, nil, nil)
	err := r.RegisterLocalFile(context.Background(), Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}, "/any")
	if err != nil {
		t.Errorf("attendu nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests Close()
// ---------------------------------------------------------------------------

func TestDefaultResolver_Close_Noop(t *testing.T) {
	r := NewDefaultResolver(nil, nil, nil, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.Close(ctx); err != nil {
		t.Errorf("Close() inattendu: %v", err)
	}
}

func TestDefaultResolver_Close_FlusheQueue(t *testing.T) {
	idx := &stubIndex{available: true}
	q := NewWriteQueue(idx, nil)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "1"}
	q.Enqueue(ref, IndexEntry{Ref: ref})

	r := NewDefaultResolver(nil, idx, nil, q, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := r.Close(ctx); err != nil {
		t.Errorf("Close() inattendu: %v", err)
	}
	// Après Close(), le job doit être persisté.
	if len(idx.persisted) != 1 {
		t.Errorf("attendu 1 entrée persistée après Close(), got %d", len(idx.persisted))
	}
}
