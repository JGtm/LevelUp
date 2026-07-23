package notifications

import "testing"

// TestPlayerTargetRoute verrouille le format canonique title-scopé et documente les
// cas de suffixe (simple, multi-segment, dynamique, vide).
func TestPlayerTargetRoute(t *testing.T) {
	cases := []struct {
		name                        string
		titleSlug, playerSlug, suff string
		want                        string
	}{
		{
			name:      "suffixe simple",
			titleSlug: "halo_infinite", playerSlug: "jgtm", suff: "home",
			want: "/t/halo_infinite/players/jgtm/home",
		},
		{
			name:      "suffixe multi-segment (route relocalisée)",
			titleSlug: "halo_infinite", playerSlug: "jgtm", suff: "stats/synthesis",
			want: "/t/halo_infinite/players/jgtm/stats/synthesis",
		},
		{
			name:      "suffixe dynamique (id de match)",
			titleSlug: "halo_infinite", playerSlug: "jgtm", suff: "matches/m-42",
			want: "/t/halo_infinite/players/jgtm/matches/m-42",
		},
		{
			// Preuve title-agnostic : un titre non-défaut produit son propre segment,
			// aucun hardcodage halo_infinite dans le helper.
			name:      "titre non-défaut",
			titleSlug: "halo_5", playerSlug: "jgtm", suff: "career/citations",
			want: "/t/halo_5/players/jgtm/career/citations",
		},
		{
			// Documenté : suffixe vide = racine joueur. Aucun émetteur ne l'utilise.
			name:      "suffixe vide → racine joueur",
			titleSlug: "halo_infinite", playerSlug: "jgtm", suff: "",
			want: "/t/halo_infinite/players/jgtm/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlayerTargetRoute(tc.titleSlug, tc.playerSlug, tc.suff)
			if got != tc.want {
				t.Errorf("PlayerTargetRoute(%q,%q,%q) = %q, want %q",
					tc.titleSlug, tc.playerSlug, tc.suff, got, tc.want)
			}
		})
	}
}
