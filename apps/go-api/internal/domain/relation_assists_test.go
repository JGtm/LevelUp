package domain

import "testing"

// TestMatchAssistedFrags_WithOfficialFrags : la base est le compte officiel de la feuille
// de match, frags sur des bots compris ; un compte officiel absent ou inférieur au film ne
// descend jamais sous les frags lus.
func TestMatchAssistedFrags_WithOfficialFrags(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	cases := []struct {
		name         string
		kills        *int
		wantOfficial int
	}{
		// Narrows 0a08d2f2 : 20 frags officiels, 20 lus par le film dont 6 sur des bots.
		{name: "officiel au-dessus du film", kills: intPtr(20), wantOfficial: 20},
		{name: "film complet", kills: intPtr(14), wantOfficial: 14},
		{name: "officiel absent", kills: nil, wantOfficial: 14},
		{name: "officiel sous le film", kills: intPtr(9), wantOfficial: 14},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := MatchAssistedFrags{FragsFilm: 14, Received: AssistTiers{Total: 10, Low: 4, Mid: 4, High: 2}}
			got := in.WithOfficialFrags(tc.kills)
			if got.FragsOfficial != tc.wantOfficial {
				t.Fatalf("official=%d, want %d", got.FragsOfficial, tc.wantOfficial)
			}
			if got.FragsFilm != 14 || got.Received != in.Received {
				t.Fatalf("mesure modifiée : %+v", got)
			}
		})
	}
}
