package livesync

import (
	"testing"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
)

// arenaResp construit un H5ServiceRecordResponse OK (ResultCode 0) portant un seul
// résultat arena avec les playlists fournies.
func arenaResp(playlists []halo5.H5ArenaPlaylistStat) *halo5.H5ServiceRecordResponse {
	return &halo5.H5ServiceRecordResponse{
		Results: []halo5.H5ServiceRecordResult{{
			Id:         "JGtm",
			ResultCode: 0,
			Result: halo5.H5ServiceRecordBody{
				ArenaStats: &halo5.H5ArenaStats{ArenaPlaylistStats: playlists},
			},
		}},
	}
}

func csrPtr(designation, tier, value int) *halo5.H5Csr {
	return &halo5.H5Csr{DesignationId: designation, Tier: tier, Csr: value}
}

func findCSR(t *testing.T, out []syncpkg.PlayerPlaylistCSR, playlistID string) syncpkg.PlayerPlaylistCSR {
	t.Helper()
	for _, c := range out {
		if c.PlaylistID == playlistID {
			return c
		}
	}
	t.Fatalf("playlist %q absente du mapping (%d lignes)", playlistID, len(out))
	return syncpkg.PlayerPlaylistCSR{}
}

func TestMapH5ArenaToPlaylistCSRs_TierAndAllTime(t *testing.T) {
	// Diamant 5 courant, pic AllTime Onyx (sous-palier ignoré).
	resp := arenaResp([]halo5.H5ArenaPlaylistStat{{
		PlaylistId:             "pl-slayer",
		MeasurementMatchesLeft: 0,
		Csr:                    csrPtr(4, 5, 1450), // Diamond, sub 5
		HighestCsr:             csrPtr(5, 3, 1600), // Onyx (sub ignoré)
	}})

	out := mapH5ArenaToPlaylistCSRs(resp)
	if len(out) != 1 {
		t.Fatalf("attendu 1 ligne, obtenu %d", len(out))
	}
	c := findCSR(t, out, "pl-slayer")

	if c.Current.Tier != "Diamond" || c.Current.SubTier != 5 || c.Current.Value != 1450 {
		t.Errorf("Current: got tier=%q sub=%d val=%v, want Diamond/5/1450",
			c.Current.Tier, c.Current.SubTier, c.Current.Value)
	}
	if c.Current.MeasurementMatchesRemaining != 0 {
		t.Errorf("Current.MeasurementMatchesRemaining: got %d, want 0", c.Current.MeasurementMatchesRemaining)
	}
	// Onyx = palier unique → sous-palier forcé à 0.
	if c.AllTime.Tier != "Onyx" || c.AllTime.SubTier != 0 || c.AllTime.Value != 1600 {
		t.Errorf("AllTime: got tier=%q sub=%d val=%v, want Onyx/0/1600",
			c.AllTime.Tier, c.AllTime.SubTier, c.AllTime.Value)
	}
}

func TestMapH5ArenaToPlaylistCSRs_Placement(t *testing.T) {
	// En placement : Csr nil, 7 matchs restants → Current sans tier + remaining=7.
	resp := arenaResp([]halo5.H5ArenaPlaylistStat{{
		PlaylistId:             "pl-ctf",
		MeasurementMatchesLeft: 7,
		Csr:                    nil,
		HighestCsr:             nil,
	}})

	out := mapH5ArenaToPlaylistCSRs(resp)
	c := findCSR(t, out, "pl-ctf")
	if c.Current.Tier != "" || c.Current.Value != 0 {
		t.Errorf("placement Current: got tier=%q val=%v, want vide/0", c.Current.Tier, c.Current.Value)
	}
	if c.Current.MeasurementMatchesRemaining != 7 {
		t.Errorf("placement remaining: got %d, want 7", c.Current.MeasurementMatchesRemaining)
	}
	if c.AllTime.Tier != "" {
		t.Errorf("placement AllTime: got tier=%q, want vide", c.AllTime.Tier)
	}
}

func TestMapH5ArenaToPlaylistCSRs_DesignationLabels(t *testing.T) {
	cases := []struct {
		designation int
		wantTier    string
	}{
		{0, "Bronze"},
		{1, "Silver"},
		{2, "Gold"},
		{3, "Platinum"},
		{4, "Diamond"},
		{5, "Onyx"},
		{6, "Champion"}, // Champion confirmé live (Rank #236) → mappé.
		{7, ""},         // hors borne → vide
		{-1, ""},
	}
	for _, tc := range cases {
		resp := arenaResp([]halo5.H5ArenaPlaylistStat{{
			PlaylistId: "pl",
			Csr:        csrPtr(tc.designation, 1, 100),
		}})
		out := mapH5ArenaToPlaylistCSRs(resp)
		if len(out) != 1 {
			t.Fatalf("designation %d: attendu 1 ligne", tc.designation)
		}
		if out[0].Current.Tier != tc.wantTier {
			t.Errorf("designation %d: tier got %q, want %q", tc.designation, out[0].Current.Tier, tc.wantTier)
		}
	}
}

func TestMapH5ArenaToPlaylistCSRs_EmptyAndNil(t *testing.T) {
	if out := mapH5ArenaToPlaylistCSRs(nil); out != nil {
		t.Errorf("nil resp: got %v, want nil", out)
	}
	if out := mapH5ArenaToPlaylistCSRs(arenaResp(nil)); out != nil {
		t.Errorf("arena sans playlist: got %v, want nil", out)
	}
	// Playlist sans PlaylistId → ignorée → résultat nil.
	resp := arenaResp([]halo5.H5ArenaPlaylistStat{{PlaylistId: "", Csr: csrPtr(2, 1, 100)}})
	if out := mapH5ArenaToPlaylistCSRs(resp); out != nil {
		t.Errorf("playlist sans id: got %v, want nil", out)
	}
}

func TestMapH5ArenaToPlaylistCSRs_SkipsNonOKResult(t *testing.T) {
	resp := &halo5.H5ServiceRecordResponse{
		Results: []halo5.H5ServiceRecordResult{{
			Id:         "JGtm",
			ResultCode: 1, // non OK → ignoré
			Result: halo5.H5ServiceRecordBody{
				ArenaStats: &halo5.H5ArenaStats{ArenaPlaylistStats: []halo5.H5ArenaPlaylistStat{{
					PlaylistId: "pl", Csr: csrPtr(2, 1, 100),
				}}},
			},
		}},
	}
	if out := mapH5ArenaToPlaylistCSRs(resp); out != nil {
		t.Errorf("ResultCode!=0: got %v, want nil", out)
	}
}
