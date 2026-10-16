package wire

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// Les extras des amis ne lisent que des player DB de stats : le joueur courant, un profil sans xuid
// et un profil auth_only (tokens seulement) sont ABSENTS de la map, donc jamais une panne de lecture ;
// un profil en pause de sync reste lisible.
func TestFriendProfilesByXUID(t *testing.T) {
	players := []domain.PlayerSummary{
		{Gamertag: "Moi", XUID: "x0"},
		{Gamertag: "Ami", XUID: "x1"},
		{Gamertag: "EnPause", XUID: "x2", SyncEnabled: false},
		{Gamertag: "Tokens", XUID: "x3", AuthOnly: true},
		{Gamertag: "SansXuid"},
	}
	got := friendProfilesByXUID(players, "x0", "halo_infinite")
	if len(got) != 2 {
		t.Fatalf("attendu 2 amis lisibles (x1, x2), obtenu %v", got)
	}
	for _, xuid := range []string{"x1", "x2"} {
		if p, ok := got[xuid]; !ok || p.TitleSlug != "halo_infinite" {
			t.Errorf("%s doit être lisible avec le titre courant, obtenu %+v", xuid, p)
		}
	}
	for _, xuid := range []string{"x0", "x3", ""} {
		if _, ok := got[xuid]; ok {
			t.Errorf("%q ne doit pas être lu (absence, pas panne)", xuid)
		}
	}
}
