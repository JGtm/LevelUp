package duckdb

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestParseCSRSeasonNumber(t *testing.T) {
	cases := []struct {
		id          string
		major, minor int
	}{
		{"CsrSeason13-1", 13, 1},
		{"CsrSeason8", 8, 0},
		{"CsrSeason2-3", 2, 3},
		{"Season13", 0, 0}, // mauvais préfixe
		{"", 0, 0},
	}
	for _, c := range cases {
		gotMajor, gotMinor := parseCSRSeasonNumber(c.id)
		if gotMajor != c.major || gotMinor != c.minor {
			t.Errorf("parseCSRSeasonNumber(%q) = (%d,%d), attendu (%d,%d)", c.id, gotMajor, gotMinor, c.major, c.minor)
		}
	}
}

func TestCSRSeasonLabel(t *testing.T) {
	if got := csrSeasonLabel("CsrSeason13-1"); got != "Saison 13" {
		t.Errorf("label CsrSeason13-1 = %q, attendu 'Saison 13'", got)
	}
	if got := csrSeasonLabel("inconnu"); got != "inconnu" {
		t.Errorf("label fallback = %q, attendu 'inconnu'", got)
	}
}

func TestSortCSRSeasonsDesc(t *testing.T) {
	opts := []domain.CSRSeasonOption{
		{SeasonID: "CsrSeason2"},
		{SeasonID: "CsrSeason13-1"},
		{SeasonID: "CsrSeason13-2"},
		{SeasonID: "CsrSeason1-2"},
	}
	sortCSRSeasonsDesc(opts)
	want := []string{"CsrSeason13-2", "CsrSeason13-1", "CsrSeason2", "CsrSeason1-2"}
	for i, w := range want {
		if opts[i].SeasonID != w {
			t.Errorf("ordre[%d] = %q, attendu %q (tri numérique décroissant, pas lexicographique)", i, opts[i].SeasonID, w)
		}
	}
}
