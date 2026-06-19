package domain

import "testing"

func TestSyncablePlayers_FiltersPaused(t *testing.T) {
	in := []PlayerSummary{
		{Gamertag: "A", TitleSlug: "halo_infinite", SyncEnabled: true},
		{Gamertag: "A", TitleSlug: "halo_5", SyncEnabled: false}, // même gamertag, titre en pause
		{Gamertag: "C", TitleSlug: "halo_infinite", SyncEnabled: true},
	}
	out := SyncablePlayers(in)
	if len(out) != 2 {
		t.Fatalf("attendu 2 couples actifs, reçu %d", len(out))
	}
	for _, p := range out {
		if !p.SyncEnabled {
			t.Fatalf("un couple en pause a passé le filtre: %s/%s", p.Gamertag, p.TitleSlug)
		}
	}
}

func TestSyncablePlayers_EmptyInput(t *testing.T) {
	if got := SyncablePlayers(nil); len(got) != 0 {
		t.Fatalf("attendu vide, reçu %d", len(got))
	}
}

func TestSyncablePlayers_AllActive_ReturnsAll(t *testing.T) {
	in := []PlayerSummary{{Gamertag: "A", SyncEnabled: true}, {Gamertag: "B", SyncEnabled: true}}
	if got := SyncablePlayers(in); len(got) != 2 {
		t.Fatalf("attendu 2, reçu %d", len(got))
	}
}
