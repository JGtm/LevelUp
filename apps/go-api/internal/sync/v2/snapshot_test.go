package v2

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// fakeSnapshotProducer enregistre les appels à CutSnapshot pour les assertions.
type fakeSnapshotProducer struct {
	calls   int
	slug    string
	players []string
	err     error
}

func (f *fakeSnapshotProducer) CutSnapshot(_ context.Context, titleSlug string, gamertags []string) error {
	f.calls++
	f.slug = titleSlug
	f.players = gamertags
	return f.err
}

// TestCutSnapshot_NilProducer_NoPanic : producteur non câblé → no-op silencieux
// (nil-guard explicite, jamais de panic).
func TestCutSnapshot_NilProducer_NoPanic(t *testing.T) {
	o := &CycleOrchestratorImpl{} // snapshotProducer nil
	o.cutSnapshot(context.Background(), []PlayerProfile{{Gamertag: "A"}})
	// Aucune assertion : l'absence de panic est le test.
}

// TestCutSnapshot_InvokesProducer : producteur câblé → CutSnapshot appelé une fois avec
// le slug du contexte et les gamertags non vides des joueurs du cycle.
func TestCutSnapshot_InvokesProducer(t *testing.T) {
	fake := &fakeSnapshotProducer{}
	o := (&CycleOrchestratorImpl{}).WithSnapshotProducer(fake)
	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_5")

	o.cutSnapshot(ctx, []PlayerProfile{
		{Gamertag: "Alpha"},
		{Gamertag: ""}, // filtré (vide)
		{Gamertag: "Bravo"},
	})

	if fake.calls != 1 {
		t.Fatalf("CutSnapshot appelé %d fois, attendu 1", fake.calls)
	}
	if fake.slug != "halo_5" {
		t.Errorf("slug = %q, attendu halo_5 (depuis le contexte)", fake.slug)
	}
	if len(fake.players) != 2 || fake.players[0] != "Alpha" || fake.players[1] != "Bravo" {
		t.Errorf("players = %v, attendu [Alpha Bravo] (vide filtré)", fake.players)
	}
}

// TestCutSnapshot_ErrorIsBestEffort : un échec du producteur ne propage pas (best-effort,
// loggé en WARN) — l'appel retourne sans panic.
func TestCutSnapshot_ErrorIsBestEffort(t *testing.T) {
	fake := &fakeSnapshotProducer{err: errors.New("boom")}
	o := (&CycleOrchestratorImpl{}).WithSnapshotProducer(fake)
	o.cutSnapshot(context.Background(), []PlayerProfile{{Gamertag: "A"}})
	if fake.calls != 1 {
		t.Fatalf("CutSnapshot appelé %d fois, attendu 1", fake.calls)
	}
}
