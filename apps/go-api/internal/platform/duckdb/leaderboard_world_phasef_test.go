package duckdb

import "testing"

// arenaRankedID : playlist Arène classée (présente dans rankedplaylists, NameFR connu).
const arenaRankedID = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"

// TestPlaylistName_Cascade valide l'ordre de résolution locale-aware :
//
//	FR : asset_translations[fr] > rankedplaylists FR (curé) > catalogue EN > EN officiel > id.
//	EN : asset_translations[en] > rankedplaylists EN (curé) > catalogue EN > FR officiel > id.
func TestPlaylistName_Cascade(t *testing.T) {
	cases := []struct {
		name       string
		id         string
		locale     string
		frOfficial string
		enOfficial string
		canonical  string
		want       string
	}{
		// FR
		{"FR officiel prioritaire", arenaRankedID, "fr", "Arène Officielle", "Official Arena", "Ranked Arena", "Arène Officielle"},
		{"FR curé bat le canonical EN", arenaRankedID, "fr", "", "", "Ranked Arena", "Arène classée"},
		{"FR canonical EN si pas de curé", "unknown-pl-id", "fr", "", "", "Big Team Social", "Big Team Social"},
		{"FR id brut en dernier recours", "unknown-pl-id", "fr", "", "", "", "unknown-pl-id"},
		// EN
		{"EN officiel prioritaire", arenaRankedID, "en", "Arène Officielle", "Official Arena", "Ranked Arena", "Official Arena"},
		{"EN curé bat le canonical", arenaRankedID, "en", "Arène classée", "", "Big Team Social", "Ranked Arena"},
		{"EN canonical si pas de curé", "unknown-pl-id", "en", "", "", "Big Team Social", "Big Team Social"},
		{"EN id brut en dernier recours", "unknown-pl-id", "en", "", "", "", "unknown-pl-id"},
	}
	for _, c := range cases {
		if got := playlistName(c.id, c.locale, c.frOfficial, c.enOfficial, c.canonical); got != c.want {
			t.Errorf("%s: playlistName(%q, %q, %q, %q, %q) = %q, want %q",
				c.name, c.id, c.locale, c.frOfficial, c.enOfficial, c.canonical, got, c.want)
		}
	}
}
