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

func TestSyncablePlayers_ExcludesAuthOnly(t *testing.T) {
	in := []PlayerSummary{
		{Gamertag: "RealPlayer", TitleSlug: "halo_infinite", SyncEnabled: true, AuthOnly: false},
		{Gamertag: "DankerGlue", TitleSlug: "halo_infinite", SyncEnabled: true, AuthOnly: true},
		{Gamertag: "QuiteSiren", TitleSlug: "halo_infinite", SyncEnabled: true, AuthOnly: true},
	}
	out := SyncablePlayers(in)
	if len(out) != 1 {
		t.Fatalf("attendu 1 joueur (auth_only exclus), reçu %d : %+v", len(out), out)
	}
	if out[0].Gamertag != "RealPlayer" {
		t.Fatalf("attendu RealPlayer, reçu %s", out[0].Gamertag)
	}
}

// Non-régression : un profil auth_only reste exclu même si sync_enabled est true
// (l'exclusion est un ET, pas un cas particulier de la pause).
func TestSyncablePlayers_ExcludesAuthOnly_EvenIfSyncEnabled(t *testing.T) {
	in := []PlayerSummary{
		{Gamertag: "AuthOnlyActive", SyncEnabled: true, AuthOnly: true},
	}
	if got := SyncablePlayers(in); len(got) != 0 {
		t.Fatalf("attendu 0 (auth_only exclu même actif), reçu %d", len(got))
	}
}
