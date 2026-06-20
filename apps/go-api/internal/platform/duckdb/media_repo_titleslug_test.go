package duckdb

import "testing"

// TestResolveMediaTitleSlug : le titre du PlayerDB pilote les requêtes de
// traduction maps/modes ; repli Halo Infinite si vide (byte-identique HI).
func TestResolveMediaTitleSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"halo_infinite", "halo_infinite"}, // HI inchangé
		{"halo_5", "halo_5"},               // title-aware
		{"", "halo_infinite"},              // repli
		{"   ", "halo_infinite"},           // blanc → repli
		{"  halo_5  ", "halo_5"},           // trim
	}
	for _, c := range cases {
		if got := resolveMediaTitleSlug(c.in); got != c.want {
			t.Errorf("resolveMediaTitleSlug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
