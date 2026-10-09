package domain

import "testing"

// TestMatchAssistedFrags_WithOfficialFrags : la base est le compte officiel, l'écart au
// film est « sans information » ; un compte officiel absent ou inférieur au film ne
// fabrique jamais de frags inconnus négatifs.
func TestMatchAssistedFrags_WithOfficialFrags(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	cases := []struct {
		name                  string
		kills                 *int
		wantOfficial, wantUnk int
	}{
		// Match CTF de référence : 20 frags officiels, 14 mesurés (6 sur des bots).
		{name: "officiel au-dessus du film", kills: intPtr(20), wantOfficial: 20, wantUnk: 6},
		{name: "film complet", kills: intPtr(14), wantOfficial: 14, wantUnk: 0},
		{name: "officiel absent", kills: nil, wantOfficial: 14, wantUnk: 0},
		{name: "officiel sous le film", kills: intPtr(9), wantOfficial: 14, wantUnk: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := MatchAssistedFrags{FragsMeasured: 14, Received: AssistTiers{Total: 10, Low: 4, Mid: 4, High: 2}}
			got := in.WithOfficialFrags(tc.kills)
			if got.FragsOfficial != tc.wantOfficial || got.FragsUnknown != tc.wantUnk {
				t.Fatalf("official=%d unknown=%d, want %d / %d", got.FragsOfficial, got.FragsUnknown, tc.wantOfficial, tc.wantUnk)
			}
			if got.FragsMeasured != 14 || got.Received != in.Received {
				t.Fatalf("mesure modifiée : %+v", got)
			}
		})
	}
}
