package duckdb

import "testing"

// arenaRankedID : playlist Arène classée (présente dans rankedplaylists, NameFR connu).
const arenaRankedID = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"

// TestPlaylistName_Cascade valide l'ordre de résolution Phase F :
// asset_translations[fr] > rankedplaylists FR (curé) > catalogue EN > id.
func TestPlaylistName_Cascade(t *testing.T) {
	cases := []struct {
		name       string
		id         string
		frOfficial string
		canonical  string
		want       string
	}{
		{"officiel FR prioritaire", arenaRankedID, "Arène Officielle", "Ranked Arena", "Arène Officielle"},
		{"curé FR bat le canonical EN", arenaRankedID, "", "Ranked Arena", "Arène classée"},
		{"canonical EN si pas de curé", "unknown-pl-id", "", "Big Team Social", "Big Team Social"},
		{"id brut en dernier recours", "unknown-pl-id", "", "", "unknown-pl-id"},
	}
	for _, c := range cases {
		if got := playlistName(c.id, c.frOfficial, c.canonical); got != c.want {
			t.Errorf("%s: playlistName(%q, %q, %q) = %q, want %q",
				c.name, c.id, c.frOfficial, c.canonical, got, c.want)
		}
	}
}
