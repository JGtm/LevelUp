package rankedplaylists

import "testing"

func TestActiveCount(t *testing.T) {
	active := Active()
	if len(active) != 4 {
		t.Fatalf("attendu 4 playlists classées actives, obtenu %d", len(active))
	}
	// Toutes les actives doivent avoir un nom EN et FR (affichées sur la page Carrière).
	for _, p := range active {
		if p.NameEN == "" || p.NameFR == "" {
			t.Errorf("playlist active %s sans nom EN/FR", p.AssetID)
		}
	}
}

func TestIsRanked(t *testing.T) {
	cases := map[string]bool{
		"edfef3ac-9cbe-4fa2-b949-8f29deafd483": true,  // Ranked Arena
		"dcb2e24e-05fb-4390-8076-32a0cdb4326e": true,  // Ranked Slayer
		"fa5aa2a3-2428-4912-a023-e1eeea7b877c": true,  // Ranked Doubles
		"EDFEF3AC-9CBE-4FA2-B949-8F29DEAFD483": true,  // casse-insensible
		"bdceefb3-1c52-4848-a6b7-d49acd13109d": false, // Quick Play (social)
		"":                                     false,
	}
	for id, want := range cases {
		if got := IsRanked(id); got != want {
			t.Errorf("IsRanked(%q) = %v, attendu %v", id, got, want)
		}
	}
}

func TestNoDuplicateAssetIDs(t *testing.T) {
	seen := make(map[string]struct{}, len(all))
	for _, p := range all {
		if _, dup := seen[p.AssetID]; dup {
			t.Errorf("asset_id dupliqué : %s", p.AssetID)
		}
		seen[p.AssetID] = struct{}{}
	}
}
