package duckdb

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

// Le libellé de palier d'une playlist récente de l'Accueil se déduit du code (palier + sous-palier de la
// ligne de rang) dans la langue de la requête : le libellé stocké (« Diamond 4 » pour Halo 5, « Or VI »
// pour Halo Infinite) n'est jamais servi.
func TestBuildPlaylistRankItem_LibelleDeduitDuCodeDansLaLangueDeLaRequete(t *testing.T) {
	cases := []struct {
		name   string
		msr    playlistMSRRow
		fr, en string
	}{
		{"Halo 5 stocke en anglais arabe", playlistMSRRow{ratingValue: 1200, tier: "Diamond", subTier: 4, tierLabel: "Diamond 4"}, "Diamant IV", "Diamond IV"},
		{"Infinite stocke en francais", playlistMSRRow{ratingValue: 1700, tier: "Gold", subTier: 6, tierLabel: "Or VI"}, "Or VI", "Gold VI"},
	}
	p := playlistPhaseBRow{playlistID: "pl", isRanked: true, lastMatchID: "m1", recentMatchIDs: []string{"m1"}}
	for _, c := range cases {
		for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: c.fr, i18n.LocaleEN: c.en} {
			item := buildPlaylistRankItem("halo_infinite", p, map[string]playlistMSRRow{"m1": c.msr}, nil, 5, loc)
			if item.TierLabel == nil || *item.TierLabel != want {
				t.Errorf("%s (%s) : TierLabel = %v, attendu %q", c.name, loc, item.TierLabel, want)
			}
		}
	}
}
