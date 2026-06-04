package sync

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeChunkFetcher : highlightChunkFetcher déterministe. found=true dès que le
// nombre d'appels dépasse readyAfter (simule un film publié après N tentatives).
type fakeChunkFetcher struct {
	calls      int
	readyAfter int
}

func (f *fakeChunkFetcher) GetHighlightEventsChunk(_ context.Context, _ string) ([]byte, int, bool, error) {
	f.calls++
	if f.calls > f.readyAfter {
		return []byte("chunk"), 1, true, nil
	}
	return nil, 0, false, nil
}

func withShortFilmRetries(t *testing.T, delays []time.Duration) {
	t.Helper()
	orig := freshFilmRetryDelays
	freshFilmRetryDelays = func() []time.Duration { return delays }
	t.Cleanup(func() { freshFilmRetryDelays = orig })
}

// Match frais, film pas prêt aux 2 premiers fetch → retry → trouvé au 3e.
// Garantie produit : le match s'enrichit COMPLET dans le sync qui le découvre.
func TestFetchHighlightChunkResilient_RetriesUntilFilmReady(t *testing.T) {
	withShortFilmRetries(t, []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond})
	f := &fakeChunkFetcher{readyAfter: 2}
	_, _, found, err := fetchHighlightChunkResilient(context.Background(), f, "m1", time.Now())
	if err != nil || !found {
		t.Fatalf("attendu found=true sans erreur, got found=%v err=%v", found, err)
	}
	if f.calls != 3 {
		t.Errorf("attendu 3 fetch (1 + 2 retry), got %d", f.calls)
	}
}

// Vieux match (hors fenêtre fraîche) : aucun retry, 1 seul fetch (un backfill
// n'attend pas un film qui est soit là soit définitivement absent).
func TestFetchHighlightChunkResilient_NoRetryForOldMatch(t *testing.T) {
	withShortFilmRetries(t, []time.Duration{time.Millisecond, time.Millisecond})
	f := &fakeChunkFetcher{readyAfter: 99}
	_, _, found, _ := fetchHighlightChunkResilient(context.Background(), f, "m1", time.Now().Add(-time.Hour))
	if found {
		t.Fatal("vieux match : pas censé trouver")
	}
	if f.calls != 1 {
		t.Errorf("vieux match : pas de retry attendu, got %d fetch", f.calls)
	}
}

// Film prêt dès le 1er fetch → aucun retry (cas nominal, zéro latence ajoutée).
func TestFetchHighlightChunkResilient_NoRetryWhenReadyImmediately(t *testing.T) {
	withShortFilmRetries(t, []time.Duration{time.Millisecond})
	f := &fakeChunkFetcher{readyAfter: 0}
	_, _, found, _ := fetchHighlightChunkResilient(context.Background(), f, "m1", time.Now())
	if !found || f.calls != 1 {
		t.Errorf("film prêt d'emblée : 1 fetch + found, got calls=%d found=%v", f.calls, found)
	}
}

// ctx annulé pendant l'attente → retourne ctx.Err() (ne bloque pas le shutdown).
func TestFetchHighlightChunkResilient_RespectsCtxCancel(t *testing.T) {
	withShortFilmRetries(t, []time.Duration{time.Hour})
	f := &fakeChunkFetcher{readyAfter: 99}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err := fetchHighlightChunkResilient(ctx, f, "m1", time.Now())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("attendu context.Canceled, got %v", err)
	}
}
