package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

func TestHasRealCSR(t *testing.T) {
	cases := []struct {
		name string
		c    PlayerPlaylistCSR
		want bool
	}{
		{"vide", PlayerPlaylistCSR{}, false},
		{"tier current", PlayerPlaylistCSR{Current: CSRRankSnapshot{Tier: "Onyx"}}, true},
		{"tier alltime", PlayerPlaylistCSR{AllTime: CSRRankSnapshot{Tier: "Diamond"}}, true},
		{"placement seul (remaining, sans tier)", PlayerPlaylistCSR{Current: CSRRankSnapshot{MeasurementMatchesRemaining: 5}}, false},
	}
	for _, c := range cases {
		if got := hasRealCSR(c.c); got != c.want {
			t.Errorf("%s: hasRealCSR = %v, attendu %v", c.name, got, c.want)
		}
	}
}

// TestFetchSeasonRankedCSRs vérifie que le backfill ne retient que les entrées à
// tier réel, renseigne le nom depuis la référence, et ignore les nil/placement.
func TestFetchSeasonRankedCSRs(t *testing.T) {
	pls := rankedplaylists.All()
	if len(pls) < 3 {
		t.Skip("référence < 3 playlists")
	}
	stub := &augmentStubClient{resp: map[string]*PlayerPlaylistCSR{
		// pl[0] : rang réel → retenu.
		pls[0].AssetID: {PlaylistID: pls[0].AssetID, Current: CSRRankSnapshot{Tier: "Gold", SubTier: 2, Value: 1100}},
		// pl[1] : placement sans tier → ignoré.
		pls[1].AssetID: {PlaylistID: pls[1].AssetID, Current: CSRRankSnapshot{MeasurementMatchesRemaining: 5}},
		// pl[2] : nil (jamais joué) → ignoré.
	}}

	out := fetchSeasonRankedCSRs(context.Background(), stub, "123", "CsrSeason12-1", pls[:3])
	if len(out) != 1 {
		t.Fatalf("attendu 1 entrée (tier réel), obtenu %d : %+v", len(out), out)
	}
	if out[0].PlaylistID != pls[0].AssetID {
		t.Errorf("playlist_id = %q, attendu %q", out[0].PlaylistID, pls[0].AssetID)
	}
	if out[0].PlaylistName != pls[0].NameEN {
		t.Errorf("nom = %q, attendu référence %q", out[0].PlaylistName, pls[0].NameEN)
	}
	if out[0].Current.Tier != "Gold" {
		t.Errorf("tier perdu: %q", out[0].Current.Tier)
	}
}
